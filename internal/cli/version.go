package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Build information, set at compile time via ldflags
var (
	Version   = "dev"
	Commit    = "none"
	Date      = "unknown"
	GoVersion = runtime.Version()
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information about the meshtastic-relay binary.`,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("meshtastic-relay %s\n", Version)
		fmt.Printf("  Commit:     %s\n", Commit)
		fmt.Printf("  Built:      %s\n", Date)
		fmt.Printf("  Go version: %s\n", GoVersion)
		fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// SetVersionInfo sets the version information from build flags
func SetVersionInfo(version, commit, date string) {
	if version != "" {
		Version = version
	}
	if commit != "" {
		Commit = commit
	}
	if date != "" {
		Date = date
	}
}
