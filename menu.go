package main

import (
	"github.com/wailsapp/wails/v2/pkg/menu"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func createMenu(app *App) *menu.Menu {
	mainMenu := menu.NewMenu()
	appMenu := mainMenu.AddSubmenu("App")

	appMenu.AddText("Quit", nil, func(_ *menu.CallbackData) {
		handleQuit(app)
	})

	return mainMenu
}

func handleQuit(app *App) {
	result, err := wailsruntime.MessageDialog(app.ctx, wailsruntime.MessageDialogOptions{
		Type:          wailsruntime.QuestionDialog,
		Title:         "Quit ePOS Proxy",
		Message:       "Stopping the proxy will prevent POS from printing receipts.\n\nAre you sure you want to quit?",
		Buttons:       []string{"Cancel", "Quit"},
		DefaultButton: "Cancel",
	})

	if err != nil {
		return
	}

	if result != "Yes" && result != "Quit" {
		return
	}

	app.Quit()
}
