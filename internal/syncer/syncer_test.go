package syncer

import (
	"immich-windows-sync/internal/db"
	"immich-windows-sync/internal/immich"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDBClient(t *testing.T) *db.Client {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "test.db")
	client, err := db.NewClient(dbFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		client.Close()
	})
	return client
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte("dummy"), 0644))
}

// ScanUnsyncedFiles: 対象拡張子でフィルタリングし、DB上でsuccess済みのファイルを除外して返すことを確認する
func TestScanUnsyncedFiles(t *testing.T) {
	targetDir := t.TempDir()
	unsyncedJPG := filepath.Join(targetDir, "photo1.jpg")
	syncedPNG := filepath.Join(targetDir, "photo2.png")
	nonTargetTXT := filepath.Join(targetDir, "doc.txt")
	unsyncedUppercaseJPG := filepath.Join(targetDir, "sub", "photo3.JPG")

	writeTestFile(t, unsyncedJPG)
	writeTestFile(t, syncedPNG)
	writeTestFile(t, nonTargetTXT)
	writeTestFile(t, unsyncedUppercaseJPG)

	dbClient := newTestDBClient(t)
	require.NoError(t, dbClient.MarkAsSyncing(syncedPNG))
	require.NoError(t, dbClient.MarkAsSuccess(syncedPNG, "immich-id"))

	s := NewSyncer(1, &immich.Client{}, dbClient)
	files, err := s.ScanUnsyncedFiles(targetDir)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{unsyncedJPG, unsyncedUppercaseJPG}, files)
}

// SyncAssets: 複数ファイルをワーカープールで並行アップロードし、
// 成功・失敗それぞれがDBに正しく記録されることを確認する
func TestSyncAssets(t *testing.T) {
	targetDir := t.TempDir()
	successPath1 := filepath.Join(targetDir, "ok1.jpg")
	successPath2 := filepath.Join(targetDir, "ok2.jpg")
	failPath := filepath.Join(targetDir, "fail1.jpg")
	writeTestFile(t, successPath1)
	writeTestFile(t, successPath2)
	writeTestFile(t, failPath)

	// deviceAssetIdフィールド（=ファイルパス）に"fail"が含まれていたらアップロード失敗、
	// それ以外は成功として扱うモックサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(10<<20))
		deviceAssetID := r.MultipartForm.Value["deviceAssetId"][0]

		if filepath.Base(deviceAssetID) == filepath.Base(failPath) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"immich-id-` + filepath.Base(deviceAssetID) + `","status":"created"}`))
	}))
	defer server.Close()

	dbClient := newTestDBClient(t)
	s := NewSyncer(2, &immich.Client{ServerURL: server.URL, APIKey: "test-key"}, dbClient)

	err := s.SyncAssets([]string{successPath1, successPath2, failPath})
	require.NoError(t, err)

	successAsset1, err := dbClient.FindByPath(successPath1)
	require.NoError(t, err)
	require.NotNil(t, successAsset1)
	assert.Equal(t, "success", successAsset1.Status)
	assert.Equal(t, "immich-id-ok1.jpg", successAsset1.ImmichID.String)

	successAsset2, err := dbClient.FindByPath(successPath2)
	require.NoError(t, err)
	require.NotNil(t, successAsset2)
	assert.Equal(t, "success", successAsset2.Status)
	assert.Equal(t, "immich-id-ok2.jpg", successAsset2.ImmichID.String)

	failAsset, err := dbClient.FindByPath(failPath)
	require.NoError(t, err)
	require.NotNil(t, failAsset)
	assert.Equal(t, "failed", failAsset.Status)
	assert.Equal(t, int64(1), failAsset.FailedCount)
}
