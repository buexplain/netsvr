# netsvr - 高性能 WebSocket 网关服务器

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://golang.org/)
[![Performance](https://img.shields.io/badge/Performance-100K%2B%20Connections-brightgreen.svg)](#压测数据)

## 基本介绍

netsvr 是一个高性能的 WebSocket 网关服务器，负责管理客户端 WebSocket 连接，并将 WebSocket 事件转发到业务侧进行处理。

### 核心特性

- 🚀 **高性能**：基于 [gnet](https://github.com/panjf2000/gnet) 异步 IO 模型，单实例支持 10 万+ 连接
- 📡 **事件转发**：将websocket事件转发到业务侧，支持 `TCP Worker`、`HTTP Callback`、`Redis Queue`、`AMQP091 Queue`
- 💾 **灵活路由**
  ：支持单播、组播、广播、主题订阅等多种消息分发模式，详细协议见[netsvr-protocol](https://github.com/buexplain/netsvr-protocol)
- 🛡️ **限流保护**：内置连接级和全局消息限流机制
- 📊 **监控完善**：提供详细的性能指标统计和状态查询

### 适用场景

- 即时通讯（IM）系统
- 实时消息推送服务
- 在线游戏服务器
- 实时数据监控系统
- 直播弹幕服务

### 架构设计

```
┌──────────────┐
│   Clients    │  (WebSocket 连接)
└──────┬───────┘
       │
       ▼
┌──────────────────────────────────────────┐
│          netsvr Gateway                  │
│                                          │
│  ┌───────────────────────────────────┐   │
│  │   Customer Module (WebSocket)     │   │
│  │   • OnOpen                        │   │
│  │   • OnMessage                     │   │
│  │   • OnClose                       │   │
│  └───────────────────────────────────┘   │
│                   │                      │
│       Event forwarding (optional)        │
│         ┌─────────┼──────────┐           │
│         ▼         ▼          ▼           │
│  ┌──────────┐ ┌────────┐ ┌───────────┐   │
│  │  Worker  │ │Callback│ │ Queue     │   │
│  │  (TCP)   │ │ (HTTP) │ │ • Redis   │   │
│  │          │ │        │ │ • AMQP091 │   │
│  └────┬─────┘ └───┬────┘ └────┬──────┘   │
└───────┼───────────┼───────────┼──────────┘
        │           │           │
        ▼           ▼           ▼
   ┌────────┐ ┌──────────┐ ┌──────────┐
   │Business│ │ Business │ │ Business │
   │Process │ │   API    │ │ Consumer │
   └────────┘ └──────────┘ └──────────┘
```

详细配置说明请参考 [netsvr.example.toml](configs/netsvr.example.toml)。

## SDK 与示例代码

netsvr 提供`php`、`golang`的 SDK，方便业务系统集成。

### Go SDK

```bash
go get github.com/buexplain/netsvr-business-go/v2
```

[查看完整文档](https://github.com/buexplain/netsvr-business-go)

### PHP SDK（非协程模式 - Laravel/ThinkPHP/Webman）

```bash
composer require buexplain/netsvr-business-serial
```

**Laravel 控制器实现`HTTP Callback`示例**：

```php
<?php

namespace App\Http\Controllers;

use Exception;
use Illuminate\Foundation\Auth\Access\AuthorizesRequests;
use Illuminate\Foundation\Validation\ValidatesRequests;
use Illuminate\Http\Request;
use Illuminate\Routing\Controller as BaseController;
use Illuminate\Support\Facades\Log;
use NetsvrProtocol\ConnClose;
use NetsvrProtocol\ConnOpen;
use NetsvrProtocol\ConnOpenResp;
use NetsvrProtocol\Transfer;
use Symfony\Component\HttpFoundation\Response as SymfonyResponse;

/**
 * websocket打开、发送消息、关闭的回调接口示例
 * 注册路由的时候一定要注册成post请求，并且加入到 VerifyCsrfToken 中间件的except属性中
 * sdk：https://github.com/buexplain/netsvr-business-serial
 * composer require google/protobuf
 * composer require buexplain/netsvr-protocol-php
 */
class Controller extends BaseController
{
    use AuthorizesRequests, ValidatesRequests;

    /**
     * websocket连接打开的回调接口
     * @param Request $request
     * @return SymfonyResponse
     * @throws Exception
     */
    public function onopen(Request $request): SymfonyResponse
    {
        $protobuf = $request->getContent();
        $cp = new ConnOpen();
        $cp->mergeFromString($protobuf);
        Log::info('onopen -->' . $cp->serializeToJsonString());
        $cpResp = new ConnOpenResp();
        $cpResp->setAllow(true);
        $cpResp->setData('欢迎登录，已为您订阅法律相关栏目: ' . $cp->serializeToJsonString());
        $cpResp->setNewSession(json_encode([
            'id' => 1,
            'nickname' => '法外狂徒',
        ]));
        $cpResp->setNewCustomerId("1");
        $cpResp->setNewTopics([
            '法治在线栏目',
            '法考专题栏目'
        ]);
        return response($cpResp->serializeToString(), 200, [
            'Content-Type' => 'application/x-protobuf',
        ]);
    }

    /**
     * websocket连接发消息的回调接口
     * @param Request $request
     * @return SymfonyResponse
     * @throws Exception
     */
    public function onmessage(Request $request): SymfonyResponse
    {
        $protobuf = $request->getContent();
        $tf = new Transfer();
        $tf->mergeFromString($protobuf);
        Log::info('onmessage -->' . $tf->serializeToJsonString());
        return response('', 204);
    }

    /**
     * websocket连接关闭的回调接口
     * @param Request $request
     * @return SymfonyResponse
     * @throws Exception
     */
    public function onclose(Request $request): SymfonyResponse
    {
        $protobuf = $request->getContent();
        $cc = new ConnClose();
        $cc->mergeFromString($protobuf);
        Log::info('onclose -->' . $cc->serializeToJsonString());
        return response('', 204);
    }
}
```

**路由配置**：

```php
Route::post('/onopen', [App\Http\Controllers\Controller::class, 'onopen']);
Route::post('/onmessage', [App\Http\Controllers\Controller::class, 'onmessage']);
Route::post('/onclose', [App\Http\Controllers\Controller::class, 'onclose']);
```

[查看完整文档](https://github.com/buexplain/netsvr-business-serial)

## 压测数据

### 压测环境

- **服务器**：腾讯云 `SA9.LARGE8` `4核` `8GiB` `AMD EPYC Turin-D (-/3.4GHz)`
- **网络**：内网带宽 `2 Gbps`、内网每秒包转发数量 `30 万 PPS`
- **部署**：1台 netsvr + 2台 business + 4台 stress

### 压测场景

- **连接数**：100,000 个 WebSocket 连接
- **用户绑定**：每个连接绑定 userId 和 2KiB session
- **消息频率**：40 个群，每群每秒 3 条 100B（纯文本内容、不含协议/元数据） 消息
- **总吞吐量**：约 30 万条消息/秒

### 压测结果

| 指标      | 数值                  |
|---------|---------------------|
| 实际连接数   | 100,000             |
| 接收消息并发数 | 121 msg/s           |
| 下发消息并发数 | 303,544 msg/s       |
| CPU 使用率 | 379.4%（4核满载）        |
| 内存使用    | 1,852 MB / 7,555 MB |
| 系统负载    | 5.03（1分钟平均）         |

### 容量估算

| 机器配置      | 极限负载 | 峰值负载 | 安全生产 |
|-----------|------|------|------|
| 4核 8GiB   | 10 万 | 8 万  | 7 万  |
| 8核 16GiB  | 20 万 | 16 万 | 14 万 |
| 16核 16GiB | 40 万 | 32 万 | 28 万 |
| 32核 64GiB | 80 万 | 64 万 | 56 万 |

> 💡 **关键发现**：netsvr 对内存需求不高，瓶颈主要在 **CPU 算力**和**网卡性能**。系统态 CPU 使用率高（~53%）和软中断高（~
> 26%）表明网络处理是主要开销。

## 许可证

本项目采用 Apache License 2.0 许可证 - 详见 [LICENSE](LICENSE) 文件