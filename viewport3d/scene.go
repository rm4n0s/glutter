package viewport3d

import (
	"math"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
)

// DisplayMode is how the meshes of triangles of a scene are drawn (SPEC §27.8).
//
// Flutter: no counterpart
type DisplayMode int

const (
	// DisplayModeUnspecified is the zero value: shaded.
	DisplayModeUnspecified DisplayMode = iota
	// DisplayModeShaded draws the faces with their materials.
	DisplayModeShaded
	// DisplayModeShadedWithEdges draws the faces and, over them, the feature edges (FeatureEdges, in the scene's edge style).
	DisplayModeShadedWithEdges
	// DisplayModeWireframe draws every edge of the triangles as a line and no faces; nothing hides anything.
	DisplayModeWireframe
	// DisplayModeHiddenLine draws the feature edges that a face does not hide, the faces being drawn in the background color.
	DisplayModeHiddenLine
)

// String names the mode.
//
// Flutter: no counterpart
func (m DisplayMode) String() string {
	switch m {
	case DisplayModeShadedWithEdges:
		return "shaded with edges"
	case DisplayModeWireframe:
		return "wireframe"
	case DisplayModeHiddenLine:
		return "hidden line"
	}
	return "shaded"
}

// DirectionalLight is a light that shines from one direction onto the whole scene.
//
// Flutter: no counterpart
type DirectionalLight struct {
	// Direction points from the scene to the light, in the world's coordinates; it need not have length 1.
	Direction vecmath.Vec3
	// Color is the color of the light; the zero value stands for white.
	Color ui.Color
	// Intensity multiplies the color; the default is 1.
	Intensity float64
}

// maxLights is how many directional lights a scene has.
const maxLights = 4

// SceneStyle is how a scene looks, apart from its items: the background, the display mode, the light and the edges. The zero
// value is the default.
//
// Flutter: no counterpart
type SceneStyle struct {
	// Background is the color the view is cleared to; the default is a dark gray-blue. Some(ui.Color{}) is transparent.
	Background foundation.Opt[ui.Color]
	// DisplayMode is how the triangles are drawn; the default is shaded.
	DisplayMode DisplayMode
	// Ambient is the light that comes from everywhere, from 0 to 1; the default is 0.3.
	Ambient foundation.Opt[float64]
	// HeadLight is the strength of the light at the camera, from 0 to 1; the default is 0.7. The CAD default (SPEC §27.8):
	// a face that looks at the camera has its color as it is.
	HeadLight foundation.Opt[float64]
	// Lights are up to 4 directional lights, added to the light at the camera.
	Lights []DirectionalLight
	// EdgeAngle is the angle in radians over which the edge between two triangles is a feature edge; the default is 30 degrees.
	EdgeAngle foundation.Opt[float64]
	// EdgeColor is the color of the edges; the default is near black over faces, and where the edges are drawn against the
	// background (wireframe, hidden line) it is the one of the two, near black or near white, that contrasts with it.
	EdgeColor foundation.Opt[ui.Color]
	// EdgeWidth is the width of the edges in logical pixels; the default is 1.
	EdgeWidth float64
}

const (
	defaultAmbient   = 0.3
	defaultHeadLight = 0.7
	defaultEdgeAngle = 30 * math.Pi / 180
)

var (
	defaultBackground = ui.NewColor(0xFF22272E)
	defaultEdgeColor  = ui.NewColor(0xFF0B0E12)
	lightEdgeColor    = ui.NewColor(0xFFDCE3EA)
)

// resolved is a SceneStyle with its defaults filled in.
type resolvedStyle struct {
	background ui.Color
	mode       DisplayMode
	ambient    float64
	headLight  float64
	lights     []DirectionalLight
	edgeAngle  float64
	edgeColor  ui.Color
	edgeWidth  float64
}

// resolve fills in the defaults of the style.
func (s SceneStyle) resolve() resolvedStyle {
	r := resolvedStyle{
		background: s.Background.OrElse(defaultBackground),
		mode:       s.DisplayMode,
		ambient:    s.Ambient.OrElse(defaultAmbient),
		headLight:  s.HeadLight.OrElse(defaultHeadLight),
		edgeAngle:  s.EdgeAngle.OrElse(defaultEdgeAngle),
		edgeColor:  defaultEdgeColor,
		edgeWidth:  s.EdgeWidth,
	}
	if r.mode == DisplayModeUnspecified {
		r.mode = DisplayModeShaded
	}
	if c, ok := s.EdgeColor.Get(); ok {
		r.edgeColor = c
	} else if (r.mode == DisplayModeWireframe || r.mode == DisplayModeHiddenLine) && r.background.ComputeLuminance() < 0.4 {
		r.edgeColor = lightEdgeColor
	}
	if r.edgeWidth <= 0 {
		r.edgeWidth = defaultLineWidth
	}
	n := len(s.Lights)
	if n > maxLights {
		n = maxLights
	}
	r.lights = s.Lights[:n]
	return r
}

// AllLayers is the layer mask of an item or a view that is in every layer.
//
// Flutter: no counterpart
const AllLayers = ^uint32(0)

// Item is one thing a scene draws: a mesh with a material, a transform in float64, a visibility, a pick ID and a layer mask
// (SPEC §27.8). Create it with NewItem, add it to a scene, and change it with its setters: every change draws the scene's
// views again.
//
// Flutter: no counterpart
type Item struct {
	scene     *Scene
	mesh      *Mesh
	material  Material
	transform vecmath.Mat4
	visible   bool
	pickID    uint32
	layers    uint32

	// model is the transform of the mesh's vertices: the item's transform after the translation to the mesh's origin.
	model vecmath.Mat4
	// sphere is the bounding sphere of the item in the world, for culling and for sorting transparent items.
	sphere vecmath.Sphere
}

// ItemOpts are the optional parameters of NewItem.
//
// Flutter: no counterpart
type ItemOpts struct {
	// Transform is the item's transform in the world, in float64; the default is the identity.
	Transform foundation.Opt[vecmath.Mat4]
	// Hidden makes the item invisible.
	Hidden bool
	// PickID is the number that picking reports for the item (SPEC §27.8); 0 is "nothing".
	PickID uint32
	// Layers is a bit mask of the layers the item is in; the default (0) is all of them.
	Layers uint32
}

// TagItemInvalid is the tag of the panic of NewItem and Item.SetMaterial for a mesh and a material that do not go together
// (a mesh of lines with a material of triangles, ...), or for a missing mesh or material. Programmer errors panic in debug
// builds (SPEC §14).
//
// Flutter: no counterpart
const TagItemInvalid = "ItemInvalid"

// NewItem creates an item that draws the mesh with the material. A mesh of triangles takes an UnlitMaterial or a
// PhongMaterial, a mesh of lines a LineMaterial and a mesh of points a PointMaterial.
//
// Flutter: no counterpart
func NewItem(mesh *Mesh, material Material, opts ...ItemOpts) *Item {
	o := firstOpts(opts)
	checkItem(mesh, material)
	it := &Item{mesh: mesh, material: material, transform: vecmath.Mat4Identity(), visible: !o.Hidden, pickID: o.PickID, layers: o.Layers}
	if o.Transform.IsSome() {
		it.transform = o.Transform.Value()
	}
	if it.layers == 0 {
		it.layers = AllLayers
	}
	it.update()
	return it
}

// checkItem panics, in debug builds, when a mesh and a material do not go together.
func checkItem(mesh *Mesh, material Material) {
	if !foundation.DebugMode {
		return
	}
	switch {
	case mesh == nil:
		panic(errors.New(TagItemInvalid, "an item has a mesh"))
	case material == nil:
		panic(errors.New(TagItemInvalid, "an item has a material"))
	case material.primitive() != mesh.Primitive():
		panic(errors.New(TagItemInvalid, "the material does not draw the primitive of the mesh", "mesh", mesh.Primitive().String(), "material", material.primitive().String()))
	}
}

// update makes the model matrix and the bounding sphere of the item.
func (it *Item) update() {
	it.model = it.transform.Mul(vecmath.NewMat4Translation(it.mesh.data.Origin))
	it.sphere = vecmath.Sphere{}
	if b := it.mesh.local; !b.IsEmpty() {
		// The bounds in the item's own coordinates with its transform: the sphere around the box's center.
		s := vecmath.Box3{Min: b.Min, Max: b.Max}.BoundingSphere()
		c := it.model.TransformPoint(s.Center)
		scale := math.Sqrt(math.Max(it.model.TransformDirection(vecmath.Vec3X).LengthSq(),
			math.Max(it.model.TransformDirection(vecmath.Vec3Y).LengthSq(), it.model.TransformDirection(vecmath.Vec3Z).LengthSq())))
		it.sphere = vecmath.Sphere{Center: c, Radius: s.Radius * scale}
	}
}

// changed tells the scene that the item changed.
func (it *Item) changed(boundsChanged bool) {
	if it.scene != nil {
		it.scene.itemChanged(boundsChanged)
	}
}

// Mesh is the item's mesh.
//
// Flutter: no counterpart
func (it *Item) Mesh() *Mesh { return it.mesh }

// Material is the item's material.
//
// Flutter: no counterpart
func (it *Item) Material() Material { return it.material }

// SetMaterial changes the material.
//
// Flutter: no counterpart
func (it *Item) SetMaterial(m Material) {
	if it.material == m {
		return
	}
	checkItem(it.mesh, m)
	if it.scene != nil {
		it.scene.releaseMaterial(it.material)
		it.scene.retainMaterial(m)
	}
	it.material = m
	it.changed(false)
}

// SetMesh changes the mesh (of the same primitive as the material).
//
// Flutter: no counterpart
func (it *Item) SetMesh(m *Mesh) {
	if it.mesh == m {
		return
	}
	checkItem(m, it.material)
	it.mesh = m
	it.update()
	it.changed(true)
}

// Transform is the item's transform in the world.
//
// Flutter: no counterpart
func (it *Item) Transform() vecmath.Mat4 { return it.transform }

// SetTransform changes the transform, in float64.
//
// Flutter: no counterpart
func (it *Item) SetTransform(t vecmath.Mat4) {
	if it.transform == t {
		return
	}
	it.transform = t
	it.update()
	it.changed(true)
}

// Visible says whether the item is drawn.
//
// Flutter: no counterpart
func (it *Item) Visible() bool { return it.visible }

// SetVisible shows or hides the item.
//
// Flutter: no counterpart
func (it *Item) SetVisible(v bool) {
	if it.visible != v {
		it.visible = v
		it.changed(true)
	}
}

// PickID is the number that picking reports for the item.
//
// Flutter: no counterpart
func (it *Item) PickID() uint32 { return it.pickID }

// SetPickID changes the pick ID.
//
// Flutter: no counterpart
func (it *Item) SetPickID(id uint32) {
	if it.pickID != id {
		it.pickID = id
		it.changed(false)
	}
}

// Layers is the item's layer mask.
//
// Flutter: no counterpart
func (it *Item) Layers() uint32 { return it.layers }

// SetLayers changes the layer mask; 0 stands for all layers.
//
// Flutter: no counterpart
func (it *Item) SetLayers(l uint32) {
	if l == 0 {
		l = AllLayers
	}
	if it.layers != l {
		it.layers = l
		it.changed(false)
	}
}

// Bounds is the box of the item in the world, in float64: the mesh's bounds under the item's transform.
//
// Flutter: no counterpart
func (it *Item) Bounds() vecmath.Box3 {
	b := it.mesh.local
	if b.IsEmpty() {
		return b
	}
	return b.Transform(it.model)
}

// Scene is a flat list of items that the app owns and changes (SPEC §27.8): it is not a scene graph, and an app's own tree of
// parts becomes items with their world transforms. A Scene is a Listenable: any change (an item added, removed or changed,
// a material changed, the style) notifies its listeners, and the views that draw it draw again.
//
// Flutter: no counterpart
type Scene struct {
	foundation.ChangeNotifier
	items []*Item
	style SceneStyle
	rs    resolvedStyle

	bounds      vecmath.Box3
	boundsValid bool
	// materials counts the items that use each material, and keeps the one listener that the scene has on it.
	materials map[Material]*materialListener
}

// materialListener is the scene's listener on a material, and the number of items that use the material.
type materialListener struct {
	handle foundation.ListenerHandle
	count  int
}

// NewScene creates an empty scene.
//
// Flutter: no counterpart
func NewScene() *Scene {
	s := &Scene{materials: map[Material]*materialListener{}, bounds: vecmath.EmptyBox3()}
	s.rs = s.style.resolve()
	return s
}

// Add puts items in the scene. An item is in one scene at a time: an item that is in another scene is moved.
//
// Flutter: no counterpart
func (s *Scene) Add(items ...*Item) {
	for _, it := range items {
		if it.scene == s {
			continue
		}
		if it.scene != nil {
			it.scene.remove(it)
		}
		it.scene = s
		s.items = append(s.items, it)
		s.retainMaterial(it.material)
	}
	s.boundsValid = false
	s.NotifyListeners()
}

// Remove takes items out of the scene.
//
// Flutter: no counterpart
func (s *Scene) Remove(items ...*Item) {
	for _, it := range items {
		s.remove(it)
	}
	s.boundsValid = false
	s.NotifyListeners()
}

// remove takes an item out without telling the listeners.
func (s *Scene) remove(it *Item) {
	if it.scene != s {
		return
	}
	if i := slices.Index(s.items, it); i >= 0 {
		s.items = slices.Delete(s.items, i, i+1)
	}
	s.releaseMaterial(it.material)
	it.scene = nil
}

// Clear takes every item out of the scene.
//
// Flutter: no counterpart
func (s *Scene) Clear() {
	for _, it := range s.items {
		s.releaseMaterial(it.material)
		it.scene = nil
	}
	s.items = s.items[:0]
	s.boundsValid = false
	s.NotifyListeners()
}

// Items are the items of the scene, in the order they were added. Do not change the slice.
//
// Flutter: no counterpart
func (s *Scene) Items() []*Item { return s.items }

// Style is the style of the scene.
//
// Flutter: no counterpart
func (s *Scene) Style() SceneStyle { return s.style }

// SetStyle changes the style.
//
// Flutter: no counterpart
func (s *Scene) SetStyle(st SceneStyle) {
	s.style = st
	s.rs = st.resolve()
	s.NotifyListeners()
}

// Bounds is the box of the visible items in the world, in float64; empty without items.
//
// Flutter: no counterpart
func (s *Scene) Bounds() vecmath.Box3 {
	if !s.boundsValid {
		b := vecmath.EmptyBox3()
		for _, it := range s.items {
			if it.visible {
				b = b.Union(it.Bounds())
			}
		}
		s.bounds, s.boundsValid = b, true
	}
	return s.bounds
}

// itemChanged is told that an item of the scene changed.
func (s *Scene) itemChanged(boundsChanged bool) {
	if boundsChanged {
		s.boundsValid = false
	}
	s.NotifyListeners()
}

// retainMaterial counts an item's use of a material, and listens to it while any item uses it.
func (s *Scene) retainMaterial(m Material) {
	l := s.materials[m]
	if l == nil {
		l = &materialListener{handle: m.AddListener(s.NotifyListeners)}
		s.materials[m] = l
	}
	l.count++
}

// releaseMaterial is the end of an item's use of a material.
func (s *Scene) releaseMaterial(m Material) {
	l := s.materials[m]
	if l == nil {
		return
	}
	if l.count--; l.count <= 0 {
		m.RemoveListener(l.handle)
		delete(s.materials, m)
	}
}
