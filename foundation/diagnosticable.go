// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package foundation

import "strings"

// Diagnosticable is an object that can describe its properties for debug
// output. Implement DebugFillProperties and add a property for each field
// worth showing; call the embedded type's DebugFillProperties first, as
// Flutter calls super.
//
// An object can also have a ToStringShort() string method (see
// ShortDescriber), and a DebugDescribeChildren() []DiagnosticsNode method
// makes it a DiagnosticableTree.
//
// Flutter: foundation/diagnostics.dart Diagnosticable
type Diagnosticable interface {
	// DebugFillProperties adds the properties of the object to the builder.
	//
	// Flutter: foundation/diagnostics.dart Diagnosticable.debugFillProperties
	DebugFillProperties(properties *DiagnosticPropertiesBuilder)
}

// DiagnosticableTree is a Diagnosticable that has children, which form a tree
// when dumped.
//
// Flutter: foundation/diagnostics.dart DiagnosticableTree
type DiagnosticableTree interface {
	Diagnosticable
	// DebugDescribeChildren returns the children of the object as nodes.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticableTree.debugDescribeChildren
	DebugDescribeChildren() []DiagnosticsNode
}

// ShortDescriber is an object that describes itself in a few words.
// ToStringShort uses it.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
type ShortDescriber interface {
	ToStringShort() string
}

// DiagnosticsNodeProvider is an object that makes its own diagnostics node,
// where the default one would not do. ToDiagnosticsNode uses it. A method of
// this kind must build the node with NewDiagnosticableNode or
// NewDiagnosticableTreeNode, not call ToDiagnosticsNode.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toDiagnosticsNode
type DiagnosticsNodeProvider interface {
	ToDiagnosticsNode(opts DiagnosticsNodeOpts) DiagnosticsNode
}

// DiagnosticsNodeOpts are the optional parameters of ToDiagnosticsNode.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toDiagnosticsNode
type DiagnosticsNodeOpts struct {
	// Name is the name of the node.
	Name Opt[string]
	// Style is how the node is drawn; the default is the one the object's
	// DebugFillProperties chooses.
	Style DiagnosticsTreeStyle
}

// ToStringShort is a brief description of the object: its ToStringShort
// method if it has one, and its type and ShortHash otherwise ("Foo#1a2b3").
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toStringShort
func ToStringShort(object any) string {
	if s, ok := object.(ShortDescriber); ok {
		return s.ToStringShort()
	}
	return DescribeIdentity(object)
}

// ToDiagnosticsNode returns a node that describes the object.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toDiagnosticsNode
func ToDiagnosticsNode(d Diagnosticable, opts ...DiagnosticsNodeOpts) DiagnosticsNode {
	o := firstOpts(opts)
	if p, ok := d.(DiagnosticsNodeProvider); ok {
		return p.ToDiagnosticsNode(o)
	}
	if tree, ok := d.(DiagnosticableTree); ok {
		return NewDiagnosticableTreeNode(tree, o)
	}
	return NewDiagnosticableNode(d, o)
}

// DiagnosticableString describes the object and its properties on one line,
// at a minimum level (the default is info). It is what Flutter's
// Diagnosticable.toString gives. Release builds give ToStringShort.
//
// Flutter: foundation/diagnostics.dart Diagnosticable.toString
func DiagnosticableString(d Diagnosticable, minLevel ...DiagnosticLevel) string {
	if !DebugMode {
		return ToStringShort(d)
	}
	level := DiagnosticLevelInfo
	if len(minLevel) > 0 {
		level = minLevel[0].orDefault(DiagnosticLevelInfo)
	}
	return ToDiagnosticsNode(d, DiagnosticsNodeOpts{Style: DiagnosticsTreeStyleSingleLine}).ToStringWith(nil, level)
}

// ToStringShallow returns a string describing the object and its properties,
// without its children. The properties are joined with joiner (the default is
// ", ") and those less important than minLevel (the default is debug) are
// left out. Release builds give DiagnosticableString.
//
// Flutter: foundation/diagnostics.dart DiagnosticableTree.toStringShallow
func ToStringShallow(d Diagnosticable, opts ...ToStringShallowOpts) string {
	o := firstOpts(opts)
	if !DebugMode {
		return DiagnosticableString(d)
	}
	joiner := o.Joiner
	if joiner == "" {
		joiner = ", "
	}
	minLevel := o.MinLevel.orDefault(DiagnosticLevelDebug)
	var sb strings.Builder
	sb.WriteString(DiagnosticableString(d))
	sb.WriteString(joiner)
	builder := NewDiagnosticPropertiesBuilder()
	d.DebugFillProperties(builder)
	var parts []string
	for _, n := range builder.Properties {
		if !n.IsFiltered(minLevel) {
			parts = append(parts, n.String())
		}
	}
	sb.WriteString(strings.Join(parts, joiner))
	return sb.String()
}

// ToStringShallowOpts are the optional parameters of ToStringShallow.
//
// Flutter: foundation/diagnostics.dart DiagnosticableTree.toStringShallow
type ToStringShallowOpts struct {
	// Joiner separates the properties; the default is ", ".
	Joiner string
	// MinLevel leaves out the properties less important than it; the default
	// is DiagnosticLevelDebug.
	MinLevel DiagnosticLevel
}

// ToStringDeep dumps the object and its descendants as text.
//
// Flutter: foundation/diagnostics.dart DiagnosticableTree.toStringDeep
func ToStringDeep(tree DiagnosticableTree, opts ...ToStringDeepOpts) string {
	return ToDiagnosticsNode(tree).ToStringDeep(opts...)
}

// DiagnosticPropertiesBuilder collects the properties of an object for
// DebugFillProperties.
//
// Flutter: foundation/diagnostics.dart DiagnosticPropertiesBuilder
type DiagnosticPropertiesBuilder struct {
	// Properties are the properties added so far.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticPropertiesBuilder.properties
	Properties []DiagnosticsNode
	// DefaultDiagnosticsTreeStyle is the style the object is drawn in, unless
	// the caller asks for another. It starts as DiagnosticsTreeStyleSparse.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticPropertiesBuilder.defaultDiagnosticsTreeStyle
	DefaultDiagnosticsTreeStyle DiagnosticsTreeStyle
	// EmptyBodyDescription is drawn for an object that has no properties and
	// no children.
	//
	// Flutter: foundation/diagnostics.dart DiagnosticPropertiesBuilder.emptyBodyDescription
	EmptyBodyDescription Opt[string]
}

// NewDiagnosticPropertiesBuilder creates an empty builder.
//
// Flutter: foundation/diagnostics.dart DiagnosticPropertiesBuilder.new
func NewDiagnosticPropertiesBuilder() *DiagnosticPropertiesBuilder {
	return &DiagnosticPropertiesBuilder{DefaultDiagnosticsTreeStyle: DiagnosticsTreeStyleSparse}
}

// NewDiagnosticPropertiesBuilderFromProperties creates a builder that starts
// with the given properties.
//
// Flutter: foundation/diagnostics.dart DiagnosticPropertiesBuilder.fromProperties
func NewDiagnosticPropertiesBuilderFromProperties(properties []DiagnosticsNode) *DiagnosticPropertiesBuilder {
	b := NewDiagnosticPropertiesBuilder()
	b.Properties = properties
	return b
}

// Add adds a property. Release builds ignore it.
//
// Flutter: foundation/diagnostics.dart DiagnosticPropertiesBuilder.add
func (b *DiagnosticPropertiesBuilder) Add(property DiagnosticsNode) {
	if DebugMode {
		b.Properties = append(b.Properties, property)
	}
}

// DiagnosticableNode is the node that describes a Diagnosticable object: its
// description is the object's ToStringShort and its properties come from
// DebugFillProperties. For a DiagnosticableTree its children come from
// DebugDescribeChildren.
//
// Flutter: foundation/diagnostics.dart DiagnosticableNode
type DiagnosticableNode struct {
	DiagnosticsNodeBase
	value    Diagnosticable
	children func() []DiagnosticsNode
	cached   *DiagnosticPropertiesBuilder
}

// NewDiagnosticableNode creates a node for an object.
//
// Flutter: foundation/diagnostics.dart DiagnosticableNode.new
func NewDiagnosticableNode(value Diagnosticable, opts ...DiagnosticsNodeOpts) *DiagnosticableNode {
	o := firstOpts(opts)
	n := &DiagnosticableNode{value: value}
	InitDiagnosticsNodeBase(&n.DiagnosticsNodeBase, n, DiagnosticsNodeBaseOpts{Name: o.Name, Style: o.Style})
	return n
}

// NewDiagnosticableTreeNode creates a node for an object with children.
//
// Flutter: foundation/diagnostics.dart DiagnosticableTreeNode.new
func NewDiagnosticableTreeNode(value DiagnosticableTree, opts ...DiagnosticsNodeOpts) *DiagnosticableNode {
	n := NewDiagnosticableNode(value, opts...)
	n.children = value.DebugDescribeChildren
	return n
}

// builder returns the properties of the object, collected once. Release builds
// have none.
func (n *DiagnosticableNode) builder() *DiagnosticPropertiesBuilder {
	if ReleaseMode {
		return nil
	}
	if n.cached == nil {
		n.cached = NewDiagnosticPropertiesBuilder()
		n.value.DebugFillProperties(n.cached)
	}
	return n.cached
}

// Value is the object the node describes.
func (n *DiagnosticableNode) Value() any { return n.value }

// Style is the style given to the node, or the one the object chose in
// DebugFillProperties; release builds draw nothing.
func (n *DiagnosticableNode) Style() DiagnosticsTreeStyle {
	if ReleaseMode {
		return DiagnosticsTreeStyleNone
	}
	if n.style != DiagnosticsTreeStyleUnspecified {
		return n.style
	}
	return n.builder().DefaultDiagnosticsTreeStyle
}

// EmptyBodyDescription is what the object chose to draw when it has nothing
// else.
func (n *DiagnosticableNode) EmptyBodyDescription() Opt[string] {
	if ReleaseMode || ProfileMode {
		return Some("")
	}
	return n.builder().EmptyBodyDescription
}

// GetProperties returns the properties the object added.
func (n *DiagnosticableNode) GetProperties() []DiagnosticsNode {
	if ReleaseMode || ProfileMode {
		return nil
	}
	return n.builder().Properties
}

// GetChildren returns the children of a tree object.
func (n *DiagnosticableNode) GetChildren() []DiagnosticsNode {
	if n.children == nil {
		return nil
	}
	return n.children()
}

// ToDescription is the ToStringShort of the object.
func (n *DiagnosticableNode) ToDescription(*TextTreeConfiguration) string {
	if !DebugMode {
		return ""
	}
	return ToStringShort(n.value)
}

// DiagnosticsBlock is a node with fixed properties and children, to group
// text in a dump.
//
// Flutter: foundation/diagnostics.dart DiagnosticsBlock
type DiagnosticsBlock struct {
	DiagnosticsNodeBase
	children    []DiagnosticsNode
	properties  []DiagnosticsNode
	level       DiagnosticLevel
	description string
	value       any
	truncate    bool
}

// DiagnosticsBlockOpts are the optional parameters of NewDiagnosticsBlock.
//
// Flutter: foundation/diagnostics.dart DiagnosticsBlock.new
type DiagnosticsBlockOpts struct {
	// Name is the name of the block.
	Name Opt[string]
	// Style is how the block is drawn; the default is whitespace.
	Style DiagnosticsTreeStyle
	// ShowName is whether the name is drawn; the default is true. A block
	// without a name never shows one.
	ShowName Opt[bool]
	// ShowSeparator is whether a colon follows the name; the default is true.
	ShowSeparator Opt[bool]
	// LinePrefix is text added before every line of the block.
	LinePrefix Opt[string]
	// Value is the object the block describes, if any.
	Value any
	// Description is the description of the block.
	Description string
	// Level is the importance of the block; the default is info.
	Level DiagnosticLevel
	// AllowTruncate is whether the properties and children may be cut short.
	AllowTruncate bool
	// Children are the children of the block.
	Children []DiagnosticsNode
	// Properties are the properties of the block.
	Properties []DiagnosticsNode
}

// NewDiagnosticsBlock creates a block.
//
// Flutter: foundation/diagnostics.dart DiagnosticsBlock.new
func NewDiagnosticsBlock(opts ...DiagnosticsBlockOpts) *DiagnosticsBlock {
	o := firstOpts(opts)
	b := &DiagnosticsBlock{
		children:    o.Children,
		properties:  o.Properties,
		level:       o.Level.orDefault(DiagnosticLevelInfo),
		description: o.Description,
		value:       o.Value,
		truncate:    o.AllowTruncate,
	}
	InitDiagnosticsNodeBase(&b.DiagnosticsNodeBase, b, DiagnosticsNodeBaseOpts{
		Name:          o.Name,
		Style:         o.Style.orDefault(DiagnosticsTreeStyleWhitespace),
		ShowName:      Some(o.ShowName.OrElse(true) && o.Name.IsSome()),
		ShowSeparator: o.ShowSeparator,
		LinePrefix:    o.LinePrefix,
	})
	return b
}

// Level is the importance of the block.
func (b *DiagnosticsBlock) Level() DiagnosticLevel { return b.level }

// Value is the object the block describes, if any.
func (b *DiagnosticsBlock) Value() any { return b.value }

// AllowTruncate is whether the properties and children may be cut short.
func (b *DiagnosticsBlock) AllowTruncate() bool { return b.truncate }

// GetChildren returns the children of the block.
func (b *DiagnosticsBlock) GetChildren() []DiagnosticsNode { return b.children }

// GetProperties returns the properties of the block.
func (b *DiagnosticsBlock) GetProperties() []DiagnosticsNode { return b.properties }

// ToDescription is the description of the block.
func (b *DiagnosticsBlock) ToDescription(*TextTreeConfiguration) string { return b.description }
