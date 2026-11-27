package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	"sentinel-scanner/apps/api/internal/modules/scanjob/model"
	"sentinel-scanner/apps/api/internal/modules/scanjob/service"
	"sentinel-scanner/apps/api/internal/platform/authn"
	"sentinel-scanner/apps/api/internal/platform/httpx"
	"sentinel-scanner/apps/api/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

const maxJSONBytes = 64 * 1024

// Handler is the HTTP adapter for scan jobs.
type Handler struct {
	logger  *slog.Logger
	service service.Service
}

// New returns a scan job handler.
func New(logger *slog.Logger, jobs service.Service) Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return Handler{logger: logger, service: jobs}
}

// Register attaches the scan job routes to the versioned API.
func (h Handler) Register(v1 *gin.RouterGroup, authenticate gin.HandlerFunc) {
	group := v1.Group("/scan-jobs", authenticate)
	group.POST("", h.Create)
	group.GET("", h.List)
	group.GET("/:id", h.Get)
	group.POST("/:id/cancel", h.Cancel)
	group.DELETE("/:id", h.Delete)
}

type createRequest struct {
	Name     string `json:"name"`
	TargetID string `json:"target_id"`
}

// Create stores a pending scan job.
func (h Handler) Create(c *gin.Context) {
	var body createRequest
	if err := decodeJSON(c, &body); err != nil {
		h.writeError(c, err)
		return
	}
	ownerID, ok := authn.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	job, err := h.service.Create(c.Request.Context(), ownerID, service.CreateRequest{
		Name:     body.Name,
		TargetID: body.TargetID,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Header("Location", "/api/v1/scan-jobs/"+job.ID)
	httpx.JSON(c, http.StatusCreated, job)
}

// List returns a page of scan jobs.
func (h Handler) List(c *gin.Context) {
	request, err := listRequest(c)
	if err != nil {
		h.writeError(c, err)
		return
	}
	ownerID, ok := authn.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	page, err := h.service.List(c.Request.Context(), ownerID, request)
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, page)
}

// Get returns one scan job.
func (h Handler) Get(c *gin.Context) {
	ownerID, ok := authn.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	job, err := h.service.Get(c.Request.Context(), ownerID, c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, job)
}

// Cancel records a cancellable job as cancelled.
func (h Handler) Cancel(c *gin.Context) {
	ownerID, ok := authn.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	job, err := h.service.Cancel(c.Request.Context(), ownerID, c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, job)
}

// Delete removes a job that is not queued or running.
func (h Handler) Delete(c *gin.Context) {
	ownerID, ok := authn.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if err := h.service.Delete(c.Request.Context(), ownerID, c.Param("id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) writeError(c *gin.Context, err error) {
	var validation *service.ValidationError
	var conflict *service.ConflictError
	switch {
	case errors.As(err, &validation):
		httpx.Error(c, http.StatusBadRequest, "validation_error", validation.Message)
	case errors.As(err, &conflict):
		httpx.Error(c, http.StatusConflict, "conflict", conflict.Message)
	case errors.Is(err, model.ErrTargetNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "target not found")
	case errors.Is(err, model.ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "scan job not found")
	default:
		h.logger.Error("scan_job_request_failed",
			"request_id", middleware.RequestIDFrom(c),
			"error", err.Error(),
		)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func decodeJSON(c *gin.Context, dest any) error {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &service.ValidationError{Message: "content type must be application/json"}
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return &service.ValidationError{Message: "request body is invalid"}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &service.ValidationError{Message: "request body is invalid"}
	}
	return nil
}

func listRequest(c *gin.Context) (service.ListRequest, error) {
	var request service.ListRequest
	if raw := c.Query("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil {
			return service.ListRequest{}, &service.ValidationError{Message: "page must be between 1 and 10000"}
		}
		request.Page = &page
	}
	if raw := c.Query("page_size"); raw != "" {
		pageSize, err := strconv.Atoi(raw)
		if err != nil {
			return service.ListRequest{}, &service.ValidationError{Message: "page_size must be between 1 and 100"}
		}
		request.PageSize = &pageSize
	}
	request.Status = c.Query("status")
	request.TargetID = c.Query("target_id")
	return request, nil
}
