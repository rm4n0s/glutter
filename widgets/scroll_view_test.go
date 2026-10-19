package widgets_test

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/scroll_view_test.dart.
// Not ported: the keyboardDismissBehavior tests (they wait for the focus system
// of step 10.5), the tests that make a builder throw (the error widget), the
// fallback ScrollActions tests (step 10.2) and the semantics.

func primaryScrollControllerApp(child Widget, controller ScrollController) Widget {
	return ltr(NewMediaQuery(NewMediaQueryData(), NewPrimaryScrollController(controller, child)))
}

func TestListViewControlTest(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	var log []string
	children := make([]Widget, len(kStates))
	for i, state := range kStates {
		state := state
		children[i] = NewGestureDetector(NewContainer(NewText(state), ContainerOpts{Height: foundation.Some(200.0), Color: foundation.Some(ui.NewColor(0xFF0000FF))}),
			GestureDetectorOpts{OnTap: func() { log = append(log, state) }, DragStartBehavior: gestures.DragStartBehaviorDown})
	}
	ts.PumpWidget(ltr(NewListView(children, ListViewOpts{DragStartBehavior: gestures.DragStartBehaviorDown})))
	// The text has no advance width in the test font: the cell is hit through its container.
	tapState := func(state string) {
		ts.TapAt(ui.Offset{Dx: 400, Dy: ts.Center(gluttertest.ByText(state)).Dy})
	}
	tapState("Alabama")
	if fmt.Sprint(log) != "[Alabama]" {
		t.Fatalf("log = %v", log)
	}
	log = nil
	if ts.Count(gluttertest.ByText("Nevada")) != 0 {
		t.Error("Nevada is built")
	}
	ts.Drag(gluttertest.ByText("Alabama"), ui.Offset{Dy: -4000})
	ts.Pump()
	if ts.Count(gluttertest.ByText("Alabama")) != 0 || ts.Center(gluttertest.ByText("Massachusetts")).Dy != 100 {
		t.Errorf("Massachusetts at %v", ts.Center(gluttertest.ByText("Massachusetts")))
	}
	tapState("Massachusetts")
	if fmt.Sprint(log) != "[Massachusetts]" {
		t.Errorf("log = %v", log)
	}
}

func TestScrollViewsSupportNullItems(t *testing.T) {
	item := func(_ BuildContext, i int) Widget {
		if i == 5 {
			return nil
		}
		return NewText("item")
	}
	tests := []struct {
		name       string
		build      func() Widget
		items      int
		separators int
	}{
		{"GridView.builder", func() Widget {
			return NewGridViewBuilder(rendering.NewSliverGridDelegateWithFixedCrossAxisCount(42), item, GridViewOpts{ItemCount: foundation.Some(42)})
		}, 5, 0},
		{"ListView.builder", func() Widget { return NewListViewBuilder(item, ListViewOpts{ItemCount: foundation.Some(42)}) }, 5, 0},
		{"ListView.separated", func() Widget {
			return NewListViewSeparated(item, func(BuildContext, int) Widget { return NewText("separator") }, 42)
		}, 5, 5},
		{"PageView", func() Widget {
			controller := NewPageController(PageControllerOpts{ViewportFraction: foundation.Some(1.0 / 5)})
			return NewPageViewBuilder(func(_ BuildContext, i int) Widget {
				if i == 2 {
					return nil
				}
				return NewText("item")
			}, PageViewOpts{ItemCount: foundation.Some(5), Controller: controller})
		}, 2, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(ltr(NewCenter(tc.build())))
			if got := ts.Count(gluttertest.ByText("item")); got != tc.items {
				t.Errorf("%d items, want %d", got, tc.items)
			}
			if got := ts.Count(gluttertest.ByText("separator")); got != tc.separators {
				t.Errorf("%d separators, want %d", got, tc.separators)
			}
		})
	}
}

func TestScrollViewCanJumpToDuringADrag(t *testing.T) {
	useAndroid(t)
	ts := gluttertest.New(t, gluttertest.Opts{})
	controller := NewScrollController()
	defer controller.Dispose()
	var log []string
	children := make([]Widget, len(kStates))
	for i, state := range kStates {
		children[i] = NewSizedBox(NewText(state), SizedBoxOpts{Height: foundation.Some(200.0)})
	}
	ts.PumpWidget(ltr(NewNotificationListener[ScrollNotification](NewListView(children, ListViewOpts{Controller: controller}), func(n ScrollNotification) bool {
		log = append(log, fmt.Sprintf("%T", n))
		return false
	})))
	if len(log) != 0 {
		t.Fatalf("log = %v", log)
	}
	gesture := ts.StartGesture(ui.Offset{Dx: 100, Dy: 100})
	gesture.MoveBy(ui.Offset{Dy: -100})
	if fmt.Sprint(log) != "[*widgets.ScrollStartNotification *widgets.UserScrollNotification *widgets.ScrollUpdateNotification]" {
		t.Fatalf("log = %v", log)
	}
	log = nil
	ts.Pump()
	controller.JumpTo(550)
	want := "[*widgets.ScrollEndNotification *widgets.UserScrollNotification *widgets.ScrollStartNotification *widgets.ScrollUpdateNotification *widgets.ScrollEndNotification]"
	if controller.Offset() != 550 || fmt.Sprint(log) != want {
		t.Fatalf("offset %v, log = %v", controller.Offset(), log)
	}
	log = nil
	ts.Pump()
	gesture.MoveBy(ui.Offset{Dy: -100})
	if controller.Offset() != 550 || len(log) != 0 {
		t.Errorf("offset %v, log = %v", controller.Offset(), log)
	}
	gesture.Up()
}

func TestScrollViewsAreNotPrimaryByDefault(t *testing.T) {
	if NewCustomScrollView(nil).Primary().IsSome() || NewListView(nil).Primary().IsSome() || NewGridViewCount(1, nil).Primary().IsSome() {
		t.Error("a vertical scroll view has an explicit primary")
	}
}

func TestScrollViewsAndThePrimaryScrollController(t *testing.T) {
	builders := map[string]func(opts ScrollViewOpts) Widget{
		"CustomScrollView": func(o ScrollViewOpts) Widget { return NewCustomScrollView(nil, o) },
		"ListView": func(o ScrollViewOpts) Widget {
			return NewListView(nil, ListViewOpts{Primary: o.Primary, ScrollDirection: o.ScrollDirection, Controller: o.Controller})
		},
		"GridView": func(o ScrollViewOpts) Widget {
			return NewGridViewCount(1, nil, GridViewOpts{Primary: o.Primary, ScrollDirection: o.ScrollDirection, Controller: o.Controller})
		},
	}
	for name, build := range builders {
		for _, platform := range []foundation.TargetPlatform{foundation.TargetPlatformAndroid, foundation.TargetPlatformIOS, foundation.TargetPlatformFuchsia,
			foundation.TargetPlatformLinux, foundation.TargetPlatformMacOS, foundation.TargetPlatformWindows} {
			mobile := platform == foundation.TargetPlatformAndroid || platform == foundation.TargetPlatformIOS || platform == foundation.TargetPlatformFuchsia
			t.Run(fmt.Sprint(name, " on ", platform), func(t *testing.T) {
				usePlatform(t, platform)
				ts := gluttertest.New(t, gluttertest.Opts{})
				controller := NewScrollController()
				defer controller.Dispose()
				ts.PumpWidget(primaryScrollControllerApp(build(ScrollViewOpts{}), controller))
				if controller.HasClients() != mobile {
					t.Errorf("hasClients = %v on %v", controller.HasClients(), platform)
				}
			})
		}
		t.Run(name+" sets the PrimaryScrollController when primary", func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			controller := NewScrollController()
			defer controller.Dispose()
			ts.PumpWidget(ltr(NewPrimaryScrollController(controller, build(ScrollViewOpts{Primary: foundation.Some(true)}))))
			if got := ts.Widget(gluttertest.ByType[*Scrollable]()).(*Scrollable).Controller(); got != controller {
				t.Errorf("controller = %v", got)
			}
		})
		t.Run(name+" is not primary horizontally or with a controller", func(t *testing.T) {
			usePlatform(t, foundation.TargetPlatformAndroid)
			ts := gluttertest.New(t, gluttertest.Opts{})
			primary := NewScrollController()
			own := NewScrollController()
			defer primary.Dispose()
			defer own.Dispose()
			ts.PumpWidget(primaryScrollControllerApp(build(ScrollViewOpts{ScrollDirection: painting.AxisHorizontal}), primary)) // horizontal
			if primary.HasClients() {
				t.Error("a horizontal view used the primary controller")
			}
			ts.PumpWidget(primaryScrollControllerApp(build(ScrollViewOpts{Controller: own}), primary))
			if primary.HasClients() || !own.HasClients() {
				t.Error("a view with a controller used the primary controller")
			}
		})
	}
}

func TestNestedScrollablesHaveANilPrimaryScrollController(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	inner := foundation.NewValueKey("inner")
	controller := NewScrollController()
	defer controller.Dispose()
	ts.PumpWidget(ltr(NewPrimaryScrollController(controller, NewListView([]Widget{
		NewContainer(NewListView(nil, ListViewOpts{Key: inner, Primary: foundation.Some(true)}),
			ContainerOpts{Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxHeight: foundation.Some(200.0)}))})},
		ListViewOpts{Primary: foundation.Some(true)}))))
	scrollable := ts.Widget(gluttertest.Descendant(gluttertest.ByKey(inner), gluttertest.ByType[*Scrollable]())).(*Scrollable)
	if scrollable.Controller() != nil {
		t.Errorf("the inner controller is %v", scrollable.Controller())
	}
}

func TestListViewPrimaryAndPhysics(t *testing.T) {
	isAlways := func(p ScrollPhysics) bool { _, ok := p.(*AlwaysScrollableScrollPhysics); return ok }
	if !isAlways(NewListView(nil, ListViewOpts{Primary: foundation.Some(true)}).Physics()) {
		t.Error("a primary ListView is not always scrollable")
	}
	if isAlways(NewListView(nil, ListViewOpts{Primary: foundation.Some(false)}).Physics()) {
		t.Error("a non-primary ListView is always scrollable")
	}
	if !isAlways(NewListView(nil).Physics()) {
		t.Error("a defaulting-to-primary ListView is not always scrollable")
	}
	if isAlways(NewListView(nil, ListViewOpts{ScrollDirection: painting.AxisHorizontal}).Physics()) {
		t.Error("a horizontal ListView is always scrollable")
	}
}

func TestListViewPrimaryLeadsToScrolling(t *testing.T) {
	useAndroid(t)
	tests := []struct {
		name string
		opts ListViewOpts
		want bool
	}{
		{"primary:true leads to scrolling", ListViewOpts{Primary: foundation.Some(true)}, true},
		{"primary:false leads to no scrolling", ListViewOpts{Primary: foundation.Some(false)}, false},
		{"AlwaysScrollableScrollPhysics overrides primary:false", ListViewOpts{Primary: foundation.Some(false), Physics: NewAlwaysScrollableScrollPhysics(nil)}, true},
		{"ScrollPhysics overrides primary:true", ListViewOpts{Primary: foundation.Some(true), Physics: NewPlainScrollPhysics(nil)}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			scrolled := false
			ts.PumpWidget(ltr(NewNotificationListener[*OverscrollNotification](NewListView(nil, tc.opts), func(*OverscrollNotification) bool {
				scrolled = true
				return false
			})))
			ts.DragFrom(ui.Offset{Dx: 100, Dy: 100}, ui.Offset{Dy: 100})
			if scrolled != tc.want {
				t.Errorf("scrolled = %v, want %v", scrolled, tc.want)
			}
		})
	}
}

func TestListViewBuilderAssertions(t *testing.T) {
	skipUnlessDebug(t)
	builder := func(BuildContext, int) Widget { return NewSizedBox(nil) }
	requireTag(t, "ListViewItemExtentConflict", func() {
		NewListView(nil, ListViewOpts{ItemExtent: foundation.Some(20.0), PrototypeItem: NewSizedBox(nil)})
	})
	requireTag(t, "ListViewItemExtentConflict", func() {
		NewListViewBuilder(builder, ListViewOpts{ItemExtent: foundation.Some(20.0), PrototypeItem: NewSizedBox(nil)})
	})
}
