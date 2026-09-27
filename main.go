package main

import (
	"embed"
	"immich-windows-sync/internal/singleinstance"
	"immich-windows-sync/internal/startup"
	"log"
	"os"
	"slices"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	lock := singleinstance.New()
	alreadyRunning, err := lock.Acquire()
	if err != nil {
		log.Fatal(err)
	}
	if alreadyRunning {
		return
	}

	// スタートアップ経由で起動された場合はウィンドウを出さずにタスクトレイだけで常駐する
	startHidden := slices.Contains(os.Args[1:], startup.HiddenFlag)

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err = wails.Run(&options.App{
		Title:       "Immich Windows Sync",
		Width:       1024,
		Height:      768,
		MinWidth:    1024,
		MinHeight:   768,
		StartHidden: startHidden,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnBeforeClose:    app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
