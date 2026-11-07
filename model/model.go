// Package model is the plain Go data of a 3D model that a loader reads from a file (SPEC §27.9, §28.5): meshes, materials, textures,
// images and a tree of nodes with float64 transforms, and for the formats that have them skins, morph targets, animations, cameras,
// lights and scenes (as data: viewport3d plays them). It has no GPU and no window, so a test can check it as it is. The loaders are
// in model/stl, model/obj and model/gltf, and viewport3d.Upload makes the meshes, the materials and the items of a scene from it.
package model

import (
	"github.com/rm4n0s/glutter/vecmath"
)

// Primitive is what the vertices of a mesh make.
//
// Flutter: no counterpart
type Primitive uint8

const (
	// PrimitiveTriangles is three indices for each triangle; the zero value.
	PrimitiveTriangles Primitive = iota
	// PrimitiveLines is two indices for each segment.
	PrimitiveLines
	// PrimitivePoints is one index for each point.
	PrimitivePoints
)

// String is the name of the primitive.
//
// Flutter: no counterpart
func (p Primitive) String() string {
	switch p {
	case PrimitiveTriangles:
		return "triangles"
	case PrimitiveLines:
		return "lines"
	case PrimitivePoints:
		return "points"
	}
	return "unknown"
}

// Mesh is the geometry of one material: vertices and the indices of the primitives that they make.
//
// Flutter: no counterpart
type Mesh struct {
	// Name is the name in the file, or empty.
	Name string
	// Primitive is what the indices make.
	Primitive Primitive
	// Positions are x, y and z of each vertex in float32. A loader that was asked for more digits than float32 has (Float64 in its
	// options) and found them in the file leaves Positions nil and fills Positions64 instead: exactly one of the two is set.
	Positions []float32
	// Positions64 are the positions in float64, when the loader was asked for them and the file has more digits than float32 does.
	Positions64 []float64
	// Normals are x, y and z of each vertex, of length 1; nil when the file has none (the renderer shades flat).
	Normals []float32
	// Tangents are x, y, z and w of each vertex: the direction of the u axis of the texture and the sign of the bitangent (w is 1 or
	// -1); nil when the file has none (a normal texture then needs them made).
	Tangents []float32
	// TexCoords are u and v of each vertex; nil when the file has none.
	TexCoords []float32
	// TexCoords1 are the second set of u and v of each vertex (glTF's TEXCOORD_1); nil when the file has none.
	TexCoords1 []float32
	// Colors are red, green, blue and alpha of each vertex, not premultiplied; nil when the file has none.
	Colors []byte
	// Joints are the four joints of each vertex that move it, as positions in Skin.Joints of the node that has the mesh (glTF's
	// JOINTS_0), and Weights the four weights of those joints (WEIGHTS_0), which add up to 1; both nil for a mesh that is not
	// skinned. Further sets of four (JOINTS_1) are left out by the loaders, with a note.
	Joints  []uint16
	Weights []float32
	// MorphTargets are the shapes that the mesh moves towards; MorphWeights are the weights that they have at rest, one for each
	// (nil is all 0). A node's MorphWeights replace them.
	MorphTargets []MorphTarget
	MorphWeights []float32
	// Indices are the vertices of the primitives: 3, 2 or 1 for each, by Primitive.
	Indices []uint32
	// Material is the material of the mesh, or nil for the default.
	Material *Material
}

// VertexCount is the number of vertices.
//
// Flutter: no counterpart
func (m *Mesh) VertexCount() int {
	if m.Positions64 != nil {
		return len(m.Positions64) / 3
	}
	return len(m.Positions) / 3
}

// PrimitiveCount is the number of triangles, segments or points.
//
// Flutter: no counterpart
func (m *Mesh) PrimitiveCount() int {
	switch m.Primitive {
	case PrimitiveLines:
		return len(m.Indices) / 2
	case PrimitivePoints:
		return len(m.Indices)
	}
	return len(m.Indices) / 3
}

// Position is the position of a vertex in float64.
//
// Flutter: no counterpart
func (m *Mesh) Position(i int) vecmath.Vec3 {
	if m.Positions64 != nil {
		return vecmath.Vec3{X: m.Positions64[3*i], Y: m.Positions64[3*i+1], Z: m.Positions64[3*i+2]}
	}
	return vecmath.Vec3{X: float64(m.Positions[3*i]), Y: float64(m.Positions[3*i+1]), Z: float64(m.Positions[3*i+2])}
}

// Bounds is the box of the vertices that the indices use (the box of every vertex when there are no indices), in the mesh's own
// coordinates; empty for a mesh with none. An index outside the vertices is left out.
//
// Flutter: no counterpart
func (m *Mesh) Bounds() vecmath.Box3 {
	b := vecmath.EmptyBox3()
	if m.Indices == nil {
		for i := 0; i < m.VertexCount(); i++ {
			b = b.Expand(m.Position(i))
		}
		return b
	}
	n := m.VertexCount()
	for _, i := range m.Indices {
		if int(i) < n {
			b = b.Expand(m.Position(int(i)))
		}
	}
	return b
}

// AlphaMode is how the alpha of a material is used.
//
// Flutter: no counterpart
type AlphaMode uint8

const (
	// AlphaOpaque ignores the alpha; the zero value.
	AlphaOpaque AlphaMode = iota
	// AlphaMask draws a pixel whose alpha is at least the cutoff and none otherwise.
	AlphaMask
	// AlphaBlend blends by the alpha.
	AlphaBlend
)

// Material is the look of a mesh.
//
// Flutter: no counterpart
type Material struct {
	// Name is the name in the file, or empty.
	Name string
	// BaseColor is the color, red, green, blue and alpha from 0 to 1, not premultiplied. The zero value (all four 0) stands for the
	// default, an opaque light gray; use Color to read it.
	BaseColor [4]float32
	// BaseColorImage is the image that the color is multiplied by, or nil. A loader that gives BaseColorTexture sets this to its image
	// too, for the renderers that do not read textures with samplers.
	BaseColorImage *Image
	// Specular is the color of the highlight (the Ks of an MTL file) and Shininess its exponent (the Ns).
	Specular  [3]float32
	Shininess float32
	// PBR is true for a material of the metallic-roughness model (from glTF, §28.5), which a loader fills in full: Metallic and
	// Roughness, the factors, and the textures. It is false for the Phong materials of the other loaders, which have Specular and
	// Shininess.
	PBR bool
	// Metallic and Roughness are the metallic-roughness values of a physically based material, from 0 to 1. glTF's defaults are 1
	// and 1, which its loader sets: the zero values here are values like the others.
	Metallic, Roughness float32
	// Emissive is the color that the material gives out, with the strength of KHR_materials_emissive_strength multiplied in (so it
	// may be above 1).
	Emissive [3]float32
	// Unlit says that the material is not lit: its color is the pixel's (KHR_materials_unlit).
	Unlit bool
	// BaseColorTexture multiplies BaseColor (and the vertex colors) in sRGB, and MetallicRoughnessTexture has the roughness in its
	// green channel and the metalness in its blue one, multiplied by Roughness and Metallic. NormalTexture has tangent space
	// normals, scaled by NormalScale; OcclusionTexture has the occlusion in its red channel, blended by OcclusionStrength; and
	// EmissiveTexture multiplies Emissive in sRGB. Each is nil for none.
	BaseColorTexture, MetallicRoughnessTexture, NormalTexture, OcclusionTexture, EmissiveTexture *TextureRef
	// NormalScale is the scale of the x and y of the normals of NormalTexture, and OcclusionStrength the strength of the
	// occlusion; glTF's defaults are 1 and 1, which its loader sets.
	NormalScale, OcclusionStrength float32
	// AlphaMode and AlphaCutoff say how the alpha is used.
	AlphaMode   AlphaMode
	AlphaCutoff float32
	// DoubleSided draws the back faces too.
	DoubleSided bool
}

// DefaultColor is the color of a material that has none: an opaque light gray.
//
// Flutter: no counterpart
var DefaultColor = [4]float32{0.78, 0.8, 0.82, 1}

// Color is the base color, or the default when the material has none (all four numbers 0; a PBR material always has its own).
//
// Flutter: no counterpart
func (m *Material) Color() [4]float32 {
	if m == nil || (!m.PBR && m.BaseColor == ([4]float32{})) {
		return DefaultColor
	}
	return m.BaseColor
}

// Image is an image as the file gave it, encoded.
//
// Flutter: no counterpart
type Image struct {
	// Name is the name in the file, or empty.
	Name string
	// URI is where the image was read from: the path in the file system of the loader, or empty.
	URI string
	// Type is the media type, such as "image/png" or "image/jpeg"; empty when the file name and the data do not tell.
	Type string
	// Data are the bytes of the file. Upload decodes them (ui.InstantiateImageCodec) when it makes a texture.
	Data []byte
}

// Node is a part of the tree of a model: a transform, meshes and children.
//
// Flutter: no counterpart
type Node struct {
	// Name is the name in the file, or empty.
	Name string
	// Transform takes the node's meshes to its parent's coordinates, in float64. The zero value is not a transform: loaders set
	// the identity (Mat4Identity); NewNode does too.
	Transform vecmath.Mat4
	// Meshes are drawn with the node's transform; several nodes may have the same mesh.
	Meshes []*Mesh
	// Children are nodes in the coordinates of this one.
	Children []*Node
	// TRS is the transform as a translation, a rotation and a scale when the file gave it so (it is what animations change), and
	// Transform is its matrix; nil for a node whose file gave a matrix (animations do not change those).
	TRS *TRS
	// Skin is the skeleton that moves the vertices of the node's meshes, or nil.
	Skin *Skin
	// Camera and Light are the camera and the light that the node carries, or nil. They look along the -z of the node.
	Camera *Camera
	Light  *Light
	// MorphWeights are the weights of the morph targets of the node's meshes, or nil to use those of each mesh.
	MorphWeights []float32
}

// NewNode makes a node with the identity transform.
//
// Flutter: no counterpart
func NewNode(name string) *Node { return &Node{Name: name, Transform: vecmath.Mat4Identity()} }

// Units says what a unit of the model is, when the file says.
//
// Flutter: no counterpart
type Units struct {
	// Name is "m", "mm", "in" and so on.
	Name string
	// MetersPerUnit is the length of a unit in meters; 0 when the file does not say.
	MetersPerUnit float64
}

// Model is what a file holds.
//
// Flutter: no counterpart
type Model struct {
	// Name is the name in the file, or empty.
	Name string
	// Nodes are the top of the tree: the nodes of the default scene.
	Nodes []*Node
	// Meshes, Materials, Textures and Images are every one of the model, each once; the nodes and the meshes point to them. A mesh of
	// a glTF file is one Mesh for each of its primitives.
	Meshes    []*Mesh
	Materials []*Material
	Textures  []*Texture
	Images    []*Image
	// Skins, Cameras and Lights are every one of the model, and Animations are its animations (glTF, §28.5).
	Skins      []*Skin
	Cameras    []*Camera
	Lights     []*Light
	Animations []*Animation
	// Scenes are the scenes of the file, and DefaultScene the one that Nodes is; nil when the file has none (Nodes are then its nodes
	// that no node has as a child).
	Scenes       []*Scene
	DefaultScene *Scene
	// Units are the units of the positions, when the file says.
	Units Units
}

// Bounds is the box of the model in the coordinates of its top nodes, in float64: the boxes of the meshes of every node, moved by
// the transforms from the node up to the top. It is empty for a model with no geometry.
//
// Flutter: no counterpart
func (m *Model) Bounds() vecmath.Box3 {
	b := vecmath.EmptyBox3()
	var walk func(n *Node, to vecmath.Mat4)
	walk = func(n *Node, to vecmath.Mat4) {
		to = to.Mul(n.Transform)
		for _, mesh := range n.Meshes {
			if mb := mesh.Bounds(); !mb.IsEmpty() {
				b = b.Union(mb.Transform(to))
			}
		}
		for _, c := range n.Children {
			walk(c, to)
		}
	}
	for _, n := range m.Nodes {
		walk(n, vecmath.Mat4Identity())
	}
	return b
}

// TriangleCount is the number of triangles of the meshes that the nodes use, a mesh that several nodes use counted for each.
//
// Flutter: no counterpart
func (m *Model) TriangleCount() int {
	n := 0
	var walk func(nd *Node)
	walk = func(nd *Node) {
		for _, mesh := range nd.Meshes {
			if mesh.Primitive == PrimitiveTriangles {
				n += mesh.PrimitiveCount()
			}
		}
		for _, c := range nd.Children {
			walk(c)
		}
	}
	for _, nd := range m.Nodes {
		walk(nd)
	}
	return n
}
