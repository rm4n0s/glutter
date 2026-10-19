package rendering_test

import (
	"math"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// concreteLayer is ConcreteLayer of layers_test.dart.
type concreteLayer struct{ rendering.LayerBase }

func newConcreteLayer() *concreteLayer {
	l := &concreteLayer{}
	rendering.InitLayer(&l.LayerBase, l)
	return l
}

func (*concreteLayer) AddToScene(*ui.SceneBuilder) {}

// alwaysLayer is _TestAlwaysNeedsAddToSceneLayer.
type alwaysLayer struct{ rendering.ContainerLayer }

func newAlwaysLayer() *alwaysLayer {
	l := &alwaysLayer{}
	rendering.InitContainerLayer(&l.ContainerLayer, l)
	return l
}

func (*alwaysLayer) AlwaysNeedsAddToScene() bool { return true }

func debugInfo(l rendering.Layer) []string {
	b := foundation.NewDiagnosticPropertiesBuilder()
	l.DebugFillProperties(b)
	var out []string
	for _, n := range b.Properties {
		if !n.IsFiltered(foundation.DiagnosticLevelInfo) {
			out = append(out, n.String())
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func engineDisposed(e ui.EngineLayer) (disposed bool) {
	defer func() { disposed = recover() != nil }()
	ui.NewSceneBuilder().AddRetained(e)
	return false
}

func TestLayersUpdateSubtreeAlwaysNeedsAddToScene(t *testing.T) {
	skipUnlessDebug(t)
	a, b, c := rendering.NewContainerLayer(), rendering.NewContainerLayer(), rendering.NewContainerLayer()
	d, e, f := newAlwaysLayer(), rendering.NewContainerLayer(), rendering.NewContainerLayer()
	a.Append(b)
	a.Append(c)
	b.Append(d)
	b.Append(e)
	d.Append(f)
	all := []rendering.Layer{a, b, c, d, e, f}
	for _, l := range all {
		l.AsLayer().DebugMarkClean()
	}
	a.UpdateSubtreeNeedsAddToScene()
	want := []bool{true, true, false, true, false, false}
	for i, l := range all {
		if got := l.AsLayer().DebugSubtreeNeedsAddToScene(); got != want[i] {
			t.Errorf("layer %d: needs add to scene = %v, want %v", i, got, want[i])
		}
	}
}

func TestLayersUpdateSubtreeNeedsAddToScene(t *testing.T) {
	skipUnlessDebug(t)
	a, b, c, d, e, f, g := rendering.NewContainerLayer(), rendering.NewContainerLayer(), rendering.NewContainerLayer(),
		rendering.NewContainerLayer(), rendering.NewContainerLayer(), rendering.NewContainerLayer(), rendering.NewContainerLayer()
	a.Append(b)
	a.Append(c)
	b.Append(d)
	b.Append(e)
	b.Append(f)
	c.Append(g)
	all := []*rendering.ContainerLayer{a, b, c, d, e, f, g}
	for _, l := range all {
		if !l.DebugSubtreeNeedsAddToScene() {
			t.Fatal("a new layer must need add to scene")
		}
	}
	for _, l := range all {
		l.DebugMarkClean()
	}
	b.MarkNeedsAddToScene()
	g.MarkNeedsAddToScene()
	a.UpdateSubtreeNeedsAddToScene()
	want := []bool{true, true, true, false, false, false, true}
	for i, l := range all {
		if got := l.DebugSubtreeNeedsAddToScene(); got != want[i] {
			t.Errorf("layer %d: needs add to scene = %v, want %v", i, got, want[i])
		}
	}
}

func TestLayersFollowerAlwaysDirty(t *testing.T) {
	skipUnlessDebug(t)
	link := rendering.NewLayerLink()
	leader := rendering.NewLeaderLayer(link)
	follower := rendering.NewFollowerLayer(link)
	leader.DebugMarkClean()
	follower.DebugMarkClean()
	leader.UpdateSubtreeNeedsAddToScene()
	follower.UpdateSubtreeNeedsAddToScene()
	if !follower.DebugSubtreeNeedsAddToScene() {
		t.Error("a follower layer must always need add to scene")
	}
	if leader.DebugSubtreeNeedsAddToScene() {
		t.Error("a leader layer is not dirty because of its follower")
	}
}

func TestLayersLeaderLink(t *testing.T) {
	t.Run("switching the link of an attached leader", func(t *testing.T) {
		link, link2 := rendering.NewLayerLink(), rendering.NewLayerLink()
		leader := rendering.NewLeaderLayer(link)
		leader.Attach(new(int))
		if link.Leader() != leader {
			t.Fatal("the attached leader is not registered")
		}
		leader.SetLink(link2)
		leader.Detach()
		if leader.Link() != link2 || link.Leader() != nil || link2.Leader() != nil {
			t.Errorf("link = %v, leader of old %v, leader of new %v", leader.Link(), link.Leader(), link2.Leader())
		}
	})
	t.Run("attach and detach order", func(t *testing.T) {
		link := rendering.NewLayerLink()
		l1, l2 := rendering.NewLeaderLayer(link), rendering.NewLeaderLayer(link)
		l1.Attach(new(int))
		l2.Attach(new(int))
		l2.Detach()
		l1.Detach()
		if link.Leader() != nil {
			t.Error("the link still has a leader")
		}
	})
	t.Run("leader is not dirty with followers", func(t *testing.T) {
		root := rendering.NewContainerLayer()
		root.Attach(new(int))
		link := rendering.NewLayerLink()
		leader := rendering.NewLeaderLayer(link)
		root.Append(leader)
		followers := []*rendering.FollowerLayer{}
		for i := 0; i < 2; i++ {
			leader.DebugMarkClean()
			leader.UpdateSubtreeNeedsAddToScene()
			if leader.DebugSubtreeNeedsAddToScene() {
				t.Fatalf("with %d followers the leader is dirty", i)
			}
			f := rendering.NewFollowerLayer(link)
			followers = append(followers, f)
			root.Append(f)
		}
		for _, f := range followers {
			f.Remove()
			leader.DebugMarkClean()
			leader.UpdateSubtreeNeedsAddToScene()
			if leader.DebugSubtreeNeedsAddToScene() {
				t.Error("the leader is dirty after a follower was removed")
			}
		}
	})
}

func TestLayersDepthFirstIterateChildren(t *testing.T) {
	a, b, c, d, e, f, g := rendering.NewContainerLayer(), rendering.NewContainerLayer(), rendering.NewContainerLayer(),
		rendering.NewContainerLayer(), rendering.NewContainerLayer(), rendering.NewContainerLayer(), rendering.NewContainerLayer()
	h, i, j := rendering.NewPictureLayer(ui.Rect{}), rendering.NewPictureLayer(ui.Rect{}), rendering.NewPictureLayer(ui.Rect{})
	a.Append(b)
	a.Append(c)
	b.Append(d)
	b.Append(e)
	b.Append(f)
	d.Append(h)
	d.Append(i)
	c.Append(g)
	g.Append(j)
	check := func(want ...rendering.Layer) {
		t.Helper()
		got := a.DepthFirstIterateChildren()
		if len(got) != len(want) {
			t.Fatalf("got %d layers, want %d", len(got), len(want))
		}
		for k := range want {
			if got[k] != want[k] {
				t.Errorf("layer %d differs", k)
			}
		}
	}
	check(b, d, h, i, e, f, c, g, j)
	d.Remove()
	check(b, e, f, c, g, j)
}

func TestLayersDebugInfo(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("clip behavior", func(t *testing.T) {
		tests := []struct {
			name  string
			layer rendering.Layer
			want  string
		}{
			{"rect default", rendering.NewClipRectLayer(), "clipBehavior: Clip.hardEdge"},
			{"rect save layer", rendering.NewClipRectLayer(rendering.ClipRectLayerOpts{ClipBehavior: ui.ClipAntiAliasWithSaveLayer}), "clipBehavior: Clip.antiAliasWithSaveLayer"},
			{"rrect default", rendering.NewClipRRectLayer(), "clipBehavior: Clip.antiAlias"},
			{"rsuperellipse default", rendering.NewClipRSuperellipseLayer(), "clipBehavior: Clip.antiAlias"},
			{"path default", rendering.NewClipPathLayer(), "clipBehavior: Clip.antiAlias"},
			{"path save layer", rendering.NewClipPathLayer(rendering.ClipPathLayerOpts{ClipBehavior: ui.ClipAntiAliasWithSaveLayer}), "clipBehavior: Clip.antiAliasWithSaveLayer"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if info := debugInfo(tc.layer); !contains(info, tc.want) {
					t.Errorf("%q not in %q", tc.want, info)
				}
			})
		}
	})
	t.Run("backdrop filter", func(t *testing.T) {
		filter := ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 1, SigmaY: 1, TileMode: ui.TileModeRepeated})
		layer := rendering.NewBackdropFilterLayer(rendering.BackdropFilterLayerOpts{Filter: filter, BlendMode: ui.BlendModeClear})
		info := debugInfo(layer)
		if !contains(info, "filter: ImageFilter.blur(1.0, 1.0, repeated)") || !contains(info, "blendMode: clear") {
			t.Errorf("info = %q", info)
		}
	})
	t.Run("picture layer", func(t *testing.T) {
		rec := ui.NewPictureRecorder()
		ui.NewCanvas(rec).DrawPaint(ui.NewPaint())
		pic := rec.EndRecording()
		layer := rendering.NewPictureLayer(ui.NewRectFromLTRB(0, 0, 1, 1))
		layer.SetPicture(pic)
		layer.SetIsComplexHint(true)
		info := debugInfo(layer)
		if !contains(info, "picture: "+foundation.DescribeIdentity(pic)) ||
			!contains(info, "raster cache hints: isComplex = true, willChange = false") {
			t.Errorf("info = %q", info)
		}
	})
	t.Run("engine layer", func(t *testing.T) {
		layer := newConcreteLayer()
		for _, s := range debugInfo(layer) {
			if strings.HasPrefix(s, "engine layer") {
				t.Fatalf("a layer without engine layer prints %q", s)
			}
		}
		e := ui.NewSceneBuilder().PushOffset(0, 0)
		layer.SetEngineLayer(e)
		if info := debugInfo(layer); !contains(info, "engine layer: "+foundation.DescribeIdentity(e)) {
			t.Errorf("info = %q", info)
		}
	})
}

func TestLayersMutationNeedsAddToScene(t *testing.T) {
	skipUnlessDebug(t)
	unit := ui.NewRectFromLTRB(0, 0, 1, 1)
	gradient := ui.NewGradientRadial(ui.Offset{}, 1, []ui.Color{ui.NewColorFromARGB(0, 0, 0, 0), ui.NewColorFromARGB(0, 0, 0, 1)})
	gradient2 := ui.NewGradientRadial(ui.Offset{Dx: 1}, 1, []ui.Color{ui.NewColorFromARGB(0, 0, 0, 0), ui.NewColorFromARGB(0, 0, 0, 1)})
	unitPath := ui.NewPath()
	unitPath.AddRect(unit)
	tests := []struct {
		name   string
		layer  rendering.Layer
		mutate []func()
	}{}
	pic := rendering.NewPictureLayer(ui.Rect{})
	off := rendering.NewOffsetLayer()
	clipRect := rendering.NewClipRectLayer(rendering.ClipRectLayerOpts{ClipRect: foundation.Some(ui.Rect{})})
	clipRRect := rendering.NewClipRRectLayer(rendering.ClipRRectLayerOpts{ClipRRect: foundation.Some(ui.RRect{})})
	clipPath := rendering.NewClipPathLayer(rendering.ClipPathLayerOpts{ClipPath: ui.NewPath()})
	opacity := rendering.NewOpacityLayer(rendering.OpacityLayerOpts{Alpha: foundation.Some(0)})
	cf := rendering.NewColorFilterLayer(ui.NewColorFilterMode(ui.NewColorFromARGB(255, 255, 0, 0), ui.BlendModeColor))
	mask := rendering.NewShaderMaskLayer(rendering.ShaderMaskLayerOpts{Shader: gradient, MaskRect: foundation.Some(ui.Rect{}), BlendMode: foundation.Some(ui.BlendModeClear)})
	backdrop := rendering.NewBackdropFilterLayer(rendering.BackdropFilterLayerOpts{Filter: ui.NewImageFilterBlur()})
	imageFilter := rendering.NewImageFilterLayer(rendering.ImageFilterLayerOpts{ImageFilter: ui.NewImageFilterBlur()})
	tests = append(tests, []struct {
		name   string
		layer  rendering.Layer
		mutate []func()
	}{
		{"picture", pic, []func(){
			func() {
				rec := ui.NewPictureRecorder()
				ui.NewCanvas(rec)
				pic.SetPicture(rec.EndRecording())
			},
			func() { pic.SetIsComplexHint(true) },
			func() { pic.SetWillChangeHint(true) },
		}},
		{"offset", off, []func(){func() { off.SetOffset(ui.Offset{Dx: 1, Dy: 1}) }}},
		{"clipRect", clipRect, []func(){
			func() { clipRect.SetClipRect(unit) },
			func() { clipRect.SetClipBehavior(ui.ClipAntiAliasWithSaveLayer) },
		}},
		{"clipRRect", clipRRect, []func(){
			func() { clipRRect.SetClipRRect(ui.NewRRectFromRectAndRadius(unit, ui.RadiusZero)) },
			func() { clipRRect.SetClipBehavior(ui.ClipAntiAliasWithSaveLayer) },
		}},
		{"clipPath", clipPath, []func(){
			func() { clipPath.SetClipPath(unitPath) },
			func() { clipPath.SetClipBehavior(ui.ClipAntiAliasWithSaveLayer) },
		}},
		{"opacity", opacity, []func(){
			func() { opacity.SetAlpha(1) },
			func() { opacity.SetOffset(ui.Offset{Dx: 1, Dy: 1}) },
		}},
		{"colorFilter", cf, []func(){func() {
			cf.SetColorFilter(ui.NewColorFilterMode(ui.NewColorFromARGB(255, 0, 255, 0), ui.BlendModeColor))
		}}},
		{"shaderMask", mask, []func(){
			func() { mask.SetMaskRect(unit) },
			func() { mask.SetBlendMode(ui.BlendModeColor) },
			func() { mask.SetShader(gradient2) },
		}},
		{"backdropFilter", backdrop, []func(){func() { backdrop.SetFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 1})) }}},
		{"imageFilter", imageFilter, []func(){func() { imageFilter.SetImageFilter(ui.NewImageFilterBlur(ui.ImageFilterBlurOpts{SigmaX: 2})) }}},
	}...)
	for _, tc := range tests {
		for i, mutate := range tc.mutate {
			t.Run(tc.name, func(t *testing.T) {
				tc.layer.AsLayer().DebugMarkClean()
				tc.layer.UpdateSubtreeNeedsAddToScene()
				if tc.layer.AsLayer().DebugSubtreeNeedsAddToScene() {
					t.Fatalf("mutation %d: the layer is dirty before the mutation", i)
				}
				mutate()
				tc.layer.UpdateSubtreeNeedsAddToScene()
				if !tc.layer.AsLayer().DebugSubtreeNeedsAddToScene() {
					t.Errorf("mutation %d did not mark the layer as needing add to scene", i)
				}
			})
		}
	}
}

func TestLayersToImageInteriorLayer(t *testing.T) {
	parent, child, grandChild := rendering.NewOffsetLayer(), rendering.NewOffsetLayer(), rendering.NewOffsetLayer()
	child.Append(grandChild)
	parent.Append(child)
	parent.BuildScene(ui.NewSceneBuilder())
	if _, err := grandChild.ToImage(ui.NewRectFromLTRB(0, 0, 10, 10), 1, ui.ToImageOpts{Rasterizer: soft.ImageRasterizer{}}); err != nil {
		t.Fatalf("ToImage: %v", err)
	}
	if _, err := grandChild.ToImageSync(ui.NewRectFromLTRB(0, 0, 10, 10), 1, ui.ToImageOpts{Rasterizer: soft.ImageRasterizer{}}); err != nil {
		t.Fatalf("ToImageSync: %v", err)
	}
	parent.BuildScene(ui.NewSceneBuilder())
}

func TestLayersHandleCounts(t *testing.T) {
	skipUnlessDebug(t)
	t.Run("append and remove", func(t *testing.T) {
		layer := rendering.NewPictureLayer(ui.Rect{})
		parent := rendering.NewContainerLayer()
		if layer.DebugHandleCount() != 0 || layer.DebugDisposed() {
			t.Fatal("a new layer has handles or is disposed")
		}
		parent.Append(layer)
		if layer.DebugHandleCount() != 1 || layer.DebugDisposed() {
			t.Errorf("after append: handles = %d, disposed = %v", layer.DebugHandleCount(), layer.DebugDisposed())
		}
		layer.Remove()
		if layer.DebugHandleCount() != 0 || !layer.DebugDisposed() {
			t.Errorf("after remove: handles = %d, disposed = %v", layer.DebugHandleCount(), layer.DebugDisposed())
		}
	})
	t.Run("handle disposes the layer", func(t *testing.T) {
		l1, l2 := newConcreteLayer(), newConcreteLayer()
		holder := rendering.NewLayerHandle[rendering.Layer](l1)
		state := func(want ...int) {
			t.Helper()
			got := []int{l1.DebugHandleCount(), l2.DebugHandleCount()}
			if got[0] != want[0] || got[1] != want[1] || l1.DebugDisposed() != (want[2] == 1) || l2.DebugDisposed() != (want[3] == 1) {
				t.Errorf("handles %v, disposed %v %v; want %v", got, l1.DebugDisposed(), l2.DebugDisposed(), want)
			}
		}
		state(1, 0, 0, 0)
		holder.SetLayer(l1)
		state(1, 0, 0, 0)
		holder.SetLayer(l2)
		state(0, 1, 1, 0)
		holder.SetLayer(nil)
		state(0, 0, 1, 1)
	})
	t.Run("engine layer and picture are disposed", func(t *testing.T) {
		layer := newConcreteLayer()
		e := ui.NewSceneBuilder().PushOffset(0, 0)
		layer.SetEngineLayer(e)
		layer.Dispose()
		if !engineDisposed(e) || layer.EngineLayer() != nil {
			t.Error("Dispose did not dispose the engine layer")
		}
		layer2 := newConcreteLayer()
		e2 := ui.NewSceneBuilder().PushOffset(0, 0)
		layer2.SetEngineLayer(e2)
		layer2.SetEngineLayer(nil)
		if !engineDisposed(e2) {
			t.Error("setting the engine layer did not dispose the old one")
		}
	})
}

func TestLayersOpacityLayerNoChildren(t *testing.T) {
	layer := rendering.NewOpacityLayer(rendering.OpacityLayerOpts{Alpha: foundation.Some(128)})
	builder := ui.NewSceneBuilder()
	layer.AddToScene(builder)
	if layer.EngineLayer() != nil {
		t.Fatal("an opacity layer without children has an engine layer")
	}
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawPaint(ui.NewPaint())
	pl := rendering.NewPictureLayer(ui.Rect{})
	pl.SetPicture(rec.EndRecording())
	layer.Append(pl)
	layer.AddToScene(builder)
	if _, ok := layer.EngineLayer().(*ui.OpacityEngineLayer); !ok {
		t.Fatalf("engine layer = %T, want an opacity engine layer", layer.EngineLayer())
	}
	layer.SetAlpha(200)
	if _, ok := layer.EngineLayer().(*ui.OpacityEngineLayer); !ok {
		t.Errorf("engine layer = %T after 128 -> 200", layer.EngineLayer())
	}
	layer.SetAlpha(255)
	if layer.EngineLayer() != nil {
		t.Error("engine layer must be dropped when alpha becomes 255")
	}
	layer.AddToScene(builder)
	if _, ok := layer.EngineLayer().(*ui.OffsetEngineLayer); !ok {
		t.Errorf("engine layer = %T, want an offset engine layer for alpha 255", layer.EngineLayer())
	}
	layer.SetAlpha(200)
	if layer.EngineLayer() != nil {
		t.Error("engine layer must be dropped when alpha stops being 255")
	}
	layer.AddToScene(builder)
	if _, ok := layer.EngineLayer().(*ui.OpacityEngineLayer); !ok {
		t.Errorf("engine layer = %T, want an opacity engine layer", layer.EngineLayer())
	}
	layer.RemoveAllChildren()
	layer.AddToScene(builder)
	if layer.EngineLayer() != nil {
		t.Error("engine layer must be disposed when there are no children")
	}
}

func TestLayersDescribeClipBounds(t *testing.T) {
	bounds := ui.NewRectFromLTRB(10, 10, 20, 20)
	path := ui.NewPath()
	path.AddRect(bounds)
	tests := []struct {
		name  string
		layer rendering.Layer
		want  foundation.Opt[ui.Rect]
	}{
		{"container", rendering.NewContainerLayer(), foundation.None[ui.Rect]()},
		{"rect", rendering.NewClipRectLayer(rendering.ClipRectLayerOpts{ClipRect: foundation.Some(bounds)}), foundation.Some(bounds)},
		{"rrect", rendering.NewClipRRectLayer(rendering.ClipRRectLayerOpts{ClipRRect: foundation.Some(ui.NewRRectFromRectXY(bounds, 2, 2))}), foundation.Some(bounds)},
		{"rsuperellipse", rendering.NewClipRSuperellipseLayer(rendering.ClipRSuperellipseLayerOpts{ClipRSuperellipse: foundation.Some(ui.NewRSuperellipseFromRectXY(bounds, 2, 2))}), foundation.Some(bounds)},
		{"path", rendering.NewClipPathLayer(rendering.ClipPathLayerOpts{ClipPath: path}), foundation.Some(bounds)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.layer.DescribeClipBounds(); got != tc.want {
				t.Errorf("DescribeClipBounds() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLayersCompositionCallbacks(t *testing.T) {
	tree := func() (root, a1, a2, b1 *rendering.ContainerLayer) {
		root, a1, a2, b1 = rendering.NewContainerLayer(), rendering.NewContainerLayer(), rendering.NewContainerLayer(), rendering.NewContainerLayer()
		root.Append(a1)
		root.Append(a2)
		a1.Append(b1)
		return
	}
	t.Run("subtree has callbacks", func(t *testing.T) {
		root := rendering.NewContainerLayer()
		var cancels []func()
		cancels = append(cancels, root.AddCompositionCallback(func(rendering.Layer) {}))
		a1, a2, b1 := rendering.NewContainerLayer(), rendering.NewContainerLayer(), rendering.NewContainerLayer()
		root.Append(a1)
		root.Append(a2)
		a1.Append(b1)
		cancels = append(cancels, b1.AddCompositionCallback(func(rendering.Layer) {}))
		got := func() [4]bool {
			return [4]bool{root.SubtreeHasCompositionCallbacks(), a1.SubtreeHasCompositionCallbacks(), a2.SubtreeHasCompositionCallbacks(), b1.SubtreeHasCompositionCallbacks()}
		}
		if g := got(); g != [4]bool{true, true, false, true} {
			t.Errorf("after both: %v", g)
		}
		cancels[0]()
		if g := got(); g != [4]bool{true, true, false, true} {
			t.Errorf("after first cancel: %v", g)
		}
		cancels[1]()
		if g := got(); g != [4]bool{false, false, false, false} {
			t.Errorf("after second cancel: %v", g)
		}
	})
	t.Run("removeChild", func(t *testing.T) {
		root, a1, a2, b1 := tree()
		b1.AddCompositionCallback(func(rendering.Layer) {})
		if !b1.SubtreeHasCompositionCallbacks() || !a1.SubtreeHasCompositionCallbacks() || !root.SubtreeHasCompositionCallbacks() || a2.SubtreeHasCompositionCallbacks() {
			t.Fatal("the counts did not reach the ancestors")
		}
		b1.Remove()
		if !b1.SubtreeHasCompositionCallbacks() || a1.SubtreeHasCompositionCallbacks() || root.SubtreeHasCompositionCallbacks() {
			t.Error("the counts were not taken from the ancestors")
		}
	})
	t.Run("no callback if removed", func(t *testing.T) {
		root, _, _, b1 := tree()
		b1.AddCompositionCallback(func(rendering.Layer) { t.Error("called back after removal") })()
		root.BuildScene(ui.NewSceneBuilder()).Dispose()
	})
	t.Run("not retained", func(t *testing.T) {
		root, _, _, b1 := tree()
		called := false
		b1.AddCompositionCallback(func(l rendering.Layer) {
			if l != rendering.Layer(b1) {
				t.Errorf("layer = %v, want b1", l)
			}
			called = true
		})
		root.BuildScene(ui.NewSceneBuilder()).Dispose()
		if !called {
			t.Error("not called")
		}
	})
	t.Run("retained", func(t *testing.T) {
		root, _, _, b1 := tree()
		builder := ui.NewSceneBuilder()
		b1.SetEngineLayer(builder.PushOffset(0, 0))
		builder.Build().Dispose()
		b1.DebugMarkClean()
		if b1.DebugSubtreeNeedsAddToScene() {
			t.Fatal("b1 is dirty")
		}
		called := false
		b1.AddCompositionCallback(func(rendering.Layer) { called = true })
		root.BuildScene(ui.NewSceneBuilder()).Dispose()
		if !called {
			t.Error("not called for a retained layer")
		}
	})
	t.Run("detach triggers callback", func(t *testing.T) {
		root, _, _, b1 := tree()
		called := false
		b1.AddCompositionCallback(func(rendering.Layer) { called = true })
		root.Attach(new(int))
		if called {
			t.Fatal("called on attach")
		}
		root.Detach()
		if !called {
			t.Error("not called on detach")
		}
	})
	t.Run("observer count", func(t *testing.T) {
		root, a1 := rendering.NewContainerLayer(), rendering.NewContainerLayer()
		root.Append(a1)
		r1 := a1.AddCompositionCallback(func(rendering.Layer) {})
		r2 := a1.AddCompositionCallback(func(rendering.Layer) {})
		r1()
		if !root.SubtreeHasCompositionCallbacks() || !a1.SubtreeHasCompositionCallbacks() {
			t.Error("one callback is left")
		}
		r2()
		if root.SubtreeHasCompositionCallbacks() || a1.SubtreeHasCompositionCallbacks() {
			t.Error("no callback is left")
		}
	})
	t.Run("remover on a disposed layer", func(t *testing.T) {
		root := rendering.NewContainerLayer()
		cb := root.AddCompositionCallback(func(rendering.Layer) {})
		root.Dispose()
		if v := panicOf(cb); v != nil {
			t.Errorf("removing on a disposed layer panicked: %v", v)
		}
	})
}

func TestLayersSupportRasterization(t *testing.T) {
	for name, l := range map[string]rendering.Layer{
		"offset": rendering.NewOffsetLayer(), "opacity": rendering.NewOpacityLayer(), "clipRect": rendering.NewClipRectLayer(),
		"clipRRect": rendering.NewClipRRectLayer(), "clipRSuperellipse": rendering.NewClipRSuperellipseLayer(),
		"imageFilter": rendering.NewImageFilterLayer(), "backdropFilter": rendering.NewBackdropFilterLayer(),
		"colorFilter": rendering.NewColorFilterLayer(nil), "shaderMask": rendering.NewShaderMaskLayer(),
	} {
		if !l.SupportsRasterization() {
			t.Errorf("%s does not support rasterization", name)
		}
	}
}

func TestLayersBuildScene(t *testing.T) {
	root := rendering.NewOffsetLayer(rendering.OffsetLayerOpts{Offset: ui.Offset{Dx: 3, Dy: 4}})
	clip := rendering.NewClipRectLayer(rendering.ClipRectLayerOpts{ClipRect: foundation.Some(ui.NewRectFromLTWH(0, 0, 5, 5))})
	rec := ui.NewPictureRecorder()
	p := ui.NewPaint()
	p.SetColor(ui.NewColorFromARGB(255, 255, 0, 0))
	ui.NewCanvas(rec).DrawRect(ui.NewRectFromLTWH(0, 0, 10, 10), p)
	pl := rendering.NewPictureLayer(ui.NewRectFromLTWH(0, 0, 10, 10))
	pl.SetPicture(rec.EndRecording())
	clip.Append(pl)
	root.Append(clip)
	scene := root.BuildScene(ui.NewSceneBuilder())
	if scene == nil {
		t.Fatal("no scene")
	}
	if root.DebugSubtreeNeedsAddToScene() {
		t.Error("the root is still dirty after building the scene")
	}
	if _, ok := root.EngineLayer().(*ui.OffsetEngineLayer); !ok {
		t.Errorf("root engine layer = %T", root.EngineLayer())
	}
	if _, ok := clip.EngineLayer().(*ui.ClipRectEngineLayer); !ok {
		t.Errorf("clip engine layer = %T", clip.EngineLayer())
	}
	// A second build retains the unchanged engine layers.
	first := clip.EngineLayer()
	root.BuildScene(ui.NewSceneBuilder())
	if clip.EngineLayer() != first {
		t.Error("an unchanged layer was not retained")
	}
}

func TestLayersTransformLayerBuildsEffectiveTransform(t *testing.T) {
	m := ui.NewMatrix4Diagonal3Values(2, 2, 1)
	l := rendering.NewTransformLayer(rendering.TransformLayerOpts{Transform: m, Offset: ui.Offset{Dx: 5, Dy: 6}})
	l.BuildScene(ui.NewSceneBuilder())
	applied := ui.NewMatrix4Identity()
	l.ApplyTransform(nil, applied)
	want := ui.NewMatrix4TranslationValues(5, 6, 0).Multiply(m)
	if !applied.Equals(want) {
		t.Errorf("applied transform = %v, want %v", applied, want)
	}
	if v := math.Abs(float64(applied.Entry(0, 3)) - 5); v > 1e-6 {
		t.Errorf("translation x = %v", applied.Entry(0, 3))
	}
}
