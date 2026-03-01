package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Defaults struct {
	DebounceSeconds      int `toml:"debounce_seconds"`
	PollIntervalSeconds  int `toml:"poll_interval_seconds"`
}

type RepoConfig struct {
	Path                 string `toml:"path"`
	Remote               string `toml:"remote"`
	Branch               string `toml:"branch"`
	DebounceSeconds      int    `toml:"debounce_seconds"`
	PollIntervalSeconds  int    `toml:"poll_interval_seconds"`
}

type Config struct {
	Defaults Defaults     `toml:"defaults"`
	Repos    []RepoConfig `toml:"repos"`
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "git-syncer", "config.toml")
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{
		Defaults: Defaults{
			DebounceSeconds:     60,
			PollIntervalSeconds: 300,
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if len(cfg.Repos) == 0 {
		return nil, fmt.Errorf("no repos configured in %s", path)
	}

	for i := range cfg.Repos {
		repo := &cfg.Repos[i]

		if repo.Path == "" {
			return nil, fmt.Errorf("repo #%d: path is required", i+1)
		}

		// Expand ~ in path
		if len(repo.Path) > 0 && repo.Path[0] == '~' {
			home, err := os.UserHomeDir()
			if err == nil {
				repo.Path = filepath.Join(home, repo.Path[1:])
			}
		}

		if repo.Remote == "" {
			repo.Remote = "origin"
		}

		// Apply defaults for unset per-repo values
		if repo.DebounceSeconds == 0 {
			repo.DebounceSeconds = cfg.Defaults.DebounceSeconds
		}
		if repo.PollIntervalSeconds == 0 {
			repo.PollIntervalSeconds = cfg.Defaults.PollIntervalSeconds
		}
	}

	return cfg, nil
}

func ValidateConfig(cfg *Config) error {
	for i, repo := range cfg.Repos {
		info, err := os.Stat(repo.Path)
		if err != nil {
			return fmt.Errorf("repo #%d (%s): path does not exist: %w", i+1, repo.Path, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("repo #%d (%s): path is not a directory", i+1, repo.Path)
		}
		if !IsGitRepo(repo.Path) {
			return fmt.Errorf("repo #%d (%s): not a git repository", i+1, repo.Path)
		}
	}
	return nil
}
