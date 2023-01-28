package business

import (
	"netsvr/internal/protocol/toServer/singleCast"
)

// SingleCast 单播
func SingleCast(singleCast *singleCast.SingleCast) {
	if singleCast.SessionId > 0 && len(singleCast.Data) > 0 {
		Catapult.Put(NewPayload(singleCast.SessionId, singleCast.Data))
	}
}
