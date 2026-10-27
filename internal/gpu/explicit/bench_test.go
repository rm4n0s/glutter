package explicit_test

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/gputest/editorscene"
	"github.com/rm4n0s/glutter/ui"
)

// BenchmarkScrolledList is the time of a frame of a tall list drawn through a small viewport, on
// the fake device: it must grow with the rows that show, not with the rows there are (SPEC §9.3,
// "Culling"). Run it with
//
//	go test ./internal/gpu/explicit -run xxx -bench ScrolledList -benchmem
func BenchmarkScrolledList(b *testing.B) {
	for _, rows := range []int{50, 500, 5000} {
		b.Run(fmt.Sprintf("rows=%d", rows), func(b *testing.B) {
			dev := fakedevice.New()
			dev.CompleteReadsOnSubmit = true
			r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) { return dev, nil })
			defer r.Close()
			// Two positions of the list, a whole number of pixels apart, so that the frames scroll
			// and the atlas is asked for the tiles of the rows that come into view.
			scenes := [2]*ui.Scene{
				gputest.ScrolledList(rows, 0),
				gputest.ScrolledList(rows, 2*gputest.ScrolledRowHeight),
			}
			for i := 0; i < 2; i++ { // warm up: the atlas and the pipelines
				if _, err := r.Draw(window, gputest.ScrolledWidth, gputest.ScrolledHeight, scenes[i%2], false); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := r.Draw(window, gputest.ScrolledWidth, gputest.ScrolledHeight, scenes[i%2], false); err != nil {
					b.Fatal(err)
				}
				dev.Reset()
			}
		})
	}
}

// BenchmarkCaretFrame is the renderer's share of a frame in which only the caret of a text editor
// blinks (SPEC §15): the two scenes are the caret on and off in examples/editor's tree with a long
// text, drawn with the options the engine gives the renderer (damage, raster cache). The bytes and
// allocations of a frame must not grow with the length of the text. Run it with
//
//	go test ./internal/gpu/explicit -run xxx -bench CaretFrame -benchmem
func BenchmarkCaretFrame(b *testing.B) {
	for _, lines := range []int{50, 500, 5000} {
		var on, off *ui.Scene // made once: b.Run calls the function again for each b.N
		b.Run(fmt.Sprintf("lines=%d", lines), func(b *testing.B) {
			if on == nil {
				on, off = editorscene.CaretScenes(b, lines)
			}
			dev := fakedevice.New()
			dev.CompleteReadsOnSubmit = true
			dev.Quiet = true
			r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) { return dev, nil },
				explicit.RendererOpts{Background: ui.NewColor(0xFF000000), HasBackground: true, Damage: true, RasterCacheAfter: 3})
			defer r.Close()
			scenes := [2]*ui.Scene{on, off}
			w, h := editorscene.Width, editorscene.Height
			for i := 0; i < 8; i++ { // warm up: the atlas, the pipelines, the raster cache
				if _, err := r.Draw(window, w, h, scenes[i%2], false); err != nil {
					b.Fatal(err)
				}
				dev.Reset()
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := r.Draw(window, w, h, scenes[i%2], false); err != nil {
					b.Fatal(err)
				}
				dev.Reset()
			}
		})
	}
}
