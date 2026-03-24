package store

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// New opens a GORM database connection for the given DSN.
// The context is accepted for API symmetry but GORM's Open is synchronous.
func New(_ context.Context, dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("store: open db: %w", err)
	}
	return db, nil
}

// AutoMigrate creates or updates all tables managed by this package.
// It enables the pgvector extension first, then runs GORM AutoMigrate
// for MerkleSnapshot, Chunk, and Job.
func AutoMigrate(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		return fmt.Errorf("store: create vector extension: %w", err)
	}
	if err := db.AutoMigrate(&MerkleSnapshot{}, &Chunk{}, &Job{}); err != nil {
		return fmt.Errorf("store: auto migrate: %w", err)
	}
	return nil
}
