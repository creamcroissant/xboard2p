package main

import (
	"fmt"
	"os"

	"github.com/creamcroissant/xboard/internal/config"
	"github.com/spf13/cobra"
)

// Build info - injected via ldflags
var (
	Version    = "dev"
	Commit     = "unknown"
	BuildTime  = "unknown"
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "xboard",
	Short: "XBoard Panel and Node",
	Long:  `XBoard is a panel and node server for managing proxies.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		_, err := config.LoadWithOptions(config.LoadOptions{ConfigPath: configPath})
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to configuration file")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
