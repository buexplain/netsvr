package manager

import (
	"bytes"
	"encoding/binary"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/configs"
	"netsvr/internal/heartbeat"
	"netsvr/internal/protocol"
	"runtime/debug"
	"sync"
	"time"
)

type CmdCallback func(data []byte, processor *ConnProcessor)

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
	receiveCh chan *protocol.Router
	//当前连接的服务编号
	workerId int
	//各种命令的回调函数
	cmdCallback map[protocol.Cmd]CmdCallback
}

func NewConnProcessor(conn net.Conn) *ConnProcessor {
	tmp := &ConnProcessor{
		conn:        conn,
		consumerCh:  make(chan struct{}),
		producerCh:  make(chan struct{}),
		consumerWg:  sync.WaitGroup{},
		sendCh:      make(chan []byte, 100),
		sendBuf:     bytes.Buffer{},
		sendDataLen: 0,
		receiveCh:   make(chan *protocol.Router, 100),
		workerId:    0,
		cmdCallback: map[protocol.Cmd]CmdCallback{},
	}
	return tmp
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
			logging.Error("Worker send coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
		} else {
			logging.Debug("Worker send coroutine is closed, workerId: %d", r.workerId)
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
	if r.sendDataLen-4 > configs.Config.WorkerSendPackLimit {
		logging.Error("Worker send pack is too large: %d", r.sendDataLen)
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
		logging.Error("Worker send to business buffer error: %v", err)
		//写缓冲区失败，重置缓冲区
		r.sendBuf.Reset()
		return
	}
	//一次性写入到连接中
	_, err = r.sendBuf.WriteTo(r.conn)
	if err != nil {
		r.ForceClose()
		logging.Error("Worker send to business error: %v", err)
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
		if unregisterWorker, ok := r.cmdCallback[protocol.Cmd_Unregister]; ok {
			unregisterWorker(nil, r)
		}
		//打印日志信息
		if err := recover(); err != nil {
			logging.Error("Worker receive coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
		} else {
			logging.Debug("Worker receive coroutine is closed, workerId: %d", r.workerId)
		}
	}()
	dataLenBuf := make([]byte, 4)
	//先分配4kb，不够的话，中途再分配
	var dataBufCap uint32 = 4096
	dataBuf := make([]byte, dataBufCap)
	var err error
	for {
		dataLenBuf = dataLenBuf[:0]
		dataLenBuf = dataLenBuf[0:4]
		//设置读超时时间
		if err = r.conn.SetReadDeadline(time.Now().Add(configs.Config.WorkerReadDeadline)); err != nil {
			r.ForceClose()
			break
		}
		//获取前4个字节，确定数据包长度
		if _, err = io.ReadFull(r.conn, dataLenBuf); err != nil {
			//读失败了，直接干掉这个连接，让business端重新连接进来，因为缓冲区的tcp流已经脏了，程序无法拆包
			//关掉重来，是最好的办法
			r.ForceClose()
			break
		}
		//这里采用大端序
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		//判断数据长度是否异常
		if dataLen > configs.Config.WorkerReceivePackLimit || dataLen < 1 {
			//如果数据太长，则接下来的make([]byte, dataBufCap)有可能导致程序崩溃，所以直接close对方吧
			r.ForceClose()
			logging.Error("Worker receive pack is too large: %d", dataLen)
			break
		}
		//判断装载数据的缓存区是否足够
		if dataLen > dataBufCap {
			for {
				dataBufCap *= 2
				if dataBufCap < dataLen {
					//一次翻倍不够，继续翻倍
					continue
				}
				if dataBufCap > configs.Config.WorkerReceivePackLimit {
					//n倍之后，溢出限制大小，则变更为限制大小值
					dataBufCap = configs.Config.WorkerReceivePackLimit
				}
				dataBuf = make([]byte, dataBufCap)
				break
			}
		}
		//设置读超时时间
		if err = r.conn.SetReadDeadline(time.Now().Add(configs.Config.WorkerReadDeadline)); err != nil {
			r.ForceClose()
			break
		}
		//获取数据包
		dataBuf = dataBuf[:0]
		dataBuf = dataBuf[0:dataLen]
		if _, err = io.ReadAtLeast(r.conn, dataBuf, int(dataLen)); err != nil {
			r.ForceClose()
			logging.Error("Worker receive error: %v", err)
			break
		}
		//business发来心跳
		if bytes.Equal(heartbeat.PingMessage, dataBuf[0:dataLen]) {
			//响应business的心跳
			r.Send(heartbeat.PongMessage)
			continue
		}
		router := &protocol.Router{}
		if err = proto.Unmarshal(dataBuf[0:dataLen], router); err != nil {
			logging.Error("Proto unmarshal protocol.Router error: %v", err)
			continue
		}
		logging.Debug("Worker receive business command: %s", router.Cmd)
		select {
		case <-r.producerCh:
			//收到关闭信号，不再生产
			return
		default:
			r.receiveCh <- router
		}
	}
}

// LoopCmd 循环处理business发来的各种请求命令
func (r *ConnProcessor) LoopCmd() {
	r.consumerWg.Add(1)
	defer func() {
		r.consumerWg.Done()
		if err := recover(); err != nil {
			logging.Error("Worker cmd coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
			time.Sleep(5 * time.Second)
			go r.LoopCmd()
		} else {
			logging.Debug("Worker cmd coroutine is closed, workerId: %d", r.workerId)
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

func (r *ConnProcessor) cmd(router *protocol.Router) {
	if callback, ok := r.cmdCallback[router.Cmd]; ok {
		callback(router.Data, r)
		return
	}
	//business搞错了指令，直接关闭连接，让business明白，不能瞎传，代码一定要通过测试
	r.ForceClose()
	logging.Error("Unknown protocol.router.Cmd: %d", router.Cmd)
}

// RegisterCmd 注册各种命令
func (r *ConnProcessor) RegisterCmd(cmd protocol.Cmd, callback CmdCallback) {
	r.cmdCallback[cmd] = callback
}

// GetWorkerId 返回business的服务编号
func (r *ConnProcessor) GetWorkerId() int {
	return r.workerId
}

// SetWorkerId 设置business的服务编号
func (r *ConnProcessor) SetWorkerId(id int) {
	r.workerId = id
}
