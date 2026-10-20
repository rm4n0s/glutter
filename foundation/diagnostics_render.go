// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package foundation

import (
	"strings"
)

// TextTreeRenderer renders a tree of DiagnosticsNodes as text.
//
// Flutter: foundation/diagnostics.dart TextTreeRenderer
type TextTreeRenderer struct {
	minLevel                      DiagnosticLevel
	wrapWidth                     int
	wrapWidthProperties           int
	maxDescendentsTruncatableNode int
}

// TextTreeRendererOpts are the optional parameters of NewTextTreeRenderer.
//
// Flutter: foundation/diagnostics.dart TextTreeRenderer.new
type TextTreeRendererOpts struct {
	// MinLevel hides the properties less important than it; the default is
	// DiagnosticLevelDebug.
	MinLevel DiagnosticLevel
	// WrapWidth is the width at which lines wrap; the default is 100.
	WrapWidth int
	// WrapWidthProperties is the width at which the properties of a node wrap
	// when the prefix is long; the default is 65.
	WrapWidthProperties int
	// MaxDescendentsTruncatableNode is how many properties and children a
	// node that may be truncated shows; the default (None) is all.
	MaxDescendentsTruncatableNode Opt[int]
}

// NewTextTreeRenderer creates a renderer.
//
// Flutter: foundation/diagnostics.dart TextTreeRenderer.new
func NewTextTreeRenderer(opts ...TextTreeRendererOpts) *TextTreeRenderer {
	o := firstOpts(opts)
	r := &TextTreeRenderer{
		minLevel:                      o.MinLevel.orDefault(DiagnosticLevelDebug),
		wrapWidth:                     o.WrapWidth,
		wrapWidthProperties:           o.WrapWidthProperties,
		maxDescendentsTruncatableNode: o.MaxDescendentsTruncatableNode.OrElse(-1),
	}
	if r.wrapWidth == 0 {
		r.wrapWidth = 100
	}
	if r.wrapWidthProperties == 0 {
		r.wrapWidthProperties = 65
	}
	return r
}

// RenderOpts are the optional parameters of TextTreeRenderer.Render.
//
// Flutter: foundation/diagnostics.dart TextTreeRenderer.render
type RenderOpts struct {
	// PrefixLineOne is added before the first line.
	PrefixLineOne string
	// PrefixOtherLines is added before the other lines; the default is
	// PrefixLineOne.
	PrefixOtherLines Opt[string]
	// ParentConfiguration is the configuration of the parent, if any.
	ParentConfiguration *TextTreeConfiguration
}

func isSingleLineStyle(style DiagnosticsTreeStyle) bool {
	return style == DiagnosticsTreeStyleSingleLine
}

// childTextConfiguration is the configuration that draws a child.
func childTextConfiguration(child DiagnosticsNode, textStyle *TextTreeConfiguration) *TextTreeConfiguration {
	childStyle := child.Style()
	if isSingleLineStyle(childStyle) || childStyle == DiagnosticsTreeStyleErrorProperty {
		return textStyle
	}
	return child.TextTreeConfiguration()
}

// Render renders the node and its subtree as text. Release builds render
// nothing.
//
// Flutter: foundation/diagnostics.dart TextTreeRenderer.render
func (r *TextTreeRenderer) Render(node DiagnosticsNode, opts ...RenderOpts) string {
	o := firstOpts(opts)
	prefixOtherLines, hasPrefixOtherLines := o.PrefixOtherLines.Get()
	return r.render(node, o.PrefixLineOne, prefixOtherLines, hasPrefixOtherLines, o.ParentConfiguration)
}

func (r *TextTreeRenderer) render(node DiagnosticsNode, prefixLineOne, prefixOtherLines string, hasPrefixOtherLines bool, parentConfiguration *TextTreeConfiguration) string {
	if ReleaseMode {
		return ""
	}
	isSingleLine := isSingleLineStyle(node.Style()) && (parentConfiguration == nil || !parentConfiguration.LineBreakProperties)
	if !hasPrefixOtherLines {
		prefixOtherLines = prefixLineOne
	}
	if linePrefix, ok := node.LinePrefix().Get(); ok {
		prefixLineOne += linePrefix
		prefixOtherLines += linePrefix
	}

	config := node.TextTreeConfiguration()
	if prefixOtherLines == "" {
		prefixOtherLines += config.PrefixOtherLinesRootNode
	}

	if node.Style() == DiagnosticsTreeStyleTruncateChildren {
		return r.renderTruncated(node, prefixLineOne, prefixOtherLines)
	}

	builder := newPrefixedStringBuilder(prefixLineOne, prefixOtherLines, max(r.wrapWidth, strLen(prefixOtherLines)+r.wrapWidthProperties))

	children := node.GetChildren()
	description := node.ToDescription(parentConfiguration)
	if config.BeforeName != "" {
		builder.write(config.BeforeName, false)
	}
	wrapName := !isSingleLine && node.AllowNameWrap()
	wrapDescription := !isSingleLine && node.AllowWrap()
	uppercaseTitle := node.Style() == DiagnosticsTreeStyleError
	name, hasName := node.Name().Get()
	if uppercaseTitle && hasName {
		name = strings.ToUpper(name)
	}
	if description == "" {
		if node.ShowName() && hasName {
			builder.write(name, wrapName)
		}
	} else {
		includeName := false
		if hasName && name != "" && node.ShowName() {
			includeName = true
			builder.write(name, wrapName)
			if node.ShowSeparator() {
				builder.write(config.AfterName, wrapName)
			}
			if config.IsNameOnOwnLine || strings.Contains(description, "\n") {
				builder.write("\n", wrapName)
			} else {
				builder.write(" ", wrapName)
			}
		}
		if !isSingleLine && builder.requiresMultipleLines() && !builder.isCurrentLineEmpty() {
			// Make sure there is a break between the current line and next one
			// if there is not one already.
			builder.write("\n", false)
		}
		if includeName {
			builder.incrementPrefixOtherLines(bodyPrefix(config, len(children)), true)
		}
		if uppercaseTitle {
			description = strings.ToUpper(description)
		}
		builder.write(trimRight(description), wrapDescription)
		if !includeName {
			builder.incrementPrefixOtherLines(bodyPrefix(config, len(children)), false)
		}
	}

	if config.SuffixLineOne != "" {
		builder.writeStretched(config.SuffixLineOne, builder.wrapWidth)
	}

	var propertiesIterable []DiagnosticsNode
	for _, n := range node.GetProperties() {
		if !n.IsFiltered(r.minLevel) {
			propertiesIterable = append(propertiesIterable, n)
		}
	}
	properties := propertiesIterable
	if r.maxDescendentsTruncatableNode >= 0 && node.AllowTruncate() {
		if len(propertiesIterable) < r.maxDescendentsTruncatableNode {
			properties = append(append([]DiagnosticsNode(nil), propertiesIterable[:min(len(propertiesIterable), r.maxDescendentsTruncatableNode)]...), NewDiagnosticsNodeMessage("..."))
		}
		if r.maxDescendentsTruncatableNode < len(children) {
			children = append(append([]DiagnosticsNode(nil), children[:r.maxDescendentsTruncatableNode]...), NewDiagnosticsNodeMessage("..."))
		}
	}

	// If the node does not show a separator and there is no description then
	// we should not place a separator between the name and the value.
	// Essentially in this case the properties are treated a bit like a value.
	emptyBody, hasEmptyBody := node.EmptyBodyDescription().Get()
	if (len(properties) > 0 || len(children) > 0 || hasEmptyBody) && (node.ShowSeparator() || description != "") {
		builder.write(config.AfterDescriptionIfBody, false)
	}

	if config.LineBreakProperties {
		builder.write(config.LineBreak, false)
	}

	if len(properties) > 0 {
		builder.write(config.BeforeProperties, false)
	}

	builder.incrementPrefixOtherLines(config.BodyIndent, false)

	if hasEmptyBody && len(properties) == 0 && len(children) == 0 && prefixLineOne != "" {
		builder.write(emptyBody, false)
		if config.LineBreakProperties {
			builder.write(config.LineBreak, false)
		}
	}

	for i, property := range properties {
		if i > 0 {
			builder.write(config.PropertySeparator, false)
		}

		propertyStyle := property.TextTreeConfiguration()
		if isSingleLineStyle(property.Style()) {
			// We have to treat single line properties slightly differently to
			// deal with cases where a single line properties output may not
			// have single linebreak.
			propertyRender := r.render(property, propertyStyle.PrefixLineOne,
				propertyStyle.ChildLinkSpace+propertyStyle.PrefixOtherLines, true, config)
			propertyLines := strings.Split(propertyRender, "\n")
			if len(propertyLines) == 1 && !config.LineBreakProperties {
				builder.write(propertyLines[0], false)
			} else {
				builder.write(propertyRender, false)
				if !strings.HasSuffix(propertyRender, "\n") {
					builder.write("\n", false)
				}
			}
		} else {
			propertyRender := r.render(property,
				builder.currentPrefixOtherLines()+propertyStyle.PrefixLineOne,
				builder.currentPrefixOtherLines()+propertyStyle.ChildLinkSpace+propertyStyle.PrefixOtherLines,
				true, config)
			builder.writeRawLines(propertyRender)
		}
	}
	if len(properties) > 0 {
		builder.write(config.AfterProperties, false)
	}

	builder.write(config.MandatoryAfterProperties, false)

	if !config.LineBreakProperties {
		builder.write(config.LineBreak, false)
	}

	prefixChildren := config.BodyIndent
	prefixChildrenRaw := prefixOtherLines + prefixChildren
	if len(children) == 0 && config.AddBlankLineIfNoChildren && builder.requiresMultipleLines() &&
		trimRight(builder.currentPrefixOtherLines()) != "" {
		builder.write(config.LineBreak, false)
	}

	if len(children) > 0 && config.ShowChildren {
		if config.IsBlankLineBetweenPropertiesAndChildren && len(properties) > 0 &&
			children[0].TextTreeConfiguration().IsBlankLineBetweenPropertiesAndChildren {
			builder.write(config.LineBreak, false)
		}

		builder.setPrefixOtherLines(prefixOtherLines)

		for i, child := range children {
			childConfig := childTextConfiguration(child, config)
			if i == len(children)-1 {
				lastChildPrefixLineOne := prefixChildrenRaw + childConfig.PrefixLastChildLineOne
				childPrefixOtherLines := prefixChildrenRaw + childConfig.ChildLinkSpace + childConfig.PrefixOtherLines
				builder.writeRawLines(r.render(child, lastChildPrefixLineOne, childPrefixOtherLines, true, config))
				if childConfig.Footer != "" {
					builder.setPrefixOtherLines(prefixChildrenRaw)
					builder.write(childConfig.ChildLinkSpace+childConfig.Footer, false)
					if childConfig.MandatoryFooter != "" {
						builder.writeStretched(childConfig.MandatoryFooter,
							max(builder.wrapWidth, r.wrapWidthProperties+strLen(childPrefixOtherLines)))
					}
					builder.write(config.LineBreak, false)
				}
			} else {
				nextChildStyle := childTextConfiguration(children[i+1], config)
				childPrefixLineOne := prefixChildrenRaw + childConfig.PrefixLineOne
				childPrefixOtherLines := prefixChildrenRaw + nextChildStyle.LinkCharacter + childConfig.PrefixOtherLines
				builder.writeRawLines(r.render(child, childPrefixLineOne, childPrefixOtherLines, true, config))
				if childConfig.Footer != "" {
					builder.setPrefixOtherLines(prefixChildrenRaw)
					builder.write(childConfig.LinkCharacter+childConfig.Footer, false)
					if childConfig.MandatoryFooter != "" {
						builder.writeStretched(childConfig.MandatoryFooter,
							max(builder.wrapWidth, r.wrapWidthProperties+strLen(childPrefixOtherLines)))
					}
					builder.write(config.LineBreak, false)
				}
			}
		}
	}
	if parentConfiguration == nil && config.MandatoryFooter != "" {
		builder.writeStretched(config.MandatoryFooter, builder.wrapWidth)
		builder.write(config.LineBreak, false)
	}
	return builder.build()
}

// bodyPrefix is the prefix of the properties of a node.
func bodyPrefix(config *TextTreeConfiguration, numChildren int) string {
	if numChildren == 0 {
		return config.PropertyPrefixNoChildren
	}
	return config.PropertyPrefixIfChildren
}

// renderTruncated renders the node with the style
// DiagnosticsTreeStyleTruncateChildren: a short list of its descendants. The
// style is different enough that it does not reuse the rest of the logic.
func (r *TextTreeRenderer) renderTruncated(node DiagnosticsNode, prefixLineOne, prefixOtherLines string) string {
	var descendants []string
	const maxDepth = 5
	depth := 0
	const maxLines = 25
	lines := 0
	var visitor func(node DiagnosticsNode)
	visitor = func(node DiagnosticsNode) {
		for _, child := range node.GetChildren() {
			if lines < maxLines {
				depth++
				descendants = append(descendants, prefixOtherLines+strings.Repeat("  ", depth)+child.String())
				if depth < maxDepth {
					visitor(child)
				}
				depth--
			} else if lines == maxLines {
				descendants = append(descendants, prefixOtherLines+"  ...(descendants list truncated after "+itoa(lines)+" lines)")
			}
			lines++
		}
	}
	visitor(node)
	var information strings.Builder
	information.WriteString(prefixLineOne)
	name := node.Name().Value()
	switch {
	case lines > 1:
		information.WriteString("This " + name + " had the following descendants (showing up to depth " + itoa(maxDepth) + "):\n")
	case len(descendants) == 1:
		information.WriteString("This " + name + " had the following child:\n")
	default:
		information.WriteString("This " + name + " has no descendants.\n")
	}
	information.WriteString(strings.Join(descendants, "\n"))
	return information.String()
}
