// Package ui owns the GTK floating panel. Metrics land in Phase 3;
// Phase 2 proves the shell: centered overlay, keyboard, ESC dismissal.
package ui

import (
	"context"
	"os"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/RakhaYandra/fukurou/internal/config"
	"github.com/RakhaYandra/fukurou/internal/shell"
)

const appID = "com.github.RakhaYandra.fukurou"

const css = `
window {
	background-color: #1a1d21;
	border-radius: 12px;
	border: 1px solid #343941;
}
.title { color: #e6e9ef; font-size: 20px; font-weight: 700; }
.subtitle { color: #9aa3b2; font-size: 13px; }
`

// Run shows the panel until dismissed (ESC, close, or ctx cancel).
// Returns the process exit code.
func Run(ctx context.Context, cfg config.Config, version string) int {
	app := gtk.NewApplication(appID, gio.ApplicationFlagsNone)
	app.ConnectActivate(func() { activate(app, cfg, version) })
	go func() {
		<-ctx.Done()
		glib.IdleAdd(app.Quit)
	}()
	return app.Run(os.Args)
}

func activate(app *gtk.Application, cfg config.Config, version string) {
	applyCSS()

	win := gtk.NewApplicationWindow(app)
	win.SetTitle("Fukurou")
	win.SetDefaultSize(cfg.Panel.Width, 420)
	win.SetResizable(false)
	win.SetDecorated(false)
	win.SetOpacity(cfg.Panel.Opacity)

	if shell.IsWayland() {
		shell.ConfigureOverlay(win.Native(), "fukurou")
	}

	box := gtk.NewBox(gtk.OrientationVertical, 8)
	box.SetHAlign(gtk.AlignCenter)
	box.SetVAlign(gtk.AlignCenter)

	title := gtk.NewLabel("Fukurou 梟")
	title.AddCSSClass("title")
	sub := gtk.NewLabel("system dashboard " + version + "\nmetrics arrive in Phase 3 — try --debug")
	sub.AddCSSClass("subtitle")
	box.Append(title)
	box.Append(sub)
	win.SetChild(box)

	keys := gtk.NewEventControllerKey()
	keys.ConnectKeyPressed(func(keyval, _ uint, _ gdk.ModifierType) bool {
		if keyval == gdk.KEY_Escape {
			app.Quit()
			return true
		}
		return false
	})
	win.AddController(keys)

	win.Present()
}

func applyCSS() {
	prov := gtk.NewCSSProvider()
	prov.LoadFromString(css)
	display := gdk.DisplayGetDefault()
	gtk.StyleContextAddProviderForDisplay(display, prov, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
