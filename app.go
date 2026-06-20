package main

import (
	"context"
	"fmt"
	"immich-windows-sync/internal/config"
	"immich-windows-sync/internal/db"
	"immich-windows-sync/internal/immich"
	appSync "immich-windows-sync/internal/sync"
	"immich-windows-sync/internal/watcher"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx     context.Context
	cfg     *config.Config
	syncer  *appSync.Syncer
	watcher *watcher.Watcher
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	cfg, err := a.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	a.cfg = cfg

	immichClient := immich.Client{
		ServerURL: cfg.Immich.ServerURL,
		APIKey:    cfg.Immich.APIKey,
	}

	appdataDir := os.Getenv("APPDATA")
	if appdataDir == "" {
		log.Fatal(fmt.Errorf(`Environment variable "APPDATA" is empty.`))
	}
	dbFile := filepath.Join(appdataDir, "immich-sync", "syncdata.db")
	dbClient, err := db.NewClient(dbFile)
	if err != nil {
		log.Fatal(err)
	}

	a.syncer = appSync.NewSyncer(5, &immichClient, dbClient)

	a.watcher, err = watcher.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) LoadConfig() (*config.Config, error) {
	return config.Load()
}

func (a *App) SaveConfig(cfg config.Config) error {
	return config.Save(cfg)
}

func (a *App) SelectFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "フォルダを選択",
	})
}

func (a *App) StartWatcher() error {
	a.SyncNow()
	err := a.watcher.Start(a.cfg.TargetFolders)
	if err != nil {
		return err
	}
	go func() {
		for event := range a.watcher.Events {
			a.syncer.SyncAssets([]string{event.Path})
		}
	}()
	return nil
}

func (a *App) StopWatcher() error {
	err := a.watcher.Stop()
	if err != nil {
		return err
	}
	return nil
}

func (a *App) SyncNow() {
	go func() {
		files := []string{}
		for _, folder := range a.cfg.TargetFolders {
			unsyncedFile, err := a.syncer.ScanUnsyncedFiles(folder)
			if err != nil {
				log.Println(err)
			}
			files = append(files, unsyncedFile...)
		}
		a.syncer.SyncAssets(files)
	}()
}
