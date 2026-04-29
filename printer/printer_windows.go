//go:build windows

package printer

import (
	"context"
	"epos-proxy/logger"
	"epos-proxy/util"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func (p *Printer) printViaSystemPrinter(data []byte) error {
	logger.Debugf("Printing via Windows: %s", p.idToString())

	file, err := os.CreateTemp("", "print-*.pdf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	defer os.Remove(file.Name())
	tmpFile := file.Name()

	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("failed to write PDF: %w", err)
	}

	file.Close()

	sumatraPath, err := util.GetSumatraPDFPath()
	if err != nil {
		return fmt.Errorf("SumatraPDF not available: %w", err)
	}

	logger.Debugf("Using SumatraPDF at: %s", sumatraPath)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	args := []string{"-print-to", p.idName, "-silent"}

	if p.PrintType == DUPLEX {
		args = append(args, "-print-settings", "duplex,long-edge")
	}
	args = append(args, tmpFile)

	cmd := exec.CommandContext(ctx, sumatraPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("print timeout")
		}
		return fmt.Errorf("sumatra print failed: %w", err)
	}

	logger.Debugf("Successfully sent to printer %s", p.idToString())
	return nil
}

func (p *Printer) ensureSystemPrinterOpen() error {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf(`
            $p = Get-Printer -Name '%s' -ErrorAction SilentlyContinue
            if ($p -eq $null) { 
                Write-Output "NOT_FOUND"
                exit 1 
            }
            if ($p.PrinterStatus -eq 1) { 
                Write-Output "OFFLINE" 
            } elseif ($p.WorkOffline -eq $true) { 
                Write-Output "WORK_OFFLINE" 
            } else { 
                Write-Output "READY" 
            }
        `, p.idName))

	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("printer check failed: %w", err)
	}

	status := strings.TrimSpace(string(output))
	logger.Debugf("Printer %s status: %s", p.idName, status)

	switch status {
	case "READY":
		return nil
	case "NOT_FOUND":
		return fmt.Errorf("printer not found: %s", p.idName)
	case "OFFLINE", "WORK_OFFLINE":
		return fmt.Errorf("printer is offline: %s", p.idName)
	default:
		return fmt.Errorf("printer unavailable (status: %s)", status)
	}
}

func (p *Printer) fetchSystemPrinterName() error {
	// on windows we are not listing serial number for system printers so PrinterID must have Printer name (idName)
	return fmt.Errorf("No printer Name (IdName) set for this printer %v", p.id)
}
