package business

import (
	"netsvr/internal/customer/session"
	"netsvr/internal/protocol/toServer/publish"
	"netsvr/pkg/quit"
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
	quit.Wg.Add(1)
	go func() {
		defer func() {
			_ = recover()
			quit.Wg.Done()
		}()
		peekAble := bitmap.Iterator()
		for peekAble.HasNext() {
			Catapult.Put(NewPayload(peekAble.Next(), publish.Data))
		}
	}()
}
