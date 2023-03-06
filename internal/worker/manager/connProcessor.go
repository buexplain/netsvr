package manager

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/configs"
	"netsvr/internal/heartbeat"
	"netsvr/internal/log"
	"netsvr/internal/protocol"
	"strings"
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
		//此刻两个管道也许已经满了，写入的协程正在阻塞中
		//贸然close掉两个管道会引起send on closed channel错误
		//所以，先检查管道是否空着，等待彻底空着，再关闭管道
		for {
			time.Sleep(time.Millisecond * 200)
			if len(r.sendCh) == 0 && len(r.receiveCh) == 0 {
				break
			}
		}
		//通知所有消费者，消费完毕后退出
		close(r.sendCh)
		close(r.receiveCh)
		//等待消费者协程退出
		r.consumerWg.Wait()
		//关闭连接
		//这里等待一下，因为连接可能已经写入了数据，所以不能立刻close它
		time.Sleep(time.Millisecond * 100)
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
		close(r.sendCh)
		close(r.receiveCh)
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
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int("workerId", r.workerId).Msg("Worker send coroutine is closed")
		} else {
			log.Logger.Debug().Int("workerId", r.workerId).Msg("Worker send coroutine is closed")
		}
	}()
	for data := range r.sendCh {
		select {
		case <-r.consumerCh:
			//连接被关闭
			return
		default:
			r.send(data)
		}
	}
}

func (r *ConnProcessor) send(data []byte) {
	//包太大，不发
	r.sendDataLen = uint32(len(data))
	if r.sendDataLen-4 > configs.Config.Worker.SendPackLimit {
		log.Logger.Error().Uint32("packLength", r.sendDataLen-4).Uint32("packLimit", configs.Config.Worker.SendPackLimit).Msg("Worker send pack is too large")
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
		log.Logger.Error().Err(err).Msg("Worker send to business buffer failed")
		//写缓冲区失败，重置缓冲区
		r.sendBuf.Reset()
		return
	}
	//设置写超时
	if err = r.conn.SetWriteDeadline(time.Now().Add(configs.Config.Worker.SendDeadline)); err != nil {
		r.ForceClose()
		log.Logger.Warn().Err(err).Msg("Worker SetWriteDeadline to business conn failed")
		return
	}
	//一次性写入到连接中
	_, err = r.sendBuf.WriteTo(r.conn)
	if err != nil {
		//如果失败，则有可能写入了部分字节，进而导致business不能解包
		//而且business没有第一时间处理数据，极有可能是阻塞住了
		//所以强制关闭连接是最好的选择
		//否则这里的阻塞会蔓延整个网关进程，导致处理客户心跳的协程都没有，最终导致所有客户连接被服务端强制关闭
		r.ForceClose()
		log.Logger.Error().Err(err).Bytes("workerToBusinessData", data).Msg("Worker send to business failed")
		return
	}
	//写入成功，重置缓冲区
	r.sendBuf.Reset()
}

func (r *ConnProcessor) Send(data []byte) {
	defer func() {
		//因为有可能已经阻塞在r.sendCh <- data的时候，收到<-r.producerCh信号
		//然后因为close(r.sendCh)，最终导致send on closed channel
		_ = recover()
	}()
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
			errStr := fmt.Sprint(err)
			//TODO 等golang发布了针对panic的错误类型的时候，这里需要优化一下
			if strings.EqualFold(errStr, "send on closed channel") {
				log.Logger.Debug().Int("workerId", r.workerId).Msg("Worker receive coroutine is closed")
			} else {
				log.Logger.Error().Stack().Err(nil).Type("recoverType", err).Interface("recover", err).Int("workerId", r.workerId).Msg("Worker receive coroutine is closed")
			}
		} else {
			log.Logger.Debug().Int("workerId", r.workerId).Msg("Worker receive coroutine is closed")
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
		if err = r.conn.SetReadDeadline(time.Now().Add(configs.Config.Worker.ReadDeadline)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).Msg("Worker SetReadDeadline to business conn failed")
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
		if dataLen > configs.Config.Worker.ReceivePackLimit || dataLen < 1 {
			//如果数据太长，则接下来的make([]byte, dataBufCap)有可能导致程序崩溃，所以直接close对方吧
			r.ForceClose()
			log.Logger.Error().Uint32("packLength", dataLen).Uint32("packLimit", configs.Config.Worker.ReceivePackLimit).Msg("Worker receive pack is too large")
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
				if dataBufCap > configs.Config.Worker.ReceivePackLimit {
					//n倍之后，溢出限制大小，则变更为限制大小值
					dataBufCap = configs.Config.Worker.ReceivePackLimit
				}
				dataBuf = make([]byte, dataBufCap)
				break
			}
		}
		//设置读超时时间
		if err = r.conn.SetReadDeadline(time.Now().Add(configs.Config.Worker.ReadDeadline)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).Msg("Worker SetReadDeadline to business conn failed")
			break
		}
		//获取数据包
		dataBuf = dataBuf[:0]
		dataBuf = dataBuf[0:dataLen]
		if _, err = io.ReadAtLeast(r.conn, dataBuf, int(dataLen)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).Msg("Worker receive failed")
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
			log.Logger.Error().Err(err).Msg("Proto unmarshal protocol.Router failed")
			continue
		}
		log.Logger.Debug().Stringer("cmd", router.Cmd).Msg("Worker receive business command")
		select {
		case <-r.producerCh:
			//收到关闭信号，不再生产，进入丢弃数据逻辑
			//如果是强制关闭，则这里会触发错误，直接退出
			//如果是优雅关闭，则这里会不断读取连接中的数据，直到r.sendCh、r.receiveCh被消费干净，进而关闭r.conn，导致这里触发错误退出
			for {
				if err = r.conn.SetReadDeadline(time.Now().Add(configs.Config.Worker.ReadDeadline)); err != nil {
					return
				}
				dataBuf = dataBuf[:0]
				if _, err = io.ReadAtLeast(r.conn, dataBuf, len(dataBuf)); err != nil {
					return
				}
			}
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
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int("workerId", r.workerId).Msg("Worker cmd coroutine is closed")
			time.Sleep(5 * time.Second)
			go r.LoopCmd()
		} else {
			log.Logger.Debug().Int("workerId", r.workerId).Msg("Worker cmd coroutine is closed")
		}
	}()
	for data := range r.receiveCh {
		//这种select、default的写法可能在default阶段刚好连接被关闭了，从而导致r.cmd(data)失败
		//优点就是相比于select去case data, ok := <-r.receiveCh、case <-r.consumerCh的写法性能更高
		//因为select去case data, ok := <-r.receiveCh、case <-r.consumerCh的写法会调用runtime.selectGo方法
		//select、default的写法是经过go语言优化的，感觉是直接if判断一样的效果，没有函数开销
		select {
		case <-r.consumerCh:
			//连接被关闭
			return
		default:
			r.cmd(data)
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
	log.Logger.Error().Interface("cmd", router.Cmd).Msg("Unknown protocol.router.Cmd")
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
