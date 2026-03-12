package db

import "database/sql"

func NewClient(dbFile string) (*Client, error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS assets (
		id                   INTEGER  PRIMARY KEY AUTOINCREMENT,
		immich_id            TEXT,
		checksum             TEXT     NOT NULL UNIQUE,
		path                 TEXT     NOT NULL UNIQUE,
		status               TEXT     NOT NULL CHECK(status IN ('success', 'syncing', 'failed')),
		failed_count         INTEGER  NOT NULL DEFAULT 0,
		latest_failed_reason TEXT,
		created_at           DATETIME NOT NULL DEFAULT (DATETIME('now')),
		updated_at           DATETIME NOT NULL DEFAULT (DATETIME('now'))
	)`)
	if err != nil {
		return nil, err
	}
	return &Client{
		db: db,
	}, nil
}
