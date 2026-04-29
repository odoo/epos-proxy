//go:build windows

package util

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	sumatraPath string
	mu          sync.Mutex
)

func GetSumatraPDFPath() (string, error) {
	mu.Lock()
	defer mu.Unlock()

	if sumatraPath != "" {
		return sumatraPath, nil
	}

	path, err := findInstalledSumatra()
	if err != nil {
		return "", err
	}

	sumatraPath = path
	return sumatraPath, nil
}

func findInstalledSumatra() (string, error) {
	exePath := filepath.Join(getInstallDir(), "SumatraPDF", "SumatraPDF.exe")
	if _, err := os.Stat(exePath); err == nil {
		return exePath, nil
	}

	return "", fmt.Errorf("SumatraPDF not found at %s", exePath)
}

func getInstallDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}

	exe, err = filepath.EvalSymlinks(exe) // resolve all symlinks to get the actual executable path
	if err != nil {
		return filepath.Dir(exe)
	}

	return filepath.Dir(exe)
}
