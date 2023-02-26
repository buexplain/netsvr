package connProcessor

import (
	"bytes"
	"encoding/binary"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/configs"
	"netsvr/internal/heartbeat"
	internalProtocol "netsvr/internal/protocol"
	"netsvr/pkg/quit"
	"netsvr/test/protocol"
	"runtime/debug"
	"sync"
	"time"
)

type WorkerCmdCallback func(data []byte, processor *ConnProcessor)
type BusinessCmdCallback func(tf *internalProtocol.Transfer, param string, processor *ConnProcessor)

type ConnProcessor struct {
	//business与worker的连接
	conn net.Conn
	//消费者退出信号
	consumerCh chan struct{}
	//生产者退出信号
	producerCh chan struct{}
	//消费者协程退出等待器
	consumerWg sync.WaitGroup
	//要发送给连接的数据
	sendCh chan []byte
	//发送缓冲区
	sendBuf     bytes.Buffer
	sendDataLen uint32
	//从连接中读取的数据
	receiveCh chan *internalProtocol.Router
	//当前连接的服务编号
	workerId int
	//worker发来的各种命令的回调函数
	workerCmdCallback map[int32]WorkerCmdCallback
	//客户发来的各种命令的回调函数
	businessCmdCallback map[protocol.Cmd]BusinessCmdCallback
}

func NewConnProcessor(conn net.Conn, workerId int) *ConnProcessor {
	tmp := &ConnProcessor{
		conn:                conn,
		consumerCh:          make(chan struct{}),
		producerCh:          make(chan struct{}),
		consumerWg:          sync.WaitGroup{},
		sendCh:              make(chan []byte, 100),
		sendBuf:             bytes.Buffer{},
		sendDataLen:         0,
		receiveCh:           make(chan *internalProtocol.Router, 100),
		workerId:            workerId,
		workerCmdCallback:   map[int32]WorkerCmdCallback{},
		businessCmdCallback: map[protocol.Cmd]BusinessCmdCallback{},
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
	}()
	for {
		select {
		case <-r.producerCh:
			return
		case <-t.C:
			//这个心跳一定要发，否则服务端会把连接干掉
			r.Send(heartbeat.PingMessage)
		}
	}
}

// GraceClose 优雅关闭
func (r *ConnProcessor) GraceClose() {
	select {
	case <-r.producerCh:
		return
	default:
		//通知所有生产者，不再生产数据
		close(r.producerCh)
		//通知所有消费者，消费完毕后退出
		close(r.sendCh)
		close(r.receiveCh)
		//等待消费者协程退出
		r.consumerWg.Wait()
		//关闭连接
		_ = r.conn.Close()
	}
}

// ForceClose 强制关闭
func (r *ConnProcessor) ForceClose() {
	select {
	case <-r.producerCh:
		return
	default:
		//通知所有生产者，不再生产数据
		close(r.producerCh)
		//通知所有消费者，立刻退出
		close(r.consumerCh)
		//关闭连接
		_ = r.conn.Close()
	}
}

func (r *ConnProcessor) LoopSend() {
	r.consumerWg.Add(1)
	defer func() {
		r.consumerWg.Done()
		//打印日志信息
		if err := recover(); err != nil {
			logging.Error("Business send coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
		} else {
			logging.Debug("Business send coroutine is closed, workerId: %d", r.workerId)
		}
	}()
	for {
		select {
		case data, ok := <-r.sendCh:
			if ok == false {
				//管道被关闭
				return
			}
			r.send(data)
		case <-r.consumerCh:
			//连接被关闭
			return
		}
	}
}

func (r *ConnProcessor) send(data []byte) {
	//包太大，不发
	r.sendDataLen = uint32(len(data))
	if r.sendDataLen-4 > configs.Config.WorkerReceivePackLimit {
		//不能超过worker的收包的大小，否则worker会断开连接
		logging.Error("Business send pack is too large: %d", r.sendDataLen)
		return
	}
	//先写包头，注意这是大端序
	r.sendBuf.WriteByte(byte(r.sendDataLen >> 24))
	r.sendBuf.WriteByte(byte(r.sendDataLen >> 16))
	r.sendBuf.WriteByte(byte(r.sendDataLen >> 8))
	r.sendBuf.WriteByte(byte(r.sendDataLen))
	//再写包体
	var err error
	if _, err = r.sendBuf.Write(data); err != nil {
		logging.Error("Business send to worker buffer error: %v", err)
		//写缓冲区失败，重置缓冲区
		r.sendBuf.Reset()
		return
	}
	//一次性写入到连接中
	_, err = r.sendBuf.WriteTo(r.conn)
	if err != nil {
		r.ForceClose()
		logging.Error("Business send to worker error: %v", err)
		return
	}
	//写入成功，重置缓冲区
	r.sendBuf.Reset()
}

func (r *ConnProcessor) Send(data []byte) {
	select {
	case <-r.producerCh:
		//收到关闭信号，不再生产
		return
	default:
		r.sendCh <- data
	}
}

func (r *ConnProcessor) LoopReceive() {
	defer func() {
		//打印日志信息
		if err := recover(); err != nil {
			quit.Execute("Business receive coroutine error")
			logging.Error("Business receive coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
		} else {
			quit.Execute("Worker server shutdown")
			logging.Debug("Business receive coroutine is closed, workerId: %d", r.workerId)
		}
	}()
	dataLenBuf := make([]byte, 4)
	dataBuf := make([]byte, configs.Config.WorkerSendPackLimit)
	for {
		dataLenBuf = dataLenBuf[:0]
		dataLenBuf = dataLenBuf[0:4]
		//获取前4个字节，确定数据包长度
		if _, err := io.ReadFull(r.conn, dataLenBuf); err != nil {
			//读失败了，直接干掉这个连接，让business重新连接，因为缓冲区的tcp流已经脏了，程序无法拆包
			r.ForceClose()
			break
		}
		//这里采用大端序
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		if dataLen > configs.Config.WorkerSendPackLimit || dataLen < 1 {
			//如果数据太长，直接close对方吧
			r.ForceClose()
			logging.Error("Business receive pack is too large: %d", dataLen)
			break
		}
		//获取数据包
		dataBuf = dataBuf[:0]
		dataBuf = dataBuf[0:dataLen]
		if _, err := io.ReadAtLeast(r.conn, dataBuf, int(dataLen)); err != nil {
			r.ForceClose()
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
		select {
		case <-r.producerCh:
			//收到关闭信号，不再生产
			return
		default:
			r.receiveCh <- router
		}
	}
}

// LoopCmd 循环处理worker发来的各种请求命令
func (r *ConnProcessor) LoopCmd() {
	r.consumerWg.Add(1)
	defer func() {
		r.consumerWg.Done()
		if err := recover(); err != nil {
			logging.Error("Business cmd coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
			time.Sleep(5 * time.Second)
			go r.LoopCmd()
		} else {
			logging.Debug("Business cmd coroutine is closed, workerId: %d", r.workerId)
		}
	}()
	for {
		select {
		case data, ok := <-r.receiveCh:
			if ok == false {
				//管道被关闭
				return
			}
			r.cmd(data)
		case <-r.consumerCh:
			//连接被关闭
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
		if callback, ok := r.businessCmdCallback[clientRoute.Cmd]; ok {
			callback(tf, clientRoute.Data, r)
			return
		}
		//客户请求了错误的命令
		logging.Debug("Unknown protocol.clientRoute.Cmd: %s", clientRoute.Cmd)
		return
	}
	//回调worker发来的命令
	if callback, ok := r.workerCmdCallback[int32(router.Cmd)]; ok {
		callback(router.Data, r)
		return
	}
	//worker传递了未知的命令
	logging.Error("Unknown internalProtocol.Router.Cmd: %s", router.Cmd)
}

func (r *ConnProcessor) RegisterWorkerCmd(cmd interface{}, callback WorkerCmdCallback) {
	if c, ok := cmd.(internalProtocol.Cmd); ok {
		r.workerCmdCallback[int32(c)] = callback
		return
	}
	if c, ok := cmd.(protocol.Cmd); ok {
		r.workerCmdCallback[int32(c)] = callback
	}
}

func (r *ConnProcessor) RegisterBusinessCmd(cmd protocol.Cmd, callback BusinessCmdCallback) {
	r.businessCmdCallback[cmd] = callback
}

// GetWorkerId 返回服务编号
func (r *ConnProcessor) GetWorkerId() int {
	return r.workerId
}

// SetWorkerId 设置服务编号
func (r *ConnProcessor) SetWorkerId(id int) {
	r.workerId = id
}

func (r *ConnProcessor) RegisterWorker(processCmdGoroutineNum uint32) error {
	router := &internalProtocol.Router{}
	router.Cmd = internalProtocol.Cmd_Register
	reg := &internalProtocol.Register{}
	reg.Id = int32(r.workerId)
	//让worker为我开启n条协程来处理我的请求
	reg.ProcessCmdGoroutineNum = processCmdGoroutineNum
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
