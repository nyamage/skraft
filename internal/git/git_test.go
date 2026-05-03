package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/nyamage/skraft/internal/git"
)

// makeTempRepo creates a temp git repo with one commit. Returns the repo dir.
func makeTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")
	return dir
}

func TestRepoRoot(t *testing.T) {
	dir := makeTempRepo(t)
	sub := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	got, err := git.RepoRoot(sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != dir {
		t.Errorf("RepoRoot(%q) = %q, want %q", sub, got, dir)
	}
}

func TestLatestTag_NoTags(t *testing.T) {
	dir := makeTempRepo(t)
	tag, err := git.LatestTag(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "" {
		t.Errorf("expected empty tag, got %q", tag)
	}
}

func TestLatestTag_WithTag(t *testing.T) {
	dir := makeTempRepo(t)
	run := func(args ...string) {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}
	run("git", "tag", "v1.2.3")
	tag, err := git.LatestTag(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v1.2.3" {
		t.Errorf("LatestTag = %q, want v1.2.3", tag)
	}
}

func TestVersion_NoTags(t *testing.T) {
	dir := makeTempRepo(t)
	v, err := git.Version(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should start with "untagged-"
	if len(v) < 9 || v[:9] != "untagged-" {
		t.Errorf("Version with no tags = %q, want prefix 'untagged-'", v)
	}
}

func TestVersion_AtTag(t *testing.T) {
	dir := makeTempRepo(t)
	run := func(args ...string) {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s", args, out)
		}
	}
	run("git", "tag", "v2.0.0")
	v, err := git.Version(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "v2.0.0" {
		t.Errorf("Version at tag = %q, want v2.0.0", v)
	}
}
