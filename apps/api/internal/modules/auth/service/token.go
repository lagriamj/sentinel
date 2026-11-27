package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const accessIssuer = "sentinel-scanner"

// Session is a new access token and the raw refresh token that pairs with it.
type Session struct {
	AccessToken    string
	ExpiresIn      int
	RefreshToken   string
	RefreshExpires time.Time
}

type accessClaims struct {
	jwt.RegisteredClaims
}

// IssueSession signs a short-lived access token and returns a new refresh token.
func IssueSession(userID string, secret []byte, now time.Time, accessTTL, refreshTTL time.Duration) (Session, string, error) {
	if userID == "" || len(secret) < 32 {
		return Session{}, "", errors.New("token issuer is not configured")
	}
	if accessTTL <= 0 || refreshTTL <= 0 {
		return Session{}, "", errors.New("token lifetime is invalid")
	}
	raw, hash, err := newRefreshToken()
	if err != nil {
		return Session{}, "", err
	}
	tokenID, err := randomID()
	if err != nil {
		return Session{}, "", err
	}
	expires := now.Add(accessTTL)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    accessIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
			ID:        tokenID,
		},
	})
	signed, err := token.SignedString(secret)
	if err != nil {
		return Session{}, "", fmt.Errorf("sign access token: %w", err)
	}
	return Session{
		AccessToken:    signed,
		ExpiresIn:      int(accessTTL.Seconds()),
		RefreshToken:   raw,
		RefreshExpires: now.Add(refreshTTL),
	}, hash, nil
}

// ParseAccessToken returns the user id from a signed access token.
func ParseAccessToken(raw string, secret []byte, now time.Time) (string, error) {
	claims := accessClaims{}
	token, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(accessIssuer), jwt.WithTimeFunc(func() time.Time {
		return now
	}))
	if err != nil || !token.Valid || claims.Subject == "" {
		return "", errors.New("access token is invalid")
	}
	return claims.Subject, nil
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func newRefreshToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:]), nil
}

// HashRefreshToken returns the stored form of a raw refresh token.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
