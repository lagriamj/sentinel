package main

import (
	"os"

	"sentinel-scanner/apps/api/internal/platform/config"
	"sentinel-scanner/apps/api/internal/platform/logging"
	"sentinel-scanner/apps/api/internal/platform/migrate"
	"sentinel-scanner/apps/api/migrations"
)

func main() {
	logger := logging.New()
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "down") {
		logger.Error("configuration_invalid", "error", "usage: migrate up|down")
		os.Exit(2)
	}

	databaseURL, err := config.LoadDatabaseURL()
	if err != nil {
		logger.Error("configuration_invalid", "error", err.Error())
		os.Exit(1)
	}

	switch os.Args[1] {
	case "up":
		err = migrate.Up(databaseURL, migrations.Files)
	case "down":
		err = migrate.Down(databaseURL, migrations.Files)
	}
	if err != nil {
		logger.Error("migration_failed", "error", err.Error())
		os.Exit(1)
	}
	logger.Info("migration_complete", "command", os.Args[1])
}
