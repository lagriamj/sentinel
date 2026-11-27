package migrate

import "testing"

func TestUpRejectsInvalidURL(t *testing.T) {
	tests := []string{"", "mysql://localhost/sentinel", "http://localhost/sentinel"}
	for _, databaseURL := range tests {
		if err := Up(databaseURL, nil); err == nil {
			t.Fatalf("Up(%q) succeeded", databaseURL)
		}
	}
}
