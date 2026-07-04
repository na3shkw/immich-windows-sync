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

// 指定フォルダを再帰的に走査（filepath.Walk）
//   └── 各ファイルについて
//         ├── 画像・動画ファイルか判定（拡張子フィルタ）
//         ├── DBで未同期かチェック（FindByPath / SearchByStatus）
//         └── 未同期なら → アップロードキューへ

// キューのファイルをgoroutineで並列アップロード
//   └── 各goroutineが
//         ├── MarkAsSyncing
//         ├── immich.UploadAsset
//         └── 成功 → MarkAsSuccess / 失敗 → MarkAsFailed

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

	failedRecords, err := s.dbClient.SearchByStatus("failed")
	if err != nil {
		return nil, err
	}

	unsyncedFiles := []string{}
	for _, file := range files {
		// TODO: DBクエリが高頻度で飛ぶので改善する
		unsyncedRecord, err := s.dbClient.FindByPath(file)
		if err != nil {
			return nil, err
		}
		if unsyncedRecord == nil {
			unsyncedFiles = append(unsyncedFiles, file)
		} else {
			// TODO: 毎回ループ回していてパフォーマンスが悪いのでは？
			for _, fr := range failedRecords {
				if fr.Path == file {
					unsyncedFiles = append(unsyncedFiles, file)
				}
			}
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
