package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"githubapifetch/models"
)

// setupTestDB creates a new test database connection with a mock
func setupTestDB(t *testing.T) (*DB, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	database := &DB{conn: sqlxDB}
	database.stmtCache.statements = make(map[string]*sqlx.Stmt)

	cleanup := func() {
		database.Close()
		db.Close()
	}

	return database, mock, cleanup
}

func TestGetLatestDate(t *testing.T) {
	tests := []struct {
		name        string
		repoName    string
		mockSetup   func(sqlmock.Sqlmock)
		expected    time.Time
		expectedErr error
	}{
		{
			name:     "successful retrieval",
			repoName: "test-repo",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"max_date"}).
					AddRow(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
				mock.ExpectQuery("SELECT MAX\\(c.date\\)").
					WithArgs("test-repo").
					WillReturnRows(rows)
			},
			expected:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedErr: nil,
		},
		{
			name:     "no commits found",
			repoName: "empty-repo",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"max_date"}).
					AddRow(sql.NullTime{})
				mock.ExpectQuery("SELECT MAX\\(c.date\\)").
					WithArgs("empty-repo").
					WillReturnRows(rows)
			},
			expected:    time.Time{},
			expectedErr: ErrNoCommitsFound,
		},
		{
			name:     "repository not found",
			repoName: "non-existent",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT MAX\\(c.date\\)").
					WithArgs("non-existent").
					WillReturnError(sql.ErrNoRows)
			},
			expected:    time.Time{},
			expectedErr: ErrRepositoryNotFound,
		},
		{
			name:        "empty repository name",
			repoName:    "",
			mockSetup:   func(mock sqlmock.Sqlmock) {},
			expected:    time.Time{},
			expectedErr: ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.mockSetup(mock)

			result, err := db.GetLatestDate(context.Background(), tt.repoName)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetByName(t *testing.T) {
	tests := []struct {
		name        string
		repoName    string
		mockSetup   func(sqlmock.Sqlmock)
		expected    *models.Repository
		expectedErr error
	}{
		{
			name:     "successful retrieval",
			repoName: "test-repo",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "owner", "url", "created_at", "updated_at",
					"description", "language", "forks_count", "stars_count",
					"open_issues_count", "watchers_count",
				}).AddRow(
					1, "test-repo", "test-owner", "https://github.com/test-owner/test-repo",
					time.Date(2025, time.June, 6, 3, 40, 24, 173519000, time.Local),
					time.Date(2025, time.June, 6, 3, 40, 24, 173520000, time.Local),
					"Test repo", "Go", 10, 100, 5, 50,
				)
				mock.ExpectQuery("SELECT id, name, owner").
					WithArgs("test-repo").
					WillReturnRows(rows)
			},
			expected: &models.Repository{
				ID:              1,
				Name:            "test-repo",
				Owner:           "test-owner",
				URL:             "https://github.com/test-owner/test-repo",
				Description:     "Test repo",
				Language:        "Go",
				ForksCount:      10,
				StarsCount:      100,
				OpenIssuesCount: 5,
				WatchersCount:   50,
				CreatedAt:       time.Date(2025, time.June, 6, 3, 40, 24, 173519000, time.Local),
				UpdatedAt:       time.Date(2025, time.June, 6, 3, 40, 24, 173520000, time.Local),
			},
			expectedErr: nil,
		},
		{
			name:     "repository not found",
			repoName: "non-existent",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, name, owner").
					WithArgs("non-existent").
					WillReturnError(sql.ErrNoRows)
			},
			expected:    nil,
			expectedErr: ErrRepositoryNotFound,
		},
		{
			name:        "empty repository name",
			repoName:    "",
			mockSetup:   func(mock sqlmock.Sqlmock) {},
			expected:    nil,
			expectedErr: ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.mockSetup(mock)

			result, err := db.GetByName(context.Background(), tt.repoName)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStoreRepository(t *testing.T) {
	tests := []struct {
		name        string
		repo        models.Repository
		mockSetup   func(sqlmock.Sqlmock)
		expectedErr error
	}{
		{
			name: "successful store",
			repo: models.Repository{
				Name:            "test-repo",
				Owner:           "test-owner",
				URL:             "https://github.com/test-owner/test-repo",
				Description:     "Test repo",
				Language:        "Go",
				ForksCount:      10,
				StarsCount:      100,
				OpenIssuesCount: 5,
				WatchersCount:   50,
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO repositories").
					WithArgs(
						"test-repo", "test-owner", "https://github.com/test-owner/test-repo",
						sqlmock.AnyArg(), sqlmock.AnyArg(), "Test repo", "Go",
						10, 100, 5, 50,
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
			},
			expectedErr: nil,
		},
		{
			name: "empty repository name",
			repo: models.Repository{
				Owner: "test-owner",
			},
			mockSetup:   func(mock sqlmock.Sqlmock) {},
			expectedErr: ErrInvalidInput,
		},
		{
			name: "empty owner",
			repo: models.Repository{
				Name: "test-repo",
			},
			mockSetup:   func(mock sqlmock.Sqlmock) {},
			expectedErr: ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.mockSetup(mock)

			err := db.StoreRepository(context.Background(), tt.repo)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestBatchInsert(t *testing.T) {
	tests := []struct {
		name        string
		commits     []models.Commit
		mockSetup   func(sqlmock.Sqlmock)
		expectedErr error
	}{
		{
			name: "successful batch insert",
			commits: []models.Commit{
				{
					SHA:        "abc123",
					RepoID:     1,
					Message:    "test commit",
					AuthorName: "test author",
					Date:       time.Now(),
					URL:        "https://github.com/test-owner/test-repo/commit/abc123",
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectPrepare("INSERT INTO commits")
				mock.ExpectExec("INSERT INTO commits").
					WithArgs(
						"abc123", 1, "test commit", "test author",
						sqlmock.AnyArg(), "https://github.com/test-owner/test-repo/commit/abc123",
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedErr: nil,
		},
		{
			name:        "empty commits slice",
			commits:     []models.Commit{},
			mockSetup:   func(mock sqlmock.Sqlmock) {},
			expectedErr: nil,
		},
		{
			name: "transaction failure",
			commits: []models.Commit{
				{
					SHA:        "abc123",
					RepoID:     1,
					Message:    "test commit",
					AuthorName: "test author",
					Date:       time.Now(),
					URL:        "https://github.com/test-owner/test-repo/commit/abc123",
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(sql.ErrConnDone)
			},
			expectedErr: ErrTransactionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.mockSetup(mock)

			err := db.BatchInsert(context.Background(), tt.commits)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetRepositoryStats(t *testing.T) {
	tests := []struct {
		name        string
		repoName    string
		mockSetup   func(sqlmock.Sqlmock)
		expected    *models.RepositoryStats
		expectedErr error
	}{
		{
			name:     "successful retrieval",
			repoName: "test-repo",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"total_commits", "unique_authors",
					"first_commit_date", "last_commit_date",
				}).AddRow(
					100, 5,
					time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				)
				mock.ExpectQuery("SELECT COUNT").
					WithArgs("test-repo").
					WillReturnRows(rows)
			},
			expected: &models.RepositoryStats{
				TotalCommits:    100,
				UniqueAuthors:   5,
				FirstCommitDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				LastCommitDate:  time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			expectedErr: nil,
		},
		{
			name:     "repository not found",
			repoName: "non-existent",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT COUNT").
					WithArgs("non-existent").
					WillReturnError(sql.ErrNoRows)
			},
			expected:    nil,
			expectedErr: ErrRepositoryNotFound,
		},
		{
			name:        "empty repository name",
			repoName:    "",
			mockSetup:   func(mock sqlmock.Sqlmock) {},
			expected:    nil,
			expectedErr: ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, cleanup := setupTestDB(t)
			defer cleanup()

			tt.mockSetup(mock)

			result, err := db.GetRepositoryStats(context.Background(), tt.repoName)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
