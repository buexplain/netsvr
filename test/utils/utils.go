package utils

import (
	"encoding/json"
	"math/rand"
	"netsvr/test/protocol"
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
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// BytesToReadOnlyString 字节切片无损转字符串
func BytesToReadOnlyString(bt []byte) string {
	if len(bt) == 0 {
		return ""
	}
	return unsafe.String(&bt[0], len(bt))
}

var charPool = [36]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0', 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z'}

// GetRandStr n=4有1413720种排列 n=3有42840种排列，n=2有1260种排列
func GetRandStr(n int) string {
	s := make([]byte, 0, n)
	for {
		n--
		if n < 0 {
			break
		}
		s = append(s, charPool[rand.Intn(35)])
	}
	return string(s)
}
