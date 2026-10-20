// Package reexport implements the `glutter-gen reexport` subcommand, which generates the wrappers that re-export package widgets from package material.
//
// Flutter: no counterpart.
package reexport

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/rm4n0s/errors"
)

const (
	defaultModule = "github.com/rm4n0s/glutter"
	defaultTarget = "material"
	defaultOut    = "material/widgets_reexport_gen.go"
)

// Spec says which identifiers of a package the target re-exports.
//
// Flutter: no counterpart.
type Spec struct {
	// Package is the path of the package below the module, such as "widgets".
	Package string
	// All re-exports every exported identifier of the package.
	All bool
	// Names are the names that Flutter's library exports from the package: a Go
	// identifier is re-exported when it is one of them, or starts with one of them
	// (optionally after New) and goes on with a capital letter or a digit, so
	// that the constructors, the options and the functions of a class come with
	// the class. The first letter of a name is made a capital.
	Names []string
	// Extra are exported Go identifiers that have no Flutter counterpart and are
	// re-exported anyway.
	Extra []string
}

// DefaultSpecs are the packages that Flutter's material.dart exports (SPEC
// §12.11): widgets.dart, which exports all of animation.dart and painting.dart,
// and the names that widgets.dart and the libraries below it re-export from
// foundation, rendering, gestures, services, physics, scheduler and dart:ui.
//
// Flutter: material.dart
func DefaultSpecs() []Spec {
	return []Spec{
		{Package: "widgets", All: true},
		{Package: "animation", All: true},
		{Package: "painting", All: true},
		{Package: "ui", Names: []string{
			"AppLifecycleState", "BlendMode", "BlurStyle", "Brightness", "Canvas", "Clip", "Color", "ColorFilter", "FilterQuality", "FontFeature",
			"FontStyle", "FontVariation", "FontWeight", "GlyphInfo", "ImageShader", "LineMetrics", "Locale", "MaskFilter", "Matrix4", "Offset",
			"Paint", "PaintingStyle", "Path", "PathFillType", "PathOperation", "PlaceholderAlignment", "RRect", "RSTransform", "RSuperellipse",
			"Radius", "Rect", "Shader", "Shadow", "Size", "StrokeCap", "StrokeJoin", "TextAffinity", "TextAlign", "TextBaseline", "TextBox",
			"TextDecoration", "TextDecorationStyle", "TextDirection", "TextHeightBehavior", "TextLeadingDistribution", "TextPosition", "TileMode",
			"VertexMode", "VoidCallback", "kTextHeightNone",
		}},
		{Package: "foundation", Names: []string{
			"Brightness", "ChangeNotifier", "DiagnosticLevel", "DiagnosticsNode", "ErrorDescription", "ErrorHint", "ErrorSummary", "FlutterError",
			"FlutterErrorDetails", "Key", "Listenable", "LocalKey", "TargetPlatform", "UniqueKey", "ValueChanged", "ValueGetter", "ValueKey",
			"ValueNotifier", "ValueSetter", "VoidCallback", "debugPrint", "debugPrintStack",
		}, Extra: []string{"Opt", "Some", "None"}},
		{Package: "rendering", Names: []string{
			"AlignmentGeometryTween", "AlignmentTween", "Axis", "AxisDirection", "BackdropKey", "BoxConstraints", "BoxConstraintsTransform",
			"CrossAxisAlignment", "CustomClipper", "CustomPainter", "CustomPainterSemantics", "DecorationPosition", "FixedColumnWidth",
			"FlexColumnWidth", "FlexFit", "FlowDelegate", "FlowPaintingContext", "FractionColumnWidth", "FractionalOffsetTween", "GrowthDirection",
			"HitTestBehavior", "ImageFilterConfig", "IntrinsicColumnWidth", "LayerLink", "MainAxisAlignment", "MainAxisSize", "MaxColumnWidth",
			"MinColumnWidth", "MouseCursor", "MultiChildLayoutDelegate", "PaintingContext", "PointerCancelEvent", "PointerCancelEventListener",
			"PointerDownEvent", "PointerDownEventListener", "PointerEvent", "PointerMoveEvent", "PointerMoveEventListener", "PointerUpEvent",
			"PointerUpEventListener", "RelativeRect", "RenderBox", "RenderObject", "RenderSemanticsGestureHandler", "SemanticsBuilderCallback",
			"ShaderCallback", "ShapeBorderClipper", "SingleChildLayoutDelegate", "SliverGridDelegate", "SliverGridDelegateWithFixedCrossAxisCount",
			"SliverGridDelegateWithMaxCrossAxisExtent", "SliverPaintOrder", "StackFit", "SystemMouseCursors", "TableBorder",
			"TableCellVerticalAlignment", "TableColumnWidth", "TextOverflow", "TextSelectionHandleType", "TextSelectionPoint", "WrapAlignment",
			"WrapCrossAlignment",
		}},
		{Package: "gestures", Names: []string{
			"DragDownDetails", "DragEndDetails", "DragStartDetails", "DragUpdateDetails", "ForcePressDetails", "GestureDragCancelCallback",
			"GestureDragDownCallback", "GestureDragEndCallback", "GestureDragStartCallback", "GestureDragUpdateCallback",
			"GestureForcePressEndCallback", "GestureForcePressPeakCallback", "GestureForcePressStartCallback", "GestureForcePressUpdateCallback",
			"GestureLongPressCallback", "GestureLongPressEndCallback", "GestureLongPressMoveUpdateCallback", "GestureLongPressStartCallback",
			"GestureLongPressUpCallback", "GestureScaleEndCallback", "GestureScaleStartCallback", "GestureScaleUpdateCallback", "GestureTapCallback",
			"GestureTapCancelCallback", "GestureTapDownCallback", "GestureTapUpCallback", "LongPressEndDetails", "LongPressMoveUpdateDetails",
			"LongPressStartDetails", "ScaleEndDetails", "ScaleStartDetails", "ScaleUpdateDetails", "TapDownDetails", "TapUpDetails", "Velocity",
		}},
		{Package: "services", Names: []string{
			"AssetBundle", "AutofillHints", "KeyEvent", "KeyboardInsertedContent", "RawKeyEvent", "RestorationBucket", "SelectionChangedCause",
			"SmartDashesType", "SmartQuotesType", "TextCapitalization", "TextEditingValue", "TextInputAction", "TextInputType", "TextSelection",
			"TextSelectionDelegate",
		}},
		{Package: "physics", Names: []string{"ScrollSpringSimulation", "Simulation", "Tolerance"}},
		{Package: "scheduler", Names: []string{"TickerProvider"}},
	}
}

// Options are the parameters of Generate.
//
// Flutter: no counterpart.
type Options struct {
	// Module is the module path, such as "github.com/rm4n0s/glutter".
	Module string
	// Target is the name of the package that gets the wrappers, such as
	// "material".
	Target string
	// Specs say what to re-export, in the order of priority: when two packages
	// have an identifier of the same name, the first wins.
	Specs []Spec
	// Packages are the type-checked source packages, by their path below the
	// module.
	Packages map[string]*types.Package
	// Declared are the exported identifiers that the target package declares
	// itself; they are not re-exported.
	Declared map[string]bool
}

// matches is whether the exported Go identifier belongs to one of the names that
// Flutter exports.
func matches(ident string, names []string) bool {
	trimmed := strings.TrimPrefix(ident, "New")
	for _, n := range names {
		if n == "" {
			continue
		}
		n = strings.ToUpper(n[:1]) + n[1:]
		for _, s := range []string{ident, trimmed} {
			if s == n || (strings.HasPrefix(s, n) && len(s) > len(n) && (unicode.IsUpper(rune(s[len(n)])) || unicode.IsDigit(rune(s[len(n)])))) {
				return true
			}
		}
	}
	return false
}

// Generate makes the source of the wrappers. It returns the formatted code.
//
// Flutter: no counterpart.
func Generate(o Options) ([]byte, error) {
	used := map[string]bool{}  // the paths of the packages that the wrappers name
	taken := map[string]bool{} // the names that have a wrapper
	var body bytes.Buffer
	var skipped, notCopied []string

	for _, spec := range o.Specs {
		pkg := o.Packages[spec.Package]
		if pkg == nil {
			return nil, errors.New("ReexportPackageMissing", "the package of a spec was not loaded", "package", spec.Package)
		}
		extra := map[string]bool{}
		for _, e := range spec.Extra {
			extra[e] = true
		}
		scope := pkg.Scope()
		for _, name := range scope.Names() {
			if !token.IsExported(name) || (!spec.All && !extra[name] && !matches(name, spec.Names)) {
				continue
			}
			if o.Declared[name] {
				skipped = append(skipped, spec.Package+"."+name)
				continue
			}
			if taken[name] {
				continue
			}
			var refs = map[string]bool{}
			qual := func(p *types.Package) string {
				if p == pkg {
					refs[p.Path()] = true
					return p.Name()
				}
				refs[p.Path()] = true
				return p.Name()
			}
			if mentionsUnexported(scope.Lookup(name).Type()) {
				continue
			}
			if v, ok := scope.Lookup(name).(*types.Var); ok {
				if _, basic := v.Type().Underlying().(*types.Basic); basic {
					// A copy of a flag or a number would not follow the writes to the original.
					notCopied = append(notCopied, spec.Package+"."+name)
					continue
				}
			}
			code, ok := wrapper(scope.Lookup(name), pkg, qual, spec.Package)
			if !ok {
				continue
			}
			internal := false
			for p := range refs {
				if strings.Contains(p, "/internal/") || strings.HasSuffix(p, "/internal") {
					internal = true
				}
			}
			if internal {
				continue
			}
			taken[name] = true
			for p := range refs {
				used[p] = true
			}
			used[pkg.Path()] = true
			body.WriteString(code)
			body.WriteString("\n")
		}
	}

	var b bytes.Buffer
	b.WriteString("// Code generated by glutter-gen reexport; DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "package %s\n\n", o.Target)
	if len(skipped) > 0 {
		sort.Strings(skipped)
		b.WriteString("// Not re-exported, because the package declares an identifier of the same name:\n")
		for _, s := range skipped {
			fmt.Fprintf(&b, "//   %s\n", s)
		}
		b.WriteString("\n")
	}
	if len(notCopied) > 0 {
		sort.Strings(notCopied)
		b.WriteString("// Not re-exported, because a copy of a variable of a basic type would not follow\n// the writes to the original; use the package that declares it:\n")
		for _, s := range notCopied {
			fmt.Fprintf(&b, "//   %s\n", s)
		}
		b.WriteString("\n")
	}
	paths := make([]string, 0, len(used))
	for p := range used {
		paths = append(paths, p)
	}
	isStd := func(p string) bool { return !strings.Contains(strings.SplitN(p, "/", 2)[0], ".") }
	sort.Slice(paths, func(i, j int) bool {
		if si, sj := isStd(paths[i]), isStd(paths[j]); si != sj {
			return si
		}
		return paths[i] < paths[j]
	})
	if len(paths) > 0 {
		b.WriteString("import (\n")
		for i, p := range paths {
			if i > 0 && isStd(paths[i-1]) && !isStd(p) {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "\t%q\n", p)
		}
		b.WriteString(")\n\n")
	}
	b.Write(body.Bytes())
	code, err := format.Source(b.Bytes())
	if err != nil {
		return nil, errors.NewErr("ReexportFormatFailed", err)
	}
	return code, nil
}

// mentionsUnexported is whether the type is spelled with a type that another
// package cannot name: an unexported type, or an interface with an unexported
// method.
func mentionsUnexported(t types.Type) bool {
	seen := map[types.Type]bool{}
	var walk func(t types.Type) bool
	walk = func(t types.Type) bool {
		if t == nil || seen[t] {
			return false
		}
		seen[t] = true
		switch t := t.(type) {
		case *types.Named:
			if !t.Obj().Exported() && t.Obj().Pkg() != nil {
				return true
			}
			for i := 0; i < t.TypeArgs().Len(); i++ {
				if walk(t.TypeArgs().At(i)) {
					return true
				}
			}
		case *types.Alias:
			if !t.Obj().Exported() && t.Obj().Pkg() != nil {
				return true
			}
			for i := 0; i < t.TypeArgs().Len(); i++ {
				if walk(t.TypeArgs().At(i)) {
					return true
				}
			}
		case *types.Pointer:
			return walk(t.Elem())
		case *types.Slice:
			return walk(t.Elem())
		case *types.Array:
			return walk(t.Elem())
		case *types.Chan:
			return walk(t.Elem())
		case *types.Map:
			return walk(t.Key()) || walk(t.Elem())
		case *types.Tuple:
			for i := 0; i < t.Len(); i++ {
				if walk(t.At(i).Type()) {
					return true
				}
			}
		case *types.Signature:
			for i := 0; i < t.TypeParams().Len(); i++ {
				if walk(t.TypeParams().At(i).Constraint()) {
					return true
				}
			}
			return walk(t.Params()) || walk(t.Results())
		case *types.Struct:
			for i := 0; i < t.NumFields(); i++ {
				if !t.Field(i).Exported() || walk(t.Field(i).Type()) {
					return true
				}
			}
		case *types.Interface:
			for i := 0; i < t.NumExplicitMethods(); i++ {
				if !t.ExplicitMethod(i).Exported() || walk(t.ExplicitMethod(i).Type()) {
					return true
				}
			}
			for i := 0; i < t.NumEmbeddeds(); i++ {
				if walk(t.EmbeddedType(i)) {
					return true
				}
			}
		case *types.Union:
			for i := 0; i < t.Len(); i++ {
				if walk(t.Term(i).Type()) {
					return true
				}
			}
		case *types.TypeParam:
			return walk(t.Constraint())
		}
		return false
	}
	return walk(t)
}

// wrapper writes the declaration that re-exports the object, or reports that it
// is not re-exported.
func wrapper(obj types.Object, pkg *types.Package, qual types.Qualifier, specPackage string) (string, bool) {
	name := obj.Name()
	pkgName := pkg.Name()
	library := specPackage + ".dart"
	if specPackage == "ui" {
		library = "dart:ui"
	}
	doc := fmt.Sprintf("// %s is %s.%s.\n//\n// Flutter: %s %s\n", name, pkgName, name, library, name)
	switch o := obj.(type) {
	case *types.TypeName:
		tparams := o.Type()
		var params, args []string
		switch t := tparams.(type) {
		case *types.Named:
			params, args = typeParamLists(t.TypeParams(), qual)
		case *types.Alias:
			params, args = typeParamLists(t.TypeParams(), qual)
		}
		decl := name
		ref := pkgName + "." + name
		if len(params) > 0 {
			decl += "[" + strings.Join(params, ", ") + "]"
			ref += "[" + strings.Join(args, ", ") + "]"
		}
		return doc + "type " + decl + " = " + ref + "\n", true
	case *types.Func:
		sig := o.Type().(*types.Signature)
		if sig.Recv() != nil {
			return "", false
		}
		return doc + funcWrapper(name, sig, pkgName, qual), true
	case *types.Var:
		return doc + "var " + name + " = " + pkgName + "." + name + "\n", true
	case *types.Const:
		return doc + "const " + name + " = " + pkgName + "." + name + "\n", true
	}
	return "", false
}

func typeParamLists(tps *types.TypeParamList, qual types.Qualifier) (params, args []string) {
	for i := 0; i < tps.Len(); i++ {
		tp := tps.At(i)
		params = append(params, tp.Obj().Name()+" "+types.TypeString(tp.Constraint(), qual))
		args = append(args, tp.Obj().Name())
	}
	return params, args
}

// funcWrapper writes a function that calls the function of the package with its
// parameters.
func funcWrapper(name string, sig *types.Signature, pkgName string, qual types.Qualifier) string {
	// The names of the parameters must not hide a package that the wrapper names.
	hidden := map[string]bool{pkgName: true}
	collect := func(t types.Type) {
		types.TypeString(t, func(p *types.Package) string { hidden[p.Name()] = true; return p.Name() })
	}
	for i := 0; i < sig.Params().Len(); i++ {
		collect(sig.Params().At(i).Type())
	}
	for i := 0; i < sig.Results().Len(); i++ {
		collect(sig.Results().At(i).Type())
	}
	var tparams, targs []string
	tparams, targs = typeParamLists(sig.TypeParams(), qual)
	decl := name
	call := pkgName + "." + name
	if len(tparams) > 0 {
		decl += "[" + strings.Join(tparams, ", ") + "]"
		call += "[" + strings.Join(targs, ", ") + "]"
	}
	var params, args []string
	for i := 0; i < sig.Params().Len(); i++ {
		v := sig.Params().At(i)
		pn := v.Name()
		if pn == "" || pn == "_" || hidden[pn] || token.Lookup(pn).IsKeyword() {
			pn = fmt.Sprintf("a%d", i)
		}
		t := v.Type()
		if sig.Variadic() && i == sig.Params().Len()-1 {
			slice := t.(*types.Slice)
			params = append(params, pn+" ..."+types.TypeString(slice.Elem(), qual))
			args = append(args, pn+"...")
			continue
		}
		params = append(params, pn+" "+types.TypeString(t, qual))
		args = append(args, pn)
	}
	var results []string
	for i := 0; i < sig.Results().Len(); i++ {
		results = append(results, types.TypeString(sig.Results().At(i).Type(), qual))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "func %s(%s)", decl, strings.Join(params, ", "))
	switch len(results) {
	case 0:
	case 1:
		b.WriteString(" " + results[0])
	default:
		b.WriteString(" (" + strings.Join(results, ", ") + ")")
	}
	b.WriteString(" {\n\t")
	if len(results) > 0 {
		b.WriteString("return ")
	}
	fmt.Fprintf(&b, "%s(%s)\n}\n", call, strings.Join(args, ", "))
	return b.String()
}

// declaredNames are the exported identifiers that the Go files of the directory
// declare at package level, except the generated ones and the tests, and the
// name of the package, which is the one of its Go files.
func declaredNames(dir string) (names map[string]bool, pkgName string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", errors.NewErr("ReexportReadDirFailed", err).SetMetadata("dir", dir)
	}
	names = map[string]bool{}
	fset := token.NewFileSet()
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") || strings.HasSuffix(n, "_gen.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, n), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, "", errors.NewErr("ReexportParseFailed", err).SetMetadata("file", filepath.Join(dir, n))
		}
		pkgName = f.Name.Name
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.IsExported() {
					names[d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							names[s.Name.Name] = true
						}
					case *ast.ValueSpec:
						for _, id := range s.Names {
							if id.IsExported() {
								names[id.Name] = true
							}
						}
					}
				}
			}
		}
	}
	if pkgName == "" {
		return nil, "", errors.New("ReexportTargetEmpty", "the target directory has no Go file that gives the name of the package", "dir", dir)
	}
	return names, pkgName, nil
}

// Run executes `glutter-gen reexport [-module path] [-target dir] [-out file]`
// with the arguments that follow the subcommand name, in the root directory of
// the module (the go command resolves the packages from the working directory).
// It type-checks the packages that Flutter's material.dart exports, and writes
// the type aliases and the wrapper functions that give package material the same
// identifiers.
//
// Flutter: material.dart
func Run(args []string) error {
	flags := flag.NewFlagSet("reexport", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	module := flags.String("module", defaultModule, "the path of the module")
	target := flags.String("target", defaultTarget, "the directory of the package that gets the wrappers")
	out := flags.String("out", defaultOut, "file to write the wrappers to")
	if err := flags.Parse(args); err != nil {
		return errors.NewErr("ReexportBadFlags", err).SetMetadata("args", args)
	}
	declared, pkgName, err := declaredNames(*target)
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return errors.NewErr("ReexportGetwdFailed", err)
	}
	fset := token.NewFileSet()
	// The importer of the sources is documented to be an ImporterFrom.
	imp := importer.ForCompiler(fset, "source", nil).(types.ImporterFrom)
	specs := DefaultSpecs()
	pkgs := map[string]*types.Package{}
	for _, s := range specs {
		p, err := imp.ImportFrom(*module+"/"+s.Package, wd, 0)
		if err != nil {
			return errors.NewErr("ReexportImportFailed", err).SetMetadata("package", s.Package)
		}
		pkgs[s.Package] = p
	}
	code, err := Generate(Options{Module: *module, Target: pkgName, Specs: specs, Packages: pkgs, Declared: declared})
	if err != nil {
		return err
	}
	if old, err := os.ReadFile(*out); err == nil && bytes.Equal(old, code) {
		return nil
	}
	if err := os.WriteFile(*out, code, 0o644); err != nil {
		return errors.NewErr("ReexportWriteFailed", err).SetMetadata("file", *out)
	}
	return nil
}
