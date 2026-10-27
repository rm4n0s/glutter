package gles

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/ui"
)

// BenchmarkScrolledList is the time of building a frame of a tall list drawn through a small
// viewport, on the fake GL: it must grow with the rows that show, not with the rows there are
// (SPEC §9.1, "Culling"). The scene is flattened once, outside the timing. Run it with
//
//	go test ./internal/gpu/gles -run xxx -bench ScrolledList -benchmem
func BenchmarkScrolledList(b *testing.B) {
	for _, rows := range []int{50, 500, 5000} {
		b.Run(fmt.Sprintf("rows=%d", rows), func(b *testing.B) {
			gl := &fakeGL{versionString: "OpenGL ES 3.2 Mesa 24.1.0", vendor: "Fake", renderer: "FakeGPU"}
			d, err := newDevice(gl)
			if err != nil {
				b.Fatal(err)
			}
			res, err := newResources(d, gl)
			if err != nil {
				b.Fatal(err)
			}
			var lists [2]*gpu.DrawList
			for i, scroll := range [2]float64{0, 2 * gputest.ScrolledRowHeight} {
				lists[i], err = gpu.Flatten(gputest.ScrolledList(rows, scroll), gpu.FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, gputest.ScrolledWidth, gputest.ScrolledHeight)})
				if err != nil {
					b.Fatal(err)
				}
			}
			frame := func(list *gpu.DrawList) {
				res.beginFrame()
				fb := &frameBuilder{w: gputest.ScrolledWidth, h: gputest.ScrolledHeight, res: res}
				for _, c := range list.Commands {
					if err := fb.command(c); err != nil {
						b.Fatal(err)
					}
				}
				res.endFrame()
			}
			frame(lists[0]) // warm up: the atlas
			frame(lists[1])
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				frame(lists[i%2])
			}
		})
	}
}
