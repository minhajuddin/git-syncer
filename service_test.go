package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLaunchdPlistPath(t *testing.T) {
	path := launchdPlistPath()
	if path == "" {
		t.Skip("cannot determine home directory")
	}
	if !strings.HasSuffix(path, filepath.Join("LaunchAgents", "com.git-syncer.plist")) {
		t.Errorf("unexpected plist path: %s", path)
	}
}

func TestSystemdUnitPath(t *testing.T) {
	path := systemdUnitPath()
	if path == "" {
		t.Skip("cannot determine home directory")
	}
	if !strings.HasSuffix(path, filepath.Join("systemd", "user", "git-syncer.service")) {
		t.Errorf("unexpected unit path: %s", path)
	}
}

func TestLogPath(t *testing.T) {
	path := logPath()
	if !strings.HasSuffix(path, "git-syncer.log") {
		t.Errorf("unexpected log path: %s", path)
	}
}

func TestLaunchdPlist_Content(t *testing.T) {
	plist := launchdPlist("/usr/local/bin/git-syncer", "/home/user/.config/git-syncer/config.toml")

	checks := []string{
		"com.git-syncer",
		"/usr/local/bin/git-syncer",
		"/home/user/.config/git-syncer/config.toml",
		"RunAtLoad",
		"<true/>",
		daemonEnvVar,
		"git-syncer.log",
	}
	for _, s := range checks {
		if !strings.Contains(plist, s) {
			t.Errorf("plist missing %q", s)
		}
	}

	// Should be valid XML-ish
	if !strings.HasPrefix(plist, "<?xml") {
		t.Error("plist should start with XML declaration")
	}
}

func TestSystemdUnit_Content(t *testing.T) {
	unit := systemdUnit("/usr/local/bin/git-syncer", "/home/user/.config/git-syncer/config.toml")

	checks := []string{
		"[Unit]",
		"[Service]",
		"[Install]",
		"ExecStart=/usr/local/bin/git-syncer start --config /home/user/.config/git-syncer/config.toml",
		"WantedBy=default.target",
		"Restart=on-failure",
		daemonEnvVar,
	}
	for _, s := range checks {
		if !strings.Contains(unit, s) {
			t.Errorf("unit missing %q", s)
		}
	}
}

func TestInitConfig_RequiredBeforeServiceInstall(t *testing.T) {
	// Verify that InitConfig creates a file that serviceInstall would accept
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	if err := InitConfig(configPath); err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file should exist after init")
	}
}

func TestResolveExePath(t *testing.T) {
	path, err := resolveExePath()
	if err != nil {
		t.Fatalf("resolveExePath failed: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty executable path")
	}
	// Should be an absolute path
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got: %s", path)
	}
}
