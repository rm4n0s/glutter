package webgpu

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

func entryOf(bindings ...explicit.ShaderBinding) explicit.EntryPoint {
	return explicit.EntryPoint{Bindings: bindings}
}

func TestPlanLayouts(t *testing.T) {
	uniform := func(group, binding int) explicit.ShaderBinding {
		return explicit.ShaderBinding{Group: group, Binding: binding, Kind: explicit.BindingUniformBuffer}
	}
	texture := func(group, binding int) explicit.ShaderBinding {
		return explicit.ShaderBinding{Group: group, Binding: binding, Kind: explicit.BindingSampledTexture}
	}
	sampler := func(group, binding int) explicit.ShaderBinding {
		return explicit.ShaderBinding{Group: group, Binding: binding, Kind: explicit.BindingSampler}
	}
	tests := []struct {
		name   string
		vs, fs explicit.EntryPoint
		want   [][]bindingPlan
	}{
		{"nothing is read", entryOf(), entryOf(), [][]bindingPlan{}},
		{"the blit: a texture and a sampler in the fragment stage", entryOf(), entryOf(texture(0, 0), sampler(0, 1)),
			[][]bindingPlan{{{0, explicit.BindingSampledTexture, shaderFragment}, {1, explicit.BindingSampler, shaderFragment}}}},
		{"a binding both stages read is visible to both", entryOf(uniform(0, 0)), entryOf(uniform(0, 0), uniform(0, 1)),
			[][]bindingPlan{{{0, explicit.BindingUniformBuffer, shaderVertex | shaderFragment}, {1, explicit.BindingUniformBuffer, shaderFragment}}}},
		{"bindings are sorted by number", entryOf(uniform(0, 2), uniform(0, 0)), entryOf(),
			[][]bindingPlan{{{0, explicit.BindingUniformBuffer, shaderVertex}, {2, explicit.BindingUniformBuffer, shaderVertex}}}},
		{"a group nothing reads is empty, so the numbers stay", entryOf(), entryOf(uniform(0, 0), texture(3, 1)),
			[][]bindingPlan{{{0, explicit.BindingUniformBuffer, shaderFragment}}, nil, nil, {{1, explicit.BindingSampledTexture, shaderFragment}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := planLayouts(tt.vs, tt.fs)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

// TestPlanLayoutsOfTheGeneratedShaders plans every pipeline of the renderer's shaders: each group
// of the plan lists the bindings of the entry points, and a uniform is read by the stage that the
// shader says. The renderer's own use of them is in the browser tests.
func TestPlanLayoutsOfTheGeneratedShaders(t *testing.T) {
	for _, s := range explicit.Shaders() {
		vs, ok1 := s.Entry("vs_main")
		fs, ok2 := s.Entry("fs_main")
		if !ok1 || !ok2 {
			t.Errorf("shader %s has no vs_main and fs_main", s.Name)
			continue
		}
		plan := planLayouts(vs, fs)
		want := 0
		for _, e := range []explicit.EntryPoint{vs, fs} {
			for _, b := range e.Bindings {
				want = max(want, b.Group+1)
			}
		}
		if len(plan) != want {
			t.Errorf("shader %s: %d groups planned, the entry points use %d", s.Name, len(plan), want)
		}
		kinds := bindingKinds(plan)
		for _, e := range []explicit.EntryPoint{vs, fs} {
			for _, b := range e.Bindings {
				if kinds[b.Group][b.Binding] != b.Kind {
					t.Errorf("shader %s: group %d binding %d has kind %v in the plan, %v in the shader", s.Name, b.Group, b.Binding, kinds[b.Group][b.Binding], b.Kind)
				}
			}
		}
		for g, bs := range plan {
			for i := 1; i < len(bs); i++ {
				if bs[i-1].Binding >= bs[i].Binding {
					t.Errorf("shader %s group %d: bindings are not in order", s.Name, g)
				}
			}
		}
	}
}
