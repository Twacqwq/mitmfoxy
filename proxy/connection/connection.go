package connection

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
)

// EnhancedConn is a net.Conn with connection session
type EnhancedConn struct {
	net.Conn

	Session *ProxyConnSession
}

func NewEnhancedConn(c net.Conn) *EnhancedConn {
	return &EnhancedConn{
		Conn:    c,
		Session: NewProxyConnSession(c),
	}
}

// Dialer is an interface for dialing connections
type Dialer interface {
	// Dial dials a connection based on the request
	Dial(context.Context, *http.Request) (net.Conn, error)
}

// ProxyConnSession is a session for a proxy connection
type ProxyConnSession struct {
	Dialer

	ClientConn *ProxyClientConn
	ServerConn *ProxyServerConn
}

func NewProxyConnSession(c net.Conn) *ProxyConnSession {
	return &ProxyConnSession{
		ClientConn: NewProxyClientConn(c),
	}
}

// ProxyClientConn is a connection from the client to the proxy
type ProxyClientConn struct {
	ID              string
	ClientHelloInfo *tls.ClientHelloInfo
	Conn            net.Conn
	TlsConn         *tls.Conn
	IsTLS           bool
}

func NewProxyClientConn(c net.Conn) *ProxyClientConn {
	return &ProxyClientConn{
		Conn: c,
	}
}

// ProxyServerConn is a connection from the proxy to the server
type ProxyServerConn struct {
	ID           string
	TlsConn      *tls.Conn
	TlsConnState *tls.ConnectionState
	Addr         string
	Client       *http.Client
	Conn         net.Conn
}

func NewProxyServerConn(c net.Conn) *ProxyServerConn {
	return &ProxyServerConn{
		Conn: c,
		Client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return c, nil
				},
				DisableCompression: true,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}
