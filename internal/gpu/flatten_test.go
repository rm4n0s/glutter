package gpu

import (
	"math"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

func paintOf(argb uint32) *ui.Paint {
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(argb))
	return p
}

func pictureOf(fn func(c *ui.Canvas)) *ui.Picture {
	rec := ui.NewPictureRecorder()
	fn(ui.NewCanvas(rec))
	return rec.EndRecording()
}

func sceneOfPicture(offset ui.Offset, pic *ui.Picture) *ui.Scene {
	b := ui.NewSceneBuilder()
	b.AddPicture(offset, pic)
	return b.Build()
}

func mustFlatten(t *testing.T, s *ui.Scene) *DrawList {
	t.Helper()
	l, err := Flatten(s, FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, 100, 100)})
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func kinds(l *DrawList) []CommandKind {
	var out []CommandKind
	for _, c := range l.Commands {
		out = append(out, c.Kind)
	}
	return out
}

func equalKinds(a, b []CommandKind) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// origin maps (0, 0) with the command's transform.
func origin(c Command) ui.Offset { return c.Transform.TransformPoint(ui.Offset{}) }

func TestFlattenTransformsAndOffsets(t *testing.T) {
	rect := ui.NewRectFromLTWH(0, 0, 10, 10)
	pic := pictureOf(func(c *ui.Canvas) {
		c.Translate(1, 2)
		c.DrawRect(rect, paintOf(0xFF112233))
		c.Save()
		c.Scale(2, 2)
		c.DrawRect(rect, paintOf(0xFF445566))
		c.Restore()
		c.DrawRect(rect, paintOf(0xFF778899))
	})
	b := ui.NewSceneBuilder()
	b.PushOffset(10, 20)
	b.PushTransform(ui.NewMatrix4TranslationValues(100, 0, 0))
	b.AddPicture(ui.Offset{Dx: 1000, Dy: 0}, pic)
	l, err := Flatten(b.Build())
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Commands) != 3 {
		t.Fatalf("%d commands, want 3", len(l.Commands))
	}
	// Every layer's offset, then the picture's, then the canvas's translation.
	want := []ui.Offset{{Dx: 1111, Dy: 22}, {Dx: 1111, Dy: 22}, {Dx: 1111, Dy: 22}}
	for i, c := range l.Commands {
		if c.Kind != CommandDraw {
			t.Errorf("command %d is %v", i, c.Kind)
		}
		if got := origin(c); got != want[i] {
			t.Errorf("command %d origin = %v, want %v", i, got, want[i])
		}
	}
	// The scale applies to the second command only.
	if got := l.Commands[1].Transform.TransformPoint(ui.Offset{Dx: 1, Dy: 1}); got != (ui.Offset{Dx: 1113, Dy: 24}) {
		t.Errorf("scaled point = %v", got)
	}
	if got := l.Commands[2].Transform.TransformPoint(ui.Offset{Dx: 1, Dy: 1}); got != (ui.Offset{Dx: 1112, Dy: 23}) {
		t.Errorf("point after Restore = %v", got)
	}
	if _, ok := l.Commands[0].Op.(ui.OpDrawRect); !ok {
		t.Errorf("Op = %T", l.Commands[0].Op)
	}
}

func TestFlattenClips(t *testing.T) {
	tests := []struct {
		name   string
		draw   func(c *ui.Canvas)
		bounds ui.Rect
		shapes int
		empty  bool
	}{
		{"none", func(*ui.Canvas) {}, ui.NewRectFromLTWH(0, 0, 100, 100), 0, false},
		{"aa rect keeps fractions", func(c *ui.Canvas) { c.ClipRect(ui.NewRectFromLTWH(10.25, 10.5, 20, 20)) },
			ui.NewRectFromLTWH(10.25, 10.5, 20, 20), 0, false},
		{"hard rect snaps to pixels", func(c *ui.Canvas) {
			c.ClipRect(ui.NewRectFromLTWH(10.25, 10.5, 20, 20), ui.ClipRectOpts{DoAntiAlias: ui.Ptr(false)})
		}, ui.Rect{Left: 10, Top: 11, Right: 30, Bottom: 31}, 0, false},
		{"nested rects intersect", func(c *ui.Canvas) {
			c.ClipRect(ui.NewRectFromLTWH(0, 0, 50, 50))
			c.ClipRect(ui.NewRectFromLTWH(30, 30, 50, 50))
		}, ui.Rect{Left: 30, Top: 30, Right: 50, Bottom: 50}, 0, false},
		{"rect under a translation", func(c *ui.Canvas) {
			c.Translate(5, 6)
			c.ClipRect(ui.NewRectFromLTWH(0, 0, 10, 10))
		}, ui.Rect{Left: 5, Top: 6, Right: 15, Bottom: 16}, 0, false},
		{"rect under a quarter turn", func(c *ui.Canvas) {
			c.Translate(50, 50)
			c.Rotate(math.Pi / 2)
			c.ClipRect(ui.NewRectFromLTWH(0, 0, 10, 20))
		}, ui.Rect{Left: 30, Top: 50, Right: 50, Bottom: 60}, 0, false},
		{"rect under a rotation is a shape", func(c *ui.Canvas) {
			c.Translate(50, 50)
			c.Rotate(math.Pi / 4)
			c.ClipRect(ui.NewRectFromLTWH(0, 0, 10, 10))
		}, ui.Rect{Left: math.Floor(50 - 10/math.Sqrt2), Top: 50, Right: math.Ceil(50 + 10/math.Sqrt2), Bottom: math.Ceil(50 + 20/math.Sqrt2)}, 1, false},
		{"rect difference is a shape and keeps the bounds", func(c *ui.Canvas) {
			c.ClipRect(ui.NewRectFromLTWH(10, 10, 10, 10), ui.ClipRectOpts{ClipOp: ui.ClipOpDifference})
		}, ui.NewRectFromLTWH(0, 0, 100, 100), 1, false},
		{"rrect", func(c *ui.Canvas) { c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10, 10, 30, 30), 5, 5)) },
			ui.NewRectFromLTWH(10, 10, 30, 30), 1, false},
		{"rrect without radii is a rect", func(c *ui.Canvas) { c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10, 10, 30, 30), 0, 0)) },
			ui.NewRectFromLTWH(10, 10, 30, 30), 0, false},
		{"superellipse", func(c *ui.Canvas) {
			c.ClipRSuperellipse(ui.NewRSuperellipseFromRectXY(ui.NewRectFromLTWH(10, 10, 30, 30), 5, 5))
		}, ui.NewRectFromLTWH(10, 10, 30, 30), 1, false},
		{"path", func(c *ui.Canvas) {
			p := ui.NewPath()
			p.AddOval(ui.NewRectFromLTWH(20, 20, 40, 40))
			c.ClipPath(p)
		}, ui.NewRectFromLTWH(20, 20, 40, 40), 1, false},
		{"disjoint clips are empty", func(c *ui.Canvas) {
			c.ClipRect(ui.NewRectFromLTWH(0, 0, 10, 10))
			c.ClipRect(ui.NewRectFromLTWH(50, 50, 10, 10))
		}, ui.Rect{Left: 50, Top: 50, Right: 10, Bottom: 10}, 0, true},
		{"restore brings the clip back", func(c *ui.Canvas) {
			c.Save()
			c.ClipRect(ui.NewRectFromLTWH(0, 0, 10, 10))
			c.Restore()
		}, ui.NewRectFromLTWH(0, 0, 100, 100), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pic := pictureOf(func(c *ui.Canvas) {
				tt.draw(c)
				c.DrawRect(ui.NewRectFromLTWH(0, 0, 5, 5), paintOf(0xFF000000))
			})
			l := mustFlatten(t, sceneOfPicture(ui.Offset{}, pic))
			if len(l.Commands) != 1 {
				t.Fatalf("%d commands", len(l.Commands))
			}
			clip := l.Commands[0].Clip
			if !clip.IsEmpty() != !tt.empty {
				t.Errorf("IsEmpty = %v, want %v (bounds %v)", clip.IsEmpty(), tt.empty, clip.Bounds)
			}
			if !tt.empty {
				const eps = 1e-4
				b, w := clip.Bounds, tt.bounds
				if math.Abs(b.Left-w.Left) > eps || math.Abs(b.Top-w.Top) > eps || math.Abs(b.Right-w.Right) > eps || math.Abs(b.Bottom-w.Bottom) > eps {
					t.Errorf("Bounds = %v, want %v", b, w)
				}
			}
			if len(clip.Shapes) != tt.shapes || clip.IsRect() != (tt.shapes == 0) {
				t.Errorf("%d shapes (IsRect %v), want %d", len(clip.Shapes), clip.IsRect(), tt.shapes)
			}
		})
	}
}

func TestFlattenClipShapesKeepTheirTransform(t *testing.T) {
	pic := pictureOf(func(c *ui.Canvas) {
		c.Translate(10, 0)
		c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(0, 0, 20, 20), 4, 4), ui.ClipOpts{DoAntiAlias: ui.Ptr(false)})
		c.Translate(0, 7)
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 5, 5), paintOf(0xFF000000))
	})
	cmd := mustFlatten(t, sceneOfPicture(ui.Offset{}, pic)).Commands[0]
	s := cmd.Clip.Shapes[0]
	if s.Kind != ClipShapeRRect || s.AntiAlias {
		t.Errorf("shape = %+v", s)
	}
	if got := s.Transform.TransformPoint(ui.Offset{}); got != (ui.Offset{Dx: 10}) {
		t.Errorf("the shape's transform maps the origin to %v, want 10,0 (not the later translation)", got)
	}
	if got := origin(cmd); got != (ui.Offset{Dx: 10, Dy: 7}) {
		t.Errorf("command origin = %v", got)
	}
}

func TestFlattenClipEqual(t *testing.T) {
	pic := pictureOf(func(c *ui.Canvas) {
		c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(0, 0, 20, 20), 4, 4))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 5, 5), paintOf(0xFF000000))
		c.DrawRect(ui.NewRectFromLTWH(5, 0, 5, 5), paintOf(0xFF000000))
		c.Translate(1, 1) // a transform does not change the clip
		c.DrawRect(ui.NewRectFromLTWH(5, 0, 5, 5), paintOf(0xFF000000))
		c.ClipRect(ui.NewRectFromLTWH(0, 0, 10, 10))
		c.DrawRect(ui.NewRectFromLTWH(5, 0, 5, 5), paintOf(0xFF000000))
	})
	cmds := mustFlatten(t, sceneOfPicture(ui.Offset{}, pic)).Commands
	if !cmds[0].Clip.Equal(cmds[1].Clip) || !cmds[1].Clip.Equal(cmds[2].Clip) {
		t.Error("commands drawn under the same clip should have equal clips")
	}
	if cmds[2].Clip.Equal(cmds[3].Clip) {
		t.Error("a narrower clip is not equal")
	}
}

func TestFlattenLayers(t *testing.T) {
	rect := ui.NewRectFromLTWH(0, 0, 10, 10)
	pic := pictureOf(func(c *ui.Canvas) { c.DrawRect(rect, paintOf(0xFF000000)) })
	b := ui.NewSceneBuilder()
	b.PushOpacity(128, ui.PushOpacityOpts{Offset: ui.Offset{Dx: 3, Dy: 4}})
	b.AddPicture(ui.Offset{}, pic)
	b.Pop()
	b.PushColorFilter(ui.NewColorFilterMode(ui.NewColor(0xFFFF0000), ui.BlendModeSrcIn))
	b.AddPicture(ui.Offset{}, pic)
	b.Pop()
	b.PushClipRect(ui.NewRectFromLTWH(0, 0, 5.5, 5.5), ui.PushClipRectOpts{ClipBehavior: ui.ClipAntiAliasWithSaveLayer})
	b.AddPicture(ui.Offset{}, pic)
	b.Pop()
	b.PushClipRect(ui.NewRectFromLTWH(0, 0, 5, 5), ui.PushClipRectOpts{ClipBehavior: ui.ClipNone})
	b.AddPicture(ui.Offset{}, pic)
	b.Pop()
	b.PushClipRect(ui.NewRectFromLTWH(0, 0, 5.5, 5.5), ui.PushClipRectOpts{ClipBehavior: ui.ClipHardEdge})
	b.AddPicture(ui.Offset{}, pic)
	b.Pop()
	l := mustFlatten(t, b.Build())

	want := []CommandKind{
		CommandBeginLayer, CommandDraw, CommandEndLayer, // opacity
		CommandBeginLayer, CommandDraw, CommandEndLayer, // color filter
		CommandBeginLayer, CommandDraw, CommandEndLayer, // clip with a save layer
		CommandDraw, // ClipNone
		CommandDraw, // hard edge
	}
	if got := kinds(l); !equalKinds(got, want) {
		t.Fatalf("kinds = %v, want %v", got, want)
	}
	cmds := l.Commands
	if _, ok := cmds[0].Layer.(*ui.OpacityLayer); !ok {
		t.Errorf("layer = %T", cmds[0].Layer)
	}
	// The offset of an opacity layer moves what is inside, not the group.
	if got := origin(cmds[0]); got != (ui.Offset{}) {
		t.Errorf("group transform maps the origin to %v", got)
	}
	if got := origin(cmds[1]); got != (ui.Offset{Dx: 3, Dy: 4}) {
		t.Errorf("content origin = %v, want 3,4", got)
	}
	// A clip with a save layer clips the group, not what is inside.
	if got := cmds[6].Clip.Bounds; got != ui.NewRectFromLTWH(0, 0, 5.5, 5.5) {
		t.Errorf("group clip = %v", got)
	}
	if got := cmds[7].Clip.Bounds; got != ui.NewRectFromLTWH(0, 0, 100, 100) {
		t.Errorf("content clip = %v, want the unclipped bounds", got)
	}
	if got := cmds[9].Clip.Bounds; got != ui.NewRectFromLTWH(0, 0, 100, 100) {
		t.Errorf("ClipNone clip = %v", got)
	}
	if got := cmds[10].Clip.Bounds; got != (ui.Rect{Left: 0, Top: 0, Right: 6, Bottom: 6}) {
		t.Errorf("hard-edge clip = %v, want snapped to 6", got)
	}
}

func TestFlattenSaveLayer(t *testing.T) {
	rect := ui.NewRectFromLTWH(0, 0, 10, 10)
	p := ui.NewPaint()
	p.SetColor(ui.NewColor(0x80000000))
	pic := pictureOf(func(c *ui.Canvas) {
		c.Translate(5, 5)
		c.SaveLayer(nil, p)
		c.Translate(1, 1)
		c.DrawRect(rect, paintOf(0xFF000000))
		c.Restore()
		c.DrawRect(rect, paintOf(0xFF000000))
		c.SaveLayer(nil, p) // never restored
		c.DrawRect(rect, paintOf(0xFF000000))
	})
	l := mustFlatten(t, sceneOfPicture(ui.Offset{}, pic))
	want := []CommandKind{
		CommandBeginLayer, CommandDraw, CommandEndLayer, CommandDraw,
		CommandBeginLayer, CommandDraw, CommandEndLayer,
	}
	if got := kinds(l); !equalKinds(got, want) {
		t.Fatalf("kinds = %v, want %v", got, want)
	}
	if _, ok := l.Commands[0].Op.(ui.OpSaveLayer); !ok || l.Commands[0].Layer != nil {
		t.Errorf("begin = %T, layer %v", l.Commands[0].Op, l.Commands[0].Layer)
	}
	// The group is composited where it was begun, and the content moves inside.
	if got := origin(l.Commands[0]); got != (ui.Offset{Dx: 5, Dy: 5}) {
		t.Errorf("group origin = %v", got)
	}
	if got := origin(l.Commands[1]); got != (ui.Offset{Dx: 6, Dy: 6}) {
		t.Errorf("content origin = %v", got)
	}
	if got := origin(l.Commands[3]); got != (ui.Offset{Dx: 5, Dy: 5}) {
		t.Errorf("after Restore origin = %v", got)
	}
}

func TestFlattenNestedPictures(t *testing.T) {
	rect := ui.NewRectFromLTWH(0, 0, 10, 10)
	inner := pictureOf(func(c *ui.Canvas) { c.DrawRect(rect, paintOf(0xFF000000)) })
	outer := pictureOf(func(c *ui.Canvas) {
		c.Translate(7, 0)
		c.ClipRect(ui.NewRectFromLTWH(0, 0, 8, 8))
		c.DrawPicture(inner)
		c.DrawPicture(inner)
	})
	l := mustFlatten(t, sceneOfPicture(ui.Offset{Dy: 1}, outer))
	if len(l.Commands) != 2 {
		t.Fatalf("%d commands", len(l.Commands))
	}
	for _, c := range l.Commands {
		if got := origin(c); got != (ui.Offset{Dx: 7, Dy: 1}) {
			t.Errorf("origin = %v", got)
		}
		if got := c.Clip.Bounds; got != (ui.Rect{Left: 7, Top: 1, Right: 15, Bottom: 9}) {
			t.Errorf("clip = %v", got)
		}
	}
}

func TestFlattenErrors(t *testing.T) {
	_, err := Flatten(nil)
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagFlattenSceneNil || !appErr.HasRoute("Flatten->FlattenFunc.FlattenSceneNil") {
		t.Errorf("Flatten(nil) = %v", err)
	}
	func() {
		defer func() {
			e, _ := recover().(error)
			appErr, ok := errors.FromError(e)
			if !ok || appErr.Tag != tagMultipleOptions || !appErr.HasRoute("Flatten->FlattenFunc.MultipleOptions") {
				t.Errorf("expected MultipleOptions, got %v", e)
			}
		}()
		_, _ = Flatten(ui.NewSceneBuilder().Build(), FlattenOpts{}, FlattenOpts{})
	}()
}

type foreignLayer struct{ ui.Layer }

func TestFlattenUnknownLayer(t *testing.T) {
	f := &flattener{sink: func(Command) error { return nil }}
	err := f.layer(foreignLayer{}, state{})
	appErr, ok := errors.FromError(err)
	if !ok || appErr.Tag != tagFlattenLayerUnknown || !appErr.HasRoute("flattener.layer.FlattenLayerUnknown") {
		t.Errorf("layer = %v", err)
	}
}

func TestFlattenEmptyScene(t *testing.T) {
	l := mustFlatten(t, ui.NewSceneBuilder().Build())
	if len(l.Commands) != 0 {
		t.Errorf("%d commands for an empty scene", len(l.Commands))
	}
}

func TestIsAxisAligned(t *testing.T) {
	rot := func(a float64) *ui.Matrix4 { return ui.NewMatrix4RotationZ(a) }
	persp := ui.NewMatrix4Identity()
	persp.SetEntry(3, 0, 0.001)
	tests := []struct {
		name string
		m    *ui.Matrix4
		want bool
	}{
		{"identity", ui.NewMatrix4Identity(), true},
		{"scale", ui.NewMatrix4Diagonal3Values(2, 3, 1), true},
		{"quarter turn", func() *ui.Matrix4 { m := rot(math.Pi / 2); m.SetEntry(0, 0, 0); m.SetEntry(1, 1, 0); return m }(), true},
		{"rotation", rot(0.3), false},
		{"skew", ui.NewMatrix4SkewX(0.2), false},
		{"perspective", persp, false},
	}
	for _, tt := range tests {
		if got := IsAxisAligned(tt.m); got != tt.want {
			t.Errorf("%s: IsAxisAligned = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestFlattenShapeClipBoundsAreWholePixels(t *testing.T) {
	// The edge of a clip shape is drawn from the shape; the bounds that narrow the
	// clip to it must not cut that edge again, so they are whole pixels.
	pic := pictureOf(func(c *ui.Canvas) {
		c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(5.5, 5.25, 50.2, 40.5), 10, 10))
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 5, 5), paintOf(0xFF000000))
	})
	b := mustFlatten(t, sceneOfPicture(ui.Offset{}, pic)).Commands[0].Clip.Bounds
	if want := (ui.Rect{Left: 5, Top: 5, Right: 56, Bottom: 46}); b != want {
		t.Errorf("Bounds = %v, want %v", b, want)
	}
}

func TestFlattenBoundaryLayersBecomeGroups(t *testing.T) {
	rect := ui.NewRectFromLTWH(0, 0, 10, 10)
	pic := pictureOf(func(c *ui.Canvas) { c.DrawRect(rect, paintOf(0xFF000000)) })
	b := ui.NewSceneBuilder()
	b.PushClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(2.5, 3.5, 60, 40), 6, 6))
	offset := b.PushOffset(5, 6)
	b.AddPicture(ui.Offset{}, pic)
	b.Pop()
	b.PushTransform(ui.NewMatrix4Diagonal3Values(2, 2, 1))
	b.AddPicture(ui.Offset{}, pic)
	b.Pop()
	b.Pop()
	scene := b.Build()

	var asked []ui.Layer
	l, err := Flatten(scene, FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, 100, 100), Boundary: func(l ui.Layer) bool {
		asked = append(asked, l)
		return l == offset.Layer()
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := []CommandKind{CommandBeginLayer, CommandDraw, CommandEndLayer, CommandDraw}
	if got := kinds(l); !equalKinds(got, want) {
		t.Fatalf("kinds = %v, want %v", got, want)
	}
	begin, inside, outside := l.Commands[0], l.Commands[1], l.Commands[3]
	if begin.Layer != offset.Layer() || origin(begin) != (ui.Offset{Dx: 5, Dy: 6}) {
		t.Errorf("begin = layer %v, contents' origin %v", begin.Layer, origin(begin))
	}
	// The group is composited under the clip it is in, and drawn inside under the whole
	// pixels of its bounds, without the shape.
	if begin.Clip.IsRect() || len(begin.Clip.Shapes) != 1 || begin.Clip.Bounds != (ui.Rect{Left: 2, Top: 3, Right: 63, Bottom: 44}) {
		// The bounds of a rounded clip are whole pixels already.
		t.Errorf("group clip = %+v", begin.Clip)
	}
	if !inside.Clip.IsRect() || inside.Clip.Bounds != (ui.Rect{Left: begin.Clip.Bounds.Left, Top: begin.Clip.Bounds.Top, Right: begin.Clip.Bounds.Right, Bottom: begin.Clip.Bounds.Bottom}) {
		t.Errorf("clip inside the group = %+v", inside.Clip)
	}
	if origin(inside) != (ui.Offset{Dx: 5, Dy: 6}) || outside.Clip.IsRect() {
		t.Errorf("inside origin %v, the other transform's clip %+v", origin(inside), outside.Clip)
	}
	// Every layer that is folded is asked about, and only those.
	if len(asked) < 3 {
		t.Errorf("asked about %d layers", len(asked))
	}
	for _, a := range asked {
		switch a.(type) {
		case *ui.ContainerLayer, *ui.TransformLayer:
		default:
			t.Errorf("asked about %T", a)
		}
	}
	// Without the option nothing changes.
	if got := kinds(mustFlatten(t, scene)); len(got) != 2 {
		t.Errorf("kinds without Boundary = %v", got)
	}
	// Fractional bounds are widened to whole pixels inside the group.
	c := ui.NewSceneBuilder()
	c.PushClipRect(ui.NewRectFromLTWH(1.25, 1.25, 10, 10))
	inner := c.PushOffset(0, 0)
	c.AddPicture(ui.Offset{}, pic)
	c.Pop()
	c.Pop()
	frac, err := Flatten(c.Build(), FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, 100, 100), Boundary: func(l ui.Layer) bool { return l == inner.Layer() }})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := frac.Commands[0].Clip.Bounds, (ui.Rect{Left: 1.25, Top: 1.25, Right: 11.25, Bottom: 11.25}); got != want {
		t.Errorf("group clip bounds = %+v, want %+v", got, want)
	}
	if got, want := frac.Commands[1].Clip.Bounds, (ui.Rect{Left: 1, Top: 1, Right: 12, Bottom: 12}); got != want {
		t.Errorf("clip inside the group = %+v, want %+v", got, want)
	}
}

// glyphs is a paragraph that draws itself as the pieces it holds.
type glyphs struct {
	ui.ParagraphLayout
	draws []ui.ParagraphDraw
}

func (g glyphs) PaintParagraph(emit func(ui.ParagraphDraw)) {
	for _, d := range g.draws {
		emit(d)
	}
}

func TestFlattenParagraphBecomesItsPieces(t *testing.T) {
	path := ui.NewPath()
	path.AddRect(ui.NewRectFromLTWH(0, 0, 1, 1))
	inner := pictureOf(func(c *ui.Canvas) { c.DrawRect(ui.NewRectFromLTWH(0, 0, 2, 2), paintOf(0xFF00FF00)) })
	para := ui.NewParagraph(glyphs{draws: []ui.ParagraphDraw{
		{Op: ui.OpDrawPath{Path: path, Paint: *paintOf(0xFF000000)}, Offset: ui.Offset{Dx: 1.3, Dy: 2.6}, Snap: true},
		{Op: ui.OpDrawPicture{Picture: inner}, Offset: ui.Offset{Dx: 20, Dy: 4}},
		{Op: ui.OpDrawRect{Rect: ui.NewRectFromLTWH(0, 0, 5, 1), Paint: *paintOf(0xFFFF0000)}, Offset: ui.Offset{Dx: 0, Dy: 9}},
	}})
	pic := pictureOf(func(c *ui.Canvas) {
		c.ClipRect(ui.NewRectFromLTWH(0, 0, 50, 50))
		c.DrawParagraph(para, ui.Offset{Dx: 10, Dy: 10})
	})
	l := mustFlatten(t, sceneOfPicture(ui.Offset{}, pic))
	if len(l.Commands) != 3 {
		t.Fatalf("%d commands: %v", len(l.Commands), kinds(l))
	}
	want := []ui.Offset{{Dx: 11.25, Dy: 13}, {Dx: 30, Dy: 14}, {Dx: 10, Dy: 19}}
	for i, c := range l.Commands {
		if c.Kind != CommandDraw {
			t.Errorf("command %d: kind %v", i, c.Kind)
		}
		if got := origin(c); got != want[i] {
			t.Errorf("command %d at %v, want %v", i, got, want[i])
		}
		if c.Clip.Bounds != ui.NewRectFromLTWH(0, 0, 50, 50) {
			t.Errorf("command %d: clip %v", i, c.Clip.Bounds)
		}
	}
	if _, ok := l.Commands[0].Op.(ui.OpDrawPath); !ok {
		t.Errorf("a glyph is a path: %T", l.Commands[0].Op)
	}
	// The picture's own commands come out under the piece's transform.
	if _, ok := l.Commands[1].Op.(ui.OpDrawRect); !ok {
		t.Errorf("a color glyph's picture is flattened into its drawing: %T", l.Commands[1].Op)
	}
}

func TestFlattenParagraphThatCannotDrawItselfDrawsNothing(t *testing.T) {
	pic := pictureOf(func(c *ui.Canvas) {
		c.DrawParagraph(ui.NewParagraph(struct{ ui.ParagraphLayout }{}), ui.Offset{})
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 1, 1), paintOf(0xFF000000))
	})
	if l := mustFlatten(t, sceneOfPicture(ui.Offset{}, pic)); len(l.Commands) != 1 {
		t.Errorf("%d commands: %v", len(l.Commands), kinds(l))
	}
}

func TestFlattenFuncMakesTheCommandsOfFlatten(t *testing.T) {
	pic := pictureOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), paintOf(0xFF000000))
		c.SaveLayer(nil, paintOf(0x80000000))
		c.DrawRect(ui.NewRectFromLTWH(2, 2, 5, 5), paintOf(0xFFFF0000))
		c.Restore()
	})
	scene := sceneOfPicture(ui.Offset{Dx: 3, Dy: 4}, pic)
	opts := FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, 100, 100)}
	want, err := Flatten(scene, opts)
	if err != nil {
		t.Fatal(err)
	}
	var got []Command
	if err := FlattenFunc(scene, func(c Command) error { got = append(got, c); return nil }, opts); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want.Commands) || len(got) != 4 {
		t.Fatalf("FlattenFunc made %d commands, Flatten %d, want 4", len(got), len(want.Commands))
	}
	for i := range got {
		if got[i].Kind != want.Commands[i].Kind || got[i].Transform != want.Commands[i].Transform || got[i].Clip.Bounds != want.Commands[i].Clip.Bounds {
			t.Errorf("command %d differs: %+v and %+v", i, got[i], want.Commands[i])
		}
	}
}

func TestFlattenFuncStopsAtTheFirstErrorOfTheSink(t *testing.T) {
	pic := pictureOf(func(c *ui.Canvas) {
		for i := 0; i < 5; i++ {
			c.DrawRect(ui.NewRectFromLTWH(float64(i), 0, 1, 1), paintOf(0xFF000000))
		}
	})
	stop := errors.New(tagTestSinkStopped, "the sink stops")
	for _, after := range []int{1, 3, 5} {
		n := 0
		err := FlattenFunc(sceneOfPicture(ui.Offset{}, pic), func(Command) error {
			n++
			if n == after {
				return stop
			}
			return nil
		})
		if err != stop {
			t.Errorf("stop after %d: error = %v, want the sink's own error", after, err)
		}
		if n != after {
			t.Errorf("stop after %d: the sink was called %d times", after, n)
		}
	}
}

func TestFlattenRetainedGroupsHaveNoContents(t *testing.T) {
	pic := pictureOf(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), paintOf(0xFF000000))
		c.DrawRect(ui.NewRectFromLTWH(20, 0, 10, 10), paintOf(0xFF000000))
	})
	build := func() (*ui.Scene, ui.Layer, ui.Layer) {
		b := ui.NewSceneBuilder()
		outer := b.PushOffset(5, 6)
		inner := b.PushOffset(1, 1)
		b.AddPicture(ui.Offset{}, pic)
		b.Pop()
		b.Pop()
		b.AddPicture(ui.Offset{Dx: 50}, pic)
		return b.Build(), outer.Layer(), inner.Layer()
	}
	tests := []struct {
		name string
		// retained says which of the groups (outer, inner) the renderer has the pixels of.
		retained func(outer, inner ui.Layer) func(ui.Layer) bool
		want     []CommandKind
	}{
		{"none retained", func(_, _ ui.Layer) func(ui.Layer) bool { return func(ui.Layer) bool { return false } },
			[]CommandKind{CommandBeginLayer, CommandBeginLayer, CommandDraw, CommandDraw, CommandEndLayer, CommandEndLayer, CommandDraw, CommandDraw}},
		{"the outer group", func(outer, _ ui.Layer) func(ui.Layer) bool { return func(l ui.Layer) bool { return l == outer } },
			[]CommandKind{CommandBeginLayer, CommandEndLayer, CommandDraw, CommandDraw}},
		{"the inner group, which is inside another", func(_, inner ui.Layer) func(ui.Layer) bool { return func(l ui.Layer) bool { return l == inner } },
			[]CommandKind{CommandBeginLayer, CommandBeginLayer, CommandDraw, CommandDraw, CommandEndLayer, CommandEndLayer, CommandDraw, CommandDraw}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scene, outer, inner := build()
			keep := tt.retained(outer, inner)
			var asked []ui.Layer
			l, err := Flatten(scene, FlattenOpts{
				Bounds:   ui.NewRectFromLTWH(0, 0, 100, 100),
				Boundary: func(l ui.Layer) bool { return l == outer || l == inner },
				Retained: func(l ui.Layer, m *ui.Matrix4, clip Clip) bool {
					asked = append(asked, l)
					if clip.Bounds != (ui.Rect{Left: 0, Top: 0, Right: 100, Bottom: 100}) || m.Storage()[12] == 0 {
						t.Errorf("asked with clip %+v and transform %v", clip.Bounds, m.Storage())
					}
					return keep(l)
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := kinds(l); !equalKinds(got, tt.want) {
				t.Errorf("kinds = %v, want %v", got, tt.want)
			}
			// Only a group that is not inside another is asked about.
			if len(asked) != 1 || asked[0] != outer {
				t.Errorf("asked about %d groups, want the outer one only", len(asked))
			}
		})
	}
}

// rowGlyphs is a paragraph of lines of 10 pixels, one rectangle each, that paints only the lines
// that can reach the rows it is asked for, and notes the rows.
type rowGlyphs struct {
	ui.ParagraphLayout
	lines int
	rows  *[2]float64
	asked *int
}

func (g rowGlyphs) draw(i int) ui.ParagraphDraw {
	return ui.ParagraphDraw{Op: ui.OpDrawRect{Rect: ui.NewRectFromLTWH(0, 0, 5, 8), Paint: *paintOf(0xFF000000)}, Offset: ui.Offset{Dy: float64(i) * 10}}
}

func (g rowGlyphs) PaintParagraph(emit func(ui.ParagraphDraw)) {
	for i := 0; i < g.lines; i++ {
		emit(g.draw(i))
	}
}

func (g rowGlyphs) PaintParagraphRows(top, bottom float64, emit func(ui.ParagraphDraw)) {
	*g.rows = [2]float64{top, bottom}
	*g.asked++
	for i := 0; i < g.lines; i++ {
		if y := float64(i) * 10; y+8 >= top && y <= bottom {
			emit(g.draw(i))
		}
	}
}

func TestFlattenExpandsOnlyTheLinesOfAParagraphThatShow(t *testing.T) {
	tests := []struct {
		name      string
		transform *ui.Matrix4
		clip      ui.Rect
		wantRows  [2]float64
		wantLines int
	}{
		{"a window on the middle", ui.NewMatrix4Identity(), ui.NewRectFromLTWH(0, 100, 50, 40), [2]float64{100 - 20, 140 - 20}, 5},
		{"scrolled up by 300 pixels", func() *ui.Matrix4 { m := ui.NewMatrix4Identity(); m.Translate(0, -300, 0); return m }(), ui.NewRectFromLTWH(0, 0, 50, 30), [2]float64{300 - 20, 330 - 20}, 4},
		{"scaled by two", func() *ui.Matrix4 { m := ui.NewMatrix4Identity(); m.Scale(2, 2, 1); return m }(), ui.NewRectFromLTWH(0, 100, 50, 40), [2]float64{50 - 20, 70 - 20}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rows [2]float64
			asked := 0
			para := ui.NewParagraph(rowGlyphs{lines: 1000, rows: &rows, asked: &asked})
			b := ui.NewSceneBuilder()
			b.PushTransform(tt.transform)
			b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) { c.DrawParagraph(para, ui.Offset{Dx: 0, Dy: 20}) }))
			b.Pop()
			l, err := Flatten(b.Build(), FlattenOpts{Bounds: tt.clip})
			if err != nil {
				t.Fatal(err)
			}
			if asked != 1 || rows != tt.wantRows {
				t.Errorf("the paragraph was asked %d times for rows %v, want once for %v", asked, rows, tt.wantRows)
			}
			if len(l.Commands) != tt.wantLines {
				t.Errorf("%d commands, want %d", len(l.Commands), tt.wantLines)
			}
		})
	}
}

func TestFlattenExpandsAWholeParagraphUnderATransformItCannotInvert(t *testing.T) {
	var rows [2]float64
	asked := 0
	para := ui.NewParagraph(rowGlyphs{lines: 30, rows: &rows, asked: &asked})
	perspective := ui.NewMatrix4Identity()
	perspective.SetEntry(3, 0, 0.001)
	b := ui.NewSceneBuilder()
	b.PushTransform(perspective)
	b.AddPicture(ui.Offset{}, pictureOf(func(c *ui.Canvas) { c.DrawParagraph(para, ui.Offset{}) }))
	b.Pop()
	l, err := Flatten(b.Build(), FlattenOpts{Bounds: ui.NewRectFromLTWH(0, 0, 50, 40)})
	if err != nil {
		t.Fatal(err)
	}
	if asked != 0 || len(l.Commands) != 30 {
		t.Errorf("rows asked %d times, %d commands, want the whole paragraph (30 commands)", asked, len(l.Commands))
	}
}

// tagTestSinkStopped is the tag of the error the sink of TestFlattenFuncStopsAtTheFirstErrorOfTheSink returns.
const tagTestSinkStopped = "TestSinkStopped"
