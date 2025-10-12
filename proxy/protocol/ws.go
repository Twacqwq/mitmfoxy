package protocol

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type PacketCaptureWebSocket struct {
	enabled  bool
	mu       sync.RWMutex
	upgrader *websocket.Upgrader
	conns    map[*websocket.Conn]struct{}
}

func (p *PacketCaptureWebSocket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !p.enabled {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	c, err := p.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Error(err)
		return
	}

	p.mu.Lock()
	p.conns[c] = struct{}{}
	p.mu.Unlock()

	go func() {
		for {
			if _, _, err := c.NextReader(); err != nil {
				c.Close()
				p.mu.Lock()
				delete(p.conns, c)
				p.mu.Unlock()
				break
			}
		}
	}()
}

func (p *PacketCaptureWebSocket) BroadcastJSON(data any) {
	if !p.enabled {
		return
	}

	for c := range p.conns {
		if err := c.WriteJSON(data); err != nil {
			logrus.Error(err)
		}
	}
}

func NewPacketCaptureWebsocket(enabled bool) *PacketCaptureWebSocket {
	return &PacketCaptureWebSocket{
		enabled: enabled,
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		conns: make(map[*websocket.Conn]struct{}),
	}
}
