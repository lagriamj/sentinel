package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	"sentinel-scanner/apps/api/internal/modules/project/model"
	"sentinel-scanner/apps/api/internal/modules/project/service"
	"sentinel-scanner/apps/api/internal/platform/authn"
	"sentinel-scanner/apps/api/internal/platform/httpx"
	"sentinel-scanner/apps/api/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

const maxJSONBytes = 64 * 1024

// Handler is the HTTP adapter for projects, members, and targets.
type Handler struct {
	logger  *slog.Logger
	service service.Service
}

// New returns a project handler.
func New(logger *slog.Logger, projects service.Service) Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return Handler{logger: logger, service: projects}
}

// Register attaches the project routes.
func (h Handler) Register(v1 *gin.RouterGroup, authenticate gin.HandlerFunc) {
	group := v1.Group("/projects", authenticate)
	group.POST("", h.Create)
	group.GET("", h.List)
	group.GET("/:id", h.Get)
	group.PATCH("/:id", h.Update)
	group.DELETE("/:id", h.Delete)
	group.GET("/:id/members", h.ListMembers)
	group.POST("/:id/members", h.AddMember)
	group.DELETE("/:id/members/:userID", h.RemoveMember)
	group.GET("/:id/targets", h.ListTargets)
	group.POST("/:id/targets", h.RegisterTarget)
	group.DELETE("/:id/targets/:targetID", h.DeleteTarget)
}

type projectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type memberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type targetRequest struct {
	URL        string `json:"url"`
	Authorized *bool  `json:"authorized"`
}

// Create stores a project owned by the caller.
func (h Handler) Create(c *gin.Context) {
	var body projectRequest
	if err := decodeJSON(c, &body); err != nil {
		h.writeError(c, err)
		return
	}
	userID, ok := caller(c)
	if !ok {
		return
	}
	project, err := h.service.CreateProject(c.Request.Context(), userID, body.Name, body.Description)
	if err != nil {
		h.writeError(c, err)
		return
	}
	h.logger.Info("project_created", "request_id", middleware.RequestIDFrom(c), "project_id", project.ID, "user_id", userID)
	c.Header("Location", "/api/v1/projects/"+project.ID)
	httpx.JSON(c, http.StatusCreated, project)
}

// List returns the caller's projects.
func (h Handler) List(c *gin.Context) {
	request, err := pageRequest(c)
	if err != nil {
		h.writeError(c, err)
		return
	}
	userID, ok := caller(c)
	if !ok {
		return
	}
	page, err := h.service.ListProjects(c.Request.Context(), userID, request)
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, page)
}

// Get returns one project.
func (h Handler) Get(c *gin.Context) {
	userID, ok := caller(c)
	if !ok {
		return
	}
	project, err := h.service.GetProject(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, project)
}

// Update replaces the project name and description.
func (h Handler) Update(c *gin.Context) {
	var body projectRequest
	if err := decodeJSON(c, &body); err != nil {
		h.writeError(c, err)
		return
	}
	userID, ok := caller(c)
	if !ok {
		return
	}
	project, err := h.service.UpdateProject(c.Request.Context(), userID, c.Param("id"), body.Name, body.Description)
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, project)
}

// Delete removes a project that has no targets.
func (h Handler) Delete(c *gin.Context) {
	userID, ok := caller(c)
	if !ok {
		return
	}
	if err := h.service.DeleteProject(c.Request.Context(), userID, c.Param("id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListMembers returns the project membership.
func (h Handler) ListMembers(c *gin.Context) {
	userID, ok := caller(c)
	if !ok {
		return
	}
	members, err := h.service.ListMembers(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, members)
}

// AddMember assigns an existing account.
func (h Handler) AddMember(c *gin.Context) {
	var body memberRequest
	if err := decodeJSON(c, &body); err != nil {
		h.writeError(c, err)
		return
	}
	userID, ok := caller(c)
	if !ok {
		return
	}
	member, err := h.service.AddMember(c.Request.Context(), userID, c.Param("id"), body.Email, body.Role)
	if err != nil {
		h.writeError(c, err)
		return
	}
	h.logger.Info("project_member_added", "request_id", middleware.RequestIDFrom(c), "project_id", c.Param("id"), "member_id", member.UserID)
	httpx.JSON(c, http.StatusCreated, member)
}

// RemoveMember removes one membership.
func (h Handler) RemoveMember(c *gin.Context) {
	userID, ok := caller(c)
	if !ok {
		return
	}
	if err := h.service.RemoveMember(c.Request.Context(), userID, c.Param("id"), c.Param("userID")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListTargets returns registered targets.
func (h Handler) ListTargets(c *gin.Context) {
	request, err := pageRequest(c)
	if err != nil {
		h.writeError(c, err)
		return
	}
	userID, ok := caller(c)
	if !ok {
		return
	}
	page, err := h.service.ListTargets(c.Request.Context(), userID, c.Param("id"), request)
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, page)
}

// RegisterTarget stores an authorized target.
func (h Handler) RegisterTarget(c *gin.Context) {
	var body targetRequest
	if err := decodeJSON(c, &body); err != nil {
		h.writeError(c, err)
		return
	}
	userID, ok := caller(c)
	if !ok {
		return
	}
	target, err := h.service.RegisterTarget(c.Request.Context(), userID, c.Param("id"), body.URL, body.Authorized)
	if err != nil {
		h.writeError(c, err)
		return
	}
	h.logger.Info("target_registered", "request_id", middleware.RequestIDFrom(c), "project_id", c.Param("id"), "target_id", target.ID)
	c.Header("Location", "/api/v1/projects/"+c.Param("id")+"/targets/"+target.ID)
	httpx.JSON(c, http.StatusCreated, target)
}

// DeleteTarget removes an unused target.
func (h Handler) DeleteTarget(c *gin.Context) {
	userID, ok := caller(c)
	if !ok {
		return
	}
	if err := h.service.DeleteTarget(c.Request.Context(), userID, c.Param("id"), c.Param("targetID")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func caller(c *gin.Context) (string, bool) {
	userID, ok := authn.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return "", false
	}
	return userID, true
}

func (h Handler) writeError(c *gin.Context, err error) {
	var validation *service.ValidationError
	var conflict *service.ConflictError
	switch {
	case errors.As(err, &validation):
		httpx.Error(c, http.StatusBadRequest, "validation_error", validation.Message)
	case errors.As(err, &conflict):
		httpx.Error(c, http.StatusConflict, "conflict", conflict.Message)
	case errors.Is(err, model.ErrForbidden):
		httpx.Error(c, http.StatusForbidden, "forbidden", "you do not have permission to change this project")
	case errors.Is(err, model.ErrUserNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "no account uses that email")
	case errors.Is(err, model.ErrMemberNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "member not found")
	case errors.Is(err, model.ErrTargetNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "target not found")
	case errors.Is(err, model.ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "project not found")
	default:
		h.logger.Error("project_request_failed",
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

func pageRequest(c *gin.Context) (service.PageRequest, error) {
	var request service.PageRequest
	if raw := c.Query("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil {
			return service.PageRequest{}, &service.ValidationError{Message: "page must be between 1 and 10000"}
		}
		request.Page = &page
	}
	if raw := c.Query("page_size"); raw != "" {
		pageSize, err := strconv.Atoi(raw)
		if err != nil {
			return service.PageRequest{}, &service.ValidationError{Message: "page_size must be between 1 and 100"}
		}
		request.PageSize = &pageSize
	}
	return request, nil
}
