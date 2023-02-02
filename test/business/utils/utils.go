package utils

import (
	"encoding/json"
	"netsvr/test/business/protocol"
	"unsafe"
)

// NewResponse 构造一个返回给客户端响应
func NewResponse(cmd protocol.Cmd, data interface{}) []byte {
	tmp := map[string]interface{}{"cmd": cmd, "data": data}
	ret, _ := json.Marshal(tmp)
	return ret
}

// StrToReadOnlyBytes 字符串无损转字节切片，转换后的切片，不能做修改操作，因为go的字符串是不可修改的
func StrToReadOnlyBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
