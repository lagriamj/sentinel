package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"sentinel-scanner/apps/api/internal/modules/project/model"
	"sentinel-scanner/apps/api/internal/scope"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Postgres stores projects in PostgreSQL.
type Postgres struct {
	db *gorm.DB
}

// NewPostgres wraps the shared database pool.
func NewPostgres(db *gorm.DB) Postgres {
	return Postgres{db: db}
}

// CreateProject inserts a project and an owner membership.
func (r Postgres) CreateProject(ctx context.Context, userID, name, description string) (model.Project, error) {
	if err := r.ready(); err != nil {
		return model.Project{}, err
	}
	var project model.Project
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := tx.Raw(`
INSERT INTO projects (name, description, created_by)
VALUES (?, ?, CAST(? AS uuid))
RETURNING id::text, name, description, created_at, updated_at
`, name, description, userID).Row()
		if err := row.Scan(&project.ID, &project.Name, &project.Description, &project.CreatedAt, &project.UpdatedAt); err != nil {
			return err
		}
		result := tx.Exec(`
INSERT INTO project_members (project_id, user_id, role)
VALUES (CAST(? AS uuid), CAST(? AS uuid), 'owner')
`, project.ID, userID)
		if result.Error != nil {
			return result.Error
		}
		project.Role = model.RoleOwner
		project.CreatedAt = project.CreatedAt.UTC()
		project.UpdatedAt = project.UpdatedAt.UTC()
		return nil
	})
	if err != nil {
		return model.Project{}, fmt.Errorf("create project: %w", err)
	}
	return project, nil
}

// ListProjects returns one page of projects the user belongs to.
func (r Postgres) ListProjects(ctx context.Context, userID string, page, pageSize int) (model.ProjectPage, error) {
	if err := r.ready(); err != nil {
		return model.ProjectPage{}, err
	}
	result := model.ProjectPage{Projects: []model.Project{}, Page: page, PageSize: pageSize}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(`
SELECT count(*)
FROM projects AS p
INNER JOIN project_members AS m ON m.project_id = p.id AND m.user_id = CAST(? AS uuid)
`, userID).Scan(&result.Total).Error; err != nil {
			return err
		}
		rows, err := tx.Raw(`
SELECT p.id::text, p.name, p.description, m.role, p.created_at, p.updated_at
FROM projects AS p
INNER JOIN project_members AS m ON m.project_id = p.id AND m.user_id = CAST(? AS uuid)
ORDER BY p.created_at DESC, p.id DESC
LIMIT ? OFFSET ?
`, userID, pageSize, (page-1)*pageSize).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			project, err := scanProject(rows)
			if err != nil {
				return err
			}
			result.Projects = append(result.Projects, project)
		}
		return rows.Err()
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return model.ProjectPage{}, fmt.Errorf("list projects: %w", err)
	}
	return result, nil
}

// GetProject returns a project when the user is a member.
func (r Postgres) GetProject(ctx context.Context, userID, projectID string) (model.Project, error) {
	if err := r.ready(); err != nil {
		return model.Project{}, err
	}
	project, err := getProject(r.db.WithContext(ctx), userID, projectID)
	if err != nil {
		return model.Project{}, fmt.Errorf("get project: %w", err)
	}
	return project, nil
}

// UpdateProject replaces the name and description for an owner.
func (r Postgres) UpdateProject(ctx context.Context, userID, projectID, name, description string) (model.Project, error) {
	if err := r.ready(); err != nil {
		return model.Project{}, err
	}
	var project model.Project
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		role, err := memberRole(tx, userID, projectID)
		if err != nil {
			return err
		}
		if role != model.RoleOwner {
			return model.ErrForbidden
		}
		result := tx.Exec(`
UPDATE projects
SET name = ?, description = ?, updated_at = now()
WHERE id = CAST(? AS uuid)
`, name, description, projectID)
		if result.Error != nil {
			return result.Error
		}
		project, err = getProject(tx, userID, projectID)
		return err
	})
	if err != nil {
		return model.Project{}, fmt.Errorf("update project: %w", err)
	}
	return project, nil
}

// DeleteProject removes an owner's project that has no targets.
func (r Postgres) DeleteProject(ctx context.Context, userID, projectID string) error {
	if err := r.ready(); err != nil {
		return err
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		role, err := memberRole(tx, userID, projectID)
		if err != nil {
			return err
		}
		if role != model.RoleOwner {
			return model.ErrForbidden
		}
		var targets int64
		if err := tx.Raw(`SELECT count(*) FROM scan_targets WHERE project_id = CAST(? AS uuid)`, projectID).Scan(&targets).Error; err != nil {
			return err
		}
		if targets > 0 {
			return model.ErrHasTargets
		}
		result := tx.Exec(`DELETE FROM projects WHERE id = CAST(? AS uuid)`, projectID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return model.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}

// ListMembers returns every member of a project.
func (r Postgres) ListMembers(ctx context.Context, userID, projectID string) ([]model.Member, error) {
	if err := r.ready(); err != nil {
		return nil, err
	}
	if _, err := memberRole(r.db.WithContext(ctx), userID, projectID); err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	rows, err := r.db.WithContext(ctx).Raw(`
SELECT m.user_id::text, u.email, m.role, m.created_at
FROM project_members AS m
INNER JOIN users AS u ON u.id = m.user_id
WHERE m.project_id = CAST(? AS uuid)
ORDER BY m.created_at ASC, m.user_id ASC
`, projectID).Rows()
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()
	members := []model.Member{}
	for rows.Next() {
		var member model.Member
		if err := rows.Scan(&member.UserID, &member.Email, &member.Role, &member.CreatedAt); err != nil {
			return nil, fmt.Errorf("list members: %w", err)
		}
		member.CreatedAt = member.CreatedAt.UTC()
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	return members, nil
}

// AddMember assigns an existing account when the actor is an owner.
func (r Postgres) AddMember(ctx context.Context, actorID, projectID, email, role string) (model.Member, error) {
	if err := r.ready(); err != nil {
		return model.Member{}, err
	}
	var member model.Member
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		actorRole, err := memberRole(tx, actorID, projectID)
		if err != nil {
			return err
		}
		if actorRole != model.RoleOwner {
			return model.ErrForbidden
		}
		row := tx.Raw(`SELECT id::text, email FROM users WHERE lower(email) = ?`, email).Row()
		if err := row.Scan(&member.UserID, &member.Email); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrUserNotFound
			}
			return err
		}
		inserted := tx.Raw(`
INSERT INTO project_members (project_id, user_id, role)
VALUES (CAST(? AS uuid), CAST(? AS uuid), ?)
RETURNING role, created_at
`, projectID, member.UserID, role).Row()
		if err := inserted.Scan(&member.Role, &member.CreatedAt); err != nil {
			if isUniqueViolation(err) {
				return model.ErrDuplicateMember
			}
			return err
		}
		member.CreatedAt = member.CreatedAt.UTC()
		return nil
	})
	if err != nil {
		return model.Member{}, fmt.Errorf("add member: %w", err)
	}
	return member, nil
}

// RemoveMember removes a member when the actor is an owner and an owner remains.
func (r Postgres) RemoveMember(ctx context.Context, actorID, projectID, memberID string) error {
	if err := r.ready(); err != nil {
		return err
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		actorRole, err := memberRole(tx, actorID, projectID)
		if err != nil {
			return err
		}
		if actorRole != model.RoleOwner {
			return model.ErrForbidden
		}
		var role string
		row := tx.Raw(`
SELECT role
FROM project_members
WHERE project_id = CAST(? AS uuid) AND user_id = CAST(? AS uuid)
FOR UPDATE
`, projectID, memberID).Row()
		if err := row.Scan(&role); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrMemberNotFound
			}
			return err
		}
		if role == model.RoleOwner {
			var owners int64
			if err := tx.Raw(`
SELECT count(*) FROM project_members
WHERE project_id = CAST(? AS uuid) AND role = 'owner'
`, projectID).Scan(&owners).Error; err != nil {
				return err
			}
			if owners < 2 {
				return model.ErrLastOwner
			}
		}
		result := tx.Exec(`
DELETE FROM project_members
WHERE project_id = CAST(? AS uuid) AND user_id = CAST(? AS uuid)
`, projectID, memberID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return model.ErrMemberNotFound
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

// ListTargets returns one page of targets for a member.
func (r Postgres) ListTargets(ctx context.Context, userID, projectID string, page, pageSize int) (model.TargetPage, error) {
	if err := r.ready(); err != nil {
		return model.TargetPage{}, err
	}
	result := model.TargetPage{Targets: []model.Target{}, Page: page, PageSize: pageSize}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := memberRole(tx, userID, projectID); err != nil {
			return err
		}
		if err := tx.Raw(`SELECT count(*) FROM scan_targets WHERE project_id = CAST(? AS uuid)`, projectID).Scan(&result.Total).Error; err != nil {
			return err
		}
		rows, err := tx.Raw(`
SELECT id::text, project_id::text, url, scheme, host, port, authorized, created_at, updated_at
FROM scan_targets
WHERE project_id = CAST(? AS uuid)
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?
`, projectID, pageSize, (page-1)*pageSize).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			target, err := scanTarget(rows)
			if err != nil {
				return err
			}
			result.Targets = append(result.Targets, target)
		}
		return rows.Err()
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return model.TargetPage{}, fmt.Errorf("list targets: %w", err)
	}
	return result, nil
}

// RegisterTarget stores a normalized target for a project member.
func (r Postgres) RegisterTarget(ctx context.Context, userID, projectID string, target scope.Target) (model.Target, error) {
	if err := r.ready(); err != nil {
		return model.Target{}, err
	}
	var stored model.Target
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := memberRole(tx, userID, projectID); err != nil {
			return err
		}
		row := tx.Raw(`
INSERT INTO scan_targets (project_id, url, scheme, host, port, authorized)
VALUES (CAST(? AS uuid), ?, ?, ?, ?, TRUE)
RETURNING id::text, project_id::text, url, scheme, host, port, authorized, created_at, updated_at
`, projectID, target.URL, target.Scheme, target.Host, target.Port).Row()
		scanned, err := scanTarget(row)
		if err != nil {
			if isUniqueViolation(err) {
				return model.ErrDuplicateTarget
			}
			return err
		}
		stored = scanned
		return nil
	})
	if err != nil {
		return model.Target{}, fmt.Errorf("register target: %w", err)
	}
	return stored, nil
}

// DeleteTarget removes a target when the caller is an owner and no job uses it.
func (r Postgres) DeleteTarget(ctx context.Context, userID, projectID, targetID string) error {
	if err := r.ready(); err != nil {
		return err
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		role, err := memberRole(tx, userID, projectID)
		if err != nil {
			return err
		}
		if role != model.RoleOwner {
			return model.ErrForbidden
		}
		result := tx.Exec(`
DELETE FROM scan_targets
WHERE id = CAST(? AS uuid) AND project_id = CAST(? AS uuid)
`, targetID, projectID)
		if result.Error != nil {
			if isForeignKeyViolation(result.Error) {
				return model.ErrTargetInUse
			}
			return result.Error
		}
		if result.RowsAffected == 0 {
			return model.ErrTargetNotFound
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("delete target: %w", err)
	}
	return nil
}

// FindForMember returns a target when the user belongs to its project.
func (r Postgres) FindForMember(ctx context.Context, userID, targetID string) (model.Target, error) {
	if err := r.ready(); err != nil {
		return model.Target{}, err
	}
	row := r.db.WithContext(ctx).Raw(`
SELECT t.id::text, t.project_id::text, t.url, t.scheme, t.host, t.port, t.authorized, t.created_at, t.updated_at
FROM scan_targets AS t
INNER JOIN project_members AS m ON m.project_id = t.project_id AND m.user_id = CAST(? AS uuid)
WHERE t.id = CAST(? AS uuid)
`, userID, targetID).Row()
	target, err := scanTarget(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Target{}, model.ErrTargetNotFound
		}
		return model.Target{}, fmt.Errorf("find target: %w", err)
	}
	return target, nil
}

func (r Postgres) ready() error {
	if r.db == nil {
		return errors.New("database pool unavailable")
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func getProject(tx *gorm.DB, userID, projectID string) (model.Project, error) {
	row := tx.Raw(`
SELECT p.id::text, p.name, p.description, m.role, p.created_at, p.updated_at
FROM projects AS p
INNER JOIN project_members AS m ON m.project_id = p.id AND m.user_id = CAST(? AS uuid)
WHERE p.id = CAST(? AS uuid)
`, userID, projectID).Row()
	project, err := scanProject(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Project{}, model.ErrNotFound
		}
		return model.Project{}, err
	}
	return project, nil
}

func memberRole(tx *gorm.DB, userID, projectID string) (string, error) {
	var role string
	var exists bool
	row := tx.Raw(`SELECT role FROM project_members WHERE project_id = CAST(? AS uuid) AND user_id = CAST(? AS uuid)`, projectID, userID).Row()
	if err := row.Scan(&role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			exists = false
		} else {
			return "", err
		}
	} else {
		exists = true
	}
	if exists {
		return role, nil
	}
	var projectExists bool
	check := tx.Raw(`SELECT EXISTS (SELECT 1 FROM projects WHERE id = CAST(? AS uuid))`, projectID).Row()
	if err := check.Scan(&projectExists); err != nil {
		return "", err
	}
	if !projectExists {
		return "", model.ErrNotFound
	}
	return "", model.ErrNotFound
}

func scanProject(row scanner) (model.Project, error) {
	var project model.Project
	var created, updated time.Time
	if err := row.Scan(&project.ID, &project.Name, &project.Description, &project.Role, &created, &updated); err != nil {
		return model.Project{}, err
	}
	project.CreatedAt = created.UTC()
	project.UpdatedAt = updated.UTC()
	return project, nil
}

func scanTarget(row scanner) (model.Target, error) {
	var target model.Target
	var created, updated time.Time
	if err := row.Scan(&target.ID, &target.ProjectID, &target.URL, &target.Scheme, &target.Host, &target.Port, &target.Authorized, &created, &updated); err != nil {
		return model.Target{}, err
	}
	target.CreatedAt = created.UTC()
	target.UpdatedAt = updated.UTC()
	return target, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
