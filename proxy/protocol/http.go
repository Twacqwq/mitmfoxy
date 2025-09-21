package protocol

import (
	"context"
	"io"
	"maps"
	"net"
	"net/http"

	"github.com/Twacqwq/mitmfoxy/proxy/connection"
	"github.com/sirupsen/logrus"
)

// HTTPHandler is a handler for HTTP protocol
type HTTPHandler struct{}

func (h *HTTPHandler) Handle(w http.ResponseWriter, r *http.Request, session *connection.ConnSession) error {
	logrus.Infof("Req: %+v", r)

	if session.HTTPClient == nil {
		session.HTTPClient = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return session.DialFn.Dial(ctx, r)
				},
				DisableCompression: true,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		return err
	}
	proxyReq.Header = r.Header

	proxyResp, err := session.HTTPClient.Do(proxyReq)
	if err != nil {
		return err
	}
	defer proxyResp.Body.Close()

	logrus.Infof("Resp: %+v", proxyResp)

	// copy response header to writer
	maps.Copy(w.Header(), proxyResp.Header)
	w.WriteHeader(proxyResp.StatusCode)

	_, err = io.Copy(w, proxyResp.Body)
	return err
}
