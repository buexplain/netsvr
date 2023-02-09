package utils

import (
	"encoding/binary"
	"hash/adler32"
	"math/rand"
	"netsvr/configs"
	"netsvr/pkg/timecache"
	"sync/atomic"
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

var uuidPrefix = adler32.Checksum([]byte(configs.Config.WorkerListenAddress))
var uuidSuffix = uint32(rand.Int31())

// UniqId 生成一个唯一id，网关地址+时间戳+自增值，共24个字符
func UniqId() string {
	buf := make([]byte, 24)
	binary.BigEndian.PutUint32(buf[12:], uuidPrefix)
	binary.BigEndian.PutUint32(buf[16:], uint32(timecache.Unix()))
	binary.BigEndian.PutUint32(buf[20:], atomic.AddUint32(&uuidSuffix, 1))
	var j int8
	for _, v := range buf[12:] {
		buf[j] = "0123456789abcdef"[v>>4]
		buf[j+1] = "0123456789abcdef"[v&0x0f]
		j += 2
	}
	return unsafe.String(&buf[0], 24)
}
