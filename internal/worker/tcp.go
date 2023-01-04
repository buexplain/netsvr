package worker

import (
	"encoding/binary"
	"errors"
	"github.com/buexplain/netsvr/internal/customer/business"
	"github.com/buexplain/netsvr/internal/protocol/cmd"
	"github.com/buexplain/netsvr/internal/protocol/registerWorker"
	"github.com/buexplain/netsvr/internal/protocol/singleCast"
	"github.com/buexplain/netsvr/internal/worker/manager"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"os"
	"time"
)

type Server struct {
	listener net.Listener
}

func (r *Server) Start() {
	var delay int64 = 0
	for {
		conn, err := r.listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if delay == 0 {
					delay = 15
				} else {
					delay *= 2
				}
				if delay > 1000 {
					delay = 1000
				}
				time.Sleep(time.Millisecond * time.Duration(delay))
				continue
			}
			return
		}
		go r.handleConnection(conn)
	}
}

func (r *Server) handleConnection(conn net.Conn) {
	dataLenBuf := make([]byte, 4)
	var workerId = 0
	defer func() {
		if workerId > 0 {
			manager.Manager.Del(workerId, conn)
			logging.Info("Unregister a worker by id: %d", workerId)
		}
	}()
	for {
		dataLenBuf[0] = 0
		dataLenBuf[1] = 0
		dataLenBuf[2] = 0
		dataLenBuf[3] = 0
		if err := conn.SetReadDeadline(time.Now().Add(time.Second * 10)); err != nil {
			logging.Error("%#v", err)
			continue
		}
		//获取前4个字节，确定数据包长度
		if _, err := io.ReadFull(conn, dataLenBuf); err != nil {
			if opErr, ok := err.(*net.OpError); ok {
				//读取超时
				if opErr.Timeout() || errors.Is(opErr.Err, os.ErrDeadlineExceeded) {
					continue
				}
				//对方连接已经不存在
				_ = conn.Close()
				break
			}
			//对方发送后立刻关闭了连接
			if err == io.ErrUnexpectedEOF || err == io.EOF {
				_ = conn.Close()
				break
			}
			logging.Error("%#v", err)
			continue
		}
		if len(dataLenBuf) != 4 {
			continue
		}
		//这里采用大端序
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		if dataLen < 2 || dataLen > 1024*1024*2 {
			logging.Error("发送的数据太大，或者是搞错字节序，dataLen: %d", dataLen)
			continue
		}
		//获取数据包
		dataBuf := make([]byte, dataLen)
		if _, err := io.ReadFull(conn, dataBuf); err != nil {
			logging.Error("%#v", err)
			continue
		}
		//解析前2个字节，确定是什么业务命令
		currentCmd := binary.BigEndian.Uint16(dataBuf[0:2])
		logging.Debug("Receive command: %d", currentCmd)
		if currentCmd == cmd.RegisterWorker {
			r := &registerWorker.RegisterWorker{}
			if err := proto.Unmarshal(dataBuf[2:], r); err != nil {
				logging.Error("%#v", err)
				continue
			}
			if manager.MinWorkerId > r.Id || r.Id > manager.MaxWorkerId {
				logging.Error("Wrong work id %d not in range: %d ~ %d", r.Id, manager.MinWorkerId, manager.MaxWorkerId)
				_ = conn.Close()
				break
			}
			workerId = int(r.Id)
			manager.Manager.Set(workerId, conn)
			logging.Info("Register a worker by id: %d", workerId)
		} else if currentCmd == cmd.SingleCast {
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

var server *Server

func Start() {
	listen, err := net.Listen("tcp", "localhost:8888")
	if err != nil {
		return
	}
	server = &Server{
		listener: listen,
	}
	server.Start()
}

func Shutdown() {
	_ = server.listener.Close()
}
