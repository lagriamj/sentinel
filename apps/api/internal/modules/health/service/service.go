package service

import (
	"context"
	"errors"
	"fmt"
)

// ErrUnavailable means a required dependency did not answer.
var ErrUnavailable = errors.New("dependency unavailable")

// Pinger is a dependency the health check can reach.
type Pinger interface {
	Ping(context.Context) error
}

// Service decides whether the API should report itself healthy.
type Service struct {
	database Pinger
	redis    Pinger
}

// New builds a health service from its dependencies.
func New(database, redis Pinger) Service {
	return Service{database: database, redis: redis}
}

// Check pings PostgreSQL and Redis. Driver errors stay inside this package.
func (s Service) Check(ctx context.Context) error {
	if err := ping(ctx, s.database); err != nil {
		return fmt.Errorf("postgres: %w", ErrUnavailable)
	}
	if err := ping(ctx, s.redis); err != nil {
		return fmt.Errorf("redis: %w", ErrUnavailable)
	}
	return nil
}

func ping(ctx context.Context, pinger Pinger) error {
	if pinger == nil {
		return ErrUnavailable
	}
	return pinger.Ping(ctx)
}
