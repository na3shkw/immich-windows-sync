package db

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T) *Client {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "test.db")
	client, err := NewClient(dbFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		client.Close()
	})
	return client
}

// NewClient: 同じDBファイルに対して複数回呼び出してもエラーにならないこと（CREATE TABLE IF NOT EXISTS）を確認する
func TestNewClient_Idempotent(t *testing.T) {
	dbFile := filepath.Join(t.TempDir(), "test.db")

	client, err := NewClient(dbFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		client.Close()
	})

	client2, err := NewClient(dbFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		client2.Close()
	})
}

// MarkAsSyncing: レコードが存在しない場合はINSERT、存在する場合はUPDATEされることを確認する
func TestMarkAsSyncing(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T)
	}{
		{
			name: "Create record if not exists",
			setup: func(t *testing.T) {
			},
		},
		{
			name: "Update record if exists",
			setup: func(t *testing.T) {
				client := newTestClient(t)
				path := "C:/photos/a.jpg"
				err := client.MarkAsSyncing(path)
				require.NoError(t, err)
				err = client.MarkAsSuccess(path, "immich-id-1")
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			client := newTestClient(t)
			path := "C:/photos/a.jpg"
			client.MarkAsSyncing(path)

			asset, err := client.FindByPath(path)
			require.NoError(t, err)
			require.NotNil(t, asset)
			assert.Equal(t, "syncing", asset.Status)
		})
	}
}

func TestMarkAsSuccess(t *testing.T) {
	client := newTestClient(t)
	path := "C:/photos/a.jpg"

	require.NoError(t, client.MarkAsSyncing(path))
	require.NoError(t, client.MarkAsSuccess(path, "immich-id-1"))

	asset, err := client.FindByPath(path)
	require.NoError(t, err)
	require.NotNil(t, asset)
	assert.Equal(t, "success", asset.Status)
	assert.Equal(t, "immich-id-1", asset.ImmichID.String)
}

func TestMarkAsFailed(t *testing.T) {
	client := newTestClient(t)
	path := "C:/photos/b.jpg"

	require.NoError(t, client.MarkAsSyncing(path))
	require.NoError(t, client.MarkAsFailed(path, "network error"))
	require.NoError(t, client.MarkAsFailed(path, "network error again"))

	asset, err := client.FindByPath(path)
	require.NoError(t, err)
	require.NotNil(t, asset)
	assert.Equal(t, "failed", asset.Status)
	assert.Equal(t, int64(2), asset.FailedCount)
	assert.Equal(t, "network error again", asset.LatestFailedReason.String)
}

func TestFindByPath_NotFound(t *testing.T) {
	client := newTestClient(t)

	asset, err := client.FindByPath("C:/photos/missing.jpg")

	require.NoError(t, err)
	assert.Nil(t, asset)
}

func TestSearchByStatus(t *testing.T) {
	client := newTestClient(t)

	require.NoError(t, client.MarkAsSyncing("C:/photos/syncing.jpg"))
	require.NoError(t, client.MarkAsSyncing("C:/photos/success.jpg"))
	require.NoError(t, client.MarkAsSuccess("C:/photos/success.jpg", "immich-id"))

	assets, err := client.SearchByStatus("syncing")

	require.NoError(t, err)
	require.Len(t, assets, 1)
	assert.Equal(t, "C:/photos/syncing.jpg", assets[0].Path)
}
