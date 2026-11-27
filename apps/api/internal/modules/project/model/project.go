package model

import (
	"errors"
	"time"
)

const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

var (
	ErrNotFound        = errors.New("project not found")
	ErrTargetNotFound  = errors.New("target not found")
	ErrMemberNotFound  = errors.New("member not found")
	ErrUserNotFound    = errors.New("user not found")
	ErrForbidden       = errors.New("forbidden")
	ErrHasTargets      = errors.New("project has targets")
	ErrTargetInUse     = errors.New("target in use")
	ErrLastOwner       = errors.New("last owner")
	ErrDuplicateMember = errors.New("duplicate member")
	ErrDuplicateTarget = errors.New("duplicate target")
)

// Project is an assessment workspace the caller belongs to.
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectPage is one page of projects.
type ProjectPage struct {
	Projects []Project `json:"data"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	Total    int64     `json:"total"`
}

// Member is a user assigned to a project.
type Member struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// MemberList is the membership of one project.
type MemberList struct {
	Members []Member `json:"data"`
}

// Target is an authorized origin registered on a project.
type Target struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	URL        string    `json:"url"`
	Scheme     string    `json:"scheme"`
	Host       string    `json:"host"`
	Port       int       `json:"port"`
	Authorized bool      `json:"authorized"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TargetPage is one page of registered targets.
type TargetPage struct {
	Targets  []Target `json:"data"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	Total    int64    `json:"total"`
}
