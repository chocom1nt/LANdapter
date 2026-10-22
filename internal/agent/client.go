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
	"mime"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
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

func NewAgent(cfg *common.AgentConfig, logger *slog.Logger) (*Agent, error) {
	if strings.TrimSpace(cfg.Token) == "" || len(strings.TrimSpace(cfg.Token)) < 32 {
		return nil, fmt.Errorf("agent token is required; issue one from the Master settings page")
	}
	if cfg.MasterScheme != "" && cfg.MasterScheme != "ws" && cfg.MasterScheme != "wss" {
		return nil, fmt.Errorf("master_scheme must be ws or wss")
	}
	id, err := loadOrGenerateUUID(cfg.UUIDFile, cfg.UUID)
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

func loadOrGenerateUUID(filePath, configuredUUID string) (uuid.UUID, error) {
	if configuredUUID = strings.TrimSpace(configuredUUID); configuredUUID != "" {
		id, err := uuid.Parse(configuredUUID)
		if err != nil || id == uuid.Nil {
			return uuid.Nil, fmt.Errorf("configured agent UUID is invalid")
		}
		return id, nil
	}
	if strings.TrimSpace(filePath) == "" {
		return uuid.Nil, fmt.Errorf("uuid_file or uuid must be configured")
	}
	data, err := os.ReadFile(filePath)
	if err == nil {
		id, parseErr := uuid.Parse(strings.TrimSpace(string(data)))
		if parseErr != nil || id == uuid.Nil {
			return uuid.Nil, fmt.Errorf("existing UUID file %q is invalid", filePath)
		}
		return id, nil
	}
	if !os.IsNotExist(err) {
		return uuid.Nil, fmt.Errorf("read UUID file: %w", err)
	}
	id := uuid.New()
	if dir := filepath.Dir(filePath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return uuid.Nil, err
		}
	}
	if err := os.WriteFile(filePath, []byte(id.String()), 0600); err != nil {
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
	scheme := a.config.MasterScheme
	if scheme == "" {
		scheme = "ws"
	}
	if a.config.MasterPort < 1 || a.config.MasterPort > 65535 || strings.TrimSpace(a.config.MasterHost) == "" {
		return fmt.Errorf("master_host and a valid master_port are required")
	}
	u := url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(strings.Trim(a.config.MasterHost, "[]"), strconv.Itoa(a.config.MasterPort)),
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
		Token:    a.config.Token,
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

			var localFiles []string
			downloadFailed := false
			for _, fid := range fileIDs {
				httpPort := a.config.MasterHTTPPort
				if httpPort == 0 {
					httpPort = 8080
				}
				httpScheme := "http"
				if a.config.MasterScheme == "wss" {
					httpScheme = "https"
				}
				host := net.JoinHostPort(strings.Trim(a.config.MasterHost, "[]"), strconv.Itoa(httpPort))
				targetURL := fmt.Sprintf("%s://%s/api/v1/agent/files/%s", httpScheme, host, fid)
				localPath, err := a.downloadFile(targetURL, tmpDir)
				if err != nil {
					a.logger.Error("Failed to download file", "file_id", fid, "error", err)
					downloadFailed = true
					break
				}
				localFiles = append(localFiles, localPath)
			}

			if downloadFailed {
				_ = os.RemoveAll(tmpDir)
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
			_ = os.RemoveAll(tmpDir)

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
func (a *Agent) downloadFile(targetURL, destDir string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+a.config.Token)
	req.Header.Set("X-Agent-ID", a.uuid.String())
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %s", resp.Status)
	}

	filename := ""
	if contentDisp := resp.Header.Get("Content-Disposition"); contentDisp != "" {
		_, params, parseErr := mime.ParseMediaType(contentDisp)
		if parseErr == nil {
			filename = params["filename"]
		}
	}
	if filename == "" {
		parsedURL, parseErr := url.Parse(targetURL)
		if parseErr != nil {
			return "", parseErr
		}
		filename = filepath.Base(parsedURL.Path)
	}
	// Strip both Unix and Windows path components before touching the filesystem.
	filename = filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		return "", fmt.Errorf("invalid downloaded filename")
	}
	fullPath := filepath.Join(destDir, filename)
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(destDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid download path")
	}
	out, err := os.OpenFile(fullPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	const maxDownloadSize = 100 << 20
	written, copyErr := io.Copy(out, io.LimitReader(resp.Body, maxDownloadSize+1))
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil || written > maxDownloadSize {
		_ = os.Remove(fullPath)
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		return "", fmt.Errorf("download exceeds 100 MiB limit")
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
