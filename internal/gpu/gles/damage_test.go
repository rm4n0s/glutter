package gles

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func damagePicture(rect ui.Rect) *ui.Picture {
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(rect, solidPaint(0xFF336699))
	return rec.EndRecording()
}

func damageScene(pics ...*ui.Picture) *ui.Scene {
	b := ui.NewSceneBuilder()
	for _, p := range pics {
		b.AddPicture(ui.Offset{}, p)
	}
	return b.Build()
}

func TestDamageIsToldToTheCompositor(t *testing.T) {
	size := ui.Size{Width: 200, Height: 150}
	still := damagePicture(ui.NewRectFromLTWH(0, 0, 50, 50))
	moving := damagePicture(ui.NewRectFromLTWH(100, 20, 30, 30))
	moved := damagePicture(ui.NewRectFromLTWH(100, 20, 30, 30)) // recorded again

	tests := []struct {
		name string
		opts RendererOpts
		// support says whether the display has the extension.
		support bool
		// invalidate calls Renderer.Invalidate before the last frame.
		invalidate bool
		failSwap   bool
		scenes     []*ui.Scene
		// swaps are the calls of eglSwapBuffers(WithDamage) in order; a rectangle list is
		// the damage in EGL's coordinates (from the bottom left).
		swaps  []string
		rects  [][]int32
		pixels []int // DamagedPixels of each frame
	}{
		{
			name: "the first frame is whole, a frame that changes nothing damages a pixel, a changed picture its tile",
			opts: RendererOpts{Damage: true}, support: true,
			scenes: []*ui.Scene{damageScene(still, moving), damageScene(still, moving), damageScene(still, moved)},
			swaps:  []string{"swapBuffers", "swapBuffersWithDamage", "swapBuffersWithDamage"},
			// The tile of the picture is x 64..128 (the picture reaches pixel 131 with its
			// anti-aliasing margin: tiles 1 and 2) and y 0..64, which is 150-64 = 86 from
			// the bottom.
			rects:  [][]int32{{0, 149, 1, 1}, {64, 86, 128, 64}},
			pixels: []int{200 * 150, 1, 128 * 64},
		},
		{
			name: "no extension, no damage", opts: RendererOpts{Damage: true}, support: false,
			scenes: []*ui.Scene{damageScene(still), damageScene(still)},
			swaps:  []string{"swapBuffers", "swapBuffers"}, pixels: []int{0, 0},
		},
		{
			name: "no option, no damage", opts: RendererOpts{}, support: true,
			scenes: []*ui.Scene{damageScene(still), damageScene(still)},
			swaps:  []string{"swapBuffers", "swapBuffers"}, pixels: []int{0, 0},
		},
		{
			name: "the same scene again is presented whole", opts: RendererOpts{Damage: true}, support: true,
			scenes: nil, // filled below
		},
		{
			name: "an invalidated surface is presented whole", opts: RendererOpts{Damage: true}, support: true, invalidate: true,
			scenes: []*ui.Scene{damageScene(still), damageScene(still)},
			swaps:  []string{"swapBuffers", "swapBuffers"}, pixels: []int{200 * 150, 200 * 150},
		},
		{
			name: "a failed swap makes the next frame whole", opts: RendererOpts{Damage: true}, support: true, failSwap: true,
			scenes: []*ui.Scene{damageScene(still), damageScene(still), damageScene(still)},
			swaps:  []string{"swapBuffers", "swapBuffers", "swapBuffers"}, pixels: []int{200 * 150, 200 * 150, 200 * 150},
		},
	}
	same := damageScene(still)
	tests[3].scenes = []*ui.Scene{same, same}
	tests[3].swaps = []string{"swapBuffers", "swapBuffers"}
	tests[3].pixels = []int{200 * 150, 200 * 150}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, egl, _ := newTestRenderer(tt.opts)
			egl.damageOK = tt.support
			s, err := r.Attach(waylandHandle)
			if err != nil {
				t.Fatal(err)
			}
			var swaps []string
			var pixels []int
			for i, scene := range tt.scenes {
				egl.calls = nil
				if tt.invalidate && i == len(tt.scenes)-1 {
					r.Invalidate(s)
				}
				egl.swapFails = tt.failSwap && i < len(tt.scenes)-1
				err := r.Render(s, scene, size, 1)
				if egl.swapFails != (err != nil) {
					t.Fatalf("frame %d: err = %v, swap fails = %v", i, err, egl.swapFails)
				}
				for _, c := range egl.calls {
					if c == "swapBuffers" || c == "swapBuffersWithDamage" {
						swaps = append(swaps, c)
					}
				}
				pixels = append(pixels, r.LastFrame().DamagedPixels)
			}
			if !equalStrings(swaps, tt.swaps) {
				t.Errorf("swaps = %v, want %v", swaps, tt.swaps)
			}
			if tt.pixels != nil && !equalInts(pixels, tt.pixels) {
				t.Errorf("damaged pixels = %v, want %v", pixels, tt.pixels)
			}
			if tt.rects != nil {
				if len(egl.damageRects) != len(tt.rects) {
					t.Fatalf("damage rects = %v, want %v", egl.damageRects, tt.rects)
				}
				for i := range tt.rects {
					if !equalInt32s(egl.damageRects[i], tt.rects[i]) {
						t.Errorf("damage %d = %v, want %v", i, egl.damageRects[i], tt.rects[i])
					}
				}
			}
		})
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalInt32s(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestAFrameCompilesOnlyTheProgramsItUses checks that opening the GL context compiles no
// shader and that a frame of rectangles compiles the ones that draw them, not every
// program of the renderer: a cold compile is several milliseconds on a software driver.
func TestAFrameCompilesOnlyTheProgramsItUses(t *testing.T) {
	r, _, _ := newTestRenderer()
	s, err := r.Attach(waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	names, err := ShaderNames()
	if err != nil {
		t.Fatal(err)
	}
	if got := r.ctx.dev.Stats().Programs; got != 0 {
		t.Fatalf("opening the context compiled %d programs", got)
	}
	scene := damageScene(damagePicture(ui.NewRectFromLTWH(10, 10, 30, 30)))
	if err := r.Render(s, scene, ui.Size{Width: 64, Height: 64}, 1); err != nil {
		t.Fatal(err)
	}
	if got := r.ctx.dev.Stats().Programs; got == 0 || got >= len(names) {
		t.Errorf("a frame of one rectangle compiled %d of %d programs", got, len(names))
	}
}
