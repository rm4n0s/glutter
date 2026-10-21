package ui

import "testing"

type menuHost struct {
	enabled []bool
	result  bool
}

func (m *menuHost) ScheduleFrame()                  {}
func (m *menuHost) RenderView(ViewID, *Scene) error { return nil }
func (m *menuHost) RequestAppExit()                 {}
func (m *menuHost) Post(fn func())                  { fn() }
func (m *menuHost) SetBrowserContextMenuEnabled(e bool) bool {
	m.enabled = append(m.enabled, e)
	return m.result
}

func TestSetBrowserContextMenuEnabled(t *testing.T) {
	t.Run("a host with a context menu is asked and answers", func(t *testing.T) {
		h := &menuHost{result: true}
		d := NewPlatformDispatcher(h)
		if !d.SetBrowserContextMenuEnabled(false) || len(h.enabled) != 1 || h.enabled[0] {
			t.Errorf("host calls %v", h.enabled)
		}
		h.result = false
		if d.SetBrowserContextMenuEnabled(true) {
			t.Error("it reported a change the host did not make")
		}
	})
	t.Run("a host without one changes nothing", func(t *testing.T) {
		d := NewPlatformDispatcher(nil)
		if d.SetBrowserContextMenuEnabled(false) {
			t.Error("a dispatcher without a host reported a change")
		}
	})
}
