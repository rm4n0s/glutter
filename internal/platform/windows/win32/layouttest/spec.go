package layouttest

import (
	"fmt"
	"reflect"
	"strings"
)

// Spec is what a test believes about the C headers.
//
// Flutter: no counterpart
type Spec struct {
	// Headers are the headers to include, such as "windows.h" or "d3d12.h".
	Headers []string
	// Defines are macros to define besides COBJMACROS, such as "INITGUID".
	Defines []string
	// IncludeDirs are extra directories of headers (-I). The mingw-w64 compiler needs none.
	IncludeDirs []string
	// Structs are the mirrors of C structures.
	Structs []Struct
	// VTables are the vtables of COM interfaces.
	VTables []VTable
	// Consts are constants (HRESULT values, flags).
	Consts []Const
}

// Struct is the Go mirror of one C structure: its C name, size and fields.
//
// Flutter: no counterpart
type Struct struct {
	// CName is the name the header gives the type, such as "RECT" or "WNDCLASSEXW".
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

// VTable is the vtable of a COM interface as the C header declares it (the type IFooVtbl).
//
// Flutter: no counterpart
type VTable struct {
	// CName is the name of the vtable structure, such as "IUnknownVtbl" or "ID3D12DeviceVtbl".
	CName string
	// Methods are the methods Go calls or implements, with the index Go uses.
	Methods []Method
}

// Method is one entry of a vtable.
//
// Flutter: no counterpart
type Method struct {
	// Name is the name of the method in the header, such as "QueryInterface".
	Name string
	// Index is the position Go uses, counted in pointers from the start of the vtable.
	Index int
}

// Const is a 32-bit constant, such as an HRESULT, with the value Go uses.
//
// Flutter: no counterpart
type Const struct {
	// Name is the macro or enumerator in the headers.
	Name string
	// Value is the value Go uses, as the unsigned number the headers write (0x80004005).
	Value uint32
}

// StructOf reads the size and the fields of a Go struct value by reflection. Each field must carry
// a `c:"name"` tag with the member's name in the C header; a field named "_" (padding) is skipped,
// and so is a field tagged `c:"-"`. A value that is not a struct, or a field without the tag, is
// a mistake of the test and panics.
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

// Source writes the C file that the spec stands for: the includes, then one _Static_assert for
// each size, offset, field size, method index and constant. The message of each assertion names
// what it checks ("RECT.left: offset"), so the compiler's output names the structure.
//
// Flutter: no counterpart
func (s Spec) Source() string {
	var b strings.Builder
	b.WriteString("/* Written by internal/platform/windows/win32/layouttest. */\n")
	for _, h := range s.Headers {
		fmt.Fprintf(&b, "#include <%s>\n", h)
	}
	b.WriteString("#include <stddef.h>\n\n")
	for _, st := range s.Structs {
		fmt.Fprintf(&b, "_Static_assert(sizeof(%s) == %d, \"%s: size\");\n", st.CName, st.Size, st.CName)
		for _, f := range st.Fields {
			fmt.Fprintf(&b, "_Static_assert(offsetof(%s, %s) == %d, \"%s.%s: offset\");\n", st.CName, f.CName, f.Offset, st.CName, f.CName)
			fmt.Fprintf(&b, "_Static_assert(sizeof(((%s *)0)->%s) == %d, \"%s.%s: size\");\n", st.CName, f.CName, f.Size, st.CName, f.CName)
		}
	}
	for _, v := range s.VTables {
		for _, m := range v.Methods {
			fmt.Fprintf(&b, "_Static_assert(offsetof(%s, %s) / sizeof(void *) == %d, \"%s.%s: index\");\n", v.CName, m.Name, m.Index, v.CName, m.Name)
		}
	}
	for _, c := range s.Consts {
		fmt.Fprintf(&b, "_Static_assert((unsigned int)(%s) == 0x%08XU, \"%s: value\");\n", c.Name, c.Value, c.Name)
	}
	return b.String()
}
