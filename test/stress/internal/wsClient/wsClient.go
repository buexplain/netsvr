/**
* Copyright 2023 buexplain@qq.com
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
	netsvrProtocol "github.com/buexplain/netsvr-protocol-go/netsvr"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"io"
	"netsvr/pkg/quit"
	"netsvr/test/pkg/protocol"
	"netsvr/test/pkg/utils"
	"netsvr/test/stress/configs"
	"netsvr/test/stress/internal/log"
	"netsvr/test/stress/internal/wsMetrics"
	"netsvr/test/stress/internal/wsShutter"
	"netsvr/test/stress/internal/wsTimer"
	"sync"
	"time"
)

type connOpenCmd struct {
	Cmd  int32 `json:"cmd"`
	Data struct {
		Code int32 `json:"code"`
		Data struct {
			RawQuery      string   `json:"rawQuery"`
			SubProtocol   []string `json:"subProtocol"`
			UniqID        string   `json:"uniqId"`
			XForwardedFor string   `json:"xForwardedFor"`
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
	topicPublishIndex     int
	OnMessage             map[protocol.Cmd]func(payload gjson.Result)
	OnClose               func()
	sendCh                chan []byte
	close                 chan struct{}
	mux                   *sync.RWMutex
	wsStatus              *wsMetrics.WsStatus
}

func New(urlStr string, status *wsMetrics.WsStatus, option func(ws *Client)) *Client {
	c, resp, err := websocket.DefaultDialer.DialContext(quit.Ctx, urlStr, nil)
	if err != nil {
		if resp != nil {
			respByte, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			log.Logger.Error().Err(err).Int("respStatusCode", resp.StatusCode).Str("respBody", string(respByte)).Msg("websocket dial failed")
		} else {
			log.Logger.Error().Err(err).Msg("websocket dial failed")
		}
		return nil
	}
	//接收连接打开的信息
	if err = c.SetReadDeadline(time.Now().Add(time.Second * 10)); err != nil {
		_ = c.Close()
		log.Logger.Error().Err(err).Msg("websocket SetReadDeadline failed")
		return nil
	}
	ret := connOpenCmd{}
	for {
		var p []byte
		_, p, err = c.ReadMessage()
		if err != nil {
			_ = c.Close()
			log.Logger.Error().Err(err).Msg("websocket ReadMessage failed")
			return nil
		}
		//先检查cmd是啥
		cmdRet := gjson.GetBytes(p, "cmd")
		cm := cmdRet.Int()
		if cm <= 0 {
			_ = c.Close()
			log.Logger.Error().Err(err).Str("receiveData", string(p)).Msgf("websocket parse message failed")
			return nil
		}
		if cm != int64(protocol.RouterRespConnOpen) {
			continue
		}
		//确定是连接打开的命令，再解析结构体
		err = json.Unmarshal(p, &ret)
		if err != nil {
			_ = c.Close()
			log.Logger.Error().Err(err).Str("receiveData", string(p)).Msgf("websocket parse message failed")
			return nil
		}
		if ret.Data.Code != 0 {
			_ = c.Close()
			log.Logger.Error().Msgf("unknown structure connOpenCmd--> %s", string(p))
			return nil
		}
		break
	}
	client := &Client{
		conn:        c,
		UniqId:      ret.Data.Data.UniqID,
		topics:      make([]string, 0, 0),
		LocalUniqId: ret.Data.Data.UniqID,
		OnMessage:   map[protocol.Cmd]func(payload gjson.Result){},
		sendCh:      make(chan []byte, 10),
		close:       make(chan struct{}),
		mux:         &sync.RWMutex{},
		wsStatus:    status,
	}
	if option != nil {
		option(client)
	}
	//开启读写协程
	go client.LoopSend()
	go client.LoopRead()
	//在线信息加1
	client.wsStatus.Online.Inc(1)
	//开启心跳
	if configs.Config.Heartbeat > 0 {
		wsTimer.WsTimer.ScheduleFunc(time.Second*time.Duration(configs.Config.Heartbeat), func() {
			client.Heartbeat()
		})
	}
	//添加到进程结束管理模块
	wsShutter.WsShutter.Add(client)
	return client
}

func (r *Client) Heartbeat() {
	select {
	case <-r.close:
		return
	default:
		r.sendCh <- netsvrProtocol.PingMessage
	}
}

func (r *Client) Close() {
	select {
	case <-r.close:
		return
	default:
		close(r.close)
		_ = r.conn.Close()
		if r.wsStatus != nil {
			r.wsStatus.Online.Dec(1)
		}
		if r.OnClose != nil {
			r.OnClose()
		}
	}
}

// InitTopic 伪造主题
func (r *Client) InitTopic(topicNum int, topicLen int) {
	topics := make([]string, 0, topicNum)
	for i := 0; i < topicNum; i++ {
		topics = append(topics, utils.GetRandStr(topicLen))
	}
	r.topics = topics
}

func (r *Client) GetPublishTopic(topicNum int) []string {
	r.mux.Lock()
	defer r.mux.Unlock()
	ret := make([]string, 0, topicNum)
	//强制给第零个主题发布消息，因为第零个主题不会被取消订阅
	ret = append(ret, r.topics[0])
	for {
		if len(ret) == topicNum {
			break
		}
		r.topicPublishIndex++
		if r.topicPublishIndex == len(r.topics) {
			r.topicPublishIndex = 0
		}
		//如果第零个与当前拿到的是一样的，则退出循环
		if ret[0] == r.topics[r.topicPublishIndex] {
			break
		}
		ret = append(ret, r.topics[r.topicPublishIndex])
	}
	return ret
}

func (r *Client) GetSubscribeTopic(topicNum int) []string {
	r.mux.Lock()
	defer r.mux.Unlock()
	ret := make([]string, 0, topicNum)
	//强制订阅第零个主题
	if r.topicSubscribeIndex == 0 {
		ret = append(ret, r.topics[0])
	}
	for {
		if len(ret) == topicNum {
			break
		}
		r.topicSubscribeIndex++
		if r.topicSubscribeIndex == len(r.topics) {
			r.topicSubscribeIndex = 0
		}
		//如果第零个与当前拿到的是一样的，则退出循环
		if len(ret) > 0 && ret[0] == r.topics[r.topicSubscribeIndex] {
			break
		}
		ret = append(ret, r.topics[r.topicSubscribeIndex])
	}
	return ret
}

func (r *Client) GetUnsubscribeTopic(topicNum int) []string {
	r.mux.Lock()
	defer r.mux.Unlock()
	//如果只有一个备选主题，则没有可取消订阅的主题
	if len(r.topics) == 1 {
		return nil
	}
	ret := make([]string, 0, topicNum)
	for {
		if len(ret) == topicNum {
			break
		}
		r.topicUnsubscribeIndex++
		if r.topicUnsubscribeIndex == len(r.topics) {
			//这里忽略第零个主题，因为第零个主题会被用于消息发布，所以不能取消订阅
			r.topicUnsubscribeIndex = 1
		}
		//如果第零个与当前拿到的是一样的，则退出循环
		if len(ret) > 0 && ret[0] == r.topics[r.topicUnsubscribeIndex] {
			break
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
		b = append(b, configs.Config.WorkerIdBytes...)
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
		case <-r.close:
			return
		default:
			_, p, err := r.conn.ReadMessage()
			if err != nil {
				select {
				case <-quit.Ctx.Done():
					break
				default:
					log.Logger.Debug().Err(err).Msg("读取服务器消息失败")
				}
				r.Close()
				return
			}
			if r.wsStatus != nil {
				r.wsStatus.Receive.Inc(int64(len(p)))
			}
			if bytes.Equal(p, netsvrProtocol.PongMessage) {
				continue
			}
			ret := gjson.GetManyBytes(p, "cmd", "data")
			if len(ret) == 2 && ret[0].Type == gjson.Number && ret[1].Type == gjson.JSON {
				if r.OnMessage != nil {
					cmd := protocol.Cmd(ret[0].Int())
					if c, ok := r.OnMessage[cmd]; ok {
						c(ret[1])
					} else {
						log.Logger.Debug().Msgf("未找到回调函数 cmd --> %d data --> %s", cmd, ret[1].Raw)
					}
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
		case <-r.close:
			return
		case p := <-r.sendCh:
			if err := r.conn.SetWriteDeadline(time.Now().Add(time.Second * 5)); err != nil {
				r.Close()
				return
			}
			if err := r.conn.WriteMessage(websocket.TextMessage, p); err != nil {
				r.Close()
				return
			}
			if r.wsStatus != nil {
				r.wsStatus.Send.Inc(int64(len(p)))
			}
		}
	}
}
