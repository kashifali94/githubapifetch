package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"

	"githubapifetch/logger"
	"githubapifetch/models"
)

// Common errors
var (
	ErrNoCommitsFound     = fmt.Errorf("no commits found")
	ErrRepositoryNotFound = fmt.Errorf("repository not found")
	ErrInvalidInput       = fmt.Errorf("invalid input")
	ErrDatabaseConnection = fmt.Errorf("database connection error")
	ErrTransactionFailed  = fmt.Errorf("transaction failed")
)

// DB represents a database connection
type DB struct {
	conn *sqlx.DB
	// Prepared statements cache
	stmtCache struct {
		sync.RWMutex
		statements map[string]*sqlx.Stmt
	}
}

// safeLogInfo safely logs info messages, falling back to standard log if logger is not initialized
func safeLogInfo(msg string, fields ...zap.Field) {
	if logger.GetLogger() != nil {
		logger.Info(msg, fields...)
	} else {
		// Fallback to standard log with constant format string
		log.Printf("%s", msg)
	}
}

// New creates a new database connection
func New() (*DB, error) {
	dsn := fmt.Sprintf(
		"user=%s password=%s dbname=%s port=%s host=%s sslmode=disable",
		viper.GetString("POSTGRES_USER"),
		viper.GetString("POSTGRES_PASSWORD"),
		viper.GetString("POSTGRES_DB"),
		viper.GetString("POSTGRES_PORT"),
		viper.GetString("POSTGRES_HOST"),
	)

	safeLogInfo("Connecting to database", zap.String("dsn", dsn))
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}

	// Configure connection pool with defaults
	maxOpenConns := 25 // Default value
	if val := viper.GetString("DB_MAX_OPEN_CONNS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxOpenConns = parsed
		}
	}

	maxIdleConns := 25 // Default value
	if val := viper.GetString("DB_MAX_IDLE_CONNS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxIdleConns = parsed
		}
	}

	connMaxLifetime := 5 * time.Minute // Default value
	if val := viper.GetString("DB_CONN_MAX_LIFETIME"); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			connMaxLifetime = parsed
		}
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	// Initialize statement cache
	database := &DB{
		conn: db,
	}
	database.stmtCache.statements = make(map[string]*sqlx.Stmt)

	safeLogInfo("Database connection established",
		zap.Int("max_open_conns", maxOpenConns),
		zap.Int("max_idle_conns", maxIdleConns),
		zap.Duration("conn_max_lifetime", connMaxLifetime))
	return database, nil
}

// getStmt returns a prepared statement from cache or creates a new one
func (db *DB) getStmt(ctx context.Context, query string) (*sqlx.Stmt, error) {
	db.stmtCache.RLock()
	stmt, exists := db.stmtCache.statements[query]
	db.stmtCache.RUnlock()

	if exists {
		return stmt, nil
	}

	db.stmtCache.Lock()
	defer db.stmtCache.Unlock()

	// Double-check after acquiring write lock
	if stmt, exists = db.stmtCache.statements[query]; exists {
		return stmt, nil
	}

	stmt, err := db.conn.PreparexContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}

	db.stmtCache.statements[query] = stmt
	return stmt, nil
}

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
		RETURNING id
	`

	var id int
	err := db.conn.QueryRowContext(ctx, query,
		repo.Name,
		repo.Owner,
		repo.URL,
		repo.CreatedAt,
		repo.UpdatedAt,
		repo.Description,
		repo.Language,
		repo.ForksCount,
		repo.StarsCount,
		repo.OpenIssuesCount,
		repo.WatchersCount,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to store repository %s/%s: %w", repo.Owner, repo.Name, err)
	}

	safeLogInfo("Repository stored successfully", zap.String("owner", repo.Owner), zap.String("name", repo.Name), zap.Int("id", id))
	return nil
}

// GetLatestDate returns the date of the most recent commit for a repository
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

// Close closes the database connection
func (db *DB) Close() error {
	// Close all prepared statements
	db.stmtCache.Lock()
	for _, stmt := range db.stmtCache.statements {
		stmt.Close()
	}
	db.stmtCache.Unlock()

	if err := db.conn.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	return nil
}
