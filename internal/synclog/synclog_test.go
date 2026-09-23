package synclog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestLogger(t *testing.T) (*Logger, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test", "sync.jsonl")
	logger, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() {
		logger.Close()
	})
	return logger, path
}

func readLines(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var lines []map[string]any
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &m))
		lines = append(lines, m)
	}
	require.NoError(t, scanner.Err())
	return lines
}

// Log: 書き込んだフィールドがそのまま1行のJSONとして記録され、timeフィールドが付与されることを確認する
func TestLogger_Log(t *testing.T) {
	logger, path := openTestLogger(t)

	logger.Log(map[string]any{"event": "upload", "status": "success", "path": "C:/photos/a.jpg"})
	logger.Log(map[string]any{"event": "upload", "status": "failed", "path": "C:/photos/b.jpg"})
	require.NoError(t, logger.Close())

	lines := readLines(t, path)
	require.Len(t, lines, 2)

	assert.Equal(t, "upload", lines[0]["event"])
	assert.Equal(t, "success", lines[0]["status"])
	assert.Equal(t, "C:/photos/a.jpg", lines[0]["path"])
	assert.NotEmpty(t, lines[0]["time"])

	assert.Equal(t, "failed", lines[1]["status"])
}

// Log: ワーカープールを想定した並行呼び出しでも、書き込みが競合・破損しないことを確認する
func TestLogger_Log_Concurrent(t *testing.T) {
	logger, path := openTestLogger(t)

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			logger.Log(map[string]any{"event": "upload", "n": i})
		}(i)
	}
	wg.Wait()
	require.NoError(t, logger.Close())

	lines := readLines(t, path)
	assert.Len(t, lines, goroutines)
}

func TestReadRecent(t *testing.T) {
	t.Run("ファイルが存在しない場合は空を返す", func(t *testing.T) {
		lines, err := ReadRecent(filepath.Join(t.TempDir(), "missing.log"), 10)

		require.NoError(t, err)
		assert.Empty(t, lines)
	})

	t.Run("末尾n行だけを返す", func(t *testing.T) {
		logger, path := openTestLogger(t)
		for i := range 5 {
			logger.Log(map[string]any{"n": i})
		}
		require.NoError(t, logger.Close())

		lines, err := ReadRecent(path, 2)

		require.NoError(t, err)
		require.Len(t, lines, 2)

		var last map[string]any
		require.NoError(t, json.Unmarshal([]byte(lines[1]), &last))
		assert.Equal(t, float64(4), last["n"])
	})
}
