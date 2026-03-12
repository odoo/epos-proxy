/*
#cgo darwin CFLAGS:  -I/opt/homebrew/opt/libusb/include/libusb-1.0
#cgo darwin LDFLAGS: /opt/homebrew/opt/libusb/lib/libusb-1.0.a -framework IOKit -framework CoreFoundation
#include <libusb-1.0/libusb.h>
*/
package main

import (
	"C"
	"context"
	"embed"
	"epos-proxy/logger"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger.InitLogger()
	logger.Log.Infof("Starting ePOS Proxy")

	app := NewApp()

	windowStartState := options.Normal
	for _, arg := range os.Args[1:] {
		if arg == "--minimized" {
			logger.Log.Infof("Application started with --minimized flag")
			windowStartState = options.Minimised
			break
		}
	}

	err := wails.Run(&options.App{
		Title:                    "ePOS Proxy",
		Width:                    800,
		Height:                   600,
		Menu:                     createMenu(app),
		EnableDefaultContextMenu: true,
		WindowStartState:         windowStartState,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "epos-proxy-single-instance",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				logger.Log.Warn("Second instance detected, focusing existing window")
				wailsruntime.WindowShow(app.ctx)
				wailsruntime.WindowUnminimise(app.ctx)
			},
		},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if !app.allowClose {
				logger.Log.Infof("Close requested, hiding window instead of quitting")
				wailsruntime.WindowHide(ctx)
				return true
			}
			logger.Log.Infof("Application closing")
			return false
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		logger.Log.Error("Application crashed: ", err)
		println("Error:", err.Error())
	}

}
