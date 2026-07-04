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
	Checksum           string
	Path               string
	Status             string
	FailedCount        int64
	LatestFailedReason sql.NullString
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (c *Client) Create(checksum string, path string) error {
	_, err := c.db.Exec(
		`INSERT INTO assets (checksum, path, status, failed_count) VALUES (?, ?, ?, ?)`,
		checksum, path, "syncing", 0,
	)
	return err
}

func (c *Client) MarkAsSyncing(path string) error {
	_, err := c.db.Exec(
		`UPDATE assets SET status = ?, updated_at = ? WHERE path = ?`,
		"syncing", time.Now(), path,
	)
	return err
}

func (c *Client) MarkAsSuccess(path string, immichId string) error {
	_, err := c.db.Exec(
		`UPDATE assets 
		SET
			status = ?,
			immich_id = ?,
			updated_at = ?
		WHERE path = ?`,
		"success", immichId, time.Now(), path,
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
			checksum,
			path,
			status,
			failed_count,
			latest_failed_reason,
			created_at,
			updated_at
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
			&asset.Checksum,
			&asset.Path,
			&asset.Status,
			&asset.FailedCount,
			&asset.LatestFailedReason,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}
	return assets, nil
}

func (c *Client) FindByPath(path string) (*Asset, error) {
	row := c.db.QueryRow(
		`SELECT
			id,
			immich_id,
			checksum,
			path,
			status,
			failed_count,
			latest_failed_reason,
			created_at,
			updated_at
		FROM assets WHERE path = ?`,
		path,
	)

	asset := Asset{}
	err := row.Scan(
		&asset.ID,
		&asset.ImmichID,
		&asset.Checksum,
		&asset.Path,
		&asset.Status,
		&asset.FailedCount,
		&asset.LatestFailedReason,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &asset, nil
}

func (c *Client) FindByChecksum(checksum string) (*Asset, error) {
	row := c.db.QueryRow(
		`SELECT
			id,
			immich_id,
			checksum,
			path,
			status,
			failed_count,
			latest_failed_reason,
			created_at,
			updated_at
		FROM assets WHERE checksum = ?`,
		checksum,
	)

	asset := Asset{}
	err := row.Scan(
		&asset.ID,
		&asset.ImmichID,
		&asset.Checksum,
		&asset.Path,
		&asset.Status,
		&asset.FailedCount,
		&asset.LatestFailedReason,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &asset, nil
}
