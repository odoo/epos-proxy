// EnableLinuxAutostart creates the XDG autostart entry manually.
//
// We intentionally override the behavior of the go-autostart library on Linux.
// The library generates a `.desktop` file where the Exec path is wrapped in quotes,
// for example:
//
//    Exec="/home/user/app"
//
// Many desktop environments (GNOME/XDG autostart) do not correctly execute quoted
// paths in `.desktop` files, causing the application to fail silently at login.
//
// To avoid this issue we manually create the `.desktop` file with an unquoted
// Exec path:
//
//    Exec=/home/user/app
//
// This ensures reliable autostart behavior across GNOME-based Linux systems.

package util

import (
	"fmt"
	"os"
	"path/filepath"

	"epos-proxy/logger"
)

func EnableLinuxAutostart() error {
	exe, err := os.Executable()
	if err != nil {
		logger.Log.Errorf("Failed to get executable path: %v", err)
		return err
	}

	logger.Log.Infof("Executable path detected: %s", exe)

	dir := filepath.Join(os.Getenv("HOME"), ".config", "autostart")
	logger.Log.Infof("Ensuring autostart directory exists: %s", dir)

	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Log.Errorf("Failed to create autostart directory: %v", err)
		return err
	}

	file := filepath.Join(dir, "epos-proxy.desktop")
	logger.Log.Infof("Creating autostart file: %s", file)

	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=ePOS Proxy
Exec=%s
Terminal=false
StartupNotify=false
X-GNOME-Autostart-enabled=true
`, exe)

	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		logger.Log.Errorf("Failed to write autostart file: %v", err)
		return err
	}

	logger.Log.Info("Linux autostart successfully enabled")
	return nil
}
