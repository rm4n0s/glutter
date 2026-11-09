//go:build darwin && !ios && glutter_display

package macos

import (
	"testing"

	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
	"github.com/rm4n0s/glutter/ui"
)

// menuRecorder is the handler of the menus: it records what the platform tells.
type menuRecorder struct{ selected, opened, closed []int }

func (r *menuRecorder) MenuSelected(id int) { r.selected = append(r.selected, id) }
func (r *menuRecorder) MenuOpened(id int)   { r.opened = append(r.opened, id) }
func (r *menuRecorder) MenuClosed(id int)   { r.closed = append(r.closed, id) }

func titlesOf(menu objc.ID) []string {
	items := send[objc.ID](menu, "itemArray")
	var out []string
	for i := range objcx.ArrayCount(items) {
		it := objcx.ArrayAt(items, i)
		if send[bool](it, "isSeparatorItem") {
			out = append(out, "-")
		} else {
			out = append(out, objcx.String(send[objc.ID](it, "title")))
		}
	}
	return out
}

func itemAt(menu objc.ID, i int) objc.ID { return objcx.ArrayAt(send[objc.ID](menu, "itemArray"), i) }

func TestDefaultMenuBarHasTheStandardMenus(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		s.onUI(func() {
			m := s.p.menus
			name := processName()
			if got := send[objc.ID](s.p.app, "mainMenu"); got != 0 {
				t.Errorf("an accessory app has the main menu %v before it is installed", got)
			}
			m.installDefault()
			menu := send[objc.ID](s.p.app, "mainMenu")
			top := titlesOf(menu)
			if len(top) != 5 || top[0] != name || top[1] != "Edit" || top[2] != "View" || top[3] != "Window" || top[4] != "Help" {
				t.Fatalf("top-level menus = %v", top)
			}
			app := send[objc.ID](itemAt(menu, 0), "submenu")
			want := []string{"About " + name, "-", "Services", "-", "Hide " + name, "Hide Others", "Show All", "-", "Quit " + name}
			if got := titlesOf(app); len(got) != len(want) {
				t.Fatalf("app menu = %v, want %v", got, want)
			} else {
				for i := range want {
					if got[i] != want[i] {
						t.Errorf("app menu item %d = %q, want %q", i, got[i], want[i])
					}
				}
			}
			quit := itemAt(app, 8)
			if objcx.String(send[objc.ID](quit, "keyEquivalent")) != "q" || send[uint64](quit, "keyEquivalentModifierMask")&menuMaskCommand == 0 {
				t.Error("Quit has no Command+Q")
			}
			if send[objc.SEL](quit, "action") != sel("terminate:") {
				t.Error("Quit does not terminate")
			}
			if send[objc.ID](s.p.app, "servicesMenu") == 0 || send[objc.ID](s.p.app, "windowsMenu") == 0 || send[objc.ID](s.p.app, "helpMenu") == 0 {
				t.Error("the services, windows or help menu of the app is not set")
			}
			// The Edit menu's commands are the view's: the view answers each of them.
			edit := send[objc.ID](itemAt(menu, 1), "submenu")
			for _, a := range []string{"undo:", "redo:", "cut:", "copy:", "paste:", "selectAll:"} {
				found := false
				items := send[objc.ID](edit, "itemArray")
				for i := range objcx.ArrayCount(items) {
					if send[objc.SEL](objcx.ArrayAt(items, i), "action") == sel(a) {
						found = true
					}
				}
				if !found {
					t.Errorf("the Edit menu has no %s", a)
				}
				if !send[bool](objc.ID(s.p.s.view.Class), "instancesRespondToSelector:", sel(a)) {
					t.Errorf("the view does not answer %s", a)
				}
			}
		})
	})
}

func shortcutData(id int, label string, hasChar bool, char string, trigger uint64, mods int) ui.PlatformMenuItemData {
	return ui.PlatformMenuItemData{ID: id, Label: label, Enabled: true, HasShortcutCharacter: hasChar, ShortcutCharacter: char,
		HasShortcutTrigger: !hasChar && trigger != 0, ShortcutTrigger: trigger, ShortcutModifiers: mods}
}

func TestPlatformMenusBecomeTheMainMenu(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		rec := &menuRecorder{}
		s.onUI(func() {
			name := processName()
			s.p.SetMenus([]ui.PlatformMenuItemData{
				{ID: 1, Label: "APP_NAME", Enabled: true, HasChildren: true, Children: []ui.PlatformMenuItemData{
					{ID: 2, HasProvided: true, ProvidedType: 0, Enabled: true},
					{ID: 3, IsDivider: true},
					{ID: 4, HasProvided: true, ProvidedType: 1, Enabled: true},
					{ID: 5, HasProvided: true, ProvidedType: 2, Enabled: true},
				}},
				{ID: 10, Label: "File", Enabled: true, HasChildren: true, Children: []ui.PlatformMenuItemData{
					shortcutData(11, "New", true, "n", 0, ui.PlatformMenuShortcutModifierMeta),
					{ID: 12, Label: "Save as APP_NAME", Enabled: true, HasTooltip: true, Tooltip: "for APP_NAME",
						HasShortcutTrigger: true, ShortcutTrigger: 0x73, ShortcutModifiers: ui.PlatformMenuShortcutModifierMeta | ui.PlatformMenuShortcutModifierShift},
					{ID: 13, Label: "Off", Enabled: false},
					shortcutData(14, "Home", false, "", 0x00100000306, 0),
				}},
				{ID: 20, Label: "Empty", Enabled: false, HasChildren: true},
			}, rec)
			menu := send[objc.ID](s.p.app, "mainMenu")
			top := titlesOf(menu)
			if len(top) != 3 || top[0] != name || top[1] != "File" || top[2] != "Empty" {
				t.Fatalf("top-level menus = %v (the app's name is %q)", top, name)
			}
			appMenu := send[objc.ID](itemAt(menu, 0), "submenu")
			if got := titlesOf(appMenu); len(got) != 4 || got[1] != "-" {
				t.Errorf("app menu = %v", got)
			}
			if quit := itemAt(appMenu, 2); send[objc.SEL](quit, "action") != sel("terminate:") ||
				objcx.String(send[objc.ID](quit, "keyEquivalent")) != "q" {
				t.Errorf("the provided Quit is %v", objcx.String(send[objc.ID](quit, "title")))
			}
			if services := itemAt(appMenu, 3); send[objc.ID](services, "submenu") == 0 {
				t.Error("the provided Services item has no submenu")
			}
			file := send[objc.ID](itemAt(menu, 1), "submenu")
			n, save, off, home := itemAt(file, 0), itemAt(file, 1), itemAt(file, 2), itemAt(file, 3)
			if objcx.String(send[objc.ID](n, "keyEquivalent")) != "n" || send[uint64](n, "keyEquivalentModifierMask") != menuMaskCommand {
				t.Errorf("New has the key %q and the mask %#x", objcx.String(send[objc.ID](n, "keyEquivalent")), send[uint64](n, "keyEquivalentModifierMask"))
			}
			if objcx.String(send[objc.ID](save, "title")) != "Save as "+name || objcx.String(send[objc.ID](save, "toolTip")) != "for "+name {
				t.Errorf("Save has the title %q and the tooltip %q", objcx.String(send[objc.ID](save, "title")), objcx.String(send[objc.ID](save, "toolTip")))
			}
			if objcx.String(send[objc.ID](save, "keyEquivalent")) != "s" || send[uint64](save, "keyEquivalentModifierMask") != menuMaskCommand|menuMaskShift {
				t.Error("Save has no Shift+Command+S")
			}
			if send[bool](off, "isEnabled") || !send[bool](n, "isEnabled") {
				t.Error("the enabled state of the items is wrong")
			}
			if objcx.String(send[objc.ID](home, "keyEquivalent")) != "" {
				t.Errorf("Home has the key %q", objcx.String(send[objc.ID](home, "keyEquivalent")))
			}
			if send[int64](save, "tag") != 12 {
				t.Errorf("Save has the tag %d", send[int64](save, "tag"))
			}
			if send[bool](itemAt(menu, 2), "isEnabled") {
				t.Error("the empty menu is enabled")
			}

			// Choosing an item calls the handler with its id.
			file.Send(sel("performActionForItemAtIndex:"), int64(1))
			file.Send(sel("performActionForItemAtIndex:"), int64(0))
			if len(rec.selected) != 2 || rec.selected[0] != 12 || rec.selected[1] != 11 {
				t.Errorf("selected = %v, want [12 11]", rec.selected)
			}
			// A menu that opens and closes tells its id.
			delegate := send[objc.ID](file, "delegate")
			delegate.Send(sel("menuWillOpen:"), file)
			delegate.Send(sel("menuDidClose:"), file)
			if len(rec.opened) != 1 || rec.opened[0] != 10 || len(rec.closed) != 1 || rec.closed[0] != 10 {
				t.Errorf("opened %v closed %v, want [10] [10]", rec.opened, rec.closed)
			}

			// No menus gives the default menu bar back on a regular app, and none on an accessory one.
			s.p.SetMenus(nil, rec)
			if got := send[objc.ID](s.p.app, "mainMenu"); got != 0 {
				t.Errorf("an accessory app has a main menu after the menus are cleared")
			}
		})
	})
}

func TestMenuEditCommandsGoToTheFocusedTextField(t *testing.T) {
	h := &recorder{}
	runApp(t, Options{}, h, func(s *session) {
		w := inputWindow(s, h)
		s.onUI(func() {
			w.SetTextInput(ui.TextInputState{Enabled: true, View: w.id, Text: "abc", SelectionAnchor: 0, SelectionCursor: 3})
			h.takeTexts()
			for _, a := range []string{"cut:", "copy:", "paste:", "selectAll:", "undo:", "redo:"} {
				w.view.Send(sel(a), objc.ID(0))
			}
			var got []string
			for _, e := range h.takeTexts() {
				got = append(got, e.Selector)
			}
			want := []string{"cut:", "copy:", "paste:", "selectAll:", "undo:", "redo:"}
			if len(got) != len(want) {
				t.Fatalf("selectors = %v, want %v", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("selector %d = %q, want %q", i, got[i], want[i])
				}
			}
			// With no field the commands do nothing.
			w.SetTextInput(ui.TextInputState{})
			w.view.Send(sel("paste:"), objc.ID(0))
			if extra := h.takeTexts(); len(extra) != 0 {
				t.Errorf("a command with no text field made %v", extra)
			}
		})
	})
}
