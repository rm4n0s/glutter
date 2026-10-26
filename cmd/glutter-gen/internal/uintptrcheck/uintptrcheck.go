// Package uintptrcheck implements the `glutter-gen uintptrcheck` subcommand, which reads the
// Windows code and reports a Go pointer that is turned into a uintptr anywhere but in the
// argument list of a call that keeps it alive (SPEC §25.2).
//
// A uintptr does not keep an object alive, and does not follow it when the stack moves. The
// compiler makes an exception for the arguments of a call to a function marked
// //go:uintptrescapes and for syscall.SyscallN: the object stays alive, on the heap, for the
// call. A conversion written earlier (into a variable, a table, a struct, or in an if that picks
// between a pointer and 0) has no such protection, and is wrong whenever the call runs Go code
// before it returns (a COM method of an object made in Go, a window procedure, a callback).
//
// The check reads the syntax only, so it matches a callee by its name: the functions marked in
// the checked packages and the Syscall functions of the standard library.
//
// Flutter: no counterpart.
package uintptrcheck

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rm4n0s/errors"
)

// DefaultDirs are the directories that call Windows, below the root of the module.
//
// Flutter: no counterpart.
var DefaultDirs = []string{
	"internal/platform/windows", "internal/gpu/d3d12", "internal/gpu/gdiwindow",
	"internal/engine", "cmd/glutter-windows",
}

// RulePtrEscape says that uintptr(unsafe.Pointer(p)) is written in the argument list of a call
// to a function marked //go:uintptrescapes, or of syscall.SyscallN, and nowhere else.
const RulePtrEscape = "ptr-escape"

// Violation is one place where a pointer is converted outside such an argument list.
//
// Flutter: no counterpart.
type Violation struct {
	// Rule is RulePtrEscape.
	Rule string
	// File is the path of the file below the root, with slashes.
	File string
	// Func is the function that holds the code: "Name", "Type.Name", or "-" outside any.
	Func string
	// Line is where the conversion is.
	Line int
	// Message says what is wrong.
	Message string
}

// Key is how an allow file names the violation: "rule file:Func".
//
// Flutter: no counterpart.
func (v Violation) Key() string { return v.Rule + " " + v.File + ":" + v.Func }

// String is the line of the report.
//
// Flutter: no counterpart.
func (v Violation) String() string {
	return fmt.Sprintf("%s:%d: [%s] %s: %s", v.File, v.Line, v.Rule, v.Func, v.Message)
}

// Run is the subcommand: it checks DefaultDirs, drops what the allow file accepts, prints the
// rest and fails with UintptrCheckViolations when there are some or the allow file is stale.
//
// Flutter: no counterpart.
func Run(args []string) error { return run(args, os.Stdout) }

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("uintptrcheck", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "root directory of the module")
	dirs := flags.String("dirs", strings.Join(DefaultDirs, ","), "directories to check, below the root, separated by commas")
	allowPath := flags.String("allow", "", "file that lists the accepted violations, one \"rule file:Func\" per line, with # comments")
	if err := flags.Parse(args); err != nil {
		return errors.NewErr(tagUintptrCheckBadFlags, err).SetMetadata("args", args)
	}
	allowed := map[string]bool{}
	if *allowPath != "" {
		var err error
		if allowed, err = ReadAllow(*allowPath); err != nil {
			return err
		}
	}
	var list []string
	for _, d := range strings.Split(*dirs, ",") {
		if d = strings.TrimSpace(d); d != "" {
			list = append(list, d)
		}
	}
	found, err := Check(*root, list)
	if err != nil {
		return err
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
			return errors.NewErr(tagUintptrCheckWriteFailed, err)
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
			return errors.NewErr(tagUintptrCheckWriteFailed, err)
		}
	}
	if _, err := fmt.Fprintf(stdout, "uintptrcheck: %d violations, %d accepted by the allow file\n", len(left), skipped); err != nil {
		return errors.NewErr(tagUintptrCheckWriteFailed, err)
	}
	if len(left) > 0 || len(stale) > 0 {
		return errors.New(tagUintptrCheckViolations, "a Go pointer becomes a uintptr outside the arguments of a call that keeps it alive", "violations", len(left), "stale", len(stale))
	}
	return nil
}

// ReadAllow reads an allow file: each line is "rule file:Func", optionally followed by
// "# why". Blank lines and lines that start with # are skipped.
//
// Flutter: no counterpart.
func ReadAllow(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.NewErr(tagUintptrCheckAllowUnreadable, err).SetMetadata("file", path)
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
			return nil, errors.New(tagUintptrCheckAllowMalformed, "an allow line is \"rule file:Func\"", "file", path, "line", sc.Text())
		}
		out[fields[0]+" "+fields[1]] = true
	}
	return out, nil
}

// parsed is one Go file of the checked directories.
type parsed struct {
	path string
	file *ast.File
}

// Check reads every .go file below the directories (whatever its build tags) and returns the
// violations, sorted by file and line.
//
// Flutter: no counterpart.
func Check(root string, dirs []string) ([]Violation, error) {
	fset := token.NewFileSet()
	var files []parsed
	for _, dir := range dirs {
		base := filepath.Join(root, filepath.FromSlash(dir))
		var paths []string
		var walkErr error
		_ = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				walkErr = err
				return fs.SkipAll
			}
			if !d.IsDir() && strings.HasSuffix(p, ".go") {
				paths = append(paths, p)
			}
			return nil
		})
		if walkErr != nil {
			return nil, errors.NewErr(tagUintptrCheckWalkFailed, walkErr).SetMetadata("dir", dir)
		}
		for _, p := range paths {
			f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
			if err != nil {
				return nil, errors.NewErr(tagUintptrCheckParseFailed, err).SetMetadata("file", p)
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return nil, errors.NewErr(tagUintptrCheckParseFailed, err).SetMetadata("file", p)
			}
			files = append(files, parsed{path: filepath.ToSlash(rel), file: f})
		}
	}
	// The callees that keep their uintptr arguments alive, by name.
	safe := map[string]bool{"SyscallN": true, "Syscall": true, "Syscall6": true, "Syscall9": true, "Syscall12": true, "Syscall15": true, "Syscall18": true}
	for _, pf := range files {
		for _, decl := range pf.file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Doc == nil {
				continue
			}
			for _, c := range fd.Doc.List {
				if strings.TrimSpace(c.Text) == "//go:uintptrescapes" {
					safe[fd.Name.Name] = true
				}
			}
		}
	}
	var out []Violation
	for _, pf := range files {
		out = append(out, checkFile(fset, pf, safe)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}

func checkFile(fset *token.FileSet, pf parsed, safe map[string]bool) []Violation {
	// The conversions that sit directly in the argument list of a safe call.
	ok := map[*ast.CallExpr]bool{}
	ast.Inspect(pf.file, func(n ast.Node) bool {
		call, isCall := n.(*ast.CallExpr)
		if !isCall || !safe[calleeName(call.Fun)] {
			return true
		}
		for _, arg := range call.Args {
			markConversions(arg, ok)
		}
		return true
	})
	var out []Violation
	for _, decl := range pf.file.Decls {
		name := "-"
		if fd, isFunc := decl.(*ast.FuncDecl); isFunc {
			name = funcName(fd)
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			call, isCall := n.(*ast.CallExpr)
			if !isCall || !isPointerConversion(call) || ok[call] {
				return true
			}
			out = append(out, Violation{
				Rule: RulePtrEscape, File: pf.path, Func: name, Line: fset.Position(call.Pos()).Line,
				Message: "uintptr(unsafe.Pointer(...)) is not in the arguments of a //go:uintptrescapes or SyscallN call",
			})
			return true
		})
	}
	return out
}

// markConversions marks the pointer conversions that the compiler sees in an argument: the
// argument itself, through parentheses and the arithmetic of uintptr values.
func markConversions(e ast.Expr, ok map[*ast.CallExpr]bool) {
	switch e := e.(type) {
	case *ast.ParenExpr:
		markConversions(e.X, ok)
	case *ast.BinaryExpr:
		markConversions(e.X, ok)
		markConversions(e.Y, ok)
	case *ast.CallExpr:
		if isPointerConversion(e) {
			ok[e] = true
		}
	}
}

// isPointerConversion reports whether call is uintptr(unsafe.Pointer(x)).
func isPointerConversion(call *ast.CallExpr) bool {
	if id, isIdent := call.Fun.(*ast.Ident); !isIdent || id.Name != "uintptr" || len(call.Args) != 1 {
		return false
	}
	inner, isCall := call.Args[0].(*ast.CallExpr)
	if !isCall || len(inner.Args) != 1 {
		return false
	}
	sel, isSel := inner.Fun.(*ast.SelectorExpr)
	if !isSel || sel.Sel.Name != "Pointer" {
		return false
	}
	pkg, isIdent := sel.X.(*ast.Ident)
	return isIdent && pkg.Name == "unsafe"
}

// calleeName is the name of the function or method that a call names.
func calleeName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	case *ast.ParenExpr:
		return calleeName(f.X)
	}
	return ""
}

// funcName is "Name" or "Type.Name".
func funcName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	t := fd.Recv.List[0].Type
	for {
		switch x := t.(type) {
		case *ast.StarExpr:
			t = x.X
			continue
		case *ast.IndexExpr:
			t = x.X
			continue
		case *ast.Ident:
			return x.Name + "." + fd.Name.Name
		}
		return fd.Name.Name
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagUintptrCheckBadFlags        = "UintptrCheckBadFlags"
	tagUintptrCheckWriteFailed     = "UintptrCheckWriteFailed"
	tagUintptrCheckViolations      = "UintptrCheckViolations"
	tagUintptrCheckAllowUnreadable = "UintptrCheckAllowUnreadable"
	tagUintptrCheckAllowMalformed  = "UintptrCheckAllowMalformed"
	tagUintptrCheckParseFailed     = "UintptrCheckParseFailed"
	tagUintptrCheckWalkFailed      = "UintptrCheckWalkFailed"
)
