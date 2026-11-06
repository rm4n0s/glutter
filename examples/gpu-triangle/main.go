// Command gpu-triangle shows package gpu: a lit cube that turns, drawn by the program's own shader into a gpu.View
// with a depth buffer and 4x multisampling, inside a Material page. A slider sets the speed of the turn and a switch
// turns the multisampling on and off. The view draws at every frame (Continuous). The program uses only the public
// packages: gpu for the device, the buffers and the pipelines, and material for the page. The cube is drawn with
// Blinn-Phong shading by shaders/cube.wgsl, which `glutter-gen gpushaders` turned into shaders/shaders_gen.go; the
// few lines of matrix math are in this file.
//
// Run it with: go run ./examples/gpu-triangle
//
// It needs a GPU renderer (Vulkan on Linux and Android, Direct3D 12 on Windows, WebGPU on the web). With the OpenGL ES,
// WebGL or software renderer the view shows its fallback, a line of text.
//
// On Windows: go run ./cmd/glutter-windows run ./examples/gpu-triangle. On the web: go run ./cmd/glutter-web serve
// ./examples/gpu-triangle. On Android: go run ./cmd/glutter-android run ./examples/gpu-triangle.
//
// Environment: GLUTTER_BACKEND=x11|wayland (Linux) or win32 (Windows), GLUTTER_RENDERER=vulkan|d3d12|webgpu,
// GLUTTER_SCREENSHOT=<file.png>, GLUTTER_SCREENSHOT_AFTER=<duration>.
package main

//go:generate go run github.com/rm4n0s/glutter/cmd/glutter-gen gpushaders ./shaders

import (
	"encoding/binary"
	"log"
	"math"
	"time"

	"github.com/rm4n0s/glutter/examples/gpu-triangle/shaders"
	"github.com/rm4n0s/glutter/gpu"
	m "github.com/rm4n0s/glutter/material"
)

func main() {
	err := m.RunApp(NewCubeApp(), m.RunAppOpts{
		AppID:  "dev.glutter.gpu-triangle",
		Window: m.WindowConfig{Title: "GPU cube", Size: m.Size{Width: 640, Height: 560}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// CubeApp is the Material app.
type CubeApp struct{ m.StatelessWidget }

// NewCubeApp creates the app.
func NewCubeApp() *CubeApp { return &CubeApp{} }

// Build makes the app with the page of the cube.
func (a *CubeApp) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "GPU cube",
		Theme:                      m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Teal.Primary)}),
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       NewCubePage(),
	})
}

// CubePage is the page: the view and its controls.
type CubePage struct{ m.StatefulWidget }

// NewCubePage creates the page.
func NewCubePage(opts ...m.WidgetOpts) *CubePage {
	return &CubePage{StatefulWidget: m.NewStatefulWidgetBase(opts...)}
}

// CreateState creates the state.
func (p *CubePage) CreateState() m.State { return &cubePageState{} }

type cubePageState struct {
	m.StateOf[*CubePage]
	cube          cube
	speed         float64
	multisampling bool
}

// InitState starts the cube at the speed of one turn in 6 seconds, with 4x multisampling.
func (s *cubePageState) InitState() {
	s.speed = 1
	s.multisampling = true
}

func (s *cubePageState) Build(bc m.BuildContext) m.Widget {
	samples := 1
	if s.multisampling {
		samples = 4
	}
	view := gpu.NewView(s.cube.paint, gpu.ViewOpts{
		DepthFormat:    gpu.TextureFormatDepth32Float,
		SampleCount:    samples,
		Continuous:     true,
		OnInit:         s.cube.init,
		OnDispose:      s.cube.dispose,
		SemanticsLabel: "A lit cube that turns",
		Fallback: m.NewCenter(m.NewPadding(m.NewEdgeInsetsAll(24), m.NewText(
			"This app needs a GPU renderer: Vulkan, Direct3D 12 or WebGPU.", m.TextOpts{TextAlign: m.TextAlignCenter}))),
	})
	textTheme := m.ThemeOf(bc).TextTheme
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("GPU cube")}),
		Body: m.NewColumn([]m.Widget{
			m.NewExpanded(view),
			m.NewPadding(m.NewEdgeInsetsSymmetric(16, 8), m.NewRow([]m.Widget{
				m.NewText("Speed", m.TextOpts{Style: textTheme.LabelLarge}),
				m.NewExpanded(m.NewSlider(s.speed, func(v float64) { s.SetState(func() { s.speed = v }) }, m.SliderOpts{Max: m.Some(4.0), Label: m.Some("speed")})),
				m.NewText("4x multisampling", m.TextOpts{Style: textTheme.LabelLarge}),
				m.NewPadding(m.NewEdgeInsetsFromLTRB(8, 0, 0, 0), m.NewSwitch(s.multisampling, func(v bool) { s.SetState(func() { s.multisampling = v }) })),
			})),
		}),
	})
}

// --- the cube ---------------------------------------------------------------------------------------------------

// cube owns the GPU resources of the view and draws the cube. The view calls init for each device and dispose when
// the device is gone or the view leaves.
type cube struct {
	dev      *gpu.Device
	module   *gpu.ShaderModule
	vertices *gpu.Buffer
	uniforms *gpu.Buffer
	// variants are the pipelines (and their bind groups) by sample count: a pipeline draws into targets of one sample
	// count only, and the switch of the page changes it.
	variants map[int]*variant
	angle    float64
	last     time.Duration
	speed    func() float64
}

type variant struct {
	pipeline *gpu.RenderPipeline
	group    *gpu.BindGroup
}

const (
	vertexCount = 36
	vertexBytes = 6 * 4 // a position and a normal
	sceneBytes  = 176   // two matrices and three vectors
)

func (c *cube) init(d *gpu.Device) {
	var err error
	c.dev = d
	c.variants = map[int]*variant{}
	c.angle, c.last = 0.7, 0 // a pose that shows three faces
	if c.module, err = d.NewShaderModule(shaders.Cube); err != nil {
		log.Printf("the shader: %v", err)
		return
	}
	data := cubeVertices()
	if c.vertices, err = d.NewBuffer(gpu.BufferDesc{Label: "cube vertices", Size: len(data), Usage: gpu.BufferUsageVertex | gpu.BufferUsageCopyDst}); err != nil {
		log.Printf("the vertices: %v", err)
		return
	}
	d.Queue().WriteBuffer(c.vertices, 0, data)
	if c.uniforms, err = d.NewBuffer(gpu.BufferDesc{Label: "cube matrices", Size: 256, Usage: gpu.BufferUsageUniform | gpu.BufferUsageCopyDst}); err != nil {
		log.Printf("the matrices: %v", err)
	}
}

func (c *cube) dispose() {
	for _, v := range c.variants {
		v.group.Release()
		v.pipeline.Release()
	}
	c.variants = nil
	for _, r := range []interface{ Release() }{c.uniforms, c.vertices, c.module} {
		if r != nil {
			r.Release()
		}
	}
	c.uniforms, c.vertices, c.module, c.dev = nil, nil, nil, nil
}

// variantFor returns the pipeline for targets of the sample count, making it the first time.
func (c *cube) variantFor(samples int) (*variant, error) {
	if v, ok := c.variants[samples]; ok {
		return v, nil
	}
	pipeline, err := c.dev.NewRenderPipeline(gpu.RenderPipelineDesc{
		Label: "cube",
		Vertex: gpu.VertexState{Module: c.module, EntryPoint: "vs_main", Buffers: []gpu.VertexBufferLayout{{
			ArrayStride: vertexBytes,
			Attributes: []gpu.VertexAttribute{
				{ShaderLocation: 0, Format: gpu.VertexFormatFloat32x3, Offset: 0},
				{ShaderLocation: 1, Format: gpu.VertexFormatFloat32x3, Offset: 12},
			},
		}}},
		Fragment:     gpu.FragmentState{Module: c.module, EntryPoint: "fs_main", Targets: []gpu.ColorTargetState{{Format: gpu.TextureFormatRGBA8Unorm}}},
		Primitive:    gpu.PrimitiveState{CullMode: gpu.CullModeBack},
		DepthStencil: &gpu.DepthStencilState{Format: gpu.TextureFormatDepth32Float, DepthWriteEnabled: true},
		Multisample:  gpu.MultisampleState{Count: samples},
	})
	if err != nil {
		return nil, err
	}
	group, err := c.dev.NewBindGroup(gpu.BindGroupDesc{Layout: pipeline.BindGroupLayout(0), Entries: []gpu.BindGroupEntry{
		{Binding: 0, Buffer: c.uniforms, Size: sceneBytes},
	}})
	if err != nil {
		pipeline.Release()
		return nil, err
	}
	v := &variant{pipeline: pipeline, group: group}
	c.variants[samples] = v
	return v, nil
}

// paint draws the cube into the frame of the view.
func (c *cube) paint(f *gpu.ViewFrame) {
	if c.dev == nil || c.uniforms == nil {
		return
	}
	v, err := c.variantFor(f.Color.SampleCount())
	if err != nil {
		log.Printf("the pipeline: %v", err)
		return
	}
	dt := f.Time - c.last
	c.last = f.Time
	if dt < 0 || dt > 100*time.Millisecond {
		dt = 0
	}
	speed := 1.0
	if c.speed != nil {
		speed = c.speed()
	}
	c.angle += dt.Seconds() * speed * 2 * math.Pi / 6
	c.dev.Queue().WriteBuffer(c.uniforms, 0, sceneUniforms(c.angle, float64(f.Width)/float64(f.Height)))

	color := gpu.RenderPassColorAttachment{View: f.Color.View(), ClearValue: gpu.Color{R: 0.07, G: 0.09, B: 0.11, A: 1}}
	if f.Resolve != nil {
		color.ResolveTarget = f.Resolve.View()
		color.StoreOp = gpu.StoreOpDiscard
	}
	pass := f.Encoder.BeginRenderPass(gpu.RenderPassDesc{
		ColorAttachments:       []gpu.RenderPassColorAttachment{color},
		DepthStencilAttachment: &gpu.RenderPassDepthStencilAttachment{View: f.Depth.View(), DepthClearValue: 1},
	})
	pass.SetPipeline(v.pipeline)
	pass.SetBindGroup(0, v.group, 0)
	pass.SetVertexBuffer(0, c.vertices, 0)
	pass.Draw(vertexCount, 1, 0, 0)
	pass.End()
}

// cubeVertices returns the 36 vertices of a cube of side 1.6: a position and the normal of the face.
func cubeVertices() []byte {
	faces := []struct{ n, u, v [3]float32 }{
		{[3]float32{1, 0, 0}, [3]float32{0, 1, 0}, [3]float32{0, 0, 1}},
		{[3]float32{-1, 0, 0}, [3]float32{0, 0, 1}, [3]float32{0, 1, 0}},
		{[3]float32{0, 1, 0}, [3]float32{0, 0, 1}, [3]float32{1, 0, 0}},
		{[3]float32{0, -1, 0}, [3]float32{1, 0, 0}, [3]float32{0, 0, 1}},
		{[3]float32{0, 0, 1}, [3]float32{1, 0, 0}, [3]float32{0, 1, 0}},
		{[3]float32{0, 0, -1}, [3]float32{0, 1, 0}, [3]float32{1, 0, 0}},
	}
	const half = 0.8
	var floats []float32
	for _, f := range faces {
		corner := func(su, sv float32) [3]float32 {
			var p [3]float32
			for i := range p {
				p[i] = half * (f.n[i] + su*f.u[i] + sv*f.v[i])
			}
			return p
		}
		p0, p1, p2, p3 := corner(-1, -1), corner(1, -1), corner(1, 1), corner(-1, 1)
		for _, p := range [][3]float32{p0, p1, p2, p0, p2, p3} {
			floats = append(floats, p[0], p[1], p[2], f.n[0], f.n[1], f.n[2])
		}
	}
	return floatBytes(floats)
}

func floatBytes(floats []float32) []byte {
	out := make([]byte, 4*len(floats))
	for i, f := range floats {
		binary.LittleEndian.PutUint32(out[4*i:], math.Float32bits(f))
	}
	return out
}

// sceneUniforms returns the bytes of the Scene struct of cube.wgsl for the angle of the cube.
func sceneUniforms(angle, aspect float64) []byte {
	// A narrow view (a phone held upright) has a narrow field of view across: the camera backs off to keep the cube in.
	back := float32(math.Max(1, 0.85/aspect))
	eye := [3]float32{0, 1.6 * back, 4.6 * back}
	model := mul(rotateY(float32(angle)), rotateX(0.5))
	view := lookAt(eye, [3]float32{0, 0, 0}, [3]float32{0, 1, 0})
	proj := perspective(math.Pi/4, float32(aspect), 0.1, 50)
	mvp := mul(proj, mul(view, model))
	floats := make([]float32, 0, sceneBytes/4)
	floats = append(floats, mvp[:]...)
	floats = append(floats, model[:]...)
	floats = append(floats, 0.5, 0.8, 0.6, 0) // towards the light
	floats = append(floats, eye[0], eye[1], eye[2], 0)
	floats = append(floats, 0.62, 0.62, 0.62, 1) // the color of the cube, which each face leans away from
	return floatBytes(floats)
}

// --- matrices: column-major 4x4, as WGSL reads them; the clip space is WebGPU's, with z from 0 to 1 ---------------

type mat4 [16]float32

func mul(a, b mat4) mat4 {
	var r mat4
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			var s float32
			for k := 0; k < 4; k++ {
				s += a[k*4+row] * b[col*4+k]
			}
			r[col*4+row] = s
		}
	}
	return r
}

func rotateY(a float32) mat4 {
	s, c := float32(math.Sin(float64(a))), float32(math.Cos(float64(a)))
	return mat4{c, 0, -s, 0, 0, 1, 0, 0, s, 0, c, 0, 0, 0, 0, 1}
}

func rotateX(a float32) mat4 {
	s, c := float32(math.Sin(float64(a))), float32(math.Cos(float64(a)))
	return mat4{1, 0, 0, 0, 0, c, s, 0, 0, -s, c, 0, 0, 0, 0, 1}
}

func perspective(fovy float64, aspect, near, far float32) mat4 {
	f := float32(1 / math.Tan(fovy/2))
	return mat4{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, far / (near - far), -1,
		0, 0, near * far / (near - far), 0,
	}
}

func lookAt(eye, center, up [3]float32) mat4 {
	z := normalize([3]float32{eye[0] - center[0], eye[1] - center[1], eye[2] - center[2]})
	x := normalize(cross(up, z))
	y := cross(z, x)
	return mat4{
		x[0], y[0], z[0], 0,
		x[1], y[1], z[1], 0,
		x[2], y[2], z[2], 0,
		-dot(x, eye), -dot(y, eye), -dot(z, eye), 1,
	}
}

func cross(a, b [3]float32) [3]float32 {
	return [3]float32{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}

func dot(a, b [3]float32) float32 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }

func normalize(a [3]float32) [3]float32 {
	l := float32(math.Sqrt(float64(dot(a, a))))
	return [3]float32{a[0] / l, a[1] / l, a[2] / l}
}
