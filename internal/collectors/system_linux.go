package collectors

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// SystemCollector reads OS, kernel, host, uptime, and session info.
// Resolution stays empty in Phase 1; the GTK shell fills it in Phase 3.
type SystemCollector struct{}

func (c *SystemCollector) Name() string { return "system" }

func (c *SystemCollector) Collect(_ context.Context) (Snapshot, error) {
	var info SystemInfo
	if data, err := readFile("/etc/os-release"); err == nil {
		info.OS = parseOSRelease(data)
	}
	var uts syscall.Utsname
	if err := syscall.Uname(&uts); err == nil {
		info.Kernel = utsToString(uts.Release[:])
	}
	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}
	if data, err := readFile("/proc/uptime"); err == nil {
		info.Uptime = parseUptime(data)
	}
	info.DE = os.Getenv("XDG_CURRENT_DESKTOP")
	info.Compositor = detectCompositor()
	return Snapshot{At: time.Now(), Available: true, Summary: formatSystemSummary(info)}, nil
}

func parseOSRelease(data []byte) string {
	var name string
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
			return strings.Trim(v, `"`)
		}
		if v, ok := strings.CutPrefix(line, "NAME="); ok {
			name = strings.Trim(v, `"`)
		}
	}
	return name
}

func utsToString(buf []int8) string {
	var b strings.Builder
	for _, c := range buf {
		if c == 0 {
			break
		}
		b.WriteByte(byte(c))
	}
	return b.String()
}

// parseUptime formats /proc/uptime seconds as "3d 4h", "2h15m", or "5m".
func parseUptime(data []byte) string {
	f := strings.Fields(string(data))
	if len(f) == 0 {
		return ""
	}
	secs, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return ""
	}
	d := time.Duration(secs * float64(time.Second))
	days := d / (24 * time.Hour)
	h := (d % (24 * time.Hour)) / time.Hour
	m := (d % time.Hour) / time.Minute
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, h)
	case h > 0:
		return fmt.Sprintf("%dh%02dm", h, m)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// detectCompositor is best-effort from environment, no exec.
func detectCompositor() string {
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" {
		return "Hyprland"
	}
	if os.Getenv("SWAYSOCK") != "" {
		return "Sway"
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return "Wayland"
	}
	if os.Getenv("DISPLAY") != "" {
		return "X11"
	}
	return ""
}

func formatSystemSummary(i SystemInfo) string {
	parts := []string{i.OS, "kernel " + i.Kernel, "up " + i.Uptime}
	if i.DE != "" {
		parts = append(parts, i.DE)
	}
	if i.Compositor != "" && i.Compositor != i.DE {
		parts = append(parts, i.Compositor)
	}
	return strings.Join(parts, " · ")
}
