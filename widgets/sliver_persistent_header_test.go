package widgets_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/sliver_persistent_header_test.dart.
// Not ported: the semantics tests, the stretch and show on screen configuration
// tests (they read private widgets), the pointer scroll test and the
// toStringDeep test.

// testDelegate is TestDelegate: a header of 200 pixels.
type testDelegate struct {
	SliverPersistentHeaderDelegateBase
}

func (testDelegate) Build(BuildContext, float64, bool) Widget {
	return NewSizedBox(NewText("Sliver Persistent Header"), SizedBoxOpts{Height: foundation.Some(200.0)})
}
func (testDelegate) MinExtent() float64                                { return 200 }
func (testDelegate) MaxExtent() float64                                { return 200 }
func (testDelegate) ShouldRebuild(SliverPersistentHeaderDelegate) bool { return false }

// testDelegate2 is TestDelegate2: a header from 100 to 200 pixels.
type testDelegate2 struct {
	SliverPersistentHeaderDelegateBase
}

func (testDelegate2) Build(BuildContext, float64, bool) Widget {
	return NewContainer(nil, ContainerOpts{Constraints: foundation.Some(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{
		MinHeight: foundation.Some(100.0), MaxHeight: foundation.Some(200.0)}))})
}
func (testDelegate2) MinExtent() float64                                { return 100 }
func (testDelegate2) MaxExtent() float64                                { return 200 }
func (testDelegate2) ShouldRebuild(SliverPersistentHeaderDelegate) bool { return false }

func bigSliver(key foundation.Key, height float64) Widget {
	return NewSliverToBoxAdapter(NewSizedBox(nil, SizedBoxOpts{Height: foundation.Some(height)}), SliverToBoxAdapterOpts{Key: key})
}

func verifyPaintPosition(t *testing.T, key *GlobalKey[State], ideal ui.Offset, visible ...bool) {
	t.Helper()
	target := key.CurrentContext().FindRenderObject().(rendering.Sliver)
	if _, ok := target.AsRenderObject().Parent().(*rendering.RenderViewport); !ok {
		t.Fatalf("the parent is %T", target.AsRenderObject().Parent())
	}
	if got := rendering.SliverPhysicalParentDataOf(target).PaintOffset; got != ideal {
		t.Errorf("paint offset = %v, want %v", got, ideal)
	}
	if len(visible) > 0 {
		if got := target.AsRenderSliver().Geometry().Visible; got != visible[0] {
			t.Errorf("visible = %v, want %v", got, visible[0])
		}
	}
}

func verifyActualBoxPosition(t *testing.T, ts *gluttertest.Tester, finder gluttertest.Finder, ideal ui.Rect) {
	t.Helper()
	box := ts.RenderObject(gluttertest.First(finder)).(rendering.Box)
	got := ui.NewRectFromPoints(box.LocalToGlobal(ui.OffsetZero, nil), box.LocalToGlobal(box.Size().BottomRight(ui.OffsetZero), nil))
	if got != ideal {
		t.Errorf("box rect = %v, want %v", got, ideal)
	}
}

func animateAndSettle(ts *gluttertest.Tester, position ScrollPosition, to float64) {
	position.AnimateTo(to, time.Minute, animation.Curves.Linear)
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: time.Second})
}

func TestSliverPersistentHeaderFloatingScrollOffsetDoesNotChange(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	const bigHeight = 1000.0
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{bigSliver(nil, bigHeight),
		NewSliverPersistentHeader(testDelegate{}, SliverPersistentHeaderOpts{Floating: true}), bigSliver(nil, bigHeight)})))
	position := scrollableOf(ts).Position()
	max := bigHeight*2 + 200 - 600 // 600 is the height of the test viewport
	if max != 1600 || position.Pixels() != 0 || position.MinScrollExtent() != 0 || position.MaxScrollExtent() != max {
		t.Fatalf("pixels %v min %v max %v", position.Pixels(), position.MinScrollExtent(), position.MaxScrollExtent())
	}
	position.AnimateTo(10000, time.Minute, animation.Curves.Linear)
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 50 * time.Millisecond})
	if position.Pixels() != max || position.MinScrollExtent() != 0 || position.MaxScrollExtent() != max {
		t.Errorf("pixels %v min %v max %v", position.Pixels(), position.MinScrollExtent(), position.MaxScrollExtent())
	}
}

func TestSliverPersistentHeaderFloatingNormalBehavior(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	const bigHeight = 1000.0
	key1, key2, key3 := NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State]()
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{bigSliver(key1, bigHeight),
		NewSliverPersistentHeader(testDelegate2{}, SliverPersistentHeaderOpts{Key: key2, Floating: true}), bigSliver(key3, bigHeight)})))
	position := scrollableOf(ts).Position()
	container := gluttertest.ByType[*Container]()
	const maxExtent = 200.0
	off := func(dy float64) ui.Offset { return ui.Offset{Dy: dy} }

	verifyPaintPosition(t, key1, ui.Offset{}, true)
	verifyPaintPosition(t, key2, off(1000), false)
	verifyPaintPosition(t, key3, off(1200), false)

	animateAndSettle(ts, position, bigHeight-600+maxExtent)
	verifyPaintPosition(t, key1, ui.Offset{}, true)
	verifyPaintPosition(t, key2, off(600-maxExtent), true)
	verifyActualBoxPosition(t, ts, container, ui.NewRectFromLTWH(0, 600-maxExtent, 800, maxExtent))
	verifyPaintPosition(t, key3, off(600), false)

	animateAndSettle(ts, position, bigHeight-600+maxExtent*2)
	verifyPaintPosition(t, key1, ui.Offset{}, true)
	verifyPaintPosition(t, key2, off(600-maxExtent*2), true)
	verifyActualBoxPosition(t, ts, container, ui.NewRectFromLTWH(0, 600-maxExtent*2, 800, maxExtent))
	verifyPaintPosition(t, key3, off(600-maxExtent), true)

	animateAndSettle(ts, position, bigHeight)
	verifyPaintPosition(t, key1, ui.Offset{}, false)
	verifyPaintPosition(t, key2, ui.Offset{}, true)
	verifyActualBoxPosition(t, ts, container, ui.NewRectFromLTWH(0, 0, 800, maxExtent))
	verifyPaintPosition(t, key3, off(maxExtent), true)

	animateAndSettle(ts, position, bigHeight+maxExtent*0.1)
	verifyPaintPosition(t, key1, ui.Offset{}, false)
	verifyPaintPosition(t, key2, ui.Offset{}, true)
	verifyActualBoxPosition(t, ts, container, ui.NewRectFromLTWH(0, 0, 800, maxExtent*0.9))
	verifyPaintPosition(t, key3, off(maxExtent*0.9), true)

	animateAndSettle(ts, position, bigHeight+maxExtent*0.5)
	verifyPaintPosition(t, key2, ui.Offset{}, true)
	verifyActualBoxPosition(t, ts, container, ui.NewRectFromLTWH(0, 0, 800, maxExtent*0.5))
	verifyPaintPosition(t, key3, off(maxExtent*0.5), true)

	animateAndSettle(ts, position, bigHeight+maxExtent*0.9)
	verifyPaintPosition(t, key2, ui.Offset{}, true)
	verifyActualBoxPosition(t, ts, container, ui.NewRectFromLTWH(0, -maxExtent*0.4, 800, maxExtent*0.5))
	verifyPaintPosition(t, key3, off(maxExtent*0.1), true)

	animateAndSettle(ts, position, bigHeight+maxExtent*2)
	verifyPaintPosition(t, key1, ui.Offset{}, false)
	verifyPaintPosition(t, key2, ui.Offset{}, false)
	verifyPaintPosition(t, key3, ui.Offset{}, true)
}

func TestSliverPersistentHeaderFloatingNoFloatingWhenAnimating(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	const bigHeight = 1000.0
	key1, key2, key3 := NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State]()
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{bigSliver(key1, bigHeight),
		NewSliverPersistentHeader(testDelegate{}, SliverPersistentHeaderOpts{Key: key2, Floating: true}), bigSliver(key3, bigHeight)})))
	position := scrollableOf(ts).Position()
	verifyPaintPosition(t, key1, ui.Offset{}, true)
	verifyPaintPosition(t, key2, ui.Offset{Dy: 1000}, false)
	verifyPaintPosition(t, key3, ui.Offset{Dy: 1200}, false)
	for _, to := range []float64{bigHeight + 400, bigHeight + 380} {
		animateAndSettle(ts, position, to)
		verifyPaintPosition(t, key1, ui.Offset{}, false)
		verifyPaintPosition(t, key2, ui.Offset{}, false)
		verifyPaintPosition(t, key3, ui.Offset{}, true)
	}
}

func TestSliverPersistentHeaderFloatingBehaviorWhenDraggingDown(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	const bigHeight = 1000.0
	key1, key2, key3 := NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State]()
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{bigSliver(key1, bigHeight),
		NewSliverPersistentHeader(testDelegate2{}, SliverPersistentHeaderOpts{Key: key2, Floating: true}), bigSliver(key3, bigHeight)})))
	position := scrollableOf(ts).Position().(*ScrollPositionWithSingleContext)
	animateAndSettle(ts, position, bigHeight+400)
	verifyPaintPosition(t, key1, ui.Offset{}, false)
	verifyPaintPosition(t, key2, ui.Offset{}, false)
	verifyPaintPosition(t, key3, ui.Offset{}, true)

	position.AnimateTo(bigHeight+380, time.Minute, animation.Curves.Linear)
	position.UpdateUserScrollDirection(rendering.ScrollDirectionForward)
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: time.Second})
	verifyPaintPosition(t, key1, ui.Offset{}, false)
	verifyPaintPosition(t, key2, ui.Offset{}, true)
	verifyActualBoxPosition(t, ts, gluttertest.ByType[*Container](), ui.NewRectFromLTWH(0, -200*0.4, 800, 200*0.5))
	verifyPaintPosition(t, key3, ui.Offset{}, true)
}

func TestSliverPersistentHeaderOverscrollGapIsBelowHeader(t *testing.T) {
	tests := []struct {
		name     string
		floating bool
	}{{"floating", true}, {"not floating", false}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			ts.PumpWidget(ltr(NewCustomScrollView([]Widget{
				NewSliverPersistentHeader(testDelegate{}, SliverPersistentHeaderOpts{Floating: tc.floating}),
				NewSliverListList([]Widget{NewSizedBox(NewText("X"), SizedBoxOpts{Height: foundation.Some(300.0)})})},
				CustomScrollViewOpts{Physics: NewBouncingScrollPhysics()})))
			header := gluttertest.Ancestor(gluttertest.ByText("Sliver Persistent Header"), gluttertest.ByType[*SizedBox]())
			if ts.TopLeft(header) != (ui.Offset{}) || ts.TopLeft(gluttertest.ByText("X")) != (ui.Offset{Dy: 200}) {
				t.Fatalf("header at %v, X at %v", ts.TopLeft(header), ts.TopLeft(gluttertest.ByText("X")))
			}
			scrollableOf(ts).Position().JumpTo(-50)
			ts.Pump()
			// The overscroll gap is below the header.
			if ts.TopLeft(header) != (ui.Offset{}) || ts.TopLeft(gluttertest.ByText("X")) != (ui.Offset{Dy: 250}) {
				t.Errorf("header at %v, X at %v", ts.TopLeft(header), ts.TopLeft(gluttertest.ByText("X")))
			}
		})
	}
}

func TestSliverPersistentHeaderPinned(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	const bigHeight = 550.0
	k := []*GlobalKey[State]{NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State]()}
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{bigSliver(k[0], bigHeight),
		NewSliverPersistentHeader(testDelegate2{}, SliverPersistentHeaderOpts{Key: k[1], Pinned: true}),
		NewSliverPersistentHeader(testDelegate2{}, SliverPersistentHeaderOpts{Key: k[2], Pinned: true}),
		bigSliver(k[3], bigHeight), bigSliver(k[4], bigHeight)})))
	position := scrollableOf(ts).Position()
	max := bigHeight*3 + 200*2 - 600
	if max != 1450 || position.Pixels() != 0 || position.MinScrollExtent() != 0 || position.MaxScrollExtent() != max {
		t.Fatalf("pixels %v min %v max %v", position.Pixels(), position.MinScrollExtent(), position.MaxScrollExtent())
	}
	position.AnimateTo(10000, time.Minute, animation.Curves.Linear)
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 10 * time.Millisecond})
	if position.Pixels() != max {
		t.Errorf("pixels = %v, want %v", position.Pixels(), max)
	}
	verifyPaintPosition(t, k[0], ui.Offset{}, false)
	verifyPaintPosition(t, k[1], ui.Offset{}, true)
	verifyPaintPosition(t, k[2], ui.Offset{Dy: 100}, true)
	verifyPaintPosition(t, k[3], ui.Offset{}, true)
	verifyPaintPosition(t, k[4], ui.Offset{Dy: 50}, true)
}

func TestSliverPersistentHeaderScrolling(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	const bigHeight = 550.0
	k := []*GlobalKey[State]{NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State](), NewGlobalKey[State]()}
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{bigSliver(k[0], bigHeight),
		NewSliverPersistentHeader(testDelegate{}, SliverPersistentHeaderOpts{Key: k[1]}),
		NewSliverPersistentHeader(testDelegate{}, SliverPersistentHeaderOpts{Key: k[2]}),
		bigSliver(k[3], bigHeight), bigSliver(k[4], bigHeight)})))
	position := scrollableOf(ts).Position()
	max := bigHeight*3 + 200*2 - 600
	if max != 1450 || position.MaxScrollExtent() != max {
		t.Fatalf("max = %v", position.MaxScrollExtent())
	}
	position.AnimateTo(10000, time.Minute, animation.Curves.Linear)
	ts.PumpAndSettle(gluttertest.PumpAndSettleOpts{Duration: 10 * time.Millisecond})
	if position.Pixels() != max {
		t.Errorf("pixels = %v, want %v", position.Pixels(), max)
	}
	for i := 0; i < 4; i++ {
		verifyPaintPosition(t, k[i], ui.Offset{})
	}
	verifyPaintPosition(t, k[4], ui.Offset{Dy: 50})
}

func TestSliverPersistentHeaderScrollingOffScreen(t *testing.T) {
	ts := gluttertest.New(t, gluttertest.Opts{})
	const bigHeight = 550.0
	ts.PumpWidget(ltr(NewCustomScrollView([]Widget{bigSliver(nil, bigHeight),
		NewSliverPersistentHeader(testDelegate{}), bigSliver(nil, bigHeight), bigSliver(nil, bigHeight)})))
	animateAndSettle(ts, scrollableOf(ts).Position(), bigHeight+200-5)
	box := ts.RenderObject(gluttertest.ByText("Sliver Persistent Header")).(rendering.Box)
	got := ui.NewRectFromPoints(box.LocalToGlobal(ui.OffsetZero, nil), box.LocalToGlobal(box.Size().BottomRight(ui.OffsetZero), nil))
	if got != ui.NewRectFromLTWH(0, -195, 800, 200) {
		t.Errorf("rect = %v", got)
	}
	_ = painting.AxisVertical
}
