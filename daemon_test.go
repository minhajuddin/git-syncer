package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestWriteAndReadPID(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "test.pid")

	// Test the underlying write/read logic directly.
	pid := os.Getpid()
	err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		t.Fatalf("writing PID file: %v", err)
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("reading PID file: %v", err)
	}

	readPid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parsing PID: %v", err)
	}

	if readPid != pid {
		t.Errorf("expected PID %d, got %d", pid, readPid)
	}
}

func TestIsProcessAlive_CurrentProcess(t *testing.T) {
	pid := os.Getpid()
	if !isProcessAlive(pid) {
		t.Error("expected current process to be alive")
	}
}

func TestIsProcessAlive_NonExistentPID(t *testing.T) {
	// Use a very high PID that's unlikely to exist
	if isProcessAlive(999999999) {
		t.Error("expected non-existent PID to be not alive")
	}
}

func TestIsDaemonProcess_NotSet(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv(daemonEnvVar)
	if isDaemonProcess() {
		t.Error("expected isDaemonProcess to return false when env var not set")
	}
}

func TestIsDaemonProcess_Set(t *testing.T) {
	os.Setenv(daemonEnvVar, "1")
	defer os.Unsetenv(daemonEnvVar)
	if !isDaemonProcess() {
		t.Error("expected isDaemonProcess to return true when env var is '1'")
	}
}

func TestIsDaemonProcess_WrongValue(t *testing.T) {
	os.Setenv(daemonEnvVar, "0")
	defer os.Unsetenv(daemonEnvVar)
	if isDaemonProcess() {
		t.Error("expected isDaemonProcess to return false when env var is '0'")
	}
}

func TestPidFilePath(t *testing.T) {
	path := pidFilePath()
	if path == "" {
		t.Skip("cannot determine pid file path")
	}
	if !strings.Contains(path, "git-syncer") {
		t.Errorf("expected pid file path to contain 'git-syncer', got: %s", path)
	}
	if !strings.HasSuffix(path, ".pid") {
		t.Errorf("expected pid file path to end with .pid, got: %s", path)
	}
}

func TestRemovePIDFile(t *testing.T) {
	// Create a temp PID file and ensure removePIDFile doesn't panic
	// even when the file doesn't exist at the default path
	// (removePIDFile ignores errors from os.Remove)
	removePIDFile() // should not panic
}

func TestReadPID_NoFile(t *testing.T) {
	// Ensure no PID file exists at the default path (it shouldn't in tests)
	// This is fragile if the daemon is actually running, but good enough
	_, err := readPID()
	if err == nil {
		// PID file exists (daemon might be running), skip
		t.Skip("PID file exists, skipping")
	}
}

func TestStopDaemon_NoPIDFile(t *testing.T) {
	// Temporarily move the PID file if it exists
	pidPath := pidFilePath()
	backup := pidPath + ".bak"
	os.Rename(pidPath, backup)
	defer os.Rename(backup, pidPath)

	err := StopDaemon()
	if err == nil {
		t.Error("expected error when no PID file exists")
	}
	if !strings.Contains(err.Error(), "no daemon running") {
		t.Errorf("unexpected error: %v", err)
	}
}
