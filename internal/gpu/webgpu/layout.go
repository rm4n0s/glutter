package webgpu

import (
	"sort"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// bindingPlan is one entry of a bind group layout: a binding of the shader, what it is, and the
// stages that read it.
type bindingPlan struct {
	Binding    int
	Kind       explicit.BindingKind
	Visibility int
	// Dimension and Sample describe a texture binding: the dimension of its view and the sample type
	// the layout declares (SPEC §27.1).
	Dimension explicit.TextureDimension
	Sample    explicit.SampleType
	// StorageFormat is the format a storage texture binding declares.
	StorageFormat explicit.TextureFormat
}

// planLayouts returns the bind group layouts of a pipeline: one for each group from 0 to the
// highest that either entry point reads, each listing the bindings the entry points read, by
// binding number, with the stages that read them (SPEC §9.3: every uniform is bound with a
// dynamic offset, so the layouts are explicit; "auto" layouts cannot have dynamic offsets). A
// group that nothing reads gets an empty layout, so that the group numbers stay what the shader says.
func planLayouts(vs, fs explicit.EntryPoint) [][]bindingPlan {
	return planStages(stagedEntry{vs, shaderVertex}, stagedEntry{fs, shaderFragment})
}

// stagedEntry is an entry point and the GPUShaderStage flag of its stage.
type stagedEntry struct {
	entry explicit.EntryPoint
	stage int
}

// planStages is planLayouts for the entry points of any stages: the two of a render pipeline, or the
// one of a compute pipeline.
func planStages(entries ...stagedEntry) [][]bindingPlan {
	type key struct{ group, binding int }
	seen := map[key]*bindingPlan{}
	highest := -1
	for _, se := range entries {
		for _, b := range se.entry.Bindings {
			k := key{b.Group, b.Binding}
			p := seen[k]
			if p == nil {
				p = &bindingPlan{Binding: b.Binding, Kind: b.Kind, Dimension: b.Dimension, Sample: b.Sample, StorageFormat: b.StorageFormat}
				seen[k] = p
			}
			p.Visibility |= se.stage
			highest = max(highest, b.Group)
		}
	}
	groups := make([][]bindingPlan, highest+1)
	for k, p := range seen {
		groups[k.group] = append(groups[k.group], *p)
	}
	for _, g := range groups {
		sort.Slice(g, func(i, j int) bool { return g[i].Binding < g[j].Binding })
	}
	return groups
}

// bindingsByNumber returns, for each group of a plan, the binding by number: what NewBindGroup
// checks the entries against.
func bindingsByNumber(plan [][]bindingPlan) []map[int]bindingPlan {
	out := make([]map[int]bindingPlan, len(plan))
	for g, bs := range plan {
		out[g] = map[int]bindingPlan{}
		for _, b := range bs {
			out[g][b.Binding] = b
		}
	}
	return out
}
