package model

import "errors"

// ErrNotFound means the account or refresh token does not exist.
var ErrNotFound = errors.New("auth record not found")

// ErrConflict means the email is already registered.
var ErrConflict = errors.New("email already registered")

// ErrReuse means a revoked refresh token was presented again.
var ErrReuse = errors.New("refresh token reuse")

// User is an operator account.
type User struct {
	ID           string
	Email        string
	PasswordHash string
}
