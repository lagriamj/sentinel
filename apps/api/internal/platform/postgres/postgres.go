package postgres

import (
	"context"
	"errors"
	"net/url"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open connects to PostgreSQL and checks that the server answers.
// Driver errors are not returned to callers because they can contain the URL.
func Open(ctx context.Context, databaseURL string) (*gorm.DB, error) {
	if err := validateURL(databaseURL); err != nil {
		return nil, err
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, errors.New("database connection failed")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.New("database pool unavailable")
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := Ping(ctx, db); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// Ping reports whether the pool can reach PostgreSQL.
func Ping(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("database pool unavailable")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return errors.New("database pool unavailable")
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return errors.New("database ping failed")
	}
	return nil
}

func validateURL(databaseURL string) error {
	if databaseURL == "" {
		return errors.New("database url is required")
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil || parsed.Host == "" {
		return errors.New("database url is invalid")
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return errors.New("database url scheme must be postgres")
	}
	return nil
}
