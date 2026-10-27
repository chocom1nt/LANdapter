package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHasUSBInterface(t *testing.T) {
	entries := []os.DirEntry{
		fakeDirEntry("1-1"),
		fakeDirEntry("1-1:1.0"),
		fakeDirEntry("2-2"),
	}
	if !hasUSBInterface(entries, "1-1") {
		t.Fatal("expected USB physical device to be matched to its interface")
	}
	if hasUSBInterface(entries, "2-2") {
		t.Fatal("unexpected USB interface match")
	}
}

type testDirEntry string

func fakeDirEntry(name string) os.DirEntry        { return testDirEntry(name) }
func (e testDirEntry) Name() string               { return string(e) }
func (e testDirEntry) IsDir() bool                { return true }
func (e testDirEntry) Type() os.FileMode          { return os.ModeDir }
func (e testDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func TestDecodeStatsOutput(t *testing.T) {
	got, ok := decodeStatsOutput([]byte(`{"cpu_percent":12.5,"mem_total_mb":2048}`)).(map[string]interface{})
	if !ok {
		t.Fatal("expected decoded statistics map")
	}
	if got["cpu_percent"] != 12.5 {
		t.Fatalf("cpu_percent = %#v, want 12.5", got["cpu_percent"])
	}
	bad, ok := decodeStatsOutput([]byte(`not-json`)).(map[string]string)
	if !ok || bad["error"] == "" {
		t.Fatalf("expected parse diagnostic for invalid stats, got %#v", bad)
	}
}

func TestLinuxHardwareID(t *testing.T) {
	tests := []struct {
		name    string
		bus     string
		vendor  string
		product string
		want    string
	}{
		{name: "PCI ID", bus: "pci", vendor: "0x8086", product: "0x15f3", want: `PCI\VEN_8086&DEV_15F3`},
		{name: "USB ID", bus: "usb", vendor: "046d", product: "c534", want: `USB\VID_046D&PID_C534`},
		{name: "incomplete ID", bus: "usb", vendor: "046d", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := linuxHardwareID(tt.bus, tt.vendor, tt.product); got != tt.want {
				t.Fatalf("linuxHardwareID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadUSBAttributeFallsBackToPhysicalParent(t *testing.T) {
	root := t.TempDir()
	physical := filepath.Join(root, "devices", "1-2")
	iface := filepath.Join(physical, "1-2:1.0")
	busAlias := filepath.Join(root, "bus", "usb", "devices", "1-2:1.0")
	if err := os.MkdirAll(iface, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(busAlias), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(physical, "idVendor"), []byte("046d\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(iface, busAlias); err != nil {
		t.Skipf("symlinks are unavailable in this environment: %v", err)
	}
	if got := readUSBAttribute(busAlias, "idVendor"); got != "046d" {
		t.Fatalf("readUSBAttribute() = %q, want 046d", got)
	}
}

func TestDecodeWindowsDevicesNormalizesMissingFields(t *testing.T) {
	got, err := decodeWindowsDevices([]byte(`[{"name":"","instance_id":"PCI\\VEN_8086&DEV_15F3","status":""}]`))
	if err != nil {
		t.Fatalf("decodeWindowsDevices() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d devices, want 1", len(got))
	}
	if got[0].Name != got[0].InstanceID {
		t.Fatalf("missing name was not normalized: %#v", got[0])
	}
	if got[0].Status != "unknown" {
		t.Fatalf("status = %q, want unknown", got[0].Status)
	}
}

func TestWindowsInventoryUsesBulkQueries(t *testing.T) {
	for _, expected := range []string{
		"Get-PnpDevice -PresentOnly",
		"Get-CimInstance -ClassName Win32_PnPEntity",
		"Get-CimInstance -ClassName Win32_PnPSignedDriver",
	} {
		if !strings.Contains(windowsInventoryScript, expected) {
			t.Errorf("windowsInventoryScript missing bulk query %q", expected)
		}
	}
	if strings.Contains(windowsInventoryScript, "Get-PnpDeviceProperty") {
		t.Fatal("windowsInventoryScript must not query properties separately for every device")
	}
}

func TestCollectDevicesUsesFreshCache(t *testing.T) {
	cached := []DeviceInfo{{Name: "cached-device", Status: "OK"}}
	a := &Agent{
		deviceInventoryCache:    cached,
		deviceInventoryCachedAt: time.Now(),
	}
	result := a.collectDevices()
	got, ok := result.([]DeviceInfo)
	if !ok {
		t.Fatalf("cached inventory type = %T, want []DeviceInfo", result)
	}
	if len(got) != 1 || got[0].Name != "cached-device" {
		t.Fatalf("collectDevices() = %#v, want cached inventory", got)
	}
}
