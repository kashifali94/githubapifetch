package github

import (
	"context"
	"encoding/json"
	"fmt"
	"githubapifetch/logger"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// RateLimit represents GitHub's rate limit information
type RateLimit struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

// Client represents a GitHub API client
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    *url.URL
}

type RepoResponse struct {
	Description     string    `json:"description"`
	HTMLURL         string    `json:"html_url"`
	Language        string    `json:"language"`
	ForksCount      int       `json:"forks_count"`
	StargazersCount int       `json:"stargazers_count"`
	OpenIssuesCount int       `json:"open_issues_count"`
	WatchersCount   int       `json:"watchers_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CommitResponse struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
	HTMLURL string `json:"html_url"`
}

func NewClient(token string) *Client {
	baseURL, _ := url.Parse("https://api.github.com")
	logger.Info("Initializing GitHub client", zap.String("base_url", baseURL.String()))
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (c *Client) FetchRepo(ctx context.Context, owner, name string) (*RepoResponse, error) {
	path := fmt.Sprintf("/repos/%s/%s", owner, name)
	reqURL := c.baseURL.ResolveReference(&url.URL{Path: path})

	logger.Info("Fetching repository",
		zap.String("owner", owner),
		zap.String("name", name),
		zap.String("url", reqURL.String()))

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error("Failed to fetch repository",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("name", name))
		return nil, fmt.Errorf("failed to fetch repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("Failed to fetch repository",
			zap.Int("status_code", resp.StatusCode),
			zap.String("owner", owner),
			zap.String("name", name))
		return nil, fmt.Errorf("failed to fetch repository: status code %d", resp.StatusCode)
	}

	var repo RepoResponse
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		logger.Error("Failed to decode repository response",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("name", name))
		return nil, fmt.Errorf("failed to decode repository response: %w", err)
	}

	logger.Info("Successfully fetched repository",
		zap.String("owner", owner),
		zap.String("name", name),
		zap.String("language", repo.Language),
		zap.Int("stars", repo.StargazersCount))

	return &repo, nil
}

// parseRateLimit parses rate limit information from response headers
func parseRateLimit(resp *http.Response) RateLimit {
	limit, _ := strconv.Atoi(resp.Header.Get("X-RateLimit-Limit"))
	remaining, _ := strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining"))
	reset, _ := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64)

	return RateLimit{
		Limit:     limit,
		Remaining: remaining,
		Reset:     time.Unix(reset, 0),
	}
}

// handleRateLimit handles rate limit responses and implements exponential backoff
func (c *Client) handleRateLimit(resp *http.Response) error {
	if resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0" {
		resetTime, _ := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64)
		waitTime := time.Until(time.Unix(resetTime, 0))
		logger.Info("Rate limit exceeded, waiting for reset",
			zap.Int("limit", parseRateLimit(resp).Limit),
			zap.Time("reset_time", time.Unix(resetTime, 0)),
			zap.Duration("wait_time", waitTime))
		time.Sleep(waitTime)
		return nil
	}
	return nil
}

// FetchCommits fetches commits from a repository with pagination support
func (c *Client) FetchCommits(ctx context.Context, owner, name string, since time.Time) ([]CommitResponse, error) {
	var allCommits []CommitResponse
	page := 1
	perPage := 100 // GitHub's maximum allowed per page

	for {
		path := fmt.Sprintf("/repos/%s/%s/commits", owner, name)
		reqURL := c.baseURL.ResolveReference(&url.URL{Path: path})

		q := reqURL.Query()
		q.Set("page", strconv.Itoa(page))
		q.Set("per_page", strconv.Itoa(perPage))
		if !since.IsZero() {
			q.Set("since", since.Format(time.RFC3339))
		}
		reqURL.RawQuery = q.Encode()

		logger.Info("Fetching commits page",
			zap.String("owner", owner),
			zap.String("name", name),
			zap.Int("page", page),
			zap.Time("since", since),
			zap.String("url", reqURL.String()))

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			logger.Error("Failed to fetch commits",
				zap.Error(err),
				zap.String("owner", owner),
				zap.String("name", name))
			return nil, fmt.Errorf("failed to fetch commits: %w", err)
		}

		// Handle rate limiting
		if err := c.handleRateLimit(resp); err != nil {
			resp.Body.Close()
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			logger.Error("Failed to fetch commits",
				zap.Int("status_code", resp.StatusCode),
				zap.String("owner", owner),
				zap.String("name", name))
			return nil, fmt.Errorf("failed to fetch commits: status code %d", resp.StatusCode)
		}

		var commits []CommitResponse
		if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
			resp.Body.Close()
			logger.Error("Failed to decode commits response",
				zap.Error(err),
				zap.String("owner", owner),
				zap.String("name", name))
			return nil, fmt.Errorf("failed to decode commits response: %w", err)
		}
		resp.Body.Close()

		// If no commits returned, we've reached the end
		if len(commits) == 0 {
			break
		}

		allCommits = append(allCommits, commits...)

		// Check if we've reached the last page
		linkHeader := resp.Header.Get("Link")
		if linkHeader == "" || !containsNextPage(linkHeader) {
			break
		}

		page++
	}

	logger.Info("Successfully fetched all commits",
		zap.String("owner", owner),
		zap.String("name", name),
		zap.Int("total_count", len(allCommits)))

	return allCommits, nil
}

// containsNextPage checks if the Link header contains a next page
func containsNextPage(linkHeader string) bool {
	return linkHeader != "" && linkHeader[len(linkHeader)-1] == '>'
}
