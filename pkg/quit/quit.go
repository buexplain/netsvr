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
			Execute(fmt.Sprintf("收到进程结束信号(%d %s)", s, s))
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
	closeReason = reason
	select {
	case <-ClosedCh:
		return
	default:
		close(ClosedCh)
	}
}
