package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewRepoWatcher_WatchesRepo(t *testing.T) {
	repo := initLocalRepo(t)

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, false)

	watcher, err := NewRepoWatcher(syncer, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewRepoWatcher failed: %v", err)
	}
	defer watcher.Stop()

	if watcher.watcher == nil {
		t.Fatal("expected fsnotify watcher to be created")
	}
}

func TestNewRepoWatcher_IgnoresGitDir(t *testing.T) {
	repo := initLocalRepo(t)

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, false)

	watcher, err := NewRepoWatcher(syncer, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewRepoWatcher failed: %v", err)
	}
	defer watcher.Stop()

	watchList := watcher.watcher.WatchList()
	for _, path := range watchList {
		if filepath.Base(path) == ".git" || filepath.Dir(path) == filepath.Join(repo, ".git") {
			t.Errorf(".git directory should not be watched: %s", path)
		}
	}
}

func TestNewRepoWatcher_WatchesSubdirectories(t *testing.T) {
	repo := initLocalRepo(t)

	os.MkdirAll(filepath.Join(repo, "sub", "nested"), 0755)

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, false)

	watcher, err := NewRepoWatcher(syncer, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewRepoWatcher failed: %v", err)
	}
	defer watcher.Stop()

	watchList := watcher.watcher.WatchList()
	foundRoot := false
	foundSub := false
	foundNested := false
	for _, path := range watchList {
		if path == repo {
			foundRoot = true
		}
		if path == filepath.Join(repo, "sub") {
			foundSub = true
		}
		if path == filepath.Join(repo, "sub", "nested") {
			foundNested = true
		}
	}
	if !foundRoot {
		t.Error("root directory not in watch list")
	}
	if !foundSub {
		t.Error("sub/ directory not in watch list")
	}
	if !foundNested {
		t.Error("sub/nested/ directory not in watch list")
	}
}

func TestRepoWatcher_DebounceFires(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, false)

	watcher, err := NewRepoWatcher(syncer, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("NewRepoWatcher failed: %v", err)
	}
	watcher.Start()
	defer watcher.Stop()

	// Create a file to trigger the watcher
	writeFile(t, repo, "trigger.txt", "trigger")

	// Wait for debounce (200ms) + git operations (~2s) + buffer
	time.Sleep(4 * time.Second)

	// Verify the file was auto-committed by CommitAndPush
	out := run(t, "git", "-C", repo, "log", "--oneline")
	if !strings.Contains(out, "auto-sync:") {
		t.Error("expected auto-sync commit after debounce fired")
	}
}

func TestRepoWatcher_DebounceResets(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, false)

	watcher, err := NewRepoWatcher(syncer, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("NewRepoWatcher failed: %v", err)
	}
	watcher.Start()
	defer watcher.Stop()

	// Write a file
	writeFile(t, repo, "first.txt", "first")
	time.Sleep(200 * time.Millisecond)

	// Write another file before debounce expires (resets timer)
	writeFile(t, repo, "second.txt", "second")

	// At this point, debounce timer has been reset. Wait for it to fire + git ops
	time.Sleep(4 * time.Second)

	out := run(t, "git", "-C", repo, "log", "--oneline")
	if !strings.Contains(out, "auto-sync:") {
		t.Error("expected auto-sync commit after debounce reset and fired")
	}

	// Both files should be committed in a single commit
	autoSyncCount := strings.Count(out, "auto-sync:")
	if autoSyncCount != 1 {
		t.Errorf("expected exactly 1 auto-sync commit (both files batched), got %d", autoSyncCount)
	}

	// Verify both files were pushed
	clone := cloneRepo(t, bare)
	if _, err := os.Stat(filepath.Join(clone, "first.txt")); err != nil {
		t.Error("first.txt not pushed")
	}
	if _, err := os.Stat(filepath.Join(clone, "second.txt")); err != nil {
		t.Error("second.txt not pushed")
	}
}

func TestRepoWatcher_Stop(t *testing.T) {
	repo := initLocalRepo(t)

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, false)

	watcher, err := NewRepoWatcher(syncer, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewRepoWatcher failed: %v", err)
	}
	watcher.Start()

	// Stop should not panic or hang
	watcher.Stop()

	// Writing after stop should not cause issues
	writeFile(t, repo, "after-stop.txt", "after stop")
	time.Sleep(300 * time.Millisecond)

	// No commit should have been made after stop
	out := run(t, "git", "-C", repo, "log", "--oneline")
	if strings.Contains(out, "auto-sync:") {
		t.Error("no auto-sync commit should happen after watcher is stopped")
	}
}

func TestRepoWatcher_NewSubdirAutoWatched(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	syncer := NewRepoSyncer(RepoConfig{
		Path:   repo,
		Remote: "origin",
		Branch: "main",
	}, false)

	watcher, err := NewRepoWatcher(syncer, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("NewRepoWatcher failed: %v", err)
	}
	watcher.Start()
	defer watcher.Stop()

	// Create a new subdirectory (should be auto-added to watch)
	newDir := filepath.Join(repo, "newdir")
	os.MkdirAll(newDir, 0755)
	time.Sleep(200 * time.Millisecond)

	// Write a file in the new subdirectory
	writeFile(t, repo, "newdir/file.txt", "in new dir")

	// Wait for debounce + git operations
	time.Sleep(4 * time.Second)

	// The file in the new subdirectory should have been detected and committed
	out := run(t, "git", "-C", repo, "log", "--oneline")
	if !strings.Contains(out, "auto-sync:") {
		t.Error("expected auto-sync commit for file in new subdirectory")
	}
}
