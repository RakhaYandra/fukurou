// Package ui owns the GTK floating panel: module cards with live metrics.
package ui

import (
	"context"
	"fmt"
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

type palette struct {
	window, border, title, modName, mono, dim, ok, card, trough, fill string
}

var palettes = map[string]palette{
	"dark": {
		window: "#1a1d21", border: "#343941", title: "#e6e9ef",
		modName: "#9aa3b2", mono: "#e6e9ef", dim: "#565f6e", ok: "#9ece6a",
		card: "#22262c", trough: "#343941", fill: "#7aa2f7",
	},
	"light": {
		window: "#f7f8fa", border: "#d6dae1", title: "#1b1f24",
		modName: "#5c6574", mono: "#1b1f24", dim: "#9aa3b2", ok: "#4c8c2b",
		card: "#ffffff", trough: "#e3e7ee", fill: "#3b6fd4",
	},
}

func cssFor(theme string) string {
	p, ok := palettes[theme]
	if !ok {
		p = palettes["dark"]
	}
	return fmt.Sprintf(`
window {
	background-color: %s;
	border-radius: 12px;
	border: 1px solid %s;
}
.title { color: %s; font-size: 16px; font-weight: 700; }
.mod-name { color: %s; font-size: 12px; font-weight: 700; letter-spacing: 1px; }
.mono { color: %s; font-size: 13px; font-family: monospace; }
.dim { color: %s; font-size: 12px; }
.ok { color: %s; font-size: 12px; }
.card {
	background-color: %s;
	border-radius: 8px;
	padding: 10px 12px;
}
levelbar > trough { background: %s; border: none; border-radius: 4px; min-height: 6px; }
levelbar > trough > block { background: %s; border: none; border-radius: 4px; box-shadow: none; }
levelbar > trough > block.filled { background: %s; border: none; border-radius: 4px; }
`, p.window, p.border, p.title, p.modName, p.mono, p.dim, p.ok, p.card, p.trough, p.trough, p.fill)
}

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
	applyCSS(cfg.Panel.Theme)

	win := gtk.NewApplicationWindow(app)
	win.SetTitle("Fukurou")
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
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolled.SetPropagateNaturalHeight(true)
	if h := maxContentHeight(monitorHeight()); h > 0 {
		scrolled.SetMaxContentHeight(h)
	}
	scrolled.SetChild(dash.box)
	root.Append(scrolled)
	win.SetChild(root)
	win.SetDefaultSize(cfg.Panel.Width, -1)

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

func applyCSS(theme string) {
	prov := gtk.NewCSSProvider()
	prov.LoadFromString(cssFor(theme))
	display := gdk.DisplayGetDefault()
	gtk.StyleContextAddProviderForDisplay(display, prov, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}

const (
	chromeHeight = 80 // header, root margins, and spacing around the card list
	maxPanelFrac = 0.9
)

func maxContentHeight(monitor int) int {
	if monitor <= 0 {
		return 0
	}
	return int(float64(monitor)*maxPanelFrac) - chromeHeight
}

func monitorHeight() int {
	display := gdk.DisplayGetDefault()
	if display == nil {
		return 0
	}
	model := display.Monitors()
	if model == nil || model.NItems() == 0 {
		return 0
	}
	obj := model.Item(0)
	if obj == nil {
		return 0
	}
	return (&gdk.Monitor{Object: obj}).Geometry().Height()
}
