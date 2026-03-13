package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"epos-proxy/config"
	"epos-proxy/logger"
	"epos-proxy/printer"
	"epos-proxy/server"
	"epos-proxy/util"

	autostart "github.com/emersion/go-autostart"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx            context.Context
	webserver      *server.Server
	config         *config.Manager
	printerManager *printer.Manager
	allowClose     bool
	autoStart      *autostart.App
}

func NewApp() *App {
	a := &App{allowClose: false}

	a.autoStart = &autostart.App{
		Name:        "epos-proxy",
		DisplayName: "ePOS Proxy",
		Exec:        []string{os.Args[0]},
	}

	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logger.Log.Infof("Application startup")

	cfg, err := config.NewManager()
	if err != nil {
		logger.Log.Fatalf("Config initialization failed: %v", err)
	}

	if err := cfg.Load(); err != nil {
		logger.Log.Warnf("Config load warning: %v", err)
	}

	logger.Log.Debugf("Config loaded from %s", cfg.Path())

	a.config = cfg
	a.printerManager = printer.NewManager()

	port, err := cfg.ResolvePort()
	if err != nil {
		logger.Log.Warn("Unable to resolve port, using default")
	}

	logger.Log.Infof("Starting proxy server on port %d", port)

	a.webserver = server.New(port, a.printerManager)
}

func (a *App) shutdown(ctx context.Context) {
	logger.Log.Infof("Stopping proxy server")

	if err := a.webserver.Stop(); err != nil {
		logger.Log.Errorf("Server stop error: %v", err)
	}
}

type Printer struct {
	Name   string `json:"name"`
	Serial string `json:"serial"`
	Ip     string `json:"ip"`
	Id     string `json:"id"`
	IsLAN  bool   `json:"isLAN"`
	LANIp  string `json:"lanIp,omitempty"`
	Online bool   `json:"online"`
}

type UnavailablePrinter struct {
	Name     string `json:"name"`
	ErrorMsg string `json:"errorMsg"`
	IsLAN    bool   `json:"isLAN"`
	LANIp    string `json:"lanIp,omitempty"`
}

type Status struct {
	ServerRunning       bool                 `json:"serverRunning"`
	DefaultIp           string               `json:"defaultIp"`
	ErrorMsg            string               `json:"errorMsg"`
	Printers            []Printer            `json:"printers"`
	UnavailablePrinters []UnavailablePrinter `json:"unavailablePrinters"`
	Os                  string               `json:"os"`
}

func (a *App) GetPrinterIp(id string) string {
	ip := fmt.Sprintf("127.0.0.1:%d/p/%s", a.webserver.Port, id)
	logger.Log.Debugf("Generated printer endpoint: %s", ip)
	return ip
}

func (a *App) Status() Status {

	logger.Log.Debug("Collecting printer status")

	printers := make([]Printer, 0)
	unavailablePrinters := make([]UnavailablePrinter, 0)

	printerInfos, err := printer.ListUSBPrinters()
	errorMsg := ""

	if err == nil {

		logger.Log.Debugf("Detected %d available USB printers", len(printerInfos.Available))

		for _, info := range printerInfos.Available {
			printers = append(printers, Printer{
				Id:     info.Id,
				Name:   info.VendorName + " " + info.ProductName,
				Serial: info.Serial,
				Ip:     a.GetPrinterIp(info.Id),
				Online: true,
			})
			logger.Log.Infof("USB printer available: %s %s", info.VendorName, info.ProductName)
		}

		for _, info := range printerInfos.Unavailable {
			unavailablePrinters = append(unavailablePrinters, UnavailablePrinter{
				Name:     info.Name,
				ErrorMsg: info.Error,
			})

			logger.Log.Warnf("USB printer unavailable: %s (%s)", info.Name, info.Error)
		}
	} else {
		errorMsg = err.Error()
		logger.Log.Errorf("USB printer detection failed: %v", err)
	}

	lanPrinters := printer.ListLANPrinters(a.config)

	for _, info := range lanPrinters {
		printers = append(printers, Printer{
			Id:    info.Id,
			Name:  fmt.Sprintf("Network - %s", info.IP),
			Ip:    a.GetPrinterIp(info.Id),
			IsLAN: true,
			LANIp: info.IP,
		})
	}

	return Status{
		ServerRunning:       a.webserver.Running(),
		DefaultIp:           fmt.Sprintf("127.0.0.1:%d", a.webserver.Port),
		Printers:            printers,
		UnavailablePrinters: unavailablePrinters,
		ErrorMsg:            errorMsg,
		Os:                  runtime.GOOS,
	}
}

func (a *App) AddLANPrinter(ip string) error {

	logger.Log.Debugf("Adding LAN printer: %s", ip)

	ip, err := printer.ValidateIPAddress(ip)
	if err != nil {
		logger.Log.Warnf("Invalid printer IP: %s", ip)
		return err
	}

	if err := printer.CheckLANPrinter(ip); err != nil {
		logger.Log.Errorf("LAN printer unreachable: %s", ip)
		return fmt.Errorf("cannot connect: %w", err)
	}

	if err := a.config.AddLANPrinter(ip); err != nil {
		logger.Log.Errorf("Failed to save LAN printer: %s", ip)
		return err
	}

	logger.Log.Debugf("LAN printer added successfully: %s", ip)

	return nil
}

func (a *App) ConfirmRemoveLANPrinter(ip string) (bool, error) {

	logger.Log.Debugf("Remove LAN printer requested: %s", ip)

	result, err := wailsruntime.MessageDialog(a.ctx, wailsruntime.MessageDialogOptions{
		Type:          wailsruntime.QuestionDialog,
		Title:         "Remove Printer",
		Message:       fmt.Sprintf("Are you sure you want to remove the printer at %s?", ip),
		Buttons:       []string{"Cancel", "Confirm"},
		DefaultButton: "Cancel",
		CancelButton:  "Cancel",
	})
	if err != nil {
		logger.Log.Errorf("Remove printer dialog failed: %v", err)
		return false, err
	}
	if result == "Confirm" || result == "Yes" {
		logger.Log.Infof("Removing LAN printer: %s", ip)
		return true, a.config.RemoveLANPrinter(ip)
	}
	logger.Log.Infof("Remove LAN printer cancelled, Remove printer dialog result: %s", result)
	return false, nil
}

func (a *App) CheckLANPrinterStatus(ip string) bool {
	logger.Log.Debugf("Checking LAN printer status: %s", ip)
	return printer.CheckLANPrinter(ip) == nil
}

func (a *App) Quit() {
	logger.Log.Infof("Quit requested by user")
	a.allowClose = true
	wailsruntime.Quit(a.ctx)
}

func (a *App) DownloadLogs() {

	logger.Log.Debug("Download logs requested")

	configDir, _ := os.UserConfigDir()
	logDir := filepath.Join(configDir, "epos-proxy", "logs")

	logger.Log.Infof("Log directory: %s", logDir)

	home, _ := os.UserHomeDir()
	downloadDir := filepath.Join(home, "Downloads")

	logger.Log.Infof("Download directory: %s", downloadDir)

	zipName := fmt.Sprintf("epos-proxy-logs-%s.zip",
		time.Now().Format("2006-01-02"))

	zipPath := filepath.Join(downloadDir, zipName)

	logger.Log.Infof("Creating logs archive: %s", zipPath)

	err := util.ZipLogs(logDir, zipPath)
	if err != nil {

		logger.Log.Errorf("Log export failed: %v", err)

		wailsruntime.MessageDialog(a.ctx, wailsruntime.MessageDialogOptions{
			Type:    wailsruntime.ErrorDialog,
			Title:   "Download Logs Failed",
			Message: err.Error(),
		})
		return
	}

	logger.Log.Infof("Logs successfully exported to: %s", zipPath)

	wailsruntime.MessageDialog(a.ctx, wailsruntime.MessageDialogOptions{
		Type:    wailsruntime.InfoDialog,
		Title:   "Logs Downloaded",
		Message: fmt.Sprintf("Logs saved to:\n%s", zipPath),
	})
}

func (a *App) IsAutostartEnabled() bool {
	return a.autoStart.IsEnabled()
}

func (a *App) EnableAutostart() error {
	logger.Log.Info("Enabling autostart")

	if runtime.GOOS == "linux" {
		return util.EnableLinuxAutostart()
	}

	if !a.autoStart.IsEnabled() {
		return a.autoStart.Enable()
	}

	return nil
}

func (a *App) DisableAutostart() error {
	logger.Log.Info("Disabling autostart")

	if a.autoStart.IsEnabled() {
		return a.autoStart.Disable()
	}

	return nil
}
