package widgets_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/overlay_layout_builder_test.dart.
// 'Still works if child and overlay child are null' has a box in place of the
// element without a render object of the Dart test (_NullLeaf), and 'Screams if
// RenderFollower is spotted in path' has a box with a RenderFollowerLayer in place
// of CompositedTransformFollower, which is not ported yet.

func TestOverlayChildLayoutBuilderBasic(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
	c1, _, _, _ := portalControllers()
	var setState StateSetter
	transform := ui.NewMatrix4Identity()
	var paintTransform ui.Matrix4
	var regularChildSize, theaterSize ui.Size
	var regularChildRectInTheater ui.Rect
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewPositioned(NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
			setState = setter
			// RenderTransform uses the size in its ApplyPaintTransform if the alignment is
			// set.
			return NewTransform(transform, NewOverlayPortalOverlayChildLayoutBuilder(c1, func(_ BuildContext, info OverlayChildLayoutInfo) Widget {
				paintTransform = info.ChildPaintTransform
				regularChildSize = info.ChildSize
				regularChildRectInTheater = paintTransform.TransformRect(ui.OffsetZero.And(info.ChildSize))
				theaterSize = info.OverlaySize
				return NewSizedBox(nil)
			}, NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(40.0), Height: foundation.Some(50.0)})),
				TransformOpts{Alignment: painting.AlignmentTopLeft})
		}), PositionedOpts{Left: foundation.Some(10.0), Top: foundation.Some(20.0)})
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	// The overlay child does not schedule a new frame by itself.
	if ts.Binding.Scheduler().HasScheduledFrame() {
		t.Error("a frame is scheduled")
	}
	if want := *ui.NewMatrix4TranslationValues(10, 20, 0); paintTransform != want {
		t.Errorf("paint transform = %v, want %v", paintTransform, want)
	}
	if regularChildSize != (ui.Size{Width: 40, Height: 50}) || theaterSize != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("sizes = %v, %v; want 40x50, 800x600", regularChildSize, theaterSize)
	}
	if want := ui.NewRectFromLTWH(10, 20, 40, 50); regularChildRectInTheater != want {
		t.Errorf("rect = %v, want %v", regularChildRectInTheater, want)
	}

	transform = ui.NewMatrix4Diagonal3Values(2, 4, 1)
	setState(func() {})
	if !ts.Binding.Scheduler().HasScheduledFrame() {
		t.Fatal("setState did not schedule a frame")
	}
	ts.Pump()
	if want := *ui.NewMatrix4TranslationValues(10, 20, 0).Multiply(transform); paintTransform != want {
		t.Errorf("paint transform = %v, want %v", paintTransform, want)
	}
	if regularChildSize != (ui.Size{Width: 40, Height: 50}) || theaterSize != (ui.Size{Width: 800, Height: 600}) {
		t.Errorf("sizes = %v, %v; want 40x50, 800x600", regularChildSize, theaterSize)
	}
	if want := ui.NewRectFromLTWH(10, 20, 80, 200); regularChildRectInTheater != want {
		t.Errorf("rect = %v, want %v", regularChildRectInTheater, want)
	}
}

func TestOverlayChildLayoutBuilderChildChangesSize(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	var setState StateSetter
	var regularChildSize ui.Size
	childSize := ui.Size{Width: 40, Height: 50}
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewPositioned(NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
			setState = setter
			return NewOverlayPortalOverlayChildLayoutBuilder(c1, func(_ BuildContext, info OverlayChildLayoutInfo) Widget {
				regularChildSize = info.ChildSize
				return NewSizedBox(nil)
			}, NewSizedBoxFromSize(childSize, nil))
		}), PositionedOpts{Left: foundation.Some(10.0), Top: foundation.Some(20.0)})
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	if regularChildSize != childSize {
		t.Fatalf("size = %v, want %v", regularChildSize, childSize)
	}
	setState(func() { childSize = ui.Size{Width: 123, Height: 321} })
	ts.Pump()
	if regularChildSize != childSize {
		t.Errorf("size = %v, want %v", regularChildSize, childSize)
	}
}

func TestOverlayChildLayoutBuilderBuilderIsCalledWhenOverlayPortalRebuilds(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	var setState StateSetter
	color := ui.NewColor(0x12345678)
	builder := func(BuildContext, OverlayChildLayoutInfo) Widget { return NewColoredBox(color, nil) }
	entry := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
		setState = setter
		return NewOverlayPortalOverlayChildLayoutBuilder(c1, builder, NewSizedBox(nil))
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	got := func() ui.Color { return ts.Widget(gluttertest.ByType[*ColoredBox]()).(*ColoredBox).Color() }
	if got() != color {
		t.Fatalf("color = %v, want %v", got(), color)
	}
	setState(func() { color = ui.NewColor(0x87654321) })
	ts.Pump()
	if got() != color {
		t.Errorf("color = %v, want %v", got(), color)
	}
}

func TestOverlayChildLayoutBuilderPositionedWorksInTheBuilder(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	key := foundation.NewValueKey("child")
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewOverlayPortalOverlayChildLayoutBuilder(c1, func(BuildContext, OverlayChildLayoutInfo) Widget {
			return NewPositioned(NewSizedBox(nil, SizedBoxOpts{Key: key}),
				PositionedOpts{Left: foundation.Some(123.0), Top: foundation.Some(37.0), Width: foundation.Some(12.0), Height: foundation.Some(23.0)})
		}, NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(10.0), Height: foundation.Some(20.0)}))
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	topLeft, size := ts.TopLeft(gluttertest.ByKey(key)), ts.Size(gluttertest.ByKey(key))
	if topLeft != (ui.Offset{Dx: 123, Dy: 37}) || size != (ui.Size{Width: 12, Height: 23}) {
		t.Errorf("rect = %v at %v, want 12x23 at (123, 37)", size, topLeft)
	}
}

func TestOverlayChildLayoutBuilderRebuildsWhenTheLayoutInfoChanges(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	var setState StateSetter
	transform := ui.NewMatrix4Identity()
	var paintTransform ui.Matrix4
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewPositioned(NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
			setState = setter
			return NewTransform(transform, NewOverlayPortalOverlayChildLayoutBuilder(c1, func(_ BuildContext, info OverlayChildLayoutInfo) Widget {
				paintTransform = info.ChildPaintTransform
				return NewSizedBox(nil)
			}, NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(40.0), Height: foundation.Some(50.0)})))
		}), PositionedOpts{Left: foundation.Some(10.0), Top: foundation.Some(20.0)})
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	if want := *ui.NewMatrix4TranslationValues(10, 20, 0); paintTransform != want {
		t.Fatalf("paint transform = %v, want %v", paintTransform, want)
	}
	transform = ui.NewMatrix4Diagonal3Values(2, 4, 1)
	setState(func() {})
	ts.Pump()
	if want := *ui.NewMatrix4TranslationValues(10, 20, 0).Multiply(transform); paintTransform != want {
		t.Errorf("paint transform = %v, want %v", paintTransform, want)
	}
}

func TestOverlayChildLayoutBuilderStillWorksIfChildIsNull(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	regularChildSize := ui.Size{Width: 1, Height: 1}
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewPositioned(NewOverlayPortalOverlayChildLayoutBuilder(c1, func(_ BuildContext, info OverlayChildLayoutInfo) Widget {
			regularChildSize = info.ChildSize
			return NewSizedBox(nil)
		}, nil), PositionedOpts{Left: foundation.Some(10.0), Top: foundation.Some(20.0)})
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	if regularChildSize != (ui.Size{}) {
		t.Errorf("size = %v, want zero", regularChildSize)
	}
}

// followerBox is a box with a RenderFollowerLayer, which makes the paint transform
// of its descendants unreliable.
type followerBox struct {
	SingleChildRenderObjectWidget
}

func (w *followerBox) CreateElement() Element { return NewSingleChildRenderObjectElement(w) }

func (w *followerBox) CreateRenderObject(BuildContext) rendering.RenderObject {
	return rendering.NewRenderFollowerLayer(rendering.NewLayerLink(), nil)
}

func TestOverlayChildLayoutBuilderScreamsIfRenderFollowerIsSpottedInPath(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return &followerBox{NewSingleChildRenderObjectWidgetBase(NewOverlayPortalOverlayChildLayoutBuilder(c1,
			func(BuildContext, OverlayChildLayoutInfo) Widget { return NewSizedBox(nil) }, nil))}
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	appErr := requireErrorTag(t, takeFirst(t, ts), "OverlayPaintTransformUnreliable")
	if !strings.Contains(appErr.Message, "RenderFollowerLayer") {
		t.Errorf("message = %q", appErr.Message)
	}
	ts.TakeErrors()
}
