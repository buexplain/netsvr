package heartbeat

import (
	"github.com/antlabs/timer"
	"netsvr/pkg/quit"
)

var PingMessage []byte = []byte("~H27890027B~")
var PongMessage []byte = []byte("~H278a0027B~")

var Timer timer.Timer

func init() {
	Timer = timer.NewTimer()
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
