package connProcessor

import (
	"bytes"
	"encoding/binary"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/internal/worker/heartbeat"
	"netsvr/pkg/quit"
	"netsvr/test/business/protocol"
	"runtime/debug"
	"time"
)

type WorkerCmdCallback func(data []byte, processor *ConnProcessor)
type ClientCmdCallback func(currentSessionId uint32, userInfo string, userId string, param string, processor *ConnProcessor)

type ConnProcessor struct {
	//business与worker的连接
	conn net.Conn
	//要发送给连接的数据
	sendCh chan []byte
	//从连接中读取的数据
	receiveCh chan *internalProtocol.Router
	//连接关闭信号
	closeCh chan struct{}
	//当前连接的服务编号
	workerId int
	//worker发来的各种命令的回调函数
	workerCmdCallback map[internalProtocol.Cmd]WorkerCmdCallback
	//客户发来的各种命令的回调函数
	clientCmdCallback map[protocol.Cmd]ClientCmdCallback
}

func NewConnProcessor(conn net.Conn, workerId int) *ConnProcessor {
	tmp := &ConnProcessor{
		conn:              conn,
		sendCh:            make(chan []byte, 10),
		receiveCh:         make(chan *internalProtocol.Router, 100),
		closeCh:           make(chan struct{}),
		workerId:          workerId,
		workerCmdCallback: map[internalProtocol.Cmd]WorkerCmdCallback{},
		clientCmdCallback: map[protocol.Cmd]ClientCmdCallback{},
	}
	return tmp
}

func (r *ConnProcessor) LoopHeartbeat() {
	t := time.NewTicker(time.Duration(35) * time.Second)
	defer func() {
		if err := recover(); err != nil {
			logging.Error("Business heartbeat coroutine is closed, error: %vn%s", err, debug.Stack())
		} else {
			logging.Debug("Business heartbeat coroutine is closed")
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
			logging.Error("Business send coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
		} else {
			logging.Debug("Business send coroutine is closed, workerId: %d", r.workerId)
		}
		//减少协程wait计数器
		quit.Wg.Done()
	}()
	for {
		select {
		case <-r.closeCh:
			//business与worker的连接已经关闭，直接退出当前协程
			return
		case <-quit.Ctx.Done():
			//business即将停止，处理通道中剩余数据，尽量保证处理后的客户消息转发到worker
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
			logging.Error("Business send to worker error: %v", err)
		}
	} else {
		r.Close()
		logging.Error("Business send to worker error: %v", err)
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

func (r *ConnProcessor) LoopReceive() {
	defer func() {
		//打印日志信息
		if err := recover(); err != nil {
			quit.Execute("Business receive coroutine error")
			logging.Error("Business receive coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
		} else {
			quit.Execute("Business receive coroutine is closed")
			logging.Debug("Business receive coroutine is closed, workerId: %d", r.workerId)
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
			//读失败了，直接干掉这个连接，让business重新连接，因为缓冲区的tcp流已经脏了，程序无法拆包
			r.Close()
			break
		}
		//这里采用大端序，小于2个字节，则说明业务命令都没有
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		if dataLen < 2 || dataLen > workerReceivePackLimit {
			//worker不按套路出牌，不老实，直接关闭它，因为tcp流已经无法拆解了
			r.Close()
			logging.Error("Business receive pack is too large: %d", dataLen)
			break
		}
		//获取数据包
		dataBuf = dataBuf[:0]
		dataBuf = dataBuf[0:dataLen]
		if _, err := io.ReadAtLeast(r.conn, dataBuf, int(dataLen)); err != nil {
			r.Close()
			logging.Error("Business receive error: %v", err)
			break
		}
		//worker发来心跳
		if bytes.Equal(heartbeat.PingMessage, dataBuf[0:dataLen]) {
			//响应客户端的心跳
			r.Send(heartbeat.PongMessage)
			continue
		}
		//worker响应心跳
		if bytes.Equal(heartbeat.PongMessage, dataBuf[0:dataLen]) {
			continue
		}
		router := &internalProtocol.Router{}
		if err := proto.Unmarshal(dataBuf[0:dataLen], router); err != nil {
			logging.Error("Proto unmarshal internalProtocol.Router error: %v", err)
			continue
		}
		logging.Debug("Business receive worker command: %s", router.Cmd)
		r.receiveCh <- router
	}
}

// LoopCmd 循环处理worker发来的各种请求命令
func (r *ConnProcessor) LoopCmd(number int) {
	defer func() {
		if err := recover(); err != nil {
			logging.Error("Business cmd coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
			time.Sleep(5 * time.Second)
			quit.Wg.Add(1)
			go r.LoopCmd(number)
		} else {
			logging.Debug("Business cmd coroutine is closed, workerId: %d", r.workerId)
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

func (r *ConnProcessor) cmd(router *internalProtocol.Router) {
	if router.Cmd == internalProtocol.Cmd_Transfer {
		//解析出worker转发过来的对象
		tf := &internalProtocol.Transfer{}
		if err := proto.Unmarshal(router.Data, tf); err != nil {
			logging.Error("Proto unmarshal internalProtocol.Transfer error: %v", err)
			return
		}
		//解析出业务路由对象
		clientRoute := protocol.ParseClientRouter(tf.Data)
		if clientRoute == nil {
			return
		}
		logging.Debug("Business receive client command: %s", clientRoute.Cmd)
		//客户发来的命令
		if callback, ok := r.clientCmdCallback[clientRoute.Cmd]; ok {
			callback(tf.SessionId, tf.UserInfo, tf.UserId, clientRoute.Data, r)
			return
		}
		//客户请求了错误的命令
		logging.Debug("Unknown protocol.clientRoute.Cmd: %d", router.Cmd)
		return
	}
	//回调worker发来的命令
	if callback, ok := r.workerCmdCallback[router.Cmd]; ok {
		callback(router.Data, r)
		return
	}
	//worker传递了错误的命令
	r.Close()
	logging.Error("Unknown internalProtocol.Router.Cmd: %d", router.Cmd)
}

func (r *ConnProcessor) RegisterSvrCmd(cmd internalProtocol.Cmd, callback WorkerCmdCallback) {
	r.workerCmdCallback[cmd] = callback
}

func (r *ConnProcessor) RegisterClientCmd(cmd protocol.Cmd, callback ClientCmdCallback) {
	r.clientCmdCallback[cmd] = callback
}

// GetWorkerId 返回服务编号
func (r *ConnProcessor) GetWorkerId() int {
	return r.workerId
}

// SetWorkerId 设置服务编号
func (r *ConnProcessor) SetWorkerId(id int) {
	r.workerId = id
}

func (r *ConnProcessor) RegisterWorker() error {
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Register
	reg := &internalProtocol.Register{}
	reg.Id = int32(r.workerId)
	//让worker为我开启10条协程
	reg.ProcessCmdGoroutineNum = 10
	reg.ProcessConnClose = true
	reg.ProcessConnOpen = true
	router.Data, _ = proto.Marshal(reg)
	data, _ := proto.Marshal(router)
	err := binary.Write(r.conn, binary.BigEndian, uint32(len(data)))
	_, err = r.conn.Write(data)
	return err
}

func (r *ConnProcessor) UnregisterWorker() {
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Unregister
	pt, _ := proto.Marshal(router)
	r.Send(pt)
}
