package collectors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeHwmon(t *testing.T, root, dir string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func withHwmonRoot(t *testing.T, root string) {
	t.Helper()
	old := hwmonRoot
	hwmonRoot = root
	t.Cleanup(func() { hwmonRoot = old })
}

func TestSensorsSkipsCPUChips(t *testing.T) {
	root := t.TempDir()
	writeHwmon(t, root, "hwmon0", map[string]string{"name": "k10temp", "temp1_input": "67000"})
	writeHwmon(t, root, "hwmon1", map[string]string{
		"name": "nvme", "temp1_input": "38500", "temp1_label": "Composite",
		"temp2_input": "40000", "temp2_label": "Sensor 2",
	})
	writeHwmon(t, root, "hwmon2", map[string]string{"name": "amdgpu", "temp1_input": "52000"})
	withHwmonRoot(t, root)

	snap, err := (&SensorsCollector{}).Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"nvme Composite 38.5°C", "nvme Sensor 2 40.0°C", "amdgpu temp1 52.0°C"}
	for _, w := range want {
		if !strings.Contains(snap.Summary, w) {
			t.Fatalf("summary missing %q: %q", w, snap.Summary)
		}
	}
	if strings.Contains(snap.Summary, "67.0") {
		t.Fatalf("CPU chip leaked into sensors: %q", snap.Summary)
	}
	if snap.HasUsage || snap.Usage != 0 {
		t.Fatalf("sensors should not render a usage bar: %+v", snap)
	}
}

func TestSensorsEmpty(t *testing.T) {
	root := t.TempDir()
	writeHwmon(t, root, "hwmon0", map[string]string{"name": "k10temp", "temp1_input": "67000"})
	withHwmonRoot(t, root)

	if _, err := (&SensorsCollector{}).Collect(t.Context()); err == nil {
		t.Fatal("expected error when only CPU chips exist")
	}
}

func TestSensorsDuplicateLabels(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"hwmon1", "hwmon2"} {
		writeHwmon(t, root, dir, map[string]string{
			"name": "nvme", "temp1_input": "40000", "temp1_label": "Composite",
		})
	}
	withHwmonRoot(t, root)

	snap, err := (&SensorsCollector{}).Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(snap.Summary, "nvme Composite 40.0°C\nnvme Composite (2) 40.0°C") {
		t.Fatalf("summary = %q", snap.Summary)
	}
}
