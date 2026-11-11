package widgets_test

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are in the spirit of packages/flutter/test/widgets/platform_menu_bar_test.dart: the menus go
// to the platform in the form of the flutter/menu channel (ui.PlatformMenuItemData), and the callbacks that
// the platform sends come back to the items.

func menuApp2(ts *gluttertest.Tester, menus []PlatformMenuItem) {
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: plainEditableStyle(), home: NewPlatformMenuBar(menus, PlatformMenuBarOpts{Child: ltrText("home")})}))
}

func asMacOS(t *testing.T) {
	t.Helper()
	skipUnlessDebug(t) // the platform override is a debug-only feature
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformMacOS))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
}

func TestPlatformMenuBarSendsTheMenusToThePlatform(t *testing.T) {
	asMacOS(t)
	ts := gluttertest.New(t)
	saved := 0
	menus := []PlatformMenuItem{
		NewPlatformMenu("File", []PlatformMenuItem{
			NewPlatformMenuItem("New", PlatformMenuItemOpts{OnSelected: func() {}, Shortcut: NewCharacterActivator("n", CharacterActivatorOpts{Meta: true}), Tooltip: foundation.Some("Make a new file")}),
			NewPlatformMenuItem("Save", PlatformMenuItemOpts{OnSelected: func() { saved++ }, Shortcut: NewSingleActivator(services.LogicalKeyboardKeyKeyS, SingleActivatorOpts{Meta: true, Shift: true})}),
			NewPlatformMenuItem("Disabled"),
		}),
		NewPlatformMenu("Empty", nil),
	}
	menuApp2(ts, menus)
	got := ts.PlatformMenus()
	if len(got) != 2 {
		t.Fatalf("%d top-level items: %+v", len(got), got)
	}
	file, empty := got[0], got[1]
	if file.Label != "File" || !file.HasChildren || !file.Enabled || len(file.Children) != 3 {
		t.Fatalf("File = %+v", file)
	}
	if empty.Label != "Empty" || empty.Enabled || !empty.HasChildren || len(empty.Children) != 0 {
		t.Errorf("Empty = %+v", empty)
	}
	n, s, d := file.Children[0], file.Children[1], file.Children[2]
	if !n.Enabled || !n.HasTooltip || n.Tooltip != "Make a new file" || !n.HasShortcutCharacter || n.ShortcutCharacter != "n" || n.ShortcutModifiers != ui.PlatformMenuShortcutModifierMeta {
		t.Errorf("New = %+v", n)
	}
	if !s.HasShortcutTrigger || s.ShortcutTrigger != services.LogicalKeyboardKeyKeyS.KeyID() ||
		s.ShortcutModifiers != ui.PlatformMenuShortcutModifierMeta|ui.PlatformMenuShortcutModifierShift {
		t.Errorf("Save = %+v", s)
	}
	if d.Enabled || d.HasShortcutCharacter || d.HasShortcutTrigger {
		t.Errorf("Disabled = %+v", d)
	}
	ids := map[int]bool{}
	var walk func([]ui.PlatformMenuItemData)
	walk = func(items []ui.PlatformMenuItemData) {
		for _, it := range items {
			if ids[it.ID] {
				t.Errorf("the id %d is used twice", it.ID)
			}
			ids[it.ID] = true
			walk(it.Children)
		}
	}
	walk(got)
	ts.SelectPlatformMenuItem(s.ID)
	if saved != 1 {
		t.Errorf("Save ran %d times", saved)
	}
}

func TestPlatformMenuDividersAreTidied(t *testing.T) {
	asMacOS(t)
	ts := gluttertest.New(t)
	item := func(l string) PlatformMenuItem {
		return NewPlatformMenuItem(l, PlatformMenuItemOpts{OnSelected: func() {}})
	}
	menuApp2(ts, []PlatformMenuItem{NewPlatformMenu("M", []PlatformMenuItem{
		NewPlatformMenuItemGroup([]PlatformMenuItem{item("a")}),
		NewPlatformMenuItemGroup([]PlatformMenuItem{item("b"), item("c")}),
		item("d"),
		NewPlatformMenuItemGroup([]PlatformMenuItem{item("e")}),
	})})
	var labels []string
	for _, c := range ts.PlatformMenus()[0].Children {
		if c.IsDivider {
			labels = append(labels, "-")
		} else {
			labels = append(labels, c.Label)
		}
	}
	// A leading divider, a repeated one and a last one are dropped.
	want := []string{"a", "-", "b", "c", "-", "d", "-", "e"}
	if !reflect.DeepEqual(labels, want) {
		t.Errorf("children = %v, want %v", labels, want)
	}
}

func TestPlatformProvidedMenuItems(t *testing.T) {
	asMacOS(t)
	ts := gluttertest.New(t)
	menuApp2(ts, []PlatformMenuItem{NewPlatformMenu("App", []PlatformMenuItem{
		NewPlatformProvidedMenuItem(PlatformProvidedMenuItemTypeAbout),
		NewPlatformProvidedMenuItem(PlatformProvidedMenuItemTypeQuit, PlatformProvidedMenuItemOpts{Enabled: foundation.Some(false)}),
		NewPlatformProvidedMenuItem(PlatformProvidedMenuItemTypeArrangeWindowsInFront),
	})})
	c := ts.PlatformMenus()[0].Children
	if len(c) != 3 {
		t.Fatalf("children = %+v", c)
	}
	for i, want := range []struct {
		typ     int
		enabled bool
	}{{0, true}, {1, false}, {11, true}} {
		if !c[i].HasProvided || c[i].ProvidedType != want.typ || c[i].Enabled != want.enabled {
			t.Errorf("item %d = %+v, want type %d enabled %v", i, c[i], want.typ, want.enabled)
		}
	}
	for typ := PlatformProvidedMenuItemTypeAbout; typ <= PlatformProvidedMenuItemTypeArrangeWindowsInFront; typ++ {
		if !PlatformProvidedMenuItemHasMenu(typ) {
			t.Errorf("macOS has no menu for the type %d", typ)
		}
	}
	if PlatformProvidedMenuItemHasMenu(PlatformProvidedMenuItemTypeUnspecified) {
		t.Error("the unspecified type has a menu")
	}
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformLinux))
	if PlatformProvidedMenuItemHasMenu(PlatformProvidedMenuItemTypeAbout) {
		t.Error("Linux has a provided menu")
	}
}

type menuIntent struct{ IntentBase }

func TestPlatformMenuCallbacksIntentsAndOpenClose(t *testing.T) {
	asMacOS(t)
	ts := gluttertest.New(t)
	var opened, closed, selected, invoked int
	menus := []PlatformMenuItem{
		NewPlatformMenu("M", []PlatformMenuItem{
			NewPlatformMenuItem("callback", PlatformMenuItemOpts{OnSelected: func() { selected++ }}),
			NewPlatformMenuItem("intent", PlatformMenuItemOpts{OnSelectedIntent: &menuIntent{}}),
		}, PlatformMenuOpts{OnOpen: func() { opened++ }, OnClose: func() { closed++ }}),
	}
	focus := NewFocusNode()
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{
		textStyle: plainEditableStyle(),
		home: NewActions(ActionMap{reflect.TypeFor[*menuIntent](): NewCallbackAction(func(*menuIntent) any { invoked++; return nil })},
			NewPlatformMenuBar(menus, PlatformMenuBarOpts{Child: NewFocus(ltrText("home"), FocusOpts{FocusNode: focus, Autofocus: true})})),
	}))
	ts.Pump()
	m := ts.PlatformMenus()[0]
	ts.OpenPlatformMenu(m.ID)
	ts.ClosePlatformMenu(m.ID)
	cb, _ := gluttertest.FindPlatformMenuItem(ts.PlatformMenus(), "callback")
	in, _ := gluttertest.FindPlatformMenuItem(ts.PlatformMenus(), "intent")
	ts.SelectPlatformMenuItem(cb.ID)
	ts.SelectPlatformMenuItem(in.ID)
	if opened != 1 || closed != 1 || selected != 1 || invoked != 1 {
		t.Errorf("opened %d closed %d selected %d invoked %d, want 1 each", opened, closed, selected, invoked)
	}
	if !in.Enabled {
		t.Error("an item with an intent is disabled")
	}
	requireTag(t, "PlatformMenuItemIDUnknown", func() { ts.SelectPlatformMenuItem(9999) })
}

func TestPlatformMenuBarUpdatesOnlyWhenTheItemsChange(t *testing.T) {
	asMacOS(t)
	ts := gluttertest.New(t)
	a := NewPlatformMenuItem("a", PlatformMenuItemOpts{OnSelected: func() {}})
	b := NewPlatformMenuItem("b", PlatformMenuItemOpts{OnSelected: func() {}})
	menu := NewPlatformMenu("M", []PlatformMenuItem{a})
	menuApp2(ts, []PlatformMenuItem{menu})
	// The first update of a bar sends the menus again, as Flutter's does: the items that it kept for the
	// comparison start empty.
	menuApp2(ts, []PlatformMenuItem{menu})
	sets := ts.PlatformMenuSets()
	menuApp2(ts, []PlatformMenuItem{menu}) // the same items: nothing is sent
	if got := ts.PlatformMenuSets(); got != sets {
		t.Errorf("an unchanged bar sent the menus again (%d to %d calls)", sets, got)
	}
	menuApp2(ts, []PlatformMenuItem{NewPlatformMenu("M", []PlatformMenuItem{a, b})})
	if got := ts.PlatformMenuSets(); got != sets+1 {
		t.Errorf("a changed bar sent %d times, want %d", got, sets+1)
	}
	if m := ts.PlatformMenus(); len(m) != 1 || len(m[0].Children) != 2 {
		t.Errorf("menus = %+v", m)
	}
}

func TestPlatformMenuBarRemovesItsMenusWhenItGoes(t *testing.T) {
	asMacOS(t)
	ts := gluttertest.New(t)
	menuApp2(ts, []PlatformMenuItem{NewPlatformMenu("M", []PlatformMenuItem{NewPlatformMenuItem("a")})})
	if len(ts.PlatformMenus()) != 1 {
		t.Fatalf("menus = %+v", ts.PlatformMenus())
	}
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: plainEditableStyle(), home: ltrText("no bar")}))
	if m := ts.PlatformMenus(); len(m) != 0 {
		t.Errorf("the menus stay after the bar is gone: %+v", m)
	}
}

func TestOnlyOnePlatformMenuBarIsActive(t *testing.T) {
	asMacOS(t)
	ts := gluttertest.New(t)
	bar := func() Widget { return NewPlatformMenuBar([]PlatformMenuItem{NewPlatformMenu("M", nil)}) }
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: plainEditableStyle(), home: NewColumn([]Widget{bar(), bar()})}))
	// The build of the second bar panicked, and the binding reported it.
	requireErrorTag(t, ts.TakeError(), "PlatformMenuBarDuplicated")
}

// refusingDelegate is a delegate that lets a bar lock it and then refuses to be unlocked.
type refusingDelegate struct{ sets int }

func (d *refusingDelegate) SetMenus([]PlatformMenuItem)           { d.sets++ }
func (d *refusingDelegate) ClearMenus()                           {}
func (d *refusingDelegate) DebugLockDelegate(BuildContext) bool   { return true }
func (d *refusingDelegate) DebugUnlockDelegate(BuildContext) bool { return false }

func TestPlatformMenuBarThatCannotUnlockTheDelegateReportsIt(t *testing.T) {
	ts := gluttertest.New(t)
	d := &refusingDelegate{}
	ts.Binding.SetPlatformMenuDelegate(d)
	menuApp2(ts, []PlatformMenuItem{NewPlatformMenu("M", nil)})
	if d.sets != 1 {
		t.Errorf("the bar set the menus %d times, want 1", d.sets)
	}
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{textStyle: plainEditableStyle(), home: ltrText("no bar")}))
	requireErrorTag(t, ts.TakeError(), "PlatformMenuBarUnlockFailed")
}

type fakeContext struct{ BuildContext }

func TestPlatformMenuDelegateLocksToOneContext(t *testing.T) {
	asMacOS(t)
	ts := gluttertest.New(t)
	d := ts.Binding.PlatformMenuDelegate()
	c1, c2 := &fakeContext{}, &fakeContext{}
	if !d.DebugLockDelegate(c1) || !d.DebugLockDelegate(c1) || d.DebugLockDelegate(c2) {
		t.Error("the delegate was not locked to the first context")
	}
	if d.DebugUnlockDelegate(c2) || !d.DebugUnlockDelegate(c1) || !d.DebugLockDelegate(c2) {
		t.Error("the delegate was unlocked by the wrong context, or not unlocked by the right one")
	}
}

func TestPlatformMenuErrorsHaveTheirTags(t *testing.T) {
	asMacOS(t)
	requireTag(t, "PlatformMenuItemSelectionConflict", func() {
		NewPlatformMenuItem("x", PlatformMenuItemOpts{OnSelected: func() {}, OnSelectedIntent: &menuIntent{}})
	})
	ts := gluttertest.New(t)
	d := ts.Binding.PlatformMenuDelegate()
	requireTag(t, "PlatformMenuItemGroupEmpty", func() { d.SetMenus([]PlatformMenuItem{NewPlatformMenuItemGroup(nil)}) })
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformLinux))
	requireTag(t, "PlatformProvidedMenuItemUnavailable", func() {
		d.SetMenus([]PlatformMenuItem{NewPlatformProvidedMenuItem(PlatformProvidedMenuItemTypeAbout)})
	})
}

func TestPlatformMenuBarOnAPlatformWithoutAMenuBar(t *testing.T) {
	asMacOS(t)
	ts := gluttertest.New(t)
	ts.NoPlatformMenuBar()
	menuApp2(ts, []PlatformMenuItem{NewPlatformMenu("M", []PlatformMenuItem{NewPlatformMenuItem("a")})})
	if m := ts.PlatformMenus(); len(m) != 0 {
		t.Errorf("a platform without a menu bar has menus: %+v", m)
	}
	if !ts.Exists(gluttertest.ByText("home")) {
		t.Error("the bar's child is not built")
	}
}
