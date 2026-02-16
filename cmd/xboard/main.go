package main

import (
	"fmt"
	"os"

	"github.com/creamcroissant/xboard/internal/config"
	"github.com/spf13/cobra"
)

// Build info - injected via ldflags
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "xboard",
	Short: "XBoard Panel and Node",
	Long:  `XBoard is a panel and node server for managing proxies.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize config
		_, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}