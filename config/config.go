package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	GitHubToken  string
	RepoOwner    string
	RepoName     string
	PollInterval int
	StartDate    time.Time
}

// NewConfig creates a new Config instance
func NewConfig() *Config {
	return &Config{}
}

// Load loads configuration from environment variables
func (c *Config) Load() error {
	// Set up Viper
	viper.SetConfigFile("/app/.env")
	viper.AutomaticEnv()

	// Read .env file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Required fields
	c.GitHubToken = viper.GetString("GITHUB_TOKEN")
	if c.GitHubToken == "" {
		return fmt.Errorf("GITHUB_TOKEN is required")
	}

	c.RepoOwner = viper.GetString("REPO_OWNER")
	if c.RepoOwner == "" {
		return fmt.Errorf("REPO_OWNER is required")
	}

	c.RepoName = viper.GetString("REPO_NAME")
	if c.RepoName == "" {
		return fmt.Errorf("REPO_NAME is required")
	}

	// Optional fields with defaults
	c.PollInterval = viper.GetInt("POLL_INTERVAL")
	if c.PollInterval == 0 {
		c.PollInterval = 3600 // Default to 1 hour
	}

	startDateStr := viper.GetString("START_DATE")
	if startDateStr == "" {
		c.StartDate = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	} else {
		var err error
		c.StartDate, err = time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return fmt.Errorf("invalid START_DATE format: %w", err)
		}
	}

	return nil
}
