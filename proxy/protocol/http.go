package protocol

import (
	"bytes"
	"io"
	"maps"
	"net/http"

	"github.com/Twacqwq/mitmfoxy/internal/model"
	"github.com/Twacqwq/mitmfoxy/proxy/connection"
)

// HTTPHandler is a handler for HTTP protocol
type httpHandler struct {
	pcw *PacketCaptureWebSocket
}

func (h *httpHandler) Handle(w http.ResponseWriter, r *http.Request, enhancedConn *connection.EnhancedConn) error {
	ServerConn, err := enhancedConn.Session.Dialer.Dial(r.Context(), r)
	if err != nil {
		return err
	}
	enhancedConn.Session.ServerConn = connection.NewProxyServerConn(ServerConn)

	var reqBuf bytes.Buffer
	reqBody := io.TeeReader(r.Body, &reqBuf)
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), reqBody)
	if err != nil {
		return err
	}
	proxyReq.Header = r.Header

	proxyResp, err := enhancedConn.Session.ServerConn.Client.Do(proxyReq)
	if err != nil {
		return err
	}
	defer proxyResp.Body.Close()

	var respBuf bytes.Buffer
	respBody := io.TeeReader(proxyResp.Body, &respBuf)

	// copy response header to writer
	maps.Copy(w.Header(), proxyResp.Header)
	w.WriteHeader(proxyResp.StatusCode)

	_, err = io.Copy(w, respBody)
	if err != nil {
		return err
	}

	go func() {
		if !h.pcw.enabled {
			return
		}
		flow := model.BuildPacketCaptureFlow(proxyResp, r, &reqBuf, &respBuf)
		h.pcw.BroadcastJSON(flow)
	}()

	return nil
}

func NewHTTPHandler(pcw *PacketCaptureWebSocket) Handler {
	return &httpHandler{
		pcw: pcw,
	}
}
