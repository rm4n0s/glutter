package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// The cursor of a pressed pointer (step 27.7): MouseRegion with HoldCursorWhilePressed
// keeps its cursor for a mouse pressed inside it, until the release.
func TestMouseRegionHoldCursorWhilePressed(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the active cursor of a device is only read in debug builds")
	}
	sash := services.SystemMouseCursors.ResizeLeftRight
	basic := services.SystemMouseCursors.Basic
	tests := []struct {
		name         string
		hold         bool
		wantAway     services.MouseCursor // while pressed, with the pointer away from the region
		wantReleased services.MouseCursor // after the release, still away
	}{
		{"holding", true, sash, basic},
		{"not holding", false, basic, basic},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t)
			ts.PumpWidget(NewStack([]Widget{
				NewMouseRegion(nil, MouseRegionOpts{Cursor: sash, HoldCursorWhilePressed: tt.hold}),
			}, StackOpts{TextDirection: ui.TextDirectionLTR}))
			cursor := func() services.MouseCursor {
				c, _ := ts.Binding.Renderer().MouseTracker().DebugDeviceActiveCursor(0)
				return c
			}
			inside := ts.Center(gluttertest.ByType[*MouseRegion]())
			away := ui.Offset{Dx: -50, Dy: -50}
			ts.Hover(inside) // the mouse is there before the press, so the release does not remove it
			g := ts.StartGesture(inside, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse, Buttons: foundation.Some(gestures.PrimaryButton)})
			ts.Pump()
			if c := cursor(); c != sash {
				t.Fatalf("pressed inside: cursor = %v, want the region's", c)
			}
			g.MoveTo(away)
			ts.Pump()
			if c := cursor(); c != tt.wantAway {
				t.Errorf("pressed, away: cursor = %v, want %v", c, tt.wantAway)
			}
			g.Up()
			ts.Pump()
			if c := cursor(); c != tt.wantReleased {
				t.Errorf("released, away: cursor = %v, want %v", c, tt.wantReleased)
			}
		})
	}
}
