package connection

import (
	"bufio"
	"context"
	"net"
	"net/http"
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

	// HTTPClient is used to make HTTP requests
	HTTPClient *http.Client
}

func NewConnSession() *ConnSession {
	return &ConnSession{}
}
