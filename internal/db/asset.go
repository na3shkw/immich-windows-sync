package db

import (
	"database/sql"
	"errors"
	"time"
)

type Client struct {
	db *sql.DB
}

type Asset struct {
	ID                 int64
	ImmichID           sql.NullString
	Path               string
	Status             string
	FailedCount        int64
	LatestFailedReason sql.NullString
	CreatedAt          time.Time
	UpdatedAt          time.Time
	// ImmichStatus はImmichのアップロードレスポンスのstatus（created/duplicate等）をそのまま記録する。
	// syncの成否を表す Status とは別軸の情報。
	ImmichStatus sql.NullString
}

// レコードが存在しなければINSERT、存在すれば同期中状態にUPDATEする
func (c *Client) MarkAsSyncing(path string) error {
	_, err := c.db.Exec(
		`INSERT INTO assets (path, status, failed_count)
		VALUES (?, 'syncing', 0)
		ON CONFLICT(path) DO UPDATE SET
			status = 'syncing',
			updated_at = ?`,
		path, time.Now(),
	)
	return err
}

func (c *Client) MarkAsSuccess(path string, immichId string, immichStatus string) error {
	_, err := c.db.Exec(
		`UPDATE assets
		SET
			status = ?,
			immich_id = ?,
			immich_status = ?,
			updated_at = ?
		WHERE path = ?`,
		"success", immichId, immichStatus, time.Now(), path,
	)
	return err
}

func (c *Client) MarkAsFailed(path string, reason string) error {
	_, err := c.db.Exec(
		`UPDATE assets
		SET
			status = ?,
			failed_count = failed_count + 1,
			latest_failed_reason = ?,
			updated_at = ?
		WHERE path = ?`,
		"failed", reason, time.Now(), path,
	)
	return err
}

func (c *Client) SearchByStatus(status string) ([]*Asset, error) {
	rows, err := c.db.Query(
		`SELECT
			id,
			immich_id,
			path,
			status,
			failed_count,
			latest_failed_reason,
			created_at,
			updated_at,
			immich_status
		FROM assets WHERE status = ?`,
		status,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*Asset
	for rows.Next() {
		asset := Asset{}
		err = rows.Scan(
			&asset.ID,
			&asset.ImmichID,
			&asset.Path,
			&asset.Status,
			&asset.FailedCount,
			&asset.LatestFailedReason,
			&asset.CreatedAt,
			&asset.UpdatedAt,
			&asset.ImmichStatus,
		)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return assets, nil
}

// CountByStatus はstatusごとのレコード件数を返す。該当レコードがないstatusも0件として含める。
func (c *Client) CountByStatus() (map[string]int64, error) {
	counts := map[string]int64{"success": 0, "syncing": 0, "failed": 0}

	rows, err := c.db.Query(`SELECT status, COUNT(*) FROM assets GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

func (c *Client) FindByPath(path string) (*Asset, error) {
	row := c.db.QueryRow(
		`SELECT
			id,
			immich_id,
			path,
			status,
			failed_count,
			latest_failed_reason,
			created_at,
			updated_at,
			immich_status
		FROM assets WHERE path = ?`,
		path,
	)

	asset := Asset{}
	err := row.Scan(
		&asset.ID,
		&asset.ImmichID,
		&asset.Path,
		&asset.Status,
		&asset.FailedCount,
		&asset.LatestFailedReason,
		&asset.CreatedAt,
		&asset.UpdatedAt,
		&asset.ImmichStatus,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &asset, nil
}
