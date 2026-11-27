package repository

import (
	"context"

	"sentinel-scanner/apps/api/internal/platform/postgres"

	"gorm.io/gorm"
)

// Postgres is the health module's database probe.
type Postgres struct {
	db *gorm.DB
}

// NewPostgres wraps the shared database pool.
func NewPostgres(db *gorm.DB) Postgres {
	return Postgres{db: db}
}

// Ping reports whether PostgreSQL accepts a connection.
func (r Postgres) Ping(ctx context.Context) error {
	return postgres.Ping(ctx, r.db)
}
