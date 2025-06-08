package main

import (
	"context"
	"flag"
	"os"
	"time"

	"githubapifetch/logger"
	"githubapifetch/service"

	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	if err := logger.Initialize("info"); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// Define command flags
	resetSyncCmd := flag.NewFlagSet("reset-sync", flag.ExitOnError)
	repoName := resetSyncCmd.String("repo", "", "Repository name to reset sync point for")
	daysAgo := resetSyncCmd.Int("days", 30, "Number of days ago to reset sync point to")

	// Check if a command was provided
	if len(os.Args) < 2 {
		// If no command provided, start the service normally
		svc, err := service.NewService()
		if err != nil {
			logger.Fatal("Failed to initialize service", zap.Error(err))
		}
		defer svc.Close()

		if err := svc.Start(); err != nil {
			logger.Fatal("Service error", zap.Error(err))
		}
		return
	}

	// Parse the command
	switch os.Args[1] {
	case "reset-sync":
		// Skip the program name and command name
		args := os.Args[2:]

		// Parse flags
		if err := resetSyncCmd.Parse(args); err != nil {
			logger.Fatal("Failed to parse reset-sync command", zap.Error(err))
		}

		// Validate required flags
		if *repoName == "" {
			logger.Fatal("Repository name is required",
				zap.String("usage", "reset-sync -repo <repo-name> [-days <number>]"),
				zap.Strings("args", args))
		}

		// Initialize service
		svc, err := service.NewService()
		if err != nil {
			logger.Fatal("Failed to initialize service", zap.Error(err))
		}
		defer svc.Close()

		// Calculate the new sync point date
		newDate := time.Now().Add(-time.Duration(*daysAgo) * 24 * time.Hour)
		logger.Info("Resetting sync point",
			zap.String("repo", *repoName),
			zap.Time("new_date", newDate),
			zap.Int("days_ago", *daysAgo),
			zap.Strings("parsed_args", args))

		// Reset sync point
		if err := svc.ResetSyncPoint(context.Background(), *repoName, newDate); err != nil {
			logger.Fatal("Failed to reset sync point", zap.Error(err))
		}

		logger.Info("Successfully reset sync point",
			zap.String("repo", *repoName),
			zap.Time("new_date", newDate))

	default:
		logger.Fatal("Unknown command", zap.String("command", os.Args[1]))
	}
}
