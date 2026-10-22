package master

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chocom1nt/LANdapter/internal/common"
	"github.com/chocom1nt/LANdapter/storage"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Server struct {
	config     *common.MasterConfig
	logger     *slog.Logger
	storage    storage.Storage
	httpServer *http.Server
	wsServer   *http.Server
	upgrader   websocket.Upgrader

	clientsMu sync.RWMutex
	clients   map[uuid.UUID]*AgentConnection
}

// CORS middleware
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Agent-ID")
		w.Header().Set("Access-Control-Max-Age", "600")
		w.Header().Set("Vary", "Origin")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func NewServer(cfg *common.MasterConfig, logger *slog.Logger, store storage.Storage) *Server {
	return &Server{
		config:  cfg,
		logger:  logger,
		storage: store,
		// Agents are native clients and should not send a browser Origin header.
		// Reject browser-originated websocket requests to mitigate drive-by access.
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == "" }},
		clients:  make(map[uuid.UUID]*AgentConnection),
	}
}

func (s *Server) Start() error {
	if len(strings.TrimSpace(os.Getenv(adminTokenEnv))) < 32 {
		return fmt.Errorf("%s must be set to a secret of at least 32 characters", adminTokenEnv)
	}
	// REST API
	muxHTTP := http.NewServeMux()
	// Обёртываем все обработчики CORS middleware
	muxHTTP.HandleFunc("/api/v1/install", corsMiddleware(s.requireAdmin(s.handleInstall)))
	muxHTTP.HandleFunc("/api/v1/clients", corsMiddleware(s.requireAdmin(s.handleClients)))
	muxHTTP.HandleFunc("/api/v1/wol", corsMiddleware(s.requireAdmin(s.handleWOL)))
	muxHTTP.HandleFunc("/api/v1/parse-driver", corsMiddleware(s.requireAdmin(s.handleParseDriver)))
	muxHTTP.HandleFunc("/api/v1/upload", corsMiddleware(s.requireAdmin(s.handleUpload)))
	muxHTTP.HandleFunc("/api/v1/clients/{id}/devices", corsMiddleware(s.requireAdmin(s.handleClientDevices)))
	muxHTTP.HandleFunc("/api/v1/clients/{id}/stats", corsMiddleware(s.requireAdmin(s.handleClientStats)))
	muxHTTP.HandleFunc("/api/v1/files", corsMiddleware(s.requireAdmin(s.handleListFiles))) // GET list
	muxHTTP.HandleFunc("/api/v1/files/", corsMiddleware(s.requireAdmin(s.handleFileOperations)))
	muxHTTP.HandleFunc("/api/v1/agent/files/", corsMiddleware(s.requireAgent(s.handleAgentFileDownload)))
	muxHTTP.HandleFunc("/api/v1/agents/enroll", corsMiddleware(s.requireAdmin(s.handleAgentEnroll)))

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.config.Host, s.config.HTTPPort),
		Handler:           muxHTTP,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	// WebSocket
	muxWS := http.NewServeMux()
	muxWS.HandleFunc("/ws", s.handleWebSocket) // для WebSocket CORS не нужен (upgrader уже разрешает)
	s.wsServer = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.config.Host, s.config.WSPort),
		Handler:           muxWS,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		s.logger.Info("Starting HTTP server", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server failed", "error", err)
		}
	}()
	go func() {
		s.logger.Info("Starting WebSocket server", "addr", s.wsServer.Addr)
		if err := s.wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("WebSocket server failed", "error", err)
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping servers...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP shutdown error", "error", err)
	}
	if err := s.wsServer.Shutdown(ctx); err != nil {
		s.logger.Error("WebSocket shutdown error", "error", err)
	}

	s.clientsMu.Lock()
	for _, conn := range s.clients {
		conn.Close()
	}
	s.clientsMu.Unlock()
	return nil
}

func (s *Server) registerClient(conn *AgentConnection) {
	s.clientsMu.Lock()
	old := s.clients[conn.client.ID]
	s.clients[conn.client.ID] = conn
	s.clientsMu.Unlock()
	if old != nil && old != conn {
		old.Close()
	}
}

// unregisterClient only removes the mapping if this connection still owns it.
// A delayed cleanup from an old connection must not remove a newer reconnect.
func (s *Server) unregisterClient(id uuid.UUID, conn *AgentConnection) bool {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	if current, ok := s.clients[id]; ok && current == conn {
		delete(s.clients, id)
		return true
	}
	return false
}

func (s *Server) sendCommand(clientID uuid.UUID, cmd *commandMsg) error {
	s.clientsMu.RLock()
	conn, ok := s.clients[clientID]
	s.clientsMu.RUnlock()
	if !ok {
		return fmt.Errorf("client %s not online", clientID)
	}
	if err := conn.enqueue(cmd); err != nil {
		return fmt.Errorf("unable to queue command for client %s: %w", clientID, err)
	}
	return nil
}
