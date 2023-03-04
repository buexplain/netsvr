package customer

import "netsvr/configs"
import "golang.org/x/time/rate"

type limiter interface {
	Allow() bool
}

type nilLimit struct {
}

func (nilLimit) Allow() bool {
	return true
}

var limitMessageNum limiter

func init() {
	if configs.Config.CustomerLimitMessageNum > 0 {
		limitMessageNum = rate.NewLimiter(rate.Limit(configs.Config.CustomerLimitMessageNum), configs.Config.CustomerLimitMessageNum)
	} else {
		limitMessageNum = nilLimit{}
	}
}
