package cmd

import (
	"github.com/RoaringBitmap/roaring"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/protocol/toServer/multicastByBitmap"
	workerManager "netsvr/internal/worker/manager"
)

// MulticastByBitmap 根据包含session id的bitmap进行组播
func MulticastByBitmap(param []byte, _ *workerManager.ConnProcessor) {
	req := multicastByBitmap.MulticastByBitmap{}
	if err := proto.Unmarshal(param, &req); err != nil {
		logging.Error("Proto unmarshal multicastByBitmap.MulticastByBitmap error: %v", err)
		return
	}
	if len(req.Data) == 0 || req.SessionIdBitmap == "" {
		return
	}
	bitmap := roaring.Bitmap{}
	if _, err := bitmap.FromBase64(req.SessionIdBitmap); err != nil {
		logging.Debug("Deserializes a bitmap from Base64 error: %v", err)
		return
	}
	peekAble := bitmap.Iterator()
	for peekAble.HasNext() {
		Catapult.Put(NewPayload(peekAble.Next(), req.Data))
	}
}
