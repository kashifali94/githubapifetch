package db

import (
	"time"
)

type Repository struct {
	ID              int
	Owner           string
	Name            string
	Description     string
	URL             string
	Language        string
	ForksCount      int
	StarsCount      int
	OpenIssuesCount int
	WatchersCount   int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastSyncAt      time.Time
}

type Commit struct {
	SHA        string
	RepoID     int
	Message    string
	AuthorName string
	Date       time.Time
	URL        string
}
