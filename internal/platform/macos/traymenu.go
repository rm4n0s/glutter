package macos

import (
	"bytes"
	"image"
	"image/png"
	"math"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the tray (SPEC §29.9): the menu of the icon as a plan of entries with the
// numbers that NSMenuItem.tag carries back, and the text of the tip. The calls to AppKit are in tray_darwin.go.

// trayEntry is an entry of the menu of the tray icon in the form that the menu is built from: ID is the tag of
// its NSMenuItem (above 0) and Children are the entries of its submenu.
type trayEntry struct {
	ID        int
	Label     string
	Separator bool
	Disabled  bool
	// Checked puts the check mark; a radio item has the same mark (an NSMenuItem has the states on and off only).
	Checked  bool
	Children []trayEntry
}

// planTrayMenu makes the plan of the menu: a hidden item is left out; a separator that would be first, last or next
// to another is left out; the entries of the tree are numbered from 1 in the order of the menu, and handlers has
// the OnPressed function of each item that has one and is not disabled. An item with children is a submenu: its
// own OnPressed is not called, as for the menus of the other desktops.
func planTrayMenu(items []platform.TrayMenuItem) (plan []trayEntry, handlers map[int]func()) {
	handlers = map[int]func(){}
	next := 0
	var build func(items []platform.TrayMenuItem) []trayEntry
	build = func(items []platform.TrayMenuItem) []trayEntry {
		var out []trayEntry
		for _, it := range items {
			if it.Hidden {
				continue
			}
			if it.Separator {
				if len(out) == 0 || out[len(out)-1].Separator {
					continue
				}
				next++
				out = append(out, trayEntry{ID: next, Separator: true})
				continue
			}
			next++
			e := trayEntry{ID: next, Label: it.Label, Disabled: it.Disabled, Checked: (it.Checkable || it.Radio) && it.Checked}
			if len(it.Children) > 0 {
				e.Children = build(it.Children)
			} else if it.OnPressed != nil && !it.Disabled {
				handlers[e.ID] = it.OnPressed
			}
			out = append(out, e)
		}
		if n := len(out); n > 0 && out[n-1].Separator {
			out = out[:n-1]
		}
		return out
	}
	return build(items), handlers
}

// trayTip is the text of the tip of the icon: the first line, and the body under it.
func trayTip(spec platform.TraySpec) string {
	title := spec.Tooltip
	if title == "" {
		title = spec.Title
	}
	switch {
	case title == "":
		return spec.TooltipBody
	case spec.TooltipBody == "":
		return title
	}
	return strings.TrimRight(title, "\n") + "\n" + spec.TooltipBody
}

// trayIconPoints is the height of the icon in the menu bar, in points.
const trayIconPoints = 18

// trayIconPNG is the picture of the icon for the menu bar as a PNG, and its size in points: the picture whose size
// in points is nearest to 18, and of those the one with the most pixels. It is nil when the icon has no picture.
func trayIconPNG(icon ui.WindowIcon) (data []byte, points float64, err error) {
	best := -1
	bestDist := 0.0
	for i, img := range icon.Images {
		scale := float64(max(img.Scale, 1))
		logical := float64(img.Size) / scale
		d := math.Abs(logical - trayIconPoints)
		if best < 0 || d < bestDist || (d == bestDist && img.Size > icon.Images[best].Size) {
			best, bestDist = i, d
		}
	}
	if best < 0 {
		return nil, 0, nil
	}
	img := icon.Images[best]
	pic := &image.NRGBA{Pix: img.RGBA, Stride: img.Size * 4, Rect: image.Rect(0, 0, img.Size, img.Size)}
	var buf bytes.Buffer
	if err := png.Encode(&buf, pic); err != nil {
		return nil, 0, errors.NewErr(tagTrayRegisterFailed, err).SetMetadata("size", img.Size)
	}
	return buf.Bytes(), float64(img.Size) / float64(max(img.Scale, 1)), nil
}

// The tags of the errors that the tray makes (rm4n0s/errors, R5); package desktop matches them by value.
const (
	tagTrayDuplicate      = "TrayDuplicate"
	tagTrayRegisterFailed = "TrayRegisterFailed"
)
