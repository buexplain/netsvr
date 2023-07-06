# netsvr

## 简介

`netsvr`是一个网关程序，它在如下架构中处于网关层：

![网络架构](https://github.com/buexplain/netsvr/blob/main/web/netStructure.png "网络架构")

### `netsvr`主要负责：

1. 承载客户的websocket连接，并支持存储连接的：唯一id、主题标签、session信息
2. 承载业务进程的tcp连接
3. 接收客户连接发来的数据，并按一定的路由策略转发到对应的业务进程
4. 接收业务进程发来的数据，并按数据中包含的cmd指令：或转发给客户连接，或自己处理并返回给业务进程

### `netsvr`具体能干嘛：
* 单播、批量单播、组播、广播
* 订阅、发布、取消订阅

更多功能请阅读[指令的proto描述](https://github.com/buexplain/netsvr-protocol)

## 实现的指令

本网关程序实现了下面这套指令，所有业务进程与网关交互都必须遵循这套指令。\
业务进程可以用php、go或其它语言实现，如果是php、go，则已经生成好对应的代码，直接引入即可。

- 指令的proto描述：[buexplain/netsvr-protocol](https://github.com/buexplain/netsvr-protocol)
- 基于指令proto生成的php代码：`composer require buexplain/netsvr-protocol-php`
- 基于指令proto生成的go代码：`go get -u github.com/buexplain/netsvr-protocol-go`

如果采用php实现业务进程的代码，则我已经基于[hyperf](https://github.com/hyperf/hyperf)封装好一个更完善的composer包：[buexplain/netsvr-business](https://github.com/buexplain/netsvr-business)，该包可以便捷的与部署在多台机器的本网关程序交互。

## 业务进程与本网关之间的TCP数据包边界处理

采用`固定包头 + 包体协议`，`包头`是一个`uint32`，表示接下来的`包体`有多长，注意`包头`表示的`长度`，不包含`包头`
自己的`4字节`。\
`包头`采用大端序，`包体`是protobuf编码的数据，心跳字符除外。\
如果采用[swoole](https://github.com/swoole/swoole-src)开发业务进程，则分包协议设置如下：

```php
$socket = new \Swoole\Coroutine\Socket(AF_INET, SOCK_STREAM, 0);
//设置分包协议
$socket->setProtocol([
    'open_length_check' => true,
    'package_length_type' => 'N',
    'package_length_offset' => 0,
    'package_body_offset' => 4,
]);
```

`php-fpm`、`Laravel Octane`等非协程模式运行的php程序。\
不方便实现接收网关主动转发客户数据的双向通信模式，但是可以用本网关做一些单向通信的业务，比如推送业务。\
php的socket与本网关程序通信示例如下：

```php
<?php

//我们以心跳为例，演示socket客户端与网关通信，之所以用心跳演示，是因为心跳是不用做protobuf编解码的。
//定义心跳字符常量，这个不是随便定的，而是网关的写死的，必须是下面这两个字符串。
const PING_MESSAGE = '~3yPvmnz~';
const PONG_MESSAGE = '~u38NvZ~';

// 创建套接字
$socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
if ($socket === false) {
    die("无法创建套接字: " . socket_strerror(socket_last_error()));
}
// 连接到网关的worker服务器
$host = "127.0.0.1";
$port = 6061;
if (socket_connect($socket, $host, $port) === false) {
    die("无法连接到网关的worker服务器 $host:{$port}，错误码：" . socket_strerror(socket_last_error()));
}
try {
    // 向网关的worker服务器发送数据
    if (send(PING_MESSAGE, $socket) === false) {
        die("无法向网关的worker服务器发送数据: " . socket_strerror(socket_last_error()));
    }
    echo "已向网关的worker服务器发送心跳数据：" . PING_MESSAGE . "\n";
    // 读取网关的worker服务器发送的响应
    $package = receive($socket);
    if ($package === PONG_MESSAGE) {
        echo "网关的worker服务器响应心跳成功：$package\n";
    } else {
        echo "收到网关的worker服务器的响应：$package\n";
    }
} finally {
    // 关闭套接字
    // 如果是常驻内的程序，但是又无法保持心跳，比如Laravel Octane
    // 则这个socket资源可以缓存一段时间，这个时间比网关的配置Worker.ReadDeadline小几秒即可
    // 避免反复连接到网关，可以给你的程序提速
    is_resource($socket) && socket_close($socket);
}
//----------------------------------------辅助函数----------------------------------------
/**
 * 发送数据
 * @param string $message
 * @param Socket $socket
 * @return false|int
 */
function send(string $message, Socket $socket): false|int
{
    //打包数据
    $data = pack('N', strlen($message)) . $message;
    return socket_write($socket, $data, strlen($data));
}

/**
 * 接收数据
 * @param Socket $socket
 * @return string|false
 */
function receive(Socket $socket): string|false
{
    //先读取包头长度，4个字节
    $packageLength = socket_read($socket, 4);
    if ($packageLength === false) {
        return false;
    }
    $ret = unpack('N', $packageLength);
    if (!is_array($ret) || !isset($ret[1]) || !is_int($ret[1])) {
        return false;
    } else {
        $packageLength = $ret[1];
    }
    //再读取包体数据
    return socket_read($socket, $packageLength);
}
```

## 客户数据转发策略

* 首先是，业务进程启动的时候需要将自己注册到网关中，发起注册的时候需要提供一个`workerId`，`workerId`的区间是：`[1,999]`。\
多个业务进程可以注册同一个`workerId`。
* 然后是，客户端发送数据的时候需要在数据头部写入三个字节（不足，补零）的`workerId`，用于表示该数据要被哪个`workerId`
的业务进程处理。
* 最后是，如果一个`workerId`包含多个业务进程，则采用轮询策略进行客户数据转发。

客户端发送数据的示例：`001{"cmd":11,"data":"{}"}`，其中的`001`表示，数据要被转发到`workerId`为`1`的业务进程。

## 代码结构介绍

- [netsvr.go](https://github.com/buexplain/netsvr/blob/main/cmd/netsvr.go) 是本网关程序的启动入口。
- [business.go](https://github.com/buexplain/netsvr/blob/main/test/business/cmd/business.go) 是为了测试本网关程序的业务进程启动入口。
- [stress.go](https://github.com/buexplain/netsvr/blob/main/test/stress/cmd/stress.go)
  是为了压测本网关程序的压测进程启动入口，它可以大规模的向网关发起websocket连接。
- [build.sh](https://github.com/buexplain/netsvr/blob/main/scripts/build.sh)
  是编译脚本，把项目clone下来后，直接跑它（依赖go环境），会自动编译出Windows、Linux、Mac三端的网关程序、业务进程程序、压测进程程序。

启动顺序是：网关 --> 业务进程 --> 压测进程

## 网关压测

详情请看：[https://learnku.com/articles/77783](https://learnku.com/articles/77783)

## 业务进程的示例项目

1. [php版的业务进程演示程序](https://github.com/buexplain/netsvr-business-demo)
