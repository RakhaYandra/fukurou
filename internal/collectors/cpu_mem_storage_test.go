package collectors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withReadFile(data map[string]string, t *testing.T) {
	t.Helper()
	old := readFile
	t.Cleanup(func() { readFile = old })
	readFile = func(path string) ([]byte, error) {
		if s, ok := data[path]; ok {
			return []byte(s), nil
		}
		return nil, os.ErrNotExist
	}
}

func TestParseCPUStat(t *testing.T) {
	s, err := parseCPUStat([]byte("cpu  100 0 50 850 0 0 0 0 0 0\ncpu0 50 0 25 425 0 0 0 0 0 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if s.total != 1000 || s.idle != 850 {
		t.Fatalf("got %+v", s)
	}
	if _, err := parseCPUStat([]byte("cpu0 1 2 3\n")); err == nil {
		t.Fatal("expected error for missing aggregate line")
	}
	if _, err := parseCPUStat([]byte("cpu 1 2\n")); err == nil {
		t.Fatal("expected error for short line")
	}
}

func TestCPUUsagePercent(t *testing.T) {
	u := cpuUsagePercent(cpuSample{idle: 850, total: 1000}, cpuSample{idle: 900, total: 1200})
	if u < 74 || u > 76 { // (1-50/200)*100 = 75
		t.Fatalf("usage = %v", u)
	}
	if got := cpuUsagePercent(cpuSample{idle: 1, total: 5}, cpuSample{idle: 1, total: 5}); got != 0 {
		t.Fatalf("zero delta = %v", got)
	}
}

func TestCPUCollectTwoTicks(t *testing.T) {
	withReadFile(map[string]string{
		"/proc/stat":    "cpu  100 0 50 850 0 0 0 0 0 0\n",
		"/proc/cpuinfo": "model name\t: Test CPU 9000\ncpu MHz\t\t: 3800.000\n",
	}, t)
	oldRoot := hwmonRoot
	hwmonRoot = t.TempDir()
	t.Cleanup(func() { hwmonRoot = oldRoot })

	c := &CPUCollector{}
	first, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !first.Available || !strings.Contains(first.Summary, "Test CPU 9000") {
		t.Fatalf("first = %+v", first)
	}
	// Second tick with busier counters.
	readFile = func(path string) ([]byte, error) {
		if path == "/proc/stat" {
			return []byte("cpu  200 0 100 900 0 0 0 0 0 0\n"), nil
		}
		if path == "/proc/cpuinfo" {
			return []byte("model name\t: Test CPU 9000\ncpu MHz\t\t: 3800.000\n"), nil
		}
		return nil, os.ErrNotExist
	}
	second, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(second.Summary, "Test CPU 9000\n75.0%") {
		t.Fatalf("second summary = %q", second.Summary)
	}
}

func TestCPUTempHwmon(t *testing.T) {
	root := t.TempDir()
	write := func(dir, name, temp string) {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "name"), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "temp1_input"), []byte(temp), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("hwmon0", "acpitz", "55000")
	write("hwmon1", "k10temp", "67250")
	old := hwmonRoot
	hwmonRoot = root
	t.Cleanup(func() { hwmonRoot = old })

	got, err := cpuTemp()
	if err != nil {
		t.Fatal(err)
	}
	if got != 67.25 {
		t.Fatalf("temp = %v", got)
	}
}

func TestParseMeminfo(t *testing.T) {
	m, err := parseMeminfo([]byte("MemTotal:       16000000 kB\nMemFree:         2000000 kB\nMemAvailable:    8000000 kB\n"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Usage != 50 {
		t.Fatalf("usage = %v", m.Usage)
	}
	if m.Used != 8000000*1024 {
		t.Fatalf("used = %v", m.Used)
	}
	// Fallback to MemFree.
	m2, err := parseMeminfo([]byte("MemTotal:       8000000 kB\nMemFree:         2000000 kB\n"))
	if err != nil {
		t.Fatal(err)
	}
	if m2.Usage != 75 {
		t.Fatalf("fallback usage = %v", m2.Usage)
	}
	if _, err := parseMeminfo([]byte("MemFree: 1 kB\n")); err == nil {
		t.Fatal("expected error without MemTotal")
	}
	if _, err := parseMeminfo([]byte("MemTotal: x kB\n")); err == nil {
		t.Fatal("expected error for malformed value")
	}
}

func TestStorageFromStatfs(t *testing.T) {
	m := storageFromStatfs("/", 1000, 400, 350, 4096)
	if m.Total != 1000*4096 || m.Free != 350*4096 || m.Used != 600*4096 {
		t.Fatalf("got %+v", m)
	}
	if m.Usage != 60 {
		t.Fatalf("usage = %v", m.Usage)
	}
	zero := storageFromStatfs("/", 0, 0, 0, 4096)
	if zero.Usage != 0 {
		t.Fatalf("zero total usage = %v", zero.Usage)
	}
}

func TestStorageCollectRoot(t *testing.T) {
	snap, err := (&StorageCollector{}).Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Available {
		t.Fatal("root should be available")
	}
}
