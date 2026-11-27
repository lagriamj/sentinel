package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"sentinel-scanner/apps/api/internal/modules/auth/model"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Postgres stores accounts and hashed refresh tokens.
type Postgres struct {
	db *gorm.DB
}

// NewPostgres wraps the shared database pool.
func NewPostgres(db *gorm.DB) Postgres {
	return Postgres{db: db}
}

// CreateUser inserts an operator. The password hash is stored as given.
func (r Postgres) CreateUser(ctx context.Context, email, passwordHash string) (model.User, error) {
	if err := r.ready(); err != nil {
		return model.User{}, err
	}
	row := r.db.WithContext(ctx).Raw(`
INSERT INTO users (email, password_hash)
VALUES (?, ?)
RETURNING id::text, email, password_hash
`, email, passwordHash).Row()
	user, err := scanUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return model.User{}, model.ErrConflict
		}
		return model.User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// FindUserByEmail returns the account for a normalized email.
func (r Postgres) FindUserByEmail(ctx context.Context, email string) (model.User, error) {
	if err := r.ready(); err != nil {
		return model.User{}, err
	}
	row := r.db.WithContext(ctx).Raw(`
SELECT id::text, email, password_hash
FROM users
WHERE lower(email) = lower(?)
`, email).Row()
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, model.ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}

// FindUserByID returns the account for a user id.
func (r Postgres) FindUserByID(ctx context.Context, id string) (model.User, error) {
	if err := r.ready(); err != nil {
		return model.User{}, err
	}
	row := r.db.WithContext(ctx).Raw(`
SELECT id::text, email, password_hash
FROM users
WHERE id = CAST(? AS uuid)
`, id).Row()
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, model.ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}

// SaveRefreshToken stores the hash of a new refresh token.
func (r Postgres) SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	if err := r.ready(); err != nil {
		return err
	}
	err := r.db.WithContext(ctx).Exec(`
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES (CAST(? AS uuid), ?, ?)
`, userID, tokenHash, expiresAt).Error
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

// RotateRefreshToken revokes the presented hash and stores its replacement.
// A revoked or expired token revokes every token for that account.
func (r Postgres) RotateRefreshToken(ctx context.Context, oldHash, newHash string, expiresAt time.Time) (string, error) {
	if err := r.ready(); err != nil {
		return "", err
	}
	var userID string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := tx.Raw(`
SELECT id::text, user_id::text, expires_at, revoked_at, replaced_by::text
FROM refresh_tokens
WHERE token_hash = ?
FOR UPDATE
`, oldHash).Row()
		var id, owner string
		var expires time.Time
		var revoked sql.NullTime
		var replaced sql.NullString
		if err := row.Scan(&id, &owner, &expires, &revoked, &replaced); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrNotFound
			}
			return err
		}
		if revoked.Valid || replaced.Valid || !expires.After(time.Now()) {
			if err := tx.Exec(`
UPDATE refresh_tokens
SET revoked_at = now()
WHERE user_id = CAST(? AS uuid) AND revoked_at IS NULL
`, owner).Error; err != nil {
				return err
			}
			return model.ErrReuse
		}
		if err := tx.Exec(`
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES (CAST(? AS uuid), ?, ?)
`, owner, newHash, expiresAt).Error; err != nil {
			return err
		}
		var newID string
		if err := tx.Raw(`SELECT id::text FROM refresh_tokens WHERE token_hash = ?`, newHash).Row().Scan(&newID); err != nil {
			return err
		}
		result := tx.Exec(`
UPDATE refresh_tokens
SET revoked_at = now(), replaced_by = CAST(? AS uuid)
WHERE id = CAST(? AS uuid) AND revoked_at IS NULL
`, newID, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return model.ErrReuse
		}
		userID = owner
		return nil
	})
	if err != nil {
		if errors.Is(err, model.ErrNotFound) || errors.Is(err, model.ErrReuse) {
			return "", err
		}
		return "", fmt.Errorf("rotate refresh token: %w", err)
	}
	return userID, nil
}

// RevokeRefreshToken marks one token revoked.
func (r Postgres) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	if err := r.ready(); err != nil {
		return err
	}
	result := r.db.WithContext(ctx).Exec(`
UPDATE refresh_tokens
SET revoked_at = now()
WHERE token_hash = ? AND revoked_at IS NULL
`, tokenHash)
	if result.Error != nil {
		return fmt.Errorf("revoke refresh token: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r Postgres) ready() error {
	if r.db == nil {
		return errors.New("database pool unavailable")
	}
	return nil
}

func scanUser(row *sql.Row) (model.User, error) {
	var user model.User
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash)
	return user, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
