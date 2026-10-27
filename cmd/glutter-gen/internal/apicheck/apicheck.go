// Package apicheck implements the `glutter-gen apicheck` subcommand, which reads the public
// packages and reports what breaks the API conventions of SPEC §12.
//
// Flutter: no counterpart.
package apicheck

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rm4n0s/errors"
)

// DefaultPackages are the public packages of the module, as directories below its root.
//
// Flutter: no counterpart.
var DefaultPackages = []string{
	"foundation", "scheduler", "animation", "physics", "gestures", "painting", "rendering",
	"semantics", "services", "widgets", "material", "ui", "desktop", "gluttertest", "svg",
}

// The rules, as they are named in a report and in an allow file.
const (
	// RuleCtorShape says that an exported NewX takes its optional parameters in a trailing
	// variadic options value (SPEC §12.1).
	RuleCtorShape = "ctor-shape"
	// RuleCtorOrder says that child, children, sliver and slivers come last among the
	// positional parameters (SPEC §12.2).
	RuleCtorOrder = "ctor-order"
	// RuleCtorSize says that a constructor has at most six required parameters (SPEC §12.1).
	RuleCtorSize = "ctor-size"
	// RuleDocRef says that the doc comment of a constructor, a function or a type ends with
	// a "Flutter:", "Glutter-specific:" or "flutter_svg:" paragraph (the last one is the reference
	// of package svg, which ports the flutter_svg package, SPEC §26).
	RuleDocRef = "doc-ref"
	// RuleEnumZero says that the zero value of an enum is <Enum>Unspecified (SPEC §12.3).
	RuleEnumZero = "enum-zero"
	// RuleMutableVar says that no exported package variable exists, as it could be
	// reassigned (CLAUDE.md, "No global mutable state").
	RuleMutableVar = "mutable-var"
)

// maxRequired is the largest number of required parameters of a constructor (SPEC §12.1).
const maxRequired = 6

// Violation is one place where the API breaks a rule.
//
// Flutter: no counterpart.
type Violation struct {
	// Rule is one of the Rule constants.
	Rule string
	// Pkg is the directory of the package below the root, such as "widgets".
	Pkg string
	// Name is the identifier, such as "NewPadding".
	Name string
	// File and Line are where the identifier is declared.
	File string
	Line int
	// Message says what is wrong.
	Message string
}

// Key is how an allow file names the violation: "rule pkg.Name".
//
// Flutter: no counterpart.
func (v Violation) Key() string { return v.Rule + " " + v.Pkg + "." + v.Name }

// String is the line of the report.
//
// Flutter: no counterpart.
func (v Violation) String() string {
	return fmt.Sprintf("%s:%d: [%s] %s: %s", v.File, v.Line, v.Rule, v.Name, v.Message)
}

// Run is the subcommand. It prints the violations that the allow file does not list and
// fails with ApiCheckViolations when there are some.
//
// Flutter: no counterpart.
func Run(args []string) error { return run(args, os.Stdout) }

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("apicheck", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "root directory of the module")
	pkgs := flags.String("pkgs", strings.Join(DefaultPackages, ","), "public packages to check, as directories below the root, separated by commas")
	allowPath := flags.String("allow", "", "file that lists the accepted violations, one \"rule pkg.Name\" per line, with # comments")
	if err := flags.Parse(args); err != nil {
		return errors.NewErr(tagApiCheckBadFlags, err).SetMetadata("args", args)
	}
	allowed := map[string]bool{}
	if *allowPath != "" {
		var err error
		if allowed, err = ReadAllow(*allowPath); err != nil {
			return err
		}
	}
	var found []Violation
	for _, pkg := range strings.Split(*pkgs, ",") {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}
		vs, err := CheckDir(*root, pkg)
		if err != nil {
			return err
		}
		found = append(found, vs...)
	}
	var left []Violation
	skipped := 0
	used := map[string]bool{}
	for _, v := range found {
		if allowed[v.Key()] {
			skipped++
			used[v.Key()] = true
			continue
		}
		left = append(left, v)
	}
	for _, v := range left {
		if _, err := fmt.Fprintln(stdout, v); err != nil {
			return errors.NewErr(tagApiCheckWriteFailed, err)
		}
	}
	var stale []string
	for key := range allowed {
		if !used[key] {
			stale = append(stale, key)
		}
	}
	sort.Strings(stale)
	for _, key := range stale {
		if _, err := fmt.Fprintf(stdout, "%s: the allow file lists a violation that is gone\n", key); err != nil {
			return errors.NewErr(tagApiCheckWriteFailed, err)
		}
	}
	if _, err := fmt.Fprintf(stdout, "apicheck: %d violations, %d accepted by the allow file\n", len(left), skipped); err != nil {
		return errors.NewErr(tagApiCheckWriteFailed, err)
	}
	if len(left) > 0 || len(stale) > 0 {
		return errors.New(tagApiCheckViolations, "the API breaks the conventions of SPEC §12", "violations", len(left), "stale", len(stale))
	}
	return nil
}

// ReadAllow reads an allow file: each line is "rule pkg.Name", optionally followed by
// "# why". Blank lines and lines that start with # are skipped.
//
// Flutter: no counterpart.
func ReadAllow(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.NewErr(tagApiCheckAllowUnreadable, err).SetMetadata("file", path)
	}
	out := map[string]bool{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line, _, _ := strings.Cut(sc.Text(), "#")
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 2 {
			return nil, errors.New(tagApiCheckAllowSyntax, "a line of the allow file is not \"rule pkg.Name\"", "line", sc.Text(), "file", path)
		}
		out[fields[0]+" "+fields[1]] = true
	}
	if err := sc.Err(); err != nil {
		return nil, errors.NewErr(tagApiCheckAllowUnreadable, err).SetMetadata("file", path)
	}
	return out, nil
}

// CheckDir reads the Go files of root/pkg (not the tests) and returns the violations, in
// the order of their files and lines.
//
// Flutter: no counterpart.
func CheckDir(root, pkg string) ([]Violation, error) {
	dir := filepath.Join(root, pkg)
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.NewErr(tagApiCheckParseFailed, err).SetMetadata("dir", dir)
	}
	c := &checker{fset: fset, pkg: pkg, root: root}
	var files []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return nil, errors.NewErr(tagApiCheckParseFailed, err).SetMetadata("file", filepath.Join(dir, name))
		}
		files = append(files, f)
	}
	c.collectEnums(files)
	c.collectAssigned(files)
	for _, f := range files {
		c.checkFile(f)
	}
	sort.SliceStable(c.out, func(i, j int) bool {
		if c.out[i].File != c.out[j].File {
			return c.out[i].File < c.out[j].File
		}
		return c.out[i].Line < c.out[j].Line
	})
	return c.out, nil
}

type checker struct {
	fset *token.FileSet
	pkg  string
	root string
	out  []Violation
	// enums maps the name of an exported integer type to the const specs of its first
	// group, so that its zero value can be looked at.
	enums map[string]enumInfo
	// assigned holds the names that some statement of the package assigns to, outside the
	// declarations.
	assigned map[string]bool
}

type enumInfo struct {
	first  string
	pos    token.Pos
	hasIt  bool
	values int
}

func (c *checker) add(rule string, pos token.Pos, name, format string, args ...any) {
	p := c.fset.Position(pos)
	file, err := filepath.Rel(c.root, p.Filename)
	if err != nil {
		file = p.Filename
	}
	c.out = append(c.out, Violation{Rule: rule, Pkg: c.pkg, Name: name, File: file, Line: p.Line, Message: fmt.Sprintf(format, args...)})
}

func (c *checker) checkFile(f *ast.File) {
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Recv != nil || !d.Name.IsExported() {
				continue
			}
			c.checkFunc(d)
		case *ast.GenDecl:
			c.checkGen(d)
		}
	}
}

// checkFunc checks an exported function without a receiver.
func (c *checker) checkFunc(d *ast.FuncDecl) {
	name := d.Name.Name
	c.checkDoc(d.Doc, d.Pos(), name)
	if !strings.HasPrefix(name, "New") || d.Type.Results == nil || len(d.Type.Results.List) == 0 {
		return
	}
	params := flatten(d.Type.Params)
	required := 0
	var last *param
	if n := len(params); n > 0 && params[n-1].variadic {
		last = &params[n-1]
		params = params[:n-1]
	}
	for _, p := range params {
		if strings.HasSuffix(typeName(p.typ), "Opts") {
			c.add(RuleCtorShape, d.Pos(), name, "the options value %q is a required parameter; it must be the trailing variadic parameter", p.name)
		}
		required++
	}
	// A variadic parameter of another type is Flutter's optional positional parameter
	// ("[double tension = 0.5]") or a list of values; SPEC §12.10 accepts it.
	_ = last
	for i, p := range params {
		switch p.name {
		case "child", "children", "sliver", "slivers":
			if i != len(params)-1 {
				c.add(RuleCtorOrder, d.Pos(), name, "%q must be the last positional parameter", p.name)
			}
		}
	}
	if required > maxRequired {
		c.add(RuleCtorSize, d.Pos(), name, "%d required parameters (the limit is %d)", required, maxRequired)
	}
}

func (c *checker) checkGen(d *ast.GenDecl) {
	switch d.Tok {
	case token.TYPE:
		for _, s := range d.Specs {
			ts := s.(*ast.TypeSpec)
			if !ts.Name.IsExported() {
				continue
			}
			doc := ts.Doc
			if doc == nil && len(d.Specs) == 1 {
				doc = d.Doc
			}
			c.checkDoc(doc, ts.Pos(), ts.Name.Name)
			if e, ok := c.enums[ts.Name.Name]; ok && e.values >= 2 && !strings.HasSuffix(e.first, "Unspecified") {
				c.add(RuleEnumZero, e.pos, ts.Name.Name, "the zero value is %s, not %sUnspecified", e.first, ts.Name.Name)
			}
		}
	case token.VAR:
		for _, s := range d.Specs {
			vs := s.(*ast.ValueSpec)
			for i, n := range vs.Names {
				if !n.IsExported() {
					continue
				}
				var value ast.Expr
				if len(vs.Values) == len(vs.Names) {
					value = vs.Values[i]
				}
				if why := mutableKind(vs.Type, value); why != "" {
					c.add(RuleMutableVar, n.Pos(), n.Name, "exported package variable that can be changed in place: %s", why)
				} else if c.assigned[n.Name] {
					c.add(RuleMutableVar, n.Pos(), n.Name, "exported package variable that the package assigns to")
				}
			}
		}
	}
}

// docRefPrefixes start the last paragraph of a doc comment: the source of the identifier.
var docRefPrefixes = []string{"Flutter:", "Glutter-specific:", "flutter_svg:"}

func isDocRef(line string) bool {
	for _, p := range docRefPrefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
}

// checkDoc checks that the last paragraph of the comment is a reference to Flutter.
func (c *checker) checkDoc(doc *ast.CommentGroup, pos token.Pos, name string) {
	if doc == nil {
		c.add(RuleDocRef, pos, name, "no doc comment")
		return
	}
	paras := strings.Split(strings.TrimSpace(doc.Text()), "\n\n")
	last := strings.TrimSpace(paras[len(paras)-1])
	if isDocRef(last) {
		return
	}
	// The reference may be the last line of a paragraph that has no blank line before it.
	lines := strings.Split(last, "\n")
	if isDocRef(strings.TrimSpace(lines[len(lines)-1])) {
		return
	}
	c.add(RuleDocRef, pos, name, "the doc comment does not end with a \"Flutter:\", \"Glutter-specific:\" or \"flutter_svg:\" reference")
}

// collectEnums finds the integer types with named values and their first value.
func (c *checker) collectEnums(files []*ast.File) {
	c.enums = map[string]enumInfo{}
	ints := map[string]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.TYPE {
				continue
			}
			for _, s := range g.Specs {
				ts := s.(*ast.TypeSpec)
				if id, ok := ts.Type.(*ast.Ident); ok && isInteger(id.Name) && ts.Name.IsExported() {
					ints[ts.Name.Name] = true
				}
			}
		}
	}
	for _, f := range files {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				continue
			}
			var typ string
			for i, s := range g.Specs {
				vs := s.(*ast.ValueSpec)
				if vs.Type != nil {
					typ = typeName(vs.Type)
				} else if len(vs.Values) > 0 {
					typ = ""
				}
				if !ints[typ] || len(vs.Names) == 0 {
					continue
				}
				e := c.enums[typ]
				e.values++
				if !e.hasIt && i >= 0 && vs.Type != nil && len(vs.Values) == 1 && isPlainIota(vs.Values[0]) {
					e.hasIt, e.first, e.pos = true, vs.Names[0].Name, vs.Names[0].Pos()
				}
				c.enums[typ] = e
			}
		}
	}
	for name, e := range c.enums {
		if !e.hasIt {
			delete(c.enums, name)
		}
	}
}

// isPlainIota says whether the value is iota itself: the first value of an enum that counts
// from zero. A value such as 1<<iota or iota+1 is not zero, so the type has no named zero.
func isPlainIota(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "iota"
}

func isInteger(name string) bool {
	switch name {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return true
	}
	return false
}

type param struct {
	name     string
	typ      ast.Expr
	variadic bool
}

// flatten lists the parameters one by one, so "a, b int" is two.
func flatten(fl *ast.FieldList) []param {
	var out []param
	if fl == nil {
		return nil
	}
	for _, f := range fl.List {
		typ := f.Type
		variadic := false
		if e, ok := typ.(*ast.Ellipsis); ok {
			typ, variadic = e.Elt, true
		}
		if len(f.Names) == 0 {
			out = append(out, param{typ: typ, variadic: variadic})
			continue
		}
		for _, n := range f.Names {
			out = append(out, param{name: n.Name, typ: typ, variadic: variadic})
		}
	}
	return out
}

// typeName is the name of a type without its package, pointer or type arguments.
func typeName(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.StarExpr:
		return typeName(e.X)
	case *ast.IndexExpr:
		return typeName(e.X)
	case *ast.IndexListExpr:
		return typeName(e.X)
	}
	return ""
}

// mutableKind says why a package variable with this type and value can be changed in place,
// or returns "" when it is a value that is only read (a struct, a pointer to one, or the
// result of a call, like Colors or Curves).
func mutableKind(typ, value ast.Expr) string {
	if typ != nil {
		switch t := typ.(type) {
		case *ast.MapType:
			return "a map"
		case *ast.ArrayType:
			return "a slice or an array"
		case *ast.FuncType:
			return "a function"
		case *ast.ChanType:
			return "a channel"
		case *ast.Ident:
			if isBasic(t.Name) {
				return "a value of type " + t.Name
			}
		}
	}
	switch v := value.(type) {
	case nil:
		if typ == nil {
			return "no value"
		}
	case *ast.BasicLit:
		return "a literal"
	case *ast.FuncLit:
		return "a function"
	case *ast.CompositeLit:
		switch v.Type.(type) {
		case *ast.MapType:
			return "a map"
		case *ast.ArrayType:
			return "a slice or an array"
		}
	case *ast.Ident:
		if v.Name == "true" || v.Name == "false" {
			return "a bool"
		}
	case *ast.CallExpr:
		if id, ok := v.Fun.(*ast.Ident); ok && (id.Name == "make" || id.Name == "new") {
			return "the result of " + id.Name
		}
	}
	return ""
}

func isBasic(name string) bool {
	switch name {
	case "bool", "string", "float32", "float64", "error", "any":
		return true
	}
	return isInteger(name)
}

// collectAssigned finds the package variables that a statement outside their declaration
// assigns to or changes.
func (c *checker) collectAssigned(files []*ast.File) {
	c.assigned = map[string]bool{}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				if n.Tok == token.DEFINE {
					return true
				}
				for _, l := range n.Lhs {
					if id, ok := l.(*ast.Ident); ok {
						c.assigned[id.Name] = true
					}
				}
			case *ast.IncDecStmt:
				if id, ok := n.X.(*ast.Ident); ok {
					c.assigned[id.Name] = true
				}
			}
			return true
		})
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagApiCheckAllowSyntax     = "ApiCheckAllowSyntax"
	tagApiCheckAllowUnreadable = "ApiCheckAllowUnreadable"
	tagApiCheckBadFlags        = "ApiCheckBadFlags"
	tagApiCheckParseFailed     = "ApiCheckParseFailed"
	tagApiCheckViolations      = "ApiCheckViolations"
	tagApiCheckWriteFailed     = "ApiCheckWriteFailed"
)
