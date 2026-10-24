package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// widgetBases are the types of package widgets that a widget embeds, besides the
// ones whose name ends in "Widget".
var widgetBases = map[string]bool{"InheritedTheme": true, "InheritedNotifier": true, "InheritedModel": true}

// toolbarReason is why the toolbars of the text selection are not on a page.
const toolbarReason = "made by the context menu of the text fields, which opens on a right click on the Text fields page (a long press or a tap on the handle on Android)"

// spellCheckReason is why the toolbar of the spell check suggestions is not on a page.
const spellCheckReason = "made by a text field whose spell check is on when a misspelled word is tapped (Android only: it needs the platform's spell checker, see examples/dev/android-spell)"

// magnifierReason is why the magnifiers have no page of their own.
const magnifierReason = "made by the text fields while a selection handle or the cursor is dragged on Android, on the Text fields page"

// notShownOnAPage are the widget constructors of package material that the
// gallery does not call by name, each with the reason.
var notShownOnAPage = map[string]string{
	"NewSpellCheckSuggestionsToolbar":                 spellCheckReason,
	"NewSpellCheckSuggestionsToolbarEditableText":     spellCheckReason,
	"NewAdaptiveTextSelectionToolbar":                 toolbarReason,
	"NewAdaptiveTextSelectionToolbarButtonItems":      toolbarReason,
	"NewAdaptiveTextSelectionToolbarEditable":         toolbarReason,
	"NewAdaptiveTextSelectionToolbarEditableText":     toolbarReason,
	"NewAdaptiveTextSelectionToolbarSelectable":       toolbarReason,
	"NewAdaptiveTextSelectionToolbarSelectableRegion": toolbarReason,
	"NewTextSelectionToolbar":                         toolbarReason,
	"NewTextSelectionToolbarTextButton":               toolbarReason,
	"NewMagnifier":                                    magnifierReason,
	"NewTextMagnifier":                                magnifierReason,
	"NewDesktopTextSelectionToolbar":                  toolbarReason,
	"NewDesktopTextSelectionToolbarButton":            toolbarReason,
	"NewDesktopTextSelectionToolbarButtonText":        toolbarReason,
	"NewDatePickerDialog":                             "opened by ShowDatePicker, which the Dialogs & pickers page calls",
	"NewDateRangePickerDialog":                        "opened by ShowDateRangePicker, which the Dialogs & pickers page calls",
	"NewTimePickerDialog":                             "opened by ShowTimePicker, which the Dialogs & pickers page calls",
	"NewButtonStyleButtonBase":                        "the base that TextButton, ElevatedButton and the other buttons embed",
	"NewDrawerController":                             "made by the Scaffold for its drawers",
	"NewFlexibleSpaceBarSettings":                     "provided by the SliverAppBar to its FlexibleSpaceBar",
	"NewMaterialAppRouter":                            "the gallery switches pages itself, so it uses MaterialApp and not MaterialApp.router",
	"NewMenuAcceleratorCallbackBinding":               "made by the MenuBar and the MenuAnchor for their accelerators",
	"NewNavigationIndicator":                          "made by the NavigationBar and the NavigationRail for the selected destination",
	"NewScaffoldMessenger":                            "the MaterialApp makes one, and the app bar page shows snack bars through it",
}

// parseDir parses the non-test Go files of a directory of the repository.
func parseDir(t *testing.T, dir string, keep func(name string) bool) map[string]*ast.File {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]*ast.File{}
	fset := token.NewFileSet()
	for _, path := range paths {
		name := filepath.Base(path)
		if strings.HasSuffix(name, "_test.go") || !keep(name) {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = f
	}
	return files
}

// materialWidgetConstructors are the exported NewX functions written by hand in
// package material (not the generated re-exports of package widgets) whose result
// is a widget: a struct that embeds a widget base, or a struct that embeds one.
func materialWidgetConstructors(t *testing.T) []string {
	t.Helper()
	files := parseDir(t, "../../material", func(name string) bool { return !strings.HasSuffix(name, "_gen.go") })
	structs := map[string]*ast.StructType{}
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if st, ok := ts.Type.(*ast.StructType); ok {
						structs[ts.Name.Name] = st
					}
				}
			}
		}
	}
	var isWidget func(name string, seen map[string]bool) bool
	isWidget = func(name string, seen map[string]bool) bool {
		st, ok := structs[name]
		if !ok || seen[name] {
			return false
		}
		seen[name] = true
		for _, field := range st.Fields.List {
			if len(field.Names) != 0 {
				continue
			}
			typ := field.Type
			if star, ok := typ.(*ast.StarExpr); ok {
				typ = star.X
			}
			switch e := typ.(type) {
			case *ast.SelectorExpr:
				if strings.HasSuffix(e.Sel.Name, "Widget") || widgetBases[e.Sel.Name] {
					return true
				}
			case *ast.Ident:
				if isWidget(e.Name, seen) {
					return true
				}
			}
		}
		return false
	}
	var names []string
	for _, f := range files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "New") || !fn.Name.IsExported() {
				continue
			}
			if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
				continue
			}
			result := fn.Type.Results.List[0].Type
			if star, ok := result.(*ast.StarExpr); ok {
				result = star.X
			}
			if idx, ok := result.(*ast.IndexExpr); ok {
				result = idx.X
			}
			if id, ok := result.(*ast.Ident); ok && isWidget(id.Name, map[string]bool{}) {
				names = append(names, fn.Name.Name)
			}
		}
	}
	sort.Strings(names)
	return names
}

// galleryUses are the names that the gallery's pages take from package material,
// as in m.NewCard.
func galleryUses(t *testing.T) map[string]bool {
	t.Helper()
	files := parseDir(t, ".", func(string) bool { return true })
	used := map[string]bool{}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "m" {
					used[sel.Sel.Name] = true
				}
			}
			return true
		})
	}
	return used
}

// TestEveryMaterialWidgetIsOnAPage fails when a widget constructor of package
// material is not called by the gallery and has no reason in notShownOnAPage.
func TestEveryMaterialWidgetIsOnAPage(t *testing.T) {
	used := galleryUses(t)
	constructors := materialWidgetConstructors(t)
	if len(constructors) < 100 {
		t.Fatalf("found only %d widget constructors: the scan is broken", len(constructors))
	}
	var missing []string
	for _, name := range constructors {
		if used[name] {
			if reason, ok := notShownOnAPage[name]; ok {
				t.Errorf("%s is on a page, so it has no reason to be in notShownOnAPage (%s)", name, reason)
			}
			continue
		}
		if _, ok := notShownOnAPage[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d widget constructors are on no page of the gallery:\n%s", len(missing), strings.Join(missing, "\n"))
	}
	for name := range notShownOnAPage {
		found := false
		for _, c := range constructors {
			found = found || c == name
		}
		if !found {
			t.Errorf("notShownOnAPage has %s, which is not a widget constructor of package material", name)
		}
	}
}
