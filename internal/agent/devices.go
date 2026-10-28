package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// DeviceInfo is the normalized, read-only hardware inventory reported by an agent.
// DriverIssue is a diagnostic hint, not a guarantee that installing a driver will fix a device.
type DeviceInfo struct {
	Name          string `json:"name"`
	Class         string `json:"class,omitempty"`
	Status        string `json:"status"`
	InstanceID    string `json:"instance_id,omitempty"`
	HardwareID    string `json:"hardware_id,omitempty"`
	Manufacturer  string `json:"manufacturer,omitempty"`
	DriverName    string `json:"driver_name,omitempty"`
	DriverVersion string `json:"driver_version,omitempty"`
	ProblemCode   int    `json:"problem_code,omitempty"`
	DriverIssue   bool   `json:"driver_issue"`
	Issue         string `json:"issue,omitempty"`
	Bus           string `json:"bus"`
}

func (a *Agent) collectDevices() interface{} {
	// Device discovery is comparatively expensive on Windows. Cache successful
	// inventories for two minutes and a partial result for 45 seconds so page
	// refreshes or repeated TUI requests do not launch the same WMI scans again.
	a.deviceInventoryMu.Lock()
	defer a.deviceInventoryMu.Unlock()

	if a.deviceInventoryCache != nil && !a.deviceInventoryCachedAt.IsZero() {
		ttl := 2 * time.Minute
		if a.deviceInventoryPartial {
			ttl = 45 * time.Second
		}
		if time.Since(a.deviceInventoryCachedAt) < ttl {
			return a.deviceInventoryCache
		}
	}

	result, partial := a.collectDevicesUncached()
	a.deviceInventoryCache = result
	a.deviceInventoryCachedAt = time.Now()
	a.deviceInventoryPartial = partial
	return result
}

func (a *Agent) collectDevicesUncached() (interface{}, bool) {
	var (
		devices []DeviceInfo
		err     error
	)
	switch runtime.GOOS {
	case "windows":
		devices, err = collectWindowsDevices()
	case "linux":
		devices, err = collectLinuxDevices()
	default:
		devices = collectNetworkInterfaces()
	}
	if err != nil {
		a.logger.Warn("Device inventory collection was partial", "error", err)
		if devices == nil {
			devices = []DeviceInfo{}
		}
		return map[string]interface{}{"devices": devices, "warning": err.Error()}, true
	}
	if devices == nil {
		devices = []DeviceInfo{}
	}
	return devices, false
}

func runPowerShell(script string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("PowerShell inventory timed out after 20 seconds")
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// windowsInventoryScript intentionally uses bulk CIM queries. Calling
// Get-PnpDeviceProperty repeatedly for each device is very slow on hosts with
// many devices and can exceed the Master request timeout.
const windowsInventoryScript = `[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$ErrorActionPreference = 'Stop'
$pnpDevices = @(Get-PnpDevice -PresentOnly -ErrorAction Stop)
$entities = @{}
Get-CimInstance -ClassName Win32_PnPEntity -ErrorAction SilentlyContinue | ForEach-Object {
  $entityId = [string]$_.PNPDeviceID
  if ([string]::IsNullOrWhiteSpace($entityId)) { $entityId = [string]$_.DeviceID }
  if (-not [string]::IsNullOrWhiteSpace($entityId)) { $entities[$entityId.ToUpperInvariant()] = $_ }
}
$drivers = @{}
try {
  Get-CimInstance -ClassName Win32_PnPSignedDriver -ErrorAction SilentlyContinue | ForEach-Object {
    $driverId = [string]$_.DeviceID
    if (-not [string]::IsNullOrWhiteSpace($driverId)) { $drivers[$driverId.ToUpperInvariant()] = $_ }
  }
} catch {}
$items = @($pnpDevices | ForEach-Object {
  $pnp = $_
  $id = [string]$pnp.InstanceId
  $key = $id.ToUpperInvariant()
  $entity = $entities[$key]
  $driver = $drivers[$key]
  $hardwareIds = $null
  if ($null -ne $entity) { $hardwareIds = $entity.HardwareID }
  $hardwareId = ''
  if ($hardwareIds -is [array] -and $hardwareIds.Length -gt 0) { $hardwareId = [string]$hardwareIds[0] }
  elseif ($null -ne $hardwareIds) { $hardwareId = [string]$hardwareIds }
  if ([string]::IsNullOrWhiteSpace($hardwareId)) { $hardwareId = $id }
  $problem = 0
  if ($null -ne $entity -and $null -ne $entity.ConfigManagerErrorCode) { $problem = [int]$entity.ConfigManagerErrorCode }
  elseif ($null -ne $pnp.Problem) { $problem = [int]$pnp.Problem }
  $issue = ''
  if ($problem -eq 28) { $issue = 'No driver installed (problem code 28)' }
  elseif ($problem -ne 0) { $issue = "Device problem code $problem" }
  $status = [string]$pnp.Status
  if ($problem -ne 0) { $status = 'Error' }
  $driverName = ''
  $driverVersion = ''
  if ($null -ne $driver) {
    $driverName = [string]$driver.DriverProviderName
    if ([string]::IsNullOrWhiteSpace($driverName)) { $driverName = [string]$driver.DriverName }
    $driverVersion = [string]$driver.DriverVersion
  }
  $bus = 'PnP'
  if ($id.StartsWith('PCI\', [System.StringComparison]::OrdinalIgnoreCase)) { $bus = 'PCI' }
  elseif ($id.StartsWith('USB\', [System.StringComparison]::OrdinalIgnoreCase)) { $bus = 'USB' }
  elseif ($id.StartsWith('BTH\', [System.StringComparison]::OrdinalIgnoreCase)) { $bus = 'Bluetooth' }
  $name = [string]$pnp.FriendlyName
  $manufacturer = ''
  if ($null -ne $entity) {
    if ([string]::IsNullOrWhiteSpace($name)) { $name = [string]$entity.Name }
    $manufacturer = [string]$entity.Manufacturer
  }
  [PSCustomObject]@{
    name = $name; class = [string]$pnp.Class; status = $status;
    instance_id = $id; hardware_id = $hardwareId; manufacturer = $manufacturer;
    driver_name = $driverName; driver_version = $driverVersion; problem_code = $problem;
    driver_issue = ($problem -ne 0); issue = $issue; bus = $bus
  }
})
ConvertTo-Json -InputObject $items -Depth 4 -Compress`

func collectWindowsDevices() ([]DeviceInfo, error) {
	out, err := runPowerShell(windowsInventoryScript)
	if err != nil {
		return nil, fmt.Errorf("query Windows Plug and Play inventory: %w", err)
	}
	return decodeWindowsDevices(out)
}

func decodeWindowsDevices(out []byte) ([]DeviceInfo, error) {
	devices := make([]DeviceInfo, 0)
	if strings.TrimSpace(string(out)) == "" {
		return devices, nil
	}
	if err := json.Unmarshal(out, &devices); err != nil {
		return nil, fmt.Errorf("decode Windows device inventory: %w", err)
	}
	for i := range devices {
		if devices[i].Name == "" {
			devices[i].Name = devices[i].InstanceID
		}
		if devices[i].Status == "" {
			devices[i].Status = "unknown"
		}
	}
	return devices, nil
}

func collectLinuxDevices() ([]DeviceInfo, error) {
	devices := make([]DeviceInfo, 0)
	var scanErrors []string
	for _, bus := range []string{"pci", "usb"} {
		root := filepath.Join("/sys/bus", bus, "devices")
		entries, err := os.ReadDir(root)
		if err != nil {
			if !os.IsNotExist(err) {
				scanErrors = append(scanErrors, err.Error())
			}
			continue
		}
		for _, entry := range entries {
			id := entry.Name()
			path := filepath.Join(root, id)
			if bus == "usb" && strings.Contains(id, ":") == false {
				// USB interface nodes carry the actual driver binding; only include a
				// physical node when no interfaces exist below it.
				if hasUSBInterface(entries, id) {
					continue
				}
			}
			device := DeviceInfo{Bus: strings.ToUpper(bus), InstanceID: id, Status: "unknown"}
			if bus == "usb" {
				device.Manufacturer = readUSBAttribute(path, "manufacturer")
				device.Name = readUSBAttribute(path, "product")
			} else {
				device.Manufacturer = readSysfs(path, "manufacturer")
				device.Name = ""
			}
			if bus == "pci" {
				vendor := readSysfs(path, "vendor")
				product := readSysfs(path, "device")
				device.HardwareID = linuxHardwareID("pci", vendor, product)
				device.Class = readSysfs(path, "class")
				if device.Name == "" {
					device.Name = strings.TrimSpace(strings.TrimPrefix(vendor, "0x") + ":" + strings.TrimPrefix(product, "0x"))
				}
			} else {
				vendor := readUSBAttribute(path, "idVendor")
				product := readUSBAttribute(path, "idProduct")
				device.HardwareID = linuxHardwareID("usb", vendor, product)
				if device.Name == "" {
					device.Name = readUSBAttribute(path, "uevent")
				}
				device.Class = readUSBAttribute(path, "bDeviceClass")
				if device.Class == "" {
					device.Class = readUSBAttribute(path, "bInterfaceClass")
				}
			}
			if target, err := os.Readlink(filepath.Join(path, "driver")); err == nil {
				device.DriverName = filepath.Base(target)
				device.Status = "driver-bound"
			} else {
				device.Status = "driver-missing"
				device.DriverIssue = true
				device.Issue = "No driver is bound in sysfs; check the kernel driver and device support"
			}
			if device.Name == "" {
				device.Name = id
			}
			devices = append(devices, device)
		}
	}
	if len(devices) == 0 && len(scanErrors) > 0 {
		return devices, fmt.Errorf("read Linux device inventory: %s", strings.Join(scanErrors, "; "))
	}
	return devices, nil
}

func linuxHardwareID(bus, vendor, product string) string {
	vendor = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(vendor)), "0X")
	product = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(product)), "0X")
	if vendor == "" || product == "" {
		return ""
	}
	switch strings.ToLower(bus) {
	case "pci":
		return "PCI\\VEN_" + vendor + "&DEV_" + product
	case "usb":
		return "USB\\VID_" + vendor + "&PID_" + product
	default:
		return ""
	}
}

func hasUSBInterface(entries []os.DirEntry, deviceID string) bool {
	prefix := deviceID + ":"
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			return true
		}
	}
	return false
}

func readUSBAttribute(root, name string) string {
	if value := readSysfs(root, name); value != "" {
		return value
	}
	// USB interface nodes (for example 1-2:1.0) often inherit vendor/product
	// identifiers from their parent physical device rather than exposing them
	// as files of their own. Resolve the sysfs link before walking to the parent.
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return ""
	}
	return readSysfs(filepath.Dir(resolved), name)
}

func readSysfs(root, name string) string {
	data, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func collectNetworkInterfaces() []DeviceInfo {
	// Unsupported platforms return an empty inventory until platform-specific discovery is implemented.
	return []DeviceInfo{}
}
