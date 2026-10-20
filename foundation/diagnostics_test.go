//go:build !glutter_release && !glutter_profile

// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package foundation

import (
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// Ported from packages/flutter/test/foundation/diagnostics_test.dart.

func TestTransitionTree(t *testing.T) {
	// Multiple styles integrating together in the same tree due to using
	// transition to go between styles that would otherwise be incompatible.
	tree := &TestTree{
		style:      DiagnosticsTreeStyleSparse,
		properties: []DiagnosticsNode{NewStringProperty("stringProperty1", Some("value1"))},
		children: []*TestTree{
			{
				style: DiagnosticsTreeStyleTransition,
				name:  "node transition",
				properties: []DiagnosticsNode{
					NewStringProperty("p1", Some("v1")),
					ToDiagnosticsNode(&TestTree{
						properties: []DiagnosticsNode{NewDiagnosticsProperty("survived", true)},
					}, DiagnosticsNodeOpts{Name: Some("tree property"), Style: DiagnosticsTreeStyleWhitespace}),
				},
				children: []*TestTree{
					{name: "dense child", style: DiagnosticsTreeStyleDense},
					{name: "dense", properties: []DiagnosticsNode{NewStringProperty("property1", Some("value1"))}, style: DiagnosticsTreeStyleDense},
					{
						name: "node B3",
						properties: []DiagnosticsNode{
							NewStringProperty("node_type", Some("<leaf node>"), StringPropertyOpts{ShowName: Some(false), Unquoted: true}),
							NewIntProperty("foo", Some(42)),
						},
						style: DiagnosticsTreeStyleDense,
					},
				},
			},
			{
				name:       "node C",
				properties: []DiagnosticsNode{NewStringProperty("foo", Some("multi\nline\nvalue!"), StringPropertyOpts{Unquoted: true})},
				style:      DiagnosticsTreeStyleSparse,
			},
		},
	}
	want := "TestTree#00000\n" +
		" │ stringProperty1: \"value1\"\n" +
		" ╞═╦══ child node transition ═══\n" +
		" │ ║ TestTree#00000:\n" +
		" │ ║   p1: \"v1\"\n" +
		" │ ║   tree property: TestTree#00000:\n" +
		" │ ║     survived: true\n" +
		" │ ║   ├child dense child: TestTree#00000\n" +
		" │ ║   ├child dense: TestTree#00000(property1: \"value1\")\n" +
		" │ ║   └child node B3: TestTree#00000(<leaf node>, foo: 42)\n" +
		" │ ╚═══════════\n" +
		" └─child node C: TestTree#00000\n" +
		"     foo:\n" +
		"       multi\n" +
		"       line\n" +
		"       value!\n"
	if got := ignoringHashCodes(ToStringDeep(tree)); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDiagnosticableString(t *testing.T) {
	tree := &TestTree{
		properties: []DiagnosticsNode{
			NewStringProperty("stringProperty1", Some("value1"), StringPropertyOpts{Unquoted: true}),
			NewDoubleProperty("doubleProperty1", Some(42.5)),
			NewDoubleProperty("roundedProperty", Some(1.0/3.0)),
			NewStringProperty("DO_NOT_SHOW", Some("DO_NOT_SHOW"), StringPropertyOpts{Level: DiagnosticLevelHidden, Unquoted: true}),
			NewStringProperty("DEBUG_ONLY", Some("DEBUG_ONLY"), StringPropertyOpts{Level: DiagnosticLevelDebug, Unquoted: true}),
		},
		// A child to verify that children are not included.
		children: []*TestTree{{name: "node A"}},
	}
	tests := []struct {
		name     string
		minLevel []DiagnosticLevel
		want     string
	}{
		{"info", nil, "TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3)"},
		{"debug", []DiagnosticLevel{DiagnosticLevelDebug}, "TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, DEBUG_ONLY: DEBUG_ONLY)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ignoringHashCodes(DiagnosticableString(tree, tt.minLevel...)); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
	shallow := ignoringHashCodes(ToStringShallow(tree))
	if want := "TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3), stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, DEBUG_ONLY: DEBUG_ONLY"; shallow != want {
		t.Errorf("ToStringShallow = %q, want %q", shallow, want)
	}
}

// property is a property under test with what it must report.
type propertyCase struct {
	name         string
	p            *DiagnosticsProperty
	str          string
	level        DiagnosticLevel
	filteredInfo bool
}

func TestPropertyStrings(t *testing.T) {
	rect := ui.NewRectFromLTRB(0, 0, 20, 20)
	rectStr := "Rect.fromLTRB(0.0, 0.0, 20.0, 20.0)"
	tests := []propertyCase{
		{"string unquoted", NewStringProperty("name", Some("value"), StringPropertyOpts{Unquoted: true}), "name: value", DiagnosticLevelInfo, false},
		{"string description", NewStringProperty("name", Some("value"), StringPropertyOpts{Description: "VALUE", IfEmpty: "<hidden>", Unquoted: true}), "name: VALUE", DiagnosticLevelInfo, false},
		{"string no name", NewStringProperty("name", Some("value"), StringPropertyOpts{ShowName: Some(false), IfEmpty: "<hidden>", Unquoted: true}), "value", DiagnosticLevelInfo, false},
		{"string if empty", NewStringProperty("name", Some(""), StringPropertyOpts{IfEmpty: "<hidden>"}), "name: <hidden>", DiagnosticLevelInfo, false},
		{"string if empty no name", NewStringProperty("name", Some(""), StringPropertyOpts{IfEmpty: "<hidden>", ShowName: Some(false)}), "<hidden>", DiagnosticLevelInfo, false},
		{"string null", NewStringProperty("name", None[string]()), "name: null", DiagnosticLevelInfo, false},
		{"string hidden", NewStringProperty("name", Some("value"), StringPropertyOpts{Level: DiagnosticLevelHidden}), `name: "value"`, DiagnosticLevelHidden, true},
		{"string null default null", NewStringProperty("name", None[string](), StringPropertyOpts{DefaultValue: DefaultNull}), "name: null", DiagnosticLevelFine, true},
		{"string quoted", NewStringProperty("name", Some("value")), `name: "value"`, DiagnosticLevelInfo, false},
		{"string quoted no name", NewStringProperty("name", Some("value"), StringPropertyOpts{ShowName: Some(false)}), `"value"`, DiagnosticLevelInfo, false},
		{"string null no name", NewStringProperty("name", None[string](), StringPropertyOpts{ShowName: Some(false)}), "null", DiagnosticLevelInfo, false},

		{"bool true", NewDiagnosticsProperty("name", true), "name: true", DiagnosticLevelInfo, false},
		{"bool false", NewDiagnosticsProperty("name", false), "name: false", DiagnosticLevelInfo, false},
		{"bool description", NewDiagnosticsProperty("name", true, DiagnosticsPropertyOpts{Description: "truthy"}), "name: truthy", DiagnosticLevelInfo, false},
		{"bool no name", NewDiagnosticsProperty("name", true, DiagnosticsPropertyOpts{ShowName: Some(false)}), "true", DiagnosticLevelInfo, false},
		{"bool null", NewDiagnosticsPropertyOpt("name", None[bool]()), "name: null", DiagnosticLevelInfo, false},
		{"bool hidden", NewDiagnosticsProperty("name", true, DiagnosticsPropertyOpts{Level: DiagnosticLevelHidden}), "name: true", DiagnosticLevelHidden, true},
		{"bool null default null", NewDiagnosticsPropertyOpt("name", None[bool](), DiagnosticsPropertyOpts{DefaultValue: DefaultNull}), "name: null", DiagnosticLevelFine, true},
		{"bool missing", NewDiagnosticsPropertyOpt("name", None[bool](), DiagnosticsPropertyOpts{IfNull: "missing"}), "name: missing", DiagnosticLevelInfo, false},
		{"tooltip", NewDiagnosticsProperty("name", "value", DiagnosticsPropertyOpts{Tooltip: "tooltip"}), "name: value (tooltip)", DiagnosticLevelInfo, false},

		{"double", NewDoubleProperty("name", Some(42.0)), "name: 42.0", DiagnosticLevelInfo, false},
		{"double rounds", NewDoubleProperty("name", Some(1.3333)), "name: 1.3", DiagnosticLevelInfo, false},
		{"double null", NewDoubleProperty("name", None[float64]()), "name: null", DiagnosticLevelInfo, false},
		{"double if null", NewDoubleProperty("name", None[float64](), NumberPropertyOpts{IfNull: "missing"}), "name: missing", DiagnosticLevelInfo, false},
		{"double unit", NewDoubleProperty("name", Some(42.0), NumberPropertyOpts{Unit: "px"}), "name: 42.0px", DiagnosticLevelInfo, false},
		{"double infinity", NewDoubleProperty("double1", Some(posInf)), "double1: Infinity", DiagnosticLevelInfo, false},
		{"double -infinity", NewDoubleProperty("double2", Some(-posInf)), "double2: -Infinity", DiagnosticLevelInfo, false},
		{"lazy double", NewDoublePropertyLazy("name", func() Opt[float64] { return Some(42.0) }), "name: 42.0", DiagnosticLevelInfo, false},
		{"lazy double rounds", NewDoublePropertyLazy("name", func() Opt[float64] { return Some(1.3333) }), "name: 1.3", DiagnosticLevelInfo, false},
		{"lazy double null", NewDoublePropertyLazy("name", func() Opt[float64] { return None[float64]() }), "name: null", DiagnosticLevelInfo, false},
		{"lazy double throws", NewDoublePropertyLazy("name", func() Opt[float64] { panic("Invalid constraints") }), "name: EXCEPTION (string)", DiagnosticLevelError, false},

		{"percent", NewPercentProperty("name", Some(0.4)), "name: 40.0%", DiagnosticLevelInfo, false},
		{"percent unit tooltip", NewPercentProperty("name", Some(0.99), NumberPropertyOpts{Unit: "invisible", Tooltip: "almost transparent"}), "name: 99.0% invisible (almost transparent)", DiagnosticLevelInfo, false},
		{"percent null tooltip", NewPercentProperty("name", None[float64](), NumberPropertyOpts{Unit: "invisible", Tooltip: "!"}), "name: null (!)", DiagnosticLevelInfo, false},
		{"percent zero", NewPercentProperty("name", Some(0.0)), "name: 0.0%", DiagnosticLevelInfo, false},
		{"percent negative", NewPercentProperty("name", Some(-10.0)), "name: 0.0%", DiagnosticLevelInfo, false},
		{"percent one", NewPercentProperty("name", Some(1.0)), "name: 100.0%", DiagnosticLevelInfo, false},
		{"percent over", NewPercentProperty("name", Some(3.0)), "name: 100.0%", DiagnosticLevelInfo, false},
		{"percent null", NewPercentProperty("name", None[float64]()), "name: null", DiagnosticLevelInfo, false},
		{"percent if null", NewPercentProperty("name", None[float64](), NumberPropertyOpts{IfNull: "missing"}), "name: missing", DiagnosticLevelInfo, false},
		{"percent if null no name", NewPercentProperty("name", None[float64](), NumberPropertyOpts{IfNull: "missing", ShowName: Some(false)}), "missing", DiagnosticLevelInfo, false},
		{"percent no name", NewPercentProperty("name", Some(0.5), NumberPropertyOpts{ShowName: Some(false)}), "50.0%", DiagnosticLevelInfo, false},

		{"flag on", NewFlagProperty("myFlag", Some(true), FlagPropertyOpts{IfTrue: "myFlag"}), "myFlag", DiagnosticLevelInfo, false},
		{"flag off with ifTrue only", NewFlagProperty("myFlag", Some(false), FlagPropertyOpts{IfTrue: "myFlag"}), "myFlag: false", DiagnosticLevelHidden, true},
		{"flag on with ifFalse only", NewFlagProperty("wasLayout", Some(true), FlagPropertyOpts{IfFalse: "no layout computed"}), "wasLayout: true", DiagnosticLevelHidden, true},
		{"flag layout computed", NewFlagProperty("wasLayout", Some(true), FlagPropertyOpts{IfTrue: "layout computed"}), "layout computed", DiagnosticLevelInfo, false},
		{"flag yes", NewFlagProperty("name", Some(true), FlagPropertyOpts{IfTrue: "YES", IfFalse: "NO", ShowName: true}), "name: YES", DiagnosticLevelInfo, false},
		{"flag no", NewFlagProperty("name", Some(false), FlagPropertyOpts{IfTrue: "YES", IfFalse: "NO", ShowName: true}), "name: NO", DiagnosticLevelInfo, false},
		{"flag yes no name", NewFlagProperty("name", Some(true), FlagPropertyOpts{IfTrue: "YES", IfFalse: "NO"}), "YES", DiagnosticLevelInfo, false},
		{"flag no no name", NewFlagProperty("name", Some(false), FlagPropertyOpts{IfTrue: "YES", IfFalse: "NO"}), "NO", DiagnosticLevelInfo, false},
		{"flag hidden", NewFlagProperty("name", Some(true), FlagPropertyOpts{IfTrue: "YES", IfFalse: "NO", ShowName: true, Level: DiagnosticLevelHidden}), "name: YES", DiagnosticLevelHidden, true},

		{"enum hello", NewEnumProperty("name", Some(exampleHello)), "name: hello", DiagnosticLevelInfo, false},
		{"enum world", NewEnumProperty("name", Some(exampleWorld)), "name: world", DiagnosticLevelInfo, false},
		{"enum deferToChild", NewEnumProperty("name", Some(exampleDeferToChild)), "name: deferToChild", DiagnosticLevelInfo, false},
		{"enum null", NewEnumProperty("name", None[exampleEnum]()), "name: null", DiagnosticLevelInfo, false},
		{"enum default", NewEnumProperty("name", Some(exampleHello), EnumPropertyOpts{DefaultValue: exampleHello}), "name: hello", DiagnosticLevelFine, true},
		{"enum hidden", NewEnumProperty("name", Some(exampleHello), EnumPropertyOpts{Level: DiagnosticLevelHidden}), "name: hello", DiagnosticLevelHidden, true},

		{"int", NewIntProperty("name", Some(42)), "name: 42", DiagnosticLevelInfo, false},
		{"int null", NewIntProperty("name", None[int]()), "name: null", DiagnosticLevelInfo, false},
		{"int null default null", NewIntProperty("name", None[int](), NumberPropertyOpts{DefaultValue: DefaultNull}), "name: null", DiagnosticLevelFine, true},
		{"int if null", NewIntProperty("name", None[int](), NumberPropertyOpts{IfNull: "missing"}), "name: missing", DiagnosticLevelInfo, false},
		{"int no name", NewIntProperty("name", Some(42), NumberPropertyOpts{ShowName: Some(false)}), "42", DiagnosticLevelInfo, false},
		{"int unit", NewIntProperty("name", Some(42), NumberPropertyOpts{Unit: "pt"}), "name: 42pt", DiagnosticLevelInfo, false},
		{"int default", NewIntProperty("name", Some(42), NumberPropertyOpts{DefaultValue: 42}), "name: 42", DiagnosticLevelFine, true},
		{"int not default", NewIntProperty("name", Some(43), NumberPropertyOpts{DefaultValue: 42}), "name: 43", DiagnosticLevelInfo, false},
		{"int hidden", NewIntProperty("name", Some(42), NumberPropertyOpts{Level: DiagnosticLevelHidden}), "name: 42", DiagnosticLevelHidden, true},

		{"object", NewDiagnosticsProperty("name", rect), "name: " + rectStr, DiagnosticLevelInfo, false},
		{"object description", NewDiagnosticsProperty("name", rect, DiagnosticsPropertyOpts{Description: "small rect"}), "name: small rect", DiagnosticLevelInfo, false},
		{"object nil", NewDiagnosticsProperty[any]("name", nil), "name: null", DiagnosticLevelInfo, false},
		{"object nil default nil", NewDiagnosticsProperty[any]("name", nil, DiagnosticsPropertyOpts{DefaultValue: DefaultNull}), "name: null", DiagnosticLevelFine, true},
		{"object nil if null", NewDiagnosticsProperty[any]("name", nil, DiagnosticsPropertyOpts{IfNull: "missing"}), "name: missing", DiagnosticLevelInfo, false},
		{"object warning no name", NewDiagnosticsProperty("name", rect, DiagnosticsPropertyOpts{ShowName: Some(false), Level: DiagnosticLevelWarning}), rectStr, DiagnosticLevelWarning, false},
		{"object no separator", NewDiagnosticsProperty("Creator", rect, DiagnosticsPropertyOpts{ShowSeparator: Some(false)}), "Creator " + rectStr, DiagnosticLevelInfo, false},
		{"object missing", NewDiagnosticsProperty[any]("name", nil, DiagnosticsPropertyOpts{MissingIfNull: true}), "name: MISSING", DiagnosticLevelWarning, false},

		{"lazy object", NewDiagnosticsPropertyLazy("name", func() ui.Rect { return rect }, DiagnosticsPropertyOpts{Description: "small rect"}), "name: small rect", DiagnosticLevelInfo, false},
		{"lazy nil", NewDiagnosticsPropertyLazy("name", func() any { return nil }, DiagnosticsPropertyOpts{Description: "missing"}), "name: missing", DiagnosticLevelInfo, false},
		{"lazy nil default nil", NewDiagnosticsPropertyLazy("name", func() any { return nil }, DiagnosticsPropertyOpts{Description: "missing", DefaultValue: DefaultNull}), "name: missing", DiagnosticLevelFine, true},
		{"lazy no name", NewDiagnosticsPropertyLazy("name", func() ui.Rect { return rect }, DiagnosticsPropertyOpts{Description: "small rect", ShowName: Some(false)}), "small rect", DiagnosticLevelInfo, false},
		{"lazy throws with description", NewDiagnosticsPropertyLazy("name", func() any { panic("Property not available") }, DiagnosticsPropertyOpts{Description: "missing", DefaultValue: DefaultNull}), "name: missing", DiagnosticLevelError, false},
		{"lazy throws", NewDiagnosticsPropertyLazy("name", func() any { panic("Property not available") }, DiagnosticsPropertyOpts{DefaultValue: DefaultNull}), "name: EXCEPTION (string)", DiagnosticLevelError, false},

		{"object flag present", NewObjectFlagProperty("onClick", &struct{ f int }{}, ObjectFlagPropertyOpts{IfPresent: "clickable"}), "clickable", DiagnosticLevelInfo, false},
		{"object flag missing", NewObjectFlagProperty[*int]("onClick", nil, ObjectFlagPropertyOpts{IfPresent: "clickable"}), "onClick: null", DiagnosticLevelHidden, true},
		{"object flag disabled", NewObjectFlagProperty[*int]("onClick", nil, ObjectFlagPropertyOpts{IfNull: "disabled"}), "disabled", DiagnosticLevelInfo, false},
		{"object flag has", NewObjectFlagPropertyHas("onClick", &struct{ f int }{}), "has onClick", DiagnosticLevelInfo, false},
		{"object flag has missing", NewObjectFlagPropertyHas[*int]("onClick", nil), "onClick: null", DiagnosticLevelHidden, true},

		{"flags", NewFlagsSummary("listeners", []FlagEntry{{"click", true}, {"move", true}}), "listeners: click, move", DiagnosticLevelInfo, false},
		{"flags reversed", NewFlagsSummary("listeners", []FlagEntry{{"move", true}, {"click", true}}), "listeners: move, click", DiagnosticLevelInfo, false},
		{"flags partial", NewFlagsSummary("listeners", []FlagEntry{{"move", false}, {"click", true}}), "listeners: click", DiagnosticLevelInfo, false},
		{"flags empty", NewFlagsSummary("listeners", []FlagEntry{{"enter", false}}), "listeners", DiagnosticLevelHidden, true},
		{"flags empty ifEmpty", NewFlagsSummary("listeners", []FlagEntry{{"enter", false}}, FlagsSummaryOpts{IfEmpty: "<none>"}), "listeners: <none>", DiagnosticLevelInfo, false},

		{"iterable ints", NewIterableProperty("ints", []int{1, 2, 3}), "ints: 1, 2, 3", DiagnosticLevelInfo, false},
		{"iterable doubles", NewIterableProperty("doubles", []float64{1, 2, 3}), "doubles: 1.0, 2.0, 3.0", DiagnosticLevelInfo, false},
		{"iterable empty", NewIterableProperty("name", []any{}), "name: []", DiagnosticLevelInfo, false},
		{"iterable nil", NewIterableProperty[any]("list", nil), "list: null", DiagnosticLevelInfo, false},
		{"iterable nil default nil", NewIterableProperty[any]("list", nil, IterablePropertyOpts{DefaultValue: DefaultNull}), "list: null", DiagnosticLevelFine, true},
		{"iterable objects", NewIterableProperty("objects", []any{rect, ui.NewColorFromARGB(255, 255, 255, 255)}), "objects: " + rectStr + ", " + ui.NewColorFromARGB(255, 255, 255, 255).String(), DiagnosticLevelInfo, false},
		{"iterable multi line", NewIterableProperty("objects", []any{rect, ui.NewColorFromARGB(255, 255, 255, 255)}, IterablePropertyOpts{Style: DiagnosticsTreeStyleWhitespace}), "objects:\n" + rectStr + "\n" + ui.NewColorFromARGB(255, 255, 255, 255).String(), DiagnosticLevelInfo, false},
		{"iterable no ifEmpty", NewIterableProperty("name", []int{}, IterablePropertyOpts{NoIfEmpty: true}), "name: []", DiagnosticLevelFine, true},

		{"message", MustNode(NewDiagnosticsNodeMessage("hello world")), "hello world", DiagnosticLevelInfo, false},
		{"message property", NewMessageProperty("diagnostics", "hello world"), "diagnostics: hello world", DiagnosticLevelInfo, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.String()
			if got != tt.str {
				t.Errorf("String() = %q, want %q", got, tt.str)
			}
			if got := tt.p.Level(); got != tt.level {
				t.Errorf("Level() = %v, want %v", got, tt.level)
			}
			if got := tt.p.IsFiltered(DiagnosticLevelInfo); got != tt.filteredInfo {
				t.Errorf("IsFiltered(info) = %v, want %v", got, tt.filteredInfo)
			}
		})
	}
}

var posInf = func() float64 { x := 1e308; return x * 10 }()

// MustNode returns n; it turns the result of a constructor into the type of
// the table.
func MustNode(n *DiagnosticsProperty) *DiagnosticsProperty { return n }

func TestPropertyValues(t *testing.T) {
	rect := ui.NewRectFromLTRB(0, 0, 20, 20)
	if v := NewDiagnosticsProperty("name", true).Value(); v != true {
		t.Errorf("bool value = %v", v)
	}
	if v := NewDoubleProperty("name", Some(42.0)).Value(); v != 42.0 {
		t.Errorf("double value = %v", v)
	}
	if v := NewIntProperty("name", None[int]()).Value(); v != nil {
		t.Errorf("null int value = %v", v)
	}
	if v := NewDiagnosticsProperty("name", rect).Value(); v != rect {
		t.Errorf("rect value = %v", v)
	}
	throwing := NewDoublePropertyLazy("name", func() Opt[float64] { panic("boom") })
	if throwing.Value() != nil || throwing.Exception() != "boom" {
		t.Errorf("throwing property: value %v, exception %v", throwing.Value(), throwing.Exception())
	}
	if got := NewDiagnosticsProperty("name", rect).PropertyType(); got != "Rect" {
		t.Errorf("PropertyType = %q, want Rect", got)
	}
	if got := NewIntProperty("p", Some(1)).Name().Value(); got != "p" {
		t.Errorf("Name = %q", got)
	}
}

func TestMessageNode(t *testing.T) {
	message := NewDiagnosticsNodeMessage("hello world")
	if got := message.Name(); got != Some("") {
		t.Errorf("message name = %v, want empty", got)
	}
	if message.Value() != nil || message.ShowName() {
		t.Errorf("message value %v showName %v", message.Value(), message.ShowName())
	}
	property := NewMessageProperty("diagnostics", "hello world")
	if property.Value() != nil || !property.ShowName() || property.Name().Value() != "diagnostics" {
		t.Errorf("message property: value %v showName %v name %v", property.Value(), property.ShowName(), property.Name())
	}
}

func TestExpandableValue(t *testing.T) {
	inner := &TestTree{properties: []DiagnosticsNode{NewIntProperty("foo", Some(1))}, children: []*TestTree{{name: "kid"}}}
	p := NewDiagnosticsProperty[Diagnosticable]("inner", inner, DiagnosticsPropertyOpts{ExpandableValue: true, Style: DiagnosticsTreeStyleSparse})
	if n := len(p.GetProperties()); n != 1 {
		t.Errorf("expanded properties = %d, want 1", n)
	}
	if n := len(p.GetChildren()); n != 1 {
		t.Errorf("expanded children = %d, want 1", n)
	}
	plain := NewDiagnosticsProperty[Diagnosticable]("inner", inner)
	if len(plain.GetProperties()) != 0 || len(plain.GetChildren()) != 0 {
		t.Error("a property that is not expandable has no properties or children")
	}
	// A tree value is shown with its short description, not its whole text.
	if got := ignoringHashCodes(plain.String()); got != "inner: TestTree#00000" {
		t.Errorf("String() = %q", got)
	}
}

func TestDiagnosticsBlock(t *testing.T) {
	block := NewDiagnosticsBlock(DiagnosticsBlockOpts{
		Name:        Some("block"),
		Description: "a block",
		Properties:  []DiagnosticsNode{NewIntProperty("n", Some(1))},
		Children:    []DiagnosticsNode{NewDiagnosticsNodeMessage("child message")},
	})
	want := "block: a block:\n  n: 1\n  child message\n"
	if got := block.ToStringDeep(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if NewDiagnosticsBlock(DiagnosticsBlockOpts{Description: "x"}).ShowName() {
		t.Error("a block without a name does not show its name")
	}
}

func TestTruncateChildren(t *testing.T) {
	var kids []*TestTree
	for i := 0; i < 3; i++ {
		kids = append(kids, &TestTree{name: string(rune('a' + i)), style: DiagnosticsTreeStyleSingleLine})
	}
	tree := &TestTree{style: DiagnosticsTreeStyleTruncateChildren, children: kids}
	got := ignoringHashCodes(ToDiagnosticsNode(tree, DiagnosticsNodeOpts{Name: Some("root")}).ToStringDeep())
	want := "This root had the following descendants (showing up to depth 5):\n" +
		"    child a: TestTree#00000\n    child b: TestTree#00000\n    child c: TestTree#00000"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWordWrap(t *testing.T) {
	long := strings.TrimSpace(strings.Repeat("word ", 30))
	deep := func(msg *DiagnosticsProperty) string {
		// A message wraps as a property of a node that puts its properties on
		// lines of their own.
		tree := &TestTree{properties: []DiagnosticsNode{msg}}
		return ToStringDeep(tree, ToStringDeepOpts{WrapWidth: 40})
	}
	got := deep(NewDiagnosticsNodeMessage(long))
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("the message did not wrap: %q", got)
	}
	var words []string
	for _, line := range lines[1:] {
		if strLen(line) > 68 {
			t.Errorf("line %q is longer than the property wrap width", line)
		}
		words = append(words, strings.Fields(line)...)
	}
	if strings.Join(words, " ") != long {
		t.Errorf("wrapping changed the text: %q", got)
	}
	noWrap := deep(NewDiagnosticsNodeMessage(long, DiagnosticsNodeMessageOpts{NoWrap: true}))
	if n := strings.Count(strings.TrimRight(noWrap, "\n"), "\n"); n != 1 {
		t.Errorf("a message that may not wrap took %d lines, want 2", n+1)
	}
}

func TestMinLevelFiltersProperties(t *testing.T) {
	tree := &TestTree{properties: []DiagnosticsNode{
		NewIntProperty("shown", Some(1)),
		NewIntProperty("fine", Some(2), NumberPropertyOpts{Level: DiagnosticLevelFine}),
		NewIntProperty("hidden", Some(3), NumberPropertyOpts{Level: DiagnosticLevelHidden}),
	}}
	tests := []struct {
		minLevel DiagnosticLevel
		want     []string
	}{
		{DiagnosticLevelDebug, []string{"shown"}},
		{DiagnosticLevelFine, []string{"shown", "fine"}},
		{DiagnosticLevelHidden, []string{"shown", "fine", "hidden"}},
		{DiagnosticLevelOff, nil},
	}
	for _, tt := range tests {
		t.Run(tt.minLevel.Name(), func(t *testing.T) {
			got := ToStringDeep(tree, ToStringDeepOpts{MinLevel: tt.minLevel})
			for _, name := range []string{"shown", "fine", "hidden"} {
				want := false
				for _, w := range tt.want {
					want = want || w == name
				}
				if has := strings.Contains(got, name+":"); has != want {
					t.Errorf("min level %v: %q present = %v, want %v\n%s", tt.minLevel, name, has, want, got)
				}
			}
		})
	}
}

func TestDescribeIdentity(t *testing.T) {
	n := NewValueNotifier(5)
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"pointer", ignoringHashCodes(DescribeIdentity(n)), "ValueNotifier<int>#00000"},
		{"nil", DescribeIdentity(nil), "Null#00000"},
		{"plain struct pointer", ignoringHashCodes(DescribeIdentity(&TestTree{})), "TestTree#00000"},
		{"qualified type argument", ignoringHashCodes(DescribeIdentity(NewValueNotifier(ui.Offset{}))), "ValueNotifier<Offset>#00000"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.want)
		}
	}
	if a, b := ShortHash(n), ShortHash(n); a != b || len(a) != 5 {
		t.Errorf("ShortHash is not stable or not 5 digits: %q %q", a, b)
	}
	if ShortHash(n) == ShortHash(NewValueNotifier(5)) {
		t.Log("two notifiers share a short hash (possible, 1 in a million)")
	}
}

func TestDiagnosticsErrors(t *testing.T) {
	tests := []struct {
		name  string
		fn    func()
		route string
	}{
		{"name ends with colon", func() { NewDiagnosticsProperty("name:", 1) }, "InitDiagnosticsNodeBase.DiagnosticsNameEndsWithColon"},
		{"flag without description", func() { NewFlagProperty("f", Some(true)) }, "NewFlagProperty.FlagPropertyNoDescription"},
		{"object flag without description", func() { NewObjectFlagProperty("f", 1) }, "NewObjectFlagProperty[...].ObjectFlagPropertyNoDescription"},
		{"two options", func() { NewDiagnosticsProperty("n", 1, DiagnosticsPropertyOpts{}, DiagnosticsPropertyOpts{}) }, "firstOpts[...]->First[...].MultipleOptions"},
		{"unspecified style", func() {
			n := NewDiagnosticsBlock(DiagnosticsBlockOpts{Description: "d"})
			n.style = DiagnosticsTreeStyleUnspecified
			n.TextTreeConfiguration()
		}, "DiagnosticsNodeBase.TextTreeConfiguration.DiagnosticsStyleUnspecified"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := recoverError(t, tt.fn)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T", err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}

func TestEnumNames(t *testing.T) {
	if got := DiagnosticLevelInfo.String(); got != "DiagnosticLevel.info" {
		t.Errorf("got %q", got)
	}
	if got := DiagnosticsTreeStyleSingleLine.Name(); got != "singleLine" {
		t.Errorf("got %q", got)
	}
	if got := DiagnosticLevel(99).Name(); got != "DiagnosticLevel(99)" {
		t.Errorf("got %q", got)
	}
	if got := DiagnosticsTreeStyle(-1).String(); got != "DiagnosticsTreeStyle(-1)" {
		t.Errorf("got %q", got)
	}
}

func TestValuesEqual(t *testing.T) {
	f := func() {}
	s := []int{1}
	tests := []struct {
		name string
		a, b any
		want bool
	}{
		{"equal ints", 3, 3, true},
		{"different ints", 3, 4, false},
		{"different types", 3, 3.0, false},
		{"both nil", nil, nil, true},
		{"one nil", nil, 3, false},
		{"same function", f, f, true},
		{"another function", f, func() {}, false},
		{"same slice", s, s, true},
		{"equal slices with other backing", []int{1}, []int{1}, false},
		{"comparable struct", ui.Offset{Dx: 1}, ui.Offset{Dx: 1}, true},
		{"strings", "a", "a", true},
	}
	for _, tt := range tests {
		if got := ValuesEqual(tt.a, tt.b); got != tt.want {
			t.Errorf("%s: ValuesEqual = %v, want %v", tt.name, got, tt.want)
		}
	}
}
