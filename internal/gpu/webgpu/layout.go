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
}

// planLayouts returns the bind group layouts of a pipeline: one for each group from 0 to the
// highest that either entry point reads, each listing the bindings the entry points read, by
// binding number, with the stages that read them (SPEC §9.3: every uniform is bound with a
// dynamic offset, so the layouts are explicit; "auto" layouts cannot have dynamic offsets). A
// group that nothing reads gets an empty layout, so that the group numbers stay what the shader says.
func planLayouts(vs, fs explicit.EntryPoint) [][]bindingPlan {
	type key struct{ group, binding int }
	seen := map[key]*bindingPlan{}
	highest := -1
	add := func(e explicit.EntryPoint, stage int) {
		for _, b := range e.Bindings {
			k := key{b.Group, b.Binding}
			p := seen[k]
			if p == nil {
				p = &bindingPlan{Binding: b.Binding, Kind: b.Kind}
				seen[k] = p
			}
			p.Visibility |= stage
			highest = max(highest, b.Group)
		}
	}
	add(vs, shaderVertex)
	add(fs, shaderFragment)
	groups := make([][]bindingPlan, highest+1)
	for k, p := range seen {
		groups[k.group] = append(groups[k.group], *p)
	}
	for _, g := range groups {
		sort.Slice(g, func(i, j int) bool { return g[i].Binding < g[j].Binding })
	}
	return groups
}

// bindingKinds returns, for each group of a plan, the kind of each binding by number: what
// NewBindGroup checks the entries against.
func bindingKinds(plan [][]bindingPlan) []map[int]explicit.BindingKind {
	kinds := make([]map[int]explicit.BindingKind, len(plan))
	for g, bs := range plan {
		kinds[g] = map[int]explicit.BindingKind{}
		for _, b := range bs {
			kinds[g][b.Binding] = b.Kind
		}
	}
	return kinds
}
