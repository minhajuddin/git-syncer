package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncOnce_NoChanges(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, true)

	// SyncOnce with no local changes and remote already up to date
	// should succeed (pull is a no-op, push is a no-op)
	if err := syncer.SyncOnce(); err != nil {
		t.Fatalf("SyncOnce failed: %v", err)
	}
}

func TestSyncOnce_WithLocalChanges(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	writeFile(t, repo, "local.txt", "local change")

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, true)

	if err := syncer.SyncOnce(); err != nil {
		t.Fatalf("SyncOnce failed: %v", err)
	}

	// Verify file was committed
	out := run(t, "git", "-C", repo, "log", "--oneline", "-1")
	if !strings.Contains(out, "auto-sync:") {
		t.Errorf("expected auto-sync commit, got: %s", out)
	}

	// Verify pushed to remote (clone and check)
	clone := cloneRepo(t, bare)
	content, err := os.ReadFile(filepath.Join(clone, "local.txt"))
	if err != nil {
		t.Fatalf("file not pushed to remote: %v", err)
	}
	if string(content) != "local change" {
		t.Errorf("unexpected content: %s", string(content))
	}
}

func TestSyncOnce_WithRemoteChanges(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)
	other := cloneRepo(t, bare)

	// Push a change from the other clone
	writeFile(t, other, "remote.txt", "remote change")
	run(t, "git", "-C", other, "add", "-A")
	run(t, "git", "-C", other, "commit", "-m", "remote commit")
	run(t, "git", "-C", other, "push", "origin", "main")

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, true)

	if err := syncer.SyncOnce(); err != nil {
		t.Fatalf("SyncOnce failed: %v", err)
	}

	// Verify remote changes were pulled
	content, err := os.ReadFile(filepath.Join(repo, "remote.txt"))
	if err != nil {
		t.Fatalf("remote file not pulled: %v", err)
	}
	if string(content) != "remote change" {
		t.Errorf("unexpected content: %s", string(content))
	}
}

func TestSyncOnce_LocalAndRemoteChanges(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)
	other := cloneRepo(t, bare)

	// Push a change from the other clone (different file)
	writeFile(t, other, "remote.txt", "remote change")
	run(t, "git", "-C", other, "add", "-A")
	run(t, "git", "-C", other, "commit", "-m", "remote commit")
	run(t, "git", "-C", other, "push", "origin", "main")

	// Make a local change (different file, no conflict)
	writeFile(t, repo, "local.txt", "local change")

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, true)

	if err := syncer.SyncOnce(); err != nil {
		t.Fatalf("SyncOnce failed: %v", err)
	}

	// Both files should exist
	if _, err := os.Stat(filepath.Join(repo, "remote.txt")); err != nil {
		t.Error("remote.txt not present after sync")
	}
	if _, err := os.Stat(filepath.Join(repo, "local.txt")); err != nil {
		t.Error("local.txt not present after sync")
	}
}

func TestCommitAndPush_NoChanges(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, true)

	// Should be a no-op
	if err := syncer.CommitAndPush(); err != nil {
		t.Fatalf("CommitAndPush failed: %v", err)
	}
}

func TestCommitAndPush_WithChanges(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	writeFile(t, repo, "file.txt", "content")

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, true)

	if err := syncer.CommitAndPush(); err != nil {
		t.Fatalf("CommitAndPush failed: %v", err)
	}

	// Verify committed and pushed
	clone := cloneRepo(t, bare)
	content, err := os.ReadFile(filepath.Join(clone, "file.txt"))
	if err != nil {
		t.Fatalf("file not pushed: %v", err)
	}
	if string(content) != "content" {
		t.Errorf("unexpected content: %s", string(content))
	}
}

func TestCommitAndPush_MultipleFiles(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	writeFile(t, repo, "a.txt", "aaa")
	writeFile(t, repo, "b.txt", "bbb")
	writeFile(t, repo, "sub/c.txt", "ccc")

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, true)

	if err := syncer.CommitAndPush(); err != nil {
		t.Fatalf("CommitAndPush failed: %v", err)
	}

	clone := cloneRepo(t, bare)
	for _, f := range []string{"a.txt", "b.txt", "sub/c.txt"} {
		if _, err := os.Stat(filepath.Join(clone, f)); err != nil {
			t.Errorf("file %s not pushed: %v", f, err)
		}
	}
}

func TestPullFromRemote_NoLocalChanges(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)
	other := cloneRepo(t, bare)

	// Push from other
	writeFile(t, other, "remote.txt", "from remote")
	run(t, "git", "-C", other, "add", "-A")
	run(t, "git", "-C", other, "commit", "-m", "remote")
	run(t, "git", "-C", other, "push", "origin", "main")

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, true)

	if err := syncer.PullFromRemote(); err != nil {
		t.Fatalf("PullFromRemote failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(repo, "remote.txt"))
	if err != nil {
		t.Fatalf("remote file not pulled: %v", err)
	}
	if string(content) != "from remote" {
		t.Errorf("unexpected content: %s", string(content))
	}
}

func TestPullFromRemote_WithLocalChanges(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)
	other := cloneRepo(t, bare)

	// Push from other
	writeFile(t, other, "remote.txt", "from remote")
	run(t, "git", "-C", other, "add", "-A")
	run(t, "git", "-C", other, "commit", "-m", "remote")
	run(t, "git", "-C", other, "push", "origin", "main")

	// Local uncommitted change
	writeFile(t, repo, "local.txt", "local uncommitted")

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, true)

	if err := syncer.PullFromRemote(); err != nil {
		t.Fatalf("PullFromRemote failed: %v", err)
	}

	// Local changes should be auto-committed
	changed, err := HasChanges(repo)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if changed {
		t.Error("expected no uncommitted changes after PullFromRemote")
	}

	// Both files should exist
	if _, err := os.Stat(filepath.Join(repo, "remote.txt")); err != nil {
		t.Error("remote.txt not pulled")
	}
	if _, err := os.Stat(filepath.Join(repo, "local.txt")); err != nil {
		t.Error("local.txt lost after pull")
	}
}

func TestPullFromRemote_Conflict(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)
	other := cloneRepo(t, bare)

	// Both modify the same file
	writeFile(t, other, "README.md", "change from other")
	run(t, "git", "-C", other, "add", "-A")
	run(t, "git", "-C", other, "commit", "-m", "other change")
	run(t, "git", "-C", other, "push", "origin", "main")

	writeFile(t, repo, "README.md", "change from repo")

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, true)

	err := syncer.PullFromRemote()
	if err == nil {
		t.Fatal("expected error for conflicting changes")
	}

	// Verify repo is in a clean state (rebase was aborted)
	out := run(t, "git", "-C", repo, "status")
	if strings.Contains(out, "rebase in progress") {
		t.Error("rebase should have been aborted")
	}
}

func TestNewRepoSyncer(t *testing.T) {
	repo := RepoConfig{
		Path:   "/some/path",
		Remote: "origin",
		Branch: "main",
	}
	syncer := NewRepoSyncer(repo, true)
	if syncer.repo.Path != "/some/path" {
		t.Errorf("unexpected path: %s", syncer.repo.Path)
	}
	if !syncer.verbose {
		t.Error("expected verbose to be true")
	}
}

func TestSyncOnce_ConsecutiveSyncs(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, false)

	// First sync: add a file
	writeFile(t, repo, "first.txt", "first")
	if err := syncer.SyncOnce(); err != nil {
		t.Fatalf("first SyncOnce failed: %v", err)
	}

	// Second sync: add another file
	writeFile(t, repo, "second.txt", "second")
	if err := syncer.SyncOnce(); err != nil {
		t.Fatalf("second SyncOnce failed: %v", err)
	}

	// Third sync: no changes
	if err := syncer.SyncOnce(); err != nil {
		t.Fatalf("third SyncOnce failed: %v", err)
	}

	// Verify both files pushed
	clone := cloneRepo(t, bare)
	for _, f := range []string{"first.txt", "second.txt"} {
		if _, err := os.Stat(filepath.Join(clone, f)); err != nil {
			t.Errorf("file %s not found in clone: %v", f, err)
		}
	}
}
