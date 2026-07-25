package immich

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestFile(t *testing.T) string {
	t.Helper()
	filePath := filepath.Join(t.TempDir(), "photo.jpg")
	require.NoError(t, os.WriteFile(filePath, []byte("dummy image data"), 0644))
	return filePath
}

// UploadAsset: リクエストが期待通りの内容（ヘッダー・マルチパートの各フィールド）で送信され、
// レスポンスが正しくパースされることを確認する
func TestUploadAsset_Success(t *testing.T) {
	filePath := newTestFile(t)

	var gotAPIKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("x-api-key")

		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		assert.Equal(t, filePath, r.MultipartForm.Value["deviceAssetId"][0])
		assert.Equal(t, "immich-windows-sync", r.MultipartForm.Value["deviceId"][0])
		assert.NotEmpty(t, r.MultipartForm.Value["fileCreatedAt"][0])
		assert.NotEmpty(t, r.MultipartForm.Value["fileModifiedAt"][0])

		fh := r.MultipartForm.File["assetData"][0]
		f, err := fh.Open()
		require.NoError(t, err)
		defer f.Close()
		bytes, err := io.ReadAll(f)
		require.NoError(t, err)
		assert.Equal(t, []byte("dummy image data"), bytes)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(UploadResult{Id: "asset-id-1", Status: "created"})
	}))
	defer server.Close()

	client := &Client{ServerURL: server.URL, APIKey: "test-api-key"}
	result, err := client.UploadAsset(filePath)

	require.NoError(t, err)
	assert.Equal(t, "asset-id-1", result.Id)
	assert.Equal(t, "test-api-key", gotAPIKey)
}

// UploadAsset がエラーを返すべきケースをまとめて検証する
func TestUploadAsset_Errors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{
			name: "non-2xx status code",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
		},
		{
			name: "invalid json response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("{invalid json"))
			},
		},
	}

	filePath := newTestFile(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			client := &Client{ServerURL: server.URL, APIKey: "test-api-key"}
			_, err := client.UploadAsset(filePath)

			assert.Error(t, err)
		})
	}
}

func TestUploadAsset_FileNotFound(t *testing.T) {
	client := &Client{ServerURL: "http://example.com", APIKey: "test-api-key"}

	_, err := client.UploadAsset(filepath.Join(t.TempDir(), "missing.jpg"))

	assert.Error(t, err)
}
