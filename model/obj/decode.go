package obj

import (
	"bytes"
	"context"
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/internal/parse"
	"github.com/rm4n0s/glutter/model/internal/poly"
	"github.com/rm4n0s/glutter/model/internal/vtx"
)

// checkEvery is how many lines are read between two looks at the context.
const checkEvery = 1 << 13

// corner is a corner of a face: the position, the texture coordinate and the normal, each counted from 0, or -1 for none.
type corner struct{ v, t, n int32 }

// builderKey says which mesh a face goes to: the node, the material and what the faces are.
type builderKey struct {
	node *model.Node
	mat  int
	prim model.Primitive
}

// builder collects the vertices and indices of one mesh. A vertex is a different (position, texture coordinate, normal) triple.
type builder struct {
	key      builderKey
	tab      *vtx.Table
	pos32    []float32
	pos64    []float64
	uv, nrm  []float32
	col      []byte
	idx      []uint32
	uvN, nrN int // how many vertices have a texture coordinate, and a normal
	noColor  int // how many of the first vertices were made before any position had colors: they are white
}

type decoder struct {
	ctx  context.Context
	opts Options
	m    *model.Model

	// What the file has given so far, to which indices refer.
	pos     []float64 // x, y, z of each position
	colors  []byte    // red, green, blue and alpha of each position, or nil when no position has colors
	uv, nrm []float32

	line int
	// The state of the directives.
	mat      int // the material in use: an index of m.Materials, or -1
	obj, cur *model.Node
	top      []*model.Node
	root     *model.Node // the node of the faces that come before any `o` or `g`
	builders []*builder
	byKey    map[builderKey]*builder
	last     *builder // the builder of the last face, which the next one most likely uses

	materials map[string]int // the index in m.Materials of each material by name
	libs      map[string]bool
	images    map[string]*model.Image
	missingFS bool

	corners []corner
	cpos    []int32
	tris    []int32
	tr      poly.Triangulator
}

func decode(ctx context.Context, data []byte, o Options) (*model.Model, error) {
	d := &decoder{
		ctx: ctx, opts: o, m: &model.Model{Name: o.Name}, mat: -1,
		byKey: map[builderKey]*builder{}, materials: map[string]int{}, libs: map[string]bool{}, images: map[string]*model.Image{},
	}
	if err := d.run(data); err != nil {
		return nil, err
	}
	return d.finish()
}

// run reads the lines.
func (d *decoder) run(data []byte) error {
	var joined []byte
	for off := 0; off < len(data); {
		d.line++
		if d.line%checkEvery == 0 {
			if err := d.ctx.Err(); err != nil {
				return errors.NewErr(model.TagModelReadFailed, err).SetMetadata("line", d.line, "offset", off)
			}
		}
		var text []byte
		if end := bytes.IndexByte(data[off:], '\n'); end < 0 {
			text, off = data[off:], len(data)
		} else {
			text, off = data[off:off+end], off+end+1
		}
		if d.line == 1 && len(text) >= 3 && text[0] == 0xEF && text[1] == 0xBB && text[2] == 0xBF {
			text = text[3:]
		}
		// A backslash at the end of a line joins it with the next.
		if t := parse.Trim(text); len(t) > 0 && t[len(t)-1] == '\\' {
			joined = append(joined[:0], t[:len(t)-1]...)
			for off < len(data) {
				d.line++
				var next []byte
				if end := bytes.IndexByte(data[off:], '\n'); end < 0 {
					next, off = data[off:], len(data)
				} else {
					next, off = data[off:off+end], off+end+1
				}
				next = parse.Trim(next)
				more := len(next) > 0 && next[len(next)-1] == '\\'
				if more {
					next = next[:len(next)-1]
				}
				joined = append(append(joined, ' '), next...)
				if !more {
					break
				}
			}
			text = joined
		}
		word, rest := parse.Field(text)
		if word == nil || word[0] == '#' {
			continue
		}
		var err error
		switch string(word) {
		case "v":
			err = d.vertex(rest)
		case "vt":
			err = d.texCoord(rest)
		case "vn":
			err = d.normal(rest)
		case "f":
			err = d.face(rest)
		case "l":
			err = d.lines(rest)
		case "p":
			err = d.points(rest)
		case "o":
			d.newObject(string(parse.Trim(rest)))
		case "g":
			d.newGroup(string(parse.Trim(rest)))
		case "usemtl":
			d.useMaterial(string(parse.Trim(rest)))
		case "mtllib":
			err = d.materialLibraries(rest)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// numbers reads up to len(out) numbers from the rest of a line and returns how many there were; ok is false when a field is not a
// finite number (field is it).
func numbers(rest []byte, out []float64) (n int, field []byte, ok bool) {
	for n < len(out) {
		var f []byte
		f, rest = parse.Field(rest)
		if f == nil {
			return n, nil, true
		}
		v, isNum := parse.Float(f)
		if !isNum || !parse.Finite(v) {
			return n, f, false
		}
		out[n] = v
		n++
	}
	return n, nil, true
}

func (d *decoder) vertex(rest []byte) error {
	var v [6]float64
	n, bad, ok := numbers(rest, v[:])
	if !ok {
		return errors.New(TagOBJInvalid, "a position has a field that is not a finite number", "line", d.line, "field", string(bad), "reason", "not a number")
	}
	if n < 3 {
		return errors.New(TagOBJInvalid, "a position has fewer than three coordinates", "line", d.line, "count", n, "reason", "short vertex")
	}
	if !d.opts.Float64 && (math.Abs(v[0]) > math.MaxFloat32 || math.Abs(v[1]) > math.MaxFloat32 || math.Abs(v[2]) > math.MaxFloat32) {
		return errors.New(TagOBJInvalid, "a position is beyond the range of float32 (use Float64)", "line", d.line, "reason", "out of range")
	}
	if len(d.pos)/3 >= math.MaxInt32 {
		return errors.New(TagOBJInvalid, "the file has too many positions", "line", d.line, "reason", "too many")
	}
	if n == 4 {
		// x y z w: the weight is of no use here.
		n = 3
	}
	hasColor := n >= 6
	if hasColor && d.colors == nil {
		// The positions before this one have none: they are white.
		d.colors = bytes.Repeat([]byte{255, 255, 255, 255}, len(d.pos)/3)
	}
	d.pos = append(d.pos, v[0], v[1], v[2])
	if d.colors != nil {
		if hasColor {
			c := func(f float64) byte { return byte(math.Max(0, math.Min(1, f))*255 + 0.5) }
			d.colors = append(d.colors, c(v[3]), c(v[4]), c(v[5]), 255)
		} else {
			d.colors = append(d.colors, 255, 255, 255, 255)
		}
	}
	return nil
}

func (d *decoder) texCoord(rest []byte) error {
	var v [3]float64
	n, bad, ok := numbers(rest, v[:])
	if !ok {
		return errors.New(TagOBJInvalid, "a texture coordinate has a field that is not a finite number", "line", d.line, "field", string(bad), "reason", "not a number")
	}
	if n < 1 {
		return errors.New(TagOBJInvalid, "a texture coordinate has no number", "line", d.line, "reason", "short texture coordinate")
	}
	// v counts from the bottom in OBJ and from the top in the model.
	d.uv = append(d.uv, float32(v[0]), float32(1-v[1]))
	return nil
}

func (d *decoder) normal(rest []byte) error {
	var v [3]float64
	n, bad, ok := numbers(rest, v[:])
	if !ok {
		return errors.New(TagOBJInvalid, "a normal has a field that is not a finite number", "line", d.line, "field", string(bad), "reason", "not a number")
	}
	if n < 3 {
		return errors.New(TagOBJInvalid, "a normal has fewer than three numbers", "line", d.line, "count", n, "reason", "short normal")
	}
	if l := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]); l > 0 {
		v[0], v[1], v[2] = v[0]/l, v[1]/l, v[2]/l
	}
	d.nrm = append(d.nrm, float32(v[0]), float32(v[1]), float32(v[2]))
	return nil
}

// index turns the number of a reference into an index from 0 by the count of what has been read: a negative number counts back
// from the last. The reason is empty when it is right.
func index(ref, count int) (i int32, reason string) {
	switch {
	case ref > 0 && ref <= count:
		return int32(ref - 1), ""
	case ref < 0 && -ref <= count:
		return int32(count + ref), ""
	case ref == 0:
		return 0, "zero index"
	}
	return 0, "index out of range"
}

// reference reads a field of a face, line or point: v, v/t, v//n or v/t/n. t and n are -1 when there are none.
func (d *decoder) reference(field []byte) (c corner, reason string) {
	c = corner{t: -1, n: -1}
	var parts [3][]byte
	np := 0
	for np < 3 {
		i := bytes.IndexByte(field, '/')
		if i < 0 {
			parts[np], field = field, nil
			np++
			break
		}
		parts[np], field = field[:i], field[i+1:]
		np++
	}
	if field != nil {
		return c, "too many parts"
	}
	ref, ok := parse.Int(parts[0])
	if !ok {
		return c, "not an index"
	}
	if c.v, reason = index(ref, len(d.pos)/3); reason != "" {
		return c, reason
	}
	if np > 1 && len(parts[1]) > 0 {
		if ref, ok = parse.Int(parts[1]); !ok {
			return c, "not an index"
		}
		if c.t, reason = index(ref, len(d.uv)/2); reason != "" {
			return c, reason
		}
	}
	if np > 2 && len(parts[2]) > 0 {
		if ref, ok = parse.Int(parts[2]); !ok {
			return c, "not an index"
		}
		if c.n, reason = index(ref, len(d.nrm)/3); reason != "" {
			return c, reason
		}
	}
	return c, ""
}

func (d *decoder) face(rest []byte) error {
	d.corners = d.corners[:0]
	for {
		var f []byte
		f, rest = parse.Field(rest)
		if f == nil {
			break
		}
		c, reason := d.reference(f)
		if reason != "" {
			return errors.New(TagOBJInvalid, "a face has a corner that is wrong", "line", d.line, "field", string(f), "reason", reason)
		}
		d.corners = append(d.corners, c)
	}
	n := len(d.corners)
	if n < 3 {
		return errors.New(TagOBJInvalid, "a face has fewer than three corners", "line", d.line, "count", n, "reason", "short face")
	}
	b := d.builderFor(model.PrimitiveTriangles)
	if n == 3 {
		for _, c := range d.corners {
			b.idx = append(b.idx, d.add(b, c))
		}
		return nil
	}
	d.cpos = d.cpos[:0]
	for _, c := range d.corners {
		d.cpos = append(d.cpos, c.v)
	}
	d.tris = d.tr.Cut(d.pos, d.cpos, d.tris[:0])
	for _, k := range d.tris {
		b.idx = append(b.idx, d.add(b, d.corners[k]))
	}
	return nil
}

func (d *decoder) lines(rest []byte) error {
	d.corners = d.corners[:0]
	for {
		var f []byte
		f, rest = parse.Field(rest)
		if f == nil {
			break
		}
		c, reason := d.reference(f)
		if reason != "" {
			return errors.New(TagOBJInvalid, "a line has a point that is wrong", "line", d.line, "field", string(f), "reason", reason)
		}
		c.t, c.n = -1, -1
		d.corners = append(d.corners, c)
	}
	if len(d.corners) < 2 {
		return errors.New(TagOBJInvalid, "a line has fewer than two points", "line", d.line, "count", len(d.corners), "reason", "short line")
	}
	b := d.builderFor(model.PrimitiveLines)
	for i := 0; i+1 < len(d.corners); i++ {
		b.idx = append(b.idx, d.add(b, d.corners[i]), d.add(b, d.corners[i+1]))
	}
	return nil
}

func (d *decoder) points(rest []byte) error {
	d.corners = d.corners[:0]
	for {
		var f []byte
		f, rest = parse.Field(rest)
		if f == nil {
			break
		}
		c, reason := d.reference(f)
		if reason != "" {
			return errors.New(TagOBJInvalid, "a point is wrong", "line", d.line, "field", string(f), "reason", reason)
		}
		c.t, c.n = -1, -1
		d.corners = append(d.corners, c)
	}
	if len(d.corners) == 0 {
		return errors.New(TagOBJInvalid, "a point statement has no point", "line", d.line, "reason", "short points")
	}
	b := d.builderFor(model.PrimitivePoints)
	for _, c := range d.corners {
		b.idx = append(b.idx, d.add(b, c))
	}
	return nil
}

// builderFor is the mesh that the next faces, lines or points go to: the one of the current node and material.
func (d *decoder) builderFor(prim model.Primitive) *builder {
	node := d.cur
	if node == nil {
		if d.root == nil {
			d.root = model.NewNode(d.opts.Name)
			d.top = append(d.top, d.root)
		}
		node = d.root
	}
	key := builderKey{node, d.mat, prim}
	if d.last != nil && d.last.key == key {
		return d.last
	}
	b := d.byKey[key]
	if b == nil {
		b = &builder{key: key, tab: vtx.New(256)}
		d.byKey[key] = b
		d.builders = append(d.builders, b)
	}
	d.last = b
	return b
}

// add returns the number of the vertex of the builder for the corner, making it if the builder has not seen it.
func (d *decoder) add(b *builder, c corner) uint32 {
	n, added := b.tab.Find(vtx.Key{int64(c.v), int64(c.t), int64(c.n)})
	if !added {
		return n
	}
	p := int(c.v) * 3
	if d.opts.Float64 {
		b.pos64 = append(b.pos64, d.pos[p], d.pos[p+1], d.pos[p+2])
	} else {
		b.pos32 = append(b.pos32, float32(d.pos[p]), float32(d.pos[p+1]), float32(d.pos[p+2]))
	}
	if b.key.prim == model.PrimitiveTriangles {
		if c.t >= 0 {
			b.uv = append(b.uv, d.uv[2*c.t], d.uv[2*c.t+1])
			b.uvN++
		} else {
			b.uv = append(b.uv, 0, 0)
		}
		if c.n >= 0 {
			b.nrm = append(b.nrm, d.nrm[3*c.n], d.nrm[3*c.n+1], d.nrm[3*c.n+2])
			b.nrN++
		} else {
			b.nrm = append(b.nrm, 0, 0, 0)
		}
	}
	if d.colors != nil {
		b.col = append(b.col, d.colors[4*c.v:4*c.v+4]...)
	} else {
		b.noColor++
	}
	return n
}

func (d *decoder) newObject(name string) {
	n := model.NewNode(name)
	d.top = append(d.top, n)
	d.obj, d.cur = n, n
}

func (d *decoder) newGroup(name string) {
	n := model.NewNode(name)
	if d.obj != nil {
		d.obj.Children = append(d.obj.Children, n)
	} else {
		d.top = append(d.top, n)
	}
	d.cur = n
}

func (d *decoder) useMaterial(name string) {
	i, ok := d.materials[name]
	if !ok {
		// A material that no library gave: the default look, with its name.
		i = d.addMaterial(&model.Material{Name: name})
	}
	d.mat = i
}

// addMaterial puts a material in the model and returns its index. A second material with a name replaces the first for the faces
// that come after it.
func (d *decoder) addMaterial(m *model.Material) int {
	d.m.Materials = append(d.m.Materials, m)
	d.materials[m.Name] = len(d.m.Materials) - 1
	return len(d.m.Materials) - 1
}

// finish makes the meshes and the tree.
func (d *decoder) finish() (*model.Model, error) {
	for _, b := range d.builders {
		mesh := d.mesh(b)
		d.m.Meshes = append(d.m.Meshes, mesh)
		b.key.node.Meshes = append(b.key.node.Meshes, mesh)
	}
	var prune func(nodes []*model.Node) []*model.Node
	prune = func(nodes []*model.Node) []*model.Node {
		kept := nodes[:0]
		for _, n := range nodes {
			n.Children = prune(n.Children)
			if len(n.Meshes) > 0 || len(n.Children) > 0 {
				kept = append(kept, n)
			}
		}
		return kept
	}
	d.m.Nodes = prune(d.top)
	if len(d.m.Meshes) == 0 {
		return nil, errors.New(TagOBJInvalid, "the file has no faces, lines or points", "line", d.line, "reason", "empty")
	}
	return d.m, nil
}

func (d *decoder) mesh(b *builder) *model.Mesh {
	mesh := &model.Mesh{Name: b.key.node.Name, Primitive: b.key.prim, Indices: b.idx, Colors: b.col}
	if len(b.col) > 0 && b.noColor > 0 {
		// The vertices made before any position had colors come first, and are white.
		mesh.Colors = append(bytes.Repeat([]byte{255}, 4*b.noColor), b.col...)
	}
	if b.key.mat >= 0 {
		mesh.Material = d.m.Materials[b.key.mat]
	}
	n := 0
	if d.opts.Float64 {
		n = len(b.pos64) / 3
		lossy := false
		for _, v := range b.pos64 {
			if float64(float32(v)) != v {
				lossy = true
				break
			}
		}
		if lossy {
			mesh.Positions64 = b.pos64
		} else {
			mesh.Positions = make([]float32, len(b.pos64))
			for i, v := range b.pos64 {
				mesh.Positions[i] = float32(v)
			}
		}
	} else {
		n = len(b.pos32) / 3
		mesh.Positions = b.pos32
	}
	if b.key.prim == model.PrimitiveTriangles {
		switch {
		case b.uvN == n:
			mesh.TexCoords = b.uv
		case b.uvN > 0:
			d.opts.logger().Warn("obj: some corners of a mesh have no texture coordinate, so none is kept", "name", mesh.Name, "with", b.uvN, "vertices", n)
		}
		switch {
		case b.nrN == n:
			mesh.Normals = b.nrm
		case b.nrN > 0:
			d.opts.logger().Warn("obj: some corners of a mesh have no normal, so none is kept", "name", mesh.Name, "with", b.nrN, "vertices", n)
		}
	}
	return mesh
}
