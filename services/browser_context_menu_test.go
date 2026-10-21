package services

import "testing"

type menuBackend struct {
	calls  []bool
	result bool
}

func (m *menuBackend) SetBrowserContextMenuEnabled(enabled bool) bool {
	m.calls = append(m.calls, enabled)
	return m.result
}

func TestBrowserContextMenu(t *testing.T) {
	t.Run("it is enabled to begin with", func(t *testing.T) {
		if !NewBrowserContextMenu(&menuBackend{}).Enabled() {
			t.Error("the browser's context menu is not enabled by default")
		}
	})
	t.Run("disabling and enabling again, on a browser", func(t *testing.T) {
		b := &menuBackend{result: true}
		m := NewBrowserContextMenu(b)
		m.DisableContextMenu()
		if m.Enabled() {
			t.Error("Enabled is true after DisableContextMenu")
		}
		m.EnableContextMenu()
		if !m.Enabled() {
			t.Error("Enabled is false after EnableContextMenu")
		}
		if len(b.calls) != 2 || b.calls[0] || !b.calls[1] {
			t.Errorf("backend calls %v, want false then true", b.calls)
		}
	})
	t.Run("it does nothing where there is no browser", func(t *testing.T) {
		for name, m := range map[string]*BrowserContextMenu{
			"a platform without a menu": NewBrowserContextMenu(&menuBackend{result: false}),
			"no backend":                NewBrowserContextMenu(nil),
		} {
			m.DisableContextMenu()
			if !m.Enabled() {
				t.Errorf("%s: Enabled changed", name)
			}
			m.EnableContextMenu()
			if !m.Enabled() {
				t.Errorf("%s: Enabled changed", name)
			}
		}
	})
}
