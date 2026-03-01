package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsGitRepo_True(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)
	if !IsGitRepo(repo) {
		t.Error("expected IsGitRepo to return true for a git repo")
	}
}

func TestIsGitRepo_False(t *testing.T) {
	dir := t.TempDir()
	if IsGitRepo(dir) {
		t.Error("expected IsGitRepo to return false for a non-git directory")
	}
}

func TestIsGitRepo_NonExistent(t *testing.T) {
	if IsGitRepo("/nonexistent/path") {
		t.Error("expected IsGitRepo to return false for a non-existent path")
	}
}

func TestHasChanges_NoChanges(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	changed, err := HasChanges(repo)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if changed {
		t.Error("expected no changes in clean repo")
	}
}

func TestHasChanges_NewFile(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	writeFile(t, repo, "new.txt", "hello")

	changed, err := HasChanges(repo)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if !changed {
		t.Error("expected changes after creating a new file")
	}
}

func TestHasChanges_ModifiedFile(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	writeFile(t, repo, "README.md", "modified content")

	changed, err := HasChanges(repo)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if !changed {
		t.Error("expected changes after modifying a file")
	}
}

func TestHasChanges_DeletedFile(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	os.Remove(filepath.Join(repo, "README.md"))

	changed, err := HasChanges(repo)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if !changed {
		t.Error("expected changes after deleting a file")
	}
}

func TestGitAdd(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	writeFile(t, repo, "new.txt", "hello")
	if err := GitAdd(repo); err != nil {
		t.Fatalf("GitAdd failed: %v", err)
	}

	// Verify file is staged
	out := run(t, "git", "-C", repo, "diff", "--cached", "--name-only")
	if !strings.Contains(out, "new.txt") {
		t.Errorf("expected new.txt to be staged, got: %s", out)
	}
}

func TestGitAdd_Subdirectory(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	writeFile(t, repo, "sub/dir/file.txt", "nested content")
	if err := GitAdd(repo); err != nil {
		t.Fatalf("GitAdd failed: %v", err)
	}

	out := run(t, "git", "-C", repo, "diff", "--cached", "--name-only")
	if !strings.Contains(out, "sub/dir/file.txt") {
		t.Errorf("expected sub/dir/file.txt to be staged, got: %s", out)
	}
}

func TestGitCommit(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	writeFile(t, repo, "new.txt", "hello")
	run(t, "git", "-C", repo, "add", "-A")

	if err := GitCommit(repo, "test commit"); err != nil {
		t.Fatalf("GitCommit failed: %v", err)
	}

	// Verify commit was created
	out := run(t, "git", "-C", repo, "log", "--oneline", "-1")
	if !strings.Contains(out, "test commit") {
		t.Errorf("expected commit message in log, got: %s", out)
	}
}

func TestGitCommit_NothingToCommit(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	err := GitCommit(repo, "empty commit")
	if err == nil {
		t.Error("expected error when committing with nothing staged")
	}
}

func TestCurrentBranch(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	branch, err := CurrentBranch(repo)
	if err != nil {
		t.Fatalf("CurrentBranch failed: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected branch 'main', got %q", branch)
	}
}

func TestGitPullRebase(t *testing.T) {
	bare := initBareRepo(t)
	repo1 := initRepo(t, bare)
	repo2 := cloneRepo(t, bare)

	// Push a change from repo1
	writeFile(t, repo1, "from-repo1.txt", "from repo1")
	run(t, "git", "-C", repo1, "add", "-A")
	run(t, "git", "-C", repo1, "commit", "-m", "from repo1")
	run(t, "git", "-C", repo1, "push", "origin", "main")

	// Pull into repo2
	if err := GitPull(repo2, "origin", "main"); err != nil {
		t.Fatalf("GitPull failed: %v", err)
	}

	// Verify the file exists
	content, err := os.ReadFile(filepath.Join(repo2, "from-repo1.txt"))
	if err != nil {
		t.Fatalf("file not pulled: %v", err)
	}
	if string(content) != "from repo1" {
		t.Errorf("unexpected content: %s", string(content))
	}
}

func TestGitPush(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	writeFile(t, repo, "pushed.txt", "pushed content")
	run(t, "git", "-C", repo, "add", "-A")
	run(t, "git", "-C", repo, "commit", "-m", "push test")

	if err := GitPush(repo, "origin", "main"); err != nil {
		t.Fatalf("GitPush failed: %v", err)
	}

	// Clone and verify
	clone := cloneRepo(t, bare)
	content, err := os.ReadFile(filepath.Join(clone, "pushed.txt"))
	if err != nil {
		t.Fatalf("pushed file not in clone: %v", err)
	}
	if string(content) != "pushed content" {
		t.Errorf("unexpected content: %s", string(content))
	}
}

func TestGitPush_NoRemote(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	writeFile(t, repo, "file.txt", "content")
	run(t, "git", "-C", repo, "add", "-A")
	run(t, "git", "-C", repo, "commit", "-m", "test")

	err := GitPush(repo, "nonexistent", "main")
	if err == nil {
		t.Error("expected error pushing to non-existent remote")
	}
}

func TestAutoCommitMessage(t *testing.T) {
	msg := AutoCommitMessage()
	if !strings.HasPrefix(msg, "auto-sync: ") {
		t.Errorf("expected message to start with 'auto-sync: ', got: %s", msg)
	}

	// Verify it contains a timestamp-like format
	datePart := strings.TrimPrefix(msg, "auto-sync: ")
	_, err := time.Parse("2006-01-02 15:04:05", datePart)
	if err != nil {
		t.Errorf("expected valid timestamp in message, got %q: %v", datePart, err)
	}
}

func TestGitRebaseAbort_NoRebaseInProgress(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	err := GitRebaseAbort(repo)
	if err == nil {
		t.Error("expected error when no rebase in progress")
	}
}
