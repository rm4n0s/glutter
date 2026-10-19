//go:build !glutter_release && !glutter_profile

package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func TestPaintImageDebugPanics(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		call func()
	}{
		{"disposed image", "PaintImageDisposed", func() {
			img := testImage(2, 2)
			img.Dispose()
			PaintImage(ui.NewCanvas(ui.NewPictureRecorder()), lrtb(0, 0, 4, 4), img)
		}},
		{"center slice with cover", "PaintImageCenterSliceFit", func() {
			PaintImage(ui.NewCanvas(ui.NewPictureRecorder()), lrtb(0, 0, 4, 4), testImage(10, 10),
				PaintImageOpts{CenterSlice: foundation.Some(lrtb(4, 4, 6, 6)), Fit: BoxFitCover})
		}},
		{"center slice cropped", "PaintImageCenterSliceCropped", func() {
			PaintImage(ui.NewCanvas(ui.NewPictureRecorder()), lrtb(0, 0, 40, 4), testImage(10, 10),
				PaintImageOpts{CenterSlice: foundation.Some(lrtb(4, 4, 6, 6)), Fit: BoxFitFitWidth})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { requireTaggedPanic(t, tt.tag, tt.call) })
	}
}
