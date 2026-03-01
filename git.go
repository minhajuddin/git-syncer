package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func gitCmd(repoPath string, args ...string) *exec.Cmd {
	fullArgs := append([]string{"-C", repoPath}, args...)
	return exec.Command("git", fullArgs...)
}

func gitRun(repoPath string, args ...string) (string, error) {
	cmd := gitCmd(repoPath, args...)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return output, fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), output, err)
	}
	return output, nil
}

func IsGitRepo(path string) bool {
	_, err := gitRun(path, "rev-parse", "--git-dir")
	return err == nil
}

func HasChanges(repoPath string) (bool, error) {
	out, err := gitRun(repoPath, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

func GitAdd(repoPath string) error {
	_, err := gitRun(repoPath, "add", "-A")
	return err
}

func GitCommit(repoPath, message string) error {
	_, err := gitRun(repoPath, "commit", "-m", message)
	return err
}

func GitPull(repoPath, remote, branch string) error {
	_, err := gitRun(repoPath, "pull", "--rebase", remote, branch)
	return err
}

func GitPush(repoPath, remote, branch string) error {
	_, err := gitRun(repoPath, "push", remote, branch)
	return err
}

func CurrentBranch(repoPath string) (string, error) {
	return gitRun(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
}

func GitRebaseAbort(repoPath string) error {
	_, err := gitRun(repoPath, "rebase", "--abort")
	return err
}

func AutoCommitMessage() string {
	return fmt.Sprintf("auto-sync: %s", time.Now().Format("2006-01-02 15:04:05"))
}
