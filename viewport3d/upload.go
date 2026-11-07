package viewport3d

import (
	"context"
	"log/slog"
	"math/bits"

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
	// Transform is the transform of the whole model in the world, in float64 (applied after the nodes'); the default is the
	// identity. Uploaded.Bounds stays in the model's own coordinates.
	Transform foundation.Opt[vecmath.Mat4]
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

	node int
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
	samplers  []*gpu.Sampler
	// nodes are the nodes of the tree, parents before their children, with the world transform that SetPose last gave them; skins
	// are the skeletons of the model, with the place of each joint in nodes.
	nodes  []uploadNode
	skins  []uploadSkin
	worlds []vecmath.Mat4
	// root is the transform of the whole model (UploadOpts.Transform).
	root vecmath.Mat4
}

// uploadNode is a node of the tree of an Uploaded.
type uploadNode struct {
	node   *model.Node
	parent int
	world  vecmath.Mat4
}

// uploadSkin is the skeleton that an Uploaded made of a model's skin, and the joints as places in the list of nodes.
type uploadSkin struct {
	model  *model.Skin
	skin   *Skin
	joints []int
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
	for _, sm := range u.samplers {
		sm.Release()
	}
	u.meshes, u.materials, u.textures, u.samplers, u.Parts, u.nodes, u.skins = nil, nil, nil, nil, nil, nil, nil
}

// SetPose shows a pose (Animation.Sample) of the model: the nodes get the local transforms of the pose where it has them (the
// ones of the model elsewhere) and the world transforms from them, the items follow their nodes, the skeletons get the world
// transforms of their joints, and the items with morph targets get the weights of the pose (those of their node or their mesh
// where it has none). A nil pose is the model at rest. Call it each frame an animation plays; a frame that changes nothing
// draws nothing again.
//
// Flutter: no counterpart
func (u *Uploaded) SetPose(p *Pose) {
	for i := range u.nodes {
		n := &u.nodes[i]
		local := p.Matrix(n.node)
		if n.parent >= 0 {
			n.world = u.nodes[n.parent].world.Mul(local)
		} else {
			n.world = u.root.Mul(local)
		}
	}
	for i := range u.Parts {
		part := &u.Parts[i]
		part.Item.SetTransform(u.nodes[part.node].world)
		if n := part.Mesh.MorphTargets; len(n) > 0 {
			w := p.Weights(part.Node)
			if w == nil {
				w = part.Node.MorphWeights
			}
			if w == nil {
				w = part.Mesh.MorphWeights
			}
			part.Item.SetMorphWeights(w)
		}
	}
	for i := range u.skins {
		sk := &u.skins[i]
		u.worlds = u.worlds[:0]
		for _, j := range sk.joints {
			u.worlds = append(u.worlds, u.nodes[j].world)
		}
		sk.skin.SetJoints(u.worlds)
	}
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
	u := &Uploaded{root: o.Transform.OrElse(vecmath.Mat4Identity())}
	meshes := map[*model.Mesh]*Mesh{}
	type matKey struct {
		mat  *model.Material
		prim model.Primitive
		// moves is whether the mesh has a skin or morph targets: with no material, only a PBR one can move it.
		moves bool
		// colors is whether the mesh has vertex colors, which a mesh of points shows only with a material that says so.
		colors bool
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
		levels := mipLevels(w, h)
		t, err := device.NewTexture(gpu.TextureDesc{Label: img.Name, Size: gpu.Extent3D{Width: w, Height: h, DepthOrArrayLayers: 1},
			MipLevelCount: levels, Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageTextureBinding})
		if err != nil {
			return nil, err
		}
		for l := 0; l < levels; l++ {
			device.Queue().WriteTexture(gpu.TextureCopy{Texture: t, MipLevel: l}, pix, gpu.DataLayout{BytesPerRow: 4 * w}, gpu.Extent3D{Width: w, Height: h, DepthOrArrayLayers: 1})
			if l+1 < levels {
				pix, w, h = halve(pix, w, h)
			}
		}
		textures[img] = t
		u.textures = append(u.textures, t)
		return t, nil
	}
	samplers := map[model.Sampler]*gpu.Sampler{}
	sampler := func(s model.Sampler) (*gpu.Sampler, error) {
		if sm, ok := samplers[s]; ok {
			return sm, nil
		}
		sm, err := device.NewSampler(samplerDesc(s))
		if err != nil {
			return nil, err
		}
		samplers[s] = sm
		u.samplers = append(u.samplers, sm)
		return sm, nil
	}
	material := func(mm *model.Mesh) (Material, error) {
		key := matKey{mm.Material, mm.Primitive, mm.Joints != nil || len(mm.MorphTargets) > 0, mm.Primitive == model.PrimitivePoints && mm.Colors != nil}
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
			mat = NewPointMaterial(PointOpts{Color: color, VertexColors: mm.Colors != nil})
		default:
			if mm.Material != nil && mm.Material.PBR {
				pm, err := pbrMaterial(mm, texture, sampler)
				if err != nil {
					return nil, err
				}
				mat = pm
				break
			}
			if mm.Material == nil && (mm.Joints != nil || len(mm.MorphTargets) > 0) {
				// Only the PBR material moves a skin and morph targets: a mesh with them and no material gets the default look in it.
				mat = NewPBRMaterial(PBROpts{Color: ui.NewColorFromARGB(255, 200, 204, 209), Roughness: foundation.Some(0.6)})
				break
			}
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

	nodeIndex := map[*model.Node]int{}
	skinOf := map[*model.Skin]*Skin{}
	var walk func(n *model.Node, parent vecmath.Mat4, parentIndex int) error
	walk = func(n *model.Node, parent vecmath.Mat4, parentIndex int) error {
		if err := ctx.Err(); err != nil {
			return errors.NewErr(TagUploadFailed, err).SetMetadata("node", n.Name)
		}
		to := parent.Mul(n.Transform)
		index := len(u.nodes)
		nodeIndex[n] = index
		u.nodes = append(u.nodes, uploadNode{node: n, parent: parentIndex, world: to})
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
			item := NewItem(mesh, mat, io)
			if n.Skin != nil && mm.Joints != nil {
				sk := skinOf[n.Skin]
				if sk == nil {
					sk = NewSkin(n.Skin.InverseBindMatrices)
					skinOf[n.Skin] = sk
					u.skins = append(u.skins, uploadSkin{model: n.Skin, skin: sk})
				}
				item.SetSkin(sk)
			}
			u.Parts = append(u.Parts, UploadedPart{Item: item, Node: n, Mesh: mm, node: index})
		}
		for _, c := range n.Children {
			if err := walk(c, to, index); err != nil {
				return err
			}
		}
		return nil
	}
	for _, n := range m.Nodes {
		if err := walk(n, u.root, -1); err != nil {
			return fail(err)
		}
	}
	// The joints of the skeletons are places in the list of nodes; a joint that the tree of the scene does not have is a node of its
	// own, with its own transform.
	for i := range u.skins {
		for _, j := range u.skins[i].model.Joints {
			idx, ok := nodeIndex[j]
			if !ok {
				idx = len(u.nodes)
				nodeIndex[j] = idx
				u.nodes = append(u.nodes, uploadNode{node: j, parent: -1, world: u.root.Mul(j.Transform)})
			}
			u.skins[i].joints = append(u.skins[i].joints, idx)
		}
	}
	u.SetPose(nil)
	u.Bounds = m.Bounds()
	return u, nil
}

// mipLevels is the number of levels of the chain of an image, down to one texel.
func mipLevels(w, h int) int { return bits.Len(uint(max(w, h))) }

// halve makes the next level of a chain: each texel is the average of four, as the image's channels are (not premultiplied; the
// alpha counts as a channel too).
func halve(pix []byte, w, h int) ([]byte, int, int) {
	nw, nh := max(w/2, 1), max(h/2, 1)
	out := make([]byte, 4*nw*nh)
	for y := 0; y < nh; y++ {
		y0, y1 := min(2*y, h-1), min(2*y+1, h-1)
		for x := 0; x < nw; x++ {
			x0, x1 := min(2*x, w-1), min(2*x+1, w-1)
			for c := 0; c < 4; c++ {
				sum := int(pix[4*(y0*w+x0)+c]) + int(pix[4*(y0*w+x1)+c]) + int(pix[4*(y1*w+x0)+c]) + int(pix[4*(y1*w+x1)+c])
				out[4*(y*nw+x)+c] = byte((sum + 2) / 4)
			}
		}
	}
	return out, nw, nh
}

// samplerDesc is the sampler of a model's: the wrap modes and the filters (linear unless the model says nearest).
func samplerDesc(s model.Sampler) gpu.SamplerDesc {
	mode := func(w model.Wrap) gpu.AddressMode {
		switch w {
		case model.WrapMirroredRepeat:
			return gpu.AddressModeMirrorRepeat
		case model.WrapClampToEdge:
			return gpu.AddressModeClampToEdge
		}
		return gpu.AddressModeRepeat
	}
	filter := func(f model.SamplerFilter) gpu.FilterMode {
		if f == model.FilterNearest {
			return gpu.FilterModeNearest
		}
		return gpu.FilterModeLinear
	}
	return gpu.SamplerDesc{Label: "viewport3d model", AddressModeU: mode(s.WrapU), AddressModeV: mode(s.WrapV), MagFilter: filter(s.Mag), MinFilter: filter(s.Min),
		MipmapFilter: filter(s.Mipmap)}
}

// pbrMaterial makes the PBRMaterial of a glTF material, with its textures and samplers.
func pbrMaterial(mm *model.Mesh, texture func(*model.Image) (*gpu.Texture, error), sampler func(model.Sampler) (*gpu.Sampler, error)) (*PBRMaterial, error) {
	m := mm.Material
	ref := func(r *model.TextureRef) (PBRTexture, error) {
		if r == nil || r.Texture == nil || r.Texture.Image == nil {
			return PBRTexture{}, nil
		}
		t, err := texture(r.Texture.Image)
		if err != nil || t == nil {
			return PBRTexture{}, err
		}
		sm, err := sampler(r.Texture.Sampler)
		if err != nil {
			return PBRTexture{}, err
		}
		out := PBRTexture{Texture: t, Sampler: sm, TexCoord: r.TexCoord}
		if r.Transform != nil {
			out.Transform = &TextureTransform{Offset: r.Transform.Offset, Rotation: r.Transform.Rotation, Scale: r.Transform.Scale}
		}
		return out, nil
	}
	o := PBROpts{
		// The factor of a glTF material is linear; the material takes sRGB.
		Color:     ui.Color{A: m.BaseColor[3], R: linearToSRGB(m.BaseColor[0]), G: linearToSRGB(m.BaseColor[1]), B: linearToSRGB(m.BaseColor[2])},
		Metallic:  foundation.Some(float64(m.Metallic)),
		Roughness: foundation.Some(float64(m.Roughness)), Emissive: [3]float64{float64(m.Emissive[0]), float64(m.Emissive[1]), float64(m.Emissive[2])},
		NormalScale: foundation.Some(float64(m.NormalScale)), OcclusionStrength: foundation.Some(float64(m.OcclusionStrength)),
		AlphaMode: AlphaMode(m.AlphaMode), AlphaCutoff: foundation.Some(float64(m.AlphaCutoff)),
		DoubleSided: m.DoubleSided, VertexColors: mm.Colors != nil, Unlit: m.Unlit,
	}
	if o.Color == (ui.Color{}) {
		o.Color = ui.Color{R: 0, G: 0, B: 0, A: 1e-6}
	}
	var err error
	for _, t := range []struct {
		from *model.TextureRef
		to   *PBRTexture
	}{{m.BaseColorTexture, &o.BaseColorTexture}, {m.MetallicRoughnessTexture, &o.MetallicRoughnessTexture}, {m.NormalTexture, &o.NormalTexture},
		{m.OcclusionTexture, &o.OcclusionTexture}, {m.EmissiveTexture, &o.EmissiveTexture}} {
		if *t.to, err = ref(t.from); err != nil {
			return nil, err
		}
	}
	return NewPBRMaterial(o), nil
}

func unit8(f float32) int {
	return int(min(max(f, 0), 1)*255 + 0.5)
}

// newMesh makes the mesh of the scene of a mesh of the model: positions in float64 become the origin and offsets from it.
func newMesh(mm *model.Mesh) (*Mesh, error) {
	data := MeshData{
		Primitive: primitiveOf(mm.Primitive), Positions: mm.Positions, Normals: mm.Normals, TexCoords: mm.TexCoords, Colors: mm.Colors,
		Indices32: mm.Indices, Tangents: mm.Tangents, TexCoords1: mm.TexCoords1, Joints: mm.Joints, Weights: mm.Weights,
	}
	for _, t := range mm.MorphTargets {
		data.MorphTargets = append(data.MorphTargets, MorphTarget{Positions: t.Positions, Normals: t.Normals})
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
