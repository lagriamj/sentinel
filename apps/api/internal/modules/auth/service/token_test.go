package service

import (
	"testing"
	"time"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	secret := []byte("test-secret-that-is-at-least-32b")
	now := time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	session, hash, err := IssueSession("11111111-1111-4111-8111-111111111111", secret, now, 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if session.ExpiresIn != 900 || session.RefreshToken == "" || len(hash) != 64 {
		t.Fatalf("session = %+v hash length %d", session, len(hash))
	}
	if HashRefreshToken(session.RefreshToken) != hash {
		t.Fatal("refresh hash mismatch")
	}
	userID, err := ParseAccessToken(session.AccessToken, secret, now.Add(time.Minute))
	if err != nil || userID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("parse = %s err = %v", userID, err)
	}
	if _, err := ParseAccessToken(session.AccessToken, secret, now.Add(16*time.Minute)); err == nil {
		t.Fatal("expected an expired token to fail")
	}
	if _, err := ParseAccessToken(session.AccessToken, []byte("another-secret-that-is-at-least-32"), now); err == nil {
		t.Fatal("expected a different secret to fail")
	}
}
