package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeAPIBase(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		allow   bool
		wantErr bool
	}{
		{name: "default local http", input: "http://127.0.0.1:8080/api/v1"},
		{name: "localhost with trailing slash", input: "http://localhost:8080/api/v1/"},
		{name: "remote plain http rejected", input: "http://192.168.1.20:8080/api/v1", wantErr: true},
		{name: "remote plain http override", input: "http://192.168.1.20:8080/api/v1", allow: true},
		{name: "query rejected", input: "https://master.example/api/v1?token=x", wantErr: true},
		{name: "wrong path rejected", input: "https://master.example/api", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeAPIBase(tt.input, tt.allow)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeAPIBase() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got == "" {
				t.Fatal("normalizeAPIBase() returned empty URL")
			}
		})
	}
}

func TestFilteredClients(t *testing.T) {
	m := model{clients: []clientRecord{
		{ID: "1", Hostname: "WS-ADMIN", OS: "Windows 11", MAC: "aa:bb"},
		{ID: "2", Hostname: "ubuntu-srv", OS: "Ubuntu", MAC: "cc:dd"},
	}}
	m.filter = "UBUNTU"
	got := m.filteredClients()
	if len(got) != 1 || got[0].ID != "2" {
		t.Fatalf("filteredClients() = %#v, want only ubuntu-srv", got)
	}
}

func TestFetchClientDetailsStructuredDevices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+strings.Repeat("x", 40) {
			t.Fatalf("unexpected authorization header: %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/api/v1/clients/client-1/devices" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"Intel Ethernet","class":"Network","status":"OK","driver_issue":false,"driver_name":"e1000e","hardware_id":"PCI\\VEN_8086&DEV_15F3","bus":"PCI"}]`))
	}))
	defer server.Close()
	t.Setenv("LANDAPTER_ADMIN_TOKEN", strings.Repeat("x", 40))
	t.Setenv("LANDAPTER_API_URL", server.URL+"/api/v1")
	t.Setenv("LANDAPTER_ALLOW_INSECURE_HTTP", "1")

	got := fetchClientDetails("client-1", "devices")
	if got.err != nil {
		t.Fatalf("fetchClientDetails() error = %v", got.err)
	}
	if len(got.devices) != 1 || got.devices[0].Name != "Intel Ethernet" || got.devices[0].DriverName != "e1000e" || got.devices[0].HardwareID != `PCI\VEN_8086&DEV_15F3` {
		t.Fatalf("unexpected devices payload: %#v", got.devices)
	}
}

func TestFetchClientDetailsSupportsLegacyJSONString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(`[{"name":"Legacy device","status":"OK"}]`)
	}))
	defer server.Close()
	t.Setenv("LANDAPTER_ADMIN_TOKEN", strings.Repeat("y", 40))
	t.Setenv("LANDAPTER_API_URL", server.URL+"/api/v1")
	t.Setenv("LANDAPTER_ALLOW_INSECURE_HTTP", "1")

	got := fetchClientDetails("client-2", "devices")
	if got.err != nil || len(got.devices) != 1 || got.devices[0].Name != "Legacy device" {
		t.Fatalf("legacy payload decode = %#v, err=%v", got.devices, got.err)
	}
}

func TestFetchClientDetailsExplainsAgentTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Agent did not respond within 30 seconds", http.StatusRequestTimeout)
	}))
	defer server.Close()
	t.Setenv("LANDAPTER_ADMIN_TOKEN", strings.Repeat("z", 40))
	t.Setenv("LANDAPTER_API_URL", server.URL+"/api/v1")
	t.Setenv("LANDAPTER_ALLOW_INSECURE_HTTP", "1")

	got := fetchClientDetails("client-timeout", "devices")
	if got.err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(got.err.Error(), "update/restart the agent") {
		t.Fatalf("timeout message should recommend updating the agent, got %q", got.err)
	}
}

func TestFetchClientDetailsExplainsInactiveConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Client is not online", http.StatusConflict)
	}))
	defer server.Close()
	t.Setenv("LANDAPTER_ADMIN_TOKEN", strings.Repeat("w", 40))
	t.Setenv("LANDAPTER_API_URL", server.URL+"/api/v1")
	t.Setenv("LANDAPTER_ALLOW_INSECURE_HTTP", "1")

	got := fetchClientDetails("client-offline", "devices")
	if got.err == nil {
		t.Fatal("expected inactive connection error")
	}
	if !strings.Contains(got.err.Error(), "no active WebSocket connection") {
		t.Fatalf("offline message should mention WebSocket state, got %q", got.err)
	}
}

func TestRenderHelpListsCriticalActions(t *testing.T) {
	m := model{width: 110, height: 32}
	var b strings.Builder
	m.renderHelp(&b, 110)
	got := b.String()
	for _, expected := range []string{"KEYBOARD REFERENCE", "Two-pane file manager", "U", "I", "X then Y", "Security:"} {
		if !strings.Contains(got, expected) {
			t.Errorf("renderHelp missing %q", expected)
		}
	}
}

func TestRenderHelpFitsSmallTerminal(t *testing.T) {
	m := model{width: 48, height: 24}
	var b strings.Builder
	m.renderHelp(&b, 48)
	for i, line := range strings.Split(strings.TrimRight(b.String(), "\n"), "\n") {
		if width := visibleLen(line); width > 48 {
			t.Errorf("help line %d exceeds terminal width: %d > 48: %q", i+1, width, line)
		}
	}
	for _, expected := range []string{"U upload", "I install", "X delete", "Security:"} {
		if !strings.Contains(b.String(), expected) {
			t.Errorf("narrow help missing %q", expected)
		}
	}
}

func TestCellAndVisibleLenRespectWideUnicode(t *testing.T) {
	got := cell("网卡", 6)
	if width := visibleLen(got); width != 6 {
		t.Fatalf("display width of padded CJK cell = %d, want 6 (%q)", width, got)
	}
	clipped := cell("网卡适配器名字很长", 8)
	if width := visibleLen(clipped); width != 8 {
		t.Fatalf("display width of truncated CJK cell = %d, want 8 (%q)", width, clipped)
	}
}

func TestCompactClientTableFitsNarrowTerminal(t *testing.T) {
	m := model{
		clients: []clientRecord{{ID: "1", Hostname: "workstation-very-long-name", OS: "Windows 11 Enterprise", Online: true}},
		height:  24,
	}
	var b strings.Builder
	m.renderClients(&b, 60)
	for i, line := range strings.Split(strings.TrimRight(b.String(), "\n"), "\n") {
		if width := visibleLen(line); width > 60 {
			t.Errorf("line %d exceeds terminal width: %d > 60: %q", i+1, width, line)
		}
	}
}

func TestCompactTabsFitSmallTerminal(t *testing.T) {
	got := renderTabs("files", 48)
	if width := visibleLen(got); width > 48 {
		t.Fatalf("compact tabs exceed 48 columns: %d (%q)", width, got)
	}
}
