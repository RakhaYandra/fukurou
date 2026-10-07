package collectors

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSupply(t *testing.T, root, name string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
		t.Fatal(err)
	}
	for f, content := range files {
		if err := os.WriteFile(filepath.Join(root, name, f), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBatteryDischarging(t *testing.T) {
	root := t.TempDir()
	writeSupply(t, root, "BAT1", map[string]string{"capacity": "77\n", "status": "Discharging\n"})
	old := powerSupplyRoot
	powerSupplyRoot = root
	t.Cleanup(func() { powerSupplyRoot = old })

	snap, err := (&BatteryCollector{}).Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Available || snap.Summary != "77% · discharging" {
		t.Fatalf("snap = %+v", snap)
	}
	if !snap.HasUsage || snap.Usage != 77 {
		t.Fatalf("usage = %v/%v", snap.Usage, snap.HasUsage)
	}
}

func TestBatteryChargingAC(t *testing.T) {
	root := t.TempDir()
	writeSupply(t, root, "BAT0", map[string]string{"capacity": "42\n", "status": "Charging\n"})
	writeSupply(t, root, "ACAD", map[string]string{"online": "1\n"})
	old := powerSupplyRoot
	powerSupplyRoot = root
	t.Cleanup(func() { powerSupplyRoot = old })

	snap, err := (&BatteryCollector{}).Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Summary != "42% · charging · AC" {
		t.Fatalf("summary = %q", snap.Summary)
	}
}

func TestBatteryNotChargingOmitsStatus(t *testing.T) {
	root := t.TempDir()
	writeSupply(t, root, "BAT1", map[string]string{"capacity": "77\n", "status": "Not charging\n"})
	writeSupply(t, root, "ACAD", map[string]string{"online": "1\n"})
	old := powerSupplyRoot
	powerSupplyRoot = root
	t.Cleanup(func() { powerSupplyRoot = old })

	snap, err := (&BatteryCollector{}).Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Summary != "77% · AC" {
		t.Fatalf("summary = %q", snap.Summary)
	}
}

func TestBatteryAbsent(t *testing.T) {
	root := t.TempDir()
	writeSupply(t, root, "ACAD", map[string]string{"online": "1\n"})
	old := powerSupplyRoot
	powerSupplyRoot = root
	t.Cleanup(func() { powerSupplyRoot = old })

	snap, err := (&BatteryCollector{}).Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Available {
		t.Fatalf("desktop without battery should be unavailable: %+v", snap)
	}
}

func TestBatteryBadCapacity(t *testing.T) {
	root := t.TempDir()
	writeSupply(t, root, "BAT1", map[string]string{"capacity": "junk\n"})
	old := powerSupplyRoot
	powerSupplyRoot = root
	t.Cleanup(func() { powerSupplyRoot = old })

	snap, err := (&BatteryCollector{}).Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Available {
		t.Fatalf("malformed capacity should be unavailable: %+v", snap)
	}
}
