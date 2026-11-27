package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"sentinel-scanner/apps/api/internal/modules/auth/model"
)

func TestRegisterLoginAndRefresh(t *testing.T) {
	store := newMemStore()
	svc := testService(store)
	ctx := context.Background()

	created, err := svc.Register(ctx, " Ada@Example.com ", "correct-horse-battery")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if created.Account.Email != "ada@example.com" || created.Session.AccessToken == "" {
		t.Fatalf("result = %+v", created.Account)
	}
	if _, err := svc.Register(ctx, "ada@example.com", "correct-horse-battery"); !errors.As(err, new(*ConflictError)) {
		t.Fatalf("duplicate = %v", err)
	}

	logged, err := svc.Login(ctx, "ada@example.com", "correct-horse-battery")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := svc.Login(ctx, "ada@example.com", "wrong-password-value"); !errors.Is(err, CredentialsError{}) {
		t.Fatalf("bad password = %v", err)
	}
	if _, err := svc.Login(ctx, "missing@example.com", "correct-horse-battery"); !errors.Is(err, CredentialsError{}) {
		t.Fatalf("missing account = %v", err)
	}

	refreshed, err := svc.Refresh(ctx, logged.Session.RefreshToken)
	if err != nil || refreshed.Account.ID != created.Account.ID {
		t.Fatalf("refresh = %+v err = %v", refreshed.Account, err)
	}
	if _, err := svc.Refresh(ctx, logged.Session.RefreshToken); !errors.Is(err, CredentialsError{}) {
		t.Fatalf("reused refresh = %v", err)
	}
	if _, err := svc.Refresh(ctx, refreshed.Session.RefreshToken); !errors.Is(err, CredentialsError{}) {
		t.Fatalf("family after reuse = %v", err)
	}
}

func TestRegisterRejectsShortPassword(t *testing.T) {
	svc := testService(newMemStore())
	_, err := svc.Register(context.Background(), "ada@example.com", "short")
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("err = %v", err)
	}
}

func testService(store Store) Service {
	svc := New(store, Config{
		Secret:     []byte("test-secret-that-is-at-least-32b"),
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
	})
	svc.hash = func(password string) (string, error) { return "$argon2id$" + password, nil }
	svc.verify = func(password, encoded string) bool { return encoded == "$argon2id$"+password }
	return svc
}

type memStore struct {
	mu     sync.Mutex
	users  map[string]model.User
	email  map[string]string
	tokens map[string]memToken
	next   int
}

type memToken struct {
	userID  string
	expires time.Time
	revoked bool
}

func newMemStore() *memStore {
	return &memStore{
		users:  map[string]model.User{},
		email:  map[string]string{},
		tokens: map[string]memToken{},
	}
}

func (s *memStore) CreateUser(_ context.Context, email, passwordHash string) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.email[email]; ok {
		return model.User{}, model.ErrConflict
	}
	s.next++
	user := model.User{ID: "11111111-1111-4111-8111-111111111111", Email: email, PasswordHash: passwordHash}
	s.users[user.ID] = user
	s.email[email] = user.ID
	return user, nil
}

func (s *memStore) FindUserByEmail(_ context.Context, email string) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.email[email]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	return s.users[id], nil
}

func (s *memStore) FindUserByID(_ context.Context, id string) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	return user, nil
}

func (s *memStore) SaveRefreshToken(_ context.Context, userID, tokenHash string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[tokenHash] = memToken{userID: userID, expires: expiresAt}
	return nil
}

func (s *memStore) RotateRefreshToken(_ context.Context, oldHash, newHash string, expiresAt time.Time) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.tokens[oldHash]
	if !ok {
		return "", model.ErrNotFound
	}
	if token.revoked || !token.expires.After(time.Now()) {
		s.revokeAll(token.userID)
		return "", model.ErrReuse
	}
	token.revoked = true
	s.tokens[oldHash] = token
	s.tokens[newHash] = memToken{userID: token.userID, expires: expiresAt}
	return token.userID, nil
}

func (s *memStore) RevokeRefreshToken(_ context.Context, tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.tokens[tokenHash]
	if !ok || token.revoked {
		return model.ErrNotFound
	}
	token.revoked = true
	s.tokens[tokenHash] = token
	return nil
}

func (s *memStore) revokeAll(userID string) {
	for hash, token := range s.tokens {
		if token.userID == userID {
			token.revoked = true
			s.tokens[hash] = token
		}
	}
}
