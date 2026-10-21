//go:build js && wasm

package main

import (
	"encoding/json"
	"image"
	"strconv"
	"syscall/js"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/gles"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/platform/web"
	"github.com/rm4n0s/glutter/internal/proof"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
)

// viewAttribute names the view a canvas shows (internal/platform/web ViewAttribute).
const viewAttribute = "data-glutter-view"

func run() error {
	switch mode := param("mode"); mode {
	case "parity":
		postResult(parity())
	case "unavailable":
		postResult(unavailable())
	case "loss":
		postResult(loss())
	case "choose", "restore":
		return engineApp(mode)
	default:
		return errors.New("GLWebTestModeUnknown", "the query parameter mode is not parity, unavailable, loss, choose or restore", "mode", mode)
	}
	// The result is posted by an asynchronous call: the program stays until the test has it, so
	// that its ending is not taken for the program's failure.
	select {}
}

// param reads a query parameter of the page.
func param(name string) string {
	q := js.Global().Get("URLSearchParams").New(js.Global().Get("location").Get("search"))
	if v := q.Call("get", name); v.Type() == js.TypeString {
		return v.String()
	}
	return ""
}

// postResult posts the result to the test server; the answer is not waited for.
func postResult(data any) {
	body, err := json.Marshal(map[string]any{"kind": "glweb", "data": data})
	if err != nil {
		panic(err)
	}
	init := map[string]any{"method": "POST", "headers": map[string]any{"Content-Type": "application/json"}, "body": string(body)}
	js.Global().Call("fetch", web.ResultsPath, init)
}

// newCanvas puts a canvas of the view, of the size, into the page.
func newCanvas(view, width, height int) js.Value {
	doc := js.Global().Get("document")
	c := doc.Call("createElement", "canvas")
	c.Call("setAttribute", viewAttribute, strconv.Itoa(view))
	c.Set("width", width)
	c.Set("height", height)
	doc.Get("body").Call("appendChild", c)
	return c
}

func handleOf(view int) gpu.SurfaceHandle {
	return gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: uintptr(view)}
}

func diff(a, b *image.RGBA) diffStats {
	d := gputest.DiffRect(a, b, a.Bounds())
	return diffStats{Pixels: d.Pixels, Over8: d.Over8, Over32: d.Over32, Over96: d.Over96, Max: d.Max}
}

func tagOf(err error) tagResult {
	if err == nil {
		return tagResult{}
	}
	if appErr, ok := errors.FromError(err); ok {
		return tagResult{Tag: appErr.Tag, Route: appErr.Route()}
	}
	return tagResult{Tag: "not an *errors.Error: " + err.Error()}
}

// parity draws every proof scene with the gles renderer on WebGL 2 and compares what it reads
// back with what the software renderer draws. The renderer's background is transparent, so
// that the frame is the scene alone, as soft draws it.
func parity() parityResult {
	var res parityResult
	for _, name := range proof.Names() {
		sc, err := proof.Lookup(name)
		if err != nil {
			res.Scenes = append(res.Scenes, sceneResult{Name: name, Error: err.Error()})
			continue
		}
		res.Scenes = append(res.Scenes, paritySceneOf(sc, &res))
	}
	return res
}

func paritySceneOf(sc proof.Scene, res *parityResult) sceneResult {
	out := sceneResult{Name: sc.Name, Width: sc.Width, Height: sc.Height}
	fail := func(step string, err error) sceneResult {
		out.Error = step + ": " + err.Error()
		if t := tagOf(err); t.Tag != "" {
			out.Error += " [" + t.Tag + " " + t.Route + "]"
		}
		return out
	}
	scene := sc.Build()
	want, err := soft.Render(scene, sc.Width, sc.Height)
	if err != nil {
		return fail("soft", err)
	}
	canvas := newCanvas(0, sc.Width, sc.Height)
	defer canvas.Call("remove")
	r := gles.NewRenderer(gles.RendererOpts{Background: ui.NewColor(0), HasBackground: true})
	defer func() { _ = r.Close() }()
	s, err := r.Attach(handleOf(0))
	if err != nil {
		return fail("Attach", err)
	}
	if res.Renderer == "" {
		res.Renderer = r.DeviceName()
	}
	size := ui.Size{Width: float64(sc.Width), Height: float64(sc.Height)}
	got, err := r.RenderCapture(s, scene, size, 1)
	if err != nil {
		return fail("RenderCapture", err)
	}
	st := r.LastFrame()
	out.Skipped, out.Unsupported = st.Skipped, st.Unsupported
	out.Frame = diff(got, want)
	snap, err := r.Snapshot(scene, size)
	if err != nil {
		return fail("Snapshot", err)
	}
	out.Snapshot, out.SnapshotVsFrame = diff(snap, want), diff(snap, got)
	again, err := r.RenderCapture(s, scene, size, 1)
	if err != nil {
		return fail("RenderCapture again", err)
	}
	out.Repeat = diff(again, got)
	return out
}

// unavailable asks for contexts that cannot be made.
func unavailable() unavailableResult {
	var res unavailableResult
	attach := func(r *gles.Renderer, h gpu.SurfaceHandle) error {
		_, err := r.Attach(h)
		return err
	}
	taken := newCanvas(1, 16, 16)
	taken.Call("getContext", "2d")
	r := gles.NewRenderer()
	res.Taken = tagOf(attach(r, handleOf(1)))
	_ = r.Close()

	r = gles.NewRenderer()
	res.NoCanvas = tagOf(attach(r, handleOf(99)))
	_ = r.Close()

	r = gles.NewRenderer()
	res.WrongKind = tagOf(attach(r, gpu.SurfaceHandle{Kind: gpu.SurfaceKindWayland, Window: 1}))
	_ = r.Close()

	newCanvas(2, 16, 16)
	newCanvas(3, 16, 16)
	r = gles.NewRenderer()
	res.Fresh = attach(r, handleOf(2)) == nil
	res.Second = tagOf(attach(r, handleOf(3)))
	_ = r.Close()
	return res
}

// lossScene draws a few things that the renderer's caches hold (a path, a clip, an image).
func lossScene() *ui.Scene {
	return gputest.CanvasScene(func(c *ui.Canvas) {
		c.DrawRect(ui.NewRectFromLTWH(0, 0, 200, 150), gputest.FillPaint(0xFFEEEEEE))
		c.ClipRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(10, 10, 160, 110), 16, 16))
		path := ui.NewPath()
		path.MoveTo(20, 20)
		path.LineTo(180, 40)
		path.LineTo(60, 130)
		path.Close()
		c.DrawPath(path, gputest.FillPaint(0xFF3366CC))
		c.DrawCircle(ui.Offset{Dx: 100, Dy: 70}, 30, gputest.FillPaint(0x80E53935))
		pix := make([]byte, 4*8*8)
		for i := range pix {
			pix[i] = byte(i * 7)
		}
		c.DrawImage(ui.NewImageFromRGBA(pix, 8, 8), ui.Offset{Dx: 120, Dy: 20}, nil)
	})
}

// loss loses the context with the browser's own extension and has it restored.
func loss() lossResult {
	var res lossResult
	canvas := newCanvas(0, 200, 150)
	restored := 0
	r := gles.NewRenderer(gles.RendererOpts{Background: ui.NewColor(0), HasBackground: true, OnContextRestored: func() { restored++ }})
	defer func() { _ = r.Close() }()
	s, err := r.Attach(handleOf(0))
	if err != nil {
		return lossResult{AfterError: "Attach: " + err.Error()}
	}
	scene := lossScene()
	size := ui.Size{Width: 200, Height: 150}
	first, err := r.RenderCapture(s, scene, size, 1)
	if err != nil {
		return lossResult{AfterError: "first frame: " + err.Error()}
	}
	res.First = true
	ext := canvas.Call("getContext", "webgl2").Call("getExtension", "WEBGL_lose_context")
	ext.Call("loseContext")
	time.Sleep(300 * time.Millisecond) // the browser's event is a task of its own
	_, err = r.RenderCapture(s, scene, size, 1)
	res.LostCapture = tagOf(err).Tag
	res.LostRenderOK = r.Render(s, scene, size, 1) == nil
	_, err = r.Snapshot(scene, size)
	res.LostSnapshot = tagOf(err).Tag

	ext.Call("restoreContext")
	for i := 0; i < 100 && restored == 0; i++ {
		time.Sleep(50 * time.Millisecond)
	}
	res.Restored = restored
	after, err := r.RenderCapture(s, scene, size, 1)
	if err != nil {
		res.AfterError = err.Error()
		return res
	}
	res.AfterVsFirst = diff(after, first)
	snap, err := r.Snapshot(scene, size)
	if err != nil {
		res.AfterError = "snapshot: " + err.Error()
		return res
	}
	res.SnapshotVsFirst = diff(snap, first)
	if err := r.Render(s, scene, size, 1); err != nil {
		res.AfterRenderError = err.Error()
	}
	return res
}

// engineApp runs a small app through the engine, which chooses its renderer, and observes the
// canvas it ends up with. With the query parameter poison the browser answers null to
// getContext("webgl2").
func engineApp(mode string) error {
	js.Global().Get("Function").New(`
		globalThis.__raf = 0;
		var raf = window.requestAnimationFrame.bind(window);
		window.requestAnimationFrame = function (cb) { globalThis.__raf++; return raf(cb); };
		if (new URLSearchParams(location.search).get("poison")) {
			var get = HTMLCanvasElement.prototype.getContext;
			HTMLCanvasElement.prototype.getContext = function (kind) {
				if (kind === "webgl2") { return null; }
				return get.apply(this, arguments);
			};
		}`).Invoke()
	go func() { postResult(observeEngine(mode)) }()
	err := m.RunApp(newApp(), m.RunAppOpts{Window: m.WindowConfig{Title: "GL web test"}})
	if err != nil {
		postResult(chooseResult{Error: tagOf(err).Tag})
	}
	select {}
}

func frames() int { return js.Global().Get("__raf").Int() }

// observeEngine waits for the app and reports its canvas.
func observeEngine(mode string) chooseResult {
	var res chooseResult
	time.Sleep(1500 * time.Millisecond)
	canvases := js.Global().Get("document").Call("querySelectorAll", "canvas["+viewAttribute+"]")
	res.Canvases = canvases.Length()
	if res.Canvases == 0 {
		return res
	}
	canvas := canvases.Index(0)
	// A canvas answers null to a kind of context that is not the one it has.
	res.WebGL2 = canvas.Call("getContext", "webgl2").Truthy()
	res.Context2D = canvas.Call("getContext", "2d").Truthy()
	if mode != "restore" || !res.WebGL2 {
		return res
	}
	res.FramesBefore = frames()
	ext := canvas.Call("getContext", "webgl2").Call("getExtension", "WEBGL_lose_context")
	ext.Call("loseContext")
	time.Sleep(400 * time.Millisecond)
	res.FramesWhileOff = frames()
	ext.Call("restoreContext")
	time.Sleep(1500 * time.Millisecond)
	res.FramesAfter = frames()
	return res
}

// app is a Material app that has nothing that moves, so that it asks for no frames while idle.
type app struct{ m.StatelessWidget }

func newApp() *app { return &app{} }

func (a *app) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "GL web test",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       m.NewScaffold(m.ScaffoldOpts{Body: m.NewCenter(m.NewText("WebGL 2"))}),
	})
}
