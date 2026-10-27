package ui

import (
	"strings"
	"testing"
)

func TestCSSForTheme(t *testing.T) {
	if got := cssFor("light"); !strings.Contains(got, "#f7f8fa") {
		t.Fatalf("light css = %q", got)
	}
	if got := cssFor("dark"); !strings.Contains(got, "#1a1d21") {
		t.Fatalf("dark css = %q", got)
	}
	if got := cssFor("solarized"); got != cssFor("dark") {
		t.Fatal("unknown theme should fall back to dark")
	}
}
