package db

import (
	"database/sql"
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
