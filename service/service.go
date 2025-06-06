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

// Service represents the main application service
type Service struct {
	config   *config.Config
	database *db.DB
	client   *github.Client
	ctx      context.Context
	cancel   context.CancelFunc
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

	logger.Info("Service initialized successfully",
		zap.String("repo_owner", cfg.RepoOwner),
		zap.String("repo_name", cfg.RepoName),
		zap.Int("poll_interval", cfg.PollInterval))

	return &Service{
		config:   cfg,
		database: database,
		client:   client,
		ctx:      ctx,
		cancel:   cancel,
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
		zap.String("repo_name", s.config.RepoName))

	// Check if context is already cancelled
	if s.ctx.Err() != nil {
		return fmt.Errorf("service context cancelled: %w", s.ctx.Err())
	}

	return processRepository(s.ctx, s.database, s.client, s.config, time.Time{})
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
			return processRepository(s.ctx, s.database, s.client, s.config, latestDate)
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

// processRepository handles the processing of a single repository
func processRepository(ctx context.Context, database DBInterface, client GitHubClientInterface, cfg *config.Config, latestDate time.Time) error {
	// Check context cancellation
	if ctx.Err() != nil {
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	// First, fetch and store repository information
	logger.Info("Fetching repository information",
		zap.String("repo_owner", cfg.RepoOwner),
		zap.String("repo_name", cfg.RepoName))

	repo, err := client.FetchRepo(ctx, cfg.RepoOwner, cfg.RepoName)
	if err != nil {
		return fmt.Errorf("failed to fetch repository %s/%s: %w", cfg.RepoOwner, cfg.RepoName, err)
	}

	// Convert to model and store
	repoModel := models.Repository{
		Name:            cfg.RepoName,
		Owner:           cfg.RepoOwner,
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

	if err := database.StoreRepository(ctx, repoModel); err != nil {
		return fmt.Errorf("failed to store repository %s/%s: %w", cfg.RepoOwner, cfg.RepoName, err)
	}

	// Get the stored repository to get its ID
	storedRepo, err := database.GetByName(ctx, cfg.RepoName)
	if err != nil {
		return fmt.Errorf("failed to get stored repository %s: %w", cfg.RepoName, err)
	}

	// Fetch commits
	logger.Info("Fetching commits",
		zap.String("repo_owner", cfg.RepoOwner),
		zap.String("repo_name", cfg.RepoName),
		zap.Time("since", latestDate))

	commits, err := client.FetchCommits(ctx, cfg.RepoOwner, cfg.RepoName, latestDate)
	if err != nil {
		return fmt.Errorf("failed to fetch commits for %s/%s: %w", cfg.RepoOwner, cfg.RepoName, err)
	}

	if len(commits) == 0 {
		logger.Info("No new commits found",
			zap.String("repo_owner", cfg.RepoOwner),
			zap.String("repo_name", cfg.RepoName))
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
		zap.String("repo_owner", cfg.RepoOwner),
		zap.String("repo_name", cfg.RepoName),
		zap.Int("commit_count", len(commits)))

	if err := database.BatchInsert(ctx, commitModels); err != nil {
		return fmt.Errorf("failed to store commits for %s/%s: %w", cfg.RepoOwner, cfg.RepoName, err)
	}

	logger.Info("Successfully processed repository",
		zap.String("repo_owner", cfg.RepoOwner),
		zap.String("repo_name", cfg.RepoName),
		zap.Int("commit_count", len(commits)))

	return nil
}
