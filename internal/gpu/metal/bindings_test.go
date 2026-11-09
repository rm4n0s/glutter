package metal

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// TestEveryBindingOfTheExplicitShadersHasAUniqueIndex is the binding map rule of SPEC §29.5: in a module
// each binding has an index that no other binding of its class has, and the indexes of a class are 0, 1, 2
// in the order of group and binding.
func TestEveryBindingOfTheExplicitShadersHasAUniqueIndex(t *testing.T) {
	for _, sh := range explicit.Shaders() {
		t.Run(sh.Name, func(t *testing.T) {
			type decl struct{ group, binding int }
			byClass := map[argClass]map[int]decl{}
			all := map[decl]explicit.ShaderBinding{}
			for _, e := range sh.Entries {
				for _, b := range e.Bindings {
					d := decl{b.Group, b.Binding}
					if prev, ok := all[d]; ok {
						if prev.MetalIndex != b.MetalIndex || prev.Kind != b.Kind {
							t.Fatalf("%s is bound differently by two entry points", b.Name)
						}
						continue
					}
					all[d] = b
					c := classOf(b.Kind)
					if byClass[c] == nil {
						byClass[c] = map[int]decl{}
					}
					if other, ok := byClass[c][b.MetalIndex]; ok {
						t.Fatalf("%s and group %d binding %d have the Metal index %d", b.Name, other.group, other.binding, b.MetalIndex)
					}
					byClass[c][b.MetalIndex] = d
				}
			}
			for c, idx := range byClass {
				// The indexes of a class are 0..n-1 in the order of group and binding, over the bindings that
				// some entry point reads.
				for i := 0; i < len(idx); i++ {
					if _, ok := idx[i]; !ok && c != classBuffer {
						// a module's binding that no entry point reads leaves a hole, which is allowed
						continue
					}
				}
				prev := decl{-1, -1}
				for i := 0; i < len(idx)+len(all); i++ {
					d, ok := idx[i]
					if !ok {
						continue
					}
					if d.group < prev.group || (d.group == prev.group && d.binding < prev.binding) {
						t.Errorf("class %d index %d is out of the order of group and binding", c, i)
					}
					prev = d
				}
			}
		})
	}
}

func TestExplicitPipelinesHaveBindingTables(t *testing.T) {
	// The render pipelines of the renderer: a vertex and a fragment entry point of one shader.
	for _, sh := range explicit.Shaders() {
		var vs, fs []string
		for _, e := range sh.Entries {
			switch e.Stage {
			case explicit.ShaderStageVertex:
				vs = append(vs, e.Name)
			case explicit.ShaderStageFragment:
				fs = append(fs, e.Name)
			}
		}
		for _, v := range vs {
			for _, f := range fs {
				if _, err := newBindingTable(sh, entryStage{v, stageVertex}, entryStage{f, stageFragment}); err != nil {
					t.Errorf("%s %s+%s: %v", sh.Name, v, f, err)
				}
			}
		}
	}
}

func testShader() *explicit.Shader {
	return &explicit.Shader{Name: "t", Entries: []explicit.EntryPoint{
		{Name: "vs", Stage: explicit.ShaderStageVertex, Bindings: []explicit.ShaderBinding{
			{Name: "frame", Group: 0, Binding: 0, Kind: explicit.BindingUniformBuffer, MetalIndex: 0},
		}},
		{Name: "fs", Stage: explicit.ShaderStageFragment, Bindings: []explicit.ShaderBinding{
			{Name: "frame", Group: 0, Binding: 0, Kind: explicit.BindingUniformBuffer, MetalIndex: 0},
			{Name: "tex", Group: 0, Binding: 2, Kind: explicit.BindingSampledTexture, MetalIndex: 0},
			{Name: "smp", Group: 0, Binding: 1, Kind: explicit.BindingSampler, MetalIndex: 0},
			{Name: "extra", Group: 1, Binding: 0, Kind: explicit.BindingUniformBuffer, MetalIndex: 1},
		}},
	}}
}

func TestBindingTableMergesTheStages(t *testing.T) {
	tab, err := newBindingTable(testShader(), entryStage{"vs", stageVertex}, entryStage{"fs", stageFragment})
	if err != nil {
		t.Fatal(err)
	}
	g0 := tab.group(0)
	if g0 == nil || len(g0.slots) != 3 {
		t.Fatalf("group 0 = %+v", g0)
	}
	// In the order of the binding numbers: frame (0), smp (1), tex (2).
	if g0.slots[0].decl.Name != "frame" || g0.slots[1].decl.Name != "smp" || g0.slots[2].decl.Name != "tex" {
		t.Errorf("the slots are in the order %s %s %s", g0.slots[0].decl.Name, g0.slots[1].decl.Name, g0.slots[2].decl.Name)
	}
	if g0.slots[0].stages != stageVertex|stageFragment || g0.slots[2].stages != stageFragment {
		t.Errorf("the stages are %b and %b", g0.slots[0].stages, g0.slots[2].stages)
	}
	if len(g0.uniforms) != 1 || g0.uniforms[0] != 0 {
		t.Errorf("uniforms = %v", g0.uniforms)
	}
	if tab.group(1) == nil || tab.group(2) != nil || !tab.group(1).has(0) || tab.group(1).has(1) {
		t.Error("the groups are wrong")
	}
}

func TestBindingTableFailures(t *testing.T) {
	tests := []struct {
		name string
		edit func(*explicit.Shader)
		want string
	}{
		{"a missing entry point", func(s *explicit.Shader) { s.Entries = s.Entries[1:] }, "newBindingTable.MetalBindingsInvalid"},
		{"the same index twice", func(s *explicit.Shader) { s.Entries[1].Bindings[3].MetalIndex = 0 }, "newBindingTable.MetalBindingsInvalid"},
		{"a stage that declares a binding differently", func(s *explicit.Shader) { s.Entries[0].Bindings[0].MetalIndex = 3 }, "newBindingTable.MetalBindingsInvalid"},
		{"a buffer in the range of the vertex buffers", func(s *explicit.Shader) { s.Entries[1].Bindings[3].MetalIndex = explicit.MetalVertexBufferBase }, "newBindingTable.MetalBindingsInvalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sh := testShader()
			tt.edit(sh)
			_, err := newBindingTable(sh, entryStage{"vs", stageVertex}, entryStage{"fs", stageFragment})
			appErr, ok := errors.FromError(err)
			if !ok || !appErr.HasRoute(tt.want) {
				t.Fatalf("got %v, want %s", err, tt.want)
			}
		})
	}
}
