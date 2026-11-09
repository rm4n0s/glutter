package shaders

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// naga writes Metal Shading Language 2.4 for a WGSL module. It names every resource of an entry point with
// `[[user(fake0)]]`, since the command has no option for a binding map; this file puts each resource at
// the index of its class that the module's binding map gives (Binding.MetalIndex, SPEC §29.5), finding the
// binding by the name of the variable, which naga keeps, and checks the result against the WGSL.

var (
	// mslResource is the parameter of an entry point that naga writes for a resource: its last word is
	// the name of the variable.
	mslResource = regexp.MustCompile(`(\w+) \[\[user\(fake\d+\)\]\]`)
	// mslBufferSizesParam is the parameter that carries the lengths of runtime-sized arrays, which naga adds
	// to every entry point of a module that has a storage buffer, whether a body reads it or not.
	mslBufferSizesParam = regexp.MustCompile(`(?m)^,[ \t]*constant _mslBufferSizes& _buffer_sizes \[\[user\(fake\d+\)\]\]\n`)
	mslBufferSizesDecl  = regexp.MustCompile(`(?s)struct _mslBufferSizes \{.*?\};\n`)
	mslRenamed          = regexp.MustCompile(`_\d+$`)
	mslUserLeft         = regexp.MustCompile(`\[\[user\(fake`)
	mslIndexAttr        = regexp.MustCompile(`\[\[(buffer|texture|sampler)\((\d+)\)\]\]`)
)

// maxMetalBuffers is how many buffers a module may bind: the vertex buffers are above them
// (explicit.MetalVertexBufferBase).
const maxMetalBuffers = 16

// rewriteMSL puts the resources of naga's MSL at the indexes of the binding map of the module m. It fails
// with ShadersMSLUnexpected when the text has a shape that the generator does not know: a carriage return
// (a raw string cannot hold it), a resource that no binding of the WGSL has, a read of the lengths of
// runtime-sized arrays (no shader reads them; an arrayLength would need the table that naga names
// _mslBufferSizes), or more buffers than maxMetalBuffers.
func rewriteMSL(src string, m Module) (string, error) {
	if strings.Contains(src, "\r") {
		return "", errors.New(tagShadersMSLUnexpected, "the MSL has a carriage return, which a raw string cannot hold")
	}
	buffers := 0
	byName := map[string]Binding{}
	for _, b := range m.Bindings {
		byName[b.Name] = b
		if b.MetalClass() == metalBuffer {
			buffers++
		}
	}
	if buffers > maxMetalBuffers {
		return "", errors.New(tagShadersMSLUnexpected, "a module binds more buffers than Metal's table leaves room for (the vertex buffers come after them)",
			"buffers", buffers, "limit", maxMetalBuffers)
	}
	text := mslBufferSizesParam.ReplaceAllString(src, "")
	if strings.Contains(text, "_buffer_sizes") {
		// The declaration is dropped only when nothing is left that reads it.
		declLess := mslBufferSizesDecl.ReplaceAllString(text, "")
		if strings.Contains(declLess, "_buffer_sizes") {
			return "", errors.New(tagShadersMSLUnexpected, "a shader reads the lengths of a runtime-sized array, which the generator does not give it")
		}
	}
	text = mslBufferSizesDecl.ReplaceAllString(text, "")
	var out strings.Builder
	last := 0
	for _, loc := range mslResource.FindAllStringSubmatchIndex(text, -1) {
		name := text[loc[2]:loc[3]]
		b, ok := lookupMSLName(byName, name)
		if !ok {
			return "", errors.New(tagShadersMSLUnexpected, "the MSL has a resource that no binding of the WGSL has", "name", name)
		}
		out.WriteString(text[last:loc[2]])
		out.WriteString(name + " [[" + b.MetalClass() + "(" + strconv.Itoa(b.MetalIndex) + ")]]")
		last = loc[1]
	}
	out.WriteString(text[last:])
	text = out.String()
	if mslUserLeft.MatchString(text) {
		return "", errors.New(tagShadersMSLUnexpected, "a resource of naga's MSL has a shape that the generator does not know")
	}
	return text, nil
}

// lookupMSLName finds the binding of a parameter name of naga's MSL. naga renames a variable whose name ends in
// a digit (`t0` is `t0_`) and one that another name of the function would hide (`src` is `src_1`).
func lookupMSLName(byName map[string]Binding, name string) (Binding, bool) {
	for _, n := range []string{name, mslRenamed.ReplaceAllString(name, ""), strings.TrimSuffix(name, "_"), strings.TrimSuffix(mslRenamed.ReplaceAllString(name, ""), "_")} {
		if b, ok := byName[n]; ok {
			return b, true
		}
	}
	return Binding{}, false
}

// checkMSL checks the MSL against the module: it has a function with the name of each entry point, and
// the resources of each function are exactly the bindings that the entry point reads, at their indexes.
// It fails with ShadersMSLMismatch.
func checkMSL(msl string, m Module) error {
	for _, e := range m.Entries {
		params, ok := mslParams(msl, e)
		if !ok {
			return errors.New(tagShadersMSLMismatch, "the MSL has no function with the name of an entry point", "entry", e.Name)
		}
		got := map[string]bool{}
		for _, a := range mslIndexAttr.FindAllStringSubmatch(params, -1) {
			key := a[1] + a[2]
			if got[key] {
				return errors.New(tagShadersMSLMismatch, "two resources of an entry point have the same index", "entry", e.Name, "index", key)
			}
			got[key] = true
		}
		want := map[string]bool{}
		for _, b := range e.Bindings {
			want[b.MetalClass()+strconv.Itoa(b.MetalIndex)] = true
		}
		for key := range want {
			if !got[key] {
				return errors.New(tagShadersMSLMismatch, "the MSL does not declare the resource of a binding that an entry point reads", "entry", e.Name, "index", key)
			}
		}
		var extra []string
		for key := range got {
			if !want[key] {
				extra = append(extra, key)
			}
		}
		if len(extra) > 0 {
			sort.Strings(extra)
			return errors.New(tagShadersMSLMismatch, "the MSL declares a resource that the entry point does not read", "entry", e.Name, "indexes", extra)
		}
	}
	return nil
}

// mslParams returns the text of the parameter list of the entry point in naga's MSL.
func mslParams(msl string, e Entry) (string, bool) {
	stage := strings.ToLower(e.Stage)
	if stage == "compute" {
		stage = "kernel"
	}
	re := regexp.MustCompile(`(?m)(?:^|\] )` + stage + ` [\w:<>]+ ` + regexp.QuoteMeta(e.Name) + `\(`)
	loc := re.FindStringIndex(msl)
	if loc == nil {
		return "", false
	}
	end := strings.Index(msl[loc[1]:], "\n) {")
	if end < 0 {
		return "", false
	}
	return msl[loc[1] : loc[1]+end], true
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagShadersMSLMismatch   = "ShadersMSLMismatch"
	tagShadersMSLUnexpected = "ShadersMSLUnexpected"
)
