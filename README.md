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

## 与`netsvr`配合，进行业务开发时的可用包

- 基于指令的proto生成的php代码：`composer require buexplain/netsvr-protocol-php`，该包比较原始，不好使用。
- 基于指令的proto生成的go代码：`go get -u github.com/buexplain/netsvr-protocol-go`，该包比较原始，不好使用。
- 基于`composer require buexplain/netsvr-protocol-php` 开发的，[hyperf框架](https://github.com/hyperf/hyperf)下的，高级sdk：[composer require buexplain/netsvr-business-coroutine](https://github.com/buexplain/netsvr-business-coroutine)
- 基于`composer require buexplain/netsvr-protocol-php` 开发的，`php-fpm`、`ThinkPHP`、`Laravel`、`Laravel Octane`等非协程模式运行的，串行执行php代码的程序下的，高级sdk：[composer require buexplain/netsvr-business-serial](https://github.com/buexplain/netsvr-business-serial)

## 客户数据转发到业务进程的策略

* 首先是，业务进程启动的时候需要将自己注册到网关中，发起注册的时候需要提供一个`workerId`，`workerId`的区间是：`[1,999]`。\
  多个业务进程可以注册同一个`workerId`。
* 然后是，客户端发送数据的时候需要在数据头部写入三个字节（不足，补零）的`workerId`，用于表示该数据要被哪个`workerId`
  的业务进程处理。
* 最后是，如果一个`workerId`包含多个业务进程，则采用轮询策略进行客户数据转发到业务进程。

客户端发送数据的示例：`001{"cmd":11,"data":"{}"}`，其中的`001`表示，数据要被转发到`workerId`为`1`的业务进程。

## 业务进程与本网关之间的TCP数据包边界处理

采用`固定包头 + 包体协议`，`包头`是一个`uint32`，表示接下来的`包体`有多长，注意`包头`表示的`长度`，不包含`包头`
自己的`4字节`。`包头`采用大端序，`包体`是protobuf编码的数据，但是，心跳字符不会被protobuf编码。

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