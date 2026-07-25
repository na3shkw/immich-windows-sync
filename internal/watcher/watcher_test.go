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

func eventTypes(events []Event) []EventType {
	types := make([]EventType, len(events))
	for i, e := range events {
		types[i] = e.Type
	}
	return types
}

// Start: 実際のファイル操作（作成・更新・削除）に応じて、対応するEventが
// Events チャネルに流れてくることを確認する
func TestWatcher_Start(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "photo.jpg")

	w, err := NewWatcher()
	require.NoError(t, err)
	require.NoError(t, w.Start([]string{dir}))
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

	require.NoError(t, os.Remove(filePath))
	events = drainEvents(t, w.Events, 300*time.Millisecond)
	assert.Contains(t, eventTypes(events), Remove)
	for _, e := range events {
		assert.Equal(t, filePath, e.Path)
	}
}
