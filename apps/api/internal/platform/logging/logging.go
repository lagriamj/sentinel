package logging

import (
	"log/slog"
	"os"
)

// New returns the process logger. Log records are JSON on stdout.
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
