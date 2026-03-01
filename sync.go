package main

import (
	"log"
	"strings"
	"sync"
)

type RepoSyncer struct {
	repo    RepoConfig
	verbose bool
	mu      sync.Mutex
}

func NewRepoSyncer(repo RepoConfig, verbose bool) *RepoSyncer {
	return &RepoSyncer{
		repo:    repo,
		verbose: verbose,
	}
}

func (s *RepoSyncer) logf(format string, args ...interface{}) {
	if s.verbose {
		log.Printf("[%s] "+format, append([]interface{}{s.repo.Path}, args...)...)
	}
}

// CommitAndPush stages, commits local changes, and pushes to remote.
// Called by the watcher after debounce.
func (s *RepoSyncer) CommitAndPush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	changed, err := HasChanges(s.repo.Path)
	if err != nil {
		return err
	}
	if !changed {
		s.logf("no local changes")
		return nil
	}

	s.logf("local changes detected, staging...")
	if err := GitAdd(s.repo.Path); err != nil {
		return err
	}

	msg := AutoCommitMessage()
	s.logf("committing: %s", msg)
	if err := GitCommit(s.repo.Path, msg); err != nil {
		return err
	}

	s.logf("pushing to %s/%s", s.repo.Remote, s.repo.Branch)
	if err := GitPush(s.repo.Path, s.repo.Remote, s.repo.Branch); err != nil {
		// Push might fail if remote has new commits; try pull+rebase first
		s.logf("push failed, attempting pull --rebase...")
		if pullErr := s.pullRebase(); pullErr != nil {
			return pullErr
		}
		if err := GitPush(s.repo.Path, s.repo.Remote, s.repo.Branch); err != nil {
			return err
		}
	}

	log.Printf("[%s] synced successfully", s.repo.Path)
	return nil
}

// PullFromRemote pulls from remote with rebase.
// Called by the poller on the configured interval.
func (s *RepoSyncer) PullFromRemote() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Auto-commit any local changes before pulling
	changed, err := HasChanges(s.repo.Path)
	if err != nil {
		return err
	}
	if changed {
		s.logf("committing local changes before pull...")
		if err := GitAdd(s.repo.Path); err != nil {
			return err
		}
		if err := GitCommit(s.repo.Path, AutoCommitMessage()); err != nil {
			return err
		}
	}

	return s.pullRebase()
}

func (s *RepoSyncer) pullRebase() error {
	s.logf("pulling from %s/%s --rebase", s.repo.Remote, s.repo.Branch)
	err := GitPull(s.repo.Path, s.repo.Remote, s.repo.Branch)
	if err == nil {
		s.logf("pull successful")
		return nil
	}

	if strings.Contains(err.Error(), "CONFLICT") || strings.Contains(err.Error(), "conflict") {
		log.Printf("[%s] ERROR: rebase conflict detected, aborting rebase", s.repo.Path)
		if abortErr := GitRebaseAbort(s.repo.Path); abortErr != nil {
			log.Printf("[%s] ERROR: failed to abort rebase: %v", s.repo.Path, abortErr)
		}
		return err
	}

	return err
}

// SyncOnce runs a full sync cycle: commit local changes, pull, push.
func (s *RepoSyncer) SyncOnce() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Step 1: Commit local changes if any
	changed, err := HasChanges(s.repo.Path)
	if err != nil {
		return err
	}
	if changed {
		s.logf("local changes detected, staging...")
		if err := GitAdd(s.repo.Path); err != nil {
			return err
		}
		msg := AutoCommitMessage()
		s.logf("committing: %s", msg)
		if err := GitCommit(s.repo.Path, msg); err != nil {
			return err
		}
	}

	// Step 2: Pull with rebase
	if err := s.pullRebase(); err != nil {
		return err
	}

	// Step 3: Push
	s.logf("pushing to %s/%s", s.repo.Remote, s.repo.Branch)
	if err := GitPush(s.repo.Path, s.repo.Remote, s.repo.Branch); err != nil {
		return err
	}

	log.Printf("[%s] synced successfully", s.repo.Path)
	return nil
}
