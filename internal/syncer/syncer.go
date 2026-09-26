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
	"sync/atomic"
)

type Syncer struct {
	workerCount    int
	immichClient   *immich.Client
	dbClient       *db.Client
	logger         *synclog.Logger
	remainingCount atomic.Int64
}

// 同期対象ファイルの拡張子
// Immichでサポートされているファイルタイプ：https://github.com/immich-app/immich/blob/v3.0.1/server/src/utils/mime-types.ts
var targetExtensions = map[string]struct{}{
	".3fr":  {},
	".3gp":  {},
	".3gpp": {},
	".ari":  {},
	".arw":  {},
	".avi":  {},
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
	".flv":  {},
	".gif":  {},
	".heic": {},
	".heif": {},
	".hif":  {},
	".iiq":  {},
	".insp": {},
	".insv": {},
	".jp2":  {},
	".jpe":  {},
	".jpeg": {},
	".jpg":  {},
	".jxl":  {},
	".k25":  {},
	".kdc":  {},
	".m2t":  {},
	".m2ts": {},
	".m4v":  {},
	".mkv":  {},
	".mov":  {},
	".mp4":  {},
	".mpe":  {},
	".mpeg": {},
	".mpg":  {},
	".mpo":  {},
	".mrw":  {},
	".mts":  {},
	".mxf":  {},
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
	".ts":   {},
	".vob":  {},
	".webm": {},
	".webp": {},
	".wmv":  {},
	".x3f":  {},
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

// RemainingCount は残り処理件数を返す。
func (s *Syncer) RemainingCount() int64 {
	return s.remainingCount.Load()
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
		extension := strings.ToLower(filepath.Ext(path))
		if _, ok := targetExtensions[extension]; ok {
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
	defer s.remainingCount.Add(-1)

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
	s.remainingCount.Add(int64(len(files)))
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
