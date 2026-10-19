package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// TestViewportGetOffsetToReveal is 'Viewport getOffsetToReveal - down/right/up/
// left' of viewport_test.dart, with the sliver list and the viewport laid out
// directly instead of through a ListView.
func TestViewportGetOffsetToReveal(t *testing.T) {
	type reveal struct {
		alignment  float64
		rect       foundation.Opt[ui.Rect]
		wantOffset float64
		wantRect   ui.Rect
	}
	small := foundation.Some(ui.NewRectFromLTWH(40, 40, 10, 10))
	none := foundation.None[ui.Rect]()
	tests := []struct {
		name     string
		axis     painting.AxisDirection
		cross    painting.AxisDirection
		viewport ui.Size
		item     ui.Size
		reveals  []reveal
	}{
		{"down", painting.AxisDirectionDown, painting.AxisDirectionRight, ui.Size{Width: 300, Height: 200}, ui.Size{Width: 300, Height: 100}, []reveal{
			{0, none, 500, ui.NewRectFromLTWH(0, 0, 300, 100)},
			{1, none, 400, ui.NewRectFromLTWH(0, 100, 300, 100)},
			{0, small, 540, ui.NewRectFromLTWH(40, 0, 10, 10)},
			{1, small, 350, ui.NewRectFromLTWH(40, 190, 10, 10)},
		}},
		{"right", painting.AxisDirectionRight, painting.AxisDirectionDown, ui.Size{Width: 200, Height: 300}, ui.Size{Width: 100, Height: 300}, []reveal{
			{0, none, 500, ui.NewRectFromLTWH(0, 0, 100, 300)},
			{1, none, 400, ui.NewRectFromLTWH(100, 0, 100, 300)},
			{0, small, 540, ui.NewRectFromLTWH(0, 40, 10, 10)},
			{1, small, 350, ui.NewRectFromLTWH(190, 40, 10, 10)},
		}},
		{"up", painting.AxisDirectionUp, painting.AxisDirectionRight, ui.Size{Width: 300, Height: 200}, ui.Size{Width: 300, Height: 100}, []reveal{
			{0, none, 500, ui.NewRectFromLTWH(0, 100, 300, 100)},
			{1, none, 400, ui.NewRectFromLTWH(0, 0, 300, 100)},
			{0, small, 550, ui.NewRectFromLTWH(40, 190, 10, 10)},
			{1, small, 360, ui.NewRectFromLTWH(40, 0, 10, 10)},
		}},
		{"left", painting.AxisDirectionLeft, painting.AxisDirectionDown, ui.Size{Width: 200, Height: 300}, ui.Size{Width: 100, Height: 300}, []reveal{
			{0, none, 500, ui.NewRectFromLTWH(100, 0, 100, 300)},
			{1, none, 400, ui.NewRectFromLTWH(0, 0, 100, 300)},
			{0, small, 550, ui.NewRectFromLTWH(190, 40, 10, 10)},
			{1, small, 360, ui.NewRectFromLTWH(0, 40, 10, 10)},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			boxes := sizedBoxes(20, tc.item.Width, tc.item.Height)
			manager := newTestChildManager(asBoxes(boxes)...)
			viewport := rendering.NewRenderViewport(tc.cross, rendering.NewViewportOffsetFixed(300), rendering.RenderViewportOpts{
				RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{AxisDirection: tc.axis},
				Children:               []rendering.Sliver{manager.createList()},
			})
			layout(t, newHarness(t), viewport, withConstraints(rendering.NewBoxConstraintsTight(tc.viewport)))
			target := boxes[5]
			for _, r := range tc.reveals {
				var opts []rendering.GetOffsetToRevealOpts
				if r.rect.IsSome() {
					opts = append(opts, rendering.GetOffsetToRevealOpts{Rect: r.rect})
				}
				got := viewport.GetOffsetToReveal(target, r.alignment, opts...)
				if got.Offset != r.wantOffset || got.Rect != r.wantRect {
					t.Errorf("GetOffsetToReveal(alignment %v, rect %v) = %v, want offset %v rect %v", r.alignment, r.rect, got, r.wantOffset, r.wantRect)
				}
			}
			if found := rendering.RenderAbstractViewportMaybeOf(target); found != rendering.RenderAbstractViewport(viewport) {
				t.Errorf("RenderAbstractViewportMaybeOf(target) = %v, want the viewport", found)
			}
		})
	}
}
