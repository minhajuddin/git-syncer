package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const daemonEnvVar = "GIT_SYNCER_DAEMON"

func pidFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/git-syncer.pid"
	}
	return filepath.Join(home, ".config", "git-syncer", "git-syncer.pid")
}

func writePIDFile() error {
	pidPath := pidFilePath()
	dir := filepath.Dir(pidPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating pid directory: %w", err)
	}
	return os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0644)
}

func removePIDFile() {
	os.Remove(pidFilePath())
}

func readPID() (int, error) {
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file: %w", err)
	}
	return pid, nil
}

func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

func isDaemonProcess() bool {
	return os.Getenv(daemonEnvVar) == "1"
}

// StartDaemon forks the current process as a background daemon.
func StartDaemon(configPath string, verbose bool) error {
	// Check if already running
	pid, err := readPID()
	if err == nil && isProcessAlive(pid) {
		return fmt.Errorf("daemon already running (PID %d)", pid)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	args := []string{"start"}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}
	if verbose {
		args = append(args, "--verbose")
	}

	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(), daemonEnvVar+"=1")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	// Detach from parent process group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting daemon: %w", err)
	}

	fmt.Printf("git-syncer daemon started (PID %d)\n", cmd.Process.Pid)
	return nil
}

// StopDaemon sends SIGTERM to the running daemon.
func StopDaemon() error {
	pid, err := readPID()
	if err != nil {
		return fmt.Errorf("no daemon running (no PID file)")
	}

	if !isProcessAlive(pid) {
		removePIDFile()
		return fmt.Errorf("daemon not running (stale PID %d), cleaned up PID file", pid)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to %d: %w", pid, err)
	}

	fmt.Printf("Sent SIGTERM to daemon (PID %d)\n", pid)
	return nil
}

// DaemonStatus prints whether the daemon is running.
func DaemonStatus() {
	pid, err := readPID()
	if err != nil {
		fmt.Println("git-syncer: not running")
		return
	}

	if isProcessAlive(pid) {
		fmt.Printf("git-syncer: running (PID %d)\n", pid)
	} else {
		fmt.Printf("git-syncer: not running (stale PID %d)\n", pid)
		removePIDFile()
	}
}
