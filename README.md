# netsvr

## 简介

<img src="assets/netsvr.jpg" alt="netsvr">

### `netsvr`主要负责：

1. 承载客户端的websocket连接，并支持存储连接的：业务系统内唯一id、订阅的主题、连接登录后的session信息
2. 承载业务进程的tcp连接
3. 接收客户端连接发来的数据，并将数据转发给业务进程
4. 接收业务进程发来的数据，并按数据中包含的cmd指令：或转发给客户端连接，或自己处理并返回给业务进程

### `netsvr`具体能干嘛：

`netsvr`能起到桥梁作用，让你的业务进程和客户端之间进行通信，从而实现：

* 你的业务进程：以单播、批量单播、组播、发布、批量发布、广播等形式，将数据主动下发给客户端连接
* 你的客户端：订阅主题的数据、取消订阅主题、发送数据给服务端、接收服务端下发的数据

更多功能请阅读：[业务进程与netsvr进程的交互协议](https://github.com/buexplain/netsvr-protocol)

## 如何使用`netsvr`

### 具体的使用方式，请参考具体的sdk：

- [hyperf框架](https://github.com/hyperf/hyperf)
  下的sdk：[composer require buexplain/netsvr-business-coroutine](https://github.com/buexplain/netsvr-business-coroutine)
- `php-fpm`、`ThinkPHP`、`Laravel`、`Laravel Octane`、`Webman`
  等非协程模式运行的sdk：[composer require buexplain/netsvr-business-serial](https://github.com/buexplain/netsvr-business-serial)
- go 语言的sdk：[go get github.com/buexplain/netsvr-business-go](https://github.com/buexplain/netsvr-business-go)

### 如果只做从服务端下发到客户端，客户端不通过websocket发送数据到服务端，则需要配置`CallbackApi`相关接口

下面是php语言的laravel框架下的websocket打开和关闭的回调接口示例：

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
use Symfony\Component\HttpFoundation\Response as SymfonyResponse;

/**
 * websocket打开和关闭的回调接口示例
 * 注册路由的时候一定要注册成post请求，并且加入到VerifyCsrfToken中间件的except属性中
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
        Log::debug('onopen -->' . $cp->serializeToJsonString());
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
        Log::debug('onclose -->' . $cc->serializeToJsonString());
        return response('', 204);
    }
}
```

## 本项目代码结构介绍

- [netsvr.go](https://github.com/buexplain/netsvr/blob/main/cmd/netsvr.go) 是netsvr程序的启动入口。
- [business.go](https://github.com/buexplain/netsvr/blob/main/test/business/cmd/business.go) 是为了测试netsvr程序的业务进程启动入口。
- [stress.go](https://github.com/buexplain/netsvr/blob/main/test/stress/cmd/stress.go)
  是为了压测netsvr程序的压测进程启动入口，它可以大规模的向网关发起websocket连接。
- [build.sh](https://github.com/buexplain/netsvr/blob/main/scripts/build.sh)
  是编译脚本，把项目clone下来后，直接跑它（依赖go环境），会自动编译出Windows、Linux、Mac三端的netsvr程序、业务进程程序、压测进程程序。

启动顺序是：netsvr --> 业务进程 --> 压测进程
> 注意：生产环境只需要netsvr程序