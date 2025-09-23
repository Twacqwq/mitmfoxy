package connection

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// EnhancedConn is a net.Conn with buffered reader and connection session
type EnhancedConn struct {
	net.Conn

	bufr    *bufio.Reader
	session *ConnSession
}

// Peek n bytes from the connection without consuming them
func (e *EnhancedConn) Peek(n int) ([]byte, error) {
	return e.bufr.Peek(n)
}

// GetSession returns the connection session
func (e *EnhancedConn) GetSession() *ConnSession {
	return e.session
}

func NewEnhancedConn(c net.Conn) *EnhancedConn {
	connSession := NewConnSession()

	return &EnhancedConn{
		Conn:    c,
		bufr:    bufio.NewReader(c),
		session: connSession,
	}
}

// Dialer is an interface for dialing connections
type Dialer interface {
	// Dial dials a connection based on the request
	Dial(context.Context, *http.Request) (net.Conn, error)
}

// ConnSession holds session-specific data for a connection
type ConnSession struct {
	// DialFn is used to dial connections
	DialFn Dialer

	// serverConnPool is used to pool server connections
	serverConnPool *ServerConnPool
}

// GetOrCreateServerConn returns a server connection for the given address
func (c *ConnSession) GetOrCreateServerConn(ctx context.Context, r *http.Request, addr string) (*serverConn, error) {
	logrus.Debugf("ServerConnPool: %+v", c.serverConnPool)

	// Try to get from the connection pool first
	if conn := c.serverConnPool.Get(addr); conn != nil {
		return conn, nil
	}

	// Dial a new connection
	rawConn, err := c.DialFn.Dial(ctx, r)
	if err != nil {
		return nil, err
	}

	//
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return rawConn, nil
			},
			DisableCompression: true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return NewServerConn(rawConn, httpClient), nil
}

// PutServerConn puts a server connection back into the pool
func (c *ConnSession) PutServerConn(addr string, conn *serverConn) {
	c.serverConnPool.Put(addr, conn)
}

func NewConnSession() *ConnSession {
	return &ConnSession{
		serverConnPool: NewServerConnPool(1<<10, 10*time.Second),
	}
}

type serverConn struct {
	net.Conn

	// HTTPClient is used to make HTTP requests
	HTTPClient *http.Client
}

func NewServerConn(c net.Conn, client *http.Client) *serverConn {
	return &serverConn{
		Conn:       c,
		HTTPClient: client,
	}
}
