package db

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"githubapifetch/models"
)

// StoreRepository stores a repository in the database
func (db *DB) StoreRepository(ctx context.Context, repo models.Repository) error {
	if repo.Name == "" || repo.Owner == "" {
		return fmt.Errorf("%w: repository name and owner cannot be empty", ErrInvalidInput)
	}

	safeLogInfo("Storing repository", zap.String("owner", repo.Owner), zap.String("name", repo.Name))
	query := `
		INSERT INTO repositories (
			name, owner, url, created_at, updated_at,
			description, language, forks_count, stars_count,
			open_issues_count, watchers_count
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (name, owner) DO UPDATE SET
			url = EXCLUDED.url,
			updated_at = EXCLUDED.updated_at,
			description = EXCLUDED.description,
			language = EXCLUDED.language,
			forks_count = EXCLUDED.forks_count,
			stars_count = EXCLUDED.stars_count,
			open_issues_count = EXCLUDED.open_issues_count,
			watchers_count = EXCLUDED.watchers_count
	`

	_, err := db.conn.ExecContext(ctx, query,
		repo.Name, repo.Owner, repo.URL, repo.CreatedAt, repo.UpdatedAt,
		repo.Description, repo.Language, repo.ForksCount, repo.StarsCount,
		repo.OpenIssuesCount, repo.WatchersCount,
	)
	if err != nil {
		return fmt.Errorf("failed to store repository: %w", err)
	}

	safeLogInfo("Repository stored successfully",
		zap.String("owner", repo.Owner),
		zap.String("name", repo.Name))
	return nil
}

// GetByName retrieves repository information by name
func (db *DB) GetByName(ctx context.Context, name string) (*models.Repository, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: repository name cannot be empty", ErrInvalidInput)
	}

	safeLogInfo("Retrieving repository by name", zap.String("name", name))
	var repo models.Repository
	query := `
		SELECT id, name, owner, url, created_at, updated_at,
			description, language, forks_count, stars_count,
			open_issues_count, watchers_count
		FROM repositories
		WHERE name = $1
	`

	if err := db.conn.GetContext(ctx, &repo, query, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: repository %s not found", ErrRepositoryNotFound, name)
		}
		return nil, fmt.Errorf("failed to get repository %s: %w", name, err)
	}

	safeLogInfo("Repository retrieved successfully", zap.String("name", name))
	return &repo, nil
}

// GetRepositoryStats returns statistics about a repository
func (db *DB) GetRepositoryStats(ctx context.Context, repoName string) (*models.RepositoryStats, error) {
	if repoName == "" {
		return nil, fmt.Errorf("%w: repository name cannot be empty", ErrInvalidInput)
	}

	stats := &models.RepositoryStats{}
	query := `
		SELECT 
			COUNT(*) as total_commits,
			COUNT(DISTINCT author_name) as unique_authors,
			MIN(c.date) as first_commit_date,
			MAX(c.date) as last_commit_date
		FROM commits c
		JOIN repositories r ON c.repository_id = r.id
		WHERE r.name = $1
	`

	if err := db.conn.GetContext(ctx, stats, query, repoName); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: no statistics found for repository %s", ErrRepositoryNotFound, repoName)
		}
		return nil, fmt.Errorf("failed to get repository statistics: %w", err)
	}

	return stats, nil
}
