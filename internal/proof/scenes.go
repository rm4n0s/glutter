// Package proof holds the scenes that cmd/glutter-proof renders to prove, with
// a picture, that a step of docs/IMPLEMENTATION.md works. A step adds its scene
// here.
//
// Flutter: no counterpart
package proof

import (
	"sort"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

// Scene is a registered proof scene.
type Scene struct {
	// Name is what -scene takes.
	Name string
	// Description says what the picture should show.
	Description string
	// Width and Height are the size of the image in pixels.
	Width, Height int
	// Build creates the scene.
	Build func() *ui.Scene
}

var scenes = map[string]Scene{}

// register adds a scene; it is called from init functions.
func register(s Scene) {
	if _, dup := scenes[s.Name]; dup {
		panic(errors.New(tagProofSceneDuplicate, "Two proof scenes have the same name.", "name", s.Name))
	}
	scenes[s.Name] = s
}

// Register adds a scene made by a package that package proof cannot import, such
// as internal/proof/widgetscenes; it is called from init functions and panics
// with the tag ProofSceneDuplicate when the name is taken.
func Register(s Scene) { register(s) }

// Names returns the names of the registered scenes in alphabetical order.
func Names() []string {
	names := make([]string, 0, len(scenes))
	for n := range scenes {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Lookup returns the scene with the name, or the error ProofSceneUnknown.
func Lookup(name string) (Scene, error) {
	s, ok := scenes[name]
	if !ok {
		return Scene{}, errors.New(tagProofSceneUnknown, "There is no proof scene with this name; -list shows them.", "name", name)
	}
	return s, nil
}

// singlePicture builds a scene of one picture drawn by fn on a canvas over a
// white background of the given size.
func singlePicture(width, height int, fn func(c *ui.Canvas)) *ui.Scene {
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec, ui.CanvasOpts{CullRect: ui.Ptr(ui.NewRectFromLTWH(0, 0, float64(width), float64(height)))})
	c.DrawColor(ui.NewColor(0xFFFFFFFF), ui.BlendModeSrcOver)
	fn(c)
	b := ui.NewSceneBuilder()
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	return b.Build()
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagProofSceneDuplicate = "ProofSceneDuplicate"
	tagProofSceneUnknown   = "ProofSceneUnknown"
)
