package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_Valid(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	toml := writeTOML(t, `
[defaults]
debounce_seconds = 30
poll_interval_seconds = 120

[[repos]]
path = "`+repo+`"
remote = "origin"
branch = "main"
`)

	cfg, err := LoadConfig(toml)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Path != repo {
		t.Errorf("expected path %q, got %q", repo, cfg.Repos[0].Path)
	}
	if cfg.Repos[0].Remote != "origin" {
		t.Errorf("expected remote 'origin', got %q", cfg.Repos[0].Remote)
	}
	if cfg.Repos[0].Branch != "main" {
		t.Errorf("expected branch 'main', got %q", cfg.Repos[0].Branch)
	}
	if cfg.Repos[0].DebounceSeconds != 30 {
		t.Errorf("expected debounce 30, got %d", cfg.Repos[0].DebounceSeconds)
	}
	if cfg.Repos[0].PollIntervalSeconds != 120 {
		t.Errorf("expected poll interval 120, got %d", cfg.Repos[0].PollIntervalSeconds)
	}
}

func TestLoadConfig_DefaultValues(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	toml := writeTOML(t, `
[[repos]]
path = "`+repo+`"
`)

	cfg, err := LoadConfig(toml)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Repos[0].Remote != "origin" {
		t.Errorf("expected default remote 'origin', got %q", cfg.Repos[0].Remote)
	}
	if cfg.Repos[0].DebounceSeconds != 60 {
		t.Errorf("expected default debounce 60, got %d", cfg.Repos[0].DebounceSeconds)
	}
	if cfg.Repos[0].PollIntervalSeconds != 300 {
		t.Errorf("expected default poll interval 300, got %d", cfg.Repos[0].PollIntervalSeconds)
	}
}

func TestLoadConfig_PerRepoOverride(t *testing.T) {
	repo1 := initLocalRepo(t)
	repo2 := initLocalRepo(t)

	toml := writeTOML(t, `
[defaults]
debounce_seconds = 60
poll_interval_seconds = 300

[[repos]]
path = "`+repo1+`"

[[repos]]
path = "`+repo2+`"
debounce_seconds = 10
poll_interval_seconds = 60
`)

	cfg, err := LoadConfig(toml)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Repos[0].DebounceSeconds != 60 {
		t.Errorf("repo1: expected debounce 60 (from defaults), got %d", cfg.Repos[0].DebounceSeconds)
	}
	if cfg.Repos[1].DebounceSeconds != 10 {
		t.Errorf("repo2: expected debounce 10 (override), got %d", cfg.Repos[1].DebounceSeconds)
	}
	if cfg.Repos[1].PollIntervalSeconds != 60 {
		t.Errorf("repo2: expected poll 60 (override), got %d", cfg.Repos[1].PollIntervalSeconds)
	}
}

func TestLoadConfig_TildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home dir")
	}

	// We need a path that actually exists under ~
	// Use a temp dir approach: create a dir, then reference it via ~/relative
	dir := t.TempDir()
	// We can't easily test ~ expansion with temp dirs, so test the logic directly
	toml := writeTOML(t, `
[[repos]]
path = "~/some-repo"
`)

	cfg, err := LoadConfig(toml)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	_ = dir
	expected := filepath.Join(home, "some-repo")
	if cfg.Repos[0].Path != expected {
		t.Errorf("expected expanded path %q, got %q", expected, cfg.Repos[0].Path)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "reading config file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadConfig_InvalidTOML(t *testing.T) {
	toml := writeTOML(t, `this is not valid toml [[[`)
	_, err := LoadConfig(toml)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
	if !strings.Contains(err.Error(), "parsing config file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadConfig_NoRepos(t *testing.T) {
	toml := writeTOML(t, `
[defaults]
debounce_seconds = 30
`)
	_, err := LoadConfig(toml)
	if err == nil {
		t.Fatal("expected error for no repos")
	}
	if !strings.Contains(err.Error(), "no repos configured") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadConfig_MissingRepoPath(t *testing.T) {
	toml := writeTOML(t, `
[[repos]]
remote = "origin"
`)
	_, err := LoadConfig(toml)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadConfig_MultipleRepos(t *testing.T) {
	repo1 := initLocalRepo(t)
	repo2 := initLocalRepo(t)

	toml := writeTOML(t, `
[[repos]]
path = "`+repo1+`"
branch = "main"

[[repos]]
path = "`+repo2+`"
branch = "main"
remote = "origin"
`)

	cfg, err := LoadConfig(toml)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(cfg.Repos))
	}
}

func TestValidateConfig_ValidRepo(t *testing.T) {
	bare := initBareRepo(t)
	repo := initRepo(t, bare)

	cfg := &Config{
		Repos: []RepoConfig{
			{Path: repo, Remote: "origin", Branch: "main"},
		},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("ValidateConfig failed: %v", err)
	}
}

func TestValidateConfig_NonExistentPath(t *testing.T) {
	cfg := &Config{
		Repos: []RepoConfig{
			{Path: "/nonexistent/path", Remote: "origin", Branch: "main"},
		},
	}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
	if !strings.Contains(err.Error(), "path does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateConfig_NotADirectory(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "file.txt")
	os.WriteFile(file, []byte("hello"), 0644)

	cfg := &Config{
		Repos: []RepoConfig{
			{Path: file, Remote: "origin", Branch: "main"},
		},
	}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for non-directory")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateConfig_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{
		Repos: []RepoConfig{
			{Path: dir, Remote: "origin", Branch: "main"},
		},
	}
	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for non-git dir")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInitConfig_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.toml")

	if err := InitConfig(path); err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading created config: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "[defaults]") {
		t.Error("config missing [defaults] section")
	}
	if !strings.Contains(content, "[[repos]]") {
		t.Error("config missing [[repos]] section")
	}
	if !strings.Contains(content, "debounce_seconds") {
		t.Error("config missing debounce_seconds")
	}
	if !strings.Contains(content, "poll_interval_seconds") {
		t.Error("config missing poll_interval_seconds")
	}
	// Should contain comments
	if !strings.Contains(content, "#") {
		t.Error("config missing comments")
	}
}

func TestInitConfig_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	os.WriteFile(path, []byte("existing"), 0644)

	err := InitConfig(path)
	if err == nil {
		t.Fatal("expected error when config already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInitConfig_ValidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := InitConfig(path); err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	// The generated config should be parseable
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("generated config is not valid: %v", err)
	}
	if len(cfg.Repos) != 1 {
		t.Errorf("expected 1 example repo, got %d", len(cfg.Repos))
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Skip("cannot determine home directory")
	}
	if !strings.Contains(path, ".config") || !strings.Contains(path, "git-syncer") {
		t.Errorf("unexpected default config path: %s", path)
	}
	if !strings.HasSuffix(path, "config.toml") {
		t.Errorf("expected path to end with config.toml, got: %s", path)
	}
}
