package shell

// #cgo pkg-config: gtk4-layer-shell-0 gtk4
// #include <gtk4-layer-shell/gtk4-layer-shell.h>
// #include <stdlib.h>
//
// static void fukurou_layer_configure(uintptr_t raw, const char *ns, int pos) {
// 	GtkWindow *win = (GtkWindow *)raw;
// 	gtk_layer_init_for_window(win);
// 	gtk_layer_set_layer(win, GTK_LAYER_SHELL_LAYER_OVERLAY);
// 	gtk_layer_set_keyboard_mode(win, GTK_LAYER_SHELL_KEYBOARD_MODE_EXCLUSIVE);
// 	gtk_layer_set_namespace(win, ns);
// 	gtk_layer_set_exclusive_zone(win, 0);
// 	// pos: 0 center (no anchors, compositor centers), 1 top, 2 bottom.
// 	if (pos == 1) {
// 		gtk_layer_set_anchor(win, GTK_LAYER_SHELL_EDGE_TOP, TRUE);
// 		gtk_layer_set_margin(win, GTK_LAYER_SHELL_EDGE_TOP, 24);
// 	} else if (pos == 2) {
// 		gtk_layer_set_anchor(win, GTK_LAYER_SHELL_EDGE_BOTTOM, TRUE);
// 		gtk_layer_set_margin(win, GTK_LAYER_SHELL_EDGE_BOTTOM, 24);
// 	}
// }
import "C"

import "unsafe"

// Position values for ConfigureOverlay.
const (
	PositionCenter = iota
	PositionTop
	PositionBottom
)

// ConfigureOverlay turns a GtkWindow (as native handle) into a floating HUD:
// overlay layer, exclusive keyboard, given namespace and vertical position.
func ConfigureOverlay(native uintptr, namespace string, position int) {
	ns := C.CString(namespace)
	defer C.free(unsafe.Pointer(ns))
	C.fukurou_layer_configure(C.uintptr_t(native), ns, C.int(position))
}
