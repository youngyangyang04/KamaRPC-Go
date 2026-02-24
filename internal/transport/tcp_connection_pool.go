package transport

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var ErrPoolClosed = errors.New("connection pool closed")

type ConnectionPool struct {
	addr string

	maxActive int

	conns []*TCPClient
	mu    sync.Mutex

	closed bool
	next   int
}

func NewConnectionPool(addr string, maxIdle, maxActive int) *ConnectionPool {
	return &ConnectionPool{
		addr:      addr,
		maxActive: maxActive,
		conns:     make([]*TCPClient, 0, maxActive),
	}
}

func (p *ConnectionPool) Acquire(ctx context.Context) (*TCPClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	// 如果没满，直接创建
	if len(p.conns) < p.maxActive {
		conn, err := newTCPClient(p.addr)
		if err != nil {
			return nil, err
		}
		p.conns = append(p.conns, conn)
		return conn, nil
	}

	// 轮询
	for i := 0; i < len(p.conns); i++ {
		idx := (p.next + i) % len(p.conns)
		conn := p.conns[idx]

		if atomic.LoadInt32(&conn.closed) == 0 {
			p.next = (idx + 1) % len(p.conns)
			return conn, nil
		}

		// 删除死亡连接
		p.conns = append(p.conns[:idx], p.conns[idx+1:]...)
		if len(p.conns) == 0 {
			break
		}
	}

	// 如果全死了，重新建
	conn, err := newTCPClient(p.addr)
	if err != nil {
		return nil, err
	}

	p.conns = append(p.conns, conn)
	return conn, nil
}

func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}
	p.closed = true

	for _, conn := range p.conns {
		conn.Close()
	}
}
