package main

import (
	"flag"
	"fmt"
	"os"
)

const usage = `git-syncer - keep git repositories in sync across machines

Usage:
  git-syncer <command> [flags]

Commands:
  init      Create a default config file with examples
  start     Start the daemon (backgrounds itself, writes PID file)
  stop      Stop the running daemon
  status    Show daemon status
  sync      Run one sync cycle for all repos and exit
  service   Manage OS service (install/uninstall)

Flags:
  -c, --config    Config file path (default: ~/.config/git-syncer/config.toml)
  -v, --verbose   Verbose logging
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	command := os.Args[1]

	// Commands that don't need flags
	switch command {
	case "stop":
		cmdStop()
		return
	case "status":
		cmdStatus()
		return
	case "service":
		cmdService()
		return
	case "help", "--help", "-h":
		fmt.Print(usage)
		return
	}

	// Parse flags for commands that need them
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	configPath := fs.String("config", DefaultConfigPath(), "config file path")
	fs.StringVar(configPath, "c", DefaultConfigPath(), "config file path (shorthand)")
	verbose := fs.Bool("verbose", false, "verbose logging")
	fs.BoolVar(verbose, "v", false, "verbose logging (shorthand)")
	fs.Parse(os.Args[2:])

	switch command {
	case "init":
		cmdInit(*configPath)
	case "start":
		cmdStart(*configPath, *verbose)
	case "sync":
		cmdSync(*configPath, *verbose)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		fmt.Print(usage)
		os.Exit(1)
	}
}
