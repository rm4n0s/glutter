package viewport3d

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/widgets"
)

// ViewOpts are the optional parameters of NewView.
//
// Flutter: no counterpart
type ViewOpts struct {
	// Key is the key of the widget.
	Key foundation.Key
	// Samples is the number of samples of the picture's multisampling, 1 or 4; the default is 4.
	Samples int
	// Layers is the bit mask of the layers that the view draws (RendererOpts.Layers); the default is all.
	Layers uint32
	// Navigation is how the camera is moved by the pointer, the touch and the keys (NavigationOpts).
	Navigation NavigationOpts
	// NoNavigation leaves the navigation out: the view shows the camera, and the app moves it.
	NoNavigation bool
	// Continuous draws at every frame, not only when the scene or the camera changes.
	Continuous bool
	// Controller asks the view to draw once more, for a change that neither the scene nor the camera tell (a texture of a
	// material that a decoder filled, say), and to pick (ViewController.Pick).
	Controller *ViewController
	// Fallback is the widget shown when the app has no GPU; the default is an empty box.
	Fallback widgets.Widget
	// SemanticsLabel describes the view to assistive technology; the default is none.
	SemanticsLabel string
	// Children are widgets over the picture, in order: an Anchored follows a point of the world (NewAnchored), any other fills the
	// view. The pointer reaches them before the navigation.
	Children []widgets.Widget
	// Axes shows the axes of the camera in a corner (NewAxes); nil shows none. AxesAlignment is the corner, the default being the
	// bottom left.
	Axes          *AxesOpts
	AxesAlignment foundation.Opt[painting.Alignment]
	// ViewCube shows the cube that goes to the standard views (NewViewCube); nil shows none. ViewCubeAlignment is the corner, the
	// default being the top right.
	ViewCube          *ViewCubeOpts
	ViewCubeAlignment foundation.Opt[painting.Alignment]
}

// overlayMargin is the space between the axes or the view cube and the edge of the view, in logical pixels.
const overlayMargin = 8.0

// TagViewInvalid is the tag of the panic of NewView for arguments that are not valid (SPEC §27.8).
const TagViewInvalid = "Viewport3DViewInvalid"

// View is the widget of a 3D scene: a gpu.View that draws a Scene as a Camera sees it with a Renderer, inside a Navigation that
// moves the camera. It draws when the scene or the camera changes, and not otherwise, so an idle view costs no CPU. It tells the
// camera the scene's bounds, and the size of the view in logical pixels (through the navigation).
//
// Flutter: no counterpart
type View struct {
	widgets.StatefulWidget
	scene  *Scene
	camera *Camera
	o      ViewOpts
}

// NewView creates a view of the scene as the camera sees it. It panics (TagViewInvalid) in debug builds for a nil scene or
// camera, and for a number of samples that is not 1 or 4.
//
// Flutter: no counterpart
func NewView(scene *Scene, camera *Camera, opts ...ViewOpts) *View {
	o := firstOpts(opts)
	if foundation.DebugMode {
		if scene == nil || camera == nil {
			panic(errors.New(TagViewInvalid, "viewport3d.NewView needs a scene and a camera", "scene", scene != nil, "camera", camera != nil))
		}
		if o.Samples != 0 && o.Samples != 1 && o.Samples != 4 {
			panic(errors.New(TagViewInvalid, "viewport3d.NewView: Samples is 1 or 4", "samples", o.Samples))
		}
	}
	if o.Samples == 0 {
		o.Samples = 4
	}
	return &View{StatefulWidget: widgets.NewStatefulWidgetBase(widgets.WidgetOpts{Key: o.Key}), scene: scene, camera: camera, o: o}
}

// Scene is the scene the view draws.
//
// Flutter: no counterpart
func (v *View) Scene() *Scene { return v.scene }

// Project is the point of the view, in logical pixels, where a point of the world is drawn, and false when the point is behind
// the camera (Camera.Project): for dimension lines and marks that a CustomPainter draws over the view.
//
// Flutter: no counterpart
func (v *View) Project(p vecmath.Vec3) (ui.Offset, bool) { return v.camera.Project(p) }

// Camera is the camera the view draws with.
//
// Flutter: no counterpart
func (v *View) Camera() *Camera { return v.camera }

// CreateState creates the state of the view.
//
// Flutter: no counterpart
func (v *View) CreateState() widgets.State { return &viewState{} }

type viewState struct {
	widgets.StateOf[*View]

	renderer *Renderer
	// ctl is the controller of the options, or one of the state's own.
	ctl *ViewController
	// device is the device of the gpu.View while it has one, for a renderer that is made later (a new scene or camera).
	device *gpu.Device
	// ownsController is whether the controller was made by the state.
	ownsController bool

	// depthFormat is the depth format that the last build gave the gpu.View.
	depthFormat gpu.TextureFormat
	// scene and camera are the ones that the state listens to, which are the widget's before a change of the widget.
	scene                  *Scene
	camera                 *Camera
	sceneHandle, camHandle foundation.ListenerHandle
	// anchors is the layer of the anchored children, when the view has overlays.
	anchors anchorHost
}

// InitState makes the renderer and starts listening.
func (s *viewState) InitState() {
	s.listen()
}

// DidUpdateWidget follows a new scene, camera or options.
func (s *viewState) DidUpdateWidget(old *View) {
	w := s.Widget()
	if old.scene != w.scene || old.camera != w.camera || old.o.Layers != w.o.Layers {
		s.unlisten()
		if s.renderer != nil {
			s.renderer.Dispose()
			s.renderer = nil
		}
		s.listen()
	}
	if old.o.Controller != w.o.Controller {
		s.syncController()
	}
	s.markNeedsPaint()
}

// Dispose stops listening and releases the renderer.
func (s *viewState) Dispose() {
	s.unlisten()
	if s.renderer != nil {
		s.renderer.Dispose()
		s.renderer = nil
	}
	if s.ctl != nil {
		s.ctl.renderer = nil
	}
}

func (s *viewState) syncController() {
	if s.ctl != nil && s.ctl.renderer == s.renderer {
		s.ctl.renderer = nil
	}
	if c := s.Widget().o.Controller; c != nil {
		s.ctl, s.ownsController = c, false
	} else if !s.ownsController || s.ctl == nil {
		s.ctl, s.ownsController = NewViewController(), true
	}
	s.ctl.renderer = s.renderer
}

func (s *viewState) listen() {
	w := s.Widget()
	s.syncController()
	w.camera.SetBounds(w.scene.Bounds())
	s.scene, s.camera = w.scene, w.camera
	s.sceneHandle = s.scene.AddListener(s.sceneChanged)
	s.camHandle = s.camera.AddListener(s.markNeedsPaint)
	s.renderer = NewRenderer(w.scene, w.camera, RendererOpts{Layers: w.o.Layers})
	s.ctl.renderer = s.renderer
	if s.device != nil {
		s.renderer.Init(s.device)
	}
}

func (s *viewState) unlisten() {
	if s.scene != nil {
		s.scene.RemoveListener(s.sceneHandle)
		s.camera.RemoveListener(s.camHandle)
		s.scene, s.camera = nil, nil
	}
}

// sceneChanged tells the camera the new bounds, and draws.
func (s *viewState) sceneChanged() {
	s.camera.SetBounds(s.scene.Bounds())
	if s.neededDepthFormat() != s.depthFormat {
		// The caps of a section need a stencil: the view makes new targets.
		s.SetState(func() {})
	}
	s.markNeedsPaint()
}

func (s *viewState) markNeedsPaint() {
	if s.ctl != nil {
		s.ctl.MarkNeedsPaint()
	}
}

// neededDepthFormat is the depth format that the scene needs: one with a stencil while section planes cut it and their caps
// are on, and Depth32Float otherwise, which has the better precision.
func (s *viewState) neededDepthFormat() gpu.TextureFormat {
	if len(s.scene.planes) > 0 && s.scene.rs.section.cap {
		return gpu.TextureFormatDepth24PlusStencil8
	}
	return gpu.TextureFormatDepth32Float
}

// Build makes the gpu.View, in a navigation unless the options leave it out.
func (s *viewState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	s.depthFormat = s.neededDepthFormat()
	var child widgets.Widget = gpu.NewView(s.paint, gpu.ViewOpts{
		DepthFormat:    s.depthFormat,
		SampleCount:    w.o.Samples,
		Continuous:     w.o.Continuous,
		Controller:     s.ctl.gpu,
		OnInit:         s.onInit,
		OnDispose:      s.onDispose,
		Fallback:       w.o.Fallback,
		SemanticsLabel: w.o.SemanticsLabel,
	})
	if !w.o.NoNavigation {
		nav := w.o.Navigation
		if nav.PivotAt == nil {
			// An orbit turns around the point under the pointer, and the wheel zooms toward it (SPEC §27.8).
			nav.PivotAt = s.pivotAt
		}
		child = NewNavigation(w.camera, child, nav)
	}
	layers := []widgets.Widget{child}
	if a := w.o.Axes; a != nil {
		corner := w.o.AxesAlignment.OrElse(painting.Alignment{X: -1, Y: 1})
		// The axes are a picture: the pointer goes through them.
		layers = append(layers, widgets.NewAlign(widgets.NewPadding(painting.NewEdgeInsetsAll(overlayMargin), widgets.NewIgnorePointer(NewAxes(w.camera, *a))),
			widgets.AlignOpts{Alignment: corner}))
	}
	if c := w.o.ViewCube; c != nil {
		corner := w.o.ViewCubeAlignment.OrElse(painting.Alignment{X: 1, Y: -1})
		layers = append(layers, widgets.NewAlign(widgets.NewPadding(painting.NewEdgeInsetsAll(overlayMargin), NewViewCube(w.camera, *c)),
			widgets.AlignOpts{Alignment: corner}))
	}
	layers = append(layers, w.o.Children...)
	if len(layers) == 1 {
		return child
	}
	return newAnchoredLayer(w.camera, &s.anchors, layers)
}

// pivotAt is the point of the scene under a point of the view, for the navigation: the first triangle of the scene that the line of
// sight meets (Scene.Raycast), or false for the sky.
func (s *viewState) pivotAt(at ui.Offset) (vecmath.Vec3, bool) {
	hit, ok := s.scene.Raycast(s.camera.Ray(at))
	return hit.Point, ok
}

// onInit gives the renderer the device of the gpu.View.
func (s *viewState) onInit(d *gpu.Device) {
	s.device = d
	if s.renderer != nil {
		s.renderer.Init(d)
	}
}

// onDispose takes the device away from the renderer and frees what it made on it.
func (s *viewState) onDispose() {
	s.device = nil
	if s.renderer != nil {
		s.renderer.Dispose()
	}
}

// paint draws the scene into the frame of the gpu.View.
func (s *viewState) paint(f *gpu.ViewFrame) {
	if s.renderer != nil {
		if s.anchors.layer != nil {
			// The anchored children that hide when covered ask for the depth in the frame that is about to be drawn.
			s.anchors.layer.askDepth(s.renderer.Pick)
		}
		s.renderer.Draw(f)
	}
}

// ViewController asks a View to draw and to pick (SPEC §27.8). Make one with NewViewController and pass it as ViewOpts.Controller.
//
// Flutter: no counterpart
type ViewController struct {
	gpu      *gpu.ViewController
	renderer *Renderer
}

// NewViewController creates a controller.
//
// Flutter: no counterpart
func NewViewController() *ViewController { return &ViewController{gpu: gpu.NewViewController()} }

// MarkNeedsPaint asks the view to draw once more, in the next frame.
//
// Flutter: no counterpart
func (c *ViewController) MarkNeedsPaint() { c.gpu.MarkNeedsPaint() }

// Pick asks for what is under a point of the view, in logical pixels (Renderer.Pick): the view draws the pick in its next frame,
// which is scheduled if there is none, and done is called with the result from the event loop after the GPU has finished. A
// controller that has no view, or a view that has no GPU, calls done at once with PickUnavailable.
//
// Flutter: no counterpart
func (c *ViewController) Pick(at ui.Offset, done func(PickResult, error), opts ...PickOpts) {
	if c.renderer == nil || c.renderer.dev == nil {
		done(PickResult{}, errors.New(TagPickUnavailable, "the controller has no view with a GPU", "point", at.String()))
		return
	}
	c.renderer.Pick(at, done, opts...)
	c.gpu.MarkNeedsPaint()
}

// PickRect asks for the items that are in a rectangle of the view, in logical pixels, as Pick does: for a selection with a box.
//
// Flutter: no counterpart
func (c *ViewController) PickRect(rect ui.Rect, done func(PickRectResult, error)) {
	if c.renderer == nil || c.renderer.dev == nil {
		done(PickRectResult{}, errors.New(TagPickUnavailable, "the controller has no view with a GPU", "rect", rect.String()))
		return
	}
	c.renderer.PickRect(rect, done)
	c.gpu.MarkNeedsPaint()
}
