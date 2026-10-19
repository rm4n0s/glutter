package rendering_test

import (
	"math"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// annotatedLayer is _TestAnnotatedLayer of layer_annotations_test.dart.
type annotatedLayer struct {
	rendering.LayerBase
	value  int
	opaque bool
	offset ui.Offset
	size   foundation.Opt[ui.Size]
}

func newAnnotated(value int, opaque bool, offset ui.Offset, size foundation.Opt[ui.Size]) *annotatedLayer {
	l := &annotatedLayer{value: value, opaque: opaque, offset: offset, size: size}
	rendering.InitLayer(&l.LayerBase, l)
	return l
}

func (*annotatedLayer) AddToScene(*ui.SceneBuilder) {}

func (l *annotatedLayer) FindAnnotations(result *rendering.AnnotationResult, pos ui.Offset, onlyFirst bool) bool {
	if !result.Accepts(l.value) {
		return false
	}
	if s, ok := l.size.Get(); ok && !l.offset.And(s).Contains(pos) {
		return false
	}
	result.Add(rendering.AnnotationEntry{Annotation: l.value, LocalPosition: pos})
	return l.opaque
}

func container(children ...rendering.Layer) *rendering.ContainerLayer {
	c := rendering.NewContainerLayer()
	for _, ch := range children {
		c.Append(ch)
	}
	return c
}

// withBackground wraps the layer so that an annotation with the value is found
// everywhere, behind the layer.
func withBackground(value int, layer rendering.Layer) rendering.Layer {
	return container(newAnnotated(value, false, ui.OffsetZero, foundation.None[ui.Size]()), layer)
}

type found struct {
	value int
	pos   ui.Offset
}

func findAll(root rendering.Layer, pos ui.Offset) []found {
	res := rendering.LayerFindAllAnnotations[int](root, pos)
	var out []found
	for _, e := range res.Entries() {
		out = append(out, found{e.Annotation.(int), e.LocalPosition})
	}
	return out
}

func sameFound(a, b []found) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].value != b[i].value || math.Abs(a[i].pos.Dx-b[i].pos.Dx) > 1e-6 || math.Abs(a[i].pos.Dy-b[i].pos.Dy) > 1e-6 {
			return false
		}
	}
	return true
}

func TestLayerAnnotationsContainer(t *testing.T) {
	none := foundation.None[ui.Size]()
	mk := func(opaque2 bool) rendering.Layer {
		return container(newAnnotated(1, false, ui.OffsetZero, none), newAnnotated(2, opaque2, ui.OffsetZero, none), newAnnotated(3, false, ui.OffsetZero, none))
	}
	tests := []struct {
		name string
		root rendering.Layer
		want []found
	}{
		{"all results, front first", mk(false), []found{{3, ui.OffsetZero}, {2, ui.OffsetZero}, {1, ui.OffsetZero}}},
		{"stops at the first opaque child", mk(true), []found{{3, ui.OffsetZero}, {2, ui.OffsetZero}}},
		{"children's opacity (true)", withBackground(1000, container(newAnnotated(2, true, ui.OffsetZero, none))), []found{{2, ui.OffsetZero}}},
		{"children's opacity (false)", withBackground(1000, container(newAnnotated(2, false, ui.OffsetZero, none))), []found{{2, ui.OffsetZero}, {1000, ui.OffsetZero}}},
		{"false opacity when finding nothing", withBackground(1000, container(newAnnotated(2, false, ui.OffsetZero, foundation.Some(ui.SizeZero)))), []found{{1000, ui.OffsetZero}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := findAll(tc.root, ui.OffsetZero); !sameFound(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
	root := mk(false)
	if v, ok := rendering.LayerFind[int](root, ui.OffsetZero); !ok || v != 3 {
		t.Errorf("Find = %v, %v; want 3", v, ok)
	}
	if got := findAll(root, ui.OffsetZero); len(got) != 3 {
		t.Errorf("found %d", len(got))
	}
	if res := rendering.LayerFindAllAnnotations[float64](root, ui.OffsetZero); len(res.Entries()) != 0 {
		t.Error("found a float64 where there are none")
	}
	if _, ok := rendering.LayerFind[float64](root, ui.OffsetZero); ok {
		t.Error("Find found a float64")
	}
}

func TestLayerAnnotationsRespectGeometry(t *testing.T) {
	size10 := foundation.Some(ui.Size{Width: 10, Height: 10})
	tri := ui.NewPath()
	tri.LineTo(2, 0)
	tri.LineTo(0, 2)
	tri.Close()
	tri = tri.Shift(ui.Offset{Dx: 10, Dy: 10})
	rrect := ui.NewRRectFromRectAndRadius(ui.NewRectFromLTWH(10, 10, 10, 10), ui.NewRadiusCircular(4))
	at10 := ui.Offset{Dx: 10, Dy: 10}
	tests := []struct {
		name string
		root rendering.Layer
		pos  ui.Offset
		want []found
	}{
		{"offset inside", withBackground(1000, func() rendering.Layer {
			o := rendering.NewOffsetLayer(rendering.OffsetLayerOpts{Offset: ui.Offset{Dx: -10}})
			o.Append(newAnnotated(1, true, ui.OffsetZero, size10))
			return o
		}()), ui.Offset{Dx: -5, Dy: 5}, []found{{1, ui.Offset{Dx: 5, Dy: 5}}}},
		{"offset outside", withBackground(1000, func() rendering.Layer {
			o := rendering.NewOffsetLayer(rendering.OffsetLayerOpts{Offset: ui.Offset{Dx: -10}})
			o.Append(newAnnotated(1, true, ui.OffsetZero, size10))
			return o
		}()), ui.Offset{Dx: 5, Dy: 5}, []found{{1000, ui.Offset{Dx: 5, Dy: 5}}}},
		{"clipRect inside", withBackground(1000, func() rendering.Layer {
			c := rendering.NewClipRectLayer(rendering.ClipRectLayerOpts{ClipRect: foundation.Some(at10.And(ui.Size{Width: 5, Height: 5}))})
			c.Append(newAnnotated(1, true, at10, size10))
			return c
		}()), ui.Offset{Dx: 11, Dy: 11}, []found{{1, ui.Offset{Dx: 11, Dy: 11}}}},
		{"clipRect outside", withBackground(1000, func() rendering.Layer {
			c := rendering.NewClipRectLayer(rendering.ClipRectLayerOpts{ClipRect: foundation.Some(at10.And(ui.Size{Width: 5, Height: 5}))})
			c.Append(newAnnotated(1, true, at10, size10))
			return c
		}()), ui.Offset{Dx: 19, Dy: 19}, []found{{1000, ui.Offset{Dx: 19, Dy: 19}}}},
		{"clipRRect inside", withBackground(1000, func() rendering.Layer {
			c := rendering.NewClipRRectLayer(rendering.ClipRRectLayerOpts{ClipRRect: foundation.Some(rrect)})
			c.Append(newAnnotated(1, true, at10, size10))
			return c
		}()), ui.Offset{Dx: 12, Dy: 12}, []found{{1, ui.Offset{Dx: 12, Dy: 12}}}},
		{"clipRRect outside", withBackground(1000, func() rendering.Layer {
			c := rendering.NewClipRRectLayer(rendering.ClipRRectLayerOpts{ClipRRect: foundation.Some(rrect)})
			c.Append(newAnnotated(1, true, at10, size10))
			return c
		}()), ui.Offset{Dx: 11, Dy: 11}, []found{{1000, ui.Offset{Dx: 11, Dy: 11}}}},
		// Flutter's test uses (11, 11), a point on the outline; Path.Contains of ui is
		// half-open there, so this uses a point inside.
		{"clipPath inside", withBackground(1000, func() rendering.Layer {
			c := rendering.NewClipPathLayer(rendering.ClipPathLayerOpts{ClipPath: tri})
			c.Append(newAnnotated(1, true, at10, size10))
			return c
		}()), ui.Offset{Dx: 10.5, Dy: 10.5}, []found{{1, ui.Offset{Dx: 10.5, Dy: 10.5}}}},
		{"clipPath outside", withBackground(1000, func() rendering.Layer {
			c := rendering.NewClipPathLayer(rendering.ClipPathLayerOpts{ClipPath: tri})
			c.Append(newAnnotated(1, true, at10, size10))
			return c
		}()), ui.Offset{Dx: 12, Dy: 12}, []found{{1000, ui.Offset{Dx: 12, Dy: 12}}}},
		{"transform inside", withBackground(1000, func() rendering.Layer {
			m := ui.NewMatrix4Diagonal3Values(2, 4, 1)
			m.SetTranslationRaw(10, 20, 0)
			tl := rendering.NewTransformLayer(rendering.TransformLayerOpts{Transform: m})
			tl.Append(newAnnotated(1, true, at10, size10))
			return tl
		}()), ui.Offset{Dx: 40, Dy: 80}, []found{{1, ui.Offset{Dx: 15, Dy: 15}}}},
		{"transform outside", withBackground(1000, func() rendering.Layer {
			m := ui.NewMatrix4Diagonal3Values(2, 4, 1)
			m.SetTranslationRaw(10, 20, 0)
			tl := rendering.NewTransformLayer(rendering.TransformLayerOpts{Transform: m})
			tl.Append(newAnnotated(1, true, at10, size10))
			return tl
		}()), ui.Offset{Dx: 20, Dy: 40}, []found{{1000, ui.Offset{Dx: 20, Dy: 40}}}},
		{"singular transform finds nothing", func() rendering.Layer {
			tl := rendering.NewTransformLayer(rendering.TransformLayerOpts{Transform: ui.NewMatrix4Zero()})
			tl.Append(newAnnotated(1, true, ui.OffsetZero, foundation.None[ui.Size]()))
			return tl
		}(), ui.OffsetZero, nil},
		{"leader respects offset", withBackground(1000, func() rendering.Layer {
			l := rendering.NewLeaderLayer(rendering.NewLayerLink(), rendering.LeaderLayerOpts{Offset: ui.Offset{Dx: -10}})
			l.Append(newAnnotated(1, true, ui.OffsetZero, size10))
			return l
		}()), ui.Offset{Dx: -5, Dy: 5}, []found{{1, ui.Offset{Dx: 5, Dy: 5}}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := findAll(tc.root, tc.pos); !sameFound(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLayerAnnotatedRegion(t *testing.T) {
	size := foundation.Some(ui.Size{Width: 10, Height: 10})
	tests := []struct {
		name   string
		layer  func() *rendering.AnnotatedRegionLayer
		child  rendering.Layer
		pos    ui.Offset
		want   []found
		absorb bool
	}{
		{"appends after children", func() *rendering.AnnotatedRegionLayer { return rendering.NewAnnotatedRegionLayer(2) },
			newAnnotated(1, false, ui.OffsetZero, foundation.None[ui.Size]()), ui.OffsetZero, []found{{1, ui.OffsetZero}, {2, ui.OffsetZero}}, false},
		{"opaque absorbs", func() *rendering.AnnotatedRegionLayer {
			return rendering.NewAnnotatedRegionLayer(2, rendering.AnnotatedRegionLayerOpts{Opaque: true})
		}, newAnnotated(1, false, ui.OffsetZero, foundation.None[ui.Size]()), ui.OffsetZero, []found{{1, ui.OffsetZero}, {2, ui.OffsetZero}}, true},
		{"child opacity is kept", func() *rendering.AnnotatedRegionLayer { return rendering.NewAnnotatedRegionLayer(2) },
			newAnnotated(1, true, ui.OffsetZero, foundation.None[ui.Size]()), ui.OffsetZero, []found{{1, ui.OffsetZero}, {2, ui.OffsetZero}}, true},
		{"clipped by size", func() *rendering.AnnotatedRegionLayer {
			return rendering.NewAnnotatedRegionLayer(2, rendering.AnnotatedRegionLayerOpts{Size: size, Offset: ui.Offset{Dx: 5, Dy: 5}})
		}, newAnnotated(1, false, ui.OffsetZero, foundation.None[ui.Size]()), ui.Offset{Dx: 1, Dy: 1}, []found{{1, ui.Offset{Dx: 1, Dy: 1}}}, false},
		{"inside size, local position uses offset", func() *rendering.AnnotatedRegionLayer {
			return rendering.NewAnnotatedRegionLayer(2, rendering.AnnotatedRegionLayerOpts{Size: size, Offset: ui.Offset{Dx: 5, Dy: 5}})
		}, newAnnotated(1, false, ui.OffsetZero, foundation.None[ui.Size]()), ui.Offset{Dx: 6, Dy: 7}, []found{{1, ui.Offset{Dx: 6, Dy: 7}}, {2, ui.Offset{Dx: 1, Dy: 2}}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := tc.layer()
			l.Append(tc.child)
			res := rendering.NewAnnotationResult[int]()
			absorbed := l.FindAnnotations(res, tc.pos, false)
			var got []found
			for _, e := range res.Entries() {
				got = append(got, found{e.Annotation.(int), e.LocalPosition})
			}
			if !sameFound(got, tc.want) || absorbed != tc.absorb {
				t.Errorf("got %v absorbed %v, want %v absorbed %v", got, absorbed, tc.want, tc.absorb)
			}
		})
	}
	t.Run("onlyFirst stops at the child", func(t *testing.T) {
		l := rendering.NewAnnotatedRegionLayer(2)
		l.Append(newAnnotated(1, false, ui.OffsetZero, foundation.None[ui.Size]()))
		v, ok := rendering.LayerFind[int](l, ui.OffsetZero)
		if !ok || v != 1 {
			t.Errorf("Find = %v, %v; want 1", v, ok)
		}
	})
}
