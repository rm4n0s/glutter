package material_test

import (
	"image/color"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/gluttertest"
	. "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// materialApp wraps child in what a Material widget needs: a Directionality
// and a Theme.
func materialApp(theme *ThemeData, child widgets.Widget) widgets.Widget {
	if theme == nil {
		theme = NewThemeData()
	}
	return widgets.NewDirectionality(ui.TextDirectionLTR, NewTheme(theme, child))
}

// pixel returns the color of the screen at the logical position (dpr 1).
func pixel(ts *gluttertest.Tester, x, y int) color.RGBA {
	return ts.Screenshot().RGBAAt(x, y)
}

// recoverError runs f and returns the error it panics with, or nil.
func recoverError(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(error)
			if !ok {
				panic(r)
			}
			err = e
		}
	}()
	f()
	return nil
}

// wantRoute fails the test unless err is an *errors.Error whose route contains route.
func wantRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s, want it to contain %s", appErr.Route(), route)
	}
}

func asAppError(err error) (*errors.Error, bool) { return errors.FromError(err) }
