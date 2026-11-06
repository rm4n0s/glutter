package devicetest

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"runtime"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// The scenes of step 30.2 (SPEC §27.1, §27.12): compute, storage resources, indirect calls, the copies,
// the read of a buffer, several encoders, instances and timestamps.

// squares is what cs_square makes of the numbers: their squares, in 32 bits.
func squares(in []uint32) []uint32 {
	out := make([]uint32, len(in))
	for i, v := range in {
		out[i] = v * v
	}
	return out
}

// firstDifference describes the first element where two lists differ, or "" when they are the same.
func firstDifference(got, want []uint32) string {
	if len(got) != len(want) {
		return fmt.Sprintf("%d numbers, want %d", len(got), len(want))
	}
	bad, first := 0, -1
	for i := range got {
		if got[i] != want[i] {
			if first < 0 {
				first = i
			}
			bad++
		}
	}
	if bad == 0 {
		return ""
	}
	return fmt.Sprintf("%d of %d numbers differ, the first at %d: got %d, want %d", bad, len(got), first, got[first], want[first])
}

// squareGroup binds the buffers of cs_square: the parameters, the input and the output.
func (e *Env) squareGroup(t T, p explicit.ComputePipeline, params, input, output explicit.Buffer) explicit.BindGroup {
	t.Helper()
	return e.computeGroup(t, p, 0,
		explicit.Binding{Binding: 0, Buffer: params, Size: 16},
		explicit.Binding{Binding: 1, Buffer: input},
		explicit.Binding{Binding: 2, Buffer: output})
}

// computeSquares squares a million numbers on the GPU and reads them back exactly.
func computeSquares(t T, env *Env) {
	const n = 1_000_000
	in := make([]uint32, n)
	for i := range in {
		in[i] = uint32(i) * 2654435761
	}
	input := env.buffer(t, explicit.BufferStorage, n*4, u32s(in...))
	output := env.buffer(t, explicit.BufferStorage|explicit.BufferCopySource, n*4, make([]byte, n*4))
	ub := env.buffer(t, explicit.BufferUniform, 256, u32s(n, 0, 0, 0))
	p := env.computePipeline(t, "cs_square")
	enc := env.frame(t)
	g := env.squareGroup(t, p, ub, input, output)
	pass := enc.BeginComputePass(explicit.ComputePassDesc{})
	pass.SetPipeline(p)
	pass.SetBindGroup(0, g, 0)
	pass.Dispatch((n+63)/64, 1, 1)
	pass.End()
	res := env.readBuffer(enc, output, 0, n*4)
	env.submit(t, enc, res)
	if !env.Pixels {
		return
	}
	if d := firstDifference(toU32s(res.data), squares(in)); d != "" {
		t.Errorf("the squares read back wrong: %s", d)
	}
}

// storageTexture paints a storage texture with a compute shader and samples it in a render pass of the
// same encoder, and reads it directly.
func storageTexture(t T, env *Env) {
	rgba := explicit.TextureFormatRGBA8Unorm
	tex := env.texture(t, explicit.TextureDesc{Width: 4, Height: 4, Format: rgba,
		Usage: explicit.TextureStorage | explicit.TextureSampled | explicit.TextureCopySource})
	target := env.colorTarget(t, 4, 4, rgba)
	cp := env.computePipeline(t, "cs_paint")
	sp := env.samplePipeline(t, "fs_float")
	enc := env.frame(t)
	g := env.computeGroup(t, cp, 1, explicit.Binding{Binding: 0, Texture: tex})
	sg, err := env.Dev.NewBindGroup(sp, 1, []explicit.Binding{{Binding: 5, Texture: tex}})
	if err != nil {
		t.Fatalf("NewBindGroup: %v", err)
	}
	env.cleanup = append(env.cleanup, sg.Destroy)
	cpass := enc.BeginComputePass(explicit.ComputePassDesc{})
	cpass.SetPipeline(cp)
	cpass.SetBindGroup(1, g)
	cpass.Dispatch(2, 2, 1)
	cpass.End()
	pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: target}}})
	pass.SetPipeline(sp)
	pass.SetBindGroup(1, sg)
	pass.Draw(3, 1, 0, 0)
	pass.End()
	sampled := env.read(enc, target, image.Rect(0, 0, 4, 4))
	direct := env.read(enc, tex, image.Rect(0, 0, 4, 4))
	env.submit(t, enc, sampled, direct)
	if !env.Pixels {
		return
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			want := []byte{byte(x * 60), byte(y * 60), 128, 255}
			if got := sampled.pixel(x, y, 4); !near(got, want, 1) {
				t.Errorf("the render pass reads %v at (%d,%d) of the storage texture, want %v", got, x, y, want)
			}
			if got := direct.pixel(x, y, 4); !near(got, want, 1) {
				t.Errorf("the storage texture reads back %v at (%d,%d), want %v", got, x, y, want)
			}
		}
	}
}

// indirectCalls draws and dispatches with arguments that a compute shader wrote, at an offset of the
// buffer.
func indirectCalls(t T, env *Env) {
	rgba := explicit.TextureFormatRGBA8Unorm
	const argsOffset = 256
	args := env.buffer(t, explicit.BufferStorage|explicit.BufferIndirect, 512, make([]byte, 512))
	argsBinding := explicit.Binding{Binding: 1, Buffer: args, Offset: argsOffset, Size: 256}

	// check compares the corners of a target with red.
	check := func(t T, what string, res *read) {
		if !env.Pixels {
			return
		}
		for _, p := range [][2]int{{0, 0}, {3, 0}, {0, 3}, {3, 3}} {
			if got := res.pixel(p[0], p[1], 4); !near(got, red[:], 0) {
				t.Errorf("%s: pixel (%d,%d) is %v, want red", what, p[0], p[1], got)
			}
		}
	}

	t.Run("DrawIndirect", func(t T) {
		target := env.colorTarget(t, 4, 4, rgba)
		vb := env.buffer(t, explicit.BufferVertex, 48, vertexBytes(bigTriangle(0, red)))
		p := env.triPipeline(t, explicit.PipelineDesc{ColorTargets: []explicit.ColorTargetState{{Format: rgba}}})
		cp := env.computePipeline(t, "cs_draw_args")
		enc := env.frame(t)
		g := env.computeGroup(t, cp, 1, argsBinding)
		cpass := enc.BeginComputePass(explicit.ComputePassDesc{})
		cpass.SetPipeline(cp)
		cpass.SetBindGroup(1, g)
		cpass.Dispatch(1, 1, 1)
		cpass.End()
		pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: target, Clear: clearOf(blue)}}})
		pass.SetPipeline(p)
		pass.SetVertexBuffer(0, vb, 0)
		pass.DrawIndirect(args, argsOffset)
		pass.End()
		res := env.read(enc, target, image.Rect(0, 0, 4, 4))
		env.submit(t, enc, res)
		check(t, "DrawIndirect", res)
	})

	t.Run("DrawIndexedIndirect", func(t T) {
		target := env.colorTarget(t, 4, 4, rgba)
		// Vertex 0 is green and off the triangle that the indices make.
		vertices := append([]vertex{{-1, -1, 0, green}}, bigTriangle(0, red)...)
		vb := env.buffer(t, explicit.BufferVertex, 64, vertexBytes(vertices))
		ib := env.buffer(t, explicit.BufferIndex, 12, u32s(1, 2, 3))
		p := env.triPipeline(t, explicit.PipelineDesc{ColorTargets: []explicit.ColorTargetState{{Format: rgba}}})
		cp := env.computePipeline(t, "cs_indexed_args")
		enc := env.frame(t)
		g := env.computeGroup(t, cp, 1, argsBinding)
		cpass := enc.BeginComputePass(explicit.ComputePassDesc{})
		cpass.SetPipeline(cp)
		cpass.SetBindGroup(1, g)
		cpass.Dispatch(1, 1, 1)
		cpass.End()
		pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: target, Clear: clearOf(blue)}}})
		pass.SetPipeline(p)
		pass.SetVertexBuffer(0, vb, 0)
		pass.SetIndexBuffer32(ib, 0)
		pass.DrawIndexedIndirect(args, argsOffset)
		pass.End()
		res := env.read(enc, target, image.Rect(0, 0, 4, 4))
		env.submit(t, enc, res)
		check(t, "DrawIndexedIndirect", res)
	})

	t.Run("DispatchIndirect", func(t T) {
		const groups, total = 100, 7000
		in := make([]uint32, total)
		for i := range in {
			in[i] = uint32(i + 1)
		}
		input := env.buffer(t, explicit.BufferStorage, total*4, u32s(in...))
		output := env.buffer(t, explicit.BufferStorage|explicit.BufferCopySource, total*4, make([]byte, total*4))
		// count is the number of elements the squares cover; a is the number of workgroups.
		ub := env.buffer(t, explicit.BufferUniform, 256, u32s(total, groups, 0, 0))
		ap := env.computePipeline(t, "cs_dispatch_args")
		sp := env.computePipeline(t, "cs_square")
		enc := env.frame(t)
		ag0 := env.computeGroup(t, ap, 0, explicit.Binding{Binding: 0, Buffer: ub, Size: 16})
		ag1 := env.computeGroup(t, ap, 1, argsBinding)
		sg := env.squareGroup(t, sp, ub, input, output)
		pass := enc.BeginComputePass(explicit.ComputePassDesc{})
		pass.SetPipeline(ap)
		pass.SetBindGroup(0, ag0, 0)
		pass.SetBindGroup(1, ag1)
		pass.Dispatch(1, 1, 1)
		pass.End()
		pass = enc.BeginComputePass(explicit.ComputePassDesc{})
		pass.SetPipeline(sp)
		pass.SetBindGroup(0, sg, 0)
		pass.DispatchIndirect(args, argsOffset)
		pass.End()
		res := env.readBuffer(enc, output, 0, total*4)
		env.submit(t, enc, res)
		if !env.Pixels {
			return
		}
		want := make([]uint32, total)
		copy(want, squares(in[:groups*64]))
		if d := firstDifference(toU32s(res.data), want); d != "" {
			t.Errorf("DispatchIndirect of %d workgroups: %s", groups, d)
		}
	})
}

// ---------------------------------------------------------------------------------------------
// Copies

// pattern returns n bytes that differ from byte to byte.
func pattern(n, seed int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(i*7 + seed*31 + 3)
	}
	return out
}

// layoutSize is the bytes of a buffer that hold a box of w×h×d pixels as the layout says, rounded up to 4.
func layoutSize(layout explicit.BufferLayout, w, h, d, bpp int) int {
	n := layout.Offset + layout.BytesPerRow*layout.RowsPerImage*(d-1) + layout.BytesPerRow*(h-1) + w*bpp
	return (n + 3) &^ 3 // a read is a multiple of 4 bytes
}

// pack lays the packed pixels of a box out in a buffer of the size as the layout says; the rest of the
// buffer is 0xEE, so that a copy that writes outside the box shows.
func pack(box []byte, w, h, d, bpp int, layout explicit.BufferLayout, size int) []byte {
	out := bytes.Repeat([]byte{0xEE}, size)
	for z := 0; z < d; z++ {
		for y := 0; y < h; y++ {
			dst := layout.Offset + z*layout.BytesPerRow*layout.RowsPerImage + y*layout.BytesPerRow
			copy(out[dst:dst+w*bpp], box[(z*h+y)*w*bpp:(z*h+y+1)*w*bpp])
		}
	}
	return out
}

// unpack is the opposite of pack.
func unpack(buf []byte, w, h, d, bpp int, layout explicit.BufferLayout) []byte {
	out := make([]byte, 0, w*h*d*bpp)
	for z := 0; z < d; z++ {
		for y := 0; y < h; y++ {
			src := layout.Offset + z*layout.BytesPerRow*layout.RowsPerImage + y*layout.BytesPerRow
			out = append(out, buf[src:src+w*bpp]...)
		}
	}
	return out
}

// copies runs each copy of the encoder: buffer to buffer, buffer to texture and texture to buffer, with
// rows that have padding at both ends.
func copies(t T, env *Env) {
	t.Run("buffer to buffer", func(t T) {
		src := env.buffer(t, explicit.BufferCopySource, 64, pattern(64, 1))
		dst := env.buffer(t, explicit.BufferCopyDestination|explicit.BufferCopySource, 64, make([]byte, 64))
		enc := env.frame(t)
		enc.CopyBuffer(src, 8, dst, 20, 36)
		res := env.readBuffer(enc, dst, 0, 64)
		part := env.readBuffer(enc, src, 4, 12)
		env.submit(t, enc, res, part)
		if !env.Pixels {
			return
		}
		want := make([]byte, 64)
		copy(want[20:56], pattern(64, 1)[8:44])
		if !bytes.Equal(res.data, want) {
			t.Errorf("the destination reads\n%v\nwant\n%v", res.data, want)
		}
		if !bytes.Equal(part.data, pattern(64, 1)[4:16]) {
			t.Errorf("a part of the source reads %v, want %v", part.data, pattern(64, 1)[4:16])
		}
	})

	for _, f := range []explicit.TextureFormat{
		explicit.TextureFormatRGBA8Unorm, explicit.TextureFormatBGRA8Unorm, explicit.TextureFormatR8Unorm,
		explicit.TextureFormatRGBA16Float, explicit.TextureFormatR32Uint, explicit.TextureFormatRGBA32Float,
	} {
		t.Run("buffer to texture and back "+formatName(f), func(t T) {
			const w, h = 9, 6
			tex := env.texture(t, explicit.TextureDesc{Width: w, Height: h, Format: f, Usage: explicit.TextureSampled | explicit.TextureCopySource})
			bpp := f.BytesPerPixel()
			box := formatBytes(f, 5*3)
			region := explicit.TextureRegion{Origin: [3]int{2, 1, 0}, Size: [3]int{5, 3, 1}}
			in := explicit.BufferLayout{Offset: max(16, bpp*2), BytesPerRow: 256, RowsPerImage: 4}
			out := explicit.BufferLayout{Offset: max(8, bpp), BytesPerRow: 512, RowsPerImage: 3}
			srcSize, dstSize := layoutSize(in, 5, 3, 1, bpp), layoutSize(out, 5, 3, 1, bpp)
			src := env.buffer(t, explicit.BufferCopySource, srcSize, pack(box, 5, 3, 1, bpp, in, srcSize))
			dst := env.buffer(t, explicit.BufferCopyDestination|explicit.BufferCopySource, dstSize, bytes.Repeat([]byte{0xEE}, dstSize))
			env.Dev.WriteTexture(tex, 0, image.Rect(0, 0, w, h), make([]byte, w*h*bpp), w*bpp)
			enc := env.frame(t)
			enc.CopyBufferToTexture(src, in, tex, region)
			enc.CopyTextureToBuffer(tex, region, dst, out)
			whole := env.read(enc, tex, image.Rect(0, 0, w, h))
			back := env.readBuffer(enc, dst, 0, dstSize)
			env.submit(t, enc, whole, back)
			if !env.Pixels {
				return
			}
			if got := unpack(back.data, 5, 3, 1, bpp, out); !bytes.Equal(got, box) {
				t.Errorf("the box reads back as\n%v\nwant\n%v", got, box)
			}
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					want := make([]byte, bpp)
					if x >= 2 && x < 7 && y >= 1 && y < 4 {
						i := ((y-1)*5 + (x - 2)) * bpp
						want = box[i : i+bpp]
					}
					if got := whole.pixel(x, y, bpp); !bytes.Equal(got, want) {
						t.Errorf("pixel (%d,%d) of the texture is %v, want %v", x, y, got, want)
					}
				}
			}
		})
	}

	for _, kind := range []explicit.TextureKind{explicit.TextureKind2DArray, explicit.TextureKind3D} {
		name := map[explicit.TextureKind]string{explicit.TextureKind2DArray: "layers of an array", explicit.TextureKind3D: "slices of a volume"}[kind]
		t.Run("buffer to texture and back "+name, func(t T) {
			const w, h, n = 6, 4, 3
			rgba := explicit.TextureFormatRGBA8Unorm
			tex := env.texture(t, explicit.TextureDesc{Kind: kind, Width: w, Height: h, DepthOrLayers: n, Format: rgba, Usage: explicit.TextureSampled | explicit.TextureCopySource})
			box := pattern(w*h*2*4, 5)
			region := explicit.TextureRegion{Origin: [3]int{0, 0, 1}, Size: [3]int{w, h, 2}}
			whole := explicit.TextureRegion{Size: [3]int{w, h, n}}
			in := explicit.BufferLayout{Offset: 16, BytesPerRow: 256, RowsPerImage: 5}
			out := explicit.BufferLayout{Offset: 0, BytesPerRow: 512, RowsPerImage: 4}
			srcSize, dstSize := layoutSize(in, w, h, 2, 4), layoutSize(out, w, h, n, 4)
			src := env.buffer(t, explicit.BufferCopySource, srcSize, pack(box, w, h, 2, 4, in, srcSize))
			dst := env.buffer(t, explicit.BufferCopyDestination|explicit.BufferCopySource, dstSize, bytes.Repeat([]byte{0xEE}, dstSize))
			env.Dev.WriteTextureRegion(tex, whole, make([]byte, w*h*n*4), w*4, h)
			enc := env.frame(t)
			enc.CopyBufferToTexture(src, in, tex, region)
			enc.CopyTextureToBuffer(tex, whole, dst, out)
			back := env.readBuffer(enc, dst, 0, dstSize)
			env.submit(t, enc, back)
			if !env.Pixels {
				return
			}
			want := append(make([]byte, w*h*4), box...)
			if got := unpack(back.data, w, h, n, 4, out); !bytes.Equal(got, want) {
				t.Errorf("the %s read back as\n%v\nwant\n%v", name, got, want)
			}
		})
	}
}

// ---------------------------------------------------------------------------------------------
// Encoders, instances and timestamps

// twoEncoders submits two encoders of a frame without waiting between them; the GPU runs them in order.
func twoEncoders(t T, env *Env) {
	t.Run("compute, then a copy and a read", func(t T) {
		const n = 256
		in := make([]uint32, n)
		for i := range in {
			in[i] = uint32(i * 1000003)
		}
		input := env.buffer(t, explicit.BufferStorage, n*4, u32s(in...))
		output := env.buffer(t, explicit.BufferStorage|explicit.BufferCopySource, n*4, make([]byte, n*4))
		moved := env.buffer(t, explicit.BufferCopyDestination|explicit.BufferCopySource, n*4, make([]byte, n*4))
		ub := env.buffer(t, explicit.BufferUniform, 256, u32s(n, 0, 0, 0))
		p := env.computePipeline(t, "cs_square")
		first := env.frame(t)
		g := env.squareGroup(t, p, ub, input, output)
		pass := first.BeginComputePass(explicit.ComputePassDesc{})
		pass.SetPipeline(p)
		pass.SetBindGroup(0, g, 0)
		pass.Dispatch(n/64, 1, 1)
		pass.End()
		if err := first.Submit(); err != nil {
			t.Fatalf("Submit: %v", err)
		}
		second := env.frame(t)
		second.CopyBuffer(output, 0, moved, 0, n*4)
		res := env.readBuffer(second, moved, 0, n*4)
		env.submit(t, second, res)
		if env.Pixels {
			if d := firstDifference(toU32s(res.data), squares(in)); d != "" {
				t.Errorf("the second encoder read the squares wrong: %s", d)
			}
		}
	})

	t.Run("a pass that loads what the first one drew", func(t T) {
		rgba := explicit.TextureFormatRGBA8Unorm
		target := env.colorTarget(t, 4, 4, rgba)
		first := env.frame(t)
		first.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: target, Clear: clearOf(red)}}}).End()
		if err := first.Submit(); err != nil {
			t.Fatalf("Submit: %v", err)
		}
		second := env.frame(t)
		second.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: target, Load: explicit.LoadLoad}}}).End()
		res := env.read(second, target, image.Rect(0, 0, 4, 4))
		env.submit(t, second, res)
		if env.Pixels {
			if got := res.pixel(2, 2, 4); !near(got, red[:], 0) {
				t.Errorf("the second encoder loads %v, want red", got)
			}
		}
	})

	t.Run("OnComplete runs once for each encoder, in order", func(t T) {
		var order []int
		first := env.frame(t)
		env.onComplete(first, func() { order = append(order, 1) })
		if err := first.Submit(); err != nil {
			t.Fatalf("Submit: %v", err)
		}
		second := env.frame(t)
		env.onComplete(second, func() { order = append(order, 2) })
		env.submit(t, second)
		if fmt.Sprint(order) != "[1 2]" {
			t.Errorf("OnComplete ran in the order %v, want [1 2]", order)
		}
	})
}

// instances draws two instances of a quad, and checks the instance index.
func instances(t T, env *Env) {
	rgba := explicit.TextureFormatRGBA8Unorm
	target := env.colorTarget(t, 8, 4, rgba)
	p := env.pipeline(t, explicit.PipelineDesc{Shader: ShaderInstance, VertexEntry: "vs_instance", FragmentEntry: "fs_instance",
		Topology: explicit.TopologyTriangleStrip, ColorTargets: []explicit.ColorTargetState{{Format: rgba}}})
	enc := env.frame(t)
	pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: target, Clear: clearOf(blue)}}})
	pass.SetPipeline(p)
	pass.Draw(4, 2, 0, 0)
	pass.End()
	res := env.read(enc, target, image.Rect(0, 0, 8, 4))
	env.submit(t, enc, res)
	if !env.Pixels {
		return
	}
	if got := res.pixel(1, 1, 4); !near(got, red[:], 0) {
		t.Errorf("instance 0 draws %v, want red", got)
	}
	if got := res.pixel(6, 1, 4); !near(got, green[:], 0) {
		t.Errorf("instance 1 draws %v, want green", got)
	}
}

// timestamps writes the timestamps of a compute pass and of a render pass, and resolves them.
func timestamps(t T, env *Env) {
	feat := env.Dev.Features()
	set, err := env.Dev.NewQuerySet(4)
	if err != nil {
		if appErr, ok := errors.FromError(err); ok && appErr.Tag == explicit.TagGPUQueryUnsupported && !feat.TimestampQuery {
			t.Skipf("the device has no timestamp queries")
		}
		t.Fatalf("NewQuerySet: %v", err)
	}
	env.cleanup = append(env.cleanup, set.Destroy)
	if set.Count() != 4 {
		t.Errorf("the query set has %d queries, want 4", set.Count())
	}
	rgba := explicit.TextureFormatRGBA8Unorm
	const n = 4096
	input := env.buffer(t, explicit.BufferStorage, n*4, make([]byte, n*4))
	output := env.buffer(t, explicit.BufferStorage, n*4, make([]byte, n*4))
	ub := env.buffer(t, explicit.BufferUniform, 256, u32s(n, 0, 0, 0))
	resolved := env.buffer(t, explicit.BufferQueryResolve|explicit.BufferCopySource, 256, make([]byte, 256))
	target := env.colorTarget(t, 4, 4, rgba)
	p := env.computePipeline(t, "cs_square")
	fp := env.fillPipeline(t, "fs_float", explicit.ColorTargetState{Format: rgba})
	enc := env.frame(t)
	g := env.squareGroup(t, p, ub, input, output)
	cpass := enc.BeginComputePass(explicit.ComputePassDesc{Timestamps: &explicit.PassTimestamps{Set: set, Begin: 0, End: 1}})
	cpass.SetPipeline(p)
	cpass.SetBindGroup(0, g, 0)
	cpass.Dispatch(n/64, 1, 1)
	cpass.End()
	pass := enc.BeginRenderPass(explicit.PassDesc{
		Color:      []explicit.ColorAttachment{{Target: target}},
		Timestamps: &explicit.PassTimestamps{Set: set, Begin: 2, End: 3},
	})
	pass.SetPipeline(fp)
	pass.Draw(3, 1, 0, 0)
	pass.End()
	enc.ResolveQuerySet(set, 0, 4, resolved, 0)
	res := env.readBuffer(enc, resolved, 0, 32)
	env.submit(t, enc, res)
	if !env.Pixels {
		return
	}
	ticks := make([]uint64, 4)
	for i := range ticks {
		ticks[i] = binary.LittleEndian.Uint64(res.data[i*8:])
	}
	for i, name := range []string{"compute pass", "render pass"} {
		begin, end := ticks[2*i], ticks[2*i+1]
		if begin == 0 || end == 0 {
			t.Errorf("the %s wrote the timestamps %d and %d, want two that are not 0", name, begin, end)
			continue
		}
		if end < begin {
			t.Errorf("the %s ended (%d) before it began (%d)", name, end, begin)
			continue
		}
		if ns := float64(end-begin) * feat.TimestampPeriod; ns > 5e9 {
			t.Errorf("the %s took %.3g seconds, which is not a tick of %g ns", name, ns/1e9, feat.TimestampPeriod)
		}
	}
}

// manyReads reads a buffer back in each of a thousand frames, and checks that the memory does not grow:
// the staging buffers of the reads are released with their frames.
func manyReads(t T, env *Env) {
	const frames, n = 1000, 16
	input := env.buffer(t, explicit.BufferStorage, n*4, make([]byte, n*4))
	output := env.buffer(t, explicit.BufferStorage|explicit.BufferCopySource, n*4, make([]byte, n*4))
	ub := env.buffer(t, explicit.BufferUniform, 256, u32s(n, 0, 0, 0))
	p := env.computePipeline(t, "cs_square")
	memory := func() int64 {
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		total := int64(m.HeapAlloc)
		if env.Memory != nil {
			total += env.Memory()
		}
		return total
	}
	var warm int64
	for i := 0; i < frames; i++ {
		in := make([]uint32, n)
		for j := range in {
			in[j] = uint32(i*n + j)
		}
		env.Dev.WriteBuffer(input, 0, u32s(in...))
		enc := env.frame(t)
		// A bind group lives as long as its frame, so each frame makes its own.
		g, err := env.Dev.NewComputeBindGroup(p, 0, []explicit.Binding{
			{Binding: 0, Buffer: ub, Size: 16}, {Binding: 1, Buffer: input}, {Binding: 2, Buffer: output}})
		if err != nil {
			t.Fatalf("NewComputeBindGroup: %v", err)
		}
		pass := enc.BeginComputePass(explicit.ComputePassDesc{})
		pass.SetPipeline(p)
		pass.SetBindGroup(0, g, 0)
		pass.Dispatch(1, 1, 1)
		pass.End()
		res := env.readBuffer(enc, output, 0, n*4)
		env.submit(t, enc, res)
		g.Destroy()
		if env.Pixels {
			if d := firstDifference(toU32s(res.data), squares(in)); d != "" {
				t.Fatalf("frame %d read the squares wrong: %s", i, d)
			}
		}
		if i == frames/10 {
			warm = memory()
		}
	}
	if grown := memory() - warm; grown > 4<<20 {
		t.Errorf("the memory grew by %d bytes between frame %d and frame %d", grown, frames/10, frames)
	}
}
