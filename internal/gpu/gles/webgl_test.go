package gles

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// The WebGL 2 binding is a js file, so these tests cover what is plain Go around it: the
// handle table, the context attributes, the loss state, the errors, and how the renderer
// behaves when the context is lost and restored, with a fake context that can be.

func TestHandleTable(t *testing.T) {
	var tab handleTable[string]
	a, b, c := tab.add("a"), tab.add("b"), tab.add("c")
	if a != 1 || b != 2 || c != 3 {
		t.Fatalf("names = %d %d %d, want 1 2 3: 0 is null", a, b, c)
	}
	tests := []struct {
		name   string
		id     uint32
		want   string
		wantOK bool
	}{
		{"first", 1, "a", true},
		{"third", 3, "c", true},
		{"null", 0, "", false},
		{"never given", 4, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tab.get(tt.id)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("get(%d) = %q, %v; want %q, %v", tt.id, got, ok, tt.want, tt.wantOK)
			}
		})
	}
	if v, ok := tab.remove(2); !ok || v != "b" {
		t.Errorf("remove(2) = %q, %v", v, ok)
	}
	if _, ok := tab.get(2); ok {
		t.Error("a removed name is still in use")
	}
	if _, ok := tab.remove(2); ok {
		t.Error("a name was removed twice")
	}
	if _, ok := tab.remove(0); ok {
		t.Error("null was removed")
	}
	if tab.count() != 2 {
		t.Errorf("count = %d, want 2", tab.count())
	}
	if d := tab.add("d"); d != 2 {
		t.Errorf("the name of a removed object is reused: got %d, want 2", d)
	}
	if got, _ := tab.get(2); got != "d" {
		t.Errorf("the reused name holds %q, want d", got)
	}
}

func TestWebContextAttributes(t *testing.T) {
	got := webContextAttributes()
	want := map[string]any{
		"alpha": true, "premultipliedAlpha": true, "antialias": false, "depth": false, "stencil": false,
		"preserveDrawingBuffer": false, "failIfMajorPerformanceCaveat": false, "powerPreference": "default",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("attributes = %v, want %v (SPEC §9.1)", got, want)
	}
	// Each call gives a map of its own: the browser's call must not see another's changes.
	got["alpha"] = false
	if webContextAttributes()["alpha"] != true {
		t.Error("the attributes are shared between calls")
	}
}

func TestLossState(t *testing.T) {
	type step struct {
		do                 func(*lossState)
		lost, needsRebuild bool
	}
	lost := func(s *lossState) { s.onLost() }
	restored := func(s *lossState) { s.onRestored() }
	rebuilt := func(s *lossState) { s.rebuilt() }
	tests := []struct {
		name  string
		steps []step
	}{
		{"fresh", []step{{func(*lossState) {}, false, false}}},
		{"lost, then restored, then rebuilt", []step{
			{lost, true, false}, // nothing to rebuild with until the browser gives the context back
			{restored, false, true},
			{rebuilt, false, false},
		}},
		{"lost twice before it is restored", []step{
			{lost, true, false}, {lost, true, false}, {restored, false, true}, {rebuilt, false, false},
		}},
		{"lost again before the rebuild", []step{
			{lost, true, false}, {restored, false, true}, {lost, true, false}, {restored, false, true}, {rebuilt, false, false},
		}},
		{"a restoration with no loss changes nothing", []step{{restored, false, false}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s lossState
			for i, st := range tt.steps {
				st.do(&s)
				if s.contextLost() != st.lost || s.needsRebuild() != st.needsRebuild {
					t.Fatalf("step %d: lost %v, needs rebuild %v; want %v, %v", i, s.contextLost(), s.needsRebuild(), st.lost, st.needsRebuild)
				}
			}
		})
	}
}

func TestIsES3AcceptsWebGL2(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"WebGL 2.0 (OpenGL ES 3.0 Chromium)", true},
		{"WebGL 2.0", true},
		{"WebGL 1.0 (OpenGL ES 2.0 Chromium)", false},
		{"WebGL 1.0", false},
		{"WebGL", false},
	}
	for _, tt := range tests {
		if got := isES3(tt.version); got != tt.want {
			t.Errorf("isES3(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
}

// lossyEGL is a fake EGL whose context the test can lose and restore, and whose surface has a
// size of its own, as the web's canvas does. The GL it loads is a new fake each time, so that
// the test sees what is made again.
type lossyEGL struct {
	*fakeEGL
	lossState
	loads []*fakeGL
	sizes [][2]int
}

func (e *lossyEGL) setSurfaceSize(_ uintptr, w, h int) { e.sizes = append(e.sizes, [2]int{w, h}) }

func (e *lossyEGL) load(eglAPI) (glAPI, error) {
	gl := &fakeGL{versionString: "WebGL 2.0 (OpenGL ES 3.0 Chromium)"}
	e.loads = append(e.loads, gl)
	return gl, nil
}

func newLossyRenderer(opts ...RendererOpts) (*Renderer, *lossyEGL) {
	e := &lossyEGL{fakeEGL: newFakeEGL()}
	r := newRenderer(func(h gpu.SurfaceHandle) (*Context, *Surface, error) {
		return openWith(e, e.load, h)
	}, opts...)
	return r, e
}

// a scene with a path, a clip, an image and vertices: everything that the caches hold.
func cachedThingsScene() *ui.Scene {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(5, 5, 50, 50), 10, 10))
	path := ui.NewPath()
	path.MoveTo(0, 0)
	path.LineTo(40, 5)
	path.LineTo(10, 40)
	path.Close()
	c.DrawPath(path, ui.NewPaint())
	c.DrawImage(ui.NewImageFromRGBA(make([]byte, 4*8*8), 8, 8), ui.Offset{}, nil)
	sb := ui.NewSceneBuilder()
	sb.AddPicture(ui.Offset{}, rec.EndRecording())
	return sb.Build()
}

func TestOpenWithAWebHandleUsesTheWebPlatform(t *testing.T) {
	e := &lossyEGL{fakeEGL: newFakeEGL()}
	c, s, err := openWith(e, e.load, gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: 7})
	if err != nil {
		t.Fatal(err)
	}
	if e.platform != eglPlatformWeb || e.nativeWindow != 7 {
		t.Errorf("platform %#x window %d, want %#x 7", e.platform, e.nativeWindow, eglPlatformWeb)
	}
	if c.Version() != "WebGL 2.0 (OpenGL ES 3.0 Chromium)" {
		t.Errorf("Version() = %q", c.Version())
	}
	s.Destroy()
	c.Close()
}

func TestRendererResizesASurfaceThatHasItsOwnSize(t *testing.T) {
	r, e := newLossyRenderer()
	s, err := r.Attach(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb})
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []ui.Size{{Width: 320, Height: 200}, {Width: 100, Height: 50}} {
		if err := r.Render(s, nil, size, 1); err != nil {
			t.Fatal(err)
		}
	}
	if want := [][2]int{{320, 200}, {100, 50}}; !reflect.DeepEqual(e.sizes, want) {
		t.Errorf("surface sizes = %v, want %v, each set before its frame", e.sizes, want)
	}
}

func TestRendererWhileTheContextIsLost(t *testing.T) {
	r, e := newLossyRenderer()
	h := gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb}
	s, err := r.Attach(h)
	if err != nil {
		t.Fatal(err)
	}
	size := ui.Size{Width: 64, Height: 64}
	if err := r.Render(s, cachedThingsScene(), size, 1); err != nil {
		t.Fatal(err)
	}
	first := e.loads[0]
	e.onLost()
	first.resetCalls()

	// Nothing is drawn and nothing is reported: there is no content to show.
	if err := r.Render(s, cachedThingsScene(), size, 1); err != nil {
		t.Errorf("Render while lost: %v", err)
	}
	if first.clears != 1 || len(first.calls) != 0 {
		t.Errorf("GL calls while the context is lost: %v", first.calls)
	}
	// A read-back and a snapshot have nothing to read.
	_, err = r.RenderCapture(s, cachedThingsScene(), size, 1)
	assertTag(t, err, "WebGLContextLost", "RenderCapture")
	_, err = r.Snapshot(cachedThingsScene(), size)
	assertTag(t, err, "WebGLContextLost", "Snapshot")
	if len(e.loads) != 1 {
		t.Errorf("the GL was bound again before the browser restored the context: %d loads", len(e.loads))
	}
}

func TestRendererRebuildsAfterTheContextIsRestored(t *testing.T) {
	r, e := newLossyRenderer(RendererOpts{RasterCacheAfter: 3})
	h := gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb}
	s, err := r.Attach(h)
	if err != nil {
		t.Fatal(err)
	}
	size := ui.Size{Width: 64, Height: 64}
	if err := r.Render(s, cachedThingsScene(), size, 1); err != nil {
		t.Fatal(err)
	}
	oldDev, oldRes, oldCache := r.ctx.dev, r.res, r.cache
	if oldRes == nil || oldCache == nil {
		t.Fatal("the first frame made no caches")
	}
	old := e.loads[0]
	liveBefore := old.liveCount("texture") + old.liveCount("buffer")
	if liveBefore == 0 {
		t.Fatal("the first frame made no GL objects")
	}
	e.onLost()
	e.onRestored()
	if err := r.Render(s, cachedThingsScene(), size, 1); err != nil {
		t.Fatalf("Render after the restoration: %v", err)
	}
	if len(e.loads) != 2 {
		t.Fatalf("the GL was bound %d times, want 2 (the context's, and again after the restoration)", len(e.loads))
	}
	if r.ctx.dev == oldDev || r.res == oldRes || r.cache == oldCache {
		t.Error("the device, the resource caches and the raster cache were kept")
	}
	if got := old.liveCount("texture") + old.liveCount("buffer"); got != liveBefore {
		t.Errorf("the objects of the lost context were touched: %d live, was %d", got, liveBefore)
	}
	fresh := e.loads[1]
	if fresh.liveCount("texture") == 0 || fresh.liveCount("buffer") == 0 {
		t.Errorf("the frame after the restoration made no objects in the new context: %d textures, %d buffers",
			fresh.liveCount("texture"), fresh.liveCount("buffer"))
	}
	if st := r.LastFrame(); st.TilesRasterized == 0 || st.ClipMasksBuilt != 1 || st.ImagesUploaded != 1 {
		t.Errorf("what the caches held was not made again: %+v", st)
	}
	// A frame after that is an ordinary one: nothing is made again.
	if err := r.Render(s, cachedThingsScene(), size, 1); err != nil {
		t.Fatal(err)
	}
	if len(e.loads) != 2 {
		t.Errorf("the GL was bound again: %d loads", len(e.loads))
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"texture", "buffer", "vao", "program", "shader", "framebuffer", "renderbuffer"} {
		if n := fresh.liveCount(kind); n != 0 {
			t.Errorf("%d %s objects of the new context leaked after Close", n, kind)
		}
	}
}

func TestContextRebuildNeedsALossyContext(t *testing.T) {
	c, _, _, _ := openFake(t) // an EGL context: it is never lost
	if c.Lost() {
		t.Error("an EGL context is lost")
	}
	if rebuilt, err := c.rebuild(); rebuilt || err != nil {
		t.Errorf("rebuild = %v, %v; want false, nil", rebuilt, err)
	}
	// A surface that follows its window is not resized.
	c.ResizeSurface(&Surface{}, 10, 10)
}

func assertTag(t *testing.T, err error, tag, fn string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag || !appErr.HasRoute(fn+"."+tag) && !appErr.HasRoute(tag) {
		t.Errorf("got %s via %s, want %s via %s", appErr.Tag, appErr.Route(), tag, fn)
	}
}

// TestBindingsHaveTheSameFunctions: the purego binding (gl_purego.go) and the WebGL 2 binding
// (gl_js.go) implement the one interface of glapi.go with the same set of functions, so that
// the renderer cannot find one binding lacking what the other has. Each file carries a build
// tag, so the test reads them as source.
func TestBindingsHaveTheSameFunctions(t *testing.T) {
	fset := token.NewFileSet()
	parse := func(file string) *ast.File {
		t.Helper()
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		return f
	}
	// The interface's methods.
	var api []string
	for _, d := range parse("glapi.go").Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, sp := range gd.Specs {
			ts, ok := sp.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "glAPI" {
				continue
			}
			for _, m := range ts.Type.(*ast.InterfaceType).Methods.List {
				for _, n := range m.Names {
					api = append(api, n.Name)
				}
			}
		}
	}
	sort.Strings(api)
	if len(api) < 80 {
		t.Fatalf("read %d methods of glAPI, expected the whole interface", len(api))
	}
	// The methods each binding has.
	methodsOf := func(file, recv string) []string {
		var names []string
		for _, d := range parse(file).Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 {
				continue
			}
			star, ok := fd.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			if id, ok := star.X.(*ast.Ident); ok && id.Name == recv {
				names = append(names, fd.Name.Name)
			}
		}
		sort.Strings(names)
		return names
	}
	for _, b := range []struct{ file, recv string }{{"gl_purego.go", "glLib"}, {"gl_js.go", "glJS"}} {
		have := map[string]bool{}
		for _, n := range methodsOf(b.file, b.recv) {
			have[n] = true
		}
		for _, n := range api {
			if !have[n] {
				t.Errorf("%s (%s) has no %s", b.file, b.recv, n)
			}
		}
	}
}
