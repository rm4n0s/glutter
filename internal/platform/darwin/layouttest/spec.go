package layouttest

import (
	"fmt"
	"reflect"
	"strings"
)

// Spec is what a test believes about the SDK's headers.
//
// Flutter: no counterpart
type Spec struct {
	// Headers are the headers to import, such as "CoreFoundation/CoreFoundation.h" or
	// "AppKit/AppKit.h".
	Headers []string
	// Defines are macros to define.
	Defines []string
	// Structs are the mirrors of C structures.
	Structs []Struct
	// Consts are integer constants (enumerators, macros) with the value that Go uses.
	Consts []Const
}

// Struct is the Go mirror of one C structure: its C name, size and fields.
//
// Flutter: no counterpart
type Struct struct {
	// CName is the name the header gives the type, such as "CGRect" or "CFRunLoopSourceContext".
	CName string
	// Size is the size of the Go type.
	Size uintptr
	// Fields are the fields, in Go's order.
	Fields []Field
}

// Field is one member of a structure.
//
// Flutter: no counterpart
type Field struct {
	// CName is the name of the member in the header.
	CName string
	// Offset is the offset of the Go field.
	Offset uintptr
	// Size is the size of the Go field.
	Size uintptr
}

// Const is an integer constant, such as an enumerator or a macro, with the value Go uses.
//
// Flutter: no counterpart
type Const struct {
	// Name is the macro or enumerator in the headers.
	Name string
	// Value is the value Go uses.
	Value int64
}

// StructOf reads the size and the fields of a Go struct value by reflection. Each field must carry a
// `c:"name"` tag with the member's name in the C header; a field named "_" (padding) is skipped, and so
// is a field tagged `c:"-"`. A value that is not a struct, or a field without the tag, is a mistake of
// the test and panics.
//
// Flutter: no counterpart
func StructOf(cName string, v any) Struct {
	t := reflect.TypeOf(v)
	if t == nil || t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("layouttest: StructOf(%q) needs a struct value, got %T", cName, v))
	}
	s := Struct{CName: cName, Size: t.Size()}
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Name == "_" {
			continue
		}
		name, ok := f.Tag.Lookup("c")
		if !ok || name == "" {
			panic(fmt.Sprintf("layouttest: field %s of %s has no c tag with its name in the header", f.Name, t))
		}
		if name == "-" {
			continue
		}
		s.Fields = append(s.Fields, Field{CName: name, Offset: f.Offset, Size: f.Type.Size()})
	}
	return s
}

// Source writes the C file that the spec stands for: the imports, then one _Static_assert for each
// size, offset, field size and constant. The message of each assertion names what it checks
// ("CGRect.origin: offset"), so the compiler's output names the structure.
//
// Flutter: no counterpart
func (s Spec) Source() string {
	var b strings.Builder
	b.WriteString("/* Written by internal/platform/darwin/layouttest. */\n")
	for _, d := range s.Defines {
		fmt.Fprintf(&b, "#define %s\n", d)
	}
	for _, h := range s.Headers {
		fmt.Fprintf(&b, "#import <%s>\n", h)
	}
	b.WriteString("#include <stddef.h>\n\n")
	for _, st := range s.Structs {
		fmt.Fprintf(&b, "_Static_assert(sizeof(%s) == %d, \"%s: size\");\n", st.CName, st.Size, st.CName)
		for _, f := range st.Fields {
			fmt.Fprintf(&b, "_Static_assert(offsetof(%s, %s) == %d, \"%s.%s: offset\");\n", st.CName, f.CName, f.Offset, st.CName, f.CName)
			fmt.Fprintf(&b, "_Static_assert(sizeof(((%s *)0)->%s) == %d, \"%s.%s: size\");\n", st.CName, f.CName, f.Size, st.CName, f.CName)
		}
	}
	for _, c := range s.Consts {
		fmt.Fprintf(&b, "_Static_assert((long long)(%s) == %dLL, \"%s: value\");\n", c.Name, c.Value, c.Name)
	}
	return b.String()
}
