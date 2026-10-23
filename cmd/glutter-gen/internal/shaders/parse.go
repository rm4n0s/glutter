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
)

// Binding is one `@group(g) @binding(b) var` declaration of a module.
type Binding struct {
	Name           string
	Group, Binding int
	Kind           Kind
}

// Entry is an entry point of a module and the bindings it reads.
type Entry struct {
	Name     string
	Stage    string // "Vertex" or "Fragment"
	Bindings []Binding
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
	groupAttr   = regexp.MustCompile(`@group\(\s*(\d+)\s*\)`)
	bindAttr    = regexp.MustCompile(`@binding\(\s*(\d+)\s*\)`)
	entryDecl   = regexp.MustCompile(`@(vertex|fragment|compute)\s+(?:@\w+(?:\([^)]*\))?\s+)*fn\s+(\w+)`)
	fnDecl      = regexp.MustCompile(`\bfn\s+(\w+)\s*\(`)
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
// it calls. A binding of another kind (a storage buffer, a texture of another type),
// a compute entry point, or two declarations of one group and binding, is an error.
func Parse(src string) (Module, error) {
	src = stripComments(src)
	var m Module
	seen := map[[2]int]string{}
	for _, d := range bindingDecl.FindAllStringSubmatch(src, -1) {
		group, _ := strconv.Atoi(groupAttr.FindStringSubmatch(d[1])[1])
		binding, _ := strconv.Atoi(bindAttr.FindStringSubmatch(d[1])[1])
		b := Binding{Name: d[3], Group: group, Binding: binding}
		typ := strings.Join(strings.Fields(d[4]), "")
		switch {
		case d[2] == "uniform":
			b.Kind = KindUniform
		case d[2] == "" && typ == "texture_2d<f32>":
			b.Kind = KindTexture
		case d[2] == "" && typ == "sampler":
			b.Kind = KindSampler
		default:
			return Module{}, errors.New(tagShadersUnsupportedBinding, "only uniform buffers, texture_2d<f32> and sampler can be bound (SPEC §9.3)",
				"name", b.Name, "address_space", d[2], "type", typ)
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
	for _, e := range entryDecl.FindAllStringSubmatch(src, -1) {
		if e[1] == "compute" {
			return Module{}, errors.New(tagShadersComputeEntry, "there is no compute in 1.1 (SPEC §9.3)", "entry", e[2])
		}
		entry := Entry{Name: e[2], Stage: strings.ToUpper(e[1][:1]) + e[1][1:]}
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
		return Module{}, errors.New(tagShadersNoEntry, "the module has no @vertex or @fragment function")
	}
	return m, nil
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
	tagShadersBindingTwice       = "ShadersBindingTwice"
	tagShadersComputeEntry       = "ShadersComputeEntry"
	tagShadersNoEntry            = "ShadersNoEntry"
	tagShadersUnbalancedBraces   = "ShadersUnbalancedBraces"
	tagShadersUnsupportedBinding = "ShadersUnsupportedBinding"
)
