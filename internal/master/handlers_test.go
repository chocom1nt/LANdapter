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

func TestHandleClients(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")
	store := storage.NewPostgresStorageWithDB(sqlxDB) // ✅ исправлено
	logger := common.InitLogger(slog.LevelInfo)
	srv := NewServer(&common.MasterConfig{}, logger, store)

	rows := sqlmock.NewRows([]string{"id", "hostname", "os", "mac", "online", "last_seen"}).
		AddRow(uuid.New(), "host1", "windows", "00:11:22:33:44:55", true, time.Now())
	mock.ExpectQuery(`SELECT .+ FROM clients`).WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/api/v1/clients", nil)
	w := httptest.NewRecorder()
	srv.handleClients(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var clients []common.Client
	json.NewDecoder(resp.Body).Decode(&clients)
	if len(clients) != 1 {
		t.Errorf("expected 1 client, got %d", len(clients))
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