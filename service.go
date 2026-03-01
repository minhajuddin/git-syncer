package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func launchdPlistPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "LaunchAgents", "com.git-syncer.plist")
}

func systemdUnitPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "systemd", "user", "git-syncer.service")
}

func logPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/git-syncer.log"
	}
	return filepath.Join(home, ".config", "git-syncer", "git-syncer.log")
}

func launchdPlist(exePath, configPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.git-syncer</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>start</string>
        <string>--config</string>
        <string>%s</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>%s</key>
        <string>1</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
</dict>
</plist>
`, exePath, configPath, daemonEnvVar, logPath(), logPath())
}

func systemdUnit(exePath, configPath string) string {
	return fmt.Sprintf(`[Unit]
Description=git-syncer - keep git repositories in sync

[Service]
Type=simple
Environment=%s=1
ExecStart=%s start --config %s
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
`, daemonEnvVar, exePath, configPath)
}

func resolveExePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("resolving executable symlinks: %w", err)
	}
	return exe, nil
}

func serviceInstall() {
	exePath, err := resolveExePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	configPath := DefaultConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: config file not found at %s\n", configPath)
		fmt.Fprintf(os.Stderr, "Run 'git-syncer init' first to create a config file.\n")
		os.Exit(1)
	}

	switch runtime.GOOS {
	case "darwin":
		installLaunchd(exePath, configPath)
	case "linux":
		installSystemd(exePath, configPath)
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported OS %q. Only macOS and Linux are supported.\n", runtime.GOOS)
		os.Exit(1)
	}
}

func installLaunchd(exePath, configPath string) {
	plistPath := launchdPlistPath()

	fmt.Println("Detected OS: macOS")
	fmt.Printf("Binary path: %s\n", exePath)
	fmt.Printf("Config path: %s\n", configPath)

	if _, err := os.Stat(plistPath); err == nil {
		fmt.Fprintf(os.Stderr, "Error: service file already exists at %s\n", plistPath)
		fmt.Fprintf(os.Stderr, "Run 'git-syncer service uninstall' first to remove it.\n")
		os.Exit(1)
	}

	fmt.Printf("Writing service file to %s...\n", plistPath)
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(plistPath, []byte(launchdPlist(exePath, configPath)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing plist: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Loading service...")
	cmd := exec.Command("launchctl", "load", plistPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading service: %s: %v\n", string(out), err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Done! git-syncer will now start automatically at login.")
	fmt.Println()
	fmt.Println("To verify, run:")
	fmt.Println("  launchctl list | grep git-syncer")
	fmt.Println()
	fmt.Println("To view logs:")
	fmt.Printf("  tail -f %s\n", logPath())
	fmt.Println()
	fmt.Println("To remove:")
	fmt.Println("  git-syncer service uninstall")
}

func installSystemd(exePath, configPath string) {
	unitPath := systemdUnitPath()

	fmt.Println("Detected OS: Linux")
	fmt.Printf("Binary path: %s\n", exePath)
	fmt.Printf("Config path: %s\n", configPath)

	if _, err := os.Stat(unitPath); err == nil {
		fmt.Fprintf(os.Stderr, "Error: service file already exists at %s\n", unitPath)
		fmt.Fprintf(os.Stderr, "Run 'git-syncer service uninstall' first to remove it.\n")
		os.Exit(1)
	}

	fmt.Printf("Writing service file to %s...\n", unitPath)
	if err := os.MkdirAll(filepath.Dir(unitPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(unitPath, []byte(systemdUnit(exePath, configPath)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing unit file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Reloading systemd daemon...")
	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reloading systemd: %s: %v\n", string(out), err)
		os.Exit(1)
	}

	fmt.Println("Enabling and starting service...")
	if out, err := exec.Command("systemctl", "--user", "enable", "--now", "git-syncer").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling service: %s: %v\n", string(out), err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Done! git-syncer will now start automatically at login.")
	fmt.Println()
	fmt.Println("To verify, run:")
	fmt.Println("  systemctl --user status git-syncer")
	fmt.Println()
	fmt.Println("To view logs:")
	fmt.Println("  journalctl --user -u git-syncer -f")
	fmt.Println()
	fmt.Println("To remove:")
	fmt.Println("  git-syncer service uninstall")
}

func serviceUninstall() {
	switch runtime.GOOS {
	case "darwin":
		uninstallLaunchd()
	case "linux":
		uninstallSystemd()
	default:
		fmt.Fprintf(os.Stderr, "Error: unsupported OS %q.\n", runtime.GOOS)
		os.Exit(1)
	}
}

func uninstallLaunchd() {
	plistPath := launchdPlistPath()

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: no service file found at %s\n", plistPath)
		fmt.Fprintf(os.Stderr, "Nothing to uninstall.\n")
		os.Exit(1)
	}

	fmt.Println("Unloading service...")
	cmd := exec.Command("launchctl", "unload", plistPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: launchctl unload: %s: %v\n", string(out), err)
	}

	fmt.Printf("Removing %s...\n", plistPath)
	if err := os.Remove(plistPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing plist: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Done! git-syncer service has been removed.")
	fmt.Println("The daemon will no longer start at login.")
}

func uninstallSystemd() {
	unitPath := systemdUnitPath()

	if _, err := os.Stat(unitPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: no service file found at %s\n", unitPath)
		fmt.Fprintf(os.Stderr, "Nothing to uninstall.\n")
		os.Exit(1)
	}

	fmt.Println("Disabling and stopping service...")
	if out, err := exec.Command("systemctl", "--user", "disable", "--now", "git-syncer").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: systemctl disable: %s: %v\n", string(out), err)
	}

	fmt.Printf("Removing %s...\n", unitPath)
	if err := os.Remove(unitPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing unit file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Reloading systemd daemon...")
	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: systemctl daemon-reload: %s: %v\n", string(out), err)
	}

	fmt.Println()
	fmt.Println("Done! git-syncer service has been removed.")
	fmt.Println("The daemon will no longer start at login.")
}
