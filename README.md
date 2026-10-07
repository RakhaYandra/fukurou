# Fukurou 梟 — Watch your system.

Lightweight floating system dashboard for Linux (Wayland/Hyprland).
Read-only: observe, don't control. No root, no telemetry, no network calls.

> Status: **v0.1 Phase 5** — dashboard live, config fully wired
> (modules, size, position, opacity, interval).

## Quickstart

```bash
go build -o bin/fukurou ./cmd/fukurou
./bin/fukurou --debug
./bin/fukurou --version
```

## Configuration

File: `~/.config/fukurou/config.yaml` (override with `--config PATH`).
Template: `configs/config.example.yaml`. Hyprland bind example:
`configs/hyprland.conf.example`.

```yaml
panel:
  width: 520        # 320–1200, clamped
  position: center  # center | top | bottom (other → center)
  opacity: 0.96     # 0.3–1.0, clamped
refresh:
  interval: 1s      # 250ms–5s, clamped
modules:
  cpu: true         # false hides the card entirely
  gpu: false
```

Rules: missing file → defaults silently (first run). Invalid YAML →
stderr warning + safe defaults, never crash. No restart-daemon: relaunch
to apply (single-shot model).

## Layout

```text
cmd/fukurou/          entrypoint (flags, signals)
internal/config/      YAML + defaults + validation
internal/collectors/  Collector interface + metric models
internal/app/         composition root
configs/              example config
docs/                 public architecture overview
```

Full product spec lives in the private `fukurou-internal` repo.

## Principles

Fast · Minimal · Readable · Non-intrusive · Read-only · Extensible

## License

MIT
