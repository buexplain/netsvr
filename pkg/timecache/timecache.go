package timecache

import (
	"sync/atomic"
	"time"
)

var currentUnix *int64

func init() {
	var unix = time.Now().Unix()
	currentUnix = &unix
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer func() {
			ticker.Stop()
		}()
		var loop uint8 = 0
		for range ticker.C {
			if loop == 0 {
				//每隔60秒做一次修正，避免定时器走歪了
				loop = 60
				atomic.StoreInt64(currentUnix, time.Now().Unix())
			} else {
				loop--
				atomic.AddInt64(currentUnix, 1)
			}
		}
	}()
}

// Unix 返回当前时间戳
func Unix() int64 {
	return atomic.LoadInt64(currentUnix)
}
