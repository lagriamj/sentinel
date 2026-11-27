package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"sentinel-scanner/apps/api/internal/modules/auth/service"
	"sentinel-scanner/apps/api/internal/platform/authn"
	"sentinel-scanner/apps/api/internal/platform/httpx"
	"sentinel-scanner/apps/api/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

const (
	maxJSONBytes = 16 * 1024
	cookieName   = "sentinel_refresh"
	cookiePath   = "/api/v1/auth"
)

// Handler is the HTTP adapter for operator accounts.
type Handler struct {
	logger *slog.Logger
	auth   service.Service
	secure bool
}

// New returns an auth handler. secure marks the refresh cookie HTTPS-only.
func New(logger *slog.Logger, auth service.Service, secure bool) Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return Handler{logger: logger, auth: auth, secure: secure}
}

// Register attaches the account routes. Login and registration are rate limited.
func (h Handler) Register(v1 *gin.RouterGroup, authenticate gin.HandlerFunc) {
	group := v1.Group("/auth")
	limited := middleware.RateLimit(10, time.Minute)
	group.POST("/register", limited, h.RegisterAccount)
	group.POST("/login", limited, h.Login)
	group.POST("/refresh", h.Refresh)
	group.POST("/logout", h.Logout)
	group.GET("/session", authenticate, h.Session)
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type sessionResponse struct {
	AccessToken string          `json:"access_token"`
	TokenType   string          `json:"token_type"`
	ExpiresIn   int             `json:"expires_in"`
	User        service.Account `json:"user"`
}

// RegisterAccount creates an operator and sets a refresh cookie.
func (h Handler) RegisterAccount(c *gin.Context) {
	var body credentialsRequest
	if err := decodeJSON(c, &body); err != nil {
		h.writeError(c, err)
		return
	}
	result, err := h.auth.Register(c.Request.Context(), body.Email, body.Password)
	if err != nil {
		h.writeError(c, err)
		return
	}
	h.logger.Info("auth_registered", "request_id", middleware.RequestIDFrom(c), "user_id", result.Account.ID)
	h.writeSession(c, http.StatusCreated, result)
}

// Login checks credentials and sets a refresh cookie.
func (h Handler) Login(c *gin.Context) {
	var body credentialsRequest
	if err := decodeJSON(c, &body); err != nil {
		h.writeError(c, err)
		return
	}
	result, err := h.auth.Login(c.Request.Context(), body.Email, body.Password)
	if err != nil {
		if errors.Is(err, service.CredentialsError{}) {
			h.logger.Warn("auth_login_rejected", "request_id", middleware.RequestIDFrom(c))
		}
		h.writeError(c, err)
		return
	}
	h.logger.Info("auth_login", "request_id", middleware.RequestIDFrom(c), "user_id", result.Account.ID)
	h.writeSession(c, http.StatusOK, result)
}

// Refresh rotates the refresh cookie.
func (h Handler) Refresh(c *gin.Context) {
	raw, _ := c.Cookie(cookieName)
	result, err := h.auth.Refresh(c.Request.Context(), raw)
	if err != nil {
		h.clearCookie(c)
		if errors.Is(err, service.CredentialsError{}) {
			h.logger.Warn("auth_refresh_rejected", "request_id", middleware.RequestIDFrom(c))
		}
		h.writeError(c, err)
		return
	}
	h.logger.Info("auth_refresh", "request_id", middleware.RequestIDFrom(c), "user_id", result.Account.ID)
	h.writeSession(c, http.StatusOK, result)
}

// Logout revokes the refresh cookie.
func (h Handler) Logout(c *gin.Context) {
	raw, _ := c.Cookie(cookieName)
	if err := h.auth.Logout(c.Request.Context(), raw); err != nil {
		h.writeError(c, err)
		return
	}
	h.clearCookie(c)
	h.logger.Info("auth_logout", "request_id", middleware.RequestIDFrom(c))
	c.Status(http.StatusNoContent)
}

// Session returns the authenticated operator.
func (h Handler) Session(c *gin.Context) {
	userID, ok := authn.UserID(c)
	if !ok {
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	account, err := h.auth.Session(c.Request.Context(), userID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, account)
}

func (h Handler) writeSession(c *gin.Context, status int, result service.Result) {
	h.setCookie(c, result.Session.RefreshToken, result.Session.RefreshExpires)
	httpx.JSON(c, status, sessionResponse{
		AccessToken: result.Session.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   result.Session.ExpiresIn,
		User:        result.Account,
	})
}

func (h Handler) setCookie(c *gin.Context, token string, expires time.Time) {
	maxAge := int(time.Until(expires).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cookieName, token, maxAge, cookiePath, "", h.secure, true)
}

func (h Handler) clearCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cookieName, "", -1, cookiePath, "", h.secure, true)
}

func (h Handler) writeError(c *gin.Context, err error) {
	var validation *service.ValidationError
	var conflict *service.ConflictError
	switch {
	case errors.As(err, &validation):
		httpx.Error(c, http.StatusBadRequest, "validation_error", validation.Message)
	case errors.As(err, &conflict):
		httpx.Error(c, http.StatusConflict, "conflict", conflict.Message)
	case errors.Is(err, service.CredentialsError{}):
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "invalid email or password")
	default:
		h.logger.Error("auth_request_failed",
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
