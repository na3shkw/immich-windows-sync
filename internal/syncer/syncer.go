package syncer

import (
	"immich-windows-sync/internal/db"
	"immich-windows-sync/internal/immich"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Syncer struct {
	workerCount  int
	immichClient *immich.Client
	dbClient     *db.Client
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

func NewSyncer(workerCount int, immichClient *immich.Client, dbClient *db.Client) *Syncer {
	return &Syncer{
		workerCount:  workerCount,
		immichClient: immichClient,
		dbClient:     dbClient,
	}
}

// 指定フォルダを再帰的に走査して拡張子でフィルタリング後・未同期のものだけを抽出してファイルパスを返す
func (s *Syncer) ScanUnsyncedFiles(targetDir string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			extension := strings.ToLower(filepath.Ext(path))
			if _, ok := targetExtensions[extension]; ok {
				files = append(files, path)
			}
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

func (s *Syncer) SyncAssets(files []string) error {
	jobsCh := make(chan string)
	var wg sync.WaitGroup
	wg.Add(s.workerCount)

	for i := 0; i < s.workerCount; i++ {
		go func() {
			defer wg.Done()
			for path := range jobsCh {
				s.dbClient.MarkAsSyncing(path)
				uploadResult, err := s.immichClient.UploadAsset(path)
				if err != nil {
					s.dbClient.MarkAsFailed(path, err.Error())
					continue
				}
				s.dbClient.MarkAsSuccess(path, uploadResult.Id)
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
