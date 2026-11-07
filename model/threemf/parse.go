package threemf

import (
	"bufio"
	"context"
	"encoding/xml"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/vecmath"
)

// The namespaces of the model part.
const (
	coreNS       = "http://schemas.microsoft.com/3dmanufacturing/core/2015/02"
	materialNS   = "http://schemas.microsoft.com/3dmanufacturing/material/2015/02"
	productionNS = "http://schemas.microsoft.com/3dmanufacturing/production/2015/06"
)

// checkEvery is how many tokens are read between two looks at the context.
const checkEvery = 1 << 14

// propertyKind is what a group of properties holds.
type propertyKind uint8

const (
	// kindOther is a group that the loader does not read (textures, composites, multiproperties).
	kindOther propertyKind = iota
	kindBase
	kindColor
)

// group is a group of properties of the resources: the base materials, or the colors of a color group.
type group struct {
	kind   propertyKind
	names  []string
	colors [][4]uint8
}

// triangle is a triangle of a mesh with its properties (pid is -1 for none).
type triangle struct {
	v   [3]int32
	pid int32
	p   [3]int32
}

// object is an object of the resources: a mesh, already made into meshes of the model, or components.
type object struct {
	id       int
	name     string
	meshes   []*model.Mesh
	children []component
	// depth is the number of levels of components under the object, counting itself.
	depth int
}

// component is a use of an object by another.
type component struct {
	object    *object
	transform vecmath.Mat4
}

// materialKey tells the materials of the model apart: a base material of a group.
type materialKey struct {
	group, index int
}

// decoder reads the model part.
type decoder struct {
	ctx  context.Context
	o    Options
	part string
	dec  *xml.Decoder
	// reader is the part's reader; it is read one byte past the limit, and when that byte is read the part is too large.
	reader *io.LimitedReader

	tokens int
	units  model.Units
	// ids are the ids of the resources (objects and property groups share them).
	ids       map[int]bool
	groups    map[int]*group
	objects   map[int]*object
	materials map[materialKey]*model.Material
	out       *model.Model
	// notes are the names of the elements and extensions that are left out, for one line of the log.
	notes map[string]bool
	// nodes counts the nodes that the build made, and dropped the triangles that had a repeated vertex.
	nodes   int
	dropped int
	built   bool
}

// run reads the XML of the model part.
func (d *decoder) run(r *io.LimitedReader) (*model.Model, error) {
	d.reader = r
	d.dec = xml.NewDecoder(bufio.NewReaderSize(r, 1<<16))
	d.ids, d.groups, d.objects = map[int]bool{}, map[int]*group{}, map[int]*object{}
	d.materials, d.notes = map[materialKey]*model.Material{}, map[string]bool{}
	d.out = &model.Model{Name: d.o.Name}
	d.units = model.Units{Name: "mm", MetersPerUnit: 0.001}
	var root xml.StartElement
	for {
		tok, err := d.next()
		if err != nil {
			return nil, err
		}
		if start, ok := tok.(xml.StartElement); ok {
			root = start
			break
		}
	}
	if root.Name.Space != coreNS || root.Name.Local != "model" {
		return nil, errors.New(TagThreeMFInvalid, "the root of the model part is not a model of the 3MF core namespace", "reason", "not a 3mf model", "part", d.part, "element", root.Name.Local)
	}
	if err := d.readModel(root); err != nil {
		return nil, err
	}
	if !d.built {
		return nil, errors.New(TagThreeMFInvalid, "the model has no build with an item", "reason", "no items", "part", d.part)
	}
	d.out.Units = d.units
	if len(d.notes) > 0 {
		names := make([]string, 0, len(d.notes))
		for n := range d.notes {
			names = append(names, n)
		}
		sort.Strings(names)
		d.o.logger().Info("3mf: parts that are not read were left out", "parts", strings.Join(names, ","))
	}
	if d.dropped > 0 {
		d.o.logger().Info("3mf: triangles that repeat a vertex were dropped", "triangles", d.dropped)
	}
	return d.out, nil
}

// next reads the next token of the XML, looking at the context now and then. A clean end of the XML is an error too: the callers read
// up to the end of an element, which the decoder checks is there.
func (d *decoder) next() (xml.Token, error) {
	if d.tokens++; d.tokens%checkEvery == 0 {
		if err := d.ctx.Err(); err != nil {
			line, col := d.dec.InputPos()
			return nil, errors.NewErr(model.TagModelReadFailed, err).SetMetadata("part", d.part, "line", line, "column", col)
		}
	}
	tok, err := d.dec.Token()
	if d.reader.N <= 0 {
		line, col := d.dec.InputPos()
		return nil, errors.New(TagThreeMFInvalid, "the model part is larger than the limit", "reason", "model part too large", "part", d.part, "line", line, "column", col)
	}
	if err == io.EOF {
		line, col := d.dec.InputPos()
		return nil, errors.New(TagThreeMFInvalid, "the XML ends before the model does", "reason", "truncated", "part", d.part, "line", line, "column", col)
	}
	if err != nil {
		line, col := d.dec.InputPos()
		return nil, errors.NewErr(TagThreeMFInvalid, err).SetMetadata("reason", "bad xml", "part", d.part, "line", line, "column", col)
	}
	return tok, nil
}

// skip reads up to the end of the element that was just started, whatever is inside it.
func (d *decoder) skip() error {
	err := d.dec.Skip()
	if d.reader.N <= 0 {
		line, col := d.dec.InputPos()
		return errors.New(TagThreeMFInvalid, "the model part is larger than the limit", "reason", "model part too large", "part", d.part, "line", line, "column", col)
	}
	if err != nil {
		line, col := d.dec.InputPos()
		return errors.NewErr(TagThreeMFInvalid, err).SetMetadata("reason", "bad xml", "part", d.part, "line", line, "column", col)
	}
	return nil
}

// note remembers an element or extension that is left out.
func (d *decoder) note(name string) { d.notes[name] = true }

// attr is the value of an attribute without a namespace, and whether it is there.
func attr(s xml.StartElement, name string) (string, bool) {
	for _, a := range s.Attr {
		if a.Name.Space == "" && a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

// readModel reads the children of the model element.
func (d *decoder) readModel(root xml.StartElement) error {
	if unit, ok := attr(root, "unit"); ok {
		u, ok := unitOf(unit)
		if !ok {
			line, col := d.dec.InputPos()
			return errors.New(TagThreeMFInvalid, "the unit is not one of 3MF's", "reason", "bad unit", "part", d.part, "unit", unit, "line", line, "column", col)
		}
		d.units = u
	}
	if req, ok := attr(root, "requiredextensions"); ok {
		spaces := map[string]string{}
		for _, a := range root.Attr {
			if a.Name.Space == "xmlns" {
				spaces[a.Name.Local] = a.Value
			}
		}
		for _, prefix := range strings.Fields(req) {
			ns := spaces[prefix]
			if ns != materialNS {
				line, col := d.dec.InputPos()
				return errors.New(TagThreeMFInvalid, "the model requires an extension that is not read", "reason", "required extension not read", "part", d.part, "extension", ns, "prefix", prefix, "line", line, "column", col)
			}
		}
	}
	for {
		tok, err := d.next()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.EndElement:
			return nil
		case xml.StartElement:
			switch {
			case t.Name.Space == coreNS && t.Name.Local == "resources":
				err = d.readResources()
			case t.Name.Space == coreNS && t.Name.Local == "build":
				err = d.readBuild()
			default:
				err = d.skip()
			}
			if err != nil {
				return err
			}
		}
	}
}

// unitOf is the unit that a name of 3MF is.
func unitOf(name string) (model.Units, bool) {
	switch name {
	case "micron":
		return model.Units{Name: "um", MetersPerUnit: 1e-6}, true
	case "millimeter":
		return model.Units{Name: "mm", MetersPerUnit: 1e-3}, true
	case "centimeter":
		return model.Units{Name: "cm", MetersPerUnit: 1e-2}, true
	case "inch":
		return model.Units{Name: "in", MetersPerUnit: 0.0254}, true
	case "foot":
		return model.Units{Name: "ft", MetersPerUnit: 0.3048}, true
	case "meter":
		return model.Units{Name: "m", MetersPerUnit: 1}, true
	}
	return model.Units{}, false
}

// id reads the id attribute of a resource: a whole number from 1 that no other resource has.
func (d *decoder) id(s xml.StartElement) (int, error) {
	text, ok := attr(s, "id")
	line, col := d.dec.InputPos()
	if !ok {
		return 0, errors.New(TagThreeMFInvalid, "a resource has no id", "reason", "no id", "part", d.part, "element", s.Name.Local, "line", line, "column", col)
	}
	v, err := strconv.ParseUint(text, 10, 31)
	if err != nil || v == 0 {
		return 0, errors.New(TagThreeMFInvalid, "the id of a resource is a whole number from 1", "reason", "bad id", "part", d.part, "id", text, "line", line, "column", col)
	}
	if d.ids[int(v)] {
		return 0, errors.New(TagThreeMFInvalid, "two resources have the same id", "reason", "id twice", "part", d.part, "id", text, "line", line, "column", col)
	}
	d.ids[int(v)] = true
	return int(v), nil
}

// readResources reads the property groups and the objects.
func (d *decoder) readResources() error {
	for {
		tok, err := d.next()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.EndElement:
			return nil
		case xml.StartElement:
			switch {
			case t.Name.Space == coreNS && t.Name.Local == "basematerials":
				err = d.readBaseMaterials(t)
			case t.Name.Space == materialNS && t.Name.Local == "colorgroup":
				err = d.readColorGroup(t)
			case t.Name.Space == coreNS && t.Name.Local == "object":
				err = d.readObject(t)
			case t.Name.Space == materialNS:
				// Textures, composites and multiproperties: the group is known by its id, and a triangle that uses it has no property.
				d.note(t.Name.Local)
				if _, ok := attr(t, "id"); ok {
					var id int
					if id, err = d.id(t); err == nil {
						d.groups[id] = &group{kind: kindOther}
						err = d.skip()
					}
				} else {
					err = d.skip()
				}
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

// readBaseMaterials reads a group of base materials: a name and a display color each.
func (d *decoder) readBaseMaterials(s xml.StartElement) error {
	id, err := d.id(s)
	if err != nil {
		return err
	}
	g := &group{kind: kindBase}
	for {
		tok, err := d.next()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.EndElement:
			if len(g.colors) == 0 {
				line, col := d.dec.InputPos()
				return errors.New(TagThreeMFInvalid, "a group of base materials has no material", "reason", "empty group", "part", d.part, "line", line, "column", col)
			}
			d.groups[id] = g
			return nil
		case xml.StartElement:
			if t.Name.Space != coreNS || t.Name.Local != "base" {
				if err := d.skip(); err != nil {
					return err
				}
				continue
			}
			name, _ := attr(t, "name")
			text, ok := attr(t, "displaycolor")
			c, valid := parseColor(text)
			if !ok || !valid {
				line, col := d.dec.InputPos()
				return errors.New(TagThreeMFInvalid, "the display color of a base material is #RRGGBB or #RRGGBBAA", "reason", "bad color", "part", d.part, "color", text, "line", line, "column", col)
			}
			g.names, g.colors = append(g.names, name), append(g.colors, c)
			if err := d.skip(); err != nil {
				return err
			}
		}
	}
}

// readColorGroup reads a group of colors of the materials extension.
func (d *decoder) readColorGroup(s xml.StartElement) error {
	id, err := d.id(s)
	if err != nil {
		return err
	}
	g := &group{kind: kindColor}
	for {
		tok, err := d.next()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.EndElement:
			if len(g.colors) == 0 {
				line, col := d.dec.InputPos()
				return errors.New(TagThreeMFInvalid, "a color group has no color", "reason", "empty group", "part", d.part, "line", line, "column", col)
			}
			d.groups[id] = g
			return nil
		case xml.StartElement:
			if t.Name.Space != materialNS || t.Name.Local != "color" {
				if err := d.skip(); err != nil {
					return err
				}
				continue
			}
			text, ok := attr(t, "color")
			c, valid := parseColor(text)
			if !ok || !valid {
				line, col := d.dec.InputPos()
				return errors.New(TagThreeMFInvalid, "the color of a color group is #RRGGBB or #RRGGBBAA", "reason", "bad color", "part", d.part, "color", text, "line", line, "column", col)
			}
			g.colors = append(g.colors, c)
			if err := d.skip(); err != nil {
				return err
			}
		}
	}
}

// parseColor reads #RRGGBB or #RRGGBBAA (sRGB), as the bytes of a color.
func parseColor(text string) ([4]uint8, bool) {
	c := [4]uint8{0, 0, 0, 255}
	if (len(text) != 7 && len(text) != 9) || text[0] != '#' {
		return c, false
	}
	for i := 0; i < (len(text)-1)/2; i++ {
		v, err := strconv.ParseUint(text[1+2*i:3+2*i], 16, 8)
		if err != nil {
			return c, false
		}
		c[i] = uint8(v)
	}
	return c, true
}

// number reads a decimal number that is finite.
func number(text string) (float64, bool) {
	v, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// transform reads the twelve numbers of a transform of 3MF: the three rows of the matrix and then the translation, which are applied
// to a row vector, so that x' = m00 x + m10 y + m20 z + m30. It is the matrix of the model package (column-major, applied to a column
// vector) with the same product.
func transform(text string) (vecmath.Mat4, bool) {
	fields := strings.Fields(text)
	if len(fields) != 12 {
		return vecmath.Mat4{}, false
	}
	var m [12]float64
	for i, f := range fields {
		v, ok := number(f)
		if !ok {
			return vecmath.Mat4{}, false
		}
		m[i] = v
	}
	return vecmath.Mat4{m[0], m[1], m[2], 0, m[3], m[4], m[5], 0, m[6], m[7], m[8], 0, m[9], m[10], m[11], 1}, true
}

// objectRef reads objectid, the id of an object that is defined before.
func (d *decoder) objectRef(s xml.StartElement) (*object, error) {
	text, _ := attr(s, "objectid")
	line, col := d.dec.InputPos()
	v, err := strconv.ParseUint(text, 10, 31)
	if err != nil || v == 0 {
		return nil, errors.New(TagThreeMFInvalid, "objectid is a whole number from 1", "reason", "bad id", "part", d.part, "element", s.Name.Local, "id", text, "line", line, "column", col)
	}
	obj, ok := d.objects[int(v)]
	if !ok {
		return nil, errors.New(TagThreeMFInvalid, "objectid names an object that is not defined before", "reason", "unknown object", "part", d.part, "element", s.Name.Local, "id", text, "line", line, "column", col)
	}
	return obj, nil
}

// readComponents reads the components of an object: the objects that it is made of, each with a transform.
func (d *decoder) readComponents(obj *object) error {
	for {
		tok, err := d.next()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.EndElement:
			return nil
		case xml.StartElement:
			if t.Name.Space != coreNS || t.Name.Local != "component" {
				if err := d.skip(); err != nil {
					return err
				}
				continue
			}
			for _, a := range t.Attr {
				if a.Name.Space == productionNS && a.Name.Local == "path" {
					line, col := d.dec.InputPos()
					return errors.New(TagThreeMFInvalid, "a component is in another part of the package (the production extension is not read)", "reason", "component in another part", "part", d.part, "path", a.Value, "line", line, "column", col)
				}
			}
			ref, err := d.objectRef(t)
			if err != nil {
				return err
			}
			c := component{object: ref, transform: vecmath.Mat4Identity()}
			if text, ok := attr(t, "transform"); ok {
				var valid bool
				if c.transform, valid = transform(text); !valid {
					line, col := d.dec.InputPos()
					return errors.New(TagThreeMFInvalid, "a transform is twelve finite numbers", "reason", "bad transform", "part", d.part, "transform", text, "line", line, "column", col)
				}
			}
			if ref.depth+1 > maxDepth {
				line, col := d.dec.InputPos()
				return errors.New(TagThreeMFInvalid, "the components are nested too deep", "reason", "components too deep", "part", d.part, "line", line, "column", col)
			}
			obj.depth = max(obj.depth, ref.depth+1)
			obj.children = append(obj.children, c)
			if err := d.skip(); err != nil {
				return err
			}
		}
	}
}

// readBuild reads the items to build: each is a top node of the model.
func (d *decoder) readBuild() error {
	for {
		tok, err := d.next()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.EndElement:
			return nil
		case xml.StartElement:
			if t.Name.Space != coreNS || t.Name.Local != "item" {
				if err := d.skip(); err != nil {
					return err
				}
				continue
			}
			ref, err := d.objectRef(t)
			if err != nil {
				return err
			}
			tr := vecmath.Mat4Identity()
			if text, ok := attr(t, "transform"); ok {
				var valid bool
				if tr, valid = transform(text); !valid {
					line, col := d.dec.InputPos()
					return errors.New(TagThreeMFInvalid, "a transform is twelve finite numbers", "reason", "bad transform", "part", d.part, "transform", text, "line", line, "column", col)
				}
			}
			node, err := d.node(ref, tr)
			if err != nil {
				return err
			}
			if part, ok := attr(t, "partnumber"); ok && part != "" {
				node.Name = part
			}
			d.out.Nodes = append(d.out.Nodes, node)
			d.built = true
			if err := d.skip(); err != nil {
				return err
			}
		}
	}
}

// node makes the node of an object with the transform, and the nodes of its components under it.
func (d *decoder) node(obj *object, tr vecmath.Mat4) (*model.Node, error) {
	if d.nodes++; d.nodes > maxItems {
		line, col := d.dec.InputPos()
		return nil, errors.New(TagThreeMFInvalid, "the build makes too many nodes", "reason", "too many nodes", "part", d.part, "line", line, "column", col)
	}
	n := model.NewNode(obj.name)
	n.Transform = tr
	n.Meshes = obj.meshes
	for _, c := range obj.children {
		child, err := d.node(c.object, c.transform)
		if err != nil {
			return nil, err
		}
		n.Children = append(n.Children, child)
	}
	return n, nil
}
