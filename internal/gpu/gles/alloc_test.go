package gles_test

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/gpu/gputest/editorscene"
	"github.com/rm4n0s/glutter/ui"
)

// caretFrameBound is what a frame in which only a caret blinks may allocate in the renderer, with
// a text of any length (step 29.15). A frame allocated 6 MB with 50 lines of text and 600 MB with
// 5,000 before; it allocates about 30 KB now.
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
			draw, closeRig, err := gles.NewBenchRig(editorscene.Width, editorscene.Height,
				gles.RendererOpts{Background: ui.NewColor(0xFF000000), HasBackground: true, Damage: true, RasterCacheAfter: 3})
			if err != nil {
				t.Fatal(err)
			}
			defer closeRig()
			scenes := [2]*ui.Scene{on, off}
			i := 0
			frame := func() {
				if err := draw(scenes[i%2]); err != nil {
					t.Fatal(err)
				}
				i++
			}
			for range 8 { // the atlas, the programs, the buffers and the raster cache fill
				frame()
			}
			bytes := editorscene.BytesPerFrame(30, frame)
			t.Logf("%d bytes for each frame", bytes)
			if bytes > caretFrameBound {
				t.Errorf("a caret frame allocates %d bytes, want at most %d", bytes, caretFrameBound)
			}
			perFrame = append(perFrame, bytes)
		})
	}
	if len(perFrame) == 2 && perFrame[1] > perFrame[0]+perFrame[0]/2+8<<10 {
		t.Errorf("%d bytes for a frame of the long text and %d for the short one: the bytes grow with the text", perFrame[1], perFrame[0])
	}
}
