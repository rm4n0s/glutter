package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

func iconOf(size int) ui.WindowIconImage {
	return ui.WindowIconImage{Size: size, RGBA: make([]byte, size*size*4)}
}

func TestWindowIconAndContentTypeOfANewWindow(t *testing.T) {
	ts, bc := windowApp(t)
	icon := ui.WindowIcon{Name: "video-x-generic", Images: []ui.WindowIconImage{iconOf(16)}}
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{Icon: icon, ContentType: ui.ContentTypeVideo})
	if err != nil {
		t.Fatal(err)
	}
	spec := ts.Windows()[0].Spec
	if spec.Icon.Name != "video-x-generic" || len(spec.Icon.Images) != 1 || spec.ContentType != ui.ContentTypeVideo {
		t.Errorf("spec = %+v", spec)
	}
	c.Destroy()

	d, err := NewDialogWindowController(bc, ui.Size{Width: 200, Height: 100}, DialogWindowControllerOpts{Icon: icon, ContentType: ui.ContentTypeGame})
	if err != nil {
		t.Fatal(err)
	}
	spec = ts.Windows()[1].Spec
	if spec.Icon.Name != "video-x-generic" || spec.ContentType != ui.ContentTypeGame {
		t.Errorf("dialog spec = %+v", spec)
	}
	d.Destroy()
}

func TestSettingTheIconAndContentTypeOfAWindow(t *testing.T) {
	ts, bc := windowApp(t)
	c, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300})
	if err != nil {
		t.Fatal(err)
	}
	view := c.RootView().ViewID()
	if err := c.SetIcon(ui.WindowIcon{Name: "applications-games"}); err != nil {
		t.Fatal(err)
	}
	if got := ts.WindowIcon(view); got.Name != "applications-games" {
		t.Errorf("icon = %+v", got)
	}
	c.SetContentType(ui.ContentTypeGame)
	if got := ts.WindowContentType(view); got != ui.ContentTypeGame {
		t.Errorf("content type = %v", got)
	}
	err = c.SetIcon(ui.WindowIcon{Images: []ui.WindowIconImage{{Size: 4, RGBA: []byte{1}}}})
	routeOf(t, err, "WindowIconInvalid")
	if got := ts.WindowIcon(view); got.Name != "applications-games" {
		t.Errorf("an invalid icon replaced the icon: %+v", got)
	}
	// A window that was destroyed ignores them.
	c.Destroy()
	if err := c.SetIcon(ui.WindowIcon{Name: "x"}); err != nil {
		t.Errorf("SetIcon on a destroyed window: %v", err)
	}
	c.SetContentType(ui.ContentTypePhoto)
	if got := ts.WindowContentType(view); got != ui.ContentTypeGame {
		t.Errorf("a destroyed window changed its content type to %v", got)
	}
}

func TestInvalidIconFailsTheWindow(t *testing.T) {
	_, bc := windowApp(t)
	_, err := NewRegularWindowController(bc, ui.Size{Width: 400, Height: 300}, RegularWindowControllerOpts{
		Icon: ui.WindowIcon{Images: []ui.WindowIconImage{{Size: 2, RGBA: nil}}}})
	if err == nil {
		t.Fatal("a window with an invalid icon was made")
	}
	routeOf(t, err, "WindowIconInvalid")
}
