package master

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

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
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        next(w, r)
    }
}

func NewServer(cfg *common.MasterConfig, logger *slog.Logger, store storage.Storage) *Server {
    return &Server{
        config:   cfg,
        logger:   logger,
        storage:  store,
        upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
        clients:  make(map[uuid.UUID]*AgentConnection),
    }
}

func (s *Server) Start() error {
    // REST API
    muxHTTP := http.NewServeMux()
    // Обёртываем все обработчики CORS middleware
    muxHTTP.HandleFunc("/api/v1/install", corsMiddleware(s.handleInstall))
    muxHTTP.HandleFunc("/api/v1/clients", corsMiddleware(s.handleClients))
    muxHTTP.HandleFunc("/api/v1/wol", corsMiddleware(s.handleWOL))
    muxHTTP.HandleFunc("/api/v1/parse-driver", corsMiddleware(s.handleParseDriver))
    muxHTTP.HandleFunc("/api/v1/upload", corsMiddleware(s.handleUpload))
    muxHTTP.HandleFunc("/api/v1/clients/{id}/devices", corsMiddleware(s.handleClientDevices))
    muxHTTP.HandleFunc("/api/v1/clients/{id}/stats", corsMiddleware(s.handleClientStats))
    muxHTTP.HandleFunc("/api/v1/files", corsMiddleware(s.handleListFiles))          // GET список
    muxHTTP.HandleFunc("/api/v1/files/", corsMiddleware(s.handleFileOperations))

    s.httpServer = &http.Server{
        Addr:    fmt.Sprintf("%s:%d", s.config.Host, s.config.HTTPPort),
        Handler: muxHTTP,
    }

    // WebSocket
    muxWS := http.NewServeMux()
    muxWS.HandleFunc("/ws", s.handleWebSocket) // для WebSocket CORS не нужен (upgrader уже разрешает)
    s.wsServer = &http.Server{
        Addr:    fmt.Sprintf("%s:%d", s.config.Host, s.config.WSPort),
        Handler: muxWS,
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
    defer s.clientsMu.Unlock()
    s.clients[conn.client.ID] = conn
}

func (s *Server) unregisterClient(id uuid.UUID) {
    s.clientsMu.Lock()
    defer s.clientsMu.Unlock()
    delete(s.clients, id)
}

func (s *Server) sendCommand(clientID uuid.UUID, cmd *commandMsg) error {
    s.clientsMu.RLock()
    conn, ok := s.clients[clientID]
    s.clientsMu.RUnlock()
    if !ok {
        return fmt.Errorf("client %s not online", clientID)
    }
    select {
    case conn.send <- cmd:
        return nil
    default:
        return fmt.Errorf("send channel blocked for client %s", clientID)
    }
}