//go:build windows && glutter_wine

package d3d12

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
)

// The failures of the compute pipelines, bind groups and passes, the copies and the queries (step 30.2):
// each has a route that the test asserts.

func TestWineComputePipelineErrors(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	tests := []struct {
		name  string
		desc  explicit.ComputePipelineDesc
		route string
	}{
		{"no shader", explicit.ComputePipelineDesc{Entry: "cs_square"}, "Device.NewComputePipeline.D3D12PipelineInvalid"},
		{"an entry point the shader lacks", explicit.ComputePipelineDesc{Shader: devicetest.ShaderCompute, Entry: "cs_missing"}, "Device.NewComputePipeline.D3D12PipelineInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := d.NewComputePipeline(tt.desc)
			if p != nil {
				t.Errorf("a pipeline was made")
			}
			_ = wantRoute(t, err, tagD3D12PipelineInvalid, tt.route)
		})
	}
}

// The debug check answers before the device does: an entry point that is not a compute entry point.
func TestWineComputePipelineOfAnotherStagePanics(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	appErr := expectPanic(t, func() {
		_, _ = d.NewComputePipeline(explicit.ComputePipelineDesc{Shader: devicetest.ShaderFill, Entry: "vs_full"})
	})
	if appErr.Tag != "GPUDescriptorInvalid" || !appErr.HasRoute("Device.NewComputePipeline->Limits.CheckComputePipeline->invalid.GPUDescriptorInvalid") {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
}

func TestWineComputeBindGroupErrors(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	p, err := d.NewComputePipeline(explicit.ComputePipelineDesc{Shader: devicetest.ShaderCompute, Entry: "cs_square"})
	if err != nil {
		t.Fatal(err)
	}
	ub, err := d.NewBuffer(256, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	in, err := d.NewBuffer(256, explicit.BufferStorage)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		group   int
		entries []explicit.Binding
		route   string
	}{
		{"a group the pipeline lacks", 3, nil, "Device.NewComputeBindGroup->Device.makeBindGroup.D3D12BindGroupInvalid"},
		{"a binding the shader lacks", 0, []explicit.Binding{{Binding: 9, Buffer: in}}, "Device.NewComputeBindGroup->Device.makeBindGroup.D3D12BindGroupInvalid"},
		{"a storage binding without an entry", 0, []explicit.Binding{{Binding: 0, Buffer: ub, Size: 16}, {Binding: 1, Buffer: in}}, "Device.NewComputeBindGroup->Device.makeBindGroup->Device.makeBindGroup.func2.D3D12BindGroupInvalid"},
		{"a storage binding without its buffer", 0, []explicit.Binding{{Binding: 0, Buffer: ub, Size: 16}, {Binding: 1}, {Binding: 2, Buffer: in}}, "Device.NewComputeBindGroup->Device.makeBindGroup->Device.makeBindGroup.func2.D3D12BindGroupInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bg, err := d.NewComputeBindGroup(p, tt.group, tt.entries)
			if bg != nil {
				t.Errorf("a bind group was made")
			}
			_ = wantRoute(t, err, tagD3D12BindGroupInvalid, tt.route)
		})
	}
	other, _ := openTestDevice(t, Options{})
	foreign, err := other.NewBuffer(256, explicit.BufferStorage)
	if err != nil {
		t.Fatal(err)
	}
	appErr := expectPanic(t, func() {
		_, _ = d.NewComputeBindGroup(p, 0, []explicit.Binding{{Binding: 0, Buffer: ub, Size: 16}, {Binding: 1, Buffer: foreign}, {Binding: 2, Buffer: in}})
	})
	if appErr.Tag != tagD3D12ForeignResource || !appErr.HasRoute("Device.NewComputeBindGroup->Device.makeBindGroup->Device.makeBindGroup.func2->Device.makeBindGroup.func1.D3D12ForeignResource") {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
	appErr = expectPanic(t, func() { _, _ = d.NewComputeBindGroup(nil, 0, nil) })
	if appErr.Tag != tagD3D12ForeignResource || !appErr.HasRoute("Device.NewComputeBindGroup.D3D12ForeignResource") {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
}

func TestWineComputePassErrors(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	p, err := d.NewComputePipeline(explicit.ComputePipelineDesc{Shader: devicetest.ShaderCompute, Entry: "cs_square"})
	if err != nil {
		t.Fatal(err)
	}
	buf, err := d.NewBuffer(256, explicit.BufferStorage|explicit.BufferCopySource|explicit.BufferCopyDestination|explicit.BufferIndirect)
	if err != nil {
		t.Fatal(err)
	}
	spare, err := d.NewBuffer(256, explicit.BufferCopyDestination)
	if err != nil {
		t.Fatal(err)
	}
	other, _ := openTestDevice(t, Options{})
	otherBuf, err := other.NewBuffer(256, explicit.BufferStorage|explicit.BufferCopySource|explicit.BufferCopyDestination|explicit.BufferIndirect)
	if err != nil {
		t.Fatal(err)
	}
	wantPanic := func(what, route, tag string, fn func()) {
		t.Helper()
		appErr := expectPanic(t, fn)
		if appErr.Tag != tag || !appErr.HasRoute(route) {
			t.Errorf("%s: %s via %s, want %s via %s", what, appErr.Tag, appErr.Route(), tag, route)
		}
	}
	e, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	wantPanic("a copy between devices", "encoder.CopyBuffer.D3D12ForeignResource", tagD3D12ForeignResource, func() { e.CopyBuffer(otherBuf, 0, buf, 0, 16) })
	wantPanic("a read between devices", "encoder.ReadBuffer.D3D12ForeignResource", tagD3D12ForeignResource, func() { e.ReadBuffer(otherBuf, 0, 16, nil) })
	cp := e.BeginComputePass(explicit.ComputePassDesc{})
	wantPanic("a second pass", "encoder.BeginComputePass.D3D12PassNotEnded", tagD3D12PassNotEnded, func() { e.BeginComputePass(explicit.ComputePassDesc{}) })
	wantPanic("a copy in a pass", "encoder.CopyBuffer.D3D12PassNotEnded", tagD3D12PassNotEnded, func() { e.CopyBuffer(buf, 0, spare, 64, 16) })
	wantPanic("a read in a pass", "encoder.ReadBuffer.D3D12PassNotEnded", tagD3D12PassNotEnded, func() { e.ReadBuffer(buf, 0, 16, nil) })
	wantPanic("Dispatch before SetPipeline", "computePass.Dispatch->computePass.requirePipeline.D3D12PipelineInvalid", tagD3D12PipelineInvalid, func() { cp.Dispatch(1, 1, 1) })
	wantPanic("DispatchIndirect before SetPipeline", "computePass.DispatchIndirect->computePass.requirePipeline.D3D12PipelineInvalid", tagD3D12PipelineInvalid, func() { cp.DispatchIndirect(buf, 0) })
	wantPanic("a pipeline of another device", "computePass.SetPipeline.D3D12ForeignResource", tagD3D12ForeignResource, func() { cp.SetPipeline(nil) })
	wantPanic("a buffer of another device", "computePass.DispatchIndirect.D3D12ForeignResource", tagD3D12ForeignResource, func() {
		cp.SetPipeline(p)
		cp.DispatchIndirect(otherBuf, 0)
	})
	cp.End()
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
}

func TestWineQuerySets(t *testing.T) {
	d, _ := openTestDevice(t, Options{})
	if !d.Features().TimestampQuery {
		_, err := d.NewQuerySet(2)
		_ = wantRoute(t, err, explicit.TagGPUQueryUnsupported, "Device.NewQuerySet.GPUQueryUnsupported")
		return
	}
	if d.Features().TimestampPeriod <= 0 {
		t.Errorf("TimestampPeriod = %v, want a time", d.Features().TimestampPeriod)
	}
	set, err := d.NewQuerySet(2)
	if err != nil {
		t.Fatal(err)
	}
	defer set.Destroy()
	other, _ := openTestDevice(t, Options{})
	dst, err := d.NewBuffer(64, explicit.BufferCopyDestination|explicit.BufferCopySource|explicit.BufferQueryResolve)
	if err != nil {
		t.Fatal(err)
	}
	otherSet, err := other.NewQuerySet(2)
	if err != nil {
		t.Fatal(err)
	}
	e, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	appErr := expectPanic(t, func() { e.ResolveQuerySet(otherSet, 0, 2, dst, 0) })
	if appErr.Tag != tagD3D12ForeignResource || !appErr.HasRoute("encoder.ResolveQuerySet.D3D12ForeignResource") {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
	appErr = expectPanic(t, func() {
		e.BeginComputePass(explicit.ComputePassDesc{Timestamps: &explicit.PassTimestamps{Set: otherSet, Begin: 0, End: 1}})
	})
	if appErr.Tag != tagD3D12ForeignResource || !appErr.HasRoute("encoder.BeginComputePass->timestampsOf.D3D12ForeignResource") {
		t.Errorf("%s via %s", appErr.Tag, appErr.Route())
	}
	if err := e.Submit(); err != nil {
		t.Fatal(err)
	}
}
