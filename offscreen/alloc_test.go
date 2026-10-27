package offscreen_test

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/gputest/editorscene"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/ui"
)

// The bounds of the two stages of an offscreen view for a frame in which only the caret of the
// editor blinks, with a text of any length (step 29.15). The software renderer rasterizes the
// glyphs of the damaged rectangles again, which is about 1.5 MB; the detacher makes a few copies
// of the layers (2 KB). They were 4 MB and 340 MB for 50 and 5,000 lines before.
const (
	detachBound = 32 << 10
	drawBound   = 3 << 20
)

func TestCaretFrameOfAnOffscreenViewAllocatesNothingForTheLengthOfTheText(t *testing.T) {
	tests := []struct {
		name  string
		lines int
	}{{"a short text", 60}, {"a long text", 800}}
	var detached, drawn []uint64
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			on, off := editorscene.CaretScenes(t, tt.lines)
			raw := [2]*ui.Scene{on, off}
			d := offscreen.NewDetacher()
			i := 0
			detach := func() { d.Detach(raw[i%2]); i++ }
			for range 4 {
				detach()
			}
			det := editorscene.BytesPerFrame(30, detach)

			dd := offscreen.NewDetacher()
			scenes := [2]*ui.Scene{dd.Detach(on), dd.Detach(off)}
			s := offscreen.NewSurface(editorscene.Width, editorscene.Height)
			j := 0
			draw := func() {
				if _, err := s.Draw(scenes[j%2]); err != nil {
					t.Fatal(err)
				}
				j++
			}
			for range 4 {
				draw()
			}
			dr := editorscene.BytesPerFrame(30, draw)
			t.Logf("detach %d bytes, draw %d bytes", det, dr)
			if det > detachBound {
				t.Errorf("Detach allocates %d bytes for a caret frame, want at most %d", det, detachBound)
			}
			if dr > drawBound {
				t.Errorf("Draw allocates %d bytes for a caret frame, want at most %d", dr, drawBound)
			}
			detached, drawn = append(detached, det), append(drawn, dr)
		})
	}
	if len(drawn) == 2 && drawn[1] > drawn[0]+drawn[0]/2+64<<10 {
		t.Errorf("Draw: %d bytes for the long text and %d for the short one: the bytes grow with the text", drawn[1], drawn[0])
	}
	if len(detached) == 2 && detached[1] > detached[0]+detached[0]/2+4<<10 {
		t.Errorf("Detach: %d bytes for the long text and %d for the short one: the bytes grow with the text", detached[1], detached[0])
	}
}
