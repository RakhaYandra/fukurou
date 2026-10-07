// Package app wires config + collectors. GTK shell lands in Phase 2.
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RakhaYandra/fukurou/internal/collectors"
	"github.com/RakhaYandra/fukurou/internal/config"
)

// collectorTimeout bounds each collector per tick; stragglers show
// as unavailable instead of stalling the dashboard.
const collectorTimeout = 2 * time.Second

// App is the composition root.
type App struct {
	cfg      config.Config
	version  string
	registry *collectors.Registry
}

// New builds the app, registering only enabled collectors.
func New(cfg config.Config, version string) *App {
	a := &App{cfg: cfg, version: version, registry: collectors.NewRegistry()}
	all := map[string]collectors.Collector{
		"system":  &collectors.SystemCollector{},
		"cpu":     &collectors.CPUCollector{},
		"memory":  &collectors.MemoryCollector{},
		"gpu":     &collectors.GPUCollector{},
		"storage": &collectors.StorageCollector{},
		"network": &collectors.NetworkCollector{},
	}
	for name, c := range all {
		if cfg.Enabled(name) {
			_ = a.registry.Register(c) // names unique by construction
		}
	}
	return a
}

// Run starts the app. Debug mode prints two live ticks and exits;
// rates (CPU %, net throughput) need the second tick to be meaningful.
func (a *App) Run(ctx context.Context, debug bool) error {
	if debug {
		return a.debug(ctx)
	}
	_ = ctx
	return fmt.Errorf("GTK shell not yet implemented (Phase 2)")
}

func (a *App) debug(ctx context.Context) error {
	a.registry.CollectAll(ctx, collectorTimeout) // seed stateful collectors
	interval := min(a.cfg.Refresh.Interval.Duration, 2*time.Second)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(interval):
	}
	fmt.Printf("fukurou %s\n\n", a.version)
	for _, snap := range a.registry.CollectAll(ctx, collectorTimeout) {
		status := "●"
		if !snap.Available || snap.Err != nil {
			status = "○"
		}
		body := snap.Summary
		if snap.Err != nil {
			body = "unavailable (" + snap.Err.Error() + ")"
		}
		fmt.Printf("%s %s\n%s\n\n", status, strings.ToUpper(snap.Name), indent(body))
	}
	return nil
}

func indent(s string) string {
	return "  " + strings.ReplaceAll(s, "\n", "\n  ")
}
