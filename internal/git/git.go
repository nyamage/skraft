package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// RepoRoot returns the git repository root detected from dir.
func RepoRoot(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// LatestTag returns the most recent semver tag reachable from HEAD,
// or empty string if no tags exist.
func LatestTag(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "describe", "--tags", "--abbrev=0").Output()
	if err != nil {
		// No tags — not an error for callers
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// ShortSHA returns the short SHA of HEAD prefixed with "untagged-".
func ShortSHA(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --short HEAD: %w", err)
	}
	return "untagged-" + strings.TrimSpace(string(out)), nil
}

// Version returns the version string for the repository.
// At a tag: "v1.2.0". Ahead of a tag: "v1.2.0-3-gabcdef".
// No tags: "untagged-abcdef".
func Version(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "describe", "--tags", "--long").Output()
	if err != nil {
		// No tags — fall back to short SHA with "untagged-" prefix
		return ShortSHA(repoRoot)
	}
	// Output format: <tag>-<N>-g<sha>  (N = commits ahead, sha = abbreviated)
	// Tags may contain "-", so parse backwards from the end.
	v := strings.TrimSpace(string(out))
	lastDash := strings.LastIndex(v, "-")
	if lastDash < 0 {
		return "untagged-" + v, nil
	}
	secondLastDash := strings.LastIndex(v[:lastDash], "-")
	if secondLastDash < 0 {
		return "untagged-" + v, nil
	}
	n := v[secondLastDash+1 : lastDash]
	tag := v[:secondLastDash]
	if n == "0" {
		// Exactly at the tag
		return tag, nil
	}
	return v, nil
}

// HeadSHA returns the full SHA of HEAD.
func HeadSHA(repoRoot string) (string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
