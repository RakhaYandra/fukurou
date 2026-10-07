// Package collectors defines the metric models and Collector interface.
// Real /proc /sys implementations land in Phase 1; Phase 0 is types + registry.
package collectors

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Collector gathers one metric domain. Implementations must be fast,
// non-blocking, and never require root.
type Collector interface {
	// Name is the module key, e.g. "cpu".
	Name() string
	// Collect returns the current snapshot or an error.
	// Errors must be self-contained; a failing collector never crashes the app.
	Collect(ctx context.Context) (Snapshot, error)
}

// Snapshot is one collector's reading at a point in time.
type Snapshot struct {
	Name      string
	At        time.Time
	Available bool
	Summary   string
	// Usage is 0-100 for bar rendering; only meaningful if HasUsage.
	Usage    float64
	HasUsage bool
	Err      error
}

// SystemMetrics aggregates all domains for one refresh tick.
type SystemMetrics struct {
	System  SystemInfo
	CPU     CPUMetrics
	Memory  MemoryMetrics
	GPU     *GPUMetrics
	Storage StorageMetrics
	Network NetworkMetrics
}

type SystemInfo struct {
	OS, Kernel, Hostname, Uptime string
	DE, Compositor, Resolution   string
}

type CPUMetrics struct {
	Model       string
	Usage       float64 // 0-100
	Temperature *float64
	Frequency   float64 // GHz
}

type MemoryMetrics struct {
	Total, Used, Available uint64 // bytes
	Usage                  float64
}

type GPUMetrics struct {
	Vendor, Model       string
	Usage               float64
	Temperature         *float64
	VRAMUsed, VRAMTotal uint64
}

type StorageMetrics struct {
	Path              string
	Total, Used, Free uint64
	Usage             float64
}

type NetworkMetrics struct {
	Interface string
	Down, Up  float64 // bytes/sec
}

// Registry holds collectors keyed by module name.
type Registry struct {
	collectors map[string]Collector
}

// NewRegistry builds an empty registry.
func NewRegistry() *Registry {
	return &Registry{collectors: map[string]Collector{}}
}

// Register adds a collector; duplicate names are rejected.
func (r *Registry) Register(c Collector) error {
	if c == nil {
		return fmt.Errorf("nil collector")
	}
	if _, exists := r.collectors[c.Name()]; exists {
		return fmt.Errorf("collector %q already registered", c.Name())
	}
	r.collectors[c.Name()] = c
	return nil
}

// CollectAll runs every registered collector concurrently, each with its
// own timeout. A failing collector yields an unavailable Snapshot;
// it never aborts the others.
func (r *Registry) CollectAll(ctx context.Context, timeout time.Duration) []Snapshot {
	names := r.Names()
	out := make([]Snapshot, len(names))
	var wg sync.WaitGroup
	for i, n := range names {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			snap, err := r.collectors[n].Collect(cctx)
			if err != nil {
				snap = Snapshot{Name: n, At: time.Now(), Err: err}
			}
			snap.Name = n
			if snap.At.IsZero() {
				snap.At = time.Now()
			}
			out[i] = snap
		}()
	}
	wg.Wait()
	return out
}

// Names returns sorted collector names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.collectors))
	for n := range r.collectors {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
