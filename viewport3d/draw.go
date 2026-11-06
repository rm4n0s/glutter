package viewport3d

import (
	"cmp"
	"log/slog"
	"math/bits"
	"slices"

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
	mods struct{ mesh, line, point *gpu.ShaderModule }
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
	lines  []lineDraw
	colors [1]gpu.RenderPassColorAttachment

	stats       FrameStats
	warnedDepth bool
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
	for _, g := range []*gpu.BindGroup{r.frameGroup, r.itemGroup} {
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
	for _, m := range []*gpu.ShaderModule{r.mods.mesh, r.mods.line, r.mods.point} {
		if m != nil {
			m.Release()
		}
	}
	r.frameGroup, r.itemGroup, r.uniforms, r.instances, r.corners, r.pointCorners, r.sampler, r.white = nil, nil, nil, nil, nil, nil, nil, nil
	r.uniformCap, r.instanceCap = 0, 0
	r.mods.mesh, r.mods.line, r.mods.point = nil, nil, nil
	r.dev = nil
}

// Stats says what the last frame did.
//
// Flutter: no counterpart
func (r *Renderer) Stats() FrameStats { return r.stats }

// modelView computes the matrix that takes the vertices of an item (relative to its mesh's origin) to the view, in float64 with
// the eye taken off the translation, and gives it as float32 (D30).
func modelView(rot *vecmath.Mat4, eye vecmath.Vec3, model *vecmath.Mat4, out *[16]float32) {
	rel := *model
	rel[12] -= eye.X
	rel[13] -= eye.Y
	rel[14] -= eye.Z
	m := rot.Mul(rel)
	for i, v := range m {
		out[i] = float32(v)
	}
}

// ensureCapacity grows the uniform and instance buffers for the frame, and makes the bind groups that go with the uniforms.
func (r *Renderer) ensureCapacity(uniformBytes, instanceBytes int, groupPipe, itemPipe *gpu.RenderPipeline) bool {
	if uniformBytes > r.uniformCap {
		if r.uniforms != nil {
			r.uniforms.Release()
			r.uniforms = nil
		}
		for _, g := range []*gpu.BindGroup{r.frameGroup, r.itemGroup} {
			if g != nil {
				g.Release()
			}
		}
		r.frameGroup, r.itemGroup = nil, nil
		c := max(4096, uniformBytes*2)
		b, err := r.dev.NewBuffer(gpu.BufferDesc{Label: "viewport3d uniforms", Size: c, Usage: gpu.BufferUsageUniform | gpu.BufferUsageCopyDst})
		if err != nil {
			slog.Error("viewport3d: the uniform buffer could not be made", "err", err)
			return false
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
			slog.Error("viewport3d: the instance buffer could not be made", "err", err)
			return false
		}
		r.instances, r.instanceCap = b, c
	}
	var err error
	if r.frameGroup == nil && groupPipe != nil {
		r.frameGroup, err = r.dev.NewBindGroup(gpu.BindGroupDesc{Label: "viewport3d frame", Layout: groupPipe.BindGroupLayout(0),
			Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: r.uniforms, Size: frameUniformSize}}})
		if err != nil {
			slog.Error("viewport3d: the frame bind group could not be made", "err", err)
			return false
		}
	}
	if r.itemGroup == nil && itemPipe != nil {
		r.itemGroup, err = r.dev.NewBindGroup(gpu.BindGroupDesc{Label: "viewport3d items", Layout: itemPipe.BindGroupLayout(2),
			Entries: []gpu.BindGroupEntry{{Binding: 0, Buffer: r.uniforms, Size: itemUniformSize}}})
		if err != nil {
			slog.Error("viewport3d: the item bind group could not be made", "err", err)
			return false
		}
	}
	return true
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

	r.items, r.opaque, r.transp, r.lines = r.items[:0], r.opaque[:0], r.transp[:0], r.lines[:0]
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
	for _, it := range r.scene.items {
		if !it.visible || it.layers&r.layers == 0 {
			r.stats.Hidden++
			continue
		}
		if it.sphere.Radius > 0 && !frustum.IntersectsSphere(it.sphere) {
			r.stats.Culled++
			continue
		}
		modelView(&rot, eye, &it.model, &mv)
		switch it.mesh.data.Primitive {
		case PrimitiveTriangles:
			if faces {
				mesh := r.meshState(it.mesh)
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
				mat := r.materialState(material, basePipe)
				if mat.group == nil {
					continue
				}
				pk := triBase
				pk.variant = mesh.variant
				pk.noCull = s.doubleSided
				blend := s.Transparent() && mode != DisplayModeHiddenLine
				pk.blend = blend
				d := drawItem{mesh: mesh, mat: mat, pk: pk, mv: mv}
				if blend {
					d.dist = it.sphere.Center.DistanceSqTo(eye)
					r.items = append(r.items, d)
					r.transp = append(r.transp, int32(len(r.items)-1))
				} else {
					sig := uint64(pk.variant)
					if pk.noCull {
						sig |= 8
					}
					d.key = sig<<56 | uint64(mat.id&0xFFFFFF)<<32 | uint64(mesh.id)
					r.items = append(r.items, d)
					r.opaque = append(r.opaque, int32(len(r.items)-1))
				}
				r.stats.Items++
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
				r.writeRecord(records, &mv)
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
			r.writeRecord(records, &mv)
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
			r.writeRecord(records, &mv)
			records++
			r.stats.Items++
		}
	}

	// The instances: the opaque ones sorted by batch, then the transparent ones back to front.
	slices.SortFunc(r.opaque, func(a, b int32) int { return cmp.Compare(r.items[a].key, r.items[b].key) })
	slices.SortFunc(r.transp, func(a, b int32) int { return cmp.Compare(r.items[b].dist, r.items[a].dist) })
	n := len(r.opaque) + len(r.transp)
	if need := n * instanceStride; len(r.ibytes) < need {
		r.ibytes = slices.Grow(r.ibytes[:0], need)[:need]
	}
	next := int32(0)
	for _, list := range [2][]int32{r.opaque, r.transp} {
		for _, i := range list {
			putMat(r.ibytes, int(next)*instanceStride, &r.items[i].mv)
			r.items[i].instance = next
			next++
		}
	}

	itemPipe := linePipe
	if itemPipe == nil {
		itemPipe = pointPipe
	}
	uniformBytes := itemBase + records*uniformAlign
	if !r.ensureCapacity(uniformBytes, max(n, 1)*instanceStride, basePipe, itemPipe) {
		return
	}
	r.writeFrame(f, style, cam, &rot, dpr)
	q := r.dev.Queue()
	q.WriteBuffer(r.uniforms, 0, r.ubytes[:uniformBytes])
	if n > 0 {
		q.WriteBuffer(r.instances, 0, r.ibytes[:n*instanceStride])
	}

	bg := style.background
	clearColor := gpu.Color{R: float64(bg.R * bg.A), G: float64(bg.G * bg.A), B: float64(bg.B * bg.A), A: float64(bg.A)}
	r.colors[0] = gpu.RenderPassColorAttachment{View: f.Color.View(), ClearValue: clearColor}
	if f.Resolve != nil {
		r.colors[0].ResolveTarget = f.Resolve.View()
		r.colors[0].StoreOp = gpu.StoreOpDiscard
	}
	pass := f.Encoder.BeginRenderPass(gpu.RenderPassDesc{
		Label:                  "viewport3d",
		ColorAttachments:       r.colors[:],
		DepthStencilAttachment: &gpu.RenderPassDepthStencilAttachment{View: f.Depth.View(), DepthClearValue: 0},
	})
	pass.SetBindGroup(0, r.frameGroup, 0)
	var curMat *matGPU
	var curMesh *meshGPU
	var curPipe *gpu.RenderPipeline
	for _, list := range [2][]int32{r.opaque, r.transp} {
		for i := 0; i < len(list); {
			d := &r.items[list[i]]
			count := 1
			// The items that share the pipeline, the material and the mesh follow each other in the opaque list: one draw.
			for j := i + 1; j < len(list) && !d.pk.blend && r.items[list[j]].key == d.key; j++ {
				count++
			}
			pipe := r.pipeline(d.pk)
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
	pass.End()
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
	switch {
	case g.index == nil:
	case g.wide:
		pass.SetIndexBuffer32(g.index, 0)
	default:
		pass.SetIndexBuffer(g.index, 0)
	}
}

// writeRecord puts the model-view matrix of a line or point draw in the uniform scratch, for record n.
func (r *Renderer) writeRecord(n int, mv *[16]float32) {
	need := itemBase + (n+1)*uniformAlign
	if len(r.ubytes) < need {
		r.ubytes = slices.Grow(r.ubytes[:len(r.ubytes)], need-len(r.ubytes))[:need]
	}
	putMat(r.ubytes, itemBase+n*uniformAlign, mv)
}

// writeFrame writes the frame's uniforms (the first 272 bytes of the uniform scratch).
func (r *Renderer) writeFrame(f *gpu.ViewFrame, style *resolvedStyle, cam *Camera, rot *vecmath.Mat4, dpr float64) {
	if len(r.ubytes) < itemBase {
		r.ubytes = slices.Grow(r.ubytes[:len(r.ubytes)], itemBase-len(r.ubytes))[:itemBase]
	}
	b := r.ubytes
	proj := cam.ProjectionMatrix().Float32()
	putMat(b, 0, &proj)
	// The light at the camera shines along the line of sight; the others are turned into the view's space.
	putVec4(b, 64, 0, 0, 1, 0)
	h := float32(style.headLight)
	putVec4(b, 80, h, h, h, 0)
	for i := 0; i < maxLights; i++ {
		off := 96 + 32*i
		if i >= len(style.lights) {
			putVec4(b, off, 0, 0, 1, 0)
			putVec4(b, off+16, 0, 0, 0, 0)
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
		putVec4(b, off, float32(d.X), float32(d.Y), float32(d.Z), 0)
		putVec4(b, off+16, float32(c.R)*float32(k), float32(c.G)*float32(k), float32(c.B)*float32(k), 0)
	}
	a := float32(style.ambient)
	putVec4(b, 224, a, a, a, 0)
	w, hh := float32(f.Width), float32(f.Height)
	putVec4(b, 240, w, hh, 1/w, 1/hh)
	putVec4(b, 256, float32(cam.Near()), float32(dpr), 0, 0)
}
