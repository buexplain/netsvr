package connProcessor

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	toServerRouter "github.com/buexplain/netsvr/internal/protocol/toServer/router"
	"github.com/buexplain/netsvr/internal/protocol/toServer/setUserLoginStatus"
	"github.com/buexplain/netsvr/internal/protocol/toServer/singleCast"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/connClose"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/connOpen"
	toWorkerRouter "github.com/buexplain/netsvr/internal/protocol/toWorker/router"
	"github.com/buexplain/netsvr/internal/protocol/toWorker/transfer"
	"github.com/buexplain/netsvr/internal/worker/heartbeat"
	"github.com/buexplain/netsvr/pkg/utils"
	"github.com/buexplain/netsvr/test/worker/protocol"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"io"
	"net"
	"time"
)

type ConnProcessor struct {
	conn    net.Conn
	writeCh chan []byte
	closeCh chan struct{}
}

func NewConnProcessor(conn net.Conn) *ConnProcessor {
	tmp := &ConnProcessor{conn: conn, writeCh: make(chan []byte, 10), closeCh: make(chan struct{})}
	return tmp
}

func (r *ConnProcessor) Heartbeat() {
	t := time.NewTicker(time.Duration(55) * time.Second)
	defer func() {
		t.Stop()
	}()
	for {
		<-t.C
		_, _ = r.Write(heartbeat.PingMessage)
	}
}

func (r *ConnProcessor) Send() {
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

func (r *ConnProcessor) Write(data []byte) (n int, err error) {
	select {
	case <-r.closeCh:
		return 0, nil
	default:
		r.writeCh <- data
		return len(data), nil
	}
}

func (r *ConnProcessor) Read() {
	dataLenBuf := make([]byte, 4)
	for {
		dataLenBuf[0] = 0
		dataLenBuf[1] = 0
		dataLenBuf[2] = 0
		dataLenBuf[3] = 0
		if _, err := io.ReadFull(r.conn, dataLenBuf); err != nil {
			_ = r.conn.Close()
			close(r.writeCh)
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
		//客户端发来的业务请求
		if toWorkerRoute.Cmd == toWorkerRouter.Cmd_Transfer {
			//解析网关的转发过来的对象
			tf := &transfer.Transfer{}
			if err := proto.Unmarshal(toWorkerRoute.Data, tf); err != nil {
				logging.Error("Proto unmarshal toWorkerRoute.Data error: %#v", err)
				continue
			}
			//解析业务数据
			clientRoute := protocol.ParseClientRouter(tf.Data)
			if clientRoute == nil {
				continue
			}
			//判断业务路由
			if clientRoute.Cmd == protocol.RouterLogin {
				r.login(tf.SessionId, clientRoute.Data)
				continue
			}
		}
	}
}

func (r *ConnProcessor) connOpen(toWorkerRoute *toWorkerRouter.Router) {
	co := &connOpen.ConnOpen{}
	if err := proto.Unmarshal(toWorkerRoute.Data, co); err != nil {
		logging.Error("解压出具体的业务数据失败: %#v", err)
		return
	}
	logging.Debug("客户端连接打开 sessionId --> %d", co.SessionId)
	toServerRoute := &toServerRouter.Router{}
	toServerRoute.Cmd = toServerRouter.Cmd_SingleCast
	//构造单播数据
	ret := &singleCast.SingleCast{}
	ret.SessionId = co.SessionId
	tmp := NewResponse(protocol.RouterRespConnOpen, map[string]interface{}{"sessionId": co.SessionId})
	ret.Data = tmp
	//将业务数据放到路由上
	toServerRoute.Data, _ = proto.Marshal(ret)
	data, _ := proto.Marshal(toServerRoute)
	_, _ = r.Write(data)
}

func (r *ConnProcessor) connClose(toWorkerRoute *toWorkerRouter.Router) {
	cls := &connClose.ConnClose{}
	if err := proto.Unmarshal(toWorkerRoute.Data, cls); err != nil {
		logging.Error("解压出具体的业务数据失败: %#v", err)
		return
	}
	logging.Debug("客户端连接关闭 user --> %s sessionId --> %d", cls.User, cls.SessionId)
}

// 登录
func (r *ConnProcessor) login(sessionId uint32, data string) {
	login := new(protocol.Login)
	if err := json.Unmarshal(utils.StrToBytes(data), login); err != nil {
		logging.Debug("Parse login request error: %#v", err)
		return
	}
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//要求网关设定登录状态
	toServerRoute.Cmd = toServerRouter.Cmd_SetUserLoginStatus
	//构建一个包含登录状态相关是业务对象
	ret := &setUserLoginStatus.SetUserLoginStatus{}
	ret.SessionId = sessionId
	//校验账号密码，判断是否登录成功
	if login.Username == "刘备" && login.Password == "123456" {
		ret.LoginStatus = true
		//响应给客户端的数据
		ret.Data = NewResponse(protocol.RouterLogin, map[string]interface{}{"code": 0, "message": "登录成功"})
		//写入到网关的数据
		ret.UserInfo = "用户名:" + login.Username + ",用户密码:" + login.Password
	} else {
		ret.LoginStatus = false
		ret.Data = NewResponse(protocol.RouterLogin, map[string]interface{}{"code": 1, "message": "登录失败，账号或密码错误"})
	}
	//将业务对象放到路由上
	toServerRoute.Data, _ = proto.Marshal(ret)
	resp, _ := proto.Marshal(toServerRoute)
	//回写给网关服务器
	_, _ = r.Write(resp)
}

func NewResponse(cmd int, data interface{}) []byte {
	tmp := map[string]interface{}{"cmd": cmd, "data": data}
	ret, _ := json.Marshal(tmp)
	return ret
}
