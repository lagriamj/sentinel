package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory     = 64 * 1024
	argonIterations = 3
	argonThreads    = 4
	argonKeyLength  = 32
	argonSaltLength = 16
)

// HashPassword returns an Argon2id PHC string.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	sum := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonThreads, argonKeyLength)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonIterations,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum),
	), nil
}

// VerifyPassword reports whether password matches an Argon2id PHC string.
func VerifyPassword(password, encoded string) bool {
	salt, sum, memory, iterations, threads, ok := decodePassword(encoded)
	if !ok {
		return false
	}
	other := argon2.IDKey([]byte(password), salt, iterations, memory, threads, uint32(len(sum)))
	return subtle.ConstantTimeCompare(sum, other) == 1
}

func decodePassword(encoded string) (salt, sum []byte, memory, iterations uint32, threads uint8, ok bool) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return nil, nil, 0, 0, 0, false
	}
	var parsedThreads uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parsedThreads); err != nil {
		return nil, nil, 0, 0, 0, false
	}
	if memory < 8 || iterations == 0 || parsedThreads == 0 || parsedThreads > 255 {
		return nil, nil, 0, 0, 0, false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 {
		return nil, nil, 0, 0, 0, false
	}
	sum, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(sum) < 16 {
		return nil, nil, 0, 0, 0, false
	}
	return salt, sum, memory, iterations, uint8(parsedThreads), true
}
