package utils

import (
	"bytes"
	"encoding/binary"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/test/business/configs"
	"netsvr/test/business/internal/log"
	"netsvr/test/business/internal/utils/connPool"
	"strings"
	"time"
)

var pool *connPool.ConnPool

func init() {
	pool = connPool.NewConnPool(configs.Config.ProcessCmdGoroutineNum, func() net.Conn {
		conn, err := net.Dial("tcp", configs.Config.WorkerListenAddress)
		if err != nil {
			log.Logger.Error().Err(err).Type("errorType", err).Msg("Business client connect worker service failed")
			return nil
		}
		return conn
	}, func(conn net.Conn) bool {
		bf := make([]byte, 4+len(netsvrProtocol.PingMessage))
		binary.BigEndian.PutUint32(bf, uint32(len(netsvrProtocol.PingMessage)))
		copy(bf[4:], netsvrProtocol.PingMessage)
		_, err := conn.Write(bf)
		if err != nil {
			log.Logger.Error().Err(err).Type("errorType", err).Msg("Business send to worker failed")
			return false
		}
		data := make([]byte, 4+len(netsvrProtocol.PongMessage))
		if _, err = io.ReadFull(conn, data); err != nil {
			log.Logger.Error().Err(err).Type("errorType", err).Msg("Business read from worker failed")
			return false
		}
		return true
	}, time.Second*45)
}

func RequestNetSvr(req proto.Message, cmd netsvrProtocol.Cmd, resp proto.Message) {
	router := &netsvrProtocol.Router{}
	router.Cmd = cmd
	if req != nil {
		router.Data, _ = proto.Marshal(req)
	}
	pt, _ := proto.Marshal(router)
	//写数据
	bf := &bytes.Buffer{}
	length := len(pt)
	bf.WriteByte(byte(length >> 24))
	bf.WriteByte(byte(length >> 16))
	bf.WriteByte(byte(length >> 8))
	bf.WriteByte(byte(length))
	var err error
	if _, err = bf.Write(pt); err != nil {
		log.Logger.Error().Err(err).Type("errorType", err).Msg("Business send to worker buffer failed")
		return
	}
	conn := pool.Get()
	defer pool.Put(conn)
	_, err = bf.WriteTo(conn)
	if err != nil {
		log.Logger.Error().Err(err).Type("errorType", err).Msg("Business send to worker failed")
		return
	}
	//读数据
loop:
	dataLenBuf := make([]byte, 4)
	if _, err = io.ReadFull(conn, dataLenBuf); err != nil {
		log.Logger.Error().Err(err).Type("errorType", err).Msg("Business read from worker failed")
		return
	}
	dataLen := binary.BigEndian.Uint32(dataLenBuf)
	dataBuf := make([]byte, dataLen)
	if _, err = io.ReadAtLeast(conn, dataBuf, int(dataLen)); err != nil {
		log.Logger.Error().Err(err).Type("errorType", err).Msg("Business read from worker failed")
		return
	}
	if bytes.Equal(netsvrProtocol.PongMessage, dataBuf) {
		goto loop
	}
	router.Reset()
	if err = proto.Unmarshal(dataBuf, router); err != nil {
		log.Logger.Error().Err(err).Type("errorType", err).Msg("Proto unmarshal internalProtocol.Router failed")
		return
	}
	if router.Cmd != cmd {
		log.Logger.Error().Err(err).Int32("response", int32(router.Cmd)).Int32("expect", int32(cmd)).Msg("Worker response cmd error")
		return
	}
	if err = proto.Unmarshal(router.Data, resp); err != nil {
		log.Logger.Error().Err(err).Type("errorType", err).Msgf("Proto unmarshal %T failed", resp)
		return
	}
}

func CheckIsOpen(addr string) bool {
	c, err := net.Dial("tcp", addr)
	if err == nil {
		_ = c.Close()
		return true
	}
	if e, ok := err.(*net.OpError); ok && strings.Contains(e.Err.Error(), "No connection") {
		return false
	}
	return true
}
