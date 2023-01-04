package main

import (
	"encoding/binary"
	"github.com/buexplain/netsvr/internal/protocol/cmd"
	"github.com/buexplain/netsvr/internal/protocol/registerWorker"
	"github.com/buexplain/netsvr/internal/protocol/singleCast"
	"github.com/buexplain/netsvr/internal/protocol/transferToWorker"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"os"
)

func main() {
	conn, _ := net.Dial("tcp", "localhost:8888")
	//注册工作进程
	reg := &registerWorker.RegisterWorker{}
	reg.Id = 1
	data, _ := proto.Marshal(reg)
	if err := binary.Write(conn, binary.BigEndian, uint32(len(data))+2); err != nil {
		os.Exit(1)
	}
	_ = binary.Write(conn, binary.BigEndian, cmd.RegisterWorker)
	_, _ = conn.Write(data)
	logging.Info("注册工作进程 ok")
	//开始工作
	dataLenBuf := make([]byte, 4)
	for {
		dataLenBuf[0] = 0
		dataLenBuf[1] = 0
		dataLenBuf[2] = 0
		dataLenBuf[3] = 0
		if _, err := io.ReadFull(conn, dataLenBuf); err != nil {
			_ = conn.Close()
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
		if _, err := io.ReadFull(conn, dataBuf); err != nil {
			logging.Error("%#v", err)
			continue
		}
		transfer := &transferToWorker.TransferToWorker{}
		_ = proto.Unmarshal(dataBuf, transfer)
		//构造单播数据
		r := &singleCast.SingleCast{}
		r.Data = transfer.Data
		r.SessionId = transfer.SessionId
		result, _ := proto.Marshal(r)
		//回写给服务器
		if err := binary.Write(conn, binary.BigEndian, uint32(len(result))+2); err == nil {
			_ = binary.Write(conn, binary.BigEndian, cmd.SingleCast)
			_, _ = conn.Write(result)
			logging.Info(string(dataBuf))
		}
	}
}
