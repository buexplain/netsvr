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

package quit

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Ctx 任意协程如果需要man函数触发退出，都需要case该ctx
var Ctx context.Context

// Cancel man函数专用，触发子协程关闭信号
var Cancel context.CancelFunc

// Wg 任意协程如果需要man函数等待，都需要加入该wg
var Wg *sync.WaitGroup

var ClosedCh chan struct{}

// 锁
var lock *sync.Mutex

// 进程关闭的原因
var closeReason string

func init() {
	Ctx, Cancel = context.WithCancel(context.Background())
	Wg = &sync.WaitGroup{}
	lock = new(sync.Mutex)
	ClosedCh = make(chan struct{})
}

// 监听外部关闭信号
func init() {
	signalCH := make(chan os.Signal, 1)
	signal.Notify(signalCH, []os.Signal{
		syscall.SIGHUP,  //hangup
		syscall.SIGTERM, //terminated
		syscall.SIGINT,  //interrupt
		syscall.SIGQUIT, //quit
	}...)
	go func() {
		for s := range signalCH {
			if s == syscall.SIGHUP {
				//忽略session断开信号
				continue
			}
			Execute(fmt.Sprintf("receive signal(%d %s)", s, s))
		}
	}()
}

func GetReason() string {
	lock.Lock()
	defer lock.Unlock()
	return closeReason
}

// Execute 发出关闭进程信息
func Execute(reason string) {
	lock.Lock()
	defer lock.Unlock()
	select {
	case <-ClosedCh:
		return
	default:
		closeReason = reason
		close(ClosedCh)
	}
}
