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
)

func RequestNetSvr(req proto.Message, cmd netsvrProtocol.Cmd, resp proto.Message) {
	router := &netsvrProtocol.Router{}
	router.Cmd = cmd
	if req != nil {
		router.Data, _ = proto.Marshal(req)
	}
	pt, _ := proto.Marshal(router)
	conn, err := net.Dial("tcp", configs.Config.WorkerListenAddress)
	if err != nil {
		log.Logger.Error().Err(err).Type("errorType", err).Msg("Business client connect worker service failed")
		return
	}
	defer func() {
		_ = conn.Close()
	}()
	//写数据
	bf := &bytes.Buffer{}
	length := len(pt)
	bf.WriteByte(byte(length >> 24))
	bf.WriteByte(byte(length >> 16))
	bf.WriteByte(byte(length >> 8))
	bf.WriteByte(byte(length))
	if _, err = bf.Write(pt); err != nil {
		log.Logger.Error().Err(err).Type("errorType", err).Msg("Business send to worker buffer failed")
		return
	}
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
