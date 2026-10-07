package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/RakhaYandra/fukurou/internal/collectors"
)

func TestFormatSnapshot(t *testing.T) {
	ok := formatSnapshot(collectors.Snapshot{Name: "cpu", Available: true, Summary: "12%\nline2"})
	if !strings.HasPrefix(ok, "● CPU\n  12%\n  line2\n") {
		t.Fatalf("ok = %q", ok)
	}
	bad := formatSnapshot(collectors.Snapshot{Name: "gpu", Err: errors.New("boom")})
	if !strings.HasPrefix(bad, "○ GPU\n  unavailable (boom)\n") {
		t.Fatalf("bad = %q", bad)
	}
	unavail := formatSnapshot(collectors.Snapshot{Name: "gpu", Summary: "unavailable"})
	if !strings.HasPrefix(unavail, "○ GPU\n") {
		t.Fatalf("unavail = %q", unavail)
	}
}
