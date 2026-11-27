package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRecoveryHidesPanicDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Recovery(slog.New(slog.NewTextHandler(io.Discard, nil))))
	router.GET("/panic", func(*gin.Context) {
		panic("database password=secret")
	})

	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
	if bodyContains(t, recorder.Body.Bytes(), "secret") {
		t.Fatalf("panic detail leaked: %s", recorder.Body.String())
	}
	assertErrorCode(t, recorder.Body.Bytes(), "internal_error")
}

func TestRequestIDIgnoresUnsafeHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/id", nil)
	request.Header.Set("X-Request-ID", "bad id\nwith space")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	got := recorder.Header().Get("X-Request-ID")
	if got == "" || got == "bad id\nwith space" {
		t.Fatalf("request id = %q", got)
	}
}

func assertErrorCode(t *testing.T, body []byte, code string) {
	t.Helper()
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if payload.Error.Code != code {
		t.Fatalf("code = %s", payload.Error.Code)
	}
}

func bodyContains(t *testing.T, body []byte, needle string) bool {
	t.Helper()
	return len(body) > 0 && contains(string(body), needle)
}

func contains(value, needle string) bool {
	return len(needle) == 0 || (len(value) >= len(needle) && search(value, needle))
}

func search(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
