package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// RepoRoot returns the git repository root detected from dir.
// The returned path matches the symlink form of the input dir (important on
// macOS where /var is a symlink to /private/var but os.TempDir returns /var/...).
func RepoRoot(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	root := strings.TrimSpace(string(out))
	// git resolves symlinks; map the result back to the caller's path form.
	// Resolve both dir and root to their canonical forms, then reconstruct
	// the root using dir's prefix when they share the same canonical path.
	resolvedDir, err2 := filepath.EvalSymlinks(dir)
	if err2 == nil && resolvedDir != dir {
		// root uses the resolved form; replace that prefix with the original dir prefix.
		// Find the common ancestor: walk up dir to find what matches root.
		resolvedRoot, err3 := filepath.EvalSymlinks(root)
		if err3 == nil {
			// resolvedRoot and resolvedDir should share a prefix
			// Reconstruct root by replacing the resolved prefix with original.
			// The difference between resolvedDir and dir gives us the mapping.
			if len(resolvedDir) <= len(resolvedRoot) && resolvedRoot[:len(resolvedDir)] == resolvedDir {
				root = dir + resolvedRoot[len(resolvedDir):]
			} else if len(resolvedRoot) <= len(resolvedDir) && resolvedDir[:len(resolvedRoot)] == resolvedRoot {
				// root is a parent of dir; find how much of dir to keep
				suffix := resolvedDir[len(resolvedRoot):]
				original := dir
				if len(original) > len(suffix) {
					root = original[:len(original)-len(suffix)]
				}
			}
		}
	}
	return root, nil
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
	out, err := exec.Command("git", "-C", repoRoot, "describe", "--tags", "--always").Output()
	if err != nil {
		// --always falls back to SHA; if that also fails, use ShortSHA
		return ShortSHA(repoRoot)
	}
	v := strings.TrimSpace(string(out))
	// If the result looks like a bare SHA (no v prefix, no hyphen-number), it has no tags
	if !strings.HasPrefix(v, "v") && !strings.Contains(v, "-") {
		return "untagged-" + v, nil
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
