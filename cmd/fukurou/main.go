// Command fukurou is a lightweight floating system dashboard for Linux/Wayland.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/RakhaYandra/fukurou/internal/app"
	"github.com/RakhaYandra/fukurou/internal/config"
	"github.com/RakhaYandra/fukurou/internal/ui"
)

var version = "v0.1.0-dev"

func main() {
	var (
		showVersion bool
		debug       bool
		configPath  string
	)
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&debug, "debug", false, "print collected metrics to stdout and exit")
	flag.StringVar(&configPath, "config", "", "path to config file (default ~/.config/fukurou/config.yaml)")
	flag.Parse()

	if showVersion {
		fmt.Println("fukurou " + version)
		return
	}

	cfg, err := config.Load(configPath)
	level := slog.LevelWarn
	if debug {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	if err != nil {
		log.Warn("config error, using defaults", "err", err)
		cfg = config.Default()
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a := app.New(cfg, version)
	if debug {
		if err := a.Run(ctx, true); err != nil {
			log.Error("debug run failed", "err", err)
			os.Exit(1)
		}
		return
	}
	os.Exit(ui.Run(ctx, cfg, version))
}
