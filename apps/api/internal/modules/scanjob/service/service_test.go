package service

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"sentinel-scanner/apps/api/internal/modules/scanjob/model"
)

type storeStub struct {
	input   model.CreateInput
	filter  model.ListFilter
	id      string
	job     model.Job
	page    model.Page
	err     error
	creates int
	lists   int
	gets    int
	cancels int
	deletes int
}

func (s *storeStub) Create(_ context.Context, input model.CreateInput) (model.Job, error) {
	s.creates++
	s.input = input
	if s.err != nil {
		return model.Job{}, s.err
	}
	job := s.job
	job.Name = input.Name
	job.Target.URL = input.Target.URL
	job.Target.Host = input.Target.Host
	job.Target.Port = input.Target.Port
	job.Target.Scheme = input.Target.Scheme
	job.Status = model.StatusPending
	return job, nil
}

func (s *storeStub) List(_ context.Context, filter model.ListFilter) (model.Page, error) {
	s.lists++
	s.filter = filter
	if s.err != nil {
		return model.Page{}, s.err
	}
	page := s.page
	if page.Jobs == nil {
		page.Jobs = []model.Job{}
	}
	return page, nil
}

func (s *storeStub) Get(_ context.Context, _, id string) (model.Job, error) {
	s.gets++
	s.id = id
	if s.err != nil {
		return model.Job{}, s.err
	}
	return s.job, nil
}

func (s *storeStub) Cancel(_ context.Context, _, id string) (model.Job, error) {
	s.cancels++
	s.id = id
	if s.err != nil {
		return model.Job{}, s.err
	}
	job := s.job
	job.Status = model.StatusCancelled
	return job, nil
}

func (s *storeStub) Delete(_ context.Context, _, id string) error {
	s.deletes++
	s.id = id
	return s.err
}

type resolverStub struct {
	addresses []net.IPAddr
	err       error
	calls     int
	host      string
}

func (r *resolverStub) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	r.calls++
	r.host = host
	if r.err != nil {
		return nil, r.err
	}
	return r.addresses, nil
}

func publicResolver() *resolverStub {
	return &resolverStub{addresses: []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}}}
}

const registeredTargetID = "22222222-2222-4222-8222-222222222222"

type targetStub struct {
	target model.Target
	err    error
}

func (t targetStub) FindForMember(context.Context, string, string) (model.Target, error) {
	if t.err != nil {
		return model.Target{}, t.err
	}
	return t.target, nil
}

func registeredHost(url, host string, port int) targetStub {
	return targetStub{target: model.Target{
		ID:         registeredTargetID,
		URL:        url,
		Scheme:     "https",
		Host:       host,
		Port:       port,
		Authorized: true,
	}}
}

func TestCreateValidatesBeforeStore(t *testing.T) {
	store := &storeStub{}
	resolver := publicResolver()
	targets := registeredHost("https://example.com", "example.com", 443)
	svc := New(store, targets, resolver)

	tests := []struct {
		name    string
		request CreateRequest
		message string
	}{
		{name: "missing name", request: CreateRequest{TargetID: registeredTargetID}, message: "name is required"},
		{name: "blank name", request: CreateRequest{Name: "  ", TargetID: registeredTargetID}, message: "name is required"},
		{name: "long name", request: CreateRequest{Name: strings.Repeat("a", 201), TargetID: registeredTargetID}, message: "name must be 200 characters or fewer"},
		{name: "control character", request: CreateRequest{Name: "bad\nname", TargetID: registeredTargetID}, message: "name contains unsupported characters"},
		{name: "missing target", request: CreateRequest{Name: "Scan"}, message: "target id is required"},
		{name: "bad target", request: CreateRequest{Name: "Scan", TargetID: "not-a-uuid"}, message: "target id must be a uuid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store.creates = 0
			resolver.calls = 0
			_, err := svc.Create(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", test.request)
			var validation *ValidationError
			if !errors.As(err, &validation) || validation.Message != test.message {
				t.Fatalf("error = %v, want %s", err, test.message)
			}
			if store.creates != 0 || resolver.calls != 0 {
				t.Fatal("create looked up a target before the request was valid")
			}
		})
	}
}

func TestCreateUsesRegisteredTarget(t *testing.T) {
	store := &storeStub{job: model.Job{ID: "job-1"}}
	resolver := publicResolver()
	job, err := New(store, registeredHost("https://example.com/a", "example.com", 443), resolver).Create(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", CreateRequest{
		Name:     "  Homepage  ",
		TargetID: registeredTargetID,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if job.ID != "job-1" || job.Status != model.StatusPending || store.input.Name != "Homepage" || store.input.TargetID != registeredTargetID {
		t.Fatalf("job = %+v input = %+v", job, store.input)
	}
	if resolver.calls != 1 || resolver.host != "example.com" {
		t.Fatalf("resolver host = %q calls = %d", resolver.host, resolver.calls)
	}
}

func TestCreateAllowsPublicIPWithoutResolution(t *testing.T) {
	store := &storeStub{job: model.Job{ID: "job-1"}}
	resolver := &resolverStub{err: errors.New("dns should not be called")}
	_, err := New(store, registeredHost("https://203.0.113.10:8443/health", "203.0.113.10", 8443), resolver).Create(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", CreateRequest{
		Name:     "Edge",
		TargetID: registeredTargetID,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if resolver.calls != 0 {
		t.Fatal("literal address was resolved")
	}
}

func TestCreateRejectsHostnameThatResolvesPrivately(t *testing.T) {
	store := &storeStub{}
	resolver := &resolverStub{addresses: []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}, {IP: net.ParseIP("10.0.0.8")}}}
	_, err := New(store, registeredHost("https://rebind.example", "rebind.example", 443), resolver).Create(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", CreateRequest{
		Name:     "Mixed",
		TargetID: registeredTargetID,
	})
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Message != "target host is not allowed" || store.creates != 0 {
		t.Fatalf("error = %v creates = %d", err, store.creates)
	}
}

func TestCreateAllowsHostWhenResolverIsUnavailable(t *testing.T) {
	store := &storeStub{job: model.Job{ID: "job-1"}}
	resolver := &resolverStub{err: &net.DNSError{IsTemporary: true}}
	job, err := New(store, registeredHost("https://example.com", "example.com", 443), resolver).Create(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", CreateRequest{
		Name:     "Offline",
		TargetID: registeredTargetID,
	})
	if err != nil || job.ID != "job-1" || store.creates != 1 {
		t.Fatalf("job = %+v err = %v", job, err)
	}
}

func TestCreateRejectsUnresolvableHost(t *testing.T) {
	store := &storeStub{}
	resolver := &resolverStub{err: &net.DNSError{IsNotFound: true}}
	_, err := New(store, registeredHost("https://missing.example", "missing.example", 443), resolver).Create(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", CreateRequest{
		Name:     "Missing",
		TargetID: registeredTargetID,
	})
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Message != "target host could not be resolved" || strings.Contains(err.Error(), "DNSError") {
		t.Fatalf("error = %v", err)
	}
}

func TestCreateHidesTargetOutsideMembership(t *testing.T) {
	_, err := New(&storeStub{}, targetStub{err: model.ErrTargetNotFound}, publicResolver()).Create(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", CreateRequest{
		Name:     "Hidden",
		TargetID: registeredTargetID,
	})
	if !errors.Is(err, model.ErrTargetNotFound) {
		t.Fatalf("error = %v", err)
	}
}

func TestListNormalizesFilters(t *testing.T) {
	store := &storeStub{page: model.Page{Total: 3}}
	pageNumber := 2
	pageSize := 100
	page, err := New(store, nil, nil).List(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", ListRequest{
		Status:   "failed",
		TargetID: "AAAAAAAA-BBBB-4CCC-8DDD-EEEEEEEEEEEE",
		Page:     &pageNumber,
		PageSize: &pageSize,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if page.Page != 2 || page.PageSize != 100 || page.Total != 3 || page.Jobs == nil {
		t.Fatalf("page = %+v", page)
	}
	if store.filter.Status != model.StatusFailed || store.filter.TargetID != "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee" {
		t.Fatalf("filter = %+v", store.filter)
	}

	pageNumber = 0
	_, err = New(store, nil, nil).List(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", ListRequest{Page: &pageNumber})
	assertValidation(t, err, "page must be between 1 and 10000")

	pageSize = 101
	_, err = New(store, nil, nil).List(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", ListRequest{PageSize: &pageSize})
	assertValidation(t, err, "page_size must be between 1 and 100")

	_, err = New(store, nil, nil).List(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", ListRequest{Status: "done"})
	assertValidation(t, err, "status is invalid")

	_, err = New(store, nil, nil).List(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", ListRequest{TargetID: "not-a-uuid"})
	assertValidation(t, err, "target id must be a uuid")

	defaults, err := New(store, nil, nil).List(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", ListRequest{})
	if err != nil {
		t.Fatalf("default list: %v", err)
	}
	if defaults.Page != 1 || defaults.PageSize != 20 {
		t.Fatalf("defaults = %+v", defaults)
	}
}

func TestGetCancelAndDelete(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	store := &storeStub{job: model.Job{ID: id, Status: model.StatusPending}}
	svc := New(store, nil, nil)

	if _, err := svc.Get(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "bad"); err == nil || store.gets != 0 {
		t.Fatalf("get error = %v calls = %d", err, store.gets)
	}
	job, err := svc.Get(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", strings.ToUpper(id))
	if err != nil || job.ID != id || store.id != id {
		t.Fatalf("get = %+v id %s err %v", job, store.id, err)
	}

	cancelled, err := svc.Cancel(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", id)
	if err != nil || cancelled.Status != model.StatusCancelled || store.cancels != 1 {
		t.Fatalf("cancel = %+v err %v", cancelled, err)
	}
	store.err = model.ErrConflict
	_, err = svc.Cancel(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", id)
	var conflict *ConflictError
	if !errors.As(err, &conflict) || conflict.Message != "scan job cannot be cancelled from its current status" {
		t.Fatalf("cancel conflict = %v", err)
	}
	if !errors.Is(err, model.ErrConflict) {
		t.Fatal("conflict does not unwrap")
	}

	store.err = model.ErrNotFound
	if err := svc.Delete(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", id); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("delete error = %v", err)
	}
	store.err = model.ErrConflict
	err = svc.Delete(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", id)
	if !errors.As(err, &conflict) || conflict.Message != "scan job cannot be deleted while queued or running" {
		t.Fatalf("delete conflict = %v", err)
	}
	store.err = nil
	if err := svc.Delete(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "nope"); err == nil || store.deletes != 2 {
		t.Fatalf("invalid delete calls = %d err %v", store.deletes, err)
	}
}

func assertValidation(t *testing.T, err error, message string) {
	t.Helper()
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Message != message {
		t.Fatalf("error = %v, want %s", err, message)
	}
}
