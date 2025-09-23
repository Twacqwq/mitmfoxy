package protocol

import (
	"io"
	"maps"
	"net/http"

	"github.com/Twacqwq/mitmfoxy/internal/netutil"
	"github.com/Twacqwq/mitmfoxy/proxy/connection"
	"github.com/sirupsen/logrus"
)

// HTTPHandler is a handler for HTTP protocol
type HTTPHandler struct{}

func (h *HTTPHandler) Handle(w http.ResponseWriter, r *http.Request, session *connection.ConnSession) error {
	logrus.Infof("Req: %+v", r)

	addr := netutil.JoinHostPort(r.URL)
	serverConn, err := session.GetOrCreateServerConn(r.Context(), r, addr)
	if err != nil {
		return err
	}

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		return err
	}
	proxyReq.Header = r.Header

	proxyResp, err := serverConn.HTTPClient.Do(proxyReq)
	if err != nil {
		return err
	}
	defer proxyResp.Body.Close()

	logrus.Infof("Resp: %+v", proxyResp)

	// copy response header to writer
	maps.Copy(w.Header(), proxyResp.Header)
	w.WriteHeader(proxyResp.StatusCode)

	_, err = io.Copy(w, proxyResp.Body)
	if err != nil {
		return err
	}

	session.PutServerConn(addr, serverConn)
	return nil
}
