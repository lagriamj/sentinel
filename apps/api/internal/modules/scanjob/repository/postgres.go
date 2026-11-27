package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"sentinel-scanner/apps/api/internal/modules/scanjob/model"

	"gorm.io/gorm"
)

const (
	jobColumns = `
j.id::text AS id,
j.name,
j.status,
j.error_message,
j.started_at,
j.finished_at,
j.created_at,
j.updated_at,
t.id::text AS target_id,
t.url AS target_url,
t.scheme,
t.host,
t.port,
t.authorized,
t.project_id::text AS project_id`

	jobJoin = "INNER JOIN scan_targets AS t ON t.id = j.target_id"
)

// Postgres stores scan jobs in PostgreSQL.
type Postgres struct {
	db *gorm.DB
}

// NewPostgres wraps the shared database pool.
func NewPostgres(db *gorm.DB) Postgres {
	return Postgres{db: db}
}

// Create stores a pending job for a target the caller may assess.
func (r Postgres) Create(ctx context.Context, input model.CreateInput) (model.Job, error) {
	if err := r.ready(); err != nil {
		return model.Job{}, err
	}
	var job model.Job
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := tx.Raw(`
INSERT INTO scan_jobs (target_id, created_by, name, status)
SELECT t.id, CAST(? AS uuid), ?, 'pending'
FROM scan_targets AS t
INNER JOIN project_members AS m ON m.project_id = t.project_id AND m.user_id = CAST(? AS uuid)
WHERE t.id = CAST(? AS uuid)
RETURNING id::text
`, input.OwnerID, input.Name, input.OwnerID, input.TargetID).Row()
		var id string
		if err := row.Scan(&id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrTargetNotFound
			}
			return err
		}
		scanned, err := getJob(tx, input.OwnerID, id)
		if err != nil {
			return err
		}
		job = scanned
		return nil
	})
	if err != nil {
		return model.Job{}, fmt.Errorf("create scan job: %w", err)
	}
	return job, nil
}

// List returns one consistent page of jobs and the matching total.
func (r Postgres) List(ctx context.Context, filter model.ListFilter) (model.Page, error) {
	if err := r.ready(); err != nil {
		return model.Page{}, err
	}
	if filter.Page < 1 || filter.PageSize < 1 {
		return model.Page{}, errors.New("invalid pagination")
	}
	page := model.Page{
		Jobs:     []model.Job{},
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		countQuery := applyFilter(tx.Table("scan_jobs AS j"), filter)
		if err := countQuery.Count(&page.Total).Error; err != nil {
			return err
		}
		var rows []jobRow
		listQuery := applyFilter(tx.Table("scan_jobs AS j"), filter).
			Select(jobColumns).
			Order("j.created_at DESC, j.id DESC").
			Limit(filter.PageSize).
			Offset((filter.Page - 1) * filter.PageSize)
		if err := listQuery.Scan(&rows).Error; err != nil {
			return err
		}
		page.Jobs = mapJobs(rows)
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return model.Page{}, fmt.Errorf("list scan jobs: %w", err)
	}
	return page, nil
}

// Get returns one scan job and its target.
func (r Postgres) Get(ctx context.Context, ownerID, id string) (model.Job, error) {
	if err := r.ready(); err != nil {
		return model.Job{}, err
	}
	job, err := getJob(r.db.WithContext(ctx), ownerID, id)
	if err != nil {
		return model.Job{}, fmt.Errorf("get scan job: %w", err)
	}
	return job, nil
}

// Cancel marks a cancellable job cancelled and records when it finished.
func (r Postgres) Cancel(ctx context.Context, ownerID, id string) (model.Job, error) {
	if err := r.ready(); err != nil {
		return model.Job{}, err
	}
	var job model.Job
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		status, err := lockStatus(tx, ownerID, id)
		if err != nil {
			return err
		}
		if !status.CanCancel() {
			return model.ErrConflict
		}
		result := tx.Exec(`
UPDATE scan_jobs
SET status = 'cancelled', finished_at = now(), updated_at = now()
WHERE id = CAST(? AS uuid) AND created_by = CAST(? AS uuid)
`, id, ownerID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return model.ErrNotFound
		}
		job, err = getJob(tx, ownerID, id)
		return err
	})
	if err != nil {
		return model.Job{}, fmt.Errorf("cancel scan job: %w", err)
	}
	return job, nil
}

// Delete removes a job whose status allows deletion.
// Results and vulnerabilities that belong to the job are removed with it.
func (r Postgres) Delete(ctx context.Context, ownerID, id string) error {
	if err := r.ready(); err != nil {
		return err
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		status, err := lockStatus(tx, ownerID, id)
		if err != nil {
			return err
		}
		if !status.CanDelete() {
			return model.ErrConflict
		}
		result := tx.Exec(`DELETE FROM scan_jobs WHERE id = CAST(? AS uuid) AND created_by = CAST(? AS uuid)`, id, ownerID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return model.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("delete scan job: %w", err)
	}
	return nil
}

func (r Postgres) ready() error {
	if r.db == nil {
		return errors.New("database pool unavailable")
	}
	return nil
}

func lockStatus(tx *gorm.DB, ownerID, id string) (model.Status, error) {
	row := tx.Raw(`
SELECT j.status
FROM scan_jobs AS j
INNER JOIN scan_targets AS t ON t.id = j.target_id
INNER JOIN project_members AS m ON m.project_id = t.project_id AND m.user_id = CAST(? AS uuid)
WHERE j.id = CAST(? AS uuid)
FOR UPDATE OF j
`, ownerID, id).Row()
	var status string
	if err := row.Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", model.ErrNotFound
		}
		return "", err
	}
	return model.Status(status), nil
}

func getJob(tx *gorm.DB, ownerID, id string) (model.Job, error) {
	var row jobRow
	result := tx.Raw(`
SELECT `+jobColumns+`
FROM scan_jobs AS j
`+jobJoin+`
INNER JOIN project_members AS m ON m.project_id = t.project_id AND m.user_id = CAST(? AS uuid)
WHERE j.id = CAST(? AS uuid)
`, ownerID, id).Scan(&row)
	if result.Error != nil {
		return model.Job{}, result.Error
	}
	if row.ID == "" {
		return model.Job{}, model.ErrNotFound
	}
	return row.job(), nil
}

func applyFilter(tx *gorm.DB, filter model.ListFilter) *gorm.DB {
	tx = tx.Joins(jobJoin).
		Joins("INNER JOIN project_members AS m ON m.project_id = t.project_id AND m.user_id = CAST(? AS uuid)", filter.OwnerID)
	if filter.Status != "" {
		tx = tx.Where("j.status = ?", string(filter.Status))
	}
	if filter.TargetID != "" {
		tx = tx.Where("j.target_id = CAST(? AS uuid)", filter.TargetID)
	}
	return tx
}

func mapJobs(rows []jobRow) []model.Job {
	jobs := make([]model.Job, 0, len(rows))
	for _, row := range rows {
		jobs = append(jobs, row.job())
	}
	return jobs
}

type jobRow struct {
	ID           string
	Name         string
	Status       string
	ErrorMessage sql.NullString
	StartedAt    sql.NullTime
	FinishedAt   sql.NullTime
	CreatedAt    time.Time
	UpdatedAt    time.Time
	TargetID     string
	TargetURL    string
	Scheme       string
	Host         string
	Port         int
	Authorized   bool
	ProjectID    string
}

func (row jobRow) job() model.Job {
	job := model.Job{
		ID:     row.ID,
		Name:   row.Name,
		Status: model.Status(row.Status),
		Target: model.Target{
			ID:         row.TargetID,
			ProjectID:  row.ProjectID,
			URL:        row.TargetURL,
			Scheme:     row.Scheme,
			Host:       row.Host,
			Port:       row.Port,
			Authorized: row.Authorized,
		},
		CreatedAt: row.CreatedAt.UTC(),
		UpdatedAt: row.UpdatedAt.UTC(),
	}
	if row.ErrorMessage.Valid {
		job.ErrorMessage = row.ErrorMessage.String
	}
	job.StartedAt = utcTime(row.StartedAt)
	job.FinishedAt = utcTime(row.FinishedAt)
	return job
}

func utcTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	utc := value.Time.UTC()
	return &utc
}
