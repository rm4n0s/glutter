package d3d12

import (
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
)

// The root signature planner (SPEC §9.8 "Binding model"): it turns the bindings of the two entry
// points of a pipeline into the root parameters of its root signature. It is plain Go, tested on
// Linux; rootsig_windows.go serializes the result and makes the object.
//
// Bind group G of the explicit renderer is register space G. In each group the uniform bindings are
// root constant buffer views (one root parameter each, 2 DWORDs, addressed by the dynamic offset of
// SetBindGroup), the sampled textures are the ranges of one descriptor table (1 DWORD) and the
// samplers the ranges of another (1 DWORD), because a table cannot hold both kinds of descriptor
// and the sampler heap is a heap of its own. Every descriptor of a table is valid (resource binding
// tier 1): a slot a shader does not read holds a null descriptor or the renderer's 1x1 stand-in.

// rootKind is the kind of a root parameter.
type rootKind uint8

const (
	// rootCBV is a root constant buffer view: a GPU address in the root arguments.
	rootCBV rootKind = iota + 1
	// rootSRVTable is a descriptor table of shader resource views.
	rootSRVTable
	// rootSamplerTable is a descriptor table of samplers.
	rootSamplerTable
)

// rootParam is one root parameter.
type rootParam struct {
	Kind rootKind
	// Group is the register space.
	Group int
	// Binding is the register of a rootCBV: b<Binding>.
	Binding int
	// Bindings are the registers of a table, in the order of its descriptors: t<B> or s<B>, one
	// range of one descriptor each.
	Bindings []int
}

// dwords is the size of the parameter in the root arguments: 2 for a root descriptor (a 64-bit
// address), 1 for a table.
func (p rootParam) dwords() int {
	if p.Kind == rootCBV {
		return 2
	}
	return 1
}

// groupPlan is where the bindings of one group are in the root parameters.
type groupPlan struct {
	// Uniforms are the binding numbers of the uniform buffers, in order: the dynamic offset number i of
	// SetBindGroup is added to the address of Uniforms[i], which is root parameter UniformParams[i].
	Uniforms      []int
	UniformParams []int
	// Textures are the binding numbers of the textures in the order of the descriptors of the table
	// that is root parameter TextureParam (-1 when the group has no texture); Samplers and
	// SamplerParam are the same for the samplers.
	Textures     []int
	TextureParam int
	Samplers     []int
	SamplerParam int
	// Shape is the key of the group's bindings (see shape).
	Shape string
}

// shape is the key of the group's bindings: bind groups made for one pipeline can be used with another
// whose group has the same shape (the renderer shares the group of one shader between the pipelines
// of its blends).
func (g groupPlan) shape() string {
	var b strings.Builder
	write := func(c byte, ns []int) {
		for _, n := range ns {
			b.WriteByte(c)
			b.WriteString(strconv.Itoa(n))
		}
	}
	write('b', g.Uniforms)
	b.WriteByte('/')
	write('t', g.Textures)
	b.WriteByte('/')
	write('s', g.Samplers)
	return b.String()
}

// rootPlan is the root signature of a pipeline.
type rootPlan struct {
	// Params are the root parameters, group by group: a group's uniforms in binding order, then its
	// texture table, then its sampler table.
	Params []rootParam
	// Groups are indexed by group number, up to the highest group a shader uses; a group no shader
	// uses is empty (its parameters are -1).
	Groups []groupPlan
	// Dwords is the size of the root arguments (at most maxRootDwords).
	Dwords int
}

// maxRootDwords is the size of the root arguments of a root signature, in DWORDs. The renderer's
// pipelines take fewer than 20 (SPEC §9.8).
const maxRootDwords = 64

// planRoot plans the root signature of a pipeline from the bindings of its vertex and fragment entry
// points. It fails with D3D12PipelineInvalid for a binding that the two entry points declare as two
// kinds, a group the limits do not allow, a binding of an unspecified kind or twice in one entry, and root
// arguments of more than 64 DWORDs.
func planRoot(vs, fs explicit.EntryPoint, limits explicit.Limits) (*rootPlan, error) {
	type slot struct{ group, binding int }
	kinds := map[slot]explicit.BindingKind{}
	maxGroup := -1
	for _, e := range []explicit.EntryPoint{vs, fs} {
		seen := map[slot]bool{}
		for _, b := range e.Bindings {
			s := slot{b.Group, b.Binding}
			if b.Group < 0 || b.Group >= limits.MaxBindGroups || b.Binding < 0 {
				return nil, errors.New(tagD3D12PipelineInvalid, "a shader binding is outside the bind groups of the device",
					"entry", e.Name, "group", b.Group, "binding", b.Binding, "max_groups", limits.MaxBindGroups)
			}
			switch b.Kind {
			case explicit.BindingUniformBuffer, explicit.BindingSampledTexture, explicit.BindingSampler:
			default:
				return nil, errors.New(tagD3D12PipelineInvalid, "a shader binding has no kind",
					"entry", e.Name, "group", b.Group, "binding", b.Binding)
			}
			if seen[s] {
				return nil, errors.New(tagD3D12PipelineInvalid, "an entry point lists a binding twice",
					"entry", e.Name, "group", b.Group, "binding", b.Binding)
			}
			seen[s] = true
			if k, ok := kinds[s]; ok && k != b.Kind {
				return nil, errors.New(tagD3D12PipelineInvalid, "the two entry points declare a binding as two kinds",
					"group", b.Group, "binding", b.Binding, "kind", k.String(), "other", b.Kind.String())
			}
			kinds[s] = b.Kind
			maxGroup = max(maxGroup, b.Group)
		}
	}
	plan := &rootPlan{Groups: make([]groupPlan, maxGroup+1)}
	for g := range plan.Groups {
		var uniforms, textures, samplers []int
		for s, k := range kinds {
			if s.group != g {
				continue
			}
			switch k {
			case explicit.BindingUniformBuffer:
				uniforms = append(uniforms, s.binding)
			case explicit.BindingSampledTexture:
				textures = append(textures, s.binding)
			case explicit.BindingSampler:
				samplers = append(samplers, s.binding)
			}
		}
		sort.Ints(uniforms)
		sort.Ints(textures)
		sort.Ints(samplers)
		gp := groupPlan{Uniforms: uniforms, Textures: textures, Samplers: samplers, TextureParam: -1, SamplerParam: -1}
		for _, b := range uniforms {
			gp.UniformParams = append(gp.UniformParams, len(plan.Params))
			plan.Params = append(plan.Params, rootParam{Kind: rootCBV, Group: g, Binding: b})
		}
		if len(textures) > 0 {
			gp.TextureParam = len(plan.Params)
			plan.Params = append(plan.Params, rootParam{Kind: rootSRVTable, Group: g, Bindings: textures})
		}
		if len(samplers) > 0 {
			gp.SamplerParam = len(plan.Params)
			plan.Params = append(plan.Params, rootParam{Kind: rootSamplerTable, Group: g, Bindings: samplers})
		}
		gp.Shape = gp.shape()
		plan.Groups[g] = gp
	}
	for _, p := range plan.Params {
		plan.Dwords += p.dwords()
	}
	if plan.Dwords > maxRootDwords {
		return nil, errors.New(tagD3D12PipelineInvalid, "the root arguments of the pipeline are larger than a root signature may be",
			"dwords", plan.Dwords, "max", maxRootDwords)
	}
	return plan, nil
}

// Key names the shape of the root signature: two pipelines whose bindings give the same key share one
// root signature object (SPEC §9.8).
func (p *rootPlan) Key() string {
	var b strings.Builder
	for g, gp := range p.Groups {
		if g > 0 {
			b.WriteByte('|')
		}
		b.WriteString(gp.Shape)
	}
	return b.String()
}

// rootDesc is the description of a root signature in the structures of D3D12_ROOT_SIGNATURE_DESC: the
// parameters and, for each table parameter, its descriptor ranges. The pointers in the parameters
// are not set (a Go slice can move on the stack): the serializer sets them.
type rootDesc struct {
	Params []d3d12RootParameter
	// Ranges[i] are the ranges of parameter i; nil for a root descriptor.
	Ranges [][]d3d12DescriptorRange
	Flags  uint32
}

// description builds the structures of the plan (root signature version 1.0): a root constant buffer
// view for each uniform, the ranges of each table (one descriptor per range, in the order of the table),
// all visible to every stage, and the flags that let the input assembler read the vertex buffers and
// deny the stages that are not used.
func (p *rootPlan) description() rootDesc {
	d := rootDesc{Flags: rootFlagAllowInputLayout | rootFlagDenyHull | rootFlagDenyDomain | rootFlagDenyGeometry}
	for _, rp := range p.Params {
		param := d3d12RootParameter{ShaderVisibility: shaderVisibilityAll}
		var ranges []d3d12DescriptorRange
		switch rp.Kind {
		case rootCBV:
			param.ParameterType = rootParameterCBV
			// D3D12_ROOT_DESCRIPTOR {ShaderRegister, RegisterSpace}: the two words of the union's start.
			param.Union[0] = uint64(uint32(rp.Binding)) | uint64(uint32(rp.Group))<<32
		case rootSRVTable, rootSamplerTable:
			param.ParameterType = rootParameterTable
			typ := descriptorRangeSRV
			if rp.Kind == rootSamplerTable {
				typ = descriptorRangeSampler
			}
			for _, b := range rp.Bindings {
				ranges = append(ranges, d3d12DescriptorRange{
					RangeType: typ, NumDescriptors: 1, BaseShaderRegister: uint32(b), RegisterSpace: uint32(rp.Group),
					OffsetInDescriptorsFromTableStart: descriptorRangeAppend,
				})
			}
			param.Union[0] = uint64(uint32(len(ranges))) // NumDescriptorRanges; the pointer is set by the serializer
		}
		d.Params = append(d.Params, param)
		d.Ranges = append(d.Ranges, ranges)
	}
	return d
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagD3D12PipelineInvalid is the tag of the error for a pipeline that cannot be made from its
	// description: no shader, an entry point the shader lacks, bindings that do not fit a root signature,
	// or an unknown target format. The metadata name what is wrong.
	tagD3D12PipelineInvalid = "D3D12PipelineInvalid"
)
