// immich-stub はImmich APIの /api/assets アップロードエンドポイントだけを模倣する
// スタブサーバー。実際のImmichサーバーやディスクには一切書き込まず、アップロードされた
// ファイルのチェックサムだけをメモリ上で記録する。動作確認後の後片付け（アップロード済み
// アセットの削除）が不要になることが、本物のImmichを使わない主な狙い。
//
// 使い方:
//
//	go run ./tools/immich-stub -addr :2283
//
// Immich Windows SyncのConnection SettingsでServer URLを http://localhost:2283 に、
// API Keyは任意の値（検証しない）に設定して動作確認する。
package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type uploadResult struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

type store struct {
	mu           sync.Mutex
	checksumToID map[string]string
	nextID       int
}

func newStore() *store {
	return &store{checksumToID: make(map[string]string)}
}

// resolve はチェックサムを既知のIDと突き合わせる。既知なら(id, true)、未知なら新規IDを発行して(id, false)を返す。
func (s *store) resolve(checksum string) (id string, duplicate bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id, ok := s.checksumToID[checksum]; ok {
		return id, true
	}
	s.nextID++
	id = "stub-asset-" + strconv.Itoa(s.nextID)
	s.checksumToID[checksum] = id
	return id, false
}

// logEvent はAIやログ整形ツールが読みやすいよう、1件1行のJSONで標準出力に書き出す。
func logEvent(fields map[string]any) {
	fields["time"] = time.Now().Format(time.RFC3339)
	data, err := json.Marshal(fields)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(data))
}

func handleUploadAsset(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseMultipartForm(64 << 20); err != nil {
			logEvent(map[string]any{"event": "upload_error", "reason": "parse multipart form failed", "error": err.Error()})
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("assetData")
		if err != nil {
			logEvent(map[string]any{"event": "upload_error", "reason": "assetData field missing", "error": err.Error()})
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		deviceAssetID := formValue(r, "deviceAssetId")

		hasher := sha1.New()
		if _, err := io.Copy(hasher, file); err != nil {
			logEvent(map[string]any{"event": "upload_error", "reason": "failed to read assetData", "error": err.Error()})
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		checksum := hex.EncodeToString(hasher.Sum(nil))

		// ファイル名に"fail"が含まれる場合はアップロード失敗をシミュレートする
		// （internal/syncer/syncer_test.go のモックサーバーと同じ規約）
		if strings.Contains(strings.ToLower(header.Filename), "fail") {
			logEvent(map[string]any{
				"event":         "upload",
				"status":        "failed",
				"filename":      header.Filename,
				"deviceAssetId": deviceAssetID,
				"checksum":      checksum,
			})
			http.Error(w, "simulated upload failure", http.StatusInternalServerError)
			return
		}

		id, duplicate := s.resolve(checksum)
		result := uploadResult{Id: id}
		statusCode := http.StatusCreated
		if duplicate {
			result.Status = "duplicate"
			statusCode = http.StatusOK
		} else {
			result.Status = "created"
		}

		logEvent(map[string]any{
			"event":         "upload",
			"status":        result.Status,
			"filename":      header.Filename,
			"deviceAssetId": deviceAssetID,
			"checksum":      checksum,
			"id":            id,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(result)
	}
}

func formValue(r *http.Request, key string) string {
	if r.MultipartForm == nil {
		return ""
	}
	values := r.MultipartForm.Value[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func main() {
	// ループバックのみに限定する。":2283"のように全インターフェースで待ち受けると
	// Windowsファイアウォールの警告ダイアログが出るが、動作確認にLAN側からのアクセスは不要。
	addr := flag.String("addr", "127.0.0.1:2283", "listen address")
	flag.Parse()

	s := newStore()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/assets", handleUploadAsset(s))

	logEvent(map[string]any{"event": "listening", "addr": *addr})
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}
