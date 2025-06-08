package db

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"githubapifetch/models"
)

// MonitorRepositoryChanges starts a goroutine to monitor repository changes
func (db *DB) MonitorRepositoryChanges(ctx context.Context, interval time.Duration, callback func(repoName string, latestDate time.Time) error) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := db.checkRepositories(ctx, callback); err != nil {
					log.Printf("Error checking repositories: %v", err)
				}
			}
		}
	}()
}

// checkRepositories checks all repositories for changes
func (db *DB) checkRepositories(ctx context.Context, callback func(repoName string, latestDate time.Time) error) error {
	var repos []models.Repository
	if err := db.conn.SelectContext(ctx, &repos, "SELECT * FROM repositories"); err != nil {
		return fmt.Errorf("failed to fetch repositories for monitoring: %w", err)
	}

	// Process repositories concurrently with a worker pool
	const maxWorkers = 5
	sem := make(chan struct{}, maxWorkers)
	errChan := make(chan error, len(repos))
	var wg sync.WaitGroup

	for _, repo := range repos {
		wg.Add(1)
		go func(repo models.Repository) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			latestDate, err := db.GetLatestDate(ctx, repo.Name)
			if err != nil {
				if err == ErrNoCommitsFound {
					log.Printf("No commits found for repository %s, skipping...", repo.Name)
					return
				}
				errChan <- fmt.Errorf("error getting latest date for repository %s: %w", repo.Name, err)
				return
			}

			if err := callback(repo.Name, latestDate); err != nil {
				errChan <- fmt.Errorf("error processing repository %s: %w", repo.Name, err)
			}
		}(repo)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred while processing repositories: %v", errs)
	}

	return nil
}
