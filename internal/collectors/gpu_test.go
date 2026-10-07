package collectors

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestParseSmiCSV(t *testing.T) {
	m, err := parseSmiCSV("NVIDIA GeForce RTX 4050 Laptop GPU, 0 %, 46, 5 MiB, 6141 MiB\n")
	if err != nil {
		t.Fatal(err)
	}
	if m.Vendor != "NVIDIA" || m.Model != "NVIDIA GeForce RTX 4050 Laptop GPU" {
		t.Fatalf("got %+v", m)
	}
	if m.Usage != 0 || *m.Temperature != 46 {
		t.Fatalf("got %+v", m)
	}
	if m.VRAMUsed != 5<<20 || m.VRAMTotal != 6141<<20 {
		t.Fatalf("got %+v", m)
	}
	if _, err := parseSmiCSV("garbage\n"); err == nil {
		t.Fatal("expected error for malformed smi output")
	}
}

func TestGPUGracefulFallback(t *testing.T) {
	// Kill every source: bad NVML path, failing smi, empty drm dir.
	oldPath := nvmlLibPath
	nvmlLibPath = "nonexistent-lib.so"
	t.Cleanup(func() { nvmlLibPath = oldPath })
	oldExec := execSmi
	execSmi = func(context.Context) (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { execSmi = oldExec })
	oldDrm := drmRoot
	drmRoot = t.TempDir()
	t.Cleanup(func() { drmRoot = oldDrm })

	snap, err := (&GPUCollector{}).Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Available {
		t.Fatal("should be unavailable with no GPU sources")
	}
	if !strings.Contains(snap.Summary, "unavailable") {
		t.Fatalf("summary = %q", snap.Summary)
	}
}

func TestNVMLReal(t *testing.T) {
	n, err := nvmlOpen(nvmlLibPath)
	if err != nil {
		t.Skipf("no NVML here: %v", err)
	}
	m, err := n.firstGPU()
	if err != nil {
		t.Fatal(err)
	}
	if m.Vendor != "NVIDIA" || m.Model == "" {
		t.Fatalf("got %+v", m)
	}
	t.Logf("NVML OK: %s %.0f%%", m.Model, m.Usage)
}
