package business

import (
	"github.com/buexplain/netsvr/internal/protocol/toServer/multicast"
)

// Multicast 组播
func Multicast(multicast *multicast.Multicast) {
	if len(multicast.Data) == 0 {
		return
	}
	for _, sessionId := range multicast.SessionIds {
		Catapult.Put(NewPayload(sessionId, multicast.Data))
	}
}
