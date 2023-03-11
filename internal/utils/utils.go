package utils

import (
	"encoding/binary"
	"math/rand"
	"net/http"
	"netsvr/configs"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

// BytesToInt 截取前digit个字节并转成int
// 该方法比strconv.Atoi快三倍,单个耗时在8.557纳秒
func BytesToInt(data []byte, digit int) int {
	if len(data) < digit {
		return 0
	}
	var step = 1
	var ret = 0
	digit--
	for ; digit >= 0; digit-- {
		ret += byteToInt(data[digit]) * step
		step *= 10
	}
	return ret
}

func byteToInt(b byte) int {
	switch b {
	case '0':
		return 0
	case '1':
		return 1
	case '2':
		return 2
	case '3':
		return 3
	case '4':
		return 4
	case '5':
		return 5
	case '6':
		return 6
	case '7':
		return 7
	case '8':
		return 8
	case '9':
		return 9
	default:
		return 0
	}
}

var uniqIdSuffix = uint32(rand.Int31())

// UniqId 生成一个唯一id，服务编号+时间戳+自增值，共18个字符
func UniqId() string {
	buf := make([]byte, 18)
	buf[9] = configs.Config.ServerId
	binary.BigEndian.PutUint32(buf[10:], uint32(time.Now().Unix()))
	binary.BigEndian.PutUint32(buf[14:], atomic.AddUint32(&uniqIdSuffix, 1))
	var j int8
	for _, v := range buf[9:] {
		//这里保持字母大写，作为服务编号之外的第二个特征吧
		buf[j] = "0123456789ABCDEF"[v>>4]
		buf[j+1] = "0123456789ABCDEF"[v&0x0f]
		j += 2
	}
	return unsafe.String(&buf[0], 18)
}

func ParseSubProtocols(r *http.Request) []string {
	h := strings.TrimSpace(r.Header.Get("Sec-Websocket-Protocol"))
	if h == "" {
		return nil
	}
	protocols := strings.Split(h, ",")
	for i := range protocols {
		protocols[i] = strings.TrimSpace(protocols[i])
	}
	return protocols
}
