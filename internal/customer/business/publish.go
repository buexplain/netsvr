package business

import (
	"github.com/buexplain/netsvr/internal/customer/session"
	"github.com/buexplain/netsvr/internal/protocol/toServer/publish"
)

// Publish 发布
func Publish(publish *publish.Publish) {
	if len(publish.Data) == 0 || publish.Topic == "" {
		return
	}
	bitmap := session.Topics.Get(publish.Topic)
	if bitmap == nil {
		return
	}
	peekAble := bitmap.Iterator()
	for peekAble.HasNext() {
		Catapult.Put(NewPayload(peekAble.Next(), publish.Data))
	}
}
