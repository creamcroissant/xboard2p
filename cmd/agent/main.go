package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/creamcroissant/xboard/internal/agent/config"
	"github.com/creamcroissant/xboard/internal/agent/service"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"

	configFile string
	version    bool
)

func init() {
	flag.StringVar(&configFile, "config", "config.yml", "Path to configuration file")
	flag.BoolVar(&version, "version", false, "Show version")
	flag.Parse()
}

func main() {
	if version {
		fmt.Printf("XBoard Agent %s\n", Version)
		return
	}

	// Setup Logger
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	// Load Config
	cfg, err := config.Load(configFile)
	if err != nil {
		slog.Error("Failed to load config", "path", configFile, "error", err)
		os.Exit(1)
	}
	if strings.TrimSpace(cfg.Update.CurrentVersion) == "" {
		cfg.Update.CurrentVersion = Version
	}

	// Initialize Agent
	agent, err := service.New(cfg)
	if err != nil {
		slog.Error("Failed to initialize agent", "error", err)
		os.Exit(1)
	}

	// Context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		slog.Info("Received signal, shutting down...", "signal", sig)
		cancel()
	}()

	// Run
	agent.Run(ctx)
}
