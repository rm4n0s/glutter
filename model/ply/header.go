package ply

import (
	"bytes"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model/internal/parse"
)

// format is how the body of the file is written.
type format uint8

const (
	formatASCII format = iota
	formatLittle
	formatBig
)

// scalar is the type of a property.
type scalar uint8

const (
	typeNone scalar = iota
	typeInt8
	typeUint8
	typeInt16
	typeUint16
	typeInt32
	typeUint32
	typeFloat32
	typeFloat64
)

// scalarNames are the names of the types, with the two spellings that files use.
var scalarNames = map[string]scalar{
	"char": typeInt8, "int8": typeInt8, "uchar": typeUint8, "uint8": typeUint8,
	"short": typeInt16, "int16": typeInt16, "ushort": typeUint16, "uint16": typeUint16,
	"int": typeInt32, "int32": typeInt32, "uint": typeUint32, "uint32": typeUint32,
	"float": typeFloat32, "float32": typeFloat32, "double": typeFloat64, "float64": typeFloat64,
}

// size is the number of bytes of a value of the type in a binary file.
func (s scalar) size() int {
	switch s {
	case typeInt8, typeUint8:
		return 1
	case typeInt16, typeUint16:
		return 2
	case typeInt32, typeUint32, typeFloat32:
		return 4
	case typeFloat64:
		return 8
	}
	return 0
}

// integer says whether the type holds whole numbers.
func (s scalar) integer() bool { return s != typeNone && s != typeFloat32 && s != typeFloat64 }

// Limits that keep a hostile header from asking for work or memory that the file cannot back.
const (
	maxHeaderBytes       = 1 << 20
	maxHeaderLines       = 1 << 14
	maxElements          = 64
	maxProperties        = 256
	maxCount             = math.MaxInt32
	maxFaceCorners       = 1 << 20
	minASCIIBytesPerItem = 2
)

// property is a property of an element: a value, or a list of values with a count in front of it.
type property struct {
	name string
	typ  scalar
	// list says that the property is a list: its count is of type countType, and typ is the type of its items.
	list      bool
	countType scalar
}

// element is a kind of record of the body, count of them, each with the properties in order.
type element struct {
	name  string
	count int
	props []property
	// line is the line of the header that made the element.
	line int
}

// header is what the head of the file says.
type header struct {
	format   format
	elements []element
	// bodyAt is the offset of the first byte after end_header's line, and lines the number of lines the header has.
	bodyAt int
	lines  int
}

// parseHeader reads the head of the file: the magic, the format, the elements with their properties, up to end_header.
func parseHeader(data []byte) (*header, error) {
	h := &header{}
	pos, line := 0, 0
	seenFormat := false
	for {
		if pos >= len(data) {
			return nil, errors.New(TagPLYInvalid, "the header has no end_header", "line", line, "reason", "no end_header")
		}
		if pos > maxHeaderBytes || line > maxHeaderLines {
			return nil, errors.New(TagPLYInvalid, "the header is too long", "line", line, "reason", "header too long")
		}
		end := bytes.IndexByte(data[pos:], '\n')
		next := len(data)
		text := data[pos:]
		if end >= 0 {
			text = data[pos : pos+end]
			next = pos + end + 1
		}
		line++
		pos = next
		text = parse.Trim(text)
		if line == 1 {
			if string(text) != "ply" {
				return nil, errors.New(TagPLYInvalid, "the file does not begin with ply", "line", line, "reason", "not a ply file")
			}
			continue
		}
		key, rest := parse.Field(text)
		switch string(key) {
		case "", "comment", "obj_info":
		case "format":
			name, rest := parse.Field(rest)
			version, rest := parse.Field(rest)
			if seenFormat {
				return nil, errors.New(TagPLYInvalid, "the format is given twice", "line", line, "reason", "format twice")
			}
			switch string(name) {
			case "ascii":
				h.format = formatASCII
			case "binary_little_endian":
				h.format = formatLittle
			case "binary_big_endian":
				h.format = formatBig
			default:
				return nil, errors.New(TagPLYInvalid, "the format is not ascii, binary_little_endian or binary_big_endian", "line", line, "reason", "unknown format", "format", string(name))
			}
			if string(version) != "1.0" {
				return nil, errors.New(TagPLYInvalid, "the version is not 1.0", "line", line, "reason", "unsupported version", "version", string(version))
			}
			if extra, _ := parse.Field(rest); extra != nil {
				return nil, errors.New(TagPLYInvalid, "the format line has more than a format and a version", "line", line, "reason", "bad format line")
			}
			seenFormat = true
		case "element":
			name, rest := parse.Field(rest)
			countText, rest := parse.Field(rest)
			if name == nil || countText == nil {
				return nil, errors.New(TagPLYInvalid, "an element has a name and a count", "line", line, "reason", "bad element")
			}
			if extra, _ := parse.Field(rest); extra != nil {
				return nil, errors.New(TagPLYInvalid, "an element line has more than a name and a count", "line", line, "reason", "bad element")
			}
			count, ok := parse.Int(countText)
			if !ok || count < 0 || count > maxCount {
				return nil, errors.New(TagPLYInvalid, "the count of an element is not a number from 0 to 2^31-1", "line", line, "reason", "bad count", "count", string(countText))
			}
			if len(h.elements) == maxElements {
				return nil, errors.New(TagPLYInvalid, "the file has too many elements", "line", line, "reason", "too many elements")
			}
			for _, e := range h.elements {
				if e.name == string(name) {
					return nil, errors.New(TagPLYInvalid, "an element is given twice", "line", line, "reason", "element twice", "element", e.name)
				}
			}
			h.elements = append(h.elements, element{name: string(name), count: count, line: line})
		case "property":
			if len(h.elements) == 0 {
				return nil, errors.New(TagPLYInvalid, "a property comes before any element", "line", line, "reason", "property without element")
			}
			e := &h.elements[len(h.elements)-1]
			if len(e.props) == maxProperties {
				return nil, errors.New(TagPLYInvalid, "an element has too many properties", "line", line, "reason", "too many properties", "element", e.name)
			}
			var p property
			word, rest := parse.Field(rest)
			if string(word) == "list" {
				p.list = true
				countName, rest2 := parse.Field(rest)
				itemName, rest3 := parse.Field(rest2)
				var nameField []byte
				nameField, rest = parse.Field(rest3)
				p.countType, p.typ = scalarNames[string(countName)], scalarNames[string(itemName)]
				if p.countType == typeNone || p.typ == typeNone {
					return nil, errors.New(TagPLYInvalid, "a list has types for its count and its items", "line", line, "reason", "bad type", "element", e.name)
				}
				if !p.countType.integer() {
					return nil, errors.New(TagPLYInvalid, "the count of a list is a whole number", "line", line, "reason", "bad list count type", "element", e.name)
				}
				p.name = string(nameField)
			} else {
				nameField, rest2 := parse.Field(rest)
				p.typ = scalarNames[string(word)]
				if p.typ == typeNone {
					return nil, errors.New(TagPLYInvalid, "the type of a property is not one of PLY's", "line", line, "reason", "bad type", "type", string(word))
				}
				p.name = string(nameField)
				rest = rest2
			}
			if p.name == "" {
				return nil, errors.New(TagPLYInvalid, "a property has a name", "line", line, "reason", "bad property", "element", e.name)
			}
			if extra, _ := parse.Field(rest); extra != nil {
				return nil, errors.New(TagPLYInvalid, "a property line has more than a type and a name", "line", line, "reason", "bad property", "element", e.name)
			}
			for _, q := range e.props {
				if q.name == p.name {
					return nil, errors.New(TagPLYInvalid, "a property is given twice", "line", line, "reason", "property twice", "element", e.name, "property", p.name)
				}
			}
			e.props = append(e.props, p)
		case "end_header":
			if !seenFormat {
				return nil, errors.New(TagPLYInvalid, "the header has no format line", "line", line, "reason", "no format")
			}
			// The body begins right after the line of end_header (a carriage return before its line feed is part of the line).
			h.bodyAt, h.lines = pos, line
			return h, nil
		default:
			return nil, errors.New(TagPLYInvalid, "a line of the header is not known", "line", line, "reason", "unknown header line", "keyword", string(key))
		}
	}
}
