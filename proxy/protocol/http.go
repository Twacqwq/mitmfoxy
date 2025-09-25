package protocol

import (
	"io"
	"maps"
	"net/http"

	"github.com/Twacqwq/mitmfoxy/proxy/connection"
	"github.com/sirupsen/logrus"
)

// HTTPHandler is a handler for HTTP protocol
type httpHandler struct{}

func (h *httpHandler) Handle(w http.ResponseWriter, r *http.Request, enhancedConn *connection.EnhancedConn) error {
	logrus.Infof("Req: %+v", r)

	ServerConn, err := enhancedConn.Session.Dialer.Dial(r.Context(), r)
	if err != nil {
		return err
	}
	enhancedConn.Session.ServerConn = connection.NewProxyServerConn(ServerConn)

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		return err
	}
	proxyReq.Header = r.Header

	proxyResp, err := enhancedConn.Session.ServerConn.Client.Do(proxyReq)
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

	return nil
}

func NewHTTPHandler() Handler {
	return &httpHandler{}
}
