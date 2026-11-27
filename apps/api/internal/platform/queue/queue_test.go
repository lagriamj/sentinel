package queue

import (
	"context"
	"testing"
)

func TestOpenRejectsInvalidURL(t *testing.T) {
	tests := []string{
		"",
		"postgres://localhost/sentinel",
		"http://localhost:6379",
		"://missing-host",
	}
	for _, redisURL := range tests {
		_, err := Open(context.Background(), redisURL)
		if err == nil {
			t.Fatalf("Open(%q) succeeded", redisURL)
		}
	}
}
