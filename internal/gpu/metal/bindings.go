package metal

import (
	"sort"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// argClass is the argument table of Metal that a binding is in.
type argClass int

const (
	classBuffer argClass = iota
	classTexture
	classSampler
)

// classOf says which argument table a kind of binding is in: the uniform and storage buffers are buffers,
// the sampled and storage textures textures, the samplers samplers (SPEC §29.5).
func classOf(k explicit.BindingKind) argClass {
	switch k {
	case explicit.BindingSampledTexture, explicit.BindingStorageTexture:
		return classTexture
	case explicit.BindingSampler:
		return classSampler
	}
	return classBuffer
}

// stages say which stages of a pipeline read a binding.
type stages uint8

const (
	stageVertex stages = 1 << iota
	stageFragment
	stageCompute
)

// slot is where one binding of a group goes: its place in the argument table of its class and the stages
// that read it.
type slot struct {
	binding int
	decl    explicit.ShaderBinding
	class   argClass
	index   int
	stages  stages
}

// groupTable is the slots of one bind group of a pipeline, in the order of their binding numbers.
type groupTable struct {
	slots []slot
	// uniforms are the binding numbers of the uniform buffers, in order: SetBindGroup gives one dynamic
	// offset to each.
	uniforms []int
}

// bindingTable is the bind groups of a pipeline, found from the entry points that the pipeline names.
type bindingTable struct {
	groups map[int]*groupTable
}

// entryStage is an entry point and the stage that runs it in the pipeline.
type entryStage struct {
	name  string
	stage stages
}

// newBindingTable reads the bindings of the entry points of a shader. A binding that two entry points both
// read is one slot read by both stages; the two must have the same index, which the generator's module-wide
// map guarantees and this checks. It fails with MetalBindingsInvalid for an entry point the shader lacks,
// two bindings that share an index of the same class, a binding declared differently by the two stages, and
// a buffer index that reaches the vertex buffers.
func newBindingTable(sh *explicit.Shader, entries ...entryStage) (*bindingTable, error) {
	t := &bindingTable{groups: map[int]*groupTable{}}
	byKey := map[[2]int]*slot{}
	used := map[[2]int]string{} // class and index to the binding that has it
	var order [][2]int
	for _, es := range entries {
		e, ok := sh.Entry(es.name)
		if !ok {
			return nil, errors.New(tagMetalBindingsInvalid, "the shader has no such entry point", "shader", sh.Name, "entry", es.name)
		}
		for _, b := range e.Bindings {
			key := [2]int{b.Group, b.Binding}
			if s, ok := byKey[key]; ok {
				if s.decl.Kind != b.Kind || s.index != b.MetalIndex {
					return nil, errors.New(tagMetalBindingsInvalid, "two entry points declare a binding differently",
						"shader", sh.Name, "group", b.Group, "binding", b.Binding)
				}
				s.stages |= es.stage
				continue
			}
			class := classOf(b.Kind)
			if class == classBuffer && b.MetalIndex >= explicit.MetalVertexBufferBase {
				return nil, errors.New(tagMetalBindingsInvalid, "a buffer binding has an index that is in the range of the vertex buffers",
					"shader", sh.Name, "binding", b.Name, "index", b.MetalIndex)
			}
			ck := [2]int{int(class), b.MetalIndex}
			if other, ok := used[ck]; ok {
				return nil, errors.New(tagMetalBindingsInvalid, "two bindings have the same Metal index",
					"shader", sh.Name, "bindings", other+","+b.Name, "index", b.MetalIndex)
			}
			used[ck] = b.Name
			byKey[key] = &slot{binding: b.Binding, decl: b, class: class, index: b.MetalIndex, stages: es.stage}
			order = append(order, key)
		}
	}
	for _, key := range order {
		g := t.groups[key[0]]
		if g == nil {
			g = &groupTable{}
			t.groups[key[0]] = g
		}
		g.slots = append(g.slots, *byKey[key])
	}
	for _, g := range t.groups {
		sort.Slice(g.slots, func(i, j int) bool { return g.slots[i].binding < g.slots[j].binding })
		for _, s := range g.slots {
			if s.decl.Kind == explicit.BindingUniformBuffer {
				g.uniforms = append(g.uniforms, s.binding)
			}
		}
	}
	return t, nil
}

// group returns the table of a bind group, nil when the pipeline has none.
func (t *bindingTable) group(n int) *groupTable { return t.groups[n] }

// has reports whether the group has a binding with the number.
func (g *groupTable) has(binding int) bool {
	for _, s := range g.slots {
		if s.binding == binding {
			return true
		}
	}
	return false
}
