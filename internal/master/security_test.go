package master

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"log/slog"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/chocom1nt/LANdapter/internal/common"
	"github.com/chocom1nt/LANdapter/storage"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestRequireAdminBearerToken(t *testing.T) {
	secret := strings.Repeat("a", 48)
	t.Setenv(adminTokenEnv, secret)
	srv := NewServer(&common.MasterConfig{}, common.InitLogger(slog.LevelError), nil)
	handler := srv.requireAdmin(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	tests := []struct {
		name  string
		token string
		want  int
	}{
		{name: "missing", want: http.StatusUnauthorized},
		{name: "wrong", token: strings.Repeat("b", 48), want: http.StatusUnauthorized},
		{name: "valid", token: secret, want: http.StatusNoContent},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)
			if resp.Code != tc.want {
				t.Fatalf("got status %d, want %d", resp.Code, tc.want)
			}
		})
	}
}

func TestEnrollAgentIssuesOneTimeTokenAndStoresOnlyHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := storage.NewPostgresStorageWithDB(sqlx.NewDb(db, "postgres"))
	srv := NewServer(&common.MasterConfig{}, common.InitLogger(slog.LevelError), store)
	id := uuid.New()
	mock.ExpectExec(`INSERT INTO clients`).WithArgs(id, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))

	body, _ := json.Marshal(map[string]string{"agent_id": id.String()})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/enroll", bytes.NewReader(body))
	resp := httptest.NewRecorder()
	srv.handleAgentEnroll(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("got status %d: %s", resp.Code, resp.Body.String())
	}

	var credential struct {
		AgentID     string `json:"agent_id"`
		Token       string `json:"token"`
		AgentConfig struct {
			UUID  string `json:"uuid"`
			Token string `json:"token"`
		} `json:"agent_config"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &credential); err != nil {
		t.Fatal(err)
	}
	if credential.AgentID != id.String() || credential.AgentConfig.UUID != id.String() {
		t.Fatalf("unexpected agent ID in credential: %+v", credential)
	}
	if len(credential.Token) < 32 || credential.AgentConfig.Token != credential.Token {
		t.Fatalf("invalid issued token")
	}
	if hashAgentToken(credential.Token) == credential.Token {
		t.Fatal("raw token was not hashed")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
