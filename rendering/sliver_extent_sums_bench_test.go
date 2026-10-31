package rendering_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// BenchmarkVariedExtentListJumpToEnd is a frame that jumps from the start to the end of a
// list of 100,000 rows of different heights (SPEC §11.4, step 29.40). "rebuilt" sets the
// builder before each jump, as a widget that is rebuilt does (the sums are made again);
// "same" keeps the builder. Run it with
//
//	go test ./rendering -run xxx -bench VariedExtentListJumpToEnd -benchmem
func BenchmarkVariedExtentListJumpToEnd(b *testing.B) {
	const rows = 100_000
	extents := randomExtents(rows, 7)
	total := sumOf(extents)
	for _, tc := range []struct {
		name    string
		rebuild bool
	}{{"rebuilt", true}, {"same", false}} {
		b.Run(tc.name, func(b *testing.B) {
			build := func(i int, _ rendering.SliverLayoutDimensions) foundation.Opt[float64] {
				if i < 0 || i >= len(extents) {
					return foundation.None[float64]()
				}
				return foundation.Some(extents[i])
			}
			manager := newLazyRowsManager(rows, true)
			sliver := rendering.NewRenderSliverVariedExtentList(manager, build)
			manager.insert, manager.remove = sliver.Insert, sliver.Remove
			s := renderhost.New(renderhost.Opts{Size: ui.Size{Width: 800, Height: 600}})
			s.Host.DiscardScenes = true
			root := rendering.NewRenderViewport(painting.AxisDirectionRight, rendering.NewViewportOffsetZero(), rendering.RenderViewportOpts{
				RenderViewportBaseOpts: rendering.RenderViewportBaseOpts{
					AxisDirection:     painting.AxisDirectionDown,
					ScrollCacheExtent: foundation.Some(rendering.NewScrollCacheExtentPixels(0)),
				},
				Children: []rendering.Sliver{sliver},
			})
			s.SetRoot(root)
			s.Pump()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if tc.rebuild {
					sliver.SetItemExtentBuilder(build)
				}
				root.SetOffset(rendering.NewViewportOffsetFixed(total - 600))
				s.Pump()
				if tc.rebuild {
					sliver.SetItemExtentBuilder(build)
				}
				root.SetOffset(rendering.NewViewportOffsetFixed(0))
				s.Pump()
			}
		})
	}
}
