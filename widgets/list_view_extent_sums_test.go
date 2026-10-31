package widgets_test

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/rendering"
	. "github.com/rm4n0s/glutter/widgets"
)

// TestListViewItemExtentBuilderJumpToEnd jumps to the end of 100,000 rows of different
// heights: the scroll extent is the total of the extents, so the jump lands on the last
// row in one frame, and the builder is asked once for each row at most (SPEC §11.4, step 29.40).
func TestListViewItemExtentBuilderJumpToEnd(t *testing.T) {
	const rows = 100_000
	extentOf := func(i int) float64 { return 20 + float64(i%7)*5 }
	total := 0.0
	for i := 0; i < rows; i++ {
		total += extentOf(i)
	}
	tests := []struct {
		name string
		opts func(calls map[int]int) ListViewOpts
	}{
		{"known item count", func(calls map[int]int) ListViewOpts {
			return ListViewOpts{ItemCount: foundation.Some(rows), ItemExtentBuilder: func(i int, _ rendering.SliverLayoutDimensions) foundation.Opt[float64] {
				calls[i]++
				return foundation.Some(extentOf(i))
			}}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{})
			controller := NewScrollController()
			defer controller.Dispose()
			calls := map[int]int{}
			opts := tc.opts(calls)
			opts.Controller = controller
			ts.PumpWidget(ltr(NewListViewBuilder(func(_ BuildContext, i int) Widget {
				return NewText(fmt.Sprintf("Item %d", i))
			}, opts)))
			if got := controller.Position().MaxScrollExtent(); got != total-600 {
				t.Fatalf("max scroll extent = %v, want the total %v less the viewport", got, total-600)
			}
			controller.JumpTo(controller.Position().MaxScrollExtent())
			ts.Pump()
			expectTexts(t, ts, []string{fmt.Sprintf("Item %d", rows-1)}, []string{"Item 0"})
			if got := controller.Offset(); got != total-600 {
				t.Errorf("offset = %v, want %v", got, total-600)
			}
			for row, n := range calls {
				if n > 1 {
					t.Fatalf("row %d was asked for %d times", row, n)
				}
			}
			// and back to the start, and to the middle
			controller.JumpTo(0)
			ts.Pump()
			expectTexts(t, ts, []string{"Item 0"}, []string{fmt.Sprintf("Item %d", rows-1)})
			for row, n := range calls {
				if n > 1 {
					t.Fatalf("row %d was asked for %d times after the jump back", row, n)
				}
			}
		})
	}
}
