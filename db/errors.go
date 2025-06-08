package db

import "fmt"

// Common errors
var (
	ErrNoCommitsFound     = fmt.Errorf("no commits found")
	ErrRepositoryNotFound = fmt.Errorf("repository not found")
	ErrInvalidInput       = fmt.Errorf("invalid input")
	ErrDatabaseConnection = fmt.Errorf("database connection error")
	ErrTransactionFailed  = fmt.Errorf("transaction failed")
)
