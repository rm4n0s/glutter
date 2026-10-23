//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"image"
	"log"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/gputest"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/gpu/webgpu"
	"github.com/rm4n0s/glutter/internal/platform/web"
	"github.com/rm4n0s/glutter/internal/proof"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
)

// viewAttribute names the view a canvas shows (internal/platform/web ViewAttribute).
const viewAttribute = "data-glutter-view"

// main runs the mode that the page asks for, and logs why it failed.
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	switch mode := param("mode"); mode {
	case "probe":
		postResult(probe())
	case "parity":
		postResult(parity())
	case "unavailable":
		postResult(unavailable())
	case "loss":
		postResult(loss())
	case "choose":
		return engineApp()
	case "failstart":
		return failStart()
	case "switch":
		return switchApp()
	default:
		return errors.New(tagGPUWebTestModeUnknown, "the query parameter mode is not probe, parity, unavailable, loss, choose, failstart or switch", "mode", mode)
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
	body, err := json.Marshal(map[string]any{"kind": "gpuweb", "data": data})
	if err != nil {
		panic(err)
	}
	init := map[string]any{"method": "POST", "headers": map[string]any{"Content-Type": "application/json"}, "body": string(body)}
	js.Global().Call("fetch", web.ResultsPath, init)
}

// eval runs JavaScript.
func eval(src string) js.Value { return js.Global().Get("Function").New(src).Invoke() }

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

// describe is an error with its tag, route and metadata, for a failure message.
func describe(err error) string {
	text := err.Error()
	if appErr, ok := errors.FromError(err); ok {
		text += " [" + appErr.Tag + " " + appErr.Route() + "]"
		if b, merr := json.Marshal(appErr.Metadata); merr == nil {
			text += " " + string(b)
		}
	}
	return text
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

// probeResult says whether the browser gives WebGPU at all.
type probeResult struct {
	Available bool   `json:"available"`
	Tag       string `json:"tag"`
	Message   string `json:"message"`
	Renderer  string `json:"renderer"`
}

func probe() probeResult {
	p, err := webgpu.Acquire(context.Background(), webgpu.Options{Timeout: time.Minute})
	if err != nil {
		t := tagOf(err)
		return probeResult{Tag: t.Tag, Message: err.Error()}
	}
	r := webgpu.NewRenderer(p.Open)
	defer r.Close()
	return probeResult{Available: true}
}

// capture is a frame that the renderer read back after Draw returned.
type capture struct {
	img *image.RGBA
	err error
}

// rig is a renderer on a canvas, with the captures it delivers.
type rig struct {
	prov     *webgpu.Provider
	r        *webgpu.Renderer
	captured chan capture
	ready    chan struct{}
	h        gpu.SurfaceHandle
	w, hgt   int
}

// newRig opens WebGPU and a renderer whose background is transparent, so that the frame is the
// scene alone, as soft draws it.
func newRig(view, w, h int) (*rig, error) {
	g := &rig{captured: make(chan capture, 8), ready: make(chan struct{}, 8), h: handleOf(view), w: w, hgt: h}
	var err error
	g.prov, err = webgpu.Acquire(context.Background(), webgpu.Options{
		Timeout:       time.Minute,
		OnDeviceReady: func(gpu.SurfaceHandle) { g.ready <- struct{}{} },
	})
	if err != nil {
		return nil, err
	}
	g.r = webgpu.NewRenderer(g.prov.Open, explicit.RendererOpts{
		Background: ui.NewColor(0), HasBackground: true,
		OnCapture: func(img *image.RGBA, err error) { g.captured <- capture{img, err} },
	})
	return g, nil
}

// frame draws the scene and waits for the read-back, which arrives in a later task.
func (g *rig) frame(scene *ui.Scene) (*image.RGBA, error) {
	img, err := g.r.Draw(g.h, g.w, g.hgt, scene, true)
	if err != nil {
		return nil, err
	}
	if img != nil {
		return nil, errors.New(tagCaptureWasSynchronous, "a WebGPU read-back came back inside Draw")
	}
	return g.await()
}

func (g *rig) await() (*image.RGBA, error) {
	select {
	case c := <-g.captured:
		return c.img, c.err
	case <-time.After(30 * time.Second):
		return nil, errors.New(tagCaptureTimeout, "the read-back did not come in 30 seconds")
	}
}

// parity draws every proof scene with the explicit renderer on WebGPU and compares what it reads
// back with what the software renderer draws.
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
		out.Error = step + ": " + describe(err)
		return out
	}
	scene := sc.Build()
	want, err := soft.Render(scene, sc.Width, sc.Height)
	if err != nil {
		return fail("soft", err)
	}
	canvas := newCanvas(0, sc.Width, sc.Height)
	defer canvas.Call("remove")
	g, err := newRig(0, sc.Width, sc.Height)
	if err != nil {
		return fail("Acquire", err)
	}
	defer g.r.Close()
	got, err := g.frame(scene)
	if err != nil {
		return fail("Draw", err)
	}
	if res.Renderer == "" {
		res.Renderer = g.r.Describe()
	}
	st := g.r.LastFrame()
	out.Skipped, out.Unsupported = st.Skipped, st.Unsupported
	out.Frame = diff(got, want)
	snap, err := g.r.Snapshot(scene, ui.Size{Width: float64(sc.Width), Height: float64(sc.Height)})
	if err != nil {
		return fail("Snapshot", err)
	}
	out.Snapshot = diff(snap, want)
	again, err := g.frame(scene)
	if err != nil {
		return fail("Draw again", err)
	}
	out.Repeat = diff(again, got)
	time.Sleep(150 * time.Millisecond) // the browser reports a broken rule in a task of its own
	res.Validation = append(res.Validation, g.prov.ValidationErrors()...)
	return out
}

// patches are the changes to the browser's WebGPU that make it fail in the ways to test.
const patches = `
globalThis.__real = {};
globalThis.__patch = {
	noGPU: function () { Object.defineProperty(navigator, 'gpu', {value: undefined, configurable: true}); },
	restoreGPU: function () { delete navigator.gpu; },
	noAdapter: function () { navigator.gpu.requestAdapter = function () { return Promise.resolve(null); }; },
	noDevice: function () {
		var g = navigator.gpu, real = Object.getPrototypeOf(g).requestAdapter;
		g.requestAdapter = async function () {
			var a = await real.call(g);
			a.requestDevice = function () { return Promise.reject(new Error('this adapter gives no device')); };
			return a;
		};
	},
	hang: function () { navigator.gpu.requestAdapter = function () { return new Promise(function () {}); }; },
	restoreAdapter: function () { delete navigator.gpu.requestAdapter; },
	badShader: function () {
		var proto = GPUDevice.prototype;
		globalThis.__real.createShaderModule = proto.createShaderModule;
		proto.createShaderModule = function (d) {
			return globalThis.__real.createShaderModule.call(this, {label: d.label, code: d.code + '\nthis is not wgsl;'});
		};
	},
	restoreShader: function () { GPUDevice.prototype.createShaderModule = globalThis.__real.createShaderModule; },
	// record every device, and make the next ones come late
	recordDevices: function () {
		globalThis.__devices = [];
		globalThis.__delay = 0;
		var real = GPUAdapter.prototype.requestDevice;
		GPUAdapter.prototype.requestDevice = async function (d) {
			if (globalThis.__refuse) { throw new Error('the browser gives no more devices'); }
			if (globalThis.__delay > 0) { await new Promise(function (r) { setTimeout(r, globalThis.__delay); }); }
			var dev = await real.call(this, d);
			globalThis.__devices.push(dev);
			dev.lost.then(function (i) { dev.__lost = i.reason + ': ' + i.message; });
			dev.addEventListener('uncapturederror', function (e) { (globalThis.__errors = globalThis.__errors || []).push(e.error.message); });
			return dev;
		};
	},
	// keep what the program writes to standard error, to report the engine's log
	captureLog: function () {
		globalThis.__log = '';
		var write = globalThis.fs.writeSync.bind(globalThis.fs);
		var decoder = new TextDecoder('utf-8');
		globalThis.fs.writeSync = function (fd, buf) { globalThis.__log += decoder.decode(buf, {stream: true}); return write(fd, buf); };
	},
	hideGPU: function () { Object.defineProperty(navigator, 'gpu', {value: undefined, configurable: true}); },
	poisonGL: function () {
		var get = HTMLCanvasElement.prototype.getContext;
		HTMLCanvasElement.prototype.getContext = function (kind) {
			if (kind === 'webgl2') { return null; }
			return get.apply(this, arguments);
		};
	},
};
`

func patch(name string) { js.Global().Get("__patch").Call(name) }

func acquireWith(timeout time.Duration) error {
	p, err := webgpu.Acquire(context.Background(), webgpu.Options{Timeout: timeout})
	if err != nil {
		return err
	}
	r := webgpu.NewRenderer(p.Open)
	r.Close()
	return nil
}

// unavailable makes WebGPU fail in each way and asks for swapchains that cannot be made.
func unavailable() unavailableResult {
	var res unavailableResult
	eval(patches)

	patch("noGPU")
	res.NoGPU = tagOf(acquireWith(time.Minute))
	patch("restoreGPU")

	patch("noAdapter")
	res.NoAdapter = tagOf(acquireWith(time.Minute))
	patch("restoreAdapter")

	patch("noDevice")
	res.NoDevice = tagOf(acquireWith(time.Minute))
	patch("restoreAdapter")

	patch("badShader")
	res.BadShader = tagOf(acquireWith(time.Minute))
	patch("restoreShader")

	patch("hang")
	res.Timeout = tagOf(acquireWith(300 * time.Millisecond))
	patch("restoreAdapter")

	var none *webgpu.Provider
	_, err := none.Open(handleOf(0))
	res.OpenOfNil = tagOf(err)

	// Swapchains: one provider, its device, canvases of every kind.
	g, err := newRig(0, 16, 16)
	if err != nil {
		res.Taken = tagOf(err)
		return res
	}
	defer g.r.Close()
	dev, err := g.prov.Open(handleOf(0))
	if err != nil {
		res.Taken = tagOf(err)
		return res
	}
	defer func() { _ = dev.Close() }()
	taken := newCanvas(1, 16, 16)
	taken.Call("getContext", "2d")
	_, err = dev.NewSwapchain(handleOf(1))
	res.Taken = tagOf(err)
	_, err = dev.NewSwapchain(handleOf(99))
	res.NoCanvas = tagOf(err)
	_, err = dev.NewSwapchain(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWayland, Window: 1})
	res.WrongKind = tagOf(err)
	newCanvas(2, 16, 16)
	sc, err := dev.NewSwapchain(handleOf(2))
	if err == nil {
		res.Fresh = sc.Configure(16, 16) == nil
		sc.Destroy()
	}
	time.Sleep(150 * time.Millisecond)
	res.Validation = g.prov.ValidationErrors()
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

// destroyLatest destroys the newest GPUDevice of the page, as a driver reset or a tab being
// discarded would take it away, and lets the browser tell.
func destroyLatest() {
	devs := js.Global().Get("__devices")
	dev := devs.Index(devs.Length() - 1)
	dev.Call("destroy")
	// The browser tells in a task of its own, in Firefox a few of them later.
	for i := 0; i < 100 && !dev.Get("__lost").Truthy(); i++ {
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(60 * time.Millisecond) // and the renderer's callback runs after the test's
}

// loss destroys the device under a running renderer, twice.
func loss() lossResult {
	var res lossResult
	eval(patches)
	patch("recordDevices")
	newCanvas(0, 200, 150)
	g, err := newRig(0, 200, 150)
	if err != nil {
		return lossResult{AfterError: "Acquire: " + err.Error()}
	}
	defer g.r.Close()
	scene := lossScene()
	first, err := g.frame(scene)
	if err != nil {
		return lossResult{AfterError: "first frame: " + err.Error()}
	}
	res.First = true

	// The new device comes 400 ms late, so that the frame after the loss finds it on its way.
	js.Global().Set("__delay", 400)
	destroyLatest()
	img, err := g.r.Draw(g.h, g.w, g.hgt, scene, true)
	res.DuringLoss = tagOf(err).Tag
	if err == nil && img == nil {
		res.DuringLoss = "pending"
	}
	// A frame that was drawn would deliver its capture, or the error of reading it; this one was
	// not drawn.
	select {
	case <-g.captured:
		res.DrewDuringLoss = true
	case <-time.After(200 * time.Millisecond):
	}
	select {
	case <-g.ready:
		res.Ready++
	case <-time.After(30 * time.Second):
	}
	img, err = g.r.Draw(g.h, g.w, g.hgt, scene, true)
	if err != nil || img != nil {
		res.AfterError = "the frame after the new device came: " + tagOf(err).Tag
		return res
	}
	after, err := g.await()
	if err != nil {
		res.AfterError = describe(err) + " devices: " + eval("return JSON.stringify((globalThis.__devices||[]).map(function(d){return d.__lost||'alive'})) + ' errors: ' + JSON.stringify(globalThis.__errors||[])").String()
		return res
	}
	res.AfterVsFirst = diff(after, first)

	// A second loss less than ten seconds after the first is not recovered.
	destroyLatest()
	_, err = g.r.Draw(g.h, g.w, g.hgt, scene, true)
	res.SecondLoss = tagOf(err).Tag
	time.Sleep(150 * time.Millisecond)
	res.Validation = g.prov.ValidationErrors()
	return res
}

// engineApp runs a small app through the engine, which chooses its renderer, and observes the
// canvas it ends up with.
func engineApp() error {
	eval(patches)
	if param("poison") != "" {
		patch("hideGPU")
	}
	if param("poisongl") != "" {
		patch("poisonGL")
	}
	go func() { postResult(observeEngine()) }()
	err := m.RunApp(newApp(), m.RunAppOpts{Window: m.WindowConfig{Title: "WebGPU test"}})
	if err != nil {
		postResult(chooseResult{Error: tagOf(err).Tag})
	}
	select {}
}

// observeEngine waits for the app and reports its canvas.
func observeEngine() chooseResult {
	var res chooseResult
	time.Sleep(4 * time.Second)
	canvases := js.Global().Get("document").Call("querySelectorAll", "canvas["+viewAttribute+"]")
	res.Canvases = canvases.Length()
	if res.Canvases == 0 {
		return res
	}
	canvas := canvases.Index(0)
	// A canvas answers null to a kind of context that is not the one it has.
	res.WebGPU = canvas.Call("getContext", "webgpu").Truthy()
	res.WebGL2 = canvas.Call("getContext", "webgl2").Truthy()
	res.Context2D = canvas.Call("getContext", "2d").Truthy()
	return res
}

// app is a Material app that has nothing that moves, so that it asks for no frames while idle.
type app struct{ m.StatelessWidget }

func newApp() *app { return &app{} }

func (a *app) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "WebGPU test",
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       m.NewScaffold(m.ScaffoldOpts{Body: m.NewCenter(m.NewText("WebGPU"))}),
	})
}

// failStart runs the app with navigator.gpu hidden, so that GLUTTER_RENDERER=webgpu cannot start: RunApp
// returns the error, the program ends with it, and the loader shows it in the page.
func failStart() error {
	eval(patches)
	patch("hideGPU")
	return m.RunApp(newApp(), m.RunAppOpts{Window: m.WindowConfig{Title: "WebGPU test"}})
}

// switchResult is the result of the mode switch.
type switchResult struct {
	// StartedOnWebGPU is whether the canvas had a webgpu context before the devices were taken away.
	StartedOnWebGPU bool `json:"startedOnWebGPU"`
	// Canvases is how many canvases of a view the page ends with; WebGPU, WebGL2 and Context2D say which
	// kind of context the first has.
	Canvases  int  `json:"canvases"`
	WebGPU    bool `json:"webgpu"`
	WebGL2    bool `json:"webgl2"`
	Context2D bool `json:"context2d"`
	// Devices is how many GPUDevices the page was given.
	Devices int `json:"devices"`
	// Log is what the program wrote to standard error.
	Log string `json:"log"`
}

// switchApp runs the app through the engine on WebGPU and takes the device away: twice within ten
// seconds (the query parameter lossTwice), or once with the browser giving no new device (refuse). The
// engine's last resort then switches the page to WebGL 2 on a new canvas. With a renderer asked for by
// name (GLUTTER_RENDERER=webgpu) there is no switch: RunApp ends with the error.
func switchApp() error {
	eval(patches)
	patch("recordDevices")
	patch("captureLog")
	named := param("GLUTTER_RENDERER") != ""
	go func() {
		res := observeSwitch(param("refuse") != "")
		if !named {
			postResult(res)
		}
	}()
	return m.RunApp(newApp(), m.RunAppOpts{Window: m.WindowConfig{Title: "WebGPU test"}})
}

// canvasKinds reports the canvas of the page and the kind of context it has.
func canvasKinds(res *switchResult) {
	canvases := js.Global().Get("document").Call("querySelectorAll", "canvas["+viewAttribute+"]")
	res.Canvases = canvases.Length()
	if res.Canvases == 0 {
		return
	}
	canvas := canvases.Index(0)
	// A canvas answers null to a kind of context that is not the one it has.
	res.WebGPU = canvas.Call("getContext", "webgpu").Truthy()
	res.WebGL2 = canvas.Call("getContext", "webgl2").Truthy()
	res.Context2D = canvas.Call("getContext", "2d").Truthy()
}

// observeSwitch waits for the app to draw on WebGPU, takes its device away, and waits for the page to end
// up on another kind of context.
func observeSwitch(refuse bool) switchResult {
	var res switchResult
	time.Sleep(4 * time.Second)
	canvasKinds(&res)
	res.StartedOnWebGPU = res.WebGPU
	if refuse {
		// The first loss is recovered by asking for a device, and the browser gives none.
		js.Global().Set("__refuse", true)
		destroyLatest()
	} else {
		destroyLatest()
		// Let the engine recover on the new device before the second loss, which is within ten seconds.
		for i := 0; i < 100 && js.Global().Get("__devices").Length() < 2; i++ {
			time.Sleep(50 * time.Millisecond)
		}
		time.Sleep(time.Second)
		destroyLatest()
	}
	// The engine logs each choice of a renderer. The canvas is not asked about its context while the
	// switch happens: asking a new canvas for a context would take it from the renderer.
	for i := 0; i < 400 && strings.Count(js.Global().Get("__log").String(), "renderer chosen") < 2; i++ { // up to 20 seconds
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(time.Second) // and the frames that the new renderer draws
	canvasKinds(&res)
	res.Devices = js.Global().Get("__devices").Length()
	res.Log = js.Global().Get("__log").String()
	return res
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCaptureTimeout        = "CaptureTimeout"
	tagCaptureWasSynchronous = "CaptureWasSynchronous"
	tagGPUWebTestModeUnknown = "GPUWebTestModeUnknown"
)
