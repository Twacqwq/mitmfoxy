package connection

import (
	"net"
	"sync"
	"time"
)

// ServerConnPool is a pool of server connections
type ServerConnPool struct {
	mu       sync.RWMutex
	conns    map[string][]*serverConn
	maxConns int
	timeout  time.Duration
}

func NewServerConnPool(maxConns int, timeout time.Duration) *ServerConnPool {
	return &ServerConnPool{
		conns:    make(map[string][]*serverConn),
		maxConns: maxConns,
		timeout:  timeout,
	}
}

// Get the connection from the connection pool, return nil if no connection is available
func (p *ServerConnPool) Get(addr string) *serverConn {
	p.mu.Lock()
	defer p.mu.Unlock()

	pool, ok := p.conns[addr]
	if !ok || len(pool) == 0 {
		return nil
	}

	for i := len(pool) - 1; i >= 0; i-- {
		conn := pool[i]
		if p.isConnValid(conn) {
			p.conns[addr] = append(pool[:i], pool[i+1:]...)
			return conn
		}
		conn.Close()
		pool = pool[:i]
	}

	// No valid connection found, remove the pool
	delete(p.conns, addr)

	return nil
}

// Put the connection back to the connection pool
func (p *ServerConnPool) Put(addr string, conn *serverConn) {
	if conn == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conns[addr] == nil {
		p.conns[addr] = make([]*serverConn, 0, p.maxConns)
	}

	// If the connection pool is full, close the connection
	if len(p.conns[addr]) >= p.maxConns {
		conn.Close()
		return
	}

	p.conns[addr] = append(p.conns[addr], conn)
}

// Close all connections in the connection pool
func (p *ServerConnPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, pool := range p.conns {
		for _, conn := range pool {
			conn.Close()
		}
		delete(p.conns, addr)
	}
}

// isConnValid is check if the connection is still valid
func (p *ServerConnPool) isConnValid(conn *serverConn) bool {
	if conn == nil || conn.Conn == nil {
		return false
	}

	if tcpConn, ok := conn.Conn.(*net.TCPConn); ok {
		if tcpConn == nil {
			return false
		}

		// Check if the connection is closed (easy way)
		// If the connection is closed, the Write operation will immediately return an error
		err := tcpConn.SetDeadline(time.Now().Add(1 * time.Nanosecond))
		if err != nil {
			return false
		}

		// Reset the deadline
		tcpConn.SetDeadline(time.Time{})
		return true
	}

	return true
}
