package main

import (
	"github.com/iamruinous/meshtastic-message-relay/internal/cli"
)

// Build information, injected at compile time
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Set version information for the CLI
	cli.SetVersionInfo(version, commit, date)

	// Execute the CLI
	cli.Execute()
}
