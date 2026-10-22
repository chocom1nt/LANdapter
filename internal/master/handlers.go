package master

import (
	"encoding/json"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chocom1nt/LANdapter/internal/common"
	"github.com/google/uuid"
)

// uploadDir – папка для хранения загруженных файлов
const uploadDir = "uploads"

var pendingRequests sync.Map

func (s *Server) sendCommandAndWait(clientID uuid.UUID, cmdType string, w http.ResponseWriter) {
	requestID := uuid.New().String()
	ch := make(chan interface{}, 1)
	key := clientID.String() + ":" + requestID
	pendingRequests.Store(key, ch)
	defer pendingRequests.Delete(key)

	cmd := &commandMsg{Type: cmdType, JobID: requestID}
	if err := s.sendCommand(clientID, cmd); err != nil {
		http.Error(w, "Client is not online", http.StatusConflict)
		return
	}

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case resp := <-ch:
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			s.logger.Warn("Failed to encode agent response", "error", err)
		}
	case <-timer.C:
		http.Error(w, "Timeout waiting for agent response", http.StatusRequestTimeout)
	}
}

func init() {
	os.MkdirAll(uploadDir, 0755)
}

type statsRequest struct {
	Type string `json:"type"` // "stats" или "devices"
}

type installRequest struct {
	FileIDs   []string `json:"file_ids"`   // идентификаторы файлов
	ClientIDs []string `json:"client_ids"` // клиенты
	Mode      string   `json:"mode"`
}

type installResponse struct {
	JobID string `json:"job_id"`
}

// handleInstall – создаёт задание и отправляет команды агентам
func (s *Server) handleInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req installRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req.FileIDs) == 0 || len(req.ClientIDs) == 0 {
		http.Error(w, "file_ids and client_ids required", http.StatusBadRequest)
		return
	}
	if len(req.FileIDs) > 100 || len(req.ClientIDs) > 500 {
		http.Error(w, "Too many files or clients in one job", http.StatusBadRequest)
		return
	}
	if req.Mode != "quiet" && req.Mode != "interactive" && req.Mode != "normal" && req.Mode != "" {
		http.Error(w, "Invalid install mode", http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = "quiet"
	}

	// Validate all IDs and files before creating a job so malformed requests do
	// not leave incomplete jobs in the database.
	for _, fileID := range req.FileIDs {
		if _, err := uuid.Parse(fileID); err != nil {
			http.Error(w, "Invalid file ID", http.StatusBadRequest)
			return
		}
		if _, err := findUploadedFile(fileID); err != nil {
			http.Error(w, "One or more files were not found", http.StatusBadRequest)
			return
		}
	}
	clientIDs := make([]uuid.UUID, 0, len(req.ClientIDs))
	clients := make([]*common.Client, 0, len(req.ClientIDs))
	seen := make(map[uuid.UUID]struct{}, len(req.ClientIDs))
	for _, rawID := range req.ClientIDs {
		id, err := uuid.Parse(rawID)
		if err != nil || id == uuid.Nil {
			http.Error(w, "Invalid client ID", http.StatusBadRequest)
			return
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		client, err := s.storage.GetClient(r.Context(), id)
		if err != nil {
			s.logger.Error("Client lookup failed", "client", id, "error", err)
			http.Error(w, "Unable to validate selected clients", http.StatusInternalServerError)
			return
		}
		if client == nil {
			http.Error(w, "Unknown client ID", http.StatusBadRequest)
			return
		}
		clientIDs = append(clientIDs, id)
		clients = append(clients, client)
	}
	if len(clientIDs) == 0 {
		http.Error(w, "At least one valid client is required", http.StatusBadRequest)
		return
	}

	jobID := uuid.New()
	job := &common.Job{ID: jobID, Files: common.JSONStringSlice(req.FileIDs), CreatedAt: time.Now()}
	if err := s.storage.CreateJob(r.Context(), job); err != nil {
		s.logger.Error("Create job failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	for i, client := range clients {
		result := &common.JobResult{ID: uuid.New(), JobID: jobID, ClientID: clientIDs[i], Status: "pending"}
		if err := s.storage.CreateJobResult(r.Context(), result); err != nil {
			s.logger.Error("Create job result failed", "client", clientIDs[i], "error", err)
			continue
		}
		if !client.Online {
			now := time.Now()
			message := "client was offline when job was dispatched"
			result.Status, result.Error, result.FinishedAt = "failed", &message, &now
			if err := s.storage.UpdateJobResult(r.Context(), result); err != nil {
				s.logger.Warn("Unable to update offline job result", "client", clientIDs[i], "error", err)
			}
			continue
		}
		payloadBytes, err := json.Marshal(map[string]interface{}{"files": req.FileIDs, "mode": req.Mode})
		if err != nil {
			s.logger.Error("Marshal install payload failed", "error", err)
			continue
		}
		now := time.Now()
		result.Status, result.StartedAt = "running", &now
		if err := s.storage.UpdateJobResult(r.Context(), result); err != nil {
			s.logger.Error("Unable to mark job as running", "client", clientIDs[i], "error", err)
			continue
		}
		cmd := &commandMsg{Type: "install", JobID: jobID.String(), Payload: payloadBytes}
		if err := s.sendCommand(clientIDs[i], cmd); err != nil {
			finished := time.Now()
			message := "failed to queue command: " + err.Error()
			result.Status, result.Error, result.FinishedAt = "failed", &message, &finished
			if updateErr := s.storage.UpdateJobResult(r.Context(), result); updateErr != nil {
				s.logger.Warn("Unable to update failed job result", "client", clientIDs[i], "error", updateErr)
			}
			continue
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(installResponse{JobID: jobID.String()}); err != nil {
		s.logger.Warn("Unable to encode install response", "error", err)
	}
}

// handleClients – возвращает список клиентов
func (s *Server) handleClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var online *bool
	if val := r.URL.Query().Get("online"); val != "" {
		b := val == "true"
		online = &b
	}

	clients, err := s.storage.ListClients(r.Context(), online)
	if err != nil {
		s.logger.Error("List clients failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}

// handleWOL – отправляет Wake-on-LAN пакеты выбранным клиентам
func (s *Server) handleWOL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ClientIDs []string `json:"client_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	var macs []string
	for _, idStr := range req.ClientIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		client, err := s.storage.GetClient(r.Context(), id)
		if err != nil || client == nil {
			continue
		}
		if client.MAC != "" {
			macs = append(macs, client.MAC)
		}
	}
	for _, mac := range macs {
		if err := sendWOL(mac); err != nil {
			s.logger.Error("WOL failed", "mac", mac, "error", err)
		}
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

// sendWOL – формирует и отправляет магический пакет
func sendWOL(mac string) error {
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return err
	}
	packet := make([]byte, 102)
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}
	for i := 0; i < 16; i++ {
		copy(packet[6+i*6:], hw)
	}
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.IPv4bcast, Port: 9})
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.Write(packet)
	return err
}

// handleParseDriver – заглушка для парсинга драйверов с сайта
func (s *Server) handleParseDriver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	// Заглушка
	drivers := []map[string]string{
		{"name": "Driver1.inf", "url": "http://example.com/driver1.inf"},
		{"name": "Driver2.exe", "url": "http://example.com/driver2.exe"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(drivers)
}

// handleUpload – загружает файл на сервер и возвращает его ID
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil { // bounded memory; 100 MiB total request cap above
		http.Error(w, "File too large or parse error", http.StatusBadRequest)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	requestedName := strings.TrimSpace(strings.ReplaceAll(header.Filename, "\\", "/"))
	originalName := filepath.Base(requestedName)
	if originalName == "" || originalName == "." || originalName == string(filepath.Separator) {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}
	fileID := uuid.New().String()
	ext := filepath.Ext(originalName)
	safeFilename := fileID + ext
	fullPath := filepath.Join(uploadDir, safeFilename)

	dst, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		s.logger.Error("Failed to create file", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		_ = dst.Close()
		_ = os.Remove(fullPath)
		s.logger.Error("Failed to save file", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(fullPath)
		http.Error(w, "Unable to finalize uploaded file", http.StatusInternalServerError)
		return
	}

	meta := map[string]interface{}{
		"id":          fileID,
		"name":        originalName,
		"type":        filepath.Ext(originalName),
		"description": "",
		"version":     "",
		"uploadedAt":  time.Now().Format(time.RFC3339),
		"size":        header.Size,
	}
	metaPath := fullPath + ".meta.json"
	metaData, marshalErr := json.MarshalIndent(meta, "", "  ")
	if marshalErr != nil {
		_ = os.Remove(fullPath)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(metaPath, metaData, 0600); err != nil {
		s.logger.Warn("Failed to write metadata", "error", err)
	}

	resp := map[string]string{
		"file_id": fileID,
		"name":    originalName,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
	s.logger.Info("File uploaded", "file_id", fileID, "name", originalName, "path", fullPath)
}

// handleFileDownload – отдаёт файл по его ID
func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimPrefix(r.URL.Path, "/api/v1/files/")
	s.serveFileDownload(w, r, fileID)
}

func (s *Server) handleAgentFileDownload(w http.ResponseWriter, r *http.Request) {
	fileID := strings.TrimPrefix(r.URL.Path, "/api/v1/agent/files/")
	s.serveFileDownload(w, r, fileID)
}

func (s *Server) serveFileDownload(w http.ResponseWriter, r *http.Request, fileID string) {
	if strings.Contains(fileID, "/") || !isSafeFileID(fileID) {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}
	filePath, err := findUploadedFile(fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	filename := filepath.Base(filePath)
	if meta, err := readFileMetadata(filePath); err == nil {
		if original, ok := meta["name"].(string); ok && strings.TrimSpace(original) != "" {
			filename = filepath.Base(original)
		}
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, filePath)
}

func (s *Server) handleClientDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	// parts: ["api", "v1", "clients", "UUID", "devices"] → берём parts[3]
	clientID, err := uuid.Parse(parts[3])
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}
	s.sendCommandAndWait(clientID, "devices", w)
}

func (s *Server) handleClientStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	clientID, err := uuid.Parse(parts[3])
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}
	s.sendCommandAndWait(clientID, "stats", w)
}

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		s.logger.Error("Failed to read uploads dir", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	files := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".meta.json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		fileID := strings.SplitN(entry.Name(), ".", 2)[0]
		if !isSafeFileID(fileID) {
			continue
		}
		meta := map[string]interface{}{
			"id":          fileID,
			"name":        entry.Name(),
			"type":        filepath.Ext(entry.Name()),
			"size":        info.Size(),
			"uploadedAt":  info.ModTime().Format(time.RFC3339),
			"description": "",
			"version":     "",
		}
		// Если есть мета-файл, читаем его и дополняем
		metaPath := filepath.Join(uploadDir, entry.Name()+".meta.json")
		if metaData, err := os.ReadFile(metaPath); err == nil {
			var extra map[string]interface{}
			if json.Unmarshal(metaData, &extra) == nil {
				for k, v := range extra {
					meta[k] = v
				}
			}
		}
		files = append(files, meta)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// DELETE /api/v1/files/{id} – удаляет файл с диска
func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Missing file ID", http.StatusBadRequest)
		return
	}
	fileID := parts[len(parts)-1]
	if !isSafeFileID(fileID) {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}
	filePath, err := findUploadedFile(fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if err := os.Remove(filePath); err != nil {
		s.logger.Error("Failed to delete file", "file", filePath, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	// Удаляем мета-файл, логируем ошибку
	metaPath := filePath + ".meta.json"
	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		s.logger.Warn("Failed to delete metadata file", "file", metaPath, "error", err)
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *Server) handleFileOperations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleFileDownload(w, r)
	case http.MethodDelete:
		s.handleDeleteFile(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func isSafeFileID(id string) bool {
	parsed, err := uuid.Parse(id)
	return err == nil && parsed != uuid.Nil && parsed.String() == strings.ToLower(id)
}

func findUploadedFile(fileID string) (string, error) {
	if !isSafeFileID(fileID) {
		return "", os.ErrNotExist
	}
	files, err := filepath.Glob(filepath.Join(uploadDir, fileID+".*"))
	if err != nil {
		return "", err
	}
	for _, candidate := range files {
		if strings.HasSuffix(candidate, ".meta.json") {
			continue
		}
		info, statErr := os.Stat(candidate)
		if statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		return candidate, nil
	}
	return "", os.ErrNotExist
}

func readFileMetadata(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath + ".meta.json")
	if err != nil {
		return nil, err
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return meta, nil
}
