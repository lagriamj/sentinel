package model

import (
	"errors"
	"strings"
	"time"
)

const (
	// StatusPending is the status of a job that has been stored and not queued.
	StatusPending Status = "pending"
	// StatusQueued means a later coordinator has handed the job to a worker.
	StatusQueued Status = "queued"
	// StatusRunning means a worker is executing the job.
	StatusRunning Status = "running"
	// StatusCompleted means the job finished successfully.
	StatusCompleted Status = "completed"
	// StatusFailed means the job stopped because execution failed.
	StatusFailed Status = "failed"
	// StatusCancelled means an operator cancelled the job.
	StatusCancelled Status = "cancelled"
)

// ErrNotFound means the scan job does not exist.
var ErrNotFound = errors.New("scan job not found")

// ErrTargetNotFound means the target is not registered for this operator.
var ErrTargetNotFound = errors.New("target not found")

// ErrConflict means the job status does not allow the requested change.
var ErrConflict = errors.New("scan job status conflict")

// ErrInvalidID means the value is not a canonical UUID.
var ErrInvalidID = errors.New("invalid id")

// Status is the lifecycle state of a scan job.
type Status string

// Valid reports whether the status is one of the stored lifecycle states.
func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusQueued, StatusRunning, StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

// CanCancel reports whether an operator may cancel a job in this status.
func (s Status) CanCancel() bool {
	switch s {
	case StatusPending, StatusQueued, StatusRunning:
		return true
	default:
		return false
	}
}

// CanDelete reports whether an operator may delete a job in this status.
// Queued and running jobs stay until they are cancelled so an in-flight
// record is not removed out from under a worker.
func (s Status) CanDelete() bool {
	switch s {
	case StatusPending, StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

// Statuses returns the lifecycle states in definition order.
func Statuses() []Status {
	return []Status{
		StatusPending,
		StatusQueued,
		StatusRunning,
		StatusCompleted,
		StatusFailed,
		StatusCancelled,
	}
}

// Target is an authorized http(s) origin registered for assessment.
type Target struct {
	ID         string `json:"id"`
	ProjectID  string `json:"project_id,omitempty"`
	URL        string `json:"url"`
	Scheme     string `json:"scheme"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Authorized bool   `json:"authorized"`
}

// TargetInput is the normalized target persisted with a new job.
type TargetInput struct {
	URL    string
	Scheme string
	Host   string
	Port   int
}

// Job is one assessment run against a registered target.
type Job struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Status       Status     `json:"status"`
	Target       Target     `json:"target"`
	ErrorMessage string     `json:"error_message,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CreateInput is the persisted form of a create request.
type CreateInput struct {
	OwnerID  string
	Name     string
	TargetID string
	Target   TargetInput
}

// ListFilter selects a page of scan jobs. Page is 1-based.
type ListFilter struct {
	OwnerID  string
	Status   Status
	TargetID string
	Page     int
	PageSize int
}

// Page is a filtered collection of scan jobs.
type Page struct {
	Jobs     []Job `json:"data"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// ParseID returns the lowercase form of a UUID.
func ParseID(value string) (string, error) {
	if len(value) != 36 {
		return "", ErrInvalidID
	}
	for i := 0; i < len(value); i++ {
		switch i {
		case 8, 13, 18, 23:
			if value[i] != '-' {
				return "", ErrInvalidID
			}
		default:
			if !isHex(value[i]) {
				return "", ErrInvalidID
			}
		}
	}
	return strings.ToLower(value), nil
}

func isHex(value byte) bool {
	return (value >= '0' && value <= '9') ||
		(value >= 'a' && value <= 'f') ||
		(value >= 'A' && value <= 'F')
}
