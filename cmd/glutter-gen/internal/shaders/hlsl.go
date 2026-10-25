package shaders

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// The HLSL pass (D25, SPEC §9.8 "Shaders"). naga's command line writes HLSL, but it cannot be told
// where to put the samplers: it declares a heap of 2048 samplers and, for each group, a buffer of
// indexes into it that the host would have to fill, and a sampler is read as heap[indexes[binding]].
// Direct3D 12 at feature level 11_0 (resource binding tier 1) wants a sampler table for each group,
// so the generator rewrites that into plain registers, and makes the registers of the other
// resources explicit. The result puts every binding at
//
//	register(<class>B, spaceG)
//
// with the class b (constant buffer), t (texture) or s (sampler), B the @binding and G the @group.
// The rewrite knows the exact shape of naga's output and fails with a tagged error for anything
// else, so that a naga that writes something new breaks generation instead of writing HLSL that
// binds wrongly.

var (
	// The two heaps of naga, and the buffer of sampler indexes of a group.
	samplerHeapDecl    = regexp.MustCompile(`^SamplerState nagaSamplerHeap\[2048\]\s*: register\(s0, space0\);$`)
	comparisonHeapDecl = regexp.MustCompile(`^SamplerComparisonState nagaComparisonSamplerHeap\[2048\]\s*: register\(s0, space1\);$`)
	samplerIndexDecl   = regexp.MustCompile(`^StructuredBuffer<uint> nagaGroup(\d+)SamplerIndexArray\s*: register\(t\d+, space255\);$`)
	samplerUse         = regexp.MustCompile(`^static const (SamplerState|SamplerComparisonState) (\w+) = (nagaSamplerHeap|nagaComparisonSamplerHeap)\[nagaGroup(\d+)SamplerIndexArray\[(\d+)\]\];$`)
	// A local that is const and has a type of the module's own (a struct), such as
	// "    const VertexOutput_vs_main vertexoutput_1 = { ... };".
	constStructLocal    = regexp.MustCompile(`(?m)^([ \t]+)const ([A-Za-z_]\w*) `)
	intrinsicType       = regexp.MustCompile(`^(?:bool|int|uint|dword|half|float|double|min\d+\w*)(?:\d(?:x\d)?)?$`)
	registerNoSpace     = regexp.MustCompile(`register\(([a-z])(\d+)\)`)
	registerWithSpace   = regexp.MustCompile(`register\(([a-z])(\d+), space(\d+)\)`)
	anyRegister         = regexp.MustCompile(`register\(`)
	leftoverFragments   = []string{"nagaSamplerHeap", "nagaComparisonSamplerHeap", "SamplerIndexArray", "space255"}
	instanceIDSemantics = regexp.MustCompile(`\bSV_InstanceID\b`)
)

// register is where an HLSL declaration is bound.
type register struct {
	class        string // "b", "t" or "s"
	space, index int
}

// Register returns where HLSL binds the binding: constant buffers in b registers, textures in t
// and samplers in s, the @binding as the register and the @group as the register space.
func (b Binding) Register() (class string, space, index int) {
	switch b.Kind {
	case KindUniform:
		class = "b"
	case KindTexture:
		class = "t"
	default:
		class = "s"
	}
	return class, b.Group, b.Binding
}

// rewriteHLSL turns the HLSL of naga (shader model 5.1) into the HLSL of the generated file: the
// sampler heap goes, each sampler becomes a register, and a resource of group 0 gets its space.
// It fails with ShadersHLSLUnexpected when a line of naga's sampler machinery has another shape
// than the one expected, when anything of that machinery is left, when a register has no space,
// or when the shader reads SV_InstanceID (which does not include the first instance of a draw, so
// a shader that needs the instance index takes it from a uniform: SPEC §9.8).
func rewriteHLSL(src string) (string, error) {
	if strings.Contains(src, "\r") {
		return "", errors.New(tagShadersHLSLUnexpected, "the HLSL has a carriage return, which a raw string cannot hold")
	}
	arrays := map[int]bool{}
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))
	for n, line := range lines {
		switch {
		case samplerHeapDecl.MatchString(line), comparisonHeapDecl.MatchString(line):
			continue
		case samplerIndexDecl.MatchString(line):
			g, _ := strconv.Atoi(samplerIndexDecl.FindStringSubmatch(line)[1])
			arrays[g] = true
			continue
		case strings.Contains(line, "nagaSamplerHeap") || strings.Contains(line, "nagaComparisonSamplerHeap") || strings.Contains(line, "SamplerIndexArray"):
			m := samplerUse.FindStringSubmatch(line)
			if m == nil {
				return "", errors.New(tagShadersHLSLUnexpected, "a line of naga's sampler heap has a shape that the generator does not know",
					"line", n+1, "text", line)
			}
			typ, name, heap := m[1], m[2], m[3]
			if (typ == "SamplerState") != (heap == "nagaSamplerHeap") {
				return "", errors.New(tagShadersHLSLUnexpected, "a sampler reads the other heap than its type",
					"line", n+1, "text", line)
			}
			group, _ := strconv.Atoi(m[4])
			binding, _ := strconv.Atoi(m[5])
			out = append(out, fmt.Sprintf("%s %s : register(s%d, space%d);", typ, name, binding, group))
			continue
		}
		out = append(out, line)
	}
	text := registerNoSpace.ReplaceAllString(strings.Join(out, "\n"), "register($1$2, space0)")
	text = dropConstStructs(text)

	// A sampler is read through the index array of its group, so naga declared the array.
	for _, g := range usedGroups(src) {
		if !arrays[g] {
			return "", errors.New(tagShadersHLSLUnexpected, "a sampler reads the index array of a group that naga did not declare", "group", g)
		}
	}
	for _, frag := range leftoverFragments {
		if strings.Contains(text, frag) {
			return "", errors.New(tagShadersHLSLUnexpected, "naga's sampler machinery is left in the HLSL", "fragment", frag)
		}
	}
	if instanceIDSemantics.MatchString(text) {
		return "", errors.New(tagShadersHLSLInstanceID, "a shader reads SV_InstanceID, which does not include the first instance of a draw: pass the value in a uniform (SPEC §9.8)")
	}
	if n, with := len(anyRegister.FindAllString(text, -1)), len(registerWithSpace.FindAllString(text, -1)); n != with {
		return "", errors.New(tagShadersHLSLUnexpected, "a register has a shape that the generator does not know",
			"registers", n, "with_space", with)
	}
	return text, nil
}

// dropConstStructs takes `const` off the locals whose type is a struct. In a local it changes
// nothing, but Wine's d3dcompiler_47 (vkd3d-shader 1.18, the compiler of the tests under Wine,
// SPEC §25.13) compiles a vertex shader that returns a `const` struct to code that ignores the
// shader's inputs and constant buffers. Microsoft's compiler takes both forms.
func dropConstStructs(src string) string {
	return constStructLocal.ReplaceAllStringFunc(src, func(m string) string {
		sub := constStructLocal.FindStringSubmatch(m)
		if intrinsicType.MatchString(sub[2]) {
			return m
		}
		return sub[1] + sub[2] + " "
	})
}

// usedGroups returns the groups whose sampler index arrays src reads from.
func usedGroups(src string) []int {
	seen := map[int]bool{}
	for _, line := range strings.Split(src, "\n") {
		if m := samplerUse.FindStringSubmatch(line); m != nil {
			g, _ := strconv.Atoi(m[4])
			seen[g] = true
		}
	}
	var groups []int
	for g := range seen {
		groups = append(groups, g)
	}
	sort.Ints(groups)
	return groups
}

// checkHLSL checks the rewritten HLSL against the WGSL module it came from: every register it
// declares is the register of a binding of the module, of the class of the binding's kind, and
// every binding that an entry point reads is declared; every entry point is a function of the HLSL.
// It fails with ShadersHLSLMismatch otherwise.
func checkHLSL(hlsl string, m Module) error {
	declared := map[register]bool{}
	for _, d := range registerWithSpace.FindAllStringSubmatch(hlsl, -1) {
		space, _ := strconv.Atoi(d[3])
		index, _ := strconv.Atoi(d[2])
		r := register{class: d[1], space: space, index: index}
		if declared[r] {
			return errors.New(tagShadersHLSLMismatch, "two declarations have the same register", "register", r.String())
		}
		declared[r] = true
	}
	known := map[register]bool{}
	for _, b := range m.Bindings {
		class, space, index := b.Register()
		known[register{class, space, index}] = true
	}
	for r := range declared {
		if !known[r] {
			return errors.New(tagShadersHLSLMismatch, "the HLSL declares a register that no binding of the WGSL has", "register", r.String())
		}
	}
	for _, e := range m.Entries {
		for _, b := range e.Bindings {
			class, space, index := b.Register()
			r := register{class, space, index}
			if !declared[r] {
				return errors.New(tagShadersHLSLMismatch, "the HLSL does not declare the register of a binding that an entry point reads",
					"entry", e.Name, "binding", b.Name, "register", r.String())
			}
		}
		fn := regexp.MustCompile(`(?m)^[A-Za-z_]\w*\s+` + regexp.QuoteMeta(e.Name) + `\(`)
		if !fn.MatchString(hlsl) {
			return errors.New(tagShadersHLSLMismatch, "the HLSL has no function with the name of an entry point", "entry", e.Name)
		}
	}
	return nil
}

// String is the register as HLSL writes it, "t3, space1".
func (r register) String() string { return fmt.Sprintf("%s%d, space%d", r.class, r.index, r.space) }

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagShadersHLSLInstanceID = "ShadersHLSLInstanceID"
	tagShadersHLSLMismatch   = "ShadersHLSLMismatch"
	tagShadersHLSLUnexpected = "ShadersHLSLUnexpected"
)
