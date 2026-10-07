package shell

import (
	"os"
	"strings"
)

// NormalizePosition maps user input to a Position constant.
// Unknown values fall back to center.
func NormalizePosition(s string) int {
	switch s {
	case "top":
		return PositionTop
	case "bottom":
		return PositionBottom
	default:
		return PositionCenter
	}
}

// IsWayland reports whether layer-shell can be used.
// GDK_BACKEND is a preference list ("wayland,x11,*"); only an explicit
// x11-first (or missing WAYLAND_DISPLAY) disables the layer path.
func IsWayland() bool {
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		return false
	}
	if first, _, _ := strings.Cut(os.Getenv("GDK_BACKEND"), ","); first == "x11" {
		return false
	}
	return true
}
