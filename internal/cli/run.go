package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/iamruinous/meshtastic-message-relay/internal/config"
	"github.com/iamruinous/meshtastic-message-relay/internal/logging"
	"github.com/iamruinous/meshtastic-message-relay/internal/relay"
	"github.com/iamruinous/meshtastic-message-relay/internal/tui"
)

var (
	dryRun      bool
	interactive bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the message relay service",
	Long: `Start the Meshtastic message relay service.

The service will connect to a Meshtastic node using the configured
connection method and forward received messages to the configured
output destinations.

Use --interactive or -i to run with an interactive TUI.`,
	RunE: runRelay,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate configuration without starting the service")
	runCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "run with interactive TUI")
}

func runRelay(_ *cobra.Command, _ []string) error {
	// Initialize logging
	logCfg := logging.Config{
		Level:  viper.GetString("logging.level"),
		Format: viper.GetString("logging.format"),
	}

	// For interactive mode, use text format and reduce log noise
	if interactive {
		logCfg.Format = "text"
		logCfg.Level = "error"
	}

	if err := logging.Initialize(logCfg); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}
	defer logging.Sync()

	// Log startup info
	if cfgFile := viper.ConfigFileUsed(); cfgFile != "" {
		logging.Info("Using config file", zap.String("path", cfgFile))
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if dryRun {
		fmt.Println("Configuration is valid!")
		fmt.Printf("  Connection: %s\n", cfg.Connection.Type)
		enabledOutputs := 0
		for _, out := range cfg.Outputs {
			if out.Enabled {
				enabledOutputs++
			}
		}
		fmt.Printf("  Outputs: %d enabled\n", enabledOutputs)
		fmt.Printf("  Filters: %d message types, %d nodes, %d channels\n",
			len(cfg.Filters.MessageTypes),
			len(cfg.Filters.NodeIDs),
			len(cfg.Filters.Channels))
		return nil
	}

	// Create relay service
	service, err := relay.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create relay service: %w", err)
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the service
	if err := service.Start(ctx); err != nil {
		return fmt.Errorf("failed to start relay service: %w", err)
	}

	if interactive {
		// Run TUI
		go func() {
			<-sigChan
			cancel()
		}()

		if err := tui.Run(service); err != nil {
			logging.Error("TUI error", zap.Error(err))
		}
	} else {
		// Wait for signal
		logging.Info("Relay service is running. Press Ctrl+C to stop.")
		<-sigChan
		logging.Info("Received shutdown signal")
	}

	// Stop the service
	if err := service.Stop(); err != nil {
		logging.Error("Error stopping service", zap.Error(err))
	}

	return nil
}
