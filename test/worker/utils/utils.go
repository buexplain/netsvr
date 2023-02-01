package utils

import (
	"encoding/json"
	"netsvr/test/worker/protocol"
)

func NewResponse(cmd protocol.Cmd, data interface{}) []byte {
	tmp := map[string]interface{}{"cmd": cmd, "data": data}
	ret, _ := json.Marshal(tmp)
	return ret
}
