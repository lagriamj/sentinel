package service

import (
	"context"
	"errors"
	"net"
	"strings"
	"unicode"
	"unicode/utf8"

	"sentinel-scanner/apps/api/internal/modules/scanjob/model"
	"sentinel-scanner/apps/api/internal/scope"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
	maxPage         = 10000
	maxNameLength = 200
)

// ValidationError is a request the API rejects before touching stored state.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil || e.Message == "" {
		return "request is invalid"
	}
	return e.Message
}

// ConflictError is a change the current job status does not allow.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	if e == nil || e.Message == "" {
		return model.ErrConflict.Error()
	}
	return e.Message
}

// Unwrap lets callers detect a status conflict with errors.Is.
func (e *ConflictError) Unwrap() error {
	return model.ErrConflict
}

// TargetSource loads a target the caller is allowed to assess.
type TargetSource interface {
	FindForMember(ctx context.Context, userID, targetID string) (model.Target, error)
}

// Store persists scan jobs.
type Store interface {
	Create(ctx context.Context, input model.CreateInput) (model.Job, error)
	List(ctx context.Context, filter model.ListFilter) (model.Page, error)
	Get(ctx context.Context, ownerID, id string) (model.Job, error)
	Cancel(ctx context.Context, ownerID, id string) (model.Job, error)
	Delete(ctx context.Context, ownerID, id string) error
}

// Resolver looks up the addresses of a target hostname.
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// CreateRequest is the operator input for a new scan job.
type CreateRequest struct {
	Name     string
	TargetID string
}

// ListRequest is the operator input for a job collection.
// Nil page fields mean the collection defaults.
type ListRequest struct {
	Status   string
	TargetID string
	Page     *int
	PageSize *int
}

// Service owns scan job rules. It stores pending jobs and does not publish work.
type Service struct {
	store    Store
	targets  TargetSource
	resolver Resolver
}

// New builds a scan job service. A nil resolver uses the process resolver.
func New(store Store, targets TargetSource, resolver Resolver) Service {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	return Service{store: store, targets: targets, resolver: resolver}
}

// Create stores a pending scan job for a target the caller may assess.
// The stored target is checked again before the job is written.
func (s Service) Create(ctx context.Context, ownerID string, request CreateRequest) (model.Job, error) {
	if s.store == nil || s.targets == nil {
		return model.Job{}, errors.New("scan job store is unavailable")
	}
	name, err := normalizeName(request.Name)
	if err != nil {
		return model.Job{}, err
	}
	owner, err := requireOwner(ownerID)
	if err != nil {
		return model.Job{}, err
	}
	if request.TargetID == "" {
		return model.Job{}, invalid("target id is required")
	}
	targetID, err := model.ParseID(request.TargetID)
	if err != nil {
		return model.Job{}, invalid("target id must be a uuid")
	}
	registered, err := s.targets.FindForMember(ctx, owner, targetID)
	if err != nil {
		if errors.Is(err, model.ErrTargetNotFound) || errors.Is(err, model.ErrNotFound) {
			return model.Job{}, model.ErrTargetNotFound
		}
		return model.Job{}, err
	}
	if !registered.Authorized {
		return model.Job{}, invalid("target is not authorized")
	}
	parsed, hostIsIP, err := scope.Parse(registered.URL)
	if err != nil {
		return model.Job{}, asValidation(err)
	}
	if !hostIsIP {
		if err := scope.Confirm(ctx, s.resolver, parsed.Host); err != nil {
			return model.Job{}, asValidation(err)
		}
	}
	return s.store.Create(ctx, model.CreateInput{
		OwnerID:  owner,
		Name:     name,
		TargetID: registered.ID,
		Target: model.TargetInput{
			URL:    parsed.URL,
			Scheme: parsed.Scheme,
			Host:   parsed.Host,
			Port:   parsed.Port,
		},
	})
}

// List returns one page of scan jobs.
func (s Service) List(ctx context.Context, ownerID string, request ListRequest) (model.Page, error) {
	if s.store == nil {
		return model.Page{}, errors.New("scan job store is unavailable")
	}
	filter, err := normalizeList(request)
	if err != nil {
		return model.Page{}, err
	}
	filter.OwnerID, err = requireOwner(ownerID)
	if err != nil {
		return model.Page{}, err
	}
	page, err := s.store.List(ctx, filter)
	if err != nil {
		return model.Page{}, err
	}
	if page.Jobs == nil {
		page.Jobs = []model.Job{}
	}
	page.Page = filter.Page
	page.PageSize = filter.PageSize
	return page, nil
}

// Get returns one scan job.
func (s Service) Get(ctx context.Context, ownerID, id string) (model.Job, error) {
	if err := s.ready(); err != nil {
		return model.Job{}, err
	}
	owner, err := requireOwner(ownerID)
	if err != nil {
		return model.Job{}, err
	}
	parsed, err := parseJobID(id)
	if err != nil {
		return model.Job{}, err
	}
	return s.store.Get(ctx, owner, parsed)
}

// Cancel moves a pending, queued, or running job to cancelled.
func (s Service) Cancel(ctx context.Context, ownerID, id string) (model.Job, error) {
	if err := s.ready(); err != nil {
		return model.Job{}, err
	}
	owner, err := requireOwner(ownerID)
	if err != nil {
		return model.Job{}, err
	}
	parsed, err := parseJobID(id)
	if err != nil {
		return model.Job{}, err
	}
	job, err := s.store.Cancel(ctx, owner, parsed)
	if errors.Is(err, model.ErrConflict) {
		return model.Job{}, &ConflictError{Message: "scan job cannot be cancelled from its current status"}
	}
	return job, err
}

// Delete removes a job that is not queued or running.
func (s Service) Delete(ctx context.Context, ownerID, id string) error {
	if err := s.ready(); err != nil {
		return err
	}
	owner, err := requireOwner(ownerID)
	if err != nil {
		return err
	}
	parsed, err := parseJobID(id)
	if err != nil {
		return err
	}
	err = s.store.Delete(ctx, owner, parsed)
	if errors.Is(err, model.ErrConflict) {
		return &ConflictError{Message: "scan job cannot be deleted while queued or running"}
	}
	return err
}

func (s Service) ready() error {
	if s.store == nil {
		return errors.New("scan job store is unavailable")
	}
	return nil
}

func normalizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", invalid("name is required")
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return "", invalid("name must be 200 characters or fewer")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", invalid("name contains unsupported characters")
		}
	}
	return name, nil
}

func normalizeList(request ListRequest) (model.ListFilter, error) {
	filter := model.ListFilter{Page: defaultPage, PageSize: defaultPageSize}
	if request.Page != nil {
		if *request.Page < 1 || *request.Page > maxPage {
			return model.ListFilter{}, invalid("page must be between 1 and 10000")
		}
		filter.Page = *request.Page
	}
	if request.PageSize != nil {
		if *request.PageSize < 1 || *request.PageSize > maxPageSize {
			return model.ListFilter{}, invalid("page_size must be between 1 and 100")
		}
		filter.PageSize = *request.PageSize
	}
	status := model.Status(request.Status)
	if request.Status != "" {
		if !status.Valid() {
			return model.ListFilter{}, invalid("status is invalid")
		}
		filter.Status = status
	}
	if request.TargetID != "" {
		id, err := model.ParseID(request.TargetID)
		if err != nil {
			return model.ListFilter{}, invalid("target id must be a uuid")
		}
		filter.TargetID = id
	}
	return filter, nil
}

func requireOwner(ownerID string) (string, error) {
	parsed, err := model.ParseID(ownerID)
	if err != nil {
		return "", errors.New("operator id is required")
	}
	return parsed, nil
}

func parseJobID(id string) (string, error) {
	parsed, err := model.ParseID(id)
	if err != nil {
		return "", invalid("scan job id must be a uuid")
	}
	return parsed, nil
}

func asValidation(err error) error {
	var rejected *scope.Error
	if errors.As(err, &rejected) {
		return invalid(rejected.Message)
	}
	return err
}

func invalid(message string) error {
	return &ValidationError{Message: message}
}
