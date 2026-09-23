package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// toEventType: fsnotify.Op から独自のEventTypeへのマッピングを確認する
func TestToEventType(t *testing.T) {
	tests := []struct {
		name string
		op   fsnotify.Op
		want EventType
	}{
		{"Create", fsnotify.Create, Create},
		{"Write", fsnotify.Write, Write},
		{"Remove", fsnotify.Remove, Remove},
		{"Rename", fsnotify.Rename, Rename},
		{"Chmod (未対応のOpはUnknown)", fsnotify.Chmod, Unknown},
		{"Create bitがWrite bitより優先される", fsnotify.Create | fsnotify.Write, Create},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toEventType(fsnotify.Event{Op: tt.op})
			assert.Equal(t, tt.want, got)
		})
	}
}

// w.Events からイベントを、idleTimeout以内に次が来なくなるまでまとめて回収する。
// fsnotifyは1回のファイル操作に対して複数のイベントを発火させることがあるため
// （例: 新規書き込みで CREATE と WRITE の2件など環境依存で変動する）、
// 1件だけを待つのではなくバースト単位で回収してから検証する。
func drainEvents(t *testing.T, ch <-chan Event, idleTimeout time.Duration) []Event {
	t.Helper()
	var events []Event
	for {
		select {
		case event := <-ch:
			events = append(events, event)
		case <-time.After(idleTimeout):
			if len(events) == 0 {
				t.Fatal("event did not arrive within timeout")
			}
			return events
		}
	}
}

// timeout以内に ch から何も届かないことを確認する
func assertNoEvent[T any](t *testing.T, ch <-chan T, timeout time.Duration) {
	t.Helper()
	select {
	case v := <-ch:
		t.Fatalf("expected no event, but got: %+v", v)
	case <-time.After(timeout):
	}
}

func eventTypes(events []Event) []EventType {
	types := make([]EventType, len(events))
	for i, e := range events {
		types[i] = e.Type
	}
	return types
}

// Start: 実際のファイル操作（作成・更新）に応じて、対応するEventが
// Events チャネルに流れてくることを確認する
func TestWatcher_Start(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "photo.jpg")

	w, err := NewWatcher()
	require.NoError(t, err)
	require.NoError(t, w.Start([]string{dir}, nil))
	t.Cleanup(func() {
		w.Stop()
	})

	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0644))
	events := drainEvents(t, w.Events, 300*time.Millisecond)
	assert.Contains(t, eventTypes(events), Create)
	for _, e := range events {
		assert.Equal(t, filePath, e.Path)
	}

	require.NoError(t, os.WriteFile(filePath, []byte("updated"), 0644))
	events = drainEvents(t, w.Events, 300*time.Millisecond)
	assert.Contains(t, eventTypes(events), Write)
	for _, e := range events {
		assert.Equal(t, filePath, e.Path)
	}
}

// Stop後にStartを呼び直しても再度監視できることを確認する
// （fsnotify.Watcherは一度Close()すると再利用できないため、内部で作り直す必要がある）
func TestWatcher_StopThenStart(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "photo.jpg")

	w, err := NewWatcher()
	require.NoError(t, err)
	require.NoError(t, w.Start([]string{dir}, nil))
	require.NoError(t, w.Stop())

	require.NoError(t, w.Start([]string{dir}, nil))
	t.Cleanup(func() {
		w.Stop()
	})

	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0644))
	events := drainEvents(t, w.Events, 300*time.Millisecond)
	assert.Contains(t, eventTypes(events), Create)
}

// Start: Remove/Renameイベントは同期対象ではないため Events へ流れてこないことを確認する
// （存在しないファイルへの同期試行が失敗し続け、DBにfailedレコードが残り続けるバグの修正）
func TestWatcher_Start_RemoveIsNotSynced(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "photo.jpg")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0644))

	w, err := NewWatcher()
	require.NoError(t, err)
	require.NoError(t, w.Start([]string{dir}, nil))
	t.Cleanup(func() {
		w.Stop()
	})

	require.NoError(t, os.Remove(filePath))
	assertNoEvent(t, w.Events, 300*time.Millisecond)
}

// Start: targetDirs配下に既に存在するサブディレクトリも再帰的に監視対象へ登録されることを確認する
func TestWatcher_Start_RecursivelyWatchesExistingSubdirectories(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(subDir, 0755))
	filePath := filepath.Join(subDir, "photo.jpg")

	w, err := NewWatcher()
	require.NoError(t, err)
	require.NoError(t, w.Start([]string{dir}, nil))
	t.Cleanup(func() {
		w.Stop()
	})

	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0644))
	events := drainEvents(t, w.Events, 300*time.Millisecond)
	assert.Contains(t, eventTypes(events), Create)
	for _, e := range events {
		assert.Equal(t, filePath, e.Path)
	}
}

// Start: excludedDirsに指定したサブディレクトリは監視対象に登録されないことを確認する
func TestWatcher_Start_ExcludedSubdirectoryIsNotWatched(t *testing.T) {
	dir := t.TempDir()
	excludedDir := filepath.Join(dir, "excluded")
	require.NoError(t, os.Mkdir(excludedDir, 0755))
	filePath := filepath.Join(excludedDir, "photo.jpg")

	w, err := NewWatcher()
	require.NoError(t, err)
	require.NoError(t, w.Start([]string{dir}, []string{excludedDir}))
	t.Cleanup(func() {
		w.Stop()
	})

	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0644))
	assertNoEvent(t, w.Events, 300*time.Millisecond)
}

// Start: 監視中のディレクトリ配下に新規作成されたサブディレクトリはNewDirectoriesへ通知され、
// まだ監視対象には自動追加されないことを確認する
func TestWatcher_Start_NotifiesNewSubdirectory(t *testing.T) {
	dir := t.TempDir()
	newSubDir := filepath.Join(dir, "new-sub")

	w, err := NewWatcher()
	require.NoError(t, err)
	require.NoError(t, w.Start([]string{dir}, nil))
	t.Cleanup(func() {
		w.Stop()
	})

	require.NoError(t, os.Mkdir(newSubDir, 0755))

	select {
	case got := <-w.NewDirectories:
		assert.Equal(t, newSubDir, got)
	case <-time.After(2 * time.Second):
		t.Fatal("NewDirectories did not receive the new subdirectory within timeout")
	}

	// まだ監視対象に追加されていないので、配下のファイル作成はEventsに流れてこない
	require.NoError(t, os.WriteFile(filepath.Join(newSubDir, "photo.jpg"), []byte("data"), 0644))
	assertNoEvent(t, w.Events, 300*time.Millisecond)
}
