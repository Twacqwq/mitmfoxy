package netutil

import (
	"net"
	"net/url"
)

// defaultPortMap is the default port for each scheme
var defaultPortMap = map[string]string{
	"http":  "80",
	"https": "443",
}

// JoinHostPort returns host:port string
// if u.Port() is empty, use default port for u.Scheme
func JoinHostPort(u *url.URL) string {
	port := u.Port()
	if len(port) == 0 {
		port = defaultPortMap[u.Scheme]
	}

	return net.JoinHostPort(u.Hostname(), port)
}
