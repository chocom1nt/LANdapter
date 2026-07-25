package master

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"log/slog"

	"github.com/chocom1nt/LANdapter/internal/common"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestWebSocketHandshake(t *testing.T) {
	srv := &Server{
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		logger:   common.InitLogger(slog.LevelInfo),
	}
	ts := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer ts.Close()

	wsURL := "ws" + ts.URL[len("http"):] + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	handshake := handshakeMsg{
		Type:     "handshake",
		UUID:     uuid.New().String(),
		Hostname: "test-host",
		OS:       "linux",
		MAC:      "00:11:22:33:44:55",
	}
	if err := conn.WriteJSON(handshake); err != nil {
		t.Fatal(err)
	}

	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Error("expected read timeout, got message")
	}
}