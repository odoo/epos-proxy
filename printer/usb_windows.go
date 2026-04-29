//go:build windows

package printer

import (
	"epos-proxy/logger"
	"epos-proxy/util"
	"fmt"
	"os/exec"
	"strings"

	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows/registry"
)

type Win32_PnPEntity struct {
	Name     string
	DeviceID string
}

func getPrinterFriendlyName(vid, pid string) string {
	logger.Debugf("Attempting to get name for VID:%s PID:%s", vid, pid)
	name, _, _ := findPnPDeviceNameByVidPid(vid, pid)

	// If WMI returned something useful (not generic "USB..." name) use it
	if name != "" && !strings.Contains(strings.ToUpper(name), "USB") {
		return name
	}

	// Fallback: look up clean model name from USBPRINT registry
	logger.Debug("Falling back to registry lookup for printer friendly name")
	if regName := findUSBPrintModel(vid, pid); regName != "" {
		return regName
	}
	logger.Debug("Using generic name for printer")

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

	logger.Debugf("Querying WMI for device with VID_%s", vid)
	q := fmt.Sprintf("SELECT Name, DeviceID FROM Win32_PnPEntity WHERE DeviceID LIKE '%%VID_%s%%'", vid)
	if err := wmi.Query(q, &entities); err != nil {
		return "", "", fmt.Errorf("WMI query failed: %w", err)
	}

	needle1 := "VID_" + vid
	needle2 := "PID_" + pid
	needleConcat := "VID_" + vid + "&PID_" + pid

	for _, e := range entities {
		logger.Debugf("Checking WMI entity: Name=%s, DeviceID=%s", e.Name, e.DeviceID)
		id := strings.ToUpper(e.DeviceID)
		if strings.Contains(id, needleConcat) {
			return e.Name, e.DeviceID, nil
		}
		if strings.Contains(id, needle1) && strings.Contains(id, needle2) {
			return e.Name, e.DeviceID, nil
		}
	}
	logger.Debugf("No matching WMI entity found for VID:%s PID:%s", vid, pid)

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
	logger.Debugf("Found ParentIdPrefix: %s for VID:%s PID:%s", prefix, vid, pid)
	prefixUpper := strings.ToUpper(prefix)

	logger.Debug("Opening registry key for USBPRINT models")
	root, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Enum\USBPRINT`,
		registry.ENUMERATE_SUB_KEYS,
	)
	if err != nil {
		logger.Errorf("Failed to open USBPRINT registry key: %v", err)
		return ""
	}
	defer root.Close()

	models, _ := root.ReadSubKeyNames(-1)
	for _, model := range models {
		modelKey, err := registry.OpenKey(root, model, registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			logger.Warnf("Failed to open registry subkey %s: %v", model, err)
			continue
		}
		instances, _ := modelKey.ReadSubKeyNames(-1)
		modelKey.Close()

		for _, instance := range instances {
			logger.Debugf("Checking USBPRINT instance %s for prefix %s", instance, prefixUpper)
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
	logger.Debugf("Opening registry key for USB device: %s", keyPath)
	devKey, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		keyPath,
		registry.ENUMERATE_SUB_KEYS,
	)
	if err != nil {
		logger.Warnf("Failed to open USB device registry key %s: %v", keyPath, err)
		return ""
	}
	defer devKey.Close()

	instances, _ := devKey.ReadSubKeyNames(-1)
	for _, instance := range instances {
		instKey, err := registry.OpenKey(devKey, instance, registry.QUERY_VALUE)
		if err != nil {
			logger.Warnf("Failed to open instance registry key %s\\%s: %v", keyPath, instance, err)
			continue
		}
		flags, _, _ := instKey.GetIntegerValue("ConfigFlags")
		prefix, _, err := instKey.GetStringValue("ParentIdPrefix")
		instKey.Close()

		if err == nil && prefix != "" && flags == 0 {
			logger.Debugf("Found ParentIdPrefix %s for instance %s\\%s", prefix, keyPath, instance)
			return prefix
		}
	}
	return ""
}

type Win32_Printer struct {
	Name        string
	DeviceID    string
	WorkOffline bool
	PortName    string
}

func listSystemPrinters() ([]SystemUsbPrinter, error) {
	var printersWMI []Win32_Printer

	// SELECT Name, DeviceID, PortName, Network, Local, Shared FROM Win32_Printer
	query := "SELECT Name, DeviceID, WorkOffline, PortName FROM Win32_Printer"
	if err := wmi.Query(query, &printersWMI); err != nil {
		return nil, err
	}

	var printers []SystemUsbPrinter
	for _, p := range printersWMI {
		if p.WorkOffline {
			continue
		}

		isLAN := strings.HasPrefix(p.Name, "PDF_NETWORK_")
		info := SystemUsbPrinter{
			Serial:   "",
			IdName:   p.Name,
			DeviceID: p.DeviceID,
			Label:    classifyPort(p.PortName),
			Type:     TypeOFFICE,
			IsLAN:    isLAN,
			IP:       util.Ternary(isLAN, strings.TrimPrefix(p.Name, "PDF_NETWORK_"), ""),
		}
		printers = append(printers, info)
	}

	return printers, nil
}

func classifyPort(portName string) string {
	port := strings.ToUpper(strings.TrimSpace(portName))

	// --- VIRTUAL ---
	if port == "PORTPROMPT:" || port == "NUL:" || port == "FILE:" {
		return "VIRTUAL"
	}

	// --- USB ---
	if strings.HasPrefix(port, "USB") {
		return "USB"
	}

	// --- NETWORK ---
	if strings.HasPrefix(port, "WSD") ||
		strings.HasPrefix(port, "IP_") ||
		strings.HasPrefix(port, "\\\\") ||
		strings.HasPrefix(port, "HTTP") ||
		strings.HasPrefix(port, "IPP") ||
		strings.HasPrefix(port, "LPR") ||
		strings.HasPrefix(port, "RAW") {
		return "NETWORK"
	}

	return ""
}

func DeleteSystemPrinter(name string) error {
	cmd := exec.Command("powershell", "-Command", "Remove-Printer -Name \""+name+"\"")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete printer %s: %v (%s)", name, err, string(out))
	}

	return nil
}

func AddLanOfficePrinter(ip string) error {
	portName := fmt.Sprintf("IP_%s", ip)
	printerName := fmt.Sprintf("PDF_NETWORK_%s", ip)

	if printerPortExists(portName) {
		return fmt.Errorf("printer port %s already exists", portName)
	}

	if printerExists(printerName) {
		return fmt.Errorf("printer %s already exists", printerName)
	}

	cmd1 := exec.Command("powershell",
		"-Command",
		fmt.Sprintf(`Add-PrinterPort -Name "%s" -PrinterHostAddress "%s"`, portName, ip),
	)

	if output, err := cmd1.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add port: %s (%v)", string(output), err)
	}

	cmd2 := exec.Command("powershell",
		"-Command",
		fmt.Sprintf(`Add-Printer -Name "%s" -DriverName "Microsoft IPP Class Driver" -PortName "%s"`, printerName, portName),
	)

	if output, err := cmd2.CombinedOutput(); err != nil {

		if rmErr := removePrinterPort(portName); rmErr != nil {
			return fmt.Errorf("failed to add printer: %s (%v); cleanup failed: %v", string(output), err, rmErr)
		}

		return fmt.Errorf("failed to add printer: %s (%v)", string(output), err)
	}

	return nil
}

func removePrinterPort(name string) error {
	cmd := exec.Command("powershell",
		"-Command",
		fmt.Sprintf(`Remove-PrinterPort -Name "%s"`, name),
	)
	return cmd.Run()
}

func printerPortExists(name string) bool {
	out, _ := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf(`Get-PrinterPort -Name "%s" -ErrorAction SilentlyContinue | Out-String`, name)).CombinedOutput()

	return strings.TrimSpace(string(out)) != ""
}

func printerExists(name string) bool {
	out, _ := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf(`Get-Printer -Name "%s" -ErrorAction SilentlyContinue | Out-String`, name)).CombinedOutput()

	return strings.TrimSpace(string(out)) != ""
}
