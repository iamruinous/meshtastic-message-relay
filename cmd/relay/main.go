package main

import (
	"fmt"
	"os"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// TODO: Implement CLI with cobra
	// TODO: Load configuration
	// TODO: Initialize connection
	// TODO: Initialize outputs
	// TODO: Start relay service
	fmt.Printf("meshtastic-relay %s (%s) built on %s\n", version, commit, date)
	fmt.Println("Not yet implemented - see README.md for roadmap")
	os.Exit(0)
}
