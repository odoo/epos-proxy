//go:build windows

package printer

import (
	"fmt"
	"strings"

	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows/registry"
)

type Win32_PnPEntity struct {
	Name     string
	DeviceID string
}

func getPrinterFriendlyName(vid, pid string) string {
	name, _, _ := findPnPDeviceNameByVidPid(vid, pid)

	// If WMI returned something useful (not generic "USB..." name) use it
	if name != "" && !strings.Contains(strings.ToUpper(name), "USB") {
		return name
	}

	// Fallback: look up clean model name from USBPRINT registry
	if regName := findUSBPrintModel(vid, pid); regName != "" {
		return regName
	}

	return fmt.Sprintf("USB ID: %s %s", vid, pid)
}

// ── WMI lookup ────────────────────────────────────────────────────────────────

func findPnPDeviceNameByVidPid(vid, pid string) (string, string, error) {
	vid = strings.ToUpper(strings.TrimSpace(vid))
	pid = strings.ToUpper(strings.TrimSpace(pid))

	if len(vid) == 0 || len(pid) == 0 {
		return "", "", fmt.Errorf("vid and pid must not be empty")
	}

	var entities []Win32_PnPEntity

	q := fmt.Sprintf("SELECT Name, DeviceID FROM Win32_PnPEntity WHERE DeviceID LIKE '%%VID_%s%%'", vid)
	if err := wmi.Query(q, &entities); err != nil {
		return "", "", fmt.Errorf("WMI query failed: %w", err)
	}

	needle1 := "VID_" + vid
	needle2 := "PID_" + pid
	needleConcat := "VID_" + vid + "&PID_" + pid

	for _, e := range entities {
		id := strings.ToUpper(e.DeviceID)
		if strings.Contains(id, needleConcat) {
			return e.Name, e.DeviceID, nil
		}
		if strings.Contains(id, needle1) && strings.Contains(id, needle2) {
			return e.Name, e.DeviceID, nil
		}
	}

	return "", "", nil
}

// ── Registry fallback ─────────────────────────────────────────────────────────

// findUSBPrintModel looks up the clean model name from the USBPRINT registry
// by linking Enum\USB (ParentIdPrefix value) → Enum\USBPRINT (instance key name).
func findUSBPrintModel(vid, pid string) string {
	vid = strings.ToUpper(vid)
	pid = strings.ToUpper(pid)

	prefix := readParentIdPrefix(vid, pid)
	if prefix == "" {
		return ""
	}
	prefixUpper := strings.ToUpper(prefix)

	root, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Enum\USBPRINT`,
		registry.ENUMERATE_SUB_KEYS,
	)
	if err != nil {
		return ""
	}
	defer root.Close()

	models, _ := root.ReadSubKeyNames(-1)
	for _, model := range models {
		modelKey, err := registry.OpenKey(root, model, registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			continue
		}
		instances, _ := modelKey.ReadSubKeyNames(-1)
		modelKey.Close()

		for _, instance := range instances {
			if strings.HasPrefix(strings.ToUpper(instance), prefixUpper) {
				return model // e.g. "EPSONTM-T30II" — clean, no AC25 suffix
			}
		}
	}
	return ""
}

// readParentIdPrefix reads the ParentIdPrefix value from the connected
// device instance under Enum\USB\VID_xxxx&PID_xxxx.
func readParentIdPrefix(vid, pid string) string {
	keyPath := fmt.Sprintf(
		`SYSTEM\CurrentControlSet\Enum\USB\VID_%s&PID_%s`,
		vid, pid,
	)
	devKey, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		keyPath,
		registry.ENUMERATE_SUB_KEYS,
	)
	if err != nil {
		return ""
	}
	defer devKey.Close()

	instances, _ := devKey.ReadSubKeyNames(-1)
	for _, instance := range instances {
		instKey, err := registry.OpenKey(devKey, instance, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		flags, _, _ := instKey.GetIntegerValue("ConfigFlags")
		prefix, _, err := instKey.GetStringValue("ParentIdPrefix")
		instKey.Close()

		if err == nil && prefix != "" && flags == 0 {
			return prefix
		}
	}
	return ""
}
