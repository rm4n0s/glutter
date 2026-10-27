package offscreen_test

import (
	"fmt"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/gputest/editorscene"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/ui"
)

// The benchmarks of SPEC §9.9: a whole 1280x800 frame through the software renderer, and
// the frame after a change through a Surface. Run them with
//
//	go test ./offscreen -run xxx -bench Frame -benchmem
//
// The targets of step 29.6: no text under 10 ms, text under 25 ms, a caret under 2 ms.

func BenchmarkFrame(b *testing.B) {
	for _, c := range []struct {
		name string
		body func() *ui.Scene
	}{
		{"NoText", func() *ui.Scene { _, s := pumpFrame(b, scaffoldApp(plainBody())); return s }},
		{"Text", func() *ui.Scene { _, s := pumpFrame(b, scaffoldApp(textBody())); return s }},
	} {
		scene := c.body()
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := soft.Render(scene, frameW, frameH); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(c.name+"Surface", func(b *testing.B) {
			s := offscreen.NewSurface(frameW, frameH)
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				s.Resize(frameW, frameH)
				if _, err := s.Draw(scene); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFrameCaret is the time of Surface.Draw when only the caret of a text field
// changed between two frames of a screen full of text.
func BenchmarkFrameCaret(b *testing.B) {
	a, c := caretFrames(b)
	d := offscreen.NewDetacher()
	scenes := [2]*ui.Scene{d.Detach(a), d.Detach(c)}
	s := offscreen.NewSurface(frameW, frameH)
	if _, err := s.Draw(scenes[0]); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		damage, err := s.Draw(scenes[(i+1)%2])
		if err != nil {
			b.Fatal(err)
		}
		if len(damage) == 0 {
			b.Fatal("the caret frame has no damage")
		}
	}
}

// BenchmarkEditorCaretFrame is the two stages of an offscreen view for a frame in which only the
// caret of the text editor of examples/editor blinks, with a text of 50, 500 and 5,000 lines (step
// 29.15): Detacher.Detach (on the UI thread) and Surface.Draw (on the drawing goroutine). The bytes
// and allocations of a frame must not grow with the length of the text.
func BenchmarkEditorCaretFrame(b *testing.B) {
	for _, lines := range []int{50, 500, 5000} {
		var raw [2]*ui.Scene // made once: b.Run calls the function again for each b.N
		scenes := func(b *testing.B) [2]*ui.Scene {
			if raw[0] == nil {
				raw[0], raw[1] = editorscene.CaretScenes(b, lines)
			}
			return raw
		}
		b.Run(fmt.Sprintf("detach/lines=%d", lines), func(b *testing.B) {
			raw := scenes(b)
			d := offscreen.NewDetacher()
			for i := 0; i < 4; i++ {
				d.Detach(raw[i%2])
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				d.Detach(raw[i%2])
			}
		})
		b.Run(fmt.Sprintf("draw/lines=%d", lines), func(b *testing.B) {
			raw := scenes(b)
			d := offscreen.NewDetacher()
			detached := [2]*ui.Scene{d.Detach(raw[0]), d.Detach(raw[1])}
			s := offscreen.NewSurface(editorscene.Width, editorscene.Height)
			for i := 0; i < 4; i++ {
				if _, err := s.Draw(detached[i%2]); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := s.Draw(detached[i%2]); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
