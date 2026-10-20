package desktop_test

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

func app(t *testing.T) (*gluttertest.Tester, widgets.BuildContext) {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	ts.PumpWidget(widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewText("main")))
	return ts, ts.Element(gluttertest.ByText("main"))
}

func TestKeepScreenOn(t *testing.T) {
	ts, bc := app(t)
	view := widgets.ViewOf(bc).ViewID()
	if err := desktop.KeepScreenOn(bc, true); err != nil {
		t.Fatal(err)
	}
	if !ts.ScreenKeptOn(view) {
		t.Error("the screen is not kept on")
	}
	if err := desktop.KeepScreenOn(bc, false); err != nil {
		t.Fatal(err)
	}
	if ts.ScreenKeptOn(view) {
		t.Error("the screen is still kept on")
	}
}

func TestKeepScreenOnWhenThePlatformCannot(t *testing.T) {
	ts, bc := app(t)
	ts.FailKeepScreenOn(errors.New("IdleInhibitUnavailable", "no protocol"))
	err := desktop.KeepScreenOn(bc, true)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != "IdleInhibitUnavailable" {
		t.Fatalf("err = %v, want the tag IdleInhibitUnavailable", err)
	}
}
