package ledger_test

import (
	"testing"
	"time"

	"github.com/nyamage/skraft/internal/ledger"
)

func openMemory(t *testing.T) *ledger.Ledger {
	t.Helper()
	l, err := ledger.Open(":memory:")
	if err != nil {
		t.Fatalf("ledger.Open: %v", err)
	}
	t.Cleanup(func() { l.Close() })
	return l
}

func TestOpen_CreatesSchema(t *testing.T) {
	l := openMemory(t)
	// If schema creation failed, Open would have returned an error.
	_ = l
}

func TestSetAndGetUploadState(t *testing.T) {
	l := openMemory(t)

	state := ledger.UploadState{
		SkillName:   "skill-a",
		Target:      "claudeai",
		Version:     "v1.2.0",
		ContentHash: "abc123",
		UploadedAt:  time.Now().UTC().Truncate(time.Second),
	}
	if err := l.SetUploadState(state); err != nil {
		t.Fatalf("SetUploadState: %v", err)
	}

	got, err := l.GetUploadState("skill-a", "claudeai")
	if err != nil {
		t.Fatalf("GetUploadState: %v", err)
	}
	if got == nil {
		t.Fatal("expected state, got nil")
	}
	if got.Version != "v1.2.0" {
		t.Errorf("Version = %q, want v1.2.0", got.Version)
	}
	if got.ContentHash != "abc123" {
		t.Errorf("ContentHash = %q, want abc123", got.ContentHash)
	}
	if !got.UploadedAt.Equal(state.UploadedAt) {
		t.Errorf("UploadedAt = %v, want %v", got.UploadedAt, state.UploadedAt)
	}
}

func TestGetUploadState_NotFound(t *testing.T) {
	l := openMemory(t)
	got, err := l.GetUploadState("nonexistent", "claudeai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestSetUploadState_Upsert(t *testing.T) {
	l := openMemory(t)

	first := ledger.UploadState{SkillName: "skill-a", Target: "claudeai", Version: "v1.0.0", ContentHash: "hash1", UploadedAt: time.Now().UTC()}
	second := ledger.UploadState{SkillName: "skill-a", Target: "claudeai", Version: "v1.1.0", ContentHash: "hash2", UploadedAt: time.Now().UTC()}

	if err := l.SetUploadState(first); err != nil {
		t.Fatal(err)
	}
	if err := l.SetUploadState(second); err != nil {
		t.Fatal(err)
	}

	got, err := l.GetUploadState("skill-a", "claudeai")
	if err != nil {
		t.Fatalf("GetUploadState after upsert: %v", err)
	}
	if got.Version != "v1.1.0" {
		t.Errorf("after upsert: Version = %q, want v1.1.0", got.Version)
	}
}
