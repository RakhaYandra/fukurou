package collectors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// hwmonRoot is /sys/class/hwmon in production, overridable in tests.
var hwmonRoot = "/sys/class/hwmon"

// CPUCollector reads model, usage, temperature, and frequency.
// Usage and rates need two ticks; the first Collect seeds state and
// reports 0 usage. Callers refresh periodically (the --debug command
// does two passes).
type CPUCollector struct {
	mu      sync.Mutex
	prev    cpuSample
	hasPrev bool
}

type cpuSample struct{ idle, total uint64 }

func (c *CPUCollector) Name() string { return "cpu" }

func (c *CPUCollector) Collect(_ context.Context) (Snapshot, error) {
	statData, err := readFile("/proc/stat")
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect CPU metrics: %w", err)
	}
	cur, err := parseCPUStat(statData)
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect CPU metrics: %w", err)
	}

	c.mu.Lock()
	prev, hasPrev := c.prev, c.hasPrev
	c.prev, c.hasPrev = cur, true
	c.mu.Unlock()

	var usage float64
	if hasPrev {
		usage = cpuUsagePercent(prev, cur)
	}

	var m CPUMetrics
	m.Usage = usage
	if data, err := readFile("/proc/cpuinfo"); err == nil {
		m.Model = parseCPUModel(data)
		m.Frequency = parseCPUFreqGHz(data)
	}
	if t, err := cpuTemp(); err == nil {
		m.Temperature = &t
	}

	return Snapshot{At: time.Now(), Available: true, Summary: formatCPUSummary(m)}, nil
}

// parseCPUStat parses the aggregate "cpu ..." line of /proc/stat.
func parseCPUStat(data []byte) (cpuSample, error) {
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		f := strings.Fields(line)[1:]
		if len(f) < 4 {
			return cpuSample{}, fmt.Errorf("short cpu line: %q", line)
		}
		nums := make([]uint64, len(f))
		for i, s := range f {
			n, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return cpuSample{}, fmt.Errorf("bad cpu counter %q: %w", s, err)
			}
			nums[i] = n
		}
		var total uint64
		for _, n := range nums {
			total += n
		}
		return cpuSample{idle: nums[3] + nums[4], total: total}, nil
	}
	return cpuSample{}, fmt.Errorf("no aggregate cpu line found")
}

// cpuUsagePercent returns 0-100 from two samples, clamped and NaN-safe.
func cpuUsagePercent(prev, cur cpuSample) float64 {
	if cur.total <= prev.total {
		return 0
	}
	dIdle := float64(cur.idle - prev.idle)
	dTotal := float64(cur.total - prev.total)
	u := 100 * (1 - dIdle/dTotal)
	if u < 0 {
		return 0
	}
	if u > 100 {
		return 100
	}
	return u
}

// parseCPUModel returns the first "model name" from /proc/cpuinfo.
func parseCPUModel(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		if name, ok := strings.CutPrefix(line, "model name"); ok {
			if _, v, ok := strings.Cut(name, ":"); ok {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

// parseCPUFreqGHz returns GHz from scaling_cur_freq or cpu MHz.
func parseCPUFreqGHz(data []byte) float64 {
	if raw, err := readFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_cur_freq"); err == nil {
		if khz, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64); err == nil && khz > 0 {
			return khz / 1e6
		}
	}
	for _, line := range strings.Split(string(data), "\n") {
		if mhz, ok := strings.CutPrefix(line, "cpu MHz"); ok {
			if _, v, found := strings.Cut(mhz, ":"); found {
				if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && f > 0 {
					return f / 1e3
				}
			}
		}
	}
	return 0
}

// cpuTemp scans hwmon for k10temp/coretemp, then acpitz, then anything
// with temp1_input. Values are millidegrees Celsius.
func cpuTemp() (float64, error) {
	entries, err := os.ReadDir(hwmonRoot)
	if err != nil {
		return 0, err
	}
	type candidate struct{ dir, name string }
	var fallback *candidate
	for _, e := range entries {
		nameRaw, err := readFile(filepath.Join(hwmonRoot, e.Name(), "name"))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(nameRaw))
		c := candidate{e.Name(), name}
		if name == "k10temp" || name == "coretemp" || name == "zenpower" {
			return readHwmonTemp(c.dir)
		}
		if fallback == nil && (name == "acpitz" || hasHwmonTemp(c.dir)) {
			fallback = &c
		}
	}
	if fallback != nil {
		return readHwmonTemp(fallback.dir)
	}
	return 0, fmt.Errorf("no temperature sensor found")
}

func hasHwmonTemp(dir string) bool {
	_, err := readFile(filepath.Join(hwmonRoot, dir, "temp1_input"))
	return err == nil
}

func readHwmonTemp(dir string) (float64, error) {
	raw, err := readFile(filepath.Join(hwmonRoot, dir, "temp1_input"))
	if err != nil {
		return 0, err
	}
	milli, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
	if err != nil {
		return 0, fmt.Errorf("bad temp value: %w", err)
	}
	return milli / 1000, nil
}

func formatCPUSummary(m CPUMetrics) string {
	var b strings.Builder
	b.WriteString(m.Model)
	fmt.Fprintf(&b, "\n%.1f%%", m.Usage)
	if m.Temperature != nil {
		fmt.Fprintf(&b, " · %.1f°C", *m.Temperature)
	}
	if m.Frequency > 0 {
		fmt.Fprintf(&b, " · %.2f GHz", m.Frequency)
	}
	return b.String()
}
