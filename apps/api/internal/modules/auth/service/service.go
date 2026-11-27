package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"sentinel-scanner/apps/api/internal/modules/auth/model"
	jobmodel "sentinel-scanner/apps/api/internal/modules/scanjob/model"
)

const (
	minPasswordLength = 12
	maxPasswordLength = 128
	maxEmailLength    = 320
)

// ValidationError is a request rejected before a credential check.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil || e.Message == "" {
		return "request is invalid"
	}
	return e.Message
}

// ConflictError means the email is already registered.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	if e == nil || e.Message == "" {
		return model.ErrConflict.Error()
	}
	return e.Message
}

func (e *ConflictError) Unwrap() error {
	return model.ErrConflict
}

// CredentialsError means the email and password do not match an account.
type CredentialsError struct{}

func (CredentialsError) Error() string {
	return "invalid email or password"
}

// Store persists accounts and hashed refresh tokens.
type Store interface {
	CreateUser(ctx context.Context, email, passwordHash string) (model.User, error)
	FindUserByEmail(ctx context.Context, email string) (model.User, error)
	FindUserByID(ctx context.Context, id string) (model.User, error)
	SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	RotateRefreshToken(ctx context.Context, oldHash, newHash string, expiresAt time.Time) (string, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
}

// Service registers operators and issues rotating sessions.
type Service struct {
	store       Store
	secret      []byte
	accessTTL   time.Duration
	refreshTTL  time.Duration
	now         func() time.Time
	hash        func(string) (string, error)
	verify      func(password, encoded string) bool
}

// Config is the token material for an auth service.
type Config struct {
	Secret     []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// New builds an auth service that hashes passwords with Argon2id.
func New(store Store, cfg Config) Service {
	return Service{
		store:      store,
		secret:     cfg.Secret,
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
		now:        time.Now,
		hash:       HashPassword,
		verify:     VerifyPassword,
	}
}

// Account is the public view of an operator.
type Account struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// Result is a signed-in operator and the tokens issued for that session.
type Result struct {
	Account Account
	Session Session
}

// Register creates an account and a session.
func (s Service) Register(ctx context.Context, email, password string) (Result, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return Result{}, err
	}
	if err := validatePassword(password); err != nil {
		return Result{}, err
	}
	if err := s.ready(); err != nil {
		return Result{}, err
	}
	encoded, err := s.hash(password)
	if err != nil {
		return Result{}, err
	}
	user, err := s.store.CreateUser(ctx, normalized, encoded)
	if errors.Is(err, model.ErrConflict) {
		return Result{}, &ConflictError{Message: "an account with that email already exists"}
	}
	if err != nil {
		return Result{}, err
	}
	return s.issue(ctx, user)
}

// Login checks the password and issues a session.
func (s Service) Login(ctx context.Context, email, password string) (Result, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return Result{}, err
	}
	if password == "" {
		return Result{}, invalid("password is required")
	}
	if err := s.ready(); err != nil {
		return Result{}, err
	}
	user, err := s.store.FindUserByEmail(ctx, normalized)
	if errors.Is(err, model.ErrNotFound) {
		s.verify(password, "$argon2id$")
		return Result{}, CredentialsError{}
	}
	if err != nil {
		return Result{}, err
	}
	if !s.verify(password, user.PasswordHash) {
		return Result{}, CredentialsError{}
	}
	return s.issue(ctx, user)
}

// Refresh rotates the presented refresh token and returns a new session.
func (s Service) Refresh(ctx context.Context, rawRefresh string) (Result, error) {
	if rawRefresh == "" {
		return Result{}, CredentialsError{}
	}
	if err := s.ready(); err != nil {
		return Result{}, err
	}
	session, hash, err := IssueSession("pending", s.secret, s.now().UTC(), s.accessTTL, s.refreshTTL)
	if err != nil {
		return Result{}, err
	}
	userID, err := s.store.RotateRefreshToken(ctx, HashRefreshToken(rawRefresh), hash, session.RefreshExpires)
	if errors.Is(err, model.ErrNotFound) || errors.Is(err, model.ErrReuse) {
		return Result{}, CredentialsError{}
	}
	if err != nil {
		return Result{}, err
	}
	user, err := s.store.FindUserByID(ctx, userID)
	if err != nil {
		return Result{}, err
	}
	signed, err := s.sign(user.ID)
	if err != nil {
		return Result{}, err
	}
	signed.RefreshToken = session.RefreshToken
	signed.RefreshExpires = session.RefreshExpires
	return Result{Account: Account{ID: user.ID, Email: user.Email}, Session: signed}, nil
}

// Logout revokes the presented refresh token.
func (s Service) Logout(ctx context.Context, rawRefresh string) error {
	if rawRefresh == "" {
		return nil
	}
	if err := s.ready(); err != nil {
		return err
	}
	err := s.store.RevokeRefreshToken(ctx, HashRefreshToken(rawRefresh))
	if errors.Is(err, model.ErrNotFound) {
		return nil
	}
	return err
}

// Session returns the account for an access-token subject.
func (s Service) Session(ctx context.Context, userID string) (Account, error) {
	if err := s.ready(); err != nil {
		return Account{}, err
	}
	parsed, err := jobmodel.ParseID(userID)
	if err != nil {
		return Account{}, CredentialsError{}
	}
	user, err := s.store.FindUserByID(ctx, parsed)
	if errors.Is(err, model.ErrNotFound) {
		return Account{}, CredentialsError{}
	}
	if err != nil {
		return Account{}, err
	}
	return Account{ID: user.ID, Email: user.Email}, nil
}

// Subject returns the user id carried by an access token.
func (s Service) Subject(raw string) (string, error) {
	userID, err := ParseAccessToken(raw, s.secret, s.now().UTC())
	if err != nil {
		return "", err
	}
	if _, err := jobmodel.ParseID(userID); err != nil {
		return "", err
	}
	return userID, nil
}

func (s Service) issue(ctx context.Context, user model.User) (Result, error) {
	session, err := s.sign(user.ID)
	if err != nil {
		return Result{}, err
	}
	if err := s.store.SaveRefreshToken(ctx, user.ID, HashRefreshToken(session.RefreshToken), session.RefreshExpires); err != nil {
		return Result{}, err
	}
	return Result{Account: Account{ID: user.ID, Email: user.Email}, Session: session}, nil
}

func (s Service) sign(userID string) (Session, error) {
	session, _, err := IssueSession(userID, s.secret, s.now().UTC(), s.accessTTL, s.refreshTTL)
	return session, err
}

func (s Service) ready() error {
	if s.store == nil || len(s.secret) < 32 || s.hash == nil || s.verify == nil || s.now == nil {
		return errors.New("auth service is unavailable")
	}
	if s.accessTTL <= 0 || s.refreshTTL <= 0 {
		return errors.New("auth service is unavailable")
	}
	return nil
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
	domain := email[at+1:]
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", invalid("email is invalid")
	}
	return email, nil
}

func validatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	if length < minPasswordLength {
		return invalid("password must be 12 characters or more")
	}
	if length > maxPasswordLength {
		return invalid("password must be 128 characters or fewer")
	}
	for _, r := range password {
		if unicode.IsControl(r) {
			return invalid("password contains unsupported characters")
		}
	}
	return nil
}

func invalid(message string) error {
	return &ValidationError{Message: message}
}
