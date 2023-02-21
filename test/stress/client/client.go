package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/lesismal/nbio/logging"
	"netsvr/configs"
	"netsvr/internal/heartbeat"
	"netsvr/pkg/quit"
	"time"
)

type Client struct {
	conn      *websocket.Conn
	uniqId    string
	OnMessage func(p []byte)
	OnClose   func()
	sendCh    chan []byte
	close     chan struct{}
}

func (r *Client) Heartbeat() bool {
	select {
	case <-r.close:
		return false
	default:
		r.sendCh <- heartbeat.PingMessage
		return true
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

func (r *Client) Send(p []byte) {
	select {
	case <-r.close:
		return
	default:
		r.sendCh <- p
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
			if err == nil {
				if r.OnMessage != nil {
					r.OnMessage(p)
				}
			} else {
				r.Close()
				return
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

type ConnOpenCmd struct {
	Cmd  int64 `json:"cmd"`
	Data struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	} `json:"data"`
}

func New(urlStr string) *Client {
	c, _, err := websocket.DefaultDialer.Dial(urlStr, nil)
	if err != nil {
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
	ret := ConnOpenCmd{}
	err = json.Unmarshal(p, &ret)
	if err != nil {
		_ = c.Close()
		logging.Error(err.Error())
		return nil
	}
	if ret.Cmd != 1 || ret.Data.Code != 0 {
		_ = c.Close()
		logging.Error("服务端返回了错误的结构体 %v", ret)
		return nil
	}
	client := &Client{
		conn:  c,
		close: make(chan struct{}),
	}
	return client
}

func main() {
	client := New(fmt.Sprintf("ws://%s%s", configs.Config.CustomerListenAddress, configs.Config.CustomerHandlePattern))
	if client == nil {
		return
	}
	client.OnMessage = func(p []byte) {
		fmt.Println(string(p))
	}
	client.OnClose = func() {
		quit.Execute("服务关闭我")
	}
	go client.LoopRead()
	go client.LoopSend()
	select {
	case <-quit.ClosedCh:
		logging.Info(quit.GetReason())
	}
}
