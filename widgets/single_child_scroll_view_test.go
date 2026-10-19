package widgets_test

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/single_child_scroll_view_test.dart.

func greenBox(height float64) Widget {
	return NewContainer(nil, ContainerOpts{Height: foundation.Some(height), Color: foundation.Some(ui.NewColor(0xFF00FF00))})
}

func TestSingleChildScrollViewRespectsClipBehavior(t *testing.T) {
	useAndroid(t) // the Dart tests run on Android: no scrollbar, so no Directionality needed
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(NewSingleChildScrollView(NewContainer(nil, ContainerOpts{Height: foundation.Some(2000.0)})))
	viewport := func() interface{ ClipBehavior() ui.Clip } {
		return ts.RenderObject(gluttertest.ByElementPredicate("single child viewport", func(e Element) bool {
			_, ok := e.RenderObject().(rendering.RenderAbstractViewport)
			return ok
		})).(interface{ ClipBehavior() ui.Clip })
	}
	if got := viewport().ClipBehavior(); got != ui.ClipHardEdge {
		t.Errorf("clip = %v, want hardEdge", got)
	}
	// The underlying Scrollable has the same clipBehavior (flutter/flutter#133330).
	if got := ts.Widget(gluttertest.ByType[*Scrollable]()).(*Scrollable).ClipBehavior(); got != ui.ClipHardEdge {
		t.Errorf("scrollable clip = %v, want hardEdge", got)
	}
	ts.PumpWidget(NewSingleChildScrollView(NewContainer(nil, ContainerOpts{Height: foundation.Some(2000.0)}), SingleChildScrollViewOpts{ClipBehavior: ui.ClipAntiAlias}))
	if got := viewport().ClipBehavior(); got != ui.ClipAntiAlias {
		t.Errorf("clip = %v, want antiAlias", got)
	}
	if got := ts.Widget(gluttertest.ByType[*Scrollable]()).(*Scrollable).ClipBehavior(); got != ui.ClipAntiAlias {
		t.Errorf("scrollable clip = %v, want antiAlias", got)
	}
}

func TestSingleChildScrollViewControl(t *testing.T) {
	useAndroid(t) // the Dart tests run on Android: no scrollbar, so no Directionality needed
	ts := gluttertest.New(t, gluttertest.Opts{})
	ts.PumpWidget(NewSingleChildScrollView(greenBox(2000)))
	box := ts.RenderObject(gluttertest.ByType[*ColoredBox]()).(rendering.Box)
	if got := box.LocalToGlobal(ui.OffsetZero, nil); got != (ui.Offset{}) {
		t.Errorf("origin = %v", got)
	}
	ts.Drag(gluttertest.ByType[*SingleChildScrollView](), ui.Offset{Dx: -200, Dy: -200})
	if got := box.LocalToGlobal(ui.OffsetZero, nil); got != (ui.Offset{Dy: -200}) {
		t.Errorf("origin after the drag = %v", got)
	}
}

func TestSingleChildScrollViewChangingControllersChangesScrollPosition(t *testing.T) {
	useAndroid(t) // the Dart tests run on Android: no scrollbar, so no Directionality needed
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := newDeferringScrollController(false)
	defer controller.Dispose()
	ts.PumpWidget(NewSingleChildScrollView(greenBox(2000)))
	ts.PumpWidget(NewSingleChildScrollView(greenBox(2000), SingleChildScrollViewOpts{Controller: controller}))
	if _, ok := scrollableOf(ts).Position().(*deferringScrollPosition); !ok {
		t.Errorf("position is a %T", scrollableOf(ts).Position())
	}
}

func TestSingleChildScrollViewSetsPrimaryScrollControllerWhenPrimary(t *testing.T) {
	useAndroid(t) // the Dart tests run on Android: no scrollbar, so no Directionality needed
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(NewPrimaryScrollController(controller, NewSingleChildScrollView(greenBox(2000), SingleChildScrollViewOpts{Primary: foundation.Some(true)})))
	if got := ts.Widget(gluttertest.ByType[*Scrollable]()).(*Scrollable).Controller(); got != ScrollController(controller) {
		t.Errorf("controller = %v", got)
	}
}

func TestSingleChildScrollViewChangingControllerInsideDirtyLayoutBuilder(t *testing.T) {
	useAndroid(t) // the Dart tests run on Android: no scrollbar, so no Directionality needed
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController()
	defer controller.Dispose()
	build := func(width float64, c ScrollController) Widget {
		return NewCenter(NewSizedBox(NewLayoutBuilder(func(BuildContext, rendering.BoxConstraints) Widget {
			return NewSingleChildScrollView(greenBox(2000), SingleChildScrollViewOpts{Controller: c})
		}), SizedBoxOpts{Width: foundation.Some(width)}))
	}
	ts.PumpWidget(build(750, nil))
	ts.PumpWidget(build(700, controller))
}

func TestSingleChildScrollViewsAreNotPrimaryByDefault(t *testing.T) {
	controller := NewScrollController()
	defer controller.Dispose()
	for name, o := range map[string]SingleChildScrollViewOpts{
		"vertical":   {},
		"horizontal": {ScrollDirection: painting.AxisHorizontal},
		"controller": {Controller: controller},
	} {
		if NewSingleChildScrollView(nil, o).Primary().IsSome() {
			t.Errorf("%s: primary is set", name)
		}
	}
}

func TestSingleChildScrollViewPrimaryScrollControllerByPlatform(t *testing.T) {
	for _, c := range []struct {
		platform foundation.TargetPlatform
		want     bool
	}{
		{foundation.TargetPlatformAndroid, true}, {foundation.TargetPlatformIOS, true}, {foundation.TargetPlatformFuchsia, true},
		{foundation.TargetPlatformLinux, false}, {foundation.TargetPlatformMacOS, false}, {foundation.TargetPlatformWindows, false},
	} {
		t.Run(c.platform.String(), func(t *testing.T) {
			usePlatform(t, c.platform)
			ts := gluttertest.New(t, gluttertest.Opts{})
			controller := NewScrollController()
			defer controller.Dispose()
			ts.PumpWidget(ltr(NewMediaQuery(NewMediaQueryData(), NewPrimaryScrollController(controller, NewSingleChildScrollView(nil)))))
			if controller.HasClients() != c.want {
				t.Errorf("hasClients = %v, want %v", controller.HasClients(), c.want)
			}
		})
	}
}

func TestSingleChildScrollViewNestedScrollablesHaveNullPrimaryScrollController(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	innerKey := foundation.NewValueKey("inner")
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(ltr(NewPrimaryScrollController(controller, NewSingleChildScrollView(
		NewContainer(NewListView(nil, ListViewOpts{Key: innerKey, Primary: foundation.Some(true)}),
			ContainerOpts{Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxHeight: foundation.Some(200.0)}))}),
		SingleChildScrollViewOpts{Primary: foundation.Some(true)}))))
	inner := ts.Widget(gluttertest.Descendant(gluttertest.ByKey(innerKey), gluttertest.ByType[*Scrollable]())).(*Scrollable)
	if inner.Controller() != nil {
		t.Errorf("inner controller = %v", inner.Controller())
	}
}

func tileColumn(n int, width float64) ([]Widget, Widget) {
	children := make([]Widget, n)
	for i := range children {
		children[i] = NewSizedBox(NewText(fmt.Sprintf("Tile %d", i)), SizedBoxOpts{Height: foundation.Some(100.0), Width: foundation.Some(width)})
	}
	return children, nil
}

func TestSingleChildScrollViewGetOffsetToReveal(t *testing.T) {
	findViewport := func(ts *gluttertest.Tester) rendering.RenderAbstractViewport {
		return ts.RenderObject(gluttertest.ByElementPredicate("viewport", func(e Element) bool {
			_, ok := e.RenderObject().(rendering.RenderAbstractViewport)
			return ok
		})).(rendering.RenderAbstractViewport)
	}
	type reveal struct {
		alignment float64
		rect      foundation.Opt[ui.Rect]
		offset    float64
		want      ui.Rect
	}
	small := foundation.Some(ui.NewRectFromLTWH(40, 40, 10, 10))
	for _, c := range []struct {
		name       string
		horizontal bool
		reverse    bool
		cases      []reveal
	}{
		{"down", false, false, []reveal{
			{0, foundation.None[ui.Rect](), 500, ui.NewRectFromLTWH(0, 0, 300, 100)},
			{1, foundation.None[ui.Rect](), 400, ui.NewRectFromLTWH(0, 100, 300, 100)},
			{0, small, 540, ui.NewRectFromLTWH(40, 0, 10, 10)},
			{1, small, 350, ui.NewRectFromLTWH(40, 190, 10, 10)},
		}},
		{"up", false, true, []reveal{
			{0, foundation.None[ui.Rect](), 500, ui.NewRectFromLTWH(0, 100, 300, 100)},
			{1, foundation.None[ui.Rect](), 400, ui.NewRectFromLTWH(0, 0, 300, 100)},
			{0, small, 550, ui.NewRectFromLTWH(40, 190, 10, 10)},
			{1, small, 360, ui.NewRectFromLTWH(40, 0, 10, 10)},
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			controller := NewScrollController(ScrollControllerOpts{InitialScrollOffset: 300})
			defer controller.Dispose()
			children, _ := tileColumn(20, 300)
			col := children
			if c.reverse {
				col = make([]Widget, len(children))
				for i := range children {
					col[len(children)-1-i] = children[i]
				}
			}
			ts.PumpWidget(ltr(NewCenter(NewSizedBox(
				NewSingleChildScrollView(NewColumn(col), SingleChildScrollViewOpts{Controller: controller, Reverse: c.reverse}),
				SizedBoxOpts{Height: foundation.Some(200.0), Width: foundation.Some(300.0)}))))
			viewport := findViewport(ts)
			target := ts.RenderObject(gluttertest.ByWidget(children[5]))
			for _, r := range c.cases {
				got := viewport.GetOffsetToReveal(target, r.alignment, rendering.GetOffsetToRevealOpts{Rect: r.rect})
				if got.Offset != r.offset || got.Rect != r.want {
					t.Errorf("alignment %v rect %v: got %v %v, want %v %v", r.alignment, r.rect, got.Offset, got.Rect, r.offset, r.want)
				}
			}
		})
	}
	t.Run("axis mismatch does not assert", func(t *testing.T) {
		ts := gluttertest.New(t, gluttertest.Opts{})
		controller := NewScrollController(ScrollControllerOpts{InitialScrollOffset: 300})
		defer controller.Dispose()
		children, _ := tileColumn(20, 300)
		ts.PumpWidget(ltr(NewCenter(NewSizedBox(NewSingleChildScrollView(NewColumn(children), SingleChildScrollViewOpts{Controller: controller}),
			SizedBoxOpts{Height: foundation.Some(200.0), Width: foundation.Some(300.0)}))))
		findViewport(ts).GetOffsetToReveal(ts.RenderObject(gluttertest.ByWidget(children[5])), 0, rendering.GetOffsetToRevealOpts{Axis: painting.AxisHorizontal})
	})
}
