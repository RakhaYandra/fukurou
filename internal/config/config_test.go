package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultSane(t *testing.T) {
	c := Default()
	if c.Panel.Width != 520 {
		t.Fatalf("width = %d", c.Panel.Width)
	}
	if c.Refresh.Interval.Duration != time.Second {
		t.Fatalf("interval = %v", c.Refresh.Interval.Duration)
	}
	for _, m := range []string{"system", "cpu", "memory", "gpu", "storage", "network", "battery"} {
		if !c.Enabled(m) {
			t.Fatalf("module %s should default on", m)
		}
	}
}

func TestLoadMissingFallsBackToDefaults(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Panel.Width != 520 {
		t.Fatalf("width = %d", c.Panel.Width)
	}
}

func TestLoadMalformedReturnsDefaultsAndError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(p, []byte("modules:\n\tcpu: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err == nil {
		t.Fatal("expected error")
	}
	if c.Panel.Width != 520 {
		t.Fatalf("width = %d", c.Panel.Width)
	}
}

func TestLoadTogglesModule(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.yaml")
	body := "version: 1\nrefresh:\n  interval: 500ms\nmodules:\n  cpu: true\n  gpu: false\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Enabled("cpu") || c.Enabled("gpu") {
		t.Fatalf("toggles not applied: %+v", c.Modules)
	}
	if c.Refresh.Interval.Duration != 500*time.Millisecond {
		t.Fatalf("interval = %v", c.Refresh.Interval.Duration)
	}
}

func TestNormalizeClamps(t *testing.T) {
	c := Default()
	c.Panel.Width = 10
	c.Panel.Opacity = 99
	c = c.Normalize()
	if c.Panel.Width != 320 || c.Panel.Opacity != 1 {
		t.Fatalf("not clamped: %+v", c.Panel)
	}
}
