package cmd

import (
	"github.com/RoaringBitmap/roaring"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
)

// MulticastByBitmap 根据包含session id的bitmap进行组播
func MulticastByBitmap(param []byte, _ *workerManager.ConnProcessor) {
	payload := protocol.MulticastByBitmap{}
	if err := proto.Unmarshal(param, &payload); err != nil {
		logging.Error("Proto unmarshal protocol.MulticastByBitmap error: %v", err)
		return
	}
	if len(payload.Data) == 0 || payload.SessionIds == "" {
		return
	}
	bitmap := roaring.Bitmap{}
	if _, err := bitmap.FromBase64(payload.SessionIds); err != nil {
		logging.Debug("Deserializes a bitmap from Base64 error: %v", err)
		return
	}
	peekAble := bitmap.Iterator()
	for peekAble.HasNext() {
		Catapult.Put(NewPayload(peekAble.Next(), payload.Data))
	}
}
