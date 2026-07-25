package master

import (
	"context"
	"encoding/json"
	"net/http"
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
	server   *Server
	conn     *websocket.Conn
	client   *common.Client
	send     chan *commandMsg
	closeOnce sync.Once
}

type handshakeMsg struct {
	Type     string `json:"type"`
	UUID     string `json:"uuid"`
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
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade failed", "error", err)
		return
	}

	var msg handshakeMsg
	if err := conn.ReadJSON(&msg); err != nil {
		s.logger.Error("Handshake read failed", "error", err)
		conn.Close()
		return
	}
	if msg.Type != "handshake" {
		s.logger.Error("Invalid handshake type", "type", msg.Type)
		conn.Close()
		return
	}

	clientID, err := uuid.Parse(msg.UUID)
	if err != nil {
		s.logger.Error("Invalid client UUID", "uuid", msg.UUID, "error", err)
		conn.Close()
		return
	}

	client := &common.Client{
		ID:       clientID,
		Hostname: msg.Hostname,
		OS:       msg.OS,
		MAC:      msg.MAC,
		Online:   true,
		LastSeen: time.Now(),
	}
	if err := s.storage.UpsertClient(r.Context(), client); err != nil {
		s.logger.Error("Failed to upsert client", "error", err)
		conn.Close()
		return
	}

	agentConn := &AgentConnection{
		server: s,
		conn:   conn,
		client: client,
		send:   make(chan *commandMsg, 10),
	}

	s.registerClient(agentConn)

	defer func() {
		s.unregisterClient(client.ID)
		s.storage.UpdateClientOnline(r.Context(), client.ID, false)
		agentConn.Close()
	}()

	go agentConn.writePump()
	agentConn.readPump()
}

func (s *Server) handleAgentResult(clientID uuid.UUID, msg *resultMsg) {
    if msg.Type == "stats" || msg.Type == "devices" {
        key := clientID.String() + ":" + msg.Type
        if ch, ok := pendingRequests.Load(key); ok {
            ch.(chan interface{}) <- msg.Data
            pendingRequests.Delete(key)
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
		case cmd, ok := <-ac.send:
			ac.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				ac.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
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

func (ac *AgentConnection) Close() {
	ac.closeOnce.Do(func() {
		ac.conn.Close()
		close(ac.send)
	})
}