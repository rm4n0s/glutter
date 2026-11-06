// Command cad shows the 3D work of Glutter 1.10 together (SPEC §27): a part in a viewport3d.View that the mouse and the touch orbit,
// pan and zoom, with a grid, the axes and the view cube; the four display modes (shaded, shaded with edges, wireframe and hidden
// line); a section plane that a slider moves, with its cap and the outline of the cut; a pick that shows the item, the triangle,
// the point of the world and the normal in a side panel; and a distance measured between two picks, drawn as a dimension line with
// an Anchored label. The part is made by code (a bracket of four pieces); "Sample" loads a gear from an STL file that is built
// into the program, and "Open" asks for an STL or OBJ file with the file dialog of package desktop (not on the web, where a page
// has no file dialog yet: there the sample is what opens).
//
// Run it with: go run ./examples/cad
//
// It needs a GPU renderer (Vulkan, Direct3D 12 or WebGPU); with another one the view shows its fallback, a line of text.
//
// On Windows: go run ./cmd/glutter-windows run ./examples/cad. On the web: go run ./cmd/glutter-web serve ./examples/cad. On
// Android: go run ./cmd/glutter-android run ./examples/cad.
//
// Environment: GLUTTER_BACKEND=x11|wayland (Linux) or win32 (Windows), GLUTTER_RENDERER=vulkan|d3d12|webgpu,
// GLUTTER_SCREENSHOT=<file.png>, GLUTTER_SCREENSHOT_AFTER=<duration>. In debug builds GLUTTER_DEV_CAD=proof starts with the
// bracket cut by a plane through its middle, drawn with its edges, and the top of a post picked (the state of the proof pictures).
package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/desktop"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/model/obj"
	"github.com/rm4n0s/glutter/model/stl"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

//go:embed sample.stl
var sampleSTL []byte

func main() {
	err := m.RunApp(NewCadApp(), m.RunAppOpts{
		AppID:  "dev.glutter.cad",
		Window: m.WindowConfig{Title: "CAD viewer", Size: m.Size{Width: 1100, Height: 680}},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// CadApp is the Material app.
type CadApp struct{ m.StatelessWidget }

// NewCadApp creates the app.
func NewCadApp() *CadApp { return &CadApp{} }

// Build makes the app with the page of the viewer.
func (a *CadApp) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title:                      "CAD viewer",
		Theme:                      m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Teal.Primary), Brightness: m.BrightnessDark}),
		DebugShowCheckedModeBanner: m.Some(false),
		Home:                       NewCadPage(),
	})
}

// CadPage is the page: the view and, beside it (or under it on a narrow screen), the panel of the viewer.
type CadPage struct{ m.StatefulWidget }

// NewCadPage creates the page.
func NewCadPage(opts ...m.WidgetOpts) *CadPage {
	return &CadPage{StatefulWidget: m.NewStatefulWidgetBase(opts...)}
}

// CreateState creates the state.
func (p *CadPage) CreateState() m.State { return &cadState{} }

// tagCadLoadFailed is the tag of the error of a file that cannot be loaded as a model.
const tagCadLoadFailed = "CadLoadFailed"

type cadState struct {
	m.StateOf[*CadPage]
	scene  *viewport3d.Scene
	camera *viewport3d.Camera
	ctl    *viewport3d.ViewController

	// cancelDevice stops the calls of gpu.OnDevice; device is the one the view draws on, nil before its first frame.
	cancelDevice func()
	device       *gpu.Device

	// The model that is shown, where it came from, and what Upload made of it.
	mdl    *model.Model
	source string
	up     *viewport3d.Uploaded
	parts  map[*viewport3d.Item]viewport3d.UploadedPart

	mode      viewport3d.DisplayMode
	sectionOn bool
	axis      int     // the axis of the section plane's normal: 0 x, 1 y, 2 z
	sectionAt float64 // from 0 (the plane is at the low end of the model) to 1 (the high end)

	measure bool
	marks   []vecmath.Vec3

	// picked is what the last pick found, as text; status says what the last load did.
	picked string
	status string
}

// InitState makes the scene, the camera and the part, and asks for the device the view draws on.
func (s *cadState) InitState() {
	s.mode = viewport3d.DisplayModeShadedWithEdges
	s.sectionAt = 0.5
	s.picked = "Tap the part to pick."
	s.scene = viewport3d.NewScene()
	s.camera = viewport3d.NewCamera(viewport3d.CameraOpts{Up: vecmath.Vec3Z, Distance: 200})
	s.ctl = viewport3d.NewViewController()
	s.mdl, s.source = builtInPart(), "the built-in part"
	binding := m.WidgetsBindingOf(s.Context())
	// The view has its device a moment after its first frame (and a new one after a device is lost): the model is uploaded to it
	// each time.
	s.cancelDevice = gpu.OnDevice(binding, func(dev *gpu.Device) {
		s.SetState(func() {
			s.device = dev
			s.upload()
		})
		if foundation.DebugMode && os.Getenv("GLUTTER_DEV_CAD") == "proof" {
			s.proof(binding)
		}
	})
}

// Dispose frees the model and stops asking for the device.
func (s *cadState) Dispose() {
	if s.cancelDevice != nil {
		s.cancelDevice()
	}
	s.release()
}

// proof puts the viewer in the state of the proof pictures: cut through the middle, with edges, the top of the boss picked.
func (s *cadState) proof(binding *m.WidgetsBinding) {
	s.SetState(func() {
		s.mode, s.sectionOn, s.axis, s.sectionAt = viewport3d.DisplayModeShadedWithEdges, true, 0, 0.5
		s.applySection()
	})
	// The top of the boss, where it is drawn; the view takes a moment to have its device, so the pick is asked for again
	// until it is not refused.
	var try func(left int)
	try = func(left int) {
		time.AfterFunc(400*time.Millisecond, func() {
			binding.Post(func() {
				at, ok := s.camera.Project(vecmath.Vec3{X: -20, Y: 12, Z: 55})
				if !ok {
					return
				}
				s.ctl.Pick(at, func(r viewport3d.PickResult, err error) {
					if err != nil && left > 0 {
						try(left - 1)
						return
					}
					s.SetState(func() { s.showPick(r, err) })
				})
			})
		})
	}
	try(10)
}

// release removes the items of the model from the scene and frees what Upload made.
func (s *cadState) release() {
	if s.up == nil {
		return
	}
	s.scene.Remove(s.up.Items()...)
	s.up.Release()
	s.up, s.parts = nil, nil
}

// upload makes the items of the model on the device and shows them: the scene, the grid under the part, the camera that fits it and
// the section plane. It is called inside SetState.
func (s *cadState) upload() {
	if s.device == nil || s.mdl == nil {
		return
	}
	s.release()
	s.marks, s.measure = nil, false
	up, err := viewport3d.Upload(context.Background(), s.device, s.mdl, viewport3d.UploadOpts{FirstPickID: 1})
	if err != nil {
		s.status = "The model could not be shown: " + err.Error()
		s.picked = ""
		return
	}
	s.up = up
	s.parts = map[*viewport3d.Item]viewport3d.UploadedPart{}
	for _, p := range up.Parts {
		s.parts[p.Item] = p
	}
	s.scene.Add(up.Items()...)
	s.picked = "Tap the part to pick."
	s.status = fmt.Sprintf("%s: %d triangles", s.source, s.mdl.TriangleCount())
	s.applyStyle()
	s.applySection()
	s.camera.SetStandardView(viewport3d.StandardViewIsometric)
	s.camera.FitTo(up.Bounds)
	// The view knows its size after its layout: fit again then, so that the part fills the view whatever its aspect.
	bounds := up.Bounds
	m.WidgetsBindingOf(s.Context()).Scheduler().AddPostFrameCallback(func(time.Duration) { s.camera.FitTo(bounds) })
}

// niceSpacing is a round distance (1, 2 or 5 times a power of ten) of about a twentieth of the size.
func niceSpacing(size float64) float64 {
	if !(size > 0) {
		return 1
	}
	raw := size / 20
	pow := math.Pow(10, math.Floor(math.Log10(raw)))
	switch f := raw / pow; {
	case f < 1.5:
		return pow
	case f < 3.5:
		return 2 * pow
	case f < 7.5:
		return 5 * pow
	}
	return 10 * pow
}

// applyStyle sets the style of the scene: the display mode, and the grid on the plane under the part.
func (s *cadState) applyStyle() {
	st := viewport3d.SceneStyle{DisplayMode: s.mode}
	if s.up != nil {
		b := s.up.Bounds
		st.Grid = viewport3d.GridStyle{Show: true, Origin: vecmath.Vec3{Z: b.Min.Z}, Normal: vecmath.Vec3Z,
			Spacing: niceSpacing(b.Size().Length()), MajorEvery: 5}
	}
	s.scene.SetStyle(st)
}

// applySection puts the section plane in the scene, or takes it out.
func (s *cadState) applySection() {
	if !s.sectionOn || s.up == nil {
		s.scene.SetSectionPlanes()
		return
	}
	b := s.up.Bounds
	lo, hi := [3]float64{b.Min.X, b.Min.Y, b.Min.Z}, [3]float64{b.Max.X, b.Max.Y, b.Max.Z}
	at := b.Center()
	normal := [3]float64{}
	normal[s.axis] = 1
	p := [3]float64{at.X, at.Y, at.Z}
	p[s.axis] = lo[s.axis] + s.sectionAt*(hi[s.axis]-lo[s.axis])
	s.scene.SetSectionPlanes(vecmath.NewPlaneFromPointNormal(vecmath.Vec3{X: p[0], Y: p[1], Z: p[2]}, vecmath.Vec3{X: normal[0], Y: normal[1], Z: normal[2]}))
}

// pick asks for what is under a point of the view: it selects the item and shows what the pick found, or, when the viewer measures,
// adds a mark.
func (s *cadState) pick(at ui.Offset) {
	s.ctl.Pick(at, func(r viewport3d.PickResult, err error) {
		s.SetState(func() { s.showPick(r, err) })
	})
}

// showPick shows what a pick found. It is called inside SetState.
func (s *cadState) showPick(r viewport3d.PickResult, err error) {
	switch {
	case err != nil:
		s.picked = "The pick failed: " + err.Error()
	case !r.Hit:
		s.scene.ClearSelection()
		s.picked = "Nothing there."
	case s.measure:
		if len(s.marks) >= 2 {
			s.marks = nil
		}
		s.marks = append(s.marks, r.Point)
		s.picked = s.pickText(r)
	default:
		s.scene.Select(r.Item)
		s.picked = s.pickText(r)
	}
}

func (s *cadState) pickText(r viewport3d.PickResult) string {
	name := "an item"
	if p, ok := s.parts[r.Item]; ok {
		name = p.Node.Name
		if name == "" {
			name = p.Mesh.Name
		}
		if name == "" {
			name = fmt.Sprintf("part %d", r.PickID)
		}
	}
	kind := strings.TrimSuffix(r.Primitive.String(), "s") // "triangles" is the name of the primitive; one of them was hit
	return fmt.Sprintf("Picked %s\n%s %d\npoint (%.3f, %.3f, %.3f)\nnormal (%.2f, %.2f, %.2f)",
		name, kind, r.Index, r.Point.X, r.Point.Y, r.Point.Z, r.Normal.X, r.Normal.Y, r.Normal.Z)
}

// distance is the distance between the two marks, and the text for it.
func (s *cadState) distance() (float64, string) {
	if len(s.marks) < 2 {
		return 0, ""
	}
	d := s.marks[0].Sub(s.marks[1]).Length()
	unit := "units"
	if s.mdl != nil && s.mdl.Units.Name != "" {
		unit = s.mdl.Units.Name
	}
	return d, fmt.Sprintf("%.2f %s", d, unit)
}

// show makes the model the one that is shown.
func (s *cadState) show(mdl *model.Model, source string) {
	s.SetState(func() {
		s.mdl, s.source = mdl, source
		s.upload()
	})
}

// loadSample reads the STL file that is built into the program.
func (s *cadState) loadSample() {
	mdl, err := stl.Decode(context.Background(), bytes.NewReader(sampleSTL), stl.Options{Name: "gear"})
	if err != nil {
		s.SetState(func() { s.status = "The sample could not be read: " + err.Error() })
		return
	}
	mdl.Units = model.Units{Name: "mm", MetersPerUnit: 0.001}
	s.show(mdl, "the sample gear")
}

// open asks for a file with the dialog of the desktop and loads it, on a goroutine of its own: neither waits on the UI thread.
func (s *cadState) open() {
	bc := s.Context()
	binding := m.WidgetsBindingOf(bc)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		res, err := desktop.OpenFiles(ctx, bc, desktop.OpenFilesOpts{
			Title: "Open a model",
			Filters: []desktop.FileFilter{{Name: "Models", Patterns: []string{"*.stl", "*.STL", "*.obj", "*.OBJ"},
				MIMETypes: []string{"model/stl", "model/obj"}}},
		})
		if err != nil {
			binding.Post(func() { s.SetState(func() { s.status = "No file dialog: " + err.Error() }) })
			return
		}
		if res.Cancelled || len(res.Paths) == 0 {
			return
		}
		mdl, err := loadFile(ctx, res.Paths[0])
		binding.Post(func() {
			if err != nil {
				s.SetState(func() { s.status = "Could not load " + filepath.Base(res.Paths[0]) + ": " + err.Error() })
				return
			}
			s.show(mdl, filepath.Base(res.Paths[0]))
		})
	}()
}

// loadFile reads an STL or OBJ file; an OBJ file finds its material library and its images next to it.
func loadFile(ctx context.Context, path string) (*model.Model, error) {
	fsys, name := os.DirFS(filepath.Dir(path)), filepath.Base(path)
	switch strings.ToLower(filepath.Ext(name)) {
	case ".stl":
		return stl.Load(ctx, fsys, name)
	case ".obj":
		return obj.Load(ctx, fsys, name)
	}
	return nil, errors.New(tagCadLoadFailed, "the file is neither STL nor OBJ", "file", name)
}

// Build makes the view and the panel.
func (s *cadState) Build(bc m.BuildContext) m.Widget {
	dim, label := s.distance()
	_ = dim
	children := []m.Widget{
		m.NewIgnorePointer(m.NewCustomPaint(nil, m.CustomPaintOpts{Painter: newDimensionPainter(s.camera, s.marks)})),
	}
	if label != "" {
		mid := s.marks[0].Add(s.marks[1]).Scale(0.5)
		children = append(children, viewport3d.NewAnchored(mid, m.NewChip(m.NewText(label)),
			viewport3d.AnchoredOpts{Alignment: m.AlignmentBottomCenter, Offset: ui.Offset{Dy: -6}}))
	}
	view := m.NewGestureDetector(
		viewport3d.NewView(s.scene, s.camera, viewport3d.ViewOpts{
			Controller:     s.ctl,
			SemanticsLabel: "3D view of the part",
			Axes:           &viewport3d.AxesOpts{},
			ViewCube:       &viewport3d.ViewCubeOpts{},
			Children:       children,
			Fallback:       m.NewCenter(m.NewText("This app needs a GPU renderer.")),
		}),
		m.GestureDetectorOpts{OnTapUp: func(d m.TapUpDetails) { s.pick(d.LocalPosition) }},
	)
	panel := s.panel(bc)
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText("CAD viewer")}),
		Body: m.NewLayoutBuilder(func(bc m.BuildContext, c m.BoxConstraints) m.Widget {
			if c.MaxWidth >= 760 {
				return m.NewRow([]m.Widget{m.NewExpanded(view), m.NewSizedBox(panel, m.SizedBoxOpts{Width: m.Some(344.0)})})
			}
			return m.NewColumn([]m.Widget{m.NewExpanded(view), m.NewSizedBox(panel, m.SizedBoxOpts{Height: m.Some(280.0)})})
		}),
	})
}

// panel is the controls and what the last pick found.
func (s *cadState) panel(bc m.BuildContext) m.Widget {
	title := m.ThemeOf(bc).TextTheme.TitleMedium
	section := func(text string) m.Widget {
		return m.NewPadding(m.NewEdgeInsetsFromLTRB(0, 12, 0, 4), m.NewText(text, m.TextOpts{Style: title}))
	}
	modes := m.NewSegmentedButton([]m.ButtonSegment[viewport3d.DisplayMode]{
		{Value: viewport3d.DisplayModeShaded, Label: m.NewText("Shaded")},
		{Value: viewport3d.DisplayModeShadedWithEdges, Label: m.NewText("Edges")},
		{Value: viewport3d.DisplayModeWireframe, Label: m.NewText("Wire")},
		{Value: viewport3d.DisplayModeHiddenLine, Label: m.NewText("Hidden")},
	}, []viewport3d.DisplayMode{s.mode}, func(sel []viewport3d.DisplayMode) {
		s.SetState(func() { s.mode = sel[0]; s.applyStyle() })
	}, m.SegmentedButtonOpts{NoShowSelectedIcon: true})
	axes := m.NewSegmentedButton([]m.ButtonSegment[int]{
		{Value: 0, Label: m.NewText("X")}, {Value: 1, Label: m.NewText("Y")}, {Value: 2, Label: m.NewText("Z")},
	}, []int{s.axis}, func(sel []int) {
		s.SetState(func() { s.axis = sel[0]; s.applySection() })
	}, m.SegmentedButtonOpts{NoShowSelectedIcon: true})
	buttons := []m.Widget{
		m.NewFilledButtonTonal(s.loadSample, m.NewText("Sample")),
		m.NewFilledButtonTonal(func() { s.SetState(func() { s.mdl, s.source = builtInPart(), "the built-in part"; s.upload() }) }, m.NewText("Part")),
	}
	if !foundation.IsWeb {
		buttons = append([]m.Widget{m.NewFilledButton(s.open, m.NewText("Open…"))}, buttons...)
	}
	_, label := s.distance()
	measureText := "Turn on Measure, then pick two points."
	if label != "" {
		measureText = "Distance " + label
	} else if s.measure && len(s.marks) == 1 {
		measureText = "Pick the second point."
	}
	return m.NewSingleChildScrollView(m.NewPadding(m.NewEdgeInsetsAll(16), m.NewColumn([]m.Widget{
		m.NewText(s.status),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(8.0)}),
		m.NewWrap(buttons, m.WrapOpts{Spacing: 8, RunSpacing: 8}),
		// The pick comes first: on a phone the panel is a short list that scrolls, and what was picked is what the user looks for.
		section("Pick"),
		m.NewText(s.picked),
		section("Display"),
		modes,
		section("Section plane"),
		m.NewSwitchListTile(s.sectionOn, func(on bool) { s.SetState(func() { s.sectionOn = on; s.applySection() }) },
			m.SwitchListTileOpts{Title: m.NewText("Cut")}),
		axes,
		m.NewSlider(s.sectionAt, func(v float64) { s.SetState(func() { s.sectionAt = v; s.applySection() }) },
			m.SliderOpts{Label: m.Some("position")}),
		section("Dimensions"),
		m.NewSwitchListTile(s.measure, func(on bool) { s.SetState(func() { s.measure = on; s.marks = nil }) },
			m.SwitchListTileOpts{Title: m.NewText("Measure")}),
		m.NewText(measureText),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})))
}
