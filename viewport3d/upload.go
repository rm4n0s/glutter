package viewport3d

import (
	"context"
	"log/slog"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
)

// TagUploadFailed is the tag of the error of Upload when the context is done before it has made all the items. The cause is the
// context's error.
//
// Flutter: no counterpart
const TagUploadFailed = "ModelUploadFailed"

// UploadOpts are the optional parameters of Upload.
//
// Flutter: no counterpart
type UploadOpts struct {
	// FirstPickID, when it is not 0, is the PickID of the first item; the next ones have the next numbers, so that picking tells
	// which part of the model was hit (UploadedPart tells which item has which number). The default (0) gives no item a PickID.
	FirstPickID uint32
	// Layers is the layer mask of the items; the default (0) is all of them.
	Layers uint32
	// Logger gets the notes of Upload (an image that cannot be decoded); slog.Default() when nil.
	Logger *slog.Logger
}

// UploadedPart is an item of an Uploaded and where it comes from.
//
// Flutter: no counterpart
type UploadedPart struct {
	// Item draws Mesh with the material of the model's mesh, at the transform of Node and its parents.
	Item *Item
	// Node and Mesh are the node and the mesh of the model that the item is made of.
	Node *model.Node
	Mesh *model.Mesh
}

// Uploaded is what Upload made of a model.
//
// Flutter: no counterpart
type Uploaded struct {
	// Parts are the items, one for each mesh of each node of the model, in the order of the tree.
	Parts []UploadedPart
	// Bounds is the box of the model in the coordinates of its top nodes.
	Bounds vecmath.Box3

	meshes    []*Mesh
	materials []Material
	textures  []*gpu.Texture
}

// Items are the items of the parts, to be added to a scene.
//
// Flutter: no counterpart
func (u *Uploaded) Items() []*Item {
	items := make([]*Item, len(u.Parts))
	for i, p := range u.Parts {
		items[i] = p.Item
	}
	return items
}

// Release frees the meshes, the materials and the textures that Upload made. The items are not removed from their scene: do that
// first (Scene.Remove), as the items draw nothing without them.
//
// Flutter: no counterpart
func (u *Uploaded) Release() {
	for _, m := range u.meshes {
		m.Release()
	}
	for _, m := range u.materials {
		m.Release()
	}
	for _, t := range u.textures {
		t.Release()
	}
	u.meshes, u.materials, u.textures, u.Parts = nil, nil, nil, nil
}

// Upload makes the meshes, the materials and the items of a model (SPEC §27.9): a mesh of the scene for each mesh of the model,
// shared by the nodes that use it; a PhongMaterial (or a LineMaterial or a PointMaterial, by the primitive of the mesh) for each
// material, with a texture for the base color image, which is decoded here (ui.InstantiateImageCodec) and made on the device; and an
// item for each mesh of each node, at the product of the transforms from the top node down to it, in float64. Positions in float64
// become the origin of the mesh in float64 and positions in float32 relative to it (D30). Add the items to a scene; release the
// result when the scene no longer shows them.
//
// It fails with MeshInvalid for a mesh of the model that is not valid (an index outside the vertices, ...), and with the errors of
// the device when a texture cannot be made. An image that cannot be decoded is left out, with a line in the log.
//
// Flutter: no counterpart
func Upload(ctx context.Context, device *gpu.Device, m *model.Model, opts ...UploadOpts) (*Uploaded, error) {
	o := firstOpts(opts)
	log := o.Logger
	if log == nil {
		log = slog.Default()
	}
	u := &Uploaded{}
	meshes := map[*model.Mesh]*Mesh{}
	type matKey struct {
		mat  *model.Material
		prim model.Primitive
	}
	materials := map[matKey]Material{}
	textures := map[*model.Image]*gpu.Texture{}

	texture := func(img *model.Image) (*gpu.Texture, error) {
		if t, ok := textures[img]; ok {
			return t, nil
		}
		textures[img] = nil
		pix, w, h, err := decodeImage(ctx, img.Data)
		if err != nil {
			log.Warn("viewport3d: an image of the model could not be decoded, the material has no texture", "image", img.Name, "err", err)
			return nil, nil
		}
		t, err := device.NewTexture(gpu.TextureDesc{Label: img.Name, Size: gpu.Extent3D{Width: w, Height: h, DepthOrArrayLayers: 1},
			Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageTextureBinding})
		if err != nil {
			return nil, err
		}
		device.Queue().WriteTexture(gpu.TextureCopy{Texture: t}, pix, gpu.DataLayout{BytesPerRow: 4 * w}, gpu.Extent3D{Width: w, Height: h, DepthOrArrayLayers: 1})
		textures[img] = t
		u.textures = append(u.textures, t)
		return t, nil
	}
	material := func(mm *model.Mesh) (Material, error) {
		key := matKey{mm.Material, mm.Primitive}
		if mat, ok := materials[key]; ok {
			return mat, nil
		}
		c := mm.Material.Color()
		color := ui.NewColorFromARGB(unit8(c[3]), unit8(c[0]), unit8(c[1]), unit8(c[2]))
		var mat Material
		switch mm.Primitive {
		case model.PrimitiveLines:
			mat = NewLineMaterial(LineOpts{Color: color})
		case model.PrimitivePoints:
			mat = NewPointMaterial(PointOpts{Color: color})
		default:
			po := PhongOpts{Color: color, VertexColors: mm.Colors != nil}
			if mm.Material != nil {
				po.DoubleSided = mm.Material.DoubleSided
				po.Transparent = mm.Material.AlphaMode == model.AlphaBlend || c[3] < 1
				if mm.Material.Shininess > 0 {
					po.Shininess = float64(mm.Material.Shininess)
				}
				if s := mm.Material.Specular; s != [3]float32{} {
					po.Specular = ui.NewColorFromARGB(255, unit8(s[0]), unit8(s[1]), unit8(s[2]))
				}
				if mm.Material.BaseColorImage != nil {
					t, err := texture(mm.Material.BaseColorImage)
					if err != nil {
						return nil, err
					}
					po.Texture = t
				}
			}
			mat = NewPhongMaterial(po)
		}
		materials[key] = mat
		u.materials = append(u.materials, mat)
		return mat, nil
	}
	fail := func(err error) (*Uploaded, error) {
		u.Release()
		return nil, err
	}

	var walk func(n *model.Node, parent vecmath.Mat4) error
	walk = func(n *model.Node, parent vecmath.Mat4) error {
		if err := ctx.Err(); err != nil {
			return errors.NewErr(TagUploadFailed, err).SetMetadata("node", n.Name)
		}
		to := parent.Mul(n.Transform)
		for _, mm := range n.Meshes {
			if mm.PrimitiveCount() == 0 {
				continue
			}
			mesh, ok := meshes[mm]
			if !ok {
				var err error
				if mesh, err = newMesh(mm); err != nil {
					return err
				}
				meshes[mm] = mesh
				u.meshes = append(u.meshes, mesh)
			}
			mat, err := material(mm)
			if err != nil {
				return err
			}
			io := ItemOpts{Transform: foundation.Some(to), Layers: o.Layers}
			if o.FirstPickID != 0 {
				io.PickID = o.FirstPickID + uint32(len(u.Parts))
			}
			u.Parts = append(u.Parts, UploadedPart{Item: NewItem(mesh, mat, io), Node: n, Mesh: mm})
		}
		for _, c := range n.Children {
			if err := walk(c, to); err != nil {
				return err
			}
		}
		return nil
	}
	for _, n := range m.Nodes {
		if err := walk(n, vecmath.Mat4Identity()); err != nil {
			return fail(err)
		}
	}
	u.Bounds = m.Bounds()
	return u, nil
}

func unit8(f float32) int {
	return int(min(max(f, 0), 1)*255 + 0.5)
}

// newMesh makes the mesh of the scene of a mesh of the model: positions in float64 become the origin and offsets from it.
func newMesh(mm *model.Mesh) (*Mesh, error) {
	data := MeshData{
		Primitive: primitiveOf(mm.Primitive), Positions: mm.Positions, Normals: mm.Normals, TexCoords: mm.TexCoords, Colors: mm.Colors,
		Indices32: mm.Indices,
	}
	if mm.Positions64 != nil {
		b := mm.Bounds()
		if b.IsEmpty() {
			b = vecmath.NewBox3(vecmath.Vec3{}, vecmath.Vec3{})
		}
		origin := b.Center()
		data.Origin = origin
		data.Positions = make([]float32, len(mm.Positions64))
		for i := 0; i < len(mm.Positions64); i += 3 {
			data.Positions[i] = float32(mm.Positions64[i] - origin.X)
			data.Positions[i+1] = float32(mm.Positions64[i+1] - origin.Y)
			data.Positions[i+2] = float32(mm.Positions64[i+2] - origin.Z)
		}
	}
	return NewMesh(data)
}

// decodeImage decodes an encoded image into not premultiplied RGBA bytes.
func decodeImage(ctx context.Context, data []byte) (pix []byte, w, h int, err error) {
	codec, err := ui.InstantiateImageCodec(ctx, data)
	if err != nil {
		return nil, 0, 0, err
	}
	defer codec.Dispose()
	frame, err := codec.GetNextFrame(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	img := frame.Image
	defer img.Dispose()
	src := img.Pixels()
	pix = make([]byte, len(src))
	copy(pix, src)
	// The pixels of an image are premultiplied; the material multiplies by the texture as it is.
	for i := 0; i+3 < len(pix); i += 4 {
		if a := uint32(pix[i+3]); a != 0 && a != 255 {
			pix[i] = byte(min(uint32(pix[i])*255/a, 255))
			pix[i+1] = byte(min(uint32(pix[i+1])*255/a, 255))
			pix[i+2] = byte(min(uint32(pix[i+2])*255/a, 255))
		}
	}
	return pix, img.Width(), img.Height(), nil
}

// primitiveOf is the primitive of the scene for a primitive of the model.
func primitiveOf(p model.Primitive) Primitive {
	switch p {
	case model.PrimitiveLines:
		return PrimitiveLines
	case model.PrimitivePoints:
		return PrimitivePoints
	}
	return PrimitiveTriangles
}
