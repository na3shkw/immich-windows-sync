package sync

import (
	"immich-windows-sync/internal/db"
	"immich-windows-sync/internal/immich"
	"os"
	"path/filepath"
	"sync"
)

type Syncer struct {
	workerCount  int
	immichClient *immich.Client
	dbClient     *db.Client
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
	// targetDir配下を再帰的に走査して拡張子でファイルをフィルタリングする
	files := []string{}
	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		files = append(files, path)
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
