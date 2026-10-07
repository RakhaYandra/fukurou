package collectors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RakhaYandra/fukurou/internal/config"
)

// TestRegistryFollowsLoadedConfig is the Phase 5 acceptance test:
// toggling a module in YAML actually removes it from the dashboard.
func TestRegistryFollowsLoadedConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	body := "version: 1\npanel:\n  width: 640\nrefresh:\n  interval: 500ms\n" +
		"modules:\n  cpu: true\n  gpu: false\n  memory: true\n" +
		"  storage: true\n  network: true\n  system: true\n  battery: true\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Panel.Width != 640 {
		t.Fatalf("width = %d", cfg.Panel.Width)
	}
	names := DefaultRegistry(cfg.Enabled).Names()
	for _, n := range names {
		if n == "gpu" {
			t.Fatalf("gpu should be disabled: %v", names)
		}
	}
	if len(names) != 6 {
		t.Fatalf("names = %v", names)
	}
}
