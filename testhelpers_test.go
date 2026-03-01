package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initBareRepo creates a bare git repo in a temp directory and returns its path.
func initBareRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bare := filepath.Join(dir, "remote.git")
	run(t, "git", "init", "--bare", bare)
	return bare
}

// initRepo creates a git repo with an initial commit and returns its path.
// If remotePath is non-empty, it's added as the "origin" remote and pushed.
func initRepo(t *testing.T, remotePath string) string {
	t.Helper()
	dir := t.TempDir()
	run(t, "git", "init", dir)
	run(t, "git", "-C", dir, "config", "user.email", "test@test.com")
	run(t, "git", "-C", dir, "config", "user.name", "Test")

	// Create initial commit so HEAD exists
	initial := filepath.Join(dir, "README.md")
	if err := os.WriteFile(initial, []byte("init\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run(t, "git", "-C", dir, "add", "-A")
	run(t, "git", "-C", dir, "commit", "-m", "initial commit")

	if remotePath != "" {
		run(t, "git", "-C", dir, "remote", "add", "origin", remotePath)
		run(t, "git", "-C", dir, "push", "-u", "origin", "main")
	}
	return dir
}

// initLocalRepo creates a standalone git repo (no remote) for tests that
// only need a valid git directory.
func initLocalRepo(t *testing.T) string {
	t.Helper()
	return initRepo(t, "")
}

// cloneRepo clones a bare repo into a new temp directory.
func cloneRepo(t *testing.T, remotePath string) string {
	t.Helper()
	dir := t.TempDir()
	clone := filepath.Join(dir, "clone")
	run(t, "git", "clone", remotePath, clone)
	run(t, "git", "-C", clone, "config", "user.email", "test@test.com")
	run(t, "git", "-C", clone, "config", "user.name", "Test")
	return clone
}

// writeFile creates or overwrites a file relative to dir.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// run executes a command and fails the test on error.
func run(t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %s: %v", name, args, string(out), err)
	}
	return string(out)
}

// writeTOML writes a config file and returns its path.
func writeTOML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
