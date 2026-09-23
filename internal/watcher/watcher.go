package watcher

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/fsnotify/fsnotify"
)

type EventType int

const (
	Unknown EventType = iota
	Create
	Write
	Remove
	Rename
)

type Event struct {
	Type EventType
	Path string
}

type Watcher struct {
	// Events はファイルの作成・更新イベントを流すチャンネル（Remove/Renameは含まれない）
	Events chan Event
	Errors chan error
	// NewDirectories は監視中のディレクトリ配下に新規作成されたサブディレクトリのパスを流すチャンネル。
	// 自動では監視対象に追加せず、呼び出し側が AddDirectory を呼ぶかどうかを判断する。
	NewDirectories chan string
	cancel         context.CancelFunc
	watcher        *fsnotify.Watcher
	excludedDirs   []string
	running        bool
}

func (w *Watcher) IsRunning() bool {
	return w.running
}

func NewWatcher() (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		Events:         make(chan Event),
		Errors:         make(chan error),
		NewDirectories: make(chan string),
		cancel:         nil,
		watcher:        watcher,
	}, nil
}

// Start は targetDirs 配下を再帰的に監視対象へ登録し、イベント処理ループを開始する。
// excludedDirs に含まれるディレクトリはその配下ごと監視対象から除外する。
// Remove/Renameイベントは同期対象ではないため Events へは流さない。
//
// fsnotify.Watcherは一度Close()すると再利用できない（Addがエラーになる）ため、
// Stop() 後に再度 Start() が呼ばれた場合は内部の fsnotify.Watcher を作り直す。
func (w *Watcher) Start(targetDirs []string, excludedDirs []string) error {
	if w.watcher == nil {
		fsWatcher, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		w.watcher = fsWatcher
	}

	w.excludedDirs = excludedDirs
	for _, dir := range targetDirs {
		err := w.addRecursive(dir)
		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go func() {
		for {
			select {
			case fsnotifyEvent, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				eventType := toEventType(fsnotifyEvent)
				eventPath := fsnotifyEvent.Name
				if eventType == Remove || eventType == Rename {
					continue
				}
				pathStat, err := os.Stat(eventPath)
				if err != nil {
					continue
				}
				// 新規ディレクトリはひとまず自動で除外リストに入れる方針のため、
				// エクスプローラーの「新しいフォルダー」作成時にプレースホルダー名・確定後の名前の
				// 両方で通知されても実害はない（呼び出し側で除外リストへの追加が冪等であればよい）。
				if eventType == Create && pathStat.IsDir() {
					if !slices.Contains(w.excludedDirs, eventPath) {
						w.NewDirectories <- eventPath
					}
				} else {
					w.Events <- Event{
						Type: eventType,
						Path: eventPath,
					}
				}
			case err := <-w.watcher.Errors:
				w.Errors <- err
			case <-ctx.Done():
				return
			}
		}
	}()
	w.running = true
	return nil
}

func (w *Watcher) Stop() error {
	w.cancel()
	err := w.watcher.Close()
	if err != nil {
		return err
	}
	w.watcher = nil
	w.running = false
	return nil
}

func (w *Watcher) addRecursive(dir string) error {
	err := w.watcher.Add(dir)
	if err != nil {
		return err
	}
	err = filepath.WalkDir(dir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != dir && !slices.Contains(w.excludedDirs, path) {
			err2 := w.addRecursive(path)
			if err2 != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func toEventType(event fsnotify.Event) EventType {
	if event.Op.Has(fsnotify.Create) {
		return Create
	}
	if event.Op.Has(fsnotify.Write) {
		return Write
	}
	if event.Op.Has(fsnotify.Remove) {
		return Remove
	}
	if event.Op.Has(fsnotify.Rename) {
		return Rename
	}
	return Unknown
}
