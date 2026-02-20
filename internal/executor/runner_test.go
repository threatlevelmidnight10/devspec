package executor

import (
	"testing"
	"time"
)

func TestMakeBranchName(t *testing.T) {
	ts := time.Date(2026, 2, 20, 15, 0, 0, 0, time.UTC)
	got := makeBranchName("agent/", "schema migration v1", ts)
	want := "agent/schema-migration-v1-20260220-150000"
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}
