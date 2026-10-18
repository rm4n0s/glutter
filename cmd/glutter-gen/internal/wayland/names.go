package wayland

import (
	"strings"
	"unicode"
)

// goName turns a protocol name into an exported Go name: "wl_output" becomes
// "WlOutput", "zwp_text_input_v3" becomes "ZwpTextInputV3", "90" becomes "90".
func goName(s string) string {
	var b strings.Builder
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' }) {
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}

// reserved are the names an argument cannot take: Go keywords, and the local
// variables of the generated methods.
var reserved = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
	// locals of the generated code
	"self": true, "argv": true, "proxy": true, "ev": true, "c": true,
	// predeclared names the generated code still uses
	"string": true, "int32": true, "uint32": true, "float64": true, "nil": true,
}

// renamed are the keywords that have a readable replacement.
var renamed = map[string]string{"interface": "iface", "type": "typ"}

// argName turns a protocol argument name into a Go parameter name:
// "surface_x" becomes "surfaceX", and clashes with keywords get an underscore.
func argName(s string) string {
	name := goName(s)
	runes := []rune(name)
	runes[0] = unicode.ToLower(runes[0])
	name = string(runes)
	if short, ok := renamed[name]; ok {
		return short
	}
	if reserved[name] {
		name += "_"
	}
	return name
}

// comment turns text from the XML into the body of a one-line Go comment.
func comment(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
