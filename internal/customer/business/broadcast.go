package business

import (
	"github.com/buexplain/netsvr/internal/customer/session"
	"github.com/buexplain/netsvr/internal/protocol/toServer/broadcast"
)

// Broadcast 广播
func Broadcast(broadcast *broadcast.Broadcast) {
	if len(broadcast.Data) == 0 {
		return
	}
	peekAble := session.Id.GetAllocated().Iterator()
	for peekAble.HasNext() {
		Catapult.Put(NewPayload(peekAble.Next(), broadcast.Data))
	}
}
