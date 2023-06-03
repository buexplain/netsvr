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

package wsShutter

import (
	"netsvr/pkg/quit"
	"netsvr/test/stress/internal/log"
	"sync"
)

type CloseInterface interface {
	Close()
}

// 监听进程退出信号，并关闭所有的websocket客户端
type wsShutter struct {
	collect map[CloseInterface]struct{}
	mux     *sync.Mutex
}

func (r *wsShutter) Add(ws CloseInterface) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.collect[ws] = struct{}{}
}

func (r *wsShutter) Del(ws CloseInterface) {
	r.mux.Lock()
	defer r.mux.Unlock()
	delete(r.collect, ws)
}

var WsShutter *wsShutter

func init() {
	WsShutter = &wsShutter{
		collect: map[CloseInterface]struct{}{},
		mux:     &sync.Mutex{},
	}
	quit.Wg.Add(1)
	go func() {
		defer func() {
			log.Logger.Debug().Msg("Stress wsShutter coroutine is closed")
			quit.Wg.Done()
		}()
		select {
		case <-quit.Ctx.Done():
			wg := &sync.WaitGroup{}
			concurrency := make(chan struct{}, 100)
			var ws CloseInterface
		loop:
			WsShutter.mux.Lock()
			for ws = range WsShutter.collect {
				delete(WsShutter.collect, ws)
				break
			}
			WsShutter.mux.Unlock()
			if ws != nil {
				concurrency <- struct{}{}
				wg.Add(1)
				go func(ws CloseInterface) {
					defer func() {
						wg.Done()
						<-concurrency
					}()
					ws.Close()
				}(ws)
				ws = nil
				goto loop
			}
			wg.Wait()
		}
	}()
}
