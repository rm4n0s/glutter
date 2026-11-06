package shaders

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// Kind is what a binding is, as the generated code spells it.
type Kind string

// The kinds of binding the design limits allow (SPEC §9.3).
const (
	KindUniform Kind = "BindingUniformBuffer"
	KindTexture Kind = "BindingSampledTexture"
	KindSampler Kind = "BindingSampler"
	// The kinds of step 30.2 (SPEC §27.1), which only a module generated with -gpu may have.
	KindStorageRO      Kind = "BindingStorageBufferReadOnly"
	KindStorage        Kind = "BindingStorageBuffer"
	KindStorageTexture Kind = "BindingStorageTexture"
)

// Binding is one `@group(g) @binding(b) var` declaration of a module.
type Binding struct {
	Name           string
	Group, Binding int
	Kind           Kind
	// Dim is the dimension of a texture, as the generated code spells it ("" is a 2D texture).
	Dim string
	// Sample is the sample type of a texture, as the generated code spells it ("" is SampleTypeFloat).
	Sample string
	// StorageFormat is the texel format of a storage texture, as the generated code spells it
	// ("TextureFormatRGBA8Unorm").
	StorageFormat string
	// Size is the size in bytes of the type of a uniform or storage buffer (gpushaders only; 0 otherwise):
	// by the layout rules of WGSL, with one element for an array that has no length.
	Size int
}

// The dimensions of a texture binding, as explicit spells them (the zero value is "").
const (
	dimArray = "TextureDim2DArray"
	dimCube  = "TextureDimCube"
	dim3D    = "TextureDim3D"
)

// The sample types of a texture binding that are not the zero value.
const (
	sampleUnfilterable = "SampleTypeUnfilterableFloat"
	sampleUint         = "SampleTypeUint"
)

// Entry is an entry point of a module and the bindings it reads.
type Entry struct {
	Name     string
	Stage    string // "Vertex", "Fragment" or "Compute"
	Bindings []Binding
	// WorkgroupSize is the @workgroup_size of a compute entry point, the missing dimensions 1.
	WorkgroupSize [3]int
	// ReadsInstanceIndex and ReadsVertexIndex say that a vertex entry point takes
	// @builtin(instance_index) or @builtin(vertex_index).
	ReadsInstanceIndex, ReadsVertexIndex bool
	// Inputs are the @location inputs of a vertex entry point (gpushaders only).
	Inputs []Input
}

// Module is what the generator reads from a WGSL file.
type Module struct {
	Bindings []Binding
	Entries  []Entry
}

var (
	lineComment  = regexp.MustCompile(`//[^\n]*`)
	blockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	// A binding declaration has its two attributes in either order, then var.
	bindingDecl = regexp.MustCompile(`((?:@(?:group|binding)\(\s*\d+\s*\)\s*){2})var\s*(?:<\s*([\w\s,]*?)\s*>)?\s*(\w+)\s*:\s*([^;]+);`)
	// A texture type: texture_2d<f32>, texture_2d_array<u32>, texture_cube<f32>, texture_3d<f32>.
	textureType = regexp.MustCompile(`^texture_(2d_array|2d|cube|3d)<(f32|u32)>$`)
	groupAttr   = regexp.MustCompile(`@group\(\s*(\d+)\s*\)`)
	bindAttr    = regexp.MustCompile(`@binding\(\s*(\d+)\s*\)`)
	// A function with its attributes before it, such as "@compute @workgroup_size(8, 8) fn main".
	entryDecl     = regexp.MustCompile(`((?:@\w+(?:\([^)]*\))?\s*)+)fn\s+(\w+)`)
	stageAttr     = regexp.MustCompile(`@(vertex|fragment|compute)\b`)
	workgroupAttr = regexp.MustCompile(`@workgroup_size\(([^)]*)\)`)
	structDecl    = regexp.MustCompile(`struct\s+(\w+)\s*\{([^}]*)\}`)
	// A storage texture type: texture_storage_2d<rgba8unorm, write>.
	storageTextureType = regexp.MustCompile(`^texture_storage_(2d_array|2d|3d)<(rgba8unorm|rgba16float|r32float|r32uint),write>$`)
	fnDecl             = regexp.MustCompile(`\bfn\s+(\w+)\s*\(`)
)

// stripComments removes the comments of WGSL, keeping the line breaks.
func stripComments(src string) string {
	src = blockComment.ReplaceAllStringFunc(src, func(s string) string {
		return strings.Repeat("\n", strings.Count(s, "\n"))
	})
	return lineComment.ReplaceAllString(src, "")
}

// Parse reads the bindings and the entry points of a WGSL module: the declarations
// `@group(g) @binding(b) var<uniform> x: T;`, `var t: texture_2d<f32>;` and
// `var s: sampler;`, and the functions marked @vertex or @fragment. An entry point
// reads a binding when its name is in the body of the entry point or of a function
// it calls. A binding of another kind (a texture of another type), a compute entry
// point, or two declarations of one group and binding, is an error.
//
// With gpu (glutter-gen shaders -gpu, the shaders of step 30.2 and later) the module may also have
// `var<storage, read>` and `var<storage, read_write>` buffers, `texture_storage_2d`, `_2d_array` and
// `_3d` textures of rgba8unorm, rgba16float, r32float and r32uint that are written, and @compute entry
// points with a literal @workgroup_size; the 2D renderer's own shaders have none of these.
func Parse(src string, gpu bool) (Module, error) { return parse(src, gpu, false) }

// parse is Parse; app is the shaders of an app (glutter-gen gpushaders): it implies gpu, adds the sizes of
// the buffers and the inputs of the vertex entry points, and gives the errors the tags of gpushaders.
func parse(src string, gpu, app bool) (Module, error) {
	src = stripComments(src)
	gpu = gpu || app
	lay := newLayouts(src)
	var m Module
	seen := map[[2]int]string{}
	for _, d := range bindingDecl.FindAllStringSubmatch(src, -1) {
		group, _ := strconv.Atoi(groupAttr.FindStringSubmatch(d[1])[1])
		binding, _ := strconv.Atoi(bindAttr.FindStringSubmatch(d[1])[1])
		b := Binding{Name: d[3], Group: group, Binding: binding}
		typ := strings.Join(strings.Fields(d[4]), "")
		space := strings.Join(strings.Fields(d[2]), "")
		switch {
		case space == "uniform":
			b.Kind = KindUniform
		case space == "" && textureType.MatchString(typ):
			m := textureType.FindStringSubmatch(typ)
			b.Kind = KindTexture
			switch m[1] {
			case "2d_array":
				b.Dim = dimArray
			case "cube":
				b.Dim = dimCube
			case "3d":
				b.Dim = dim3D
			}
			if m[2] == "u32" {
				b.Sample = sampleUint
			} else if !filteredRead(src, b.Name) {
				b.Sample = sampleUnfilterable
			}
		case space == "" && typ == "sampler":
			b.Kind = KindSampler
		case gpu && (space == "storage" || space == "storage,read"):
			b.Kind = KindStorageRO
		case gpu && space == "storage,read_write":
			b.Kind = KindStorage
		case gpu && space == "" && storageTextureType.MatchString(typ):
			m := storageTextureType.FindStringSubmatch(typ)
			b.Kind = KindStorageTexture
			switch m[1] {
			case "2d_array":
				b.Dim = dimArray
			case "3d":
				b.Dim = dim3D
			}
			b.StorageFormat = storageFormats[m[2]]
		default:
			if app {
				return Module{}, errors.New(tagGPUShadersBindingUnsupported, "only uniform buffers, storage buffers (read and read_write), sampled textures (texture_2d, texture_2d_array, texture_cube and texture_3d of f32 or u32), storage textures (texture_storage_2d, _2d_array and _3d of rgba8unorm, rgba16float, r32float and r32uint, write) and sampler can be bound (SPEC §27.3)",
					"name", b.Name, "address_space", d[2], "type", typ)
			}
			return Module{}, errors.New(tagShadersUnsupportedBinding, "only uniform buffers, sampled textures (texture_2d, texture_2d_array, texture_cube and texture_3d of f32 or u32) and sampler can be bound (SPEC §9.3, §27.1); storage buffers and storage textures need -gpu",
				"name", b.Name, "address_space", d[2], "type", typ)
		}
		if app && (b.Kind == KindUniform || b.Kind == KindStorageRO || b.Kind == KindStorage) {
			size, err := lay.bindingSize(typ)
			if err != nil {
				return Module{}, err
			}
			b.Size = size
		}
		key := [2]int{group, binding}
		if other, ok := seen[key]; ok {
			return Module{}, errors.New(tagShadersBindingTwice, "two variables have the same group and binding",
				"group", group, "binding", binding, "names", other+","+b.Name)
		}
		seen[key] = b.Name
		m.Bindings = append(m.Bindings, b)
	}
	sort.Slice(m.Bindings, func(i, j int) bool {
		a, b := m.Bindings[i], m.Bindings[j]
		return a.Group < b.Group || (a.Group == b.Group && a.Binding < b.Binding)
	})

	bodies, err := functionBodies(src)
	if err != nil {
		return Module{}, err
	}
	structs := map[string]string{}
	for _, st := range structDecl.FindAllStringSubmatch(src, -1) {
		structs[st[1]] = st[2]
	}
	for _, e := range entryDecl.FindAllStringSubmatch(src, -1) {
		stage := stageAttr.FindStringSubmatch(e[1])
		if stage == nil {
			continue
		}
		name := e[2]
		if stage[1] == "compute" && !gpu {
			return Module{}, errors.New(tagShadersComputeEntry, "there is no compute in the 2D renderer's shaders; use -gpu (SPEC §9.3, §27.1)", "entry", name)
		}
		entry := Entry{Name: name, Stage: strings.ToUpper(stage[1][:1]) + stage[1][1:]}
		if stage[1] == "compute" {
			size, err := workgroupSize(e[1], name)
			if err != nil {
				return Module{}, err
			}
			entry.WorkgroupSize = size
		}
		if stage[1] == "vertex" {
			entry.ReadsInstanceIndex = takesBuiltin(bodies[name], structs, "instance_index")
			entry.ReadsVertexIndex = takesBuiltin(bodies[name], structs, "vertex_index")
			if app {
				head := bodies[name]
				if i := strings.IndexByte(head, '{'); i >= 0 {
					head = head[:i]
				}
				inputs, err := lay.inputs(head)
				if err != nil {
					return Module{}, err
				}
				entry.Inputs = inputs
			}
		}
		reach := reachable(entry.Name, bodies)
		for _, b := range m.Bindings {
			re := regexp.MustCompile(`\b` + regexp.QuoteMeta(b.Name) + `\b`)
			for _, fn := range reach {
				if re.MatchString(bodies[fn]) {
					entry.Bindings = append(entry.Bindings, b)
					break
				}
			}
		}
		m.Entries = append(m.Entries, entry)
	}
	if len(m.Entries) == 0 {
		return Module{}, errors.New(tagShadersNoEntry, "the module has no @vertex, @fragment or @compute function")
	}
	return m, nil
}

// storageFormats maps the WGSL names of the formats of a storage texture to explicit's.
var storageFormats = map[string]string{
	"rgba8unorm":  "TextureFormatRGBA8Unorm",
	"rgba16float": "TextureFormatRGBA16Float",
	"r32float":    "TextureFormatR32Float",
	"r32uint":     "TextureFormatR32Uint",
}

// workgroupSize reads the literal sizes of a @workgroup_size attribute among the attributes of an entry
// point; the missing ones are 1. It fails with ShadersWorkgroupSize for a compute entry point without
// the attribute and for a size that is not a literal (an override or a const: the generator could not
// check the limits).
func workgroupSize(attrs, entry string) ([3]int, error) {
	m := workgroupAttr.FindStringSubmatch(attrs)
	if m == nil {
		return [3]int{}, errors.New(tagShadersWorkgroupSize, "a compute entry point needs @workgroup_size", "entry", entry)
	}
	size := [3]int{1, 1, 1}
	parts := strings.Split(strings.TrimSuffix(strings.TrimSpace(m[1]), ","), ",")
	if len(parts) > 3 {
		return [3]int{}, errors.New(tagShadersWorkgroupSize, "@workgroup_size has at most three sizes", "entry", entry, "attribute", m[0])
	}
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || n < 1 {
			return [3]int{}, errors.New(tagShadersWorkgroupSize, "a workgroup size is a positive integer literal", "entry", entry, "attribute", m[0])
		}
		size[i] = n
	}
	return size, nil
}

// takesBuiltin reports whether the function (its signature and body) takes @builtin(name), directly
// or in a struct it names.
func takesBuiltin(fn string, structs map[string]string, name string) bool {
	attr := regexp.MustCompile(`@builtin\(\s*` + name + `\s*\)`)
	if attr.MatchString(fn) {
		return true
	}
	head := fn
	if i := strings.IndexByte(fn, '{'); i >= 0 {
		head = fn[:i]
	}
	for st, body := range structs {
		if regexp.MustCompile(`\b`+regexp.QuoteMeta(st)+`\b`).MatchString(head) && attr.MatchString(body) {
			return true
		}
	}
	return false
}

// filteredRead reports whether a texture may be read with a filter: its name is anywhere except as
// the first argument of textureLoad, textureDimensions, textureNumLevels or textureNumLayers (a
// textureSample call, or a function that takes the texture as an argument). A texture that is
// not may be a float32 texture that has no filtering (SampleTypeUnfilterableFloat).
func filteredRead(src, name string) bool {
	use := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
	safe := regexp.MustCompile(`\b(?:textureLoad|textureDimensions|textureNumLevels|textureNumLayers)\s*\(\s*` + regexp.QuoteMeta(name) + `\b`)
	decl := regexp.MustCompile(`\bvar\s+` + regexp.QuoteMeta(name) + `\s*:`)
	return len(use.FindAllStringIndex(src, -1)) != len(safe.FindAllStringIndex(src, -1))+len(decl.FindAllStringIndex(src, -1))
}

// functionBodies maps the name of each function to its signature and body, found
// by matching braces.
func functionBodies(src string) (map[string]string, error) {
	bodies := map[string]string{}
	for _, loc := range fnDecl.FindAllStringSubmatchIndex(src, -1) {
		name := src[loc[2]:loc[3]]
		open := strings.IndexByte(src[loc[1]:], '{')
		if open < 0 {
			return nil, errors.New(tagShadersUnbalancedBraces, "a function has no body", "function", name)
		}
		start := loc[1] + open
		depth := 0
		end := -1
		for i := start; i < len(src) && end < 0; i++ {
			switch src[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					end = i + 1
				}
			}
		}
		if end < 0 {
			return nil, errors.New(tagShadersUnbalancedBraces, "the braces of a function are not closed", "function", name)
		}
		bodies[name] = src[loc[0]:end]
	}
	return bodies, nil
}

// reachable returns the function and the functions it calls, directly or not.
func reachable(name string, bodies map[string]string) []string {
	var out []string
	seen := map[string]bool{}
	var walk func(string)
	walk = func(fn string) {
		if seen[fn] {
			return
		}
		seen[fn] = true
		out = append(out, fn)
		for other := range bodies {
			if other != fn && regexp.MustCompile(`\b`+regexp.QuoteMeta(other)+`\s*\(`).MatchString(strings.TrimPrefix(bodies[fn], "fn "+fn)) {
				walk(other)
			}
		}
	}
	walk(name)
	return out
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// tagGPUShadersBindingUnsupported is the tag of the error of a binding that an app's shader may not have
	// (glutter-gen gpushaders, SPEC §27.3); the metadata are "name", "address_space" and "type".
	tagGPUShadersBindingUnsupported = "GPUShadersBindingUnsupported"
	tagShadersBindingTwice          = "ShadersBindingTwice"
	tagShadersComputeEntry          = "ShadersComputeEntry"
	tagShadersNoEntry               = "ShadersNoEntry"
	tagShadersUnbalancedBraces      = "ShadersUnbalancedBraces"
	tagShadersUnsupportedBinding    = "ShadersUnsupportedBinding"
	tagShadersWorkgroupSize         = "ShadersWorkgroupSize"
)
