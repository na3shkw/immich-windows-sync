// Package synclog は同期活動（アップロード成功/失敗、Watcherの開始/停止など）を
// 1件1行のJSONとしてファイルに書き出す。アプリ内のログ表示だけでなく、
// アプリを再起動しても過去の記録を追えるようにするための永続化用途。
package synclog

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	mu   sync.Mutex
	file *os.File
}

func Open(path string) (*Logger, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{file: file}, nil
}

func (l *Logger) Close() error {
	return l.file.Close()
}

// Log は fields を1件1行のJSONとしてファイルに追記する。
// SyncAssets のワーカープール（複数goroutine）から同時に呼ばれるため、
// ファイルへの書き込みは l.mu で保護している。
func (l *Logger) Log(fields map[string]any) {
	fields["time"] = time.Now().Format(time.RFC3339)
	bytes, err := json.Marshal(fields)
	if err != nil {
		return
	}
	bytes = append(bytes, '\n')
	l.mu.Lock()
	l.file.Write(bytes)
	l.mu.Unlock()
}

// ReadRecent は path のログファイルから末尾n行を読み込んで返す。
// ファイルが存在しない場合は空スライスを返す（アプリ初回起動などまだ何も記録されていない場合）。
func ReadRecent(path string, n int) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	trimmed := strings.TrimRight(string(data), "\n")
	if trimmed == "" {
		return []string{}, nil
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}
