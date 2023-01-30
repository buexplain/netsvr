package connProcessor

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/RoaringBitmap/roaring"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/internal/protocol/reCtx"
	"netsvr/internal/protocol/toServer/broadcast"
	"netsvr/internal/protocol/toServer/multicast"
	"netsvr/internal/protocol/toServer/multicastByBitmap"
	"netsvr/internal/protocol/toServer/publish"
	"netsvr/internal/protocol/toServer/registerWorker"
	"netsvr/internal/protocol/toServer/reqNetSvrStatus"
	"netsvr/internal/protocol/toServer/reqSessionInfo"
	"netsvr/internal/protocol/toServer/reqTopicsConnCount"
	"netsvr/internal/protocol/toServer/reqTopicsSessionId"
	"netsvr/internal/protocol/toServer/reqTotalSessionId"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/internal/protocol/toServer/setSessionUser"
	"netsvr/internal/protocol/toServer/setUserLoginStatus"
	"netsvr/internal/protocol/toServer/singleCast"
	"netsvr/internal/protocol/toServer/subscribe"
	"netsvr/internal/protocol/toServer/unsubscribe"
	"netsvr/internal/protocol/toWorker/connClose"
	"netsvr/internal/protocol/toWorker/connOpen"
	"netsvr/internal/protocol/toWorker/respNetSvrStatus"
	"netsvr/internal/protocol/toWorker/respSessionInfo"
	"netsvr/internal/protocol/toWorker/respTopicsConnCount"
	"netsvr/internal/protocol/toWorker/respTopicsSessionId"
	"netsvr/internal/protocol/toWorker/respTotalSessionId"
	toWorkerRouter "netsvr/internal/protocol/toWorker/router"
	"netsvr/internal/protocol/toWorker/transfer"
	"netsvr/internal/worker/heartbeat"
	"netsvr/pkg/quit"
	"netsvr/pkg/utils"
	"netsvr/test/worker/protocol"
	"netsvr/test/worker/userDb"
	"runtime/debug"
	"strconv"
	"time"
)

type ConnProcessor struct {
	conn     net.Conn
	sendCh   chan []byte
	closeCh  chan struct{}
	workerId int
}

func NewConnProcessor(conn net.Conn, workerId int) *ConnProcessor {
	tmp := &ConnProcessor{conn: conn, sendCh: make(chan []byte, 10), closeCh: make(chan struct{}), workerId: workerId}
	return tmp
}

func (r *ConnProcessor) LoopHeartbeat() {
	t := time.NewTicker(time.Duration(35) * time.Second)
	defer func() {
		if err := recover(); err != nil {
			logging.Error("Worker heartbeat coroutine is closed, error: %vn%s", err, debug.Stack())
		} else {
			logging.Debug("Worker heartbeat coroutine is closed")
		}
		t.Stop()
		quit.Wg.Done()
	}()
	for {
		select {
		case <-r.closeCh:
			return
		case <-quit.Ctx.Done():
			return
		case <-t.C:
			r.Send(heartbeat.PingMessage)
		}
	}
}

func (r *ConnProcessor) GetWorkerId() int {
	return r.workerId
}

func (r *ConnProcessor) Close() {
	select {
	case <-r.closeCh:
		return
	default:
		close(r.closeCh)
		_ = r.conn.Close()
		quit.Execute("Worker conn is closed")
	}
}

func (r *ConnProcessor) LoopSend() {
	defer func() {
		if err := recover(); err != nil {
			logging.Error("Worker send coroutine is closed, error: %vn%s", err, debug.Stack())
		} else {
			logging.Debug("Worker send coroutine is closed")
		}
		quit.Wg.Done()
	}()
	for {
		select {
		case <-r.closeCh:
			return
		case <-quit.Ctx.Done():
			//进程即将停止，处理通道中剩余数据，尽量保证用户消息转发到工作进程
			r.done()
			return
		case data := <-r.sendCh:
			r.write(data)
		}
	}
}

func (r *ConnProcessor) done() {
	//处理通道中的剩余数据
	for {
		for i := len(r.sendCh); i > 0; i-- {
			v := <-r.sendCh
			r.write(v)
		}
		if len(r.sendCh) == 0 {
			break
		}
	}
	//再次处理通道中的剩余数据，直到超时退出
	for {
		select {
		case v := <-r.sendCh:
			r.write(v)
		case <-time.After(1 * time.Second):
			return
		}
	}
}

func (r *ConnProcessor) write(data []byte) {
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

func (r *ConnProcessor) Send(data []byte) {
	select {
	case <-r.closeCh:
		return
	case <-quit.Ctx.Done():
		return
	default:
		r.sendCh <- data
		return
	}
}

func (r *ConnProcessor) LoopRead() {
	defer func() {
		if err := recover(); err != nil {
			logging.Error("Worker read coroutine is closed, error: %vn%s", err, debug.Stack())
		} else {
			logging.Debug("Worker read coroutine is closed")
		}
	}()
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
			r.Send(heartbeat.PongMessage)
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
		//网关返回了自己的状态信息
		if toWorkerRoute.Cmd == toWorkerRouter.Cmd_RespNetSvrStatus {
			r.respNetSvrStatus(toWorkerRoute)
			continue
		}
		//网关返回了所有在线的session id
		if toWorkerRoute.Cmd == toWorkerRouter.Cmd_RespTotalSessionId {
			r.respTotalSessionId(toWorkerRoute)
			continue
		}
		//网关返回了主题的连接数
		if toWorkerRoute.Cmd == toWorkerRouter.Cmd_RespTopicsConnCount {
			r.respTopicsConnCount(toWorkerRoute)
			continue
		}
		//网关返回了主题的连接session id
		if toWorkerRoute.Cmd == toWorkerRouter.Cmd_RespTopicsSessionId {
			r.respTopicsSessionId(toWorkerRoute)
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
			//获取网关的信息
			if clientRoute.Cmd == protocol.RouterNetSvrStatus {
				r.netSvrStatus(tf.SessionId)
				continue
			}
			//获取网关所有在线的session id
			if clientRoute.Cmd == protocol.RouterTotalSessionId {
				r.totalSessionId(tf.SessionId)
				continue
			}
			//登录
			if clientRoute.Cmd == protocol.RouterLogin {
				r.login(tf.SessionId, clientRoute.Data)
				continue
			}
			//退出登录
			if clientRoute.Cmd == protocol.RouterLogout {
				r.logout(tf.SessionId, userDb.ParseNetSvrInfo(tf.User))
				continue
			}
			//更新网关中的用户信息
			if clientRoute.Cmd == protocol.RouterUpdateSessionUserInfo {
				r.updateSessionUserInfo(tf.SessionId, userDb.ParseNetSvrInfo(tf.User))
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
			//处理广播
			if clientRoute.Cmd == protocol.RouterBroadcast {
				r.broadcast(userDb.ParseNetSvrInfo(tf.User), clientRoute.Data)
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
			//获取网关中的某几个主题的连接数
			if clientRoute.Cmd == protocol.RouterTopicsConnCount {
				r.topicsConnCount(tf.SessionId, clientRoute.Data)
				continue
			}
			//获取网关中的某几个主题的连接session id
			if clientRoute.Cmd == protocol.RouterTopicsSessionId {
				r.topicsSessionId(tf.SessionId, clientRoute.Data)
				continue
			}
			//处理发布
			if clientRoute.Cmd == protocol.RouterPublish {
				r.publish(userDb.ParseNetSvrInfo(tf.User), clientRoute.Data)
				continue
			}
		}

		logging.Error("Unknown cmd: %d", toWorkerRoute.Cmd)
	}
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
	pt, _ := proto.Marshal(toServerRoute)
	r.Send(pt)
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
	r.Send(pt)
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

// 获取网关的信息
func (r *ConnProcessor) netSvrStatus(currentSessionId uint32) {
	req := &reqNetSvrStatus.ReqNetSvrStatus{ReCtx: &reCtx.ReCtx{}}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = protocol.RouterNetSvrStatus
	//将session id存储到请求上下文中去，当网关返回数据的时候用的上
	req.ReCtx.Data = utils.StrToBytes(strconv.FormatInt(int64(currentSessionId), 10))
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_ReqNetSvrStatus
	toServerRoute.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(toServerRoute)
	r.Send(pt)
}

// 处理网关发送的网关状态信息
func (r *ConnProcessor) respNetSvrStatus(toWorkerRoute *toWorkerRouter.Router) {
	netSvrStatus := &respNetSvrStatus.RespNetSvrStatus{}
	if err := proto.Unmarshal(toWorkerRoute.Data, netSvrStatus); err != nil {
		logging.Error("Proto unmarshal respNetSvrStatus.RespNetSvrStatus error:%v", err)
		return
	}
	//不是客户端请求网关数据，则忽略
	if netSvrStatus.ReCtx.Cmd != protocol.RouterNetSvrStatus {
		return
	}
	//解析请求上下文中存储的session id
	targetSessionId, _ := strconv.ParseInt(string(netSvrStatus.ReCtx.Data), 10, 64)
	if targetSessionId == 0 {
		return
	}
	//将网关的信息单播给客户端
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	ret := &singleCast.SingleCast{}
	ret.SessionId = uint32(targetSessionId)
	msg := map[string]interface{}{
		"customerConnCount":     netSvrStatus.CustomerConnCount,
		"topicCount":            netSvrStatus.TopicCount,
		"catapultWaitSendCount": netSvrStatus.CatapultWaitSendCount,
		"catapultConsumer":      netSvrStatus.CatapultConsumer,
		"catapultChanCap":       netSvrStatus.CatapultChanCap,
	}
	ret.Data = NewResponse(protocol.RouterNetSvrStatus, map[string]interface{}{"code": 0, "message": "获取网关状态信息成功", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	r.Send(pt)
}

// 获取网关所有在线的session id
func (r *ConnProcessor) totalSessionId(currentSessionId uint32) {
	req := &reqTotalSessionId.ReqTotalSessionId{ReCtx: &reCtx.ReCtx{}}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = protocol.RouterTotalSessionId
	//将session id存储到请求上下文中去，当网关返回数据的时候用的上
	req.ReCtx.Data = utils.StrToBytes(strconv.FormatInt(int64(currentSessionId), 10))
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_ReqTotalSessionId
	toServerRoute.Data, _ = proto.Marshal(req)
	pt, _ := proto.Marshal(toServerRoute)
	r.Send(pt)
}

// 处理网关发送的所有在线的session id
func (r *ConnProcessor) respTotalSessionId(toWorkerRoute *toWorkerRouter.Router) {
	totalSessionId := &respTotalSessionId.RespTotalSessionId{}
	if err := proto.Unmarshal(toWorkerRoute.Data, totalSessionId); err != nil {
		logging.Error("Proto unmarshal respTotalSessionId.RespTotalSessionId error:%v", err)
		return
	}
	//不是客户端请求网关数据，则忽略
	if totalSessionId.ReCtx.Cmd != protocol.RouterTotalSessionId {
		return
	}
	//解析请求上下文中存储的session id
	targetSessionId, _ := strconv.ParseInt(string(totalSessionId.ReCtx.Data), 10, 64)
	if targetSessionId == 0 {
		return
	}
	//将结果单播给客户端
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	ret := &singleCast.SingleCast{}
	ret.SessionId = uint32(targetSessionId)
	msg := map[string]interface{}{
		"totalSessionId": totalSessionId.Bitmap,
	}
	ret.Data = NewResponse(protocol.RouterTotalSessionId, map[string]interface{}{"code": 0, "message": "获取网关所有在线的session id成功", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	r.Send(pt)
}

// 登录
func (r *ConnProcessor) login(currentSessionId uint32, data string) {
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
	ret.SessionId = currentSessionId
	//查找用户
	user := userDb.Collect.GetUser(login.Username)
	//校验账号密码，判断是否登录成功
	if user != nil && user.Password == login.Password {
		//更新用户的session id
		userDb.Collect.SetSessionId(user.Id, currentSessionId)
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
	r.Send(pt)
}

// 退出登录
func (r *ConnProcessor) logout(currentSessionId uint32, currentUser *userDb.NetSvrInfo) {
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//要求网关设定登录状态
	toServerRoute.Cmd = toServerRouter.Cmd_SetUserLoginStatus
	//构建一个包含登录状态相关是业务对象
	ret := &setUserLoginStatus.SetUserLoginStatus{}
	ret.SessionId = currentSessionId
	ret.LoginStatus = false
	ret.Data = NewResponse(protocol.RouterLogout, map[string]interface{}{"code": 0, "message": "退出登录成功"})
	//更新用户的信息
	if currentUser != nil {
		user := userDb.Collect.GetUser(currentUser.Name)
		if user != nil {
			//更新用户的session id
			userDb.Collect.SetSessionId(user.Id, 0)
		}
	}
	//将业务对象放到路由上
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	//回写给网关服务器
	r.Send(pt)
}

// 更新网关中的用户信息
func (r *ConnProcessor) updateSessionUserInfo(currentSessionId uint32, currentUser *userDb.NetSvrInfo) {
	currentUser.LastUpdateTime = time.Now()
	msg := map[string]interface{}{"netSvrInfo": currentUser}
	ret := &setSessionUser.SetSessionUser{}
	ret.SessionId = currentSessionId
	ret.UserInfo = currentUser.Encode()
	ret.Data = NewResponse(protocol.RouterUpdateSessionUserInfo, map[string]interface{}{"code": 0, "message": "更新网关中的用户信息成功", "data": msg})
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_SetSessionUser
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	r.Send(pt)
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
	r.Send(pt)
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
		r.Send(pt)
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
	r.Send(pt)
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
	//构造客户端需要的数据
	msg := map[string]interface{}{"fromUser": currentUser.Name, "message": target.Message}
	ret.Data = NewResponse(protocol.RouterBroadcast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	r.Send(pt)
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
	r.Send(pt)
	//查询该用户的订阅信息
	req := &reqSessionInfo.ReqSessionInfo{ReCtx: &reCtx.ReCtx{}}
	req.SessionId = currentSessionId
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = protocol.RouterSubscribe
	toServerRoute.Cmd = toServerRouter.Cmd_ReqSessionInfo
	toServerRoute.Data, _ = proto.Marshal(req)
	pt, _ = proto.Marshal(toServerRoute)
	r.Send(pt)
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
	r.Send(pt)
	//查询该用户的订阅信息
	req := &reqSessionInfo.ReqSessionInfo{ReCtx: &reCtx.ReCtx{}}
	req.SessionId = currentSessionId
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = protocol.RouterUnsubscribe
	toServerRoute.Cmd = toServerRouter.Cmd_ReqSessionInfo
	toServerRoute.Data, _ = proto.Marshal(req)
	pt, _ = proto.Marshal(toServerRoute)
	r.Send(pt)
}

// 处理网关发送的某个用户的session信息
func (r *ConnProcessor) respSessionInfo(toWorkerRoute *toWorkerRouter.Router) {
	sessionInfo := &respSessionInfo.RespSessionInfo{}
	if err := proto.Unmarshal(toWorkerRoute.Data, sessionInfo); err != nil {
		logging.Error("Proto unmarshal respSessionInfo.RespSessionInfo error:%v", err)
		return
	}
	//根据自定义的cmd判断到底谁发起的请求网关session info信息
	if sessionInfo.ReCtx.Cmd == protocol.RouterSubscribe {
		r.respSessionInfoBySubscribe(sessionInfo)
	} else if sessionInfo.ReCtx.Cmd == protocol.RouterUnsubscribe {
		r.respSessionInfoByUnsubscribe(sessionInfo)
	}
}

// 用户订阅后拉取的用户信息
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
	r.Send(pt)
}

// 用户取消订阅后拉取的用户信息
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
	r.Send(pt)
}

// 获取网关中的某几个主题的连接数
func (r *ConnProcessor) topicsConnCount(currentSessionId uint32, data string) {
	//解析客户端发来的数据
	target := new(protocol.TopicsConnCount)
	if err := json.Unmarshal(utils.StrToBytes(data), target); err != nil {
		logging.Error("Parse protocol.TopicsConnCount request error: %v", err)
		return
	}
	req := reqTopicsConnCount.ReqTopicsConnCount{ReCtx: &reCtx.ReCtx{}}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = protocol.RouterTopicsConnCount
	//将session id存储到请求上下文中去，当网关返回数据的时候用的上
	req.ReCtx.Data = utils.StrToBytes(strconv.FormatInt(int64(currentSessionId), 10))
	req.Topics = target.Topics
	req.GetAll = target.GetAll
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_ReqTopicsConnCount
	toServerRoute.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(toServerRoute)
	r.Send(pt)
}

// 处理网关发送的主题的连接数
func (r *ConnProcessor) respTopicsConnCount(toWorkerRoute *toWorkerRouter.Router) {
	topicsConnCount := &respTopicsConnCount.RespTopicsConnCount{}
	if err := proto.Unmarshal(toWorkerRoute.Data, topicsConnCount); err != nil {
		logging.Error("Proto unmarshal respTopicsConnCount.RespTopicsConnCount error:%v", err)
		return
	}
	//不是客户端请求网关数据，则忽略
	if topicsConnCount.ReCtx.Cmd != protocol.RouterTopicsConnCount {
		return
	}
	//解析请求上下文中存储的session id
	targetSessionId, _ := strconv.ParseInt(string(topicsConnCount.ReCtx.Data), 10, 64)
	if targetSessionId == 0 {
		return
	}
	//将结果单播给客户端
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	ret := &singleCast.SingleCast{}
	ret.SessionId = uint32(targetSessionId)
	ret.Data = NewResponse(protocol.RouterTopicsConnCount, map[string]interface{}{"code": 0, "message": "获取网关中主题的连接数成功", "data": topicsConnCount.Items})
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	r.Send(pt)
}

// 获取网关中的某几个主题的连接session id
func (r *ConnProcessor) topicsSessionId(currentSessionId uint32, data string) {
	//解析客户端发来的数据
	target := new(protocol.TopicsSessionId)
	if err := json.Unmarshal(utils.StrToBytes(data), target); err != nil {
		logging.Error("Parse protocol.TopicsSessionId request error: %v", err)
		return
	}
	req := reqTopicsSessionId.ReqTopicsSessionId{ReCtx: &reCtx.ReCtx{}}
	//将发起请求的原因给到网关，网关会在响应的数据里面原样返回
	req.ReCtx.Cmd = protocol.RouterTopicsSessionId
	//将session id存储到请求上下文中去，当网关返回数据的时候用的上
	req.ReCtx.Data = utils.StrToBytes(strconv.FormatInt(int64(currentSessionId), 10))
	req.Topics = target.Topics
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_ReqTopicsSessionId
	toServerRoute.Data, _ = proto.Marshal(&req)
	pt, _ := proto.Marshal(toServerRoute)
	r.Send(pt)
}

// 处理网关发送的主题的连接session id
func (r *ConnProcessor) respTopicsSessionId(toWorkerRoute *toWorkerRouter.Router) {
	topicsSessionId := &respTopicsSessionId.RespTopicsSessionId{}
	if err := proto.Unmarshal(toWorkerRoute.Data, topicsSessionId); err != nil {
		logging.Error("Proto unmarshal respTopicsSessionId.RespTopicsSessionId error:%v", err)
		return
	}
	//不是客户端请求网关数据，则忽略
	if topicsSessionId.ReCtx.Cmd != protocol.RouterTopicsSessionId {
		return
	}
	//解析请求上下文中存储的session id
	targetSessionId, _ := strconv.ParseInt(string(topicsSessionId.ReCtx.Data), 10, 64)
	if targetSessionId == 0 {
		return
	}
	//将结果单播给客户端
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	ret := &singleCast.SingleCast{}
	ret.SessionId = uint32(targetSessionId)
	ret.Data = NewResponse(protocol.RouterTopicsSessionId, map[string]interface{}{"code": 0, "message": "获取网关中的主题的连接session id成功", "data": topicsSessionId.Items})
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	r.Send(pt)
}

// 处理客户的发布请求
func (r *ConnProcessor) publish(currentUser *userDb.NetSvrInfo, data string) {
	//解析客户端发来的数据
	target := new(protocol.Publish)
	if err := json.Unmarshal(utils.StrToBytes(data), target); err != nil {
		logging.Error("Parse protocol.Publish request error: %v", err)
		return
	}
	msg := map[string]interface{}{"fromUser": currentUser.Name, "message": target.Message}
	ret := &publish.Publish{}
	ret.Topic = target.Topic
	ret.Data = NewResponse(protocol.RouterPublish, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_Publish
	toServerRoute.Data, _ = proto.Marshal(ret)
	pt, _ := proto.Marshal(toServerRoute)
	r.Send(pt)
}

func NewResponse(cmd int, data interface{}) []byte {
	tmp := map[string]interface{}{"cmd": cmd, "data": data}
	ret, _ := json.Marshal(tmp)
	return ret
}
