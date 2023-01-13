package business

import (
	"github.com/RoaringBitmap/roaring"
	"github.com/buexplain/netsvr/internal/protocol/toServer/multicastByBitmap"
	"github.com/lesismal/nbio/logging"
)

// MulticastByBitmap 根据包含session id的bitmap进行组播
func MulticastByBitmap(multicastByBitmap *multicastByBitmap.MulticastByBitmap) {
	if len(multicastByBitmap.Data) == 0 || multicastByBitmap.SessionIdBitmap == "" {
		return
	}
	bitmap := roaring.Bitmap{}
	if _, err := bitmap.FromBase64(multicastByBitmap.SessionIdBitmap); err != nil {
		logging.Debug("Deserializes a bitmap from Base64 error: %v", err)
		return
	}
	peekAble := bitmap.Iterator()
	for peekAble.HasNext() {
		Catapult.Put(NewPayload(peekAble.Next(), multicastByBitmap.Data))
	}
}
