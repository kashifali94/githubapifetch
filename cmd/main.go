package main

import (
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

	// Initialize service
	svc, err := service.NewService()
	if err != nil {
		logger.Fatal("Failed to initialize service", zap.Error(err))
	}
	defer svc.Close()

	// Start the service
	if err := svc.Start(); err != nil {
		logger.Fatal("Service error", zap.Error(err))
	}
}
