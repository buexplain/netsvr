package main

import (
	"encoding/binary"
	"github.com/buexplain/netsvr/internal/protocol/toServer/registerWorker"
	toServerRouter "github.com/buexplain/netsvr/internal/protocol/toServer/router"
	"github.com/buexplain/netsvr/pkg/quit"
	"github.com/buexplain/netsvr/test/worker/connProcessor"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"net"
	"os"
)

func init() {
	logging.SetLevel(logging.LevelDebug)
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
	c := connProcessor.NewConnProcessor(conn)
	go c.Send()
	go c.Heartbeat()
	c.Read()
}
