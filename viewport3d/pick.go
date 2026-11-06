package viewport3d

import (
	"encoding/binary"
	"image"
	"log/slog"
	"math"
	"slices"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
)

// TagPickUnavailable is the tag of the error of a pick that cannot be made: the view has no renderer or no GPU device, or the
// renderer was disposed while the pick waited for its frame.
//
// Flutter: no counterpart
const TagPickUnavailable = "PickUnavailable"

// pickFrameStride is the room of the frame of one pick pass in the uniform buffer (the frame's data, up to the alignment).
const pickFrameStride = 2 * uniformAlign

// defaultPickRadius is how far from the point of Pick, in logical pixels, a primitive may be to be picked.
const defaultPickRadius = 2.0

// PickOpts are the optional parameters of Pick.
//
// Flutter: no counterpart
type PickOpts struct {
	// Radius is how far from the point, in logical pixels, a primitive may be to be picked, so that a line of one pixel or a
	// point can be hit; the nearest to the point wins. The default is 2.
	Radius float64
}

// PickResult is what is under a point of the view (SPEC §27.8).
//
// Flutter: no counterpart
type PickResult struct {
	// Hit is whether any item was within the radius of the point; the other fields are empty when it is not.
	Hit bool
	// Item is the item that was hit, and PickID its pick ID.
	Item   *Item
	PickID uint32
	// Primitive says whether a triangle, a segment or a point of the item's mesh was hit, and Index which one (the order of the
	// mesh's triangles, segments or points).
	Primitive Primitive
	Index     int
	// Point is the point of the world that was hit, in float64, made from the depth and the inverse matrices (so it is as exact
	// as the depth, far from the origin too), Normal the unit normal of the triangle that was hit, facing the camera (zero for a
	// segment and a point), and Depth the depth of the pixel, reversed-Z.
	Point, Normal vecmath.Vec3
	Depth         float64
	// At is the center of the pixel that was hit, in logical pixels of the view.
	At ui.Offset
}

// PickRectResult is what is in a rectangle of the view: each item that has a pixel there that is not hidden by another item, once,
// in the order of the scene.
//
// Flutter: no counterpart
type PickRectResult struct {
	Items []*Item
}

// pickRequest is a pick that waits for the next frame.
type pickRequest struct {
	at     ui.Offset
	radius float64
	rect   ui.Rect
	single bool
	done   func(PickResult, error)
	doneN  func(PickRectResult, error)
}

// fail tells the request's function of an error.
func (q *pickRequest) fail(err error) {
	if q.single {
		q.done(PickResult{}, err)
	} else {
		q.doneN(PickRectResult{}, err)
	}
}

// Pick asks for what is under a point of the view, in logical pixels. The items are drawn with their numbers into a small target
// around the point, in the next frame only (never every frame), and read back; done is called, from the event loop, with the result
// once the GPU has finished, which is in the frame after the pick's own. An error is the read of the GPU (GPUReadFailed, a lost
// device) or PickUnavailable.
//
// Flutter: no counterpart
func (r *Renderer) Pick(at ui.Offset, done func(PickResult, error), opts ...PickOpts) {
	o := firstOpts(opts)
	if o.Radius <= 0 {
		o.Radius = defaultPickRadius
	}
	q := pickRequest{at: at, radius: o.Radius, single: true, done: done}
	if r.dev == nil {
		q.fail(errors.New(TagPickUnavailable, "the renderer has no GPU device", "point", at.String()))
		return
	}
	r.picks = append(r.picks, q)
}

// PickRect asks for the items that are in a rectangle of the view, in logical pixels, as Pick does: for a selection with a box.
//
// Flutter: no counterpart
func (r *Renderer) PickRect(rect ui.Rect, done func(PickRectResult, error)) {
	q := pickRequest{rect: rect, doneN: done}
	if r.dev == nil {
		q.fail(errors.New(TagPickUnavailable, "the renderer has no GPU device", "rect", rect.String()))
		return
	}
	r.picks = append(r.picks, q)
}

// failPicks tells the picks that wait for a frame that they will not get one.
func (r *Renderer) failPicks() {
	picks := r.picks
	r.picks = nil
	for i := range picks {
		picks[i].fail(errors.New(TagPickUnavailable, "the renderer was disposed before the frame of the pick"))
	}
}

// pickDraw is an item of the pick pass.
type pickDraw struct {
	item   *Item
	g      *meshGPU
	mat    *matGPU
	record int
}

// pickJob is one pick of a frame: its rectangle of pixels, where its frame's data is, and what the camera was.
type pickJob struct {
	req     pickRequest
	rect    image.Rectangle
	frame   int
	invProj vecmath.Mat4
	invRot  vecmath.Mat4
	eye     vecmath.Vec3
	width   int
	height  int
	dpr     float64
	// items are the items of the pass, the number of an item being its index plus one.
	items []*Item
}

// pickPlan is what Draw works out for the picks of a frame before the uniform data is written.
type pickPlan struct {
	draws []pickDraw
	jobs  []*pickJob
}

// pickCtx is what the planning of the picks needs of the frame.
type pickCtx struct {
	f       *gpu.ViewFrame
	cam     *Camera
	rot     *vecmath.Mat4
	eye     vecmath.Vec3
	frustum vecmath.Frustum
	dpr     float64
	base    pipeKey
	style   *resolvedStyle
}

// pickKey is the key of a pick pipeline.
func pickKey(k pipeKind) pipeKey {
	return pipeKey{kind: k, samples: 1, color: gpu.TextureFormatRG32Uint, depth: gpu.TextureFormatDepth32Float}
}

// planPicks makes the pick pass's items and its jobs: the records of the items are written from record *records on, and each
// job has the frame data of its own after them. It leaves in *records the number of records used, and returns the plan with the
// offset in the uniform buffer where the frames of the jobs end.
func (r *Renderer) planPicks(c *pickCtx, records *int) (*pickPlan, int) {
	plan := &pickPlan{}
	linePipe, pointPipe := r.pipeline(pickKey(pipePickLine)), r.pipeline(pickKey(pipePickPoint))
	var mv [16]float32
	var items []*Item
	for _, it := range r.scene.items {
		if !it.visible || it.layers&r.layers == 0 {
			continue
		}
		if it.sphere.Radius > 0 && !c.frustum.IntersectsSphere(it.sphere) {
			continue
		}
		d := pickDraw{item: it}
		switch it.mesh.data.Primitive {
		case PrimitiveTriangles:
			d.g = r.meshState(it.mesh)
			if d.g.pos == nil || !r.cornersOf(it.mesh, d.g) || d.g.triangles == 0 {
				continue
			}
		case PrimitiveLines:
			d.g = r.meshState(it.mesh)
			if linePipe == nil || d.g.segments == nil || d.g.count == 0 {
				continue
			}
			d.mat = r.materialState(it.material, linePipe)
		default:
			d.g = r.meshState(it.mesh)
			if pointPipe == nil || d.g.pos == nil || d.g.count == 0 || !r.pointIDsOf(d.g) {
				continue
			}
			d.mat = r.materialState(it.material, pointPipe)
		}
		if d.mat != nil && d.mat.group == nil {
			continue
		}
		items = append(items, it)
		d.record = *records
		*records++
		modelView(c.rot, c.eye, &it.model, &mv)
		off := r.record(d.record)
		putMat(r.ubytes, off, &mv)
		putVec4(r.ubytes, off+64, 0, 0, 0, 0)
		putVec4(r.ubytes, off+80, float32(len(items)), 0, 0, 0)
		putVec4(r.ubytes, off+96, 0, 0, 0, 0)
		plan.draws = append(plan.draws, d)
	}
	frameBase := itemBase + *records*uniformAlign
	proj := c.cam.ProjectionMatrix()
	invProj, _ := proj.Inverse()
	invRot, _ := c.rot.Inverse()
	W, H := c.f.Width, c.f.Height
	end := frameBase
	for k := range r.picks {
		q := r.picks[k]
		rect := pickRect(&q, W, H, c.dpr)
		job := &pickJob{req: q, rect: rect, frame: frameBase + k*pickFrameStride, invProj: invProj, invRot: invRot, eye: c.eye, width: W, height: H,
			dpr: c.dpr, items: slices.Clone(items)}
		if !rect.Empty() {
			pm := pickMatrix(rect, W, H).Mul(proj).Float32()
			r.writeFrameAt(job.frame, &pm, float32(rect.Dx()), float32(rect.Dy()), c.style, c.cam, c.rot, c.dpr)
		}
		plan.jobs = append(plan.jobs, job)
		end = job.frame + pickFrameStride
	}
	r.picks = r.picks[:0]
	return plan, end
}

// pickRect is the rectangle of pixels of the frame that a request looks at, empty when it is outside the frame.
func pickRect(q *pickRequest, w, h int, dpr float64) image.Rectangle {
	var rect image.Rectangle
	if q.single {
		cx, cy := int(math.Floor(q.at.Dx*dpr)), int(math.Floor(q.at.Dy*dpr))
		rad := int(math.Ceil(q.radius * dpr))
		rect = image.Rect(cx-rad, cy-rad, cx+rad+1, cy+rad+1)
	} else {
		rect = image.Rect(int(math.Floor(q.rect.Left*dpr)), int(math.Floor(q.rect.Top*dpr)), int(math.Ceil(q.rect.Right*dpr)), int(math.Ceil(q.rect.Bottom*dpr)))
	}
	return rect.Intersect(image.Rect(0, 0, w, h))
}

// pickMatrix is the matrix that takes the rectangle of pixels of a frame (w by h pixels, the origin at the top left) to the whole of the
// clip space, so that a target the size of the rectangle shows only it.
func pickMatrix(rect image.Rectangle, w, h int) vecmath.Mat4 {
	W, H := float64(w), float64(h)
	x0, x1 := 2*float64(rect.Min.X)/W-1, 2*float64(rect.Max.X)/W-1
	y1, y0 := 1-2*float64(rect.Min.Y)/H, 1-2*float64(rect.Max.Y)/H
	m := vecmath.Mat4Identity()
	m[0] = 2 / (x1 - x0)
	m[5] = 2 / (y1 - y0)
	m[12] = -(x1 + x0) / (x1 - x0)
	m[13] = -(y1 + y0) / (y1 - y0)
	return m
}

// pointIDsOf makes the buffer with the number of each point of a mesh of points, once.
func (r *Renderer) pointIDsOf(g *meshGPU) bool {
	if g.pointIDs != nil {
		return true
	}
	ids := make([]float32, g.count)
	for i := range ids {
		ids[i] = float32(i)
	}
	b, err := r.newBuffer("viewport3d point numbers", floatBytes(nil, ids), gpu.BufferUsageVertex)
	if err != nil {
		slog.Error("viewport3d: the numbers of the points of a mesh could not be uploaded", "err", err)
		return false
	}
	g.pointIDs = b
	return true
}

// drawPicks encodes the pick passes of the plan after the frame's own pass, and asks for the reads when the frame has been
// submitted.
func (r *Renderer) drawPicks(f *gpu.ViewFrame, plan *pickPlan) {
	for _, job := range plan.jobs {
		if job.rect.Empty() {
			job := job
			f.AfterSubmit(func() { r.pickDone(job, nil, nil, 0, 0, nil) })
			continue
		}
		r.drawPick(f, plan, job)
	}
}

// pickTargets are the textures and buffers of one pick.
type pickTargets struct {
	ids, depthValue, depth *gpu.Texture
	idsBuf, depthBuf       *gpu.Buffer
}

func (t *pickTargets) release() {
	for _, tex := range []*gpu.Texture{t.ids, t.depthValue, t.depth} {
		if tex != nil {
			tex.Release()
		}
	}
	for _, b := range []*gpu.Buffer{t.idsBuf, t.depthBuf} {
		if b != nil {
			b.Release()
		}
	}
}

// drawPick encodes one pick pass and the copies of its targets.
func (r *Renderer) drawPick(f *gpu.ViewFrame, plan *pickPlan, job *pickJob) {
	w, h := job.rect.Dx(), job.rect.Dy()
	size := gpu.Extent3D{Width: w, Height: h, DepthOrArrayLayers: 1}
	t := &pickTargets{}
	var err error
	fail := func() {
		t.release()
		f.AfterSubmit(func() { job.req.fail(err) })
	}
	attach := gpu.TextureUsageRenderAttachment | gpu.TextureUsageCopySrc
	if t.ids, err = r.dev.NewTexture(gpu.TextureDesc{Label: "viewport3d pick ids", Size: size, Format: gpu.TextureFormatRG32Uint, Usage: attach}); err != nil {
		fail()
		return
	}
	if t.depthValue, err = r.dev.NewTexture(gpu.TextureDesc{Label: "viewport3d pick depth value", Size: size, Format: gpu.TextureFormatR32Float, Usage: attach}); err != nil {
		fail()
		return
	}
	if t.depth, err = r.dev.NewTexture(gpu.TextureDesc{Label: "viewport3d pick depth", Size: size, Format: gpu.TextureFormatDepth32Float, Usage: gpu.TextureUsageRenderAttachment}); err != nil {
		fail()
		return
	}
	align := r.dev.Limits().BytesPerRowAlignment
	idsRow := (w*8 + align - 1) / align * align
	depthRow := (w*4 + align - 1) / align * align
	if t.idsBuf, err = r.dev.NewBuffer(gpu.BufferDesc{Label: "viewport3d pick ids", Size: idsRow * h, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst}); err != nil {
		fail()
		return
	}
	if t.depthBuf, err = r.dev.NewBuffer(gpu.BufferDesc{Label: "viewport3d pick depth", Size: depthRow * h, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst}); err != nil {
		fail()
		return
	}
	pass := f.Encoder.BeginRenderPass(gpu.RenderPassDesc{
		Label: "viewport3d pick",
		ColorAttachments: []gpu.RenderPassColorAttachment{
			{View: t.ids.View()},
			{View: t.depthValue.View()},
		},
		DepthStencilAttachment: &gpu.RenderPassDepthStencilAttachment{View: t.depth.View(), DepthClearValue: 0},
	})
	pass.SetBindGroup(0, r.frameGroup, uint32(job.frame))
	for i := range plan.draws {
		d := &plan.draws[i]
		off := uint32(itemBase + d.record*uniformAlign)
		switch d.item.mesh.data.Primitive {
		case PrimitiveTriangles:
			pipe := r.pipeline(pickKey(pipePickTri))
			if pipe == nil {
				continue
			}
			pass.SetPipeline(pipe)
			pass.SetBindGroup(1, r.sectionGroup, off)
			pass.SetVertexBuffer(0, d.g.corners, 0)
			pass.Draw(3*d.g.triangles, 1, 0, 0)
		case PrimitiveLines:
			pass.SetPipeline(r.pipeline(pickKey(pipePickLine)))
			pass.SetBindGroup(1, d.mat.group, 0)
			pass.SetBindGroup(2, r.itemGroup, off)
			pass.SetVertexBuffer(0, d.g.segments, 0)
			pass.SetVertexBuffer(1, r.corners, 0)
			pass.Draw(6, d.g.count, 0, 0)
		default:
			pass.SetPipeline(r.pipeline(pickKey(pipePickPoint)))
			pass.SetBindGroup(1, d.mat.group, 0)
			pass.SetBindGroup(2, r.itemGroup, off)
			pass.SetVertexBuffer(0, d.g.pos, 0)
			pass.SetVertexBuffer(1, r.pointCorners, 0)
			pass.SetVertexBuffer(2, d.g.pointIDs, 0)
			pass.Draw(6, d.g.count, 0, 0)
		}
	}
	pass.End()
	f.Encoder.CopyTextureToBuffer(gpu.TextureCopy{Texture: t.ids}, gpu.BufferCopy{Buffer: t.idsBuf, Layout: gpu.DataLayout{BytesPerRow: idsRow}}, size)
	f.Encoder.CopyTextureToBuffer(gpu.TextureCopy{Texture: t.depthValue}, gpu.BufferCopy{Buffer: t.depthBuf, Layout: gpu.DataLayout{BytesPerRow: depthRow}}, size)
	f.AfterSubmit(func() {
		var ids, depth []byte
		var idsErr, depthErr error
		pending := 2
		finish := func() {
			if pending--; pending > 0 {
				return
			}
			t.release()
			if idsErr == nil {
				idsErr = depthErr
			}
			r.pickDone(job, ids, depth, idsRow, depthRow, idsErr)
		}
		t.idsBuf.Read(0, idsRow*h, func(data []byte, err error) { ids, idsErr = data, err; finish() })
		t.depthBuf.Read(0, depthRow*h, func(data []byte, err error) { depth, depthErr = data, err; finish() })
	})
}

// pickDone turns what the pass wrote into the result of the pick and calls its function.
func (r *Renderer) pickDone(job *pickJob, ids, depth []byte, idsRow, depthRow int, err error) {
	if err != nil {
		job.req.fail(err)
		return
	}
	if job.req.single {
		job.req.done(job.single(ids, depth, idsRow, depthRow), nil)
		return
	}
	job.req.doneN(job.many(ids, depth, idsRow), nil)
}

// texel reads the item number and the primitive number of a pixel of the rectangle.
func texel(ids []byte, row, x, y int) (item, prim uint32) {
	o := y*row + 8*x
	return binary.LittleEndian.Uint32(ids[o:]), binary.LittleEndian.Uint32(ids[o+4:])
}

// many collects the items that are in the rectangle, in the order of the scene's items.
func (j *pickJob) many(ids []byte, depthBytes []byte, idsRow int) PickRectResult {
	if ids == nil {
		return PickRectResult{}
	}
	seen := make([]bool, len(j.items))
	for y := 0; y < j.rect.Dy(); y++ {
		for x := 0; x < j.rect.Dx(); x++ {
			if item, _ := texel(ids, idsRow, x, y); item > 0 && int(item) <= len(j.items) {
				seen[item-1] = true
			}
		}
	}
	var res PickRectResult
	for i, ok := range seen {
		if ok {
			res.Items = append(res.Items, j.items[i])
		}
	}
	return res
}

// single finds the pixel nearest to the point of the pick that has an item, and works out what was hit there.
func (j *pickJob) single(ids, depthBytes []byte, idsRow, depthRow int) PickResult {
	if ids == nil {
		return PickResult{}
	}
	cx := int(math.Floor(j.req.at.Dx*j.dpr)) - j.rect.Min.X
	cy := int(math.Floor(j.req.at.Dy*j.dpr)) - j.rect.Min.Y
	best, bestDist, bestDepth := -1, math.MaxInt, float32(-1)
	var bx, by int
	for y := 0; y < j.rect.Dy(); y++ {
		for x := 0; x < j.rect.Dx(); x++ {
			item, _ := texel(ids, idsRow, x, y)
			if item == 0 || int(item) > len(j.items) {
				continue
			}
			dist := (x-cx)*(x-cx) + (y-cy)*(y-cy)
			depth := math.Float32frombits(binary.LittleEndian.Uint32(depthBytes[y*depthRow+4*x:]))
			if dist < bestDist || dist == bestDist && depth > bestDepth {
				best, bestDist, bestDepth, bx, by = int(item), dist, depth, x, y
			}
		}
	}
	if best < 0 {
		return PickResult{}
	}
	_, prim := texel(ids, idsRow, bx, by)
	it := j.items[best-1]
	px, py := j.rect.Min.X+bx, j.rect.Min.Y+by
	res := PickResult{Hit: true, Item: it, PickID: it.pickID, Primitive: it.mesh.data.Primitive, Index: int(prim), Depth: float64(bestDepth),
		At: ui.Offset{Dx: (float64(px) + 0.5) / j.dpr, Dy: (float64(py) + 0.5) / j.dpr}}
	nx := 2*(float64(px)+0.5)/float64(j.width) - 1
	ny := 1 - 2*(float64(py)+0.5)/float64(j.height)
	v := j.invProj.MulVec4(vecmath.Vec4{X: nx, Y: ny, Z: float64(bestDepth), W: 1})
	if v.W != 0 {
		res.Point = j.eye.Add(j.invRot.TransformPoint(vecmath.Vec3{X: v.X / v.W, Y: v.Y / v.W, Z: v.Z / v.W}))
	}
	if res.Primitive == PrimitiveTriangles {
		res.Normal = it.triangleNormal(res.Index, j.eye.Sub(res.Point))
	}
	return res
}

// triangleNormal is the unit normal in the world of the i-th triangle of the item's mesh, turned to face toward.
func (it *Item) triangleNormal(i int, toward vecmath.Vec3) vecmath.Vec3 {
	m := it.mesh
	if 3*i+2 >= m.IndexCount() {
		return vecmath.Vec3{}
	}
	a := it.model.TransformPoint(m.position(m.index(3 * i)))
	b := it.model.TransformPoint(m.position(m.index(3*i + 1)))
	c := it.model.TransformPoint(m.position(m.index(3*i + 2)))
	n := b.Sub(a).Cross(c.Sub(a)).Normalize()
	if n.Dot(toward) < 0 {
		n = n.Neg()
	}
	return n
}

// RayHit is what Scene.Raycast finds.
//
// Flutter: no counterpart
type RayHit struct {
	// Item is the item that the ray met first, and Index the triangle of its mesh.
	Item  *Item
	Index int
	// T is the distance along the ray, Point the point of the world that the ray meets, and Normal the unit normal of the triangle,
	// turned to face the ray's origin.
	T      float64
	Point  vecmath.Vec3
	Normal vecmath.Vec3
}

// Raycast finds where a ray of the world first meets a triangle of the scene's visible items, in float64, on the CPU: a ray from
// Camera.Ray tells what is under a point of the view at once, with no frame and no GPU, which the navigation's pivot needs. It
// tests every triangle of the items that the ray meets the box of, so it is for an occasional question, not for every frame; Pick
// answers from the picture, with one frame's delay, in any number of triangles. What a section plane cuts away is not met.
//
// Flutter: no counterpart
func (s *Scene) Raycast(ray vecmath.Ray) (RayHit, bool) {
	best := RayHit{T: math.Inf(1)}
	found := false
	for _, it := range s.items {
		if !it.visible || it.mesh.data.Primitive != PrimitiveTriangles {
			continue
		}
		if t, ok := ray.IntersectBox(it.Bounds()); !ok || t > best.T {
			continue
		}
		inv, ok := it.model.Inverse()
		if !ok {
			continue
		}
		local := ray.Transform(inv)
		m := it.mesh
		for i := 0; i+2 < m.IndexCount(); i += 3 {
			t, hit := local.IntersectTriangle(m.position(m.index(i)), m.position(m.index(i+1)), m.position(m.index(i+2)))
			if !hit || t >= best.T {
				continue
			}
			point := ray.At(t)
			if s.cutAway(point) {
				continue
			}
			best = RayHit{Item: it, Index: i / 3, T: t, Point: point}
			found = true
		}
	}
	if !found {
		return RayHit{}, false
	}
	best.Normal = best.Item.triangleNormal(best.Index, ray.Origin.Sub(best.Point))
	return best, true
}

// cutAway says whether a section plane cuts away a point of the world.
func (s *Scene) cutAway(p vecmath.Vec3) bool {
	for _, pl := range s.planes {
		if pl.DistanceTo(p) > 0 {
			return true
		}
	}
	return false
}
