package main

import (
	"bytes"
	"encoding/binary"
	"github.com/buexplain/netsvr/internal/protocol/cmd"
	"github.com/buexplain/netsvr/internal/protocol/registerWorker"
	"github.com/buexplain/netsvr/internal/protocol/singleCast"
	"github.com/buexplain/netsvr/internal/protocol/transferToWorker"
	"github.com/buexplain/netsvr/internal/worker/heartbeat"
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"os"
)

type Connection struct {
	conn    net.Conn
	writeCh chan []byte
	closeCh chan struct{}
}

func NewConnection(conn net.Conn) *Connection {
	tmp := &Connection{conn: conn, writeCh: make(chan []byte, 10), closeCh: make(chan struct{})}
	return tmp
}

func (r *Connection) Send() {
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
				if _, err = r.conn.Write(data); err != nil {
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

func (r *Connection) Write(data []byte) (n int, err error) {
	select {
	case <-r.closeCh:
		return 0, nil
	default:
		r.writeCh <- data
		return len(data), nil
	}
}

func (r *Connection) Read() {
	dataLenBuf := make([]byte, 4)
	for {
		dataLenBuf[0] = 0
		dataLenBuf[1] = 0
		dataLenBuf[2] = 0
		dataLenBuf[3] = 0
		if _, err := io.ReadFull(r.conn, dataLenBuf); err != nil {
			_ = r.conn.Close()
			close(r.writeCh)
			logging.Error("%#v", err)
			break
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
		if _, err := io.ReadFull(r.conn, dataBuf); err != nil {
			_ = r.conn.Close()
			close(r.writeCh)
			logging.Error("%#v", err)
			break
		}
		//服务端响应心跳
		if bytes.Equal(dataBuf, heartbeat.PongMessage) {
			logging.Info("服务端响应心跳")
			continue
		}
		//服务端发来心跳
		if bytes.Equal(dataBuf, heartbeat.PingMessage) {
			logging.Info("服务端发来心跳")
			_, _ = r.Write(heartbeat.PongMessage)
			continue
		}
		transfer := &transferToWorker.TransferToWorker{}
		_ = proto.Unmarshal(dataBuf, transfer)
		//构造单播数据
		ret := &singleCast.SingleCast{}
		ret.Data = transfer.Data
		ret.SessionId = transfer.SessionId
		result, _ := proto.Marshal(ret)
		//回写给服务器
		//TODO 设计一个 cmd data的结构体，用于包裹cmd.SingleCast命令
		if err := binary.Write(r.conn, binary.BigEndian, uint32(len(result))+2); err == nil {
			_ = binary.Write(r.conn, binary.BigEndian, cmd.SingleCast)
			_, _ = r.Write(result)
			logging.Info(string(dataBuf))
		}
	}
}

func main() {
	conn, err := net.Dial("tcp", "localhost:8888")
	if err != nil {
		logging.Error("连接服务端失败，%#v", err)
		os.Exit(1)
	}
	go func() {
		<-quit.ClosedCh
		_ = conn.Close()
	}()
	//注册工作进程
	reg := &registerWorker.RegisterWorker{}
	reg.Id = 1
	data, _ := proto.Marshal(reg)
	if err := binary.Write(conn, binary.BigEndian, uint32(len(data))+2); err != nil {
		os.Exit(1)
	}
	_ = binary.Write(conn, binary.BigEndian, cmd.RegisterWorker)
	_, _ = conn.Write(data)
	logging.Info("注册工作进程 %d ok", reg.Id)
	c := NewConnection(conn)
	go c.Send()
	c.Read()
}
