package svg

import (
	"fmt"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/internal/leaktest"
	"github.com/rm4n0s/glutter/widgets"
)

// TestLeaksSvgPictures shows a picture that differs each time, with a cache of a small limit (step
// 29.21, SPEC §15 "Memory over time"): the cache stays within its limit, and the heap and the
// goroutines stay flat (the pictures load on goroutines of their own).
func TestLeaksSvgPictures(t *testing.T) {
	const limit = 8 << 10
	biggest := 0
	defer func() {
		if biggest == 0 {
			t.Error("nothing was ever cached: the test does not exercise the cache")
		}
	}()
	leaktest.Run(t, "svg/pictures that differ", func() func() {
		ts := gluttertest.New(t, gluttertest.Opts{DiscardScenes: true})
		cache := NewCache(CacheOpts{MaximumSizeBytes: foundation.Some(limit)})
		show := true
		var setShow widgets.StateSetter
		i := 0
		ts.PumpWidget(widgets.NewDirectionality(0, NewSvgCacheScope(cache, widgets.NewStatefulBuilder(func(_ widgets.BuildContext, setState widgets.StateSetter) widgets.Widget {
			setShow = setState
			if !show {
				return widgets.NewSizedBox(nil)
			}
			return widgets.NewSizedBox(NewSvgPictureString(fmt.Sprintf(
				`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="10"><rect width="%d" height="10" fill="#f00"/></svg>`, i%1000)),
				widgets.SizedBoxOpts{Width: foundation.Some(40.0), Height: foundation.Some(20.0)})
		}))))
		return func() {
			i++
			setShow(func() { show = true })
			ts.Pump(16 * time.Millisecond)
			setShow(func() { show = false })
			ts.Pump(16 * time.Millisecond)
			biggest = max(biggest, cache.SizeBytes())
			if n := cache.SizeBytes(); n > limit {
				t.Fatalf("the cache holds %d bytes, its limit is %d", n, limit)
			}
		}
	}, leaktest.Opts{N: 150, Warmup: 100})
}
