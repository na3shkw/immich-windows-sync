package db

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T) *Client {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "test", "test.db")
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
				err = client.MarkAsSuccess(path, "immich-id-1", "created")
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
	require.NoError(t, client.MarkAsSuccess(path, "immich-id-1", "duplicate"))

	asset, err := client.FindByPath(path)
	require.NoError(t, err)
	require.NotNil(t, asset)
	assert.Equal(t, "success", asset.Status)
	assert.Equal(t, "immich-id-1", asset.ImmichID.String)
	assert.Equal(t, "duplicate", asset.ImmichStatus.String)
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

// created_at / updated_at: 状態を更新するメソッドの実行後も、UTCの "YYYY-MM-DD HH:MM:SS" 形式で保存されていることを確認する
// 業務上のカラムの値は TestMarkAs* が検証するため、ここでは時刻のカラムの保存形式だけを見る
// updated_at は SQL 内で DATETIME('now') を指定して更新する（AGENTS.md、ADR 0008）
// time.Now() を渡すとローカル時刻の文字列（"+0900 JST m=+..." など）で保存され、UTCと混在するため、その退行を検知する
func TestTimestamps_StoredAsUTC(t *testing.T) {
	const layout = "2006-01-02 15:04:05"
	const oldValue = "2000-01-01 00:00:00"

	tests := []struct {
		name string
		mark func(c *Client, path string) error
	}{
		{
			name: "MarkAsSyncing",
			mark: func(c *Client, path string) error { return c.MarkAsSyncing(path) },
		},
		{
			name: "MarkAsSuccess",
			mark: func(c *Client, path string) error { return c.MarkAsSuccess(path, "immich-id-1", "created") },
		},
		{
			name: "MarkAsFailed",
			mark: func(c *Client, path string) error { return c.MarkAsFailed(path, "network error") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t)
			path := "C:/photos/a.jpg"
			require.NoError(t, client.MarkAsSyncing(path))
			// updated_at が更新されたことが分かるよう、古い値に置き換えておく
			_, err := client.db.Exec(`UPDATE assets SET updated_at = ? WHERE path = ?`, oldValue, path)
			require.NoError(t, err)

			require.NoError(t, tt.mark(client, path))

			var createdAt, updatedAt string
			require.NoError(t, client.db.QueryRow(
				`SELECT CAST(created_at AS TEXT), CAST(updated_at AS TEXT) FROM assets WHERE path = ?`, path,
			).Scan(&createdAt, &updatedAt))
			for column, raw := range map[string]string{"created_at": createdAt, "updated_at": updatedAt} {
				got, err := time.Parse(layout, raw)
				require.NoError(t, err, "%s: %q", column, raw)
				assert.WithinDuration(t, time.Now().UTC(), got, 5*time.Second, column)
			}
		})
	}
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
	require.NoError(t, client.MarkAsSuccess("C:/photos/success.jpg", "immich-id", "created"))

	assets, err := client.SearchByStatus("syncing")

	require.NoError(t, err)
	require.Len(t, assets, 1)
	assert.Equal(t, "C:/photos/syncing.jpg", assets[0].Path)
}

func TestCountByStatus(t *testing.T) {
	client := newTestClient(t)

	require.NoError(t, client.MarkAsSyncing("C:/photos/syncing.jpg"))
	require.NoError(t, client.MarkAsSyncing("C:/photos/success1.jpg"))
	require.NoError(t, client.MarkAsSuccess("C:/photos/success1.jpg", "immich-id-1", "created"))
	require.NoError(t, client.MarkAsSyncing("C:/photos/success2.jpg"))
	require.NoError(t, client.MarkAsSuccess("C:/photos/success2.jpg", "immich-id-2", "created"))
	require.NoError(t, client.MarkAsSyncing("C:/photos/failed.jpg"))
	require.NoError(t, client.MarkAsFailed("C:/photos/failed.jpg", "network error"))

	counts, err := client.CountByStatus()

	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"success": 2, "syncing": 1, "failed": 1}, counts)
}

func TestDeleteByPath(t *testing.T) {
	client := newTestClient(t)
	path := "C:/photos/c.jpg"

	require.NoError(t, client.MarkAsSyncing(path))
	require.NoError(t, client.DeleteByPath(path))

	asset, err := client.FindByPath(path)
	require.NoError(t, err)
	assert.Nil(t, asset)

	// 存在しないパスの削除はエラーにならない
	require.NoError(t, client.DeleteByPath(path))
}
