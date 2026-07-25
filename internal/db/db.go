package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func NewClient(dbFile string) (*Client, error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, err
	}
	// SQLiteは複数コネクションからの同時書き込みに弱く、SQLITE_BUSYで書き込みが失敗しうるため1本に制限する
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS assets (
		id                   INTEGER  PRIMARY KEY AUTOINCREMENT,
		immich_id            TEXT,
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

func (c *Client) Close() error {
	return c.db.Close()
}
