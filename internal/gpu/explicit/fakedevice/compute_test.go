package fakedevice

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// computeShader is a shader with a compute entry point and a vertex one, written by hand: the fields
// are what glutter-gen shaders -gpu writes.
func computeShader() *explicit.Shader {
	return &explicit.Shader{Name: "test", Entries: []explicit.EntryPoint{
		{Name: "cs", Stage: explicit.ShaderStageCompute, WorkgroupSize: [3]int{64, 1, 1}, Bindings: []explicit.ShaderBinding{
			{Name: "params", Group: 0, Binding: 0, Kind: explicit.BindingUniformBuffer},
			{Name: "input", Group: 0, Binding: 1, Kind: explicit.BindingStorageBufferReadOnly},
			{Name: "output", Group: 0, Binding: 2, Kind: explicit.BindingStorageBuffer},
			{Name: "image", Group: 1, Binding: 0, Kind: explicit.BindingStorageTexture, StorageFormat: explicit.TextureFormatRGBA8Unorm},
		}},
		{Name: "big", Stage: explicit.ShaderStageCompute, WorkgroupSize: [3]int{8, 8, 4}},
		{Name: "wide", Stage: explicit.ShaderStageCompute, WorkgroupSize: [3]int{512, 1, 1}},
		{Name: "vs_instance", Stage: explicit.ShaderStageVertex, ReadsInstanceIndex: true},
		{Name: "vs_vertex", Stage: explicit.ShaderStageVertex, ReadsVertexIndex: true},
		{Name: "vs_plain", Stage: explicit.ShaderStageVertex},
		{Name: "vs_writes", Stage: explicit.ShaderStageVertex, Bindings: []explicit.ShaderBinding{
			{Name: "w", Group: 0, Binding: 0, Kind: explicit.BindingStorageBuffer}}},
		{Name: "fs", Stage: explicit.ShaderStageFragment},
	}}
}

func TestComputePipelines(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name  string
		entry string
		tag   string // "" for none
		route string
	}{
		{"a compute entry point", "cs", "", ""},
		{"a workgroup of the most invocations in each dimension", "big", "", ""},
		{"a workgroup wider than the limit", "wide", "GPULimitExceeded", "Device.NewComputePipeline->Limits.CheckComputePipeline->exceeded.GPULimitExceeded"},
		{"a vertex entry point", "vs_plain", "GPUDescriptorInvalid", "Device.NewComputePipeline->Limits.CheckComputePipeline->invalid.GPUDescriptorInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			do := func() {
				_, _ = d.NewComputePipeline(explicit.ComputePipelineDesc{Shader: computeShader(), Entry: tt.entry})
			}
			if tt.tag == "" {
				if err := panicOf(do); err != nil {
					t.Fatalf("panic: %v", err)
				}
				return
			}
			_ = expectInvalid(t, tt.tag, tt.route, do)
		})
	}
}

func TestOnlyComputeWritesStorage(t *testing.T) {
	skipUnlessDebug(t)
	d := New()
	_ = expectInvalid(t, "GPUDescriptorInvalid", "Device.NewPipeline->Limits.CheckPipeline->Limits.checkStageBindings->invalid.GPUDescriptorInvalid", func() {
		_, _ = d.NewPipeline(explicit.PipelineDesc{Shader: computeShader(), VertexEntry: "vs_writes", FragmentEntry: "fs", Target: explicit.TextureFormatRGBA8Unorm})
	})
}

func TestBindGroupEntriesFitTheirBindings(t *testing.T) {
	skipUnlessDebug(t)
	d := New()
	cp, err := d.NewComputePipeline(explicit.ComputePipelineDesc{Shader: computeShader(), Entry: "cs"})
	if err != nil {
		t.Fatal(err)
	}
	buf := func(size int, u explicit.BufferUsage) explicit.Buffer {
		b, err := d.NewBuffer(size, u)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	uniform := buf(256, explicit.BufferUniform)
	storage := buf(4096, explicit.BufferStorage)
	vertex := buf(4096, explicit.BufferVertex)
	tex := func(f explicit.TextureFormat, u explicit.TextureUsage) explicit.Texture {
		x, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: f, Usage: u})
		if err != nil {
			t.Fatal(err)
		}
		return x
	}
	img := tex(explicit.TextureFormatRGBA8Unorm, explicit.TextureStorage)
	good := []explicit.Binding{{Binding: 0, Buffer: uniform}, {Binding: 1, Buffer: storage, Offset: 256, Size: 1024}, {Binding: 2, Buffer: storage}}
	tests := []struct {
		name    string
		group   int
		entries []explicit.Binding
		ok      bool
		tag     string
		route   string
	}{
		{"every kind", 0, good, true, "", ""},
		{"a storage texture", 1, []explicit.Binding{{Binding: 0, Texture: img}}, true, "", ""},
		{"a storage buffer bound as a uniform", 0, []explicit.Binding{{Binding: 0, Buffer: storage}}, false, "GPUDescriptorInvalid", "Device.NewComputeBindGroup->Device.checkBindings->Limits.CheckBinding->invalid.GPUDescriptorInvalid"},
		{"a uniform buffer bound as storage", 0, []explicit.Binding{{Binding: 2, Buffer: uniform}}, false, "GPUDescriptorInvalid", "Device.NewComputeBindGroup->Device.checkBindings->Limits.CheckBinding->invalid.GPUDescriptorInvalid"},
		{"a vertex buffer bound as storage", 0, []explicit.Binding{{Binding: 1, Buffer: vertex}}, false, "GPUDescriptorInvalid", "Device.NewComputeBindGroup->Device.checkBindings->Limits.CheckBinding->invalid.GPUDescriptorInvalid"},
		{"a storage offset that is not aligned", 0, []explicit.Binding{{Binding: 1, Buffer: storage, Offset: 4, Size: 256}}, false, "GPULimitExceeded", "Device.NewComputeBindGroup->Device.checkBindings->Limits.CheckBinding->misaligned.GPULimitExceeded"},
		{"a storage range past the buffer", 0, []explicit.Binding{{Binding: 1, Buffer: storage, Offset: 256, Size: 8192}}, false, "GPUDescriptorInvalid", "Device.NewComputeBindGroup->Device.checkBindings->Limits.CheckBinding->invalid.GPUDescriptorInvalid"},
		{"a texture without storage usage", 1, []explicit.Binding{{Binding: 0, Texture: tex(explicit.TextureFormatRGBA8Unorm, explicit.TextureSampled)}}, false, "GPUDescriptorInvalid", "Device.NewComputeBindGroup->Device.checkBindings->Limits.CheckBinding->invalid.GPUDescriptorInvalid"},
		{"a storage texture of another format", 1, []explicit.Binding{{Binding: 0, Texture: tex(explicit.TextureFormatR32Float, explicit.TextureStorage)}}, false, "GPUDescriptorInvalid", "Device.NewComputeBindGroup->Device.checkBindings->Limits.CheckBinding->invalid.GPUDescriptorInvalid"},
		{"a group over the limit", 4, good, false, "GPULimitExceeded", "Device.NewComputeBindGroup->Limits.CheckBindGroup->exceeded.GPULimitExceeded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			do := func() { _, _ = d.NewComputeBindGroup(cp, tt.group, tt.entries) }
			if tt.ok {
				if err := panicOf(do); err != nil {
					t.Fatalf("panic: %v", err)
				}
				return
			}
			_ = expectInvalid(t, tt.tag, tt.route, do)
		})
	}
}

func TestDispatchesAndIndirectCalls(t *testing.T) {
	skipUnlessDebug(t)
	d := New()
	cp, _ := d.NewComputePipeline(explicit.ComputePipelineDesc{Shader: computeShader(), Entry: "cs"})
	indirect, _ := d.NewBuffer(64, explicit.BufferIndirect|explicit.BufferStorage)
	plain, _ := d.NewBuffer(64, explicit.BufferStorage)
	e, _ := d.Begin()
	p := e.BeginComputePass(explicit.ComputePassDesc{})
	p.SetPipeline(cp)
	max := d.Limits().MaxComputeWorkgroupsPerDimension
	tests := []struct {
		name  string
		do    func()
		tag   string
		route string
	}{
		{"the most workgroups", func() { p.Dispatch(max, 1, 1) }, "", ""},
		{"one workgroup more than the most", func() { p.Dispatch(1, max+1, 1) }, "GPULimitExceeded", "computePass.Dispatch->Limits.CheckDispatch->exceeded.GPULimitExceeded"},
		{"a negative dimension", func() { p.Dispatch(-1, 1, 1) }, "GPUDescriptorInvalid", "computePass.Dispatch->Limits.CheckDispatch->invalid.GPUDescriptorInvalid"},
		{"an indirect dispatch", func() { p.DispatchIndirect(indirect, 16) }, "", ""},
		{"an indirect dispatch without the usage", func() { p.DispatchIndirect(plain, 0) }, "GPUDescriptorInvalid", "computePass.DispatchIndirect->Limits.CheckIndirect->invalid.GPUDescriptorInvalid"},
		{"an indirect dispatch at an odd offset", func() { p.DispatchIndirect(indirect, 2) }, "GPUDescriptorInvalid", "computePass.DispatchIndirect->Limits.CheckIndirect->invalid.GPUDescriptorInvalid"},
		{"an indirect dispatch past the end", func() { p.DispatchIndirect(indirect, 60) }, "GPUDescriptorInvalid", "computePass.DispatchIndirect->Limits.CheckIndirect->invalid.GPUDescriptorInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tag == "" {
				if err := panicOf(tt.do); err != nil {
					t.Fatalf("panic: %v", err)
				}
				return
			}
			_ = expectInvalid(t, tt.tag, tt.route, tt.do)
		})
	}
	p.End()
	rp := e.BeginPass(mustTexture(t, d), explicit.LoadClear, colorZero)
	rp.SetPipeline(mustPipeline(t, d, "vs_plain"))
	_ = expectInvalid(t, "GPUDescriptorInvalid", "pass.DrawIndirect->Limits.CheckIndirect->invalid.GPUDescriptorInvalid", func() { rp.DrawIndirect(plain, 0) })
	_ = expectInvalid(t, "GPUDescriptorInvalid", "pass.DrawIndexedIndirect->Limits.CheckIndirect->invalid.GPUDescriptorInvalid", func() { rp.DrawIndexedIndirect(plain, 0) })
	if err := panicOf(func() { rp.DrawIndirect(indirect, 0); rp.DrawIndexedIndirect(indirect, 4) }); err != nil {
		t.Errorf("indirect draws panicked: %v", err)
	}
	_ = expectInvalid(t, "GPUDescriptorInvalid", "pass.DrawIndexedIndirect->Limits.CheckIndirect->invalid.GPUDescriptorInvalid", func() { rp.DrawIndexedIndirect(indirect, 48) })
	rp.End()
}

func TestTheFirstInstanceAndTheFirstVertexAreReadOnlyAsZero(t *testing.T) {
	skipUnlessDebug(t)
	tests := []struct {
		name  string
		entry string
		do    func(p explicit.Pass)
		bad   bool
	}{
		{"instance_index with a first instance of 0", "vs_instance", func(p explicit.Pass) { p.Draw(3, 2, 0, 0) }, false},
		{"instance_index with a first instance", "vs_instance", func(p explicit.Pass) { p.Draw(3, 2, 0, 1) }, true},
		{"instance_index, indexed, with a first instance", "vs_instance", func(p explicit.Pass) { p.DrawIndexed(3, 2, 0, 0, 5) }, true},
		{"instance_index with a first vertex", "vs_instance", func(p explicit.Pass) { p.Draw(3, 2, 7, 0) }, false},
		{"vertex_index with a first vertex", "vs_vertex", func(p explicit.Pass) { p.Draw(3, 1, 4, 0) }, true},
		{"vertex_index with a base vertex", "vs_vertex", func(p explicit.Pass) { p.DrawIndexed(3, 1, 0, -2, 0) }, true},
		{"vertex_index with a first index", "vs_vertex", func(p explicit.Pass) { p.DrawIndexed(3, 1, 9, 0, 0) }, false},
		{"vertex_index with a first instance", "vs_vertex", func(p explicit.Pass) { p.Draw(3, 1, 0, 3) }, false},
		{"neither, with every first", "vs_plain", func(p explicit.Pass) { p.Draw(3, 1, 4, 3); p.DrawIndexed(3, 1, 9, -2, 3) }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			e, _ := d.Begin()
			p := e.BeginPass(mustTexture(t, d), explicit.LoadClear, colorZero)
			p.SetPipeline(mustPipeline(t, d, tt.entry))
			if !tt.bad {
				if err := panicOf(func() { tt.do(p) }); err != nil {
					t.Fatalf("panic: %v", err)
				}
				return
			}
			appErr := expectInvalid(t, "GPUFirstInstanceRead", "CheckFirstIndices.GPUFirstInstanceRead", func() { tt.do(p) })
			if appErr.Metadata["entry"] != tt.entry {
				t.Errorf("entry = %v", appErr.Metadata["entry"])
			}
		})
	}
}

func TestCopiesAndReadsOfBuffers(t *testing.T) {
	skipUnlessDebug(t)
	d := New()
	src, _ := d.NewBuffer(64, explicit.BufferCopySource)
	dst, _ := d.NewBuffer(64, explicit.BufferCopyDestination|explicit.BufferCopySource)
	nothing, _ := d.NewBuffer(64, explicit.BufferVertex)
	d.WriteBuffer(src, 0, []byte{1, 2, 3, 4, 5, 6, 7, 8})
	e, _ := d.Begin()
	tests := []struct {
		name  string
		do    func()
		tag   string
		route string
	}{
		{"a copy", func() { e.CopyBuffer(src, 0, dst, 8, 8) }, "", ""},
		{"a copy from a buffer without the usage", func() { e.CopyBuffer(nothing, 0, dst, 0, 4) }, "GPUDescriptorInvalid", "encoder.CopyBuffer->Limits.CheckCopyBuffer->invalid.GPUDescriptorInvalid"},
		{"a copy into a buffer without the usage", func() { e.CopyBuffer(src, 0, nothing, 0, 4) }, "GPUDescriptorInvalid", "encoder.CopyBuffer->Limits.CheckCopyBuffer->invalid.GPUDescriptorInvalid"},
		{"a copy of a size that is not a multiple of 4", func() { e.CopyBuffer(src, 0, dst, 0, 6) }, "GPUDescriptorInvalid", "encoder.CopyBuffer->Limits.CheckCopyBuffer->invalid.GPUDescriptorInvalid"},
		{"a copy past the end", func() { e.CopyBuffer(src, 60, dst, 0, 8) }, "GPUDescriptorInvalid", "encoder.CopyBuffer->Limits.CheckCopyBuffer->invalid.GPUDescriptorInvalid"},
		{"a read", func() { e.ReadBuffer(dst, 8, 8, func([]byte, error) {}) }, "", ""},
		{"a read of a buffer without the usage", func() { e.ReadBuffer(nothing, 0, 4, func([]byte, error) {}) }, "GPUDescriptorInvalid", "encoder.ReadBuffer->Limits.CheckReadBuffer->invalid.GPUDescriptorInvalid"},
		{"a read past the end", func() { e.ReadBuffer(dst, 60, 8, func([]byte, error) {}) }, "GPUDescriptorInvalid", "encoder.ReadBuffer->Limits.CheckReadBuffer->invalid.GPUDescriptorInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tag == "" {
				if err := panicOf(tt.do); err != nil {
					t.Fatalf("panic: %v", err)
				}
				return
			}
			_ = expectInvalid(t, tt.tag, tt.route, tt.do)
		})
	}
	var got []byte
	e.ReadBuffer(dst, 8, 8, func(b []byte, err error) {
		if err != nil {
			t.Error(err)
		}
		got = b
	})
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("the read was answered before the GPU finished")
	}
	d.CompleteReads()
	if !reflectEqual(got, []byte{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Errorf("read %v", got)
	}
}

func TestCopiesBetweenBuffersAndTextures(t *testing.T) {
	skipUnlessDebug(t)
	d := New()
	rgba := explicit.TextureFormatRGBA8Unorm
	tex, _ := d.NewTexture(explicit.TextureDesc{Width: 5, Height: 3, Format: rgba, Usage: explicit.TextureCopySource | explicit.TextureSampled})
	depth, _ := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatDepth24PlusStencil8, Usage: explicit.TextureRenderTarget})
	ms, _ := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: rgba, SampleCount: 4, Usage: explicit.TextureRenderTarget})
	buf, _ := d.NewBuffer(4096, explicit.BufferCopySource|explicit.BufferCopyDestination)
	small, _ := d.NewBuffer(256, explicit.BufferCopySource|explicit.BufferCopyDestination)
	e, _ := d.Begin()
	box := func(w, h int) explicit.TextureRegion { return explicit.TextureRegion{Size: [3]int{w, h, 1}} }
	layout := explicit.BufferLayout{BytesPerRow: 256, RowsPerImage: 3}
	tests := []struct {
		name string
		do   func()
		bad  bool
	}{
		{"buffer to texture", func() { e.CopyBufferToTexture(buf, layout, tex, box(5, 3)) }, false},
		{"texture to buffer", func() { e.CopyTextureToBuffer(tex, box(5, 3), buf, layout) }, false},
		{"rows not aligned", func() {
			e.CopyBufferToTexture(buf, explicit.BufferLayout{BytesPerRow: 20, RowsPerImage: 3}, tex, box(5, 3))
		}, true},
		{"a pitch shorter than the row", func() {
			e.CopyBufferToTexture(buf, explicit.BufferLayout{BytesPerRow: 0, RowsPerImage: 3}, tex, box(5, 3))
		}, true},
		{"too few rows per image", func() {
			e.CopyBufferToTexture(buf, explicit.BufferLayout{BytesPerRow: 256, RowsPerImage: 2}, tex, box(5, 3))
		}, true},
		{"a box past the texture", func() { e.CopyBufferToTexture(buf, layout, tex, box(6, 3)) }, true},
		{"a box of no size", func() { e.CopyBufferToTexture(buf, layout, tex, box(0, 3)) }, true},
		{"rows past the buffer", func() { e.CopyBufferToTexture(small, layout, tex, box(5, 3)) }, true},
		{"a depth texture written", func() { e.CopyBufferToTexture(buf, layout, depth, box(4, 4)) }, true},
		{"a depth-stencil texture read", func() { e.CopyTextureToBuffer(depth, box(4, 4), buf, layout) }, true},
		{"a multisampled texture read", func() { e.CopyTextureToBuffer(ms, box(4, 4), buf, layout) }, true},
		{"the offset is not a multiple of the pixel", func() {
			e.CopyBufferToTexture(buf, explicit.BufferLayout{Offset: 2, BytesPerRow: 256, RowsPerImage: 3}, tex, box(5, 3))
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.bad {
				if err := panicOf(tt.do); err != nil {
					t.Fatalf("panic: %v", err)
				}
				return
			}
			err := panicOf(tt.do)
			if err == nil {
				t.Fatal("no panic")
			}
			// Both copies call CheckBufferTexture through their own method.
			_ = expectInvalid(t, "GPUDescriptorInvalid", "Limits.CheckBufferTexture->invalid.GPUDescriptorInvalid", tt.do)
		})
	}
}

func TestQuerySetsAndTimestamps(t *testing.T) {
	skipUnlessDebug(t)
	d := New()
	if _, err := d.NewQuerySet(2); err == nil {
		t.Fatal("a device without timestamp queries made a query set")
	} else {
		expectTag(t, err, "GPUQueryUnsupported", "Device.NewQuerySet.GPUQueryUnsupported")
	}
	d.Supported.TimestampQuery = true
	set, err := d.NewQuerySet(4)
	if err != nil {
		t.Fatal(err)
	}
	resolve, _ := d.NewBuffer(1024, explicit.BufferQueryResolve|explicit.BufferCopySource)
	plain, _ := d.NewBuffer(1024, explicit.BufferCopySource)
	e, _ := d.Begin()
	tests := []struct {
		name  string
		do    func()
		tag   string
		route string
	}{
		{"a compute pass with timestamps", func() {
			e.BeginComputePass(explicit.ComputePassDesc{Timestamps: &explicit.PassTimestamps{Set: set, Begin: 0, End: 1}}).End()
		}, "", ""},
		{"a render pass with timestamps", func() {
			e.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: mustTexture(t, d)}}, Timestamps: &explicit.PassTimestamps{Set: set, Begin: 2, End: 3}}).End()
		}, "", ""},
		{"a query past the set", func() {
			e.BeginComputePass(explicit.ComputePassDesc{Timestamps: &explicit.PassTimestamps{Set: set, Begin: 0, End: 4}})
		}, "GPUDescriptorInvalid", "encoder.BeginComputePass->Limits.CheckTimestamps->invalid.GPUDescriptorInvalid"},
		{"one query twice", func() {
			e.BeginComputePass(explicit.ComputePassDesc{Timestamps: &explicit.PassTimestamps{Set: set, Begin: 1, End: 1}})
		}, "GPUDescriptorInvalid", "encoder.BeginComputePass->Limits.CheckTimestamps->invalid.GPUDescriptorInvalid"},
		{"a resolve", func() { e.ResolveQuerySet(set, 0, 4, resolve, 256) }, "", ""},
		{"a resolve into a buffer without the usage", func() { e.ResolveQuerySet(set, 0, 4, plain, 0) }, "GPUDescriptorInvalid", "encoder.ResolveQuerySet->Limits.CheckResolveQuerySet->invalid.GPUDescriptorInvalid"},
		{"a resolve at an unaligned offset", func() { e.ResolveQuerySet(set, 0, 2, resolve, 8) }, "GPULimitExceeded", "encoder.ResolveQuerySet->Limits.CheckResolveQuerySet->misaligned.GPULimitExceeded"},
		{"a resolve of queries the set lacks", func() { e.ResolveQuerySet(set, 2, 3, resolve, 0) }, "GPUDescriptorInvalid", "encoder.ResolveQuerySet->Limits.CheckResolveQuerySet->invalid.GPUDescriptorInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tag == "" {
				if err := panicOf(tt.do); err != nil {
					t.Fatalf("panic: %v", err)
				}
				return
			}
			_ = expectInvalid(t, tt.tag, tt.route, tt.do)
		})
	}
}

func TestATimestampPassOnADeviceWithoutQueriesPanics(t *testing.T) {
	skipUnlessDebug(t)
	d := New()
	e, _ := d.Begin()
	_ = expectInvalid(t, "GPUDescriptorInvalid", "encoder.BeginComputePass->Limits.CheckTimestamps->invalid.GPUDescriptorInvalid", func() {
		e.BeginComputePass(explicit.ComputePassDesc{Timestamps: &explicit.PassTimestamps{Set: nil, Begin: 0, End: 1}})
	})
}

func TestOnCompleteRunsAfterTheGPUFinished(t *testing.T) {
	d := New()
	var order []string
	for _, name := range []string{"first", "second"} {
		e, _ := d.Begin()
		e.OnComplete(func() { order = append(order, name) })
		if err := e.Submit(); err != nil {
			t.Fatal(err)
		}
	}
	if len(order) != 0 {
		t.Fatalf("%v ran before the GPU finished", order)
	}
	d.CompleteReads()
	if !reflectEqual(order, []string{"first", "second"}) {
		t.Errorf("order = %v, want the order of the submissions", order)
	}
	d.CompleteReadsOnSubmit = true
	e, _ := d.Begin()
	e.OnComplete(func() { order = append(order, "third") })
	_ = e.Submit()
	if len(order) != 3 {
		t.Errorf("order = %v: a device that waits runs it in Submit", order)
	}
}

func TestIndexBuffersOfBothSizes(t *testing.T) {
	d := New()
	idx, _ := d.NewBuffer(64, explicit.BufferIndex)
	e, _ := d.Begin()
	p := e.BeginPass(mustTexture(t, d), explicit.LoadClear, colorZero)
	p.SetIndexBuffer(idx, 0)
	p.SetIndexBuffer32(idx, 4)
	p.End()
	ops := d.Ops()
	if !containsOp(ops, "SetIndexBuffer") || !containsOp(ops, "SetIndexBuffer32") {
		t.Errorf("ops = %v", ops)
	}
}
