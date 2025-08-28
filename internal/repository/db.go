// Package repository handles database connections and data access.
package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB holds the database connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// Connect establishes a connection to PostgreSQL using the provided database URL.
func Connect(ctx context.Context, dbURL string) (*DB, error) {
	if dbURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	// Configure connection pool
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Set connection pool settings
	config.MaxConns = 30
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 30

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("db connected",
		slog.String("max_conns", fmt.Sprintf("%d", config.MaxConns)),
		slog.String("min_conns", fmt.Sprintf("%d", config.MinConns)),
	)

	return &DB{Pool: pool}, nil
}

// Close closes the database connection pool.
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		slog.Info("db connection closed")
	}
}

// Health checks if the database connection is healthy.
func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
