package threemf

import (
	"encoding/xml"
	"math"
	"strconv"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/internal/vtx"
)

// readObject reads an object of the resources: a mesh or components.
func (d *decoder) readObject(s xml.StartElement) error {
	id, err := d.id(s)
	if err != nil {
		return err
	}
	name, _ := attr(s, "name")
	obj := &object{id: id, name: name, depth: 1}
	pid, pindex := int32(-1), int32(0)
	if text, ok := attr(s, "pid"); ok {
		v, err := strconv.ParseUint(text, 10, 31)
		line, col := d.dec.InputPos()
		if err != nil {
			return errors.New(TagThreeMFInvalid, "pid is a whole number", "reason", "bad id", "part", d.part, "pid", text, "line", line, "column", col)
		}
		pid = int32(v)
		if text, ok := attr(s, "pindex"); ok {
			v, err := strconv.ParseUint(text, 10, 31)
			if err != nil {
				return errors.New(TagThreeMFInvalid, "pindex is a whole number", "reason", "bad index", "part", d.part, "pindex", text, "line", line, "column", col)
			}
			pindex = int32(v)
		}
	}
	for {
		tok, err := d.next()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.EndElement:
			if len(obj.meshes) == 0 && len(obj.children) == 0 {
				line, col := d.dec.InputPos()
				return errors.New(TagThreeMFInvalid, "an object has a mesh or components", "reason", "empty object", "part", d.part, "object", id, "line", line, "column", col)
			}
			if len(obj.meshes) > 0 && len(obj.children) > 0 {
				line, col := d.dec.InputPos()
				return errors.New(TagThreeMFInvalid, "an object has a mesh or components, not both", "reason", "mesh and components", "part", d.part, "object", id, "line", line, "column", col)
			}
			d.objects[id] = obj
			return nil
		case xml.StartElement:
			switch {
			case t.Name.Space == coreNS && t.Name.Local == "mesh":
				err = d.readMesh(obj, pid, pindex)
			case t.Name.Space == coreNS && t.Name.Local == "components":
				err = d.readComponents(obj)
			default:
				d.note(t.Name.Local)
				err = d.skip()
			}
			if err != nil {
				return err
			}
		}
	}
}

// readMesh reads the vertices and the triangles of a mesh and makes its meshes of the model.
func (d *decoder) readMesh(obj *object, pid, pindex int32) error {
	var pos []float64
	var tris []triangle
	for {
		tok, err := d.next()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.EndElement:
			if len(pos) == 0 {
				line, col := d.dec.InputPos()
				return errors.New(TagThreeMFInvalid, "a mesh has no vertex", "reason", "no vertices", "part", d.part, "object", obj.id, "line", line, "column", col)
			}
			if len(tris) == 0 {
				line, col := d.dec.InputPos()
				return errors.New(TagThreeMFInvalid, "a mesh has no triangle", "reason", "no triangles", "part", d.part, "object", obj.id, "line", line, "column", col)
			}
			d.makeMeshes(obj, pos, tris)
			return nil
		case xml.StartElement:
			switch {
			case t.Name.Space == coreNS && t.Name.Local == "vertices":
				pos, err = d.readVertices(pos)
			case t.Name.Space == coreNS && t.Name.Local == "triangles":
				tris, err = d.readTriangles(tris, len(pos)/3, pid, pindex)
			default:
				d.note(t.Name.Local)
				err = d.skip()
			}
			if err != nil {
				return err
			}
		}
	}
}

// readVertices reads the vertices of a mesh, appending their positions.
func (d *decoder) readVertices(pos []float64) ([]float64, error) {
	for {
		tok, err := d.next()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.EndElement:
			return pos, nil
		case xml.StartElement:
			if t.Name.Space != coreNS || t.Name.Local != "vertex" {
				if err := d.skip(); err != nil {
					return nil, err
				}
				continue
			}
			for _, axis := range [3]string{"x", "y", "z"} {
				text, _ := attr(t, axis)
				v, ok := number(text)
				if !ok {
					line, col := d.dec.InputPos()
					return nil, errors.New(TagThreeMFInvalid, "a coordinate of a vertex is a finite number", "reason", "bad number", "part", d.part, "attribute", axis, "value", text, "line", line, "column", col)
				}
				if !d.o.Float64 && math.Abs(v) > math.MaxFloat32 {
					line, col := d.dec.InputPos()
					return nil, errors.New(TagThreeMFInvalid, "a coordinate is beyond the range of float32 (use Float64)", "reason", "out of range", "part", d.part, "attribute", axis, "line", line, "column", col)
				}
				pos = append(pos, v)
			}
			if err := d.skip(); err != nil {
				return nil, err
			}
		}
	}
}

// index reads an optional index attribute: -1 when it is not there.
func index(s xml.StartElement, name string) (int32, bool, bool) {
	text, ok := attr(s, name)
	if !ok {
		return -1, false, true
	}
	v, err := strconv.ParseUint(text, 10, 31)
	return int32(v), true, err == nil
}

// readTriangles reads the triangles of a mesh, appending them with their properties, which are those of the object where a triangle
// has none.
func (d *decoder) readTriangles(tris []triangle, vertices int, objPid, objPindex int32) ([]triangle, error) {
	for {
		tok, err := d.next()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.EndElement:
			return tris, nil
		case xml.StartElement:
			if t.Name.Space != coreNS || t.Name.Local != "triangle" {
				if err := d.skip(); err != nil {
					return nil, err
				}
				continue
			}
			var tr triangle
			for i, name := range [3]string{"v1", "v2", "v3"} {
				v, _, ok := index(t, name)
				if !ok || v < 0 {
					line, col := d.dec.InputPos()
					return nil, errors.New(TagThreeMFInvalid, "a corner of a triangle is a whole number", "reason", "bad index", "part", d.part, "attribute", name, "line", line, "column", col)
				}
				if int(v) >= vertices {
					line, col := d.dec.InputPos()
					return nil, errors.New(TagThreeMFInvalid, "a triangle names a vertex that the mesh does not have", "reason", "index out of range", "part", d.part, "attribute", name, "vertices", vertices, "line", line, "column", col)
				}
				tr.v[i] = v
			}
			pid, hasPid, ok := index(t, "pid")
			if !ok {
				line, col := d.dec.InputPos()
				return nil, errors.New(TagThreeMFInvalid, "pid is a whole number", "reason", "bad id", "part", d.part, "line", line, "column", col)
			}
			first := objPindex
			if !hasPid {
				pid = objPid
			}
			tr.pid = -1
			if pid >= 0 {
				if p1, has, ok := index(t, "p1"); !ok {
					line, col := d.dec.InputPos()
					return nil, errors.New(TagThreeMFInvalid, "p1 is a whole number", "reason", "bad index", "part", d.part, "line", line, "column", col)
				} else if has {
					first = p1
				}
				g, known := d.groups[int(pid)]
				if !known {
					line, col := d.dec.InputPos()
					return nil, errors.New(TagThreeMFInvalid, "a triangle names a group of properties that is not defined before", "reason", "unknown property group", "part", d.part, "pid", int(pid), "line", line, "column", col)
				}
				tr.p = [3]int32{first, first, first}
				for i, name := range [2]string{"p2", "p3"} {
					v, has, ok := index(t, name)
					if !ok {
						line, col := d.dec.InputPos()
						return nil, errors.New(TagThreeMFInvalid, "p2 and p3 are whole numbers", "reason", "bad index", "part", d.part, "attribute", name, "line", line, "column", col)
					}
					if has {
						tr.p[1+i] = v
					}
				}
				if g.kind != kindOther {
					n := len(g.colors)
					for _, p := range tr.p {
						if int(p) >= n {
							line, col := d.dec.InputPos()
							return nil, errors.New(TagThreeMFInvalid, "a triangle names a property that the group does not have", "reason", "index out of range", "part", d.part, "pid", int(pid), "properties", n, "line", line, "column", col)
						}
					}
					tr.pid = pid
				}
			}
			if tr.v[0] == tr.v[1] || tr.v[1] == tr.v[2] || tr.v[0] == tr.v[2] {
				d.dropped++
			} else {
				tris = append(tris, tr)
			}
			if err := d.skip(); err != nil {
				return nil, err
			}
		}
	}
}

// meshBuild is a mesh of the model that is being made: the triangles of one material, with the vertices that they use.
type meshBuild struct {
	key    materialKey
	table  *vtx.Table
	pos    []float64
	colors []byte
	idx    []uint32
}

// makeMeshes makes the meshes of an object: one for each base material that its triangles use, and one for those that use none. A
// vertex is a different vertex of the mesh for each color that a triangle gives it.
func (d *decoder) makeMeshes(obj *object, pos []float64, tris []triangle) {
	none := materialKey{group: -1, index: -1}
	colored := false
	for _, t := range tris {
		if t.pid >= 0 && d.groups[int(t.pid)].kind == kindColor {
			colored = true
			break
		}
	}
	var order []*meshBuild
	byKey := map[materialKey]*meshBuild{}
	for _, t := range tris {
		key := none
		if t.pid >= 0 && d.groups[int(t.pid)].kind == kindBase {
			key = materialKey{group: int(t.pid), index: int(t.p[0])}
		}
		mb := byKey[key]
		if mb == nil {
			mb = &meshBuild{key: key, table: vtx.New(len(tris))}
			byKey[key] = mb
			order = append(order, mb)
		}
		for k := 0; k < 3; k++ {
			vk := vtx.Key{int64(t.v[k]), -1, -1}
			var color [4]uint8
			if colored {
				color = [4]uint8{255, 255, 255, 255}
			}
			if t.pid >= 0 {
				if g := d.groups[int(t.pid)]; g.kind == kindColor {
					vk[1], vk[2] = int64(t.pid), int64(t.p[k])
					color = g.colors[t.p[k]]
				}
			}
			n, added := mb.table.Find(vk)
			if added {
				v := int(t.v[k])
				mb.pos = append(mb.pos, pos[3*v], pos[3*v+1], pos[3*v+2])
				if colored {
					mb.colors = append(mb.colors, color[:]...)
				}
			}
			mb.idx = append(mb.idx, n)
		}
	}
	for _, mb := range order {
		mesh := &model.Mesh{Name: obj.name, Primitive: model.PrimitiveTriangles, Indices: mb.idx, Colors: mb.colors}
		if d.o.Float64 && !exactInFloat32(mb.pos) {
			mesh.Positions64 = mb.pos
		} else {
			mesh.Positions = make([]float32, len(mb.pos))
			for i, v := range mb.pos {
				mesh.Positions[i] = float32(v)
			}
		}
		if mb.key != none {
			mesh.Material = d.material(mb.key)
		}
		obj.meshes = append(obj.meshes, mesh)
		d.out.Meshes = append(d.out.Meshes, mesh)
	}
}

// material is the material of a base material of a group, made the first time.
func (d *decoder) material(k materialKey) *model.Material {
	if m, ok := d.materials[k]; ok {
		return m
	}
	g := d.groups[k.group]
	c := g.colors[k.index]
	m := &model.Material{Name: g.names[k.index], BaseColor: [4]float32{float32(c[0]) / 255, float32(c[1]) / 255, float32(c[2]) / 255, float32(c[3]) / 255}}
	if c[3] < 255 {
		m.AlphaMode = model.AlphaBlend
	}
	d.materials[k] = m
	d.out.Materials = append(d.out.Materials, m)
	return m
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
