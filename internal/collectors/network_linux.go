package collectors

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ignoredIfaces are never chosen as the primary interface.
var ignoredIfaces = map[string]bool{
	"lo": true,
}

// NetworkCollector reports throughput of the default-route interface.
// Like CPU usage, rates need two ticks; the first Collect seeds state.
type NetworkCollector struct {
	mu      sync.Mutex
	iface   string
	rx, tx  uint64
	at      time.Time
	hasPrev bool
}

func (c *NetworkCollector) Name() string { return "network" }

func (c *NetworkCollector) Collect(_ context.Context) (Snapshot, error) {
	routeData, err := readFile("/proc/net/route")
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect network metrics: %w", err)
	}
	iface, err := defaultIface(routeData)
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect network metrics: %w", err)
	}
	now := time.Now()
	rx, err := readUintFile("/sys/class/net/" + iface + "/statistics/rx_bytes")
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect network metrics: %w", err)
	}
	tx, err := readUintFile("/sys/class/net/" + iface + "/statistics/tx_bytes")
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect network metrics: %w", err)
	}

	c.mu.Lock()
	prevIface, prevRx, prevTx, prevAt, hasPrev := c.iface, c.rx, c.tx, c.at, c.hasPrev
	c.iface, c.rx, c.tx, c.at, c.hasPrev = iface, rx, tx, now, true
	c.mu.Unlock()

	m := NetworkMetrics{Interface: iface}
	if hasPrev && prevIface == iface && now.After(prevAt) {
		dt := now.Sub(prevAt).Seconds()
		m.Down = float64(rx-prevRx) / dt
		m.Up = float64(tx-prevTx) / dt
	}
	return Snapshot{At: now, Available: true, Summary: formatNetworkSummary(m)}, nil
}

// defaultIface returns the interface holding the default route,
// skipping loopback and virtual devices.
func defaultIface(routeData []byte) (string, error) {
	for _, line := range strings.Split(string(routeData), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || f[0] == "Iface" {
			continue
		}
		if f[1] != "00000000" {
			continue
		}
		iface := f[0]
		if ignoredIfaces[iface] ||
			strings.HasPrefix(iface, "docker") ||
			strings.HasPrefix(iface, "br-") ||
			strings.HasPrefix(iface, "veth") ||
			strings.HasPrefix(iface, "virbr") {
			continue
		}
		return iface, nil
	}
	return "", fmt.Errorf("no default route interface found")
}

func readUintFile(path string) (uint64, error) {
	raw, err := readFile(path)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("bad counter %s: %w", path, err)
	}
	return n, nil
}

func formatNetworkSummary(m NetworkMetrics) string {
	return fmt.Sprintf("%s ↓ %s ↑ %s", m.Interface, formatRate(m.Down), formatRate(m.Up))
}

func formatRate(bps float64) string {
	switch {
	case bps >= 1<<30:
		return fmt.Sprintf("%.1f GB/s", bps/(1<<30))
	case bps >= 1<<20:
		return fmt.Sprintf("%.1f MB/s", bps/(1<<20))
	case bps >= 1<<10:
		return fmt.Sprintf("%.0f KB/s", bps/(1<<10))
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}
