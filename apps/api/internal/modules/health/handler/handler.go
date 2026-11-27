package handler

import (
	"log/slog"
	"net/http"

	"sentinel-scanner/apps/api/internal/modules/health/model"
	"sentinel-scanner/apps/api/internal/modules/health/service"
	"sentinel-scanner/apps/api/internal/platform/httpx"
	"sentinel-scanner/apps/api/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

// Handler is the HTTP adapter for health checks.
type Handler struct {
	logger  *slog.Logger
	service service.Service
}

// New returns a health handler.
func New(logger *slog.Logger, health service.Service) Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return Handler{logger: logger, service: health}
}

// Get reports whether required dependencies are reachable.
func (h Handler) Get(c *gin.Context) {
	if err := h.service.Check(c.Request.Context()); err != nil {
		h.logger.Error("health_check_failed",
			"request_id", middleware.RequestIDFrom(c),
			"error", err.Error(),
		)
		httpx.Error(c, http.StatusServiceUnavailable, "unavailable", "service is unavailable")
		return
	}
	httpx.JSON(c, http.StatusOK, model.Status{Status: "ok"})
}
