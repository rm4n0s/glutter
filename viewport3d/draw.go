package viewport3d

import (
	"cmp"
	"image"
	"log/slog"
	"math"
	"math/bits"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
)

// edgePull is how far, relative to their depth, the feature edges are pulled towards the camera, so that an edge is not hidden
// by the faces it lies on (z-fighting): a hundred-thousandth, which is the precision of a 24-bit depth at a quarter of the range.
const edgePull = 3e-5

// RendererOpts are the optional parameters of NewRenderer.
//
// Flutter: no counterpart
type RendererOpts struct {
	// Layers is the bit mask of the layers that the renderer draws: an item is drawn when its layer mask has a bit in common
	// with it. The default (0) is all of them.
	Layers uint32
}

// FrameStats says what the last frame of a renderer did.
//
// Flutter: no counterpart
type FrameStats struct {
	// Items is the number of items drawn, Culled the number that were outside the view and Hidden those that were not visible or
	// in another layer.
	Items, Culled, Hidden int
	// Draws is the number of draw calls, Batches the number of those that are batches of triangles (one draw for the items that
	// share a mesh and a material), and Instances the number of instances of triangles.
	Draws, Batches, Instances int
	// EdgeDraws is the number of draws of the edges of meshes (one for each item that has edges drawn).
	EdgeDraws int
	// CapDraws is the number of caps drawn (one for each closed item and each plane that cuts it) and OutlineDraws the number of
	// draws of the outlines of the cuts.
	CapDraws, OutlineDraws int
	// GridDraws is the number of grids drawn (0 or 1).
	GridDraws int
	// Pipelines is the number of pipelines the renderer has made.
	Pipelines int
}

// Renderer draws a Scene with a Camera into the targets of a gpu.ViewFrame: the meshes with instancing, the materials, the
// transparent items back to front and the display modes (SPEC §27.8). View uses one in a gpu.View; an app that draws into its
// own gpu.View (or a texture) makes its own and calls Init and Dispose from the view's OnInit and OnDispose, and Draw from its
// paint function. The view needs a depth target (Depth32Float) and may be multisampled. A renderer works on the UI thread.
//
// Flutter: no counterpart
type Renderer struct {
	scene  *Scene
	camera *Camera
	layers uint32

	dev  *gpu.Device
	mods struct{ mesh, line, point, section, pick, sel, grid, pbr, shadow, tone, env *gpu.ShaderModule }
	// pipes are the pipelines by key; a key whose pipeline failed has nil.
	pipes        map[pipeKey]*gpu.RenderPipeline
	corners      *gpu.Buffer
	pointCorners *gpu.Buffer
	sampler      *gpu.Sampler
	white        *gpu.Texture

	uniforms    *gpu.Buffer
	uniformCap  int
	instances   *gpu.Buffer
	instanceCap int
	frameGroup  *gpu.BindGroup
	itemGroup   *gpu.BindGroup
	// sectionGroup is the group of the record of an item for the section pipelines, which have no material and so use group 1.
	sectionGroup *gpu.BindGroup

	meshes []*Mesh
	mats   []*materialCore
	nextID uint32

	// fill is the material of the faces in the hidden line mode, and edges the one of the feature edges.
	fill  *UnlitMaterial
	edges *LineMaterial

	// The scratch of a frame, kept so that a steady frame allocates nothing.
	segs   []float32
	ubytes []byte
	ibytes []byte
	items  []drawItem
	opaque []int32
	transp []int32
	// popaque and ptransp are the items of PBR materials, drawn into the float target (pbrdraw.go).
	popaque []int32
	ptransp []int32
	lines   []lineDraw
	sect    []sectionDraw
	// selItems are the selected items that have faces, for the outline of the selection.
	selItems []*Item
	colors   [1]gpu.RenderPassColorAttachment

	stats       FrameStats
	warnedDepth bool
	// mask is the texture that the outline of the selection is made from, with its size and the group that reads it.
	mask         *gpu.Texture
	maskGroup    *gpu.BindGroup
	maskW, maskH int
	// picks are the picks that wait for the next frame (pick.go).
	picks         []pickRequest
	warnedStencil bool

	// pbr is what the PBR materials need: the environment, the shadow map, the float target and the skins. pending are the resources
	// that the commands of the last frame name, which are let go of when the next frame starts.
	pbr     pbrState
	pending []releaser
}

// drawItem is an item of triangles in the frame's draw list.
type drawItem struct {
	mesh *meshGPU
	mat  *matGPU
	pk   pipeKey
	// key sorts the opaque items so that those that share the pipeline, the material and the mesh are next to each other.
	key uint64
	// dist is the squared distance from the eye to the item, for sorting transparent items back to front.
	dist float64
	mv   [16]float32
	// instance is the item's index in the instance buffer.
	instance int32
	// selected is whether the item is selected, and tint the highlight that its instance carries (rgb, and how much of it).
	selected bool
	tint     [4]float32
	// pbr says that the item has a PBR material, whose instance carries extra (the numbers of its skin and morph targets, pbrExtra).
	pbr   bool
	extra [4]float32
}

// lineDraw is a draw of line segments or points: the edges of a mesh, an item of lines, or an item of points.
type lineDraw struct {
	mesh        *meshGPU
	mat         *matGPU
	pk          pipeKey
	record      int
	instanceCnt int
	// buf is the buffer that holds the instances: the segments, or the positions of the points.
	buf *gpu.Buffer
	// corners is the buffer of the quad's corners.
	corners *gpu.Buffer
	edge    bool
}

// NewRenderer creates a renderer of the scene as the camera sees it. It makes nothing on the GPU until Init gives it the
// device.
//
// Flutter: no counterpart
func NewRenderer(scene *Scene, camera *Camera, opts ...RendererOpts) *Renderer {
	o := firstOpts(opts)
	r := &Renderer{scene: scene, camera: camera, layers: o.Layers, pipes: map[pipeKey]*gpu.RenderPipeline{}}
	if r.layers == 0 {
		r.layers = AllLayers
	}
	r.fill = NewUnlitMaterial()
	r.edges = NewLineMaterial()
	r.edges.pull = edgePull
	return r
}

// Init gives the renderer the device it draws with: call it from gpu.ViewOpts.OnInit, which runs again after a device is lost
// and replaced. The resources are made when they are first needed.
//
// Flutter: no counterpart
func (r *Renderer) Init(d *gpu.Device) {
	if r.dev != nil && r.dev != d {
		r.Dispose()
	}
	r.dev = d
}

// Dispose releases everything the renderer holds on the GPU: pipelines, buffers, bind groups, and the GPU copies of the meshes
// and the materials that it drew (the meshes and the materials stay usable). Call it from gpu.ViewOpts.OnDispose. The renderer
// can be given a device again with Init.
//
// Flutter: no counterpart
func (r *Renderer) Dispose() {
	r.failPicks()
	r.releaseMask()
	r.pbr.release(r)
	r.releasePending()
	for len(r.meshes) > 0 {
		r.forgetMesh(r.meshes[len(r.meshes)-1])
	}
	for len(r.mats) > 0 {
		r.forgetMaterial(r.mats[len(r.mats)-1])
	}
	for _, p := range r.pipes {
		if p != nil {
			p.Release()
		}
	}
	clear(r.pipes)
	for _, g := range []*gpu.BindGroup{r.frameGroup, r.itemGroup, r.sectionGroup} {
		if g != nil {
			g.Release()
		}
	}
	for _, b := range []*gpu.Buffer{r.uniforms, r.instances, r.corners, r.pointCorners} {
		if b != nil {
			b.Release()
		}
	}
	if r.sampler != nil {
		r.sampler.Release()
	}
	if r.white != nil {
		r.white.Release()
	}
	for _, m := range []*gpu.ShaderModule{r.mods.mesh, r.mods.line, r.mods.point, r.mods.section, r.mods.pick, r.mods.sel, r.mods.grid, r.mods.pbr, r.mods.shadow, r.mods.tone, r.mods.env} {
		if m != nil {
			m.Release()
		}
	}
	r.frameGroup, r.itemGroup, r.sectionGroup, r.uniforms, r.instances, r.corners, r.pointCorners, r.sampler, r.white = nil, nil, nil, nil, nil, nil, nil, nil, nil
	r.uniformCap, r.instanceCap = 0, 0
	r.mods.mesh, r.mods.line, r.mods.point, r.mods.section, r.mods.pick, r.mods.sel, r.mods.grid = nil, nil, nil, nil, nil, nil, nil
	r.mods.pbr, r.mods.shadow, r.mods.tone, r.mods.env = nil, nil, nil, nil
	r.dev = nil
}

// Stats says what the last frame did.
//
// Flutter: no counterpart
func (r *Renderer) Stats() FrameStats { return r.stats }

// modelView computes the matrix that takes the vertices of an item (relative to its mesh's origin) to the view, in float64 with
// the eye taken off the translation, and gives it as float32 (D30).
func modelView(rot *vecmath.Mat4, eye vecmath.Vec3, model *vecmath.Mat4, out *[16]float32) {
	// Both matrices are affine (the last row is 0, 0, 0, 1) for every item but a degenerate one: the product then needs the three
	// upper rows only, which is 36 products instead of 64, and the bottom row is known.
	if rot[3] == 0 && rot[7] == 0 && rot[11] == 0 && rot[15] == 1 && model[3] == 0 && model[7] == 0 && model[11] == 0 && model[15] == 1 {
		tx, ty, tz := model[12]-eye.X, model[13]-eye.Y, model[14]-eye.Z
		for c := 0; c < 3; c++ {
			b0, b1, b2 := model[c*4], model[c*4+1], model[c*4+2]
			out[c*4] = float32(rot[0]*b0 + rot[4]*b1 + rot[8]*b2)
			out[c*4+1] = float32(rot[1]*b0 + rot[5]*b1 + rot[9]*b2)
			out[c*4+2] = float32(rot[2]*b0 + rot[6]*b1 + rot[10]*b2)
			out[c*4+3] = 0
		}
		out[12] = float32(rot[0]*tx + rot[4]*ty + rot[8]*tz + rot[12])
		out[13] = float32(rot[1]*tx + rot[5]*ty + rot[9]*tz + rot[13])
		out[14] = float32(rot[2]*tx + rot[6]*ty + rot[10]*tz + rot[14])
		out[15] = 1
		return
	}
	rel := *model
	rel[12] -= eye.X
	rel[13] -= eye.Y
	rel[14] -= eye.Z
	m := rot.Mul(rel)
	for i, v := range m {
		out[i] = float32(v)
	}
}

// frustumHoldsBox says whether the whole box is inside the frustum (all its corners are on the inner side of every plane). An empty
// box is not.
func frustumHoldsBox(f *vecmath.Frustum, b vecmath.Box3) bool {
	if b.IsEmpty() {
		return false
	}
	for i := range f.Planes {
		pl := &f.Planes[i]
		for c := 0; c < 8; c++ {
			corner := vecmath.Vec3{X: b.Min.X, Y: b.Min.Y, Z: b.Min.Z}
			if c&1 != 0 {
				corner.X = b.Max.X
			}
			if c&2 != 0 {
				corner.Y = b.Max.Y
			}
			if c&4 != 0 {
				corner.Z = b.Max.Z
			}
			if pl.DistanceTo(corner) < 0 {
				return false
			}
		}
	}
	return true
}

// ensureCapacity grows the uniform and instance buffers for the frame, and makes the bind groups that go with the uniforms.
func (r *Renderer) ensureCapacity(uniformBytes, instanceBytes int, groupPipe, itemPipe, sectionPipe *gpu.RenderPipeline) error {
	if uniformBytes > r.uniformCap {
		if r.uniforms != nil {
			r.uniforms.Release()
			r.uniforms = nil
		}
		for _, g := range []*gpu.BindGroup{r.frameGroup, r.itemGroup, r.sectionGroup} {
			if g != nil {
				g.Release()
			}
		}
		r.frameGroup, r.itemGroup, r.sectionGroup = nil, nil, nil
		c := max(4096, uniformBytes*2)
		b, err := r.dev.NewBuffer(gpu.BufferDesc{Label: "viewport3d uniforms", Size: c, Usage: gpu.BufferUsageUniform | gpu.BufferUsageCopyDst})
		if err != nil {
			return err
		}
		r.uniforms, r.uniformCap = b, c
	}
	if instanceBytes > r.instanceCap {
		if r.instances != nil {
			r.instances.Release()
			r.instances = nil
		}
		c := max(64*instanceStride, instanceBytes*2)
		b, err := r.dev.NewBuffer(gpu.BufferDesc{Label: "viewport3d instances", Size: c, Usage: gpu.BufferUsageVertex | gpu.BufferUsageCopyDst})
		if err != nil {
			return err
		}
		r.instances, r.instanceCap = b, c
	}
	var err error
	if r.frameGroup == nil && groupPipe != nil {
		r.frameGroup, err = r.dev.NewBindGroup(gpu.BindGroupDesc{Label: "viewport3d frame", Layout: groupPipe.BindGroupLayout(0),
			Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: r.uniforms, Size: frameUniformSize}}})
		if err != nil {
			return err
		}
	}
	if r.itemGroup == nil && itemPipe != nil {
		r.itemGroup, err = r.dev.NewBindGroup(gpu.BindGroupDesc{Label: "viewport3d items", Layout: itemPipe.BindGroupLayout(2),
			Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: r.uniforms, Size: itemUniformSize}}})
		if err != nil {
			return err
		}
	}
	if r.sectionGroup == nil && sectionPipe != nil {
		r.sectionGroup, err = r.dev.NewBindGroup(gpu.BindGroupDesc{Label: "viewport3d sections", Layout: sectionPipe.BindGroupLayout(1),
			Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: r.uniforms, Size: itemUniformSize}}})
		if err != nil {
			return err
		}
	}
	return nil
}

// Draw draws the scene into the targets of the frame, with the frame's encoder: a pass that clears them to the scene's background
// and draws the opaque faces (a batch of instances for the items that share a mesh and a material), the transparent faces back to
// front, the edges of the display mode, the line items and the point items. It makes the pipelines, the buffers and the GPU
// copies of the meshes and materials when they are first needed.
//
// Flutter: no counterpart
func (r *Renderer) Draw(f *gpu.ViewFrame) {
	r.stats = FrameStats{Pipelines: len(r.pipes)}
	if r.dev == nil {
		return
	}
	r.releasePending()
	if f.Depth == nil {
		if !r.warnedDepth {
			r.warnedDepth = true
			slog.Error("viewport3d: the view has no depth target: give gpu.ViewOpts a DepthFormat")
		}
		return
	}
	if f.Width <= 0 || f.Height <= 0 {
		return
	}
	dpr := f.DevicePixelRatio
	if dpr <= 0 {
		dpr = 1
	}
	cam, style := r.camera, &r.scene.rs
	cam.SetViewportSize(ui.Size{Width: float64(f.Width) / dpr, Height: float64(f.Height) / dpr})
	if err := r.ensureShared(); err != nil {
		slog.Error("viewport3d: the shared resources could not be made", "err", err)
		return
	}
	base := pipeKey{samples: f.Color.SampleCount(), color: f.Color.Format(), depth: f.Depth.Format()}
	triBase := base
	triBase.kind = pipeTriangles
	basePipe := r.pipeline(triBase)
	if basePipe == nil {
		return
	}

	frustum := cam.Frustum()
	eye := cam.Eye()
	rot := cam.RelativeViewMatrix()
	mode := style.mode
	faces := mode == DisplayModeShaded || mode == DisplayModeShadedWithEdges || mode == DisplayModeHiddenLine
	edges := mode != DisplayModeShaded
	if mode == DisplayModeHiddenLine {
		r.fill.SetColor(style.background)
	}
	r.edges.SetColor(style.edgeColor)
	r.edges.SetWidth(style.edgeWidth)

	r.items, r.opaque, r.transp, r.lines, r.sect, r.selItems = r.items[:0], r.opaque[:0], r.transp[:0], r.lines[:0], r.sect[:0], r.selItems[:0]
	r.popaque, r.ptransp = r.popaque[:0], r.ptransp[:0]
	r.pbr.reset()
	dc := drawCtx{style: style, base: base, eye: eye, planes: len(r.scene.planes)}
	if faces && style.shadow.enabled && mode != DisplayModeHiddenLine {
		dc.shadowOn = r.shadowSetup(style)
	}
	// The section planes cut the faces, the lines and the points; the caps need a stencil in the depth target.
	planes := len(r.scene.planes)
	capsOK := style.section.cap
	if capsOK && f.Depth.Format() != gpu.TextureFormatDepth24PlusStencil8 {
		capsOK = false
		if planes > 0 && !r.warnedStencil {
			r.warnedStencil = true
			slog.Warn("viewport3d: the view has no stencil, so the sections have no caps: give gpu.ViewOpts the DepthFormat Depth24PlusStencil8 (viewport3d.NewView does)")
		}
	}
	dc.capsOK = capsOK
	// linePipe and pointPipe give the layouts of the groups of lines and points; each draw looks up the pipeline of its own key.
	var linePipe, pointPipe *gpu.RenderPipeline
	needLine := func() *gpu.RenderPipeline {
		if linePipe == nil {
			lk := base
			lk.kind = pipeLines
			linePipe = r.pipeline(lk)
		}
		return linePipe
	}
	records := 0
	var mv [16]float32
	// A scene that is wholly in view needs no test of its items. The last mesh and material are kept: a batch is many items that
	// share them.
	inView := frustumHoldsBox(&frustum, r.scene.Bounds())
	var lastMesh *Mesh
	var lastMeshGPU *meshGPU
	var lastMaterial Material
	var lastMatGPU *matGPU
	for _, it := range r.scene.items {
		if !it.visible || it.layers&r.layers == 0 {
			r.stats.Hidden++
			continue
		}
		// A skinned item or one with morph targets moves out of the box of its mesh at rest: it is never culled by it.
		if !inView && it.sphere.Radius > 0 && it.skin == nil && len(it.morphWeights) == 0 && !frustum.IntersectsSphere(it.sphere) {
			r.stats.Culled++
			continue
		}
		modelView(&rot, eye, &it.model, &mv)
		switch it.mesh.data.Primitive {
		case PrimitiveTriangles:
			// An item of a PBR material is drawn into the float target (pbrdraw.go); its edges are drawn as the others'.
			pbrItem := false
			if faces && mode != DisplayModeHiddenLine {
				if pm, ok := it.material.(*PBRMaterial); ok {
					pbrItem = true
					if it.mesh != lastMesh {
						lastMesh, lastMeshGPU = it.mesh, r.meshState(it.mesh)
					}
					if lastMeshGPU.pos != nil {
						r.addPBRItem(it, pm, lastMeshGPU, &mv, &dc)
					}
				}
			}
			if faces && !pbrItem {
				if it.mesh != lastMesh {
					lastMesh, lastMeshGPU = it.mesh, r.meshState(it.mesh)
				}
				mesh := lastMeshGPU
				if mesh.pos == nil {
					continue
				}
				material := Material(it.material)
				if mode == DisplayModeHiddenLine {
					material = r.fill
				}
				s := surfaceOf(material)
				if s == nil {
					continue
				}
				if material != lastMaterial {
					lastMaterial, lastMatGPU = material, r.materialState(material, basePipe)
				}
				mat := lastMatGPU
				if mat.group == nil {
					continue
				}
				pk := triBase
				pk.variant = mesh.variant
				pk.noCull = s.doubleSided
				blend := s.Transparent() && mode != DisplayModeHiddenLine
				pk.blend = blend
				// The item is built in its place in the list: it is large, and copying it twice shows in a frame of 10,000.
				r.items = append(r.items, drawItem{mesh: mesh, mat: mat, pk: pk, mv: mv})
				index := int32(len(r.items) - 1)
				d := &r.items[index]
				if it.selected {
					d.selected, d.tint = true, tintOf(&style.selection)
					if style.selection.outline {
						r.selItems = append(r.selItems, it)
					}
				}
				if blend {
					d.dist = it.sphere.Center.DistanceSqTo(eye)
					r.transp = append(r.transp, index)
				} else {
					sig := uint64(pk.variant)
					if pk.noCull {
						sig |= 8
					}
					if d.selected {
						sig |= 16
					}
					d.key = sig<<56 | uint64(mat.id&0xFFFFFF)<<32 | uint64(mesh.id)
					r.opaque = append(r.opaque, index)
				}
				r.stats.Items++
				if dc.shadowOn && !blend {
					r.addCaster(it, mesh, false, [4]float32{})
				}
				if planes > 0 && mode != DisplayModeHiddenLine {
					sd := sectionDraw{mesh: it.mesh, g: mesh, mv: mv}
					if capsOK && !blend && it.mesh.Closed() {
						sd.capped = true
						sd.cap = style.section.capColor.OrElse(s.color)
					}
					r.sect = append(r.sect, sd)
				}
			}
			if edges {
				var em *Mesh
				if mode == DisplayModeWireframe {
					em = allEdges(it.mesh)
				} else {
					em, _ = FeatureEdges(it.mesh, style.edgeAngle)
				}
				lp := needLine()
				if em == nil || lp == nil {
					continue
				}
				seg := r.meshState(em)
				if seg.segments == nil || seg.count == 0 {
					continue
				}
				mat := r.materialState(r.edges, lp)
				pk := base
				pk.kind, pk.noDepthTest = pipeLines, mode == DisplayModeWireframe
				r.lines = append(r.lines, lineDraw{mesh: seg, mat: mat, pk: pk, record: records, instanceCnt: seg.count, buf: seg.segments, corners: r.corners, edge: true})
				r.writeRecord(records, &mv, [4]float32{})
				records++
			}
		case PrimitiveLines:
			lp := needLine()
			if lp == nil {
				continue
			}
			seg := r.meshState(it.mesh)
			if seg.segments == nil || seg.count == 0 {
				continue
			}
			mat := r.materialState(it.material, lp)
			pk := base
			pk.kind = pipeLines
			r.lines = append(r.lines, lineDraw{mesh: seg, mat: mat, pk: pk, record: records, instanceCnt: seg.count, buf: seg.segments, corners: r.corners})
			r.writeRecord(records, &mv, r.lineTint(it, style))
			records++
			r.stats.Items++
		case PrimitivePoints:
			pk := base
			pk.kind = pipePoints
			if pointPipe = r.pipeline(pk); pointPipe == nil {
				continue
			}
			pts := r.meshState(it.mesh)
			if pts.pos == nil || pts.count == 0 {
				continue
			}
			mat := r.materialState(it.material, pointPipe)
			r.lines = append(r.lines, lineDraw{mesh: pts, mat: mat, pk: pk, record: records, instanceCnt: pts.count, buf: pts.pos, corners: r.pointCorners})
			r.writeRecord(records, &mv, r.lineTint(it, style))
			records++
			r.stats.Items++
		}
	}

	// The instances: the opaque ones sorted by batch, then the transparent ones back to front.
	byKey := func(a, b int32) int { return cmp.Compare(r.items[a].key, r.items[b].key) }
	if !slices.IsSortedFunc(r.opaque, byKey) { // a scene of one batch is sorted already
		slices.SortFunc(r.opaque, byKey)
	}
	slices.SortFunc(r.transp, func(a, b int32) int { return cmp.Compare(r.items[b].dist, r.items[a].dist) })
	r.sortPBR()
	pbrItems := len(r.popaque) + len(r.ptransp)
	if pbrItems == 0 {
		dc.shadowOn = false
	}
	casters := 0
	if dc.shadowOn {
		casters = len(r.pbr.simple) + len(r.pbr.full)
	}
	n := len(r.opaque) + len(r.transp) + pbrItems + casters
	if need := n * instanceStride; len(r.ibytes) < need {
		r.ibytes = slices.Grow(r.ibytes[:0], need)[:need]
	}
	next := int32(0)
	for _, list := range [4][]int32{r.opaque, r.transp, r.popaque, r.ptransp} {
		for _, i := range list {
			d := &r.items[i]
			putMat(r.ibytes, int(next)*instanceStride, &d.mv)
			putVec4(r.ibytes, int(next)*instanceStride+64, d.tint[0], d.tint[1], d.tint[2], d.tint[3])
			if d.pbr {
				putVec4(r.ibytes, int(next)*instanceStride+80, d.extra[0], d.extra[1], d.extra[2], d.extra[3])
			}
			d.instance = next
			next++
		}
	}
	if dc.shadowOn {
		for _, list := range [2][]shadowCaster{r.pbr.simple, r.pbr.full} {
			for i := range list {
				c := &list[i]
				putMat(r.ibytes, int(next)*instanceStride, &c.mv)
				putVec4(r.ibytes, int(next)*instanceStride+64, 0, 0, 0, 0)
				putVec4(r.ibytes, int(next)*instanceStride+80, c.extra[0], c.extra[1], c.extra[2], c.extra[3])
				c.instance = next
				next++
			}
		}
	}

	// The outline of the selection: a record with its color and radius, and the rectangle that it covers.
	selRecord, selRadius := -1, 0
	var selPipe *gpu.RenderPipeline
	var selRect image.Rectangle
	if len(r.selItems) > 0 {
		selRadius = min(max(int(math.Round(style.selection.width*dpr)), 1), maxOutlineRadius)
		sp := base
		sp.kind = pipeSelect
		if selPipe = r.pipeline(sp); selPipe != nil {
			if selRect = r.selectionRect(cam, r.selItems, selRadius, f.Width, f.Height, dpr); selRect.Empty() {
				selPipe = nil
			}
		}
		if selPipe != nil {
			selRecord = records
			records++
			off := r.record(selRecord)
			ident := vecmath.Mat4Identity().Float32()
			putMat(r.ubytes, off, &ident)
			c := style.selection.color
			putVec4(r.ubytes, off+64, float32(c.R), float32(c.G), float32(c.B), float32(c.A))
			putVec4(r.ubytes, off+80, float32(selRadius), 0, 0, 0)
			putVec4(r.ubytes, off+96, 0, 0, 0, 0)
		}
	}
	sectionBase := records
	var sectionPipe *gpu.RenderPipeline
	if planes > 0 && len(r.sect) > 0 {
		sk := base
		sk.kind = pipeOutline
		if sectionPipe = r.pipeline(sk); sectionPipe == nil {
			r.sect = r.sect[:0]
		} else {
			r.writeSectionRecords(records, style)
			records += planes * len(r.sect)
		}
	}
	// The grid: a record, and a pipeline that gives the layout of the group of the records when nothing else does.
	gridRecord := -1
	if style.grid.show {
		gk := base
		gk.kind = pipeGrid
		if gp := r.pipeline(gk); gp != nil {
			gridRecord = records
			records++
			putGrid(r.ubytes, r.record(gridRecord), &style.grid, &rot, eye, dpr)
			if sectionPipe == nil {
				sectionPipe = gp
			}
		}
	}
	// The picks that wait for this frame: a pass for each, after the frame's own.
	var picks *pickPlan
	pickEnd := 0
	if len(r.picks) > 0 {
		p := r.pipeline(pickKey(pipePickTri))
		if p == nil {
			failed := r.picks
			r.picks = nil
			for i := range failed {
				failed[i].fail(errors.New(TagPickUnavailable, "the pipeline of the pick could not be made"))
			}
		} else if sectionPipe == nil {
			sectionPipe = p
		}
	}
	if len(r.picks) > 0 {
		picks, pickEnd = r.planPicks(&pickCtx{f: f, cam: cam, rot: &rot, eye: eye, frustum: frustum, dpr: dpr, base: base, style: style}, &records)
	}
	itemPipe := linePipe
	if itemPipe == nil {
		itemPipe = pointPipe
	}
	uniformBytes := max(itemBase+records*uniformAlign, pickEnd)
	if err := r.ensureCapacity(uniformBytes, max(n, 1)*instanceStride, basePipe, itemPipe, sectionPipe); err != nil {
		slog.Error("viewport3d: the buffers of the frame could not be made", "err", err)
		if picks != nil {
			for _, j := range picks.jobs {
				j.req.fail(err)
			}
		}
		return
	}
	r.writeFrame(f, style, cam, &rot, dpr)
	if len(r.ubytes) < uniformBytes {
		r.ubytes = slices.Grow(r.ubytes[:len(r.ubytes)], uniformBytes-len(r.ubytes))[:uniformBytes]
	}
	q := r.dev.Queue()
	q.WriteBuffer(r.uniforms, 0, r.ubytes[:uniformBytes])
	if n > 0 {
		q.WriteBuffer(r.instances, 0, r.ibytes[:n*instanceStride])
	}

	if selPipe != nil && !r.ensureMask(f.Width, f.Height, selPipe) {
		selPipe = nil
	}
	if selPipe != nil {
		r.drawMask(f)
	} else if r.mask != nil {
		r.releaseMask()
	}

	bg := clearColor(style.background)
	// The PBR items are drawn first, with their tone mapping over the background; the pass below then draws the other items over
	// them and loads what is there.
	loadOp := gpu.LoadOpClear
	if pbrItems > 0 && r.drawPBR(f, &dc, &rot, bg) {
		loadOp = gpu.LoadOpLoad
	}
	r.colors[0] = gpu.RenderPassColorAttachment{View: f.Color.View(), LoadOp: loadOp, ClearValue: bg}
	if f.Resolve != nil {
		r.colors[0].ResolveTarget = f.Resolve.View()
		r.colors[0].StoreOp = gpu.StoreOpDiscard
	}
	pass := f.Encoder.BeginRenderPass(gpu.RenderPassDesc{
		Label:            "viewport3d",
		ColorAttachments: r.colors[:],
		DepthStencilAttachment: &gpu.RenderPassDepthStencilAttachment{View: f.Depth.View(), DepthLoadOp: loadOp, StencilLoadOp: loadOp,
			DepthClearValue: 0, StencilClearValue: 0},
	})
	pass.SetBindGroup(0, r.frameGroup, 0)
	var curMat *matGPU
	var curMesh *meshGPU
	var curPipe *gpu.RenderPipeline
	for li, list := range [2][]int32{r.opaque, r.transp} {
		if li == 1 {
			// The caps, the outlines of the cuts and the grid, over the opaque faces and under the transparent ones.
			if planes > 0 && len(r.sect) > 0 {
				r.drawSections(pass, base, sectionBase, planes)
			}
			if gridRecord >= 0 {
				r.drawGrid(pass, base, gridRecord)
			}
			curMat, curMesh, curPipe = nil, nil, nil
		}
		var lastKey pipeKey
		var lastKeyPipe *gpu.RenderPipeline
		keyKnown := false
		for i := 0; i < len(list); {
			d := &r.items[list[i]]
			count := 1
			// The items that share the pipeline, the material and the mesh follow each other in the opaque list: one draw.
			for j := i + 1; j < len(list) && !d.pk.blend && r.items[list[j]].key == d.key; j++ {
				count++
			}
			if !keyKnown || d.pk != lastKey {
				lastKey, lastKeyPipe, keyKnown = d.pk, r.pipeline(d.pk), true
			}
			pipe := lastKeyPipe
			if pipe == nil {
				i += count
				continue
			}
			if pipe != curPipe {
				pass.SetPipeline(pipe)
				curPipe, curMat, curMesh = pipe, nil, nil
			}
			if curMat != d.mat {
				pass.SetBindGroup(1, d.mat.group, 0)
				curMat = d.mat
			}
			if curMesh != d.mesh {
				r.bindMesh(pass, d.mesh)
				curMesh = d.mesh
			}
			pass.SetVertexBuffer(1+bits.OnesCount8(d.mesh.variant), r.instances, int(d.instance)*instanceStride)
			if d.mesh.index != nil {
				pass.DrawIndexed(d.mesh.count, count, 0, 0, 0)
			} else {
				pass.Draw(d.mesh.count, count, 0, 0)
			}
			r.stats.Draws++
			r.stats.Batches++
			r.stats.Instances += count
			i += count
		}
	}
	// The edges, the lines and the points, over the faces.
	var lastPipe *gpu.RenderPipeline
	var lastMat *matGPU
	for i := range r.lines {
		l := &r.lines[i]
		pipe := r.pipeline(l.pk)
		if pipe == nil {
			continue
		}
		if pipe != lastPipe {
			pass.SetPipeline(pipe)
			lastPipe, lastMat = pipe, nil
		}
		if lastMat != l.mat {
			pass.SetBindGroup(1, l.mat.group, 0)
			lastMat = l.mat
		}
		pass.SetBindGroup(2, r.itemGroup, uint32(itemBase+l.record*uniformAlign))
		pass.SetVertexBuffer(0, l.buf, 0)
		pass.SetVertexBuffer(1, l.corners, 0)
		pass.Draw(6, l.instanceCnt, 0, 0)
		r.stats.Draws++
		if l.edge {
			r.stats.EdgeDraws++
		}
	}
	if selPipe != nil {
		// The outline of the selection, over everything.
		pass.SetPipeline(selPipe)
		pass.SetBindGroup(1, r.maskGroup, uint32(itemBase+selRecord*uniformAlign))
		pass.SetScissorRect(selRect.Min.X, selRect.Min.Y, selRect.Dx(), selRect.Dy())
		pass.Draw(3, 1, 0, 0)
	}
	pass.End()
	if picks != nil {
		r.drawPicks(f, picks)
	}
	r.stats.Pipelines = len(r.pipes)
}

// bindMesh binds the vertex buffers of a mesh of triangles: the position, then the normal, texture coordinates and color it has,
// and its indices.
func (r *Renderer) bindMesh(pass *gpu.RenderPass, g *meshGPU) {
	slot := 0
	pass.SetVertexBuffer(slot, g.pos, 0)
	slot++
	if g.normals != nil {
		pass.SetVertexBuffer(slot, g.normals, 0)
		slot++
	}
	if g.uvs != nil {
		pass.SetVertexBuffer(slot, g.uvs, 0)
		slot++
	}
	if g.colors != nil {
		pass.SetVertexBuffer(slot, g.colors, 0)
	}
	r.setIndices(pass, g)
}

// setIndices binds the indices of a mesh, when it has any.
func (r *Renderer) setIndices(pass *gpu.RenderPass, g *meshGPU) {
	switch {
	case g.index == nil:
	case g.wide:
		pass.SetIndexBuffer32(g.index, 0)
	default:
		pass.SetIndexBuffer(g.index, 0)
	}
}

// lineTint is the tint of the record of a line or point item: the highlight color when it is selected, else nothing.
func (r *Renderer) lineTint(it *Item, style *resolvedStyle) [4]float32 {
	if !it.selected {
		return [4]float32{}
	}
	t := tintOf(&style.selection)
	t[3] = 1
	return t
}

// writeRecord puts the model-view matrix of a line or point draw in the uniform scratch, for record n.
func (r *Renderer) writeRecord(n int, mv *[16]float32, tint [4]float32) {
	off := r.record(n)
	putMat(r.ubytes, off, mv)
	// The tint, and the numbers of the shader that reads the record.
	putVec4(r.ubytes, off+64, tint[0], tint[1], tint[2], tint[3])
	putVec4(r.ubytes, off+80, 0, 0, 0, 0)
	putVec4(r.ubytes, off+96, 0, 0, 0, 0)
}

// writeFrame writes the frame's uniforms (the first 272 bytes of the uniform scratch).
func (r *Renderer) writeFrame(f *gpu.ViewFrame, style *resolvedStyle, cam *Camera, rot *vecmath.Mat4, dpr float64) {
	proj := cam.ProjectionMatrix().Float32()
	r.writeFrameAt(0, &proj, float32(f.Width), float32(f.Height), style, cam, rot, dpr)
}

// writeFrameAt writes the frame's uniform data at the offset of the uniform scratch, for the projection and the size of the
// target in pixels: the frame of the picking has its own.
func (r *Renderer) writeFrameAt(off int, proj *[16]float32, w, h float32, style *resolvedStyle, cam *Camera, rot *vecmath.Mat4, dpr float64) {
	if need := off + frameUniformSize; len(r.ubytes) < need {
		r.ubytes = slices.Grow(r.ubytes[:len(r.ubytes)], need-len(r.ubytes))[:need]
	}
	if len(r.ubytes) < itemBase {
		r.ubytes = slices.Grow(r.ubytes[:len(r.ubytes)], itemBase-len(r.ubytes))[:itemBase]
	}
	b := r.ubytes[off : off+frameUniformSize]
	putMat(b, 0, proj)
	// The light at the camera shines along the line of sight; the others are turned into the view's space.
	putVec4(b, 64, 0, 0, 1, 0)
	hl := float32(style.headLight)
	putVec4(b, 80, hl, hl, hl, 0)
	for i := 0; i < maxLights; i++ {
		lo := 96 + 32*i
		if i >= len(style.lights) {
			putVec4(b, lo, 0, 0, 1, 0)
			putVec4(b, lo+16, 0, 0, 0, 0)
			continue
		}
		l := style.lights[i]
		d := rot.TransformDirection(l.Direction.Normalize())
		c := l.Color
		if c == (ui.Color{}) {
			c = ui.NewColor(0xFFFFFFFF)
		}
		k := l.Intensity
		if k == 0 {
			k = 1
		}
		putVec4(b, lo, float32(d.X), float32(d.Y), float32(d.Z), 0)
		putVec4(b, lo+16, float32(c.R)*float32(k), float32(c.G)*float32(k), float32(c.B)*float32(k), 0)
	}
	a := float32(style.ambient)
	putVec4(b, 224, a, a, a, 0)
	putVec4(b, 240, w, h, 1/w, 1/h)
	putVec4(b, 256, float32(cam.Near()), float32(dpr), 0, 0)
	putSectionPlanes(b, r.scene.planes, rot, cam.Eye())
}
