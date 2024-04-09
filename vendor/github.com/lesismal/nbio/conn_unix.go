// Copyright 2020 lesismal. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build linux || darwin || netbsd || freebsd || openbsd || dragonfly
// +build linux darwin netbsd freebsd openbsd dragonfly

package nbio

import (
	"encoding/binary"
	"errors"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/lesismal/nbio/mempool"
)

var (
	emptyToWrite = toWrite{}

	poolToWrite = sync.Pool{
		New: func() interface{} {
			return &toWrite{}
		},
	}
)

func newToWriteBuf(buf []byte) *toWrite {
	t := poolToWrite.New().(*toWrite)
	b := mempool.Malloc(len(buf))
	copy(b, buf)
	t.buf = b
	return t
}

func newToWriteFile(fd int, offset, remain int64) *toWrite {
	t := poolToWrite.New().(*toWrite)
	t.fd = fd
	t.offset = offset
	t.remain = remain
	return t
}

func releaseToWrite(t *toWrite) {
	if t.buf != nil {
		mempool.Free(t.buf)
	}
	if t.fd > 0 {
		syscall.Close(t.fd)
	}
	*t = emptyToWrite
	poolToWrite.Put(t)
}

type toWrite struct {
	fd     int
	buf    []byte
	offset int64
	remain int64
}

// Conn implements net.Conn.
type Conn struct {
	mux sync.Mutex

	p *poller

	fd int

	connUDP *udpConn

	rTimer *time.Timer
	wTimer *time.Timer

	left      int
	writeList []*toWrite

	typ      ConnType
	closed   bool
	isWAdded bool
	closeErr error

	lAddr net.Addr
	rAddr net.Addr

	ReadBuffer []byte

	session interface{}

	execList []func()

	readEvents int32

	dataHandler func(c *Conn, data []byte)
}

// Hash returns a hash code.
func (c *Conn) Hash() int {
	return c.fd
}

// AsyncRead .
func (c *Conn) AsyncRead() {
	g := c.p.g

	if g.isOneshot {
		g.IOExecute(func(buffer []byte) {
			for i := 0; i < g.MaxConnReadTimesPerEventLoop; i++ {
				rc, n, err := c.ReadAndGetConn(buffer)
				if n > 0 {
					g.onData(rc, buffer[:n])
				}
				if errors.Is(err, syscall.EINTR) {
					continue
				}
				if errors.Is(err, syscall.EAGAIN) {
					break
				}
				if err != nil {
					c.closeWithError(err)
					return
				}
				if n < len(buffer) {
					break
				}
			}
			c.ResetPollerEvent()
		})
	}

	cnt := atomic.AddInt32(&c.readEvents, 1)
	if cnt > 2 {
		atomic.AddInt32(&c.readEvents, -1)
		return
	}
	if cnt > 1 {
		return
	}

	g.IOExecute(func(buffer []byte) {
		for {
			for i := 0; i < g.MaxConnReadTimesPerEventLoop; i++ {
				rc, n, err := c.ReadAndGetConn(buffer)
				if n > 0 {
					g.onData(rc, buffer[:n])
				}
				if errors.Is(err, syscall.EINTR) {
					continue
				}
				if errors.Is(err, syscall.EAGAIN) {
					break
				}
				if err != nil {
					c.closeWithError(err)
					return
				}
				if n < len(buffer) {
					break
				}
			}
			if atomic.AddInt32(&c.readEvents, -1) == 0 {
				return
			}
		}
	})
}

// Read implements Read.
func (c *Conn) Read(b []byte) (int, error) {
	// use lock to prevent multiple conn data confusion when fd is reused on unix.
	c.mux.Lock()
	if c.closed {
		c.mux.Unlock()
		return 0, net.ErrClosed
	}

	_, n, err := c.doRead(b)
	c.mux.Unlock()
	if err == nil {
		c.p.g.afterRead(c)
	}

	return n, err
}

// ReadUDP .
func (c *Conn) ReadUDP(b []byte) (*Conn, int, error) {
	return c.ReadAndGetConn(b)
}

// ReadAndGetConn .
func (c *Conn) ReadAndGetConn(b []byte) (*Conn, int, error) {
	// use lock to prevent multiple conn data confusion when fd is reused on unix.
	c.mux.Lock()
	if c.closed {
		c.mux.Unlock()
		return c, 0, net.ErrClosed
	}

	dstConn, n, err := c.doRead(b)
	c.mux.Unlock()
	if err == nil {
		c.p.g.afterRead(c)
	}

	return dstConn, n, err
}

func (c *Conn) doRead(b []byte) (*Conn, int, error) {
	switch c.typ {
	case ConnTypeTCP, ConnTypeUnix:
		return c.readStream(b)
	case ConnTypeUDPServer, ConnTypeUDPClientFromDial:
		return c.readUDP(b)
	case ConnTypeUDPClientFromRead:
	default:
	}
	return c, 0, errors.New("invalid udp conn for reading")
}

func (c *Conn) readStream(b []byte) (*Conn, int, error) {
	nread, err := syscall.Read(c.fd, b)
	return c, nread, err
}

func (c *Conn) readUDP(b []byte) (*Conn, int, error) {
	nread, rAddr, err := syscall.Recvfrom(c.fd, b, 0)
	if c.closeErr == nil {
		c.closeErr = err
	}
	if err != nil {
		return c, 0, err
	}

	var g = c.p.g
	var dstConn = c
	if c.typ == ConnTypeUDPServer {
		uc, ok := c.connUDP.getConn(c.p, c.fd, rAddr)
		if g.UDPReadTimeout > 0 {
			uc.SetReadDeadline(time.Now().Add(g.UDPReadTimeout))
		}
		if !ok {
			g.onOpen(uc)
		}
		dstConn = uc
	}

	return dstConn, nread, err
}

// Write implements Write.
func (c *Conn) Write(b []byte) (int, error) {
	c.p.g.beforeWrite(c)

	c.mux.Lock()
	if c.closed {
		c.mux.Unlock()
		return -1, net.ErrClosed
	}

	n, err := c.write(b)
	if err != nil && !errors.Is(err, syscall.EINTR) && !errors.Is(err, syscall.EAGAIN) {
		c.closed = true
		c.mux.Unlock()
		c.closeWithErrorWithoutLock(err)
		return n, err
	}

	if len(c.writeList) == 0 {
		if c.wTimer != nil {
			c.wTimer.Stop()
			c.wTimer = nil
		}
	} else {
		c.modWrite()
	}

	c.mux.Unlock()
	return n, err
}

// Writev implements Writev.
func (c *Conn) Writev(in [][]byte) (int, error) {
	c.p.g.beforeWrite(c)

	c.mux.Lock()
	if c.closed {
		c.mux.Unlock()

		return 0, net.ErrClosed
	}

	var n int
	var err error
	switch len(in) {
	case 1:
		n, err = c.write(in[0])
	default:
		n, err = c.writev(in)
	}
	if err != nil && !errors.Is(err, syscall.EINTR) && !errors.Is(err, syscall.EAGAIN) {
		c.closed = true
		c.mux.Unlock()
		c.closeWithErrorWithoutLock(err)
		return n, err
	}
	if len(c.writeList) == 0 {
		if c.wTimer != nil {
			c.wTimer.Stop()
			c.wTimer = nil
		}
	} else {
		c.modWrite()
	}

	c.mux.Unlock()
	return n, err
}

func (c *Conn) writeStream(b []byte) (int, error) {
	return syscall.Write(c.fd, b)
}

func (c *Conn) writeUDPClientFromDial(b []byte) (int, error) {
	return syscall.Write(c.fd, b)
}

func (c *Conn) writeUDPClientFromRead(b []byte) (int, error) {
	err := syscall.Sendto(c.fd, b, 0, c.connUDP.rAddr)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

// Close implements Close.
func (c *Conn) Close() error {
	return c.closeWithError(nil)
}

// CloseWithError .
func (c *Conn) CloseWithError(err error) error {
	return c.closeWithError(err)
}

// LocalAddr implements LocalAddr.
func (c *Conn) LocalAddr() net.Addr {
	return c.lAddr
}

// RemoteAddr implements RemoteAddr.
func (c *Conn) RemoteAddr() net.Addr {
	return c.rAddr
}

// SetDeadline implements SetDeadline.
func (c *Conn) SetDeadline(t time.Time) error {
	c.mux.Lock()
	if !c.closed {
		if !t.IsZero() {
			g := c.p.g
			if c.rTimer == nil {
				c.rTimer = g.AfterFunc(time.Until(t), func() { c.closeWithError(errReadTimeout) })
			} else {
				c.rTimer.Reset(time.Until(t))
			}
			if c.wTimer == nil {
				c.wTimer = g.AfterFunc(time.Until(t), func() { c.closeWithError(errWriteTimeout) })
			} else {
				c.wTimer.Reset(time.Until(t))
			}
		} else {
			if c.rTimer != nil {
				c.rTimer.Stop()
				c.rTimer = nil
			}
			if c.wTimer != nil {
				c.wTimer.Stop()
				c.wTimer = nil
			}
		}
	}
	c.mux.Unlock()
	return nil
}

func (c *Conn) setDeadline(timer **time.Timer, returnErr error, t time.Time) error {
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.closed {
		return nil
	}
	if !t.IsZero() {
		if *timer == nil {
			*timer = c.p.g.AfterFunc(time.Until(t), func() { c.closeWithError(returnErr) })
		} else {
			(*timer).Reset(time.Until(t))
		}
	} else if *timer != nil {
		(*timer).Stop()
		(*timer) = nil
	}
	return nil
}

// SetReadDeadline implements SetReadDeadline.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.setDeadline(&c.rTimer, errReadTimeout, t)
}

// SetWriteDeadline implements SetWriteDeadline.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.setDeadline(&c.wTimer, errWriteTimeout, t)
}

// SetNoDelay implements SetNoDelay.
func (c *Conn) SetNoDelay(nodelay bool) error {
	if nodelay {
		return syscall.SetsockoptInt(c.fd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1)
	}
	return syscall.SetsockoptInt(c.fd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 0)
}

// SetReadBuffer implements SetReadBuffer.
func (c *Conn) SetReadBuffer(bytes int) error {
	return syscall.SetsockoptInt(c.fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, bytes)
}

// SetWriteBuffer implements SetWriteBuffer.
func (c *Conn) SetWriteBuffer(bytes int) error {
	return syscall.SetsockoptInt(c.fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF, bytes)
}

// SetKeepAlive implements SetKeepAlive.
func (c *Conn) SetKeepAlive(keepalive bool) error {
	if keepalive {
		return syscall.SetsockoptInt(c.fd, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1)
	}
	return syscall.SetsockoptInt(c.fd, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 0)
}

// SetKeepAlivePeriod implements SetKeepAlivePeriod.
func (c *Conn) SetKeepAlivePeriod(d time.Duration) error {
	if runtime.GOOS == "linux" {
		d += (time.Second - time.Nanosecond)
		secs := int(d.Seconds())
		if err := syscall.SetsockoptInt(c.fd, IPPROTO_TCP, TCP_KEEPINTVL, secs); err != nil {
			return err
		}
		return syscall.SetsockoptInt(c.fd, IPPROTO_TCP, TCP_KEEPIDLE, secs)
	}
	return errors.New("not supported")
}

// SetLinger implements SetLinger.
func (c *Conn) SetLinger(onoff int32, linger int32) error {
	return syscall.SetsockoptLinger(c.fd, syscall.SOL_SOCKET, syscall.SO_LINGER, &syscall.Linger{
		Onoff:  onoff,  // 1
		Linger: linger, // 0
	})
}

// Session returns user session.
func (c *Conn) Session() interface{} {
	return c.session
}

// SetSession sets user session.
func (c *Conn) SetSession(session interface{}) {
	c.session = session
}

func (c *Conn) modWrite() {
	if !c.closed && !c.isWAdded {
		c.isWAdded = true
		c.p.modWrite(c.fd)
	}
}

func (c *Conn) resetRead() {
	if !c.closed && c.isWAdded {
		c.isWAdded = false
		p := c.p
		p.resetRead(c.fd)
	}
}

func (c *Conn) write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}

	if c.overflow(len(b)) {
		return -1, errOverflow
	}

	if len(c.writeList) == 0 {
		n, err := c.doWrite(b)
		if err != nil && !errors.Is(err, syscall.EINTR) && !errors.Is(err, syscall.EAGAIN) {
			return n, err
		}
		if n < 0 {
			n = 0
		}
		left := len(b) - n
		if left > 0 && c.typ == ConnTypeTCP {
			t := newToWriteBuf(b[n:])
			c.appendWrite(t)
		}
		return len(b), nil
	}

	t := newToWriteBuf(b)
	c.appendWrite(t)

	return len(b), nil
}

func (c *Conn) writev(in [][]byte) (int, error) {
	size := 0
	for _, v := range in {
		size += len(v)
	}
	if c.overflow(size) {
		return -1, errOverflow
	}
	if len(c.writeList) > 0 {
		for _, v := range in {
			t := newToWriteBuf(v)
			c.appendWrite(t)
		}
		return size, nil
	}

	nwrite, err := writev(c.fd, in)
	if nwrite > 0 {
		n := nwrite
		onWrittenSize := c.p.g.onWrittenSize
		if n < size {
			for i := 0; i < len(in) && n > 0; i++ {
				b := in[i]
				if n == 0 {
					t := newToWriteBuf(b)
					c.appendWrite(t)
				} else {
					if n < len(b) {
						if onWrittenSize != nil {
							onWrittenSize(c, b[:n], n)
						}
						t := newToWriteBuf(b[n:])
						c.appendWrite(t)
						n = 0
					} else {
						if onWrittenSize != nil {
							onWrittenSize(c, b, len(b))
						}
						n -= len(b)
					}
				}
			}
			c.left += (size - n)
		}
	} else {
		nwrite = 0
	}

	// if err != nil && !errors.Is(err, syscall.EINTR) && !errors.Is(err, syscall.EAGAIN) {
	// 	return nwrite, err
	// }

	return nwrite, err
}

func (c *Conn) appendWrite(t *toWrite) {
	c.writeList = append(c.writeList, t)
	if t.buf != nil {
		c.left += len(t.buf)
	}
}

func (c *Conn) flush() error {
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.closed {
		return net.ErrClosed
	}

	if len(c.writeList) == 0 {
		return nil
	}

	onWrittenSize := c.p.g.onWrittenSize

	iovc := make([][]byte, 4)[0:0]
	writeBuffers := func() error {
		var (
			n    int
			err  error
			head *toWrite
		)

		if len(c.writeList) == 1 {
			head = c.writeList[0]
			buf := head.buf[head.offset:]
			for len(buf) > 0 && err == nil {
				n, err = syscall.Write(c.fd, buf)
				if n > 0 {
					if c.p.g.onWrittenSize != nil && n > 0 {
						c.p.g.onWrittenSize(c, buf[:n], n)
					}
					c.left -= n
					head.offset += int64(n)
					buf = buf[n:]
					if len(buf) == 0 {
						releaseToWrite(head)
						c.writeList = nil
					}
				}
			}
			return err
		}

		iovc = iovc[0:0]
		for i := 0; i < len(c.writeList); i++ {
			head = c.writeList[i]
			if head.buf != nil {
				iovc = append(iovc, head.buf[head.offset:])
			} else {
				break
			}
		}

		for len(iovc) > 0 && err == nil {
			n, err = writev(c.fd, iovc)
			if n > 0 {
				c.left -= n
				for n > 0 {
					head = c.writeList[0]
					headLeft := len(head.buf) - int(head.offset)
					if n < headLeft {
						if onWrittenSize != nil {
							onWrittenSize(c, head.buf[head.offset:head.offset+int64(n)], n)
						}
						head.offset += int64(n)
						iovc[0] = iovc[0][n:]
						break
					} else {
						if onWrittenSize != nil {
							onWrittenSize(c, head.buf[head.offset:], headLeft)
						}
						releaseToWrite(head)
						c.writeList = c.writeList[1:]
						if len(c.writeList) == 0 {
							c.writeList = nil
						}
						iovc = iovc[1:]
						n -= headLeft
					}
				}
			}
		}
		return err
	}

	writeFile := func() error {
		v := c.writeList[0]
		for v.remain > 0 {
			var offset = v.offset
			n, err := syscall.Sendfile(c.fd, v.fd, &offset, int(v.remain))
			if n > 0 {
				if onWrittenSize != nil {
					onWrittenSize(c, nil, n)
				}
				v.remain -= int64(n)
				v.offset += int64(n)
				if v.remain <= 0 {
					releaseToWrite(c.writeList[0])
					c.writeList = c.writeList[1:]
				}
			}
			if err != nil {
				return err
			}
		}
		return nil
	}

	for len(c.writeList) > 0 {
		var err error
		if c.writeList[0].fd == 0 {
			err = writeBuffers()
		} else {
			err = writeFile()
		}

		if errors.Is(err, syscall.EINTR) {
			continue
		}
		if errors.Is(err, syscall.EAGAIN) {
			// c.modWrite()
			return nil
		}
		if err != nil {
			c.closed = true
			c.closeWithErrorWithoutLock(err)
			return err
		}
	}

	c.resetRead()

	return nil
}

func (c *Conn) doWrite(b []byte) (int, error) {
	var n int
	var err error
	switch c.typ {
	case ConnTypeTCP, ConnTypeUnix:
		n, err = c.writeStream(b)
	case ConnTypeUDPServer:
	case ConnTypeUDPClientFromDial:
		n, err = c.writeUDPClientFromDial(b)
	case ConnTypeUDPClientFromRead:
		n, err = c.writeUDPClientFromRead(b)
	default:
	}
	if c.p.g.onWrittenSize != nil && n > 0 {
		c.p.g.onWrittenSize(c, b[:n], n)
	}
	return n, err
}

func (c *Conn) overflow(n int) bool {
	g := c.p.g
	return g.MaxWriteBufferSize > 0 && (c.left+n > g.MaxWriteBufferSize)
}

func (c *Conn) closeWithError(err error) error {
	c.mux.Lock()
	if !c.closed {
		c.closed = true

		if c.wTimer != nil {
			c.wTimer.Stop()
			c.wTimer = nil
		}
		if c.rTimer != nil {
			c.rTimer.Stop()
			c.rTimer = nil
		}

		c.mux.Unlock()
		return c.closeWithErrorWithoutLock(err)
	}
	c.mux.Unlock()
	return nil
}

func (c *Conn) closeWithErrorWithoutLock(err error) error {
	c.closeErr = err

	if c.writeList != nil {
		for _, t := range c.writeList {
			releaseToWrite(t)
		}
		c.writeList = nil
	}

	if c.p != nil {
		c.p.deleteConn(c)
	}

	switch c.typ {
	case ConnTypeTCP, ConnTypeUnix:
		err = syscall.Close(c.fd)
	case ConnTypeUDPServer, ConnTypeUDPClientFromDial, ConnTypeUDPClientFromRead:
		err = c.connUDP.Close()
	default:
	}

	return err
}

// NBConn converts net.Conn to *Conn.
func NBConn(conn net.Conn) (*Conn, error) {
	if conn == nil {
		return nil, errors.New("invalid conn: nil")
	}
	c, ok := conn.(*Conn)
	if !ok {
		var err error
		c, err = dupStdConn(conn)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

type udpConn struct {
	parent *Conn

	rAddr    syscall.Sockaddr
	rAddrKey udpAddrKey

	mux   sync.RWMutex
	conns map[udpAddrKey]*Conn
}

func (u *udpConn) Close() error {
	parent := u.parent
	if parent.connUDP != u {
		parent.mux.Lock()
		delete(parent.connUDP.conns, u.rAddrKey)
		parent.mux.Unlock()
	} else {
		syscall.Close(u.parent.fd)
		for _, c := range u.conns {
			c.Close()
		}
		u.conns = nil
	}
	return nil
}

func (u *udpConn) getConn(p *poller, fd int, rsa syscall.Sockaddr) (*Conn, bool) {
	rAddrKey := getUDPNetAddrKey(rsa)
	u.mux.RLock()
	c, ok := u.conns[rAddrKey]
	u.mux.RUnlock()

	if !ok {
		c = &Conn{
			p:     p,
			fd:    fd,
			lAddr: u.parent.lAddr,
			rAddr: getUDPNetAddr(rsa),
			typ:   ConnTypeUDPClientFromRead,
			connUDP: &udpConn{
				rAddr:    rsa,
				rAddrKey: rAddrKey,
				parent:   u.parent,
			},
		}
		u.mux.Lock()
		u.conns[rAddrKey] = c
		u.mux.Unlock()
	}

	return c, ok
}

type udpAddrKey [22]byte

func getUDPNetAddrKey(sa syscall.Sockaddr) udpAddrKey {
	var ret udpAddrKey
	if sa == nil {
		return ret
	}

	switch vt := sa.(type) {
	case *syscall.SockaddrInet4:
		copy(ret[:], vt.Addr[:])
		binary.LittleEndian.PutUint16(ret[16:], uint16(vt.Port))
	case *syscall.SockaddrInet6:
		copy(ret[:], vt.Addr[:])
		binary.LittleEndian.PutUint16(ret[16:], uint16(vt.Port))
		binary.LittleEndian.PutUint32(ret[18:], vt.ZoneId)
	}
	return ret
}

func getUDPNetAddr(sa syscall.Sockaddr) *net.UDPAddr {
	ret := &net.UDPAddr{}
	switch vt := sa.(type) {
	case *syscall.SockaddrInet4:
		ret.IP = make([]byte, len(vt.Addr))
		copy(ret.IP[:], vt.Addr[:])
		ret.Port = vt.Port
	case *syscall.SockaddrInet6:
		ret.IP = make([]byte, len(vt.Addr))
		copy(ret.IP[:], vt.Addr[:])
		ret.Port = vt.Port
		i, err := net.InterfaceByIndex(int(vt.ZoneId))
		if err == nil && i != nil {
			ret.Zone = i.Name
		}
	}
	return ret
}
