package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"sentinel-scanner/apps/api/internal/modules/scanjob/model"
	"sentinel-scanner/apps/api/internal/platform/migrate"
	"sentinel-scanner/apps/api/internal/platform/postgres"
	"sentinel-scanner/apps/api/migrations"

	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL != "" {
		if err := migrate.Up(databaseURL, migrations.Files); err != nil {
			fmt.Fprintf(os.Stderr, "migrate test database: %v\n", err)
			os.Exit(1)
		}
		db, err := postgres.Open(context.Background(), databaseURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open test database: %v\n", err)
			os.Exit(1)
		}
		testDB = db
	}
	code := m.Run()
	if testDB != nil {
		if sqlDB, err := testDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	os.Exit(code)
}

const ownerID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

var (
	testDB   *gorm.DB
	testLock sync.Mutex
)

func TestPostgresUnavailable(t *testing.T) {
	repo := NewPostgres(nil)
	ctx := context.Background()
	if _, err := repo.Create(ctx, model.CreateInput{}); err == nil {
		t.Fatal("expected create without a pool to fail")
	}
	if _, err := repo.List(ctx, model.ListFilter{Page: 1, PageSize: 20}); err == nil {
		t.Fatal("expected list without a pool to fail")
	}
	if _, err := repo.Get(ctx, ownerID, "11111111-1111-4111-8111-111111111111"); err == nil {
		t.Fatal("expected get without a pool to fail")
	}
	if _, err := repo.Cancel(ctx, ownerID, "11111111-1111-4111-8111-111111111111"); err == nil {
		t.Fatal("expected cancel without a pool to fail")
	}
	if err := repo.Delete(ctx, ownerID, "11111111-1111-4111-8111-111111111111"); err == nil {
		t.Fatal("expected delete without a pool to fail")
	}
}

func TestPostgresScanJobs(t *testing.T) {
	db := openDB(t)
	repo := NewPostgres(db)
	ctx := context.Background()

	first := createJob(t, repo, "First", "https://203.0.113.10/app")
	if first.Status != model.StatusPending {
		t.Fatalf("status = %s", first.Status)
	}
	if first.Target.Authorized != true || first.Target.Port != 443 || first.Target.Host != "203.0.113.10" {
		t.Fatalf("target = %+v", first.Target)
	}
	if first.FinishedAt != nil {
		t.Fatal("pending job has finished_at")
	}

	time.Sleep(10 * time.Millisecond)
	second := createJob(t, repo, "Second", "https://203.0.113.10/app")
	if second.Target.ID != first.Target.ID {
		t.Fatalf("target reused = %s, want %s", second.Target.ID, first.Target.ID)
	}
	var targets int64
	if err := db.Raw(`SELECT count(*) FROM scan_targets`).Scan(&targets).Error; err != nil {
		t.Fatalf("count targets: %v", err)
	}
	if targets != 1 {
		t.Fatalf("targets = %d", targets)
	}

	time.Sleep(10 * time.Millisecond)
	third := createJob(t, repo, "Third", "https://203.0.113.11:8443/health")
	if third.Target.ID == first.Target.ID || third.Target.Port != 8443 {
		t.Fatalf("third target = %+v", third.Target)
	}

	got, err := repo.Get(ctx, ownerID, first.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "First" || got.Target.URL != "https://203.0.113.10/app" {
		t.Fatalf("job = %+v", got)
	}
	if _, err := repo.Get(ctx, ownerID, "99999999-9999-4999-8999-999999999999"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("missing get error = %v", err)
	}

	if err := db.Exec(`
UPDATE scan_jobs
SET status = 'failed', finished_at = now(), error_message = 'engine failed', updated_at = now()
WHERE id = CAST(? AS uuid)
`, third.ID).Error; err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	pending, err := repo.List(ctx, model.ListFilter{OwnerID: ownerID, Status: model.StatusPending, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if pending.Total != 2 || len(pending.Jobs) != 2 {
		t.Fatalf("pending = total %d len %d", pending.Total, len(pending.Jobs))
	}
	if pending.Jobs[0].ID != second.ID || pending.Jobs[1].ID != first.ID {
		t.Fatalf("pending order = %s, %s", pending.Jobs[0].ID, pending.Jobs[1].ID)
	}

	byTarget, err := repo.List(ctx, model.ListFilter{OwnerID: ownerID, TargetID: first.Target.ID, Page: 1, PageSize: 1})
	if err != nil {
		t.Fatalf("list target: %v", err)
	}
	if byTarget.Total != 2 || len(byTarget.Jobs) != 1 || byTarget.Jobs[0].ID != second.ID {
		t.Fatalf("target page = %+v", byTarget)
	}
	secondPage, err := repo.List(ctx, model.ListFilter{OwnerID: ownerID, TargetID: first.Target.ID, Page: 2, PageSize: 1})
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if secondPage.Total != 2 || len(secondPage.Jobs) != 1 || secondPage.Jobs[0].ID != first.ID {
		t.Fatalf("page 2 = %+v", secondPage)
	}

	cancelled, err := repo.Cancel(ctx, ownerID, first.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cancelled.Status != model.StatusCancelled || cancelled.FinishedAt == nil {
		t.Fatalf("cancelled = %+v", cancelled)
	}
	if _, err := repo.Cancel(ctx, ownerID, first.ID); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("second cancel error = %v", err)
	}
	if _, err := repo.Cancel(ctx, ownerID, third.ID); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("cancel failed job error = %v", err)
	}

	if err := db.Exec(`
UPDATE scan_jobs
SET status = 'running', started_at = now(), finished_at = NULL, updated_at = now()
WHERE id = CAST(? AS uuid)
`, second.ID).Error; err != nil {
		t.Fatalf("mark running: %v", err)
	}
	runningCancelled, err := repo.Cancel(ctx, ownerID, second.ID)
	if err != nil {
		t.Fatalf("cancel running: %v", err)
	}
	if runningCancelled.Status != model.StatusCancelled || runningCancelled.StartedAt == nil || runningCancelled.FinishedAt == nil {
		t.Fatalf("cancelled running = %+v", runningCancelled)
	}

	if err := repo.Delete(ctx, ownerID, third.ID); err != nil {
		t.Fatalf("delete failed job: %v", err)
	}
	if _, err := repo.Get(ctx, ownerID, third.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("deleted get error = %v", err)
	}
	if err := repo.Delete(ctx, ownerID, third.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("second delete error = %v", err)
	}
}

func TestPostgresDeleteRetainsQueuedJob(t *testing.T) {
	db := openDB(t)
	repo := NewPostgres(db)
	job := createJob(t, repo, "Queued", "https://203.0.113.20/")
	if err := db.Exec(`
UPDATE scan_jobs
SET status = 'queued', updated_at = now()
WHERE id = CAST(? AS uuid)
`, job.ID).Error; err != nil {
		t.Fatalf("mark queued: %v", err)
	}
	if err := repo.Delete(context.Background(), ownerID, job.ID); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("delete queued error = %v", err)
	}
	if _, err := repo.Get(context.Background(), ownerID, job.ID); err != nil {
		t.Fatalf("queued job was removed: %v", err)
	}
}

func TestPostgresDeleteCascadesResultsAndFindings(t *testing.T) {
	db := openDB(t)
	repo := NewPostgres(db)
	job := createJob(t, repo, "Cascade", "https://203.0.113.30/report")
	if err := db.Exec(`
INSERT INTO scan_results (scan_job_id, engine, summary)
VALUES (CAST(? AS uuid), 'nuclei', 'no findings')
`, job.ID).Error; err != nil {
		t.Fatalf("insert result: %v", err)
	}
	if err := db.Exec(`
INSERT INTO vulnerabilities (scan_job_id, scan_result_id, target_id, name, severity, location)
SELECT CAST(? AS uuid), scan_results.id, CAST(? AS uuid), 'Example finding', 'low', '/report'
FROM scan_results
WHERE scan_job_id = CAST(? AS uuid)
`, job.ID, job.Target.ID, job.ID).Error; err != nil {
		t.Fatalf("insert vulnerability: %v", err)
	}

	if err := repo.Delete(context.Background(), ownerID, job.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var results, findings, targets int64
	if err := db.Raw(`SELECT count(*) FROM scan_results`).Scan(&results).Error; err != nil {
		t.Fatalf("count results: %v", err)
	}
	if err := db.Raw(`SELECT count(*) FROM vulnerabilities`).Scan(&findings).Error; err != nil {
		t.Fatalf("count findings: %v", err)
	}
	if err := db.Raw(`SELECT count(*) FROM scan_targets`).Scan(&targets).Error; err != nil {
		t.Fatalf("count targets: %v", err)
	}
	if results != 0 || findings != 0 || targets != 1 {
		t.Fatalf("results=%d findings=%d targets=%d", results, findings, targets)
	}
}

func TestPostgresSchemaConstraints(t *testing.T) {
	db := openDB(t)
	repo := NewPostgres(db)
	job := createJob(t, repo, "Constraints", "https://203.0.113.40/")

	err := db.Exec(`
UPDATE scan_jobs
SET status = 'completed', updated_at = now()
WHERE id = CAST(? AS uuid)
`, job.ID).Error
	if err == nil {
		t.Fatal("completed job without finished_at was stored")
	}

	const hash = "$argon2id$v=19$m=65536,t=1,p=1$c2FsdHNhbHQ$aGFzaA"
	if err := db.Exec(`INSERT INTO users (email, password_hash) VALUES (?, ?)`, "ada@example.com", "plaintext-password").Error; err == nil {
		t.Fatal("plaintext password was stored")
	}
	if err := db.Exec(`INSERT INTO users (email, password_hash) VALUES (?, ?)`, "ada@example.com", hash).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (email, password_hash) VALUES (?, ?)`, "Ada@Example.com", hash).Error; err == nil {
		t.Fatal("case-variant email was stored")
	}
	if err := db.Exec(`
INSERT INTO vulnerabilities (scan_job_id, target_id, name, severity)
VALUES (CAST(? AS uuid), CAST(? AS uuid), 'Bad severity', 'urgent')
`, job.ID, job.Target.ID).Error; err == nil {
		t.Fatal("invalid severity was stored")
	}
}

func TestPostgresConcurrentCreateReusesTarget(t *testing.T) {
	db := openDB(t)
	repo := NewPostgres(db)
	targetID := ensureTarget(t, repo, model.TargetInput{
		URL: "https://203.0.113.50/shared", Scheme: "https", Host: "203.0.113.50", Port: 443,
	})
	const workers = 8
	errCh := make(chan error, workers)
	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := repo.Create(context.Background(), model.CreateInput{
				OwnerID:  ownerID,
				Name:     "Concurrent",
				TargetID: targetID,
			})
			errCh <- err
		}()
	}
	group.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent create: %v", err)
		}
	}
	var targets, jobs int64
	if err := db.Raw(`SELECT count(*) FROM scan_targets`).Scan(&targets).Error; err != nil {
		t.Fatalf("count targets: %v", err)
	}
	if err := db.Raw(`SELECT count(*) FROM scan_jobs`).Scan(&jobs).Error; err != nil {
		t.Fatalf("count jobs: %v", err)
	}
	if targets != 1 || jobs != int64(workers) {
		t.Fatalf("targets=%d jobs=%d", targets, jobs)
	}
}

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	if testDB == nil {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	testLock.Lock()
	t.Cleanup(testLock.Unlock)
	clean(t, testDB)
	return testDB
}

func clean(t *testing.T, db *gorm.DB) {
	t.Helper()
	err := db.Exec("TRUNCATE TABLE vulnerabilities, scan_results, scan_jobs, scan_targets, project_members, projects, refresh_tokens, users RESTART IDENTITY CASCADE").Error
	if err != nil {
		t.Fatalf("clean database: %v", err)
	}
}

func createJob(t *testing.T, repo Postgres, name, rawURL string) model.Job {
	t.Helper()
	targetID := ensureTarget(t, repo, targetFromURL(t, rawURL))
	job, err := repo.Create(context.Background(), model.CreateInput{OwnerID: ownerID, Name: name, TargetID: targetID})
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if job.ID == "" || job.Target.ID == "" {
		t.Fatalf("create %s returned empty ids: %+v", name, job)
	}
	return job
}

func ensureOwner(t *testing.T, repo Postgres) {
	t.Helper()
	err := repo.db.Exec(`
INSERT INTO users (id, email, password_hash)
VALUES (CAST(? AS uuid), 'owner@example.com', '$argon2id$test-hash-value')
ON CONFLICT (id) DO NOTHING
`, ownerID).Error
	if err != nil {
		t.Fatalf("insert owner: %v", err)
	}
}

const projectID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"

func ensureTarget(t *testing.T, repo Postgres, input model.TargetInput) string {
	t.Helper()
	ensureOwner(t, repo)
	err := repo.db.Exec(`
INSERT INTO projects (id, name, created_by)
VALUES (CAST(? AS uuid), 'Test', CAST(? AS uuid))
ON CONFLICT (id) DO NOTHING
`, projectID, ownerID).Error
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	err = repo.db.Exec(`
INSERT INTO project_members (project_id, user_id, role)
VALUES (CAST(? AS uuid), CAST(? AS uuid), 'owner')
ON CONFLICT (project_id, user_id) DO NOTHING
`, projectID, ownerID).Error
	if err != nil {
		t.Fatalf("insert membership: %v", err)
	}
	var id string
	err = repo.db.Raw(`
INSERT INTO scan_targets (project_id, url, scheme, host, port, authorized)
VALUES (CAST(? AS uuid), ?, ?, ?, ?, TRUE)
ON CONFLICT (project_id, url) DO UPDATE SET url = EXCLUDED.url
RETURNING id::text
`, projectID, input.URL, input.Scheme, input.Host, input.Port).Scan(&id).Error
	if err != nil || id == "" {
		t.Fatalf("insert target: %v", err)
	}
	return id
}

func targetFromURL(t *testing.T, rawURL string) model.TargetInput {
	t.Helper()
	switch rawURL {
	case "https://203.0.113.10/app":
		return model.TargetInput{URL: rawURL, Scheme: "https", Host: "203.0.113.10", Port: 443}
	case "https://203.0.113.11:8443/health":
		return model.TargetInput{URL: rawURL, Scheme: "https", Host: "203.0.113.11", Port: 8443}
	case "https://203.0.113.20/":
		return model.TargetInput{URL: "https://203.0.113.20", Scheme: "https", Host: "203.0.113.20", Port: 443}
	case "https://203.0.113.30/report":
		return model.TargetInput{URL: rawURL, Scheme: "https", Host: "203.0.113.30", Port: 443}
	case "https://203.0.113.40/":
		return model.TargetInput{URL: "https://203.0.113.40", Scheme: "https", Host: "203.0.113.40", Port: 443}
	default:
		t.Fatalf("unsupported test url %s", rawURL)
		return model.TargetInput{}
	}
}
