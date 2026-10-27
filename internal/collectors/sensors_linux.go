package collectors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// cpuHwmonChips already surface through the CPU card; skip them here.
var cpuHwmonChips = map[string]bool{
	"k10temp": true, "coretemp": true, "zenpower": true, "acpitz": true,
}

// SensorsCollector reports hwmon temps the CPU and GPU cards already skip.
type SensorsCollector struct{}

func (c *SensorsCollector) Name() string { return "sensors" }

func (c *SensorsCollector) Collect(_ context.Context) (Snapshot, error) {
	readings, err := readHwmonTemps()
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect sensors: %w", err)
	}
	if len(readings) == 0 {
		return Snapshot{}, fmt.Errorf("collect sensors: no hwmon temperatures")
	}
	var b strings.Builder
	for i, r := range readings {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%s %.1f°C", r.label, r.celsius)
	}
	return Snapshot{At: time.Now(), Available: true, Summary: b.String()}, nil
}

type sensorReading struct {
	label   string
	celsius float64
}

func readHwmonTemps() ([]sensorReading, error) {
	entries, err := os.ReadDir(hwmonRoot)
	if err != nil {
		return nil, err
	}
	var out []sensorReading
	for _, e := range entries {
		nameRaw, err := readFile(filepath.Join(hwmonRoot, e.Name(), "name"))
		if err != nil {
			continue
		}
		chip := strings.TrimSpace(string(nameRaw))
		if cpuHwmonChips[chip] {
			continue
		}
		out = append(out, hwmonChipTemps(e.Name(), chip)...)
	}
	counts := map[string]int{}
	for i := range out {
		counts[out[i].label]++
		if n := counts[out[i].label]; n > 1 {
			out[i].label = fmt.Sprintf("%s (%d)", out[i].label, n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].label < out[j].label })
	return out, nil
}

func hwmonChipTemps(dir, chip string) []sensorReading {
	paths, _ := filepath.Glob(filepath.Join(hwmonRoot, dir, "temp*_input"))
	var out []sensorReading
	for _, p := range paths {
		raw, err := readFile(p)
		if err != nil {
			continue
		}
		milli, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
		if err != nil {
			continue
		}
		base := strings.TrimSuffix(filepath.Base(p), "_input")
		label := chip + " " + base
		if l, err := readFile(filepath.Join(filepath.Dir(p), base+"_label")); err == nil {
			if s := strings.TrimSpace(string(l)); s != "" {
				label = s
			}
		}
		if !strings.HasPrefix(label, chip) {
			label = chip + " " + label
		}
		out = append(out, sensorReading{label: label, celsius: milli / 1000})
	}
	return out
}
