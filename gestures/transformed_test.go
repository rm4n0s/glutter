package gestures

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

// childMatrix maps the global coordinates to the ones of a child scaled around
// the global center (400, 300), which shows the point (50, 75) there.
func childMatrix(scale float64) *ui.Matrix4 {
	m := ui.NewMatrix4TranslationValues(50, 75, 0)
	m.MultiplyBy(ui.NewMatrix4Diagonal3Values(float32(1/scale), float32(1/scale), 1))
	m.MultiplyBy(ui.NewMatrix4TranslationValues(-400, -300, 0))
	return m
}

func nearly(a, b float64) bool { return math.Abs(a-b) < 1e-4 }

func TestTransformedTapEvaluatesSlopInGlobalSpace(t *testing.T) {
	for _, tc := range []struct {
		name  string
		scale float64
	}{{"scaled up", 2}, {"scaled down", 0.5}} {
		t.Run(tc.name, func(t *testing.T) {
			m := childMatrix(tc.scale)
			center := off(400, 300)
			run := func(moveBy float64) (taps, cancels int, down []TapDownDetails, up []TapUpDetails) {
				g := newGestureTester(t)
				tap := NewTapGestureRecognizer(g.binding)
				tap.OnTap = func() { taps++ }
				tap.OnTapCancel = func() { cancels++ }
				tap.OnTapDown = func(d TapDownDetails) { down = append(down, d) }
				tap.OnTapUp = func(d TapUpDetails) { up = append(up, d) }
				pointer := newTestPointer()
				d := pointer.down(center).Transformed(m).(*PointerDownEvent)
				tap.AddPointer(d)
				g.closeArena(1)
				g.route(d)
				g.elapse(PressTimeout)
				g.route(pointer.move(center.Add(off(0, moveBy))).Transformed(m))
				g.route(pointer.up().Transformed(m))
				return
			}

			taps, cancels, down, up := run(TouchSlop - 1)
			if taps != 1 || cancels != 0 || len(down) != 1 || len(up) != 1 {
				t.Fatalf("within the slop: taps %d, cancels %d, downs %d, ups %d", taps, cancels, len(down), len(up))
			}
			if down[0].LocalPosition != off(50, 75) || down[0].GlobalPosition != center {
				t.Errorf("down = %+v", down[0])
			}
			wantLocal := 75 + (TouchSlop-1)/tc.scale
			if !nearly(up[0].LocalPosition.Dy, wantLocal) || up[0].LocalPosition.Dx != 50 || !nearly(up[0].GlobalPosition.Dy, 300+TouchSlop-1) {
				t.Errorf("up = %+v, want local y %v", up[0], wantLocal)
			}

			taps, cancels, down, up = run(TouchSlop + 1)
			if taps != 0 || cancels != 1 || len(down) != 1 || len(up) != 0 {
				t.Fatalf("past the slop: taps %d, cancels %d, downs %d, ups %d", taps, cancels, len(down), len(up))
			}
			if down[0].LocalPosition != off(50, 75) || down[0].GlobalPosition != center {
				t.Errorf("down = %+v", down[0])
			}
		})
	}
}

func TestTransformedDragReportsLocalAndGlobalPositions(t *testing.T) {
	g := newGestureTester(t)
	m := childMatrix(2)
	drag := NewVerticalDragGestureRecognizer(g.binding, DragGestureRecognizerOpts{DragStartBehavior: DragStartBehaviorDown})
	var starts []DragStartDetails
	var updates []DragUpdateDetails
	drag.OnStart = func(d DragStartDetails) { starts = append(starts, d) }
	drag.OnUpdate = func(d DragUpdateDetails) { updates = append(updates, d) }
	pointer := newTestPointer()
	center := off(400, 300)
	down := pointer.down(center).Transformed(m).(*PointerDownEvent)
	drag.AddPointer(down)
	g.closeArena(1)
	g.route(down)
	g.route(pointer.move(center.Add(off(0, 40))).Transformed(m))
	if len(starts) != 1 || starts[0].GlobalPosition != center || starts[0].LocalPosition != off(50, 75) {
		t.Fatalf("starts = %+v", starts)
	}
	if len(updates) != 1 {
		t.Fatalf("updates = %+v", updates)
	}
	u := updates[0]
	// The child is scaled by 2, so 40 global pixels are 20 local ones.
	if !nearly(u.GlobalPosition.Dy, 340) || !nearly(u.LocalPosition.Dy, 95) || !nearly(u.Delta.Dy, 20) {
		t.Errorf("update = %+v", u)
	}
	drag.Dispose()
}
