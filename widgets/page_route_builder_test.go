package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// TestPageRouteBuilderBarriersShow is 'Barriers show when using PageRouteBuilder'
// of page_route_builder_test.dart. The texts of the Dart page are boxes here:
// golden images do not depend on a font.
func TestPageRouteBuilderBarriersShow(t *testing.T) {
	debugBlack54 := ui.NewColor(0x8A000000)
	debugTeal := ui.NewColor(0xFF009688)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 400, Height: 300}})

	modalPage := func(bc BuildContext, _, _ animation.Animation[float64]) Widget {
		return NewSafeArea(NewStack([]Widget{
			NewGestureDetector(NewSizedBoxExpand(nil), GestureDetectorOpts{OnTap: func() { NavigatorPop(bc) }}),
			NewAlign(NewContainer(nil, ContainerOpts{Height: foundation.Some(150.0), Color: foundation.Some(debugTeal)}),
				AlignOpts{Alignment: painting.AlignmentBottomCenter}),
		}))
	}
	home := NewBuilder(func(bc BuildContext) Widget {
		return NewColumn([]Widget{
			NewExpanded(NewCenter(NewColoredBox(ui.NewColor(0xFFE0E0E0), NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(120.0), Height: foundation.Some(40.0)})))),
			NewAlign(NewGestureDetector(NewColoredBox(ui.NewColor(0xFF6750A4), NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(40.0), Height: foundation.Some(40.0)})),
				GestureDetectorOpts{OnTap: func() {
					NavigatorPush(bc, NewPageRouteBuilder(modalPage, PageRouteBuilderOpts{
						BarrierColor: foundation.Some(debugBlack54),
						NotOpaque:    true,
					}))
				}, Behavior: rendering.HitTestBehaviorOpaque}), AlignOpts{Alignment: painting.AlignmentBottomRight}),
		})
	})
	ts.PumpWidget(navTestApp(navTestAppOpts{home: home}))
	ts.Tap(gluttertest.ByType[*GestureDetector]())
	ts.PumpAndSettle()
	if got := ts.Count(gluttertest.ByType[*ModalBarrier]()); got != 2 {
		t.Fatalf("%d barriers, want 2", got)
	}
	ts.MatchesGolden("page_route_builder_barrier")
}
