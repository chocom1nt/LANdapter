package master

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chocom1nt/LANdapter/internal/common"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 120 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 1048576
)

type AgentConnection struct {
	server    *Server
	conn      *websocket.Conn
	client    *common.Client
	send      chan *commandMsg
	done      chan struct{}
	sendMu    sync.Mutex // serializes queueing against connection shutdown
	closeOnce sync.Once
}

type handshakeMsg struct {
	Type     string `json:"type"`
	UUID     string `json:"uuid"`
	Token    string `json:"token"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	MAC      string `json:"mac"`
}

type commandMsg struct {
	Type    string          `json:"type"`
	JobID   string          `json:"job_id"`
	Payload json.RawMessage `json:"payload"`
}

type resultMsg struct {
	Type           string      `json:"type"`
	JobID          string      `json:"job_id"`
	Status         string      `json:"status"`
	Output         string      `json:"output,omitempty"`
	Error          string      `json:"error,omitempty"`
	Data           interface{} `json:"data,omitempty"`
	SnapshotBefore interface{} `json:"snapshot_before,omitempty"`
	SnapshotAfter  interface{} `json:"snapshot_after,omitempty"`
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil {
		http.Error(w, "Storage unavailable", http.StatusServiceUnavailable)
		return
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Warn("WebSocket upgrade rejected", "error", err)
		return
	}
	defer func() { _ = conn.Close() }()

	// Bound the unauthenticated handshake by both size and time.
	// Handshake messages should be small; enlarge the limit only after authentication.
	conn.SetReadLimit(4096)
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		s.logger.Warn("Unable to set handshake deadline", "error", err)
		return
	}
	var msg handshakeMsg
	if err := conn.ReadJSON(&msg); err != nil {
		s.logger.Warn("Handshake read failed", "error", err)
		return
	}
	if msg.Type != "handshake" || len(msg.Token) < 32 || len(msg.Token) > 256 {
		s.logger.Warn("Invalid agent handshake")
		return
	}
	clientID, err := uuid.Parse(msg.UUID)
	if err != nil || clientID == uuid.Nil {
		s.logger.Warn("Invalid client UUID")
		return
	}

	storedHash, err := s.storage.GetClientTokenHash(r.Context(), clientID)
	if err != nil {
		s.logger.Error("Unable to verify agent credential", "client", clientID, "error", err)
		return
	}
	providedHash := hashAgentToken(msg.Token)
	if len(storedHash) != len(providedHash) || subtle.ConstantTimeCompare([]byte(storedHash), []byte(providedHash)) != 1 {
		s.logger.Warn("Rejected agent with invalid credential", "client", clientID)
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid agent credential"),
			time.Now().Add(time.Second))
		return
	}
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		s.logger.Warn("Unable to clear handshake deadline", "error", err)
		return
	}

	client := &common.Client{
		ID:       clientID,
		Hostname: strings.TrimSpace(msg.Hostname),
		OS:       strings.TrimSpace(msg.OS),
		MAC:      strings.TrimSpace(msg.MAC),
		Online:   true,
		LastSeen: time.Now(),
	}
	if client.Hostname == "" {
		client.Hostname = "unknown"
	}
	if client.OS == "" {
		client.OS = "unknown"
	}
	if err := s.storage.UpsertClient(r.Context(), client); err != nil {
		s.logger.Error("Failed to upsert authenticated client", "error", err)
		return
	}

	agentConn := &AgentConnection{
		server: s,
		conn:   conn,
		client: client,
		send:   make(chan *commandMsg, 10),
		done:   make(chan struct{}),
	}
	s.registerClient(agentConn)

	defer func() {
		if s.unregisterClient(client.ID, agentConn) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := s.storage.UpdateClientOnline(ctx, client.ID, false); err != nil {
				s.logger.Warn("Failed to mark disconnected client offline", "client", client.ID, "error", err)
			}
		}
		agentConn.Close()
	}()

	go agentConn.writePump()
	agentConn.readPump()
}

func (s *Server) handleAgentResult(clientID uuid.UUID, msg *resultMsg) {
	if msg.Type == "stats" || msg.Type == "devices" {
		if _, err := uuid.Parse(msg.JobID); err != nil {
			s.logger.Warn("Invalid request ID in agent response", "client", clientID)
			return
		}
		key := clientID.String() + ":" + msg.JobID
		if ch, ok := pendingRequests.LoadAndDelete(key); ok {
			select {
			case ch.(chan interface{}) <- msg.Data:
			default:
				s.logger.Warn("Agent response channel already closed or full", "client", clientID)
			}
		}
		return
	}

	jobID, err := uuid.Parse(msg.JobID)
	if err != nil {
		s.logger.Error("Invalid job ID in result", "job_id", msg.JobID)
		return
	}

	result, err := s.storage.GetJobResult(context.Background(), jobID, clientID)
	if err != nil {
		s.logger.Error("Failed to get job result", "error", err)
		return
	}
	if result == nil {
		s.logger.Error("Job result not found", "job", jobID, "client", clientID)
		return
	}

	now := time.Now()
	if msg.Status == "success" {
		result.Status = "success"
		result.Output = &msg.Output
		result.FinishedAt = &now
	} else {
		result.Status = "failed"
		result.Error = &msg.Error
		result.FinishedAt = &now
	}

	if msg.SnapshotBefore != nil {
		if b, err := json.Marshal(msg.SnapshotBefore); err == nil {
			raw := json.RawMessage(b)
			result.SnapshotBefore = &raw
		} else {
			s.logger.Warn("Failed to marshal snapshot_before", "error", err)
		}
	}
	if msg.SnapshotAfter != nil {
		if b, err := json.Marshal(msg.SnapshotAfter); err == nil {
			raw := json.RawMessage(b)
			result.SnapshotAfter = &raw
		} else {
			s.logger.Warn("Failed to marshal snapshot_after", "error", err)
		}
	}

	if err := s.storage.UpdateJobResult(context.Background(), result); err != nil {
		s.logger.Error("Failed to update job result", "error", err)
	}
}

func (ac *AgentConnection) readPump() {
	defer ac.Close()
	ac.conn.SetReadLimit(maxMsgSize)
	ac.conn.SetReadDeadline(time.Now().Add(pongWait))
	ac.conn.SetPongHandler(func(string) error {
		ac.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg resultMsg
		if err := ac.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				ac.server.logger.Error("WebSocket read error", "error", err)
			} else {
				ac.server.logger.Warn("WebSocket read error (normal)", "error", err)
			}
			break
		}
		ac.conn.SetReadDeadline(time.Now().Add(pongWait))
		ac.server.handleAgentResult(ac.client.ID, &msg)
	}
}

func (ac *AgentConnection) writePump() {
	defer ac.Close()
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ac.done:
			return
		case cmd := <-ac.send:
			ac.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ac.conn.WriteJSON(cmd); err != nil {
				ac.server.logger.Error("Write command error", "error", err)
				return
			}
		case <-ticker.C:
			ac.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ac.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (ac *AgentConnection) enqueue(cmd *commandMsg) error {
	ac.sendMu.Lock()
	defer ac.sendMu.Unlock()
	select {
	case <-ac.done:
		return fmt.Errorf("connection is closed")
	default:
	}
	select {
	case <-ac.done:
		return fmt.Errorf("connection is closed")
	case ac.send <- cmd:
		return nil
	default:
		return fmt.Errorf("send queue is full")
	}
}

func (ac *AgentConnection) Close() {
	ac.closeOnce.Do(func() {
		ac.sendMu.Lock()
		close(ac.done)
		_ = ac.conn.Close()
		ac.sendMu.Unlock()
	})
}
