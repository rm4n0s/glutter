package windows

import (
	"bytes"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the plain-Go part of the system tray of Windows (SPEC §25.9): the notify icon and what
// is sent about it, the diff of a rebuilt model, the events of the icon, and the plan of the popup menu
// (the item numbering and the labels).

// The limit of the tooltip, in UTF-16 units without the NUL (szTip holds 128).
const maxTipUnits = 127

// trayChange tells what differs between two models of the tray.
type trayChange uint8

const (
	// changeIcon: the pictures of the icon differ.
	changeIcon trayChange = 1 << iota
	// changeTip: the tooltip differs.
	changeTip
	// changeMenu: the menu differs. Nothing is sent for it: the menu is made when it is shown.
	changeMenu
)

// tipOf is the text of the tooltip: the first line, then the text under it, cut to szTip.
func tipOf(spec platform.TraySpec) string {
	tip := spec.Tooltip
	if spec.TooltipBody != "" {
		if tip != "" {
			tip += "\n"
		}
		tip += spec.TooltipBody
	}
	return limitText(tip, maxTipUnits)
}

// diffTray says what differs between the old model and the new one. The functions of the models are not
// compared: they are replaced at every update.
func diffTray(old, cur platform.TraySpec) trayChange {
	var c trayChange
	if !sameIcon(old.Icon, cur.Icon) {
		c |= changeIcon
	}
	if tipOf(old) != tipOf(cur) {
		c |= changeTip
	}
	if !sameMenu(old.Menu, cur.Menu) {
		c |= changeMenu
	}
	return c
}

func sameIcon(a, b ui.WindowIcon) bool {
	if a.Name != b.Name || len(a.Images) != len(b.Images) {
		return false
	}
	for i := range a.Images {
		x, y := a.Images[i], b.Images[i]
		if x.Size != y.Size || x.Scale != y.Scale || !bytes.Equal(x.RGBA, y.RGBA) {
			return false
		}
	}
	return true
}

func sameMenu(a, b []platform.TrayMenuItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if x.Label != y.Label || x.Separator != y.Separator || x.Disabled != y.Disabled || x.Hidden != y.Hidden ||
			x.Checkable != y.Checkable || x.Radio != y.Radio || x.Checked != y.Checked || !sameMenu(x.Children, y.Children) {
			return false
		}
	}
	return true
}

// menuEntry is an item of the popup menu as it is inserted: its number, its text and its flags.
type menuEntry struct {
	// ID is the command number that TrackPopupMenuEx returns for the item. The items are numbered
	// depth first from 1 in the order of the model (a separator and the item of a submenu have one too);
	// an item that is hidden is left out and takes no number.
	ID uint32
	// Label is the text with each & doubled: a single one marks a mnemonic.
	Label               string
	Separator, Disabled bool
	Checked, Radio      bool
	Children            []menuEntry
}

// menuPlan is the popup menu of a model: its entries and the function of each item.
type menuPlan struct {
	Entries []menuEntry
	// pressed finds the function of an item by its number.
	pressed map[uint32]func()
}

// escapeLabel doubles each & of a label, and drops a NUL, since the text goes to Windows as a C string.
func escapeLabel(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\x00", ""), "&", "&&")
}

// planMenu numbers the items of the model and makes their labels.
func planMenu(items []platform.TrayMenuItem) *menuPlan {
	p := &menuPlan{pressed: map[uint32]func(){}}
	var next uint32
	var build func(items []platform.TrayMenuItem) []menuEntry
	build = func(items []platform.TrayMenuItem) []menuEntry {
		var out []menuEntry
		for _, it := range items {
			if it.Hidden && !it.Separator {
				continue
			}
			next++
			e := menuEntry{ID: next}
			if it.Separator {
				e.Separator = true
				out = append(out, e)
				continue
			}
			e.Label = escapeLabel(it.Label)
			e.Disabled = it.Disabled
			e.Radio = it.Radio
			e.Checked = (it.Checkable || it.Radio) && it.Checked
			if it.OnPressed != nil && len(it.Children) == 0 {
				p.pressed[e.ID] = it.OnPressed
			}
			e.Children = build(it.Children)
			out = append(out, e)
		}
		return out
	}
	p.Entries = build(items)
	return p
}

// trayIcon is the icon of the app in the notification area (SPEC §25.9). It belongs to the UI thread.
type trayIcon struct {
	d    *Desktop
	spec platform.TraySpec
	icon uintptr
	// iconOwned says the icon was made here and is freed here.
	iconOwned bool
	// inMenu is true while the popup menu is open: a second click does not open another.
	inMenu bool
	closed bool
}

var _ platform.Tray = (*trayIcon)(nil)

// NewTray adds the icon of the app to the notification area with Shell_NotifyIconW: the service window
// is the owner, the callback message is the platform's own, and NOTIFYICON_VERSION_4 is asked for (an
// icon on an older shell keeps the old messages). An app has one tray at a time (TrayDuplicate); an icon
// that cannot be added is TrayRegisterFailed. It must be called on the UI thread.
//
// Flutter: no counterpart (tray_manager)
func (d *Desktop) NewTray(spec platform.TraySpec) (platform.Tray, error) {
	if d.tray != nil {
		return nil, errors.New(tagTrayDuplicate, "the app already has a system tray; an app has one at a time")
	}
	t := &trayIcon{d: d, spec: spec}
	h, owned, err := d.shell.MakeIcon(spec.Icon, false)
	if err != nil {
		return nil, errors.New(tagTrayRegisterFailed, "the icon of the tray could not be made", "reason", err.Error())
	}
	t.icon, t.iconOwned = h, owned
	if err := t.add(); err != nil {
		t.freeIcon()
		return nil, errors.New(tagTrayRegisterFailed, "the shell did not take the icon of the tray", "reason", err.Error())
	}
	d.tray = t
	return t, nil
}

// add sends NIM_ADD for the icon and asks for version 4. A shell that refuses the version is not an
// error: the icon works with the old messages.
func (t *trayIcon) add() error {
	data := iconData{ID: trayIconID, Flags: win32.NIFMessage | win32.NIFIcon | win32.NIFTip | win32.NIFShowTip, Icon: t.icon, Tip: tipOf(t.spec)}
	if err := t.d.shell.Notify(win32.NIMAdd, data); err != nil {
		return err
	}
	if err := t.d.shell.Notify(win32.NIMSetVersion, iconData{ID: trayIconID}); err != nil {
		t.d.log.Debug("the shell does not take version 4 of the notify icon; its old messages are read", "err", err)
	}
	return nil
}

// readd adds the icon again after Explorer restarted.
func (t *trayIcon) readd() {
	if t.closed {
		return
	}
	if err := t.add(); err != nil {
		t.d.log.Warn("the tray icon could not be added again after Explorer restarted", "err", err)
	}
}

func (t *trayIcon) freeIcon() {
	if t.iconOwned && t.icon != 0 {
		t.d.shell.FreeIcon(t.icon)
	}
	t.icon, t.iconOwned = 0, false
}

// Update gives the icon its new model: NIM_MODIFY is sent only for what differs (the icon and the
// tooltip); a new menu is used when the menu is next shown.
//
// Flutter: no counterpart
func (t *trayIcon) Update(spec platform.TraySpec) {
	if t.closed {
		return
	}
	change := diffTray(t.spec, spec)
	t.spec = spec
	var data iconData
	data.ID = trayIconID
	if change&changeIcon != 0 {
		h, owned, err := t.d.shell.MakeIcon(spec.Icon, false)
		if err != nil {
			t.d.log.Warn("the new icon of the tray could not be made; the old one stays", "err", err)
		} else {
			old, oldOwned := t.icon, t.iconOwned
			t.icon, t.iconOwned = h, owned
			data.Flags |= win32.NIFIcon
			data.Icon = h
			defer func() {
				if oldOwned && old != 0 {
					t.d.shell.FreeIcon(old)
				}
			}()
		}
	}
	if change&changeTip != 0 {
		data.Flags |= win32.NIFTip | win32.NIFShowTip
		data.Tip = tipOf(spec)
	}
	if data.Flags == 0 {
		return
	}
	if err := t.d.shell.Notify(win32.NIMModify, data); err != nil {
		t.d.log.Warn("the shell did not take the change of the tray icon", "err", err)
	}
}

// Close deletes the icon (NIM_DELETE) and frees its picture.
//
// Flutter: no counterpart
func (t *trayIcon) Close() {
	if t.closed {
		return
	}
	t.closed = true
	if err := t.d.shell.Notify(win32.NIMDelete, iconData{ID: trayIconID}); err != nil {
		t.d.log.Debug("the tray icon could not be deleted", "err", err)
	}
	t.freeIcon()
	if t.d.tray == t {
		t.d.tray = nil
	}
}

// event handles an event of the icon: NIN_SELECT and NIN_KEYSELECT are the primary action (OnActivate,
// else the menu opens when the tray has one), the middle button the secondary action, and
// WM_CONTEXTMENU opens the menu.
func (t *trayIcon) event(event uint32, at point) {
	switch event {
	case win32.NINSelect, win32.NINKeySelect:
		switch {
		case t.spec.OnActivate != nil:
			t.d.shell.Post(t.spec.OnActivate)
		case len(t.spec.Menu) > 0:
			t.showMenu(at)
		}
	case win32.WMMButtonUp:
		if t.spec.OnSecondaryActivate != nil {
			t.d.shell.Post(t.spec.OnSecondaryActivate)
		}
	case win32.WMContextMenu:
		t.showMenu(at)
	}
}

// showMenu shows the popup menu at the point and posts the function of the item that the user chose.
func (t *trayIcon) showMenu(at point) {
	if t.inMenu || len(t.spec.Menu) == 0 {
		return
	}
	t.inMenu = true
	defer func() { t.inMenu = false }()
	plan := planMenu(t.spec.Menu)
	cmd, err := t.d.shell.ShowMenu(plan, at)
	if err != nil {
		t.d.log.Warn("the menu of the tray icon could not be shown", "err", err)
		return
	}
	if fn := plan.pressed[cmd]; cmd != 0 && fn != nil {
		t.d.shell.Post(fn)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5). They are the tags of package desktop.
const (
	tagTrayDuplicate      = "TrayDuplicate"
	tagTrayRegisterFailed = "TrayRegisterFailed"
)
