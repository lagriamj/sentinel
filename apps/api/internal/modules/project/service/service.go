package service

import (
	"context"
	"errors"
	"net"
	"strings"
	"unicode"
	"unicode/utf8"

	"sentinel-scanner/apps/api/internal/modules/project/model"
	jobmodel "sentinel-scanner/apps/api/internal/modules/scanjob/model"
	"sentinel-scanner/apps/api/internal/scope"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
	maxPage         = 10000
	maxNameLength   = 200
	maxDescription  = 2000
	maxEmailLength  = 320
)

// ValidationError is a request rejected before stored state changes.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil || e.Message == "" {
		return "request is invalid"
	}
	return e.Message
}

// ConflictError is a change the current records do not allow.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	if e == nil || e.Message == "" {
		return "request conflicts with the current record"
	}
	return e.Message
}

// Store persists projects, membership, and registered targets.
type Store interface {
	CreateProject(ctx context.Context, userID, name, description string) (model.Project, error)
	ListProjects(ctx context.Context, userID string, page, pageSize int) (model.ProjectPage, error)
	GetProject(ctx context.Context, userID, projectID string) (model.Project, error)
	UpdateProject(ctx context.Context, userID, projectID, name, description string) (model.Project, error)
	DeleteProject(ctx context.Context, userID, projectID string) error
	ListMembers(ctx context.Context, userID, projectID string) ([]model.Member, error)
	AddMember(ctx context.Context, actorID, projectID, email, role string) (model.Member, error)
	RemoveMember(ctx context.Context, actorID, projectID, memberID string) error
	ListTargets(ctx context.Context, userID, projectID string, page, pageSize int) (model.TargetPage, error)
	RegisterTarget(ctx context.Context, userID, projectID string, target scope.Target) (model.Target, error)
	DeleteTarget(ctx context.Context, userID, projectID, targetID string) error
	FindForMember(ctx context.Context, userID, targetID string) (model.Target, error)
}

// Resolver looks up a hostname before a target is stored.
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// PageRequest is optional pagination. Nil fields use the defaults.
type PageRequest struct {
	Page     *int
	PageSize *int
}

// Service owns project, membership, and target rules.
type Service struct {
	store    Store
	resolver Resolver
}

// New builds a project service. A nil resolver uses the process resolver.
func New(store Store, resolver Resolver) Service {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	return Service{store: store, resolver: resolver}
}

// CreateProject stores a project and makes the caller its owner.
func (s Service) CreateProject(ctx context.Context, userID, name, description string) (model.Project, error) {
	if err := s.ready(); err != nil {
		return model.Project{}, err
	}
	user, err := requireUser(userID)
	if err != nil {
		return model.Project{}, err
	}
	normalizedName, err := normalizeName(name)
	if err != nil {
		return model.Project{}, err
	}
	normalizedDescription, err := normalizeDescription(description)
	if err != nil {
		return model.Project{}, err
	}
	return s.store.CreateProject(ctx, user, normalizedName, normalizedDescription)
}

// ListProjects returns the projects the caller belongs to.
func (s Service) ListProjects(ctx context.Context, userID string, request PageRequest) (model.ProjectPage, error) {
	if err := s.ready(); err != nil {
		return model.ProjectPage{}, err
	}
	user, err := requireUser(userID)
	if err != nil {
		return model.ProjectPage{}, err
	}
	page, pageSize, err := normalizePage(request)
	if err != nil {
		return model.ProjectPage{}, err
	}
	result, err := s.store.ListProjects(ctx, user, page, pageSize)
	if err != nil {
		return model.ProjectPage{}, err
	}
	if result.Projects == nil {
		result.Projects = []model.Project{}
	}
	result.Page = page
	result.PageSize = pageSize
	return result, nil
}

// GetProject returns one project when the caller is a member.
func (s Service) GetProject(ctx context.Context, userID, projectID string) (model.Project, error) {
	user, id, err := s.memberArgs(ctx, userID, projectID)
	if err != nil {
		return model.Project{}, err
	}
	return s.store.GetProject(ctx, user, id)
}

// UpdateProject replaces the name and description when the caller is an owner.
func (s Service) UpdateProject(ctx context.Context, userID, projectID, name, description string) (model.Project, error) {
	user, id, err := s.memberArgs(ctx, userID, projectID)
	if err != nil {
		return model.Project{}, err
	}
	normalizedName, err := normalizeName(name)
	if err != nil {
		return model.Project{}, err
	}
	normalizedDescription, err := normalizeDescription(description)
	if err != nil {
		return model.Project{}, err
	}
	return s.store.UpdateProject(ctx, user, id, normalizedName, normalizedDescription)
}

// DeleteProject removes a project that has no registered targets.
func (s Service) DeleteProject(ctx context.Context, userID, projectID string) error {
	user, id, err := s.memberArgs(ctx, userID, projectID)
	if err != nil {
		return err
	}
	err = s.store.DeleteProject(ctx, user, id)
	return mapConflict(err, map[error]string{
		model.ErrHasTargets: "project cannot be deleted while it has targets",
	})
}

// ListMembers returns the membership of a project.
func (s Service) ListMembers(ctx context.Context, userID, projectID string) (model.MemberList, error) {
	user, id, err := s.memberArgs(ctx, userID, projectID)
	if err != nil {
		return model.MemberList{}, err
	}
	members, err := s.store.ListMembers(ctx, user, id)
	if err != nil {
		return model.MemberList{}, err
	}
	if members == nil {
		members = []model.Member{}
	}
	return model.MemberList{Members: members}, nil
}

// AddMember assigns an existing account to the project.
func (s Service) AddMember(ctx context.Context, actorID, projectID, email, role string) (model.Member, error) {
	user, id, err := s.memberArgs(ctx, actorID, projectID)
	if err != nil {
		return model.Member{}, err
	}
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return model.Member{}, err
	}
	normalizedRole, err := normalizeRole(role)
	if err != nil {
		return model.Member{}, err
	}
	member, err := s.store.AddMember(ctx, user, id, normalizedEmail, normalizedRole)
	if errors.Is(err, model.ErrDuplicateMember) {
		return model.Member{}, &ConflictError{Message: "that account is already a member"}
	}
	return member, err
}

// RemoveMember removes a member when another owner remains.
func (s Service) RemoveMember(ctx context.Context, actorID, projectID, memberID string) error {
	user, id, err := s.memberArgs(ctx, actorID, projectID)
	if err != nil {
		return err
	}
	member, err := jobmodel.ParseID(memberID)
	if err != nil {
		return invalid("member id must be a uuid")
	}
	err = s.store.RemoveMember(ctx, user, id, member)
	return mapConflict(err, map[error]string{
		model.ErrLastOwner: "project must keep an owner",
	})
}

// ListTargets returns registered targets for a project the caller belongs to.
func (s Service) ListTargets(ctx context.Context, userID, projectID string, request PageRequest) (model.TargetPage, error) {
	user, id, err := s.memberArgs(ctx, userID, projectID)
	if err != nil {
		return model.TargetPage{}, err
	}
	page, pageSize, err := normalizePage(request)
	if err != nil {
		return model.TargetPage{}, err
	}
	result, err := s.store.ListTargets(ctx, user, id, page, pageSize)
	if err != nil {
		return model.TargetPage{}, err
	}
	if result.Targets == nil {
		result.Targets = []model.Target{}
	}
	result.Page = page
	result.PageSize = pageSize
	return result, nil
}

// RegisterTarget stores an authorized target on a project.
func (s Service) RegisterTarget(ctx context.Context, userID, projectID, rawURL string, authorized *bool) (model.Target, error) {
	user, id, err := s.memberArgs(ctx, userID, projectID)
	if err != nil {
		return model.Target{}, err
	}
	if authorized == nil {
		return model.Target{}, invalid("authorization confirmation is required")
	}
	if !*authorized {
		return model.Target{}, invalid("target is not authorized")
	}
	target, hostIsIP, err := scope.Parse(rawURL)
	if err != nil {
		return model.Target{}, asValidation(err)
	}
	if !hostIsIP {
		if err := scope.Confirm(ctx, s.resolver, target.Host); err != nil {
			return model.Target{}, asValidation(err)
		}
	}
	stored, err := s.store.RegisterTarget(ctx, user, id, target)
	if errors.Is(err, model.ErrDuplicateTarget) {
		return model.Target{}, &ConflictError{Message: "target is already registered"}
	}
	return stored, err
}

// DeleteTarget removes a target that no scan job uses.
func (s Service) DeleteTarget(ctx context.Context, userID, projectID, targetID string) error {
	user, id, err := s.memberArgs(ctx, userID, projectID)
	if err != nil {
		return err
	}
	parsed, err := jobmodel.ParseID(targetID)
	if err != nil {
		return invalid("target id must be a uuid")
	}
	err = s.store.DeleteTarget(ctx, user, id, parsed)
	return mapConflict(err, map[error]string{
		model.ErrTargetInUse: "target cannot be deleted while a scan job uses it",
	})
}

// FindForMember returns a target the caller may use for a scan job.
func (s Service) FindForMember(ctx context.Context, userID, targetID string) (model.Target, error) {
	if err := s.ready(); err != nil {
		return model.Target{}, err
	}
	user, err := requireUser(userID)
	if err != nil {
		return model.Target{}, err
	}
	parsed, err := jobmodel.ParseID(targetID)
	if err != nil {
		return model.Target{}, invalid("target id must be a uuid")
	}
	return s.store.FindForMember(ctx, user, parsed)
}

func (s Service) memberArgs(ctx context.Context, userID, projectID string) (string, string, error) {
	if err := s.ready(); err != nil {
		return "", "", err
	}
	user, err := requireUser(userID)
	if err != nil {
		return "", "", err
	}
	id, err := jobmodel.ParseID(projectID)
	if err != nil {
		return "", "", invalid("project id must be a uuid")
	}
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	return user, id, nil
}

func (s Service) ready() error {
	if s.store == nil || s.resolver == nil {
		return errors.New("project store is unavailable")
	}
	return nil
}

func normalizePage(request PageRequest) (int, int, error) {
	page := defaultPage
	pageSize := defaultPageSize
	if request.Page != nil {
		if *request.Page < 1 || *request.Page > maxPage {
			return 0, 0, invalid("page must be between 1 and 10000")
		}
		page = *request.Page
	}
	if request.PageSize != nil {
		if *request.PageSize < 1 || *request.PageSize > maxPageSize {
			return 0, 0, invalid("page_size must be between 1 and 100")
		}
		pageSize = *request.PageSize
	}
	return page, pageSize, nil
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

func normalizeDescription(raw string) (string, error) {
	description := strings.TrimSpace(raw)
	if utf8.RuneCountInString(description) > maxDescription {
		return "", invalid("description must be 2000 characters or fewer")
	}
	for _, r := range description {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", invalid("description contains unsupported characters")
		}
	}
	return description, nil
}

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", invalid("email is required")
	}
	if utf8.RuneCountInString(email) > maxEmailLength {
		return "", invalid("email is too long")
	}
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 || strings.Contains(email, " ") {
		return "", invalid("email is invalid")
	}
	if !strings.Contains(email[at+1:], ".") {
		return "", invalid("email is invalid")
	}
	return email, nil
}

func normalizeRole(raw string) (string, error) {
	role := strings.ToLower(strings.TrimSpace(raw))
	if role != model.RoleOwner && role != model.RoleMember {
		return "", invalid("role must be owner or member")
	}
	return role, nil
}

func requireUser(userID string) (string, error) {
	parsed, err := jobmodel.ParseID(userID)
	if err != nil {
		return "", errors.New("operator id is required")
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

func mapConflict(err error, messages map[error]string) error {
	for sentinel, message := range messages {
		if errors.Is(err, sentinel) {
			return &ConflictError{Message: message}
		}
	}
	return err
}

func invalid(message string) error {
	return &ValidationError{Message: message}
}
