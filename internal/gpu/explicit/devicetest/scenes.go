package devicetest

import (
	"encoding/binary"
	"image"
	"math"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// Scenes returns every scene.
//
// Flutter: no counterpart
func Scenes() []Scene {
	return []Scene{
		{"formats are written and read back", formatsRoundTrip},
		{"formats are cleared and drawn into", formatsRendered},
		{"depth test with three triangles", depthTest},
		{"4 samples resolved", multisampled},
		{"each face of a cube", cubeFaces},
		{"each layer of an array", arrayLayers},
		{"a 3D texture is sampled", volumeSampled},
		{"integer and float32 textures are read by a shader", unfilterableTextures},
		{"four color attachments", fourAttachments},
		{"clip space and culling", clipSpaceAndCulling},
		{"topologies", topologies},
		{"stencil and color write mask", stencilAndWriteMask},
		{"vertex formats", vertexFormats},
	}
}

// ---------------------------------------------------------------------------------------------
// Formats

// formatBytes returns n pixels of a format with values that are valid for it (no NaN in the floats)
// and different from pixel to pixel.
func formatBytes(f explicit.TextureFormat, n int) []byte {
	bpp := f.BytesPerPixel()
	out := make([]byte, 0, n*bpp)
	for i := 0; i < n; i++ {
		switch f {
		case explicit.TextureFormatRGBA16Float:
			for c := 0; c < 4; c++ {
				out = binary.LittleEndian.AppendUint16(out, explicit.Float16Bits(float32((i+c)%97)/8))
			}
		case explicit.TextureFormatRGBA32Float:
			for c := 0; c < 4; c++ {
				out = binary.LittleEndian.AppendUint32(out, math.Float32bits(float32((i*3+c)%97)*0.37))
			}
		case explicit.TextureFormatR32Float:
			out = binary.LittleEndian.AppendUint32(out, math.Float32bits(float32(i%97)*0.37-9))
		case explicit.TextureFormatR32Uint:
			out = binary.LittleEndian.AppendUint32(out, uint32(i)*2654435761)
		case explicit.TextureFormatRG32Uint:
			out = binary.LittleEndian.AppendUint32(out, uint32(i)*2654435761)
			out = binary.LittleEndian.AppendUint32(out, uint32(i)*40503+7)
		default:
			for c := 0; c < bpp; c++ {
				out = append(out, byte(i*7+c*13+3))
			}
		}
	}
	return out
}

// colorFormats are the formats of the color textures of the explicit device.
var colorFormats = []explicit.TextureFormat{
	explicit.TextureFormatRGBA8Unorm, explicit.TextureFormatBGRA8Unorm, explicit.TextureFormatR8Unorm,
	explicit.TextureFormatRGBA16Float, explicit.TextureFormatRGBA32Float, explicit.TextureFormatR32Float,
	explicit.TextureFormatR32Uint, explicit.TextureFormatRG32Uint,
}

func formatName(f explicit.TextureFormat) string {
	return map[explicit.TextureFormat]string{
		explicit.TextureFormatRGBA8Unorm: "rgba8unorm", explicit.TextureFormatBGRA8Unorm: "bgra8unorm",
		explicit.TextureFormatR8Unorm: "r8unorm", explicit.TextureFormatRGBA16Float: "rgba16float",
		explicit.TextureFormatRGBA32Float: "rgba32float", explicit.TextureFormatR32Float: "r32float",
		explicit.TextureFormatR32Uint: "r32uint", explicit.TextureFormatRG32Uint: "rg32uint",
		explicit.TextureFormatDepth32Float: "depth32float", explicit.TextureFormatDepth24PlusStencil8: "depth24plus-stencil8",
	}[f]
}

// formatsRoundTrip writes a texture of every color format with rows that have padding and reads it
// back exactly, whole and in part.
func formatsRoundTrip(t T, env *Env) {
	const w, h = 5, 3
	for _, f := range colorFormats {
		t.Run(formatName(f), func(t T) {
			tex := env.texture(t, explicit.TextureDesc{Width: w, Height: h, Format: f,
				Usage: explicit.TextureSampled | explicit.TextureCopySource | explicit.TextureRenderTarget})
			bpp := f.BytesPerPixel()
			want := formatBytes(f, w*h)
			stride := w*bpp + 8
			padded := make([]byte, stride*h)
			for y := 0; y < h; y++ {
				copy(padded[y*stride:], want[y*w*bpp:(y+1)*w*bpp])
			}
			env.Dev.WriteTexture(tex, 0, image.Rect(0, 0, w, h), padded, stride)
			enc := env.frame(t)
			whole := env.read(enc, tex, image.Rect(0, 0, w, h))
			part := env.read(enc, tex, image.Rect(1, 1, 4, 3))
			env.submit(t, enc, whole, part)
			if !env.Pixels {
				return
			}
			if whole.rowBytes != w*bpp || string(whole.data) != string(want) {
				t.Errorf("the whole texture reads back as\n%v\nwant\n%v (row bytes %d)", whole.data, want, whole.rowBytes)
			}
			for y := 1; y < 3; y++ {
				for x := 1; x < 4; x++ {
					got := part.pixel(x-1, y-1, bpp)
					if string(got) != string(want[(y*w+x)*bpp:(y*w+x+1)*bpp]) {
						t.Errorf("pixel (%d,%d) of the part reads %v, want %v", x, y, got, want[(y*w+x)*bpp:(y*w+x+1)*bpp])
					}
				}
			}
		})
	}
}

// formatsRendered clears each format and draws a constant into it with a shader.
func formatsRendered(t T, env *Env) {
	const size = 6
	type expect struct {
		wantClear, wantDraw []byte
		tol                 int
	}
	f32 := func(vs ...float32) []byte {
		var out []byte
		for _, v := range vs {
			out = binary.LittleEndian.AppendUint32(out, math.Float32bits(v))
		}
		return out
	}
	u32 := func(vs ...uint32) []byte {
		var out []byte
		for _, v := range vs {
			out = binary.LittleEndian.AppendUint32(out, v)
		}
		return out
	}
	f16 := func(vs ...float32) []byte {
		var out []byte
		for _, v := range vs {
			out = binary.LittleEndian.AppendUint16(out, explicit.Float16Bits(v))
		}
		return out
	}
	cases := []struct {
		format   explicit.TextureFormat
		fragment string
		clear    [4]float64
		expect
	}{
		{explicit.TextureFormatRGBA8Unorm, "fs_float", [4]float64{1, 0, 0.5, 1}, expect{[]byte{255, 0, 128, 255}, []byte{64, 128, 191, 255}, 1}},
		{explicit.TextureFormatBGRA8Unorm, "fs_float", [4]float64{1, 0, 0.5, 1}, expect{[]byte{128, 0, 255, 255}, []byte{191, 128, 64, 255}, 1}},
		{explicit.TextureFormatR8Unorm, "fs_float", [4]float64{1, 0, 0, 0}, expect{[]byte{255}, []byte{64}, 1}},
		{explicit.TextureFormatRGBA16Float, "fs_float", [4]float64{0.25, 0.5, 2, 1}, expect{f16(0.25, 0.5, 2, 1), f16(0.25, 0.5, 0.75, 1), 0}},
		{explicit.TextureFormatRGBA32Float, "fs_float", [4]float64{0.25, 0.5, 2, 1}, expect{f32(0.25, 0.5, 2, 1), f32(0.25, 0.5, 0.75, 1), 0}},
		{explicit.TextureFormatR32Float, "fs_float", [4]float64{-3.5, 0, 0, 0}, expect{f32(-3.5), f32(0.25), 0}},
		{explicit.TextureFormatR32Uint, "fs_uint", [4]float64{5, 0, 0, 0}, expect{u32(5), u32(7), 0}},
		{explicit.TextureFormatRG32Uint, "fs_uint", [4]float64{5, 6, 0, 0}, expect{u32(5, 6), u32(7, 9), 0}},
	}
	for _, c := range cases {
		t.Run(formatName(c.format), func(t T) {
			tex := env.colorTarget(t, size, size, c.format)
			p := env.fillPipeline(t, c.fragment, explicit.ColorTargetState{Format: c.format})
			bpp := c.format.BytesPerPixel()
			enc := env.frame(t)
			enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: tex, Clear: c.clear}}}).End()
			cleared := env.read(enc, tex, image.Rect(0, 0, size, size))
			pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: tex, Load: explicit.LoadClear}}})
			pass.SetPipeline(p)
			pass.Draw(3, 1, 0, 0)
			pass.End()
			drawn := env.read(enc, tex, image.Rect(0, 0, size, size))
			env.submit(t, enc, cleared, drawn)
			if !env.Pixels {
				return
			}
			for _, px := range [][2]int{{0, 0}, {3, 2}, {5, 5}} {
				if got := cleared.pixel(px[0], px[1], bpp); !near(got, c.wantClear, c.tol) {
					t.Errorf("cleared pixel %v = %v, want %v", px, got, c.wantClear)
				}
				if got := drawn.pixel(px[0], px[1], bpp); !near(got, c.wantDraw, c.tol) {
					t.Errorf("drawn pixel %v = %v, want %v", px, got, c.wantDraw)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------------------------
// Depth, stencil, multisampling

// depthTest draws three triangles that cover the target at three depths, in several orders and
// with the depth write on and off, reversed-Z included.
func depthTest(t T, env *Env) {
	const size = 8
	zA, zB, zC := float32(0.3), float32(0.6), float32(0.1)
	verts := append(append(bigTriangle(zA, red), bigTriangle(zB, green)...), bigTriangle(zC, blue)...)
	vb := env.buffer(t, explicit.BufferVertex, len(verts)*16, vertexBytes(verts))
	order := func(names string) []int {
		var out []int
		for _, c := range names {
			out = append(out, int(c-'A'))
		}
		return out
	}
	tests := []struct {
		name    string
		depth   explicit.TextureFormat
		write   bool
		compare explicit.CompareFunc
		clear   float64
		draw    string
		want    [4]byte
	}{
		{"the nearest wins, drawn near to far", explicit.TextureFormatDepth32Float, true, explicit.CompareLessEqual, 1, "CAB", blue},
		{"the nearest wins, drawn far to near", explicit.TextureFormatDepth32Float, true, explicit.CompareLessEqual, 1, "BAC", blue},
		{"the nearest wins, in depth24plus-stencil8", explicit.TextureFormatDepth24PlusStencil8, true, explicit.CompareLess, 1, "BCA", blue},
		{"the last wins without the depth write", explicit.TextureFormatDepth32Float, false, explicit.CompareLessEqual, 1, "ACB", green},
		{"the last wins without the depth write, in depth24plus-stencil8", explicit.TextureFormatDepth24PlusStencil8, false, explicit.CompareLessEqual, 1, "BCA", red},
		{"the same order with the depth write", explicit.TextureFormatDepth32Float, true, explicit.CompareLessEqual, 1, "ACB", blue},
		{"reversed-Z: the largest wins", explicit.TextureFormatDepth32Float, true, explicit.CompareGreaterEqual, 0, "CBA", green},
		{"reversed-Z: the largest wins, drawn in the other order", explicit.TextureFormatDepth32Float, true, explicit.CompareGreater, 0, "BAC", green},
		{"the test never passes", explicit.TextureFormatDepth32Float, true, explicit.CompareNever, 1, "ABC", [4]byte{0, 0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t T) {
			color := env.colorTarget(t, size, size, explicit.TextureFormatRGBA8Unorm)
			depth := env.texture(t, explicit.TextureDesc{Width: size, Height: size, Format: tt.depth, Usage: explicit.TextureRenderTarget})
			p := env.triPipeline(t, explicit.PipelineDesc{
				ColorTargets: []explicit.ColorTargetState{{Format: explicit.TextureFormatRGBA8Unorm}},
				DepthStencil: explicit.DepthStencilState{Format: tt.depth, DepthWrite: tt.write, DepthCompare: tt.compare},
			})
			enc := env.frame(t)
			pass := enc.BeginRenderPass(explicit.PassDesc{
				Color: []explicit.ColorAttachment{{Target: color}},
				Depth: &explicit.DepthAttachment{Target: depth, ClearDepth: tt.clear, DepthStore: explicit.StoreDiscard, StencilStore: explicit.StoreDiscard},
			})
			pass.SetPipeline(p)
			pass.SetVertexBuffer(0, vb, 0)
			for _, i := range order(tt.draw) {
				pass.Draw(3, 1, 3*i, 0)
			}
			pass.End()
			res := env.read(enc, color, image.Rect(0, 0, size, size))
			env.submit(t, enc, res)
			if !env.Pixels {
				return
			}
			for _, px := range [][2]int{{0, 0}, {4, 4}, {7, 7}} {
				if got := res.pixel(px[0], px[1], 4); !near(got, tt.want[:], 0) {
					t.Errorf("pixel %v = %v, want %v", px, got, tt.want)
				}
			}
		})
	}
}

// multisampled draws a triangle with 4 samples, resolves it, and compares the edge with the
// coverage of the triangle.
func multisampled(t T, env *Env) {
	const size = 32
	tri := []vertex{{-0.9, -0.9, 0, white}, {0.9, -0.5, 0, white}, {-0.3, 0.9, 0, white}}
	vb := env.buffer(t, explicit.BufferVertex, 48, vertexBytes(tri))
	rgba := explicit.TextureFormatRGBA8Unorm
	ms := env.texture(t, explicit.TextureDesc{Width: size, Height: size, Format: rgba, SampleCount: 4, Usage: explicit.TextureRenderTarget})
	resolved := env.colorTarget(t, size, size, rgba)
	single := env.colorTarget(t, size, size, rgba)
	p4 := env.triPipeline(t, explicit.PipelineDesc{ColorTargets: []explicit.ColorTargetState{{Format: rgba}}, SampleCount: 4})
	p1 := env.triPipeline(t, explicit.PipelineDesc{ColorTargets: []explicit.ColorTargetState{{Format: rgba}}})
	enc := env.frame(t)
	black := [4]float64{0, 0, 0, 1}
	pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: ms, Resolve: resolved, Clear: black, Store: explicit.StoreDiscard}}})
	pass.SetPipeline(p4)
	pass.SetVertexBuffer(0, vb, 0)
	pass.Draw(3, 1, 0, 0)
	pass.End()
	pass = enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: single, Clear: black}}})
	pass.SetPipeline(p1)
	pass.SetVertexBuffer(0, vb, 0)
	pass.Draw(3, 1, 0, 0)
	pass.End()
	msRead := env.read(enc, resolved, image.Rect(0, 0, size, size))
	singleRead := env.read(enc, single, image.Rect(0, 0, size, size))
	env.submit(t, enc, msRead, singleRead)
	if !env.Pixels {
		return
	}
	// The coverage of a pixel, from a grid of 16 x 16 points inside it. The triangle is in clip
	// space (y up) and the pixel row 0 is the top.
	coverage := func(px, py int) float64 {
		inside := 0
		for sy := 0; sy < 16; sy++ {
			for sx := 0; sx < 16; sx++ {
				x := (float64(px) + (float64(sx)+0.5)/16) / size * 2
				y := 1 - (float64(py)+(float64(sy)+0.5)/16)/size*2
				if inTriangle(x-1, y, [3][2]float64{{-0.9, -0.9}, {0.9, -0.5}, {-0.3, 0.9}}) {
					inside++
				}
			}
		}
		return float64(inside) / 256
	}
	partial, hard := 0, 0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			want := coverage(x, y)
			got := float64(msRead.pixel(x, y, 4)[0]) / 255
			if math.Abs(got-want) > 0.35 {
				t.Errorf("pixel (%d,%d): 4 samples give %.2f, the coverage is %.2f", x, y, got, want)
			}
			if got > 0.05 && got < 0.95 {
				partial++
			}
			if s := singleRead.pixel(x, y, 4)[0]; s != 0 && s != 255 {
				hard++
			}
		}
	}
	if partial < 10 {
		t.Errorf("only %d pixels have an edge value with 4 samples, want the whole edge", partial)
	}
	if hard != 0 {
		t.Errorf("%d pixels of the triangle drawn with one sample are not 0 or 255", hard)
	}
}

// inTriangle reports whether the point is inside the triangle (either winding).
func inTriangle(x, y float64, v [3][2]float64) bool {
	sign := func(a, b [2]float64) float64 { return (x-b[0])*(a[1]-b[1]) - (a[0]-b[0])*(y-b[1]) }
	d1, d2, d3 := sign(v[0], v[1]), sign(v[1], v[2]), sign(v[2], v[0])
	neg := d1 < 0 || d2 < 0 || d3 < 0
	pos := d1 > 0 || d2 > 0 || d3 > 0
	return !neg || !pos
}

// ---------------------------------------------------------------------------------------------
// Texture kinds

var faceColors = [][4]byte{
	{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {255, 255, 0, 255}, {0, 255, 255, 255}, {255, 0, 255, 255},
}

func clearOf(c [4]byte) [4]float64 {
	return [4]float64{float64(c[0]) / 255, float64(c[1]) / 255, float64(c[2]) / 255, float64(c[3]) / 255}
}

// params writes the sampling parameters of a scene, one per 256 bytes (the dynamic offsets).
func params(values ...[4]float32) []byte {
	out := make([]byte, 256*len(values))
	for i, v := range values {
		for c, f := range v {
			binary.LittleEndian.PutUint32(out[i*256+c*4:], math.Float32bits(f))
		}
	}
	return out
}

// samplePipeline is a pipeline of sample.wgsl with the fragment entry.
func (e *Env) samplePipeline(t T, fragment string) explicit.Pipeline {
	t.Helper()
	return e.pipeline(t, explicit.PipelineDesc{Shader: ShaderSample, VertexEntry: "vs_uv", FragmentEntry: fragment,
		ColorTargets: []explicit.ColorTargetState{{Format: explicit.TextureFormatRGBA8Unorm}}})
}

// sampleEach draws a triangle for each parameter with the pipeline, reading the bindings of group 1
// from the entries, into a target of its own, and returns what each target holds.
func (e *Env) sampleEach(t T, p explicit.Pipeline, entries []explicit.Binding, values ...[4]float32) [][]byte {
	t.Helper()
	ub := e.buffer(t, explicit.BufferUniform, 256*len(values), params(values...))
	enc := e.frame(t)
	g0, err := e.Dev.NewBindGroup(p, 0, []explicit.Binding{{Binding: 0, Buffer: ub, Size: 16}})
	if err != nil {
		t.Fatalf("NewBindGroup 0: %v", err)
	}
	g1, err := e.Dev.NewBindGroup(p, 1, entries)
	if err != nil {
		t.Fatalf("NewBindGroup 1: %v", err)
	}
	var reads []*read
	for i := range values {
		target := e.colorTarget(t, 4, 4, explicit.TextureFormatRGBA8Unorm)
		pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: target}}})
		pass.SetPipeline(p)
		pass.SetBindGroup(0, g0, uint32(i*256))
		pass.SetBindGroup(1, g1)
		pass.Draw(3, 1, 0, 0)
		pass.End()
		reads = append(reads, e.read(enc, target, image.Rect(0, 0, 4, 4)))
	}
	e.submit(t, enc, reads...)
	out := make([][]byte, len(reads))
	for i, r := range reads {
		if e.Pixels {
			out[i] = append([]byte(nil), r.pixel(1, 2, 4)...)
		}
	}
	return out
}

// cubeFaces draws into each face of a cube map and reads it, then samples the cube by direction.
func cubeFaces(t T, env *Env) {
	rgba := explicit.TextureFormatRGBA8Unorm
	cube := env.texture(t, explicit.TextureDesc{Kind: explicit.TextureKindCube, Width: 8, Height: 8, Format: rgba, MipLevels: 2,
		Usage: explicit.TextureRenderTarget | explicit.TextureSampled | explicit.TextureCopySource})
	if got := cube.Desc(); got.DepthOrLayers != 6 || got.Kind != explicit.TextureKindCube {
		t.Fatalf("Desc = %+v", got)
	}
	enc := env.frame(t)
	var reads []*read
	for f := 0; f < 6; f++ {
		enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: cube.Layer(0, f), Clear: clearOf(faceColors[f])}}}).End()
		reads = append(reads, env.read(enc, cube.Layer(0, f), image.Rect(0, 0, 8, 8)))
	}
	env.submit(t, enc, reads...)
	if env.Pixels {
		for f, r := range reads {
			for _, px := range [][2]int{{0, 0}, {4, 4}, {7, 7}} {
				if got := r.pixel(px[0], px[1], 4); !near(got, faceColors[f][:], 0) {
					t.Errorf("face %d pixel %v = %v, want %v", f, px, got, faceColors[f])
				}
			}
		}
	}

	p := env.samplePipeline(t, "fs_cube")
	smp := env.sampler(t, explicit.SamplerDesc{MinFilter: explicit.FilterNearest, MagFilter: explicit.FilterNearest})
	dirs := [][4]float32{{1, 0, 0, 0}, {-1, 0, 0, 0}, {0, 1, 0, 0}, {0, -1, 0, 0}, {0, 0, 1, 0}, {0, 0, -1, 0}}
	got := env.sampleEach(t, p, []explicit.Binding{{Binding: 0, Texture: cube}, {Binding: 3, Sampler: smp}}, dirs...)
	if env.Pixels {
		for f := range dirs {
			if !near(got[f], faceColors[f][:], 0) {
				t.Errorf("the direction %v samples %v, want the face %d: %v", dirs[f][:3], got[f], f, faceColors[f])
			}
		}
	}
}

// arrayLayers draws into each layer of a 2D array, reads it, and samples the array by layer.
func arrayLayers(t T, env *Env) {
	rgba := explicit.TextureFormatRGBA8Unorm
	const layers = 3
	array := env.texture(t, explicit.TextureDesc{Kind: explicit.TextureKind2DArray, Width: 4, Height: 4, DepthOrLayers: layers, Format: rgba,
		Usage: explicit.TextureRenderTarget | explicit.TextureSampled | explicit.TextureCopySource})
	enc := env.frame(t)
	var reads []*read
	for l := 0; l < layers; l++ {
		enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: array.Layer(0, l), Clear: clearOf(faceColors[l+1])}}}).End()
		reads = append(reads, env.read(enc, array.Layer(0, l), image.Rect(0, 0, 4, 4)))
	}
	env.submit(t, enc, reads...)
	if env.Pixels {
		for l, r := range reads {
			if got := r.pixel(2, 2, 4); !near(got, faceColors[l+1][:], 0) {
				t.Errorf("layer %d reads %v, want %v", l, got, faceColors[l+1])
			}
		}
	}
	p := env.samplePipeline(t, "fs_array")
	smp := env.sampler(t, explicit.SamplerDesc{MinFilter: explicit.FilterNearest, MagFilter: explicit.FilterNearest})
	got := env.sampleEach(t, p, []explicit.Binding{{Binding: 1, Texture: array}, {Binding: 3, Sampler: smp}},
		[4]float32{0, 0, 0, 0}, [4]float32{0, 0, 1, 0}, [4]float32{0, 0, 2, 0})
	if env.Pixels {
		for l := 0; l < layers; l++ {
			if !near(got[l], faceColors[l+1][:], 0) {
				t.Errorf("sampling layer %d gives %v, want %v", l, got[l], faceColors[l+1])
			}
		}
	}
}

// volumeSampled writes the slices of a 3D texture and samples it between and across slices.
func volumeSampled(t T, env *Env) {
	rgba := explicit.TextureFormatRGBA8Unorm
	const n = 4
	vol := env.texture(t, explicit.TextureDesc{Kind: explicit.TextureKind3D, Width: n, Height: n, DepthOrLayers: n, Format: rgba, Usage: explicit.TextureSampled})
	data := make([]byte, 0, n*n*n*4)
	slice := func(z int) [4]byte { return [4]byte{byte(z * 80), byte(255 - z*80), 0, 255} }
	for z := 0; z < n; z++ {
		for i := 0; i < n*n; i++ {
			c := slice(z)
			data = append(data, c[:]...)
		}
	}
	env.Dev.WriteTextureRegion(vol, explicit.TextureRegion{Size: [3]int{n, n, n}}, data, n*4, n)
	p := env.samplePipeline(t, "fs_volume")
	nearest := env.sampler(t, explicit.SamplerDesc{MinFilter: explicit.FilterNearest, MagFilter: explicit.FilterNearest})
	linear := env.sampler(t, explicit.SamplerDesc{MinFilter: explicit.FilterLinear, MagFilter: explicit.FilterLinear})
	var depths [][4]float32
	for z := 0; z < n; z++ {
		depths = append(depths, [4]float32{0, 0, (float32(z) + 0.5) / n, 0})
	}
	got := env.sampleEach(t, p, []explicit.Binding{{Binding: 2, Texture: vol}, {Binding: 3, Sampler: nearest}}, depths...)
	if env.Pixels {
		for z := 0; z < n; z++ {
			if c := slice(z); !near(got[z], c[:], 0) {
				t.Errorf("nearest at slice %d gives %v, want %v", z, got[z], c)
			}
		}
	}
	between := env.sampleEach(t, p, []explicit.Binding{{Binding: 2, Texture: vol}, {Binding: 3, Sampler: linear}}, [4]float32{0, 0, 0.25, 0})
	if env.Pixels {
		a, b := slice(0), slice(1)
		want := []byte{byte((int(a[0]) + int(b[0]) + 1) / 2), byte((int(a[1]) + int(b[1]) + 1) / 2), 0, 255}
		if !near(between[0], want, 2) {
			t.Errorf("linear between slices 0 and 1 gives %v, want %v", between[0], want)
		}
	}
}

// unfilterableTextures reads integer textures and float32 textures with textureLoad.
func unfilterableTextures(t T, env *Env) {
	rgba := explicit.TextureFormatRGBA8Unorm
	uints := env.texture(t, explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatR32Uint, Usage: explicit.TextureSampled})
	data := make([]byte, 0, 64)
	for i := 0; i < 16; i++ {
		data = binary.LittleEndian.AppendUint32(data, uint32(10*i))
	}
	env.Dev.WriteTexture(uints, 0, image.Rect(0, 0, 4, 4), data, 16)
	floats := env.texture(t, explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA32Float, Usage: explicit.TextureSampled})
	fdata := make([]byte, 0, 256)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			for _, v := range []float32{float32(x) / 4, float32(y) / 4, 0.5, 1} {
				fdata = binary.LittleEndian.AppendUint32(fdata, math.Float32bits(v))
			}
		}
	}
	env.Dev.WriteTexture(floats, 0, image.Rect(0, 0, 4, 4), fdata, 64)

	for _, tt := range []struct {
		name     string
		fragment string
		entries  []explicit.Binding
		want     func(x, y int) [4]byte
		tol      int
	}{
		{"r32uint with textureLoad", "fs_uint", []explicit.Binding{{Binding: 4, Texture: uints}},
			func(x, y int) [4]byte { return [4]byte{byte(10 * (y*4 + x)), 0, 0, 255} }, 1},
		{"rgba32float with textureLoad", "fs_float", []explicit.Binding{{Binding: 5, Texture: floats}},
			func(x, y int) [4]byte { return [4]byte{byte(x * 64), byte(y * 64), 128, 255} }, 1},
	} {
		t.Run(tt.name, func(t T) {
			p := env.samplePipeline(t, tt.fragment)
			g1, err := env.Dev.NewBindGroup(p, 1, tt.entries)
			if err != nil {
				t.Fatalf("%v", err)
			}
			target := env.colorTarget(t, 4, 4, rgba)
			enc := env.frame(t)
			pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: target}}})
			pass.SetPipeline(p)
			pass.SetBindGroup(1, g1)
			pass.Draw(3, 1, 0, 0)
			pass.End()
			res := env.read(enc, target, image.Rect(0, 0, 4, 4))
			env.submit(t, enc, res)
			if !env.Pixels {
				return
			}
			for y := 0; y < 4; y++ {
				for x := 0; x < 4; x++ {
					want := tt.want(x, y)
					if got := res.pixel(x, y, 4); !near(got, want[:], tt.tol) {
						t.Errorf("pixel (%d,%d) = %v, want %v", x, y, got, want)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------------------------
// Passes and pipeline state

// fourAttachments draws into four color attachments of different formats, one with blending.
func fourAttachments(t T, env *Env) {
	const size = 8
	rgba := explicit.TextureFormatRGBA8Unorm
	a := env.colorTarget(t, size, size, rgba)
	b := env.colorTarget(t, size, size, rgba)
	c := env.colorTarget(t, size, size, explicit.TextureFormatR32Uint)
	d := env.colorTarget(t, size, size, explicit.TextureFormatRGBA16Float)
	p := env.fillPipeline(t, "fs_four",
		explicit.ColorTargetState{Format: rgba},
		explicit.ColorTargetState{Format: rgba, Blend: explicit.BlendState{Enabled: true, SrcFactor: explicit.BlendSrcAlpha, DstFactor: explicit.BlendOneMinusSrcAlpha}},
		explicit.ColorTargetState{Format: explicit.TextureFormatR32Uint},
		explicit.ColorTargetState{Format: explicit.TextureFormatRGBA16Float})
	enc := env.frame(t)
	pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{
		{Target: a}, {Target: b, Clear: [4]float64{1, 1, 1, 1}}, {Target: c}, {Target: d},
	}})
	pass.SetPipeline(p)
	pass.Draw(3, 1, 0, 0)
	pass.End()
	rect := image.Rect(0, 0, size, size)
	ra, rb, rc, rd := env.read(enc, a, rect), env.read(enc, b, rect), env.read(enc, c, rect), env.read(enc, d, rect)
	env.submit(t, enc, ra, rb, rc, rd)
	if !env.Pixels {
		return
	}
	f16 := func(vs ...float32) []byte {
		var out []byte
		for _, v := range vs {
			out = binary.LittleEndian.AppendUint16(out, explicit.Float16Bits(v))
		}
		return out
	}
	for _, px := range [][2]int{{0, 0}, {4, 3}, {7, 7}} {
		if got := ra.pixel(px[0], px[1], 4); !near(got, []byte{255, 0, 0, 128}, 1) {
			t.Errorf("attachment 0 at %v = %v, want 255 0 0 128", px, got)
		}
		if got := rb.pixel(px[0], px[1], 4); !near(got, []byte{255, 128, 128, 191}, 2) {
			t.Errorf("attachment 1 (blended) at %v = %v, want 255 128 128 191", px, got)
		}
		if got := rc.pixel(px[0], px[1], 4); !near(got, []byte{42, 0, 0, 0}, 0) {
			t.Errorf("attachment 2 at %v = %v, want 42", px, got)
		}
		if got := rd.pixel(px[0], px[1], 8); !near(got, f16(0.5, 1.5, 2.5, 3.5), 0) {
			t.Errorf("attachment 3 at %v = %v, want %v", px, got, f16(0.5, 1.5, 2.5, 3.5))
		}
	}
}

// clipSpaceAndCulling checks that a triangle with y > 0 is in the top half, and the cull modes and
// front faces against the winding.
func clipSpaceAndCulling(t T, env *Env) {
	const size = 16
	rgba := explicit.TextureFormatRGBA8Unorm
	ccw := []vertex{{-0.8, -0.8, 0, red}, {0.8, -0.8, 0, red}, {0, 0.8, 0, red}}
	cw := []vertex{ccw[0], ccw[2], ccw[1]}
	top := []vertex{{-0.5, 0.1, 0, green}, {0.5, 0.1, 0, green}, {0, 0.9, 0, green}}
	verts := append(append(append([]vertex(nil), ccw...), cw...), top...)
	vb := env.buffer(t, explicit.BufferVertex, len(verts)*16, vertexBytes(verts))

	draw := func(t T, p explicit.Pipeline, first int) *read {
		target := env.colorTarget(t, size, size, rgba)
		enc := env.frame(t)
		pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: target}}})
		pass.SetPipeline(p)
		pass.SetVertexBuffer(0, vb, 0)
		pass.Draw(3, 1, first, 0)
		pass.End()
		res := env.read(enc, target, image.Rect(0, 0, size, size))
		env.submit(t, enc, res)
		return res
	}
	t.Run("a triangle with y above 0 is in the top half", func(t T) {
		res := draw(t, env.triPipeline(t, explicit.PipelineDesc{ColorTargets: []explicit.ColorTargetState{{Format: rgba}}}), 6)
		if !env.Pixels {
			return
		}
		inTop, inBottom := 0, 0
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if res.pixel(x, y, 4)[1] == 255 {
					if y < size/2 {
						inTop++
					} else {
						inBottom++
					}
				}
			}
		}
		if inTop < 10 || inBottom != 0 {
			t.Errorf("%d pixels in the top half and %d in the bottom half, want the triangle in the top half only", inTop, inBottom)
		}
		if got := res.pixel(8, 4, 4); !near(got, green[:], 0) {
			t.Errorf("pixel (8,4) = %v, want green", got)
		}
	})
	for _, tt := range []struct {
		name    string
		cull    explicit.CullMode
		front   explicit.FrontFace
		first   int
		visible bool
	}{
		{"counter-clockwise, no culling", explicit.CullNone, explicit.FrontFaceCCW, 0, true},
		{"clockwise, no culling", explicit.CullNone, explicit.FrontFaceCCW, 3, true},
		{"counter-clockwise, back culled", explicit.CullBack, explicit.FrontFaceCCW, 0, true},
		{"clockwise, back culled", explicit.CullBack, explicit.FrontFaceCCW, 3, false},
		{"counter-clockwise, front culled", explicit.CullFront, explicit.FrontFaceCCW, 0, false},
		{"clockwise, front culled", explicit.CullFront, explicit.FrontFaceCCW, 3, true},
		{"counter-clockwise, back culled, clockwise is front", explicit.CullBack, explicit.FrontFaceCW, 0, false},
		{"clockwise, back culled, clockwise is front", explicit.CullBack, explicit.FrontFaceCW, 3, true},
	} {
		t.Run(tt.name, func(t T) {
			res := draw(t, env.triPipeline(t, explicit.PipelineDesc{ColorTargets: []explicit.ColorTargetState{{Format: rgba}}, CullMode: tt.cull, FrontFace: tt.front}), tt.first)
			if !env.Pixels {
				return
			}
			want := []byte{0, 0, 0, 0}
			if tt.visible {
				want = red[:]
			}
			if got := res.pixel(8, 6, 4); !near(got, want, 0) {
				t.Errorf("pixel (8,6) = %v, want %v", got, want)
			}
		})
	}
}

// topologies draws with the strip topologies, lines and points.
func topologies(t T, env *Env) {
	const size = 16
	rgba := explicit.TextureFormatRGBA8Unorm
	draw := func(t T, topology explicit.PrimitiveTopology, verts []vertex) *read {
		vb := env.buffer(t, explicit.BufferVertex, len(verts)*16, vertexBytes(verts))
		p := env.triPipeline(t, explicit.PipelineDesc{ColorTargets: []explicit.ColorTargetState{{Format: rgba}}, Topology: topology})
		target := env.colorTarget(t, size, size, rgba)
		enc := env.frame(t)
		pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: target}}})
		pass.SetPipeline(p)
		pass.SetVertexBuffer(0, vb, 0)
		pass.Draw(len(verts), 1, 0, 0)
		pass.End()
		res := env.read(enc, target, image.Rect(0, 0, size, size))
		env.submit(t, enc, res)
		return res
	}
	count := func(r *read) (n int) {
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if r.pixel(x, y, 4)[3] != 0 {
					n++
				}
			}
		}
		return n
	}
	t.Run("a triangle strip is a quad", func(t T) {
		res := draw(t, explicit.TopologyTriangleStrip, []vertex{{-0.5, -0.5, 0, red}, {0.5, -0.5, 0, red}, {-0.5, 0.5, 0, red}, {0.5, 0.5, 0, red}})
		if env.Pixels && count(res) != 8*8 {
			t.Errorf("the quad has %d pixels, want 64", count(res))
		}
	})
	t.Run("a line list", func(t T) {
		res := draw(t, explicit.TopologyLineList, []vertex{{-0.9, 0.1, 0, red}, {0.9, 0.1, 0, red}, {-0.9, -0.5, 0, green}, {0.9, -0.5, 0, green}})
		if env.Pixels {
			if n := count(res); n < 24 || n > 40 {
				t.Errorf("two lines have %d pixels, want about 30", n)
			}
			if got := res.pixel(8, 7, 4); got[0] != 255 && res.pixel(8, 6, 4)[0] != 255 {
				t.Errorf("no red pixel on the first line near (8,7): %v", got)
			}
		}
	})
	t.Run("a line strip", func(t T) {
		res := draw(t, explicit.TopologyLineStrip, []vertex{{-0.9, -0.9, 0, red}, {0.9, -0.9, 0, red}, {0.9, 0.9, 0, red}})
		if env.Pixels {
			if n := count(res); n < 24 || n > 40 {
				t.Errorf("two connected lines have %d pixels, want about 30", n)
			}
		}
	})
	t.Run("a point list", func(t T) {
		res := draw(t, explicit.TopologyPointList, []vertex{{0.0625, 0.0625, 0, red}, {-0.4375, -0.4375, 0, green}})
		if env.Pixels {
			if got := res.pixel(8, 7, 4); !near(got, red[:], 0) {
				t.Errorf("the pixel of the first point (8,7) = %v, want red", got)
			}
			if got := res.pixel(4, 11, 4); !near(got, green[:], 0) {
				t.Errorf("the pixel of the second point (4,11) = %v, want green", got)
			}
			if n := count(res); n != 2 {
				t.Errorf("two points have %d pixels, want 2", n)
			}
		}
	})
}

// stencilAndWriteMask marks half of the target in the stencil without writing color, then draws
// where the stencil is set; and draws with some channels masked.
func stencilAndWriteMask(t T, env *Env) {
	const size = 16
	rgba := explicit.TextureFormatRGBA8Unorm
	ds := explicit.TextureFormatDepth24PlusStencil8
	verts := append([]vertex{{-1, -1, 0, red}, {0, -1, 0, red}, {-1, 1, 0, red}, {0, 1, 0, red}}, bigTriangle(0, red)...)
	vb := env.buffer(t, explicit.BufferVertex, len(verts)*16, vertexBytes(verts))
	mark := explicit.StencilFaceState{Compare: explicit.CompareAlways, PassOp: explicit.StencilReplace}
	test := explicit.StencilFaceState{Compare: explicit.CompareEqual}
	pMark := env.triPipeline(t, explicit.PipelineDesc{
		ColorTargets: []explicit.ColorTargetState{{Format: rgba, WriteMask: explicit.ColorWriteNone}}, Topology: explicit.TopologyTriangleStrip,
		DepthStencil: explicit.DepthStencilState{Format: ds, StencilFront: mark, StencilBack: mark},
	})
	pTest := env.triPipeline(t, explicit.PipelineDesc{
		ColorTargets: []explicit.ColorTargetState{{Format: rgba}},
		DepthStencil: explicit.DepthStencilState{Format: ds, StencilFront: test, StencilBack: test},
	})
	color := env.colorTarget(t, size, size, rgba)
	depth := env.texture(t, explicit.TextureDesc{Width: size, Height: size, Format: ds, Usage: explicit.TextureRenderTarget})
	enc := env.frame(t)
	pass := enc.BeginRenderPass(explicit.PassDesc{
		Color: []explicit.ColorAttachment{{Target: color}},
		Depth: &explicit.DepthAttachment{Target: depth, ClearDepth: 1, ClearStencil: 0},
	})
	pass.SetStencilReference(1)
	pass.SetPipeline(pMark)
	pass.SetVertexBuffer(0, vb, 0)
	pass.Draw(4, 1, 0, 0)
	pass.SetPipeline(pTest)
	pass.Draw(3, 1, 4, 0)
	pass.End()
	res := env.read(enc, color, image.Rect(0, 0, size, size))

	// Only the red and alpha channels of a white triangle are written over green.
	masked := env.triPipeline(t, explicit.PipelineDesc{ColorTargets: []explicit.ColorTargetState{{Format: rgba, WriteMask: explicit.ColorWriteRed | explicit.ColorWriteAlpha}}})
	whites := bigTriangle(0, [4]byte{255, 0, 255, 255})
	wb := env.buffer(t, explicit.BufferVertex, 48, vertexBytes(whites))
	maskTarget := env.colorTarget(t, size, size, rgba)
	pass = enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: maskTarget, Clear: [4]float64{0, 1, 0, 0}}}})
	pass.SetPipeline(masked)
	pass.SetVertexBuffer(0, wb, 0)
	pass.Draw(3, 1, 0, 0)
	pass.End()
	maskRes := env.read(enc, maskTarget, image.Rect(0, 0, size, size))
	env.submit(t, enc, res, maskRes)
	if !env.Pixels {
		return
	}
	if got := res.pixel(3, 8, 4); !near(got, red[:], 0) {
		t.Errorf("where the stencil is 1, pixel (3,8) = %v, want red", got)
	}
	if got := res.pixel(12, 8, 4); !near(got, []byte{0, 0, 0, 0}, 0) {
		t.Errorf("where the stencil is 0, pixel (12,8) = %v, want clear", got)
	}
	if got := maskRes.pixel(5, 5, 4); !near(got, []byte{255, 255, 0, 255}, 0) {
		t.Errorf("with the write mask red and alpha, pixel (5,5) = %v, want 255 255 0 255", got)
	}
}

// vertexFormats reads one attribute of several formats from a vertex buffer.
func vertexFormats(t T, env *Env) {
	const size = 4
	rgba := explicit.TextureFormatRGBA8Unorm
	layout := explicit.VertexBufferLayout{Stride: 20, Attributes: []explicit.VertexAttribute{
		{Location: 0, Format: explicit.VertexSnorm16x2, Offset: 0},
		{Location: 1, Format: explicit.VertexUnorm8x4, Offset: 4},
		{Location: 2, Format: explicit.VertexUint32, Offset: 8},
		{Location: 3, Format: explicit.VertexFloat16x2, Offset: 12},
		{Location: 4, Format: explicit.VertexUint8x4, Offset: 16},
	}}
	var data []byte
	for _, c := range [][2]int16{{-32767, -32767}, {32767, -32767}, {-32767, 32767}, {32767, 32767}} {
		data = binary.LittleEndian.AppendUint16(data, uint16(c[0]))
		data = binary.LittleEndian.AppendUint16(data, uint16(c[1]))
		data = append(data, 200, 0, 0, 0)
		data = binary.LittleEndian.AppendUint32(data, 77)
		data = binary.LittleEndian.AppendUint16(data, explicit.Float16Bits(0.5))
		data = binary.LittleEndian.AppendUint16(data, explicit.Float16Bits(0))
		data = append(data, 133, 0, 0, 0)
	}
	vb := env.buffer(t, explicit.BufferVertex, len(data), data)
	p := env.pipeline(t, explicit.PipelineDesc{Shader: ShaderVformats, VertexEntry: "vs_formats", FragmentEntry: "fs_formats",
		VertexBuffers: []explicit.VertexBufferLayout{layout}, Topology: explicit.TopologyTriangleStrip,
		ColorTargets: []explicit.ColorTargetState{{Format: rgba}}})
	target := env.colorTarget(t, size, size, rgba)
	enc := env.frame(t)
	pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: target}}})
	pass.SetPipeline(p)
	pass.SetVertexBuffer(0, vb, 0)
	pass.Draw(4, 1, 0, 0)
	pass.End()
	res := env.read(enc, target, image.Rect(0, 0, size, size))
	env.submit(t, enc, res)
	if !env.Pixels {
		return
	}
	for _, px := range [][2]int{{0, 0}, {2, 1}, {3, 3}} {
		if got := res.pixel(px[0], px[1], 4); !near(got, []byte{200, 77, 128, 133}, 1) {
			t.Errorf("pixel %v = %v, want 200 77 128 133", px, got)
		}
	}
}
