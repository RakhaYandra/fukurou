package collectors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// powerSupplyRoot is /sys/class/power_supply in production, overridable.
var powerSupplyRoot = "/sys/class/power_supply"

// BatteryCollector reads the first BAT* supply: charge %, status, AC line.
// Desktops without a battery yield unavailable, never an error.
type BatteryCollector struct{}

func (c *BatteryCollector) Name() string { return "battery" }

func (c *BatteryCollector) Collect(_ context.Context) (Snapshot, error) {
	m, err := readBattery()
	if err != nil {
		return Snapshot{At: time.Now(), Available: false, Summary: "battery unavailable"}, nil
	}
	return Snapshot{
		At: time.Now(), Available: true,
		Summary: formatBatterySummary(m), Usage: float64(m.Capacity), HasUsage: true,
	}, nil
}

type batteryMetrics struct {
	Capacity int
	Status   string // Charging, Discharging, Full, ...
	ACOnline bool
}

func readBattery() (batteryMetrics, error) {
	entries, err := os.ReadDir(powerSupplyRoot)
	if err != nil {
		return batteryMetrics{}, err
	}
	var bat string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "BAT") {
			bat = e.Name()
			break
		}
	}
	if bat == "" {
		return batteryMetrics{}, fmt.Errorf("no battery found")
	}
	capRaw, err := readFile(filepath.Join(powerSupplyRoot, bat, "capacity"))
	if err != nil {
		return batteryMetrics{}, fmt.Errorf("battery capacity: %w", err)
	}
	capacity, err := strconv.Atoi(strings.TrimSpace(string(capRaw)))
	if err != nil {
		return batteryMetrics{}, fmt.Errorf("bad battery capacity: %w", err)
	}
	var m batteryMetrics
	m.Capacity = min(max(capacity, 0), 100)
	if status, err := readFile(filepath.Join(powerSupplyRoot, bat, "status")); err == nil {
		m.Status = strings.TrimSpace(string(status))
	}
	// Any online AC-type supply counts.
	for _, e := range entries {
		online, err := readFile(filepath.Join(powerSupplyRoot, e.Name(), "online"))
		if err == nil && strings.TrimSpace(string(online)) == "1" {
			m.ACOnline = true
			break
		}
	}
	return m, nil
}

func formatBatterySummary(m batteryMetrics) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d%%", m.Capacity)
	switch m.Status {
	case "", "Unknown", "Not charging":
		// Omitted: AC suffix below already tells the story.
	default:
		fmt.Fprintf(&b, " · %s", strings.ToLower(m.Status))
	}
	if m.ACOnline && m.Status != "Discharging" {
		b.WriteString(" · AC")
	}
	return b.String()
}
