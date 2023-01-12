package utils

import "unsafe"

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

// StrToBytes 字符串无损转字节切片
func StrToBytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}
