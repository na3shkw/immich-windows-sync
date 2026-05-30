package sync

import (
	"immich-windows-sync/internal/db"
	"immich-windows-sync/internal/immich"
)

type Syncer struct {
	workerCount  int
	immichClient *immich.Client
	dbClient     *db.Client
}
