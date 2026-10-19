//go:build !glutter_release && !glutter_profile

package foundation

import "testing"

// Ported from packages/flutter/test/foundation/diagnostics_test.dart.

// styleCase is one call of goldenStyleTest.
type styleCase struct {
	desc           string
	name           string
	style          DiagnosticsTreeStyle
	lastChildStyle DiagnosticsTreeStyle
	propertyStyle  DiagnosticsTreeStyle
	golden         string
}

func controlTree(style, lastChildStyle DiagnosticsTreeStyle) *TestTree {
	last := lastChildStyle
	if last == DiagnosticsTreeStyleUnspecified {
		last = style
	}
	return &TestTree{
		children: []*TestTree{
			{name: "node A", style: style},
			{
				name: "node B",
				children: []*TestTree{
					{name: "node B1", style: style},
					{name: "node B2", style: style},
					{name: "node B3", style: last},
				},
				style: style,
			},
			{name: "node C", style: last},
		},
		style: lastChildStyle,
	}
}

func TestTreeStylesControl(t *testing.T) {
	tests := []styleCase{
		{desc: "dense", name: "", style: DiagnosticsTreeStyleDense, lastChildStyle: DiagnosticsTreeStyleUnspecified,
			golden: "TestTree#00000\n├child node A: TestTree#00000\n├child node B: TestTree#00000\n│├child node B1: TestTree#00000\n│├child node B2: TestTree#00000\n│└child node B3: TestTree#00000\n└child node C: TestTree#00000\n"},
		{desc: "sparse", name: "", style: DiagnosticsTreeStyleSparse, lastChildStyle: DiagnosticsTreeStyleUnspecified,
			golden: "TestTree#00000\n ├─child node A: TestTree#00000\n ├─child node B: TestTree#00000\n │ ├─child node B1: TestTree#00000\n │ ├─child node B2: TestTree#00000\n │ └─child node B3: TestTree#00000\n └─child node C: TestTree#00000\n"},
		{desc: "dashed", name: "", style: DiagnosticsTreeStyleOffstage, lastChildStyle: DiagnosticsTreeStyleUnspecified,
			golden: "TestTree#00000\n ╎╌child node A: TestTree#00000\n ╎╌child node B: TestTree#00000\n ╎ ╎╌child node B1: TestTree#00000\n ╎ ╎╌child node B2: TestTree#00000\n ╎ └╌child node B3: TestTree#00000\n └╌child node C: TestTree#00000\n"},
		{desc: "leaf children", name: "", style: DiagnosticsTreeStyleSparse, lastChildStyle: DiagnosticsTreeStyleTransition,
			golden: "TestTree#00000\n ├─child node A: TestTree#00000\n ├─child node B: TestTree#00000\n │ ├─child node B1: TestTree#00000\n │ ├─child node B2: TestTree#00000\n │ ╘═╦══ child node B3 ═══\n │   ║ TestTree#00000\n │   ╚═══════════\n ╘═╦══ child node C ═══\n   ║ TestTree#00000\n   ╚═══════════\n"},
		{desc: "error children", name: "", style: DiagnosticsTreeStyleSparse, lastChildStyle: DiagnosticsTreeStyleError,
			golden: "TestTree#00000\n ├─child node A: TestTree#00000\n ├─child node B: TestTree#00000\n │ ├─child node B1: TestTree#00000\n │ ├─child node B2: TestTree#00000\n │ ╘═╦══╡ CHILD NODE B3: TESTTREE#00000 ╞═══════════════════════════════\n │   ╚══════════════════════════════════════════════════════════════════\n ╘═╦══╡ CHILD NODE C: TESTTREE#00000 ╞════════════════════════════════\n   ╚══════════════════════════════════════════════════════════════════\n"},
		{desc: "transition", name: "", style: DiagnosticsTreeStyleTransition, lastChildStyle: DiagnosticsTreeStyleUnspecified,
			golden: "TestTree#00000:\n  ╞═╦══ child node A ═══\n  │ ║ TestTree#00000\n  │ ╚═══════════\n  ╞═╦══ child node B ═══\n  │ ║ TestTree#00000:\n  │ ║   ╞═╦══ child node B1 ═══\n  │ ║   │ ║ TestTree#00000\n  │ ║   │ ╚═══════════\n  │ ║   ╞═╦══ child node B2 ═══\n  │ ║   │ ║ TestTree#00000\n  │ ║   │ ╚═══════════\n  │ ║   ╘═╦══ child node B3 ═══\n  │ ║     ║ TestTree#00000\n  │ ║     ╚═══════════\n  │ ╚═══════════\n  ╘═╦══ child node C ═══\n    ║ TestTree#00000\n    ╚═══════════\n"},
		{desc: "error", name: "", style: DiagnosticsTreeStyleError, lastChildStyle: DiagnosticsTreeStyleUnspecified,
			golden: "══╡ TESTTREE#00000 ╞═════════════════════════════════════════════\n╞═╦══╡ CHILD NODE A: TESTTREE#00000 ╞════════════════════════════════\n│ ╚══════════════════════════════════════════════════════════════════\n╞═╦══╡ CHILD NODE B: TESTTREE#00000 ╞════════════════════════════════\n│ ║ ╞═╦══╡ CHILD NODE B1: TESTTREE#00000 ╞═══════════════════════════════\n│ ║ │ ╚══════════════════════════════════════════════════════════════════\n│ ║ ╞═╦══╡ CHILD NODE B2: TESTTREE#00000 ╞═══════════════════════════════\n│ ║ │ ╚══════════════════════════════════════════════════════════════════\n│ ║ ╘═╦══╡ CHILD NODE B3: TESTTREE#00000 ╞═══════════════════════════════\n│ ║   ╚══════════════════════════════════════════════════════════════════\n│ ╚══════════════════════════════════════════════════════════════════\n╘═╦══╡ CHILD NODE C: TESTTREE#00000 ╞════════════════════════════════\n  ╚══════════════════════════════════════════════════════════════════\n═════════════════════════════════════════════════════════════════\n"},
		{desc: "whitespace", name: "", style: DiagnosticsTreeStyleWhitespace, lastChildStyle: DiagnosticsTreeStyleUnspecified,
			golden: "TestTree#00000:\n  child node A: TestTree#00000\n  child node B: TestTree#00000:\n    child node B1: TestTree#00000\n    child node B2: TestTree#00000\n    child node B3: TestTree#00000\n  child node C: TestTree#00000\n"},
		{desc: "single line", name: "", style: DiagnosticsTreeStyleSingleLine, lastChildStyle: DiagnosticsTreeStyleUnspecified,
			golden: "TestTree#00000"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			tree := controlTree(tt.style, tt.lastChildStyle)
			got := ignoringHashCodes(ToDiagnosticsNode(tree, DiagnosticsNodeOpts{Style: tt.style}).ToStringDeep())
			if got != tt.golden {
				t.Errorf("%s:\ngot:\n%s\nwant:\n%s", tt.desc, got, tt.golden)
			}
		})
	}
}

func propertiesTree(style, lastChildStyle, propertyStyle DiagnosticsTreeStyle) *TestTree {
	last := lastChildStyle
	if last == DiagnosticsTreeStyleUnspecified {
		last = style
	}
	str := func(name, value string, o StringPropertyOpts) DiagnosticsNode {
		o.Unquoted = true
		o.Style = propertyStyle
		return NewStringProperty(name, Some(value), o)
	}
	return &TestTree{
		properties: []DiagnosticsNode{
			str("stringProperty1", "value1", StringPropertyOpts{}),
			NewDoubleProperty("doubleProperty1", Some(42.5), NumberPropertyOpts{Style: propertyStyle}),
			NewDoubleProperty("roundedProperty", Some(1.0/3.0), NumberPropertyOpts{Style: propertyStyle}),
			str("DO_NOT_SHOW", "DO_NOT_SHOW", StringPropertyOpts{Level: DiagnosticLevelHidden}),
			NewDiagnosticsProperty[any]("DO_NOT_SHOW_NULL", nil, DiagnosticsPropertyOpts{DefaultValue: DefaultNull, Style: propertyStyle}),
			NewDiagnosticsProperty[any]("nullProperty", nil, DiagnosticsPropertyOpts{Style: propertyStyle}),
			str("node_type", "<root node>", StringPropertyOpts{ShowName: Some(false)}),
		},
		children: []*TestTree{
			{name: "node A", style: style},
			{
				name: "node B",
				properties: []DiagnosticsNode{
					str("p1", "v1", StringPropertyOpts{}),
					str("p2", "v2", StringPropertyOpts{}),
				},
				children: []*TestTree{
					{name: "node B1", style: style},
					{name: "node B2", properties: []DiagnosticsNode{str("property1", "value1", StringPropertyOpts{})}, style: style},
					{
						name: "node B3",
						properties: []DiagnosticsNode{
							str("node_type", "<leaf node>", StringPropertyOpts{ShowName: Some(false)}),
							NewIntProperty("foo", Some(42), NumberPropertyOpts{Style: propertyStyle}),
						},
						style: last,
					},
				},
				style: style,
			},
			{name: "node C", properties: []DiagnosticsNode{str("foo", "multi\nline\nvalue!", StringPropertyOpts{})}, style: last},
		},
		style: lastChildStyle,
	}
}

func TestTreeStylesWithProperties(t *testing.T) {
	tests := []styleCase{
		{desc: "sparse", name: "", style: DiagnosticsTreeStyleSparse, lastChildStyle: DiagnosticsTreeStyleUnspecified, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "TestTree#00000\n │ stringProperty1: value1\n │ doubleProperty1: 42.5\n │ roundedProperty: 0.3\n │ nullProperty: null\n │ <root node>\n │\n ├─child node A: TestTree#00000\n ├─child node B: TestTree#00000\n │ │ p1: v1\n │ │ p2: v2\n │ │\n │ ├─child node B1: TestTree#00000\n │ ├─child node B2: TestTree#00000\n │ │   property1: value1\n │ │\n │ └─child node B3: TestTree#00000\n │     <leaf node>\n │     foo: 42\n │\n └─child node C: TestTree#00000\n     foo:\n       multi\n       line\n       value!\n"},
		{desc: "sparse with indented single line properties", name: "", style: DiagnosticsTreeStyleSparse, lastChildStyle: DiagnosticsTreeStyleUnspecified, propertyStyle: DiagnosticsTreeStyleErrorProperty,
			golden: "TestTree#00000\n │ stringProperty1:\n │   value1\n │ doubleProperty1:\n │   42.5\n │ roundedProperty:\n │   0.3\n │ nullProperty:\n │   null\n │ <root node>\n │\n ├─child node A: TestTree#00000\n ├─child node B: TestTree#00000\n │ │ p1:\n │ │   v1\n │ │ p2:\n │ │   v2\n │ │\n │ ├─child node B1: TestTree#00000\n │ ├─child node B2: TestTree#00000\n │ │   property1:\n │ │     value1\n │ │\n │ └─child node B3: TestTree#00000\n │     <leaf node>\n │     foo:\n │       42\n │\n └─child node C: TestTree#00000\n     foo:\n       multi\n       line\n       value!\n"},
		{desc: "dense", name: "", style: DiagnosticsTreeStyleDense, lastChildStyle: DiagnosticsTreeStyleUnspecified, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, nullProperty: null, <root node>)\n├child node A: TestTree#00000\n├child node B: TestTree#00000(p1: v1, p2: v2)\n│├child node B1: TestTree#00000\n│├child node B2: TestTree#00000(property1: value1)\n│└child node B3: TestTree#00000(<leaf node>, foo: 42)\n└child node C: TestTree#00000(foo: multi\\nline\\nvalue!)\n"},
		{desc: "dashed", name: "", style: DiagnosticsTreeStyleOffstage, lastChildStyle: DiagnosticsTreeStyleUnspecified, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "TestTree#00000\n │ stringProperty1: value1\n │ doubleProperty1: 42.5\n │ roundedProperty: 0.3\n │ nullProperty: null\n │ <root node>\n │\n ╎╌child node A: TestTree#00000\n ╎╌child node B: TestTree#00000\n ╎ │ p1: v1\n ╎ │ p2: v2\n ╎ │\n ╎ ╎╌child node B1: TestTree#00000\n ╎ ╎╌child node B2: TestTree#00000\n ╎ ╎   property1: value1\n ╎ ╎\n ╎ └╌child node B3: TestTree#00000\n ╎     <leaf node>\n ╎     foo: 42\n ╎\n └╌child node C: TestTree#00000\n     foo:\n       multi\n       line\n       value!\n"},
		{desc: "transition children", name: "", style: DiagnosticsTreeStyleSparse, lastChildStyle: DiagnosticsTreeStyleTransition, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "TestTree#00000\n │ stringProperty1: value1\n │ doubleProperty1: 42.5\n │ roundedProperty: 0.3\n │ nullProperty: null\n │ <root node>\n │\n ├─child node A: TestTree#00000\n ├─child node B: TestTree#00000\n │ │ p1: v1\n │ │ p2: v2\n │ │\n │ ├─child node B1: TestTree#00000\n │ ├─child node B2: TestTree#00000\n │ │   property1: value1\n │ │\n │ ╘═╦══ child node B3 ═══\n │   ║ TestTree#00000:\n │   ║   <leaf node>\n │   ║   foo: 42\n │   ╚═══════════\n ╘═╦══ child node C ═══\n   ║ TestTree#00000:\n   ║   foo:\n   ║     multi\n   ║     line\n   ║     value!\n   ╚═══════════\n"},
		{desc: "error children", name: "", style: DiagnosticsTreeStyleSparse, lastChildStyle: DiagnosticsTreeStyleError, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "TestTree#00000\n │ stringProperty1: value1\n │ doubleProperty1: 42.5\n │ roundedProperty: 0.3\n │ nullProperty: null\n │ <root node>\n │\n ├─child node A: TestTree#00000\n ├─child node B: TestTree#00000\n │ │ p1: v1\n │ │ p2: v2\n │ │\n │ ├─child node B1: TestTree#00000\n │ ├─child node B2: TestTree#00000\n │ │   property1: value1\n │ │\n │ ╘═╦══╡ CHILD NODE B3: TESTTREE#00000 ╞═══════════════════════════════\n │   ║ <leaf node>\n │   ║ foo: 42\n │   ╚══════════════════════════════════════════════════════════════════\n ╘═╦══╡ CHILD NODE C: TESTTREE#00000 ╞════════════════════════════════\n   ║ foo:\n   ║   multi\n   ║   line\n   ║   value!\n   ╚══════════════════════════════════════════════════════════════════\n"},
		{desc: "transition", name: "", style: DiagnosticsTreeStyleTransition, lastChildStyle: DiagnosticsTreeStyleUnspecified, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "TestTree#00000:\n  stringProperty1: value1\n  doubleProperty1: 42.5\n  roundedProperty: 0.3\n  nullProperty: null\n  <root node>\n  ╞═╦══ child node A ═══\n  │ ║ TestTree#00000\n  │ ╚═══════════\n  ╞═╦══ child node B ═══\n  │ ║ TestTree#00000:\n  │ ║   p1: v1\n  │ ║   p2: v2\n  │ ║   ╞═╦══ child node B1 ═══\n  │ ║   │ ║ TestTree#00000\n  │ ║   │ ╚═══════════\n  │ ║   ╞═╦══ child node B2 ═══\n  │ ║   │ ║ TestTree#00000:\n  │ ║   │ ║   property1: value1\n  │ ║   │ ╚═══════════\n  │ ║   ╘═╦══ child node B3 ═══\n  │ ║     ║ TestTree#00000:\n  │ ║     ║   <leaf node>\n  │ ║     ║   foo: 42\n  │ ║     ╚═══════════\n  │ ╚═══════════\n  ╘═╦══ child node C ═══\n    ║ TestTree#00000:\n    ║   foo:\n    ║     multi\n    ║     line\n    ║     value!\n    ╚═══════════\n"},
		{desc: "whitespace", name: "", style: DiagnosticsTreeStyleWhitespace, lastChildStyle: DiagnosticsTreeStyleUnspecified, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "TestTree#00000:\n  stringProperty1: value1\n  doubleProperty1: 42.5\n  roundedProperty: 0.3\n  nullProperty: null\n  <root node>\n  child node A: TestTree#00000\n  child node B: TestTree#00000:\n    p1: v1\n    p2: v2\n    child node B1: TestTree#00000\n    child node B2: TestTree#00000:\n      property1: value1\n    child node B3: TestTree#00000:\n      <leaf node>\n      foo: 42\n  child node C: TestTree#00000:\n    foo:\n      multi\n      line\n      value!\n"},
		{desc: "flat", name: "", style: DiagnosticsTreeStyleFlat, lastChildStyle: DiagnosticsTreeStyleUnspecified, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "TestTree#00000:\nstringProperty1: value1\ndoubleProperty1: 42.5\nroundedProperty: 0.3\nnullProperty: null\n<root node>\nchild node A: TestTree#00000\nchild node B: TestTree#00000:\np1: v1\np2: v2\nchild node B1: TestTree#00000\nchild node B2: TestTree#00000:\nproperty1: value1\nchild node B3: TestTree#00000:\n<leaf node>\nfoo: 42\nchild node C: TestTree#00000:\nfoo:\n  multi\n  line\n  value!\n"},
		{desc: "single line", name: "", style: DiagnosticsTreeStyleSingleLine, lastChildStyle: DiagnosticsTreeStyleUnspecified, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, nullProperty: null, <root node>)"},
		{desc: "single line", name: "some name", style: DiagnosticsTreeStyleSingleLine, lastChildStyle: DiagnosticsTreeStyleUnspecified, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "some name: TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, nullProperty: null, <root node>)"},
		{desc: "indented single line", name: "", style: DiagnosticsTreeStyleErrorProperty, lastChildStyle: DiagnosticsTreeStyleUnspecified, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, nullProperty: null, <root node>)\n"},
		{desc: "indented single line", name: "some name", style: DiagnosticsTreeStyleErrorProperty, lastChildStyle: DiagnosticsTreeStyleUnspecified, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "some name:\n  TestTree#00000(stringProperty1: value1, doubleProperty1: 42.5, roundedProperty: 0.3, nullProperty: null, <root node>)\n"},
		{desc: "single line last child", name: "", style: DiagnosticsTreeStyleSparse, lastChildStyle: DiagnosticsTreeStyleSingleLine, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "TestTree#00000\n │ stringProperty1: value1\n │ doubleProperty1: 42.5\n │ roundedProperty: 0.3\n │ nullProperty: null\n │ <root node>\n │\n ├─child node A: TestTree#00000\n ├─child node B: TestTree#00000\n │ │ p1: v1\n │ │ p2: v2\n │ │\n │ ├─child node B1: TestTree#00000\n │ ├─child node B2: TestTree#00000\n │ │   property1: value1\n │ │\n │ └─child node B3: TestTree#00000(<leaf node>, foo: 42)\n └─child node C: TestTree#00000(foo: multi\\nline\\nvalue!)\n"},
		{desc: "indented single line last child", name: "", style: DiagnosticsTreeStyleSparse, lastChildStyle: DiagnosticsTreeStyleErrorProperty, propertyStyle: DiagnosticsTreeStyleSingleLine,
			golden: "TestTree#00000\n │ stringProperty1: value1\n │ doubleProperty1: 42.5\n │ roundedProperty: 0.3\n │ nullProperty: null\n │ <root node>\n │\n ├─child node A: TestTree#00000\n ├─child node B: TestTree#00000\n │ │ p1: v1\n │ │ p2: v2\n │ │\n │ ├─child node B1: TestTree#00000\n │ ├─child node B2: TestTree#00000\n │ │   property1: value1\n │ │\n │ └─child node B3:\n │     TestTree#00000(<leaf node>, foo: 42)\n └─child node C:\n     TestTree#00000(foo: multi\\nline\\nvalue!)\n"},
	}
	for _, tt := range tests {
		t.Run(tt.desc+"/"+tt.name, func(t *testing.T) {
			tree := propertiesTree(tt.style, tt.lastChildStyle, tt.propertyStyle)
			opts := DiagnosticsNodeOpts{Style: tt.style}
			if tt.name != "" {
				opts.Name = Some(tt.name)
			}
			got := ignoringHashCodes(ToDiagnosticsNode(tree, opts).ToStringDeep())
			if got != tt.golden {
				t.Errorf("%s:\ngot:\n%s\nwant:\n%s", tt.desc, got, tt.golden)
			}
		})
	}
}
