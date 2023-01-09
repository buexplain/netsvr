package manager

import (
	"bytes"
	"encoding/binary"
	"github.com/buexplain/netsvr/configs"
	"github.com/buexplain/netsvr/internal/customer/business"
	"github.com/buexplain/netsvr/internal/protocol/toServer/registerWorker"
	toServerRouter "github.com/buexplain/netsvr/internal/protocol/toServer/router"
	"github.com/buexplain/netsvr/internal/protocol/toServer/setUserLoginStatus"
	"github.com/buexplain/netsvr/internal/protocol/toServer/singleCast"
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

type Connection struct {
	conn           net.Conn
	writeCh        chan []byte
	lastActiveTime *int64
	closeCh        chan struct{}
}

func NewConnection(conn net.Conn) *Connection {
	var lastActiveTime int64 = 0
	tmp := &Connection{conn: conn, writeCh: make(chan []byte, 100), lastActiveTime: &lastActiveTime, closeCh: make(chan struct{})}
	return tmp
}

func (r *Connection) done() {
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

func (r *Connection) execute(data []byte) {
	if err := binary.Write(r.conn, binary.BigEndian, uint32(len(data))); err == nil {
		if _, err = r.conn.Write(data); err == nil {
			atomic.StoreInt64(r.lastActiveTime, timecache.Unix())
		} else {
			logging.Error("Worker write error: %#v", err)
		}
	} else {
		logging.Error("Worker write error: %#v", err)
	}
}
func (r *Connection) close() {
	select {
	case <-r.closeCh:
	default:
		close(r.closeCh)
		_ = r.conn.Close()
	}
}

func (r *Connection) Send() {
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

func (r *Connection) Write(data []byte) (n int, err error) {
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

func (r *Connection) Read() {
	var workerId = 0
	heartbeatNode := heartbeat.Timer.ScheduleFunc(time.Duration(configs.Config.WorkerHeartbeatIntervalSecond)*time.Second, func() {
		if timecache.Unix()-atomic.LoadInt64(r.lastActiveTime) < configs.Config.WorkerHeartbeatIntervalSecond {
			//还在活跃期内，不做处理
			return
		}
		//超过活跃期，服务端主动发送心跳
		_, _ = r.Write(heartbeat.PingMessage)
	})
	defer func() {
		//收集异常退出的信息
		if err := recover(); err != nil {
			logging.Error("Abnormal exit a worker of id: %d, error: %#v", workerId)
		}
		//注销掉工作进程的id
		if workerId > 0 {
			Manager.Del(workerId, r)
			logging.Info("Unregister a worker by id: %d", workerId)
		}
		//停止心跳检查
		heartbeatNode.Stop()
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
			logging.Error("Worker data is too large", dataLen)
			continue
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
			logging.Error("%#v", err)
			continue
		}
		logging.Debug("Receive worker command: %d", toServerRoute.Cmd)
		if toServerRoute.Cmd == toServerRouter.Cmd_RegisterWorker {
			//注册工作进程
			data := &registerWorker.RegisterWorker{}
			if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
				logging.Error("%#v", err)
				continue
			}
			if MinWorkerId > data.Id || data.Id > MaxWorkerId {
				r.close()
				logging.Error("Wrong work id %d not in range: %d ~ %d", data.Id, MinWorkerId, MaxWorkerId)
				break
			}
			workerId = int(data.Id)
			Manager.Set(workerId, r)
			if data.ProcessConnClose {
				SetProcessConnCloseWorkerId(data.Id)
			}
			if data.ProcessConnOpen {
				SetProcessConnOpenWorkerId(data.Id)
			}
			logging.Info("Register a worker by id: %d", workerId)
		} else if toServerRoute.Cmd == toServerRouter.Cmd_SetUserLoginStatus {
			//用户登录成功
			data := &setUserLoginStatus.SetUserLoginStatus{}
			if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
				logging.Error("%#v", err)
				continue
			}
			business.SetUserLoginStatus.Send(data)
		} else if toServerRoute.Cmd == toServerRouter.Cmd_SingleCast {
			//单播
			data := &singleCast.SingleCast{}
			if err := proto.Unmarshal(toServerRoute.Data, data); err != nil {
				logging.Error("%#v", err)
				continue
			}
			business.SingleCast.Send(data)
		}
	}
}
