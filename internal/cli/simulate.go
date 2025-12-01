package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/iamruinous/meshtastic-message-relay/pkg/meshtastic/simulator"
)

var (
	simNodeNum   uint32
	simLongName  string
	simShortName string
	simInterval  time.Duration
	simVerbose   bool
	simSymlink   string
)

var simulateCmd = &cobra.Command{
	Use:   "simulate",
	Short: "Run a simulated Meshtastic device",
	Long: `Run a simulated Meshtastic device for testing.

This creates a virtual serial port that behaves like a real Meshtastic node.
Connect to it using the path printed by this command.

The simulator will:
- Respond to configuration requests
- Send node information for simulated mesh nodes
- Periodically send simulated text messages
- Display received packets (with --verbose)

Example:
  # Start simulator
  meshtastic-relay simulate --verbose

  # In another terminal, connect to the simulated device
  meshtastic-relay run --config config.yaml -c /dev/pts/X
`,
	RunE: runSimulate,
}

func init() {
	rootCmd.AddCommand(simulateCmd)

	simulateCmd.Flags().Uint32Var(&simNodeNum, "node-num", 0x12345678, "simulated node number")
	simulateCmd.Flags().StringVar(&simLongName, "long-name", "Simulated Node", "node long name")
	simulateCmd.Flags().StringVar(&simShortName, "short-name", "SIM1", "node short name (4 chars)")
	simulateCmd.Flags().DurationVar(&simInterval, "interval", 30*time.Second, "message send interval (0 to disable)")
	simulateCmd.Flags().BoolVarP(&simVerbose, "verbose", "v", false, "verbose output")
	simulateCmd.Flags().StringVar(&simSymlink, "symlink", "", "create symlink to PTY at this path")
}

func runSimulate(cmd *cobra.Command, args []string) error {
	config := simulator.DefaultConfig()
	config.NodeNum = simNodeNum
	config.LongName = simLongName
	config.ShortName = simShortName
	config.MessageInterval = simInterval
	config.Verbose = simVerbose

	device := simulator.New(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the simulator
	path, err := device.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start simulator: %w", err)
	}
	defer device.Stop()

	// Create symlink if requested
	if simSymlink != "" {
		if err := os.Symlink(path, simSymlink); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create symlink: %v\n", err)
		} else {
			fmt.Printf("Created symlink: %s -> %s\n", simSymlink, path)
			defer os.Remove(simSymlink)
		}
	}

	fmt.Printf("Simulated Meshtastic device started\n")
	fmt.Printf("  Device path: %s\n", path)
	fmt.Printf("  Node number: !%08x\n", config.NodeNum)
	fmt.Printf("  Long name:   %s\n", config.LongName)
	fmt.Printf("  Short name:  %s\n", config.ShortName)
	fmt.Printf("  Simulated nodes: %d\n", len(config.SimulatedNodes))
	if config.MessageInterval > 0 {
		fmt.Printf("  Message interval: %v\n", config.MessageInterval)
	} else {
		fmt.Printf("  Auto messages: disabled\n")
	}
	fmt.Println()
	fmt.Println("Connect with: meshtastic-relay run --connection.serial.port", path)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Print simulated nodes
	fmt.Println("Simulated mesh nodes:")
	for _, node := range config.SimulatedNodes {
		fmt.Printf("  - !%08x %s (%s)\n", node.NodeNum, node.LongName, node.ShortName)
	}
	fmt.Println()

	// Wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	return nil
}
