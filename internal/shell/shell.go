package shell

import (
	"os"
	"strings"
)

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
