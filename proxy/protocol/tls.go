package protocol

import (
	"context"
	"crypto/tls"
	"io"
	"maps"
	"net"
	"net/http"

	"github.com/Twacqwq/mitmfoxy/internal/cert"
	"github.com/Twacqwq/mitmfoxy/proxy/connection"
	"github.com/sirupsen/logrus"
)

type tlsHandler struct {
	server      *http.Server
	tlsListener *tlsListener
	certManager *cert.Manager
}

func (t *tlsHandler) Handle(w http.ResponseWriter, r *http.Request, enhancedConn *connection.EnhancedConn) error {
	// 200 Connection Established
	w.WriteHeader(http.StatusOK)
	hijackConn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return err
	}

	serverConn, err := enhancedConn.Session.Dialer.Dial(r.Context(), r)
	if err != nil {
		return err
	}
	enhancedConn.Session.ServerConn = connection.NewProxyServerConn(serverConn)

	// tls handshake
	if err := t.Handshake(context.Background(), hijackConn, enhancedConn); err != nil {
		logrus.Error(err)
		return err
	}

	// Forward traffic
	t.tlsListener.forward(&forwardConn{enhancedConn.Session.ClientConn.TlsConn, enhancedConn})
	return nil
}

func (t *tlsHandler) Handshake(ctx context.Context, hijackConn net.Conn, enhancedConn *connection.EnhancedConn) error {
	clientTlsConn := tls.Server(hijackConn, &tls.Config{
		SessionTicketsDisabled: true,
		GetConfigForClient: func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
			enhancedConn.Session.ClientConn.ClientHelloInfo = chi

			var nextProtocols []string
			chErr := make(chan error, 1)
			chState := make(chan *tls.ConnectionState)

			go func() {
				if err := t.tlsServerHandshake(ctx, enhancedConn, chState); err != nil {
					logrus.Error(err)
					chErr <- err
					return
				}
			}()

			// wait server handshake done
			select {
			case err := <-chErr:
				return nil, err
			case <-ctx.Done():
				return nil, ctx.Err()
			case tlsConnState := <-chState:
				nextProtocols = append([]string{tlsConnState.NegotiatedProtocol}, nextProtocols...)
			}
			close(chState)
			close(chErr)

			logrus.Infof("SNI: %s", chi.ServerName)
			c, err := t.certManager.GetCert(chi.ServerName)
			if err != nil {
				logrus.Errorf("get cert error: %v", err)
				return nil, err
			}

			return &tls.Config{
				SessionTicketsDisabled: true,
				Certificates:           []tls.Certificate{*c},
				NextProtos:             nextProtocols,
			}, nil
		},
	})

	// tls client handshake
	if err := clientTlsConn.HandshakeContext(ctx); err != nil {
		logrus.Errorf("tls handshake error: %v", err)
		return err
	}
	enhancedConn.Session.ClientConn.TlsConn = clientTlsConn

	return nil
}

func (t *tlsHandler) tlsServerHandshake(ctx context.Context, enhancedConn *connection.EnhancedConn, chState chan *tls.ConnectionState) error {
	chi := enhancedConn.Session.ClientConn.ClientHelloInfo
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         chi.ServerName,
		NextProtos:         chi.SupportedProtos,
		CipherSuites:       chi.CipherSuites,
	}

	if len(chi.SupportedVersions) > 0 {
		tlsConfig.MinVersion, tlsConfig.MaxVersion = chi.SupportedVersions[0], chi.SupportedVersions[0]
		for _, ver := range chi.SupportedVersions {
			if ver < tlsConfig.MinVersion {
				tlsConfig.MinVersion = ver
			}
			if ver > tlsConfig.MaxVersion {
				tlsConfig.MaxVersion = ver
			}
		}
	}

	enhancedConn.Session.ServerConn.TlsConn = tls.Client(enhancedConn.Session.ServerConn.Conn, tlsConfig)
	if err := enhancedConn.Session.ServerConn.TlsConn.HandshakeContext(ctx); err != nil {
		return err
	}

	enhancedConn.Session.ServerConn.Client = &http.Client{
		Transport: &http.Transport{
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return enhancedConn.Session.ServerConn.TlsConn, nil
			},
			ForceAttemptHTTP2:  true,
			DisableCompression: true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	tlsState := enhancedConn.Session.ServerConn.TlsConn.ConnectionState()
	chState <- &tlsState
	return nil
}

func (t *tlsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	traceConn, ok := r.Context().Value(connection.TLSConnContextKey).(*forwardConn)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(r.URL.Host) == 0 {
		r.URL.Host = r.Host
	}
	if len(r.URL.Scheme) == 0 {
		r.URL.Scheme = "https"
	}

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	proxyReq.Header = r.Header

	proxyResp, err := traceConn.enhancedConn.Session.ServerConn.Client.Do(proxyReq)
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer proxyResp.Body.Close()

	logrus.Infof("Resp: %+v", proxyResp)

	// copy response header to writer
	maps.Copy(w.Header(), proxyResp.Header)
	w.WriteHeader(proxyResp.StatusCode)

	_, err = io.Copy(w, proxyResp.Body)
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func NewTLSHandler(certManager *cert.Manager) Handler {
	handler := &tlsHandler{
		tlsListener: &tlsListener{
			chConn: make(chan net.Conn),
		},
		certManager: certManager,
	}
	handler.server = &http.Server{
		Handler: handler,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, connection.TLSConnContextKey, c)
		},
	}

	go func() {
		logrus.Infof("tls server started")
		if err := handler.server.Serve(handler.tlsListener); err != nil {
			logrus.Errorf("tls server error: %v", err)
			return
		}
	}()

	return handler
}

type tlsListener struct {
	net.Listener

	chConn chan net.Conn
}

func (t *tlsListener) Accept() (net.Conn, error) {
	return <-t.chConn, nil
}

func (t *tlsListener) forward(conn net.Conn) {
	t.chConn <- conn
}

type forwardConn struct {
	net.Conn

	enhancedConn *connection.EnhancedConn
}
