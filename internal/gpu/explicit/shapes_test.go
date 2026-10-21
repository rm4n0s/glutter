package explicit_test

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/ui"
)

func solid(argb uint32) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(argb))
	return p
}

// sceneOf records a drawing into a scene.
func sceneOf(fn func(c *ui.Canvas)) *ui.Scene {
	rec := ui.NewPictureRecorder()
	fn(ui.NewCanvas(rec))
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	return sb.Build()
}

// instances decodes the floats of the last write of instance data.
func instances(t *testing.T, dev *fakedevice.Device) [][]float32 {
	t.Helper()
	for i := len(dev.Writes) - 1; i >= 0; i-- {
		w := dev.Writes[i]
		if len(w.Data)%gpu.ShapeStride != 0 || len(w.Data) == 0 || w.Offset != 0 {
			continue
		}
		var out [][]float32
		for off := 0; off < len(w.Data); off += gpu.ShapeStride {
			in := make([]float32, gpu.ShapeFloats)
			for j := range in {
				in[j] = math.Float32frombits(binary.LittleEndian.Uint32(w.Data[off+4*j:]))
			}
			out = append(out, in)
		}
		return out
	}
	t.Fatal("no instance data was written")
	return nil
}

// scenePass returns the calls of the pass that draws the scene in the last frame: the one before
// the blit, after any passes that make mip levels.
func scenePass(dev *fakedevice.Device) []fakedevice.Call {
	all := dev.Calls()
	begin := 0
	for i, c := range all {
		if c.Op == "Begin" {
			begin = i
		}
	}
	var passes [][]fakedevice.Call
	in := false
	for _, c := range all[begin:] {
		switch c.Op {
		case "BeginPass":
			passes = append(passes, nil)
			in = true
		case "EndPass":
			in = false
		default:
			if in {
				passes[len(passes)-1] = append(passes[len(passes)-1], c)
			}
		}
	}
	if len(passes) < 2 {
		return nil
	}
	return passes[len(passes)-2]
}

func TestDrawsShapesAsInstances(t *testing.T) {
	r, dev, _ := newRenderer(t)
	scene := sceneOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(10, 20, 40, 30), solid(0xFFFF8000))
		c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(60, 20, 40, 30), 6, 6), solid(0xFF0080FF))
	})
	if _, err := r.Draw(window, 200, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	got := instances(t, dev)
	if len(got) != 2 {
		t.Fatalf("%d instances, want 2", len(got))
	}
	// The rectangle is an exact box, with a pixel more for the edge on each side.
	box := got[0]
	if box[0] != 9 || box[1] != 19 || box[2] != 51 || box[3] != 51 || box[10] != gpu.ShapeKindBox {
		t.Errorf("rect instance = %v", box[:12])
	}
	if box[12] != 1 || box[13] != float32(0x80)/255 || box[14] != 0 || box[15] != 1 {
		t.Errorf("rect color = %v", box[12:16])
	}
	if got[1][10] != gpu.ShapeKindRRect {
		t.Errorf("rrect kind = %v", got[1][10])
	}
	// The bind groups are made as the draw calls need them: the uniforms, the clip mask (the white
	// texel here) and the textures of the paint (stand-ins here).
	want := []string{"SetPipeline", "NewBindGroup", "SetBindGroup", "NewBindGroup", "SetBindGroup", "NewBindGroup", "SetBindGroup", "SetScissor", "SetVertexBuffer", "Draw"}
	var ops []string
	for _, c := range scenePass(dev) {
		ops = append(ops, c.Op)
	}
	if !reflect.DeepEqual(ops, want) {
		t.Errorf("scene pass ops %v, want %v", ops, want)
	}
	if st := r.LastFrame(); st.Commands != 2 || st.Drawn != 2 || st.Instances != 2 || st.Batches != 1 {
		t.Errorf("stats %+v", st)
	}
	for _, c := range scenePass(dev) {
		if c.Op == "Draw" && !strings.HasPrefix(c.Args, "vertices=6 instances=2 ") {
			t.Errorf("draw %s, want one draw of 2 instances", c.Args)
		}
	}
}

func TestBatchesByBlendAndScissor(t *testing.T) {
	r, dev, _ := newRenderer(t)
	plus := solid(0xFF202020)
	plus.SetBlendMode(ui.BlendModePlus)
	scene := sceneOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000)) // srcOver
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000)) // merged with the first
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), plus)              // another blend: another call
		c.Save()
		c.ClipRect(ui.NewRectFromLTWH(5, 5, 20, 20))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 50, 50), plus) // another scissor: another call
		c.Restore()
	})
	if _, err := r.Draw(window, 200, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	var draws, scissors, pipelines []string
	for _, c := range scenePass(dev) {
		switch c.Op {
		case "Draw":
			draws = append(draws, c.Args[:strings.Index(c.Args, " firstVertex")])
		case "SetScissor":
			scissors = append(scissors, c.Args)
		case "SetPipeline":
			pipelines = append(pipelines, c.Args)
		}
	}
	if want := []string{"vertices=6 instances=2", "vertices=6 instances=1", "vertices=6 instances=1"}; !reflect.DeepEqual(draws, want) {
		t.Errorf("draws %v, want %v", draws, want)
	}
	if want := []string{"(0,0)-(200,100)", "(0,0)-(200,100)", "(5,5)-(25,25)"}; !reflect.DeepEqual(scissors, want) {
		t.Errorf("scissors %v, want %v", scissors, want)
	}
	if len(pipelines) != 2 { // srcOver, then plus for both of the last two
		t.Errorf("pipelines set %d times, want 2: %v", len(pipelines), pipelines)
	}
	var made int
	for _, op := range dev.Ops() {
		if op == "NewPipeline" {
			made++
		}
	}
	if made != 3 { // the blit, srcOver and plus
		t.Errorf("%d pipelines made, want the blit and two for shapes", made)
	}
}

func TestInstanceDataIsUploadedOncePerFrameIntoTheSlot(t *testing.T) {
	r, dev, _ := newRenderer(t)
	scene := sceneOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000)) })
	for i := 0; i < 5; i++ {
		if _, err := r.Draw(window, 64, 64, scene, false); err != nil {
			t.Fatal(err)
		}
	}
	var buffers int
	for _, op := range dev.Ops() {
		if op == "NewBuffer" {
			buffers++
		}
	}
	// One instance buffer and one uniform buffer for each of the two frames in flight.
	if buffers != 2*explicit.FramesInFlight {
		t.Errorf("%d buffers made over 5 frames, want %d", buffers, 2*explicit.FramesInFlight)
	}
	var offsets []string
	for _, c := range dev.Calls() {
		if c.Op == "SetBindGroup" && c.Args == "group=0 offsets=[0 256 512]" {
			offsets = append(offsets, c.Args)
		}
	}
	if len(offsets) != 5 {
		t.Errorf("the frame, draw and paint uniforms are at 0, 256 and 512 in %d of 5 frames", len(offsets))
	}
}

func TestInstanceBufferGrows(t *testing.T) {
	r, dev, _ := newRenderer(t)
	many := sceneOf(func(c *ui.Canvas) {
		for i := 0; i < 2000; i++ { // 2000 * 192 bytes is more than the first 64 KiB
			c.DrawRect(ui.NewRectFromLTWH(float64(i%100), float64(i/100), 5, 5), solid(0xFF000000))
		}
	})
	// Two frames cover both slots.
	for i := 0; i < 2; i++ {
		if _, err := r.Draw(window, 128, 128, many, false); err != nil {
			t.Fatal(err)
		}
	}
	if got := instances(t, dev); len(got) != 2000 {
		t.Errorf("%d instances written, want 2000", len(got))
	}
	for _, c := range dev.Calls() {
		if c.Op == "NewBuffer" && strings.HasPrefix(c.Args, "size=") {
			var size int
			if _, err := fmtSscan(c.Args, &size); err == nil && size < 2000*gpu.ShapeStride && strings.Contains(c.Args, "usage=1") && size < 64<<10 {
				t.Errorf("an instance buffer of %d bytes is too small to start with", size)
			}
		}
	}
}

func TestUnsupportedFeatures(t *testing.T) {
	linear := ui.NewPaint()
	linear.SetShader(ui.NewGradientLinear(ui.Offset{}, ui.Offset{Dx: 10}, []ui.Color{ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF)}))
	multiply := solid(0xFF000000)
	multiply.SetBlendMode(ui.BlendModeMultiply)
	blur := solid(0xFF000000)
	blur.SetMaskFilter(ui.NewMaskFilterBlur(ui.BlurStyleNormal, 3))
	tests := []struct {
		name    string
		feature string
		draw    func(c *ui.Canvas)
	}{
		{"a mask filter", "mask filter", func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), blur) }},
		{"an image filter", "image filter", func(c *ui.Canvas) {
			p := solid(0xFF000000)
			p.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 2}))
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), p)
		}},
		{"an image filter on a layer", "image filter", func(c *ui.Canvas) {
			p := solid(0x80000000)
			p.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2, SigmaY: 2}))
			c.SaveLayer(nil, p)
			c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), solid(0xFF000000))
			c.Restore()
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene := sceneOf(tt.draw)
			// Strictly, the frame fails before it touches the device.
			r, dev, _ := newRenderer(t)
			_, err := r.Draw(window, 100, 100, scene, false)
			appErr, ok := errors.FromError(err)
			if !ok || appErr.Tag != "GPUNotImplemented" || !appErr.HasRoute("Renderer.build->frameBuilder.command") {
				t.Fatalf("got %v, want GPUNotImplemented through Renderer.build", err)
			}
			if got := appErr.Metadata["feature"]; got != tt.feature {
				t.Errorf("feature = %v, want %q", got, tt.feature)
			}
			if slicesContains(dev.Ops(), "Begin") {
				t.Errorf("a frame that fails to build began on the device: %v", dev.Ops())
			}

			// With SkipUnsupported the rest is drawn and the feature is counted once.
			r, _, _ = newRenderer(t, explicit.RendererOpts{SkipUnsupported: true})
			if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
				t.Fatal(err)
			}
			if st := r.LastFrame(); !reflect.DeepEqual(st.Unsupported, []string{tt.feature}) || st.Skipped == 0 {
				t.Errorf("stats %+v, want %q skipped", st, tt.feature)
			}
		})
	}
}

func TestNothingToDraw(t *testing.T) {
	transparent := solid(0xFF000000)
	transparent.SetBlendMode(ui.BlendModeDst)
	scene := sceneOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), transparent)                       // a paint that draws nothing
		c.DrawRect(ui.NewRectFromLTWH(500, 500, 10, 10), solid(0xFF000000))             // outside the surface
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 0, 0), solid(0xFF000000))                   // empty
		c.DrawLine(ui.Offset{Dx: 1, Dy: 1}, ui.Offset{Dx: 1, Dy: 1}, solid(0xFF000000)) // a butt cap on no length
	})
	r, dev, _ := newRenderer(t)
	if _, err := r.Draw(window, 100, 100, scene, false); err != nil {
		t.Fatal(err)
	}
	if st := r.LastFrame(); st.Instances != 0 || st.Batches != 0 || st.Skipped != 0 {
		t.Errorf("stats %+v, want nothing drawn and nothing skipped", st)
	}
	if got := scenePass(dev); len(got) != 0 {
		t.Errorf("the scene pass drew %v", got)
	}
	if slicesContains(dev.Ops(), "WriteBuffer") {
		t.Errorf("an empty frame wrote buffers: %v", dev.Ops())
	}
}

func slicesContains(ops []string, op string) bool {
	for _, o := range ops {
		if o == op {
			return true
		}
	}
	return false
}

func fmtSscan(s string, size *int) (int, error) {
	var usage int
	return fmt.Sscanf(s, "size=%d usage=%d", size, &usage)
}

// testImage is an n x n image of one opaque color.
func testImage(n int) *ui.Image {
	pix := make([]byte, 4*n*n)
	for i := 0; i < len(pix); i += 4 {
		copy(pix[i:], []byte{0x20, 0x40, 0x80, 0xFF})
	}
	return ui.NewImageFromRGBA(pix, n, n)
}
