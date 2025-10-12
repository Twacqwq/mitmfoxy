package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Twacqwq/mitmfoxy/internal/cert"
	"github.com/Twacqwq/mitmfoxy/internal/netutil"
	"github.com/Twacqwq/mitmfoxy/proxy/connection"
	"github.com/Twacqwq/mitmfoxy/proxy/protocol"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// proxy listen addr
	Addr string

	// cert file
	CertFile string

	// key file
	KeyFile string

	// use websocket to recv packet capture
	UseWebsocket bool
}

type proxy struct {
	// proxy server
	server *http.Server

	// protocol handler map
	// scheme -> protocol.handler
	// e.g http -> http handler
	protocols map[string]protocol.Handler
}

func New(conf *Config) *proxy {
	p := &proxy{
		server: &http.Server{
			Addr: conf.Addr,
			ConnContext: func(ctx context.Context, c net.Conn) context.Context {
				enhancedConn := connection.NewEnhancedConn(c)
				return context.WithValue(ctx, connection.EnhancedConnContextKey, enhancedConn)
			},
		},
		protocols: make(map[string]protocol.Handler),
	}

	// init cert manager
	certManager, err := cert.NewManager(conf.CertFile, conf.KeyFile)
	if err != nil {
		certManager = &cert.Manager{}
	}

	// register protocol handler
	p.RegisterProtocolHandler("http", protocol.NewHTTPHandler())
	p.RegisterProtocolHandler("https", protocol.NewTLSHandler(certManager))

	mux := http.NewServeMux()
	mux.Handle("/", p)
	mux.Handle("/ws", protocol.NewPacketCaptureWebsocket(conf.UseWebsocket))

	p.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			p.ServeHTTP(w, r)
			return
		}

		mux.ServeHTTP(w, r)
	})
	return p
}

// Start is run proxy server
func (p *proxy) Start() error {
	ln, err := net.Listen("tcp", p.server.Addr)
	if err != nil {
		return err
	}

	logrus.Infof("Listen Addr: %s", p.server.Addr)
	return p.server.Serve(ln)
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get enhanced conn from context
	enhancedConn := connection.MustGetEnhancedConnFromContext(r.Context())

	if r.Method == http.MethodConnect {
		if len(r.URL.Scheme) == 0 {
			r.URL.Scheme = "https"
		}
	} else {
		if !r.URL.IsAbs() || len(r.Host) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	// get handler with scheme
	handler, ok := p.protocols[r.URL.Scheme]
	if !ok {
		http.Error(w, "Unsupported protocol", http.StatusBadRequest)
		return
	}

	if enhancedConn.Session.Dialer == nil {
		enhancedConn.Session.Dialer = p
	}

	// handle request
	if err := handler.Handle(w, r, enhancedConn); err != nil {
		logrus.Errorf("Protocol handling error: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
	}
}

// RegisterProtocolHandler registers a protocol handler for a given scheme
func (p *proxy) RegisterProtocolHandler(scheme string, handler protocol.Handler) {
	p.protocols[scheme] = handler
}

// Dial dials a connection based on the request
func (p *proxy) Dial(ctx context.Context, r *http.Request) (net.Conn, error) {
	// get proxy url
	proxyUrl, err := p.GetProxyURL(ctx, r)
	if err != nil {
		return nil, err
	}

	// get target addr
	addr := netutil.JoinHostPort(r.URL)
	if len(addr) == 0 {
		return nil, errors.New("invalid target address")
	}

	if proxyUrl != nil {
		// TODO external proxy
	}

	return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", addr)
}

func (p *proxy) GetProxyURL(ctx context.Context, r *http.Request) (*url.URL, error) {
	if len(r.URL.Scheme) == 0 {
		r.URL.Scheme = "https"
	}

	return http.ProxyFromEnvironment(&http.Request{
		URL: &url.URL{
			Scheme: r.URL.Scheme,
			Host:   r.Host,
		},
	})
}
