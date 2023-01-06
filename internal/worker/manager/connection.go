package manager

import (
	"bytes"
	"encoding/binary"
	"github.com/buexplain/netsvr/configs"
	"github.com/buexplain/netsvr/internal/customer/business"
	"github.com/buexplain/netsvr/internal/protocol/cmd"
	"github.com/buexplain/netsvr/internal/protocol/registerWorker"
	"github.com/buexplain/netsvr/internal/protocol/singleCast"
	"github.com/buexplain/netsvr/internal/worker/heartbeat"
	"github.com/buexplain/netsvr/pkg/timecache"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"sync/atomic"
	"time"
)

type connection struct {
	conn           net.Conn
	writeCh        chan []byte
	lastActiveTime *int64
	closeCh        chan struct{}
}

func NewConnection(conn net.Conn) *connection {
	var lastActiveTime int64 = 0
	tmp := &connection{conn: conn, writeCh: make(chan []byte, 10), lastActiveTime: &lastActiveTime, closeCh: make(chan struct{})}
	return tmp
}

func (r *connection) Send() {
	defer func() {
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
			return
		case data := <-r.writeCh:
			if err := binary.Write(r.conn, binary.BigEndian, uint32(len(data))); err == nil {
				if _, err = r.conn.Write(data); err == nil {
					atomic.StoreInt64(r.lastActiveTime, timecache.Unix())
				} else {
					logging.Error("Worker write error: %#v", err)
					continue
				}
			} else {
				logging.Error("Worker write error: %#v", err)
				continue
			}
		}
	}
}

func (r *connection) Write(data []byte) (n int, err error) {
	select {
	case <-r.closeCh:
		return 0, nil
	default:
		r.writeCh <- data
		return len(data), nil
	}
}

func (r *connection) Read() {
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
			//读失败了，直接干掉这个连接，让客户端重新连接进来
			close(r.closeCh)
			_ = r.conn.Close()
			logging.Error("Worker read error: %#v", err)
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
			close(r.closeCh)
			_ = r.conn.Close()
			logging.Error("Worker read error: %#v", err)
			break
		}
		//更新客户端的最后活跃时间
		atomic.StoreInt64(r.lastActiveTime, timecache.Unix())
		//客户端发来心跳
		if bytes.Equal(heartbeat.PingMessage, dataBuf) {
			//响应客户端的心跳
			if err := binary.Write(r, binary.BigEndian, uint32(len(heartbeat.PongMessage))); err == nil {
				_, _ = r.Write(heartbeat.PongMessage)
			}
			continue
		}
		//客户端响应心跳
		if bytes.Equal(heartbeat.PongMessage, dataBuf) {
			continue
		}
		//解析前2个字节，确定是什么业务命令
		currentCmd := binary.BigEndian.Uint16(dataBuf[0:2])
		logging.Debug("Receive worker command: %d", currentCmd)
		if currentCmd == cmd.RegisterWorker {
			//注册工作进程
			data := &registerWorker.RegisterWorker{}
			if err := proto.Unmarshal(dataBuf[2:], data); err != nil {
				logging.Error("%#v", err)
				continue
			}
			if MinWorkerId > data.Id || data.Id > MaxWorkerId {
				logging.Error("Wrong work id %d not in range: %d ~ %d", data.Id, MinWorkerId, MaxWorkerId)
				_ = r.conn.Close()
				break
			}
			workerId = int(data.Id)
			Manager.Set(workerId, r)
			logging.Info("Register a worker by id: %d", workerId)
		} else if currentCmd == cmd.SingleCast {
			//单播
			r := &singleCast.SingleCast{}
			if err := proto.Unmarshal(dataBuf[2:], r); err != nil {
				logging.Error("%#v", err)
				continue
			}
			business.SingleCast(r)
			continue
		}
	}
}
