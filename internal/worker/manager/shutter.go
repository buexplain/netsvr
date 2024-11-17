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

package manager

import (
	"netsvr/internal/log"
	"netsvr/pkg/quit"
	"sync"
)

// 监听进程退出信号，并关闭所有的与worker服务器的连接
type shutter struct {
	collect map[*ConnProcessor]struct{}
	mux     *sync.Mutex
}

func (r *shutter) Add(processor *ConnProcessor) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.collect[processor] = struct{}{}
}

func (r *shutter) Del(processor *ConnProcessor) {
	r.mux.Lock()
	defer r.mux.Unlock()
	delete(r.collect, processor)
}

var Shutter *shutter

func init() {
	Shutter = &shutter{
		collect: map[*ConnProcessor]struct{}{},
		mux:     &sync.Mutex{},
	}
	quit.Wg.Add(1)
	go func() {
		defer func() {
			log.Logger.Debug().Msg("Worker shutter coroutine is closed")
			quit.Wg.Done()
		}()
		select {
		case <-quit.Ctx.Done():
			wg := &sync.WaitGroup{}
			concurrency := make(chan struct{}, 32)
			var p *ConnProcessor
		loop:
			Shutter.mux.Lock()
			for p = range Shutter.collect {
				delete(Shutter.collect, p)
				break
			}
			Shutter.mux.Unlock()
			if p != nil {
				concurrency <- struct{}{}
				wg.Add(1)
				go func(p *ConnProcessor) {
					defer func() {
						wg.Done()
						<-concurrency
					}()
					p.ForceClose()
				}(p)
				p = nil
				goto loop
			}
			wg.Wait()
		}
	}()
}
