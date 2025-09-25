package protocol

import (
	"net/http"

	"github.com/Twacqwq/mitmfoxy/proxy/connection"
)

// Handler is the interface that wraps the Handle method.
type Handler interface {
	// Handle handles the request and writes response to w.
	// The enhancedConn contains connection-specific data.
	Handle(w http.ResponseWriter, r *http.Request, enhancedConn *connection.EnhancedConn) error
}
