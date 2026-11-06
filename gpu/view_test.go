package gpu_test

import (
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// viewRig is a tester with a GPU, a view in the middle of the window, and what the view's functions saw.
type viewRig struct {
	t        *testing.T
	ts       *gluttertest.Tester
	rec      *gluttertest.GPURecorder
	frames   []gpu.ViewFrame
	inits    []*gpu.Device
	disposed int
	// scenesAt is how many scenes had been delivered when each frame was drawn.
	scenesAt []int
}

func newViewRig(t *testing.T, dpr float64) *viewRig {
	t.Helper()
	ts := gluttertest.New(t, gluttertest.Opts{GPU: true, DevicePixelRatio: dpr})
	return &viewRig{t: t, ts: ts, rec: ts.GPUDevice()}
}

func (r *viewRig) paint(f *gpu.ViewFrame) {
	r.frames = append(r.frames, *f)
	r.scenesAt = append(r.scenesAt, len(r.ts.Scenes()))
	attachment := gpu.RenderPassColorAttachment{View: f.Color.View(), ClearValue: gpu.Color{A: 1}}
	if f.Resolve != nil {
		attachment.ResolveTarget = f.Resolve.View()
	}
	pass := f.Encoder.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{attachment}})
	pass.End()
}

func (r *viewRig) opts(o gpu.ViewOpts) gpu.ViewOpts {
	o.OnInit = func(d *gpu.Device) { r.inits = append(r.inits, d) }
	o.OnDispose = func() { r.disposed++ }
	return o
}

// sized puts a view in a box of the size in the middle of the window.
func sized(w, h float64, child Widget) Widget {
	return NewCenter(NewSizedBox(child, SizedBoxOpts{Width: foundation.Some(w), Height: foundation.Some(h)}))
}

// textureLayers returns the texture layers of the last scene.
func (r *viewRig) textureLayers() []*ui.TextureLayer {
	var out []*ui.TextureLayer
	scenes := r.ts.Scenes()
	if len(scenes) == 0 {
		return nil
	}
	var walk func(l ui.Layer)
	walk = func(l ui.Layer) {
		if tl, ok := l.(*ui.TextureLayer); ok {
			out = append(out, tl)
		}
		for _, c := range l.Children() {
			walk(c)
		}
	}
	walk(scenes[len(scenes)-1].Root())
	return out
}

func TestAViewDrawsIntoTargetsOfItsSize(t *testing.T) {
	tests := []struct {
		name         string
		dpr          float64
		opts         gpu.ViewOpts
		w, h         float64
		wantW, wantH int
		resolve      bool
		depth        bool
		format       gpu.TextureFormat
		textures     int
	}{
		{"the default view", 1, gpu.ViewOpts{}, 100, 50, 100, 50, false, false, gpu.TextureFormatRGBA8Unorm, 1},
		{"a device pixel ratio of 2", 2, gpu.ViewOpts{}, 100, 50, 200, 100, false, false, gpu.TextureFormatRGBA8Unorm, 1},
		{"a ratio that does not divide", 1.5, gpu.ViewOpts{}, 101, 51, 152, 77, false, false, gpu.TextureFormatRGBA8Unorm, 1},
		{"depth", 1, gpu.ViewOpts{DepthFormat: gpu.TextureFormatDepth32Float}, 64, 64, 64, 64, false, true, gpu.TextureFormatRGBA8Unorm, 2},
		{"four samples", 1, gpu.ViewOpts{SampleCount: 4}, 64, 64, 64, 64, true, false, gpu.TextureFormatRGBA8Unorm, 2},
		{"four samples with depth", 2, gpu.ViewOpts{SampleCount: 4, DepthFormat: gpu.TextureFormatDepth24PlusStencil8}, 32, 32, 64, 64, true, true, gpu.TextureFormatRGBA8Unorm, 3},
		{"half float", 1, gpu.ViewOpts{Format: gpu.TextureFormatRGBA16Float}, 40, 40, 40, 40, false, false, gpu.TextureFormatRGBA16Float, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newViewRig(t, tt.dpr)
			r.ts.PumpWidget(sized(tt.w, tt.h, gpu.NewView(r.paint, r.opts(tt.opts))))
			if len(r.frames) != 1 {
				t.Fatalf("the paint function ran %d times, want once", len(r.frames))
			}
			f := r.frames[0]
			if f.Width != tt.wantW || f.Height != tt.wantH || f.DevicePixelRatio != tt.dpr {
				t.Errorf("frame = %dx%d at %v, want %dx%d at %v", f.Width, f.Height, f.DevicePixelRatio, tt.wantW, tt.wantH, tt.dpr)
			}
			if f.Device == nil || f.Encoder == nil || f.Color == nil {
				t.Fatalf("frame = %+v lacks the device, the encoder or the color target", f)
			}
			if (f.Resolve != nil) != tt.resolve || (f.Depth != nil) != tt.depth {
				t.Errorf("resolve %v, depth %v; want %v, %v", f.Resolve != nil, f.Depth != nil, tt.resolve, tt.depth)
			}
			if f.Color.Format() != tt.format || f.Color.Size().Width != tt.wantW || f.Color.Size().Height != tt.wantH {
				t.Errorf("color = %v %v, want %v %dx%d", f.Color.Format(), f.Color.Size(), tt.format, tt.wantW, tt.wantH)
			}
			wantSamples := max(tt.opts.SampleCount, 1)
			if f.Color.SampleCount() != wantSamples || (f.Depth != nil && f.Depth.SampleCount() != wantSamples) {
				t.Errorf("sample counts = %d, %v; want %d", f.Color.SampleCount(), f.Depth, wantSamples)
			}
			if f.Resolve != nil && (f.Resolve.SampleCount() != 1 || f.Resolve.Format() != tt.format) {
				t.Errorf("resolve = %v with %d samples", f.Resolve.Format(), f.Resolve.SampleCount())
			}
			if got := r.rec.Device().Live().Textures; got != tt.textures {
				t.Errorf("%d textures alive, want %d", got, tt.textures)
			}
			if len(r.inits) != 1 || r.inits[0] != f.Device {
				t.Errorf("OnInit ran %d times", len(r.inits))
			}
			layers := r.textureLayers()
			if len(layers) != 1 || layers[0].Width != tt.w || layers[0].Height != tt.h {
				t.Fatalf("texture layers = %+v, want one of %vx%v", layers, tt.w, tt.h)
			}
			// The window shows the resolved texture, and the submit happened.
			shown := f.Color
			if f.Resolve != nil {
				shown = f.Resolve
			}
			if shown.Size().Width != tt.wantW {
				t.Errorf("the shown texture is %v", shown.Size())
			}
			if n := countOps(r.rec.Ops(), "Submit"); n == 0 {
				t.Errorf("the view did not submit its encoder: %v", r.rec.Ops())
			}
		})
	}
}

func countOps(ops []string, name string) int {
	n := 0
	for _, o := range ops {
		if o == name {
			n++
		}
	}
	return n
}

func TestAViewDrawsBeforeTheSceneIsComposited(t *testing.T) {
	r := newViewRig(t, 1)
	r.ts.PumpWidget(sized(50, 50, gpu.NewView(r.paint, r.opts(gpu.ViewOpts{}))))
	before := r.scenesAt[0]
	after := len(r.ts.Scenes())
	if after != before+1 {
		t.Errorf("the paint function ran with %d scenes delivered and the frame delivered %d: it must run before the frame's scene", before, after)
	}
}

func TestAnIdleViewDrawsNothing(t *testing.T) {
	r := newViewRig(t, 1)
	r.ts.PumpWidget(sized(50, 50, gpu.NewView(r.paint, r.opts(gpu.ViewOpts{}))))
	if r.ts.Binding.Scheduler().HasScheduledFrame() {
		t.Fatalf("an idle view scheduled a frame")
	}
	r.rec.Reset()
	r.ts.Pump(16 * time.Millisecond)
	r.ts.Pump(16 * time.Millisecond)
	if len(r.frames) != 1 || len(r.rec.Ops()) != 0 {
		t.Errorf("the idle view drew %d times and the device got %v", len(r.frames), r.rec.Ops())
	}
}

func TestTheControllerMarksTheViewForOneDraw(t *testing.T) {
	r := newViewRig(t, 1)
	c := gpu.NewViewController()
	r.ts.PumpWidget(sized(50, 50, gpu.NewView(r.paint, r.opts(gpu.ViewOpts{Controller: c}))))
	c.MarkNeedsPaint()
	if !r.ts.Binding.Scheduler().HasScheduledFrame() {
		t.Fatalf("MarkNeedsPaint scheduled no frame")
	}
	r.ts.Pump(16 * time.Millisecond)
	if len(r.frames) != 2 {
		t.Fatalf("the view drew %d times, want 2", len(r.frames))
	}
	if r.ts.Binding.Scheduler().HasScheduledFrame() {
		t.Errorf("the view asked for a frame after a single draw")
	}
	r.ts.Pump(16 * time.Millisecond)
	if len(r.frames) != 2 {
		t.Errorf("the view drew again without being marked")
	}
}

func TestASiblingRepaintDoesNotDrawTheView(t *testing.T) {
	r := newViewRig(t, 1)
	state := 0
	build := func() Widget {
		return NewColumn([]Widget{
			NewSizedBox(NewColoredBox(ui.NewColor(0xFF000000+uint32(state)), nil), SizedBoxOpts{Width: foundation.Some(20.0), Height: foundation.Some(20.0)}),
			NewSizedBox(gpu.NewView(r.paint, r.opts(gpu.ViewOpts{})), SizedBoxOpts{Width: foundation.Some(50.0), Height: foundation.Some(50.0)}),
		})
	}
	r.ts.PumpWidget(build())
	state = 5
	r.ts.PumpWidget(build())
	if len(r.frames) != 1 {
		t.Errorf("the view drew %d times when a sibling changed, want once", len(r.frames))
	}
	if len(r.textureLayers()) != 1 {
		t.Errorf("the scene lost the view's texture")
	}
}

func TestAContinuousViewDrawsAtEveryFrame(t *testing.T) {
	r := newViewRig(t, 1)
	r.ts.PumpWidget(sized(50, 50, gpu.NewView(r.paint, r.opts(gpu.ViewOpts{Continuous: true}))))
	if !r.ts.Binding.Scheduler().HasScheduledFrame() {
		t.Fatalf("a continuous view asked for no frame")
	}
	first := len(r.frames)
	for i := 0; i < 5; i++ {
		r.ts.Pump(16 * time.Millisecond)
	}
	if len(r.frames)-first != 5 {
		t.Errorf("a continuous view drew %d times in 5 frames", len(r.frames)-first)
	}
	if !r.ts.Binding.Scheduler().HasScheduledFrame() {
		t.Errorf("a continuous view stopped asking for frames")
	}
	last := r.frames[len(r.frames)-1].Time
	if last < 4*16*time.Millisecond {
		t.Errorf("the time of the frame is %v, want the elapsed time of the ticker", last)
	}
	// Turning it off stops the frames.
	r.ts.PumpWidget(sized(50, 50, gpu.NewView(r.paint, r.opts(gpu.ViewOpts{}))))
	n := len(r.frames)
	r.ts.Pump(16 * time.Millisecond)
	r.ts.Pump(16 * time.Millisecond)
	if len(r.frames) != n || r.ts.Binding.Scheduler().HasScheduledFrame() {
		t.Errorf("the view went on drawing after it stopped being continuous")
	}
}

func TestAResizeMakesNewTargetsAndReleasesTheOldOnes(t *testing.T) {
	r := newViewRig(t, 1)
	r.ts.PumpWidget(sized(50, 50, gpu.NewView(r.paint, r.opts(gpu.ViewOpts{DepthFormat: gpu.TextureFormatDepth32Float}))))
	first := r.textureLayers()[0].TextureID
	r.ts.PumpWidget(sized(80, 60, gpu.NewView(r.paint, r.opts(gpu.ViewOpts{DepthFormat: gpu.TextureFormatDepth32Float}))))
	if len(r.frames) != 2 {
		t.Fatalf("the view drew %d times after a resize, want 2", len(r.frames))
	}
	f := r.frames[1]
	if f.Width != 80 || f.Height != 60 || f.Depth.Size().Width != 80 {
		t.Errorf("frame = %dx%d, depth %v; want 80x60", f.Width, f.Height, f.Depth.Size())
	}
	second := r.textureLayers()[0].TextureID
	if second == first || second == 0 {
		t.Errorf("texture ids %d then %d: a resize makes a new texture", first, second)
	}
	// The old targets are released after the frame.
	if got := r.rec.Device().Live().Textures; got != 2 {
		t.Errorf("%d textures alive after the resize frame, want 2 (the new color and depth)", got)
	}
	if r.ts.GPUDevice().GPUTextures().Len() != 1 {
		t.Errorf("the registry has %d textures, want 1", r.ts.GPUDevice().GPUTextures().Len())
	}
	// A change of the pixel ratio makes new targets too.
	r.ts.SetDevicePixelRatio(2)
	r.ts.Pump(16 * time.Millisecond)
	f = r.frames[len(r.frames)-1]
	if f.Width != 160 || f.Height != 120 || f.DevicePixelRatio != 2 {
		t.Errorf("after the ratio changed the frame is %dx%d at %v", f.Width, f.Height, f.DevicePixelRatio)
	}
}

func TestAViewWithNoRoomHasNoTargets(t *testing.T) {
	r := newViewRig(t, 1)
	r.ts.PumpWidget(sized(0, 0, gpu.NewView(r.paint, r.opts(gpu.ViewOpts{}))))
	if len(r.frames) != 0 || r.rec.Device().Live().Textures != 0 {
		t.Errorf("an empty view drew %d times and made %d textures", len(r.frames), r.rec.Device().Live().Textures)
	}
	if len(r.textureLayers()) != 0 {
		t.Errorf("an empty view has a texture layer")
	}
}

func TestRemovingTheViewReleasesEverything(t *testing.T) {
	r := newViewRig(t, 1)
	d := must(gpu.DeviceOfBinding(r.ts.Binding))
	r.ts.PumpWidget(sized(50, 50, gpu.NewView(r.paint, r.opts(gpu.ViewOpts{SampleCount: 4, DepthFormat: gpu.TextureFormatDepth32Float}))))
	if d.Resources().Textures != 3 {
		t.Fatalf("the view made %d textures, want 3", d.Resources().Textures)
	}
	r.ts.PumpWidget(NewCenter(NewSizedBox(nil)))
	r.ts.Pump(16 * time.Millisecond)
	if r.disposed != 1 {
		t.Errorf("OnDispose ran %d times, want once", r.disposed)
	}
	if d.Resources().Total() != 0 || r.rec.Device().Live().Textures != 0 {
		t.Errorf("resources alive: %+v, backend %+v", d.Resources(), r.rec.Device().Live())
	}
	if r.rec.GPUTextures().Len() != 0 {
		t.Errorf("the registry still has %d textures", r.rec.GPUTextures().Len())
	}
}

func TestAViewShowsItsFallbackWithoutAGPU(t *testing.T) {
	ts := newPlainTester(t)
	inited := 0
	key := foundation.NewValueKey("fallback")
	ts.PumpWidget(sized(50, 50, gpu.NewView(func(*gpu.ViewFrame) { t.Error("the paint function ran without a GPU") },
		gpu.ViewOpts{Fallback: NewSizedBox(nil, SizedBoxOpts{Key: key}), OnInit: func(*gpu.Device) { inited++ }})))
	if ts.Count(gluttertest.ByKey(key)) != 1 {
		t.Errorf("the fallback is not shown")
	}
	if inited != 0 {
		t.Errorf("OnInit ran without a GPU")
	}
	if ts.Binding.Scheduler().HasScheduledFrame() {
		t.Errorf("a view with no GPU asked for frames")
	}
}

func TestADeviceLossShowsTheFallbackAndInitsAgain(t *testing.T) {
	r := newViewRig(t, 1)
	key := foundation.NewValueKey("fallback")
	o := r.opts(gpu.ViewOpts{Fallback: NewSizedBox(nil, SizedBoxOpts{Key: key}), Continuous: true})
	r.ts.PumpWidget(sized(50, 50, gpu.NewView(r.paint, o)))
	old := r.inits[0]
	r.rec.Lose()
	// The renderer opens the new device at once, as in the tester; the view goes through the fallback in between.
	r.ts.Pump(16 * time.Millisecond)
	r.ts.Pump(16 * time.Millisecond)
	if r.disposed != 1 {
		t.Errorf("OnDispose ran %d times after the loss, want once", r.disposed)
	}
	if len(r.inits) != 2 || r.inits[1] == old || r.inits[1].IsLost() {
		t.Fatalf("OnInit ran %d times; the second device must be a new one", len(r.inits))
	}
	if ts := r.ts; ts.Count(gluttertest.ByKey(key)) != 0 {
		t.Errorf("the fallback is still shown after the new device")
	}
	last := r.frames[len(r.frames)-1]
	if last.Device != r.inits[1] {
		t.Errorf("the view draws on the old device")
	}
	if len(r.textureLayers()) != 1 {
		t.Errorf("the scene has %d texture layers after the loss, want 1", len(r.textureLayers()))
	}
}

func TestAPanicInThePaintFunctionIsReported(t *testing.T) {
	r := newViewRig(t, 1)
	r.ts.PumpWidget(sized(50, 50, gpu.NewView(func(*gpu.ViewFrame) { panic("boom") }, r.opts(gpu.ViewOpts{}))))
	err := r.ts.TakeError()
	if err == nil {
		t.Fatal("the panic of the paint function was not reported")
	}
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != foundation.PaintPanicked {
		t.Errorf("error = %v, want the tag %s", err, foundation.PaintPanicked)
	}
	if len(r.textureLayers()) != 1 {
		t.Errorf("the view lost its texture layer after the panic")
	}
}

func TestNewViewChecksItsOptions(t *testing.T) {
	skipUnlessDebug(t)
	paint := func(*gpu.ViewFrame) {}
	tests := []struct {
		name string
		make func()
	}{
		{"no paint function", func() { gpu.NewView(nil) }},
		{"a format that is not a view's", func() { gpu.NewView(paint, gpu.ViewOpts{Format: gpu.TextureFormatBGRA8Unorm}) }},
		{"a depth format that is a color format", func() { gpu.NewView(paint, gpu.ViewOpts{DepthFormat: gpu.TextureFormatRGBA8Unorm}) }},
		{"two samples", func() { gpu.NewView(paint, gpu.ViewOpts{SampleCount: 2}) }},
		{"two options", func() { gpu.NewView(paint, gpu.ViewOpts{}, gpu.ViewOpts{}) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := panicOf(tt.make)
			wantTag(t, err, gpu.TagGPUViewInvalid, "NewView.GPUViewInvalid")
		})
	}
}

func TestTheViewIsReleasedWhenTheAppEnds(t *testing.T) {
	r := newViewRig(t, 1)
	d := must(gpu.DeviceOfBinding(r.ts.Binding))
	r.ts.PumpWidget(sized(50, 50, gpu.NewView(r.paint, r.opts(gpu.ViewOpts{SampleCount: 4, DepthFormat: gpu.TextureFormatDepth32Float}))))
	r.ts.Binding.DetachRootWidget() // what RunApp does before the device is closed
	if r.disposed != 1 || d.Resources().Total() != 0 {
		t.Errorf("OnDispose ran %d times and %+v is alive", r.disposed, d.Resources())
	}
}
