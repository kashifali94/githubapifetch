package service

import (
	"context"
	"fmt"
	"githubapifetch/config"
	"githubapifetch/db"
	"githubapifetch/github"
	"githubapifetch/logger"
	"githubapifetch/models"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// DBInterface abstracts the database operations needed by the service
// (for testability)
type DBInterface interface {
	StoreRepository(ctx context.Context, repo models.Repository) error
	GetByName(ctx context.Context, name string) (*models.Repository, error)
	BatchInsert(ctx context.Context, commits []models.Commit) error
	MonitorRepositoryChanges(ctx context.Context, interval time.Duration, callback func(string, time.Time) error)
	Close() error
}

// GitHubClientInterface abstracts the GitHub client operations needed by the service
// (for testability)
type GitHubClientInterface interface {
	FetchRepo(ctx context.Context, owner, name string) (*github.RepoResponse, error)
	FetchCommits(ctx context.Context, owner, name string, since time.Time) ([]github.CommitResponse, error)
}

// Service errors
var (
	ErrServiceInit     = fmt.Errorf("service initialization error")
	ErrServiceShutdown = fmt.Errorf("service shutdown error")
)

// RepositoryProcessor handles the core repository processing logic
type RepositoryProcessor struct {
	db     DBInterface
	client GitHubClientInterface
}

// NewRepositoryProcessor creates a new processor
func NewRepositoryProcessor(db DBInterface, client GitHubClientInterface) *RepositoryProcessor {
	return &RepositoryProcessor{
		db:     db,
		client: client,
	}
}

// Process handles a single repository processing operation
func (p *RepositoryProcessor) Process(ctx context.Context, owner, name string, since time.Time) error {
	// Check context cancellation
	if ctx.Err() != nil {
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	// First, fetch and store repository information
	logger.Info("Fetching repository information",
		zap.String("repo_owner", owner),
		zap.String("repo_name", name))

	repo, err := p.client.FetchRepo(ctx, owner, name)
	if err != nil {
		return fmt.Errorf("failed to fetch repository %s/%s: %w", owner, name, err)
	}

	// Convert to model and store
	repoModel := models.Repository{
		Name:            name,
		Owner:           owner,
		Description:     repo.Description,
		URL:             repo.HTMLURL,
		Language:        repo.Language,
		ForksCount:      repo.ForksCount,
		StarsCount:      repo.StargazersCount,
		OpenIssuesCount: repo.OpenIssuesCount,
		WatchersCount:   repo.WatchersCount,
		CreatedAt:       repo.CreatedAt,
		UpdatedAt:       repo.UpdatedAt,
	}

	if err := p.db.StoreRepository(ctx, repoModel); err != nil {
		return fmt.Errorf("failed to store repository %s/%s: %w", owner, name, err)
	}

	// Get the stored repository to get its ID
	storedRepo, err := p.db.GetByName(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get stored repository %s: %w", name, err)
	}

	// Fetch commits
	logger.Info("Fetching commits",
		zap.String("repo_owner", owner),
		zap.String("repo_name", name),
		zap.Time("since", since))

	commits, err := p.client.FetchCommits(ctx, owner, name, since)
	if err != nil {
		return fmt.Errorf("failed to fetch commits for %s/%s: %w", owner, name, err)
	}

	if len(commits) == 0 {
		logger.Info("No new commits found",
			zap.String("repo_owner", owner),
			zap.String("repo_name", name))
		return nil
	}

	// Convert commits to models
	var commitModels []models.Commit
	for _, commit := range commits {
		commitModel := models.Commit{
			SHA:        commit.SHA,
			RepoID:     storedRepo.ID,
			Message:    commit.Commit.Message,
			AuthorName: commit.Commit.Author.Name,
			Date:       commit.Commit.Author.Date,
			URL:        commit.HTMLURL,
		}
		commitModels = append(commitModels, commitModel)
	}

	// Store commits in batches
	logger.Info("Storing commits",
		zap.String("repo_owner", owner),
		zap.String("repo_name", name),
		zap.Int("commit_count", len(commits)))

	if err := p.db.BatchInsert(ctx, commitModels); err != nil {
		return fmt.Errorf("failed to store commits for %s/%s: %w", owner, name, err)
	}

	logger.Info("Successfully processed repository",
		zap.String("repo_owner", owner),
		zap.String("repo_name", name),
		zap.Int("commit_count", len(commits)))

	return nil
}

// Service represents the main application service
type Service struct {
	config    *config.Config
	database  DBInterface
	client    GitHubClientInterface
	processor *RepositoryProcessor
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewService creates a new service instance
func NewService() (*Service, error) {
	// Load configuration
	cfg := config.NewConfig()
	if err := cfg.Load(); err != nil {
		return nil, fmt.Errorf("%w: failed to load configuration: %v", ErrServiceInit, err)
	}

	// Initialize database
	database, err := db.New()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to initialize database: %v", ErrServiceInit, err)
	}

	// Initialize GitHub client
	client := github.NewClient(cfg.GitHubToken)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create repository processor
	processor := NewRepositoryProcessor(database, client)

	logger.Info("Service initialized successfully",
		zap.String("repo_owner", cfg.RepoOwner),
		zap.String("repo_name", cfg.RepoName),
		zap.Int("poll_interval", cfg.PollInterval))

	return &Service{
		config:    cfg,
		database:  database,
		client:    client,
		processor: processor,
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// Start initializes and starts the service
func (s *Service) Start() error {
	// Process initial repository
	if err := s.processInitialRepository(); err != nil {
		logger.Warn("Error processing initial repository",
			zap.Error(err),
			zap.String("repo_owner", s.config.RepoOwner),
			zap.String("repo_name", s.config.RepoName))
		// Continue despite initial processing error
	}

	// Start repository monitoring
	s.startMonitoring()

	// Wait for interrupt signal
	s.waitForShutdown()

	return nil
}

// processInitialRepository processes the initial repository state
func (s *Service) processInitialRepository() error {
	logger.Info("Processing initial repository",
		zap.String("repo_owner", s.config.RepoOwner),
		zap.String("repo_name", s.config.RepoName),
		zap.Time("start_date", s.config.StartDate))

	// Check if context is already cancelled
	if s.ctx.Err() != nil {
		return fmt.Errorf("service context cancelled: %w", s.ctx.Err())
	}

	return s.processor.Process(s.ctx, s.config.RepoOwner, s.config.RepoName, s.config.StartDate)
}

// startMonitoring starts the repository monitoring process
func (s *Service) startMonitoring() {
	logger.Info("Starting repository monitoring",
		zap.Int("poll_interval", s.config.PollInterval))

	s.database.MonitorRepositoryChanges(
		s.ctx,
		time.Duration(s.config.PollInterval)*time.Second,
		func(repoName string, latestDate time.Time) error {
			// Check if context is already cancelled
			if s.ctx.Err() != nil {
				return fmt.Errorf("service context cancelled: %w", s.ctx.Err())
			}

			return s.processor.Process(s.ctx, s.config.RepoOwner, repoName, latestDate)
		},
	)
}

// waitForShutdown waits for the shutdown signal
func (s *Service) waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutdown signal received, initiating graceful shutdown")
	s.cancel()
}

// Close performs cleanup operations
func (s *Service) Close() error {
	logger.Info("Closing service")
	s.cancel()
	if err := s.database.Close(); err != nil {
		return fmt.Errorf("%w: failed to close database: %v", ErrServiceShutdown, err)
	}
	return nil
}

// ResetSyncPoint resets the sync point for a repository to a specific date.
// This will trigger a new fetch of commits from the specified date.
func (s *Service) ResetSyncPoint(ctx context.Context, repoName string, newDate time.Time) error {
	if repoName == "" {
		return fmt.Errorf("repository name cannot be empty")
	}

	// Get the repository to find its owner
	repo, err := s.database.GetByName(ctx, repoName)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	// Process the repository with the new date
	if err := s.processor.Process(ctx, repo.Owner, repo.Name, newDate); err != nil {
		return fmt.Errorf("failed to process repository with new sync point: %w", err)
	}

	return nil
}
