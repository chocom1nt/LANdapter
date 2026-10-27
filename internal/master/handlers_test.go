package master

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"log/slog"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chocom1nt/LANdapter/internal/common"
	"github.com/chocom1nt/LANdapter/storage"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestHandleClientsUsesActiveWebSocketForOnlineStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")
	store := storage.NewPostgresStorageWithDB(sqlxDB)
	logger := common.InitLogger(slog.LevelInfo)
	srv := NewServer(&common.MasterConfig{}, logger, store)

	activeID := uuid.New()
	staleID := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "hostname", "os", "mac", "online", "last_seen"}).
		AddRow(activeID, "host-active", "windows", "00:11:22:33:44:55", false, now).
		AddRow(staleID, "host-stale", "linux", "00:11:22:33:44:66", true, now)
	mock.ExpectQuery(`(?s)SELECT.+FROM clients`).WillReturnRows(rows)
	srv.clients[activeID] = &AgentConnection{}

	req := httptest.NewRequest("GET", "/api/v1/clients", nil)
	w := httptest.NewRecorder()
	srv.handleClients(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var clients []common.Client
	if err := json.NewDecoder(resp.Body).Decode(&clients); err != nil {
		t.Fatal(err)
	}
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}
	if !clients[0].Online || clients[0].ID != activeID {
		t.Errorf("active WebSocket should be reported online: %#v", clients[0])
	}
	if clients[1].Online || clients[1].ID != staleID {
		t.Errorf("stale database online flag should be overridden: %#v", clients[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestHandleClientsOnlineFilterUsesActiveConnections(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := storage.NewPostgresStorageWithDB(sqlx.NewDb(db, "postgres"))
	srv := NewServer(&common.MasterConfig{}, common.InitLogger(slog.LevelInfo), store)
	activeID := uuid.New()
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "hostname", "os", "mac", "online", "last_seen"}).
		AddRow(activeID, "host-active", "linux", "", false, now)
	mock.ExpectQuery(`(?s)SELECT.+FROM clients`).WillReturnRows(rows)
	srv.clients[activeID] = &AgentConnection{}

	req := httptest.NewRequest("GET", "/api/v1/clients?online=true", nil)
	w := httptest.NewRecorder()
	srv.handleClients(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var clients []common.Client
	if err := json.NewDecoder(w.Body).Decode(&clients); err != nil {
		t.Fatal(err)
	}
	if len(clients) != 1 || !clients[0].Online || clients[0].ID != activeID {
		t.Fatalf("online filter should use live connections, got %#v", clients)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestHandleFileDownload_SafePath(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest("GET", "/api/v1/files/../../etc/passwd", nil)
	w := httptest.NewRecorder()
	srv.handleFileDownload(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Invalid file ID")) {
		t.Error("expected error message about invalid file ID")
	}
}
