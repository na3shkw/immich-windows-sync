package tray

import (
	_ "embed"
	"sync"

	"github.com/getlantern/systray"
)

// Status はトレイアイコンで表示する同期状態。
type Status int

const (
	StatusSynced Status = iota
	StatusSyncing
	StatusPaused
	StatusError
)

//go:embed icons/synced.ico
var syncedIcon []byte

//go:embed icons/syncing.ico
var syncingIcon []byte

//go:embed icons/paused.ico
var pausedIcon []byte

//go:embed icons/error.ico
var errorIcon []byte

func (s Status) icon() []byte {
	switch s {
	case StatusSyncing:
		return syncingIcon
	case StatusPaused:
		return pausedIcon
	case StatusError:
		return errorIcon
	default:
		return syncedIcon
	}
}

func (s Status) tooltip() string {
	switch s {
	case StatusSyncing:
		return "Immich Windows Sync - 同期中"
	case StatusPaused:
		return "Immich Windows Sync - 一時停止中"
	case StatusError:
		return "Immich Windows Sync - エラーあり"
	default:
		return "Immich Windows Sync - 同期済み"
	}
}

var (
	mu      sync.Mutex
	ready   bool
	current = StatusSynced
)

// MarkReady は systray.Run の onReady から呼ぶ。
// systray はトレイの準備ができる前に SetIcon を呼ぶと nil ポインタ参照で panic するため、
// それまでに SetStatus された状態はここで初めて反映する。
func MarkReady() {
	mu.Lock()
	defer mu.Unlock()
	ready = true
	apply(current)
}

// SetStatus はトレイアイコンとツールチップを状態に応じて切り替える。
// 複数の goroutine から呼ばれても安全。トレイの準備前に呼んだ場合は状態だけ記録しておく。
func SetStatus(s Status) {
	mu.Lock()
	defer mu.Unlock()
	if s == current && ready {
		return
	}
	current = s
	if ready {
		apply(s)
	}
}

func apply(s Status) {
	systray.SetIcon(s.icon())
	systray.SetTooltip(s.tooltip())
}

// ResolveStatus は同期状況を示す値を受け取ってトレイに表示するアイコンを判定する。
func ResolveStatus(remainingCount, failedCount int64, watcherRunning bool) Status {
	if remainingCount > 0 {
		return StatusSyncing
	}
	if failedCount > 0 {
		return StatusError
	}
	if watcherRunning {
		return StatusSynced
	}
	return StatusPaused
}
