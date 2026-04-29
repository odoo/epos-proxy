//go:build !windows

package printer

import (
	"epos-proxy/logger"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (p *Printer) printViaSystemPrinter(data []byte) error {
	logger.Debugf("Printing via office printer (CUPS): %s", p.idToString())

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("print-%d.pdf", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp PDF: %w", err)
	}
	defer os.Remove(tmpFile)

	cmd := exec.Command("lp", "-d", p.idName, tmpFile)
	logger.Debugf("Executing CUPS print command: %v", cmd.Args)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("CUPS print failed: %w", err)
	}

	logger.Debugf("Successfully sent to office printer %s", p.idToString())
	return nil
}

func (p *Printer) ensureSystemPrinterOpen() error {
	out, err := exec.Command("lpstat", "-p", p.idName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get printer status: %v (%s) printer: %v", err, string(out), p)
	}

	status := string(out)
	if strings.Contains(status, "disabled") || strings.Contains(status, "stopped") {
		return fmt.Errorf("office printer %s is unavailable: %s", p.idName, status)
	}

	logger.Debugf("Office printer %s is available: %s", p.idName, status)
	return nil
}

func (p *Printer) fetchSystemPrinterName() error {
	systemUsbPrinters, err := listSystemPrinters()
	if err != nil {
		return fmt.Errorf("failed to list system printers: %w", err)
	}

	for _, sp := range systemUsbPrinters {
		if sp.Serial == p.id.Serial {
			p.idName = sp.IdName
			return nil
		}
	}

	return fmt.Errorf("Cups name not found for printer %v (May be disconnected)", p.id)
}
