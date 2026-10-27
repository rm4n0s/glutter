package explicit_test

import (
	"sync"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/internal/gpu/gputest/editorscene"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/ui"
)

// caretFrameBound is what a frame in which only a caret blinks may allocate in the renderer, with
// a text of any length (step 29.15). A frame allocated 6 MB with 50 lines of text and 600 MB with
// 5,000 before; it allocates about 40 KB now.
const caretFrameBound = 256 << 10

func TestCaretFrameAllocatesLittleAndNothingForTheLengthOfTheText(t *testing.T) {
	tests := []struct {
		name  string
		lines int
	}{{"a short text", 60}, {"a long text", 1500}}
	var perFrame []uint64
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			on, off := editorscene.CaretScenes(t, tt.lines)
			dev := fakedevice.New()
			dev.CompleteReadsOnSubmit, dev.Quiet = true, true
			r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) { return dev, nil },
				explicit.RendererOpts{Background: ui.NewColor(0xFF000000), HasBackground: true, Damage: true, RasterCacheAfter: 3})
			defer r.Close()
			scenes := [2]*ui.Scene{on, off}
			i := 0
			draw := func() {
				if _, err := r.Draw(window, editorscene.Width, editorscene.Height, scenes[i%2], false); err != nil {
					t.Fatal(err)
				}
				i++
			}
			for range 8 { // the atlas, the pipelines, the buffers and the raster cache fill
				draw()
			}
			bytes := editorscene.BytesPerFrame(30, draw)
			t.Logf("%d bytes for each frame, %+v", bytes, r.LastFrame())
			if bytes > caretFrameBound {
				t.Errorf("a caret frame allocates %d bytes, want at most %d", bytes, caretFrameBound)
			}
			// The layers of the text that did not change are drawn from the raster cache, and the
			// text outside the window is not expanded: the frame holds the commands of the window.
			if st := r.LastFrame(); st.CachedLayersDrawn == 0 || st.Commands > 4000 {
				t.Errorf("cached layers drawn %d, commands %d: the unchanged layers must come from the cache", st.CachedLayersDrawn, st.Commands)
			}
			perFrame = append(perFrame, bytes)
		})
	}
	if len(perFrame) == 2 && perFrame[1] > perFrame[0]+perFrame[0]/2+8<<10 {
		t.Errorf("%d bytes for a frame of the long text and %d for the short one: the bytes grow with the text", perFrame[1], perFrame[0])
	}
}

// Two renderers draw the frames of one detached scene on two goroutines at once (SPEC §9.9): what
// a renderer reuses between frames is its own, and what the scene shares is only read. Run with
// -race.
func TestTwoRenderersDrawOneDetachedCaretSceneAtOnce(t *testing.T) {
	on, off := editorscene.CaretScenes(t, 300)
	d := offscreen.NewDetacher()
	scenes := [2]*ui.Scene{d.Detach(on), d.Detach(off)}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dev := fakedevice.New()
			dev.CompleteReadsOnSubmit, dev.Quiet = true, true
			r := explicit.NewRenderer(func(gpu.SurfaceHandle) (explicit.Device, error) { return dev, nil },
				explicit.RendererOpts{Damage: true, RasterCacheAfter: 3})
			defer r.Close()
			for i := 0; i < 20; i++ {
				if _, err := r.Draw(window, editorscene.Width, editorscene.Height, scenes[i%2], false); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
