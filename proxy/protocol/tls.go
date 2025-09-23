package protocol

import (
	"fmt"
	"net/http"

	"github.com/Twacqwq/mitmfoxy/proxy/connection"
)

// TlsHandler is a handler for HTTPS/TLS protocol
type TlsHandler struct{}

func (t *TlsHandler) Handle(w http.ResponseWriter, r *http.Request, session *connection.ConnSession) error {
	if r.Method != http.MethodConnect {
		return fmt.Errorf("the request method must be CONNECT")
	}

	// Connection Established
	// w.WriteHeader(http.StatusOK)

	// // hijack client connection
	// hijacker, ok := w.(http.Hijacker)
	// if !ok {
	// 	return errors.New("hijacking not supported")
	// }
	// clientConn, _, err := hijacker.Hijack()
	// if err != nil {
	// 	return err
	// }

	// get server connection
	// serverConn, err := session.DialFn.Dial(context.Background(), r)
	// if err != nil {
	// 	clientConn.Close()
	// 	return err
	// }

	return nil
}
