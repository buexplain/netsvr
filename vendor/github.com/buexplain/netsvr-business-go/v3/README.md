# netsvr-business-go

这是一个可以快速开发 websocket 全双工通信业务的包，它基于 [https://github.com/buexplain/netsvr](https://github.com/buexplain/netsvr) 进行工作。

模块路径：`github.com/buexplain/netsvr-business-go/v3`

## 使用示例

请参考：[netsvr项目的business模块](https://github.com/buexplain/netsvr/blob/main/test/business/cmd/business.go)

## 快速接入

- **发送与查询**：用 `taskSocket` 连接池管理器创建 `NetBus`，调用其方法向网关发指令。
- **接收事件**：用 `mainSocket.New(...)` 建立与网关的长连接，传入实现了 `contract.EventInterface` 的对象。

## API

### 入口

- `NewNetBus(taskSocketPoolManger *taskSocket.Manger) *NetBus`
- `(*NetBus) Close()`

### 发送（单向，网关不响应）

| 分组 | 方法 |
| --- | --- |
| 全量广播 | `BroadcastBulk`、`Broadcast` |
| 按 uniqId | `SingleCastBulk`、`SendToUniqId`、`SendToUniqIds` |
| 按 customerId | `SingleCastBulkByCustomerId`、`SendToCustomerId`、`SendToCustomerIds` |
| 按 topic | `TopicPublishBulk`、`PublishToTopic`、`PublishToTopics`、`TopicDelete` |
| 连接信息与订阅 | `ConnInfoUpdate`、`ConnInfoDelete`、`TopicSubscribe`、`TopicUnsubscribe` |
| 强制下线 | `ForceOffline`、`ForceOfflineByCustomerId`、`ForceOfflineGuest` |

### 查询（返回 `*ret.XxxRet`，`Data` 的 key 是网关地址）

`CheckOnline`、`UniqIdList`、`UniqIdCount`、`CustomerIdList`、`CustomerIdCount`、`TopicList`、`TopicCount`、`TopicUniqIdList`、`TopicUniqIdCount`、`TopicCustomerIdList`、`TopicCustomerIdCount`、`TopicCustomerIdToUniqIdsList`、`ConnInfo`、`ConnInfoByCustomerId`、`Metrics`、`Limit`

返回值辅助方法：`CheckOnlineRet.Has`、`UniqIdCountRet.Count`、`TopicCountRet.Count`、`ConnInfoRet.ToMap`

### 事件接收

`mainSocket.New`、`mainSocket.NewManager`；`(*MainSocket)` 的 `Connect` / `Register` / `Unregister` / `LoopHeartbeat` / `LoopReceive` / `Close`；`contract.EventInterface` 的 `OnOpen` / `OnMessage` / `OnClose`

## 说明

- **完整的参数、返回值与逐方法说明见源码注释**（IDE 悬停或 `go doc`）。
- **指令语义以协议注释为准**（[netsvr-protocol](https://github.com/buexplain/netsvr-protocol)），包括「`topics` 为空即统计全部主题」「单网关 / 多网关的重复统计」等。
- **入参与返回值直接使用协议生成类型**（`netsvr-protocol-go`），本包不额外定义 DTO。
- **多机部署的路由**：按 uniqId 的方法会拆分到目标网关；按 customerId / topic 的方法会发给所有网关。
