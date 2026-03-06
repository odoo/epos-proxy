package main

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"epos-proxy/config"
	"epos-proxy/printer"
	"epos-proxy/server"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx            context.Context
	webserver      *server.Server
	config         *config.Manager
	printerManager *printer.Manager
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cfg, err := config.NewManager()
	if err != nil {
		log.Fatalf("config init: %v", err)
	}
	if err := cfg.Load(); err != nil {
		log.Printf("config load warning: %v", err)
	}
	log.Printf("[config] loaded from %s\n", cfg.Path())
	a.config = cfg
	a.printerManager = printer.NewManager()
	port, _ := cfg.ResolvePort()
	a.webserver = server.New(port, a.printerManager)
}

func (a *App) shutdown(ctx context.Context) {
	_ = a.webserver.Stop()
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
	return fmt.Sprintf("127.0.0.1:%d/p/%s", a.webserver.Port, id)
}

func (a *App) Status() Status {
	printers := make([]Printer, 0)
	unavailablePrinters := make([]UnavailablePrinter, 0)

	printerInfos, err := printer.ListUSBPrinters()
	errorMsg := ""

	if err == nil {
		for _, info := range printerInfos.Available {
			printers = append(printers, Printer{
				Id:     info.Id,
				Name:   info.VendorName + " " + info.ProductName,
				Serial: info.Serial,
				Ip:     a.GetPrinterIp(info.Id),
				Online: true,
			})
		}

		for _, info := range printerInfos.Unavailable {
			unavailablePrinters = append(unavailablePrinters, UnavailablePrinter{
				Name:     info.Name,
				ErrorMsg: info.Error,
			})

		}
	} else {
		errorMsg = err.Error()
	}

	lanPrinters := printer.ListLANPrinters(a.config)
	for _, info := range lanPrinters {
		printers = append(printers, Printer{
			Id:     info.Id,
			Name:   fmt.Sprintf("Network - %s", info.IP),
			Ip:     a.GetPrinterIp(info.Id),
			IsLAN:  true,
			LANIp:  info.IP,
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
	ip, err := printer.ValidateIPAddress(ip)
	if err != nil {
		return err
	}

	if err := printer.CheckLANPrinter(ip); err != nil {
		return fmt.Errorf("cannot connect: %w", err)
	}

	return a.config.AddLANPrinter(ip)
}

func (a *App) ConfirmRemoveLANPrinter(ip string) (bool, error) {
	result, err := wailsruntime.MessageDialog(a.ctx, wailsruntime.MessageDialogOptions{
		Type:          wailsruntime.WarningDialog,
		Title:         "Remove Printer",
		Message:       fmt.Sprintf("Are you sure you want to remove the printer at %s?", ip),
		Buttons:       []string{"Remove", "Cancel"},
		DefaultButton: "Cancel",
		CancelButton:  "Cancel",
	})
	if err != nil {
		return false, err
	}
	if result == "Remove" {
		return true, a.config.RemoveLANPrinter(ip)
	}
	return false, nil
}

func (a *App) CheckLANPrinterStatus(ip string) bool {
	return printer.CheckLANPrinter(ip) == nil
}
