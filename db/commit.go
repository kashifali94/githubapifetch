package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"githubapifetch/models"
)

// GetLatestDate retrieves the latest commit date for a repository
func (db *DB) GetLatestDate(ctx context.Context, repoName string) (time.Time, error) {
	if repoName == "" {
		return time.Time{}, fmt.Errorf("%w: repository name cannot be empty", ErrInvalidInput)
	}

	var latestDate sql.NullTime
	query := `
		SELECT MAX(c.date) as max_date
		FROM commits c
		JOIN repositories r ON c.repository_id = r.id
		WHERE r.name = $1
	`

	if err := db.conn.GetContext(ctx, &latestDate, query, repoName); err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, fmt.Errorf("%w: repository %s not found", ErrRepositoryNotFound, repoName)
		}
		return time.Time{}, fmt.Errorf("failed to get latest commit date for repository %s: %w", repoName, err)
	}

	if !latestDate.Valid {
		return time.Time{}, fmt.Errorf("%w: repository %s", ErrNoCommitsFound, repoName)
	}

	return latestDate.Time, nil
}

// BatchInsert performs batch insertion of commits
func (db *DB) BatchInsert(ctx context.Context, commits []models.Commit) error {
	if len(commits) == 0 {
		return nil
	}

	safeLogInfo("Starting batch insertion of commits", zap.Int("count", len(commits)))
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTransactionFailed, err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO commits (sha, repository_id, message, author_name, date, url)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (sha) DO UPDATE SET
			message = EXCLUDED.message,
			author_name = EXCLUDED.author_name,
			date = EXCLUDED.date,
			url = EXCLUDED.url
		WHERE commits.date < EXCLUDED.date
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare commit insert statement: %w", err)
	}
	defer stmt.Close()

	// Use a worker pool for batch processing
	const batchSize = 1000
	const maxWorkers = 5
	sem := make(chan struct{}, maxWorkers)
	errChan := make(chan error, len(commits))
	var wg sync.WaitGroup

	for i := 0; i < len(commits); i += batchSize {
		end := i + batchSize
		if end > len(commits) {
			end = len(commits)
		}

		batch := commits[i:end]
		wg.Add(1)
		go func(batch []models.Commit) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			for _, commit := range batch {
				if _, err := stmt.ExecContext(ctx,
					commit.SHA,
					commit.RepoID,
					commit.Message,
					commit.AuthorName,
					commit.Date,
					commit.URL,
				); err != nil {
					errChan <- fmt.Errorf("failed to insert commit %s: %w", commit.SHA, err)
					return
				}
			}
		}(batch)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while inserting commits: %v", errs)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}

	safeLogInfo("Successfully inserted commits", zap.Int("count", len(commits)))
	return nil
}
