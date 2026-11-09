// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"slices"
	"unicode/utf8"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// ShortcutSerialization is the parts of a shortcut that a menu shows: either a
// trigger key or a character, and the modifiers. Dart's nullable booleans of
// the other kind are false here (use Trigger and Character to tell the kinds
// apart).
//
// Deviation: Flutter's toChannelRepresentation, a map for the flutter/menu channel,
// is the unexported menuData here, which fills a ui.PlatformMenuItemData.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization
type ShortcutSerialization struct {
	trigger                   foundation.Opt[services.LogicalKeyboardKey]
	character                 foundation.Opt[string]
	alt, control, meta, shift bool
}

// ShortcutSerializationOpts are the modifiers of a ShortcutSerialization; Shift
// is for the trigger form only, as a character carries its own case.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.modifier
type ShortcutSerializationOpts struct {
	// Alt is whether the Alt key is part of the shortcut.
	Alt bool
	// Control is whether the Control key is part of the shortcut.
	Control bool
	// Meta is whether the Meta key is part of the shortcut.
	Meta bool
	// Shift is whether the Shift key is part of the shortcut.
	Shift bool
}

// NewShortcutSerializationCharacter serializes a shortcut that is a single
// character with modifiers. In debug builds it panics with the tag
// ShortcutSerializationCharacterLength when the character is not one character.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.character
func NewShortcutSerializationCharacter(character string, opts ...ShortcutSerializationOpts) ShortcutSerialization {
	o := firstOpts(opts)
	if foundation.DebugMode && utf8.RuneCountInString(character) != 1 {
		panic(errors.New(tagShortcutSerializationCharacterLength, "the character of a ShortcutSerialization must be one character", "character", character))
	}
	return ShortcutSerialization{character: foundation.Some(character), alt: o.Alt, control: o.Control, meta: o.Meta}
}

// NewShortcutSerializationModifier serializes a shortcut that is a trigger key
// with modifiers. In debug builds it panics with the tag
// ShortcutSerializationModifierTrigger when the trigger is a modifier key.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.modifier
func NewShortcutSerializationModifier(trigger services.LogicalKeyboardKey, opts ...ShortcutSerializationOpts) ShortcutSerialization {
	o := firstOpts(opts)
	if foundation.DebugMode {
		switch trigger {
		case services.LogicalKeyboardKeyAlt, services.LogicalKeyboardKeyAltLeft, services.LogicalKeyboardKeyAltRight,
			services.LogicalKeyboardKeyControl, services.LogicalKeyboardKeyControlLeft, services.LogicalKeyboardKeyControlRight,
			services.LogicalKeyboardKeyMeta, services.LogicalKeyboardKeyMetaLeft, services.LogicalKeyboardKeyMetaRight,
			services.LogicalKeyboardKeyShift, services.LogicalKeyboardKeyShiftLeft, services.LogicalKeyboardKeyShiftRight:
			panic(errors.New(tagShortcutSerializationModifierTrigger, "specifying a modifier key as a trigger is not allowed; use the modifier options instead", "trigger", trigger.String()))
		}
	}
	return ShortcutSerialization{trigger: foundation.Some(trigger), alt: o.Alt, control: o.Control, meta: o.Meta, shift: o.Shift}
}

// Trigger is the trigger key, if the shortcut is a trigger with modifiers.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.trigger
func (s ShortcutSerialization) Trigger() foundation.Opt[services.LogicalKeyboardKey] {
	return s.trigger
}

// Character is the character, if the shortcut is a character with modifiers.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.character
func (s ShortcutSerialization) Character() foundation.Opt[string] { return s.character }

// Alt is whether the Alt key is part of the shortcut.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.alt
func (s ShortcutSerialization) Alt() bool { return s.alt }

// Control is whether the Control key is part of the shortcut.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.control
func (s ShortcutSerialization) Control() bool { return s.control }

// Meta is whether the Meta key is part of the shortcut.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.meta
func (s ShortcutSerialization) Meta() bool { return s.meta }

// Shift is whether the Shift key is part of the shortcut; it is always false
// for a character.
//
// Flutter: widgets/platform_menu_bar.dart ShortcutSerialization.shift
func (s ShortcutSerialization) Shift() bool { return s.shift }

// MenuSerializableShortcut is a ShortcutActivator that a menu can show: a
// SingleActivator or a CharacterActivator, or another that can be reduced to
// one of the two.
//
// Flutter: widgets/platform_menu_bar.dart MenuSerializableShortcut
type MenuSerializableShortcut interface {
	ShortcutActivator
	// SerializeForMenu is the shortcut as a trigger or a character with its
	// modifiers.
	//
	// Flutter: widgets/platform_menu_bar.dart MenuSerializableShortcut.serializeForMenu
	SerializeForMenu() ShortcutSerialization
}

// SerializeForMenu is the trigger and the modifiers of the activator.
//
// Flutter: widgets/shortcuts.dart SingleActivator.serializeForMenu
func (a SingleActivator) SerializeForMenu() ShortcutSerialization {
	return NewShortcutSerializationModifier(a.trigger, ShortcutSerializationOpts{Shift: a.shift, Alt: a.alt, Meta: a.meta, Control: a.control})
}

// SerializeForMenu is the character and the modifiers of the activator.
//
// Flutter: widgets/shortcuts.dart CharacterActivator.serializeForMenu
func (a CharacterActivator) SerializeForMenu() ShortcutSerialization {
	return NewShortcutSerializationCharacter(a.character, ShortcutSerializationOpts{Alt: a.alt, Control: a.control, Meta: a.meta})
}

// fill puts the shortcut in the data of a menu item, as Flutter's toChannelRepresentation does: the
// character or the trigger key id, and the modifiers.
func (s ShortcutSerialization) fill(d *ui.PlatformMenuItemData) {
	mods := 0
	if s.alt {
		mods |= ui.PlatformMenuShortcutModifierAlt
	}
	if s.control {
		mods |= ui.PlatformMenuShortcutModifierControl
	}
	if s.meta {
		mods |= ui.PlatformMenuShortcutModifierMeta
	}
	if c, ok := s.character.Get(); ok {
		d.ShortcutCharacter, d.HasShortcutCharacter = c, true
	} else if t, ok := s.trigger.Get(); ok {
		d.ShortcutTrigger, d.HasShortcutTrigger = t.KeyID(), true
		if s.shift {
			mods |= ui.PlatformMenuShortcutModifierShift
		}
	}
	d.ShortcutModifiers = mods
}

// PlatformMenuDelegate is what the binding uses to put the menus of a PlatformMenuBar in the platform's
// menu bar. DefaultPlatformMenuDelegate does it through the engine.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuDelegate
type PlatformMenuDelegate interface {
	// SetMenus sets the top-level menus of the menu bar, replacing the others.
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformMenuDelegate.setMenus
	SetMenus(topLevelMenus []PlatformMenuItem)
	// ClearMenus removes the menus that were set, which gives the platform's default menu bar back.
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformMenuDelegate.clearMenus
	ClearMenus()
	// DebugLockDelegate says whether the context may use the delegate: there is one active
	// PlatformMenuBar at a time, and a lock keeps a second from taking it. In release builds it
	// is true.
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformMenuDelegate.debugLockDelegate
	DebugLockDelegate(bc BuildContext) bool
	// DebugUnlockDelegate says whether the context may let go of the delegate (it is the one that
	// locked it). In release builds it is true.
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformMenuDelegate.debugUnlockDelegate
	DebugUnlockDelegate(bc BuildContext) bool
}

// MenuItemSerializableIDGenerator gives the id that an item has in the data that goes to the platform.
//
// Flutter: widgets/platform_menu_bar.dart MenuItemSerializableIdGenerator
type MenuItemSerializableIDGenerator func(item PlatformMenuItem) int

// DefaultPlatformMenuDelegate is the delegate of the binding: it gives the items numbers, sends them to
// the platform through the engine (ui.PlatformMenuHost) and runs the callbacks of the item that the user
// chose, opened or closed. On a platform without a menu bar it does nothing.
//
// Deviation: Flutter sends the menus on the flutter/menu platform channel (SPEC §12.10).
//
// Flutter: widgets/platform_menu_bar.dart DefaultPlatformMenuDelegate
type DefaultPlatformMenuDelegate struct {
	b      *WidgetsBinding
	idMap  map[int]PlatformMenuItem
	serial int
	locked BuildContext
}

var _ ui.PlatformMenuHandler = (*DefaultPlatformMenuDelegate)(nil)

// NewDefaultPlatformMenuDelegate makes the delegate of the binding.
//
// Flutter: widgets/platform_menu_bar.dart DefaultPlatformMenuDelegate.new
func NewDefaultPlatformMenuDelegate(b *WidgetsBinding) *DefaultPlatformMenuDelegate {
	return &DefaultPlatformMenuDelegate{b: b, idMap: map[int]PlatformMenuItem{}}
}

// ClearMenus removes the menus.
//
// Flutter: widgets/platform_menu_bar.dart DefaultPlatformMenuDelegate.clearMenus
func (d *DefaultPlatformMenuDelegate) ClearMenus() { d.SetMenus(nil) }

// SetMenus gives the items their ids and sends the menus to the platform.
//
// Flutter: widgets/platform_menu_bar.dart DefaultPlatformMenuDelegate.setMenus
func (d *DefaultPlatformMenuDelegate) SetMenus(topLevelMenus []PlatformMenuItem) {
	clear(d.idMap)
	var data []ui.PlatformMenuItemData
	for _, item := range topLevelMenus {
		data = append(data, item.ToChannelRepresentation(d, d.getID)...)
	}
	d.b.d.SetPlatformMenus(data, d)
}

func (d *DefaultPlatformMenuDelegate) getID(item PlatformMenuItem) int {
	d.serial++
	d.idMap[d.serial] = item
	return d.serial
}

// DebugLockDelegate locks the delegate to the context, in debug builds: it is false when another context
// has it.
//
// Flutter: widgets/platform_menu_bar.dart DefaultPlatformMenuDelegate.debugLockDelegate
func (d *DefaultPlatformMenuDelegate) DebugLockDelegate(bc BuildContext) bool {
	if foundation.DebugMode {
		if d.locked != nil && d.locked != bc {
			return false
		}
		d.locked = bc
	}
	return true
}

// DebugUnlockDelegate unlocks the delegate, in debug builds: it is false when another context has it.
//
// Flutter: widgets/platform_menu_bar.dart DefaultPlatformMenuDelegate.debugUnlockDelegate
func (d *DefaultPlatformMenuDelegate) DebugUnlockDelegate(bc BuildContext) bool {
	if foundation.DebugMode {
		if d.locked != nil && d.locked != bc {
			return false
		}
		d.locked = nil
	}
	return true
}

// item finds the item of an id that the platform sent back. In debug builds an unknown id panics with
// the tag PlatformMenuItemIDUnknown; in release builds it is ignored.
func (d *DefaultPlatformMenuDelegate) item(id int, call string) PlatformMenuItem {
	item, ok := d.idMap[id]
	if !ok && foundation.DebugMode {
		panic(errors.New(tagPlatformMenuItemIDUnknown, "the platform sent a menu event for an item with an id that was not recognized", "call", call, "id", id))
	}
	return item
}

// MenuSelected runs the callback of the item, or invokes its intent in the focused widget.
//
// Flutter: widgets/platform_menu_bar.dart DefaultPlatformMenuDelegate._methodCallHandler (Menu.selectedCallback)
func (d *DefaultPlatformMenuDelegate) MenuSelected(id int) {
	item := d.item(id, "selected")
	if item == nil {
		return
	}
	if item.OnSelected() != nil {
		item.OnSelected()()
	}
	if intent := item.OnSelectedIntent(); intent != nil {
		if focus := d.b.FocusManager().PrimaryFocus(); focus != nil && focus.Context() != nil {
			ActionsMaybeInvoke(focus.Context(), intent)
		}
	}
}

// MenuOpened runs the OnOpen callback of the menu.
//
// Flutter: widgets/platform_menu_bar.dart DefaultPlatformMenuDelegate._methodCallHandler (Menu.opened)
func (d *DefaultPlatformMenuDelegate) MenuOpened(id int) {
	if item := d.item(id, "opened"); item != nil && item.OnOpen() != nil {
		item.OnOpen()()
	}
}

// MenuClosed runs the OnClose callback of the menu.
//
// Flutter: widgets/platform_menu_bar.dart DefaultPlatformMenuDelegate._methodCallHandler (Menu.closed)
func (d *DefaultPlatformMenuDelegate) MenuClosed(id int) {
	if item := d.item(id, "closed"); item != nil && item.OnClose() != nil {
		item.OnClose()()
	}
}

// PlatformMenuBar is a widget that gives the platform the menus of the app, for the platform to show in
// its menu bar (the menu bar at the top of the screen on macOS). It draws nothing: it builds its child, or
// an empty box. There may be only one active at a time; in debug builds a second one panics with the tag
// PlatformMenuBarDuplicated. On a platform without a menu bar it does nothing.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuBar
type PlatformMenuBar struct {
	StatefulWidget
	menus []PlatformMenuItem
	child Widget
}

// PlatformMenuBarOpts are the optional parameters of NewPlatformMenuBar.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuBar.new
type PlatformMenuBarOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Child is the widget below this one; the menu bar draws nothing of its own.
	Child Widget
}

// NewPlatformMenuBar creates the widget with the top-level menus.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuBar.new
func NewPlatformMenuBar(menus []PlatformMenuItem, opts ...PlatformMenuBarOpts) *PlatformMenuBar {
	o := firstOpts(opts)
	return &PlatformMenuBar{StatefulWidget: NewStatefulWidgetBase(WidgetOpts{Key: o.Key}), menus: menus, child: o.Child}
}

// Menus are the top-level menus.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuBar.menus
func (w *PlatformMenuBar) Menus() []PlatformMenuItem { return w.menus }

// Child is the widget below this one.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuBar.child
func (w *PlatformMenuBar) Child() Widget { return w.child }

// CreateState creates the state.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuBar.createState
func (w *PlatformMenuBar) CreateState() State { return &platformMenuBarState{} }

type platformMenuBarState struct {
	StateOf[*PlatformMenuBar]
	descendants []PlatformMenuItem
}

func (s *platformMenuBarState) delegate() PlatformMenuDelegate {
	return WidgetsBindingOf(s.Context()).PlatformMenuDelegate()
}

func (s *platformMenuBarState) InitState() {
	if !s.delegate().DebugLockDelegate(s.Context()) {
		panic(errors.New(tagPlatformMenuBarDuplicated, "more than one active PlatformMenuBar detected: only one active platform-rendered menu bar is allowed at a time"))
	}
	s.delegate().ClearMenus()
	s.updateMenu()
}

func (s *platformMenuBarState) Dispose() {
	if !s.delegate().DebugUnlockDelegate(s.Context()) {
		panic(errors.New(tagPlatformMenuBarUnlockFailed, "tried to unlock the platform menu delegate with a context that does not hold it"))
	}
	s.delegate().ClearMenus()
}

func (s *platformMenuBarState) DidUpdateWidget(old *PlatformMenuBar) {
	var next []PlatformMenuItem
	for _, item := range s.Widget().menus {
		next = append(next, item)
		next = append(next, item.Descendants()...)
	}
	if !slices.Equal(next, s.descendants) {
		s.descendants = next
		s.updateMenu()
	}
}

func (s *platformMenuBarState) updateMenu() { s.delegate().SetMenus(s.Widget().menus) }

func (s *platformMenuBarState) Build(BuildContext) Widget {
	if c := s.Widget().child; c != nil {
		return c
	}
	return NewSizedBox(nil)
}

// PlatformMenuItem is an entry of a platform menu: a plain item, a menu with items in it, a group of items
// between dividers, or an item that the platform provides.
//
// Deviation: Flutter's PlatformMenuItem is a class that its subclasses extend, with default getters; here
// it is an interface, PlatformMenuItemBase is the plain item that NewPlatformMenuItem makes, and the others
// embed it (SPEC §12.10).
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem
type PlatformMenuItem interface {
	// Label is the text of the item.
	Label() string
	// Tooltip is the tooltip of the item, if it has one.
	Tooltip() foundation.Opt[string]
	// Shortcut is the shortcut of the item, nil when it has none.
	Shortcut() MenuSerializableShortcut
	// OnSelected is called when the user chooses the item; nil when there is none.
	OnSelected() func()
	// OnOpen is called when the menu of the item opens (menus only); nil when there is none.
	OnOpen() func()
	// OnClose is called when the menu of the item closes (menus only); nil when there is none.
	OnClose() func()
	// OnSelectedIntent is the intent that is invoked in the focused widget when the user chooses the item;
	// nil when there is none.
	OnSelectedIntent() Intent
	// Descendants are the items below this one, all levels, depth first.
	Descendants() []PlatformMenuItem
	// Members are the items of a group.
	Members() []PlatformMenuItem
	// ToChannelRepresentation is the data that goes to the platform for the item, with the ids that getID
	// gives.
	ToChannelRepresentation(delegate PlatformMenuDelegate, getID MenuItemSerializableIDGenerator) []ui.PlatformMenuItemData
}

// PlatformMenuItemOpts are the optional parameters of NewPlatformMenuItem.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem.new
type PlatformMenuItemOpts struct {
	// Tooltip is the tooltip of the item.
	Tooltip foundation.Opt[string]
	// Shortcut is the shortcut that chooses the item.
	Shortcut MenuSerializableShortcut
	// OnSelected is called when the user chooses the item; an item with neither it nor OnSelectedIntent is
	// disabled.
	OnSelected func()
	// OnSelectedIntent is invoked in the focused widget when the user chooses the item. Only one of
	// OnSelected and OnSelectedIntent may be given.
	OnSelectedIntent Intent
}

// PlatformMenuItemBase is the plain item of a platform menu, and what the other items embed.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem
type PlatformMenuItemBase struct {
	label            string
	tooltip          foundation.Opt[string]
	shortcut         MenuSerializableShortcut
	onSelected       func()
	onSelectedIntent Intent
}

// NewPlatformMenuItem creates a plain item. In debug builds it panics with the tag
// PlatformMenuItemSelectionConflict when both OnSelected and OnSelectedIntent are given.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem.new
func NewPlatformMenuItem(label string, opts ...PlatformMenuItemOpts) *PlatformMenuItemBase {
	o := firstOpts(opts)
	if foundation.DebugMode && o.OnSelected != nil && o.OnSelectedIntent != nil {
		panic(errors.New(tagPlatformMenuItemSelectionConflict, "only one of OnSelected or OnSelectedIntent may be specified", "label", label))
	}
	return &PlatformMenuItemBase{label: label, tooltip: o.Tooltip, shortcut: o.Shortcut, onSelected: o.OnSelected, onSelectedIntent: o.OnSelectedIntent}
}

// Label is the text of the item.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem.label
func (i *PlatformMenuItemBase) Label() string { return i.label }

// Tooltip is the tooltip of the item.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem.tooltip
func (i *PlatformMenuItemBase) Tooltip() foundation.Opt[string] { return i.tooltip }

// Shortcut is the shortcut of the item.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem.shortcut
func (i *PlatformMenuItemBase) Shortcut() MenuSerializableShortcut { return i.shortcut }

// OnSelected is called when the user chooses the item.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem.onSelected
func (i *PlatformMenuItemBase) OnSelected() func() { return i.onSelected }

// OnOpen is nil: only a menu opens.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem.onOpen
func (i *PlatformMenuItemBase) OnOpen() func() { return nil }

// OnClose is nil: only a menu closes.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem.onClose
func (i *PlatformMenuItemBase) OnClose() func() { return nil }

// OnSelectedIntent is the intent that is invoked when the user chooses the item.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem.onSelectedIntent
func (i *PlatformMenuItemBase) OnSelectedIntent() Intent { return i.onSelectedIntent }

// Descendants is empty: a plain item has no items below it.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem.descendants
func (i *PlatformMenuItemBase) Descendants() []PlatformMenuItem { return nil }

// Members is empty: only a group has members.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem.members
func (i *PlatformMenuItemBase) Members() []PlatformMenuItem { return nil }

// ToChannelRepresentation is the data of the item: its id, label, tooltip, whether it is enabled (it has a
// callback or an intent) and its shortcut.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItem.toChannelRepresentation
func (i *PlatformMenuItemBase) ToChannelRepresentation(_ PlatformMenuDelegate, getID MenuItemSerializableIDGenerator) []ui.PlatformMenuItemData {
	d := ui.PlatformMenuItemData{
		ID: getID(i), Label: i.label, Enabled: i.onSelected != nil || i.onSelectedIntent != nil,
	}
	d.Tooltip, d.HasTooltip = i.tooltip.Get()
	if i.shortcut != nil {
		i.shortcut.SerializeForMenu().fill(&d)
	}
	return []ui.PlatformMenuItemData{d}
}

// PlatformMenu is a menu of the menu bar, with items in it. It is disabled when it has none.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenu
type PlatformMenu struct {
	*PlatformMenuItemBase
	onOpen, onClose func()
	menus           []PlatformMenuItem
}

// PlatformMenuOpts are the optional parameters of NewPlatformMenu.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenu.new
type PlatformMenuOpts struct {
	// Tooltip is the tooltip of the menu.
	Tooltip foundation.Opt[string]
	// OnOpen is called when the menu opens.
	OnOpen func()
	// OnClose is called when the menu closes.
	OnClose func()
}

// NewPlatformMenu creates a menu with the label and the items in it.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenu.new
func NewPlatformMenu(label string, menus []PlatformMenuItem, opts ...PlatformMenuOpts) *PlatformMenu {
	o := firstOpts(opts)
	return &PlatformMenu{
		PlatformMenuItemBase: &PlatformMenuItemBase{label: label, tooltip: o.Tooltip},
		onOpen:               o.OnOpen, onClose: o.OnClose, menus: menus,
	}
}

// Menus are the items of the menu.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenu.menus
func (m *PlatformMenu) Menus() []PlatformMenuItem { return m.menus }

// OnOpen is called when the menu opens.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenu.onOpen
func (m *PlatformMenu) OnOpen() func() { return m.onOpen }

// OnClose is called when the menu closes.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenu.onClose
func (m *PlatformMenu) OnClose() func() { return m.onClose }

// Descendants are the items below the menu, all levels, depth first.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenu.descendants
func (m *PlatformMenu) Descendants() []PlatformMenuItem { return platformMenuDescendants(m) }

func platformMenuDescendants(m *PlatformMenu) []PlatformMenuItem {
	var out []PlatformMenuItem
	for _, child := range m.menus {
		out = append(out, child)
		out = append(out, child.Descendants()...)
	}
	return out
}

// ToChannelRepresentation is the data of the menu with its children. Dividers that are first, last or
// next to another are dropped, because groups and plain items may both add them.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenu.toChannelRepresentation
func (m *PlatformMenu) ToChannelRepresentation(delegate PlatformMenuDelegate, getID MenuItemSerializableIDGenerator) []ui.PlatformMenuItemData {
	var children []ui.PlatformMenuItemData
	for _, child := range m.menus {
		children = append(children, child.ToChannelRepresentation(delegate, getID)...)
	}
	// Strip the leading dividers and the duplicate ones, then the last one.
	kept := children[:0:0]
	for _, c := range children {
		if len(kept) == 0 && c.IsDivider {
			continue
		}
		if len(kept) > 0 && kept[len(kept)-1].IsDivider && c.IsDivider {
			continue
		}
		kept = append(kept, c)
	}
	if n := len(kept); n > 0 && kept[n-1].IsDivider {
		kept = kept[:n-1]
	}
	d := ui.PlatformMenuItemData{
		ID: getID(m), Label: m.label, Enabled: len(m.menus) > 0, Children: kept, HasChildren: true,
	}
	d.Tooltip, d.HasTooltip = m.tooltip.Get()
	return []ui.PlatformMenuItemData{d}
}

// PlatformMenuItemGroup is a group of items that the platform shows between dividers.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItemGroup
type PlatformMenuItemGroup struct {
	*PlatformMenuItemBase
	members []PlatformMenuItem
}

// NewPlatformMenuItemGroup creates a group of items. In debug builds sending a group with no
// members to the platform panics with the tag PlatformMenuItemGroupEmpty.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItemGroup.new
func NewPlatformMenuItemGroup(members []PlatformMenuItem) *PlatformMenuItemGroup {
	return &PlatformMenuItemGroup{PlatformMenuItemBase: &PlatformMenuItemBase{}, members: members}
}

// Members are the items of the group.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItemGroup.members
func (g *PlatformMenuItemGroup) Members() []PlatformMenuItem { return g.members }

// ToChannelRepresentation is a divider, the items of the group and a divider.
//
// Flutter: widgets/platform_menu_bar.dart PlatformMenuItemGroup.toChannelRepresentation
func (g *PlatformMenuItemGroup) ToChannelRepresentation(delegate PlatformMenuDelegate, getID MenuItemSerializableIDGenerator) []ui.PlatformMenuItemData {
	if foundation.DebugMode && len(g.members) == 0 {
		panic(errors.New(tagPlatformMenuItemGroupEmpty, "there must be at least one member in a PlatformMenuItemGroup"))
	}
	out := []ui.PlatformMenuItemData{{ID: getID(g), IsDivider: true}}
	for _, item := range g.members {
		out = append(out, item.ToChannelRepresentation(delegate, getID)...)
	}
	return append(out, ui.PlatformMenuItemData{ID: getID(g), IsDivider: true})
}

// PlatformProvidedMenuItemType is a menu item that the platform provides, with its own title, action and
// shortcut. The zero value is PlatformProvidedMenuItemTypeUnspecified, which no item has.
//
// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType
type PlatformProvidedMenuItemType int

// The values of PlatformProvidedMenuItemType, in Flutter's order after the zero value (the index that goes
// to the platform is the value minus one).
const (
	// PlatformProvidedMenuItemTypeUnspecified is the zero value, which no item has.
	PlatformProvidedMenuItemTypeUnspecified PlatformProvidedMenuItemType = iota
	// PlatformProvidedMenuItemTypeAbout shows the standard About panel (macOS).
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType.about
	PlatformProvidedMenuItemTypeAbout
	// PlatformProvidedMenuItemTypeQuit quits the app (macOS).
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType.quit
	PlatformProvidedMenuItemTypeQuit
	// PlatformProvidedMenuItemTypeServicesSubmenu is the Services submenu (macOS).
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType.servicesSubmenu
	PlatformProvidedMenuItemTypeServicesSubmenu
	// PlatformProvidedMenuItemTypeHide hides the app (macOS).
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType.hide
	PlatformProvidedMenuItemTypeHide
	// PlatformProvidedMenuItemTypeHideOtherApplications hides the other apps (macOS).
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType.hideOtherApplications
	PlatformProvidedMenuItemTypeHideOtherApplications
	// PlatformProvidedMenuItemTypeShowAllApplications shows all the apps again (macOS).
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType.showAllApplications
	PlatformProvidedMenuItemTypeShowAllApplications
	// PlatformProvidedMenuItemTypeStartSpeaking starts speaking the selected text (macOS).
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType.startSpeaking
	PlatformProvidedMenuItemTypeStartSpeaking
	// PlatformProvidedMenuItemTypeStopSpeaking stops speaking (macOS).
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType.stopSpeaking
	PlatformProvidedMenuItemTypeStopSpeaking
	// PlatformProvidedMenuItemTypeToggleFullScreen enters or leaves full screen (macOS).
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType.toggleFullScreen
	PlatformProvidedMenuItemTypeToggleFullScreen
	// PlatformProvidedMenuItemTypeMinimizeWindow minimizes the window (macOS).
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType.minimizeWindow
	PlatformProvidedMenuItemTypeMinimizeWindow
	// PlatformProvidedMenuItemTypeZoomWindow zooms the window (macOS).
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType.zoomWindow
	PlatformProvidedMenuItemTypeZoomWindow
	// PlatformProvidedMenuItemTypeArrangeWindowsInFront brings all the windows to the front (macOS).
	//
	// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItemType.arrangeWindowsInFront
	PlatformProvidedMenuItemTypeArrangeWindowsInFront
)

// PlatformProvidedMenuItem is a menu item that the platform provides, with its own title, action and
// shortcut: About, Quit, Hide and the others of PlatformProvidedMenuItemType. Only macOS has them.
//
// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItem
type PlatformProvidedMenuItem struct {
	*PlatformMenuItemBase
	typ     PlatformProvidedMenuItemType
	enabled bool
}

// PlatformProvidedMenuItemOpts are the optional parameters of NewPlatformProvidedMenuItem.
//
// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItem.new
type PlatformProvidedMenuItemOpts struct {
	// Enabled says the user can choose the item; the default is true.
	Enabled foundation.Opt[bool]
}

// NewPlatformProvidedMenuItem creates the item of the type.
//
// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItem.new
func NewPlatformProvidedMenuItem(typ PlatformProvidedMenuItemType, opts ...PlatformProvidedMenuItemOpts) *PlatformProvidedMenuItem {
	o := firstOpts(opts)
	return &PlatformProvidedMenuItem{PlatformMenuItemBase: &PlatformMenuItemBase{}, typ: typ, enabled: o.Enabled.OrElse(true)}
}

// Type is the kind of the item.
//
// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItem.type
func (i *PlatformProvidedMenuItem) Type() PlatformProvidedMenuItemType { return i.typ }

// Enabled says whether the user can choose the item.
//
// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItem.enabled
func (i *PlatformProvidedMenuItem) Enabled() bool { return i.enabled }

// PlatformProvidedMenuItemHasMenu says whether the platform that the app runs on provides the item: macOS
// provides every type, the other platforms none.
//
// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItem.hasMenu
func PlatformProvidedMenuItemHasMenu(menu PlatformProvidedMenuItemType) bool {
	if foundation.DefaultTargetPlatform() != foundation.TargetPlatformMacOS {
		return false
	}
	return menu >= PlatformProvidedMenuItemTypeAbout && menu <= PlatformProvidedMenuItemTypeArrangeWindowsInFront
}

// ToChannelRepresentation is the data of the item: its id, whether it is enabled and its type. In debug
// builds it panics with the tag PlatformProvidedMenuItemUnavailable when the platform does not provide
// the type.
//
// Flutter: widgets/platform_menu_bar.dart PlatformProvidedMenuItem.toChannelRepresentation
func (i *PlatformProvidedMenuItem) ToChannelRepresentation(_ PlatformMenuDelegate, getID MenuItemSerializableIDGenerator) []ui.PlatformMenuItemData {
	if foundation.DebugMode && !PlatformProvidedMenuItemHasMenu(i.typ) {
		panic(errors.New(tagPlatformProvidedMenuItemUnavailable, "this platform has no platform provided menu for the type; call PlatformProvidedMenuItemHasMenu to find out before making one",
			"platform", foundation.DefaultTargetPlatform().String(), "type", int(i.typ)))
	}
	return []ui.PlatformMenuItemData{{ID: getID(i), Enabled: i.enabled, HasProvided: true, ProvidedType: int(i.typ) - 1}}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagPlatformMenuBarDuplicated            = "PlatformMenuBarDuplicated"
	tagPlatformMenuBarUnlockFailed          = "PlatformMenuBarUnlockFailed"
	tagPlatformMenuItemIDUnknown            = "PlatformMenuItemIDUnknown"
	tagPlatformMenuItemSelectionConflict    = "PlatformMenuItemSelectionConflict"
	tagPlatformMenuItemGroupEmpty           = "PlatformMenuItemGroupEmpty"
	tagPlatformProvidedMenuItemUnavailable  = "PlatformProvidedMenuItemUnavailable"
	tagShortcutSerializationCharacterLength = "ShortcutSerializationCharacterLength"
	tagShortcutSerializationModifierTrigger = "ShortcutSerializationModifierTrigger"
)
