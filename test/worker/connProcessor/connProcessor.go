package connProcessor

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/RoaringBitmap/roaring"
	"github.com/buexplain/netsvr/internal/protocol/toServer/broadcast"
	"github.com/buexplain/netsvr/internal/protocol/toServer/multicast"
	"github.com/buexplain/netsvr/internal/protocol/toServer/multicastByBitmap"
	"github.com/buexplain/netsvr/internal/protocol/toServer/registerWorker"
	"github.com/buexplain/netsvr/internal/protocol/toServer/reqSessionInfo"
	toServerRouter "github.com/buexplain/netsvr/internal/protocol/toServer/router"
	"github.com/buexplain/netsvr/internal/protocol/toServer/setUserLoginStatus"
	"github.com/buexplain/netsvr/internal/protocol/toServer/singleCast"
	"github.com/buexplain/netsvr/internal/protocol/toServer/subscribe"
	"github.com/buexplain/netsvr/internal/protocol/toServer/unsubscribe"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/connClose"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/connOpen"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/respSessionInfo"
	toWorkerRouter "github.com/buexplain/netsvr/internal/protocol/toWorker/router"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/transfer"
	"github.com/buexplain/netsvr/internal/worker/heartbeat"
	"github.com/buexplain/netsvr/pkg/utils"
	"github.com/buexplain/netsvr/test/worker/protocol"
	"github.com/buexplain/netsvr/test/worker/userDb"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"time"
)

type ConnProcessor struct {
	conn     net.Conn
	writeCh  chan []byte
	closeCh  chan struct{}
	workerId int
}

func NewConnProcessor(conn net.Conn, workerId int) *ConnProcessor {
	tmp := &ConnProcessor{conn: conn, writeCh: make(chan []byte, 10), closeCh: make(chan struct{}), workerId: workerId}
	return tmp
}

func (r *ConnProcessor) Heartbeat() {
	t := time.NewTicker(time.Duration(35) * time.Second)
	defer func() {
		t.Stop()
	}()
	for {
		<-t.C
		_, _ = r.Write(heartbeat.PingMessage)
	}
}

func (r *ConnProcessor) GetWorkerId() int {
	return r.workerId
}

func (r *ConnProcessor) RegisterWorker() error {
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_RegisterWorker
	reg := &registerWorker.RegisterWorker{}
	reg.Id = int32(r.workerId)
	reg.ProcessConnClose = true
	reg.ProcessConnOpen = true
	toServerRoute.Data, _ = proto.Marshal(reg)
	data, _ := proto.Marshal(toServerRoute)
	err := binary.Write(r.conn, binary.BigEndian, uint32(len(data)))
	_, err = r.conn.Write(data)
	return err
}

func (r *ConnProcessor) UnregisterWorker() {
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_UnregisterWorker
	data, _ := proto.Marshal(toServerRoute)
	if err := binary.Write(r.conn, binary.BigEndian, uint32(len(data))); err == nil {
	}
	_, _ = r.conn.Write(data)
}

func (r *ConnProcessor) Close() {
	select {
	case <-r.closeCh:
	default:
		close(r.closeCh)
		//这里暂停一下，等待数据发送出去
		time.Sleep(time.Millisecond * 100)
		_ = r.conn.Close()
	}
}

func (r *ConnProcessor) execute(data []byte) {
	if err := binary.Write(r.conn, binary.BigEndian, uint32(len(data))); err == nil {
		if _, err = r.conn.Write(data); err != nil {
			r.Close()
			logging.Error("Worker write error: %v", err)
		}
	} else {
		r.Close()
		logging.Error("Worker write error: %v", err)
	}
}

func (r *ConnProcessor) done() {
	//处理通道中的剩余数据
	for {
		for i := len(r.writeCh); i > 0; i-- {
			v := <-r.writeCh
			r.execute(v)
		}
		if len(r.writeCh) == 0 {
			break
		}
	}
	//再次处理通道中的剩余数据，直到超时退出
	for {
		select {
		case v := <-r.writeCh:
			r.execute(v)
		case <-time.After(1 * time.Second):
			return
		}
	}
}

func (r *ConnProcessor) LoopSend() {
	defer func() {
		//收集异常退出的信息
		if err := recover(); err != nil {
			logging.Error("Abnormal exit a worker, error: %v", err)
		} else {
			logging.Debug("Worker read coroutine is close")
		}
	}()
	for {
		select {
		// TODO 思考进程停止逻辑，目标是处理缓冲中的现有数据
		case _ = <-r.closeCh:
			r.done()
			return
		case v := <-r.writeCh:
			r.execute(v)
		}
	}
}

func (r *ConnProcessor) Write(data []byte) (n int, err error) {
	select {
	case <-r.closeCh:
		return 0, nil
	default:
		r.writeCh <- data
		return len(data), nil
	}
}

func (r *ConnProcessor) LoopRead() {
	dataLenBuf := make([]byte, 4)
	for {
		dataLenBuf[0] = 0
		dataLenBuf[1] = 0
		dataLenBuf[2] = 0
		dataLenBuf[3] = 0
		if _, err := io.ReadFull(r.conn, dataLenBuf); err != nil {
			r.Close()
			break
		}
		if len(dataLenBuf) != 4 {
			continue
		}
		//这里采用大端序
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		if dataLen < 2 || dataLen > 1024*1024*2 {
			logging.Error("发送的数据太大，或者是搞错字节序，dataLen: %d", dataLen)
			continue
		}
		//获取数据包
		dataBuf := make([]byte, dataLen)
		if _, err := io.ReadFull(r.conn, dataBuf); err != nil {
			r.Close()
			logging.Error("%v", err)
			break
		}
		//服务端响应心跳
		if bytes.Equal(dataBuf, heartbeat.PongMessage) {
			continue
		}
		//服务端发来心跳
		if bytes.Equal(dataBuf, heartbeat.PingMessage) {
			logging.Info("服务端发来心跳")
			_, _ = r.Write(heartbeat.PongMessage)
			continue
		}
		//解压看看服务端传递了什么
		toWorkerRoute := &toWorkerRouter.Router{}
		if err := proto.Unmarshal(dataBuf, toWorkerRoute); err != nil {
			logging.Error("解压服务端数据失败: %v", err)
			continue
		}
		//客户端连接成功
		if toWorkerRoute.Cmd == toWorkerRouter.Cmd_ConnOpen {
			r.connOpen(toWorkerRoute)
			continue
		}
		//客户端连接
		if toWorkerRoute.Cmd == toWorkerRouter.Cmd_ConnClose {
			r.connClose(toWorkerRoute)
			continue
		}
		//网关返回了某个用户的session信息
		if toWorkerRoute.Cmd == toWorkerRouter.Cmd_RespSessionInfo {
			r.respSessionInfo(toWorkerRoute)
			continue
		}
		//客户端发来的业务请求
		if toWorkerRoute.Cmd == toWorkerRouter.Cmd_Transfer {
			//解析网关的转发过来的对象
			tf := &transfer.Transfer{}
			if err := proto.Unmarshal(toWorkerRoute.Data, tf); err != nil {
				logging.Error("Proto unmarshal transfer.Transfer error: %v", err)
				continue
			}
			//解析业务数据
			clientRoute := protocol.ParseClientRouter(tf.Data)
			if clientRoute == nil {
				continue
			}
			//判断业务路由
			if clientRoute.Cmd == protocol.RouterLogin {
				r.login(tf.SessionId, clientRoute.Data)
				continue
			}
			//处理单播
			if clientRoute.Cmd == protocol.RouterSingleCast {
				r.singleCast(tf.SessionId, userDb.ParseNetSvrInfo(tf.User), clientRoute.Data)
				continue
			}
			//处理组播
			if clientRoute.Cmd == protocol.RouterMulticast {
				r.multicast(tf.SessionId, userDb.ParseNetSvrInfo(tf.User), clientRoute.Data)
				continue
			}
			//处理订阅
			if clientRoute.Cmd == protocol.RouterSubscribe {
				r.subscribe(tf.SessionId, clientRoute.Data)
				continue
			}
			//处理取消订阅
			if clientRoute.Cmd == protocol.RouterUnsubscribe {
				r.unsubscribe(tf.SessionId, clientRoute.Data)
				continue
			}
		}
	}
}

// 客户端打开连接
func (r *ConnProcessor) connOpen(toWorkerRoute *toWorkerRouter.Router) {
	co := &connOpen.ConnOpen{}
	if err := proto.Unmarshal(toWorkerRoute.Data, co); err != nil {
		logging.Error("Proto unmarshal connOpen.ConnOpen error:%v", err)
		return
	}
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	//构造单播数据
	ret := &singleCast.SingleCast{}
	ret.SessionId = co.SessionId
	ret.Data = NewResponse(protocol.RouterRespConnOpen, map[string]interface{}{"code": 0, "message": fmt.Sprintf("连接网关成功，sessionId: %d", co.SessionId)})
	//将业务数据放到路由上
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	_, _ = r.Write(pt)
}

// 客户端关闭连接
func (r *ConnProcessor) connClose(toWorkerRoute *toWorkerRouter.Router) {
	cls := &connClose.ConnClose{}
	if err := proto.Unmarshal(toWorkerRoute.Data, cls); err != nil {
		logging.Error("Proto unmarshal connClose.ConnClose error:%v", err)
		return
	}
	//解析网关中存储的用户信息
	user := userDb.ParseNetSvrInfo(cls.User)
	if user != nil {
		//用户关闭连接，更新数据库中的session id为0
		userDb.Collect.SetSessionId(user.Id, 0)
	}
}

// 登录
func (r *ConnProcessor) login(sessionId uint32, data string) {
	//解析客户端发来的数据
	login := new(protocol.Login)
	if err := json.Unmarshal(utils.StrToBytes(data), login); err != nil {
		logging.Error("Parse protocol.Login request error: %v", err)
		return
	}
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//要求网关设定登录状态
	toServerRoute.Cmd = toServerRouter.Cmd_SetUserLoginStatus
	//构建一个包含登录状态相关是业务对象
	ret := &setUserLoginStatus.SetUserLoginStatus{}
	ret.SessionId = sessionId
	//查找用户
	user := userDb.Collect.GetUser(login.Username)
	//校验账号密码，判断是否登录成功
	if user != nil && user.Password == login.Password {
		//更新用户的session id
		userDb.Collect.SetSessionId(user.Id, sessionId)
		ret.LoginStatus = true
		//响应给客户端的数据
		ret.Data = NewResponse(protocol.RouterLogin, map[string]interface{}{"code": 0, "message": "登录成功", "data": user.ToClientInfo()})
		//存储到网关的用户信息
		ret.UserInfo = user.ToNetSvrInfo()
	} else {
		ret.LoginStatus = false
		ret.Data = NewResponse(protocol.RouterLogin, map[string]interface{}{"code": 1, "message": "登录失败，账号或密码错误"})
	}
	//将业务对象放到路由上
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	//回写给网关服务器
	_, _ = r.Write(pt)
}

// 单播
func (r *ConnProcessor) singleCast(currentSessionId uint32, currentUser *userDb.NetSvrInfo, data string) {
	//解析客户端发来的数据
	target := new(protocol.SingleCast)
	if err := json.Unmarshal(utils.StrToBytes(data), target); err != nil {
		logging.Error("Parse protocol.SingleCast request error: %v", err)
		return
	}
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//告诉网关要进行一次单播操作
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	//构造网关需要的单播数据
	ret := &singleCast.SingleCast{}
	//查询目标用户的sessionId
	targetSessionId := userDb.Collect.GetSessionId(target.UserId)
	if targetSessionId == 0 {
		//目标用户不存在，返回信息给到发送者
		ret.SessionId = currentSessionId
		ret.Data = NewResponse(protocol.RouterSingleCast, map[string]interface{}{"code": 1, "message": "未找到目标用户"})
	} else {
		//目标用户存在，将信息转发给目标用户
		ret.SessionId = targetSessionId
		msg := map[string]interface{}{"fromUser": currentUser.Name, "message": target.Message}
		ret.Data = NewResponse(protocol.RouterSingleCast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	}
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	_, _ = r.Write(pt)
}

// 组播
func (r *ConnProcessor) multicast(currentSessionId uint32, currentUser *userDb.NetSvrInfo, data string) {
	//解析客户端发来的数据
	target := new(protocol.Multicast)
	if err := json.Unmarshal(utils.StrToBytes(data), target); err != nil {
		logging.Error("Parse protocol.Multicast request error: %v", err)
		return
	}
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//查询目标用户的sessionId
	bitmap := roaring.Bitmap{}
	for _, userId := range target.UserIds {
		sessionId := userDb.Collect.GetSessionId(userId)
		if sessionId > 0 {
			//把找到的在线session id都存储在bitmap对象中
			bitmap.Add(sessionId)
		}
	}
	//没有找到任何目标用户，通知发送方，目标用户不存在
	if bitmap.GetCardinality() == 0 {
		//这里可以采用单播的命令 toServerRouter.Cmd_SingleCast，采用组播是为了测试网关的组播功能是否正常
		//告诉网关要进行一次基于组播操作
		toServerRoute.Cmd = toServerRouter.Cmd_Multicast
		//构造网关需要的组播数据
		ret := &multicast.Multicast{}
		//目标用户不存在，返回信息给到发送者
		ret.SessionIds = []uint32{currentSessionId}
		ret.Data = NewResponse(protocol.RouterMulticast, map[string]interface{}{"code": 1, "message": "未找到目标用户"})
		toServerRoute.Data, _ = proto.Marshal(ret)
		pt, _ := proto.Marshal(toServerRoute)
		_, _ = r.Write(pt)
		return
	}
	//告诉网关要进行一次基于bitmap的组播操作
	toServerRoute.Cmd = toServerRouter.Cmd_MulticastByBitmap
	//构造网关需要的组播数据
	ret := &multicastByBitmap.MulticastByBitmap{}
	//将bitmap对象序列化成字符串
	ret.SessionIdBitmap, _ = bitmap.ToBase64()
	//构造客户端需要的数据
	msg := map[string]interface{}{"fromUser": currentUser.Name, "message": target.Message}
	ret.Data = NewResponse(protocol.RouterMulticast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	_, _ = r.Write(pt)
}

// 广播
func (r *ConnProcessor) broadcast(currentUser *userDb.NetSvrInfo, data string) {
	//解析客户端发来的数据
	target := new(protocol.Broadcast)
	if err := json.Unmarshal(utils.StrToBytes(data), target); err != nil {
		logging.Error("Parse protocol.Broadcast request error: %v", err)
		return
	}
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//告诉网关要进行一次广播操作
	toServerRoute.Cmd = toServerRouter.Cmd_Broadcast
	//构造网关需要的广播数据
	ret := &broadcast.Broadcast{}
	//将bitmap对象序列化成字符串
	ret.Data = utils.StrToBytes(target.Message)
	//构造客户端需要的数据
	msg := map[string]interface{}{"fromUser": currentUser.Name, "message": target.Message}
	ret.Data = NewResponse(protocol.RouterBroadcast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	_, _ = r.Write(pt)
}

// 处理客户的订阅请求
func (r *ConnProcessor) subscribe(currentSessionId uint32, data string) {
	//解析客户端发来的数据
	target := new(protocol.Subscribe)
	if err := json.Unmarshal(utils.StrToBytes(data), target); err != nil {
		logging.Error("Parse protocol.Subscribe request error: %v", err)
		return
	}
	if len(target.Topics) == 0 {
		return
	}
	//提交订阅信息到网关
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//告诉网关要进行一次订阅操作
	toServerRoute.Cmd = toServerRouter.Cmd_Subscribe
	//构造网关需要的订阅数据
	ret := &subscribe.Subscribe{}
	ret.SessionId = currentSessionId
	ret.Topics = target.Topics
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	_, _ = r.Write(pt)
	//查询该用户的订阅信息
	req := &reqSessionInfo.ReqSessionInfo{}
	req.SessionId = currentSessionId
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.Cmd = protocol.RouterSubscribe
	toServerRoute.Cmd = toServerRouter.Cmd_ReqSessionInfo
	toServerRoute.Data, _ = proto.Marshal(req)
	pt, _ = proto.Marshal(toServerRoute)
	_, _ = r.Write(pt)
}

// 处理客户的取消订阅请求
func (r *ConnProcessor) unsubscribe(currentSessionId uint32, data string) {
	//解析客户端发来的数据
	target := new(protocol.Unsubscribe)
	if err := json.Unmarshal(utils.StrToBytes(data), target); err != nil {
		logging.Error("Parse protocol.Unsubscribe request error: %v", err)
		return
	}
	if len(target.Topics) == 0 {
		return
	}
	//提交取消订阅信息到网关
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//告诉网关要进行一次取消订阅操作
	toServerRoute.Cmd = toServerRouter.Cmd_Unsubscribe
	//构造网关需要的取消订阅数据
	ret := &unsubscribe.Unsubscribe{}
	ret.SessionId = currentSessionId
	ret.Topics = target.Topics
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	_, _ = r.Write(pt)
	//查询该用户的订阅信息
	req := &reqSessionInfo.ReqSessionInfo{}
	req.SessionId = currentSessionId
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.Cmd = protocol.RouterUnsubscribe
	toServerRoute.Cmd = toServerRouter.Cmd_ReqSessionInfo
	toServerRoute.Data, _ = proto.Marshal(req)
	pt, _ = proto.Marshal(toServerRoute)
	_, _ = r.Write(pt)
}

// 处理网关发送的某个用户的session信息
func (r *ConnProcessor) respSessionInfo(toWorkerRoute *toWorkerRouter.Router) {
	sessionInfo := &respSessionInfo.RespSessionInfo{}
	if err := proto.Unmarshal(toWorkerRoute.Data, sessionInfo); err != nil {
		logging.Error("Proto unmarshal respSessionInfo.RespSessionInfo error:%v", err)
		return
	}
	//根据自定义的cmd判断到底谁发起的请求网关session info信息
	if sessionInfo.Cmd == protocol.RouterSubscribe {
		r.respSessionInfoBySubscribe(sessionInfo)
	} else if sessionInfo.Cmd == protocol.RouterUnsubscribe {
		r.respSessionInfoByUnsubscribe(sessionInfo)
	}
}

func (r *ConnProcessor) respSessionInfoBySubscribe(sessionInfo *respSessionInfo.RespSessionInfo) {
	if len(sessionInfo.Topics) == 0 {
		return
	}
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//告诉网关要进行一次单播操作
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	//构造网关需要的单播数据
	ret := &singleCast.SingleCast{}
	ret.SessionId = sessionInfo.SessionId
	msg := map[string]interface{}{"topics": sessionInfo.Topics}
	ret.Data = NewResponse(protocol.RouterSubscribe, map[string]interface{}{"code": 0, "message": "订阅成功", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	_, _ = r.Write(pt)
}

func (r *ConnProcessor) respSessionInfoByUnsubscribe(sessionInfo *respSessionInfo.RespSessionInfo) {
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//告诉网关要进行一次单播操作
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	//构造网关需要的单播数据
	ret := &singleCast.SingleCast{}
	ret.SessionId = sessionInfo.SessionId
	msg := map[string]interface{}{"topics": sessionInfo.Topics}
	ret.Data = NewResponse(protocol.RouterUnsubscribe, map[string]interface{}{"code": 0, "message": "取消订阅成功", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	_, _ = r.Write(pt)
}

func NewResponse(cmd int, data interface{}) []byte {
	tmp := map[string]interface{}{"cmd": cmd, "data": data}
	ret, _ := json.Marshal(tmp)
	return ret
}
