package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func cmdStart(configPath string, verbose bool) {
	if !isDaemonProcess() {
		// Fork into background
		if err := StartDaemon(configPath, verbose); err != nil {
			log.Fatalf("Error: %v", err)
		}
		return
	}

	// We are the daemon process
	if err := writePIDFile(); err != nil {
		log.Fatalf("Error writing PID file: %v", err)
	}
	defer removePIDFile()

	cfg, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	if err := ValidateConfig(cfg); err != nil {
		log.Fatalf("Config validation error: %v", err)
	}

	log.Printf("git-syncer daemon starting, managing %d repo(s)", len(cfg.Repos))

	var watchers []*RepoWatcher
	var stopPollers []chan struct{}

	for i := range cfg.Repos {
		repo := cfg.Repos[i]

		// Resolve branch if not set
		if repo.Branch == "" {
			branch, err := CurrentBranch(repo.Path)
			if err != nil {
				log.Fatalf("Error getting current branch for %s: %v", repo.Path, err)
			}
			repo.Branch = branch
			cfg.Repos[i] = repo
		}

		syncer := NewRepoSyncer(repo, verbose)

		// Start filesystem watcher with debounce
		debounce := time.Duration(repo.DebounceSeconds) * time.Second
		watcher, err := NewRepoWatcher(syncer, debounce)
		if err != nil {
			log.Fatalf("Error creating watcher for %s: %v", repo.Path, err)
		}
		watcher.Start()
		watchers = append(watchers, watcher)

		// Start polling goroutine for pulling
		stopPoll := make(chan struct{})
		stopPollers = append(stopPollers, stopPoll)
		go pollLoop(syncer, time.Duration(repo.PollIntervalSeconds)*time.Second, stopPoll)

		log.Printf("  [%s] watching (debounce=%ds, poll=%ds, remote=%s, branch=%s)",
			repo.Path, repo.DebounceSeconds, repo.PollIntervalSeconds, repo.Remote, repo.Branch)
	}

	// Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("Received %s, shutting down...", sig)

	// Stop all watchers and pollers
	for _, w := range watchers {
		w.Stop()
	}
	for _, ch := range stopPollers {
		close(ch)
	}

	log.Println("git-syncer daemon stopped")
}

func pollLoop(syncer *RepoSyncer, interval time.Duration, stop chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if err := syncer.PullFromRemote(); err != nil {
				log.Printf("[%s] ERROR: pull failed: %v", syncer.repo.Path, err)
			}
		}
	}
}

func cmdInit(configPath string) {
	if err := InitConfig(configPath); err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Config file created at: %s\n\n", configPath)
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit the config file to add your repositories")
	fmt.Println("  2. Run 'git-syncer sync --verbose' to test a one-shot sync")
	fmt.Println("  3. Run 'git-syncer start' to start the background daemon")
}

const serviceUsage = `Usage: git-syncer service <action>

Actions:
  install     Install the OS service (launchd on macOS, systemd on Linux)
  uninstall   Remove the OS service
`

func cmdService() {
	if len(os.Args) < 3 {
		fmt.Print(serviceUsage)
		os.Exit(1)
	}
	switch os.Args[2] {
	case "install":
		serviceInstall()
	case "uninstall":
		serviceUninstall()
	default:
		fmt.Fprintf(os.Stderr, "Unknown action: %s\n\n", os.Args[2])
		fmt.Print(serviceUsage)
		os.Exit(1)
	}
}

func cmdStop() {
	if err := StopDaemon(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func cmdStatus() {
	DaemonStatus()
}

func cmdSync(configPath string, verbose bool) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	if err := ValidateConfig(cfg); err != nil {
		log.Fatalf("Config validation error: %v", err)
	}

	for i := range cfg.Repos {
		repo := cfg.Repos[i]

		if repo.Branch == "" {
			branch, err := CurrentBranch(repo.Path)
			if err != nil {
				log.Printf("[%s] ERROR: getting branch: %v", repo.Path, err)
				continue
			}
			repo.Branch = branch
		}

		syncer := NewRepoSyncer(repo, verbose)
		if err := syncer.SyncOnce(); err != nil {
			log.Printf("[%s] ERROR: sync failed: %v", repo.Path, err)
		}
	}
}
