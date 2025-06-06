// Package models defines the core data structures used throughout the application.
package models

import "time"

// Repository represents a GitHub repository
type Repository struct {
	ID              int       `db:"id" json:"id"`
	Name            string    `db:"name" json:"name"`
	Owner           string    `db:"owner" json:"owner"`
	Description     string    `db:"description" json:"description"`
	URL             string    `db:"url" json:"url"`
	Language        string    `db:"language" json:"language"`
	ForksCount      int       `db:"forks_count" json:"forks_count"`
	StarsCount      int       `db:"stars_count" json:"stars_count"`
	OpenIssuesCount int       `db:"open_issues_count" json:"open_issues_count"`
	WatchersCount   int       `db:"watchers_count" json:"watchers_count"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

// Commit represents a GitHub commit
type Commit struct {
	ID         int       `db:"id" json:"id"`
	SHA        string    `db:"sha" json:"sha"`
	RepoID     int       `db:"repository_id" json:"repository_id"`
	Message    string    `db:"message" json:"message"`
	AuthorName string    `db:"author_name" json:"author_name"`
	Date       time.Time `db:"date" json:"date"`
	URL        string    `db:"url" json:"url"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// AuthorStats represents commit statistics for a specific author.
type AuthorStats struct {
	AuthorName string `db:"author_name" json:"author_name"`
	Count      int    `db:"count" json:"count"`
}

// PaginationParams represents parameters for paginated queries
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// NewPaginationParams creates a new PaginationParams with validated values.
// If page or pageSize are less than 1, they will be set to their default values.
func NewPaginationParams(page, pageSize int) PaginationParams {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 100
	}
	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}
}

// RepositoryStats represents statistics about a repository
type RepositoryStats struct {
	TotalCommits    int       `db:"total_commits" json:"total_commits"`
	UniqueAuthors   int       `db:"unique_authors" json:"unique_authors"`
	FirstCommitDate time.Time `db:"first_commit_date" json:"first_commit_date"`
	LastCommitDate  time.Time `db:"last_commit_date" json:"last_commit_date"`
}
