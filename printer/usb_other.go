//go:build !windows

package printer

import (
	"epos-proxy/logger"
	"epos-proxy/util"
	"fmt"
	"os/exec"
	"strings"
)

func getPrinterFriendlyName(vid, pid string) string {
	return fmt.Sprintf("VID:%s PID:%s", vid, pid)
}

func listSystemPrinters() ([]SystemUsbPrinter, error) {
	var printers []SystemUsbPrinter
	out, err := exec.Command("lpstat", "-v").Output()

	if err != nil {
		// Check if it's exit status 1 (no printers configured)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return []SystemUsbPrinter{}, nil
			}
		}
		return nil, err
	}

	statusMap, err := getSystemPrinterStatusMap()
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "device for ") {
			continue
		}

		lineName := strings.TrimPrefix(line, "device for ")
		name, uri, found := strings.Cut(lineName, ":")
		if !found {
			logger.Warnf("Invalid line format, skipping: %s", line)
			continue
		}

		name = strings.TrimSpace(name)
		if !strings.Contains(statusMap[name], "enabled") {
			continue
		}

		uri = strings.TrimSpace(uri)

		label := ""
		if strings.HasPrefix(uri, "ipp") {
			label = "NETWORK"
		} else if strings.HasPrefix(uri, "usb") {
			label = "USB"
		} else if strings.HasPrefix(uri, "cups-pdf:/") {
			// strings.HasPrefix(uri, "implicitclass://") ||
			label = "VIRTUAL"
		} else {
			continue
		}

		data := parseUSBURI(uri)
		isLAN := strings.HasPrefix(name, "PDF_NETWORK_")

		printers = append(printers, SystemUsbPrinter{
			Serial:  data.Serial,
			IdName:  name,
			CupsUri: uri,
			Label:   label,
			Type:    getPrinterTypeFromCupsURI(uri),
			IsLAN:   isLAN,
			IP:      util.Ternary(isLAN, strings.TrimPrefix(name, "PDF_NETWORK_"), ""),
		})
	}

	return printers, nil
}

func getSystemPrinterStatusMap() (map[string]string, error) {
	out, err := exec.Command("lpstat", "-p").Output()
	if err != nil {
		return nil, err
	}

	statusMap := make(map[string]string)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "printer ") {
			continue
		}

		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}

		name := parts[1]
		status := parts[2]
		statusMap[name] = status
	}

	return statusMap, nil
}

func AddLanOfficePrinter(ip string) error {
	printerName := fmt.Sprintf("PDF_NETWORK_%s", ip)
	uri := fmt.Sprintf("ipp://%s/ipp/print", ip)

	cmd := exec.Command("lpadmin", "-p", printerName, "-E", "-v", uri, "-m", "everywhere")
	if err := cmd.Run(); err != nil {
		if printerExists(printerName) {
			if rmErr := DeleteSystemPrinter(printerName); rmErr != nil {
				return fmt.Errorf("lpadmin failed: %v; cleanup also failed: %v", err, rmErr)
			}
		}
		return fmt.Errorf("failed to add CUPS printer: %w", err)
	}

	return nil
}

func DeleteSystemPrinter(name string) error {
	cmd := exec.Command("lpadmin", "-x", name)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete printer %s: %v (%s)", name, err, string(out))
	}

	return nil
}

func printerExists(name string) bool {
	return exec.Command("lpstat", "-p", name).Run() == nil
}

func getPrinterTypeFromCupsURI(uri string) PrinterType {
	if strings.HasPrefix(uri, "usb://") {
		return TypeANY
	}
	return TypeOFFICE
}

type USBInfo struct {
	Serial string
}

func parseUSBURI(uri string) USBInfo {
	var info USBInfo
	var query string
	uri = strings.TrimPrefix(uri, "usb://")
	if idx := strings.Index(uri, "?"); idx != -1 {
		query = uri[idx+1:]
		uri = uri[:idx]
	}

	if query != "" {
		for _, q := range strings.Split(query, "&") {
			if strings.HasPrefix(q, "serial=") {
				info.Serial = strings.TrimPrefix(q, "serial=")
			}
		}
	}

	return info
}
