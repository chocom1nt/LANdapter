package master

import (
	"encoding/json"
	"io"
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
    // Создаём канал для ответа
    ch := make(chan interface{}, 1)
    key := clientID.String() + ":" + cmdType
    pendingRequests.Store(key, ch)
    defer pendingRequests.Delete(key)

    // Формируем команду
    cmd := &commandMsg{
        Type:  cmdType,
        JobID: uuid.New().String(),
        // Payload можно не передавать
    }
    if err := s.sendCommand(clientID, cmd); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Ожидаем ответ или таймаут
    select {
    case resp := <-ch:
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    case <-time.After(5 * time.Second):
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

	var req installRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.FileIDs) == 0 || len(req.ClientIDs) == 0 {
		http.Error(w, "file_ids and client_ids required", http.StatusBadRequest)
		return
	}

	jobID := uuid.New()
	job := &common.Job{
		ID:        jobID,
		Files:     common.JSONStringSlice(req.FileIDs),
		CreatedAt: time.Now(),
	}
	if err := s.storage.CreateJob(r.Context(), job); err != nil {
		s.logger.Error("Create job failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	for _, idStr := range req.ClientIDs {
		clientID, err := uuid.Parse(idStr)
		if err != nil {
			s.logger.Warn("Invalid client ID", "id", idStr)
			continue
		}
		client, err := s.storage.GetClient(r.Context(), clientID)
		if err != nil || client == nil {
			s.logger.Warn("Client not found", "client", clientID)
			continue
		}
		result := &common.JobResult{
			ID:       uuid.New(),
			JobID:    jobID,
			ClientID: clientID,
			Status:   "pending",
		}
		if err := s.storage.CreateJobResult(r.Context(), result); err != nil {
			s.logger.Error("Create job result failed", "error", err)
			continue
		}
		if client.Online {
			payloadData := map[string]interface{}{
				"files": req.FileIDs,
				"mode":  req.Mode,
			}
			payloadBytes, _ := json.Marshal(payloadData)
			cmd := &commandMsg{
				Type:    "install",
				JobID:   jobID.String(),
				Payload: payloadBytes,
			}
			if err := s.sendCommand(clientID, cmd); err == nil {
				result.Status = "running"
				now := time.Now()
				result.StartedAt = &now
				s.storage.UpdateJobResult(r.Context(), result)
			} else {
				s.logger.Error("Send command failed", "client", clientID, "error", err)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(installResponse{JobID: jobID.String()})
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
	if err := r.ParseMultipartForm(100 << 20); err != nil { // 100 MB
		http.Error(w, "File too large or parse error", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileID := uuid.New().String()
	ext := filepath.Ext(header.Filename)
	safeFilename := fileID + ext
	fullPath := filepath.Join(uploadDir, safeFilename)

	dst, err := os.Create(fullPath)
	if err != nil {
		s.logger.Error("Failed to create file", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		s.logger.Error("Failed to save file", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	 meta := map[string]interface{}{
        "name":        header.Filename,
        "type":        filepath.Ext(header.Filename),
        "description": "",
        "version":     "",
        "uploadedAt":  time.Now().Format(time.RFC3339),
        "size":        header.Size,
    }
    metaPath := fullPath + ".meta.json"
    metaData, _ := json.MarshalIndent(meta, "", "  ")
    if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
        s.logger.Warn("Failed to write metadata", "error", err)
    }

	resp := map[string]string{
		"file_id": fileID,
		"name":    header.Filename,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
    s.logger.Info("File uploaded", "file_id", fileID, "name", header.Filename, "path", fullPath)
}

// handleFileDownload – отдаёт файл по его ID
func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request) {
    parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
    if len(parts) < 3 {
        http.Error(w, "Missing file ID", http.StatusBadRequest)
        return
    }
    fileID := parts[len(parts)-1]
    // Безопасная очистка ID – разрешаем только буквы, цифры, дефис и точку
    if !isSafeFileID(fileID) {
        http.Error(w, "Invalid file ID", http.StatusBadRequest)
        return
    }
    files, err := filepath.Glob(filepath.Join(uploadDir, fileID+".*"))
    if err != nil || len(files) == 0 {
        http.Error(w, "File not found", http.StatusNotFound)
        return
    }
    // Убедимся, что путь внутри uploadDir
    filePath := files[0]
    if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean(uploadDir)+string(os.PathSeparator)) {
        http.Error(w, "Access denied", http.StatusForbidden)
        return
    }
    w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
    w.Header().Set("Content-Type", "application/octet-stream")
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

    var files []map[string]interface{}
    for _, entry := range entries {
        if entry.IsDir() || strings.HasSuffix(entry.Name(), ".meta.json") {
            continue
        }
        info, err := entry.Info()
        if err != nil {
            continue
        }
        fileID := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
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
    // Защита от path traversal
    if !isSafeFileID(fileID) {
        http.Error(w, "Invalid file ID", http.StatusBadRequest)
        return
    }
    files, err := filepath.Glob(filepath.Join(uploadDir, fileID+".*"))
    if err != nil || len(files) == 0 {
        http.Error(w, "File not found", http.StatusNotFound)
        return
    }
    // Проверка, что путь внутри uploadDir
    filePath := files[0]
    if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean(uploadDir)+string(os.PathSeparator)) {
        http.Error(w, "Access denied", http.StatusForbidden)
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
    // Разрешаем только буквы, цифры, дефис, точку и подчёркивание
    for _, ch := range id {
        if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
            (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.') {
            return false
        }
    }
    return true
}