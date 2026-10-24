package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests cover SelectionOverlay, as the SelectionOverlay group of
// packages/flutter/test/widgets/text_selection_test.dart does.

// handleControls builds handles that can be found by a key per type.
type handleControls struct{}

func handleKey(t rendering.TextSelectionHandleType) foundation.Key {
	return foundation.NewValueKey("handle-" + t.String())
}

func (c handleControls) BuildHandle(_ BuildContext, t rendering.TextSelectionHandleType, _ float64, onTap foundation.VoidCallback) Widget {
	return NewGestureDetector(NewSizedBox(nil, SizedBoxOpts{Key: handleKey(t), Width: foundation.Some(22.0), Height: foundation.Some(22.0)}),
		GestureDetectorOpts{OnTap: func() {
			if onTap != nil {
				onTap()
			}
		}, Behavior: rendering.HitTestBehaviorOpaque})
}

func (handleControls) GetHandleAnchor(rendering.TextSelectionHandleType, float64) ui.Offset {
	return ui.Offset{Dx: 11, Dy: 0}
}
func (handleControls) GetHandleSize(float64) ui.Size { return ui.Size{Width: 22, Height: 22} }

type overlayFixture struct {
	ts                  *gluttertest.Tester
	bc                  BuildContext
	start, end, toolbar *rendering.LayerLink
	startVisible        *foundation.ValueNotifier[bool]
	toolbarVisible      *foundation.ValueNotifier[bool]
	events              []string
	overlay             *SelectionOverlay
	lastStart           gestures.DragStartDetails
}

func newOverlayFixture(t *testing.T, startType, endType rendering.TextSelectionHandleType) *overlayFixture {
	t.Helper()
	f := &overlayFixture{
		ts: gluttertest.New(t), start: rendering.NewLayerLink(), end: rendering.NewLayerLink(), toolbar: rendering.NewLayerLink(),
		startVisible: foundation.NewValueNotifier(true), toolbarVisible: foundation.NewValueNotifier(true),
	}
	pumpContainer(f.ts, NewStack([]Widget{
		NewPositioned(NewCompositedTransformTarget(f.start, NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(1.0), Height: foundation.Some(1.0)})),
			PositionedOpts{Left: foundation.Some(100.0), Top: foundation.Some(100.0)}),
		NewPositioned(NewCompositedTransformTarget(f.end, NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(1.0), Height: foundation.Some(1.0)})),
			PositionedOpts{Left: foundation.Some(300.0), Top: foundation.Some(100.0)}),
		NewPositioned(NewCompositedTransformTarget(f.toolbar, NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(1.0), Height: foundation.Some(1.0)})),
			PositionedOpts{Left: foundation.Some(200.0), Top: foundation.Some(50.0)}),
		NewBuilder(func(bc BuildContext) Widget { f.bc = bc; return NewSizedBox(nil) }),
	}))
	f.overlay = NewSelectionOverlay(f.bc, startType, 14, endType, 14,
		[]rendering.TextSelectionPoint{{Point: ui.Offset{Dx: 100, Dy: 114}}, {Point: ui.Offset{Dx: 300, Dy: 114}}},
		handleControls{}, f.start, f.end, f.toolbar, SelectionOverlayOpts{
			StartHandlesVisible: f.startVisible, ToolbarVisible: f.toolbarVisible,
			OnStartHandleDragStart:  func(d gestures.DragStartDetails) { f.events = append(f.events, "startStart"); f.lastStart = d },
			OnStartHandleDragUpdate: func(gestures.DragUpdateDetails) { f.events = append(f.events, "startUpdate") },
			OnStartHandleDragEnd:    func(gestures.DragEndDetails) { f.events = append(f.events, "startEnd") },
			OnEndHandleDragStart:    func(gestures.DragStartDetails) { f.events = append(f.events, "endStart") },
			OnEndHandleDragUpdate:   func(gestures.DragUpdateDetails) { f.events = append(f.events, "endUpdate") },
			OnEndHandleDragEnd:      func(gestures.DragEndDetails) { f.events = append(f.events, "endEnd") },
		})
	t.Cleanup(f.overlay.Dispose)
	return f
}

func menuBuilder(label string) WidgetBuilder {
	return func(BuildContext) Widget {
		return NewAlign(NewSizedBox(ltrText(label), SizedBoxOpts{Width: foundation.Some(50.0), Height: foundation.Some(20.0)}), AlignOpts{Alignment: painting.AlignmentBottomRight})
	}
}

func TestSelectionOverlayShowsAndHidesTheHandles(t *testing.T) {
	f := newOverlayFixture(t, rendering.TextSelectionHandleTypeLeft, rendering.TextSelectionHandleTypeRight)
	left, right := gluttertest.ByKey(handleKey(rendering.TextSelectionHandleTypeLeft)), gluttertest.ByKey(handleKey(rendering.TextSelectionHandleTypeRight))
	f.overlay.ShowHandles()
	f.ts.Pump()
	if !f.ts.Exists(left) || !f.ts.Exists(right) {
		t.Fatal("the handles are not in the overlay")
	}
	// The handle is placed at its link, shifted by the anchor.
	if got := f.ts.TopLeft(left); got.Dx != 100-11 {
		t.Errorf("start handle left = %v, want %v", got.Dx, 100-11)
	}
	f.overlay.ShowHandles() // a second call does nothing
	f.ts.Pump()
	if f.ts.Count(left) != 1 {
		t.Fatalf("%d start handles after showing twice", f.ts.Count(left))
	}
	f.overlay.HideHandles()
	f.ts.Pump()
	if f.ts.Exists(left) || f.ts.Exists(right) {
		t.Fatal("the handles stayed after HideHandles")
	}
}

func TestSelectionOverlayCollapsedSelectionShowsOnlyTheStartHandle(t *testing.T) {
	f := newOverlayFixture(t, rendering.TextSelectionHandleTypeCollapsed, rendering.TextSelectionHandleTypeCollapsed)
	f.overlay.ShowHandles()
	f.ts.Pump()
	if f.ts.Count(gluttertest.ByKey(handleKey(rendering.TextSelectionHandleTypeCollapsed))) != 1 {
		t.Fatalf("collapsed handles = %d, want 1", f.ts.Count(gluttertest.ByKey(handleKey(rendering.TextSelectionHandleTypeCollapsed))))
	}
}

func TestSelectionOverlayShowsTheToolbarAndHidesIt(t *testing.T) {
	f := newOverlayFixture(t, rendering.TextSelectionHandleTypeLeft, rendering.TextSelectionHandleTypeRight)
	f.overlay.ShowToolbar(f.bc, menuBuilder("Cut"))
	f.ts.Pump()
	if !f.overlay.ToolbarIsVisible() || !f.ts.Exists(gluttertest.ByText("Cut")) {
		t.Fatal("the toolbar is not shown")
	}
	f.overlay.HideToolbar()
	f.ts.Pump()
	if f.overlay.ToolbarIsVisible() || f.ts.Exists(gluttertest.ByText("Cut")) {
		t.Fatal("the toolbar stayed")
	}
	f.overlay.ShowHandles()
	f.overlay.ShowToolbar(f.bc, menuBuilder("Cut"))
	f.ts.Pump()
	f.overlay.Hide()
	f.ts.Pump()
	if f.overlay.ToolbarIsVisible() || f.ts.Exists(gluttertest.ByKey(handleKey(rendering.TextSelectionHandleTypeLeft))) {
		t.Fatal("Hide left the toolbar or the handles")
	}
}

func TestSelectionOverlayToolbarFollowsItsVisibility(t *testing.T) {
	f := newOverlayFixture(t, rendering.TextSelectionHandleTypeLeft, rendering.TextSelectionHandleTypeRight)
	f.overlay.ShowToolbar(f.bc, menuBuilder("Cut"))
	f.ts.PumpAndSettle()
	opacity := func() float64 {
		return f.ts.Widget(gluttertest.Ancestor(gluttertest.ByText("Cut"), gluttertest.ByType[*FadeTransition]())).(*FadeTransition).Opacity().Value()
	}
	if opacity() != 1 {
		t.Fatalf("toolbar opacity = %v, want 1", opacity())
	}
	f.toolbarVisible.SetValue(false)
	f.ts.Pump()
	f.ts.Pump(SelectionOverlayFadeDuration)
	if opacity() != 0 {
		t.Fatalf("toolbar opacity after hiding = %v, want 0", opacity())
	}
}

func TestSelectionOverlayHandleDragCallbacks(t *testing.T) {
	f := newOverlayFixture(t, rendering.TextSelectionHandleTypeLeft, rendering.TextSelectionHandleTypeRight)
	f.overlay.ShowHandles()
	f.ts.Pump()
	handle := gluttertest.ByKey(handleKey(rendering.TextSelectionHandleTypeLeft))
	at := f.ts.Center(handle)
	g := f.ts.StartGesture(at)
	g.MoveTo(at.Add(ui.Offset{Dx: 30, Dy: 0}))
	f.ts.Pump()
	g.MoveTo(at.Add(ui.Offset{Dx: 60, Dy: 0}))
	f.ts.Pump()
	g.Up()
	f.ts.Pump()
	want := []string{"startStart", "startUpdate", "startEnd"}
	if len(f.events) < 3 || f.events[0] != want[0] || f.events[len(f.events)-1] != want[2] {
		t.Fatalf("drag events = %v, want %v with updates between", f.events, want)
	}
	if !f.ts.Exists(handle) {
		t.Fatal("the dragged handle was removed")
	}
	f.events = nil
	// The end handle is dragged on its own.
	end := gluttertest.ByKey(handleKey(rendering.TextSelectionHandleTypeRight))
	at = f.ts.Center(end)
	g = f.ts.StartGesture(at)
	g.MoveTo(at.Add(ui.Offset{Dx: 0, Dy: 40}))
	f.ts.Pump()
	g.Up()
	f.ts.Pump()
	if len(f.events) == 0 || f.events[0] != "endStart" || f.events[len(f.events)-1] != "endEnd" {
		t.Fatalf("end drag events = %v", f.events)
	}
}

func TestSelectionOverlaySettersMarkTheHandlesForRebuild(t *testing.T) {
	f := newOverlayFixture(t, rendering.TextSelectionHandleTypeLeft, rendering.TextSelectionHandleTypeRight)
	f.overlay.ShowHandles()
	f.ts.Pump()
	f.overlay.SetStartHandleType(rendering.TextSelectionHandleTypeCollapsed)
	f.overlay.SetEndHandleType(rendering.TextSelectionHandleTypeCollapsed)
	f.ts.Pump()
	if f.ts.Exists(gluttertest.ByKey(handleKey(rendering.TextSelectionHandleTypeLeft))) ||
		!f.ts.Exists(gluttertest.ByKey(handleKey(rendering.TextSelectionHandleTypeCollapsed))) {
		t.Fatal("the handle did not change type")
	}
	if f.overlay.StartHandleType() != rendering.TextSelectionHandleTypeCollapsed || f.overlay.LineHeightAtStart() != 14 {
		t.Fatalf("getters: %v %v", f.overlay.StartHandleType(), f.overlay.LineHeightAtStart())
	}
}

// Ported from SelectionOverlay.selectionEndpoints of widgets/text_selection.dart: only Android
// gives a selection click when the endpoints change while a handle is dragged.
func TestSelectionOverlayEndpointsGiveHapticFeedbackWhileDraggingOnAndroid(t *testing.T) {
	tests := []struct {
		platform foundation.TargetPlatform
		want     int
	}{
		{foundation.TargetPlatformAndroid, 1},
		{foundation.TargetPlatformFuchsia, 0},
		{foundation.TargetPlatformIOS, 0},
		{foundation.TargetPlatformLinux, 0},
	}
	for _, tc := range tests {
		t.Run(tc.platform.String(), func(t *testing.T) {
			usePlatform(t, tc.platform)
			f := newOverlayFixture(t, rendering.TextSelectionHandleTypeLeft, rendering.TextSelectionHandleTypeRight)
			clicks := 0
			f.ts.Binding.HapticFeedback().OnPerform = func(k ui.HapticFeedbackKind) {
				if k == ui.HapticFeedbackSelectionClick {
					clicks++
				}
			}
			moved := []rendering.TextSelectionPoint{{Point: ui.Offset{Dx: 110, Dy: 114}}, {Point: ui.Offset{Dx: 300, Dy: 114}}}
			// Not dragging: the endpoints change without feedback.
			f.overlay.SetSelectionEndpoints(moved)
			if clicks != 0 {
				t.Fatalf("clicks without a drag = %d, want 0", clicks)
			}
			f.overlay.ShowHandles()
			f.ts.Pump()
			at := f.ts.Center(gluttertest.ByKey(handleKey(rendering.TextSelectionHandleTypeLeft)))
			g := f.ts.StartGesture(at)
			g.MoveTo(at.Add(ui.Offset{Dx: 30, Dy: 0}))
			f.ts.Pump()
			f.overlay.SetSelectionEndpoints([]rendering.TextSelectionPoint{{Point: ui.Offset{Dx: 120, Dy: 114}}, {Point: ui.Offset{Dx: 300, Dy: 114}}})
			// The same endpoints again do not click.
			f.overlay.SetSelectionEndpoints([]rendering.TextSelectionPoint{{Point: ui.Offset{Dx: 120, Dy: 114}}, {Point: ui.Offset{Dx: 300, Dy: 114}}})
			g.Up()
			f.ts.Pump()
			if clicks != tc.want {
				t.Errorf("clicks = %d, want %d", clicks, tc.want)
			}
		})
	}
}
