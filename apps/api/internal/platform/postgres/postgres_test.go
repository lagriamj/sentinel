package postgres

import (
	"context"
	"testing"
)

func TestOpenRejectsInvalidURL(t *testing.T) {
	tests := []string{
		"",
		"not a url",
		"mysql://localhost/sentinel",
		"http://localhost/sentinel",
	}
	for _, databaseURL := range tests {
		_, err := Open(context.Background(), databaseURL)
		if err == nil {
			t.Fatalf("Open(%q) succeeded", databaseURL)
		}
	}
}
