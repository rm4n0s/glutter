package gpu_test

import (
	"testing"

	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/fakedevice"
	"github.com/rm4n0s/glutter/internal/leaktest"
)

// TestLeaksFrames makes, uses and releases the resources of a frame 1,000 times (step 30.4, SPEC §15 "Memory over
// time"): the live heap, the goroutines, the resources that package gpu counts and the ones the backend holds must
// not grow.
func TestLeaksFrames(t *testing.T) {
	tests := []struct {
		name  string
		frame func(h *harness, s *scene) func(i int)
	}{
		{"a frame with its own buffer, texture and bind group", func(h *harness, s *scene) func(int) {
			return func(i int) {
				d := h.d
				ub := must(d.NewBuffer(gpu.BufferDesc{Size: 256, Usage: gpu.BufferUsageUniform | gpu.BufferUsageCopyDst}))
				tex := must(d.NewTexture(gpu.TextureDesc{Size: gpu.Extent3D{Width: 16, Height: 16}, Format: gpu.TextureFormatRGBA8Unorm,
					Usage: gpu.TextureUsageRenderAttachment | gpu.TextureUsageTextureBinding}))
				bg := must(d.NewBindGroup(gpu.BindGroupDesc{Layout: s.pipe.BindGroupLayout(0), Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: ub, Size: 64}}}))
				d.Queue().WriteBuffer(ub, 0, make([]byte, 64))
				enc := d.NewCommandEncoder()
				p := enc.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{View: tex.View(), ClearValue: gpu.Color{A: 1}}}})
				p.SetPipeline(s.pipe)
				p.SetBindGroup(0, bg, 0)
				p.SetBindGroup(1, s.textures)
				p.SetVertexBuffer(0, s.vertices, 0)
				p.Draw(3, 1, 0, 0)
				p.End()
				if err := d.Queue().Submit(enc.Finish()); err != nil {
					panic(err)
				}
				bg.Release()
				tex.Release()
				ub.Release()
			}
		}},
		{"reads and writes of a shared buffer", func(h *harness, s *scene) func(int) {
			return func(i int) {
				d := h.d
				d.Queue().WriteBuffer(s.src, 0, make([]byte, 256))
				enc := d.NewCommandEncoder()
				enc.CopyBufferToBuffer(s.src, 0, s.dst, 0, 256)
				if err := d.Queue().Submit(enc.Finish()); err != nil {
					panic(err)
				}
				s.dst.Read(0, 256, func([]byte, error) {})
				d.Queue().OnSubmittedWorkDone(func() {})
			}
		}},
		{"compute passes", func(h *harness, s *scene) func(int) {
			return func(i int) {
				d := h.d
				enc := d.NewCommandEncoder()
				p := enc.BeginComputePass(gpu.ComputePassDesc{})
				p.SetPipeline(s.compute)
				p.SetBindGroup(0, s.computeData, 0)
				p.DispatchWorkgroups(2, 2, 1)
				p.End()
				if err := d.Queue().Submit(enc.Finish()); err != nil {
					panic(err)
				}
			}
		}},
		{"pipelines made and released", func(h *harness, s *scene) func(int) {
			return func(i int) {
				d := h.d
				m := must(d.NewShaderModule(testShader))
				p := must(d.NewRenderPipeline(h.pipelineDesc(m, nil)))
				c := must(d.NewComputePipeline(gpu.ComputePipelineDesc{Module: m, EntryPoint: "cs"}))
				smp := must(d.NewSampler(gpu.SamplerDesc{}))
				c.Release()
				p.Release()
				smp.Release()
				m.Release()
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := gluttertest.New(t, gluttertest.Opts{GPU: true, DiscardScenes: true})
			rec := ts.GPUDevice()
			dev := rec.Device()
			dev.CompleteReadsOnSubmit, dev.Quiet = true, true
			d, err := gpu.DeviceOfBinding(ts.Binding)
			if err != nil {
				t.Fatal(err)
			}
			h := &harness{t: t, ts: ts, rec: rec, d: d}
			s := newSceneOn(t, h)
			base := d.Resources().Total()
			live := func(f func(fakedevice.Live) int) func() int { return func() int { return f(dev.Live()) } }
			leaktest.Run(t, "gpu/"+tt.name, func() func() {
				frame := tt.frame(h, s)
				i := 0
				return func() {
					frame(i)
					rec.CompleteReads()
					ts.Pump()
					ts.ClearLogs()
					rec.ClearLogs()
					i++
				}
			}, leaktest.Opts{N: 250, Warmup: 20, Counters: map[string]func() int{
				"gpu resources": func() int { return d.Resources().Total() },
				"buffers":       live(func(l fakedevice.Live) int { return l.Buffers }),
				"textures":      live(func(l fakedevice.Live) int { return l.Textures }),
				"samplers":      live(func(l fakedevice.Live) int { return l.Samplers }),
				"pipelines":     live(func(l fakedevice.Live) int { return l.Pipelines }),
			}})
			if got := d.Resources().Total(); got != base {
				t.Errorf("the scenario left %d resources alive, it began with %d", got-base, base)
			}
		})
	}
}
