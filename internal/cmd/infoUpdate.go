package cmd

import (
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"google.golang.org/protobuf/proto"
	"netsvr/internal/customer/info"
	"netsvr/internal/customer/manager"
	"netsvr/internal/customer/topic"
	"netsvr/internal/heartbeat"
	"netsvr/internal/protocol"
	workerManager "netsvr/internal/worker/manager"
	"time"
)

// InfoUpdate 更新连接的info信息
func InfoUpdate(param []byte, _ *workerManager.ConnProcessor) {
	payload := &protocol.InfoUpdate{}
	if err := proto.Unmarshal(param, payload); err != nil {
		logging.Error("Proto unmarshal protocol.InfoUpdate error: %v", err)
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
	//设置uniqId
	if payload.NewUniqId != "" && payload.NewUniqId != payload.UniqId {
		//如果新的uniqId已经存在，则要移除掉所有关系，因为接下来，这个新的uniqId会被作用在新的连接上
		conflictConn := manager.Manager.Get(payload.NewUniqId)
		if conflictConn != nil {
			//从连接管理器中删除
			manager.Manager.Del(payload.NewUniqId)
			//删除订阅关系、删除uniqId、关闭心跳
			if conflictSession, ok := conflictConn.Session().(*info.Info); ok {
				topics, uniqId, _ := conflictSession.Clear()
				topic.Topic.Del(topics, uniqId)
			}
			//判断是否转发数据
			if len(payload.DataAsNewUniqIdExisted) == 0 {
				//无须转达任何数据，直接关闭连接
				_ = conflictConn.Close()
			} else {
				//写入数据，并在一定倒计时后关闭连接
				_ = conflictConn.WriteMessage(websocket.TextMessage, payload.DataAsNewUniqIdExisted)
				heartbeat.Timer.AfterFunc(time.Second*3, func() {
					_ = conflictConn.Close()
				})
			}
		}
		//处理连接管理器中的关系
		manager.Manager.Del(payload.UniqId)
		manager.Manager.Set(payload.NewUniqId, conn)
		//处理主题管理器中的关系
		topics := session.SetUniqIdAndGetTopics(payload.NewUniqId)
		//删除旧关系，构建新关系
		topic.Topic.Del(topics, payload.UniqId)
		topic.Topic.Set(topics, payload.NewUniqId)
		//重置目标uniqId，因为接下来的设置主题，发送消息的逻辑可能会用到
		payload.UniqId = payload.NewUniqId
	}
	//设置session
	if payload.NewSession != "" {
		session.SetSession(payload.NewSession)
	}
	//设置主题
	if len(payload.NewTopics) > 0 {
		topics := session.PullTopics()
		topic.Topic.Del(topics, payload.UniqId)
		session.Subscribe(payload.NewTopics)
		topic.Topic.Set(payload.NewTopics, payload.UniqId)
	}
	//有数据，则转发给客户
	if len(payload.Data) > 0 {
		Catapult.Put(payload)
	}
}
