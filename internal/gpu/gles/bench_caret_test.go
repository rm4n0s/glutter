package gles_test

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/gpu/gputest/editorscene"
	"github.com/rm4n0s/glutter/ui"
)

// BenchmarkCaretFrame is the renderer's share of a frame in which only the caret of a text editor
// blinks (SPEC §15), on the fake GL, with the options the engine gives the renderer: the bytes and
// allocations of a frame must not grow with the length of the text. Run it with
//
//	go test ./internal/gpu/gles -run xxx -bench CaretFrame -benchmem
func BenchmarkCaretFrame(b *testing.B) {
	for _, lines := range []int{50, 500, 5000} {
		var on, off *ui.Scene // made once: b.Run calls the function again for each b.N
		b.Run(fmt.Sprintf("lines=%d", lines), func(b *testing.B) {
			if on == nil {
				on, off = editorscene.CaretScenes(b, lines)
			}
			draw, closeRig, err := gles.NewBenchRig(editorscene.Width, editorscene.Height,
				gles.RendererOpts{Background: ui.NewColor(0xFF000000), HasBackground: true, Damage: true, RasterCacheAfter: 3})
			if err != nil {
				b.Fatal(err)
			}
			defer closeRig()
			scenes := [2]*ui.Scene{on, off}
			for i := 0; i < 8; i++ { // warm up: the atlas, the programs, the raster cache
				if err := draw(scenes[i%2]); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := draw(scenes[i%2]); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
