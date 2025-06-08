package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"githubapifetch/logger"

	"github.com/stretchr/testify/assert"
)

func init() {
	// Initialize logger for tests
	_ = logger.Initialize("debug")
}

func TestNewClient(t *testing.T) {
	token := "test-token"
	client := NewClient(token)

	assert.NotNil(t, client)
	assert.Equal(t, token, client.token)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
}

func TestFetchRepo(t *testing.T) {
	testCases := []struct {
		name           string
		owner          string
		repoName       string
		mockResponse   *RepoResponse
		mockStatusCode int
		expectedError  bool
	}{
		{
			name:     "successful fetch",
			owner:    "test-owner",
			repoName: "test-repo",
			mockResponse: &RepoResponse{
				Description:     "Test repository",
				HTMLURL:         "https://github.com/test-owner/test-repo",
				Language:        "Go",
				ForksCount:      10,
				StargazersCount: 100,
				OpenIssuesCount: 5,
				WatchersCount:   50,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			mockStatusCode: http.StatusOK,
			expectedError:  false,
		},
		{
			name:           "repository not found",
			owner:          "test-owner",
			repoName:       "non-existent",
			mockResponse:   nil,
			mockStatusCode: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name:           "unauthorized",
			owner:          "test-owner",
			repoName:       "test-repo",
			mockResponse:   nil,
			mockStatusCode: http.StatusUnauthorized,
			expectedError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request headers
				assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
				assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))

				// Verify request URL
				expectedPath := "/repos/" + tc.owner + "/" + tc.repoName
				assert.Equal(t, expectedPath, r.URL.Path)

				// Set response
				w.WriteHeader(tc.mockStatusCode)
				if tc.mockResponse != nil {
					json.NewEncoder(w).Encode(tc.mockResponse)
				}
			}))
			defer server.Close()

			// Create client with test server URL
			client := &Client{
				token: "test-token",
				httpClient: &http.Client{
					Timeout: 30 * time.Second,
				},
			}

			// Override the base URL for testing
			baseURL, _ := url.Parse(server.URL)
			client.baseURL = baseURL

			// Test FetchRepo
			repo, err := client.FetchRepo(context.Background(), tc.owner, tc.repoName)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Nil(t, repo)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, repo)
				assert.Equal(t, tc.mockResponse.Description, repo.Description)
				assert.Equal(t, tc.mockResponse.HTMLURL, repo.HTMLURL)
				assert.Equal(t, tc.mockResponse.Language, repo.Language)
				assert.Equal(t, tc.mockResponse.ForksCount, repo.ForksCount)
				assert.Equal(t, tc.mockResponse.StargazersCount, repo.StargazersCount)
				assert.Equal(t, tc.mockResponse.OpenIssuesCount, repo.OpenIssuesCount)
				assert.Equal(t, tc.mockResponse.WatchersCount, repo.WatchersCount)
			}
		})
	}
}

func TestFetchCommits(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name            string
		owner           string
		repoName        string
		since           time.Time
		mockResponses   [][]CommitResponse
		mockStatusCodes []int
		mockHeaders     []http.Header
		expectedError   bool
	}{
		{
			name:     "successful fetch with pagination",
			owner:    "test-owner",
			repoName: "test-repo",
			since:    now.Add(-24 * time.Hour),
			mockResponses: [][]CommitResponse{
				{
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
							Message: "Test commit 1",
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
				{
					{
						SHA: "def456",
						Commit: struct {
							Message string `json:"message"`
							Author  struct {
								Name  string    `json:"name"`
								Email string    `json:"email"`
								Date  time.Time `json:"date"`
							} `json:"author"`
						}{
							Message: "Test commit 2",
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
						HTMLURL: "https://github.com/test-owner/test-repo/commit/def456",
					},
				},
			},
			mockStatusCodes: []int{http.StatusOK, http.StatusOK},
			mockHeaders: []http.Header{
				{
					"Link": []string{"<https://api.github.com/repos/test-owner/test-repo/commits?page=2>; rel=\"next\""},
				},
				{
					"Link": []string{},
				},
			},
			expectedError: false,
		},
		{
			name:     "rate limit handling",
			owner:    "test-owner",
			repoName: "test-repo",
			since:    now.Add(-24 * time.Hour),
			mockResponses: [][]CommitResponse{
				{
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
			},
			mockStatusCodes: []int{http.StatusForbidden, http.StatusOK},
			mockHeaders: []http.Header{
				{
					"X-RateLimit-Limit":     []string{"5000"},
					"X-RateLimit-Remaining": []string{"0"},
					"X-RateLimit-Reset":     []string{strconv.FormatInt(now.Add(time.Hour).Unix(), 10)},
				},
				{
					"X-RateLimit-Limit":     []string{"5000"},
					"X-RateLimit-Remaining": []string{"4999"},
					"X-RateLimit-Reset":     []string{strconv.FormatInt(now.Add(time.Hour).Unix(), 10)},
				},
			},
			expectedError: false,
		},
		{
			name:            "repository not found",
			owner:           "test-owner",
			repoName:        "non-existent",
			since:           now.Add(-24 * time.Hour),
			mockResponses:   [][]CommitResponse{nil},
			mockStatusCodes: []int{http.StatusNotFound},
			mockHeaders:     []http.Header{{}},
			expectedError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request headers
				assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
				assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))

				// Verify request URL and query parameters
				expectedPath := "/repos/" + tc.owner + "/" + tc.repoName + "/commits"
				assert.Equal(t, expectedPath, r.URL.Path)
				if !tc.since.IsZero() {
					assert.Equal(t, tc.since.Format(time.RFC3339), r.URL.Query().Get("since"))
				}

				// Set response headers
				for key, values := range tc.mockHeaders[requestCount] {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}

				// Set response status code
				w.WriteHeader(tc.mockStatusCodes[requestCount])

				// Set response body
				if tc.mockResponses[requestCount] != nil {
					json.NewEncoder(w).Encode(tc.mockResponses[requestCount])
				}

				requestCount++
			}))
			defer server.Close()

			// Create client with test server URL
			client := &Client{
				token: "test-token",
				httpClient: &http.Client{
					Timeout: 30 * time.Second,
				},
			}

			// Override the base URL for testing
			baseURL, _ := url.Parse(server.URL)
			client.baseURL = baseURL

			// Test FetchCommits
			commits, err := client.FetchCommits(context.Background(), tc.owner, tc.repoName, tc.since)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Nil(t, commits)
			} else {
				assert.NoError(t, err)
				if len(tc.mockResponses) > 0 {
					assert.NotNil(t, commits)
					// Calculate expected total commits
					expectedCount := 0
					for _, response := range tc.mockResponses {
						if response != nil {
							expectedCount += len(response)
						}
					}
					assert.Equal(t, expectedCount, len(commits))
				} else {
					assert.Empty(t, commits)
				}
			}
		})
	}
}
