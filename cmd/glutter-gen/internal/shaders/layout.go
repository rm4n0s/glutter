package shaders

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// The reflection that gpushaders adds to the one of shaders (SPEC §27.3): the size of the type behind a
// uniform or storage binding, by the memory layout rules of WGSL, and the inputs of a vertex entry point.

// Input is an `@location(n)` input of a vertex entry point.
type Input struct {
	Name     string
	Location int
	// Type is the shader-side type as the generated code spells it ("Float32x4", "Uint32").
	Type string
}

var (
	// A name that WGSL gives to a vector or a matrix of f32, i32 or u32.
	vecAlias = regexp.MustCompile(`^vec([234])([fiu])$`)
	matAlias = regexp.MustCompile(`^mat([234])x([234])f$`)
	vecType  = regexp.MustCompile(`^vec([234])<(f32|i32|u32)>$`)
	matType  = regexp.MustCompile(`^mat([234])x([234])<f32>$`)
	// array<T, N> and array<T>; the element type is matched with its angle brackets balanced by arrayParts.
	attrDecl  = regexp.MustCompile(`@(\w+)(?:\(([^)]*)\))?`)
	constDecl = regexp.MustCompile(`\bconst\s+(\w+)\s*(?::\s*\w+)?\s*=\s*(\d+)[iu]?\s*;`)
	aliasDecl = regexp.MustCompile(`\balias\s+(\w+)\s*=\s*([^;]+);`)
)

// member is a member of a struct, from its declaration.
type member struct {
	name, typ string
	attrs     map[string]string
}

// layouts reads the types of a module: the structs, the constants that name an array length, and the aliases.
type layouts struct {
	structs map[string][]member
	consts  map[string]int
	aliases map[string]string
}

// newLayouts reads the declarations of the module (comments already removed).
func newLayouts(src string) *layouts {
	l := &layouts{structs: map[string][]member{}, consts: map[string]int{}, aliases: map[string]string{}}
	for _, st := range structDecl.FindAllStringSubmatch(src, -1) {
		l.structs[st[1]] = parseMembers(st[2])
	}
	for _, c := range constDecl.FindAllStringSubmatch(src, -1) {
		n, _ := strconv.Atoi(c[2])
		l.consts[c[1]] = n
	}
	for _, a := range aliasDecl.FindAllStringSubmatch(src, -1) {
		l.aliases[a[1]] = strings.Join(strings.Fields(a[2]), "")
	}
	return l
}

// parseMembers splits the body of a struct into its members, with their attributes.
func parseMembers(body string) []member {
	var out []member
	for _, part := range splitTop(body, ',') {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		m := member{attrs: map[string]string{}}
		for _, a := range attrDecl.FindAllStringSubmatch(part, -1) {
			m.attrs[a[1]] = strings.Join(strings.Fields(a[2]), "")
		}
		rest := strings.TrimSpace(attrDecl.ReplaceAllString(part, ""))
		name, typ, ok := strings.Cut(rest, ":")
		if !ok {
			continue
		}
		m.name = strings.TrimSpace(name)
		m.typ = strings.Join(strings.Fields(typ), "")
		out = append(out, m)
	}
	return out
}

// splitTop splits s at sep where the angle brackets and parentheses are balanced.
func splitTop(s string, sep byte) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<', '(':
			depth++
		case '>', ')':
			depth--
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}

// alignUp rounds n up to a multiple of a.
func alignUp(n, a int) int { return (n + a - 1) / a * a }

// sizeOf returns the size and alignment of a type by the layout rules of WGSL (the "Alignment and size"
// section of the specification), which uniform and storage buffers follow. An array that has no length
// counts one element, as the minimum binding size of WebGPU does. It fails with ShadersLayoutUnsupported
// for a type it does not know.
func (l *layouts) sizeOf(typ string, depth int) (size, align int, err error) {
	if depth > 16 {
		return 0, 0, errors.New(tagShadersLayoutUnsupported, "a type refers to itself or nests too deep", "type", typ)
	}
	if a, ok := l.aliases[typ]; ok {
		return l.sizeOf(a, depth+1)
	}
	switch typ {
	case "f32", "i32", "u32", "atomic<u32>", "atomic<i32>":
		return 4, 4, nil
	}
	if m := vecAlias.FindStringSubmatch(typ); m != nil {
		typ = "vec" + m[1] + "<" + map[string]string{"f": "f32", "i": "i32", "u": "u32"}[m[2]] + ">"
	}
	if m := matAlias.FindStringSubmatch(typ); m != nil {
		typ = "mat" + m[1] + "x" + m[2] + "<f32>"
	}
	if m := vecType.FindStringSubmatch(typ); m != nil {
		switch m[1] {
		case "2":
			return 8, 8, nil
		case "3":
			return 12, 16, nil
		}
		return 16, 16, nil
	}
	if m := matType.FindStringSubmatch(typ); m != nil {
		cols, _ := strconv.Atoi(m[1])
		rows, _ := strconv.Atoi(m[2])
		colSize, colAlign, _ := l.sizeOf("vec"+strconv.Itoa(rows)+"<f32>", depth+1)
		return cols * alignUp(colSize, colAlign), colAlign, nil
	}
	if strings.HasPrefix(typ, "array<") && strings.HasSuffix(typ, ">") {
		inner := typ[len("array<") : len(typ)-1]
		parts := splitTop(inner, ',')
		elemSize, elemAlign, err := l.sizeOf(parts[0], depth+1)
		if err != nil {
			return 0, 0, err
		}
		count := 1
		if len(parts) == 2 {
			count, err = l.count(strings.TrimSpace(parts[1]))
			if err != nil {
				return 0, 0, err
			}
		} else if len(parts) > 2 {
			return 0, 0, errors.New(tagShadersLayoutUnsupported, "an array has an element type and a length", "type", typ)
		}
		return count * alignUp(elemSize, elemAlign), elemAlign, nil
	}
	if members, ok := l.structs[typ]; ok {
		return l.structSize(typ, members, depth)
	}
	return 0, 0, errors.New(tagShadersLayoutUnsupported, "the generator does not know the size of this type (f32, i32, u32, vectors, f32 matrices, atomics, arrays and structs of them)", "type", typ)
}

// structSize lays out the members of a struct: each at the next multiple of its alignment, with the
// attributes @align and @size.
func (l *layouts) structSize(name string, members []member, depth int) (size, align int, err error) {
	align = 1
	offset := 0
	for _, m := range members {
		msize, malign, err := l.sizeOf(m.typ, depth+1)
		if err != nil {
			return 0, 0, err
		}
		if v, ok := m.attrs["align"]; ok {
			n, err := l.count(v)
			if err != nil || n < malign {
				return 0, 0, errors.New(tagShadersLayoutUnsupported, "@align is a literal that is at least the alignment of the type", "struct", name, "member", m.name, "attribute", v)
			}
			malign = n
		}
		if v, ok := m.attrs["size"]; ok {
			n, err := l.count(v)
			if err != nil || n < msize {
				return 0, 0, errors.New(tagShadersLayoutUnsupported, "@size is a literal that is at least the size of the type", "struct", name, "member", m.name, "attribute", v)
			}
			msize = n
		}
		offset = alignUp(offset, malign) + msize
		align = max(align, malign)
	}
	return alignUp(offset, align), align, nil
}

// count reads a literal or the name of a constant: the length of an array or the value of an attribute.
func (l *layouts) count(s string) (int, error) {
	s = strings.TrimRight(s, "iu")
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		return n, nil
	}
	if n, ok := l.consts[s]; ok && n > 0 {
		return n, nil
	}
	return 0, errors.New(tagShadersLayoutUnsupported, "a length is a positive integer literal or a constant of one", "value", s)
}

// bindingSize is the size of the type of a uniform or storage binding.
func (l *layouts) bindingSize(typ string) (int, error) {
	size, _, err := l.sizeOf(typ, 0)
	return size, err
}

// inputs reads the @location parameters of a vertex entry point: directly in its parameter list, or
// as the members of a struct it takes. head is the text from "fn" to the opening brace.
func (l *layouts) inputs(head string) ([]Input, error) {
	open := strings.IndexByte(head, '(')
	if open < 0 {
		return nil, nil
	}
	depth, end := 0, -1
	for i := open; i < len(head) && end < 0; i++ {
		switch head[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				end = i
			}
		}
	}
	if end < 0 {
		return nil, nil
	}
	var out []Input
	for _, p := range parseMembers(head[open+1 : end]) {
		if members, ok := l.structs[p.typ]; ok {
			for _, m := range members {
				if in, ok, err := l.input(m); err != nil {
					return nil, err
				} else if ok {
					out = append(out, in)
				}
			}
			continue
		}
		if in, ok, err := l.input(p); err != nil {
			return nil, err
		} else if ok {
			out = append(out, in)
		}
	}
	return out, nil
}

// input makes the Input of a parameter or member that has @location.
func (l *layouts) input(m member) (Input, bool, error) {
	loc, ok := m.attrs["location"]
	if !ok {
		return Input{}, false, nil
	}
	n, err := strconv.Atoi(loc)
	if err != nil || n < 0 {
		return Input{}, false, errors.New(tagShadersLayoutUnsupported, "a location is a non-negative integer literal", "input", m.name, "location", loc)
	}
	typ := m.typ
	if a, ok := l.aliases[typ]; ok {
		typ = a
	}
	scalar := map[string]string{"f": "f32", "i": "i32", "u": "u32"}
	if v := vecAlias.FindStringSubmatch(typ); v != nil {
		typ = "vec" + v[1] + "<" + scalar[v[2]] + ">"
	}
	var shape, count string
	switch {
	case typ == "f32" || typ == "i32" || typ == "u32":
		shape, count = typ, ""
	case vecType.MatchString(typ):
		v := vecType.FindStringSubmatch(typ)
		shape, count = v[2], "x"+v[1]
	default:
		return Input{}, false, errors.New(tagShadersLayoutUnsupported, "a vertex input is a scalar or a vector of f32, i32 or u32", "input", m.name, "type", m.typ)
	}
	name := map[string]string{"f32": "Float32", "i32": "Sint32", "u32": "Uint32"}[shape]
	return Input{Name: m.name, Location: n, Type: name + count}, true, nil
}

// The tag of the error that this file makes (rm4n0s/errors, R5).
const (
	tagShadersLayoutUnsupported = "ShadersLayoutUnsupported"
)
