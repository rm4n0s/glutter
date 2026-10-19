package ui

import "strings"

// Name returns the Dart enum name, such as "italic".
//
// Flutter: dart:core Enum.name
func (e FontStyle) Name() string { return strings.TrimPrefix(e.String(), "FontStyle.") }

// Name returns the Dart enum name, such as "center".
//
// Flutter: dart:core Enum.name
func (e TextAlign) Name() string { return strings.TrimPrefix(e.String(), "TextAlign.") }

// Name returns the Dart enum name, such as "alphabetic".
//
// Flutter: dart:core Enum.name
func (e TextBaseline) Name() string { return strings.TrimPrefix(e.String(), "TextBaseline.") }

// Name returns the Dart enum name, such as "wavy".
//
// Flutter: dart:core Enum.name
func (e TextDecorationStyle) Name() string {
	return strings.TrimPrefix(e.String(), "TextDecorationStyle.")
}

// Name returns the Dart enum name, such as "even".
//
// Flutter: dart:core Enum.name
func (e TextLeadingDistribution) Name() string {
	return strings.TrimPrefix(e.String(), "TextLeadingDistribution.")
}

// Name returns the Dart enum name, such as "ltr".
//
// Flutter: dart:core Enum.name
func (e TextDirection) Name() string { return strings.TrimPrefix(e.String(), "TextDirection.") }

// Name returns the Dart enum name, such as "downstream".
//
// Flutter: dart:core Enum.name
func (e TextAffinity) Name() string { return strings.TrimPrefix(e.String(), "TextAffinity.") }

// Name returns the Dart enum name, such as "middle".
//
// Flutter: dart:core Enum.name
func (e PlaceholderAlignment) Name() string {
	return strings.TrimPrefix(e.String(), "PlaceholderAlignment.")
}

// Name returns the Dart enum name, such as "tight".
//
// Flutter: dart:core Enum.name
func (e BoxHeightStyle) Name() string { return strings.TrimPrefix(e.String(), "BoxHeightStyle.") }

// Name returns the Dart enum name, such as "tight".
//
// Flutter: dart:core Enum.name
func (e BoxWidthStyle) Name() string { return strings.TrimPrefix(e.String(), "BoxWidthStyle.") }

// Name returns the Dart enum name, such as "srcOver".
//
// Flutter: dart:core Enum.name
func (e BlendMode) Name() string { return strings.TrimPrefix(e.String(), "BlendMode.") }

// Name returns the Dart enum name, such as "hardEdge".
//
// Flutter: dart:core Enum.name
func (e Clip) Name() string { return strings.TrimPrefix(e.String(), "Clip.") }

// Name returns the Dart enum name, such as "medium".
//
// Flutter: dart:core Enum.name
func (e FilterQuality) Name() string { return strings.TrimPrefix(e.String(), "FilterQuality.") }
