/**
* Copyright 2022 buexplain@qq.com
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package wsClient

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"netsvr/pkg/heartbeat"
	"netsvr/pkg/quit"
	"netsvr/test/pkg/protocol"
	"netsvr/test/pkg/utils"
	"netsvr/test/stress/internal/log"
	"sync"
	"time"
)

type connOpenCmd struct {
	Cmd  int32 `json:"cmd"`
	Data struct {
		Code int32 `json:"code"`
		Data struct {
			RawQuery      string   `json:"rawQuery"`
			RemoteIP      string   `json:"remoteIP"`
			SubProtocol   []string `json:"subProtocol"`
			UniqID        string   `json:"uniqId"`
			XForwardedFor string   `json:"xForwardedFor"`
			XRealIP       string   `json:"xRealIP"`
		} `json:"data"`
		Message string `json:"message"`
	} `json:"data"`
}

type Client struct {
	conn                  *websocket.Conn
	UniqId                string
	LocalUniqId           string
	topics                []string
	topicSubscribeIndex   int
	topicUnsubscribeIndex int
	OnMessage             map[protocol.Cmd]func(payload gjson.Result)
	OnClose               func()
	sendCh                chan []byte
	close                 chan struct{}
	mux                   *sync.RWMutex
}

func New(urlStr string) *Client {
	c, _, err := websocket.DefaultDialer.Dial(urlStr, nil)
	if err != nil {
		log.Logger.Error().Msgf("连接失败" + err.Error())
		return nil
	}
	//接受连接打开的信息
	if err = c.SetReadDeadline(time.Now().Add(time.Second * 10)); err != nil {
		_ = c.Close()
		log.Logger.Error().Msgf("连接服务器失败 %v", err)
		return nil
	}
	ret := connOpenCmd{}
	for {
		var p []byte
		_, p, err = c.ReadMessage()
		if err != nil {
			_ = c.Close()
			log.Logger.Error().Err(err).Msg("读取服务器消息失败")
			return nil
		}
		//先检查cmd是啥
		cmdRet := gjson.GetBytes(p, "cmd")
		cm := cmdRet.Int()
		if cm <= 0 {
			_ = c.Close()
			log.Logger.Error().Str("receiveData", string(p)).Msgf("解析服务器消息失败 %v", err)
			return nil
		}
		if cm != int64(protocol.RouterRespConnOpen) {
			continue
		}
		//确定是连接打开的命令，再解析结构体
		err = json.Unmarshal(p, &ret)
		if err != nil {
			_ = c.Close()
			log.Logger.Error().Str("receiveData", string(p)).Msgf("解析服务器消息失败 %v", err)
			return nil
		}
		if ret.Data.Code != 0 {
			_ = c.Close()
			log.Logger.Error().Msgf("服务端返回了错误的结构体 connOpenCmd--> %s", string(p))
			return nil
		}
		break
	}
	client := &Client{
		conn:        c,
		UniqId:      ret.Data.Data.UniqID,
		topics:      make([]string, 0),
		LocalUniqId: ret.Data.Data.UniqID,
		OnMessage:   map[protocol.Cmd]func(payload gjson.Result){},
		sendCh:      make(chan []byte, 10),
		close:       make(chan struct{}),
		mux:         &sync.RWMutex{},
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

// InitTopic 伪造主题
func (r *Client) InitTopic() {
	topics := make([]string, 0, 600)
	for i := 0; i < 600; i++ {
		topics = append(topics, utils.GetRandStr(2))
	}
	r.topics = topics
}

func (r *Client) GetTopic() string {
	r.mux.RLock()
	defer r.mux.RUnlock()
	return r.topics[0]
}

func (r *Client) GetSubscribeTopic() []string {
	r.mux.Lock()
	defer r.mux.Unlock()
	ret := make([]string, 0, 100)
	for {
		if len(ret) == 100 {
			break
		}
		r.topicSubscribeIndex++
		if r.topicSubscribeIndex == len(r.topics) {
			r.topicSubscribeIndex = 0
		}
		ret = append(ret, r.topics[r.topicSubscribeIndex])
	}
	return ret
}

func (r *Client) GetUnsubscribeTopic() []string {
	r.mux.Lock()
	defer r.mux.Unlock()
	ret := make([]string, 0, 50)
	for {
		if len(ret) == 50 {
			break
		}
		r.topicUnsubscribeIndex++
		if r.topicUnsubscribeIndex == len(r.topics) {
			//这里忽略第零个主题，因为第零个主题会被用于消息发布
			r.topicUnsubscribeIndex = 1
		}
		ret = append(ret, r.topics[r.topicUnsubscribeIndex])
	}
	return ret
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
				log.Logger.Error().Msgf("格式化请求的数据对象错误 %v", err)
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
			log.Logger.Error().Msgf("格式化请求的路由对象错误 %v", err)
			return
		}
		b := make([]byte, 0, len(ret)+3)
		b = append(b, []byte("001")...)
		b = append(b, ret...)
		r.sendCh <- b
	}
}

func (r *Client) LoopRead() {
	if err := r.conn.SetReadDeadline(time.Time{}); err != nil {
		r.Close()
		return
	}
	for {
		select {
		case <-quit.Ctx.Done():
		case <-r.close:
			return
		default:
			_, p, err := r.conn.ReadMessage()
			if err != nil {
				log.Logger.Debug().Err(err).Msg("读取服务器消息失败")
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
					log.Logger.Debug().Msgf("未找到回调函数 %s", ret[1].Raw)
				}
			} else {
				log.Logger.Error().Msgf("结构体不合法 %s", p)
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
			if err := r.conn.SetWriteDeadline(time.Now().Add(time.Second * 5)); err != nil {
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
