# Architecture (public overview)

Modular layered architecture. UI never touches `/proc` directly.

```text
┌───────────────┐
│      UI       │  GTK4 + layer-shell (Phase 2)
├───────────────┤
│  Application  │  orchestration, state, scheduler
├───────────────┤
│ Collectors    │  CPU / MEM / GPU / storage / net / system
├───────────────┤
│ Linux         │  /proc /sys / sensors / GPU APIs
└───────────────┘
```

Data flow: `Linux → Collector → Snapshot → App state → GTK`.
Each collector isolated: GPU failure → "Unavailable", app keeps running.

```go
type Collector interface {
    Name() string
    Collect(ctx context.Context) (Snapshot, error)
}
```

Concurrency: collectors run off the UI thread via goroutines +
`context.Context`; UI polls snapshots on the refresh interval (default 1s).

Decisions: Go (binary + concurrency), GTK4 (native Linux),
layer-shell (HUD, not a window), read-only (no root, least privilege),
YAML config, no DB in v0.1.
