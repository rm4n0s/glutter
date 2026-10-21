package widgets

import (
	"testing"
)

// TestBindingHasTheBrowserContextMenu: the binding controls the browser's context menu, which is
// enabled by default; on a host that is not a browser nothing changes (the renderhost has none).
func TestBindingHasTheBrowserContextMenu(t *testing.T) {
	bt := newBindingTree(t)
	m := bt.b.BrowserContextMenu()
	if m == nil || !m.Enabled() {
		t.Fatalf("BrowserContextMenu() = %v, want an enabled menu", m)
	}
	m.DisableContextMenu()
	if !m.Enabled() {
		t.Error("the menu was disabled on a host without a browser")
	}
}
