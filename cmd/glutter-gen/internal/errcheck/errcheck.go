// Package errcheck implements the `glutter-gen errcheck` subcommand, which reads the whole
// module and reports what breaks the rules of github.com/rm4n0s/errors (its AI_GUIDE.md,
// summarized in CLAUDE.md, "Error handling", and SPEC §14.4).
//
// Flutter: no counterpart.
package errcheck

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// errorsPath is the import path of the errors package.
const errorsPath = "github.com/rm4n0s/errors"

// modulePath is the import path of the module that is checked.
const modulePath = "github.com/rm4n0s/glutter"

// The rules, as they are named in a report and in an allow file.
const (
	// RuleStdErrors says that the standard library's errors package is not imported under
	// the name errors; rm4n0s/errors re-exports Is, As and Unwrap (R6).
	RuleStdErrors = "std-errors"
	// RuleFmtErrorf says that fmt.Errorf is not called; errors.NewErrf is its counterpart.
	RuleFmtErrorf = "fmt-errorf"
	// RuleMetadataChain says that the metadata of a new error is given in one call: to
	// errors.New itself, or in one SetMetadata after NewErr and NewErrf (R2).
	RuleMetadataChain = "metadata-chain"
	// RuleMetadataPairs says that metadata is alternating string keys and values (R2).
	RuleMetadataPairs = "metadata-pairs"
	// RuleSentinelCtor says that a package-level variable is not made by errors.New,
	// NewErr or NewErrf, which would capture the stack of init (R5).
	RuleSentinelCtor = "sentinel-ctor"
	// RuleErrorHelper says that a function that only builds an error and returns it is not
	// written: the constructor is called in the function that fails, so that the route is
	// right (R4).
	RuleErrorHelper = "error-helper"
	// RuleErrorRewrap says that an error that comes from a function of the module is
	// returned as it is, not wrapped again by NewErr or NewErrf (R3, case A).
	RuleErrorRewrap = "error-rewrap"
	// RuleErrorUntagged says that an error that comes from outside the module is tagged by
	// NewErr where it first appears, not returned as it is (R3, case B).
	RuleErrorUntagged = "error-untagged"
	// RuleTagBranch says that code does not compare the Tag of an error with a string
	// literal, or branch on its message; it uses a tag constant or errors.Is (R5). Tests are
	// not checked: a test pins the string that is the contract (a public tag, or the text of a
	// message that a ported Flutter test asserts).
	RuleTagBranch = "tag-branch"
	// RuleTagLiteral says that the tag given to errors.New, NewErr or NewErrf is a constant
	// or an expression, not a string literal (R5).
	RuleTagLiteral = "tag-literal"
	// RuleErrorToJson says that an error is logged with slog as "err", err, not through
	// ToJson (guide §5). Tests may call ToJson to replay an error.
	RuleErrorToJson = "error-tojson"
)

// Violation is one place where the code breaks a rule.
//
// Flutter: no counterpart.
type Violation struct {
	// Rule is one of the Rule constants.
	Rule string
	// File is the path of the file below the root, with slashes.
	File string
	// Func is the function that holds the code: "Name", "Type.Name", or "-" outside any.
	Func string
	// Line is where the problem is.
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

// Pass is one way of building the module: a platform and a set of build tags.
//
// Flutter: no counterpart.
type Pass struct {
	GOOS, GOARCH string
	Tags         []string
}

// AndroidPass is the build for Android. The module is built with cgo and the NDK's clang
// there (SPEC §24.2), so that the *_android.go files of internal/platform/android are read.
//
// Flutter: no counterpart.
var AndroidPass = Pass{GOOS: "android", GOARCH: "arm64"}

// WindowsPass is the build for Windows with the tag of the Wine tests, so that the *_windows.go
// files and the tests that run inside Wine are read (SPEC §25.13). It needs no toolchain besides
// the go command: Windows is built without cgo.
//
// Flutter: no counterpart.
var WindowsPass = Pass{GOOS: "windows", GOARCH: "amd64", Tags: []string{"glutter_wine"}}

// DefaultPasses are the builds that together cover every file of the module: Linux, the
// web, Linux with the tags of the display, browser and Wine tests (the host-side tests that start
// Wine), Android (SPEC §24.13) and Windows with the tag of the Wine tests (SPEC §25.13).
//
// Flutter: no counterpart.
var DefaultPasses = []Pass{
	{GOOS: "linux", GOARCH: "amd64"},
	{GOOS: "js", GOARCH: "wasm"},
	{GOOS: "linux", GOARCH: "amd64", Tags: []string{"glutter_display", "glutter_browser", "glutter_wine"}},
	AndroidPass,
	WindowsPass,
}

// androidMinSDK is the API level of the NDK's clang (SPEC §24.1, D24).
const androidMinSDK = 26

// androidTriples maps a GOARCH to the triple of the NDK's clang.
var androidTriples = map[string]string{"arm64": "aarch64-linux-android", "amd64": "x86_64-linux-android"}

// ndkClang finds the NDK's clang for an architecture: the NDK is $ANDROID_NDK_HOME, else the
// newest $ANDROID_HOME/ndk/<version>. It fails with ErrCheckAndroidNDKNotFound, which names the
// package to install, when there is none.
func ndkClang(getenv func(string) string, goarch string) (string, error) {
	ndk := getenv("ANDROID_NDK_HOME")
	if ndk == "" {
		home := getenv("ANDROID_HOME")
		if home == "" {
			home = getenv("ANDROID_SDK_ROOT")
		}
		versions, _ := filepath.Glob(filepath.Join(home, "ndk", "*"))
		sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
		if home != "" && len(versions) > 0 {
			ndk = versions[len(versions)-1]
		}
	}
	triple, ok := androidTriples[goarch]
	if !ok {
		return "", errors.New(TagErrCheckAndroidNDKNotFound, "no NDK clang is known for this architecture", "goarch", goarch)
	}
	cc := filepath.Join(ndk, "toolchains", "llvm", "prebuilt", "linux-x86_64", "bin", fmt.Sprintf("%s%d-clang", triple, androidMinSDK))
	if ndk == "" {
		return "", errors.New(TagErrCheckAndroidNDKNotFound, "the Android NDK was not found; set ANDROID_NDK_HOME or ANDROID_HOME",
			"install", `sdkmanager "ndk;30.0.16248370"`)
	}
	if _, err := os.Stat(cc); err != nil {
		return "", errors.New(TagErrCheckAndroidNDKNotFound, "the Android NDK has no clang for this architecture",
			"clang", cc, "install", `sdkmanager "ndk;30.0.16248370"`)
	}
	return cc, nil
}

// Run is the subcommand. It prints the violations that the allow file does not list and
// fails with ErrCheckViolations when there are some.
//
// Flutter: no counterpart.
func Run(args []string) error { return run(args, os.Stdout) }

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("errcheck", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "root directory of the module")
	allowPath := flags.String("allow", "", "file that lists the accepted violations, one \"rule file:Func\" per line, with # comments")
	if err := flags.Parse(args); err != nil {
		return errors.NewErr(tagErrCheckBadFlags, err).SetMetadata("args", args)
	}
	allowed := map[string]bool{}
	if *allowPath != "" {
		var err error
		if allowed, err = ReadAllow(*allowPath); err != nil {
			return err
		}
	}
	found, err := CheckModule(context.Background(), *root, DefaultPasses)
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
			return errors.NewErr(tagErrCheckWriteFailed, err)
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
			return errors.NewErr(tagErrCheckWriteFailed, err)
		}
	}
	if _, err := fmt.Fprintf(stdout, "errcheck: %d violations, %d accepted by the allow file\n", len(left), skipped); err != nil {
		return errors.NewErr(tagErrCheckWriteFailed, err)
	}
	if len(left) > 0 || len(stale) > 0 {
		return errors.New(tagErrCheckViolations, "the code breaks the rules of rm4n0s/errors", "violations", len(left), "stale", len(stale))
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
		return nil, errors.NewErr(tagErrCheckAllowUnreadable, err).SetMetadata("file", path)
	}
	out := map[string]bool{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line, _, _ := strings.Cut(sc.Text(), "#")
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 2 || !strings.Contains(fields[1], ":") {
			return nil, errors.New(tagErrCheckAllowSyntax, "a line of the allow file is not \"rule file:Func\"", "line", sc.Text(), "file", path)
		}
		out[fields[0]+" "+fields[1]] = true
	}
	if err := sc.Err(); err != nil {
		return nil, errors.NewErr(tagErrCheckAllowUnreadable, err).SetMetadata("file", path)
	}
	return out, nil
}

// listedPackage is what `go list -json` says about a package.
type listedPackage struct {
	Dir, ImportPath, Export            string
	DepOnly                            bool
	GoFiles, TestGoFiles, XTestGoFiles []string
}

// CheckModule checks every package of the module below root for each pass and returns the
// violations, without duplicates, in the order of their files and lines.
//
// Flutter: no counterpart.
func CheckModule(ctx context.Context, root string, passes []Pass) ([]Violation, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, errors.NewErr(tagErrCheckRootInvalid, err).SetMetadata("root", root)
	}
	seen := map[string]Violation{}
	for _, pass := range passes {
		vs, err := checkPass(ctx, absRoot, pass)
		if err != nil {
			return nil, err
		}
		for _, v := range vs {
			seen[v.File+":"+strconv.Itoa(v.Line)+":"+v.Rule+":"+v.Message] = v
		}
	}
	out := make([]Violation, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Rule < out[j].Rule
	})
	return out, nil
}

// checkPass lists the packages of one build with the compiled export data of everything
// they import (the go build cache makes that cheap), and reads each one with go/types.
func checkPass(ctx context.Context, root string, pass Pass) ([]Violation, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-e", "-export", "-deps", "-test",
		"-json=ImportPath,Dir,Export,DepOnly,GoFiles,TestGoFiles,XTestGoFiles",
		"-tags", strings.Join(pass.Tags, ","), "./...")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOOS="+pass.GOOS, "GOARCH="+pass.GOARCH, "CGO_ENABLED=0")
	if pass.GOOS == "android" {
		cc, err := ndkClang(os.Getenv, pass.GOARCH)
		if err != nil {
			return nil, err
		}
		cmd.Env = append(cmd.Env, "CGO_ENABLED=1", "CC="+cc)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.NewErr(tagErrCheckListFailed, err).SetMetadata("stderr", stderr.String(), "goos", pass.GOOS, "tags", pass.Tags)
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	exports := map[string]string{}
	var pkgs []listedPackage
	for dec.More() {
		var p listedPackage
		if err := dec.Decode(&p); err != nil {
			return nil, errors.NewErr(tagErrCheckListInvalid, err)
		}
		if strings.ContainsAny(p.ImportPath, " ") || strings.HasSuffix(p.ImportPath, ".test") {
			continue
		}
		if p.Export != "" {
			exports[p.ImportPath] = p.Export
		}
		if !p.DepOnly && strings.HasPrefix(p.ImportPath, modulePath) {
			pkgs = append(pkgs, p)
		}
	}

	fset := token.NewFileSet()
	imp := importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		file, ok := exports[path]
		if !ok {
			return nil, errors.New(tagErrCheckNoExport, "a package has no export data", "path", path)
		}
		f, err := os.Open(file)
		if err != nil {
			return nil, errors.NewErr(tagErrCheckExportUnreadable, err).SetMetadata("path", path)
		}
		return f, nil
	})
	var found []Violation
	for _, p := range pkgs {
		inPkg := append(append([]string{}, p.GoFiles...), p.TestGoFiles...)
		for i, group := range [][]string{inPkg, p.XTestGoFiles} {
			if len(group) == 0 {
				continue
			}
			files := make([]*ast.File, 0, len(group))
			for _, name := range group {
				path := filepath.Join(p.Dir, name)
				f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
				if err != nil {
					return nil, errors.NewErr(tagErrCheckParseFailed, err).SetMetadata("file", path)
				}
				files = append(files, f)
			}
			info := &types.Info{
				Uses:       map[*ast.Ident]types.Object{},
				Defs:       map[*ast.Ident]types.Object{},
				Types:      map[ast.Expr]types.TypeAndValue{},
				Selections: map[*ast.SelectorExpr]*types.Selection{},
			}
			conf := types.Config{Importer: imp, Error: func(error) {}}
			path := p.ImportPath
			if i == 1 {
				path += "_test"
			}
			// A type error leaves the information partial; the rules read what is there.
			_, _ = conf.Check(path, fset, files, info)
			c := &checker{fset: fset, info: info, root: root}
			for _, f := range files {
				c.checkFile(f)
			}
			found = append(found, c.found...)
		}
	}
	return found, nil
}

type checker struct {
	fset  *token.FileSet
	info  *types.Info
	root  string
	found []Violation
	// assignments are the assignments of the function being read, by variable.
	assignments map[types.Object]assignList
}

// assignment is one value given to a variable: the call that gave it, or nil.
type assignment struct {
	pos  token.Pos
	call *ast.CallExpr
}

type assignList []assignment

// before returns the call of the last assignment before pos, or nil.
func (l assignList) before(pos token.Pos) *ast.CallExpr {
	var best *assignment
	for i := range l {
		if l[i].pos < pos && (best == nil || l[i].pos > best.pos) {
			best = &l[i]
		}
	}
	if best == nil {
		return nil
	}
	return best.call
}

// collectAssignments reads the assignments to variables in a function body.
func (c *checker) collectAssignments(body *ast.BlockStmt) {
	c.assignments = map[types.Object]assignList{}
	record := func(id *ast.Ident, pos token.Pos, call *ast.CallExpr) {
		obj := c.info.Defs[id]
		if obj == nil {
			obj = c.info.Uses[id]
		}
		if obj != nil {
			c.assignments[obj] = append(c.assignments[obj], assignment{pos: pos, call: call})
		}
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for i, l := range n.Lhs {
				id, ok := l.(*ast.Ident)
				if !ok {
					continue
				}
				var call *ast.CallExpr
				switch {
				case len(n.Rhs) == 1:
					call, _ = ast.Unparen(n.Rhs[0]).(*ast.CallExpr)
				case i < len(n.Rhs):
					call, _ = ast.Unparen(n.Rhs[i]).(*ast.CallExpr)
				}
				record(id, n.End(), call)
			}
		case *ast.ValueSpec:
			for i, id := range n.Names {
				var call *ast.CallExpr
				switch {
				case len(n.Values) == 1:
					call, _ = ast.Unparen(n.Values[0]).(*ast.CallExpr)
				case i < len(n.Values):
					call, _ = ast.Unparen(n.Values[i]).(*ast.CallExpr)
				}
				record(id, n.End(), call)
			}
		}
		return true
	})
}

func (c *checker) add(rule string, pos token.Pos, fn, format string, args ...any) {
	p := c.fset.Position(pos)
	rel, err := filepath.Rel(c.root, p.Filename)
	if err != nil {
		rel = p.Filename
	}
	c.found = append(c.found, Violation{
		Rule: rule, File: filepath.ToSlash(rel), Func: fn, Line: p.Line,
		Message: fmt.Sprintf(format, args...),
	})
}

func (c *checker) isTest(pos token.Pos) bool {
	return strings.HasSuffix(c.fset.Position(pos).Filename, "_test.go")
}

func (c *checker) checkFile(f *ast.File) {
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if path == "errors" && (imp.Name == nil || imp.Name.Name == "errors") {
			c.add(RuleStdErrors, imp.Pos(), "-", "the standard library's errors is imported; import %s, which re-exports Is, As and Unwrap, or alias the standard one (stdErrors)", errorsPath)
		}
	}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			if d.Tok != token.VAR {
				continue
			}
			for _, spec := range d.Specs {
				vs := spec.(*ast.ValueSpec)
				c.assignments = nil
				for _, v := range vs.Values {
					c.walk(v, "-")
					if c.isErrorsCtor(v, "New", "NewErr", "NewErrf") {
						c.add(RuleSentinelCtor, v.Pos(), "-", "a package-level variable is made by an errors constructor; use &errors.Error{Tag: ...} for a sentinel")
					}
				}
			}
		case *ast.FuncDecl:
			if d.Body == nil {
				continue
			}
			name := funcName(d)
			c.collectAssignments(d.Body)
			c.walk(d.Body, name)
			c.checkHelper(d, name)
			c.checkFlow(d, name)
		}
	}
}

func funcName(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return d.Name.Name
	}
	t := d.Recv.List[0].Type
	if s, ok := t.(*ast.StarExpr); ok {
		t = s.X
	}
	if ix, ok := t.(*ast.IndexExpr); ok {
		t = ix.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name + "." + d.Name.Name
	}
	return d.Name.Name
}

// errorsFunc says which function of rm4n0s/errors a call is, or "".
func (c *checker) errorsFunc(e ast.Expr) string {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	if pn, ok := c.info.Uses[id].(*types.PkgName); ok && pn.Imported().Path() == errorsPath {
		return sel.Sel.Name
	}
	return ""
}

func (c *checker) isErrorsCtor(e ast.Expr, names ...string) bool {
	got := c.errorsFunc(e)
	for _, n := range names {
		if got == n {
			return true
		}
	}
	return false
}

// isMethodOnError says whether call is x.name(...) with x an *errors.Error.
func (c *checker) isMethodOnError(call *ast.CallExpr, name string) (*ast.SelectorExpr, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return nil, false
	}
	if s, ok := c.info.Selections[sel]; ok {
		return sel, isAppError(s.Recv())
	}
	return nil, false
}

func isAppError(t types.Type) bool {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	n, ok := t.(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == errorsPath && n.Obj().Name() == "Error"
}

func (c *checker) walk(root ast.Node, fn string) {
	ast.Inspect(root, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			c.checkCall(n, fn)
		case *ast.BinaryExpr:
			c.checkCompare(n, fn)
		case *ast.SwitchStmt:
			c.checkSwitch(n, fn)
		}
		return true
	})
}

func (c *checker) checkCall(call *ast.CallExpr, fn string) {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "fmt" && sel.Sel.Name == "Errorf" {
			if pn, ok := c.info.Uses[id].(*types.PkgName); ok && pn.Imported().Path() == "fmt" {
				c.add(RuleFmtErrorf, call.Pos(), fn, "fmt.Errorf is called; use errors.New with metadata, or errors.NewErrf")
			}
		}
	}
	switch c.errorsFunc(call) {
	case "New":
		if len(call.Args) >= 2 {
			c.checkTag(call, fn)
			c.checkPairs(call.Args[2:], call, fn)
		}
	case "NewErr", "NewErrf":
		c.checkTag(call, fn)
		c.checkRewrap(call, fn)
	}
	if sel, ok := c.isMethodOnError(call, "SetMetadata"); ok {
		c.checkPairs(call.Args, call, fn)
		switch recv := ast.Unparen(sel.X).(type) {
		case *ast.CallExpr:
			if c.isErrorsCtor(recv, "New") {
				c.add(RuleMetadataChain, call.Pos(), fn, "SetMetadata after errors.New; give the pairs to errors.New")
			} else if _, ok := c.isMethodOnError(recv, "SetMetadata"); ok {
				c.add(RuleMetadataChain, call.Pos(), fn, "two SetMetadata calls in a chain; give all the pairs in one")
			}
		}
	}
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok && len(call.Args) == 2 && !c.isTest(call.Pos()) && c.isPackageFunc(sel, "strings", "Contains", "HasPrefix", "HasSuffix", "EqualFold") && c.isErrorText(call.Args[0]) {
		c.add(RuleTagBranch, call.Pos(), fn, "the message of an error is searched; branch on its Tag")
	}
	if _, ok := c.isMethodOnError(call, "ToJson"); ok && !c.isTest(call.Pos()) {
		c.add(RuleErrorToJson, call.Pos(), fn, "ToJson is called to log an error; log it with slog as \"err\", err")
	}
}

// isPackageFunc says whether sel is pkg.Name with Name among names.
func (c *checker) isPackageFunc(sel *ast.SelectorExpr, pkg string, names ...string) bool {
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	pn, ok := c.info.Uses[id].(*types.PkgName)
	if !ok || pn.Imported().Path() != pkg {
		return false
	}
	for _, n := range names {
		if sel.Sel.Name == n {
			return true
		}
	}
	return false
}

func (c *checker) checkTag(call *ast.CallExpr, fn string) {
	if len(call.Args) == 0 {
		return
	}
	if lit, ok := ast.Unparen(call.Args[0]).(*ast.BasicLit); ok && lit.Kind == token.STRING {
		c.add(RuleTagLiteral, call.Pos(), fn, "the tag %s is a string literal; declare it as a constant", lit.Value)
	}
}

func (c *checker) checkPairs(args []ast.Expr, call *ast.CallExpr, fn string) {
	if call.Ellipsis.IsValid() {
		return
	}
	if len(args)%2 != 0 {
		c.add(RuleMetadataPairs, call.Pos(), fn, "the metadata has %d values; it needs key, value pairs", len(args))
	}
	for i := 0; i < len(args); i += 2 {
		if lit, ok := args[i].(*ast.BasicLit); ok && lit.Kind != token.STRING {
			c.add(RuleMetadataPairs, args[i].Pos(), fn, "the metadata key %s is not a string", lit.Value)
		}
	}
}

// checkRewrap reports a NewErr or NewErrf that wraps an error that came from the module.
func (c *checker) checkRewrap(call *ast.CallExpr, fn string) {
	var wrapped []ast.Expr
	switch c.errorsFunc(call) {
	case "NewErr":
		if len(call.Args) == 2 {
			wrapped = call.Args[1:]
		}
	case "NewErrf":
		if len(call.Args) > 2 {
			wrapped = call.Args[2:]
		}
	}
	for _, w := range wrapped {
		if t := c.info.TypeOf(w); t == nil || t.String() != "error" {
			continue
		}
		if src := c.sourceOf(w, call.Pos()); src != nil {
			if own, name := c.isOwnFunc(src); own {
				c.add(RuleErrorRewrap, call.Pos(), fn, "wraps the error of %s, which is already tagged; return it as it is", name)
			}
		}
	}
}

// sourceOf finds the call that gave an error expression its value: the call itself, or the
// nearest assignment of an identifier before pos in the same function.
func (c *checker) sourceOf(e ast.Expr, pos token.Pos) *ast.CallExpr {
	switch e := ast.Unparen(e).(type) {
	case *ast.CallExpr:
		return e
	case *ast.Ident:
		obj := c.info.Uses[e]
		if obj == nil {
			return nil
		}
		return c.assignments[obj].before(pos)
	}
	return nil
}

func (c *checker) isOwnFunc(call *ast.CallExpr) (bool, string) {
	var id *ast.Ident
	switch f := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		id = f
	case *ast.SelectorExpr:
		id = f.Sel
	default:
		return false, ""
	}
	fn, ok := c.info.Uses[id].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false, ""
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Results().Len() == 0 || sig.Results().At(sig.Results().Len()-1).Type().String() != "error" {
		return false, ""
	}
	return strings.HasPrefix(fn.Pkg().Path(), modulePath), fn.FullName()
}

// checkHelper reports a function that only builds an error and returns it (or panics with
// it): the constructor belongs in the function that fails (R4).
//
// An exported function or method is not a helper: it is part of an API or implements an
// interface, and its whole job can be to fail (an abstract method that panics).
func (c *checker) checkHelper(d *ast.FuncDecl, name string) {
	if c.isTest(d.Pos()) || d.Name.IsExported() {
		return
	}
	if d.Type.Results == nil || len(d.Type.Results.List) == 0 {
		if len(d.Body.List) == 1 {
			if es, ok := d.Body.List[0].(*ast.ExprStmt); ok {
				if call, ok := es.X.(*ast.CallExpr); ok {
					if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "panic" && len(call.Args) == 1 && c.isErrorsCtor(call.Args[0], "New", "NewErr", "NewErrf") {
						c.add(RuleErrorHelper, d.Pos(), name, "the function only panics with a new error; call errors.New in the function that fails")
					}
				}
			}
		}
		return
	}
	if len(d.Type.Results.List) != 1 || len(d.Type.Results.List[0].Names) > 1 {
		return
	}
	if t := c.info.TypeOf(d.Type.Results.List[0].Type); t == nil || (t.String() != "error" && !isAppError(t)) {
		return
	}
	ctors, others := 0, 0
	ast.Inspect(d.Body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		if rs, ok := n.(*ast.ReturnStmt); ok {
			if len(rs.Results) == 1 && c.isErrorsCtor(rs.Results[0], "New", "NewErr", "NewErrf") {
				ctors++
			} else {
				others++
			}
		}
		return true
	})
	if ctors > 0 && others == 0 {
		c.add(RuleErrorHelper, d.Pos(), name, "the function only builds and returns an error; call errors.New in the function that fails")
	}
}

// checkFlow reports an error from outside the module that is returned as it is.
func (c *checker) checkFlow(d *ast.FuncDecl, name string) {
	if c.isTest(d.Pos()) {
		return
	}
	ast.Inspect(d.Body, func(n ast.Node) bool {
		rs, ok := n.(*ast.ReturnStmt)
		if !ok || len(rs.Results) == 0 {
			return true
		}
		last := ast.Unparen(rs.Results[len(rs.Results)-1])
		var src *ast.CallExpr
		switch e := last.(type) {
		case *ast.Ident:
			if t := c.info.TypeOf(e); t == nil || t.String() != "error" {
				return true
			}
			if _, isVar := c.info.Uses[e].(*types.Var); !isVar {
				return true
			}
			src = c.sourceOf(e, rs.Pos())
		case *ast.CallExpr:
			if len(rs.Results) != 1 {
				return true
			}
			src = e
		}
		if src == nil {
			return true
		}
		if foreign, callee := c.isForeignFunc(src); foreign {
			c.add(RuleErrorUntagged, rs.Pos(), name, "returns the error of %s as it is; tag it here with errors.NewErr", callee)
		}
		return true
	})
}

// isForeignFunc says whether call is to a function from outside the module whose last
// result is an error. The functions of rm4n0s/errors itself are not foreign.
func (c *checker) isForeignFunc(call *ast.CallExpr) (bool, string) {
	var id *ast.Ident
	switch f := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		id = f
	case *ast.SelectorExpr:
		id = f.Sel
	default:
		return false, ""
	}
	fn, ok := c.info.Uses[id].(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false, ""
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Results().Len() == 0 || sig.Results().At(sig.Results().Len()-1).Type().String() != "error" {
		return false, ""
	}
	path := fn.Pkg().Path()
	if strings.HasPrefix(path, modulePath) || path == errorsPath {
		return false, ""
	}
	return true, fn.FullName()
}

func (c *checker) isErrorText(e ast.Expr) bool {
	e = ast.Unparen(e)
	if call, ok := e.(*ast.CallExpr); ok && len(call.Args) == 0 {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Error" {
			if t := c.info.TypeOf(sel.X); t != nil && types.Implements(t, errorInterface) {
				return true
			}
		}
	}
	if sel, ok := e.(*ast.SelectorExpr); ok && sel.Sel.Name == "Message" {
		if t := c.info.TypeOf(sel.X); t != nil && isAppError(t) {
			return true
		}
	}
	return false
}

var errorInterface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

func (c *checker) isTagOf(e ast.Expr) bool {
	sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Tag" {
		return false
	}
	t := c.info.TypeOf(sel.X)
	return t != nil && isAppError(t)
}

func isStringLit(e ast.Expr) bool {
	lit, ok := ast.Unparen(e).(*ast.BasicLit)
	return ok && lit.Kind == token.STRING
}

func (c *checker) checkCompare(b *ast.BinaryExpr, fn string) {
	if (b.Op != token.EQL && b.Op != token.NEQ) || c.isTest(b.Pos()) {
		return
	}
	for _, pair := range [][2]ast.Expr{{b.X, b.Y}, {b.Y, b.X}} {
		if !isStringLit(pair[1]) {
			continue
		}
		if c.isTagOf(pair[0]) {
			c.add(RuleTagBranch, b.Pos(), fn, "the Tag is compared with a string literal; compare with a tag constant or use errors.Is")
		} else if c.isErrorText(pair[0]) {
			c.add(RuleTagBranch, b.Pos(), fn, "the message of an error is compared; branch on its Tag")
		}
	}
}

func (c *checker) checkSwitch(s *ast.SwitchStmt, fn string) {
	if s.Tag == nil || c.isTest(s.Pos()) {
		return
	}
	isTag, isText := c.isTagOf(s.Tag), c.isErrorText(s.Tag)
	if !isTag && !isText {
		return
	}
	for _, cc := range s.Body.List {
		for _, e := range cc.(*ast.CaseClause).List {
			if isStringLit(e) {
				c.add(RuleTagBranch, e.Pos(), fn, "a switch on an error's Tag or message has a string literal case; use tag constants")
			}
		}
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagErrCheckAndroidNDKNotFound is the tag of the error for a missing Android NDK; the
	// metadata "install" has the sdkmanager command.
	TagErrCheckAndroidNDKNotFound = "ErrCheckAndroidNDKNotFound"
	tagErrCheckAllowSyntax        = "ErrCheckAllowSyntax"
	tagErrCheckAllowUnreadable    = "ErrCheckAllowUnreadable"
	tagErrCheckBadFlags           = "ErrCheckBadFlags"
	tagErrCheckExportUnreadable   = "ErrCheckExportUnreadable"
	tagErrCheckListFailed         = "ErrCheckListFailed"
	tagErrCheckListInvalid        = "ErrCheckListInvalid"
	tagErrCheckNoExport           = "ErrCheckNoExport"
	tagErrCheckParseFailed        = "ErrCheckParseFailed"
	tagErrCheckRootInvalid        = "ErrCheckRootInvalid"
	tagErrCheckViolations         = "ErrCheckViolations"
	tagErrCheckWriteFailed        = "ErrCheckWriteFailed"
)
