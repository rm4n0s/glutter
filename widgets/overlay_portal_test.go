package widgets_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/semantics"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/overlay_portal_test.dart.
// Not ported: the semantics cases ('The overlay portal update semantics does not
// dirty overlay', the 'Semantics' group), 'Root location uses view boundary' (it
// needs several views), and the leak tracking settings.

// portalControllers are the four controllers that the Dart tests share. Each is
// told to show before it is attached to a portal, as the setUp of the tests does.
func portalControllers() (c1, c2, c3, c4 *OverlayPortalController) {
	c1 = NewOverlayPortalController(OverlayPortalControllerOpts{DebugLabel: "controller1"})
	c2 = NewOverlayPortalController(OverlayPortalControllerOpts{DebugLabel: "controller2"})
	c3 = NewOverlayPortalController(OverlayPortalControllerOpts{DebugLabel: "controller3"})
	c4 = NewOverlayPortalController(OverlayPortalControllerOpts{DebugLabel: "controller4"})
	for _, c := range []*OverlayPortalController{c1, c2, c3, c4} {
		c.Show()
	}
	return
}

func removeEntry(entry *OverlayEntry) {
	entry.Remove()
	entry.Dispose()
}

// statefulEntry is OverlayStatefulEntry.
func statefulEntry(builder StatefulBuilderFunc) *OverlayEntry {
	return NewOverlayEntry(func(BuildContext) Widget { return NewStatefulBuilder(builder) })
}

func emptyBox(BuildContext) Widget { return NewSizedBox(nil) }

// manyRelayoutBoundaries is _ManyRelayoutBoundaries: levels nested 50 x 50 boxes.
func manyRelayoutBoundaries(levels int, child Widget) Widget {
	result := child
	if levels > 1 {
		result = manyRelayoutBoundaries(levels-1, child)
	}
	return NewSizedBoxSquare(foundation.Some(50.0), result)
}

// verifyTreeIsClean checks that no render object of the tree needs layout.
func verifyTreeIsClean(t *testing.T, ts *gluttertest.Tester) {
	t.Helper()
	root := ts.Binding.Renderer().RenderViews()[0]
	var visit func(ro rendering.RenderObject)
	visit = func(ro rendering.RenderObject) {
		if ro.DebugNeedsLayout() {
			t.Errorf("%s is dirty", ro)
			return
		}
		ro.VisitChildren(visit)
	}
	visit(root)
}

func TestOverlayPortalOverlayChildSeesTheRightInheritedWidgets(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	buildCount := 0
	var seen ui.TextDirection
	textDirection := ui.TextDirectionRTL
	var setState StateSetter
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
			setState = setter
			return NewDirectionality(textDirection, NewOverlayPortal(c1, func(bc BuildContext) Widget {
				buildCount++
				seen = DirectionalityOf(bc)
				return NewSizedBox(nil)
			}, NewSizedBox(nil)))
		})
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	if buildCount != 1 || seen != ui.TextDirectionRTL {
		t.Fatalf("buildCount = %d, direction = %v; want 1, rtl", buildCount, seen)
	}
	setState(func() { textDirection = ui.TextDirectionLTR })
	ts.Pump()
	if buildCount != 2 || seen != ui.TextDirectionLTR {
		t.Errorf("buildCount = %d, direction = %v; want 2, ltr", buildCount, seen)
	}
}

func TestOverlayPortalRootOverlayChildGetsMediaQueryOfRootOverlay(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewOverlayPortalController()
	rootPadding := painting.NewEdgeInsetsAll(10)
	innerPadding := painting.NewEdgeInsetsAll(20)
	var got MediaQueryData
	var outer, inner *OverlayEntry
	outer = NewOverlayEntry(func(BuildContext) Widget {
		return NewMediaQuery(NewMediaQueryData().CopyWith(MediaQueryDataCopyOpts{Padding: foundation.Some(innerPadding)}),
			NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{inner}}))
	})
	inner = NewOverlayEntry(func(BuildContext) Widget {
		return NewOverlayPortal(controller, func(bc BuildContext) Widget {
			got = MediaQueryOf(bc)
			return NewSizedBox(nil)
		}, NewSizedBox(nil), OverlayPortalOpts{OverlayLocation: OverlayChildLocationRootOverlay})
	})
	defer removeEntry(outer)
	defer removeEntry(inner)
	ts.PumpWidget(NewMediaQuery(NewMediaQueryData().CopyWith(MediaQueryDataCopyOpts{Padding: foundation.Some(rootPadding)}),
		ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{outer}}))))
	controller.Show()
	ts.Pump()
	if got.Padding != rootPadding {
		t.Errorf("padding = %v, want %v", got.Padding, rootPadding)
	}
}

func TestOverlayPortalOverlayChildGetsMediaQueryOfTheOverlay(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewOverlayPortalController()
	padding := painting.NewEdgeInsetsAll(10)
	viewInsets := painting.NewEdgeInsetsFromLTRB(0, 0, 0, 300)
	viewPadding := painting.NewEdgeInsetsFromLTRB(0, 50, 0, 20)
	size := ui.Size{Width: 800, Height: 600}
	var got MediaQueryData
	entry := NewOverlayEntry(func(bc BuildContext) Widget {
		return NewMediaQuery(MediaQueryOf(bc).CopyWith(MediaQueryDataCopyOpts{
			Padding: foundation.Some(painting.EdgeInsetsZero), ViewInsets: foundation.Some(painting.EdgeInsetsZero), ViewPadding: foundation.Some(painting.EdgeInsetsZero),
		}), NewOverlayPortal(controller, func(bc BuildContext) Widget {
			got = MediaQueryOf(bc)
			return NewSizedBox(nil)
		}, NewSizedBox(nil)))
	})
	defer removeEntry(entry)
	ts.PumpWidget(NewMediaQuery(NewMediaQueryData().CopyWith(MediaQueryDataCopyOpts{
		Padding: foundation.Some(padding), ViewInsets: foundation.Some(viewInsets), ViewPadding: foundation.Some(viewPadding), Size: foundation.Some(size),
	}), ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}}))))
	controller.Show()
	ts.Pump()
	if got.Padding != padding || got.ViewInsets != viewInsets || got.ViewPadding != viewPadding || got.Size != size {
		t.Errorf("media query = padding %v, insets %v, view padding %v, size %v", got.Padding, got.ViewInsets, got.ViewPadding, got.Size)
	}
}

func TestOverlayPortalSafeToDeactivateAndReactivate(t *testing.T) {
	t.Run("whole overlay moves", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c1, _, _, _ := portalControllers()
		entry := NewOverlayEntry(func(BuildContext) Widget { return NewOverlayPortal(c1, emptyBox, NewSizedBox(nil)) })
		defer removeEntry(entry)
		widget := NewDirectionality(ui.TextDirectionLTR, NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}}),
			WidgetOpts{Key: NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "key"})})
		ts.PumpWidget(widget)
		ts.PumpWidget(NewSizedBox(widget))
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
	t.Run("portal moves in a column", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c1, _, _, _ := portalControllers()
		portal := NewOverlayPortal(c1, emptyBox, NewSizedBox(nil), OverlayPortalOpts{Key: NewGlobalKey[State]()})
		children := []Widget{portal, NewSizedBox(nil)}
		var setState StateSetter
		entry := NewOverlayEntry(func(BuildContext) Widget {
			return NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
				setState = setter
				return NewColumn(children)
			})
		})
		defer removeEntry(entry)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
		c1.Show()
		ts.Pump()
		setState(func() { children = []Widget{NewSizedBox(nil), portal} })
		ts.Pump()
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
}

func TestOverlayPortalSafeToHideOverlayChildAndRemoveOrReparent(t *testing.T) {
	// Regression test for https://github.com/flutter/flutter/issues/129025.
	newWidget := func(c1 *OverlayPortalController, entry **OverlayEntry) Widget {
		*entry = NewOverlayEntry(func(BuildContext) Widget { return NewOverlayPortal(c1, emptyBox, NewSizedBox(nil)) })
		return NewDirectionality(ui.TextDirectionLTR, NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{*entry}}),
			WidgetOpts{Key: NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "key"})})
	}
	t.Run("remove", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c1, _, _, _ := portalControllers()
		var entry *OverlayEntry
		widget := newWidget(c1, &entry)
		defer removeEntry(entry)
		c1.Show()
		ts.PumpWidget(widget)
		c1.Hide()
		ts.PumpWidget(NewSizedBox(nil))
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
	t.Run("reparent the portal", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c1, _, _, _ := portalControllers()
		portal := NewOverlayPortal(c1, emptyBox, NewSizedBox(nil), OverlayPortalOpts{Key: NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "key"})})
		children := []Widget{NewSizedBox(nil), portal}
		var setState StateSetter
		entry := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
			setState = setter
			return NewColumn(children)
		})
		defer removeEntry(entry)
		widget := ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}}))
		c1.Show()
		ts.PumpWidget(widget)
		c1.Hide()
		setState(func() { children = []Widget{portal, NewSizedBox(nil)} })
		ts.PumpWidget(widget)
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
	t.Run("reparent the overlay", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c1, _, _, _ := portalControllers()
		var entry *OverlayEntry
		widget := newWidget(c1, &entry)
		defer removeEntry(entry)
		c1.Show()
		ts.PumpWidget(widget)
		c1.Hide()
		ts.PumpWidget(NewSizedBox(widget))
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
}

// Regression test for https://github.com/flutter/flutter/issues/133545.
func TestOverlayPortalOverlayChildIsReachableWithoutRelayoutBoundary(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	key := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "key"})
	entry := NewOverlayEntry(func(BuildContext) Widget {
		// The Positioned widget prevents a relayout boundary from being introduced
		// between the Overlay and the OverlayPortal.
		return NewPositioned(NewOverlayPortal(c1, func(BuildContext) Widget { return NewSizedBox(nil, SizedBoxOpts{Key: key}) }, NewSizedBox(nil)),
			PositionedOpts{Top: foundation.Some(0.0), Left: foundation.Some(0.0)})
	})
	defer removeEntry(entry)
	c1.Hide()
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	c1.Show()
	ts.Pump()
	if got := ts.Count(gluttertest.ByKey(key)); got != 1 {
		t.Errorf("%d overlay children found, want 1", got)
	}
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
	verifyTreeIsClean(t, ts)
}

func TestOverlayPortalSameControllerOnTwoPortalsPanics(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewOverlayPortalController(OverlayPortalControllerOpts{DebugLabel: "local controller"})
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewColumn([]Widget{
			NewOverlayPortal(controller, emptyBox, NewSizedBox(nil)),
			NewOverlayPortal(controller, emptyBox, NewSizedBox(nil)),
		})
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	err := takeFirst(t, ts)
	appErr := requireErrorTag(t, err, "OverlayPortalControllerAttached")
	if !strings.Contains(appErr.Message, "failed to attach") || !strings.Contains(appErr.Message, "already attached to") {
		t.Errorf("message = %q", appErr.Message)
	}
	ts.TakeErrors()
}

// Regression test for https://github.com/flutter/flutter/issues/153903.
func TestOverlayPortalSizesItselfWithUnboundedConstraints(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	size := ui.Size{Width: 40, Height: 40}
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewOverlayPortal(c1, emptyBox, NewSizedBoxFromSize(size, nil))
	}, OverlayEntryOpts{CanSizeOverlay: true})
	defer removeEntry(entry)
	c1.Show()
	ts.PumpWidget(NewDirectionality(ui.TextDirectionLTR, NewUnconstrainedBox(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}}))))
	if got := ts.Size(gluttertest.ByType[*Overlay]()); got != size {
		t.Errorf("overlay size = %v, want %v", got, size)
	}
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
}

func TestOverlayPortalShowAndHide(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewOverlayPortalController(OverlayPortalControllerOpts{DebugLabel: "local controller"})
	target := NewSizedBox(nil)
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewOverlayPortal(controller, func(BuildContext) Widget { return target }, nil)
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	steps := []struct {
		name  string
		do    func()
		found int
	}{
		{"initially", func() {}, 0},
		{"pump", func() { ts.Pump() }, 0},
		{"show", func() { controller.Show(); ts.Pump() }, 1},
		{"hide", func() { controller.Hide(); ts.Pump() }, 0},
		{"show again", func() { controller.Show(); ts.Pump() }, 1},
	}
	for _, step := range steps {
		step.do()
		if got := ts.Count(gluttertest.ByWidget(target)); got != step.found {
			t.Errorf("%s: %d overlay children, want %d", step.name, got, step.found)
		}
	}
}

func TestOverlayPortalOverlayChildBuilderIsNotCalledUntilShow(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewOverlayPortalController(OverlayPortalControllerOpts{DebugLabel: "local controller"})
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewOverlayPortal(controller, func(BuildContext) Widget { panic("unreachable") }, NewSizedBox(nil))
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
}

func TestOverlayPortalOverlayChildCanUsePositioned(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	dimensions := 30.0
	var setState StateSetter
	entry := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
		setState = setter
		return NewOverlayPortal(c1, func(BuildContext) Widget {
			return NewPositioned(NewPlaceholder(nil), PositionedOpts{Width: foundation.Some(dimensions), Height: foundation.Some(dimensions)})
		}, NewSizedBox(nil))
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	for _, want := range []float64{30, 50} {
		if got := ts.TopLeft(gluttertest.ByType[*Placeholder]()); got != (ui.Offset{}) {
			t.Errorf("top left = %v, want zero", got)
		}
		if got := ts.Size(gluttertest.ByType[*Placeholder]()); got != (ui.Size{Width: want, Height: want}) {
			t.Errorf("size = %v, want %vx%v", got, want, want)
		}
		setState(func() { dimensions = 50 })
		ts.Pump()
	}
}

func TestOverlayPortalOverlayChildCanBeHitTested(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	offset := 0.0
	var setState StateSetter
	isHit := false
	entry := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
		setState = setter
		return NewOverlayPortal(c1, func(BuildContext) Widget {
			return NewPositioned(NewGestureDetector(nil, GestureDetectorOpts{OnTap: func() { isHit = true }, Behavior: rendering.HitTestBehaviorOpaque}),
				PositionedOpts{Left: foundation.Some(offset), Top: foundation.Some(offset), Width: foundation.Some(1.0), Height: foundation.Some(1.0)})
		}, NewSizedBox(nil))
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	ts.TapAt(ui.Offset{Dx: 0.5, Dy: 0.5})
	if !isHit {
		t.Fatal("the overlay child was not hit at its first place")
	}
	isHit = false
	setState(func() { offset = 50 })
	ts.Pump()
	ts.TapAt(ui.Offset{Dx: 0.5, Dy: 0.5})
	if isHit {
		t.Error("the overlay child was hit where it no longer is")
	}
	ts.TapAt(ui.Offset{Dx: 50.5, Dy: 50.5})
	if !isHit {
		t.Error("the overlay child was not hit at its new place")
	}
}

func TestOverlayPortalWorksInALayoutBuilder(t *testing.T) {
	t.Run("1", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c1, _, _, _ := portalControllers()
		entry := NewOverlayEntry(func(BuildContext) Widget {
			return NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget {
				return NewOverlayPortal(c1, emptyBox, NewSizedBox(nil))
			})
		})
		defer removeEntry(entry)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
	layoutBuilder := func(c2 *OverlayPortalController) func(BuildContext, rendering.BoxConstraints) Widget {
		return func(BuildContext, rendering.BoxConstraints) Widget {
			return NewOverlayPortal(c2, emptyBox, NewSizedBox(nil))
		}
	}
	t.Run("2", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c1, c2, _, _ := portalControllers()
		var setState StateSetter
		shouldShowChild := false
		entry := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
			setState = setter
			var child Widget
			if shouldShowChild {
				child = NewLayoutBuilder(layoutBuilder(c2))
			}
			return NewOverlayPortal(c1, emptyBox, child)
		})
		defer removeEntry(entry)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Fatalf("errors: %v", errs)
		}
		setState(func() { shouldShowChild = true })
		ts.Pump()
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
	t.Run("3", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c1, c2, _, _ := portalControllers()
		var setState StateSetter
		shouldShowChild := false
		c1.Hide()
		c2.Hide()
		entry := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
			setState = setter
			var child Widget
			if shouldShowChild {
				child = NewLayoutBuilder(layoutBuilder(c2))
			}
			// The Positioned widget ensures there is no relayout boundary between the
			// Overlay and the OverlayPortal.
			return NewPositioned(NewOverlayPortal(c1, emptyBox, child), PositionedOpts{Top: foundation.Some(0.0), Left: foundation.Some(0.0)})
		})
		defer removeEntry(entry)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
		c1.Show()
		c2.Show()
		setState(func() { shouldShowChild = true })
		ts.Pump()
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("errors: %v", errs)
		}
	})
}

func TestOverlayPortalWithoutAnOverlayPanics(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	c1.Show()
	ts.PumpWidget(ltr(NewSizedBoxSquare(foundation.Some(50.0), NewOverlayPortal(c1, emptyBox, NewSizedBox(nil)))))
	appErr := requireErrorTag(t, takeFirst(t, ts), "OverlayMissing")
	if !strings.Contains(appErr.Message, "OverlayPortal widgets require an Overlay widget ancestor") {
		t.Errorf("message = %q", appErr.Message)
	}
	ts.TakeErrors()
}

// renderBoxAdapter is WidgetToRenderBoxAdapter: a leaf whose render object is the
// given box.
type renderBoxAdapter struct {
	LeafRenderObjectWidget
	box rendering.Box
}

// newRenderBoxAdapter has a GlobalObjectKey of its own, like the Dart widget, so
// that the box can be reparented.
func newRenderBoxAdapter(box rendering.Box) *renderBoxAdapter {
	return &renderBoxAdapter{NewLeafRenderObjectWidgetBase(WidgetOpts{Key: NewGlobalObjectKey[State](box)}), box}
}

func (w *renderBoxAdapter) CreateRenderObject(BuildContext) rendering.RenderObject { return w.box }

// rebuildLayoutBuilderSubtree marks the closest LayoutBuilder above the box for
// build.
func rebuildLayoutBuilderSubtree(t *testing.T, ts *gluttertest.Tester, descendant rendering.RenderObject) {
	t.Helper()
	for node := descendant.Parent(); node != nil; node = node.Parent() {
		for _, e := range ts.Elements(gluttertest.ByType[*LayoutBuilder]()) {
			if e.FindRenderObject() == node {
				e.MarkNeedsBuild()
				return
			}
		}
	}
	t.Fatal("no LayoutBuilder above the render object")
}

func TestOverlayPortalWidgetIsLaidOutBeforeOverlayChild(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	widgetKey := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "widget"})
	childBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
	overlayChildBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
	layoutCount := 0
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return manyRelayoutBoundaries(50, NewBuilder(func(BuildContext) Widget {
			return NewOverlayPortal(c1, func(BuildContext) Widget {
				return NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget {
					surrogate := widgetKey.CurrentContext().FindRenderObject()
					if surrogate.DebugNeedsLayout() && !surrogate.AsRenderObject().DebugDoingThisLayout() {
						t.Error("the layout surrogate is dirty and not laying out when the overlay child builds")
					}
					layoutCount++
					return newRenderBoxAdapter(overlayChildBox)
				})
			}, newRenderBoxAdapter(childBox), OverlayPortalOpts{Key: widgetKey})
		}))
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	if layoutCount != 1 {
		t.Fatalf("layoutCount = %d, want 1", layoutCount)
	}
	// Make the render object of the widget dirty, and check in the callback of the
	// LayoutBuilder that it is already laid out.
	widgetKey.CurrentContext().FindRenderObject().MarkNeedsLayout()
	// Dirty both render subtree branches.
	childBox.MarkNeedsLayout()
	rebuildLayoutBuilderSubtree(t, ts, overlayChildBox)
	// Make sure that the depth of childBox is greater than that of the overlay
	// child, and that its parent is not dirty (childBox is a dirty relayout
	// boundary).
	if widgetKey.CurrentContext().FindRenderObject().Depth() >= overlayChildBox.Depth() {
		t.Error("the depth of the portal is not less than that of the overlay child")
	}
	ts.Pump()
	if layoutCount != 2 {
		t.Errorf("layoutCount = %d, want 2", layoutCount)
	}
	verifyTreeIsClean(t, ts)
}

// pumpToLayout builds and lays out the tree without painting, like
// tester.pump(null, EnginePhase.layout).
func pumpToLayout(ts *gluttertest.Tester) {
	ts.FlushPosted()
	b := ts.Binding
	b.BuildOwner().BuildScope(b.RootElement(), nil)
	b.Renderer().RootPipelineOwner().FlushLayout()
}

// layoutCounter is _RenderLayoutCounter: it counts the layouts that its parent
// asks for, and the ones that it does itself as a relayout boundary.
type layoutCounter struct {
	rendering.RenderProxyBox
	layoutCount       int
	parentDoingLayout bool
}

func newLayoutCounter() *layoutCounter {
	r := &layoutCounter{}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

func (r *layoutCounter) Layout(constraints rendering.Constraints, opts ...rendering.LayoutOpts) {
	if r.parentDoingLayout {
		panic("the parent was already doing layout")
	}
	r.parentDoingLayout = true
	r.layoutCount++
	r.RenderProxyBox.Layout(constraints, opts...)
	r.parentDoingLayout = false
}

func (r *layoutCounter) PerformLayout() {
	r.RenderProxyBox.PerformLayout()
	if !r.parentDoingLayout {
		r.layoutCount++
	}
}

// Regression test for https://github.com/flutter/flutter/issues/174133. A Table
// adopts the render objects of its children only after every row has been
// mounted, and an OverlayPortal cell can mount its overlay child in that window,
// before the layout surrogate has been adopted by its parent.
func TestOverlayPortalInsideATableRowDoesNotCrash(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewOverlayPortalController()
	controller.Show()
	overlayKey := foundation.NewValueKey("overlay-child")
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewTable([]TableRow{NewTableRow([]Widget{
			NewOverlayPortal(controller, func(BuildContext) Widget {
				return NewAlign(NewSizedBox(nil, SizedBoxOpts{Key: overlayKey, Width: foundation.Some(10.0), Height: foundation.Some(10.0)}),
					AlignOpts{Alignment: painting.AlignmentTopLeft})
			}, NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(10.0), Height: foundation.Some(10.0)})),
		})})
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("errors: %v", errs)
	}
	if got := ts.Count(gluttertest.ByKey(overlayKey)); got != 1 {
		t.Fatalf("%d overlay children, want 1", got)
	}
	// Confirm that the overlay child completed layout: the depth invariant must be
	// restored for the deferred layout box to be laid out.
	if got := ts.Size(gluttertest.ByKey(overlayKey)); got != (ui.Size{Width: 10, Height: 10}) {
		t.Errorf("size = %v, want 10x10", got)
	}
}

func TestOverlayPortalAddingOrRemovingOverlayChildDoesNotRedirtyTheOverlayMoreThanOnce(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	widgetKey := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "widget"})
	overlayKey := NewGlobalKey[*OverlayState](GlobalKeyOpts{DebugLabel: "overlay"})
	childBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
	overlayChildBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
	counter := newLayoutCounter()
	layoutCount := 0
	c1.Hide()
	entry1 := NewOverlayEntry(func(BuildContext) Widget { return newRenderBoxAdapter(counter) })
	entry2 := NewOverlayEntry(func(BuildContext) Widget {
		return manyRelayoutBoundaries(50, NewBuilder(func(BuildContext) Widget {
			return NewOverlayPortal(c1, func(BuildContext) Widget {
				return NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget {
					layoutCount++
					if ts.RenderObject(gluttertest.ByType[*Overlay]()).DebugNeedsLayout() {
						t.Error("the overlay needs layout when the overlay child builds")
					}
					return newRenderBoxAdapter(overlayChildBox)
				})
			}, newRenderBoxAdapter(childBox), OverlayPortalOpts{Key: widgetKey})
		}))
	})
	defer removeEntry(entry1)
	defer removeEntry(entry2)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{Key: overlayKey, InitialEntries: []*OverlayEntry{entry1, entry2}})))
	steps := []struct {
		name                           string
		do                             func()
		wantLayoutCount, wantOverlayed int
	}{
		{"initial", func() {}, 0, 1},
		{"add the overlay child", func() { c1.Show(); ts.Pump() }, 1, 1},
		{"remove the overlay child", func() { c1.Hide(); ts.Pump() }, 1, 1},
	}
	for _, step := range steps {
		step.do()
		if layoutCount != step.wantLayoutCount || counter.layoutCount != step.wantOverlayed {
			t.Errorf("%s: layoutCount = %d, overlay counter = %d; want %d, %d", step.name, layoutCount, counter.layoutCount, step.wantLayoutCount, step.wantOverlayed)
		}
		verifyTreeIsClean(t, ts)
	}
}

// Regression test for https://github.com/flutter/flutter/issues/134656.
func TestOverlayPortalAddingOrRemovingOverlayChildCausesRepaint(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start func(c *OverlayPortalController)
		flip  func(c *OverlayPortalController)
	}{
		{"adding a child", (*OverlayPortalController).Hide, (*OverlayPortalController).Show},
		{"removing a child", (*OverlayPortalController).Show, (*OverlayPortalController).Hide},
	} {
		t.Run(tc.name, func(t *testing.T) {
			skipUnlessDebug(t)
			ts := gluttertest.New(t, gluttertest.Opts{})
			c1, _, _, _ := portalControllers()
			childKey := foundation.NewValueKey("child")
			entry := NewOverlayEntry(func(BuildContext) Widget {
				return NewRepaintBoundary(NewOverlayPortal(c1, emptyBox, NewSizedBox(nil, SizedBoxOpts{Key: childKey})))
			})
			defer removeEntry(entry)
			tc.start(c1)
			ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
			theater := ts.RenderObject(gluttertest.ByType[*Overlay]())
			child := ts.RenderObject(gluttertest.ByKey(childKey))
			if theater.AsRenderObject().DebugNeedsPaint() || child.AsRenderObject().DebugNeedsPaint() {
				t.Fatal("the render objects need paint before the change")
			}
			tc.flip(c1)
			pumpToLayout(ts)
			if !theater.AsRenderObject().DebugNeedsPaint() {
				t.Error("the theater does not need paint")
			}
			if child.AsRenderObject().DebugNeedsPaint() {
				t.Error("the child needs paint")
			}
			// Discard the dirty render tree.
			ts.PumpWidget(NewSizedBox(nil))
		})
	}
}

func TestOverlayPortalAddingAndRemovingInALayoutBuilderDuringLayout(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	c1.Hide()
	var setState StateSetter
	size := ui.Size{}
	overlayPortal := NewOverlayPortal(c1, func(BuildContext) Widget { return NewPlaceholder(nil) }, NewPlaceholder(nil),
		OverlayPortalOpts{Key: NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "widget"})})
	entry := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
		setState = setter
		return NewCenter(NewSizedBoxFromSize(size, NewLayoutBuilder(func(_ BuildContext, constraints rendering.BoxConstraints) Widget {
			// This layout callback adds or removes an OverlayPortal during layout.
			if constraints.MaxHeight > 0 {
				return overlayPortal
			}
			return NewSizedBox(nil)
		})))
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{Key: NewGlobalKey[*OverlayState](), InitialEntries: []*OverlayEntry{entry}})))
	c1.Show()
	ts.Pump()
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Fatalf("errors: %v", errs)
	}
	for _, step := range []struct {
		name string
		size ui.Size
	}{{"adds the portal", ui.Size{Width: 300, Height: 300}}, {"removes the portal", ui.Size{}}} {
		setState(func() { size = step.size })
		ts.Pump()
		if errs := ts.TakeErrors(); len(errs) != 0 {
			t.Errorf("%s: errors: %v", step.name, errs)
		}
	}
}

func TestOverlayPortalChangeOverlayConstraints(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	widgetKey := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "widget outer"})
	overlayKey := NewGlobalKey[*OverlayState](GlobalKeyOpts{DebugLabel: "overlay"})
	childBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
	overlayChildBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
	counter := newLayoutCounter()
	layoutCount := 0
	var setState StateSetter
	dimension := 100.0
	entry1 := NewOverlayEntry(func(BuildContext) Widget { return newRenderBoxAdapter(counter) })
	entry2 := NewOverlayEntry(func(BuildContext) Widget {
		return NewCenter(manyRelayoutBoundaries(50, NewBuilder(func(BuildContext) Widget {
			return NewOverlayPortal(c1, func(BuildContext) Widget {
				return NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget {
					layoutCount++
					// Both overlays need to be clean at this point.
					for _, e := range ts.Elements(gluttertest.ByType[*Overlay]()) {
						ro := e.FindRenderObject()
						if ro.DebugNeedsLayout() && !ro.AsRenderObject().DebugDoingThisLayout() {
							t.Error("an overlay needs layout when the overlay child builds")
						}
					}
					return newRenderBoxAdapter(overlayChildBox)
				})
			}, newRenderBoxAdapter(childBox), OverlayPortalOpts{Key: widgetKey})
		})))
	})
	defer removeEntry(entry1)
	defer removeEntry(entry2)
	ts.PumpWidget(NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
		setState = setter
		return NewCenter(ltr(NewSizedBoxSquare(foundation.Some(dimension),
			NewOverlay(OverlayOpts{Key: overlayKey, InitialEntries: []*OverlayEntry{entry1, entry2}}))))
	}))
	if layoutCount != 1 || counter.layoutCount != 1 {
		t.Fatalf("layoutCount = %d, counter = %d; want 1, 1", layoutCount, counter.layoutCount)
	}
	if childBox.Size() != (ui.Size{Width: 50, Height: 50}) || overlayChildBox.Size() != (ui.Size{Width: 100, Height: 100}) {
		t.Fatalf("sizes = %v, %v; want 50x50, 100x100", childBox.Size(), overlayChildBox.Size())
	}
	verifyTreeIsClean(t, ts)
	// The incoming constraints changed.
	setState(func() { dimension = 150 })
	ts.Pump()
	if childBox.Size() != (ui.Size{Width: 50, Height: 50}) || overlayChildBox.Size() != (ui.Size{Width: 150, Height: 150}) {
		t.Errorf("sizes = %v, %v; want 50x50, 150x150", childBox.Size(), overlayChildBox.Size())
	}
	if layoutCount != 2 || counter.layoutCount != 2 {
		t.Errorf("layoutCount = %d, counter = %d; want 2, 2", layoutCount, counter.layoutCount)
	}
	verifyTreeIsClean(t, ts)
}

// ancestorTheaters are the render objects of the Overlays that are ancestors of
// the render object, closest first (_ancestorRenderTheaters).
func ancestorTheaters(ts *gluttertest.Tester, ro rendering.RenderObject) []rendering.RenderObject {
	theaters := map[rendering.RenderObject]bool{}
	for _, e := range ts.Elements(gluttertest.ByType[*Overlay]()) {
		theaters[e.FindRenderObject()] = true
	}
	var out []rendering.RenderObject
	for node := ro; node != nil; node = node.Parent() {
		if theaters[node] {
			out = append(out, node)
		}
	}
	return out
}

func singleTheater(t *testing.T, ts *gluttertest.Tester, ro rendering.RenderObject) rendering.RenderObject {
	t.Helper()
	theaters := ancestorTheaters(ts, ro)
	if len(theaters) != 1 {
		t.Fatalf("%d ancestor theaters, want 1", len(theaters))
	}
	return theaters[0]
}

func cleanOverlaysCheck(t *testing.T, ts *gluttertest.Tester) {
	t.Helper()
	// Both overlays need to be clean at this point.
	for _, e := range ts.Elements(gluttertest.ByType[*Overlay]()) {
		ro := e.FindRenderObject()
		if ro.DebugNeedsLayout() && !ro.AsRenderObject().DebugDoingThisLayout() {
			t.Error("an overlay needs layout when the overlay child builds")
		}
	}
}

func TestOverlayPortalCanTargetTheRootOverlay(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	widgetKey := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "widget outer"})
	rootOverlayKey := NewGlobalKey[*OverlayState](GlobalKeyOpts{DebugLabel: "root overlay"})
	localOverlayKey := NewGlobalKey[*OverlayState](GlobalKeyOpts{DebugLabel: "local overlay"})
	childBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
	overlayChildBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
	counter := newLayoutCounter()
	layoutCount := 0
	location := OverlayChildLocationNearestOverlay
	var setState StateSetter
	var entries []*OverlayEntry
	entry := func(builder WidgetBuilder) *OverlayEntry {
		e := NewOverlayEntry(builder)
		entries = append(entries, e)
		return e
	}
	defer func() {
		// The builders run again on every rebuild and make entries that are never
		// inserted, which are only dropped.
		for _, e := range entries {
			if e.Mounted() {
				removeEntry(e)
			}
		}
	}()
	// This tree has 3 nested Overlays.
	ts.PumpWidget(NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
		setState = setter
		return NewCenter(ltr(NewOverlay(OverlayOpts{Key: rootOverlayKey, InitialEntries: []*OverlayEntry{entry(func(BuildContext) Widget {
			return NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry(func(BuildContext) Widget {
				return NewOverlay(OverlayOpts{Key: localOverlayKey, InitialEntries: []*OverlayEntry{
					// Overlay.performLayout calls counter.layout.
					entry(func(BuildContext) Widget { return newRenderBoxAdapter(counter) }),
					entry(func(BuildContext) Widget {
						return NewCenter(NewBuilder(func(BuildContext) Widget {
							return NewOverlayPortal(c1, func(BuildContext) Widget {
								return NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget {
									layoutCount++
									cleanOverlaysCheck(t, ts)
									return newRenderBoxAdapter(overlayChildBox)
								})
							}, newRenderBoxAdapter(childBox), OverlayPortalOpts{Key: widgetKey, OverlayLocation: location})
						}))
					}),
				}})
			})}})
		})}})))
	}))
	if layoutCount != 1 || counter.layoutCount != 1 {
		t.Fatalf("layoutCount = %d, counter = %d; want 1, 1", layoutCount, counter.layoutCount)
	}
	if got := len(ancestorTheaters(ts, overlayChildBox)); got != 3 {
		t.Fatalf("%d ancestor theaters, want 3", got)
	}
	verifyTreeIsClean(t, ts)
	// Now the portal targets the root overlay.
	setState(func() { location = OverlayChildLocationRootOverlay })
	ts.Pump()
	if layoutCount != 2 || counter.layoutCount != 1 {
		t.Errorf("layoutCount = %d, counter = %d; want 2, 1", layoutCount, counter.layoutCount)
	}
	if got := singleTheater(t, ts, overlayChildBox); got != ts.RenderObject(gluttertest.ByKey(rootOverlayKey)) {
		t.Error("the overlay child is not in the root overlay")
	}
	verifyTreeIsClean(t, ts)
	_ = localOverlayKey
}

// overlaySwaps is OverlaySwaps: its Overlay is replaced by another, with another
// entry, when swaps is called.
type overlaySwaps struct {
	StatefulWidget
	entry, entryAfterSwap *OverlayEntry
}

func (w *overlaySwaps) CreateState() State { return &overlaySwapsState{} }

type overlaySwapsState struct {
	StateOf[*overlaySwaps]
	overlayKey foundation.Key
	entry      *OverlayEntry
}

func (s *overlaySwapsState) InitState() {
	s.overlayKey = foundation.NewUniqueKey()
	s.entry = s.Widget().entry
}

func (s *overlaySwapsState) swaps() {
	s.SetState(func() {
		s.overlayKey = foundation.NewUniqueKey()
		s.entry = s.Widget().entryAfterSwap
	})
}

func (s *overlaySwapsState) Build(BuildContext) Widget {
	return NewOverlay(OverlayOpts{Key: s.overlayKey, InitialEntries: []*OverlayEntry{s.entry}})
}

func TestOverlayPortalListensToOverlayChanges(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	// The global key makes the container be reparented instead of destroyed when
	// the Overlay is swapped.
	container := NewGlobalObjectKey[State]("container")
	controller := NewOverlayPortalController()
	overlayPortal := foundation.NewUniqueKey()
	overlayBody := NewSizedBoxSquare(foundation.Some(100.0), NewOverlayPortal(controller, func(BuildContext) Widget {
		return NewPlaceholder(nil, PlaceholderOpts{Key: overlayPortal})
	}, nil))
	entry1 := NewOverlayEntry(func(BuildContext) Widget { return NewContainer(overlayBody, ContainerOpts{Key: container}) })
	entry2 := NewOverlayEntry(func(BuildContext) Widget { return NewContainer(overlayBody, ContainerOpts{Key: container}) })
	defer removeEntry(entry1)
	defer removeEntry(entry2)
	ts.PumpWidget(ltr(&overlaySwaps{StatefulWidget: NewStatefulWidgetBase(), entry: entry1, entryAfterSwap: entry2}))
	controller.Show()
	ts.Pump()
	parentTheater := singleTheater(t, ts, ts.RenderObject(gluttertest.ByKey(overlayPortal)))
	gluttertest.StateOf[*overlaySwapsState](ts, gluttertest.ByType[*overlaySwaps]()).swaps()
	ts.Pump()
	newParentTheater := singleTheater(t, ts, ts.RenderObject(gluttertest.ByKey(overlayPortal)))
	if parentTheater == newParentTheater {
		t.Error("the overlay child is still in the old theater")
	}
}

func TestOverlayPortalListensToRootOverlayChanges(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	oldRoot := NewGlobalKey[*OverlayState]()
	newRoot := NewGlobalKey[*OverlayState]()
	overlayPortal := foundation.NewUniqueKey()
	overlayBody := NewSizedBoxSquare(foundation.Some(100.0), NewOverlayPortal(c1, func(BuildContext) Widget {
		return NewPlaceholder(nil, PlaceholderOpts{Key: overlayPortal})
	}, nil, OverlayPortalOpts{OverlayLocation: OverlayChildLocationRootOverlay}))
	innerEntry := NewOverlayEntry(func(BuildContext) Widget { return NewContainer(overlayBody) })
	midEntry := NewOverlayEntry(func(BuildContext) Widget {
		return NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{innerEntry}})
	})
	defer removeEntry(innerEntry)
	defer removeEntry(midEntry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{Key: oldRoot, InitialEntries: []*OverlayEntry{midEntry}})))
	parentTheater := singleTheater(t, ts, ts.RenderObject(gluttertest.ByKey(overlayPortal)))
	if parentTheater != oldRoot.CurrentContext().FindRenderObject() {
		t.Fatal("the overlay child is not in the old root")
	}
	outerEntry := NewOverlayEntry(func(BuildContext) Widget {
		return NewOverlay(OverlayOpts{Key: oldRoot, InitialEntries: []*OverlayEntry{midEntry}})
	})
	defer removeEntry(outerEntry)
	// Add a new root.
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{Key: newRoot, InitialEntries: []*OverlayEntry{outerEntry}})))
	parentTheater = singleTheater(t, ts, ts.RenderObject(gluttertest.ByKey(overlayPortal)))
	if parentTheater != newRoot.CurrentContext().FindRenderObject() {
		t.Error("the overlay child is not in the new root")
	}
}

func TestOverlayPortalControllerCanBeAssignedToAnotherAfterDeactivate(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewOverlayPortalController()
	overlayKey := NewGlobalKey[*OverlayState]()
	portal := func(BuildContext) Widget {
		return NewOverlayPortal(controller, func(BuildContext) Widget { return NewPlaceholder(nil) }, nil)
	}
	entry1 := NewOverlayEntry(portal)
	entry2 := NewOverlayEntry(portal)
	defer func() {
		removeEntry(entry1)
		entry2.Dispose()
	}()
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{Key: overlayKey, InitialEntries: []*OverlayEntry{entry1}})))
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{Key: overlayKey, InitialEntries: []*OverlayEntry{entry2}})))
	verifyTreeIsClean(t, ts)
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
}

func TestOverlayPortalReactivationMaintainsPortalState(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewOverlayPortalController()
	portalKey := NewGlobalKey[State]()
	portal := func(BuildContext) Widget {
		return NewOverlayPortal(controller, func(BuildContext) Widget { return NewPlaceholder(nil) }, nil, OverlayPortalOpts{Key: portalKey})
	}
	entry1 := NewOverlayEntry(portal)
	entry2 := NewOverlayEntry(portal)
	defer removeEntry(entry1)
	defer removeEntry(entry2)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry1}})))
	controller.Show()
	ts.PumpWidget(ltr(NewSizedBox(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry2}}))))
	if got := ts.Count(gluttertest.ByType[*Placeholder]()); got != 1 {
		t.Errorf("%d placeholders, want 1", got)
	}
	if !controller.IsShowing() {
		t.Error("the controller is not showing")
	}
}

func TestOverlayPortalAttachTargetIsRestoredAfterReparenting(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	portalKey := NewGlobalKey[State]()
	childBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
	overlayChildBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
	moveToSecondOverlay := false
	child := newRenderBoxAdapter(childBox)
	overlayChild := newRenderBoxAdapter(overlayChildBox)
	portal := func() Widget {
		return NewOverlayPortal(c1, func(BuildContext) Widget { return overlayChild }, child, OverlayPortalOpts{Key: portalKey})
	}
	entry1 := NewOverlayEntry(func(BuildContext) Widget {
		if !moveToSecondOverlay {
			return portal()
		}
		return NewSizedBox(nil)
	})
	entry2 := NewOverlayEntry(func(BuildContext) Widget {
		if moveToSecondOverlay {
			return portal()
		}
		return NewSizedBox(nil)
	})
	defer removeEntry(entry1)
	defer removeEntry(entry2)
	ts.PumpWidget(ltr(NewStack([]Widget{
		NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry1}}),
		NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry2}}),
	})))
	// Move to the second overlay.
	moveToSecondOverlay = true
	entry1.MarkNeedsBuild()
	entry2.MarkNeedsBuild()
	ts.Pump()
	verifyTreeIsClean(t, ts)
}

// verifyOverlayChildReadyForLayout checks that the layout surrogate of the portal is
// laid out, or laying out, when the overlay child is built.
func verifyOverlayChildReadyForLayout(t *testing.T, key *GlobalKey[State]) {
	t.Helper()
	surrogate := key.CurrentContext().FindRenderObject()
	if surrogate.DebugNeedsLayout() && !surrogate.AsRenderObject().DebugDoingThisLayout() {
		t.Error("the layout surrogate is dirty and not laying out when the overlay child builds")
	}
}

func TestOverlayPortalGlobalKeyReparenting(t *testing.T) {
	t.Run("child is laid out before overlay child after OverlayEntry shuffle", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c1, _, _, _ := portalControllers()
		layoutCount := 0
		widgetKey := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "widget"})
		childBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
		overlayChildBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
		entry1 := NewOverlayEntry(func(BuildContext) Widget {
			return manyRelayoutBoundaries(50, NewBuilder(func(BuildContext) Widget {
				return NewOverlayPortal(c1, func(BuildContext) Widget {
					return NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget {
						layoutCount++
						verifyOverlayChildReadyForLayout(t, widgetKey)
						return newRenderBoxAdapter(overlayChildBox)
					})
				}, newRenderBoxAdapter(childBox), OverlayPortalOpts{Key: widgetKey})
			}))
		})
		entry2 := NewOverlayEntry(func(BuildContext) Widget { return NewPlaceholder(nil) })
		entry3 := NewOverlayEntry(func(BuildContext) Widget { return NewPlaceholder(nil) })
		defer removeEntry(entry1)
		defer removeEntry(entry2)
		defer removeEntry(entry3)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry1, entry2, entry3}})))
		if layoutCount != 1 {
			t.Fatalf("layoutCount = %d, want 1", layoutCount)
		}
		verifyTreeIsClean(t, ts)
		widgetKey.CurrentContext().FindRenderObject().MarkNeedsLayout()
		childBox.MarkNeedsLayout()
		rebuildLayoutBuilderSubtree(t, ts, overlayChildBox)
		// Make sure that the depth of childBox is greater than that of the overlay child.
		if widgetKey.CurrentContext().FindRenderObject().Depth() >= overlayChildBox.Depth() {
			t.Error("the depth of the portal is not less than that of the overlay child before the shuffle")
		}
		gluttertest.StateOf[*OverlayState](ts, gluttertest.ByType[*Overlay]()).Rearrange([]*OverlayEntry{entry3, entry2, entry1})
		ts.Pump()
		if layoutCount != 2 {
			t.Errorf("layoutCount = %d, want 2", layoutCount)
		}
		if widgetKey.CurrentContext().FindRenderObject().Depth() >= overlayChildBox.Depth() {
			t.Error("the depth of the portal is not less than that of the overlay child after the shuffle")
		}
		verifyTreeIsClean(t, ts)
	})

	t.Run("widget is laid out before overlay child after reparenting", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c1, _, _, _ := portalControllers()
		targetKey := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "target widget"})
		childBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
		overlayChildBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
		var setState1, setState2 StateSetter
		targetMovedToEntry3 := false
		layoutCount1, layoutCount2 := 0, 0
		portal := func(layoutCount *int) Widget {
			return NewOverlayPortal(c1, func(BuildContext) Widget {
				return NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget {
					*layoutCount++
					verifyOverlayChildReadyForLayout(t, targetKey)
					return newRenderBoxAdapter(overlayChildBox)
				})
			}, newRenderBoxAdapter(childBox), OverlayPortalOpts{Key: targetKey})
		}
		entries := []*OverlayEntry{
			NewOverlayEntry(func(BuildContext) Widget {
				return manyRelayoutBoundaries(50, NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
					setState1 = setter
					if targetMovedToEntry3 {
						return NewSizedBox(nil)
					}
					return portal(&layoutCount1)
				}))
			}),
			NewOverlayEntry(func(BuildContext) Widget { return NewPlaceholder(nil) }),
			NewOverlayEntry(func(BuildContext) Widget {
				return NewSizedBox(NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
					setState2 = setter
					if !targetMovedToEntry3 {
						return NewSizedBox(nil)
					}
					return portal(&layoutCount2)
				}))
			}),
		}
		defer func() {
			for _, e := range entries {
				removeEntry(e)
			}
		}()
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: entries})))
		if layoutCount1 != 1 || layoutCount2 != 0 {
			t.Fatalf("layout counts = %d, %d; want 1, 0", layoutCount1, layoutCount2)
		}
		targetKey.CurrentContext().FindRenderObject().MarkNeedsLayout()
		childBox.MarkNeedsLayout()
		rebuildLayoutBuilderSubtree(t, ts, overlayChildBox)
		setState1(func() {})
		setState2(func() {})
		targetMovedToEntry3 = true
		ts.Pump()
		if targetKey.CurrentContext().FindRenderObject().Depth() >= overlayChildBox.Depth() {
			t.Error("the depth of the portal is not less than that of the overlay child")
		}
		verifyTreeIsClean(t, ts)
		if layoutCount1 != 1 || layoutCount2 != 1 {
			t.Errorf("layout counts = %d, %d; want 1, 1", layoutCount1, layoutCount2)
		}
	})

	t.Run("swap child and overlayChild", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c1, _, _, _ := portalControllers()
		childBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
		overlayChildBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
		var setState StateSetter
		swap := false
		// The adapters have a global key of their own.
		child1 := newRenderBoxAdapter(overlayChildBox)
		child2 := newRenderBoxAdapter(childBox)
		entry := NewOverlayEntry(func(BuildContext) Widget {
			return manyRelayoutBoundaries(50, NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
				setState = setter
				overlayChild, child := child2, child1
				if swap {
					overlayChild, child = child1, child2
				}
				return NewOverlayPortal(c1, func(BuildContext) Widget { return overlayChild }, child)
			}))
		})
		defer removeEntry(entry)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
		setState(func() { swap = true })
		ts.Pump()
		verifyTreeIsClean(t, ts)
	})

	t.Run("forgetChild", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		c1, c2, _, _ := portalControllers()
		childBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
		overlayChildBox := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
		var setState1, setState2 StateSetter
		takeChildren := false
		child1 := newRenderBoxAdapter(overlayChildBox)
		child2 := newRenderBoxAdapter(childBox)
		c1.Hide()
		entry1 := NewOverlayEntry(func(BuildContext) Widget {
			return NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
				setState2 = setter
				var child Widget
				if takeChildren {
					child = child1
				}
				return NewOverlayPortal(c1, func(BuildContext) Widget { return child2 }, child)
			})
		})
		entry2 := NewOverlayEntry(func(BuildContext) Widget {
			return manyRelayoutBoundaries(50, NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
				setState1 = setter
				var child Widget
				if !takeChildren {
					child = child2
				}
				return NewOverlayPortal(c2, func(BuildContext) Widget { return child1 }, child)
			}))
		})
		defer removeEntry(entry1)
		defer removeEntry(entry2)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry1, entry2}})))
		c1.Show()
		c2.Hide()
		setState2(func() { takeChildren = true })
		setState1(func() {})
		ts.Pump()
		verifyTreeIsClean(t, ts)
	})
}

// paintOrder is _PaintOrder: a widget that records when it paints.
type paintOrder struct {
	SingleChildRenderObjectWidget
	name string
	log  *[]string
}

func newPaintOrder(name string, log *[]string) *paintOrder {
	return &paintOrder{NewSingleChildRenderObjectWidgetBase(nil), name, log}
}

func (w *paintOrder) CreateElement() Element { return NewSingleChildRenderObjectElement(w) }

func (w *paintOrder) CreateRenderObject(BuildContext) rendering.RenderObject {
	r := &paintRecorder{onPaint: func() { *w.log = append(*w.log, w.name) }}
	rendering.InitRenderProxyBox(&r.RenderProxyBox, r, nil)
	return r
}

func (w *paintOrder) UpdateRenderObject(_ BuildContext, ro rendering.RenderObject) {
	ro.(*paintRecorder).onPaint = func() { *w.log = append(*w.log, w.name) }
}

type paintRecorder struct {
	rendering.RenderProxyBox
	onPaint func()
}

func (r *paintRecorder) Paint(context *rendering.PaintingContext, offset ui.Offset) {
	r.onPaint()
	r.RenderProxyBox.Paint(context, offset)
}

func TestOverlayPortalGlobalKeyReparentingPaintOrder(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, c2, c3, c4 := portalControllers()
	outerKey := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "Original Outer Widget"})
	innerKey := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "Original Inner Widget"})
	var setState StateSetter
	var log []string
	child1, child2, child3, child4 := newPaintOrder("child1", &log), newPaintOrder("child2", &log), newPaintOrder("child3", &log), newPaintOrder("child4", &log)
	build := func(outer, inner *GlobalKey[State], outerController, innerController *OverlayPortalController) Widget {
		// Expected order: child1, the child of the inner portal, child4.
		return NewColumn([]Widget{
			NewOverlayPortal(c1, func(BuildContext) Widget { return child1 }, nil),
			NewOverlayPortal(innerController, func(BuildContext) Widget {
				return NewOverlayPortal(outerController, func(BuildContext) Widget { return child3 }, child2, OverlayPortalOpts{Key: outer})
			}, nil, OverlayPortalOpts{Key: inner}),
			NewOverlayPortal(c4, func(BuildContext) Widget { return child4 }, nil),
		})
	}
	var widget Widget = NewColumn([]Widget{
		NewOverlayPortal(c1, func(BuildContext) Widget { return child1 }, nil),
		NewOverlayPortal(c2, func(BuildContext) Widget {
			return NewOverlayPortal(c3, func(BuildContext) Widget { return child3 }, child2, OverlayPortalOpts{Key: innerKey})
		}, nil, OverlayPortalOpts{Key: outerKey}),
		NewOverlayPortal(c4, func(BuildContext) Widget { return child4 }, nil),
	})
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
			setState = setter
			return widget
		})
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	want := []string{"child1", "child2", "child3", "child4"}
	if !slices.Equal(log, want) {
		t.Fatalf("paint order = %v, want %v", log, want)
	}
	log = nil
	// Swap the nested OverlayPortal.
	widget = build(outerKey, innerKey, c2, c3)
	setState(func() {})
	ts.Pump()
	want = []string{"child1", "child3", "child2", "child4"}
	if !slices.Equal(log, want) {
		t.Errorf("paint order after the swap = %v, want %v", log, want)
	}
}

// swapHarness is the shared state of the 'Swapping' group.
type swapHarness struct {
	swapped              bool
	setState1, setState2 StateSetter
}

func (h *swapHarness) set(newValue bool) {
	h.swapped = newValue
	if h.setState1 != nil {
		h.setState1(func() {})
	}
	if h.setState2 != nil {
		h.setState2(func() {})
	}
}

// pick is the counter that is in the first place unless the harness is swapped.
func (h *swapHarness) pick(first, second *layoutCounter) *layoutCounter {
	if h.swapped {
		return second
	}
	return first
}

func TestOverlayPortalGlobalKeyReparentingSwapping(t *testing.T) {
	// The layout builder variants build the portal inside a LayoutBuilder.
	maybeLayoutBuilder := func(withLayoutBuilder bool, portal func() Widget) Widget {
		if !withLayoutBuilder {
			return portal()
		}
		return NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget { return portal() })
	}
	for _, withLayoutBuilder := range []bool{false, true} {
		name := func(base string) string {
			if withLayoutBuilder {
				return base + ", featuring LayoutBuilder"
			}
			return base
		}
		t.Run(name("between OverlayEntry & overlayChild"), func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			c1, _, _, _ := portalControllers()
			var h swapHarness
			counter1, counter2 := newLayoutCounter(), newLayoutCounter()
			entry1 := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
				h.setState1 = setter
				// The adapter is keyed by the render box.
				return newRenderBoxAdapter(h.pick(counter1, counter2))
			})
			entry2 := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
				h.setState2 = setter
				return maybeLayoutBuilder(withLayoutBuilder, func() Widget {
					return NewOverlayPortal(c1, func(BuildContext) Widget { return newRenderBoxAdapter(h.pick(counter2, counter1)) }, NewSizedBox(nil))
				})
			})
			defer removeEntry(entry1)
			defer removeEntry(entry2)
			ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry1, entry2}})))
			checkSwapCounts(t, ts, &h, counter1, counter2)
		})
		t.Run(name("between overlayChild & overlayChild"), func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			c1, c2, _, _ := portalControllers()
			var h swapHarness
			counter1, counter2 := newLayoutCounter(), newLayoutCounter()
			entry1 := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
				h.setState1 = setter
				return maybeLayoutBuilder(withLayoutBuilder, func() Widget {
					return NewOverlayPortal(c1, func(BuildContext) Widget { return newRenderBoxAdapter(h.pick(counter1, counter2)) }, NewSizedBox(nil))
				})
			})
			entry2 := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
				h.setState2 = setter
				return maybeLayoutBuilder(withLayoutBuilder, func() Widget {
					return NewOverlayPortal(c2, func(BuildContext) Widget { return newRenderBoxAdapter(h.pick(counter2, counter1)) }, NewSizedBox(nil))
				})
			})
			defer removeEntry(entry1)
			defer removeEntry(entry2)
			ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry1, entry2}})))
			checkSwapCounts(t, ts, &h, counter1, counter2)
		})
		t.Run(name("between child & overlayChild"), func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			c1, _, _, _ := portalControllers()
			var h swapHarness
			counter1, counter2 := newLayoutCounter(), newLayoutCounter()
			entry := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
				h.setState1 = setter
				return maybeLayoutBuilder(withLayoutBuilder, func() Widget {
					return NewOverlayPortal(c1, func(BuildContext) Widget { return newRenderBoxAdapter(h.pick(counter1, counter2)) },
						newRenderBoxAdapter(h.pick(counter2, counter1)))
				})
			})
			defer removeEntry(entry)
			ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
			checkSwapCounts(t, ts, &h, counter1, counter2)
		})
	}
}

// checkSwapCounts swaps the two render boxes, and swaps them back, and checks that
// each is laid out once more each time.
func checkSwapCounts(t *testing.T, ts *gluttertest.Tester, h *swapHarness, counter1, counter2 *layoutCounter) {
	t.Helper()
	if counter1.layoutCount != 1 || counter2.layoutCount != 1 {
		t.Fatalf("layout counts = %d, %d; want 1, 1", counter1.layoutCount, counter2.layoutCount)
	}
	for want, swapped := range []bool{true, false} {
		h.set(swapped)
		ts.Pump()
		if counter1.layoutCount != want+2 || counter2.layoutCount != want+2 {
			t.Errorf("after swapping to %v: layout counts = %d, %d; want %d, %d", swapped, counter1.layoutCount, counter2.layoutCount, want+2, want+2)
		}
	}
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
}

func TestOverlayPortalSafeToMoveTheOverlayChildToAnotherOverlayAndRemoveTheOldOne(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	c1.Show()
	key := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "key"})
	oldOverlayKey := NewGlobalKey[*OverlayState](GlobalKeyOpts{DebugLabel: "old overlay"})
	newOverlayKey := NewGlobalKey[*OverlayState](GlobalKeyOpts{DebugLabel: "new overlay"})
	overlayChildKey := NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "overlay child key"})
	portal := func(BuildContext) Widget {
		return NewOverlayPortal(c1, func(BuildContext) Widget { return NewSizedBox(nil, SizedBoxOpts{Key: overlayChildKey}) },
			NewSizedBox(nil), OverlayPortalOpts{Key: key})
	}
	entry1 := NewOverlayEntry(portal)
	entry2 := NewOverlayEntry(portal)
	defer removeEntry(entry1)
	defer removeEntry(entry2)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{Key: oldOverlayKey, InitialEntries: []*OverlayEntry{entry1}})))
	counts := func() [3]int {
		return [3]int{ts.Count(gluttertest.ByKey(overlayChildKey)), ts.Count(gluttertest.ByKey(newOverlayKey)), ts.Count(gluttertest.ByKey(oldOverlayKey))}
	}
	if got := counts(); got != [3]int{1, 0, 1} {
		t.Fatalf("counts = %v, want [1 0 1]", got)
	}
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{Key: newOverlayKey, InitialEntries: []*OverlayEntry{entry2}})))
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
	if got := counts(); got != [3]int{1, 1, 0} {
		t.Errorf("counts = %v, want [1 1 0]", got)
	}
}

func TestOverlayPortalPaintOrderShowBringsToTop(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, c2, _, _ := portalControllers()
	c1.Hide()
	var log []string
	child1, child2 := newPaintOrder("child1", &log), newPaintOrder("child2", &log)
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewColumn([]Widget{
			NewOverlayPortal(c1, func(BuildContext) Widget { return child1 }, nil),
			NewOverlayPortal(c2, func(BuildContext) Widget { return child2 }, nil),
		})
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	steps := []struct {
		name string
		do   func()
		want []string
	}{
		{"only child2 is visible", func() {}, []string{"child2"}},
		{"show child1", func() { c1.Show(); ts.Pump() }, []string{"child2", "child1"}},
		{"show child2", func() { c2.Show(); ts.Pump() }, []string{"child1", "child2"}},
		{"hide both", func() { c2.Hide(); c1.Hide(); ts.Pump() }, nil},
	}
	for i, step := range steps {
		if i > 0 {
			log = nil
		}
		step.do()
		if !slices.Equal(log, step.want) {
			t.Errorf("%s: paint order = %v, want %v", step.name, log, step.want)
		}
	}
}

func TestOverlayPortalPaintOrderDoesNotChangeAfterGlobalKeyReparenting(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, c2, _, _ := portalControllers()
	key := NewGlobalKey[State]()
	var setState StateSetter
	reparented := false
	// The adapters have a global key of their own.
	child1Box := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
	child2Box := rendering.NewRenderConstrainedBox(rendering.NewBoxConstraints(), nil)
	child1, child2 := newRenderBoxAdapter(child1Box), newRenderBoxAdapter(child2Box)
	overlayPortal1 := NewOverlayPortal(c1, func(BuildContext) Widget { return child1 }, NewSizedBox(nil), OverlayPortalOpts{Key: key})
	overlayPortal2 := NewOverlayPortal(c2, func(BuildContext) Widget { return child2 }, NewSizedBox(nil))
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewColumn([]Widget{
			NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
				setState = setter
				if reparented {
					return NewSizedBox(overlayPortal1)
				}
				return overlayPortal1
			}),
			overlayPortal2,
		})
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	theater := ts.RenderObject(gluttertest.ByType[*Overlay]())
	visited := func() []rendering.RenderObject {
		var out []rendering.RenderObject
		theater.VisitChildren(func(c rendering.RenderObject) { out = append(out, c) })
		return out
	}
	containsInOrder := func(list []rendering.RenderObject, a, b rendering.RenderObject) bool {
		ia, ib := slices.Index(list, a), slices.Index(list, b)
		return ia >= 0 && ib > ia
	}
	children := visited()
	if len(children) != 3 {
		t.Fatalf("%d children visited, want 3", len(children))
	}
	if !containsInOrder(children, child1Box.Parent(), child2Box.Parent()) {
		t.Fatal("the overlay children are not in order")
	}
	setState(func() { reparented = true })
	ts.Pump()
	// The child list stays the same.
	if !containsInOrder(visited(), child1Box.Parent(), child2Box.Parent()) {
		t.Error("the overlay children are not in order after the reparenting")
	}
}

// labeledBox is a box with a semantics label, in place of the Text widgets of the
// Dart tests: the test font does not have the metrics of the Ahem font that they
// rely on.
func labeledBox(label string, w, h float64) Widget {
	return NewSemantics(NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(w), Height: foundation.Some(h)}),
		SemanticsOpts{Label: foundation.Some(label)})
}

// semanticNodesWith are the nodes of the semantics tree with the label.
func semanticNodesWith(ts *gluttertest.Tester, label string) []*semantics.SemanticsNode {
	var out []*semantics.SemanticsNode
	root := ts.Binding.Renderer().RenderViews()[0].Owner().SemanticsOwner().RootSemanticsNode()
	var visit func(n *semantics.SemanticsNode) bool
	visit = func(n *semantics.SemanticsNode) bool {
		if n.Label() == label {
			out = append(out, n)
		}
		n.VisitChildren(visit)
		return true
	}
	if root != nil {
		visit(root)
	}
	return out
}

func singleSemanticNode(t *testing.T, ts *gluttertest.Tester, label string) *semantics.SemanticsNode {
	t.Helper()
	nodes := semanticNodesWith(ts, label)
	if len(nodes) != 1 {
		t.Fatalf("%d semantics nodes with the label %q, want 1", len(nodes), label)
	}
	return nodes[0]
}

func TestOverlayPortalSemantics(t *testing.T) {
	skipUnlessDebug(t) // the semantics nodes are read from the render objects
	t.Run("ordering", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
		ts.EnsureSemantics()
		c1, _, _, _ := portalControllers()
		entry := NewOverlayEntry(func(BuildContext) Widget {
			return NewRow([]Widget{
				NewSemantics(NewOverlayPortal(c1, func(BuildContext) Widget {
					return NewPositioned(labeledBox("BBBB", 40, 10), PositionedOpts{Left: foundation.Some(0.0), Top: foundation.Some(0.0)})
				}, labeledBox("A", 10, 10)), SemanticsOpts{Container: true, ExplicitChildNodes: true}),
				labeledBox("CC", 20, 10),
			}, RowOpts{MainAxisSize: rendering.MainAxisSizeMin, CrossAxisAlignment: rendering.CrossAxisAlignmentEnd})
		})
		defer removeEntry(entry)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
		a := singleSemanticNode(t, ts, "A")
		b := singleSemanticNode(t, ts, "BBBB")
		cc := singleSemanticNode(t, ts, "CC")
		// The node that wraps the overlay child is traversed as a child of the node of
		// the portal's child, as the Dart test expects, though it is a child of the root
		// of the semantics tree in hit test order.
		wrapper := b.Parent()
		if wrapper == nil || wrapper.TraversalParent() != a {
			t.Errorf("the traversal parent of the overlay child's wrapper is %v, want the node of A", wrapper.TraversalParent())
		}
		if cc.TraversalParent() == a {
			t.Error("the traversal parent of CC is the node of A")
		}
		if b.Rect() != ui.NewRectFromLTWH(0, 0, 40, 10) {
			t.Errorf("rect of BBBB = %v", b.Rect())
		}
		if cc.Rect() != ui.NewRectFromLTWH(0, 0, 20, 10) {
			t.Errorf("rect of CC = %v", cc.Rect())
		}
	})

	t.Run("overlay child clipping", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
		ts.EnsureSemantics()
		c1, _, _, _ := portalControllers()
		entry := NewOverlayEntry(func(BuildContext) Widget {
			return NewListView([]Widget{
				// Clips the OverlayPortal, making it only half visible.
				NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(595.0)}),
				NewSemantics(NewOverlayPortal(c1, func(BuildContext) Widget {
					return NewPositioned(NewListView([]Widget{
						NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(3.0)}), // Clips B so it is only 7 pixels tall.
						labeledBox("B", 10, 10),
					}), PositionedOpts{Left: foundation.Some(0.0), Right: foundation.Some(0.0), Top: foundation.Some(0.0), Height: foundation.Some(10.0)})
				}, labeledBox("A", 800, 10)), SemanticsOpts{Container: true, ExplicitChildNodes: true}),
			})
		})
		defer removeEntry(entry)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
		portal := singleSemanticNode(t, ts, "A")
		child := singleSemanticNode(t, ts, "B")
		if portal.Rect() != ui.NewRectFromLTWH(0, 0, 800, 5) {
			t.Errorf("rect of the clipped portal = %v, want 800x5", portal.Rect())
		}
		if child.Rect() != ui.NewRectFromLTWH(0, 0, 800, 7) {
			t.Errorf("rect of the clipped overlay child = %v, want 800x7", child.Rect())
		}
		if portal.Transform() != nil {
			t.Errorf("transform of the portal = %v, want none", portal.Transform())
		}
		// The parent semantics node is created by the ListView.
		if tr := child.Transform(); tr == nil || *tr != *ui.NewMatrix4TranslationValues(0, 3, 0) {
			t.Errorf("transform of the overlay child = %v, want a translation of (0, 3)", tr)
		}
	})

	t.Run("node is hidden", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
		ts.EnsureSemantics()
		c1, _, _, _ := portalControllers()
		entry := NewOverlayEntry(func(BuildContext) Widget {
			return NewListView([]Widget{
				// Clips the OverlayPortal, making it completely invisible.
				NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(600.0)}),
				NewSemantics(NewOverlayPortal(c1, func(BuildContext) Widget {
					return NewPositioned(labeledBox("B", 10, 10), PositionedOpts{Left: foundation.Some(0.0), Top: foundation.Some(0.0)})
				}, labeledBox("A", 800, 10)), SemanticsOpts{Container: true, ExplicitChildNodes: true}),
			})
		})
		defer removeEntry(entry)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
		portal := singleSemanticNode(t, ts, "A")
		child := singleSemanticNode(t, ts, "B")
		if portal.Rect() != ui.NewRectFromLTWH(0, 0, 800, 10) || child.Rect() != ui.NewRectFromLTWH(0, 0, 10, 10) {
			t.Errorf("rects = %v, %v; want 800x10, 10x10", portal.Rect(), child.Rect())
		}
		if portal.Transform() != nil || child.Transform() != nil {
			t.Errorf("transforms = %v, %v; want none", portal.Transform(), child.Transform())
		}
	})

	t.Run("node is dropped but the element is kept alive", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 800, Height: 600}})
		ts.EnsureSemantics()
		c1, _, _, _ := portalControllers()
		controller := NewScrollController(ScrollControllerOpts{InitialScrollOffset: 10})
		defer controller.Dispose()
		no := foundation.Some(false)
		entry := NewOverlayEntry(func(BuildContext) Widget {
			return NewListView([]Widget{
				// Clips the OverlayPortal, making it completely invisible.
				NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(600.0)}),
				NewKeepAlive(true, NewSemantics(NewOverlayPortal(c1, func(BuildContext) Widget {
					return NewPositioned(labeledBox("B", 10, 10), PositionedOpts{Left: foundation.Some(0.0), Top: foundation.Some(0.0)})
				}, labeledBox("A", 800, 10)), SemanticsOpts{Container: true, ExplicitChildNodes: true})),
			}, ListViewOpts{
				Controller: controller, ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(0)),
				AddAutomaticKeepAlives: no, AddRepaintBoundaries: no, AddSemanticIndexes: no,
			})
		})
		defer removeEntry(entry)
		ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
		if len(semanticNodesWith(ts, "A")) == 0 || len(semanticNodesWith(ts, "B")) == 0 {
			t.Fatal("A and B have no semantics nodes at first")
		}
		controller.JumpTo(0)
		ts.Pump()
		if got := len(semanticNodesWith(ts, "A")); got != 0 {
			t.Errorf("%d nodes with the label A after the scroll, want none", got)
		}
		if got := len(semanticNodesWith(ts, "B")); got == 0 {
			t.Error("the overlay child has no semantics node after the scroll")
		}
	})
}

func TestOverlayPortalOverlayChildCanComputeThePaintTransformOfTheRegularChild(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	c1, _, _, _ := portalControllers()
	var setState StateSetter
	padding := painting.NewEdgeInsetsFromLTRB(10, 0, 0, 0)
	var computed ui.Matrix4
	zOffset := float32(123)
	entry := statefulEntry(func(_ BuildContext, setter StateSetter) Widget {
		setState = setter
		return NewTransform(ui.NewMatrix4TranslationValues(0, 0, zOffset), NewPadding(padding, NewOverlayPortal(c1, func(BuildContext) Widget {
			return NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget {
				placeholder := ts.RenderObject(gluttertest.ByType[*Placeholder]())
				overlay := ts.RenderObject(gluttertest.ByType[*Overlay]())
				computed = *placeholder.GetTransformTo(overlay)
				if !placeholder.(rendering.Box).HasSize() {
					t.Error("the regular child has no size when the overlay child builds")
				}
				return NewSizedBox(nil)
			})
		}, NewPlaceholder(nil))))
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	// During the initial layout, the Padding would not have computed the offset of its
	// child if the overlay child was laid out via the tree walk, as RenderPadding
	// lays the child out before it computes the offset.
	if want := *ui.NewMatrix4TranslationValues(10, 0, 123); computed != want {
		t.Errorf("paint transform = %v, want %v", computed, want)
	}
	setState(func() {
		padding = painting.NewEdgeInsetsFromLTRB(0, 20, 0, 0)
		zOffset = 321
	})
	ts.Pump()
	if want := *ui.NewMatrix4TranslationValues(0, 20, 321); computed != want {
		t.Errorf("paint transform = %v, want %v", computed, want)
	}
}

func TestOverlayPortalDoesNotCrashAtZeroArea(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.SetViewSize(ui.Size{})
	controller := NewOverlayPortalController()
	ts.PumpWidget(ltr(NewCenter(NewOverlayPortal(controller, func(BuildContext) Widget { return NewSizedBox(nil) }, nil))))
	if got := ts.Size(gluttertest.ByType[*OverlayPortal]()); got != (ui.Size{}) {
		t.Errorf("size = %v, want zero", got)
	}
	controller.Show()
}

// Regression test for https://github.com/flutter/flutter/issues/180569.
func TestOverlayPortalDoesNotThrowWhenReparentingAndOverlayChildRequestsRelayout(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var setState StateSetter
	controller := NewOverlayPortalController()
	controller.Show()
	// MetaData is not ported, so the overlay child is a box.
	childKey := foundation.NewValueKey("overlay child")
	portal := NewOverlayPortal(controller, func(BuildContext) Widget { return NewSizedBox(nil, SizedBoxOpts{Key: childKey}) }, NewPlaceholder(nil),
		OverlayPortalOpts{Key: NewGlobalKey[State](GlobalKeyOpts{DebugLabel: "OverlayPortal"})})
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget {
			return NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
				setState = setter
				// This subtree is inflated again whenever it rebuilds, because of the new
				// GlobalKey.
				return NewKeyedSubtree(portal, WidgetOpts{Key: NewGlobalKey[State]()})
			})
		})
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	// The overlay child calls MarkNeedsLayout.
	ts.RenderObject(gluttertest.ByKey(childKey)).MarkNeedsLayout()
	// Triggers the reparenting.
	setState(func() {})
	ts.Pump()
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("errors: %v", errs)
	}
}

func TestOverlayPortalControllerHideWithoutShowPanics(t *testing.T) {
	skipUnlessDebug(t)
	controller := NewOverlayPortalController(OverlayPortalControllerOpts{DebugLabel: "unattached"})
	requireTag(t, "OverlayPortalHideWithoutShow", controller.Hide)
	controller.Show()
	if !controller.IsShowing() {
		t.Error("the controller is not showing")
	}
	controller.Hide() // Does not panic.
	if controller.IsShowing() {
		t.Error("the controller is showing")
	}
	controller.Toggle()
	if !controller.IsShowing() {
		t.Error("Toggle did not show")
	}
	if got := controller.String(); got != "OverlayPortalController(unattached) DETACHED" {
		t.Errorf("String = %q", got)
	}
}

func TestOverlayPortalControllerShowDuringBuildPanics(t *testing.T) {
	skipUnlessDebug(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewOverlayPortalController()
	var setState StateSetter
	showInBuild := false
	entry := NewOverlayEntry(func(BuildContext) Widget {
		return NewColumn([]Widget{
			NewOverlayPortal(controller, emptyBox, NewSizedBox(nil)),
			NewStatefulBuilder(func(_ BuildContext, setter StateSetter) Widget {
				setState = setter
				if showInBuild {
					controller.Show()
				}
				return NewSizedBox(nil)
			}),
		})
	})
	defer removeEntry(entry)
	ts.PumpWidget(ltr(NewOverlay(OverlayOpts{InitialEntries: []*OverlayEntry{entry}})))
	showInBuild = true
	setState(func() {})
	ts.Pump()
	_ = requireErrorTag(t, takeFirst(t, ts), "OverlayPortalShowDuringBuild")
	ts.TakeErrors()
}
