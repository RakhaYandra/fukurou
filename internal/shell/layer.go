package shell

// #cgo pkg-config: gtk4-layer-shell-0 gtk4
// #include <gtk4-layer-shell/gtk4-layer-shell.h>
// #include <stdlib.h>
//
// static void fukurou_layer_configure(uintptr_t raw, const char *ns) {
// 	GtkWindow *win = (GtkWindow *)raw;
// 	gtk_layer_init_for_window(win);
// 	gtk_layer_set_layer(win, GTK_LAYER_SHELL_LAYER_OVERLAY);
// 	gtk_layer_set_keyboard_mode(win, GTK_LAYER_SHELL_KEYBOARD_MODE_EXCLUSIVE);
// 	gtk_layer_set_namespace(win, ns);
// 	// No anchors: all default to FALSE, compositor centers the surface.
// 	gtk_layer_set_exclusive_zone(win, 0);
// }
import "C"

import "unsafe"

// ConfigureOverlay turns a GtkWindow (as native handle) into a centered
// floating HUD: overlay layer, exclusive keyboard, given namespace.
func ConfigureOverlay(native uintptr, namespace string) {
	ns := C.CString(namespace)
	defer C.free(unsafe.Pointer(ns))
	C.fukurou_layer_configure(C.uintptr_t(native), ns)
}
