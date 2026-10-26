package svg

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"io"
	"regexp"
	"strings"

	"github.com/rm4n0s/errors"
)

const (
	nsSVG   = "http://www.w3.org/2000/svg"
	nsXLink = "http://www.w3.org/1999/xlink"
	nsXML   = "http://www.w3.org/XML/1998/namespace"
)

// element is an element of the XML source, before the cascade and the tree. Text is a child whose name is "#text".
type element struct {
	name     string
	attrs    []xmlAttr
	children []*element
	text     string // for "#text"
	parent   *element
	// id is the value of the id attribute, "" if none.
	id string
}

type xmlAttr struct {
	name, value string
}

// attr returns the value of an attribute and whether it is present.
func (e *element) attr(name string) (string, bool) {
	for i := range e.attrs {
		if e.attrs[i].name == name {
			return e.attrs[i].value, true
		}
	}
	return "", false
}

// attrOr returns the value of an attribute, or def if it is not present.
func (e *element) attrOr(name, def string) string {
	if v, ok := e.attr(name); ok {
		return v
	}
	return def
}

// href returns the value of href, or of xlink:href if there is no href (SVG 2 prefers href).
//
// SVG 2: linking.html#XLinkRefAttrs
func (e *element) href() (string, bool) {
	if v, ok := e.attr("href"); ok {
		return v, true
	}
	return e.attr("xlink:href")
}

// textContent returns the text of the element's "#text" children.
func (e *element) textContent() string {
	var sb strings.Builder
	for _, c := range e.children {
		if c.name == "#text" {
			sb.WriteString(c.text)
		}
	}
	return sb.String()
}

// knownElements are the elements of the SVG namespace that the parser reads. Every other element is skipped with
// its children, as browsers do for an element they do not know.
var knownElements = map[string]bool{
	"svg": true, "g": true, "defs": true, "symbol": true, "use": true, "switch": true, "a": true,
	"path": true, "rect": true, "circle": true, "ellipse": true, "line": true, "polyline": true, "polygon": true,
	"text": true, "tspan": true, "textPath": true, "tref": true, "image": true,
	"linearGradient": true, "radialGradient": true, "stop": true, "pattern": true,
	"clipPath": true, "mask": true, "marker": true, "style": true, "title": true, "desc": true,
	"filter": true, "feBlend": true, "feColorMatrix": true, "feComponentTransfer": true, "feComposite": true,
	"feConvolveMatrix": true, "feDiffuseLighting": true, "feDisplacementMap": true, "feDistantLight": true,
	"feDropShadow": true, "feFlood": true, "feFuncA": true, "feFuncB": true, "feFuncG": true, "feFuncR": true,
	"feGaussianBlur": true, "feImage": true, "feMerge": true, "feMergeNode": true, "feMorphology": true,
	"feOffset": true, "fePointLight": true, "feSpecularLighting": true, "feSpotLight": true, "feTile": true,
	"feTurbulence": true,
}

// textElements are the elements whose white-space-only text is content.
var textElements = map[string]bool{"text": true, "tspan": true, "textPath": true, "tref": true, "a": true}

var entityRE = regexp.MustCompile(`<!ENTITY\s+([^\s%]+)\s+(?:"([^"]*)"|'([^']*)')\s*>`)

// readXML reads the document into an element tree. It returns the root element, whatever its name; the caller checks
// that it is svg. A document that is not well formed is an SvgParseFailed error.
//
// Elements and attributes: those of the SVG namespace (or of none: a document without xmlns is read as SVG, as
// tools do) and the xlink:href and xml:space/xml:lang attributes. CDATA sections and the predefined and DTD-defined
// entities become text.
func readXML(data []byte) (*element, error) {
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, errors.NewErr(TagSvgParseFailed, err).SetMetadata("line", 0, "column", 0)
		}
		data, err = io.ReadAll(io.LimitReader(zr, maxDecompressed))
		if err != nil {
			return nil, errors.NewErr(TagSvgParseFailed, err).SetMetadata("line", 0, "column", 0)
		}
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	d := xml.NewDecoder(bytes.NewReader(data))
	d.CharsetReader = charsetReader
	d.Entity = map[string]string{}
	var root, cur *element
	skip := 0 // how many levels of a skipped subtree are open
	depth := 0
	inText := 0
	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF { //nolint:errorlint // io.EOF is returned as it is
				break
			}
			line, col := d.InputPos()
			if se, ok := err.(*xml.SyntaxError); ok && se.Line > 0 {
				line = se.Line
			}
			return nil, errors.NewErr(TagSvgParseFailed, err).SetMetadata("line", line, "column", col)
		}
		switch t := tok.(type) {
		case xml.Directive:
			if bytes.HasPrefix(t, []byte("DOCTYPE")) {
				for _, m := range entityRE.FindAllSubmatch(t, -1) {
					v := m[2]
					if v == nil {
						v = m[3]
					}
					d.Entity[string(m[1])] = string(v)
				}
			}
		case xml.StartElement:
			if skip > 0 {
				skip++
				continue
			}
			inSVG := t.Name.Space == nsSVG || t.Name.Space == ""
			if root != nil && (!inSVG || !knownElements[t.Name.Local]) {
				skip = 1
				continue
			}
			depth++
			if depth > maxDepth {
				line, col := d.InputPos()
				return nil, errors.New(TagSvgParseFailed, "the elements are nested too deeply", "line", line, "column", col, "limit", maxDepth)
			}
			e := &element{name: t.Name.Local, parent: cur}
			if !inSVG {
				e.name = "{" + t.Name.Space + "}" + t.Name.Local
			}
			e.attrs = make([]xmlAttr, 0, len(t.Attr))
			for _, a := range t.Attr {
				switch a.Name.Space {
				case "", nsSVG:
					if a.Name.Local == "xmlns" {
						continue
					}
					e.attrs = append(e.attrs, xmlAttr{a.Name.Local, a.Value})
				case nsXLink:
					if a.Name.Local == "href" {
						e.attrs = append(e.attrs, xmlAttr{"xlink:href", a.Value})
					}
				case nsXML:
					if a.Name.Local == "space" || a.Name.Local == "lang" {
						e.attrs = append(e.attrs, xmlAttr{"xml:" + a.Name.Local, a.Value})
					}
				}
				if a.Name.Local == "id" && (a.Name.Space == "" || a.Name.Space == nsSVG) {
					e.id = a.Value
				}
			}
			if cur != nil {
				cur.children = append(cur.children, e)
			} else {
				root = e
			}
			cur = e
			if textElements[e.name] {
				inText++
			}
		case xml.EndElement:
			if skip > 0 {
				skip--
				continue
			}
			depth--
			if cur != nil {
				if textElements[cur.name] {
					inText--
				}
				cur = cur.parent
			}
		case xml.CharData:
			if skip > 0 || cur == nil {
				continue
			}
			if !keepText(cur, inText) {
				continue
			}
			cur.children = append(cur.children, &element{name: "#text", text: string(t), parent: cur})
		}
	}
	if root == nil {
		line, col := d.InputPos()
		return nil, errors.New(TagSvgParseFailed, "the document has no element", "line", line, "column", col)
	}
	return root, nil
}

// keepText reports whether text inside cur is content: text of style, title and desc, and the text of text
// elements, where white space between tspans counts.
func keepText(cur *element, inText int) bool {
	switch cur.name {
	case "style", "title", "desc":
		return true
	}
	return inText > 0
}

// maxDepth is how deeply elements may nest.
const maxDepth = 1024

// maxDecompressed is the most that a compressed document may expand to.
const maxDecompressed = 256 << 20

// charsetReader reads documents that declare a single-byte encoding that is a subset of Latin-1.
func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(charset) {
	case "utf-8", "utf8", "us-ascii", "ascii":
		return input, nil
	case "iso-8859-1", "latin1", "latin-1", "windows-1252":
		b, err := io.ReadAll(input)
		if err != nil {
			return nil, errors.NewErr(TagSvgParseFailed, err).SetMetadata("line", 0, "column", 0)
		}
		out := make([]byte, 0, len(b)+len(b)/8)
		for _, c := range b {
			if c < 0x80 {
				out = append(out, c)
			} else {
				out = append(out, 0xc0|c>>6, 0x80|c&0x3f)
			}
		}
		return bytes.NewReader(out), nil
	}
	return nil, errors.New(TagSvgParseFailed, "the encoding of the document is not supported", "line", 1, "column", 1, "charset", charset)
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	// TagSvgParseFailed is the tag of the error for a document whose XML is not well formed (or is empty, compressed
	// wrongly or in an encoding that is not supported); the metadata are "line" and "column".
	TagSvgParseFailed = "SvgParseFailed"
)
