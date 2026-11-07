package gltf

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/vecmath"
)

// Names of the extensions that are read.
const (
	extensionMeshQuantization = "KHR_mesh_quantization"
	extensionTextureTransform = "KHR_texture_transform"
	extensionEmissiveStrength = "KHR_materials_emissive_strength"
	extensionUnlit            = "KHR_materials_unlit"
	extensionLights           = "KHR_lights_punctual"
)

// known are the extensions that the loader reads.
var known = map[string]bool{
	extensionMeshQuantization: true,
	extensionTextureTransform: true,
	extensionEmissiveStrength: true,
	extensionUnlit:            true,
	extensionLights:           true,
	extensionMeshopt:          true,
	extensionMeshoptKHR:       true,
}

// loader builds a model from a document; it holds what is read once (buffers, views, accessors) and what is built once.
type loader struct {
	ctx  context.Context
	doc  *document
	bin  []byte
	opts Options
	log  *slog.Logger

	// declared are the extensions that the file lists in extensionsUsed.
	declared map[string]bool
	// logged are the notes that were made, so that none is made twice.
	logged map[string]bool

	buffers   map[int][]byte
	views     map[int]view
	accessors map[int]*accessor

	m         *model.Model
	images    []*model.Image
	textures  []*model.Texture
	materials []*model.Material
	// meshes has the model meshes of each glTF mesh, one for each primitive.
	meshes  [][]*model.Mesh
	nodes   []*model.Node
	cameras []*model.Camera
	lights  []*model.Light
	skins   []*model.Skin
}

func newLoader(ctx context.Context, doc *document, bin []byte, o Options) *loader {
	return &loader{
		ctx: ctx, doc: doc, bin: bin, opts: o, log: o.logger(),
		declared:  map[string]bool{},
		logged:    map[string]bool{},
		buffers:   map[int][]byte{},
		views:     map[int]view{},
		accessors: map[int]*accessor{},
	}
}

// supported says whether the loader reads the extension.
func (l *loader) supported(name string) bool { return known[name] }

// note logs a message once.
func (l *loader) note(key, msg string, args ...any) {
	if l.logged[key] {
		return
	}
	l.logged[key] = true
	l.log.Info(msg, args...)
}

// build makes the model.
func (l *loader) build() (*model.Model, error) {
	doc := l.doc
	if doc.Asset.Version == "" {
		return nil, errors.New(TagGLTFInvalid, "the file has no asset version", "path", "/asset/version", "reason", "missing property")
	}
	if major, _, _ := strings.Cut(doc.Asset.Version, "."); major != "2" {
		return nil, errors.New(TagGLTFInvalid, "the file is not glTF 2.x", "path", "/asset/version", "reason", "bad version", "version", doc.Asset.Version)
	}
	if mv := doc.Asset.MinVersion; mv != "" {
		if major, minor, _ := strings.Cut(mv, "."); major != "2" || (minor != "0" && minor != "") {
			return nil, errors.New(TagGLTFInvalid, "the file needs a newer version of glTF than 2.0", "path", "/asset/minVersion", "reason", "bad version", "minVersion", mv)
		}
	}

	for _, name := range doc.ExtensionsUsed {
		l.declared[name] = true
	}
	for i, name := range doc.ExtensionsRequired {
		if !l.supported(name) {
			return nil, errors.New(TagGLTFExtensionUnsupported, "the file requires an extension that is not read", "path", ptr("extensionsRequired", i), "extension", name)
		}
	}
	for _, name := range doc.ExtensionsUsed {
		if !l.supported(name) {
			l.note("extension "+name, "glTF: an optional extension is not read and is left out", "extension", name)
		}
	}

	l.m = &model.Model{Name: l.opts.Name, Units: model.Units{Name: "m", MetersPerUnit: 1}}

	if err := l.buildMaterials(); err != nil {
		return nil, err
	}
	if err := l.buildCameras(); err != nil {
		return nil, err
	}
	if err := l.buildLights(); err != nil {
		return nil, err
	}
	if err := l.buildNodes(); err != nil {
		return nil, err
	}
	if err := l.buildMeshes(); err != nil {
		return nil, err
	}
	if err := l.linkNodes(); err != nil {
		return nil, err
	}
	if err := l.buildSkins(); err != nil {
		return nil, err
	}
	if err := l.checkSkinnedMeshes(); err != nil {
		return nil, err
	}
	if err := l.buildScenes(); err != nil {
		return nil, err
	}
	if err := l.buildAnimations(); err != nil {
		return nil, err
	}

	l.m.Cameras, l.m.Lights, l.m.Skins = l.cameras, l.lights, l.skins
	return l.m, nil
}

// buildCameras makes the cameras.
func (l *loader) buildCameras() error {
	for i := range l.doc.Cameras {
		cj := &l.doc.Cameras[i]
		here := ptr("cameras", i)
		c := &model.Camera{Name: cj.Name}
		switch cj.Type {
		case "perspective":
			p := cj.Perspective
			if p == nil || p.YFov == nil || p.ZNear == nil {
				return errors.New(TagGLTFInvalid, "a perspective camera has no yfov or no znear", "path", here+"/perspective", "reason", "missing property")
			}
			if *p.YFov <= 0 || *p.ZNear <= 0 || p.AspectRatio < 0 || (p.ZFar != 0 && p.ZFar <= *p.ZNear) {
				return errors.New(TagGLTFInvalid, "a perspective camera has a value that is out of range", "path", here+"/perspective", "reason", "bad camera")
			}
			c.YFov, c.AspectRatio, c.ZNear, c.ZFar = *p.YFov, p.AspectRatio, *p.ZNear, p.ZFar
		case "orthographic":
			o := cj.Orthographic
			if o == nil || o.XMag == nil || o.YMag == nil || o.ZFar == nil || o.ZNear == nil {
				return errors.New(TagGLTFInvalid, "an orthographic camera lacks a property", "path", here+"/orthographic", "reason", "missing property")
			}
			if *o.XMag == 0 || *o.YMag == 0 || *o.ZNear < 0 || *o.ZFar <= *o.ZNear {
				return errors.New(TagGLTFInvalid, "an orthographic camera has a value that is out of range", "path", here+"/orthographic", "reason", "bad camera")
			}
			c.Orthographic = true
			c.XMag, c.YMag, c.ZNear, c.ZFar = *o.XMag, *o.YMag, *o.ZNear, *o.ZFar
		default:
			return errors.New(TagGLTFInvalid, "a camera has a type that is not known", "path", here+"/type", "reason", "bad type", "type", cj.Type)
		}
		l.cameras = append(l.cameras, c)
	}
	return nil
}

// buildLights makes the lights of KHR_lights_punctual.
func (l *loader) buildLights() error {
	raw, ok := l.doc.Extensions[extensionLights]
	if !ok || !l.supported(extensionLights) {
		return nil
	}
	here := ptr("extensions", extensionLights)
	var ext lightsJSON
	if err := json.Unmarshal(raw, &ext); err != nil {
		return errors.NewErr(TagGLTFInvalid, err).SetMetadata("path", here, "reason", "a value has the wrong type")
	}
	for i := range ext.Lights {
		lj := &ext.Lights[i]
		lp := ptr("extensions", extensionLights, "lights", i)
		light := &model.Light{
			Name: lj.Name, Color: [3]float32{1, 1, 1}, Intensity: 1,
			InnerConeAngle: 0, OuterConeAngle: math.Pi / 4,
		}
		switch lj.Type {
		case "directional":
			light.Type = model.LightDirectional
		case "point":
			light.Type = model.LightPoint
		case "spot":
			light.Type = model.LightSpot
		default:
			return errors.New(TagGLTFInvalid, "a light has a type that is not known", "path", lp+"/type", "reason", "bad type", "type", lj.Type)
		}
		if lj.Color != nil {
			if len(lj.Color) != 3 || !inRange(lj.Color, 0, 1) {
				return errors.New(TagGLTFInvalid, "a light has a color that is not three numbers from 0 to 1", "path", lp+"/color", "reason", "bad color")
			}
			copy(light.Color[:], lj.Color)
		}
		if lj.Intensity != nil {
			if *lj.Intensity < 0 || !finite(*lj.Intensity) {
				return errors.New(TagGLTFInvalid, "a light has an intensity that is negative", "path", lp+"/intensity", "reason", "bad intensity")
			}
			light.Intensity = *lj.Intensity
		}
		if lj.Range != nil {
			if *lj.Range <= 0 || !finite(*lj.Range) {
				return errors.New(TagGLTFInvalid, "a light has a range that is not positive", "path", lp+"/range", "reason", "bad range")
			}
			light.Range = *lj.Range
		}
		if lj.Spot != nil && light.Type == model.LightSpot {
			if lj.Spot.InnerConeAngle != nil {
				light.InnerConeAngle = *lj.Spot.InnerConeAngle
			}
			if lj.Spot.OuterConeAngle != nil {
				light.OuterConeAngle = *lj.Spot.OuterConeAngle
			}
			if light.InnerConeAngle < 0 || light.OuterConeAngle > math.Pi/2 || light.InnerConeAngle >= light.OuterConeAngle {
				return errors.New(TagGLTFInvalid, "a spot light has cone angles that are out of range", "path", lp+"/spot", "reason", "bad cone angle")
			}
		}
		l.lights = append(l.lights, light)
	}
	return nil
}

func finite(v float32) bool { return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0) }

// inRange says whether every number is finite and from lo to hi.
func inRange(v []float32, lo, hi float32) bool {
	for _, x := range v {
		if !finite(x) || x < lo || x > hi {
			return false
		}
	}
	return true
}

// buildNodes makes a model node of each node, with its transform; the meshes, the children and the skins are linked later.
func (l *loader) buildNodes() error {
	l.nodes = make([]*model.Node, len(l.doc.Nodes))
	for i := range l.doc.Nodes {
		nj := &l.doc.Nodes[i]
		here := ptr("nodes", i)
		n := model.NewNode(nj.Name)
		l.nodes[i] = n

		hasTRS := nj.Translation != nil || nj.Rotation != nil || nj.Scale != nil
		switch {
		case nj.Matrix != nil:
			if hasTRS {
				return errors.New(TagGLTFInvalid, "a node has a matrix and a translation, a rotation or a scale", "path", here, "reason", "matrix and TRS")
			}
			if len(nj.Matrix) != 16 || !finite64(nj.Matrix) {
				return errors.New(TagGLTFInvalid, "a node has a matrix that is not 16 finite numbers", "path", here+"/matrix", "reason", "bad matrix")
			}
			copy(n.Transform[:], nj.Matrix)
		default:
			trs := model.IdentityTRS()
			if nj.Translation != nil {
				if len(nj.Translation) != 3 || !finite64(nj.Translation) {
					return errors.New(TagGLTFInvalid, "a node has a translation that is not 3 finite numbers", "path", here+"/translation", "reason", "bad translation")
				}
				trs.Translation = vecmath.Vec3{X: nj.Translation[0], Y: nj.Translation[1], Z: nj.Translation[2]}
			}
			if nj.Rotation != nil {
				if len(nj.Rotation) != 4 || !finite64(nj.Rotation) {
					return errors.New(TagGLTFInvalid, "a node has a rotation that is not 4 finite numbers", "path", here+"/rotation", "reason", "bad rotation")
				}
				q := vecmath.Quat{X: nj.Rotation[0], Y: nj.Rotation[1], Z: nj.Rotation[2], W: nj.Rotation[3]}
				if q.Length() == 0 {
					return errors.New(TagGLTFInvalid, "a node has a rotation of length 0", "path", here+"/rotation", "reason", "bad rotation")
				}
				trs.Rotation = q.Normalize()
			}
			if nj.Scale != nil {
				if len(nj.Scale) != 3 || !finite64(nj.Scale) {
					return errors.New(TagGLTFInvalid, "a node has a scale that is not 3 finite numbers", "path", here+"/scale", "reason", "bad scale")
				}
				trs.Scale = vecmath.Vec3{X: nj.Scale[0], Y: nj.Scale[1], Z: nj.Scale[2]}
			}
			n.TRS = &trs
			n.Transform = trs.Matrix()
		}

		if nj.Camera != nil {
			if *nj.Camera < 0 || *nj.Camera >= len(l.cameras) {
				return errors.New(TagGLTFInvalid, "a node has a camera that does not exist", "path", here+"/camera", "reason", "index out of range", "index", *nj.Camera)
			}
			n.Camera = l.cameras[*nj.Camera]
		}
		if raw, ok := nj.Extensions[extensionLights]; ok && l.supported(extensionLights) {
			var ext nodeLightJSON
			if err := json.Unmarshal(raw, &ext); err != nil {
				return errors.NewErr(TagGLTFInvalid, err).SetMetadata("path", ptr("nodes", i, "extensions", extensionLights), "reason", "a value has the wrong type")
			}
			if ext.Light == nil || *ext.Light < 0 || *ext.Light >= len(l.lights) {
				return errors.New(TagGLTFInvalid, "a node has a light that does not exist", "path", ptr("nodes", i, "extensions", extensionLights, "light"), "reason", "index out of range")
			}
			n.Light = l.lights[*ext.Light]
		}
		if nj.Weights != nil {
			n.MorphWeights = append([]float32(nil), nj.Weights...)
		}
	}
	return nil
}

func finite64(v []float64) bool {
	for _, x := range v {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return false
		}
	}
	return true
}

// linkNodes gives each node its meshes and its children, and checks that the nodes make a forest: every node has at most one
// parent and none is its own ancestor.
func (l *loader) linkNodes() error {
	parent := make([]int, len(l.doc.Nodes))
	for i := range parent {
		parent[i] = -1
	}
	for i := range l.doc.Nodes {
		nj := &l.doc.Nodes[i]
		n := l.nodes[i]
		if nj.Mesh != nil {
			if *nj.Mesh < 0 || *nj.Mesh >= len(l.meshes) {
				return errors.New(TagGLTFInvalid, "a node has a mesh that does not exist", "path", ptr("nodes", i, "mesh"), "reason", "index out of range", "index", *nj.Mesh)
			}
			n.Meshes = l.meshes[*nj.Mesh]
		}
		if nj.Skin != nil && (*nj.Skin < 0 || *nj.Skin >= len(l.doc.Skins)) {
			return errors.New(TagGLTFInvalid, "a node has a skin that does not exist", "path", ptr("nodes", i, "skin"), "reason", "index out of range", "index", *nj.Skin)
		}
		for k, c := range nj.Children {
			if c < 0 || c >= len(l.nodes) {
				return errors.New(TagGLTFInvalid, "a node has a child that does not exist", "path", ptr("nodes", i, "children", k), "reason", "index out of range", "index", c)
			}
			if parent[c] != -1 {
				return errors.New(TagGLTFInvalid, "a node is the child of two nodes", "path", ptr("nodes", i, "children", k), "reason", "two parents", "index", c)
			}
			parent[c] = i
			n.Children = append(n.Children, l.nodes[c])
		}
	}
	// a cycle has no root: every node of it has a parent, so it is found from any of its nodes
	for i := range parent {
		steps := 0
		for p := parent[i]; p != -1; p = parent[p] {
			if steps++; steps > len(parent) {
				return errors.New(TagGLTFInvalid, "the nodes have a cycle", "path", ptr("nodes", i), "reason", "cycle")
			}
		}
	}
	return nil
}

// buildSkins makes the skins and gives them to their nodes.
func (l *loader) buildSkins() error {
	for i := range l.doc.Skins {
		sj := &l.doc.Skins[i]
		here := ptr("skins", i)
		if len(sj.Joints) == 0 {
			return errors.New(TagGLTFInvalid, "a skin has no joints", "path", here+"/joints", "reason", "no joints")
		}
		s := &model.Skin{Name: sj.Name}
		seen := make(map[int]bool, len(sj.Joints))
		for k, j := range sj.Joints {
			if j < 0 || j >= len(l.nodes) {
				return errors.New(TagGLTFInvalid, "a skin has a joint that does not exist", "path", ptr("skins", i, "joints", k), "reason", "index out of range", "index", j)
			}
			if seen[j] {
				return errors.New(TagGLTFInvalid, "a skin has a joint twice", "path", ptr("skins", i, "joints", k), "reason", "duplicate joint", "index", j)
			}
			seen[j] = true
			s.Joints = append(s.Joints, l.nodes[j])
		}
		if sj.Skeleton != nil {
			if *sj.Skeleton < 0 || *sj.Skeleton >= len(l.nodes) {
				return errors.New(TagGLTFInvalid, "a skin has a skeleton that does not exist", "path", here+"/skeleton", "reason", "index out of range", "index", *sj.Skeleton)
			}
			s.Skeleton = l.nodes[*sj.Skeleton]
		}
		s.InverseBindMatrices = make([]vecmath.Mat4, len(s.Joints))
		for k := range s.InverseBindMatrices {
			s.InverseBindMatrices[k] = vecmath.Mat4Identity()
		}
		if sj.InverseBindMatrices != nil {
			a, err := l.accessor(*sj.InverseBindMatrices, here+"/inverseBindMatrices")
			if err != nil {
				return err
			}
			if a.typ != "MAT4" || a.ct != ctFloat {
				return errors.New(TagGLTFInvalid, "the inverse bind matrices are not MAT4 of floats", "path", here+"/inverseBindMatrices", "reason", "bad accessor type", "type", a.typ, "componentType", a.ct)
			}
			if a.count < len(s.Joints) {
				return errors.New(TagGLTFInvalid, "a skin has fewer inverse bind matrices than joints", "path", here+"/inverseBindMatrices", "reason", "bad count", "count", a.count, "joints", len(s.Joints))
			}
			vals, err := l.floats(a)
			if err != nil {
				return err
			}
			for k := range s.InverseBindMatrices {
				for c := 0; c < 16; c++ {
					s.InverseBindMatrices[k][c] = float64(vals[k*16+c])
				}
			}
		}
		l.skins = append(l.skins, s)
	}
	for i := range l.doc.Nodes {
		if s := l.doc.Nodes[i].Skin; s != nil {
			l.nodes[i].Skin = l.skins[*s]
		}
	}
	return nil
}

// checkSkinnedMeshes checks that the joints of the skinned primitives of a node that has a skin are in the skin.
func (l *loader) checkSkinnedMeshes() error {
	for i := range l.doc.Nodes {
		n := l.nodes[i]
		if n.Skin == nil || len(n.Meshes) == 0 {
			continue
		}
		for _, mesh := range n.Meshes {
			if mesh.Joints == nil || mesh.Weights == nil {
				continue // a primitive that is not skinned moves with the node
			}
			for v, w := range mesh.Weights {
				if w > 0 && int(mesh.Joints[v]) >= len(n.Skin.Joints) {
					return errors.New(TagGLTFInvalid, "a vertex has a joint that its skin does not have", "path", ptr("nodes", i, "skin"), "reason", "joint out of range", "joint", mesh.Joints[v], "joints", len(n.Skin.Joints))
				}
			}
		}
	}
	return nil
}

// buildScenes makes the scenes, and the top nodes of the model: those of the default scene, or of the first, or the nodes that no
// node has as a child.
func (l *loader) buildScenes() error {
	for i := range l.doc.Scenes {
		sj := &l.doc.Scenes[i]
		s := &model.Scene{Name: sj.Name}
		for k, n := range sj.Nodes {
			if n < 0 || n >= len(l.nodes) {
				return errors.New(TagGLTFInvalid, "a scene has a node that does not exist", "path", ptr("scenes", i, "nodes", k), "reason", "index out of range", "index", n)
			}
			s.Nodes = append(s.Nodes, l.nodes[n])
		}
		l.m.Scenes = append(l.m.Scenes, s)
	}
	if sc := l.doc.Scene; sc != nil {
		if *sc < 0 || *sc >= len(l.m.Scenes) {
			return errors.New(TagGLTFInvalid, "the default scene does not exist", "path", "/scene", "reason", "index out of range", "index", *sc)
		}
		l.m.DefaultScene = l.m.Scenes[*sc]
	} else if len(l.m.Scenes) > 0 {
		l.m.DefaultScene = l.m.Scenes[0]
	}
	if l.m.DefaultScene != nil {
		l.m.Nodes = l.m.DefaultScene.Nodes
		return nil
	}
	isChild := make(map[*model.Node]bool)
	for _, n := range l.nodes {
		for _, c := range n.Children {
			isChild[c] = true
		}
	}
	for _, n := range l.nodes {
		if !isChild[n] {
			l.m.Nodes = append(l.m.Nodes, n)
		}
	}
	return nil
}
