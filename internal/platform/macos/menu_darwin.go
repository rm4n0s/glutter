//go:build darwin && !ios

package macos

import (
	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// This file is the menu bar of macOS (SPEC §29.8): the default menus of an app that has none of its own, and
// the menus that a PlatformMenuBar gives, made into NSMenus. It is a port of FlutterMenuPlugin.mm.

var _ platform.MenuBar = (*Platform)(nil)

// menuBar is the state of the menu bar. It belongs to the UI thread.
type menuBar struct {
	p *Platform
	// handler gets the callbacks of the items of the app's menus.
	handler ui.PlatformMenuHandler
	// target is the instance of MenuTarget that the app's items send their action to.
	target objc.ID
	// delegates are the menu delegates of the menus that the app made (an NSMenu does not keep its delegate),
	// with the handles that let them find their ids.
	delegates []objc.ID
	handles   []objcx.Handle
	// def is the default menu bar; provided are its top-level items, which the items that the platform
	// provides are copied from.
	def      objc.ID
	provided []objc.ID
	// installed says that the main menu of the app is one that this menu bar made.
	installed bool
}

// menuDelegateInfo is what a menu delegate knows: the menu bar and the id of the item whose menu it is.
type menuDelegateInfo struct {
	m  *menuBar
	id int
}

func newMenuBar(p *Platform) *menuBar {
	m := &menuBar{p: p}
	objcx.WithPool(func() {
		m.target = p.s.menuTarget.New(p.handle)
		m.def = objcx.Retain(buildDefaultMenu(processName()))
		items := send[objc.ID](m.def, "itemArray")
		for i := range objcx.ArrayCount(items) {
			m.provided = append(m.provided, objcx.Retain(objcx.ArrayAt(items, i)))
		}
	})
	return m
}

// installDefault makes the default menu bar the main menu of the app.
func (m *menuBar) installDefault() {
	m.dropDelegates()
	m.handler = nil
	m.p.app.Send(sel("setMainMenu:"), m.def)
	m.installed = true
}

// close lets go of everything.
func (m *menuBar) close() {
	if m.installed {
		m.p.app.Send(sel("setMainMenu:"), objc.ID(0))
		m.installed = false
	}
	m.dropDelegates()
	objcx.Release(m.target)
	for _, it := range m.provided {
		objcx.Release(it)
	}
	objcx.Release(m.def)
	m.target, m.def, m.provided = 0, 0, nil
}

func (m *menuBar) dropDelegates() {
	for _, d := range m.delegates {
		objcx.Release(d)
	}
	for _, h := range m.handles {
		objcx.DropHandle(h)
	}
	m.delegates, m.handles = nil, nil
}

// SetMenus replaces the menus of the menu bar with the app's top-level menus; an empty list gives the default
// menu bar back.
//
// Deviation: Flutter's plugin makes an empty menu bar of an empty list (which is what a PlatformMenuBar that
// goes leaves); here the default menu bar returns, so that the app keeps Quit and the Edit menu (SPEC §12.10).
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterMenuPlugin.mm FlutterMenuPlugin setMenus
func (p *Platform) SetMenus(items []ui.PlatformMenuItemData, h ui.PlatformMenuHandler) {
	m := p.menus
	if m == nil {
		return
	}
	if len(items) == 0 {
		if p.accessory {
			m.dropDelegates()
			m.handler = nil
			p.app.Send(sel("setMainMenu:"), objc.ID(0))
			m.installed = false
			return
		}
		m.installDefault()
		return
	}
	objcx.WithPool(func() {
		m.dropDelegates()
		m.handler = h
		menu := class("NSMenu").Send(sel("alloc")).Send(sel("init"))
		for _, d := range items {
			item := m.item(d)
			if item == 0 {
				continue
			}
			menu.Send(sel("addItem:"), item)
			objcx.Release(item)
		}
		p.app.Send(sel("setMainMenu:"), menu)
		m.installed = true
		objcx.Release(menu)
	})
}

// item makes the NSMenuItem of an item of the app's menus (+1), or 0 for a provided item that this system
// has no counterpart for.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterMenuPlugin.mm menuItemFromFlutterRepresentation
func (m *menuBar) item(d ui.PlatformMenuItemData) objc.ID {
	if d.IsDivider {
		return objcx.Retain(class("NSMenuItem").Send(sel("separatorItem")))
	}
	if d.HasProvided {
		item := m.providedItem(d.ProvidedType)
		if item != 0 {
			item.Send(sel("setEnabled:"), d.Enabled)
		}
		return item
	}
	name := processName()
	key, mask := keyEquivalent(d)
	title := objcx.NewString(replaceAppName(d.Label, name))
	keyStr := objcx.NewString(key)
	action := objc.SEL(0)
	if d.ID != 0 {
		action = sel("flutterMenuItemSelected:")
	}
	item := class("NSMenuItem").Send(sel("alloc")).Send(sel("initWithTitle:action:keyEquivalent:"), title, action, keyStr)
	objcx.Release(keyStr)
	if d.HasTooltip {
		tip := objcx.NewString(replaceAppName(d.Tooltip, name))
		item.Send(sel("setToolTip:"), tip)
		objcx.Release(tip)
	}
	if key != "" {
		item.Send(sel("setKeyEquivalentModifierMask:"), mask)
	}
	if d.ID != 0 {
		item.Send(sel("setTag:"), int64(d.ID))
		item.Send(sel("setTarget:"), m.target)
	}
	item.Send(sel("setEnabled:"), d.Enabled)
	if d.HasChildren && len(d.Children) > 0 {
		submenu := class("NSMenu").Send(sel("alloc")).Send(sel("initWithTitle:"), title)
		h := objcx.NewHandle(&menuDelegateInfo{m: m, id: d.ID})
		delegate := m.p.s.menuDelegate.New(h)
		m.handles = append(m.handles, h)
		m.delegates = append(m.delegates, delegate)
		submenu.Send(sel("setDelegate:"), delegate)
		submenu.Send(sel("setAutoenablesItems:"), false)
		for _, c := range d.Children {
			child := m.item(c)
			if child == 0 {
				continue
			}
			submenu.Send(sel("addItem:"), child)
			objcx.Release(child)
		}
		item.Send(sel("setSubmenu:"), submenu)
		objcx.Release(submenu)
	}
	objcx.Release(title)
	return item
}

// providedItem is a copy (+1) of the item of the default menu bar that has the action of the provided type, 0
// when there is none. AppKit does not take an item that is in two menus, so the copy is what goes in the new
// menu.
//
// Flutter: shell/platform/darwin/macos/framework/Source/FlutterMenuPlugin.mm createPlatformProvidedMenu
func (m *menuBar) providedItem(index int) objc.ID {
	name, ok := providedSelector(index)
	if !ok {
		return 0
	}
	selector := sel(name)
	var found objc.ID
	if index == providedServicesIndex {
		if len(m.provided) > 0 {
			found = findItem(send[objc.ID](m.provided[0], "submenu"), selector, 0)
		}
	} else {
		for _, top := range m.provided {
			if found = findItemIn(top, selector); found != 0 {
				break
			}
		}
	}
	if found == 0 {
		return 0
	}
	return send[objc.ID](found, "copy")
}

// findItem finds the item with the action in the menu and its submenus (not owned by the caller).
func findItem(menu objc.ID, selector objc.SEL, _ int) objc.ID {
	if menu == 0 {
		return 0
	}
	items := send[objc.ID](menu, "itemArray")
	for i := range objcx.ArrayCount(items) {
		if found := findItemIn(objcx.ArrayAt(items, i), selector); found != 0 {
			return found
		}
	}
	return 0
}

// findItemIn is findItem for an item: the item itself, or what is in its submenu.
func findItemIn(item objc.ID, selector objc.SEL) objc.ID {
	if send[objc.SEL](item, "action") == selector {
		return item
	}
	if sub := send[objc.ID](item, "submenu"); sub != 0 && send[int64](sub, "numberOfItems") > 0 {
		return findItem(sub, selector, 0)
	}
	return 0
}

// buildDefaultMenu makes the menu bar of an app that has none of its own: the template of Flutter's macOS
// apps, a little smaller. The application menu has About, Services, Hide, Hide Others, Show All and Quit; the
// Edit menu has Undo, Redo, Cut, Copy, Paste, Select All and Speech (the commands go to the first responder,
// the view, which hands them to the focused text field); View has Enter Full Screen; Window has Minimize,
// Zoom and Bring All to Front, and is the menu that AppKit lists the windows in; Help has the search field of
// AppKit (the app has no help book, so its item is disabled). Every item that the platform provides is in it,
// for the items of a PlatformMenuBar to be copied from.
//
// Flutter: packages/flutter_tools/templates/app_shared/macos.tmpl/Runner/Base.lproj/MainMenu.xib
func buildDefaultMenu(name string) objc.ID {
	menu := class("NSMenu").Send(sel("alloc")).Send(sel("init"))
	add := func(into objc.ID, title, action, key string, mask uint64) objc.ID {
		t, k := objcx.NewString(title), objcx.NewString(key)
		act := objc.SEL(0)
		if action != "" {
			act = sel(action)
		}
		item := class("NSMenuItem").Send(sel("alloc")).Send(sel("initWithTitle:action:keyEquivalent:"), t, act, k)
		if mask != 0 {
			item.Send(sel("setKeyEquivalentModifierMask:"), mask)
		}
		into.Send(sel("addItem:"), item)
		objcx.Release(t)
		objcx.Release(k)
		objcx.Release(item)
		return item
	}
	sep := func(into objc.ID) { into.Send(sel("addItem:"), class("NSMenuItem").Send(sel("separatorItem"))) }
	// sub makes the menu of a top-level item or a submenu item.
	sub := func(item objc.ID, title string) objc.ID {
		t := objcx.NewString(title)
		m := class("NSMenu").Send(sel("alloc")).Send(sel("initWithTitle:"), t)
		objcx.Release(t)
		item.Send(sel("setSubmenu:"), m)
		objcx.Release(m)
		return m
	}

	appMenu := sub(add(menu, name, "", "", 0), name)
	add(appMenu, "About "+name, "orderFrontStandardAboutPanel:", "", 0)
	sep(appMenu)
	services := sub(add(appMenu, "Services", "", "", 0), "Services")
	class("NSApplication").Send(sel("sharedApplication")).Send(sel("setServicesMenu:"), services)
	sep(appMenu)
	add(appMenu, "Hide "+name, "hide:", "h", menuMaskCommand)
	add(appMenu, "Hide Others", "hideOtherApplications:", "h", menuMaskCommand|menuMaskOption)
	add(appMenu, "Show All", "unhideAllApplications:", "", 0)
	sep(appMenu)
	add(appMenu, "Quit "+name, "terminate:", "q", menuMaskCommand)

	edit := sub(add(menu, "Edit", "", "", 0), "Edit")
	add(edit, "Undo", "undo:", "z", menuMaskCommand)
	add(edit, "Redo", "redo:", "z", menuMaskCommand|menuMaskShift)
	sep(edit)
	add(edit, "Cut", "cut:", "x", menuMaskCommand)
	add(edit, "Copy", "copy:", "c", menuMaskCommand)
	add(edit, "Paste", "paste:", "v", menuMaskCommand)
	add(edit, "Select All", "selectAll:", "a", menuMaskCommand)
	sep(edit)
	speech := sub(add(edit, "Speech", "", "", 0), "Speech")
	add(speech, "Start Speaking", "startSpeaking:", "", 0)
	add(speech, "Stop Speaking", "stopSpeaking:", "", 0)

	view := sub(add(menu, "View", "", "", 0), "View")
	add(view, "Enter Full Screen", "toggleFullScreen:", "f", menuMaskCommand|menuMaskControl)

	window := sub(add(menu, "Window", "", "", 0), "Window")
	add(window, "Close Window", "performClose:", "w", menuMaskCommand)
	add(window, "Minimize", "performMiniaturize:", "m", menuMaskCommand)
	add(window, "Zoom", "performZoom:", "", 0)
	sep(window)
	add(window, "Bring All to Front", "arrangeInFront:", "", 0)
	class("NSApplication").Send(sel("sharedApplication")).Send(sel("setWindowsMenu:"), window)

	help := sub(add(menu, "Help", "", "", 0), "Help")
	add(help, name+" Help", "", "", 0)
	class("NSApplication").Send(sel("sharedApplication")).Send(sel("setHelpMenu:"), help)
	return objcx.Autorelease(menu)
}

// editMethods are the commands of the Edit menu that AppKit sends to the first responder, which is the view:
// each is the command of the same name for the text field that has the focus.
func editMethods() []objcx.Method {
	command := func(name string) objcx.Method {
		return objcx.Method{Name: name, Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
			if w := windowOf(self); w != nil && !w.closed {
				w.p.guard(func() { w.doCommand(name) })
			}
		}}
	}
	return []objcx.Method{command("cut:"), command("copy:"), command("paste:"), command("selectAll:"), command("undo:"), command("redo:")}
}

// menuClasses are the classes of the menus: the target of the items of the app and the delegate that tells
// when a menu opens and closes.
func menuClasses() (target, delegate objcx.ClassDef) {
	target = objcx.ClassDef{
		Name: "MenuTarget", Super: "NSObject",
		Methods: []objcx.Method{
			{Name: "flutterMenuItemSelected:", Fn: func(self objc.ID, _ objc.SEL, sender objc.ID) {
				p := platformOf(self)
				if p == nil || p.menus == nil || p.menus.handler == nil {
					return
				}
				id := int(send[int64](sender, "tag"))
				p.guard(func() { p.menus.handler.MenuSelected(id) })
			}},
		},
	}
	delegate = objcx.ClassDef{
		Name: "MenuDelegate", Super: "NSObject", Protocols: []string{"NSMenuDelegate"},
		Methods: []objcx.Method{
			{Name: "menuWillOpen:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if info, ok := objcx.LookupAs[*menuDelegateInfo](objcx.HandleOf(self)); ok && info.m.handler != nil {
					info.m.p.guard(func() { info.m.handler.MenuOpened(info.id) })
				}
			}},
			{Name: "menuDidClose:", Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
				if info, ok := objcx.LookupAs[*menuDelegateInfo](objcx.HandleOf(self)); ok && info.m.handler != nil {
					info.m.p.guard(func() { info.m.handler.MenuClosed(info.id) })
				}
			}},
		},
	}
	return target, delegate
}
