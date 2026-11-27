package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"sentinel-scanner/apps/api/internal/modules/scanjob/model"
	"sentinel-scanner/apps/api/internal/modules/scanjob/service"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

type storeStub struct {
	job    model.Job
	page   model.Page
	err    error
	filter model.ListFilter
	input  model.CreateInput
}

func (s *storeStub) Create(_ context.Context, input model.CreateInput) (model.Job, error) {
	s.input = input
	if s.err != nil {
		return model.Job{}, s.err
	}
	job := s.job
	job.Name = input.Name
	job.Status = model.StatusPending
	job.Target.URL = input.Target.URL
	job.Target.Scheme = input.Target.Scheme
	job.Target.Host = input.Target.Host
	job.Target.Port = input.Target.Port
	job.Target.Authorized = true
	return job, nil
}

func (s *storeStub) List(_ context.Context, filter model.ListFilter) (model.Page, error) {
	s.filter = filter
	if s.err != nil {
		return model.Page{}, s.err
	}
	if s.page.Jobs == nil {
		s.page.Jobs = []model.Job{}
	}
	return s.page, nil
}

func (s *storeStub) Get(_ context.Context, _, id string) (model.Job, error) {
	if s.err != nil {
		return model.Job{}, s.err
	}
	job := s.job
	job.ID = id
	return job, nil
}

func (s *storeStub) Cancel(_ context.Context, _, id string) (model.Job, error) {
	if s.err != nil {
		return model.Job{}, s.err
	}
	job := s.job
	job.ID = id
	job.Status = model.StatusCancelled
	return job, nil
}

func (s *storeStub) Delete(context.Context, string, string) error {
	return s.err
}

type resolverStub struct{}

func (resolverStub) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}}, nil
}

func TestScanJobRoutes(t *testing.T) {
	now := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	job := model.Job{
		ID:        "11111111-1111-4111-8111-111111111111",
		Name:      "Homepage",
		Status:    model.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
		Target: model.Target{
			ID:         "22222222-2222-4222-8222-222222222222",
			URL:        "https://example.com/a",
			Scheme:     "https",
			Host:       "example.com",
			Port:       443,
			Authorized: true,
		},
	}
	store := &storeStub{job: job, page: model.Page{Total: 1, Jobs: []model.Job{job}}}
	router := testRouter(store)

	t.Run("create", func(t *testing.T) {
		recorder := perform(router, http.MethodPost, "/api/v1/scan-jobs", `{"name":" Homepage ","target_id":"22222222-2222-4222-8222-222222222222"}`)
		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
		}
		if recorder.Header().Get("Location") != "/api/v1/scan-jobs/"+job.ID {
			t.Fatalf("location = %s", recorder.Header().Get("Location"))
		}
		var payload model.Job
		decode(t, recorder, &payload)
		if payload.Status != model.StatusPending || payload.Target.URL != "https://example.com/a" || payload.Target.Authorized != true {
			t.Fatalf("payload = %+v", payload)
		}
	})

	t.Run("list", func(t *testing.T) {
		recorder := perform(router, http.MethodGet, "/api/v1/scan-jobs?status=pending&page=2&page_size=10&target_id=22222222-2222-4222-8222-222222222222", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
		}
		var payload struct {
			Data     []model.Job `json:"data"`
			Page     int         `json:"page"`
			PageSize int         `json:"page_size"`
			Total    int64       `json:"total"`
		}
		decode(t, recorder, &payload)
		if payload.Page != 2 || payload.PageSize != 10 || payload.Total != 1 || len(payload.Data) != 1 {
			t.Fatalf("payload = %+v", payload)
		}
		if store.filter.Status != model.StatusPending || store.filter.Page != 2 || store.filter.TargetID != job.Target.ID {
			t.Fatalf("filter = %+v", store.filter)
		}
	})

	t.Run("get", func(t *testing.T) {
		recorder := perform(router, http.MethodGet, "/api/v1/scan-jobs/"+job.ID, "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("cancel", func(t *testing.T) {
		recorder := perform(router, http.MethodPost, "/api/v1/scan-jobs/"+job.ID+"/cancel", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
		}
		var payload model.Job
		decode(t, recorder, &payload)
		if payload.Status != model.StatusCancelled {
			t.Fatalf("status = %s", payload.Status)
		}
	})

	t.Run("delete", func(t *testing.T) {
		recorder := perform(router, http.MethodDelete, "/api/v1/scan-jobs/"+job.ID, "")
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
		}
		if recorder.Body.Len() != 0 {
			t.Fatalf("body = %s", recorder.Body.String())
		}
	})
}

func TestScanJobValidationAndErrors(t *testing.T) {
	jobID := "11111111-1111-4111-8111-111111111111"
	store := &storeStub{}
	router := testRouter(store)

	cases := []struct {
		name    string
		method  string
		path    string
		body    string
		status  int
		code    string
		message string
		prepare func()
	}{
		{
			name:    "malformed json",
			method:  http.MethodPost,
			path:    "/api/v1/scan-jobs",
			body:    "{",
			status:  http.StatusBadRequest,
			code:    "validation_error",
			message: "request body is invalid",
		},
		{
			name:    "unknown field",
			method:  http.MethodPost,
			path:    "/api/v1/scan-jobs",
			body:    `{"name":"Scan","target_id":"22222222-2222-4222-8222-222222222222","status":"running"}`,
			status:  http.StatusBadRequest,
			code:    "validation_error",
			message: "request body is invalid",
		},
		{
			name:    "missing target",
			method:  http.MethodPost,
			path:    "/api/v1/scan-jobs",
			body:    `{"name":"Scan"}`,
			status:  http.StatusBadRequest,
			code:    "validation_error",
			message: "target id is required",
		},
		{
			name:    "bad target",
			method:  http.MethodPost,
			path:    "/api/v1/scan-jobs",
			body:    `{"name":"Scan","target_id":"nope"}`,
			status:  http.StatusBadRequest,
			code:    "validation_error",
			message: "target id must be a uuid",
		},
		{
			name:    "bad page",
			method:  http.MethodGet,
			path:    "/api/v1/scan-jobs?page=0",
			status:  http.StatusBadRequest,
			code:    "validation_error",
			message: "page must be between 1 and 10000",
		},
		{
			name:    "bad status",
			method:  http.MethodGet,
			path:    "/api/v1/scan-jobs?status=done",
			status:  http.StatusBadRequest,
			code:    "validation_error",
			message: "status is invalid",
		},
		{
			name:    "bad id",
			method:  http.MethodGet,
			path:    "/api/v1/scan-jobs/not-a-uuid",
			status:  http.StatusBadRequest,
			code:    "validation_error",
			message: "scan job id must be a uuid",
		},
		{
			name:    "missing job",
			method:  http.MethodGet,
			path:    "/api/v1/scan-jobs/" + jobID,
			status:  http.StatusNotFound,
			code:    "not_found",
			message: "scan job not found",
			prepare: func() { store.err = model.ErrNotFound },
		},
		{
			name:    "cancel conflict",
			method:  http.MethodPost,
			path:    "/api/v1/scan-jobs/" + jobID + "/cancel",
			status:  http.StatusConflict,
			code:    "conflict",
			message: "scan job cannot be cancelled from its current status",
			prepare: func() { store.err = model.ErrConflict },
		},
		{
			name:    "delete conflict",
			method:  http.MethodDelete,
			path:    "/api/v1/scan-jobs/" + jobID,
			status:  http.StatusConflict,
			code:    "conflict",
			message: "scan job cannot be deleted while queued or running",
			prepare: func() { store.err = model.ErrConflict },
		},
		{
			name:    "database failure",
			method:  http.MethodDelete,
			path:    "/api/v1/scan-jobs/" + jobID,
			status:  http.StatusInternalServerError,
			code:    "internal_error",
			message: "internal server error",
			prepare: func() { store.err = errors.New("pq: password=secret syntax error") },
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			store.err = nil
			if test.prepare != nil {
				test.prepare()
			}
			recorder := perform(router, test.method, test.path, test.body)
			if recorder.Code != test.status {
				t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
			}
			assertError(t, recorder, test.code, test.message)
			if strings.Contains(recorder.Body.String(), "password=secret") || strings.Contains(recorder.Body.String(), "syntax") {
				t.Fatalf("internal error leaked: %s", recorder.Body.String())
			}
		})
	}
}

func TestCreateRequiresJSONContentType(t *testing.T) {
	router := testRouter(&storeStub{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/scan-jobs", strings.NewReader(`{"name":"Scan","target_id":"22222222-2222-4222-8222-222222222222"}`))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
	assertError(t, recorder, "validation_error", "content type must be application/json")
}

type targetStub struct {
	target model.Target
}

func (t targetStub) FindForMember(context.Context, string, string) (model.Target, error) {
	if t.target.URL == "" {
		t.target = model.Target{
			ID:         "22222222-2222-4222-8222-222222222222",
			URL:        "https://example.com/a",
			Scheme:     "https",
			Host:       "example.com",
			Port:       443,
			Authorized: true,
		}
	}
	return t.target, nil
}

func testRouter(store *storeStub) *gin.Engine {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := gin.New()
	handler := New(logger, service.New(store, targetStub{target: store.job.Target}, resolverStub{}))
	handler.Register(router.Group("/api/v1"), func(c *gin.Context) {
		c.Set("auth_user_id", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
		c.Next()
	})
	return router
}

func perform(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	if body != "" {
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode: %v body = %s", err, recorder.Body.String())
	}
}

func assertError(t *testing.T, recorder *httptest.ResponseRecorder, code, message string) {
	t.Helper()
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	decode(t, recorder, &payload)
	if payload.Error.Code != code || payload.Error.Message != message {
		t.Fatalf("error = %+v", payload.Error)
	}
}
