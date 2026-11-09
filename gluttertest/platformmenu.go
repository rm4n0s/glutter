package gluttertest

import (
	"github.com/rm4n0s/glutter/ui"
)

// PlatformMenus are the menus that the app last put in the menu bar of the fake platform (a PlatformMenuBar):
// the top-level items, each with its children, in the form that goes to the platform. It is empty when the app
// has none.
//
// Flutter: no counterpart
func (t *Tester) PlatformMenus() []ui.PlatformMenuItemData {
	t.host.Lock()
	defer t.host.Unlock()
	return append([]ui.PlatformMenuItemData(nil), t.host.PlatformMenus...)
}

// PlatformMenuSets is how many times the app set the menus of the menu bar: a PlatformMenuBar that did not
// change does not set them again.
//
// Flutter: no counterpart
func (t *Tester) PlatformMenuSets() int {
	t.host.Lock()
	defer t.host.Unlock()
	return t.host.MenuSets
}

// platformMenuHandler is the handler of the last menus, or nil.
func (t *Tester) platformMenuHandler() ui.PlatformMenuHandler {
	t.host.Lock()
	defer t.host.Unlock()
	return t.host.MenuHandler
}

// SelectPlatformMenuItem chooses the item of the menu bar with the id, as the user does: the app's callback
// or intent runs. It does nothing without menus.
//
// Flutter: no counterpart
func (t *Tester) SelectPlatformMenuItem(id int) {
	t.adopt()
	if h := t.platformMenuHandler(); h != nil {
		h.MenuSelected(id)
	}
}

// OpenPlatformMenu opens the menu of the menu bar with the id, as the user does: its OnOpen runs.
//
// Flutter: no counterpart
func (t *Tester) OpenPlatformMenu(id int) {
	t.adopt()
	if h := t.platformMenuHandler(); h != nil {
		h.MenuOpened(id)
	}
}

// ClosePlatformMenu closes the menu of the menu bar with the id: its OnClose runs.
//
// Flutter: no counterpart
func (t *Tester) ClosePlatformMenu(id int) {
	t.adopt()
	if h := t.platformMenuHandler(); h != nil {
		h.MenuClosed(id)
	}
}

// FindPlatformMenuItem returns the first item with the label among the items of the menu bar and below, depth
// first, and whether there is one.
//
// Flutter: no counterpart
func FindPlatformMenuItem(items []ui.PlatformMenuItemData, label string) (ui.PlatformMenuItemData, bool) {
	for _, it := range items {
		if it.Label == label && !it.IsDivider {
			return it, true
		}
		if found, ok := FindPlatformMenuItem(it.Children, label); ok {
			return found, true
		}
	}
	return ui.PlatformMenuItemData{}, false
}

// NoPlatformMenuBar makes the fake platform one without a menu bar, as Linux and Windows are: setting the
// menus does nothing.
//
// Flutter: no counterpart
func (t *Tester) NoPlatformMenuBar() {
	t.host.Lock()
	defer t.host.Unlock()
	t.host.NoMenuBar = true
}
