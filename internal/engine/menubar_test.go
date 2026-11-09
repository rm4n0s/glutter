package engine

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// barConn is a connection that has a menu bar; the other tests' connections have none.
type barConn struct {
	connection
	menus   []ui.PlatformMenuItemData
	handler ui.PlatformMenuHandler
}

func (c *barConn) SetMenus(m []ui.PlatformMenuItemData, h ui.PlatformMenuHandler) {
	c.menus, c.handler = m, h
}

// barlessConn is a connection with no menu bar.
type barlessConn struct{ connection }

type menuHandler struct{}

func (menuHandler) MenuSelected(int) {}
func (menuHandler) MenuOpened(int)   {}
func (menuHandler) MenuClosed(int)   {}

func TestSetPlatformMenusGoesToAPlatformWithAMenuBar(t *testing.T) {
	bar := &barConn{}
	h := &handler{conn: bar}
	items := []ui.PlatformMenuItemData{{ID: 1, Label: "File"}}
	if !h.SetPlatformMenus(items, menuHandler{}) {
		t.Fatal("a platform with a menu bar reported none")
	}
	if len(bar.menus) != 1 || bar.menus[0].Label != "File" || bar.handler == nil {
		t.Errorf("the platform got %+v and %v", bar.menus, bar.handler)
	}
	// A platform without one reports that and gets nothing.
	if (&handler{conn: barlessConn{}}).SetPlatformMenus(items, menuHandler{}) {
		t.Error("a platform without a menu bar reported one")
	}
}
