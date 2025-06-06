package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"githubassign/config"
	"githubassign/github"
	"githubassign/models"
)

// MockDB is a mock implementation of the database interface
type MockDB struct {
	mock.Mock
}

func (m *MockDB) StoreRepository(ctx context.Context, repo models.Repository) error {
	args := m.Called(ctx, repo)
	return args.Error(0)
}

func (m *MockDB) GetByName(ctx context.Context, name string) (*models.Repository, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Repository), args.Error(1)
}

func (m *MockDB) BatchInsert(ctx context.Context, commits []models.Commit) error {
	args := m.Called(ctx, commits)
	return args.Error(0)
}

func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockGitHubClient is a mock implementation of the GitHub client
type MockGitHubClient struct {
	mock.Mock
}

func (m *MockGitHubClient) FetchRepo(ctx context.Context, owner, name string) (*github.RepoResponse, error) {
	args := m.Called(ctx, owner, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.RepoResponse), args.Error(1)
}

func (m *MockGitHubClient) FetchCommits(ctx context.Context, owner, name string, since time.Time) ([]github.CommitResponse, error) {
	args := m.Called(ctx, owner, name, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]github.CommitResponse), args.Error(1)
}

func TestProcessRepository(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name           string
		owner          string
		repoName       string
		since          time.Time
		mockRepo       *github.RepoResponse
		mockCommits    []github.CommitResponse
		mockStoredRepo *models.Repository
		setupMocks     func(*MockDB, *MockGitHubClient)
		expectedError  error
	}{
		{
			name:     "successful processing",
			owner:    "test-owner",
			repoName: "test-repo",
			since:    now.Add(-24 * time.Hour),
			mockRepo: &github.RepoResponse{
				Description:     "Test repository",
				HTMLURL:         "https://github.com/test-owner/test-repo",
				Language:        "Go",
				ForksCount:      10,
				StargazersCount: 100,
				OpenIssuesCount: 5,
				WatchersCount:   50,
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			mockCommits: []github.CommitResponse{
				{
					SHA: "abc123",
					Commit: struct {
						Message string `json:"message"`
						Author  struct {
							Name  string    `json:"name"`
							Email string    `json:"email"`
							Date  time.Time `json:"date"`
						} `json:"author"`
					}{
						Message: "Test commit",
						Author: struct {
							Name  string    `json:"name"`
							Email string    `json:"email"`
							Date  time.Time `json:"date"`
						}{
							Name:  "Test Author",
							Email: "test@example.com",
							Date:  now,
						},
					},
					HTMLURL: "https://github.com/test-owner/test-repo/commit/abc123",
				},
			},
			mockStoredRepo: &models.Repository{
				ID:        1,
				Name:      "test-repo",
				Owner:     "test-owner",
				CreatedAt: now,
				UpdatedAt: now,
			},
			setupMocks: func(mockDB *MockDB, mockClient *MockGitHubClient) {
				mockClient.On("FetchRepo", mock.Anything, "test-owner", "test-repo").
					Return(&github.RepoResponse{
						Description:     "Test repository",
						HTMLURL:         "https://github.com/test-owner/test-repo",
						Language:        "Go",
						ForksCount:      10,
						StargazersCount: 100,
						OpenIssuesCount: 5,
						WatchersCount:   50,
						CreatedAt:       now,
						UpdatedAt:       now,
					}, nil)

				mockDB.On("StoreRepository", mock.Anything, mock.MatchedBy(func(repo models.Repository) bool {
					return repo.Name == "test-repo" && repo.Owner == "test-owner"
				})).Return(nil)

				mockDB.On("GetByName", mock.Anything, "test-repo").
					Return(&models.Repository{
						ID:        1,
						Name:      "test-repo",
						Owner:     "test-owner",
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)

				mockClient.On("FetchCommits", mock.Anything, "test-owner", "test-repo", mock.Anything).
					Return([]github.CommitResponse{
						{
							SHA: "abc123",
							Commit: struct {
								Message string `json:"message"`
								Author  struct {
									Name  string    `json:"name"`
									Email string    `json:"email"`
									Date  time.Time `json:"date"`
								} `json:"author"`
							}{
								Message: "Test commit",
								Author: struct {
									Name  string    `json:"name"`
									Email string    `json:"email"`
									Date  time.Time `json:"date"`
								}{
									Name:  "Test Author",
									Email: "test@example.com",
									Date:  now,
								},
							},
							HTMLURL: "https://github.com/test-owner/test-repo/commit/abc123",
						},
					}, nil)

				mockDB.On("BatchInsert", mock.Anything, mock.MatchedBy(func(commits []models.Commit) bool {
					return len(commits) == 1 && commits[0].SHA == "abc123"
				})).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:     "fetch repo error",
			owner:    "test-owner",
			repoName: "test-repo",
			since:    now.Add(-24 * time.Hour),
			setupMocks: func(mockDB *MockDB, mockClient *MockGitHubClient) {
				mockClient.On("FetchRepo", mock.Anything, "test-owner", "test-repo").
					Return(nil, assert.AnError)
			},
			expectedError: assert.AnError,
		},
		{
			name:     "store repo error",
			owner:    "test-owner",
			repoName: "test-repo",
			since:    now.Add(-24 * time.Hour),
			mockRepo: &github.RepoResponse{
				Description: "Test repository",
				HTMLURL:     "https://github.com/test-owner/test-repo",
			},
			setupMocks: func(mockDB *MockDB, mockClient *MockGitHubClient) {
				mockClient.On("FetchRepo", mock.Anything, "test-owner", "test-repo").
					Return(&github.RepoResponse{
						Description: "Test repository",
						HTMLURL:     "https://github.com/test-owner/test-repo",
					}, nil)

				mockDB.On("StoreRepository", mock.Anything, mock.Anything).
					Return(assert.AnError)
			},
			expectedError: assert.AnError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &MockDB{}
			mockClient := &MockGitHubClient{}

			if tc.setupMocks != nil {
				tc.setupMocks(mockDB, mockClient)
			}

			cfg := &config.Config{
				RepoOwner: tc.owner,
				RepoName:  tc.repoName,
			}

			err := processRepository(context.Background(), mockDB, mockClient, cfg, tc.since)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			mockDB.AssertExpectations(t)
			mockClient.AssertExpectations(t)
		})
	}
}
