package cupertino_test

import (
	"testing"

	"github.com/rm4n0s/glutter/cupertino"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// TestCupertinoThumbPainter checks what the painter draws: the shadows, the border and the thumb, as circles of the
// shortest side.
func TestCupertinoThumbPainter(t *testing.T) {
	rect := ui.NewRectFromCircle(ui.Offset{Dx: 20, Dy: 20}, cupertino.CupertinoThumbPainterRadius)
	for _, tt := range []struct {
		name    string
		painter cupertino.CupertinoThumbPainter
		shadows int
	}{
		{"slider", cupertino.NewCupertinoThumbPainter(), 3},
		{"switch", cupertino.NewCupertinoThumbPainterSwitchThumb(), 2},
		{"colored", cupertino.NewCupertinoThumbPainter(cupertino.CupertinoThumbPainterOpts{Color: foundation.Some(ui.NewColor(0xFF00FF00))}), 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := ui.NewPictureRecorder()
			canvas := ui.NewCanvas(recorder)
			tt.painter.Paint(canvas, rect)
			ops := recorder.EndRecording().DisplayList().Ops()
			if got, want := len(ops), tt.shadows+2; got != want {
				t.Fatalf("%d ops, want %d (%d shadows, the border and the thumb)", got, want, tt.shadows)
			}
			last, ok := ops[len(ops)-1].(ui.OpDrawRRect)
			if !ok {
				t.Fatalf("the last op is %T", ops[len(ops)-1])
			}
			if last.Paint.Color() != tt.painter.Color() {
				t.Errorf("thumb color = %v, want %v", last.Paint.Color(), tt.painter.Color())
			}
			if got, want := last.RRect.OuterRect(), rect; got != want {
				t.Errorf("thumb rect = %v, want %v", got, want)
			}
			if got := last.RRect.TlRadius().X; got != cupertino.CupertinoThumbPainterRadius {
				t.Errorf("thumb radius = %v, want %v", got, cupertino.CupertinoThumbPainterRadius)
			}
		})
	}
}
