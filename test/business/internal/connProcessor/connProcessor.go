package connProcessor

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/pkg/heartbeat"
	netsvrProtocol "netsvr/pkg/protocol"
	"netsvr/pkg/quit"
	"netsvr/test/business/internal/log"
	"netsvr/test/pkg/protocol"
	"sync"
	"time"
)

type WorkerCmdCallback func(data []byte, processor *ConnProcessor)
type BusinessCmdCallback func(tf *netsvrProtocol.Transfer, param string, processor *ConnProcessor)

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
	receiveCh chan *netsvrProtocol.Router
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
		receiveCh:           make(chan *netsvrProtocol.Router, 100),
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
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Msg("Business heartbeat coroutine is closed")
		} else {
			log.Logger.Debug().Msg("Business heartbeat coroutine is closed")
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
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int("workerId", r.workerId).Msg("Business send coroutine is closed")
		} else {
			log.Logger.Debug().Int("workerId", r.workerId).Msg("Business send coroutine is closed")
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
	r.sendDataLen = uint32(len(data))
	//先写包头，注意这是大端序
	r.sendBuf.WriteByte(byte(r.sendDataLen >> 24))
	r.sendBuf.WriteByte(byte(r.sendDataLen >> 16))
	r.sendBuf.WriteByte(byte(r.sendDataLen >> 8))
	r.sendBuf.WriteByte(byte(r.sendDataLen))
	//再写包体
	var err error
	if _, err = r.sendBuf.Write(data); err != nil {
		log.Logger.Error().Err(err).Msg("Business send to worker buffer failed")
		//写缓冲区失败，重置缓冲区
		r.sendBuf.Reset()
		return
	}
	//设置写超时
	if err = r.conn.SetWriteDeadline(time.Now().Add(time.Second * 60)); err != nil {
		r.ForceClose()
		log.Logger.Warn().Err(err).Msg("Business SetWriteDeadline to worker conn failed")
		return
	}
	//一次性写入到连接中
	_, err = r.sendBuf.WriteTo(r.conn)
	if err != nil {
		r.ForceClose()
		log.Logger.Error().Err(err).Type("errorType", err).Msg("Business send to worker failed")
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
		//打印日志信息
		if err := recover(); err != nil {
			quit.Execute("Business receive coroutine error")
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int("workerId", r.workerId).Msg("Business receive coroutine is closed")
		} else {
			quit.Execute("Worker server shutdown")
			log.Logger.Debug().Int("workerId", r.workerId).Msg("Business receive coroutine is closed")
		}
	}()
	//包头专用
	dataLenBuf := make([]byte, 4)
	//包体专用
	var dataBufCap uint32 = 0
	var dataBuf []byte
	var err error
	for {
		dataLenBuf = dataLenBuf[:0]
		dataLenBuf = dataLenBuf[0:4]
		//获取前4个字节，确定数据包长度
		if _, err = io.ReadFull(r.conn, dataLenBuf); err != nil {
			//读失败了，直接干掉这个连接，让business重新连接，因为缓冲区的tcp流已经脏了，程序无法拆包
			r.ForceClose()
			break
		}
		//这里采用大端序
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		//判断装载数据的缓存区是否足够
		if dataLen > dataBufCap {
			//分配一块更大的，如果dataLen非常的大，则有可能导致内存分配失败
			dataBufCap = dataLen
			dataBuf = make([]byte, dataBufCap)
		} else {
			//清空当前的
			dataBuf = dataBuf[:0]
			dataBuf = dataBuf[0:dataLen]
		}
		//获取数据包
		if _, err = io.ReadAtLeast(r.conn, dataBuf, int(dataLen)); err != nil {
			r.ForceClose()
			log.Logger.Error().Err(err).Msg("Business receive failed")
			break
		}
		//worker响应心跳
		if bytes.Equal(heartbeat.PongMessage, dataBuf[0:dataLen]) {
			continue
		}
		router := &netsvrProtocol.Router{}
		if err := proto.Unmarshal(dataBuf[0:dataLen], router); err != nil {
			log.Logger.Error().Err(err).Msg("Proto unmarshal internalProtocol.Router failed")
			continue
		}
		log.Logger.Debug().Stringer("cmd", router.Cmd).Msg("Business receive worker command")
		select {
		case <-r.producerCh:
			//收到关闭信号，不再生产，进入丢弃数据逻辑
			//如果是强制关闭，则这里会触发错误，直接退出
			//如果是优雅关闭，则这里会不断读取连接中的数据，直到r.sendCh、r.receiveCh被消费干净，进而关闭r.conn，导致这里触发错误退出
			for {
				if err := r.conn.SetReadDeadline(time.Now().Add(time.Second * 60)); err != nil {
					return
				}
				dataBuf = dataBuf[:0]
				if _, err := io.ReadAtLeast(r.conn, dataBuf, len(dataBuf)); err != nil {
					return
				}
			}
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
			log.Logger.Error().Stack().Err(nil).Interface("recover", err).Int("workerId", r.workerId).Msg("Business cmd coroutine is closed")
			time.Sleep(5 * time.Second)
			go r.LoopCmd()
		} else {
			log.Logger.Debug().Int("workerId", r.workerId).Msg("Business cmd coroutine is closed")
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

func (r *ConnProcessor) cmd(router *netsvrProtocol.Router) {
	if router.Cmd == netsvrProtocol.Cmd_Transfer {
		//解析出worker转发过来的对象
		tf := &netsvrProtocol.Transfer{}
		if err := proto.Unmarshal(router.Data, tf); err != nil {
			log.Logger.Error().Err(err).Msg("Proto unmarshal internalProtocol.Transfer failed")
			return
		}
		//解析出业务路由对象
		clientRoute := new(protocol.ClientRouter)
		if err := json.Unmarshal(tf.Data, clientRoute); err != nil {
			log.Logger.Debug().Err(err).Msg("Parse protocol.ClientRouter failed")
			return
		}
		log.Logger.Debug().Stringer("cmd", clientRoute.Cmd).Msg("Business receive client command")
		//客户发来的命令
		if callback, ok := r.businessCmdCallback[clientRoute.Cmd]; ok {
			callback(tf, clientRoute.Data, r)
			return
		}
		//客户请求了错误的命令
		log.Logger.Debug().Interface("cmd", clientRoute.Cmd).Msg("Unknown protocol.clientRoute.Cmd")
		return
	}
	//回调worker发来的命令
	if callback, ok := r.workerCmdCallback[int32(router.Cmd)]; ok {
		callback(router.Data, r)
		return
	}
	//worker传递了未知的命令
	log.Logger.Error().Interface("cmd", router.Cmd).Msg("Unknown internalProtocol.Router.Cmd")
}

func (r *ConnProcessor) RegisterWorkerCmd(cmd interface{}, callback WorkerCmdCallback) {
	if c, ok := cmd.(netsvrProtocol.Cmd); ok {
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
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_Register
	reg := &netsvrProtocol.Register{}
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
	router := &netsvrProtocol.Router{}
	router.Cmd = netsvrProtocol.Cmd_Unregister
	pt, _ := proto.Marshal(router)
	r.Send(pt)
}
