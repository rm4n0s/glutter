package explicit_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/ui"
)

// opsOfLastFrame returns the ops of the device since the last Begin.
func opsOfLastFrame(dev *fakedevice.Device) []fakedevice.Call {
	all := dev.Calls()
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].Op == "Begin" {
			return all[i:]
		}
	}
	return all
}

// paintFloats decodes the Paint uniform of the call: the last write into a uniform buffer holds,
// from the offset 256, the frame uniform's successors; the paint is found by its offset.
func paintFloats(t *testing.T, dev *fakedevice.Device, offset int) []float32 {
	t.Helper()
	for i := len(dev.Writes) - 1; i >= 0; i-- {
		w := dev.Writes[i]
		if w.Usage&explicit.BufferUniform == 0 {
			continue
		}
		if len(w.Data) < offset+55*16 {
			t.Fatalf("the ring has %d bytes, want a paint at %d", len(w.Data), offset)
		}
		return floatsOf(w.Data[offset : offset+55*16])
	}
	t.Fatal("no uniform data was written")
	return nil
}

func TestGradientsSetThePaintUniform(t *testing.T) {
	r, dev, _ := newRenderer(t)
	g := ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 100}, []ui.Color{ui.NewColor(0xFFFF0000), ui.NewColor(0xFF0000FF)})
	p := ui.NewPaint()
	p.SetShader(g)
	scene := sceneOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 100, 50), p) })
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.Skipped != 0 || st.Instances != 1 {
		t.Fatalf("stats %+v", st)
	}
	// The ring: the frame uniform at 0, the draw uniform at 256, the paint at 512.
	f := paintFloats(t, dev, 512)
	if f[0] != 1 || f[3] != 2 { // linear, two stops
		t.Errorf("shader %v", f[0:4])
	}
	if f[4] != 1 || f[7] != 1 { // the inverse of the identity
		t.Errorf("xf0 %v", f[4:8])
	}
	if f[12] != 0 || f[14] != 100 || f[16] < 0.0000999 || f[16] > 0.0001001 { // from (0,0), direction (100,0), 1/10000
		t.Errorf("p0 %v p1 %v", f[12:16], f[16:20])
	}
	if f[28] != 1 { // dithered
		t.Errorf("flags %v", f[28:32])
	}
	if f[32] != -1 { // the blend unit blends
		t.Errorf("blend %v", f[32:36])
	}
	// The first stop is red, the last blue, at 0 and 1.
	colors := f[15*4:]
	if colors[0] != 1 || colors[2] != 0 || colors[4+2] != 1 {
		t.Errorf("ramp colors %v", colors[:8])
	}
	pos := f[47*4:]
	if pos[0] != 0 || pos[1] != 1 {
		t.Errorf("ramp positions %v", pos[:4])
	}
}

func TestPaintsOfOneSpecShareAnEntryOfTheRing(t *testing.T) {
	r, dev, _ := newRenderer(t)
	scene := sceneOf(func(c *ui.Canvas) {
		for i := 0; i < 5; i++ {
			c.DrawRect(ui.NewRectFromLTWH(float64(i*10), 0, 5, 5), solid(0xFF000000))
			c.DrawPath(star(), solid(0xFF000000)) // another pipeline, the same paint spec
		}
	})
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, c := range scenePass(dev) {
		if c.Op == "SetBindGroup" && strings.HasPrefix(c.Args, "group=0") {
			fields := strings.Fields(c.Args)
			seen[strings.Trim(fields[len(fields)-1], "]")]++
		}
	}
	if len(seen) != 1 {
		t.Errorf("the paint entries of the calls: %v, want one for all", seen)
	}
}

func TestBlendsThatReadTheDestinationCopyIt(t *testing.T) {
	r, dev, _ := newRenderer(t)
	multiply := solid(0xFF3366CC)
	multiply.SetBlendMode(ui.BlendModeMultiply)
	scene := sceneOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 100, 100), solid(0xFFFFFFFF))
		c.DrawRect(ui.NewRectFromLTWH(10, 20, 30, 40), multiply)
		c.DrawRect(ui.NewRectFromLTWH(50, 20, 30, 40), multiply)
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000))
	})
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.Skipped != 0 || st.Batches != 4 {
		t.Errorf("stats %+v: the two reading draws are calls of their own", st)
	}
	// Each reading draw copies the pixels under it, between two passes: the scene is drawn in
	// five passes (before the first, between, after) rather than one, and the blit is the last.
	var copies, passes []string
	for _, c := range opsOfLastFrame(dev) {
		switch c.Op {
		case "CopyTexture":
			copies = append(copies, c.Args)
		case "BeginPass":
			passes = append(passes, c.Args)
		}
	}
	if want := []string{"rect=(9,19)-(41,61) at=(0,0)", "rect=(49,19)-(81,61) at=(0,0)"}; !reflect.DeepEqual(copies, want) {
		t.Errorf("copies %v, want %v", copies, want)
	}
	if len(passes) != 1+2+1 { // the first, one after each copy, the blit
		t.Errorf("%d passes %v", len(passes), passes)
	}
	if !strings.Contains(passes[1], "load=1") { // LoadLoad keeps what is drawn
		t.Errorf("the pass after a copy does not keep the frame: %s", passes[1])
	}
	// The destination texture is of the size of the biggest copy, rounded up to the granule.
	var dst []string
	for _, a := range calls(dev, "NewTexture") {
		if strings.HasPrefix(a, "64x64 format=1 levels=1 usage=1") {
			dst = append(dst, a)
		}
	}
	if len(dst) != 1 {
		t.Errorf("destination textures %v", calls(dev, "NewTexture"))
	}
}

func TestLayersAreDrawnIntoTexturesAndComposited(t *testing.T) {
	r, dev, _ := newRenderer(t)
	scene := sceneOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 100, 100), solid(0xFFFFFFFF))
		c.SaveLayer(&ui.Rect{Left: 10, Top: 10, Right: 70, Bottom: 50}, solid(0x80000000))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 100, 100), solid(0xFFFF0000))
		c.Restore()
		c.DrawRect(ui.NewRectFromLTWH(80, 80, 10, 10), solid(0xFF000000))
	})
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	var layers []string
	for _, a := range calls(dev, "NewTexture") {
		if strings.HasPrefix(a, "64x64 format=1 levels=1 usage=7") { // the layer, rounded to the granule
			layers = append(layers, a)
		}
	}
	if len(layers) != 1 {
		t.Fatalf("layer textures %v", calls(dev, "NewTexture"))
	}
	// The passes: the frame (until the layer begins), the layer, the frame again (the composite and
	// the last rect), and the blit.
	var passes, pipelines []string
	for _, c := range opsOfLastFrame(dev) {
		switch c.Op {
		case "BeginPass":
			passes = append(passes, c.Args)
		case "SetPipeline":
			pipelines = append(pipelines, strings.TrimPrefix(c.Args, "shader="))
		}
	}
	if len(passes) != 4 {
		t.Fatalf("%d passes %v, want 4", len(passes), passes)
	}
	if !strings.Contains(passes[1], "load=0") || !strings.Contains(passes[2], "load=1") {
		t.Errorf("the layer is cleared and its parent kept: %v", passes)
	}
	if want := []string{"shape", "shape", "layer", "shape", "blit"}; !reflect.DeepEqual(pipelines, want) {
		t.Errorf("pipelines %v, want %v", pipelines, want)
	}
	if st := r.LastFrame(); st.Skipped != 0 || st.Instances != 3 || st.Batches != 5 {
		t.Errorf("stats %+v", st)
	}
	// The layer covers its bounds: the scissor of what is drawn in it is that rectangle.
	var scissors []string
	for _, c := range opsOfLastFrame(dev) {
		if c.Op == "SetScissor" {
			scissors = append(scissors, c.Args)
		}
	}
	if len(scissors) < 3 || scissors[1] != "(0,0)-(60,40)" {
		t.Errorf("scissors %v: the layer's content is cut to its rectangle, from its own corner", scissors)
	}
	// The frame uniform of the layer holds its rectangle's origin and the texture's size.
	found := false
	for i := len(dev.Writes) - 1; i >= 0 && !found; i-- {
		w := dev.Writes[i]
		if w.Usage&explicit.BufferUniform == 0 {
			continue
		}
		for off := 0; off+16 <= len(w.Data); off += 256 {
			if reflect.DeepEqual(floatsOf(w.Data[off:off+16]), []float32{64, 64, 10, 10}) {
				found = true
			}
		}
	}
	if !found {
		t.Error("no frame uniform of a 64 x 64 texture at (10, 10) for the layer")
	}
	// The texture goes back to the pool: the next frame makes none.
	before := len(calls(dev, "NewTexture"))
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if got := len(calls(dev, "NewTexture")) - before; got != 0 {
		t.Errorf("the second frame made %d textures", got)
	}
}

func TestNestedLayersAndCulling(t *testing.T) {
	r, dev, _ := newRenderer(t)
	scene := sceneOf(func(c *ui.Canvas) {
		c.SaveLayer(nil, solid(0xCC000000))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 50, 50), solid(0xFFFF0000))
		c.SaveLayer(&ui.Rect{Left: 200, Top: 200, Right: 300, Bottom: 300}, solid(0x80000000)) // off the surface
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 50, 50), solid(0xFF00FF00))
		c.Restore()
		c.SaveLayer(&ui.Rect{Left: 0, Top: 0, Right: 30, Bottom: 30}, solid(0x80000000))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 50, 50), solid(0xFF0000FF))
		c.Restore()
		c.Restore()
	})
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	// Two layers made it: the outer and the one of 30 x 30; the empty one is culled whole.
	var begins int
	for _, c := range opsOfLastFrame(dev) {
		if c.Op == "BeginPass" {
			begins++
		}
	}
	// Frame, outer layer, inner layer, the outer again after the inner composite, the frame after the
	// outer composite, and the blit.
	if begins != 6 {
		t.Errorf("%d passes, want 6", begins)
	}
	if st := r.LastFrame(); st.Skipped != 0 || st.Commands != 3 || st.Drawn != 3 {
		t.Errorf("stats %+v", st)
	}
}

func TestLayerPaintsAndLayersOfOtherKinds(t *testing.T) {
	blur := solid(0x80000000)
	plus := solid(0xFF202020)
	plus.SetBlendMode(ui.BlendModePlus)
	multiply := solid(0xFFFFFFFF)
	multiply.SetBlendMode(ui.BlendModeMultiply)
	tests := []struct {
		name  string
		draw  func(c *ui.Canvas)
		skips int
	}{
		{"a layer composited with plus", func(c *ui.Canvas) {
			c.SaveLayer(nil, plus)
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000))
			c.Restore()
		}, 0},
		{"a layer composited by a mode that reads the destination", func(c *ui.Canvas) {
			c.SaveLayer(nil, multiply)
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000))
			c.Restore()
		}, 0},
		{"a layer with a color filter", func(c *ui.Canvas) {
			p := blur
			p.SetColorFilter(ui.NewColorFilterMode(ui.NewColor(0xFFFF0000), ui.BlendModeSrcIn))
			c.SaveLayer(nil, p)
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000))
			c.Restore()
		}, 0},
		{"a layer that draws nothing", func(c *ui.Canvas) {
			c.SaveLayer(nil, solid(0x80000000))
			c.Restore()
		}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, dev, _ := newRenderer(t)
			if _, err := r.Draw(window, 100, 100, sceneOf(tt.draw), false); err != nil {
				t.Fatal(err)
			}
			if st := r.LastFrame(); st.Skipped != tt.skips {
				t.Errorf("stats %+v", st)
			}
			if got := opsOfLastFrame(dev); got[len(got)-1].Op != "Present" && got[len(got)-1].Op != "Submit" {
				t.Errorf("the frame did not end: %v", got[len(got)-1])
			}
		})
	}
}

func TestRectOfACopyIsFromTheTargetsCorner(t *testing.T) {
	// A draw that reads the destination inside a layer copies from the layer's texture, from the
	// corner of the layer, not of the surface.
	r, dev, _ := newRenderer(t)
	multiply := solid(0xFF3366CC)
	multiply.SetBlendMode(ui.BlendModeMultiply)
	scene := sceneOf(func(c *ui.Canvas) {
		c.SaveLayer(&ui.Rect{Left: 30, Top: 30, Right: 90, Bottom: 90}, solid(0xFF000000))
		c.DrawRect(ui.NewRectFromLTWH(40, 40, 20, 20), multiply)
		c.Restore()
	})
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	copies := calls(dev, "CopyTexture")
	if want := []string{"rect=(9,9)-(31,31) at=(0,0)"}; !reflect.DeepEqual(copies, want) {
		t.Errorf("copies %v, want %v: the rectangle of the layer, from its corner", copies, want)
	}
}
