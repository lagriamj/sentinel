package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"sentinel-scanner/apps/api/internal/modules/health/service"

	"github.com/gin-gonic/gin"
)

type stub struct {
	err error
}

func (s stub) Ping(context.Context) error {
	return s.err
}

func TestGetHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("ok", func(t *testing.T) {
		recorder := serve(logger, service.New(stub{}, stub{}))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
		}
		var payload struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if payload.Status != "ok" {
			t.Fatalf("status payload = %s", payload.Status)
		}
	})

	t.Run("dependency down", func(t *testing.T) {
		recorder := serve(logger, service.New(stub{err: context.DeadlineExceeded}, stub{}))
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
		}
		var payload struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if payload.Error.Code != "unavailable" {
			t.Fatalf("code = %s", payload.Error.Code)
		}
		if payload.Error.Message != "service is unavailable" {
			t.Fatalf("message = %s", payload.Error.Message)
		}
	})
}

func serve(logger *slog.Logger, health service.Service) *httptest.ResponseRecorder {
	router := gin.New()
	router.GET("/api/v1/health", New(logger, health).Get)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
