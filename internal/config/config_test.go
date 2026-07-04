package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Load: 設定ファイルが存在しない場合はゼロ値のConfigをエラーなしで返すことを確認する
func TestLoad_NotExist(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	cfg, err := Load()

	require.NoError(t, err)
	assert.Empty(t, cfg.Immich.ServerURL)
	assert.Empty(t, cfg.TargetFolders)
}

// Load がエラーを返すべきケースをまとめて検証する
func TestLoad_Errors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T)
	}{
		{
			name: "APPDATA is empty",
			setup: func(t *testing.T) {
				t.Setenv("APPDATA", "")
			},
		},
		{
			name: "Broken JSON file",
			setup: func(t *testing.T) {
				tmpdir := t.TempDir()
				t.Setenv("APPDATA", tmpdir)
				configPath := filepath.Join(tmpdir, "immich-sync", "config.json")
				err := os.MkdirAll(filepath.Dir(configPath), 0644)
				require.NoError(t, err)
				err = os.WriteFile(configPath, []byte("{invalid json\n"), 0644)
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			_, err := Load()

			assert.Error(t, err)
		})
	}
}

// Save → Load のラウンドトリップで、保存した内容がそのまま読み込めることを確認する
func TestSaveThenLoad(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())

	want := Config{
		Immich: ImmichConfig{
			ServerURL: "http://example.com:2283",
			APIKey:    "api_key",
		},
		TargetFolders: []string{
			"hoge/foo",
			"bar/baz",
		},
	}
	err := Save(want)
	require.NoError(t, err)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, want, *cfg)
}
