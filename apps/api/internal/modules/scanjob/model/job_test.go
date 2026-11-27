package model

import "testing"

func TestStatusRules(t *testing.T) {
	if got := len(Statuses()); got != 6 {
		t.Fatalf("statuses = %d", got)
	}
	for _, status := range Statuses() {
		if !status.Valid() {
			t.Fatalf("%s is not valid", status)
		}
	}
	if Status("done").Valid() || Status("").Valid() {
		t.Fatal("expected unknown statuses to be rejected")
	}

	if !StatusPending.CanCancel() || !StatusQueued.CanCancel() || !StatusRunning.CanCancel() {
		t.Fatal("expected open jobs to be cancellable")
	}
	if StatusCompleted.CanCancel() || StatusFailed.CanCancel() || StatusCancelled.CanCancel() {
		t.Fatal("expected terminal jobs to stay unchanged on cancel")
	}

	if !StatusPending.CanDelete() || !StatusCompleted.CanDelete() || !StatusFailed.CanDelete() || !StatusCancelled.CanDelete() {
		t.Fatal("expected inactive jobs to be deletable")
	}
	if StatusQueued.CanDelete() || StatusRunning.CanDelete() {
		t.Fatal("expected queued and running jobs to be retained")
	}
}

func TestParseID(t *testing.T) {
	const canonical = "11111111-1111-4111-8111-111111111111"
	got, err := ParseID("11111111-1111-4111-8111-111111111111")
	if err != nil || got != canonical {
		t.Fatalf("ParseID() = %q, %v", got, err)
	}
	upper, err := ParseID("AAAAAAAA-BBBB-4CCC-8DDD-EEEEEEEEEEEE")
	if err != nil || upper != "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee" {
		t.Fatalf("ParseID(upper) = %q, %v", upper, err)
	}

	rejected := []string{
		"",
		"11111111-1111-4111-8111-11111111111",
		"11111111111141118111111111111111",
		"11111111-1111-4111-8111-11111111111g",
		"11111111-1111-4111-8111-111111111111 ",
	}
	for _, value := range rejected {
		if _, err := ParseID(value); err == nil {
			t.Fatalf("ParseID(%q) succeeded", value)
		}
	}
}
