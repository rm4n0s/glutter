package gpu

import (
	"math"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu/textdraw"
	"github.com/rm4n0s/glutter/ui"
)

// CommandKind says what a Command asks of a renderer.
//
// Flutter: no counterpart
type CommandKind int

// The kinds of command.
const (
	// CommandKindUnspecified is the zero value: not a command.
	//
	// Flutter: no counterpart
	CommandKindUnspecified CommandKind = iota
	// CommandDraw draws one operation of a picture with the command's transform
	// and clip.
	//
	// Flutter: no counterpart
	CommandDraw
	// CommandBeginLayer starts a group that is drawn separately and composited at
	// the matching CommandEndLayer: a SaveLayer, or a scene layer that needs a
	// group (opacity, filters, masks, and clips with a save layer).
	//
	// Flutter: no counterpart
	CommandBeginLayer
	// CommandEndLayer ends the innermost group and composites it.
	//
	// Flutter: no counterpart
	CommandEndLayer
)

// Command is one step of drawing, with the state it is drawn in. A renderer runs
// the commands of a DrawList in order.
//
// Flutter: no counterpart
type Command struct {
	// Kind says which fields are meaningful.
	Kind CommandKind
	// Op is the drawing operation of a CommandDraw, or the ui.OpSaveLayer of a
	// CommandBeginLayer that a canvas made. Clips, transforms and Save are not
	// commands: they are folded into Transform and Clip.
	Op ui.DisplayOp
	// Layer is the scene layer of a CommandBeginLayer that a scene made, or nil.
	// Its Offset, if it has one, is already part of the transform of the commands
	// inside the group.
	Layer ui.Layer
	// Transform maps the coordinates of Op to the surface's pixels. For a group it
	// is the transform in effect where the group is composited.
	Transform ui.Matrix4
	// Clip is what may be drawn into. For CommandBeginLayer it is the clip the
	// group is composited with, which for a clip layer with a save layer includes
	// the layer's own shape.
	Clip Clip
}

// DrawList is a scene flattened into the commands that draw it, in order.
//
// Flutter: no counterpart (flutter::DisplayList of the whole layer tree)
type DrawList struct {
	// Commands are in drawing order; each CommandBeginLayer has a matching
	// CommandEndLayer after it.
	Commands []Command
}

// ClipShapeKind says what a ClipShape is.
//
// Flutter: no counterpart
type ClipShapeKind int

// The kinds of clip shape.
const (
	// ClipShapeUnspecified is the zero value: not a shape.
	//
	// Flutter: no counterpart
	ClipShapeUnspecified ClipShapeKind = iota
	// ClipShapeRect is a rectangle that is not axis-aligned in the surface (the
	// transform rotates or skews it) or a rectangle that is subtracted
	// (Difference).
	//
	// Flutter: no counterpart
	ClipShapeRect
	// ClipShapeRRect is a rounded rectangle.
	//
	// Flutter: no counterpart
	ClipShapeRRect
	// ClipShapeRSuperellipse is a rounded superellipse.
	//
	// Flutter: no counterpart
	ClipShapeRSuperellipse
	// ClipShapePath is a path.
	//
	// Flutter: no counterpart
	ClipShapePath
)

// ClipShape is a clip that is not a rectangle of pixels. The shape is in its own
// coordinates; Transform maps them to the surface.
//
// Flutter: no counterpart
type ClipShape struct {
	// Kind says which of the shape fields is used.
	Kind ClipShapeKind
	// Rect is the shape of ClipShapeRect.
	Rect ui.Rect
	// RRect is the shape of ClipShapeRRect.
	RRect ui.RRect
	// RSuperellipse is the shape of ClipShapeRSuperellipse.
	RSuperellipse ui.RSuperellipse
	// Path is the shape of ClipShapePath.
	Path *ui.Path
	// Difference says that the shape is cut out of the clip instead of being
	// intersected with it.
	Difference bool
	// AntiAlias says whether the edge is smoothed.
	AntiAlias bool
	// Transform maps the shape to the surface.
	Transform ui.Matrix4
}

// Clip is the part of the surface that drawing may change. Bounds is a rectangle
// in pixels (whose edges are fractional when the clip is anti-aliased); Shapes
// narrow it further and are applied in order. A rectangle clip that the transform
// keeps axis-aligned is folded into Bounds, so that a renderer can use the
// scissor test; only the other shapes are listed.
//
// Flutter: no counterpart
type Clip struct {
	// Bounds is the rectangle that everything drawn is cut to. Its edges are
	// whole pixels, except for the edges of anti-aliased clips.
	Bounds ui.Rect
	// Shapes are the clips that are not folded into Bounds, in the order they were
	// made. Treat the slice as read-only.
	Shapes []ClipShape
}

// IsEmpty says that the clip lets nothing through.
//
// Flutter: no counterpart
func (c Clip) IsEmpty() bool { return c.Bounds.IsEmpty() }

// IsRect says that the clip is exactly Bounds.
//
// Flutter: no counterpart
func (c Clip) IsRect() bool { return len(c.Shapes) == 0 }

// Equal says whether two clips are the same clip.
//
// Flutter: no counterpart
func (c Clip) Equal(o Clip) bool {
	if c.Bounds != o.Bounds || len(c.Shapes) != len(o.Shapes) {
		return false
	}
	// Shapes hold paths and matrices; two clips made from the same state share the
	// slice, which is the case that matters for batching.
	return len(c.Shapes) == 0 || &c.Shapes[0] == &o.Shapes[0]
}

// withShape returns the clip with a shape added. The bounds narrow to the pixels
// the shape can reach, given as a rectangle in the shape's coordinates, unless
// the shape is subtracted. They narrow to whole pixels: the shape's own edge,
// anti-aliased or not, is for the renderer to draw from the shape, and must not
// be applied again by the bounds.
func (c Clip) withShape(s ClipShape, extent ui.Rect) Clip {
	shapes := make([]ClipShape, len(c.Shapes)+1)
	copy(shapes, c.Shapes)
	shapes[len(c.Shapes)] = s
	b := c.Bounds
	if !s.Difference {
		reach := s.Transform.TransformRect(extent)
		if reach.IsFinite() {
			b = b.Intersect(ui.Rect{Left: math.Floor(reach.Left), Top: math.Floor(reach.Top), Right: math.Ceil(reach.Right), Bottom: math.Ceil(reach.Bottom)})
		}
	}
	return Clip{Bounds: b, Shapes: shapes}
}

// IsAxisAligned reports whether the transform keeps rectangles axis-aligned: it
// has no rotation by other than a multiple of 90 degrees, no skew and no
// perspective. A rotation off by less than a millionth (what float32 gives for
// a quarter turn) counts as none.
//
// Flutter: no counterpart
func IsAxisAligned(m *ui.Matrix4) bool {
	s := m.Storage()
	if s[3] != 0 || s[7] != 0 || s[15] != 1 {
		return false
	}
	return (negligible(s[1], s[0]) && negligible(s[4], s[5])) || (negligible(s[0], s[1]) && negligible(s[5], s[4]))
}

// negligible reports whether v is a millionth of the magnitude it is compared
// with (the other entry of its column) or less.
func negligible(v, with float32) bool {
	const eps = 1e-6
	return math.Abs(float64(v)) <= eps*math.Abs(float64(with))+1e-12
}

// FlattenOpts are the optional arguments of Flatten.
//
// Flutter: no counterpart
type FlattenOpts struct {
	// Bounds is the surface in pixels; the clip starts as this rectangle. The zero
	// value starts with an unbounded clip.
	Bounds ui.Rect
	// Boundary, when it is not nil, is asked about every ui.TransformLayer and
	// ui.ContainerLayer (the layers Flatten folds into the commands under them),
	// and a layer it answers true for becomes a group, so that a renderer can keep
	// what the group draws for the next frame. The group is composited under the
	// clip it is in, and what is inside it is drawn under a clip of whole pixels
	// that has none of that clip's shapes, so that the group's pixels do not depend
	// on them. The group's Command.Transform is the transform of its contents.
	Boundary func(l ui.Layer) bool
}

// Flatten turns a scene into the list of commands that draws it. The layer tree
// and the pictures' operations are walked in painting order. Transforms and clips
// are folded into each command; layers that need a group become a pair of
// CommandBeginLayer and CommandEndLayer. Nothing is dropped: a renderer decides
// what it can draw. A clip that leaves nothing to draw does not remove the
// commands inside it; a renderer may skip a command whose Clip is empty.
//
// It fails with FlattenSceneNil for a nil scene and FlattenLayerUnknown for a
// layer of a type it does not know.
//
// Flutter: no counterpart (flutter::LayerTree::Preroll and Paint)
func Flatten(scene *ui.Scene, opts ...FlattenOpts) (*DrawList, error) {
	if scene == nil {
		return nil, errors.New("FlattenSceneNil", "Flatten needs a scene")
	}
	var o FlattenOpts
	switch len(opts) {
	case 0:
	case 1:
		o = opts[0]
	default:
		panic(errors.New("MultipleOptions", "pass at most one options value", "type", "FlattenOpts", "count", len(opts)))
	}
	bounds := o.Bounds
	if bounds == (ui.Rect{}) {
		const huge = 1 << 30
		bounds = ui.Rect{Left: -huge, Top: -huge, Right: huge, Bottom: huge}
	}
	f := &flattener{list: &DrawList{}, boundary: o.Boundary}
	st := state{m: *ui.NewMatrix4Identity(), clip: Clip{Bounds: bounds}}
	if err := f.layer(scene.Root(), st); err != nil {
		return nil, err
	}
	return f.list, nil
}

// state is the transform and the clip that drawing happens in.
type state struct {
	m    ui.Matrix4
	clip Clip
}

type flattener struct {
	list     *DrawList
	boundary func(ui.Layer) bool
}

func (f *flattener) emit(c Command) { f.list.Commands = append(f.list.Commands, c) }

func (f *flattener) begin(layer ui.Layer, op ui.DisplayOp, outer state) {
	f.emit(Command{Kind: CommandBeginLayer, Layer: layer, Op: op, Transform: outer.m, Clip: outer.clip})
}

func (f *flattener) end() { f.emit(Command{Kind: CommandEndLayer}) }

func translated(m ui.Matrix4, dx, dy float64) ui.Matrix4 {
	out := m
	out.Translate(float32(dx), float32(dy), 0)
	return out
}

func (f *flattener) children(l ui.Layer, st state) error {
	for _, c := range l.Children() {
		if err := f.layer(c, st); err != nil {
			return err
		}
	}
	return nil
}

// group draws the children of l inside a group composited in the state outer.
func (f *flattener) group(l ui.Layer, outer, inner state) error {
	f.begin(l, nil, outer)
	err := f.children(l, inner)
	f.end()
	return err
}

func (f *flattener) layer(l ui.Layer, st state) error {
	switch v := l.(type) {
	case *ui.ContainerLayer:
		if f.boundary != nil && f.boundary(l) {
			return f.boundaryGroup(l, st, st)
		}
		return f.children(l, st)
	case *ui.TransformLayer:
		m := st.m
		m.MultiplyBy(&v.Matrix)
		if f.boundary != nil && f.boundary(l) {
			return f.boundaryGroup(l, st, state{m: m, clip: st.clip})
		}
		return f.children(l, state{m: m, clip: st.clip})
	case *ui.PictureLayer:
		return f.picture(v.Picture, state{m: translated(st.m, v.Offset.Dx, v.Offset.Dy), clip: st.clip})
	case *ui.ClipRectLayer:
		return f.clipLayer(l, st, v.ClipBehavior, func(c Clip, m *ui.Matrix4, aa bool) Clip {
			return clipRect(c, m, v.Rect, aa, false)
		})
	case *ui.ClipRRectLayer:
		return f.clipLayer(l, st, v.ClipBehavior, func(c Clip, m *ui.Matrix4, aa bool) Clip { return clipRRect(c, m, v.RRect, aa) })
	case *ui.ClipRSuperellipseLayer:
		return f.clipLayer(l, st, v.ClipBehavior, func(c Clip, m *ui.Matrix4, aa bool) Clip {
			return clipRSuperellipse(c, m, v.RSuperellipse, aa)
		})
	case *ui.ClipPathLayer:
		return f.clipLayer(l, st, v.ClipBehavior, func(c Clip, m *ui.Matrix4, aa bool) Clip { return clipPath(c, m, v.Path, aa) })
	case *ui.OpacityLayer:
		return f.group(l, st, state{m: translated(st.m, v.Offset.Dx, v.Offset.Dy), clip: st.clip})
	case *ui.ColorFilterLayer, *ui.BackdropFilterLayer, *ui.ShaderMaskLayer:
		return f.group(l, st, st)
	case *ui.ImageFilterLayer:
		return f.group(l, st, state{m: translated(st.m, v.Offset.Dx, v.Offset.Dy), clip: st.clip})
	}
	return errors.NewErrf("FlattenLayerUnknown", "Flatten does not know the layer %T.", l)
}

// boundaryGroup draws the children of l, with the state inner, inside a group that is
// composited in the state outer; the children are drawn under the whole pixels of
// the clip's bounds only.
func (f *flattener) boundaryGroup(l ui.Layer, outer, inner state) error {
	b := outer.clip.Bounds
	whole := Clip{Bounds: ui.Rect{Left: math.Floor(b.Left), Top: math.Floor(b.Top), Right: math.Ceil(b.Right), Bottom: math.Ceil(b.Bottom)}}
	f.emit(Command{Kind: CommandBeginLayer, Layer: l, Transform: inner.m, Clip: outer.clip})
	err := f.children(l, state{m: inner.m, clip: whole})
	f.end()
	return err
}

// clipLayer draws the children of l clipped by the shape that apply makes.
// ClipNone does not clip; ClipAntiAliasWithSaveLayer draws the children into a
// group and clips the group as a whole.
func (f *flattener) clipLayer(l ui.Layer, st state, behavior ui.Clip, apply func(c Clip, m *ui.Matrix4, aa bool) Clip) error {
	switch behavior {
	case ui.ClipNone:
		return f.children(l, st)
	case ui.ClipAntiAliasWithSaveLayer:
		clipped := state{m: st.m, clip: apply(st.clip, &st.m, true)}
		return f.group(l, clipped, st)
	}
	return f.children(l, state{m: st.m, clip: apply(st.clip, &st.m, behavior != ui.ClipHardEdge)})
}

// stackEntry is a Save or SaveLayer that has not been restored.
type stackEntry struct {
	saved state
	layer bool
}

// picture flattens the operations of a picture, starting in the state st.
func (f *flattener) picture(p *ui.Picture, st state) error {
	if p == nil {
		return nil
	}
	var stack []stackEntry
	restore := func() {
		e := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		st = e.saved
		if e.layer {
			f.end()
		}
	}
	for _, op := range p.DisplayList().Ops() {
		switch v := op.(type) {
		case ui.OpSave:
			stack = append(stack, stackEntry{saved: st})
		case ui.OpSaveLayer:
			stack = append(stack, stackEntry{saved: st, layer: true})
			f.begin(nil, v, st)
		case ui.OpRestore:
			if len(stack) > 0 {
				restore()
			}
		case ui.OpTransform:
			st.m.MultiplyBy(&v.Matrix)
		case ui.OpClipRect:
			st.clip = clipRect(st.clip, &st.m, v.Rect, v.AntiAlias, v.Op == ui.ClipOpDifference)
		case ui.OpClipRRect:
			st.clip = clipRRect(st.clip, &st.m, v.RRect, v.AntiAlias)
		case ui.OpClipRSuperellipse:
			st.clip = clipRSuperellipse(st.clip, &st.m, v.RSuperellipse, v.AntiAlias)
		case ui.OpClipPath:
			st.clip = clipPath(st.clip, &st.m, v.Path, v.AntiAlias)
		case ui.OpDrawPicture:
			if err := f.picture(v.Picture, st); err != nil {
				return err
			}
		case ui.OpDrawParagraph:
			// A paragraph is drawn as the glyphs, shadows, backgrounds and decorations it
			// is made of, each under its own transform.
			var err error
			textdraw.Expand(v, &st.m, func(op ui.DisplayOp, m *ui.Matrix4) {
				if err != nil {
					return
				}
				sub := st
				sub.m = *m
				if pic, ok := op.(ui.OpDrawPicture); ok {
					err = f.picture(pic.Picture, sub)
					return
				}
				f.emit(Command{Kind: CommandDraw, Op: op, Transform: *m, Clip: st.clip})
			})
			if err != nil {
				return err
			}
		default:
			f.emit(Command{Kind: CommandDraw, Op: op, Transform: st.m, Clip: st.clip})
		}
	}
	// A recording may end with layers still open; they are composited as if
	// restored.
	for len(stack) > 0 {
		restore()
	}
	return nil
}

// snap rounds an edge to the nearest pixel boundary: a clip without
// anti-aliasing keeps the pixels whose centers are inside.
func snap(v float64) float64 { return math.Floor(v + 0.5) }

// clipRect intersects (or, with difference, subtracts) a rectangle under the
// transform m.
func clipRect(c Clip, m *ui.Matrix4, r ui.Rect, aa, difference bool) Clip {
	if !difference && IsAxisAligned(m) {
		d := m.TransformRect(r)
		if !aa {
			d = ui.Rect{Left: snap(d.Left), Top: snap(d.Top), Right: snap(d.Right), Bottom: snap(d.Bottom)}
		}
		return Clip{Bounds: c.Bounds.Intersect(d), Shapes: c.Shapes}
	}
	return c.withShape(ClipShape{Kind: ClipShapeRect, Rect: r, Difference: difference, AntiAlias: aa, Transform: *m}, r)
}

func clipRRect(c Clip, m *ui.Matrix4, r ui.RRect, aa bool) Clip {
	if r.IsRect() {
		return clipRect(c, m, ui.Rect{Left: r.Left, Top: r.Top, Right: r.Right, Bottom: r.Bottom}, aa, false)
	}
	return c.withShape(ClipShape{Kind: ClipShapeRRect, RRect: r, AntiAlias: aa, Transform: *m}, r.OuterRect())
}

func clipRSuperellipse(c Clip, m *ui.Matrix4, r ui.RSuperellipse, aa bool) Clip {
	return c.withShape(ClipShape{Kind: ClipShapeRSuperellipse, RSuperellipse: r, AntiAlias: aa, Transform: *m}, r.OuterRect())
}

func clipPath(c Clip, m *ui.Matrix4, p *ui.Path, aa bool) Clip {
	return c.withShape(ClipShape{Kind: ClipShapePath, Path: p, AntiAlias: aa, Transform: *m}, p.GetBounds())
}
