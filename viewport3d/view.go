package viewport3d

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
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
	// material that a decoder filled, say).
	Controller *gpu.ViewController
	// Fallback is the widget shown when the app has no GPU; the default is an empty box.
	Fallback widgets.Widget
	// SemanticsLabel describes the view to assistive technology; the default is none.
	SemanticsLabel string
}

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
	// device is the device of the gpu.View while it has one, for a renderer that is made later (a new scene or camera).
	device     *gpu.Device
	controller *gpu.ViewController
	// ownsController is whether the controller was made by the state.
	ownsController bool

	// scene and camera are the ones that the state listens to, which are the widget's before a change of the widget.
	scene                  *Scene
	camera                 *Camera
	sceneHandle, camHandle foundation.ListenerHandle
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
}

func (s *viewState) syncController() {
	if s.ownsController && s.Widget().o.Controller != nil {
		s.ownsController = false
	}
	if c := s.Widget().o.Controller; c != nil {
		s.controller = c
	} else if !s.ownsController || s.controller == nil {
		s.controller = gpu.NewViewController()
		s.ownsController = true
	}
}

func (s *viewState) listen() {
	w := s.Widget()
	s.syncController()
	w.camera.SetBounds(w.scene.Bounds())
	s.scene, s.camera = w.scene, w.camera
	s.sceneHandle = s.scene.AddListener(s.sceneChanged)
	s.camHandle = s.camera.AddListener(s.markNeedsPaint)
	s.renderer = NewRenderer(w.scene, w.camera, RendererOpts{Layers: w.o.Layers})
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
	s.markNeedsPaint()
}

func (s *viewState) markNeedsPaint() {
	if s.controller != nil {
		s.controller.MarkNeedsPaint()
	}
}

// Build makes the gpu.View, in a navigation unless the options leave it out.
func (s *viewState) Build(bc widgets.BuildContext) widgets.Widget {
	w := s.Widget()
	var child widgets.Widget = gpu.NewView(s.paint, gpu.ViewOpts{
		DepthFormat:    gpu.TextureFormatDepth32Float,
		SampleCount:    w.o.Samples,
		Continuous:     w.o.Continuous,
		Controller:     s.controller,
		OnInit:         s.onInit,
		OnDispose:      s.onDispose,
		Fallback:       w.o.Fallback,
		SemanticsLabel: w.o.SemanticsLabel,
	})
	if !w.o.NoNavigation {
		child = NewNavigation(w.camera, child, w.o.Navigation)
	}
	return child
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
		s.renderer.Draw(f)
	}
}
