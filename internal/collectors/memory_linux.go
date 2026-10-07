package collectors

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// MemoryCollector reads /proc/meminfo. Used = Total - Available.
type MemoryCollector struct{}

func (c *MemoryCollector) Name() string { return "memory" }

func (c *MemoryCollector) Collect(_ context.Context) (Snapshot, error) {
	data, err := readFile("/proc/meminfo")
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect memory metrics: %w", err)
	}
	m, err := parseMeminfo(data)
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect memory metrics: %w", err)
	}
	return Snapshot{At: time.Now(), Available: true, Summary: formatMemorySummary(m), Usage: m.Usage, HasUsage: true}, nil
}

func parseMeminfo(data []byte) (MemoryMetrics, error) {
	fields := map[string]uint64{}
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		key := strings.TrimSuffix(f[0], ":")
		if key != "MemTotal" && key != "MemAvailable" && key != "MemFree" {
			continue
		}
		n, err := strconv.ParseUint(f[1], 10, 64)
		if err != nil {
			return MemoryMetrics{}, fmt.Errorf("bad meminfo value %q: %w", line, err)
		}
		fields[key] = n * 1024 // kB -> bytes
	}
	total, ok := fields["MemTotal"]
	if !ok || total == 0 {
		return MemoryMetrics{}, fmt.Errorf("MemTotal missing or zero")
	}
	avail, ok := fields["MemAvailable"]
	if !ok {
		avail, ok = fields["MemFree"] // ancient kernels
		if !ok {
			return MemoryMetrics{}, fmt.Errorf("MemAvailable and MemFree missing")
		}
	}
	used := total - min(avail, total)
	return MemoryMetrics{
		Total: total, Used: used, Available: avail,
		Usage: 100 * float64(used) / float64(total),
	}, nil
}

func formatMemorySummary(m MemoryMetrics) string {
	return fmt.Sprintf("%.1f/%.1f GiB (%.0f%%)",
		float64(m.Used)/(1<<30), float64(m.Total)/(1<<30), m.Usage)
}
