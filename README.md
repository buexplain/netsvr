# netsvr

## 简介

`netsvr`是一个网关程序，它在如下架构中处于网关层：

![网络架构](https://github.com/buexplain/netsvr/blob/main/web/netStructure.png "网络架构")

`netsvr`主要负责：

1. 承载客户的websocket连接，并支持存储连接的：唯一id、主题标签、session信息
2. 承载业务进程的tcp连接
3. 接收客户连接发来的数据，并按一定的路由策略转发到对应的业务进程
4. 接收业务进程发来的数据，并按数据中包含的指令，或转发给客户连接，或自己处理并返回给业务进程

## 实现的指令

进入这里查看 https://github.com/buexplain/netsvr-protocol

## 客户数据转发策略

首先是业务进程启动的时候需要将自己注册到网关中，发起注册的时候需要提供一个`workerId`，`workerId`的区间是：`[1,999]`。

多个业务进程可以注册同一个`workerId`。

然后是客户端发送数据的时候需要在数据头部写入三个字节（不足，补零）的`workerId`，用于表示该数据要被哪个`workerId`的业务进程处理。

如果一个`workerId`包含多个业务进程，则采用轮询策略进行转发。

客户端发送数据的示例：`001{"cmd":11,"data":"{}"}`，其中的`001`表示，数据要被转发到`workerId`为`1`的业务进程。

## 代码结构介绍

- [netsvr.go](https://github.com/buexplain/netsvr/blob/main/cmd/netsvr.go) 是本网关程序的启动入口
- [business.go](https://github.com/buexplain/netsvr/blob/main/test/business/cmd/business.go) 是为了测试本网关程序的业务进程启动入口
- [stress.go](https://github.com/buexplain/netsvr/blob/main/test/stress/cmd/stress.go) 是为了压测本网关程序的压测进程启动入口