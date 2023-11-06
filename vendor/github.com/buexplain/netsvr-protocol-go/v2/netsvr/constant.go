package netsvr

//借个位置占坑，才能生成依赖关系
import _ "github.com/buexplain/netsvr-protocol/v2"

// WorkerIdMax business的编号范围
const WorkerIdMax = 999
const WorkerIdMin = 1

// PingMessage 发起心跳的字符串
var PingMessage = []byte("~3yPvmnz~")

// PongMessage 响应心跳的字符串
var PongMessage = []byte("~u38NvZ~")
