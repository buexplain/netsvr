package wsClient

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/lesismal/nbio/logging"
	"github.com/tidwall/gjson"
	"netsvr/internal/heartbeat"
	"netsvr/pkg/quit"
	"netsvr/test/protocol"
	"netsvr/test/utils"
	"time"
)

type connOpenCmd struct {
	Cmd  int32 `json:"cmd"`
	Data struct {
		Code    int32  `json:"code"`
		Message string `json:"message"`
		Data    string `json:"data"`
	} `json:"data"`
}

type Client struct {
	conn      *websocket.Conn
	UniqId    string
	OnMessage map[protocol.Cmd]func(payload gjson.Result)
	OnClose   func()
	sendCh    chan []byte
	close     chan struct{}
}

func New(urlStr string) *Client {
	c, _, err := websocket.DefaultDialer.Dial(urlStr, nil)
	if err != nil {
		logging.Error("连接失败" + err.Error())
		return nil
	}
	//接受连接打开的信息
	if err = c.SetReadDeadline(time.Now().Add(time.Second * 10)); err != nil {
		_ = c.Close()
		logging.Error(err.Error())
		return nil
	}
	var p []byte
	_, p, err = c.ReadMessage()
	if err != nil {
		_ = c.Close()
		logging.Error(err.Error())
		return nil
	}
	ret := connOpenCmd{}
	err = json.Unmarshal(p, &ret)
	if err != nil {
		_ = c.Close()
		logging.Error(err.Error())
		return nil
	}
	if ret.Cmd != int32(protocol.RouterRespConnOpen) || ret.Data.Code != 0 {
		_ = c.Close()
		logging.Error("服务端返回了错误的结构体 connOpenCmd--> %v", ret)
		return nil
	}
	client := &Client{
		conn:      c,
		UniqId:    ret.Data.Data,
		OnMessage: map[protocol.Cmd]func(payload gjson.Result){},
		sendCh:    make(chan []byte, 10),
		close:     make(chan struct{}),
	}
	return client
}

func (r *Client) Heartbeat() {
	select {
	case <-r.close:
		return
	default:
		r.sendCh <- heartbeat.PingMessage
	}
}

func (r *Client) Close() {
	select {
	case <-r.close:
		return
	default:
		close(r.close)
		_ = r.conn.Close()
		if r.OnClose != nil {
			r.OnClose()
		}
	}
}

func (r *Client) Send(cmd protocol.Cmd, data interface{}) {
	select {
	case <-r.close:
		return
	default:
		var err error
		var dataBytes []byte
		if data != nil {
			dataBytes, err = json.Marshal(data)
			if err != nil {
				logging.Error("格式化请求的数据对象错误 %v", err)
				return
			}
		}
		var dataStr string
		if dataBytes == nil {
			dataStr = "{}"
		} else {
			dataStr = utils.BytesToReadOnlyString(dataBytes)
		}
		tmp := map[string]interface{}{"cmd": cmd, "data": dataStr}
		var ret []byte
		ret, err = json.Marshal(tmp)
		if err != nil {
			logging.Error("格式化请求的路由对象错误 %v", err)
			return
		}
		b := make([]byte, 0, len(ret)+3)
		b = append(b, []byte("001")...)
		b = append(b, ret...)
		r.sendCh <- b
	}
}

func (r *Client) LoopRead() {
	for {
		select {
		case <-quit.Ctx.Done():
		case <-r.close:
			return
		default:
			if err := r.conn.SetReadDeadline(time.Now().Add(time.Second * 60)); err != nil {
				r.Close()
				return
			}
			_, p, err := r.conn.ReadMessage()
			if err != nil {
				r.Close()
				return
			}
			if bytes.Equal(p, heartbeat.PongMessage) {
				return
			}
			ret := gjson.GetManyBytes(p, "cmd", "data")
			if len(ret) == 2 && ret[0].Type == gjson.Number && ret[1].Type == gjson.JSON {
				cmd := protocol.Cmd(ret[0].Int())
				if c, ok := r.OnMessage[cmd]; ok {
					c(ret[1])
				} else {
					logging.Debug("未找到回调函数 %s", ret[1].Raw)
				}
			} else {
				logging.Error("结构体不合法 %s", p)
			}
		}
	}
}

func (r *Client) LoopSend() {
	for {
		select {
		case <-quit.Ctx.Done():
		case <-r.close:
			return
		case p := <-r.sendCh:
			if err := r.conn.SetWriteDeadline(time.Now().Add(time.Second * 60)); err != nil {
				r.Close()
				return
			}
			if err := r.conn.WriteMessage(1, p); err != nil {
				r.Close()
				return
			}
		}
	}
}
