package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os/exec"
	"path/filepath"
	"strings"

	"log/slog"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/chocom1nt/LANdapter/internal/common"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Agent struct {
	config    *common.AgentConfig
	logger    *slog.Logger
	uuid      uuid.UUID
	hostname  string
	os        string
	mac       string
	conn      *websocket.Conn
	installer *Installer
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

func NewAgent(cfg *common.AgentConfig, logger *slog.Logger) (*Agent, error) {
	id, err := loadOrGenerateUUID(cfg.UUIDFile)
	if err != nil {
		return nil, err
	}
	hostname, _ := os.Hostname()
	return &Agent{
		config:    cfg,
		logger:    logger,
		uuid:      id,
		hostname:  hostname,
		os:        runtime.GOOS,
		mac:       getMAC(),
		installer: NewInstaller(logger, cfg.InstallerArgs),
	}, nil
}

func loadOrGenerateUUID(filePath string) (uuid.UUID, error) {
	data, err := os.ReadFile(filePath)
	if err == nil {
		id, err := uuid.Parse(string(data))
		if err == nil {
			return id, nil
		}
	}
	id := uuid.New()
	if err := os.WriteFile(filePath, []byte(id.String()), 0644); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (a *Agent) Run(ctx context.Context) error {
	backoff := time.Second
	const maxBackoff = time.Minute

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := a.connect(ctx); err != nil {
			a.logger.Error("Connection failed", "error", err)
		} else {
			a.logger.Info("Connected, waiting for commands")
			if err := a.readLoop(ctx); err != nil {
				a.logger.Warn("Read loop ended", "error", err)
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
		}
		a.logger.Info("Reconnecting...")
	}
}

func (a *Agent) connect(ctx context.Context) error {
	u := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("%s:%d", a.config.MasterHost, a.config.MasterPort),
		Path:   "/ws",
	}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	a.conn = conn

	msg := handshakeMsg{
		Type:     "handshake",
		UUID:     a.uuid.String(),
		Hostname: a.hostname,
		OS:       a.os,
		MAC:      a.mac,
	}
	if err := conn.WriteJSON(msg); err != nil {
		conn.Close()
		return err
	}
	return nil
}

func getMAC() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			mac := iface.HardwareAddr.String()
			if mac != "" {
				return mac
			}
		}
	}
	return ""
}

// readLoop – основной цикл получения команд
func (a *Agent) readLoop(ctx context.Context) error {
	defer a.conn.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var cmd commandMsg
		if err := a.conn.ReadJSON(&cmd); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				a.logger.Warn("Read error", "error", err)
			} else {
				a.logger.Warn("Read error (normal)", "error", err)
			}
			return err
		}

		switch cmd.Type {
		case "install":
			var installData struct {
				Files []string `json:"files"`
				Mode  string   `json:"mode"`
			}
			if err := json.Unmarshal(cmd.Payload, &installData); err != nil {
				a.logger.Error("Invalid install payload", "error", err)
				continue
			}
			fileIDs := installData.Files
			mode := installData.Mode
			a.logger.Info("Received install command", "job_id", cmd.JobID, "file_ids", fileIDs)

			// Собираем снапшот ДО
			snapshotBefore := a.collectStats()

			tmpDir, err := os.MkdirTemp("", "installer-")
			if err != nil {
				a.logger.Error("Failed to create temp dir", "error", err)
				res := resultMsg{
					Type:   "result",
					JobID:  cmd.JobID,
					Status: "failed",
					Error:  "temp dir creation failed: " + err.Error(),
				}
				a.conn.WriteJSON(res)
				continue
			}
			defer os.RemoveAll(tmpDir)

			var localFiles []string
			downloadFailed := false
			for _, fid := range fileIDs {
				httpPort := 8080
				url := fmt.Sprintf("http://%s:%d/api/v1/files/%s", a.config.MasterHost, httpPort, fid)
				localPath, err := a.downloadFile(url, tmpDir)
				if err != nil {
					a.logger.Error("Failed to download file", "file_id", fid, "error", err)
					downloadFailed = true
					break
				}
				localFiles = append(localFiles, localPath)
			}

			if downloadFailed {
				res := resultMsg{
					Type:   "result",
					JobID:  cmd.JobID,
					Status: "failed",
					Error:  "file download failed",
				}
				a.conn.WriteJSON(res)
				continue
			}

			output, err := a.installer.Install(localFiles, mode)
			status := "success"
			errMsg := ""
			if err != nil {
				status = "failed"
				errMsg = err.Error()
			}

			snapshotAfter := a.collectStats()

			res := resultMsg{
				Type:           "result",
				JobID:          cmd.JobID,
				Status:         status,
				Output:         output,
				Error:          errMsg,
				SnapshotBefore: snapshotBefore,
				SnapshotAfter:  snapshotAfter,
			}
			if err := a.conn.WriteJSON(res); err != nil {
				a.logger.Error("Failed to send result", "error", err)
			}

		case "stats":
			stats := a.collectStats()
			res := resultMsg{
				Type:  "stats",
				JobID: cmd.JobID,
				Data:  stats,
			}
			if err := a.conn.WriteJSON(res); err != nil {
				a.logger.Error("Failed to send stats", "error", err)
			}

		case "devices":
			devices := a.collectDevices()
			res := resultMsg{
				Type:  "devices",
				JobID: cmd.JobID,
				Data:  devices,
			}
			if err := a.conn.WriteJSON(res); err != nil {
				a.logger.Error("Failed to send devices", "error", err)
			}
		}
	}
}

// downloadFile – скачивает файл с мастера
func (a *Agent) downloadFile(url, destDir string) (string, error) {
    // Создаём клиент с таймаутом (исправление №2)
    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("download failed with status: %s", resp.Status)
    }

    filename := ""
    contentDisp := resp.Header.Get("Content-Disposition")
    if contentDisp != "" {
        // Парсим: attachment; filename="file.exe"
        parts := strings.Split(contentDisp, "filename=")
        if len(parts) > 1 {
            filename = strings.Trim(parts[1], "\"")
        }
    }
    if filename == "" {
        // fallback – последний сегмент URL
        parts := strings.Split(url, "/")
        filename = parts[len(parts)-1]
    }

    // Безопасное имя файла – только базовое имя, удаляем возможные пути
    safeFilename := filepath.Base(filename)
    fullPath := filepath.Join(destDir, safeFilename)

    // Дополнительная проверка: убедимся, что путь внутри destDir
    if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(destDir)+string(os.PathSeparator)) {
        return "", fmt.Errorf("invalid file path: %s", fullPath)
    }

    out, err := os.Create(fullPath)
    if err != nil {
        return "", err
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    if err != nil {
        return "", err
    }
    return fullPath, nil
}

// collectDevices – сбор информации об устройствах
func (a *Agent) collectDevices() interface{} {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("powershell", "-NoProfile", "-Command",
			`[Console]::OutputEncoding = [System.Text.Encoding]::UTF8;
			 Get-PnpDevice | Select-Object FriendlyName, Class, Status | ConvertTo-Json -Compress`)
		out, err := cmd.Output()
		if err != nil {
			a.logger.Error("Devices command failed", "error", err)
			return map[string]string{"error": err.Error()}
		}
		return string(out)
	} else {
		cmd := exec.Command("bash", "-c",
			`echo "=== USB ==="; lsusb; echo "=== PCI ==="; lspci; echo "=== CPU ==="; lscpu`)
		out, err := cmd.Output()
		if err != nil {
			a.logger.Error("Devices command failed", "error", err)
			return map[string]string{"error": err.Error()}
		}
		return string(out)
	}
}

// collectStats – сбор системной статистики
func (a *Agent) collectStats() interface{} {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("powershell", "-NoProfile", "-Command",
			`[Console]::OutputEncoding = [System.Text.Encoding]::UTF8;
			 $cpu = (Get-Counter '\Processor(_Total)\% Processor Time').CounterSamples.CookedValue;
			 $memAvail = (Get-Counter '\Memory\Available MBytes').CounterSamples.CookedValue;
			 $memTotal = (Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory / 1MB;
			 $uptime = (Get-Date) - (Get-CimInstance Win32_OperatingSystem).LastBootUpTime;
			 [PSCustomObject]@{
				 cpu_percent = [math]::Round($cpu, 2);
				 mem_available_mb = [math]::Round($memAvail, 0);
				 mem_total_mb = [math]::Round($memTotal, 0);
				 uptime_seconds = [math]::Round($uptime.TotalSeconds);
				 uptime_human = $uptime.ToString();
			 } | ConvertTo-Json -Compress`)
		out, err := cmd.Output()
		if err != nil {
			a.logger.Error("Stats command failed", "error", err)
			return map[string]string{"error": err.Error()}
		}
		return string(out)
	} else {
		cmd := exec.Command("bash", "-c",
			`cpu=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1);
			 mem=$(free -m | grep Mem | awk '{print $3, $2}');
			 uptime=$(cat /proc/uptime | awk '{print $1}');
			 echo "{\"cpu_percent\":$cpu,\"mem_used_mb\":$(echo $mem | cut -d' ' -f1),\"mem_total_mb\":$(echo $mem | cut -d' ' -f2),\"uptime_seconds\":$uptime}"`)
		out, err := cmd.Output()
		if err != nil {
			a.logger.Error("Stats command failed", "error", err)
			return map[string]string{"error": err.Error()}
		}
		return string(out)
	}
}