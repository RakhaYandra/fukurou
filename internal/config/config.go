// Package config loads Fukurou's YAML configuration with safe defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration. Keep it small: observe, don't control.
type Config struct {
	Version int             `yaml:"version"`
	Panel   PanelConfig     `yaml:"panel"`
	Refresh RefreshConfig   `yaml:"refresh"`
	Modules map[string]bool `yaml:"modules"`
}

type PanelConfig struct {
	Width    int     `yaml:"width"`
	Position string  `yaml:"position"`
	Opacity  float64 `yaml:"opacity"`
}

type RefreshConfig struct {
	Interval Duration `yaml:"interval"`
}

// Duration wraps time.Duration for YAML (e.g. "1s", "500ms").
type Duration struct{ time.Duration }

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = parsed
	return nil
}

// Default returns sane defaults; never fails.
func Default() Config {
	return Config{
		Version: 1,
		Panel:   PanelConfig{Width: 520, Position: "center", Opacity: 0.96},
		Refresh: RefreshConfig{Interval: Duration{time.Second}},
		Modules: map[string]bool{
			"system": true, "cpu": true, "memory": true,
			"gpu": true, "storage": true, "network": true, "battery": true,
		},
	}
}

// DefaultPath is ~/.config/fukurou/config.yaml.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "fukurou", "config.yaml")
}

// Load reads path (or default). Missing file → defaults, nil error.
// Malformed file → defaults + non-nil error (caller logs, keeps running).
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = DefaultPath()
	}
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	loaded := Default()
	// Start from defaults so omitted keys keep working.
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		return cfg, fmt.Errorf("parse %s: %w", path, err)
	}
	return loaded.Normalize(), nil
}

// Normalize clamps out-of-range values instead of failing.
func (c Config) Normalize() Config {
	if c.Panel.Width < 320 {
		c.Panel.Width = 320
	}
	if c.Panel.Width > 1200 {
		c.Panel.Width = 1200
	}
	if c.Panel.Opacity < 0.3 {
		c.Panel.Opacity = 0.3
	}
	if c.Panel.Opacity > 1 {
		c.Panel.Opacity = 1
	}
	d := c.Refresh.Interval.Duration
	if d < 250*time.Millisecond {
		d = 250 * time.Millisecond
	}
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	c.Refresh.Interval.Duration = d
	if c.Modules == nil {
		c.Modules = Default().Modules
	}
	return c
}

// Enabled reports whether a module is on (unknown → false).
func (c Config) Enabled(name string) bool {
	on, ok := c.Modules[name]
	return ok && on
}
