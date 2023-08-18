package connPool

import (
	"net"
	"time"
)

type ConnPool struct {
	pool chan net.Conn
	//心跳函数
	heartbeat func(conn net.Conn) bool
	//net.Conn构造函数
	factory func() net.Conn
	//用于连接归还失败时候异步构建新连接
	compensatorCh chan struct{}
}

func NewConnPool(size int, factory func() net.Conn, heartbeat func(conn net.Conn) bool, heartbeatDuration time.Duration) *ConnPool {
	cp := &ConnPool{
		pool:          make(chan net.Conn, size),
		heartbeat:     heartbeat,
		factory:       factory,
		compensatorCh: make(chan struct{}, size),
	}
	for i := 0; i < size; i++ {
		c := factory()
		if c != nil {
			cp.pool <- c
		}
	}
	go cp.loopHeartbeat(heartbeatDuration)
	go cp.compensator()
	return cp
}

// 异步构建一个新连接，给到连接池
func (r *ConnPool) compensator() {
	f := func() {
		for {
			conn := r.factory()
			if conn == nil {
				//构建连接失败，间隔1秒再次构建
				time.Sleep(time.Second * 1)
				continue
			}
			select {
			case r.pool <- conn:
				//构建成功，归还到连接池放着
				return
			default:
				//莫名其妙的逻辑导致连接池满了，这里直接丢弃即可
				_ = conn.Close()
			}
		}
	}
	//监听补偿连接池的连接的信号
	for {
		<-r.compensatorCh
		//收到一个信号，开始补偿连接到连接池
		f()
	}
}

func (r *ConnPool) loopHeartbeat(heartbeatDuration time.Duration) {
	t := time.NewTicker(heartbeatDuration)
	defer t.Stop()
	heartbeatOk := make(map[net.Conn]struct{}, cap(r.pool))
	for {
		<-t.C
		//遍历pool中的所有连接，并进行心跳处理
		for i := 0; i < cap(r.pool); i++ {
			select {
			case conn := <-r.pool:
				//拿到连接，判断是否已经发送过心跳
				if _, ok := heartbeatOk[conn]; ok {
					break
				}
				//没有心跳过，进行一次心跳操作
				if !r.heartbeat(conn) {
					//操作失败
					r.Put(nil)
				} else {
					//操作成功，记录起来
					heartbeatOk[conn] = struct{}{}
					r.pool <- conn
				}
			default:
				//没拿到连接，说明连接池内的连接被频繁使用，跳过，进行下一次获取连接
				break
			}
		}
		//清空心跳记录
		for conn := range heartbeatOk {
			delete(heartbeatOk, conn)
		}
	}
}

func (r *ConnPool) Get() net.Conn {
	return <-r.pool
}

func (r *ConnPool) Put(conn net.Conn) {
	if conn == nil {
		//连接归还失败，为了避免连接池在此情况下被掏空，进而导致Get连接操作被阻塞，这里发送一个异步构建新连接的信号
		select {
		case r.compensatorCh <- struct{}{}:
			return
		default:
			return
		}
	} else {
		r.pool <- conn
	}
}
