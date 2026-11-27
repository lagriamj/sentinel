package repository

import (
	"strings"
	"testing"

	"sentinel-scanner/apps/api/internal/modules/scanjob/model"
	"sentinel-scanner/apps/api/migrations"
)

func TestAssessmentMigrationMatchesJobModel(t *testing.T) {
	up, err := migrations.Files.ReadFile("000002_create_assessment_schema.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := migrations.Files.ReadFile("000002_create_assessment_schema.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	upSQL := string(up)
	downSQL := string(down)
	for _, table := range []string{"users", "scan_targets", "scan_jobs", "scan_results", "vulnerabilities"} {
		if !strings.Contains(upSQL, "CREATE TABLE "+table+" (") {
			t.Fatalf("up migration missing %s", table)
		}
		if !strings.Contains(downSQL, "DROP TABLE IF EXISTS "+table+";") {
			t.Fatalf("down migration missing %s", table)
		}
	}
	for _, status := range model.Statuses() {
		if !strings.Contains(upSQL, "'"+string(status)+"'") {
			t.Fatalf("up migration missing status %s", status)
		}
	}
	if !strings.Contains(upSQL, "password_hash LIKE '$argon2id$%'") {
		t.Fatal("users table does not require an argon2id hash")
	}
	if !strings.Contains(upSQL, "CONSTRAINT scan_targets_authorized CHECK (authorized)") {
		t.Fatal("targets can be stored without authorization")
	}
}
