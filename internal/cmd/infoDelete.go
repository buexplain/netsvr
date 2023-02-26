package cmd

import (
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
	"netsvr/pkg/utils"
)

// InfoDelete 删除连接的info信息
func InfoDelete(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.InfoDelete{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal protocol.InfoDelete error: %v", err)
		return
	}
	if payload.UniqId == "" {
		return
	}
	conn := manager.Manager.Get(payload.UniqId)
	if conn == nil {
		return
	}
	session, ok := conn.Session().(*info.Info)
	if !ok {
		return
	}
	if session.IsClosed() {
		return
	}
	session.MuxLock()
	if session.IsClosed() {
		session.MuxUnLock()
		return
	}
	//记录下老的uniqId
	previousUniqId := payload.UniqId
	//在高并发下，这个payload.UniqId不一定是manager.Manager.Get时候的，所以一定要重新再从session里面拿出来，保持一致，否则接下来的逻辑会导致连接泄漏
	payload.UniqId = session.GetUniqId()
	//删除主题
	if payload.DelTopic {
		topics := session.PullTopics()
		topic.Topic.Del(topics, payload.UniqId, previousUniqId)
	}
	//删除session
	if payload.DelSession {
		session.SetSession("")
	}
	//删除uniqId
	if payload.DelUniqId {
		//生成一个新的uniqId
		newUniqId := utils.UniqId()
		//处理连接管理器中的关系
		manager.Manager.Del(payload.UniqId)
		manager.Manager.Set(newUniqId, conn)
		//处理主题管理器中的关系
		topics := session.SetUniqIdAndGetTopics(newUniqId)
		//删除旧关系，构建新关系
		topic.Topic.Del(topics, payload.UniqId, previousUniqId)
		topic.Topic.Set(topics, newUniqId)
	}
	session.MuxUnLock()
	//有数据，则转发给客户
	if len(payload.Data) > 0 {
		if err := conn.WriteMessage(websocket.TextMessage, payload.Data); err != nil {
			_ = conn.Close()
		}
	}
}
