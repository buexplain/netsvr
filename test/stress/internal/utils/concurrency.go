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

package utils

import (
	"golang.org/x/time/rate"
	"netsvr/pkg/quit"
	"netsvr/test/stress/internal/log"
	"sync"
)

func Concurrency(iteratorNum int, secondSpeed int, callback func()) {
	if secondSpeed <= 0 {
		secondSpeed = 10
	}
	l := rate.NewLimiter(rate.Limit(secondSpeed), secondSpeed)
	ch := make(chan struct{}, int(l.Limit()))
	wg := &sync.WaitGroup{}
	for i := 0; i < iteratorNum; i++ {
		if err := l.Wait(quit.Ctx); err != nil {
			return
		}
		select {
		case <-quit.Ctx.Done():
			return
		default:
		}
		ch <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() {
				if err := recover(); err != nil {
					log.Logger.Error().Msgf("%v", err)
				}
				//先ch再wg
				<-ch
				wg.Done()
			}()
			callback()
		}()
	}
	//先wg再ch
	wg.Wait()
	close(ch)
}
