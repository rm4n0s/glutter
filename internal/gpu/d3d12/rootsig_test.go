package d3d12

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// entryOf finds the entry point of a shader.
func entryOf(t *testing.T, sh *explicit.Shader, name string) explicit.EntryPoint {
	t.Helper()
	e, ok := sh.Entry(name)
	if !ok {
		t.Fatalf("shader %s has no entry point %s", sh.Name, name)
	}
	return e
}

// TestPlanForEveryShaderOfTheRenderer plans the root signature of the pipeline of each shader (all of
// them have vs_main and fs_main): the spaces are the groups, the registers are the bindings, the root
// arguments are under 20 DWORDs, and the dynamic offsets and tables map to the right parameters
// (SPEC §9.8 "Binding model").
func TestPlanForEveryShaderOfTheRenderer(t *testing.T) {
	limits := explicit.DefaultLimits()
	for _, sh := range explicit.Shaders() {
		t.Run(sh.Name, func(t *testing.T) {
			vs, fs := entryOf(t, sh, "vs_main"), entryOf(t, sh, "fs_main")
			plan, err := planRoot(vs, fs, limits)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Dwords >= 20 {
				t.Errorf("the root arguments are %d DWORDs, SPEC §9.8 says fewer than 20", plan.Dwords)
			}
			// What the entry points declare, by group and binding.
			want := map[[2]int]explicit.BindingKind{}
			for _, e := range []explicit.EntryPoint{vs, fs} {
				for _, b := range e.Bindings {
					want[[2]int{b.Group, b.Binding}] = b.Kind
					if b.Register.Space != b.Group || b.Register.Index != b.Binding {
						t.Errorf("%s: binding %d of group %d is at %v, not register B of space G", b.Name, b.Binding, b.Group, b.Register)
					}
				}
			}
			got := map[[2]int]explicit.BindingKind{}
			dwords := 0
			for i, p := range plan.Params {
				dwords += p.dwords()
				switch p.Kind {
				case rootCBV:
					got[[2]int{p.Group, p.Binding}] = explicit.BindingUniformBuffer
				case rootSRVTable:
					for _, b := range p.Bindings {
						got[[2]int{p.Group, b}] = explicit.BindingSampledTexture
					}
				case rootSamplerTable:
					for _, b := range p.Bindings {
						got[[2]int{p.Group, b}] = explicit.BindingSampler
					}
				}
				if i > 0 && p.Group < plan.Params[i-1].Group {
					t.Errorf("parameter %d is of group %d after one of group %d", i, p.Group, plan.Params[i-1].Group)
				}
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("the parameters cover %v, the shader declares %v", got, want)
			}
			if dwords != plan.Dwords {
				t.Errorf("Dwords = %d, the parameters add up to %d", plan.Dwords, dwords)
			}
			// The dynamic offsets and the tables of each group point at the parameters that hold them.
			for g, gp := range plan.Groups {
				for i, param := range gp.UniformParams {
					p := plan.Params[param]
					if p.Kind != rootCBV || p.Group != g || p.Binding != gp.Uniforms[i] {
						t.Errorf("group %d: offset %d maps to parameter %d = %+v", g, i, param, p)
					}
				}
				if gp.TextureParam >= 0 {
					if p := plan.Params[gp.TextureParam]; p.Kind != rootSRVTable || p.Group != g || !reflect.DeepEqual(p.Bindings, gp.Textures) {
						t.Errorf("group %d: the texture table is parameter %d = %+v", g, gp.TextureParam, p)
					}
				} else if len(gp.Textures) != 0 {
					t.Errorf("group %d has textures and no table", g)
				}
				if gp.SamplerParam >= 0 {
					if p := plan.Params[gp.SamplerParam]; p.Kind != rootSamplerTable || p.Group != g || !reflect.DeepEqual(p.Bindings, gp.Samplers) {
						t.Errorf("group %d: the sampler table is parameter %d = %+v", g, gp.SamplerParam, p)
					}
				} else if len(gp.Samplers) != 0 {
					t.Errorf("group %d has samplers and no table", g)
				}
			}
			// The description has the registers and spaces, and a table per range.
			desc := plan.description()
			if len(desc.Params) != len(plan.Params) || len(desc.Ranges) != len(plan.Params) {
				t.Fatalf("description has %d parameters and %d range lists for %d", len(desc.Params), len(desc.Ranges), len(plan.Params))
			}
			for i, p := range plan.Params {
				dp := desc.Params[i]
				switch p.Kind {
				case rootCBV:
					if dp.ParameterType != rootParameterCBV || dp.Union[0] != uint64(p.Binding)|uint64(p.Group)<<32 {
						t.Errorf("parameter %d: %+v for b%d space%d", i, dp, p.Binding, p.Group)
					}
				default:
					wantType := descriptorRangeSRV
					if p.Kind == rootSamplerTable {
						wantType = descriptorRangeSampler
					}
					if dp.ParameterType != rootParameterTable || dp.Union[0] != uint64(len(p.Bindings)) || len(desc.Ranges[i]) != len(p.Bindings) {
						t.Fatalf("parameter %d: %+v with %d ranges for %v", i, dp, len(desc.Ranges[i]), p.Bindings)
					}
					for k, r := range desc.Ranges[i] {
						if r.RangeType != wantType || r.NumDescriptors != 1 || int(r.BaseShaderRegister) != p.Bindings[k] || int(r.RegisterSpace) != p.Group || r.OffsetInDescriptorsFromTableStart != descriptorRangeAppend {
							t.Errorf("parameter %d range %d = %+v", i, k, r)
						}
					}
				}
				if dp.ShaderVisibility != shaderVisibilityAll {
					t.Errorf("parameter %d is visible to %d", i, dp.ShaderVisibility)
				}
			}
			if desc.Flags&rootFlagAllowInputLayout == 0 {
				t.Errorf("the flags %#x do not allow the input layout", desc.Flags)
			}
		})
	}
}

// TestPlanOfTheShapeShader checks the plan of one shader by hand: shape.wgsl has the frame, draw and paint
// uniforms in group 0, the clip mask in group 1, nothing in group 2, and the image and the copy of the
// destination in group 3.
func TestPlanOfTheShapeShader(t *testing.T) {
	plan, err := planRoot(entryOf(t, explicit.ShaderShape, "vs_main"), entryOf(t, explicit.ShaderShape, "fs_main"), explicit.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	want := []rootParam{
		{Kind: rootCBV, Group: 0, Binding: 0},
		{Kind: rootCBV, Group: 0, Binding: 1},
		{Kind: rootCBV, Group: 0, Binding: 2},
		{Kind: rootSRVTable, Group: 1, Bindings: []int{0}},
		{Kind: rootSRVTable, Group: 3, Bindings: []int{0, 1}},
	}
	if !reflect.DeepEqual(plan.Params, want) {
		t.Errorf("parameters\n got %+v\nwant %+v", plan.Params, want)
	}
	if plan.Dwords != 3*2+1+1 {
		t.Errorf("Dwords = %d, want 8", plan.Dwords)
	}
	if len(plan.Groups) != 4 || plan.Groups[2].TextureParam != -1 || plan.Groups[2].SamplerParam != -1 || len(plan.Groups[2].Uniforms) != 0 {
		t.Errorf("group 2 should be empty: %+v", plan.Groups)
	}
	if got := plan.Groups[0].UniformParams; !reflect.DeepEqual(got, []int{0, 1, 2}) {
		t.Errorf("group 0 offsets map to %v", got)
	}
	if got := plan.Groups[3].TextureParam; got != 4 {
		t.Errorf("group 3 table is parameter %d", got)
	}
	if got, want := plan.Key(), "b0b1b2//|/t0/|//|/t0t1/"; got != want {
		t.Errorf("Key = %q, want %q", got, want)
	}
}

// TestRootSignaturesAreSharedByShape: two pipelines of one shader have one key, and different shaders
// with different bindings have different ones.
func TestRootSignaturesAreSharedByShape(t *testing.T) {
	limits := explicit.DefaultLimits()
	key := func(sh *explicit.Shader) string {
		p, err := planRoot(entryOf(t, sh, "vs_main"), entryOf(t, sh, "fs_main"), limits)
		if err != nil {
			t.Fatal(err)
		}
		return p.Key()
	}
	keys := map[string]string{}
	for _, sh := range explicit.Shaders() {
		k := key(sh)
		if k != key(sh) {
			t.Errorf("%s: two plans of one shader have two keys", sh.Name)
		}
		if other, dup := keys[k]; dup {
			t.Logf("%s and %s share a root signature (%q)", sh.Name, other, k)
		}
		keys[k] = sh.Name
	}
	if len(keys) < 4 {
		t.Errorf("%d shapes for %d shaders", len(keys), len(explicit.Shaders()))
	}
}

func TestPlanErrors(t *testing.T) {
	limits := explicit.DefaultLimits()
	b := func(group, binding int, kind explicit.BindingKind) explicit.ShaderBinding {
		return explicit.ShaderBinding{Name: "x", Group: group, Binding: binding, Kind: kind}
	}
	many := explicit.EntryPoint{Name: "fs"}
	for i := 0; i < 12; i++ { // 12 uniforms in group 0 and 12 in group 1: 48 DWORDs; with 3 more groups > 64
		many.Bindings = append(many.Bindings, b(0, i, explicit.BindingUniformBuffer), b(1, i, explicit.BindingUniformBuffer), b(2, i, explicit.BindingUniformBuffer))
	}
	tests := []struct {
		name   string
		vs, fs explicit.EntryPoint
		limits explicit.Limits
	}{
		{"a binding that is two kinds", explicit.EntryPoint{Name: "vs", Bindings: []explicit.ShaderBinding{b(0, 0, explicit.BindingUniformBuffer)}},
			explicit.EntryPoint{Name: "fs", Bindings: []explicit.ShaderBinding{b(0, 0, explicit.BindingSampledTexture)}}, limits},
		{"a group the limits do not allow", explicit.EntryPoint{Name: "vs", Bindings: []explicit.ShaderBinding{b(4, 0, explicit.BindingUniformBuffer)}}, explicit.EntryPoint{Name: "fs"}, limits},
		{"a negative binding", explicit.EntryPoint{Name: "vs", Bindings: []explicit.ShaderBinding{b(0, -1, explicit.BindingUniformBuffer)}}, explicit.EntryPoint{Name: "fs"}, limits},
		{"a binding without a kind", explicit.EntryPoint{Name: "vs", Bindings: []explicit.ShaderBinding{b(0, 0, explicit.BindingUnspecified)}}, explicit.EntryPoint{Name: "fs"}, limits},
		{"a binding twice in one entry", explicit.EntryPoint{Name: "vs", Bindings: []explicit.ShaderBinding{b(0, 0, explicit.BindingUniformBuffer), b(0, 0, explicit.BindingUniformBuffer)}}, explicit.EntryPoint{Name: "fs"}, limits},
		{"more than 64 DWORDs", explicit.EntryPoint{Name: "vs"}, many, explicit.Limits{MaxBindGroups: 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := planRoot(tt.vs, tt.fs, tt.limits)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != tagD3D12PipelineInvalid || !appErr.HasRoute("planRoot.D3D12PipelineInvalid") {
				t.Errorf("got %s via %s, want D3D12PipelineInvalid via planRoot", appErr.Tag, appErr.Route())
			}
		})
	}
}

func TestSamePlanForTheSameEntryTwice(t *testing.T) {
	// A binding that both entry points read is one parameter, not two.
	u := explicit.ShaderBinding{Name: "frame", Group: 0, Binding: 0, Kind: explicit.BindingUniformBuffer}
	plan, err := planRoot(explicit.EntryPoint{Bindings: []explicit.ShaderBinding{u}}, explicit.EntryPoint{Bindings: []explicit.ShaderBinding{u}}, explicit.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Params) != 1 || plan.Dwords != 2 {
		t.Errorf("plan = %+v", plan)
	}
}
