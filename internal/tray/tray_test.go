package tray

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveStatus(t *testing.T) {
	tests := []struct {
		name           string
		activeSyncs    int64
		failedCount    int64
		watcherRunning bool
		want           Status
	}{
		{"Syncing", 10, 0, true, StatusSyncing},
		{"Synced", 0, 0, true, StatusSynced},
		{"Paused", 0, 0, false, StatusPaused},
		{"Error", 0, 1, true, StatusError},
		{"Error(Pausedより優先)", 0, 1, false, StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveStatus(tt.activeSyncs, tt.failedCount, tt.watcherRunning)
			assert.Equal(t, tt.want, got)
		})
	}
}
