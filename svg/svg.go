// Package svg draws SVG images in Glutter apps, with the widgets of Flutter's flutter_svg package.
//
// SvgPicture loads an SVG from an asset, a string, memory, a file or the network, parses it once (the Cache keeps
// the parsed document by source and theme), and draws it scaled to its box with a BoxFit and an Alignment. Parse,
// Document.Picture and its relatives are for an app that draws an SVG itself, in a CustomPainter.
//
// The package is apart from material, as flutter_svg is apart from Flutter: it is not re-exported by it. The parser
// and the painter are in internal/svg and are plain Go, so SVG looks the same on every platform. SPEC §26 says what
// is parsed and what is drawn.
//
// flutter_svg: svg.dart
package svg

import (
	"github.com/rm4n0s/glutter/internal/opts"
	"github.com/rm4n0s/glutter/internal/svg"
	"github.com/rm4n0s/glutter/ui"
)

// The tags of the errors of the package (SPEC §26.5, rm4n0s/errors).
const (
	// TagSvgParseFailed is the tag of an XML that is not well formed; the metadata are "line" and "column".
	TagSvgParseFailed = svg.TagSvgParseFailed
	// TagSvgNotSvg is the tag of a document whose root element is not svg; the metadata is "element".
	TagSvgNotSvg = svg.TagSvgNotSvg
	// TagSvgInvalidValue is the tag of a value that must be valid and is not, such as a viewBox with a negative size.
	TagSvgInvalidValue = svg.TagSvgInvalidValue
)

// ParseOpts are the optional parameters of Parse.
//
// flutter_svg: no counterpart
type ParseOpts struct {
	// Theme gives currentColor, em and ex; the zero value is the default theme.
	Theme SvgTheme
	// Locale picks the child of a switch element by its systemLanguage; the zero value is English.
	Locale ui.Locale
}

// Document is a parsed SVG document, for one theme. It is never changed, so it can be drawn from several goroutines.
//
// flutter_svg: no counterpart (vector_graphics PictureInfo)
type Document struct {
	d *svg.Document
}

// Parse reads an SVG document (XML, optionally compressed with gzip). It fails with SvgParseFailed when the XML is
// not well formed, SvgNotSvg when the root is not an svg element and SvgInvalidValue for a value that must be valid
// and is not; any other mistake of the document is ignored, as browsers do.
//
// flutter_svg: no counterpart (vg.encodeSvg)
func Parse(data []byte, o ...ParseOpts) (*Document, error) {
	po := opts.First(o)
	d, err := svg.Parse(data, svg.ParseOpts{Theme: po.Theme.withDefaults().internal(), Locale: po.Locale})
	if err != nil {
		return nil, err
	}
	return &Document{d: d}, nil
}

// PictureOpts are the optional parameters of PictureWith.
//
// flutter_svg: no counterpart
type PictureOpts struct {
	// Scale is the number of device pixels to a unit of the document (1 if it is not positive). It sets the
	// resolution of pattern tiles and filters, and the width of strokes that do not scale; the rest is vector.
	Scale float64
	// Text is the text engine that sets the text of the document, with the fonts of the app (the PlatformDispatcher's
	// TextEngine). With nil the text is set in the fonts that Glutter bundles.
	Text ui.ParagraphFactory
}

// Picture records the document into a picture of the document's size (Size), to draw at scale 1.
//
// flutter_svg: no counterpart (vector_graphics PictureInfo.picture)
func (d *Document) Picture() *ui.Picture { return d.d.Picture() }

// PictureAt is Picture for a picture that is drawn scale device pixels to a unit of the document.
//
// flutter_svg: no counterpart
func (d *Document) PictureAt(scale float64) *ui.Picture { return d.d.PictureAt(scale) }

// PictureWith is PictureAt with the text engine of the app.
//
// flutter_svg: no counterpart
func (d *Document) PictureWith(o PictureOpts) *ui.Picture {
	return d.d.PictureWith(svg.PictureOpts{Scale: o.Scale, Text: o.Text})
}

// Size is the size of the picture: the width and height of the document, or its viewBox.
//
// flutter_svg: no counterpart (vector_graphics PictureInfo.size)
func (d *Document) Size() ui.Size { return d.d.Size() }

// Title is the text of the title element of the document, which labels the picture for screen readers when the app
// gives no label.
//
// flutter_svg: no counterpart
func (d *Document) Title() string { return d.d.Title }

// Desc is the text of the desc element of the document.
//
// flutter_svg: no counterpart
func (d *Document) Desc() string { return d.d.Desc }
