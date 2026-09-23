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
	if err := addImmichStatusColumnIfMissing(db); err != nil {
		return nil, err
	}
	return &Client{
		db: db,
	}, nil
}

// addImmichStatusColumnIfMissing はImmichのアップロードレスポンスのstatus（created/duplicate等）を
// 記録するための immich_status 列を追加する。CREATE TABLE IF NOT EXISTS は既存テーブルを
// 書き換えないため、既にDBファイルが存在するケースに対応する軽量マイグレーションとして用意している。
func addImmichStatusColumnIfMissing(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(assets)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "immich_status" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec(`ALTER TABLE assets ADD COLUMN immich_status TEXT`)
	return err
}

func (c *Client) Close() error {
	return c.db.Close()
}
