//go:build darwin && !ios

package macos

import (
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// The tray (SPEC §29.9): an NSStatusItem with the icon as a template image and the menu of the spec as an NSMenu.
// The plan of the menu and the picture are in traymenu.go.

const (
	// statusItemVariableLength is NSVariableStatusItemLength.
	statusItemVariableLength = -1.0
	// The event masks that make the status item's button send its action on a left and on a right release, and the
	// types of the events (NSEventTypeRightMouseUp is 4).
	eventMaskLeftMouseUp  uint64 = 1 << 2
	eventMaskRightMouseUp uint64 = 1 << 4
	eventTypeRightMouseUp uint64 = 4
	menuItemStateOn       int64  = 1
)

// trayIcon is the icon of the app in the menu bar. Its methods belong to the UI thread.
type trayIcon struct {
	d        *desktop
	spec     platform.TraySpec
	item     objc.ID
	menu     objc.ID
	target   objc.ID
	handle   objcx.Handle
	handlers map[int]func()
	closed   bool
}

var _ platform.Tray = (*trayIcon)(nil)

// trayTargetClass is the class of the target of the status item and of its menu items, registered once for the
// process.
var trayTargetClass = sync.OnceValues(func() (*objcx.RegisteredClass, error) {
	return objcx.RegisterClass(objcx.ClassDef{
		Name: "TrayTarget", Super: "NSObject",
		Methods: []objcx.Method{
			// A chosen menu item: its tag is its number in the plan.
			{Name: "trayItemPressed:", Fn: func(self objc.ID, _ objc.SEL, sender objc.ID) {
				if t, ok := objcx.LookupAs[*trayIcon](objcx.HandleOf(self)); ok && !t.closed {
					id := int(send[int64](sender, "tag"))
					t.d.p.guard(func() { t.pressed(id) })
				}
			}},
			// A click on the icon when it has no menu: the left button is the primary action, the right one (or a
			// Control-click) the secondary.
			{Name: "trayIconClicked:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if t, ok := objcx.LookupAs[*trayIcon](objcx.HandleOf(self)); ok && !t.closed {
					secondary := false
					if ev := t.d.p.currentEvent(); ev != 0 {
						secondary = send[uint64](ev, "type") == eventTypeRightMouseUp || send[uint64](ev, "modifierFlags")&modControl != 0
					}
					t.d.p.guard(func() { t.clicked(secondary) })
				}
			}},
		},
	})
})

// NewTray puts the icon in the menu bar (SPEC §29.9). It fails with TrayDuplicate when the app has a tray, and with
// TrayRegisterFailed when the picture cannot be made or the status bar gives no item. It must be called on the UI
// thread.
//
// Flutter: no counterpart (tray_manager)
func (d *desktop) NewTray(spec platform.TraySpec) (platform.Tray, error) {
	if d.tray != nil {
		return nil, errors.New(tagTrayDuplicate, "the app already has a system tray; an app has one at a time")
	}
	cls, err := trayTargetClass()
	if err != nil {
		return nil, errors.New(tagTrayRegisterFailed, "the class of the tray's target could not be made", "reason", err.Error())
	}
	if err := spec.Icon.Validate(); err != nil {
		return nil, errors.New(tagTrayRegisterFailed, "the icon of the tray is not valid", "reason", err.Error())
	}
	_, _, err = trayIconPNG(spec.Icon)
	if err != nil {
		return nil, err
	}
	t := &trayIcon{d: d, spec: spec}
	t.handle = objcx.NewHandle(t)
	t.target = cls.New(t.handle)
	objcx.WithPool(func() {
		bar := class("NSStatusBar").Send(sel("systemStatusBar"))
		t.item = objcx.Retain(objc.Send[objc.ID](bar, sel("statusItemWithLength:"), statusItemVariableLength))
	})
	if t.item == 0 {
		objcx.Release(t.target)
		objcx.DropHandle(t.handle)
		return nil, errors.New(tagTrayRegisterFailed, "the status bar gave no item for the icon")
	}
	t.apply()
	d.tray = t
	return t, nil
}

// Update gives the icon its new model: the picture, the tip, the menu and the callbacks.
func (t *trayIcon) Update(spec platform.TraySpec) {
	if t.closed {
		return
	}
	t.spec = spec
	t.apply()
}

// apply puts the spec on the status item.
func (t *trayIcon) apply() {
	objcx.WithPool(func() {
		button := send[objc.ID](t.item, "button")
		t.setImage(button)
		if tip := trayTip(t.spec); tip != "" {
			s := objcx.NewString(tip)
			button.Send(sel("setToolTip:"), s)
			objcx.Release(s)
		} else {
			button.Send(sel("setToolTip:"), objc.ID(0))
		}
		plan, handlers := planTrayMenu(t.spec.Menu)
		t.handlers = handlers
		if t.menu != 0 {
			t.item.Send(sel("setMenu:"), objc.ID(0))
			objcx.Release(t.menu)
			t.menu = 0
		}
		if len(plan) > 0 {
			t.menu = t.buildMenu(plan, "")
			t.item.Send(sel("setMenu:"), t.menu)
			return
		}
		button.Send(sel("setTarget:"), t.target)
		button.Send(sel("setAction:"), sel("trayIconClicked:"))
		button.Send(sel("sendActionOn:"), eventMaskLeftMouseUp|eventMaskRightMouseUp)
	})
}

// setImage sets the template image of the button, or the title when the icon has no picture.
func (t *trayIcon) setImage(button objc.ID) {
	data, points, err := trayIconPNG(t.spec.Icon)
	if err != nil || data == nil {
		if err != nil {
			t.d.log.Warn("the picture of the tray icon could not be made", "err", err)
		}
		button.Send(sel("setImage:"), objc.ID(0))
		title := t.spec.Title
		if title == "" {
			title = "•"
		}
		s := objcx.NewString(title)
		button.Send(sel("setTitle:"), s)
		objcx.Release(s)
		return
	}
	ns := class("NSData").Send(sel("dataWithBytes:length:"), &data[0], uint64(len(data)))
	img := class("NSImage").Send(sel("alloc")).Send(sel("initWithData:"), ns)
	if img == 0 {
		button.Send(sel("setImage:"), objc.ID(0))
		return
	}
	img.Send(sel("setSize:"), nsSize{Width: points, Height: points})
	img.Send(sel("setTemplate:"), true)
	button.Send(sel("setImage:"), img)
	button.Send(sel("setTitle:"), objc.ID(0))
	objcx.Release(img)
}

// buildMenu makes the NSMenu of the entries (+1).
func (t *trayIcon) buildMenu(plan []trayEntry, title string) objc.ID {
	ttl := objcx.NewString(title)
	menu := class("NSMenu").Send(sel("alloc")).Send(sel("initWithTitle:"), ttl)
	objcx.Release(ttl)
	menu.Send(sel("setAutoenablesItems:"), false)
	for _, e := range plan {
		if e.Separator {
			menu.Send(sel("addItem:"), class("NSMenuItem").Send(sel("separatorItem")))
			continue
		}
		label, none := objcx.NewString(e.Label), objcx.NewString("")
		item := class("NSMenuItem").Send(sel("alloc")).Send(sel("initWithTitle:action:keyEquivalent:"), label, sel("trayItemPressed:"), none)
		objcx.Release(label)
		objcx.Release(none)
		item.Send(sel("setTarget:"), t.target)
		item.Send(sel("setTag:"), int64(e.ID))
		item.Send(sel("setEnabled:"), !e.Disabled)
		if e.Checked {
			item.Send(sel("setState:"), menuItemStateOn)
		}
		if len(e.Children) > 0 {
			sub := t.buildMenu(e.Children, e.Label)
			item.Send(sel("setSubmenu:"), sub)
			objcx.Release(sub)
		}
		menu.Send(sel("addItem:"), item)
		objcx.Release(item)
	}
	return menu
}

// pressed is a menu item chosen: its function runs.
func (t *trayIcon) pressed(id int) {
	if h := t.handlers[id]; h != nil {
		h()
	}
}

// clicked is a click on the icon that has no menu.
func (t *trayIcon) clicked(secondary bool) {
	switch {
	case secondary && t.spec.OnSecondaryActivate != nil:
		t.spec.OnSecondaryActivate()
	case !secondary && t.spec.OnActivate != nil:
		t.spec.OnActivate()
	}
}

// Close takes the icon out of the menu bar. Closing twice does nothing.
func (t *trayIcon) Close() {
	if t.closed {
		return
	}
	t.closed = true
	objcx.WithPool(func() {
		bar := class("NSStatusBar").Send(sel("systemStatusBar"))
		bar.Send(sel("removeStatusItem:"), t.item)
	})
	if t.menu != 0 {
		objcx.Release(t.menu)
		t.menu = 0
	}
	objcx.Release(t.item)
	objcx.Release(t.target)
	objcx.DropHandle(t.handle)
	t.item, t.target = 0, 0
	if t.d.tray == t {
		t.d.tray = nil
	}
}
