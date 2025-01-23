package netSvrPool

import (
	"encoding/binary"
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/v4/netsvr"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"netsvr/test/pkg/utils/netSvrPool/connPool"
	"time"
)

var pool *connPool.ConnPool
var logger zerolog.Logger

// Init 初始化与网关交互所需的连接池 TODO 删除
func Init(size int, workerListenAddress string, heartbeatMessage []byte) {
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
		bf := make([]byte, 4+len(heartbeatMessage))
		binary.BigEndian.PutUint32(bf, uint32(len(heartbeatMessage)))
		copy(bf[4:], heartbeatMessage)
		_, err := conn.Write(bf)
		if err != nil {
			logger.Error().Err(err).Type("errorType", err).Msg("Business send to worker failed")
			return false
		}
		return true
	}, time.Second*25)
}

// Request 负责与网关进程交互的函数，类似于http请求方法，发送一个req请求，接收一个resp响应
func Request(req proto.Message, cmd netsvrProtocol.Cmd, resp proto.Message) {
	data := make([]byte, 8)
	//先写业务层的cmd
	binary.BigEndian.PutUint32(data[4:8], uint32(cmd))
	var err error
	if req != nil {
		//请求体不为空，则将其编码进去
		pm := proto.MarshalOptions{}
		data, err = pm.MarshalAppend(data, req)
		if err != nil {
			logger.Error().Err(err).Msgf("Proto marshal %T failed", req)
			return
		}
	}
	//再写包头
	binary.BigEndian.PutUint32(data[0:4], uint32(len(data)-4))
	//获取与网关建立好的连接
	conn := pool.Get()
	//将数据发送到网关
	var writeLen int
	for {
		writeLen, err = conn.Write(data)
		if err != nil {
			//写入数据失败，tcp管道已污染，视为对端已经无法拆包，归还一个空连接到连接池
			pool.Put(conn, false)
			logger.Error().Err(err).Msg("Business send to worker failed")
			return
		}
		//没有错误，但是只写入部分数据，继续写入
		if writeLen < len(data) {
			data = data[writeLen:]
			continue
		}
		//写入成功
		break
	}
	//不需要接收数据，直接退出
	if resp == nil {
		//归还连接
		pool.Put(conn, true)
		return
	}
	//需要接收网关返回的数据数据
	data = make([]byte, 4)
	if _, err = io.ReadFull(conn, data); err != nil {
		//读取失败，归还一个空连接到连接池
		pool.Put(conn, false)
		logger.Error().Err(err).Msg("Business read from worker failed")
		return
	}
	dataLen := binary.BigEndian.Uint32(data)
	data = make([]byte, dataLen)
	if _, err = io.ReadAtLeast(conn, data, int(dataLen)); err != nil {
		//读取失败，归还一个空连接到连接池
		pool.Put(conn, false)
		logger.Error().Err(err).Msg("Business read from worker failed")
		return
	}
	//归还连接
	pool.Put(conn, true)
	//解析网关返回的数据
	if len(data) < 4 {
		return
	}
	//跳过包头的cmd，直接解析数据部分
	if err = proto.Unmarshal(data[4:], resp); err != nil {
		logger.Error().Err(err).Type("errorType", err).Msgf("Proto unmarshal %T failed", resp)
		return
	}
}
