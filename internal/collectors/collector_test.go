package collectors

import (
	"context"
	"testing"
)

type stub struct{ name string }

func (s stub) Name() string { return s.name }
func (s stub) Collect(ctx context.Context) (Snapshot, error) {
	return Snapshot{Name: s.name, Available: false, Summary: "unavailable"}, nil
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
