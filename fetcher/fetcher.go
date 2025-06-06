package fetcher

import (
	"context"
	"fmt"
	"log"
	"time"

	"githubapifetch/github"
	"githubapifetch/models"
)

// DBInterface defines the database operations needed by the fetcher
type DBInterface interface {
	StoreRepository(ctx context.Context, repo models.Repository) error
	GetByName(ctx context.Context, name string) (*models.Repository, error)
	BatchInsert(ctx context.Context, commits []models.Commit) error
	Close() error
}

// GitHubClientInterface defines the GitHub client operations needed by the fetcher
type GitHubClientInterface interface {
	FetchRepo(ctx context.Context, owner, name string) (*github.RepoResponse, error)
	FetchCommits(ctx context.Context, owner, name string, since time.Time) ([]github.CommitResponse, error)
}

// FetchAndStore fetches repository and commit data and stores it in the database
func FetchAndStore(ctx context.Context, database DBInterface, client GitHubClientInterface, owner, name string, since time.Time) error {
	// Fetch repository information
	repo, err := client.FetchRepo(ctx, owner, name)
	if err != nil {
		return fmt.Errorf("failed to fetch repository: %w", err)
	}

	// Convert to model
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

	// Store repository information
	if err := database.StoreRepository(ctx, repoModel); err != nil {
		return fmt.Errorf("failed to store repository: %w", err)
	}

	log.Printf("Repository %s/%s stored successfully", owner, name)

	// Get the stored repository to get its ID
	storedRepo, err := database.GetByName(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get stored repository: %w", err)
	}

	// Fetch commits
	commits, err := client.FetchCommits(ctx, owner, name, since)
	if err != nil {
		return fmt.Errorf("failed to fetch commits: %w", err)
	}

	if len(commits) == 0 {
		log.Printf("No new commits found for repository %s/%s", owner, name)
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
	if err := database.BatchInsert(ctx, commitModels); err != nil {
		return fmt.Errorf("failed to store commits: %w", err)
	}

	log.Printf("Successfully stored %d commits for repository %s/%s", len(commits), owner, name)
	return nil
}
