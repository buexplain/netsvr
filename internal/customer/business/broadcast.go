package business

import (
	"github.com/buexplain/netsvr/internal/customer/session"
	"github.com/buexplain/netsvr/internal/protocol/toServer/broadcast"
	"github.com/buexplain/netsvr/pkg/quit"
)

// Broadcast 广播
func Broadcast(broadcast *broadcast.Broadcast) {
	if len(broadcast.Data) == 0 {
		return
	}
	bitmap := session.Id.GetAllocated()
	quit.Wg.Add(1)
	go func() {
		defer func() {
			quit.Wg.Done()
		}()
		peekAble := bitmap.Iterator()
		for peekAble.HasNext() {
			Catapult.Put(NewPayload(peekAble.Next(), broadcast.Data))
		}
	}()
}
