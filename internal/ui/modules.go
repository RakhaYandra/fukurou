package ui

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/RakhaYandra/fukurou/internal/collectors"
)

// moduleCard is one metric section: status dot + name, mono body, usage bar.
type moduleCard struct {
	root   *gtk.Box
	status *gtk.Label
	body   *gtk.Label
	bar    *gtk.LevelBar
}

func newModuleCard(name string) *moduleCard {
	root := gtk.NewBox(gtk.OrientationVertical, 4)
	root.AddCSSClass("card")

	head := gtk.NewBox(gtk.OrientationHorizontal, 6)
	status := gtk.NewLabel("○")
	status.AddCSSClass("dim")
	title := gtk.NewLabel(strings.ToUpper(name))
	title.AddCSSClass("mod-name")
	head.Append(status)
	head.Append(title)

	body := gtk.NewLabel("…")
	body.SetXAlign(0)
	body.SetWrap(true)
	body.AddCSSClass("mono")

	bar := gtk.NewLevelBar()
	bar.SetMode(gtk.LevelBarModeContinuous)
	bar.SetMinValue(0)
	bar.SetMaxValue(100)
	bar.SetValue(0)

	root.Append(head)
	root.Append(body)
	root.Append(bar)
	return &moduleCard{root: root, status: status, body: body, bar: bar}
}

func (c *moduleCard) update(snap collectors.Snapshot) {
	if !snap.Available || snap.Err != nil {
		c.status.SetLabel("○")
		c.status.RemoveCSSClass("ok")
		c.status.AddCSSClass("dim")
		c.body.SetLabel("unavailable")
		c.bar.SetVisible(false)
		return
	}
	c.status.SetLabel("●")
	c.status.RemoveCSSClass("dim")
	c.status.AddCSSClass("ok")
	c.body.SetLabel(snap.Summary)
	if snap.HasUsage {
		c.bar.SetVisible(true)
		c.bar.SetValue(snap.Usage)
	} else {
		c.bar.SetVisible(false)
	}
}
