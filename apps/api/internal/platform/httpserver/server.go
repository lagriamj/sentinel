package httpserver

import (
	"errors"
	"log/slog"

	healthhandler "sentinel-scanner/apps/api/internal/modules/health/handler"
	"sentinel-scanner/apps/api/internal/platform/httpx"
	"sentinel-scanner/apps/api/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

type protectedRoutes interface {
	Register(v1 *gin.RouterGroup, authenticate gin.HandlerFunc)
}

// New wires middleware and versioned API routes.
func New(logger *slog.Logger, trustedProxies []string, health healthhandler.Handler, accounts, projects, scans protectedRoutes, authenticate gin.HandlerFunc) (*gin.Engine, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if gin.Mode() != gin.TestMode {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	if err := engine.SetTrustedProxies(trustedProxies); err != nil {
		return nil, errors.New("trusted proxy configuration is invalid")
	}
	engine.Use(
		middleware.Recovery(logger),
		middleware.RequestID(),
		middleware.SecurityHeaders(),
		middleware.AccessLog(logger),
	)
	engine.NoRoute(httpx.NotFound)

	v1 := engine.Group("/api/v1")
	v1.GET("/health", health.Get)
	accounts.Register(v1, authenticate)
	projects.Register(v1, authenticate)
	scans.Register(v1, authenticate)

	return engine, nil
}
