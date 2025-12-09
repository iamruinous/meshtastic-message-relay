// Package cli provides the command-line interface for the relay.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	logLevel  string
	logFormat string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "meshtastic-relay",
	Short: "A flexible message relay for Meshtastic nodes",
	Long: `Meshtastic Message Relay listens for messages from Meshtastic nodes
and forwards them to configurable endpoints like Apprise, webhooks,
files, or stdout.

Supports multiple connection methods (serial, TCP, MQTT) and
multiple output destinations simultaneously.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Persistent flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ~/.config/meshtastic-relay/config.yml)")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format (json, text)")

	// Bind flags to viper
	_ = viper.BindPFlag("logging.level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("logging.format", rootCmd.PersistentFlags().Lookup("log-format"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in common locations (in priority order)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml") // Supports both .yaml and .yml extensions
		viper.AddConfigPath("$HOME/.config/meshtastic-relay")
		viper.AddConfigPath("/etc/meshtastic-relay")
		viper.AddConfigPath(".")
	}

	// Environment variables
	viper.SetEnvPrefix("MESH_RELAY")
	viper.AutomaticEnv()

	// Read config file if it exists (errors are intentionally ignored)
	_ = viper.ReadInConfig()
}

// GetConfigFile returns the config file being used
func GetConfigFile() string {
	return viper.ConfigFileUsed()
}
