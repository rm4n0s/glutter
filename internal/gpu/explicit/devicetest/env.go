package devicetest

import (
	"encoding/binary"
	"image"
	"math"
	"sync/atomic"
	"time"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// Env is what a scene runs on: a device, and how to wait for the reads of a frame.
//
// Flutter: no counterpart
type Env struct {
	// Dev is the device under test.
	Dev explicit.Device
	// Pixels says that the device draws, so that the scenes compare what they read back. The fake
	// device does not: its scenes check only that the descriptors are valid.
	Pixels bool
	// Settle does what lets the reads of the frames submitted so far be delivered, and is called
	// until they are (30 seconds at most). A device that waits for the GPU in Submit needs nothing;
	// the fake device completes the reads, and WebGPU lets the browser's event loop run.
	Settle func()

	cleanup []func()
	pending atomic.Int64
}

// T is what a scene needs of a test: T through gotest.Wrap, and Collector for a program
// that is not a test (the one that runs the scenes in a browser).
//
// Flutter: no counterpart
type T interface {
	// Helper marks the calling function as a helper.
	Helper()
	// Errorf records a failure and goes on.
	Errorf(format string, args ...any)
	// Fatalf records a failure and ends the scene.
	Fatalf(format string, args ...any)
	// Run runs fn as a part of the scene, with a name.
	Run(name string, fn func(t T))
}

// Scene is one test that every backend runs.
//
// Flutter: no counterpart
type Scene struct {
	// Name names the subtest.
	Name string
	// Run draws with env.Dev and checks the result.
	Run func(t T, env *Env)
}

// Run runs every scene as a subtest and destroys the resources each one made.
//
// Flutter: no counterpart
func Run(t T, env *Env) {
	t.Helper()
	for _, s := range Scenes() {
		t.Run(s.Name, func(t T) {
			defer env.release()
			s.Run(t, env)
		})
	}
}

func (e *Env) release() {
	for i := len(e.cleanup) - 1; i >= 0; i-- {
		e.cleanup[i]()
	}
	e.cleanup = nil
}

// texture makes a texture that the scene destroys.
func (e *Env) texture(t T, desc explicit.TextureDesc) explicit.Texture {
	t.Helper()
	tex, err := e.Dev.NewTexture(desc)
	if err != nil {
		t.Fatalf("NewTexture(%+v): %v", desc, err)
	}
	e.cleanup = append(e.cleanup, tex.Destroy)
	return tex
}

// buffer makes a buffer that the scene destroys, holding data.
func (e *Env) buffer(t T, usage explicit.BufferUsage, size int, data []byte) explicit.Buffer {
	t.Helper()
	b, err := e.Dev.NewBuffer(size, usage)
	if err != nil {
		t.Fatalf("NewBuffer: %v", err)
	}
	e.cleanup = append(e.cleanup, b.Destroy)
	if len(data) > 0 {
		e.Dev.WriteBuffer(b, 0, data)
	}
	return b
}

// sampler makes a sampler that the scene destroys.
func (e *Env) sampler(t T, desc explicit.SamplerDesc) explicit.Sampler {
	t.Helper()
	s, err := e.Dev.NewSampler(desc)
	if err != nil {
		t.Fatalf("NewSampler: %v", err)
	}
	e.cleanup = append(e.cleanup, s.Destroy)
	return s
}

// pipeline makes a pipeline that the scene destroys.
func (e *Env) pipeline(t T, desc explicit.PipelineDesc) explicit.Pipeline {
	t.Helper()
	p, err := e.Dev.NewPipeline(desc)
	if err != nil {
		t.Fatalf("NewPipeline(%s %s): %v", desc.VertexEntry, desc.FragmentEntry, err)
	}
	e.cleanup = append(e.cleanup, p.Destroy)
	return p
}

// frame begins a frame.
func (e *Env) frame(t T) explicit.Encoder {
	t.Helper()
	enc, err := e.Dev.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	return enc
}

// Read is the result of a read of a texture, valid after Env.submit.
type read struct {
	data     []byte
	rowBytes int
	done     bool
	err      error
}

// read asks for the bytes of the rectangle of a texture.
func (e *Env) read(enc explicit.Encoder, tex explicit.Texture, r image.Rectangle) *read {
	res := &read{}
	e.pending.Add(1)
	enc.ReadTextureBytes(tex, r, func(data []byte, rowBytes int, err error) {
		res.data, res.rowBytes, res.err, res.done = data, rowBytes, err, true
		e.pending.Add(-1)
	})
	return res
}

// readTimeout is how long submit waits for the reads of a frame.
const readTimeout = 30 * time.Second

// submit sends the frame and waits for its reads.
func (e *Env) submit(t T, enc explicit.Encoder, reads ...*read) {
	t.Helper()
	if err := enc.Submit(); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	for start := time.Now(); e.pending.Load() > 0; {
		if e.Settle == nil || time.Since(start) > readTimeout {
			t.Fatalf("%d reads were not delivered", e.pending.Load())
		}
		e.Settle()
	}
	for i, r := range reads {
		if !r.done {
			t.Fatalf("read %d was not delivered", i)
		}
		if r.err != nil {
			t.Fatalf("read %d: %v", i, r.err)
		}
	}
}

// pixel returns the bytes of the pixel (x, y) of a read whose pixels take bpp bytes.
func (r *read) pixel(x, y, bpp int) []byte {
	i := y*r.rowBytes + x*bpp
	return r.data[i : i+bpp]
}

// near reports whether the bytes differ by at most tol each.
func near(got, want []byte, tol int) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		d := int(got[i]) - int(want[i])
		if d < -tol || d > tol {
			return false
		}
	}
	return true
}

// vertex is a vertex of tri.wgsl: a position and a color.
type vertex struct {
	x, y, z float32
	rgba    [4]byte
}

// vertexBytes packs vertices as the buffer of triVertexLayout holds them: 16 bytes each.
func vertexBytes(vs []vertex) []byte {
	out := make([]byte, 0, len(vs)*16)
	for _, v := range vs {
		out = binary.LittleEndian.AppendUint32(out, math.Float32bits(v.x))
		out = binary.LittleEndian.AppendUint32(out, math.Float32bits(v.y))
		out = binary.LittleEndian.AppendUint32(out, math.Float32bits(v.z))
		out = append(out, v.rgba[:]...)
	}
	return out
}

// triVertexLayout is the vertex buffer of tri.wgsl.
func triVertexLayout() []explicit.VertexBufferLayout {
	return []explicit.VertexBufferLayout{{Stride: 16, Attributes: []explicit.VertexAttribute{
		{Location: 0, Format: explicit.VertexFloat32x3, Offset: 0},
		{Location: 1, Format: explicit.VertexUnorm8x4, Offset: 12},
	}}}
}

// bigTriangle is a triangle that covers the whole target at the depth z.
func bigTriangle(z float32, rgba [4]byte) []vertex {
	return []vertex{{-1, -1, z, rgba}, {3, -1, z, rgba}, {-1, 3, z, rgba}}
}

var (
	red   = [4]byte{255, 0, 0, 255}
	green = [4]byte{0, 255, 0, 255}
	blue  = [4]byte{0, 0, 255, 255}
	white = [4]byte{255, 255, 255, 255}
)

// triPipeline is a pipeline of tri.wgsl with the state.
func (e *Env) triPipeline(t T, desc explicit.PipelineDesc) explicit.Pipeline {
	t.Helper()
	desc.Shader, desc.VertexEntry, desc.FragmentEntry = ShaderTri, "vs_tri", "fs_tri"
	desc.VertexBuffers = triVertexLayout()
	return e.pipeline(t, desc)
}

// fillPipeline is a pipeline of fill.wgsl drawing the full-screen triangle with the fragment entry.
func (e *Env) fillPipeline(t T, fragment string, targets ...explicit.ColorTargetState) explicit.Pipeline {
	t.Helper()
	return e.pipeline(t, explicit.PipelineDesc{Shader: ShaderFill, VertexEntry: "vs_full", FragmentEntry: fragment, ColorTargets: targets})
}

// colorTarget makes a render target that can be read.
func (e *Env) colorTarget(t T, w, h int, f explicit.TextureFormat) explicit.Texture {
	t.Helper()
	return e.texture(t, explicit.TextureDesc{Width: w, Height: h, Format: f, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource | explicit.TextureSampled})
}
