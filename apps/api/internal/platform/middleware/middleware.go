package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"time"

	"sentinel-scanner/apps/api/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

const requestIDHeader = "X-Request-ID"

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestID accepts a safe client-supplied id or generates one.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if !requestIDPattern.MatchString(id) {
			id = newRequestID()
		}
		c.Set(string(requestIDKey), id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

// RequestIDFrom returns the id stored by RequestID.
func RequestIDFrom(c *gin.Context) string {
	value, _ := c.Get(string(requestIDKey))
	id, _ := value.(string)
	return id
}

// Recovery logs panics and returns the shared error envelope.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			logger.Error("panic", "request_id", RequestIDFrom(c))
			httpx.Error(c, http.StatusInternalServerError, "internal_error", "internal server error")
		}()
		c.Next()
	}
}

// SecurityHeaders sets response headers that do not depend on the route.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}

// AccessLog writes one JSON record per request.
func AccessLog(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		if c.Request.URL.Path == "/api/v1/health" {
			return
		}
		logger.Info("request",
			"request_id", RequestIDFrom(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(started).Milliseconds(),
		)
	}
}

// RateLimit rejects callers that exceed limit requests inside window.
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	type bucket struct {
		count int
		reset time.Time
	}
	var mu sync.Mutex
	buckets := map[string]bucket{}
	return func(c *gin.Context) {
		if limit < 1 || window <= 0 {
			c.Next()
			return
		}
		now := time.Now()
		key := c.ClientIP()
		mu.Lock()
		current := buckets[key]
		if now.After(current.reset) {
			current = bucket{reset: now.Add(window)}
		}
		current.count++
		buckets[key] = current
		allowed := current.count <= limit
		mu.Unlock()
		if !allowed {
			httpx.Error(c, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}
		c.Next()
	}
}

func newRequestID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(raw[:])
}
