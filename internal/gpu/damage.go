package gpu

import (
	"image"
	"math"
	"reflect"

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
// picture (by identity: a picture is immutable, and a layer that the framework did
// not repaint keeps its picture) with the transform it is drawn under, and the
// parameters of the layers around it (clips, opacity, filters). A tile whose hash
// differs from the last frame's is damaged. A picture that was recorded again is
// damaged even if it drew the same, and what the tracker cannot bound (a draw that
// fills the clip, an image filter on a layer, an unknown layer) damages the whole
// clip or the whole surface; it errs on the side of drawing more.
//
// The tracker holds on to the last scene's pictures and parameters so that their
// addresses are not reused while they are compared. Use it on one thread.
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

// pictureInfo is what a picture can change: the rectangle, in the coordinates of the
// picture, that its operations reach.
type pictureInfo struct {
	bounds ui.Rect
	any    bool // bounds has something in it
	// unbounded says that the picture draws something that fills the clip it is in
	// (or whose reach is not known).
	unbounded bool
	// full says that the picture reads the pixels around what it draws (an image
	// filter on a layer), so that a change next to it changes it too.
	full bool
}

func (i *pictureInfo) add(r ui.Rect) {
	if r.IsEmpty() {
		return
	}
	if !i.any {
		i.bounds, i.any = r, true
		return
	}
	i.bounds = i.bounds.ExpandToInclude(r)
}

// addUnder adds a rectangle of the child picture seen under the matrix.
func (i *pictureInfo) addUnder(m *ui.Matrix4, r ui.Rect) {
	if r.IsEmpty() {
		return
	}
	d := m.TransformRect(r)
	if !d.IsFinite() {
		i.unbounded = true
		return
	}
	i.add(d)
}

// measure returns what the picture reaches, from the cache of the frames or by
// walking its operations.
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
		measureOps(w, i, p.DisplayList().Ops(), ui.NewMatrix4Identity())
	}
	return i
}

func measureOps(w *damageWalk, i *pictureInfo, ops []ui.DisplayOp, ctm *ui.Matrix4) {
	m := *ctm
	var stack []ui.Matrix4
	for _, op := range ops {
		switch v := op.(type) {
		case ui.OpSave:
			stack = append(stack, m)
		case ui.OpSaveLayer:
			stack = append(stack, m)
			p := &v.Paint
			if p.ImageFilter() != nil {
				i.full = true
			}
			if p.BlendMode() != ui.BlendModeSrcOver || p.ColorFilter() != nil {
				// The layer's blend or filter can change the pixels of the whole layer.
				if v.HasBounds {
					i.addUnder(&m, v.Bounds)
				} else {
					i.unbounded = true
				}
			}
		case ui.OpRestore:
			if len(stack) > 0 {
				m = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			}
		case ui.OpTransform:
			m.MultiplyBy(&v.Matrix)
		case ui.OpClipRect, ui.OpClipRRect, ui.OpClipRSuperellipse, ui.OpClipPath:
			// A clip only removes; ignoring it keeps the bounds a superset.
		case ui.OpDrawPicture:
			child := w.measure(v.Picture)
			i.full = i.full || child.full
			i.unbounded = i.unbounded || child.unbounded
			if child.any {
				i.addUnder(&m, child.bounds)
			}
		case ui.OpDrawParagraph:
			ok := textdraw.Expand(v, &m, func(inner ui.DisplayOp, im *ui.Matrix4) {
				measureOps(w, i, []ui.DisplayOp{inner}, im)
			})
			if !ok {
				i.unbounded = true
			}
		case ui.OpDrawColor, ui.OpDrawPaint, ui.OpDrawAtlas:
			i.unbounded = true
		case ui.OpDrawImage:
			if v.Image == nil {
				break
			}
			measurePaint(i, &v.Paint)
			i.addUnder(&m, ui.NewRectFromLTWH(v.Offset.Dx, v.Offset.Dy, float64(v.Image.Width()), float64(v.Image.Height())))
		case ui.OpDrawImageRect:
			measurePaint(i, &v.Paint)
			i.addUnder(&m, v.Dst)
		case ui.OpDrawImageNine:
			measurePaint(i, &v.Paint)
			i.addUnder(&m, v.Dst)
		case ui.OpDrawVertices:
			measurePaint(i, &v.Paint)
			if v.Vertices == nil {
				break
			}
			if pos := v.Vertices.Positions(); len(pos) > 0 {
				i.addUnder(&m, pointsBounds(pos))
			}
		case ui.OpDrawShadow:
			if b, ok := ShadowBounds(v); ok {
				i.addUnder(&m, b.Inflate(2))
			} else {
				i.unbounded = true
			}
		default:
			if p := OpPaint(op); p != nil {
				measurePaint(i, p)
			}
			if b, ok := ShapeBounds(op); ok {
				i.addUnder(&m, b)
			} else {
				i.unbounded = true
			}
		}
	}
}

// measurePaint makes the picture unbounded for a paint whose filters spread what it
// draws past its geometry.
func measurePaint(i *pictureInfo, p *ui.Paint) {
	if p.MaskFilter() != nil || p.ImageFilter() != nil {
		i.unbounded = true
	}
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

const hashPrime = 1099511628211

func mix(h, v uint64) uint64 { return (h ^ v) * hashPrime }

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
		bounds := st.clip
		if !info.unbounded {
			bounds = bounds.Intersect(w.device(&m, info.bounds))
			if !info.any {
				return
			}
		}
		sig := hashMatrix(st.sig, &m)
		id, _ := w.identity(v.Picture)
		w.add(bounds, mix(sig, id))
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
