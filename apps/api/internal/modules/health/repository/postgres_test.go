package repository

import (
	"context"
	"testing"
)

func TestPingWithoutPool(t *testing.T) {
	err := NewPostgres(nil).Ping(context.Background())
	if err == nil {
		t.Fatal("expected ping without a pool to fail")
	}
}
