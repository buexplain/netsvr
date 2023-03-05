// Package timer 定时器模块
// 提供定时间隔或者是延时处理服务
package timer

import (
	gTimer "github.com/antlabs/timer"
	"netsvr/pkg/quit"
)

var Timer gTimer.Timer

func init() {
	Timer = gTimer.NewTimer()
	go Timer.Run()
	//收到进程结束信号，则立马停止定时器
	go func() {
		defer func() {
			_ = recover()
		}()
		<-quit.ClosedCh
		Timer.Stop()
	}()
}
