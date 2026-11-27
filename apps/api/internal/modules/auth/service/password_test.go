package service

import "testing"

func TestHashPasswordRoundTrip(t *testing.T) {
	encoded, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if len(encoded) < 16 || encoded[:10] != "$argon2id$" {
		t.Fatalf("encoded = %s", encoded)
	}
	if !VerifyPassword("correct-horse-battery", encoded) {
		t.Fatal("expected matching password to verify")
	}
	if VerifyPassword("wrong-horse-battery", encoded) {
		t.Fatal("expected a different password to fail")
	}
	if VerifyPassword("correct-horse-battery", "plaintext") {
		t.Fatal("expected a malformed hash to fail")
	}
}
