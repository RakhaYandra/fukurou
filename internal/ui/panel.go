// Package ui owns the GTK floating panel: module cards with live metrics.
package ui

import (
	"context"
	"time"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/RakhaYandra/fukurou/internal/collectors"
	"github.com/RakhaYandra/fukurou/internal/config"
	"github.com/RakhaYandra/fukurou/internal/shell"
)

const appID = "com.github.RakhaYandra.fukurou"

// collectorTimeout bounds each collector per tick; stragglers render
// as unavailable instead of stalling the dashboard.
const collectorTimeout = 2 * time.Second

const css = `
window {
	background-color: #1a1d21;
	border-radius: 12px;
	border: 1px solid #343941;
}
.title { color: #e6e9ef; font-size: 16px; font-weight: 700; }
.mod-name { color: #9aa3b2; font-size: 12px; font-weight: 700; letter-spacing: 1px; }
.mono { color: #e6e9ef; font-size: 13px; font-family: monospace; }
.dim { color: #565f6e; font-size: 12px; }
.ok { color: #9ece6a; font-size: 12px; }
.card {
	background-color: #22262c;
	border-radius: 8px;
	padding: 10px 12px;
}
levelbar trough { background-color: #343941; border-radius: 4px; min-height: 6px; }
levelbar block.filled { background-color: #7aa2f7; border-radius: 4px; }
`

// Run shows the live dashboard until dismissed (ESC, close, or ctx cancel).
// Returns the process exit code.
func Run(ctx context.Context, cfg config.Config, version string) int {
	// ApplicationFlags(0) == G_APPLICATION_DEFAULT_FLAGS; avoids the
	// deprecated ApplicationFlagsNone alias in this gotk4 version.
	app := gtk.NewApplication(appID, gio.ApplicationFlags(0))
	app.ConnectActivate(func() { activate(ctx, app, cfg, version) })
	go func() {
		<-ctx.Done()
		glib.IdleAdd(app.Quit)
	}()
	// Flags are parsed in main; GTK must not see them (GApplication
	// rejects unknown options like --config).
	return app.Run(nil)
}

func activate(ctx context.Context, app *gtk.Application, cfg config.Config, version string) {
	applyCSS()

	win := gtk.NewApplicationWindow(app)
	win.SetTitle("Fukurou")
	win.SetDefaultSize(cfg.Panel.Width, -1)
	win.SetResizable(false)
	win.SetDecorated(false)
	win.SetOpacity(cfg.Panel.Opacity)

	if shell.IsWayland() {
		shell.ConfigureOverlay(win.Native(), "fukurou", shell.NormalizePosition(cfg.Panel.Position))
	}

	root := gtk.NewBox(gtk.OrientationVertical, 10)
	root.SetMarginTop(20)
	root.SetMarginBottom(20)
	root.SetMarginStart(20)
	root.SetMarginEnd(20)

	head := gtk.NewBox(gtk.OrientationHorizontal, 8)
	title := gtk.NewLabel("Fukurou 梟")
	title.AddCSSClass("title")
	ver := gtk.NewLabel(version)
	ver.AddCSSClass("dim")
	head.Append(title)
	head.Append(ver)
	root.Append(head)

	dash := newDashboard(cfg.Enabled)
	root.Append(dash.box)
	win.SetChild(root)

	reg := collectors.DefaultRegistry(cfg.Enabled)
	refresh := func() {
		snaps := reg.CollectAll(ctx, collectorTimeout)
		glib.IdleAdd(func() { dash.update(snaps) })
	}

	keys := gtk.NewEventControllerKey()
	keys.ConnectKeyPressed(func(keyval, _ uint, _ gdk.ModifierType) bool {
		switch keyval {
		case gdk.KEY_Escape, gdk.KEY_q:
			app.Quit()
			return true
		case gdk.KEY_r:
			refresh()
			return true
		}
		return false
	})
	win.AddController(keys)

	win.Present()

	go func() {
		refresh() // immediate seed; second tick fills CPU % and net rates
		ticker := time.NewTicker(cfg.Refresh.Interval.Duration)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refresh()
			}
		}
	}()
}

func applyCSS() {
	prov := gtk.NewCSSProvider()
	prov.LoadFromString(css)
	display := gdk.DisplayGetDefault()
	gtk.StyleContextAddProviderForDisplay(display, prov, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
