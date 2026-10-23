package webgpu

import (
	"bytes"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/ui"
)

func TestRendererWithoutAProviderFailsToDraw(t *testing.T) {
	r := NewRenderer((*Provider)(nil).Open)
	defer r.Close()
	_, err := r.Draw(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: 1}, 8, 8, nil, false)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != TagUnavailable || !appErr.HasRoute("Open") {
		t.Fatalf("Draw = %v, want WebGPUUnavailable via Open", err)
	}
}

// TestSnapshotIsDrawnBySoft: a WebGPU read-back finishes in a later task, so the snapshot of
// Picture.ToImage is the software renderer's (SPEC §9.3, §12.10): the same pixels, over
// transparent, premultiplied.
func TestSnapshotIsDrawnBySoft(t *testing.T) {
	scene := gputest.CanvasScene(func(c *ui.Canvas) {
		c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(4, 4, 40, 24), 6, 6), gputest.FillPaint(0x80FF0000))
	})
	r := NewRenderer((*Provider)(nil).Open)
	defer r.Close()
	size := ui.Size{Width: 48, Height: 32}
	got, err := r.Snapshot(scene, size)
	if err != nil {
		t.Fatal(err)
	}
	want, err := soft.Render(scene, 48, 32)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Pix, want.Pix) || got.Bounds() != want.Bounds() {
		t.Error("the snapshot is not what soft draws")
	}
	img, err := r.RasterizeScene(scene, 48, 32)
	if err != nil || img.Width() != 48 || img.Height() != 32 {
		t.Fatalf("RasterizeScene = %v, %v", img, err)
	}
	for _, bad := range []ui.Size{{}, {Width: -1, Height: 4}, {Width: 4}} {
		_, err := r.Snapshot(scene, bad)
		if appErr, ok := errors.FromError(err); !ok || appErr.Tag != tagRendererSizeInvalid || !appErr.HasRoute("Renderer.Snapshot.RendererSizeInvalid") {
			t.Errorf("Snapshot(%v) = %v, want RendererSizeInvalid via Renderer.Snapshot", bad, err)
		}
	}
}
