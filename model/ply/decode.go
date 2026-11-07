package ply

import (
	"context"
	"encoding/binary"
	"math"
	"slices"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/internal/parse"
	"github.com/rm4n0s/glutter/model/internal/poly"
)

// checkEvery is how many records are read between two looks at the context.
const checkEvery = 1 << 14

// cursor reads the values of the body: the tokens of an ASCII file, or the bytes of a binary one.
type cursor struct {
	data   []byte
	pos    int
	line   int
	format format
}

// where is the metadata key and value that place the cursor in the file: the line of an ASCII file, the byte of a binary one.
func (c *cursor) where() (string, int) {
	if c.format == formatASCII {
		return "line", c.line
	}
	return "offset", c.pos
}

func isSeparator(b byte) bool { return b == '\n' || parse.IsSpace(b) }

// token reads the next field of an ASCII body, counting lines; it is empty at the end of the data.
func (c *cursor) token() []byte {
	for c.pos < len(c.data) && isSeparator(c.data[c.pos]) {
		if c.data[c.pos] == '\n' {
			c.line++
		}
		c.pos++
	}
	start := c.pos
	for c.pos < len(c.data) && !isSeparator(c.data[c.pos]) {
		c.pos++
	}
	return c.data[start:c.pos]
}

// value reads one number of the type, as a float64 (every type of PLY is exact in it).
func (c *cursor) value(t scalar) (float64, error) {
	if c.format == formatASCII {
		tok := c.token()
		if len(tok) == 0 {
			return 0, errors.New(TagPLYInvalid, "the body ends before its last value", "line", c.line, "reason", "truncated")
		}
		f, ok := parse.Float(tok)
		if !ok {
			return 0, errors.New(TagPLYInvalid, "a value is not a number", "line", c.line, "reason", "not a number", "field", string(tok[:min(len(tok), 32)]))
		}
		return f, nil
	}
	n := t.size()
	if c.pos+n > len(c.data) {
		return 0, errors.New(TagPLYInvalid, "the body ends before its last value", "offset", c.pos, "reason", "truncated")
	}
	b := c.data[c.pos : c.pos+n]
	c.pos += n
	big := c.format == formatBig
	switch t {
	case typeInt8:
		return float64(int8(b[0])), nil
	case typeUint8:
		return float64(b[0]), nil
	case typeInt16:
		return float64(int16(u16(b, big))), nil
	case typeUint16:
		return float64(u16(b, big)), nil
	case typeInt32:
		return float64(int32(u32(b, big))), nil
	case typeUint32:
		return float64(u32(b, big)), nil
	case typeFloat32:
		return float64(math.Float32frombits(u32(b, big))), nil
	}
	return math.Float64frombits(u64(b, big)), nil
}

func u16(b []byte, big bool) uint16 {
	if big {
		return binary.BigEndian.Uint16(b)
	}
	return binary.LittleEndian.Uint16(b)
}

func u32(b []byte, big bool) uint32 {
	if big {
		return binary.BigEndian.Uint32(b)
	}
	return binary.LittleEndian.Uint32(b)
}

func u64(b []byte, big bool) uint64 {
	if big {
		return binary.BigEndian.Uint64(b)
	}
	return binary.LittleEndian.Uint64(b)
}

// count reads the count of a list: a whole number from 0 to limit.
func (c *cursor) count(t scalar, limit int) (int, error) {
	v, err := c.value(t)
	if err != nil {
		return 0, err
	}
	if v < 0 || v != math.Trunc(v) || v > float64(limit) {
		k, at := c.where()
		return 0, errors.New(TagPLYInvalid, "the count of a list is a whole number that fits the file", k, at, "reason", "bad list count")
	}
	return int(v), nil
}

// skip reads a property and drops it.
func (c *cursor) skip(p *property) error {
	if !p.list {
		_, err := c.value(p.typ)
		return err
	}
	n, err := c.count(p.countType, maxCount)
	if err != nil {
		return err
	}
	if c.format != formatASCII {
		if n > (len(c.data)-c.pos)/p.typ.size() {
			return errors.New(TagPLYInvalid, "the body ends before its last value", "offset", c.pos, "reason", "truncated")
		}
		c.pos += n * p.typ.size()
		return nil
	}
	for i := 0; i < n; i++ {
		if _, err := c.value(p.typ); err != nil {
			return err
		}
	}
	return nil
}

// slot is what a property of the vertex element is for.
type slot uint8

const (
	slotNone slot = iota
	slotX
	slotY
	slotZ
	slotNX
	slotNY
	slotNZ
	slotRed
	slotGreen
	slotBlue
	slotAlpha
	slotU
	slotV
)

// slotOf is the use of a vertex property by its name.
func slotOf(name string) slot {
	switch name {
	case "x":
		return slotX
	case "y":
		return slotY
	case "z":
		return slotZ
	case "nx":
		return slotNX
	case "ny":
		return slotNY
	case "nz":
		return slotNZ
	case "red", "r", "diffuse_red":
		return slotRed
	case "green", "g", "diffuse_green":
		return slotGreen
	case "blue", "b", "diffuse_blue":
		return slotBlue
	case "alpha", "a", "diffuse_alpha":
		return slotAlpha
	case "s", "u", "texture_s", "texture_u":
		return slotU
	case "t", "v", "texture_t", "texture_v":
		return slotV
	}
	return slotNone
}

// decoder reads the body into a mesh.
type decoder struct {
	ctx context.Context
	o   Options
	h   *header
	c   cursor

	// The vertices: the positions as float64 (the faces are cut on them), the normals, the colors and the texture coordinates.
	nv      int
	pos     []float64
	normals []float32
	colors  []byte
	uvs     []float32
	// wide says that a position property is float64, so that the positions may need it.
	wide bool

	// corners are the vertices of every face one after the other, and ends the end of each face in corners; the faces are cut into
	// triangles when the whole file is read, since a file may give its faces before its vertices.
	corners []int32
	ends    []int32
	indices []uint32
	// dropped counts the faces with fewer than three corners.
	dropped int
	tr      poly.Triangulator
	tris    []int32
}

func (d *decoder) run() (*model.Model, error) {
	var vertex, face *element
	for i := range d.h.elements {
		switch e := &d.h.elements[i]; e.name {
		case "vertex":
			vertex = e
		case "face":
			face = e
		}
	}
	if vertex == nil {
		return nil, errors.New(TagPLYInvalid, "the file has no vertex element", "line", d.h.lines, "reason", "no vertex element")
	}
	skipped := map[string]bool{}
	for i := range d.h.elements {
		e := &d.h.elements[i]
		var err error
		switch e {
		case vertex:
			err = d.readVertices(e)
		case face:
			err = d.readFaces(e, vertex.count)
		default:
			skipped[e.name] = true
			err = d.skipElement(e)
		}
		if err != nil {
			return nil, err
		}
	}
	if len(skipped) > 0 {
		names := make([]string, 0, len(skipped))
		for n := range skipped {
			names = append(names, n)
		}
		slices.Sort(names)
		d.o.logger().Info("ply: elements that are not read were skipped", "elements", strings.Join(names, ","))
	}
	return d.finish(face)
}

// skipElement reads the records of an element that the loader does not use.
func (d *decoder) skipElement(e *element) error {
	allFixed := true
	stride := 0
	for i := range e.props {
		if e.props[i].list {
			allFixed = false
			break
		}
		stride += e.props[i].typ.size()
	}
	if d.c.format != formatASCII && allFixed {
		if e.count > 0 && stride > 0 && e.count > (len(d.c.data)-d.c.pos)/stride {
			return errors.New(TagPLYInvalid, "the body ends before its last value", "offset", d.c.pos, "reason", "truncated", "element", e.name)
		}
		d.c.pos += e.count * stride
		return nil
	}
	if d.c.format == formatASCII && len(e.props) > 0 && e.count > (len(d.c.data)-d.c.pos)/minASCIIBytesPerItem {
		return errors.New(TagPLYInvalid, "the body ends before its last value", "line", d.c.line, "reason", "truncated", "element", e.name)
	}
	for i := 0; i < e.count; i++ {
		if i%checkEvery == 0 {
			if err := d.ctx.Err(); err != nil {
				k, at := d.c.where()
				return errors.NewErr(model.TagModelReadFailed, err).SetMetadata(k, at, "element", e.name)
			}
		}
		for j := range e.props {
			if err := d.c.skip(&e.props[j]); err != nil {
				return err
			}
		}
	}
	return nil
}

// plausible says whether a body of the format that is left could hold count records of at least perRecord bytes (binary) or two bytes
// (ASCII): a count that it cannot hold is a lie, and is not allocated for.
func (d *decoder) plausible(count, perRecord int) bool {
	left := len(d.c.data) - d.c.pos
	if d.c.format == formatASCII {
		return count <= left/minASCIIBytesPerItem || count == 0
	}
	return perRecord == 0 || count <= left/perRecord
}

// readVertices reads the vertex element.
func (d *decoder) readVertices(e *element) error {
	slots := make([]slot, len(e.props))
	have := map[slot]int{}
	minBytes := 0
	var ignored []string
	for i := range e.props {
		p := &e.props[i]
		if !p.list {
			slots[i] = slotOf(p.name)
			minBytes += p.typ.size()
		} else {
			minBytes += p.countType.size()
		}
		if slots[i] == slotNone {
			ignored = append(ignored, p.name)
			continue
		}
		have[slots[i]] = i
		if slots[i] <= slotZ && p.typ == typeFloat64 {
			d.wide = true
		}
	}
	if _, ok := have[slotX]; !ok {
		return errors.New(TagPLYInvalid, "the vertices have no x, y and z", "line", e.line, "reason", "no position")
	}
	if _, ok := have[slotY]; !ok {
		return errors.New(TagPLYInvalid, "the vertices have no x, y and z", "line", e.line, "reason", "no position")
	}
	if _, ok := have[slotZ]; !ok {
		return errors.New(TagPLYInvalid, "the vertices have no x, y and z", "line", e.line, "reason", "no position")
	}
	if e.count == 0 {
		return errors.New(TagPLYInvalid, "the file has no vertex", "line", e.line, "reason", "no vertices")
	}
	if !d.plausible(e.count, minBytes) {
		k, at := d.c.where()
		return errors.New(TagPLYInvalid, "the count of the vertices is more than the body can hold", k, at, "reason", "truncated", "count", e.count)
	}
	d.nv = e.count
	d.pos = make([]float64, 3*d.nv)
	_, nx := have[slotNX]
	_, ny := have[slotNY]
	_, nz := have[slotNZ]
	hasNormal := nx && ny && nz
	_, cr := have[slotRed]
	_, cg := have[slotGreen]
	_, cb := have[slotBlue]
	_, ca := have[slotAlpha]
	hasColor := cr && cg && cb
	_, cu := have[slotU]
	_, cv := have[slotV]
	hasUV := cu && cv
	if hasNormal {
		d.normals = make([]float32, 3*d.nv)
	}
	if hasColor {
		d.colors = make([]byte, 4*d.nv)
		for i := 3; i < len(d.colors); i += 4 {
			d.colors[i] = 255
		}
	}
	if hasUV {
		d.uvs = make([]float32, 2*d.nv)
	}
	if (cr || cg || cb || ca) && !hasColor {
		ignored = append(ignored, "colors that are not red, green and blue together")
	}
	if len(ignored) > 0 {
		d.o.logger().Info("ply: vertex properties that are not read were skipped", "properties", strings.Join(ignored, ","))
	}
	for i := 0; i < d.nv; i++ {
		if i%checkEvery == 0 {
			if err := d.ctx.Err(); err != nil {
				k, at := d.c.where()
				return errors.NewErr(model.TagModelReadFailed, err).SetMetadata(k, at, "element", "vertex", "vertex", i)
			}
		}
		for j := range e.props {
			p := &e.props[j]
			if p.list {
				if err := d.c.skip(p); err != nil {
					return err
				}
				continue
			}
			v, err := d.c.value(p.typ)
			if err != nil {
				return err
			}
			s := slots[j]
			switch {
			case s == slotNone:
			case s <= slotZ:
				if math.IsNaN(v) || math.IsInf(v, 0) {
					k, at := d.c.where()
					return errors.New(TagPLYInvalid, "a position is not finite", k, at, "reason", "not finite", "vertex", i)
				}
				d.pos[3*i+int(s-slotX)] = v
			case s <= slotNZ:
				if d.normals == nil {
					continue
				}
				if math.IsNaN(v) || math.IsInf(v, 0) {
					k, at := d.c.where()
					return errors.New(TagPLYInvalid, "a normal is not finite", k, at, "reason", "not finite", "vertex", i)
				}
				d.normals[3*i+int(s-slotNX)] = float32(v)
			case s <= slotAlpha:
				if d.colors != nil {
					d.colors[4*i+int(s-slotRed)] = colorByte(v, p.typ)
				}
			default:
				if d.uvs == nil {
					continue
				}
				if math.IsNaN(v) || math.IsInf(v, 0) {
					k, at := d.c.where()
					return errors.New(TagPLYInvalid, "a texture coordinate is not finite", k, at, "reason", "not finite", "vertex", i)
				}
				if s == slotU {
					d.uvs[2*i] = float32(v)
				} else {
					// The origin of the model's textures is at the top left; PLY's is at the bottom left.
					d.uvs[2*i+1] = float32(1 - v)
				}
			}
		}
	}
	if d.normals != nil && !normalize(d.normals) {
		d.normals = nil
		d.o.logger().Info("ply: the normals were not used because one is of length 0")
	}
	return nil
}

// colorByte makes a byte of a color value of the type: the types of whole numbers hold 0 to 255 (16 bits and 32 bits are scaled
// down), and a float holds 0 to 1.
func colorByte(v float64, t scalar) byte {
	switch t {
	case typeFloat32, typeFloat64:
		v *= 255
	case typeUint16:
		v /= 257
	case typeUint32:
		v /= 16843009
	case typeInt16:
		v = max(v, 0) / 128.5
	case typeInt32:
		v = max(v, 0) / 8421504.5
	}
	return byte(min(max(math.Round(v), 0), 255))
}

// normalize makes every normal of length 1, and says false when one cannot be (its length is 0).
func normalize(n []float32) bool {
	for i := 0; i < len(n); i += 3 {
		x, y, z := float64(n[i]), float64(n[i+1]), float64(n[i+2])
		l := math.Sqrt(x*x + y*y + z*z)
		if l == 0 || math.IsInf(l, 0) || math.IsNaN(l) {
			return false
		}
		n[i], n[i+1], n[i+2] = float32(x/l), float32(y/l), float32(z/l)
	}
	return true
}

// readFaces reads the face element: the corners of each polygon, cut into triangles.
func (d *decoder) readFaces(e *element, vertices int) error {
	index := -1
	minBytes := 0
	var ignored []string
	for i := range e.props {
		p := &e.props[i]
		if p.list && index < 0 && (p.name == "vertex_indices" || p.name == "vertex_index") {
			index = i
		} else {
			ignored = append(ignored, p.name)
		}
		if p.list {
			minBytes += p.countType.size()
		} else {
			minBytes += p.typ.size()
		}
	}
	if e.count == 0 {
		return nil
	}
	if index < 0 {
		return errors.New(TagPLYInvalid, "the faces have no list vertex_indices", "line", e.line, "reason", "no vertex indices")
	}
	if !d.plausible(e.count, minBytes) {
		k, at := d.c.where()
		return errors.New(TagPLYInvalid, "the count of the faces is more than the body can hold", k, at, "reason", "truncated", "count", e.count)
	}
	if len(ignored) > 0 {
		d.o.logger().Info("ply: face properties that are not read were skipped", "properties", strings.Join(ignored, ","))
	}
	d.corners = make([]int32, 0, 3*e.count)
	d.ends = make([]int32, 0, e.count)
	for i := 0; i < e.count; i++ {
		if i%checkEvery == 0 {
			if err := d.ctx.Err(); err != nil {
				k, at := d.c.where()
				return errors.NewErr(model.TagModelReadFailed, err).SetMetadata(k, at, "element", "face", "face", i)
			}
		}
		for j := range e.props {
			p := &e.props[j]
			if j != index {
				if err := d.c.skip(p); err != nil {
					return err
				}
				continue
			}
			n, err := d.c.count(p.countType, maxFaceCorners)
			if err != nil {
				return err
			}
			for k := 0; k < n; k++ {
				v, err := d.c.value(p.typ)
				if err != nil {
					return err
				}
				if v < 0 || v != math.Trunc(v) || v >= float64(vertices) {
					at, num := d.c.where()
					return errors.New(TagPLYInvalid, "a face names a vertex that the file does not have", at, num, "reason", "index out of range", "face", i, "vertices", vertices)
				}
				d.corners = append(d.corners, int32(v))
			}
			d.ends = append(d.ends, int32(len(d.corners)))
		}
	}
	return nil
}

// finish makes the model of what was read: one node with one mesh, of triangles when the file has faces and of points when it has
// none.
func (d *decoder) finish(face *element) (*model.Model, error) {
	mesh := &model.Mesh{Name: d.o.Name, Normals: d.normals, Colors: d.colors, TexCoords: d.uvs}
	if d.wide && !exactInFloat32(d.pos) {
		mesh.Positions64 = d.pos
	} else {
		mesh.Positions = make([]float32, len(d.pos))
		for i, v := range d.pos {
			mesh.Positions[i] = float32(v)
		}
	}
	d.triangulate()
	if d.dropped > 0 {
		d.o.logger().Info("ply: faces with fewer than three corners were dropped", "faces", d.dropped)
	}
	switch {
	case face == nil || face.count == 0:
		mesh.Primitive = model.PrimitivePoints
		mesh.Indices = make([]uint32, d.nv)
		for i := range mesh.Indices {
			mesh.Indices[i] = uint32(i)
		}
	case len(d.indices) == 0:
		return nil, errors.New(TagPLYInvalid, "no face has three corners", "line", face.line, "reason", "no triangles")
	default:
		mesh.Primitive = model.PrimitiveTriangles
		mesh.Indices = d.indices
	}
	m := &model.Model{Name: d.o.Name}
	node := model.NewNode(d.o.Name)
	node.Meshes = []*model.Mesh{mesh}
	m.Meshes = []*model.Mesh{mesh}
	m.Nodes = []*model.Node{node}
	return m, nil
}

// triangulate makes the triangles of the faces: a triangle as it is, a polygon cut by the triangulator on the positions, and a
// face of fewer than three corners dropped.
func (d *decoder) triangulate() {
	d.indices = make([]uint32, 0, len(d.corners))
	start := int32(0)
	for _, end := range d.ends {
		face := d.corners[start:end]
		start = end
		switch {
		case len(face) < 3:
			d.dropped++
		case len(face) == 3:
			d.indices = append(d.indices, uint32(face[0]), uint32(face[1]), uint32(face[2]))
		default:
			d.tris = d.tr.Cut(d.pos, face, d.tris[:0])
			for _, c := range d.tris {
				d.indices = append(d.indices, uint32(face[c]))
			}
		}
	}
}

// exactInFloat32 says whether every number is the same in float32.
func exactInFloat32(v []float64) bool {
	for _, x := range v {
		if float64(float32(x)) != x {
			return false
		}
	}
	return true
}
