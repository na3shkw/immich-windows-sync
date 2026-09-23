package main

import (
	"context"
	"fmt"
	"immich-windows-sync/internal/config"
	"immich-windows-sync/internal/db"
	"immich-windows-sync/internal/immich"
	"immich-windows-sync/internal/startup"
	"immich-windows-sync/internal/synclog"
	"immich-windows-sync/internal/syncer"
	"immich-windows-sync/internal/watcher"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	cfg             *config.Config
	immichClient    *immich.Client
	syncer          *syncer.Syncer
	watcher         *watcher.Watcher
	startupRegistry *startup.Startup
	syncLog         *synclog.Logger
	syncLogPath     string
	// quitting はトレイの「終了」経由での終了かどうかを示す。
	// beforeClose がウィンドウを隠すだけにするか、本当に終了させるかの判定に使う。
	quitting bool
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

	a.immichClient = &immich.Client{
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

	a.syncLogPath = filepath.Join(appdataDir, "immich-sync", "sync.jsonl")
	a.syncLog, err = synclog.Open(a.syncLogPath)
	if err != nil {
		log.Fatal(err)
	}

	a.syncer = syncer.NewSyncer(5, a.immichClient, dbClient, a.syncLog)

	a.watcher, err = watcher.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	// StartWatcher/SaveConfigはWatcherの再起動のためにStop→Startを繰り返すことがあるが、
	// Events/NewDirectoriesのチャンネル自体はWatcherの生存期間を通じて一つだけなので、
	// それを読み出すgoroutineもアプリ起動時に一度だけ立ち上げれば十分（StartWatcherの中で
	// 毎回立ち上げるとgoroutineが際限なく増えてしまう）。
	// ファイルコピー等で同一パスに対してCreate/Writeが短時間に複数回発火することがあるため、
	// pathDebouncerで1回にまとめてから同期をトリガーする。
	debouncer := newPathDebouncer(500*time.Millisecond, func(path string) {
		a.syncer.SyncAssets([]string{path})
	})
	go func() {
		for event := range a.watcher.Events {
			debouncer.Trigger(event.Path)
		}
	}()
	go func() {
		for dir := range a.watcher.NewDirectories {
			if err := a.excludeNewSubfolder(dir); err != nil {
				log.Println(err)
			}
		}
	}()

	isDev := runtime.Environment(a.ctx).BuildType == "dev"
	a.startupRegistry = startup.NewStartup(isDev)

	go systray.Run(a.onTrayReady, a.onTrayExit)

	go func() {
		if err := a.StartWatcher(); err != nil {
			log.Println(err)
		}
	}()
}

// shutdown is called when the app terminates. It releases resources
// acquired during startup (e.g. the SQLite connection).
func (a *App) shutdown(ctx context.Context) {
	if err := a.syncer.Close(); err != nil {
		log.Println(err)
	}
	if err := a.syncLog.Close(); err != nil {
		log.Println(err)
	}
	systray.Quit()
}

// beforeClose はウィンドウを閉じる操作（閉じるボタン、または quit 経由の runtime.Quit）のたびに呼ばれる。
// quitting が立っていなければウィンドウを隠すだけにしてプロセスは常駐させ続ける。
func (a *App) beforeClose(ctx context.Context) bool {
	if a.quitting {
		return false
	}
	runtime.WindowHide(ctx)
	return true
}

// quit はトレイメニューの「終了」から呼ばれる、アプリを実際に終了させるための入り口。
func (a *App) quit() {
	a.quitting = true
	runtime.Quit(a.ctx)
}

func (a *App) onTrayReady() {
	systray.SetIcon(trayIcon)
	systray.SetTooltip("Immich Windows Sync")

	mShow := systray.AddMenuItem("開く", "ウィンドウを開く")
	mQuit := systray.AddMenuItem("終了", "アプリケーションを終了")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				runtime.WindowShow(a.ctx)
			case <-mQuit.ClickedCh:
				a.quit()
				return
			}
		}
	}()
}

func (a *App) onTrayExit() {}

func (a *App) LoadConfig() (*config.Config, error) {
	return config.Load()
}

// SaveConfig は設定をディスクへ保存し、実行中のバックエンドの状態（a.cfg・immichクライアント・
// Watcherの監視対象）にも反映させる。単にファイルへ書き込むだけだと、実行中のWatcherが
// 起動時点のフォルダ一覧を使い続けてしまい、後から追加したフォルダが監視されない不具合があったため。
func (a *App) SaveConfig(cfg config.Config) error {
	if err := config.Save(cfg); err != nil {
		return err
	}

	targetsChanged := !slices.Equal(a.cfg.TargetFolders, cfg.TargetFolders)
	excludedChanged := !slices.Equal(a.cfg.ExcludedFolders, cfg.ExcludedFolders)

	a.cfg = &cfg
	a.immichClient.ServerURL = cfg.Immich.ServerURL
	a.immichClient.APIKey = cfg.Immich.APIKey

	if (targetsChanged || excludedChanged) && a.watcher.IsRunning() {
		if err := a.watcher.Stop(); err != nil {
			return err
		}
		if err := a.StartWatcher(); err != nil {
			return err
		}
	}

	// 新規サブフォルダの自動除外など、バックエンド側の判断で設定が変わることがあるため、
	// フロントエンドに変更を伝えて画面を最新化してもらう
	runtime.EventsEmit(a.ctx, "configChanged")
	return nil
}

func (a *App) SelectFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "フォルダを選択",
	})
}

func (a *App) StartWatcher() error {
	a.SyncNow()
	err := a.watcher.Start(a.cfg.TargetFolders, a.cfg.ExcludedFolders)
	if err == nil {
		a.syncLog.Log(map[string]any{"event": "watcher_started"})
	}
	return err
}

// excludeNewSubfolderは、監視中のフォルダ配下に新規作成されたサブフォルダを除外リストに追加する。
// 新規サブフォルダはひとまず自動的に同期対象外とし、同期したい場合はFolder Management画面で
// 除外を解除してもらう方針（確認プロンプトを出す方式は、エクスプローラーでの新規フォルダ作成が
// 内部的に「プレースホルダー名で作成→ユーザーが名前を確定するとリネーム」の2段階になっており、
// タイミングによっては二重にプロンプトが出てしまう問題があったため見送った）。
// 既に除外済みの場合は何もしない（冪等）。
func (a *App) excludeNewSubfolder(path string) error {
	if slices.Contains(a.cfg.ExcludedFolders, path) {
		return nil
	}
	a.cfg.ExcludedFolders = append(a.cfg.ExcludedFolders, path)
	a.syncLog.Log(map[string]any{"event": "subfolder_auto_excluded", "path": path})
	return a.SaveConfig(*a.cfg)
}

// RemoveExcludedFolder は除外リストからフォルダを取り除く（除外解除）。
func (a *App) RemoveExcludedFolder(path string) error {
	updated := make([]string, 0, len(a.cfg.ExcludedFolders))
	for _, f := range a.cfg.ExcludedFolders {
		if f != path {
			updated = append(updated, f)
		}
	}
	a.cfg.ExcludedFolders = updated
	return a.SaveConfig(*a.cfg)
}

func (a *App) IsWatcherRunning() bool {
	return a.watcher.IsRunning()
}

func (a *App) StopWatcher() error {
	err := a.watcher.Stop()
	if err != nil {
		return err
	}
	a.syncLog.Log(map[string]any{"event": "watcher_stopped"})
	return nil
}

// GetSyncSummary はstatusごとの同期済みアセット件数を返す。
func (a *App) GetSyncSummary() (map[string]int64, error) {
	return a.syncer.CountByStatus()
}

// FailedAsset はフロントエンドへ渡す用の失敗アセット情報。
type FailedAsset struct {
	Path        string    `json:"path"`
	Reason      string    `json:"reason"`
	FailedCount int64     `json:"failedCount"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (a *App) GetFailedAssets() ([]FailedAsset, error) {
	assets, err := a.syncer.FailedAssets()
	if err != nil {
		return nil, err
	}
	result := make([]FailedAsset, 0, len(assets))
	for _, asset := range assets {
		result = append(result, FailedAsset{
			Path:        asset.Path,
			Reason:      asset.LatestFailedReason.String,
			FailedCount: asset.FailedCount,
			UpdatedAt:   asset.UpdatedAt,
		})
	}
	return result, nil
}

// GetRecentLogLines は永続化された同期ログファイルの末尾n行を返す。
func (a *App) GetRecentLogLines(n int) ([]string, error) {
	return synclog.ReadRecent(a.syncLogPath, n)
}

// OpenLogFolder は同期ログファイルが置かれているフォルダをエクスプローラーで開く。
// explorer.exeは正常終了時でも非0の終了コードを返すことがあるため、Startのみで完了を待たない。
func (a *App) OpenLogFolder() error {
	return exec.Command("explorer", filepath.Dir(a.syncLogPath)).Start()
}

func (a *App) SyncNow() {
	go func() {
		files := []string{}
		for _, folder := range a.cfg.TargetFolders {
			unsyncedFile, err := a.syncer.ScanUnsyncedFiles(folder, a.cfg.ExcludedFolders)
			if err != nil {
				log.Println(err)
			}
			files = append(files, unsyncedFile...)
		}
		a.syncer.SyncAssets(files)
	}()
}

func (a *App) RegisterStartup() error {
	err := a.startupRegistry.Register()
	if err != nil {
		return err
	}
	return nil
}

func (a *App) UnRegisterStartup() error {
	err := a.startupRegistry.UnRegister()
	if err != nil {
		return err
	}
	return nil
}

func (a *App) IsStartupRegistered() (bool, error) {
	result, err := a.startupRegistry.IsRegistered()
	if err != nil {
		return false, err
	}
	return result, nil
}
