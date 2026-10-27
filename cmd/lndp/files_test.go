package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLocalEntriesDirectoriesFirstAndParent(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "setup.exe"), []byte("installer"), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := readLocalEntries(dir)
	if err != nil {
		t.Fatalf("readLocalEntries: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want parent, directory, file: %#v", len(entries), entries)
	}
	if entries[0].Name != ".." || !entries[0].IsDir {
		t.Fatalf("first item should be parent directory: %#v", entries[0])
	}
	if entries[1].Name != "subdir" || !entries[1].IsDir {
		t.Fatalf("directories should precede files: %#v", entries[1])
	}
	if entries[2].Name != "setup.exe" || entries[2].IsDir || entries[2].Size != int64(len("installer")) {
		t.Fatalf("unexpected file row: %#v", entries[2])
	}
}

func TestReadLocalEntriesMissingDirectory(t *testing.T) {
	_, err := readLocalEntries(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected an error for a missing directory")
	}
}

func TestTargetLocalPathsIncludesSelectionsAcrossDirectories(t *testing.T) {
	m := model{
		localCursor:  0,
		localEntries: []localEntry{{Name: "visible.msi", Path: filepath.Join("/tmp", "visible.msi")}},
		selectedLocal: map[string]bool{
			filepath.Join("/tmp", "visible.msi"):         true,
			filepath.Join("/tmp", "other", "driver.inf"): true,
		},
	}
	got := m.targetLocalPaths()
	if len(got) != 2 {
		t.Fatalf("targetLocalPaths returned %#v, want both selections", got)
	}
	if got[0] > got[1] {
		t.Fatalf("target paths should be stable-sorted, got %#v", got)
	}
}

func TestTargetRemoteFileIDsFallsBackToCursor(t *testing.T) {
	m := model{remoteCursor: 1, remoteFiles: []packageRecord{{ID: "a"}, {ID: "b"}}}
	got := m.targetRemoteFileIDs()
	if len(got) != 1 || got[0] != "b" {
		t.Fatalf("targetRemoteFileIDs = %#v, want current package b", got)
	}
}

func TestSelectedOnlineClientIDsExcludesOfflineSelections(t *testing.T) {
	m := model{
		clients:         []clientRecord{{ID: "up", Hostname: "online", Online: true}, {ID: "down", Hostname: "offline", Online: false}},
		selectedClients: map[string]bool{"up": true, "down": true},
	}
	got := m.selectedOnlineClientIDs()
	if len(got) != 1 || got[0] != "up" {
		t.Fatalf("selectedOnlineClientIDs = %#v, want only online selected client", got)
	}
}

func TestGaugeAndHumanSize(t *testing.T) {
	if got := visibleLen(gauge(50, 10)); got != 12 { // brackets plus ten visible bar blocks
		t.Fatalf("gauge visible width = %d, want 12", got)
	}
	if got := humanSize(1024); got != "1.0 KiB" {
		t.Fatalf("humanSize(1024) = %q, want 1.0 KiB", got)
	}
}

func TestSelectedOfflineClientDoesNotFallBackToUnselectedOnlineClient(t *testing.T) {
	m := model{
		clients:         []clientRecord{{ID: "up", Hostname: "online", Online: true}, {ID: "down", Hostname: "offline", Online: false}},
		cursor:          0,
		selectedClients: map[string]bool{"down": true},
	}
	if got := m.selectedOnlineClientIDs(); len(got) != 0 {
		t.Fatalf("selectedOnlineClientIDs = %#v, want no targets when only explicit selection is offline", got)
	}
}

func TestRenderDevicesKeepsHardwareDetailsForSelectedRow(t *testing.T) {
	m := model{
		clients: []clientRecord{{ID: "client-1", Hostname: "workstation", Online: true}},
		height:  24,
		devices: []deviceRecord{
			{Name: "Network adapter", Bus: "PCI", Status: "OK", HardwareID: "PCI\\VEN_8086", Manufacturer: "Intel"},
			{Name: "Unknown device", Bus: "USB", Status: "Unknown", DriverIssue: true, HardwareID: "USB\\VID_1234", Manufacturer: "Vendor", Issue: "Driver missing"},
			{Name: "Audio device", Bus: "HDA", Status: "OK", HardwareID: "HDAUDIO\\FUNC_01", Manufacturer: "Realtek"},
		},
		deviceCursor: 1,
	}
	var b strings.Builder
	m.renderDevices(&b, 100)
	got := b.String()
	if strings.Count(got, "Hardware ID") != 1 || !strings.Contains(got, "USB\\VID_1234") {
		t.Fatalf("renderDevices should show details only for selected device: %s", got)
	}
	if strings.Contains(got, "PCI\\VEN_8086") || strings.Contains(got, "HDAUDIO\\FUNC_01") {
		t.Fatalf("renderDevices printed non-selected device details: %s", got)
	}
}

func TestClipTextRespectsWideUnicodeDisplayColumns(t *testing.T) {
	got := clipText("网卡适配器", 5)
	if width := visibleLen(got); width > 5 {
		t.Fatalf("clipText width = %d, want <= 5: %q", width, got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("clipText should mark a truncated value, got %q", got)
	}
}

func TestRenderFileManagerFitsNarrowTerminal(t *testing.T) {
	longDir := filepath.Join(t.TempDir(), "directory-with-a-long-name")
	if err := os.Mkdir(longDir, 0o755); err != nil {
		t.Fatal(err)
	}
	m := model{
		width:           48,
		height:          24,
		page:            "files",
		filePane:        "local",
		localDir:        longDir,
		localEntries:    []localEntry{{Name: "installer-file-with-long-name.msi", Path: filepath.Join(longDir, "installer-file-with-long-name.msi"), Size: 1024}},
		remoteFiles:     []packageRecord{{ID: "package-id-that-is-long", Name: "installer-package-with-long-name.msi", Size: 2048}},
		selectedLocal:   map[string]bool{},
		selectedRemote:  map[string]bool{},
		selectedClients: map[string]bool{},
		installMode:     "interactive",
	}
	var b strings.Builder
	m.renderFileManager(&b, 48)
	for i, line := range strings.Split(strings.TrimRight(b.String(), "\n"), "\n") {
		if width := visibleLen(line); width > 48 {
			t.Errorf("line %d exceeds terminal width: %d > 48: %q", i+1, width, line)
		}
	}
}
