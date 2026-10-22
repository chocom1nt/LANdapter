package master

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

const adminTokenEnv = "LANDAPTER_ADMIN_TOKEN"

func hashAgentToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newAgentToken() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(secret), nil
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

func validAdminToken(provided string) bool {
	expected := strings.TrimSpace(os.Getenv(adminTokenEnv))
	if len(expected) < 32 || len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)
		if r.Method == http.MethodOptions {
			next(w, r)
			return
		}
		if !validAdminToken(bearerToken(r)) {
			if len(strings.TrimSpace(os.Getenv(adminTokenEnv))) < 32 {
				http.Error(w, "Administrative API authentication is not configured", http.StatusServiceUnavailable)
			} else {
				http.Error(w, "Invalid or missing administrative bearer token", http.StatusUnauthorized)
			}
			return
		}
		next(w, r)
	}
}

// requireAgent authorizes agent-only file downloads with the per-agent token.
// The UUID is not a secret; it is bound to the token hash stored in PostgreSQL.
func (s *Server) requireAgent(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(r.Header.Get("X-Agent-ID")))
		token := bearerToken(r)
		if err != nil || id == uuid.Nil || len(token) < 32 || s.storage == nil {
			http.Error(w, "Invalid agent credentials", http.StatusUnauthorized)
			return
		}
		storedHash, err := s.storage.GetClientTokenHash(r.Context(), id)
		if err != nil {
			s.logger.Error("Unable to verify agent download credential", "client", id, "error", err)
			http.Error(w, "Authentication unavailable", http.StatusServiceUnavailable)
			return
		}
		providedHash := hashAgentToken(token)
		if len(storedHash) != len(providedHash) || subtle.ConstantTimeCompare([]byte(storedHash), []byte(providedHash)) != 1 {
			http.Error(w, "Invalid agent credentials", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// handleAgentEnroll creates or rotates a per-agent credential. Only a SHA-256
// hash is persisted; the raw token is returned exactly once to the caller.
func (s *Server) handleAgentEnroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var req struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	var id uuid.UUID
	if strings.TrimSpace(req.AgentID) == "" {
		id = uuid.New()
	} else {
		parsed, err := uuid.Parse(strings.TrimSpace(req.AgentID))
		if err != nil || parsed == uuid.Nil {
			http.Error(w, "agent_id must be a valid, non-zero UUID", http.StatusBadRequest)
			return
		}
		id = parsed
	}

	token, err := newAgentToken()
	if err != nil {
		s.logger.Error("Agent token generation failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if s.storage == nil {
		http.Error(w, "Storage unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := s.storage.SetClientTokenHash(r.Context(), id, hashAgentToken(token)); err != nil {
		s.logger.Error("Agent credential persistence failed", "agent_id", id, "error", err)
		http.Error(w, "Unable to save agent credential", http.StatusInternalServerError)
		return
	}

	// Invalidate an active connection immediately when rotating a credential.
	s.clientsMu.RLock()
	active := s.clients[id]
	s.clientsMu.RUnlock()
	if active != nil {
		active.Close()
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"agent_id": id.String(),
		"token":    token,
		"agent_config": map[string]string{
			"uuid":  id.String(),
			"token": token,
		},
	}); err != nil {
		s.logger.Warn("Failed to encode issued credential", "error", err)
	}
}
