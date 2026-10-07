# Fukurou 梟 — Watch your system.

Lightweight floating system dashboard for Linux (Wayland/Hyprland).
Read-only: observe, don't control. No root, no telemetry, no network calls.

> Status: **v0.1 Phase 2** — floating panel works (layer-shell overlay,
> centered, ESC/SIGTERM dismiss). Metrics render in Phase 3.

## Quickstart

```bash
go build -o bin/fukurou ./cmd/fukurou
./bin/fukurou --debug
./bin/fukurou --version
```

Config: `~/.config/fukurou/config.yaml` (see `configs/config.example.yaml`).
Invalid config → log + safe defaults, never crash.

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
