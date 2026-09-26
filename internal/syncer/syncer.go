package syncer

import (
	"immich-windows-sync/internal/db"
	"immich-windows-sync/internal/immich"
	"immich-windows-sync/internal/synclog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type Syncer struct {
	workerCount  int
	immichClient *immich.Client
	dbClient     *db.Client
	logger       *synclog.Logger
}

// 同期対象ファイルの拡張子
// Immichでサポートされているファイルタイプ：https://github.com/immich-app/immich/blob/v3.0.1/server/src/utils/mime-types.ts
//
// 動画の拡張子は現在コメントアウトして同期対象外にしている。
// immich.Client.UploadAsset がファイル全体をメモリに読み込んでから送信する実装のため、
// 数GB級の動画がワーカー数ぶん並行してアップロードされるとメモリ不足になりうる。
// ストリーミング送信（io.Pipe 等）に対応したら有効化する。
var targetExtensions = map[string]struct{}{
	".3fr": {},
	// ".3gp":  {},
	// ".3gpp": {},
	".ari": {},
	".arw": {},
	// ".avi":  {},
	".avif": {},
	".bmp":  {},
	".cap":  {},
	".cin":  {},
	".cr2":  {},
	".cr3":  {},
	".crw":  {},
	".dcr":  {},
	".dng":  {},
	".erf":  {},
	".fff":  {},
	// ".flv":  {},
	".gif":  {},
	".heic": {},
	".heif": {},
	".hif":  {},
	".iiq":  {},
	".insp": {},
	// ".insv": {},
	".jp2":  {},
	".jpe":  {},
	".jpeg": {},
	".jpg":  {},
	".jxl":  {},
	".k25":  {},
	".kdc":  {},
	// ".m2t":  {},
	// ".m2ts": {},
	// ".m4v":  {},
	// ".mkv":  {},
	// ".mov":  {},
	// ".mp4":  {},
	// ".mpe":  {},
	// ".mpeg": {},
	// ".mpg":  {},
	".mpo": {},
	".mrw": {},
	// ".mts":  {},
	// ".mxf":  {},
	".nef":  {},
	".nrw":  {},
	".orf":  {},
	".ori":  {},
	".pef":  {},
	".png":  {},
	".psd":  {},
	".raf":  {},
	".raw":  {},
	".rw2":  {},
	".rwl":  {},
	".sr2":  {},
	".srf":  {},
	".srw":  {},
	".svg":  {},
	".tif":  {},
	".tiff": {},
	// ".ts":   {},
	// ".vob":  {},
	// ".webm": {},
	".webp": {},
	// ".wmv":  {},
	".x3f": {},
}

func NewSyncer(workerCount int, immichClient *immich.Client, dbClient *db.Client, logger *synclog.Logger) *Syncer {
	return &Syncer{
		workerCount:  workerCount,
		immichClient: immichClient,
		dbClient:     dbClient,
		logger:       logger,
	}
}

// アプリ終了時にdbClientを閉じる
func (s *Syncer) Close() error {
	return s.dbClient.Close()
}

// CountByStatus はstatusごとの同期済みアセット件数を返す。
func (s *Syncer) CountByStatus() (map[string]int64, error) {
	return s.dbClient.CountByStatus()
}

// FailedAssets は現在failed状態のアセット一覧を返す。
func (s *Syncer) FailedAssets() ([]*db.Asset, error) {
	return s.dbClient.SearchByStatus("failed")
}

// hasTargetExtension は拡張子が同期対象（targetExtensions）かどうかを大文字小文字を区別せずに判定する。
func hasTargetExtension(path string) bool {
	_, ok := targetExtensions[strings.ToLower(filepath.Ext(path))]
	return ok
}

// IsSyncTarget は path が現在ディスク上に存在する、同期対象拡張子の通常ファイルかどうかを返す。
// Watcher のイベントにはディレクトリ（子ファイルの変更で親にも Write が発火する）や
// ブラウザのダウンロード途中ファイル（.crdownload 等）・desktop.ini なども含まれるため、
// アップロード前にこれで絞り込む。
func IsSyncTarget(path string) bool {
	if !hasTargetExtension(path) {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// PruneFailed は failed レコードのうち、もう同期対象ではなくなったもの
// （削除・リネームされたファイルや、過去に誤ってアップロードを試みたディレクトリ・非対象ファイル）を削除し、
// 削除した件数を返す。これらは再スキャンでも再試行されないため、残しておくとトレイの Error 表示が解除されない。
func (s *Syncer) PruneFailed() (int, error) {
	assets, err := s.dbClient.SearchByStatus("failed")
	if err != nil {
		return 0, err
	}
	pruned := 0
	for _, a := range assets {
		if IsSyncTarget(a.Path) {
			continue
		}
		if err := s.dbClient.DeleteByPath(a.Path); err != nil {
			return pruned, err
		}
		s.logger.Log(map[string]any{"event": "failed_record_pruned", "path": a.Path})
		pruned++
	}
	return pruned, nil
}

// 指定フォルダを再帰的に走査して拡張子でフィルタリング後・未同期のものだけを抽出してファイルパスを返す
// excludedDirs に含まれるディレクトリはその配下ごとスキャン対象から除外する
func (s *Syncer) ScanUnsyncedFiles(targetDir string, excludedDirs []string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if slices.Contains(excludedDirs, path) {
				return filepath.SkipDir
			}
			return nil
		}
		if hasTargetExtension(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	assets, err := s.dbClient.SearchByStatus("success")
	if err != nil {
		return nil, err
	}
	syncedFiles := make(map[string]*db.Asset, len(assets))
	for _, a := range assets {
		syncedFiles[a.Path] = a
	}

	unsyncedFiles := []string{}
	for _, file := range files {
		if _, ok := syncedFiles[file]; !ok {
			unsyncedFiles = append(unsyncedFiles, file)
		}
	}

	return unsyncedFiles, nil
}

func (s *Syncer) syncOne(path string) error {
	s.dbClient.MarkAsSyncing(path)

	uploadResult, err := s.immichClient.UploadAsset(path)
	if err != nil {
		s.dbClient.MarkAsFailed(path, err.Error())
		s.logger.Log(map[string]any{"event": "upload", "status": "failed", "path": path, "reason": err.Error()})
		return err
	}
	s.dbClient.MarkAsSuccess(path, uploadResult.Id, uploadResult.Status)
	s.logger.Log(map[string]any{
		"event":        "upload",
		"status":       "success",
		"path":         path,
		"immichId":     uploadResult.Id,
		"immichStatus": uploadResult.Status,
	})
	return nil
}

func (s *Syncer) SyncAssets(files []string) error {
	jobsCh := make(chan string)
	var wg sync.WaitGroup
	wg.Add(s.workerCount)

	for i := 0; i < s.workerCount; i++ {
		go func() {
			defer wg.Done()
			for path := range jobsCh {
				_ = s.syncOne(path)
			}
		}()
	}

	for _, file := range files {
		jobsCh <- file
	}
	close(jobsCh)
	wg.Wait()
	return nil
}
