package gpu

import (
	"image"
	"math"
	"reflect"

	"github.com/rm4n0s/glutter/internal/gpu/opsig"
	"github.com/rm4n0s/glutter/internal/gpu/textdraw"
	"github.com/rm4n0s/glutter/ui"
)

// Damage is the part of a surface that a frame changed from the frame before it
// (SPEC §9.1, "Damage"): what a renderer tells the compositor when it presents.
//
// Flutter: no counterpart (flutter::FrameDamage)
type Damage struct {
	// Full says that the whole surface may have changed; Rects is then empty.
	Full bool
	// Rects are the changed pixels, from the top left, when Full is false. There is
	// none when the frame is the same as the one before it.
	Rects []image.Rectangle
}

const (
	// damageTile is the side of the squares the surface is divided into, in pixels.
	damageTile = 64
	// maxDamageRects is the most rectangles a Damage lists; more are merged into one.
	maxDamageRects = 8
)

// DamageTracker finds what changed between the scenes of a surface. It divides the
// surface into tiles, and for each tile hashes what the scene draws there: every
// drawing operation of every picture that reaches the tile, by its contents (opsig),
// with the transform it is drawn under and the parameters of the clips and layers
// around it, in the picture and in the scene (clips, opacity, filters). A tile whose
// hash differs from the last frame's is damaged. A picture that was recorded again
// is damaged where it draws something else than the picture before it, and nowhere
// when it draws the same: a rebuild of the framework records a layer again whether
// or not its drawing changed. What the tracker cannot bound (a draw that fills the
// clip, an image filter on a layer, an unknown layer) damages the whole clip or the
// whole surface, and an operation it cannot compare is damaged whenever its picture
// is recorded again; it errs on the side of drawing more.
//
// The operations of a picture are measured once, when the tracker first sees the
// picture (a picture is immutable, and a layer that the framework did not repaint
// keeps its picture). The tracker holds on to the pictures of the last scene, and so
// to everything they draw, so that the addresses in their signatures are not reused
// while they are compared. Use it on one thread.
//
// Flutter: no counterpart (flutter::DiffContext)
type DamageTracker struct {
	w, h       int
	cols, rows int
	tiles      []uint64
	next       []uint64
	valid      bool
	scene      *ui.Scene
	prev       map[*ui.Picture]*pictureInfo
	keep       []any
	// TextureVersion, when set, gives the version of a registered texture (0 for an id that is not registered):
	// a texture layer is damaged when the version of its texture changes (SPEC §27.4).
	TextureVersion func(id int64) uint64
}

// NewDamageTracker returns a tracker that has seen no frame, so that the first Update
// reports the whole surface.
//
// Flutter: no counterpart
func NewDamageTracker() *DamageTracker { return &DamageTracker{} }

// Invalidate forgets the last frame: the next Update reports the whole surface. Call
// it when the pixels the compositor has are not the last frame's (the window lost its
// content, or a frame was drawn and not presented).
//
// Flutter: no counterpart
func (t *DamageTracker) Invalidate() { t.valid = false }

// Update says what the scene changes from the scene given to the last call, for a
// surface of w x h pixels. The whole surface is reported for the first frame, after
// Invalidate, when the size changed, for the same scene again (a request to draw it
// once more means that the window needs all of it), and for a scene with something
// that depends on the pixels around it.
//
// Flutter: no counterpart
func (t *DamageTracker) Update(scene *ui.Scene, w, h int) Damage {
	if scene == nil || w <= 0 || h <= 0 {
		t.valid = false
		return Damage{Full: true}
	}
	cols, rows := (w+damageTile-1)/damageTile, (h+damageTile-1)/damageTile
	if cap(t.next) < cols*rows {
		t.next = make([]uint64, cols*rows)
	}
	next := t.next[:cols*rows]
	clear(next)
	wk := &damageWalk{t: t, tiles: next, cols: cols, rows: rows, w: w, h: h, cur: map[*ui.Picture]*pictureInfo{}}
	wk.layer(scene.Root(), walkState{m: *ui.NewMatrix4Identity(), clip: ui.Rect{Right: float64(w), Bottom: float64(h)}})

	same := t.valid && t.scene != scene && t.w == w && t.h == h && !wk.full
	var d Damage
	if same {
		d = damageOf(t.tiles, next, cols, rows, w, h)
	} else {
		d = Damage{Full: true}
	}
	t.tiles, t.next = next, t.tiles
	t.w, t.h, t.cols, t.rows = w, h, cols, rows
	t.valid, t.scene, t.prev, t.keep = true, scene, wk.cur, wk.keep
	return d
}

// damageOf lists the rectangles of the tiles that differ.
func damageOf(old, cur []uint64, cols, rows, w, h int) Damage {
	var rects []image.Rectangle
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; {
			if old[y*cols+x] == cur[y*cols+x] {
				x++
				continue
			}
			start := x
			for x < cols && old[y*cols+x] != cur[y*cols+x] {
				x++
			}
			r := image.Rect(start*damageTile, y*damageTile, min(x*damageTile, w), min((y+1)*damageTile, h))
			// A run under one of the same columns extends it.
			merged := false
			for i := range rects {
				if rects[i].Min.X == r.Min.X && rects[i].Max.X == r.Max.X && rects[i].Max.Y == r.Min.Y {
					rects[i].Max.Y = r.Max.Y
					merged = true
					break
				}
			}
			if !merged {
				rects = append(rects, r)
			}
		}
	}
	if len(rects) > maxDamageRects {
		all := rects[0]
		for _, r := range rects[1:] {
			all = all.Union(r)
		}
		rects = []image.Rectangle{all}
	}
	return Damage{Rects: rects}
}

// pictureInfo is what a picture draws, for the damage: an item for each of its drawing
// operations (or, for a picture of many operations, for each cell of a grid over them).
// It depends only on the contents of the picture, so two pictures that record the same
// have the same items.
type pictureInfo struct {
	items []pictureItem
	// full says that the picture reads the pixels around what it draws (an image
	// filter on a layer), so that a change next to it changes it too.
	full bool
}

// pictureItem is a part of a picture: where it can change pixels, and the signature of what
// it draws there (the operation, the transform it is drawn under inside the picture, and
// the clips and layers of the picture around it).
type pictureItem struct {
	// bounds is the rectangle, in the coordinates of the picture, that the item reaches.
	bounds ui.Rect
	// unbounded says that the item fills the clip the picture is drawn in (or that its
	// reach is not known); bounds is not used.
	unbounded bool
	sig       uint64
}

const (
	// maxPictureItems is the most items a picture keeps one for each operation; a picture
	// with more is folded into a grid, so that a frame costs no more than this for a
	// picture that was kept, however much it draws.
	maxPictureItems = 512
	// pictureGrid is the number of cells of the grid along each axis.
	pictureGrid = 16
)

// measure returns what the picture draws, from the cache of the frames or by walking its
// operations.
func (w *damageWalk) measure(p *ui.Picture) *pictureInfo {
	if i, ok := w.cur[p]; ok {
		return i
	}
	if i, ok := w.t.prev[p]; ok {
		w.cur[p] = i
		return i
	}
	i := &pictureInfo{}
	w.cur[p] = i
	if p != nil {
		ms := measureState{m: *ui.NewMatrix4Identity()}
		ms.msig = hashMatrix(0, &ms.m)
		w.measureOps(i, p, p.DisplayList().Ops(), ms)
		i.fold()
	}
	return i
}

// measureState is what the operations of a picture are measured under: the transform, the
// clip, and the signature of the clips and layers around them, all inside the picture.
type measureState struct {
	m ui.Matrix4
	// msig is the signature of m.
	msig uint64
	// sig is the signature of the clips and layers around the operation.
	sig uint64
	// clip is the rectangle, in the coordinates of the picture, that the clips around the
	// operation let through (a superset), when hasClip is set.
	clip    ui.Rect
	hasClip bool
}

// narrow intersects the clip with a rectangle given in the coordinates of the operation.
func (ms *measureState) narrow(local ui.Rect) {
	r := ms.m.TransformRect(local)
	if !r.IsFinite() {
		return
	}
	if ms.hasClip {
		r = r.Intersect(ms.clip)
	}
	ms.clip, ms.hasClip = r, true
}

// add adds an item for an operation with the signature sig that reaches the rectangle
// local, in the coordinates of the operation, or everything when unbounded is set.
func (i *pictureInfo) add(ms *measureState, local ui.Rect, unbounded bool, sig uint64) {
	it := pictureItem{sig: mix(mix(ms.sig, ms.msig), sig)}
	if !unbounded {
		// A rectangle with no area is kept: a hairline along an axis has such bounds, and
		// draws a pixel's width around them.
		it.bounds = ms.m.TransformRect(local)
		unbounded = !it.bounds.IsFinite()
	}
	if ms.hasClip {
		// The clip bounds what the item can change, whatever it reaches.
		if unbounded {
			it.bounds, unbounded = ms.clip, false
		} else {
			it.bounds = it.bounds.Intersect(ms.clip)
		}
		if it.bounds.Right < it.bounds.Left || it.bounds.Bottom < it.bounds.Top {
			return
		}
	}
	it.unbounded = unbounded
	i.items = append(i.items, it)
}

// measureOps adds the items of the operations ops of the picture p, under the state ms.
func (w *damageWalk) measureOps(i *pictureInfo, p *ui.Picture, ops []ui.DisplayOp, ms measureState) {
	var stack []measureState
	for n, op := range ops {
		sig, known := opsig.Op(op)
		if !known {
			// An operation with no signature is taken by the identity of its picture: it
			// is damaged whenever the picture is recorded again.
			id, _ := w.identity(p)
			sig = mix(mix(id, uint64(n)), sig)
		}
		switch v := op.(type) {
		case ui.OpSave:
			stack = append(stack, ms)
		case ui.OpSaveLayer:
			stack = append(stack, ms)
			paint := &v.Paint
			if paint.ImageFilter() != nil {
				i.full = true
			}
			if paint.BlendMode() != ui.BlendModeSrcOver || paint.ColorFilter() != nil {
				// The layer's blend or filter can change the pixels of the whole layer.
				i.add(&ms, v.Bounds, !v.HasBounds, sig)
			}
			ms.sig = mix(mix(ms.sig, ms.msig), sig)
		case ui.OpRestore:
			if len(stack) > 0 {
				ms = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			}
		case ui.OpTransform:
			ms.m.MultiplyBy(&v.Matrix)
			ms.msig = hashMatrix(0, &ms.m)
		case ui.OpClipRect:
			if v.Op != ui.ClipOpDifference {
				ms.narrow(v.Rect)
			}
			ms.sig = mix(mix(ms.sig, ms.msig), sig)
		case ui.OpClipRRect:
			ms.narrow(v.RRect.OuterRect())
			ms.sig = mix(mix(ms.sig, ms.msig), sig)
		case ui.OpClipRSuperellipse:
			ms.narrow(v.RSuperellipse.OuterRect())
			ms.sig = mix(mix(ms.sig, ms.msig), sig)
		case ui.OpClipPath:
			if v.Path != nil {
				if b := v.Path.GetBounds(); b != (ui.Rect{}) || v.Path.IsEmpty() {
					ms.narrow(b)
				}
			}
			ms.sig = mix(mix(ms.sig, ms.msig), sig)
		case ui.OpDrawPicture:
			child := w.measure(v.Picture)
			i.full = i.full || child.full
			for _, it := range child.items {
				i.add(&ms, it.bounds, it.unbounded, it.sig)
			}
		case ui.OpDrawParagraph:
			if b, ok := textdraw.Bounds(v); ok {
				// The box of the paragraph holds what it draws; measuring the glyphs one by one
				// would cost the length of the text each time the picture is recorded again.
				i.add(&ms, b, false, sig)
				break
			}
			var box boundsAcc
			ok := textdraw.Expand(v, &ms.m, func(inner ui.DisplayOp, im *ui.Matrix4) {
				b, unbounded := drawBounds(inner)
				if !unbounded {
					b = im.TransformRect(b)
					unbounded = !b.IsFinite()
				}
				box.add(b, unbounded)
			})
			if !ok || box.unbounded {
				i.add(&ms, ui.Rect{}, true, sig)
			} else if box.any {
				// The box is in the coordinates of the picture already.
				inPicture := ms
				inPicture.m = *ui.NewMatrix4Identity()
				i.add(&inPicture, box.r, false, sig)
			}
		default:
			b, unbounded := drawBounds(op)
			i.add(&ms, b, unbounded, sig)
		}
	}
}

// boundsAcc is the box of some rectangles; a rectangle with no area counts (the bounds of a
// hairline along an axis).
type boundsAcc struct {
	r         ui.Rect
	any       bool
	unbounded bool
}

func (a *boundsAcc) add(r ui.Rect, unbounded bool) {
	switch {
	case unbounded:
		a.unbounded = true
	case !a.any:
		a.r, a.any = r, true
	default:
		a.r = a.r.ExpandToInclude(r)
	}
}

// drawBounds returns the rectangle, in its own coordinates, that a drawing operation can
// change, or says that it fills its clip or that its reach is not known. It is for the
// operations that draw (not a picture or a paragraph, which are measured by their parts).
func drawBounds(op ui.DisplayOp) (r ui.Rect, unbounded bool) {
	switch v := op.(type) {
	case ui.OpDrawColor, ui.OpDrawPaint, ui.OpDrawAtlas:
		return ui.Rect{}, true
	case ui.OpDrawPicture:
		if v.Picture == nil {
			return ui.Rect{}, false
		}
		// A picture inside a paragraph (a color glyph) draws inside its cull rectangle.
		return v.Picture.CullRect(), false
	case ui.OpDrawImage:
		if v.Image == nil {
			return ui.Rect{}, false
		}
		return ui.NewRectFromLTWH(v.Offset.Dx, v.Offset.Dy, float64(v.Image.Width()), float64(v.Image.Height())), spreads(&v.Paint)
	case ui.OpDrawImageRect:
		return v.Dst, spreads(&v.Paint)
	case ui.OpDrawImageNine:
		return v.Dst, spreads(&v.Paint)
	case ui.OpDrawVertices:
		if v.Vertices == nil {
			return ui.Rect{}, spreads(&v.Paint)
		}
		pos := v.Vertices.Positions()
		if len(pos) == 0 {
			return ui.Rect{}, spreads(&v.Paint)
		}
		return pointsBounds(pos), spreads(&v.Paint)
	case ui.OpDrawShadow:
		if b, ok := ShadowBounds(v); ok {
			return b.Inflate(2), false
		}
		return ui.Rect{}, true
	}
	if img, mask := PaintFilters(op); img || mask {
		return ui.Rect{}, true
	}
	if b, ok := ShapeBounds(op); ok {
		return b, false
	}
	return ui.Rect{}, true
}

// spreads says that the filters of a paint spread what it draws past its geometry.
func spreads(p *ui.Paint) bool { return p.MaskFilter() != nil || p.ImageFilter() != nil }

// fold replaces the items of a picture of many operations by the cells of a grid over the
// items that have bounds: a cell is an item whose signature is made of the signatures of
// the items that reach it, in order, and whose bounds hold what they reach of it. An item
// that fills the clip stays an item of its own, and is in every cell, so that its place in
// the order still counts.
func (i *pictureInfo) fold() {
	if len(i.items) <= maxPictureItems {
		return
	}
	var box boundsAcc
	for _, it := range i.items {
		box.add(it.bounds, it.unbounded)
	}
	if !box.any {
		return
	}
	cw, ch := box.r.Width()/pictureGrid, box.r.Height()/pictureGrid
	if !(cw > 0) || !(ch > 0) {
		return
	}
	var cells [pictureGrid * pictureGrid]struct {
		sig uint64
		box boundsAcc
	}
	cellOf := func(v, origin, size float64) int {
		return max(0, min(pictureGrid-1, int(math.Floor((v-origin)/size))))
	}
	var folded []pictureItem
	for _, it := range i.items {
		if it.unbounded {
			folded = append(folded, it)
			for c := range cells {
				cells[c].sig = mix(cells[c].sig, it.sig)
			}
			continue
		}
		x0, x1 := cellOf(it.bounds.Left, box.r.Left, cw), cellOf(it.bounds.Right, box.r.Left, cw)
		y0, y1 := cellOf(it.bounds.Top, box.r.Top, ch), cellOf(it.bounds.Bottom, box.r.Top, ch)
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				c := &cells[y*pictureGrid+x]
				cell := ui.Rect{
					Left: box.r.Left + float64(x)*cw, Top: box.r.Top + float64(y)*ch,
					Right: box.r.Left + float64(x+1)*cw, Bottom: box.r.Top + float64(y+1)*ch,
				}
				if x == pictureGrid-1 {
					cell.Right = box.r.Right
				}
				if y == pictureGrid-1 {
					cell.Bottom = box.r.Bottom
				}
				// The part of the item in the cell. Every cell that the item was given gets
				// it, also when rounding puts an edge of the item a hair outside the cell: the
				// item's own extent then stands for the part, which only makes it larger.
				part := it.bounds.Intersect(cell)
				if part.Right < part.Left {
					part.Left, part.Right = it.bounds.Left, it.bounds.Right
				}
				if part.Bottom < part.Top {
					part.Top, part.Bottom = it.bounds.Top, it.bounds.Bottom
				}
				c.sig = mix(c.sig, it.sig)
				c.box.add(part, false)
			}
		}
	}
	for c := range cells {
		if cells[c].box.any {
			folded = append(folded, pictureItem{bounds: cells[c].box.r, sig: cells[c].sig})
		}
	}
	i.items = folded
}

// walkState is the transform, the clip and the layer parameters around a layer.
type walkState struct {
	m    ui.Matrix4
	clip ui.Rect // in pixels, a superset of what the layers around it let through
	sig  uint64  // the parameters of the layers around it
}

// damageWalk is one Update: it walks the scene and hashes it into tiles.
type damageWalk struct {
	t          *DamageTracker
	tiles      []uint64
	cols, rows int
	w, h       int
	cur        map[*ui.Picture]*pictureInfo
	keep       []any
	full       bool
}

// mix adds a word to a hash, mixing every bit of the word into every bit of the result (as
// Hasher does), so that the signatures of two rectangles that differ in a high bit of two
// coordinates never cancel.
func mix(h, v uint64) uint64 {
	x := (h ^ v) * 0x9E3779B97F4A7C15
	x ^= x >> 32
	x *= 0xD6E8FEB86659FD93
	return x ^ x>>29
}

func mixFloats(h uint64, vs ...float64) uint64 {
	for _, v := range vs {
		h = mix(h, math.Float64bits(v))
	}
	return h
}

func hashMatrix(h uint64, m *ui.Matrix4) uint64 {
	for _, v := range m.Storage() {
		h = mix(h, uint64(math.Float32bits(v)))
	}
	return h
}

// identity returns a number for an object that is the same for the same object, and
// keeps the object alive until the next Update. It says false for what has no
// address.
func (w *damageWalk) identity(x any) (uint64, bool) {
	if x == nil {
		return 0, true
	}
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Func, reflect.UnsafePointer:
		w.keep = append(w.keep, x)
		return uint64(v.Pointer()), true
	}
	return 0, false
}

func (w *damageWalk) mixIdentity(h uint64, x any) uint64 {
	id, ok := w.identity(x)
	if !ok {
		w.full = true
	}
	return mix(h, id)
}

// device returns the pixels that a rectangle under the matrix covers, one more on each
// side for an anti-aliased edge, limited to the surface.
func (w *damageWalk) device(m *ui.Matrix4, r ui.Rect) ui.Rect {
	d := m.TransformRect(r)
	if !d.IsFinite() {
		return ui.Rect{Right: float64(w.w), Bottom: float64(w.h)}
	}
	return d.Inflate(1)
}

// add hashes an item into the tiles it reaches.
func (w *damageWalk) add(bounds ui.Rect, sig uint64) {
	if !bounds.IsFinite() || bounds.IsEmpty() || bounds.Right <= 0 || bounds.Bottom <= 0 || bounds.Left >= float64(w.w) || bounds.Top >= float64(w.h) {
		return
	}
	x0, y0 := int(math.Floor(max(bounds.Left, 0)))/damageTile, int(math.Floor(max(bounds.Top, 0)))/damageTile
	x1 := min(int(math.Ceil(min(bounds.Right, float64(w.w))))-1, w.w-1) / damageTile
	y1 := min(int(math.Ceil(min(bounds.Bottom, float64(w.h))))-1, w.h-1) / damageTile
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			i := y*w.cols + x
			w.tiles[i] = mix(w.tiles[i], sig)
		}
	}
}

func (w *damageWalk) children(l ui.Layer, st walkState) {
	for _, c := range l.Children() {
		w.layer(c, st)
	}
}

func (w *damageWalk) layer(l ui.Layer, st walkState) {
	switch v := l.(type) {
	case *ui.ContainerLayer:
		w.children(l, st)
	case *ui.TransformLayer:
		m := st.m
		m.MultiplyBy(&v.Matrix)
		w.children(l, walkState{m: m, clip: st.clip, sig: st.sig})
	case *ui.PictureLayer:
		m := st.m
		m.Translate(float32(v.Offset.Dx), float32(v.Offset.Dy), 0)
		info := w.measure(v.Picture)
		if info.full {
			w.full = true
		}
		sig := hashMatrix(st.sig, &m)
		for i := range info.items {
			it := &info.items[i]
			bounds := st.clip
			if !it.unbounded {
				bounds = bounds.Intersect(w.device(&m, it.bounds))
			}
			w.add(bounds, mix(sig, it.sig))
		}
	case *ui.TextureLayer:
		m := st.m
		bounds := st.clip.Intersect(w.device(&m, ui.NewRectFromLTWH(v.Offset.Dx, v.Offset.Dy, v.Width, v.Height)))
		version := uint64(0)
		if w.t.TextureVersion != nil && !v.Freeze {
			version = w.t.TextureVersion(v.TextureID)
		}
		sig := hashMatrix(st.sig, &m)
		sig = mixFloats(sig, v.Offset.Dx, v.Offset.Dy, v.Width, v.Height, float64(v.FilterQuality), float64(b2i(v.Freeze)))
		w.add(bounds, mix(mix(sig, uint64(v.TextureID)), version))
	case *ui.ClipRectLayer:
		w.clipLayer(l, st, v.ClipBehavior, v.Rect, func(h uint64) uint64 {
			return mixFloats(h, v.Rect.Left, v.Rect.Top, v.Rect.Right, v.Rect.Bottom)
		})
	case *ui.ClipRRectLayer:
		w.clipLayer(l, st, v.ClipBehavior, v.RRect.OuterRect(), func(h uint64) uint64 {
			r := v.RRect
			return mixFloats(h, r.Left, r.Top, r.Right, r.Bottom, r.TlRadiusX, r.TlRadiusY, r.TrRadiusX, r.TrRadiusY, r.BrRadiusX, r.BrRadiusY, r.BlRadiusX, r.BlRadiusY)
		})
	case *ui.ClipRSuperellipseLayer:
		w.clipLayer(l, st, v.ClipBehavior, v.RSuperellipse.OuterRect(), func(h uint64) uint64 {
			r := ui.RRect(v.RSuperellipse)
			return mixFloats(h, r.Left, r.Top, r.Right, r.Bottom, r.TlRadiusX, r.TlRadiusY, r.TrRadiusX, r.TrRadiusY, r.BrRadiusX, r.BrRadiusY, r.BlRadiusX, r.BlRadiusY)
		})
	case *ui.ClipPathLayer:
		w.clipLayer(l, st, v.ClipBehavior, v.Path.GetBounds(), func(h uint64) uint64 { return w.mixIdentity(h, v.Path) })
	case *ui.OpacityLayer:
		m := st.m
		m.Translate(float32(v.Offset.Dx), float32(v.Offset.Dy), 0)
		sig := hashMatrix(mixFloats(mix(st.sig, uint64(v.Alpha)), v.Offset.Dx, v.Offset.Dy), &st.m)
		w.children(l, walkState{m: m, clip: st.clip, sig: sig})
	case *ui.ColorFilterLayer:
		// A filter can turn the transparent pixels of its layer into colored ones, so
		// it is an item of its own that covers its clip.
		sig := w.mixIdentity(hashMatrix(st.sig, &st.m), v.Filter)
		w.add(st.clip, sig)
		w.children(l, walkState{m: st.m, clip: st.clip, sig: sig})
	case *ui.ShaderMaskLayer:
		sig := w.mixIdentity(hashMatrix(st.sig, &st.m), v.Shader)
		sig = mixFloats(sig, v.MaskRect.Left, v.MaskRect.Top, v.MaskRect.Right, v.MaskRect.Bottom, float64(v.BlendMode), float64(v.FilterQuality))
		w.add(st.clip.Intersect(w.device(&st.m, v.MaskRect)), sig)
		w.children(l, walkState{m: st.m, clip: st.clip, sig: sig})
	default:
		// Image filters and backdrop filters read the pixels around what they
		// cover, and an unknown layer cannot be bounded.
		w.full = true
	}
}

func (w *damageWalk) clipLayer(l ui.Layer, st walkState, behavior ui.Clip, local ui.Rect, param func(uint64) uint64) {
	if behavior == ui.ClipNone {
		w.children(l, st)
		return
	}
	sig := param(mix(hashMatrix(st.sig, &st.m), uint64(behavior)))
	clip := st.clip.Intersect(w.device(&st.m, local))
	w.children(l, walkState{m: st.m, clip: clip, sig: sig})
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
