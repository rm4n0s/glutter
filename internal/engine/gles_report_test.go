//go:build linux || js

package engine

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
)

// TestGLESReportsWhatAFrameLeftOut checks the lines that the OpenGL ES renderer of the engine
// logs for the frames that left something out: a full coverage atlas once, however many frames
// have one and whatever number of shapes they leave out, and what the renderer cannot draw yet
// when the number of commands changes.
func TestGLESReportsWhatAFrameLeftOut(t *testing.T) {
	full := []string{gpu.FeatureAtlasFull}
	both := []string{"operation ui.OpDrawThing", gpu.FeatureAtlasFull}
	tests := []struct {
		name   string
		frames []gles.FrameStats
		atlas  int // the lines about the atlas
		cannot int // the lines about what the renderer cannot draw yet
	}{
		{"nothing left out", []gles.FrameStats{{Commands: 9, Drawn: 9}, {Commands: 9, Drawn: 9}}, 0, 0},
		{"a full atlas in three frames, with other numbers of shapes", []gles.FrameStats{
			{Commands: 9, Drawn: 6, Skipped: 3, AtlasFull: 3, Unsupported: full},
			{Commands: 9, Drawn: 5, Skipped: 4, AtlasFull: 4, Unsupported: full},
			{Commands: 9, Drawn: 9},
			{Commands: 9, Drawn: 8, Skipped: 1, AtlasFull: 2, Unsupported: full},
		}, 1, 0},
		{"shapes left out inside shadows, which count as drawn", []gles.FrameStats{
			{Commands: 9, Drawn: 9, AtlasFull: 5, Unsupported: full},
			{Commands: 9, Drawn: 9, AtlasFull: 5, Unsupported: full},
		}, 1, 0},
		{"a feature the renderer cannot draw", []gles.FrameStats{
			{Commands: 9, Drawn: 8, Skipped: 1, Unsupported: both[:1]},
			{Commands: 9, Drawn: 8, Skipped: 1, Unsupported: both[:1]},
			{Commands: 9, Drawn: 7, Skipped: 2, Unsupported: both[:1]},
		}, 0, 2},
		{"both in a frame", []gles.FrameStats{
			{Commands: 9, Drawn: 7, Skipped: 2, AtlasFull: 1, Unsupported: both},
			{Commands: 9, Drawn: 7, Skipped: 2, AtlasFull: 1, Unsupported: both},
		}, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			r := &glesRenderer{log: slog.New(slog.NewTextHandler(&out, nil))}
			for _, st := range tt.frames {
				r.reportLeftOut(st)
			}
			atlas := strings.Count(out.String(), "the coverage atlas is full")
			cannot := strings.Count(out.String(), "left out what it cannot draw yet")
			if atlas != tt.atlas || cannot != tt.cannot || strings.Count(out.String(), "level=WARN") != tt.atlas+tt.cannot {
				t.Errorf("%d lines about the atlas and %d about what cannot be drawn, want %d and %d:\n%s", atlas, cannot, tt.atlas, tt.cannot, out.String())
			}
		})
	}
}
