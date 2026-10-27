// Command lndp is the interactive LANdapter terminal console.
// It uses the authenticated Master API to manage clients, inspect devices,
// view live resource metrics and browse/upload/install packages.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/chocom1nt/LANdapter/internal/common"
	"github.com/mattn/go-runewidth"
)

const (
	defaultAPIBase  = "http://127.0.0.1:8080/api/v1"
	maxResponseSize = 4 << 20
	statsRefresh    = 5 * time.Second
)

type clientRecord struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	MAC      string `json:"mac"`
	Online   bool   `json:"online"`
	LastSeen string `json:"last_seen"`
}

type deviceRecord struct {
	Name          string `json:"name"`
	Class         string `json:"class"`
	Status        string `json:"status"`
	InstanceID    string `json:"instance_id"`
	HardwareID    string `json:"hardware_id"`
	Manufacturer  string `json:"manufacturer"`
	DriverName    string `json:"driver_name"`
	DriverVersion string `json:"driver_version"`
	ProblemCode   int    `json:"problem_code"`
	DriverIssue   bool   `json:"driver_issue"`
	Issue         string `json:"issue"`
	Bus           string `json:"bus"`
}

type packageRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Size        int64  `json:"size"`
	UploadedAt  string `json:"uploadedAt"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type detailLoadedMsg struct {
	clientID string
	kind     string
	devices  []deviceRecord
	stats    map[string]interface{}
	err      error
}

type clientsLoadedMsg struct {
	clients []clientRecord
	err     error
}

type statsRefreshMsg struct{}

type model struct {
	clients         []clientRecord
	cursor          int
	filter          string
	searching       bool
	loading         bool
	err             error
	details         bool
	showHelp        bool
	page            string
	devices         []deviceRecord
	deviceCursor    int
	deviceOffset    int
	stats           map[string]interface{}
	detailLoading   bool
	detailErr       error
	detailClientID  string
	selectedClients map[string]bool
	width           int
	height          int

	localDir       string
	localEntries   []localEntry
	localCursor    int
	remoteFiles    []packageRecord
	remoteCursor   int
	filePane       string
	selectedLocal  map[string]bool
	selectedRemote map[string]bool
	filesLoading   bool
	filesErr       error
	uploading      bool
	installing     bool
	installMode    string
	confirmAction  string
	notice         string
	noticeError    bool
}

func initialModel() model {
	wd, _ := os.Getwd()
	return model{
		loading: true, page: "clients", width: 110, height: 32,
		localDir: wd, filePane: "local", installMode: "quiet",
		selectedClients: make(map[string]bool), selectedLocal: make(map[string]bool), selectedRemote: make(map[string]bool),
	}
}

func main() {
	if err := common.LoadProjectDotEnv(); err != nil {
		fmt.Fprintln(os.Stderr, "lndp: cannot load environment:", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lndp:", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd { return loadClientsCmd }

func loadClientsCmd() tea.Msg {
	clients, err := fetchClients()
	return clientsLoadedMsg{clients: clients, err: err}
}

func fetchClients() ([]clientRecord, error) {
	body, err := apiRequest(http.MethodGet, "/clients", nil, "", 20*time.Second)
	if err != nil {
		return nil, fmt.Errorf("load clients: %w", err)
	}
	var clients []clientRecord
	if err := json.Unmarshal(body, &clients); err != nil {
		return nil, fmt.Errorf("decode clients response: %w", err)
	}
	if clients == nil {
		clients = []clientRecord{}
	}
	return clients, nil
}

func fetchClientDetails(clientID, kind string) detailLoadedMsg {
	result := detailLoadedMsg{clientID: clientID, kind: kind}
	if kind != "devices" && kind != "stats" {
		result.err = fmt.Errorf("unsupported detail type %q", kind)
		return result
	}
	body, err := apiRequest(http.MethodGet, "/clients/"+url.PathEscape(clientID)+"/"+kind, nil, "", 40*time.Second)
	if err != nil {
		result.err = err
		return result
	}
	var raw json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		result.err = fmt.Errorf("decode %s response: %w", kind, err)
		return result
	}
	// Older agents returned JSON serialized as a string; newer ones return a
	// structured value. Support both wire formats during upgrades.
	var stringPayload string
	if len(raw) > 0 && raw[0] == '"' && json.Unmarshal(raw, &stringPayload) == nil {
		raw = json.RawMessage(stringPayload)
	}
	if kind == "devices" {
		if err := json.Unmarshal(raw, &result.devices); err != nil {
			var envelope struct {
				Devices []deviceRecord `json:"devices"`
				Warning string         `json:"warning"`
			}
			if envelopeErr := json.Unmarshal(raw, &envelope); envelopeErr == nil && envelope.Devices != nil {
				result.devices = envelope.Devices
				if envelope.Warning != "" {
					result.err = fmt.Errorf("partial inventory: %s", envelope.Warning)
				}
			} else {
				result.err = fmt.Errorf("decode device inventory: %w", err)
			}
		}
		if result.devices == nil {
			result.devices = []deviceRecord{}
		}
	} else if err := json.Unmarshal(raw, &result.stats); err != nil {
		result.err = fmt.Errorf("decode statistics: %w", err)
	}
	return result
}

func loadClientDetailsCmd(clientID, kind string) tea.Cmd {
	return func() tea.Msg { return fetchClientDetails(clientID, kind) }
}

func statsRefreshCmd() tea.Cmd {
	return tea.Tick(statsRefresh, func(time.Time) tea.Msg { return statsRefreshMsg{} })
}

func readAPIConfig() (string, string, error) {
	token := strings.TrimSpace(os.Getenv("LANDAPTER_ADMIN_TOKEN"))
	if len(token) < 32 || strings.IndexFunc(token, func(r rune) bool { return r <= 0x20 || r == 0x7f }) >= 0 {
		return "", "", errors.New("LANDAPTER_ADMIN_TOKEN must be set to a secret of at least 32 characters without whitespace")
	}
	base := strings.TrimSpace(os.Getenv("LANDAPTER_API_URL"))
	if base == "" {
		base = defaultAPIBase
	}
	allowInsecureHTTP := os.Getenv("LANDAPTER_ALLOW_INSECURE_HTTP") == "1"
	base, err := normalizeAPIBase(base, allowInsecureHTTP)
	if err != nil {
		return "", "", fmt.Errorf("invalid LANDAPTER_API_URL: %w", err)
	}
	return base, token, nil
}

func normalizeAPIBase(raw string, allowInsecureHTTP bool) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("must be an absolute HTTP(S) URL ending in /api/v1")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("only HTTP and HTTPS URLs are supported")
	}
	if parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("URL must not contain credentials, a query, or a fragment")
	}
	for _, part := range strings.Split(parsed.EscapedPath(), "/") {
		decoded, decodeErr := url.PathUnescape(part)
		if decodeErr != nil || decoded == "." || decoded == ".." {
			return "", errors.New("URL path must not contain dot segments")
		}
	}
	path := strings.TrimRight(parsed.Path, "/")
	if path != "/api/v1" && !strings.HasSuffix(path, "/api/v1") {
		return "", errors.New("URL path must end in /api/v1")
	}
	if parsed.Scheme == "http" && !isLoopbackHost(parsed.Hostname()) && !allowInsecureHTTP {
		return "", errors.New("use HTTPS for a remote Master; set LANDAPTER_ALLOW_INSECURE_HTTP=1 only on a trusted isolated network")
	}
	parsed.Path, parsed.RawPath = path, ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func apiRequest(method, path string, body []byte, contentType string, timeout time.Duration) ([]byte, error) {
	base, token, err := readAPIConfig()
	if err != nil {
		return nil, err
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, base+path, reader)
	if err != nil {
		return nil, fmt.Errorf("build Master API request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	client := &http.Client{Timeout: timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Master API request failed: %w", err)
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("read Master API response: %w", err)
	}
	if len(response) > maxResponseSize {
		return nil, fmt.Errorf("Master API response exceeded %d bytes", maxResponseSize)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := strings.TrimSpace(string(response))
		switch resp.StatusCode {
		case http.StatusRequestTimeout:
			return nil, fmt.Errorf("agent did not respond before timeout; update/restart the agent and retry: %s", detail)
		case http.StatusConflict:
			return nil, fmt.Errorf("client has no active WebSocket connection; check that the agent is running and reconnect it: %s", detail)
		default:
			if detail != "" {
				return nil, fmt.Errorf("Master API returned HTTP %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), detail)
			}
			return nil, fmt.Errorf("Master API returned HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		}
	}
	return response, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func (m model) filteredClients() []clientRecord {
	if m.filter == "" {
		return m.clients
	}
	needle := strings.ToLower(strings.TrimSpace(m.filter))
	filtered := make([]clientRecord, 0, len(m.clients))
	for _, c := range m.clients {
		if strings.Contains(strings.ToLower(c.Hostname), needle) || strings.Contains(strings.ToLower(c.OS), needle) || strings.Contains(strings.ToLower(c.MAC), needle) || strings.Contains(strings.ToLower(c.ID), needle) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (m model) selectedClient() (clientRecord, bool) {
	clients := m.filteredClients()
	if len(clients) == 0 || m.cursor < 0 || m.cursor >= len(clients) {
		return clientRecord{}, false
	}
	return clients[m.cursor], true
}

func (m model) selectedOnlineClientIDs() []string {
	ids := make([]string, 0)
	explicitSelection := countSelectedClients(m.selectedClients) > 0
	for _, c := range m.clients {
		if m.selectedClients[c.ID] && c.Online {
			ids = append(ids, c.ID)
		}
	}
	if !explicitSelection {
		if c, ok := m.selectedClient(); ok && c.Online {
			ids = append(ids, c.ID)
		}
	}
	return ids
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case clientsLoadedMsg:
		m.loading, m.err = false, msg.err
		if msg.err == nil {
			m.clients = msg.clients
			for id := range m.selectedClients {
				found := false
				for _, c := range m.clients {
					if c.ID == id {
						found = true
						break
					}
				}
				if !found {
					delete(m.selectedClients, id)
				}
			}
		}
		m.cursor = clamp(m.cursor, 0, len(m.filteredClients())-1)
		if m.page == "devices" || m.page == "stats" {
			if client, ok := m.selectedClient(); ok {
				m.detailLoading, m.detailClientID = true, client.ID
				return m, loadClientDetailsCmd(client.ID, m.page)
			}
			m.detailLoading = false
		}
	case detailLoadedMsg:
		if msg.clientID != m.detailClientID || msg.kind != m.page {
			return m, nil
		}
		m.detailLoading, m.detailErr = false, msg.err
		if msg.kind == "devices" {
			m.devices = msg.devices
			m.deviceCursor = clamp(m.deviceCursor, 0, len(m.devices)-1)
			rows := max(1, m.height-17)
			if m.deviceCursor < m.deviceOffset {
				m.deviceOffset = m.deviceCursor
			} else if m.deviceCursor >= m.deviceOffset+rows {
				m.deviceOffset = m.deviceCursor - rows + 1
			}
			m.deviceOffset = clamp(m.deviceOffset, 0, max(0, len(m.devices)-rows))
		} else if msg.kind == "stats" {
			m.stats = msg.stats
		}
		if m.page == "stats" {
			return m, statsRefreshCmd()
		}
	case statsRefreshMsg:
		if m.page != "stats" {
			return m, nil
		}
		if m.detailLoading {
			return m, nil
		}
		if client, ok := m.selectedClient(); ok && client.Online {
			m.detailLoading, m.detailClientID = true, client.ID
			return m, loadClientDetailsCmd(client.ID, "stats")
		}
		m.detailErr = errors.New("selected client is offline; live metrics are unavailable")
		return m, statsRefreshCmd()
	case localEntriesLoadedMsg:
		if filepathClean(msg.dir) != filepathClean(m.localDir) {
			return m, nil
		}
		m.filesLoading = false
		m.filesErr = msg.err
		if msg.err == nil {
			m.localEntries = msg.entries
			m.localCursor = clamp(m.localCursor, 0, len(m.localEntries)-1)
		}
	case remoteFilesLoadedMsg:
		m.filesLoading = false
		m.filesErr = msg.err
		if msg.err == nil {
			m.remoteFiles = msg.files
			for id := range m.selectedRemote {
				found := false
				for _, f := range msg.files {
					if f.ID == id {
						found = true
						break
					}
				}
				if !found {
					delete(m.selectedRemote, id)
				}
			}
			m.remoteCursor = clamp(m.remoteCursor, 0, len(m.remoteFiles)-1)
		}
	case uploadFinishedMsg:
		m.uploading = false
		for _, path := range msg.uploadedPaths {
			delete(m.selectedLocal, path)
		}
		for _, f := range msg.uploaded {
			m.selectedRemote[f.ID] = true
		}
		if msg.err != nil {
			m.setNotice("Upload finished with an error: "+msg.err.Error(), true)
		} else {
			m.setNotice(fmt.Sprintf("Uploaded %d package(s) to Master", len(msg.uploaded)), false)
		}
		return m, loadRemoteFilesCmd
	case installFinishedMsg:
		m.installing = false
		if msg.err != nil {
			m.setNotice("Install request failed: "+msg.err.Error(), true)
		} else {
			m.setNotice("Install job created: "+msg.jobID, false)
			m.page = "clients"
		}
	case deleteFinishedMsg:
		if msg.err != nil {
			m.setNotice("Delete failed: "+msg.err.Error(), true)
		} else {
			m.setNotice(fmt.Sprintf("Deleted %d package(s)", msg.deleted), false)
		}
		m.confirmAction = ""
		return m, loadRemoteFilesCmd
	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if m.searching {
			cmd := m.handleSearchKey(key)
			return m, cmd
		}
		if m.confirmAction != "" {
			switch strings.ToLower(key) {
			case "y":
				if m.confirmAction == "delete" {
					ids := m.targetRemoteFileIDs()
					if len(ids) == 0 {
						m.confirmAction = ""
						m.setNotice("No packages selected", true)
						return m, nil
					}
					m.notice = "Deleting packages…"
					return m, deletePackagesCmd(ids)
				}
			case "n", "esc":
				m.confirmAction = ""
				m.setNotice("Action cancelled", false)
			}
			return m, nil
		}
		if m.showHelp {
			switch strings.ToLower(key) {
			case "esc", "?", "f1", "enter", "q":
				m.showHelp = false
			}
			return m, nil
		}
		if key == "q" || key == "ctrl+c" {
			return m, tea.Quit
		}
		if key == "tab" {
			cmd := m.cyclePage()
			return m, cmd
		}
		switch key {
		case "1", "c":
			cmd := m.switchPage("clients")
			return m, cmd
		case "2", "d":
			cmd := m.switchPage("devices")
			return m, cmd
		case "3", "s":
			cmd := m.switchPage("stats")
			return m, cmd
		case "4", "f":
			cmd := m.switchPage("files")
			return m, cmd
		case "r":
			cmd := m.refreshCurrentPage()
			return m, cmd
		case "?", "f1":
			m.showHelp = true
			return m, nil
		case "/":
			if m.page != "files" {
				m.page = "clients"
				m.searching = true
				m.filter = ""
				m.cursor = 0
				return m, nil
			}
		}
		if m.page == "files" {
			cmd := m.handleFilesKey(key)
			return m, cmd
		}
		if m.page == "devices" {
			rows := max(1, m.height-17)
			switch key {
			case "up", "k":
				m.deviceCursor = max(0, m.deviceCursor-1)
			case "down", "j":
				m.deviceCursor = min(max(0, len(m.devices)-1), m.deviceCursor+1)
			case "pgup":
				m.deviceCursor = max(0, m.deviceCursor-rows)
			case "pgdown":
				m.deviceCursor = min(max(0, len(m.devices)-1), m.deviceCursor+rows)
			default:
				rows = 0
			}
			if rows > 0 {
				if m.deviceCursor < m.deviceOffset {
					m.deviceOffset = m.deviceCursor
				} else if m.deviceCursor >= m.deviceOffset+rows {
					m.deviceOffset = m.deviceCursor - rows + 1
				}
				m.deviceOffset = clamp(m.deviceOffset, 0, max(0, len(m.devices)-rows))
				return m, nil
			}
		}
		oldCursor := m.cursor
		switch key {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filteredClients())-1 {
				m.cursor++
			}
		case "left", "h":
			if m.page == "clients" {
				m.cursor = max(0, m.cursor-1)
			}
		case "right", "l":
			if m.page == "clients" {
				m.cursor = min(len(m.filteredClients())-1, m.cursor+1)
			}
		case " ", "space":
			if c, ok := m.selectedClient(); ok {
				toggleSelection(m.selectedClients, c.ID)
			}
		case "enter":
			if m.page == "clients" {
				m.details = !m.details
			}
		case "esc":
			m.filter = ""
			m.cursor = 0
			m.details = false
		}
		if oldCursor != m.cursor && (m.page == "devices" || m.page == "stats") {
			m.devices, m.stats = nil, nil
			m.deviceOffset = 0
			if c, ok := m.selectedClient(); ok {
				m.detailLoading, m.detailErr, m.detailClientID = true, nil, c.ID
				return m, loadClientDetailsCmd(c.ID, m.page)
			}
		}
	}
	return m, nil
}

func (m *model) setNotice(message string, isError bool) { m.notice, m.noticeError = message, isError }

func (m *model) switchPage(page string) tea.Cmd {
	if page != "clients" && page != "devices" && page != "stats" && page != "files" {
		return nil
	}
	m.page, m.detailErr, m.details = page, nil, false
	if page == "devices" || page == "stats" {
		m.devices, m.stats = nil, nil
		m.deviceCursor, m.deviceOffset = 0, 0
		if c, ok := m.selectedClient(); ok {
			if !c.Online {
				m.detailLoading = false
				m.detailErr = errors.New("selected client is offline")
				if page == "stats" {
					return statsRefreshCmd()
				}
				return nil
			}
			m.detailLoading, m.detailClientID = true, c.ID
			cmd := loadClientDetailsCmd(c.ID, page)
			if page == "stats" {
				return tea.Batch(cmd, statsRefreshCmd())
			}
			return cmd
		}
		m.detailLoading = false
	}
	if page == "files" {
		m.filesLoading = true
		return tea.Batch(loadLocalDirectoryCmd(m.localDir), loadRemoteFilesCmd)
	}
	return nil
}

func (m *model) cyclePage() tea.Cmd {
	switch m.page {
	case "clients":
		return m.switchPage("devices")
	case "devices":
		return m.switchPage("stats")
	case "stats":
		return m.switchPage("files")
	default:
		return m.switchPage("clients")
	}
}

func (m *model) refreshCurrentPage() tea.Cmd {
	m.notice = ""
	if m.page == "clients" {
		m.loading, m.err = true, nil
		return loadClientsCmd
	}
	if m.page == "devices" || m.page == "stats" {
		if c, ok := m.selectedClient(); ok {
			if !c.Online {
				m.detailErr = errors.New("selected client is offline")
				return nil
			}
			m.detailLoading, m.detailErr, m.detailClientID = true, nil, c.ID
			return loadClientDetailsCmd(c.ID, m.page)
		}
		return nil
	}
	m.filesLoading = true
	return tea.Batch(loadLocalDirectoryCmd(m.localDir), loadRemoteFilesCmd)
}

func (m *model) handleSearchKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.searching = false
		m.filter = ""
		m.cursor = 0
	case "enter":
		m.searching = false
	case "backspace":
		runes := []rune(m.filter)
		if len(runes) > 0 {
			m.filter = string(runes[:len(runes)-1])
			m.cursor = 0
		}
	default:
		if utf8.RuneCountInString(key) == 1 && key != "/" {
			m.filter += key
			m.cursor = 0
		}
	}
	return nil
}

func (m model) View() tea.View {
	var b strings.Builder
	width := max(48, min(m.width, 140))
	b.WriteString(style(" LANdapter ", "1;37;44") + style(" lndp ", "1;36") + style(" · endpoint / Master console ", "90") + "\n")
	b.WriteString(style(strings.Repeat("━", width), "36") + "\n")
	online := 0
	for _, c := range m.clients {
		if c.Online {
			online++
		}
	}
	fmt.Fprintf(&b, " %s %s   %s %s   %s %s\n", style("CLIENTS", "90"), style(strconv.Itoa(len(m.clients)), "1;37"), style("ONLINE", "90"), style(strconv.Itoa(online), "1;32"), style("OFFLINE", "90"), style(strconv.Itoa(len(m.clients)-online), "1;31"))
	b.WriteString(renderTabs(m.page, width) + "\n")
	b.WriteString(style(strings.Repeat("─", width), "90") + "\n")
	if m.notice != "" {
		code := "32"
		if m.noticeError {
			code = "31"
		}
		b.WriteString(style("● "+m.notice, code) + "\n")
	}
	if m.err != nil {
		b.WriteString(style("Could not load clients: "+m.err.Error(), "1;31") + "\nCheck LANDAPTER_API_URL, LANDAPTER_ADMIN_TOKEN and Master logs.\n")
	}
	if m.loading {
		b.WriteString(style("◌ Loading clients…", "33") + "\n")
	}
	if m.showHelp {
		m.renderHelp(&b, width)
	} else {
		switch m.page {
		case "clients":
			m.renderClients(&b, width)
		case "devices":
			m.renderDevices(&b, width)
		case "stats":
			m.renderStats(&b, width)
		case "files":
			m.renderFileManager(&b, width)
		}
	}
	if m.confirmAction != "" {
		b.WriteString("\n" + style(" DESTRUCTIVE ACTION ", "1;37;41") + " " + style("Press y to confirm · n / Esc to cancel", "1;31") + "\n")
	}
	if m.showHelp {
		b.WriteString("\n" + style("Press Esc, Enter, ?, or F1 to return", "90") + "\n")
	} else {
		b.WriteString("\n" + style("↑/↓ Navigate  Tab/1–4 Tabs  / Search  ? Help  Space Select  r Refresh  q Quit", "90") + "\n")
	}
	view := tea.NewView(b.String())
	view.AltScreen = true
	view.WindowTitle = "LANdapter · lndp"
	return view
}

func (m model) renderHelp(b *strings.Builder, width int) {
	b.WriteString(style("  KEYBOARD REFERENCE", "1;37;44") + "\n")
	b.WriteString(style(strings.Repeat("─", width), "90") + "\n")

	if width < 82 {
		lines := []string{
			"1/c Clients · 2/d Devices · 3/s Monitor",
			"4/f Files · Tab next tab",
			"↑/↓ or j/k move · Space mark/select",
			"Enter open/toggle · / search clients",
			"U upload marked local files",
			"I install selected packages on online targets",
			"M cycle quiet/normal/interactive installer",
			"X delete selected packages (confirm y)",
			"R refresh · Esc cancel/clear search",
			"Q/Ctrl+C quit · ?/F1 show this help",
		}
		for _, line := range lines {
			b.WriteString("  " + cell(line, max(1, width-2)) + "\n")
		}
	} else {
		rows := [][4]string{
			{"1 / c", "Clients and targets", "2 / d", "Devices and driver hints"},
			{"3 / s", "Live CPU/RAM/uptime", "4 / f", "Two-pane file manager"},
			{"Tab", "Next tab", "↑ / ↓, j / k", "Move through lists"},
			{"Space", "Mark file/client/package", "Enter", "Open directory/details"},
			{"/", "Search clients", "R", "Refresh current tab"},
			{"U", "Upload marked local files", "I", "Install on online targets"},
			{"M", "Cycle installer UI mode", "X then Y", "Delete package after prompt"},
			{"Esc", "Clear/cancel/back", "Q / Ctrl+C", "Quit console"},
		}
		leftWidth := (width - 5) / 2
		rightWidth := width - 5 - leftWidth
		renderCell := func(key, description string, columnWidth int) string {
			keyWidth := min(13, max(8, columnWidth/3))
			descriptionWidth := max(1, columnWidth-keyWidth-1)
			return style(cell(key, keyWidth), "1;36") + " " + cell(description, descriptionWidth)
		}
		for _, row := range rows {
			fmt.Fprintf(b, "  %s │ %s\n", renderCell(row[0], row[1], leftWidth), renderCell(row[2], row[3], rightWidth))
		}
	}
	b.WriteString("\n" + style(cell("Security: install/delete actions affect real clients and uploaded packages; verify targets.", width), "1;33") + "\n")
}

func (m model) renderClients(b *strings.Builder, width int) {
	clients := m.filteredClients()
	if m.searching {
		fmt.Fprintf(b, "Search: /%s▏\n", m.filter)
	} else if m.filter != "" {
		fmt.Fprintf(b, "Filter: %q  (Esc clears)\n", m.filter)
	}
	selected := countSelectedClients(m.selectedClients)
	fmt.Fprintf(b, "\n %s  %d shown / %d total · %d explicitly selected\n", style("CLIENT INVENTORY", "1;36"), len(clients), len(m.clients), selected)
	b.WriteString(style(strings.Repeat("─", width), "90") + "\n")
	compact := width < 112
	hostWidth, osWidth := 24, 20
	if compact {
		hostWidth = max(10, min(20, (width-22)/2))
		osWidth = max(8, min(18, width-hostWidth-23))
		fmt.Fprintf(b, "%-3s %-*s %-*s %s\n", "", hostWidth, "HOSTNAME", osWidth, "OS", "STATE")
	} else {
		fmt.Fprintf(b, "%-3s %-24s %-20s %-18s %-10s %s\n", "", "HOSTNAME", "OPERATING SYSTEM", "MAC ADDRESS", "STATE", "LAST SEEN")
	}
	rows := max(3, m.height-13)
	start := 0
	if m.cursor >= rows {
		start = m.cursor - rows + 1
	}
	end := min(len(clients), start+rows)
	for i := start; i < end; i++ {
		c := clients[i]
		mark := " "
		if i == m.cursor {
			mark = "▸"
		}
		sel := " "
		if m.selectedClients[c.ID] {
			sel = "✓"
		}
		state, stateColor := "● offline", "31"
		if c.Online {
			state, stateColor = "● online", "32"
		}
		var row string
		if compact {
			if c.Online {
				state = "● up"
			} else {
				state = "● down"
			}
			row = fmt.Sprintf("%-1s %-1s %-*s %-*s %s", mark, sel, hostWidth, cell(c.Hostname, hostWidth), osWidth, cell(c.OS, osWidth), state)
		} else {
			row = fmt.Sprintf("%-1s %-1s %-24s %-20s %-18s %-10s %s", mark, sel, cell(c.Hostname, 24), cell(c.OS, 20), cell(c.MAC, 18), state, cell(formatLastSeen(c.LastSeen), 19))
		}
		if i == m.cursor {
			row = style(row, "1;30;47")
		} else {
			row = style(row, stateColor)
		}
		b.WriteString(row + "\n")
	}
	if len(clients) == 0 && !m.loading {
		b.WriteString(style("No clients match. Press / to search or r to refresh.", "33") + "\n")
	}
	if m.details {
		if c, ok := m.selectedClient(); ok {
			b.WriteString("\n" + style("SELECTED CLIENT DETAILS", "1;36") + "\n")
			fmt.Fprintf(b, "ID        %s\nHostname  %s\nOS        %s\nMAC       %s\nLast seen %s\n", c.ID, c.Hostname, c.OS, c.MAC, formatLastSeen(c.LastSeen))
		}
	}
	if compact {
		b.WriteString("\n" + style("Space mark targets · Enter details · ? help · r refresh", "90") + "\n")
	} else {
		b.WriteString("\n" + style("Space mark multiple targets · Enter details · i in Files tab installs to marked online clients", "90") + "\n")
	}
}

func (m model) renderDevices(b *strings.Builder, width int) {
	c, ok := m.selectedClient()
	if !ok {
		b.WriteString("Select a client from the Clients tab first.\n")
		return
	}
	fmt.Fprintf(b, "%s %s  %s\n", style("DEVICE INVENTORY", "1;36"), style(c.Hostname, "1;37"), stateBadge(c.Online))
	if m.detailLoading {
		b.WriteString(style("◌ Querying the agent…", "33") + "\n")
	}
	if m.detailErr != nil {
		b.WriteString(style("! "+m.detailErr.Error(), "1;31") + "\n")
	}
	b.WriteString(style(strings.Repeat("─", width), "90") + "\n")
	compact := width < 108
	if compact {
		nameWidth := max(12, min(24, width-28))
		driverWidth := max(8, min(14, (width-nameWidth-14)/2))
		statusWidth := max(8, width-nameWidth-driverWidth-5)
		fmt.Fprintf(b, "%-*s %-*s %s\n", nameWidth, "DEVICE", driverWidth, "DRIVER", "STATUS")
		rows := max(1, m.height-17)
		start := clamp(m.deviceOffset, 0, max(0, len(m.devices)-rows))
		end := min(len(m.devices), start+rows)
		for i := start; i < end; i++ {
			d := m.devices[i]
			status, col := d.Status, "32"
			if d.DriverIssue {
				status, col = "DRIVER ISSUE", "1;31"
			} else if strings.Contains(strings.ToLower(status), "unknown") {
				col = "33"
			}
			statusCell := cell(status, statusWidth)
			if i != m.deviceCursor {
				statusCell = style(statusCell, col)
			}
			row := fmt.Sprintf("%-*s %-*s %s", nameWidth, cell(d.Name, nameWidth), driverWidth, cell(d.DriverName, driverWidth), statusCell)
			if i == m.deviceCursor {
				row = style(row, "1;30;47")
			}
			b.WriteString(row + "\n")
		}
	} else {
		fmt.Fprintf(b, "%-6s %-28s %-16s %-20s %-12s %s\n", "BUS", "DEVICE", "CLASS", "DRIVER", "VERSION", "STATUS")
		rows := max(1, m.height-17)
		start := clamp(m.deviceOffset, 0, max(0, len(m.devices)-rows))
		end := min(len(m.devices), start+rows)
		for i := start; i < end; i++ {
			d := m.devices[i]
			status := d.Status
			col := "32"
			if d.DriverIssue {
				status = "DRIVER ISSUE"
				col = "1;31"
			} else if strings.Contains(strings.ToLower(status), "unknown") {
				col = "33"
			}
			statusCell := cell(status, 16)
			if i != m.deviceCursor {
				statusCell = style(statusCell, col)
			}
			row := fmt.Sprintf("%-6s %-28s %-16s %-20s %-12s %s", cell(d.Bus, 6), cell(d.Name, 28), cell(d.Class, 16), cell(d.DriverName, 20), cell(d.DriverVersion, 12), statusCell)
			if i == m.deviceCursor {
				row = style(row, "1;30;47")
			}
			b.WriteString(row + "\n")
		}
	}
	rows := max(1, m.height-17)
	start := clamp(m.deviceOffset, 0, max(0, len(m.devices)-rows))
	end := min(len(m.devices), start+rows)
	if len(m.devices) == 0 && !m.detailLoading && m.detailErr == nil {
		b.WriteString("No devices returned. Confirm the agent is online and up to date.\n")
	} else if len(m.devices) > 0 && m.deviceCursor >= 0 && m.deviceCursor < len(m.devices) {
		d := m.devices[m.deviceCursor]
		b.WriteString("\n" + style("SELECTED DEVICE DETAILS", "1;36") + "\n")
		id := d.HardwareID
		if id == "" {
			id = d.InstanceID
		}
		if id != "" {
			fmt.Fprintf(b, "  Hardware ID  %s\n", cell(id, max(16, width-17)))
		}
		if d.Manufacturer != "" {
			fmt.Fprintf(b, "  Manufacturer  %s\n", cell(d.Manufacturer, max(16, width-17)))
		}
		if d.DriverIssue || d.Issue != "" {
			issue := d.Issue
			if issue == "" {
				issue = "Driver may be missing or require attention"
			}
			fmt.Fprintf(b, "  Diagnostic    %s\n", style(cell(issue, max(16, width-17)), "1;33"))
		}
		fmt.Fprintf(b, "  Device %d of %d\n", m.deviceCursor+1, len(m.devices))
	}
	if len(m.devices) > rows {
		fmt.Fprintf(b, "Showing devices %d–%d of %d. Use ↑/↓ or PgUp/PgDn to scroll.\n", start+1, end, len(m.devices))
	}
	b.WriteString("\n" + style("Driver issue flags are diagnostic hints; verify vendor/device IDs before installing drivers.", "90") + "\n")
}

func (m model) renderStats(b *strings.Builder, width int) {
	c, ok := m.selectedClient()
	if !ok {
		b.WriteString("Select a client from the Clients tab first.\n")
		return
	}
	fmt.Fprintf(b, "%s %s  %s\n", style("SYSTEM MONITOR", "1;36"), style(c.Hostname, "1;37"), stateBadge(c.Online))
	if m.detailLoading {
		b.WriteString(style("◌ Updating metrics…", "33") + "\n")
	}
	if m.detailErr != nil {
		b.WriteString(style("! "+m.detailErr.Error(), "1;31") + "\n")
	}
	if len(m.stats) == 0 {
		b.WriteString("\nWaiting for the first metrics sample.\n")
		return
	}
	cpu := number(m.stats["cpu_percent"])
	total := number(m.stats["mem_total_mb"])
	used := number(m.stats["mem_used_mb"])
	avail := number(m.stats["mem_available_mb"])
	if used == 0 && total > 0 && avail > 0 {
		used = maxFloat(0, total-avail)
	}
	memPercent := 0.0
	if total > 0 {
		memPercent = used / total * 100
	}
	barWidth := min(44, max(16, width/3))
	b.WriteString("\n" + style("  htop / RESOURCE SUMMARY", "1;37;44") + "\n")
	fmt.Fprintf(b, "  CPU  %s %5.1f%%\n", gauge(cpu, barWidth), cpu)
	fmt.Fprintf(b, "  MEM  %s %5.1f%%  %s / %s\n", gauge(memPercent, barWidth), memPercent, formatMB(used), formatMB(total))
	b.WriteString("\n" + style("  SYSTEM UPTIME", "1;36") + "\n")
	uptime := fmt.Sprint(m.stats["uptime_human"])
	if uptime == "<nil>" || uptime == "" {
		uptime = formatDuration(number(m.stats["uptime_seconds"]))
	}
	fmt.Fprintf(b, "  Host       %s\n  OS         %s\n  Uptime     %s\n  CPU load   %.1f%%\n  Memory     %s used of %s\n", c.Hostname, c.OS, uptime, cpu, formatMB(used), formatMB(total))
	if v, ok := m.stats["mem_available_mb"]; ok {
		fmt.Fprintf(b, "  Available  %s\n", formatMB(number(v)))
	}
	if v, ok := m.stats["error"]; ok {
		fmt.Fprintf(b, "\n%s\n", style("Agent metrics error: "+fmt.Sprint(v), "1;31"))
	}
	fmt.Fprintf(b, "\n%s\n", style("Live sample refreshes every 5s while this tab is active. Last seen: "+formatLastSeen(c.LastSeen), "90"))
}

func renderTabs(current string, width int) string {
	tabs := []struct{ key, label, compact, page string }{{"1", "Clients", "Cli", "clients"}, {"2", "Devices", "Dev", "devices"}, {"3", "Monitor", "Mon", "stats"}, {"4", "Files / Install", "Files", "files"}}
	if width < 64 {
		for i := range tabs {
			tabs[i].label = tabs[i].compact
		}
	}
	var parts []string
	for _, t := range tabs {
		label := " " + t.key + " " + t.label + " "
		if t.page == current {
			parts = append(parts, style(label, "1;37;44"))
		} else {
			parts = append(parts, style(label, "90"))
		}
	}
	return strings.Join(parts, style(" │ ", "90"))
}

func stateBadge(online bool) string {
	if online {
		return style(" ONLINE ", "1;30;42")
	}
	return style(" OFFLINE ", "1;37;41")
}

func style(text, code string) string {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func number(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

func gauge(percent float64, width int) string {
	percent = maxFloat(0, minFloat(100, percent))
	filled := int(percent*float64(width)/100 + 0.5)
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	code := "32"
	if percent >= 85 {
		code = "1;31"
	} else if percent >= 65 {
		code = "33"
	}
	return "[" + style(bar, code) + "]"
}

func formatMB(value float64) string {
	if value < 0 {
		value = 0
	}
	if value >= 1024 {
		return fmt.Sprintf("%.1f GiB", value/1024)
	}
	return fmt.Sprintf("%.0f MiB", value)
}
func formatDuration(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	d := time.Duration(seconds * float64(time.Second))
	days := int(d / (24 * time.Hour))
	d %= 24 * time.Hour
	return fmt.Sprintf("%dd %02dh %02dm", days, int(d/time.Hour), int(d%time.Hour/time.Minute))
}
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func formatLastSeen(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return parsed.Local().Format("2006-01-02 15:04:05")
}

func cell(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = strings.TrimSpace(value)
	if runewidth.StringWidth(value) > width {
		value = runewidth.Truncate(value, width, "…")
	}
	return value + strings.Repeat(" ", max(0, width-runewidth.StringWidth(value)))
}

var ansiPattern = regexp.MustCompile("\\x1b\\[[0-9;]*m")

func visibleLen(s string) int { return runewidth.StringWidth(ansiPattern.ReplaceAllString(s, "")) }
func padStyled(s string, width int) string {
	n := visibleLen(s)
	if n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}

func clamp(value, minValue, maxValue int) int {
	if maxValue < minValue {
		return minValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

// Ensure deterministic client order from Master payloads in tests and offline mode.
func sortClients(clients []clientRecord) {
	sort.SliceStable(clients, func(i, j int) bool {
		if clients[i].Online != clients[j].Online {
			return clients[i].Online
		}
		return strings.ToLower(clients[i].Hostname) < strings.ToLower(clients[j].Hostname)
	})
}

func toggleSelection(selection map[string]bool, key string) {
	if selection[key] {
		delete(selection, key)
		return
	}
	selection[key] = true
}
