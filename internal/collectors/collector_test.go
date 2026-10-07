package collectors

import (
	"context"
	"errors"
	"testing"
	"time"
)

var errBoom = errors.New("boom")

type stub struct{ name string }

func (s stub) Name() string { return s.name }
func (s stub) Collect(ctx context.Context) (Snapshot, error) {
	return Snapshot{Name: s.name, Available: false, Summary: "unavailable"}, nil
}

type errStub struct{ name string }

func (s errStub) Name() string { return s.name }
func (s errStub) Collect(ctx context.Context) (Snapshot, error) {
	return Snapshot{}, errBoom
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(stub{"cpu"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(stub{"cpu"}); err == nil {
		t.Fatal("expected duplicate error")
	}
	if err := r.Register(nil); err == nil {
		t.Fatal("expected nil error")
	}
	if got := r.Names(); len(got) != 1 || got[0] != "cpu" {
		t.Fatalf("names = %v", got)
	}
}

func TestCollectAllIsolatesFailures(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(stub{"ok"}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(errStub{"bad"}); err != nil {
		t.Fatal(err)
	}
	snaps := r.CollectAll(t.Context(), time.Second)
	if len(snaps) != 2 {
		t.Fatalf("snaps = %v", snaps)
	}
	byName := map[string]Snapshot{}
	for _, s := range snaps {
		byName[s.Name] = s
	}
	if got := byName["ok"]; got.Err != nil || got.Summary != "unavailable" {
		t.Fatalf("ok = %+v", got)
	}
	bad := byName["bad"]
	if bad.Available || !errors.Is(bad.Err, errBoom) {
		t.Fatalf("bad = %+v", bad)
	}
}

func TestDefaultRegistryRespectsFilter(t *testing.T) {
	r := DefaultRegistry(func(name string) bool { return name == "cpu" })
	if got := r.Names(); len(got) != 1 || got[0] != "cpu" {
		t.Fatalf("names = %v", got)
	}
	if len(DefaultRegistry(func(string) bool { return true }).Names()) != 7 {
		t.Fatal("expected 7 collectors")
	}
}
