package client

import (
	"encoding/json"
	"github.com/RoaringBitmap/roaring"
	"github.com/lesismal/nbio/logging"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/protocol/toServer/multicast"
	"netsvr/internal/protocol/toServer/multicastByBitmap"
	toServerRouter "netsvr/internal/protocol/toServer/router"
	"netsvr/pkg/utils"
	"netsvr/test/worker/connProcessor"
	"netsvr/test/worker/protocol"
	"netsvr/test/worker/userDb"
	workerUtils "netsvr/test/worker/utils"
)

// Multicast 组播
func Multicast(currentSessionId uint32, userStr string, param string, processor *connProcessor.ConnProcessor) {
	//解析客户端发来的数据
	target := new(protocol.Multicast)
	if err := json.Unmarshal(utils.StrToBytes(param), target); err != nil {
		logging.Error("Parse protocol.Multicast request error: %v", err)
		return
	}
	currentUser := userDb.ParseNetSvrInfo(userStr)
	//构建一个发给网关的路由
	toServerRoute := &toServerRouter.Router{}
	//查询目标用户的sessionId
	bitmap := roaring.Bitmap{}
	for _, userId := range target.UserIds {
		sessionId := userDb.Collect.GetSessionId(userId)
		if sessionId > 0 {
			//把找到的在线session id都存储在bitmap对象中
			bitmap.Add(sessionId)
		}
	}
	//没有找到任何目标用户，通知发送方，目标用户不存在
	if bitmap.GetCardinality() == 0 {
		//这里可以采用单播的命令 toServerRouter.Cmd_SingleCast，采用组播是为了测试网关的组播功能是否正常
		//告诉网关要进行一次基于组播操作
		toServerRoute.Cmd = toServerRouter.Cmd_Multicast
		//构造网关需要的组播数据
		ret := &multicast.Multicast{}
		//目标用户不存在，返回信息给到发送者
		ret.SessionIds = []uint32{currentSessionId}
		ret.Data = workerUtils.NewResponse(protocol.RouterMulticast, map[string]interface{}{"code": 1, "message": "未找到目标用户"})
		toServerRoute.Data, _ = proto.Marshal(ret)
		pt, _ := proto.Marshal(toServerRoute)
		processor.Send(pt)
		return
	}
	//告诉网关要进行一次基于bitmap的组播操作
	toServerRoute.Cmd = toServerRouter.Cmd_MulticastByBitmap
	//构造网关需要的组播数据
	ret := &multicastByBitmap.MulticastByBitmap{}
	//将bitmap对象序列化成字符串
	ret.SessionIdBitmap, _ = bitmap.ToBase64()
	//构造客户端需要的数据
	msg := map[string]interface{}{"fromUser": currentUser.Name, "message": target.Message}
	ret.Data = workerUtils.NewResponse(protocol.RouterMulticast, map[string]interface{}{"code": 0, "message": "收到一条信息", "data": msg})
	toServerRoute.Data, _ = proto.Marshal(ret)
	//发送给网关
	pt, _ := proto.Marshal(toServerRoute)
	processor.Send(pt)
}
