package service

import (
	"context"
	"errors"
	"testing"
)

type stub struct {
	err error
}

func (s stub) Ping(context.Context) error {
	return s.err
}

func TestCheckRequiresBothDependencies(t *testing.T) {
	err := New(stub{}, stub{}).Check(context.Background())
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	err = New(stub{err: errors.New("dial tcp password=secret")}, stub{}).Check(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
	if err.Error() == "dial tcp password=secret" || containsSecret(err.Error()) {
		t.Fatalf("driver error leaked: %v", err)
	}
}

func TestCheckRejectsMissingDependency(t *testing.T) {
	err := New(nil, stub{}).Check(context.Background())
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error = %v", err)
	}
}

func containsSecret(value string) bool {
	const needle = "password=secret"
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
