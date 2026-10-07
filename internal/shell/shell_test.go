package shell

import "testing"

func TestNormalizePosition(t *testing.T) {
	cases := map[string]int{
		"center": PositionCenter, "top": PositionTop, "bottom": PositionBottom,
		"": PositionCenter, "sideways": PositionCenter, "TOP": PositionCenter,
	}
	for in, want := range cases {
		if got := NormalizePosition(in); got != want {
			t.Errorf("NormalizePosition(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestIsWayland(t *testing.T) {
	cases := []struct {
		display, backend string
		want             bool
	}{
		{"wayland-1", "", true},
		{"wayland-1", "wayland", true},
		{"wayland-1", "wayland,x11,*", true},
		{"wayland-1", "x11", false},
		{"", "", false},
		{"", "wayland", false},
	}
	for _, tc := range cases {
		t.Setenv("WAYLAND_DISPLAY", tc.display)
		t.Setenv("GDK_BACKEND", tc.backend)
		if got := IsWayland(); got != tc.want {
			t.Errorf("display=%q backend=%q: got %v, want %v", tc.display, tc.backend, got, tc.want)
		}
	}
}
