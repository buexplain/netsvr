package worker

import (
	"encoding/binary"
	"github.com/buexplain/netsvr/internal/customer/business"
	"github.com/buexplain/netsvr/internal/protocol/cmd"
	"github.com/buexplain/netsvr/internal/protocol/singleCast"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	listener net.Listener
	sessions sync.Map
	stopOnce sync.Once
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
	dataLenBuf := make([]byte, 0, 4)
	for {
		dataLenBuf = dataLenBuf[:0]
		//获取前4个字节，确定数据包长度
		if _, err := io.ReadFull(conn, dataLenBuf); err != nil {
			logging.Error("%#v", err)
			continue
		}
		//这里采用大端序，并且最少都是3个字节
		dataLen := binary.BigEndian.Uint32(dataLenBuf)
		if dataLen < 3 || dataLen > 1024*1024*2 {
			logging.Error("发送的数据太大。或者是搞错字节序")
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
		if currentCmd == cmd.SingleCast {
			r := &singleCast.SingleCast{}
			if err := proto.Unmarshal(dataBuf[2:], r); err != nil {
				logging.Error("%#v", err)
				continue
			}
			business.SingleCast(r)
		}
	}
}

var server *Server

func Start() {
	server = &Server{}
}

func Shutdown() {

}
