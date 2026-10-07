package ui

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"

	"github.com/RakhaYandra/fukurou/internal/collectors"
)

// displayOrder is fixed regardless of registry sort: identity first,
// then load, then the rest.
var displayOrder = []string{"system", "cpu", "memory", "gpu", "storage", "network"}

// Dashboard owns the module cards in display order. All methods run on
// the GTK main thread (updates arrive via glib.IdleAdd).
type Dashboard struct {
	box   *gtk.Box
	cards map[string]*moduleCard
}

func newDashboard(enabled func(string) bool) *Dashboard {
	box := gtk.NewBox(gtk.OrientationVertical, 10)
	d := &Dashboard{box: box, cards: map[string]*moduleCard{}}
	for _, name := range displayOrder {
		if !enabled(name) {
			continue
		}
		card := newModuleCard(name)
		d.cards[name] = card
		box.Append(card.root)
	}
	return d
}

// update refreshes visible cards; unknown or disabled names are ignored.
func (d *Dashboard) update(snaps []collectors.Snapshot) {
	for _, snap := range snaps {
		if card, ok := d.cards[snap.Name]; ok {
			card.update(snap)
		}
	}
}
