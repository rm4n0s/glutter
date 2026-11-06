package stl

import (
	"bytes"
	"context"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/internal/parse"
)

func decodeASCII(ctx context.Context, data []byte, o Options) (*model.Model, error) {
	if o.Float64 {
		return parseASCII[float64](ctx, data, o)
	}
	return parseASCII[float32](ctx, data, o)
}

// kw says whether the field is the keyword, in any case.
func kw(field []byte, word string) bool {
	if len(field) != len(word) {
		return false
	}
	for i := 0; i < len(word); i++ {
		c := field[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != word[i] {
			return false
		}
	}
	return true
}

// ascii states: what the next line has to be.
const (
	stOutside  = iota // outside a solid
	stSolid           // in a solid, before or between facets
	stFacet           // after "facet normal": "outer loop"
	stLoop            // after "outer loop": three vertices, then "endloop"
	stEndFacet        // after "endloop": "endfacet"
)

// parseASCII reads the ASCII file into one soup for each solid.
func parseASCII[T float32 | float64](ctx context.Context, data []byte, o Options) (*model.Model, error) {
	m := newModel(o.Name)
	var (
		s      *soup[T]
		name   string
		state  = stOutside
		verts  int
		normal [3]float32
		line   = 0
		off    = 0
	)
	finish := func() error {
		mesh, err := build(ctx, s, o)
		if err != nil {
			return err
		}
		mesh.Name = name
		addMesh(m, mesh)
		s = nil
		return nil
	}
	for off < len(data) {
		line++
		if line%checkEvery == 0 {
			if err := ctx.Err(); err != nil {
				return nil, errors.NewErr(model.TagModelReadFailed, err).SetMetadata("line", line, "offset", off)
			}
		}
		end := bytes.IndexByte(data[off:], '\n')
		var text []byte
		start := off
		if end < 0 {
			text, off = data[off:], len(data)
		} else {
			text, off = data[off:off+end], off+end+1
		}
		if line == 1 && len(text) >= 3 && text[0] == 0xEF && text[1] == 0xBB && text[2] == 0xBF {
			text = text[3:]
		}
		word, rest := parse.Field(text)
		if word == nil {
			continue
		}
		switch {
		case kw(word, "solid"):
			if state != stOutside {
				return nil, errors.New(TagSTLInvalid, "solid inside a solid", "line", line, "offset", start, "reason", "nesting")
			}
			s = &soup[T]{}
			name = string(parse.Trim(rest))
			if name == "" {
				name = o.Name
			}
			state = stSolid
		case kw(word, "facet"):
			if state != stSolid {
				return nil, errors.New(TagSTLInvalid, "facet where none can be", "line", line, "offset", start, "reason", "unexpected facet")
			}
			next, rest := parse.Field(rest)
			if !kw(next, "normal") {
				return nil, errors.New(TagSTLInvalid, "facet is not followed by normal", "line", line, "offset", start, "reason", "facet normal")
			}
			for i := 0; i < 3; i++ {
				var f []byte
				f, rest = parse.Field(rest)
				v, ok := parse.Float(f)
				if !ok {
					return nil, errors.New(TagSTLInvalid, "the normal of a facet has a field that is not a number",
						"line", line, "offset", start, "field", string(f), "reason", "not a number")
				}
				if !parse.Finite(v) {
					v = 0
				}
				normal[i] = float32(v)
			}
			state = stFacet
		case kw(word, "outer"):
			if next, _ := parse.Field(rest); state != stFacet || !kw(next, "loop") {
				return nil, errors.New(TagSTLInvalid, "outer loop where none can be", "line", line, "offset", start, "reason", "unexpected loop")
			}
			state, verts = stLoop, 0
		case kw(word, "vertex"):
			if state != stLoop || verts == 3 {
				return nil, errors.New(TagSTLInvalid, "vertex where none can be", "line", line, "offset", start, "reason", "unexpected vertex")
			}
			for i := 0; i < 3; i++ {
				var f []byte
				f, rest = parse.Field(rest)
				v, ok := parse.Float(f)
				if !ok {
					return nil, errors.New(TagSTLInvalid, "a vertex has a field that is not a number",
						"line", line, "offset", start, "field", string(f), "reason", "not a number")
				}
				if !parse.Finite(v) {
					return nil, errors.New(TagSTLInvalid, "a coordinate is not a finite number",
						"line", line, "offset", start, "field", string(f), "reason", "not finite")
				}
				if !parse.Finite(float64(T(v))) {
					return nil, errors.New(TagSTLInvalid, "a coordinate is beyond the range of the numbers that the mesh holds (use Float64)",
						"line", line, "offset", start, "field", string(f), "reason", "out of range")
				}
				s.pos = append(s.pos, T(v))
			}
			verts++
		case kw(word, "endloop"):
			if state != stLoop || verts != 3 {
				return nil, errors.New(TagSTLInvalid, "endloop of a loop that has not three vertices",
					"line", line, "offset", start, "vertices", verts, "reason", "vertex count")
			}
			state = stEndFacet
		case kw(word, "endfacet"):
			if state != stEndFacet {
				return nil, errors.New(TagSTLInvalid, "endfacet where none can be", "line", line, "offset", start, "reason", "unexpected endfacet")
			}
			s.norm = append(s.norm, normal[0], normal[1], normal[2])
			state = stSolid
		case kw(word, "endsolid"):
			if state != stSolid {
				return nil, errors.New(TagSTLInvalid, "endsolid inside a facet", "line", line, "offset", start, "reason", "unexpected endsolid")
			}
			if err := finish(); err != nil {
				return nil, err
			}
			state = stOutside
		default:
			return nil, errors.New(TagSTLInvalid, "a line that is not STL",
				"line", line, "offset", start, "field", string(word), "reason", "unknown keyword")
		}
	}
	switch state {
	case stSolid:
		// A file cut after a whole facet, or one that has no endsolid: what was read is the solid.
		if err := finish(); err != nil {
			return nil, err
		}
	case stFacet, stLoop, stEndFacet:
		return nil, errors.New(TagSTLInvalid, "the file ends inside a facet", "line", line, "offset", len(data), "reason", "truncated")
	}
	if len(m.Nodes) == 0 {
		return nil, errors.New(TagSTLInvalid, "the file has no solid", "line", line, "offset", len(data), "reason", "empty")
	}
	return m, nil
}
