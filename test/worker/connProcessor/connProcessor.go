package connProcessor

import (
	"bytes"
	"encoding/binary"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/internal/protocol/toServer/registerWorker"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	toWorkerRouter "netsvr/internal/protocol/toWorker/router"
	"netsvr/internal/protocol/toWorker/transfer"
	"netsvr/internal/worker/heartbeat"
	"netsvr/pkg/quit"
	"netsvr/test/worker/protocol"
	"runtime/debug"
	"time"
)

type SvrCmdCallback func(data []byte, processor *ConnProcessor)
type ClientCmdCallback func(currentSessionId uint32, userStr string, param string, processor *ConnProcessor)

type ConnProcessor struct {
	//连接
	conn net.Conn
	//要发送给连接的数据
	sendCh chan []byte
	//从连接中读取的数据
	receiveCh chan *toWorkerRouter.Router
	//连接关闭信号
	closeCh chan struct{}
	//当前连接的编号id
	workerId int
	//网关发来的各种命令的回调函数
	svrCmdCallback map[toWorkerRouter.Cmd]SvrCmdCallback
	//用户发来的各种命令的回调函数
	clientCmdCallback map[protocol.Cmd]ClientCmdCallback
}

func NewConnProcessor(conn net.Conn, workerId int) *ConnProcessor {
	tmp := &ConnProcessor{
		conn:              conn,
		sendCh:            make(chan []byte, 10),
		receiveCh:         make(chan *toWorkerRouter.Router, 100),
		closeCh:           make(chan struct{}),
		workerId:          workerId,
		svrCmdCallback:    map[toWorkerRouter.Cmd]SvrCmdCallback{},
		clientCmdCallback: map[protocol.Cmd]ClientCmdCallback{},
	}
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

func (r *ConnProcessor) Close() {
	select {
	case <-r.closeCh:
	default:
		close(r.closeCh)
		time.Sleep(100 * time.Millisecond)
		_ = r.conn.Close()
	}
}

func (r *ConnProcessor) LoopSend() {
	defer func() {
		//写协程退出，直接关闭连接
		r.Close()
		//打印日志信息
		if err := recover(); err != nil {
			logging.Error("Worker send coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
		} else {
			logging.Debug("Worker send coroutine is closed, workerId: %d", r.workerId)
		}
		//减少协程wait计数器
		quit.Wg.Done()
	}()
	for {
		select {
		case <-r.closeCh:
			//工作进程的连接已经关闭，直接退出当前协程
			return
		case <-quit.Ctx.Done():
			//网关进程即将停止，处理通道中剩余数据，尽量保证用户消息转发到工作进程
			r.loopSendDone()
			return
		case data := <-r.sendCh:
			r.send(data)
		}
	}
}

func (r *ConnProcessor) loopSendDone() {
	//处理通道中的剩余数据
	for {
		for i := len(r.sendCh); i > 0; i-- {
			v := <-r.sendCh
			r.send(v)
		}
		if len(r.sendCh) == 0 {
			break
		}
	}
	//再次处理通道中的剩余数据，直到超时退出
	for {
		select {
		case v := <-r.sendCh:
			r.send(v)
		case <-time.After(1 * time.Second):
			return
		}
	}
}

func (r *ConnProcessor) send(data []byte) {
	if err := binary.Write(r.conn, binary.BigEndian, uint32(len(data))); err == nil {
		if _, err = r.conn.Write(data); err != nil {
			r.Close()
			logging.Error("Worker send error: %v", err)
		}
	} else {
		r.Close()
		logging.Error("Worker send error: %v", err)
	}
}

func (r *ConnProcessor) Send(data []byte) {
	select {
	case <-r.closeCh:
		//工作进程即将关闭，停止转发消息到工作进程
		return
	case <-quit.Ctx.Done():
		//网关进程即将停止，停止转发消息到工作进程
		return
	default:
		r.sendCh <- data
		return
	}
}

func (r *ConnProcessor) LoopReceive() {
	defer func() {
		//打印日志信息
		if err := recover(); err != nil {
			logging.Error("Worker receive coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
		} else {
			logging.Debug("Worker receive coroutine is closed, workerId: %d", r.workerId)
		}
	}()
	var workerReceivePackLimit uint32 = 1024 * 1024 * 2
	dataLenBuf := make([]byte, 4)
	dataBuf := make([]byte, workerReceivePackLimit)
	for {
		dataLenBuf = dataLenBuf[:0]
		dataLenBuf = dataLenBuf[0:4]
		//获取前4个字节，确定数据包长度
		if _, err := io.ReadFull(r.conn, dataLenBuf); err != nil {
			//读失败了，直接干掉这个连接，让客户端重新连接进来，因为缓冲区的tcp流已经脏了，程序无法拆包
			r.Close()
			break
		}
		//这里采用大端序，小于2个字节，则说明业务命令都没有
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		if dataLen < 2 || dataLen > workerReceivePackLimit {
			//工作进程不按套路出牌，不老实，直接关闭它，因为tcp流已经无法拆解了
			r.Close()
			logging.Error("Worker data is too large: %d", dataLen)
			break
		}
		//获取数据包
		dataBuf = dataBuf[:0]
		dataBuf = dataBuf[0:dataLen]
		if _, err := io.ReadAtLeast(r.conn, dataBuf, int(dataLen)); err != nil {
			r.Close()
			logging.Error("Worker receive error: %v", err)
			break
		}
		//网关发来心跳
		if bytes.Equal(heartbeat.PingMessage, dataBuf[0:dataLen]) {
			//响应客户端的心跳
			r.Send(heartbeat.PongMessage)
			continue
		}
		//网关响应心跳
		if bytes.Equal(heartbeat.PongMessage, dataBuf[0:dataLen]) {
			continue
		}
		toWorkerRoute := &toWorkerRouter.Router{}
		if err := proto.Unmarshal(dataBuf[0:dataLen], toWorkerRoute); err != nil {
			logging.Error("Proto unmarshal toWorkerRouter.Router error: %v", err)
			continue
		}
		logging.Debug("Receive worker command: %s", toWorkerRoute.Cmd)
		r.receiveCh <- toWorkerRoute
	}
}

// LoopCmd 循环处理网关进程发来的各种请求命令
func (r *ConnProcessor) LoopCmd(number int) {
	defer func() {
		if err := recover(); err != nil {
			logging.Error("Worker cmd coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
			time.Sleep(5 * time.Second)
			quit.Wg.Add(1)
			go r.LoopCmd(number)
		} else {
			logging.Debug("Worker cmd coroutine is closed, workerId: %d", r.workerId)
		}
		quit.Wg.Done()
	}()
	for {
		select {
		case <-r.closeCh:
			return
		case <-quit.Ctx.Done():
			r.loopCmdDone(number)
			return
		case data := <-r.receiveCh:
			r.cmd(data)
		}
	}
}

func (r *ConnProcessor) loopCmdDone(number int) {
	//处理通道中的剩余数据
	empty := 0
	for {
		//所有协程遇到多次没拿到数据的情况，视为通道中没有数据了
		if empty > 5 {
			break
		}
		select {
		case v := <-r.receiveCh:
			r.cmd(v)
		default:
			empty++
		}
	}
	//留下0号协程，进行一个超时等待处理
	if number != 0 {
		return
	}
	//再次处理通道中的剩余数据，直到超时退出
	for {
		select {
		case v := <-r.receiveCh:
			r.cmd(v)
		case <-time.After(1 * time.Second):
			return
		}
	}
}

func (r *ConnProcessor) cmd(toWorkerRoute *toWorkerRouter.Router) {
	if toWorkerRoute.Cmd == toWorkerRouter.Cmd_Transfer {
		//解析出网关转发过来的对象
		tf := &transfer.Transfer{}
		if err := proto.Unmarshal(toWorkerRoute.Data, tf); err != nil {
			logging.Error("Proto unmarshal transfer.Transfer error: %v", err)
			return
		}
		//解析出业务路由对象
		clientRoute := protocol.ParseClientRouter(tf.Data)
		if clientRoute == nil {
			return
		}
		//用户发来的命令
		if callback, ok := r.clientCmdCallback[clientRoute.Cmd]; ok {
			callback(tf.SessionId, tf.User, clientRoute.Data, r)
			return
		}
		//用户请求了错误的命令
		logging.Error("Unknown clientRoute.Cmd: %d", toWorkerRoute.Cmd)
		return
	}
	//回调网关发来的命令
	if callback, ok := r.svrCmdCallback[toWorkerRoute.Cmd]; ok {
		callback(toWorkerRoute.Data, r)
		return
	}
	//网关传递了错误的命令
	r.Close()
	logging.Error("Unknown toWorkerRoute.Cmd: %d", toWorkerRoute.Cmd)
}

func (r *ConnProcessor) RegisterSvrCmd(cmd toWorkerRouter.Cmd, callback SvrCmdCallback) {
	r.svrCmdCallback[cmd] = callback
}

func (r *ConnProcessor) RegisterClientCmd(cmd protocol.Cmd, callback ClientCmdCallback) {
	r.clientCmdCallback[cmd] = callback
}

// GetWorkerId 返回工作进程的编号
func (r *ConnProcessor) GetWorkerId() int {
	return r.workerId
}

// SetWorkerId 设置工作进程的编号
func (r *ConnProcessor) SetWorkerId(id int) {
	r.workerId = id
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
