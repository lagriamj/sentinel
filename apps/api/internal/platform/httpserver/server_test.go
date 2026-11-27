package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	healthhandler "sentinel-scanner/apps/api/internal/modules/health/handler"
	healthservice "sentinel-scanner/apps/api/internal/modules/health/service"
	scanhandler "sentinel-scanner/apps/api/internal/modules/scanjob/handler"
	"sentinel-scanner/apps/api/internal/platform/httpx"
	"sentinel-scanner/apps/api/internal/modules/scanjob/model"
	scanservice "sentinel-scanner/apps/api/internal/modules/scanjob/service"

	"github.com/gin-gonic/gin"
)

type stub struct{}

func (stub) Ping(context.Context) error { return nil }

func TestHealthRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router, err := New(logger, nil, healthHandler(logger), routeStub{}, routeStub{}, scanHandler(logger), denyAuth)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff header")
	}
}

func TestUnknownRouteUsesErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router, err := New(logger, nil, healthHandler(logger), routeStub{}, routeStub{}, scanHandler(logger), denyAuth)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d", recorder.Code)
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Error.Code != "not_found" {
		t.Fatalf("code = %s", payload.Error.Code)
	}
}

func TestScanJobRouteRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router, err := New(logger, nil, healthHandler(logger), routeStub{}, routeStub{}, scanHandler(logger), denyAuth)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/scan-jobs", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func healthHandler(logger *slog.Logger) healthhandler.Handler {
	return healthhandler.New(logger, healthservice.New(stub{}, stub{}))
}

type routeStub struct{}

func (routeStub) Register(*gin.RouterGroup, gin.HandlerFunc) {}

func denyAuth(c *gin.Context) {
	httpx.Error(c, http.StatusUnauthorized, "unauthorized", "authentication is required")
}

func scanHandler(logger *slog.Logger) scanhandler.Handler {
	return scanhandler.New(logger, scanservice.New(scanStore{}, nil, nil))
}

type scanStore struct{}

func (scanStore) Create(context.Context, model.CreateInput) (model.Job, error) {
	return model.Job{}, errors.New("not implemented")
}

func (scanStore) List(context.Context, model.ListFilter) (model.Page, error) {
	return model.Page{}, nil
}

func (scanStore) Get(context.Context, string, string) (model.Job, error) {
	return model.Job{}, errors.New("not implemented")
}

func (scanStore) Cancel(context.Context, string, string) (model.Job, error) {
	return model.Job{}, errors.New("not implemented")
}

func (scanStore) Delete(context.Context, string, string) error {
	return errors.New("not implemented")
}
