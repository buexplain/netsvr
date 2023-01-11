package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/buexplain/netsvr/internal/protocol/toServer/registerWorker"
	toServerRouter "github.com/buexplain/netsvr/internal/protocol/toServer/router"
	"github.com/buexplain/netsvr/internal/protocol/toServer/singleCast"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/connClose"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/connOpen"
	toWorkerRouter "github.com/buexplain/netsvr/internal/protocol/toWorker/router"
	"github.com/buexplain/netsvr/internal/worker/heartbeat"
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"os"
	"time"
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

func (r *Connection) Heartbeat() {
	t := time.NewTicker(time.Duration(55) * time.Second)
	defer func() {
		t.Stop()
	}()
	for {
		<-t.C
		_, _ = r.Write(heartbeat.PingMessage)
		logging.Info("主动发送心跳")
	}
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
		//解压看看服务端传递了什么
		toWorkerRoute := &toWorkerRouter.Router{}
		if err := proto.Unmarshal(dataBuf, toWorkerRoute); err != nil {
			logging.Error("解压服务端数据失败: %#v", err)
			continue
		}
		//客户端连接成功
		if toWorkerRoute.Cmd == toWorkerRouter.Cmd_ConnOpen {
			r.connOpen(toWorkerRoute)
			continue
		}
		//客户端连接
		if toWorkerRoute.Cmd == toWorkerRouter.Cmd_ConnClose {
			r.connClose(toWorkerRoute)
			continue
		}
	}
}

func (r *Connection) connOpen(toWorkerRoute *toWorkerRouter.Router) {
	co := &connOpen.ConnOpen{}
	if err := proto.Unmarshal(toWorkerRoute.Data, co); err != nil {
		logging.Error("解压出具体的业务数据失败: %#v", err)
		return
	}
	logging.Debug("客户端连接打开 %s --> %d", co.RemoteAddr, co.SessionId)
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	//构造单播数据
	ret := &singleCast.SingleCast{}
	ret.SessionId = co.SessionId
	tmp := NewResponse(1, map[string]interface{}{"sessionId": co.SessionId, "clientAddr": co.RemoteAddr})
	ret.Data = tmp
	//将业务数据放到路由上
	toServerRoute.Data, _ = proto.Marshal(ret)
	data, _ := proto.Marshal(toServerRoute)
	_, _ = r.Write(data)
}

func (r *Connection) connClose(toWorkerRoute *toWorkerRouter.Router) {
	cls := &connClose.ConnClose{}
	if err := proto.Unmarshal(toWorkerRoute.Data, cls); err != nil {
		logging.Error("解压出具体的业务数据失败: %#v", err)
		return
	}
	logging.Debug("客户端连接关闭 %s --> %s --> %d", cls.User, cls.RemoteAddr, cls.SessionId)
}

func NewResponse(cmd int, data interface{}) []byte {
	tmp := map[string]interface{}{"cmd": cmd, "data": data}
	ret, _ := json.Marshal(tmp)
	return ret
}

func main() {
	logging.SetLevel(logging.LevelDebug)
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
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_RegisterWorker
	reg := &registerWorker.RegisterWorker{}
	reg.Id = 1
	reg.ProcessConnClose = true
	reg.ProcessConnOpen = true
	toServerRoute.Data, _ = proto.Marshal(reg)
	data, _ := proto.Marshal(toServerRoute)
	if err := binary.Write(conn, binary.BigEndian, uint32(len(data))); err != nil {
		os.Exit(1)
	}
	_, _ = conn.Write(data)
	logging.Debug("注册工作进程 %d ok", reg.Id)
	c := NewConnection(conn)
	go c.Send()
	go c.Heartbeat()
	c.Read()
}
