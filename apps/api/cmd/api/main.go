package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authhandler "sentinel-scanner/apps/api/internal/modules/auth/handler"
	authrepo "sentinel-scanner/apps/api/internal/modules/auth/repository"
	authservice "sentinel-scanner/apps/api/internal/modules/auth/service"
	healthhandler "sentinel-scanner/apps/api/internal/modules/health/handler"
	healthrepo "sentinel-scanner/apps/api/internal/modules/health/repository"
	healthservice "sentinel-scanner/apps/api/internal/modules/health/service"
	projecthandler "sentinel-scanner/apps/api/internal/modules/project/handler"
	projectmodel "sentinel-scanner/apps/api/internal/modules/project/model"
	projectrepo "sentinel-scanner/apps/api/internal/modules/project/repository"
	projectservice "sentinel-scanner/apps/api/internal/modules/project/service"
	scanhandler "sentinel-scanner/apps/api/internal/modules/scanjob/handler"
	scanmodel "sentinel-scanner/apps/api/internal/modules/scanjob/model"
	scanrepo "sentinel-scanner/apps/api/internal/modules/scanjob/repository"
	scanservice "sentinel-scanner/apps/api/internal/modules/scanjob/service"
	"sentinel-scanner/apps/api/internal/platform/authn"
	"sentinel-scanner/apps/api/internal/platform/config"
	"sentinel-scanner/apps/api/internal/platform/httpserver"
	"sentinel-scanner/apps/api/internal/platform/logging"
	"sentinel-scanner/apps/api/internal/platform/postgres"
	"sentinel-scanner/apps/api/internal/platform/queue"
)

type targetGate struct {
	projects projectservice.Service
}

func (g targetGate) FindForMember(ctx context.Context, userID, targetID string) (scanmodel.Target, error) {
	target, err := g.projects.FindForMember(ctx, userID, targetID)
	if errors.Is(err, projectmodel.ErrTargetNotFound) {
		return scanmodel.Target{}, scanmodel.ErrTargetNotFound
	}
	if err != nil {
		return scanmodel.Target{}, err
	}
	return scanmodel.Target{
		ID:         target.ID,
		ProjectID:  target.ProjectID,
		URL:        target.URL,
		Scheme:     target.Scheme,
		Host:       target.Host,
		Port:       target.Port,
		Authorized: target.Authorized,
	}, nil
}

func main() {
	logger := logging.New()
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration_invalid", "error", err.Error())
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, logger, cfg); err != nil {
		logger.Error("server_stopped", "error", err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger, cfg config.Config) error {
	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	jobs, err := queue.Open(ctx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer jobs.Close()

	health := healthhandler.New(logger, healthservice.New(healthrepo.NewPostgres(db), jobs))
	accounts := authservice.New(authrepo.NewPostgres(db), authservice.Config{
		Secret:     []byte(cfg.AuthSecret),
		AccessTTL:  cfg.AccessTokenTTL,
		RefreshTTL: cfg.RefreshTokenTTL,
	})
	authHTTP := authhandler.New(logger, accounts, cfg.CookieSecure)
	projects := projectservice.New(projectrepo.NewPostgres(db), nil)
	projectHTTP := projecthandler.New(logger, projects)
	scans := scanhandler.New(logger, scanservice.New(scanrepo.NewPostgres(db), targetGate{projects: projects}, nil))
	router, err := httpserver.New(logger, cfg.TrustedProxies, health, authHTTP, projectHTTP, scans, authn.Authenticate(accounts))
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	logger.Info("api_listening", "addr", cfg.HTTPAddr)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
