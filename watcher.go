package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type RepoWatcher struct {
	syncer   *RepoSyncer
	watcher  *fsnotify.Watcher
	debounce time.Duration
	stopCh   chan struct{}
}

func NewRepoWatcher(syncer *RepoSyncer, debounce time.Duration) (*RepoWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	rw := &RepoWatcher{
		syncer:   syncer,
		watcher:  watcher,
		debounce: debounce,
		stopCh:   make(chan struct{}),
	}

	// Recursively add all directories (excluding .git)
	err = filepath.Walk(syncer.repo.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		watcher.Close()
		return nil, err
	}

	return rw, nil
}

func (rw *RepoWatcher) Start() {
	go rw.loop()
}

func (rw *RepoWatcher) Stop() {
	close(rw.stopCh)
	rw.watcher.Close()
}

func (rw *RepoWatcher) loop() {
	var timer *time.Timer
	var timerC <-chan time.Time

	for {
		select {
		case <-rw.stopCh:
			if timer != nil {
				timer.Stop()
			}
			return

		case event, ok := <-rw.watcher.Events:
			if !ok {
				return
			}

			// Ignore .git directory events
			if strings.Contains(event.Name, string(filepath.Separator)+".git"+string(filepath.Separator)) ||
				strings.HasSuffix(event.Name, string(filepath.Separator)+".git") {
				continue
			}

			// If a new directory was created, add it to the watcher
			if event.Has(fsnotify.Create) {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					rw.watcher.Add(event.Name)
				}
			}

			// Reset debounce timer
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(rw.debounce)
			timerC = timer.C

			rw.syncer.logf("change detected: %s %s", event.Op, event.Name)

		case err, ok := <-rw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[%s] watcher error: %v", rw.syncer.repo.Path, err)

		case <-timerC:
			// Debounce period elapsed, commit and push
			log.Printf("[%s] debounce elapsed, committing and pushing...", rw.syncer.repo.Path)
			if err := rw.syncer.CommitAndPush(); err != nil {
				log.Printf("[%s] ERROR: commit and push failed: %v", rw.syncer.repo.Path, err)
			}
			timer = nil
			timerC = nil
		}
	}
}
