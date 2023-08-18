package utils

import (
	"bytes"
	"encoding/binary"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/test/pkg/utils/connPool"
	"time"
)

var pool *connPool.ConnPool
var logger zerolog.Logger

// InitPool 初始化与网关交互所需的连接池
func InitPool(size int, workerListenAddress string) {
	pool = connPool.NewConnPool(size, func() net.Conn {
		//连接到网关的worker服务器
		conn, err := net.Dial("tcp", workerListenAddress)
		if err != nil {
			logger.Error().Err(err).Type("errorType", err).Msg("Business client connect worker service failed")
			return nil
		}
		return conn
	}, func(conn net.Conn) bool {
		//发送心跳字符串
		bf := make([]byte, 4+len(netsvrProtocol.PingMessage))
		binary.BigEndian.PutUint32(bf, uint32(len(netsvrProtocol.PingMessage)))
		copy(bf[4:], netsvrProtocol.PingMessage)
		_, err := conn.Write(bf)
		if err != nil {
			logger.Error().Err(err).Type("errorType", err).Msg("Business send to worker failed")
			return false
		}
		//接收心跳字符
		data := make([]byte, 4+len(netsvrProtocol.PongMessage))
		if _, err = io.ReadFull(conn, data); err != nil {
			logger.Error().Err(err).Type("errorType", err).Msg("Business read from worker failed")
			return false
		}
		return true
	}, time.Second*45)
}

// RequestNetSvr 负责与网关进程交互的函数，类似于http请求方法，发送一个req请求，接收一个resp响应
func RequestNetSvr(req proto.Message, cmd netsvrProtocol.Cmd, resp proto.Message) {
	//构建一个发送到网关的router对象
	router := &netsvrProtocol.Router{}
	router.Cmd = cmd
	if req != nil {
		//请求体不为空，则将其编码到router对象上
		router.Data, _ = proto.Marshal(req)
	}
	pt, _ := proto.Marshal(router)
	//把router对象的protobuf数据，按大端序构造一个buf
	bf := &bytes.Buffer{}
	length := len(pt)
	bf.WriteByte(byte(length >> 24))
	bf.WriteByte(byte(length >> 16))
	bf.WriteByte(byte(length >> 8))
	bf.WriteByte(byte(length))
	var err error
	if _, err = bf.Write(pt); err != nil {
		logger.Error().Err(err).Type("errorType", err).Msg("Business send to worker buffer failed")
		return
	}
	//获取与网关建立好的连接
	conn := pool.Get()
	//将buf数据发送到网关
	for {
		_, err = bf.WriteTo(conn)
		if err != nil {
			//写入失败，tcp短写，则继续写入
			if err == io.ErrShortWrite {
				continue
			}
			//写入数据失败，视为连接已经被损坏，归还一个空连接到连接池
			pool.Put(nil)
			logger.Error().Err(err).Type("errorType", err).Msg("Business send to worker failed")
			return
		}
		break
	}
	//不需要接收数据，直接退出
	if resp == nil {
		//归还连接
		pool.Put(conn)
		return
	}
	//需要接收网关返回的数据数据
loop:
	dataLenBuf := make([]byte, 4)
	if _, err = io.ReadFull(conn, dataLenBuf); err != nil {
		//读取失败，归还一个空连接到连接池
		pool.Put(nil)
		logger.Error().Err(err).Type("errorType", err).Msg("Business read from worker failed")
		return
	}
	dataLen := binary.BigEndian.Uint32(dataLenBuf)
	dataBuf := make([]byte, dataLen)
	if _, err = io.ReadAtLeast(conn, dataBuf, int(dataLen)); err != nil {
		//读取失败，归还一个空连接到连接池
		pool.Put(nil)
		logger.Error().Err(err).Type("errorType", err).Msg("Business read from worker failed")
		return
	}
	//跳过心跳字符，继续读取数据
	if bytes.Equal(netsvrProtocol.PongMessage, dataBuf) {
		goto loop
	}
	//归还连接
	pool.Put(conn)
	//解析网关返回的数据
	router.Reset()
	if err = proto.Unmarshal(dataBuf, router); err != nil {
		logger.Error().Err(err).Type("errorType", err).Msg("Proto unmarshal internalProtocol.Router failed")
		return
	}
	if router.Cmd != cmd {
		logger.Error().Err(err).Int32("response", int32(router.Cmd)).Int32("expect", int32(cmd)).Msg("Worker response cmd error")
		return
	}
	if err = proto.Unmarshal(router.Data, resp); err != nil {
		logger.Error().Err(err).Type("errorType", err).Msgf("Proto unmarshal %T failed", resp)
		return
	}
}
