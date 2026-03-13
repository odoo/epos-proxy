package main

import (
	"epos-proxy/logger"

	"github.com/wailsapp/wails/v2/pkg/menu"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func createMenu(app *App) *menu.Menu {
	mainMenu := menu.NewMenu()
	appMenu := mainMenu.AddSubmenu("App")

	appMenu.AddText("Download Logs", nil, func(_ *menu.CallbackData) {
		app.DownloadLogs()
	})

	appMenu.AddCheckbox("Auto Start", app.IsAutostartEnabled(), nil, func(cb *menu.CallbackData) {
		handleAutoStartToggle(app, cb)
	})

	appMenu.AddText("Quit", nil, func(_ *menu.CallbackData) {
		handleQuit(app)
	})

	return mainMenu
}

func handleAutoStartToggle(app *App, cb *menu.CallbackData) {
	checked := cb.MenuItem.Checked

	logger.Log.Infof("Auto Start toggled: %v", checked)

	if checked {
		if err := app.EnableAutostart(); err != nil {
			logger.Log.Errorf("Failed to enable autostart: %v", err)
		}
		return
	}

	if err := app.DisableAutostart(); err != nil {
		logger.Log.Errorf("Failed to disable autostart: %v", err)
	}
}

func handleQuit(app *App) {
	logger.Log.Debug("Quit menu item selected")

	result, err := wailsruntime.MessageDialog(app.ctx, wailsruntime.MessageDialogOptions{
		Type:          wailsruntime.QuestionDialog,
		Title:         "Quit ePOS Proxy",
		Message:       "Stopping the proxy will prevent POS from printing receipts.\n\nAre you sure you want to quit?",
		Buttons:       []string{"Cancel", "Quit"},
		DefaultButton: "Cancel",
	})

	if err != nil {
		logger.Log.Errorf("Failed to show quit dialog: %v", err)
		return
	}

	// linux doesnot use Buttons ovverrides and uses No | Yes for quetion dialog
	if result != "Yes" && result != "Quit" {
		return
	}

	logger.Log.Debug("Confirmed quit action")
	app.Quit()
}
