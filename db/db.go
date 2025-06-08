package db

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"

	"githubapifetch/logger"
)

// DB represents a database connection
type DB struct {
	conn *sqlx.DB
	// Prepared statements cache
	stmtCache struct {
		sync.RWMutex
		statements map[string]*sqlx.Stmt
	}
}

// safeLogInfo safely logs info messages, falling back to standard log if logger is not initialized
func safeLogInfo(msg string, fields ...zap.Field) {
	if logger.GetLogger() != nil {
		logger.Info(msg, fields...)
	} else {
		// Fallback to standard log with constant format string
		log.Printf("%s", msg)
	}
}

// New creates a new database connection
func New() (*DB, error) {
	dsn := fmt.Sprintf(
		"user=%s password=%s dbname=%s port=%s host=%s sslmode=disable",
		viper.GetString("POSTGRES_USER"),
		viper.GetString("POSTGRES_PASSWORD"),
		viper.GetString("POSTGRES_DB"),
		viper.GetString("POSTGRES_PORT"),
		viper.GetString("POSTGRES_HOST"),
	)

	safeLogInfo("Connecting to database", zap.String("dsn", dsn))
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}

	// Configure connection pool with defaults
	maxOpenConns := 25 // Default value
	if val := viper.GetString("DB_MAX_OPEN_CONNS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxOpenConns = parsed
		}
	}

	maxIdleConns := 25 // Default value
	if val := viper.GetString("DB_MAX_IDLE_CONNS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxIdleConns = parsed
		}
	}

	connMaxLifetime := 5 * time.Minute // Default value
	if val := viper.GetString("DB_CONN_MAX_LIFETIME"); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			connMaxLifetime = parsed
		}
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	// Initialize statement cache
	database := &DB{
		conn: db,
	}
	database.stmtCache.statements = make(map[string]*sqlx.Stmt)

	safeLogInfo("Database connection established",
		zap.Int("max_open_conns", maxOpenConns),
		zap.Int("max_idle_conns", maxIdleConns),
		zap.Duration("conn_max_lifetime", connMaxLifetime))
	return database, nil
}

// getStmt returns a prepared statement from cache or creates a new one
func (db *DB) getStmt(ctx context.Context, query string) (*sqlx.Stmt, error) {
	db.stmtCache.RLock()
	stmt, exists := db.stmtCache.statements[query]
	db.stmtCache.RUnlock()

	if exists {
		return stmt, nil
	}

	db.stmtCache.Lock()
	defer db.stmtCache.Unlock()

	// Double-check after acquiring write lock
	if stmt, exists = db.stmtCache.statements[query]; exists {
		return stmt, nil
	}

	stmt, err := db.conn.PreparexContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}

	db.stmtCache.statements[query] = stmt
	return stmt, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	// Close all prepared statements
	db.stmtCache.Lock()
	for _, stmt := range db.stmtCache.statements {
		stmt.Close()
	}
	db.stmtCache.Unlock()

	// Close the database connection
	return db.conn.Close()
}
