package gltf

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
)

// The modes of a primitive.
const (
	modePoints        = 0
	modeLines         = 1
	modeLineLoop      = 2
	modeLineStrip     = 3
	modeTriangles     = 4
	modeTriangleStrip = 5
	modeTriangleFan   = 6
)

// attributeBase splits a semantic such as "TEXCOORD_1" into its name and its set (0 when it has none); ok is false when the
// suffix is not a number.
func attributeBase(semantic string) (name string, set int, ok bool) {
	name, suffix, found := strings.Cut(semantic, "_")
	if !found {
		return name, 0, true
	}
	n, err := strconv.Atoi(suffix)
	if err != nil || n < 0 || suffix != strconv.Itoa(n) {
		return semantic, 0, false
	}
	return name, n, true
}

// checkAttribute checks that an accessor is one that the semantic may have. The types of the base specification are always
// allowed, and those of KHR_mesh_quantization only when the file lists the extension. target is true for the attribute of a morph
// target, whose tangents have three components.
func (l *loader) checkAttribute(name string, a *accessor, where string, target bool) error {
	quant := l.declared[extensionMeshQuantization]
	isInt := a.ct == ctByte || a.ct == ctUbyte || a.ct == ctShort || a.ct == ctUshort
	normU := a.normalized && (a.ct == ctUbyte || a.ct == ctUshort)
	normS := a.normalized && (a.ct == ctByte || a.ct == ctShort)
	ok, typ := false, ""
	switch name {
	case "POSITION":
		typ = "VEC3"
		ok = a.ct == ctFloat || (quant && isInt)
	case "NORMAL":
		typ = "VEC3"
		ok = a.ct == ctFloat || (quant && normS)
	case "TANGENT":
		typ = "VEC4"
		if target {
			typ = "VEC3"
		}
		ok = a.ct == ctFloat || (quant && normS)
	case "TEXCOORD":
		typ = "VEC2"
		ok = a.ct == ctFloat || normU || (quant && isInt)
	case "COLOR":
		typ = a.typ
		ok = (a.typ == "VEC3" || a.typ == "VEC4") && (a.ct == ctFloat || normU)
	case "JOINTS":
		typ = "VEC4"
		ok = !a.normalized && (a.ct == ctUbyte || a.ct == ctUshort)
	case "WEIGHTS":
		typ = "VEC4"
		ok = a.ct == ctFloat || normU
	}
	if a.typ != typ {
		return errors.New(TagGLTFInvalid, "an attribute has an accessor of a type that it cannot have", "path", where, "reason", "bad accessor type", "type", a.typ, "want", typ)
	}
	if !ok {
		return errors.New(TagGLTFInvalid, "an attribute has an accessor of a component type that it cannot have (without KHR_mesh_quantization)", "path", where, "reason", "bad componentType", "componentType", a.ct, "normalized", a.normalized)
	}
	return nil
}

// buildMeshes makes the model meshes: one for each primitive of each mesh.
func (l *loader) buildMeshes() error {
	l.meshes = make([][]*model.Mesh, len(l.doc.Meshes))
	for i := range l.doc.Meshes {
		if err := l.checkContext(); err != nil {
			return err
		}
		mj := &l.doc.Meshes[i]
		here := ptr("meshes", i)
		if len(mj.Primitives) == 0 {
			return errors.New(TagGLTFInvalid, "a mesh has no primitives", "path", here+"/primitives", "reason", "no primitives")
		}

		var names []string
		if mj.Extras != nil {
			var extras meshExtrasJSON
			if err := json.Unmarshal(mj.Extras, &extras); err == nil {
				names = extras.TargetNames
			}
		}

		targets := -1
		for k := range mj.Primitives {
			pj := &mj.Primitives[k]
			pp := ptr("meshes", i, "primitives", k)
			if targets >= 0 && len(pj.Targets) != targets {
				return errors.New(TagGLTFInvalid, "the primitives of a mesh have different numbers of morph targets", "path", pp+"/targets", "reason", "bad targets", "targets", len(pj.Targets))
			}
			targets = len(pj.Targets)
			if mj.Weights != nil && len(mj.Weights) != targets {
				return errors.New(TagGLTFInvalid, "a mesh has weights that are not as many as its morph targets", "path", here+"/weights", "reason", "bad weights", "weights", len(mj.Weights), "targets", targets)
			}

			mesh, err := l.primitive(pj, pp)
			if err != nil {
				return err
			}
			if mesh == nil {
				continue
			}
			mesh.Name = mj.Name
			if mj.Weights != nil {
				mesh.MorphWeights = append([]float32(nil), mj.Weights...)
			}
			if len(names) == targets {
				for t := range mesh.MorphTargets {
					mesh.MorphTargets[t].Name = names[t]
				}
			}
			l.meshes[i] = append(l.meshes[i], mesh)
			l.m.Meshes = append(l.m.Meshes, mesh)
		}
	}
	return nil
}

// primitive makes the mesh of a primitive, or nil (with a note) when it has nothing to draw.
func (l *loader) primitive(pj *primitiveJSON, where string) (*model.Mesh, error) {
	mode := modeTriangles
	if pj.Mode != nil {
		mode = *pj.Mode
	}
	if mode < modePoints || mode > modeTriangleFan {
		return nil, errors.New(TagGLTFInvalid, "a primitive has a mode that is not known", "path", where+"/mode", "reason", "bad mode", "mode", mode)
	}

	mesh := &model.Mesh{}
	if pj.Material != nil {
		if *pj.Material < 0 || *pj.Material >= len(l.materials) {
			return nil, errors.New(TagGLTFInvalid, "a primitive has a material that does not exist", "path", where+"/material", "reason", "index out of range", "index", *pj.Material)
		}
		mesh.Material = l.materials[*pj.Material]
	}

	names := make([]string, 0, len(pj.Attributes))
	for name := range pj.Attributes {
		names = append(names, name)
	}
	slices.Sort(names)

	vertices := -1
	var weights []float32
	var jointSet []uint32
	for _, semantic := range names {
		idx := pj.Attributes[semantic]
		ap := where + "/attributes/" + semantic
		base, set, ok := attributeBase(semantic)
		if !ok || strings.HasPrefix(base, "_") {
			continue // an attribute of the application
		}
		switch {
		case base == "POSITION" || base == "NORMAL" || base == "TANGENT":
		case base == "TEXCOORD" && set <= 1, base == "COLOR" && set == 0, base == "JOINTS" && set == 0, base == "WEIGHTS" && set == 0:
		case base == "TEXCOORD" || base == "COLOR" || base == "JOINTS" || base == "WEIGHTS":
			l.note("attribute "+semantic, "glTF: an attribute set is not read and is left out", "attribute", semantic)
			continue
		default:
			continue
		}

		a, err := l.accessor(idx, ap)
		if err != nil {
			return nil, err
		}
		if err := l.checkAttribute(base, a, ap, false); err != nil {
			return nil, err
		}
		if vertices < 0 {
			vertices = a.count
		}
		if a.count != vertices {
			return nil, errors.New(TagGLTFInvalid, "the attributes of a primitive do not have the same count", "path", ap, "reason", "bad count", "count", a.count, "want", vertices)
		}

		if base == "JOINTS" {
			if jointSet, err = l.uints(a); err != nil {
				return nil, err
			}
			continue
		}
		vals, err := l.floats(a)
		if err != nil {
			return nil, err
		}
		switch base {
		case "POSITION":
			mesh.Positions = vals
		case "NORMAL":
			mesh.Normals = vals
		case "TANGENT":
			mesh.Tangents = vals
		case "TEXCOORD":
			if set == 0 {
				mesh.TexCoords = vals
			} else {
				mesh.TexCoords1 = vals
			}
		case "COLOR":
			mesh.Colors = colorBytes(vals, a.ncomp)
		case "WEIGHTS":
			weights = vals
		}
	}
	if mesh.Positions == nil {
		return nil, errors.New(TagGLTFInvalid, "a primitive has no POSITION", "path", where+"/attributes", "reason", "no POSITION")
	}
	if (jointSet == nil) != (weights == nil) {
		return nil, errors.New(TagGLTFInvalid, "a primitive has JOINTS_0 or WEIGHTS_0 but not both", "path", where+"/attributes", "reason", "joints and weights")
	}
	if jointSet != nil {
		mesh.Joints = make([]uint16, len(jointSet))
		for i, j := range jointSet {
			mesh.Joints[i] = uint16(j)
		}
		mesh.Weights = weights
	}

	// the indices
	var src []uint32
	if pj.Indices != nil {
		ip := where + "/indices"
		a, err := l.accessor(*pj.Indices, ip)
		if err != nil {
			return nil, err
		}
		if a.typ != "SCALAR" {
			return nil, errors.New(TagGLTFInvalid, "the indices of a primitive are not SCALAR", "path", ip, "reason", "bad accessor type", "type", a.typ)
		}
		if src, err = l.uints(a); err != nil {
			return nil, err
		}
		for k, v := range src {
			if int(v) >= vertices {
				return nil, errors.New(TagGLTFInvalid, "an index is not the index of a vertex", "path", ip, "reason", "index out of range", "item", k, "index", v, "vertices", vertices)
			}
		}
	} else {
		src = make([]uint32, vertices)
		for i := range src {
			src[i] = uint32(i)
		}
	}

	switch mode {
	case modePoints:
		mesh.Primitive = model.PrimitivePoints
		mesh.Indices = src
	case modeLines, modeLineLoop, modeLineStrip:
		mesh.Primitive = model.PrimitiveLines
		if mode == modeLines && len(src)%2 != 0 {
			return nil, errors.New(TagGLTFInvalid, "the number of indices of lines is not a multiple of two", "path", where, "reason", "bad count", "count", len(src))
		}
		mesh.Indices = lineIndices(mode, src)
	default:
		mesh.Primitive = model.PrimitiveTriangles
		if mode == modeTriangles && len(src)%3 != 0 {
			return nil, errors.New(TagGLTFInvalid, "the number of indices of triangles is not a multiple of three", "path", where, "reason", "bad count", "count", len(src))
		}
		mesh.Indices = triangleIndices(mode, src)
	}
	if len(mesh.Indices) == 0 {
		l.note("empty "+where, "glTF: a primitive has nothing to draw and is left out", "path", where)
		return nil, nil
	}

	for t := range pj.Targets {
		tp := where + "/targets/" + strconv.Itoa(t)
		target, err := l.morphTarget(pj.Targets[t], tp, vertices)
		if err != nil {
			return nil, err
		}
		mesh.MorphTargets = append(mesh.MorphTargets, target)
	}
	return mesh, nil
}

// morphTarget reads the changes of a morph target.
func (l *loader) morphTarget(attrs map[string]int, where string, vertices int) (model.MorphTarget, error) {
	var t model.MorphTarget
	names := make([]string, 0, len(attrs))
	for name := range attrs {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		ap := where + "/" + name
		if name != "POSITION" && name != "NORMAL" && name != "TANGENT" {
			continue
		}
		a, err := l.accessor(attrs[name], ap)
		if err != nil {
			return t, err
		}
		if err := l.checkAttribute(name, a, ap, true); err != nil {
			return t, err
		}
		if a.count != vertices {
			return t, errors.New(TagGLTFInvalid, "a morph target does not have as many elements as the vertices", "path", ap, "reason", "bad count", "count", a.count, "want", vertices)
		}
		vals, err := l.floats(a)
		if err != nil {
			return t, err
		}
		switch name {
		case "POSITION":
			t.Positions = vals
		case "NORMAL":
			t.Normals = vals
		case "TANGENT":
			t.Tangents = vals
		}
	}
	return t, nil
}

// colorBytes makes red, green, blue and alpha bytes of colors of 3 or 4 floats from 0 to 1.
func colorBytes(vals []float32, ncomp int) []byte {
	n := len(vals) / ncomp
	out := make([]byte, 4*n)
	b := func(v float32) byte {
		switch {
		case v <= 0:
			return 0
		case v >= 1:
			return 255
		}
		return byte(v*255 + 0.5)
	}
	for i := 0; i < n; i++ {
		out[4*i], out[4*i+1], out[4*i+2], out[4*i+3] = b(vals[i*ncomp]), b(vals[i*ncomp+1]), b(vals[i*ncomp+2]), 255
		if ncomp == 4 {
			out[4*i+3] = b(vals[i*ncomp+3])
		}
	}
	return out
}

// lineIndices makes a list of segments of the indices of lines, a line loop or a line strip.
func lineIndices(mode int, src []uint32) []uint32 {
	switch mode {
	case modeLineLoop, modeLineStrip:
		if len(src) < 2 {
			return nil
		}
		out := make([]uint32, 0, 2*len(src))
		for i := 0; i+1 < len(src); i++ {
			out = append(out, src[i], src[i+1])
		}
		if mode == modeLineLoop {
			out = append(out, src[len(src)-1], src[0])
		}
		return out
	}
	return src
}

// triangleIndices makes a list of triangles of the indices of triangles, a strip or a fan.
func triangleIndices(mode int, src []uint32) []uint32 {
	switch mode {
	case modeTriangleStrip:
		if len(src) < 3 {
			return nil
		}
		out := make([]uint32, 0, 3*(len(src)-2))
		for i := 0; i+2 < len(src); i++ {
			// the winding alternates
			if i%2 == 0 {
				out = append(out, src[i], src[i+1], src[i+2])
			} else {
				out = append(out, src[i+1], src[i], src[i+2])
			}
		}
		return out
	case modeTriangleFan:
		if len(src) < 3 {
			return nil
		}
		out := make([]uint32, 0, 3*(len(src)-2))
		for i := 1; i+1 < len(src); i++ {
			out = append(out, src[0], src[i], src[i+1])
		}
		return out
	}
	return src
}
