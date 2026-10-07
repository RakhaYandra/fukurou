// Package app wires config + collectors. GTK shell lands in Phase 2.
package app

import (
	"context"
	"fmt"

	"github.com/RakhaYandra/fukurou/internal/collectors"
	"github.com/RakhaYandra/fukurou/internal/config"
)

// App is the composition root.
type App struct {
	cfg      config.Config
	version  string
	registry *collectors.Registry
}

// New builds the app. Collectors register here as Phase 1 lands them.
func New(cfg config.Config, version string) *App {
	return &App{cfg: cfg, version: version, registry: collectors.NewRegistry()}
}

// Run starts the app. Debug mode prints status and exits (Phase 1 fills real data).
func (a *App) Run(ctx context.Context, debug bool) error {
	if debug {
		fmt.Printf("fukurou %s\n", a.version)
		fmt.Printf("modules: %v\n", a.registry.Names())
		fmt.Printf("refresh: %v\n", a.cfg.Refresh.Interval.Duration)
		fmt.Println("collectors: not yet implemented (Phase 1)")
		return nil
	}
	_ = ctx
	return fmt.Errorf("GTK shell not yet implemented (Phase 2)")
}
