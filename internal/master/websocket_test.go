package master

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"log/slog"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chocom1nt/LANdapter/internal/common"
	"github.com/chocom1nt/LANdapter/storage"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
)

func TestWebSocketRejectsInvalidAgentCredential(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := storage.NewPostgresStorageWithDB(sqlx.NewDb(db, "postgres"))
	srv := NewServer(&common.MasterConfig{}, common.InitLogger(slog.LevelError), store)
	server := httptest.NewServer(http.HandlerFunc(srv.handleWebSocket))
	defer server.Close()

	id := uuid.New()
	expectedToken := "correct-agent-token-that-is-at-least-32-bytes"
	mock.ExpectQuery(`SELECT token_hash FROM clients WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"token_hash"}).AddRow(hashAgentToken(expectedToken)))

	wsURL := "ws" + server.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	handshake := handshakeMsg{Type: "handshake", UUID: id.String(), Token: "wrong-agent-token-that-is-also-over-32-bytes"}
	if err := conn.WriteJSON(handshake); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected server to close a connection with an invalid credential")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
