package manager

import (
	"bytes"
	"encoding/binary"
	"github.com/buexplain/netsvr/configs"
	"github.com/buexplain/netsvr/internal/customer/business"
	customerManager "github.com/buexplain/netsvr/internal/customer/manager"
	"github.com/buexplain/netsvr/internal/customer/session"
	"github.com/buexplain/netsvr/internal/protocol/toServer/broadcast"
	"github.com/buexplain/netsvr/internal/protocol/toServer/multicast"
	"github.com/buexplain/netsvr/internal/protocol/toServer/multicastByBitmap"
	"github.com/buexplain/netsvr/internal/protocol/toServer/publish"
	"github.com/buexplain/netsvr/internal/protocol/toServer/registerWorker"
	"github.com/buexplain/netsvr/internal/protocol/toServer/reqSessionInfo"
	"github.com/buexplain/netsvr/internal/protocol/toServer/reqTopicsSessionId"
	"github.com/buexplain/netsvr/internal/protocol/toServer/respNetSvrStatus"
	toServerRouter "github.com/buexplain/netsvr/internal/protocol/toServer/router"
	"github.com/buexplain/netsvr/internal/protocol/toServer/setSessionUser"
	"github.com/buexplain/netsvr/internal/protocol/toServer/setUserLoginStatus"
	"github.com/buexplain/netsvr/internal/protocol/toServer/singleCast"
	"github.com/buexplain/netsvr/internal/protocol/toServer/subscribe"
	"github.com/buexplain/netsvr/internal/protocol/toServer/unsubscribe"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/respSessionInfo"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/respTopicsSessionId"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/respTotalSessionId"
	toWorkerRouter "github.com/buexplain/netsvr/internal/protocol/toWorker/router"
	"github.com/buexplain/netsvr/internal/worker/heartbeat"
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/buexplain/netsvr/pkg/timecache"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"sync/atomic"
	"time"
)

type ConnProcessor struct {
	conn    net.Conn
	writeCh chan []byte
	//当前连接最后发送消息的时间
	lastActiveTime *int64
	closeCh        chan struct{}
	//当前连接的编号id
	workerId int
}

func NewConnProcessor(conn net.Conn) *ConnProcessor {
	var lastActiveTime int64 = 0
	tmp := &ConnProcessor{
		conn:           conn,
		writeCh:        make(chan []byte, 100),
		lastActiveTime: &lastActiveTime,
		closeCh:        make(chan struct{}),
		workerId:       0,
	}
	return tmp
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
		case <-time.After(3 * time.Second):
			return
		}
	}
}

func (r *ConnProcessor) execute(data []byte) {
	if err := binary.Write(r.conn, binary.BigEndian, uint32(len(data))); err == nil {
		if _, err = r.conn.Write(data); err != nil {
			r.close()
			logging.Error("Worker write error: %#v", err)
		}
	} else {
		r.close()
		logging.Error("Worker write error: %#v", err)
	}
}
func (r *ConnProcessor) close() {
	select {
	case <-r.closeCh:
	default:
		close(r.closeCh)
		_ = r.conn.Close()
	}
}

func (r *ConnProcessor) Send() {
	defer func() {
		//写协程退出，直接关闭连接
		r.close()
		quit.Wg.Done()
		//收集异常退出的信息
		if err := recover(); err != nil {
			logging.Error("Abnormal exit a worker, error: %#v", err)
		} else {
			logging.Debug("Worker read coroutine is close")
		}
	}()
	for {
		select {
		case _ = <-r.closeCh:
			//连接已被关闭，丢弃所有的数据
			close(r.writeCh)
			return
		case <-quit.Ctx.Done():
			//进程即将停止，处理通道中剩余数据，尽量保证用户消息转发到工作进程
			r.done()
			return
		case data := <-r.writeCh:
			r.execute(data)
		}
	}
}

func (r *ConnProcessor) Write(data []byte) (n int, err error) {
	select {
	case <-r.closeCh:
		//工作进程即将关闭，停止转发消息到工作进程
		return 0, nil
	case <-quit.Ctx.Done():
		//进程即将关闭，停止转发消息到工作进程
		return 0, nil
	default:
		r.writeCh <- data
		return len(data), nil
	}
}

func (r *ConnProcessor) Read() {
	heartbeatNode := heartbeat.Timer.ScheduleFunc(time.Duration(configs.Config.WorkerHeartbeatIntervalSecond)*time.Second, func() {
		if timecache.Unix()-atomic.LoadInt64(r.lastActiveTime) < configs.Config.WorkerHeartbeatIntervalSecond {
			//还在活跃期内，不做处理
			return
		}
		//超过活跃期，服务端主动发送心跳
		_, _ = r.Write(heartbeat.PingMessage)
	})
	defer func() {
		//注销掉工作进程的id
		if r.workerId > 0 {
			Manager.Del(r.workerId, r)
			logging.Info("Unregister a worker by id: %d", r.workerId)
		}
		//停止心跳检查
		heartbeatNode.Stop()
		//收集异常退出的信息
		if err := recover(); err != nil {
			logging.Error("Abnormal exit a worker of id: %d, error: %#v", r.workerId, err)
		}
	}()
	dataLenBuf := make([]byte, 4)
	for {
		dataLenBuf[0] = 0
		dataLenBuf[1] = 0
		dataLenBuf[2] = 0
		dataLenBuf[3] = 0
		//获取前4个字节，确定数据包长度
		if _, err := io.ReadFull(r.conn, dataLenBuf); err != nil {
			//读失败了，直接干掉这个连接，让客户端重新连接进来，因为缓冲区的tcp流已经脏了，程序无法拆包
			r.close()
			break
		}
		if len(dataLenBuf) != 4 {
			continue
		}
		//这里采用大端序，小于2个字节，则说明业务命令都没有
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		if dataLen < 2 || dataLen > configs.Config.WorkerReadPackLimit {
			//工作进程不按套路出牌，不老实，直接关闭它
			r.close()
			logging.Error("Worker data is too large: %d", dataLen)
			break
		}
		//获取数据包
		dataBuf := make([]byte, dataLen)
		if _, err := io.ReadFull(r.conn, dataBuf); err != nil {
			r.close()
			logging.Error("Worker read error: %#v", err)
			break
		}
		//更新客户端的最后活跃时间
		atomic.StoreInt64(r.lastActiveTime, timecache.Unix())
		//客户端发来心跳
		if bytes.Equal(heartbeat.PingMessage, dataBuf) {
			//响应客户端的心跳
			_, _ = r.Write(heartbeat.PongMessage)
			continue
		}
		//客户端响应心跳
		if bytes.Equal(heartbeat.PongMessage, dataBuf) {
			continue
		}
		toServerRoute := &toServerRouter.Router{}
		if err := proto.Unmarshal(dataBuf, toServerRoute); err != nil {
			logging.Error("Proto unmarshal toServerRouter.Router error: %#v", err)
			continue
		}
		logging.Debug("Receive worker command: %d", toServerRoute.Cmd)
		switch toServerRoute.Cmd {
		case toServerRouter.Cmd_RegisterWorker:
			//注册工作进程
			if r.registerWorker(toServerRoute) == false {
				//注册失败，直接关闭连接，因为注册失败是没有给工作进程返回失败提示的，而且工作进程是必须注册成功的，所以直接关闭是上策
				r.close()
				return
			}
		case toServerRouter.Cmd_SetUserLoginStatus:
			//变更用户的登录状态
			r.setUserLoginStatus(toServerRoute)
			break
		case toServerRouter.Cmd_SingleCast:
			//单播
			r.singleCast(toServerRoute)
			break
		case toServerRouter.Cmd_Broadcast:
			//广播
			r.broadcast(toServerRoute)
			break
		case toServerRouter.Cmd_Multicast:
			//组播
			r.multicast(toServerRoute)
			break
		case toServerRouter.Cmd_MulticastByBitmap:
			//根据包含session id的bitmap进行组播
			r.multicastByBitmap(toServerRoute)
			break
		case toServerRouter.Cmd_Subscribe:
			//订阅
			r.subscribe(toServerRoute)
			break
		case toServerRouter.Cmd_Unsubscribe:
			//取消订阅
			r.unsubscribe(toServerRoute)
			break
		case toServerRouter.Cmd_Publish:
			//发布消息
			r.publish(toServerRoute)
			break
		case toServerRouter.Cmd_ReqTotalSessionId:
			//获取网关中全部的session id
			r.reqTotalSessionId()
			break
		case toServerRouter.Cmd_ReqTopicsSessionId:
			//获取网关中的某几个主题的session id
			r.reqTopicsSessionId(toServerRoute)
			break
		case toServerRouter.Cmd_ReqSessionInfo:
			//根据session id获取网关中的用户信息
			r.reqSessionInfo(toServerRoute)
			break
		case toServerRouter.Cmd_SetSessionUser:
			//设置网关的session面存储的用户信息
			r.setSessionUser(toServerRoute)
			break
		case toServerRouter.Cmd_ReqNetSvrStatus:
			//返回网关的状态
			r.reqNetSvrStatus()
			break
		default:
			//工作进程搞错了指令，直接关闭连接，让工作进程明白，不能瞎传，代码一定要通过测试
			r.close()
			logging.Error("Unknown toServerRoute.Cmd: %d", toServerRoute.Cmd)
			return
		}
	}
}

// 注册工作进程
func (r *ConnProcessor) registerWorker(toServerRoute *toServerRouter.Router) bool {
	data := &registerWorker.RegisterWorker{}
	if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
		logging.Error("Proto unmarshal registerWorker.RegisterWorker error: %#v", err)
		return false
	}
	if MinWorkerId > data.Id || data.Id > MaxWorkerId {
		logging.Error("Wrong work id %d not in range: %d ~ %d", data.Id, MinWorkerId, MaxWorkerId)
		return false
	}
	r.workerId = int(data.Id)
	Manager.Set(r.workerId, r)
	if data.ProcessConnClose {
		SetProcessConnCloseWorkerId(data.Id)
	}
	if data.ProcessConnOpen {
		SetProcessConnOpenWorkerId(data.Id)
	}
	logging.Info("Register a worker by id: %d", r.workerId)
	return true
}

// 变更用户的登录状态
func (r *ConnProcessor) setUserLoginStatus(toServerRoute *toServerRouter.Router) {
	data := &setUserLoginStatus.SetUserLoginStatus{}
	if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
		logging.Error("Proto unmarshal setUserLoginStatus.SetUserLoginStatus error: %#v", err)
		return
	}
	business.SetUserLoginStatus(data)
}

// 单播
func (r *ConnProcessor) singleCast(toServerRoute *toServerRouter.Router) {
	data := &singleCast.SingleCast{}
	if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
		logging.Error("Proto unmarshal singleCast.SingleCast error: %#v", err)
		return
	}
	business.SingleCast(data)
}

// 广播
func (r *ConnProcessor) broadcast(toServerRoute *toServerRouter.Router) {
	data := &broadcast.Broadcast{}
	if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
		logging.Error("Proto unmarshal broadcast.Broadcast error: %#v", err)
		return
	}
	business.Broadcast(data)
}

// 组播
func (r *ConnProcessor) multicast(toServerRoute *toServerRouter.Router) {
	data := &multicast.Multicast{}
	if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
		logging.Error("Proto unmarshal multicast.Multicast error: %#v", err)
		return
	}
	business.Multicast(data)
}

// 根据包含session id的bitmap进行组播
func (r *ConnProcessor) multicastByBitmap(toServerRoute *toServerRouter.Router) {
	data := &multicastByBitmap.MulticastByBitmap{}
	if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
		logging.Error("Proto unmarshal multicastByBitmap.MulticastByBitmap error: %#v", err)
		return
	}
	business.MulticastByBitmap(data)
}

// 订阅
func (r *ConnProcessor) subscribe(toServerRoute *toServerRouter.Router) {
	data := &subscribe.Subscribe{}
	if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
		logging.Error("Proto unmarshal subscribe.Subscribe error: %#v", err)
		return
	}
	business.Subscribe(data)
}

// 取消订阅
func (r *ConnProcessor) unsubscribe(toServerRoute *toServerRouter.Router) {
	data := &unsubscribe.Unsubscribe{}
	if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
		logging.Error("Proto unmarshal unsubscribe.Unsubscribe error: %#v", err)
		return
	}
	business.Unsubscribe(data)
}

// 发布
func (r *ConnProcessor) publish(toServerRoute *toServerRouter.Router) {
	data := &publish.Publish{}
	if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
		logging.Error("Proto unmarshal publish.Publish error: %#v", err)
		return
	}
	business.Publish(data)
}

// 获取网关中全部的session id
func (r *ConnProcessor) reqTotalSessionId() {
	bitmap := session.Id.GetAllocated()
	data := &respTotalSessionId.RespTotalSessionId{}
	data.Bitmap, _ = bitmap.ToBase64()
	route := &toWorkerRouter.Router{}
	route.Cmd = toWorkerRouter.Cmd_RespTopicsSessionId
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	_, _ = r.Write(b)
}

// 获取网关中的某几个主题的session id
func (r *ConnProcessor) reqTopicsSessionId(toServerRoute *toServerRouter.Router) {
	req := &reqTopicsSessionId.ReqTopicsSessionId{}
	if err := proto.Unmarshal(toServerRoute.Data, req); err != nil {
		logging.Error("Proto unmarshal reqTopicsSessionId.ReqTopicsSessionId error: %#v", err)
		return
	}
	bitmap := session.Topics.Gets(req.Topics)
	data := &respTopicsSessionId.RespTopicsSessionId{}
	data.Bitmap, _ = bitmap.ToBase64()
	route := &toWorkerRouter.Router{}
	route.Cmd = toWorkerRouter.Cmd_RespTopicsSessionId
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	_, _ = r.Write(b)
}

// 根据session id获取网关中的用户信息
func (r *ConnProcessor) reqSessionInfo(toServerRoute *toServerRouter.Router) {
	req := &reqSessionInfo.ReqSessionInfo{}
	if err := proto.Unmarshal(toServerRoute.Data, req); err != nil {
		logging.Error("Proto unmarshal reqSessionInfo.ReqSessionInfo error: %#v", err)
		return
	}
	data := &respSessionInfo.RespSessionInfo{}
	data.SessionId = req.SessionId
	wsConn := customerManager.Manager.Get(req.SessionId)
	if wsConn != nil {
		if info, ok := wsConn.Session().(*session.Info); ok {
			info.GetToRespSessionInfo(data)
		}
	}
	route := &toWorkerRouter.Router{}
	route.Cmd = toWorkerRouter.Cmd_RespSessionInfo
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	_, _ = r.Write(b)
}

// 设置网关的session面存储的用户信息
func (r *ConnProcessor) setSessionUser(toServerRoute *toServerRouter.Router) {
	//设置网关的session面存储的用户信息
	data := &setSessionUser.SetSessionUser{}
	if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
		logging.Error("Proto unmarshal setSessionUser.SetSessionUser error: %#v", err)
		return
	}
	business.SetSessionUser(data)
}

// 返回网关的状态
func (r *ConnProcessor) reqNetSvrStatus() {
	data := &respNetSvrStatus.RespNetSvrStatus{}
	data.CustomerConnCount = int32(session.Id.CountAllocated())
	data.TopicCount = int32(session.Topics.Count())
	data.CatapultWaitSendCount = int32(business.Catapult.CountWaitSend())
	data.CatapultConsumer = int32(configs.Config.CatapultConsumer)
	data.CatapultChanCap = int32(configs.Config.CatapultChanCap)
	route := &toWorkerRouter.Router{}
	route.Cmd = toWorkerRouter.Cmd_RespNetSvrStatus
	route.Data, _ = proto.Marshal(data)
	b, _ := proto.Marshal(route)
	_, _ = r.Write(b)
}
