package manager

import (
	"bytes"
	"encoding/binary"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/configs"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/internal/worker/heartbeat"
	"netsvr/pkg/quit"
	"netsvr/pkg/timecache"
	"runtime/debug"
	"sync/atomic"
	"time"
)

type CmdCallback func(data []byte, processor *ConnProcessor)

type ConnProcessor struct {
	//连接
	conn net.Conn
	//要发送给连接的数据
	sendCh chan []byte
	//从连接中读取的数据
	readCh chan []byte
	//连接最后发送消息的时间
	lastActiveTime *int64
	//连接关闭信号
	closeCh chan struct{}
	//当前连接的编号id
	workerId int
	//各种命令的回调函数
	cmdCallback map[toServerRouter.Cmd]CmdCallback
}

func NewConnProcessor(conn net.Conn) *ConnProcessor {
	var lastActiveTime int64 = 0
	tmp := &ConnProcessor{
		conn:           conn,
		sendCh:         make(chan []byte, 100),
		readCh:         make(chan []byte, 100),
		lastActiveTime: &lastActiveTime,
		closeCh:        make(chan struct{}),
		workerId:       0,
		cmdCallback:    map[toServerRouter.Cmd]CmdCallback{},
	}
	return tmp
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

func (r *ConnProcessor) LoopRead() {
	heartbeatNode := heartbeat.Timer.ScheduleFunc(time.Duration(configs.Config.WorkerHeartbeatIntervalSecond)*time.Second, func() {
		if timecache.Unix()-atomic.LoadInt64(r.lastActiveTime) < configs.Config.WorkerHeartbeatIntervalSecond {
			//还在活跃期内，不做处理
			return
		}
		//超过活跃期，服务端主动发送心跳
		r.Send(heartbeat.PingMessage)
	})
	defer func() {
		if unregisterWorker, ok := r.cmdCallback[toServerRouter.Cmd_UnregisterWorker]; ok {
			unregisterWorker(nil, r)
		}
		//停止心跳检查
		heartbeatNode.Stop()
		//打印日志信息
		if err := recover(); err != nil {
			logging.Error("Worker read coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
		} else {
			logging.Debug("Worker read coroutine is closed, workerId: %d", r.workerId)
		}
		//减少协程wait计数器
		quit.Wg.Done()
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
			r.Close()
			break
		}
		if len(dataLenBuf) != 4 {
			continue
		}
		//这里采用大端序，小于2个字节，则说明业务命令都没有
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		if dataLen < 2 || dataLen > configs.Config.WorkerReadPackLimit {
			//工作进程不按套路出牌，不老实，直接关闭它
			r.Close()
			logging.Error("Worker data is too large: %d", dataLen)
			break
		}
		//获取数据包
		dataBuf := make([]byte, dataLen)
		if _, err := io.ReadFull(r.conn, dataBuf); err != nil {
			r.Close()
			logging.Error("Worker read error: %v", err)
			break
		}
		//更新客户端的最后活跃时间
		atomic.StoreInt64(r.lastActiveTime, timecache.Unix())
		//客户端发来心跳
		if bytes.Equal(heartbeat.PingMessage, dataBuf) {
			//响应客户端的心跳
			r.Send(heartbeat.PongMessage)
			continue
		}
		//客户端响应心跳
		if bytes.Equal(heartbeat.PongMessage, dataBuf) {
			continue
		}
		r.readCh <- dataBuf
	}
}

// LoopCmd 循环处理工作进程发来的各种请求命令
func (r *ConnProcessor) LoopCmd(number int) {
	defer func() {
		quit.Wg.Done()
		if err := recover(); err != nil {
			logging.Error("Worker cmd coroutine is closed, workerId: %d, error: %v\n%s", r.workerId, err, debug.Stack())
			time.Sleep(5 * time.Second)
			quit.Wg.Add(1)
			go r.LoopCmd(number)
		} else {
			logging.Debug("Worker cmd coroutine is closed, workerId: %d", r.workerId)
		}
	}()
	for {
		select {
		case <-r.closeCh:
			return
		case <-quit.Ctx.Done():
			r.loopCmdDone(number)
			return
		case data := <-r.readCh:
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
		case v := <-r.readCh:
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
		case v := <-r.readCh:
			r.cmd(v)
		case <-time.After(1 * time.Second):
			return
		}
	}
}

func (r *ConnProcessor) cmd(data []byte) {
	toServerRoute := &toServerRouter.Router{}
	if err := proto.Unmarshal(data, toServerRoute); err != nil {
		logging.Error("Proto unmarshal toServerRouter.Router error: %v", err)
		return
	}
	logging.Debug("Receive worker command: %s", toServerRoute.Cmd)
	if callback, ok := r.cmdCallback[toServerRoute.Cmd]; ok {
		callback(toServerRoute.Data, r)
		return
	}
	//工作进程搞错了指令，直接关闭连接，让工作进程明白，不能瞎传，代码一定要通过测试
	r.Close()
	logging.Error("Unknown toServerRoute.Cmd: %d", toServerRoute.Cmd)
}

// RegisterCmd 注册各种命令
func (r *ConnProcessor) RegisterCmd(cmd toServerRouter.Cmd, callback CmdCallback) {
	r.cmdCallback[cmd] = callback
}

// GetWorkerId 返回工作进程的编号
func (r *ConnProcessor) GetWorkerId() int {
	return r.workerId
}

// SetWorkerId 设置工作进程的编号
func (r *ConnProcessor) SetWorkerId(id int) {
	r.workerId = id
}
