package migrate

import (
	"database/sql"
	"errors"
	"io/fs"
	"net/url"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Up applies all pending migrations.
func Up(databaseURL string, files fs.FS) error {
	migrator, err := open(databaseURL, files)
	if err != nil {
		return err
	}
	defer closeMigrator(migrator)
	return sanitize(migrator.Up())
}

// Down rolls back the most recently applied migration.
func Down(databaseURL string, files fs.FS) error {
	migrator, err := open(databaseURL, files)
	if err != nil {
		return err
	}
	defer closeMigrator(migrator)
	err = migrator.Steps(-1)
	if errors.Is(err, migrate.ErrNilVersion) {
		return errors.New("no applied migrations")
	}
	return sanitize(err)
}

func open(databaseURL string, files fs.FS) (*migrate.Migrate, error) {
	if err := validateURL(databaseURL); err != nil {
		return nil, err
	}
	source, err := iofs.New(files, ".")
	if err != nil {
		return nil, errors.New("migration files are unreadable")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, errors.New("database connection failed")
	}
	// Files may contain several statements. The splitter cuts on every semicolon
	// and does not understand dollar quotes, so migration SQL stays free of
	// PL/pgSQL function bodies.
	driver, err := postgres.WithInstance(db, &postgres.Config{
		MultiStatementEnabled: true,
		MultiStatementMaxSize: postgres.DefaultMultiStatementMaxSize,
	})
	if err != nil {
		_ = db.Close()
		return nil, errors.New("database connection failed")
	}
	migrator, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		_ = db.Close()
		return nil, errors.New("migration setup failed")
	}
	return migrator, nil
}

func closeMigrator(migrator *migrate.Migrate) {
	_, _ = migrator.Close()
}

func sanitize(err error) error {
	if err == nil || errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	message := err.Error()
	if strings.Contains(message, "://") || strings.Contains(message, "@") {
		return errors.New("migration failed")
	}
	return err
}

func validateURL(databaseURL string) error {
	parsed, err := url.Parse(databaseURL)
	if err != nil || parsed.Host == "" {
		return errors.New("database url is invalid")
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return errors.New("database url scheme must be postgres")
	}
	return nil
}
