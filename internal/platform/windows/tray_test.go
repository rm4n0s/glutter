package windows

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

func item(label string, opts ...func(*platform.TrayMenuItem)) platform.TrayMenuItem {
	it := platform.TrayMenuItem{Label: label}
	for _, o := range opts {
		o(&it)
	}
	return it
}

func sep() platform.TrayMenuItem { return platform.TrayMenuItem{Separator: true} }

func TestPlanMenuNumbersItemsDepthFirst(t *testing.T) {
	var pressed []string
	press := func(name string) func() { return func() { pressed = append(pressed, name) } }
	items := []platform.TrayMenuItem{
		item("Open", func(i *platform.TrayMenuItem) { i.OnPressed = press("open") }),
		sep(),
		item("More", func(i *platform.TrayMenuItem) {
			i.Children = []platform.TrayMenuItem{
				item("A", func(i *platform.TrayMenuItem) { i.OnPressed = press("a") }),
				sep(),
				item("B", func(i *platform.TrayMenuItem) {
					i.Children = []platform.TrayMenuItem{item("B1", func(i *platform.TrayMenuItem) { i.OnPressed = press("b1") })}
				}),
			}
			i.OnPressed = press("never") // a submenu has no command
		}),
		item("Hidden", func(i *platform.TrayMenuItem) { i.Hidden = true; i.OnPressed = press("hidden") }),
		item("Quit", func(i *platform.TrayMenuItem) { i.OnPressed = press("quit") }),
	}
	plan := planMenu(items)
	var flat []string
	var walk func(es []menuEntry, depth int)
	walk = func(es []menuEntry, depth int) {
		for _, e := range es {
			name := e.Label
			if e.Separator {
				name = "-"
			}
			flat = append(flat, strings.Repeat(">", depth)+name+"#"+string(rune('0'+e.ID)))
			walk(e.Children, depth+1)
		}
	}
	walk(plan.Entries, 0)
	want := []string{"Open#1", "-#2", "More#3", ">A#4", ">-#5", ">B#6", ">>B1#7", "Quit#8"}
	if !reflect.DeepEqual(flat, want) {
		t.Errorf("the numbered menu = %v, want %v", flat, want)
	}
	for id, name := range map[uint32]string{1: "open", 4: "a", 7: "b1", 8: "quit"} {
		plan.pressed[id]()
		if pressed[len(pressed)-1] != name {
			t.Errorf("command %d called %q, want %q", id, pressed[len(pressed)-1], name)
		}
	}
	for _, id := range []uint32{2, 3, 5, 6, 9} {
		if plan.pressed[id] != nil {
			t.Errorf("command %d has a function", id)
		}
	}
}

func TestPlanMenuFlagsAndLabels(t *testing.T) {
	tests := []struct {
		name string
		in   platform.TrayMenuItem
		want menuEntry
	}{
		{"an ampersand is doubled", item("Save && Quit"), menuEntry{ID: 1, Label: "Save &&&& Quit"}},
		{"a single ampersand is doubled", item("R&D"), menuEntry{ID: 1, Label: "R&&D"}},
		{"no ampersand", item("Plain"), menuEntry{ID: 1, Label: "Plain"}},
		{"a NUL is dropped", item("a\x00b"), menuEntry{ID: 1, Label: "ab"}},
		{"disabled", item("x", func(i *platform.TrayMenuItem) { i.Disabled = true }), menuEntry{ID: 1, Label: "x", Disabled: true}},
		{"checked and checkable", item("x", func(i *platform.TrayMenuItem) { i.Checkable, i.Checked = true, true }), menuEntry{ID: 1, Label: "x", Checked: true}},
		{"checked without checkable shows nothing", item("x", func(i *platform.TrayMenuItem) { i.Checked = true }), menuEntry{ID: 1, Label: "x"}},
		{"radio", item("x", func(i *platform.TrayMenuItem) { i.Radio, i.Checked = true, true }), menuEntry{ID: 1, Label: "x", Radio: true, Checked: true}},
		{"a separator", sep(), menuEntry{ID: 1, Separator: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := planMenu([]platform.TrayMenuItem{tt.in}).Entries
			if len(got) != 1 || !reflect.DeepEqual(got[0], tt.want) {
				t.Errorf("planMenu = %+v, want [%+v]", got, tt.want)
			}
		})
	}
}

func rgba(size int, v byte) ui.WindowIcon {
	pix := make([]byte, size*size*4)
	for i := range pix {
		pix[i] = v
	}
	return ui.WindowIcon{Images: []ui.WindowIconImage{{Size: size, Scale: 1, RGBA: pix}}}
}

func TestTipOfAndDiffTray(t *testing.T) {
	base := platform.TraySpec{Tooltip: "Glutter", Icon: rgba(16, 1), Menu: []platform.TrayMenuItem{item("A")}}
	tests := []struct {
		name string
		cur  func(*platform.TraySpec)
		want trayChange
	}{
		{"nothing", func(*platform.TraySpec) {}, 0},
		{"only the functions", func(s *platform.TraySpec) { s.OnActivate = func() {} }, 0},
		{"the tooltip", func(s *platform.TraySpec) { s.Tooltip = "Other" }, changeTip},
		{"the text under the tooltip", func(s *platform.TraySpec) { s.TooltipBody = "more" }, changeTip},
		{"the icon pixels", func(s *platform.TraySpec) { s.Icon = rgba(16, 2) }, changeIcon},
		{"the icon size", func(s *platform.TraySpec) { s.Icon = rgba(24, 1) }, changeIcon},
		{"the icon name", func(s *platform.TraySpec) { s.Icon = ui.WindowIcon{Name: "x"} }, changeIcon},
		{"a label", func(s *platform.TraySpec) { s.Menu = []platform.TrayMenuItem{item("B")} }, changeMenu},
		{"a new item", func(s *platform.TraySpec) { s.Menu = append(s.Menu, sep()) }, changeMenu},
		{"a check", func(s *platform.TraySpec) {
			s.Menu = []platform.TrayMenuItem{item("A", func(i *platform.TrayMenuItem) { i.Checked = true })}
		}, changeMenu},
		{"a submenu", func(s *platform.TraySpec) {
			s.Menu = []platform.TrayMenuItem{item("A", func(i *platform.TrayMenuItem) { i.Children = []platform.TrayMenuItem{item("x")} })}
		}, changeMenu},
		{"all", func(s *platform.TraySpec) { s.Tooltip = "Z"; s.Icon = rgba(16, 9); s.Menu = nil }, changeIcon | changeTip | changeMenu},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cur := base
			cur.Menu = append([]platform.TrayMenuItem(nil), base.Menu...)
			tt.cur(&cur)
			if got := diffTray(base, cur); got != tt.want {
				t.Errorf("diffTray = %b, want %b", got, tt.want)
			}
		})
	}
	if got := tipOf(platform.TraySpec{Tooltip: "Title", TooltipBody: "Body"}); got != "Title\nBody" {
		t.Errorf("tip with a body = %q", got)
	}
	if got := tipOf(platform.TraySpec{TooltipBody: "Body"}); got != "Body" {
		t.Errorf("tip with only a body = %q", got)
	}
	if got := tipOf(platform.TraySpec{Tooltip: strings.Repeat("x", 200)}); len(got) != 126+len("…") {
		t.Errorf("a tip of 200 characters is %d bytes, want 126 characters and an ellipsis", len(got))
	}
}

func TestNewTrayAddsTheIconAndAsksForVersion4(t *testing.T) {
	d, sh := newTestDesktop()
	spec := platform.TraySpec{Tooltip: "Tip", Icon: rgba(16, 1)}
	tr, err := d.NewTray(spec)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sh.commands(), []string{"add#1", "version#1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %v, want %v", got, want)
	}
	add := sh.notify[0].Data
	if add.Flags&(win32.NIFMessage|win32.NIFIcon|win32.NIFTip) != win32.NIFMessage|win32.NIFIcon|win32.NIFTip || add.Tip != "Tip" || add.Icon == 0 {
		t.Errorf("the add = %+v", add)
	}
	// An app has one tray.
	_, err = d.NewTray(spec)
	assertTagRoute(t, err, tagTrayDuplicate, "Desktop.NewTray."+tagTrayDuplicate)
	tr.Close()
	if got := sh.commands(); got[len(got)-1] != "delete#1" {
		t.Errorf("Close sent %v", got)
	}
	if len(sh.freedIcons) != 1 {
		t.Errorf("the icon was freed %d times", len(sh.freedIcons))
	}
	tr.Close() // twice does nothing
	if n := len(sh.commands()); n != 3 {
		t.Errorf("Close twice sent more: %v", sh.commands())
	}
	// And after the first is gone, a new one can be made.
	if _, err := d.NewTray(spec); err != nil {
		t.Errorf("a new tray after Close: %v", err)
	}
}

func TestNewTrayFailures(t *testing.T) {
	t.Run("the shell refuses the icon", func(t *testing.T) {
		d, sh := newTestDesktop()
		sh.notifyErr = func(c notifyCall) error { return errors.New(win32.TagWin32CallFailed, "no tray") }
		_, err := d.NewTray(platform.TraySpec{Icon: rgba(16, 1)})
		assertTagRoute(t, err, tagTrayRegisterFailed, "Desktop.NewTray."+tagTrayRegisterFailed)
		if len(sh.freedIcons) != 1 || d.tray != nil {
			t.Errorf("the icon was freed %d times and the tray is %v", len(sh.freedIcons), d.tray)
		}
	})
	t.Run("the icon cannot be made", func(t *testing.T) {
		d, sh := newTestDesktop()
		sh.iconErr = errors.New(tagWindowIconInvalid, "bad picture")
		_, err := d.NewTray(platform.TraySpec{Icon: rgba(16, 1)})
		assertTagRoute(t, err, tagTrayRegisterFailed, "Desktop.NewTray."+tagTrayRegisterFailed)
	})
	t.Run("a shell without version 4 still has the icon", func(t *testing.T) {
		d, sh := newTestDesktop()
		sh.notifyErr = func(c notifyCall) error {
			if c.Command == win32.NIMSetVersion {
				return errors.New(win32.TagWin32CallFailed, "no version 4")
			}
			return nil
		}
		if _, err := d.NewTray(platform.TraySpec{}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestTrayUpdateSendsOnlyWhatDiffers(t *testing.T) {
	d, sh := newTestDesktop()
	spec := platform.TraySpec{Tooltip: "Tip", Icon: rgba(16, 1), Menu: []platform.TrayMenuItem{item("A")}}
	tr, err := d.NewTray(spec)
	if err != nil {
		t.Fatal(err)
	}
	n := len(sh.notify)

	same := spec
	same.OnActivate = func() {}
	tr.Update(same)
	if len(sh.notify) != n {
		t.Fatalf("an update with nothing different sent %v", sh.commands()[n:])
	}
	menuOnly := spec
	menuOnly.Menu = []platform.TrayMenuItem{item("B")}
	tr.Update(menuOnly)
	if len(sh.notify) != n {
		t.Fatalf("a new menu sent %v: it is made when it is shown", sh.commands()[n:])
	}

	tipOnly := menuOnly
	tipOnly.Tooltip = "New tip"
	tr.Update(tipOnly)
	c := sh.lastNotify()
	if c.Command != win32.NIMModify || c.Data.Flags != win32.NIFTip|win32.NIFShowTip || c.Data.Tip != "New tip" {
		t.Errorf("a new tooltip sent %+v", c)
	}

	newIcon := tipOnly
	newIcon.Icon = rgba(16, 7)
	oldIcon := d.tray.icon
	tr.Update(newIcon)
	c = sh.lastNotify()
	if c.Command != win32.NIMModify || c.Data.Flags != win32.NIFIcon || c.Data.Icon == 0 || c.Data.Icon == oldIcon {
		t.Errorf("a new icon sent %+v", c)
	}
	if !reflect.DeepEqual(sh.freedIcons, []uintptr{oldIcon}) {
		t.Errorf("the old icon was freed as %v, want [%d]", sh.freedIcons, oldIcon)
	}

	n = len(sh.notify)
	tr.Close()
	tr.Update(spec) // after Close
	if len(sh.notify) != n+1 {
		t.Errorf("an update after Close sent %v", sh.commands()[n+1:])
	}
}

func TestTrayEvents(t *testing.T) {
	type calls struct{ activate, secondary, pressed int }
	setup := func(withActivate bool, menu bool) (*Desktop, *fakeShell, *calls) {
		d, sh := newTestDesktop()
		c := &calls{}
		spec := platform.TraySpec{OnSecondaryActivate: func() { c.secondary++ }}
		if withActivate {
			spec.OnActivate = func() { c.activate++ }
		}
		if menu {
			spec.Menu = []platform.TrayMenuItem{item("Go", func(i *platform.TrayMenuItem) { i.OnPressed = func() { c.pressed++ } })}
		}
		if _, err := d.NewTray(spec); err != nil {
			t.Fatal(err)
		}
		return d, sh, c
	}
	v4 := func(event uint32) (uintptr, uintptr) { return 0, uintptr(event) | uintptr(trayIconID)<<16 }

	t.Run("NIN_SELECT is OnActivate", func(t *testing.T) {
		d, sh, c := setup(true, true)
		w, l := v4(win32.NINSelect)
		d.onIconMessage(w, l)
		w, l = v4(win32.NINKeySelect)
		d.onIconMessage(w, l)
		if c.activate != 2 || len(sh.menus) != 0 {
			t.Errorf("activate %d, menus %d", c.activate, len(sh.menus))
		}
	})
	t.Run("a click opens the menu when there is no OnActivate", func(t *testing.T) {
		d, sh, c := setup(false, true)
		sh.menuCmd = 1
		w, l := v4(win32.NINSelect)
		d.onIconMessage(w, l)
		if len(sh.menus) != 1 || c.pressed != 1 {
			t.Errorf("menus %d, pressed %d", len(sh.menus), c.pressed)
		}
	})
	t.Run("a click does nothing without OnActivate and a menu", func(t *testing.T) {
		d, sh, c := setup(false, false)
		w, l := v4(win32.NINSelect)
		d.onIconMessage(w, l)
		if len(sh.menus) != 0 || c.activate != 0 {
			t.Errorf("menus %d, activate %d", len(sh.menus), c.activate)
		}
	})
	t.Run("the middle button", func(t *testing.T) {
		d, _, c := setup(true, false)
		w, l := v4(win32.WMMButtonUp)
		d.onIconMessage(w, l)
		if c.secondary != 1 || c.activate != 0 {
			t.Errorf("secondary %d, activate %d", c.secondary, c.activate)
		}
	})
	t.Run("WM_CONTEXTMENU shows the menu at the anchor and calls the item that was chosen", func(t *testing.T) {
		d, sh, c := setup(true, true)
		sh.menuCmd = 1
		d.onIconMessage(uintptr(40)|uintptr(60)<<16, uintptr(win32.WMContextMenu)|uintptr(trayIconID)<<16)
		if len(sh.menus) != 1 || sh.menuAt[0] != (point{40, 60}) || c.pressed != 1 {
			t.Errorf("menus %d at %v, pressed %d", len(sh.menus), sh.menuAt, c.pressed)
		}
	})
	t.Run("a menu closed without a choice calls nothing", func(t *testing.T) {
		d, sh, c := setup(true, true)
		sh.menuCmd = 0
		w, l := v4(win32.WMContextMenu)
		d.onIconMessage(w, l)
		if len(sh.menus) != 1 || c.pressed != 0 {
			t.Errorf("menus %d, pressed %d", len(sh.menus), c.pressed)
		}
	})
	t.Run("a menu the shell cannot show is logged", func(t *testing.T) {
		d, sh, c := setup(true, true)
		sh.menuErr = errors.New(win32.TagWin32CallFailed, "no menu")
		w, l := v4(win32.WMContextMenu)
		d.onIconMessage(w, l)
		if c.pressed != 0 {
			t.Errorf("pressed %d", c.pressed)
		}
	})
	t.Run("the legacy messages work too", func(t *testing.T) {
		d, _, c := setup(true, false)
		d.onIconMessage(uintptr(trayIconID), uintptr(win32.NINSelect))
		if c.activate != 1 {
			t.Errorf("activate %d", c.activate)
		}
	})
	t.Run("the events of another icon are not the tray's", func(t *testing.T) {
		d, _, c := setup(true, false)
		d.onIconMessage(0, uintptr(win32.NINSelect)|uintptr(balloonIconID)<<16)
		if c.activate != 0 {
			t.Errorf("activate %d", c.activate)
		}
	})
}

func TestTaskbarCreatedAddsTheIconsAgain(t *testing.T) {
	d, sh := newTestDesktop()
	if d.isTaskbarCreated(0xC000) || !d.isTaskbarCreated(0xC123) {
		t.Fatal("isTaskbarCreated does not know its message")
	}
	if _, err := d.NewTray(platform.TraySpec{Tooltip: "again", Icon: rgba(16, 1)}); err != nil {
		t.Fatal(err)
	}
	n := len(sh.notify)
	d.onTaskbarCreated()
	got := sh.commands()[n:]
	if want := []string{"add#1", "version#1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("after TaskbarCreated: %v, want %v", got, want)
	}
	if add := sh.notify[n].Data; add.Tip != "again" || add.Icon == 0 {
		t.Errorf("the second add = %+v", add)
	}
	// With no message registered, nothing is taken for it.
	d2 := newDesktop(newFakeShell(), quietLog(), 0)
	if d2.isTaskbarCreated(0) {
		t.Error("message 0 is TaskbarCreated")
	}
}

func TestClosedServicesIgnoreTheShell(t *testing.T) {
	d, sh := newTestDesktop()
	if _, err := d.NewTray(platform.TraySpec{OnActivate: func() { t.Error("called after Close") }}); err != nil {
		t.Fatal(err)
	}
	d.Close()
	d.Close()
	n := len(sh.notify)
	d.onIconMessage(0, uintptr(win32.NINSelect)|uintptr(trayIconID)<<16)
	d.onTaskbarCreated()
	if len(sh.notify) != n {
		t.Errorf("a closed Desktop sent %v", sh.commands()[n:])
	}
}

// The tags of the errors that the fakes of these tests make (rm4n0s/errors, R5).
const (
	tagWindowIconInvalid = "WindowIconInvalid"
)
