// Package renderingtest is a temporary harness for the tests of the rendering
// library: it builds a RendererBinding on a fake engine, lays out and paints a
// render tree through the pipeline, and renders the scene with the software
// renderer. It stands in for gluttertest (step 7.5), which replaces it.
//
// Flutter: no counterpart (test/rendering/rendering_tester.dart)
package renderingtest

import (
	"context"
	"image"
	"sync"
	"testing"

	"github.com/rm4n0s/errors"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/gpu/soft"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/layout"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/ui"
)

// Host is the fake engine.
type Host = renderhost.Host

// Opts are the optional parameters of New.
type Opts struct {
	// Size is the size of the view in physical pixels; the default is 800x600.
	Size ui.Size
	// DevicePixelRatio defaults to 1.
	DevicePixelRatio float64
	// Debug are the debug flags; the default is a fresh set.
	Debug *rendering.DebugOptions
	// TextEngine lays out the text; the default is BundledTextEngine().
	TextEngine ui.ParagraphFactory
	// CollectPanics makes the harness collect the panics that layout and paint
	// recover in Errors. By default it panics again with them, so that a test
	// that does not expect one fails where it happens.
	CollectPanics bool
}

// Harness is a rendering binding on a fake engine, with one view.
type Harness struct {
	*renderhost.Stack
	// TextEngine is the engine that lays out the text.
	TextEngine ui.ParagraphFactory
	// Errors collects the errors that the binding reported.
	Errors []error
}

// New creates the harness with the render view attached to the binding's root
// pipeline owner, ready for SetRoot.
func New(t *testing.T, opts ...Opts) *Harness {
	t.Helper()
	var o Opts
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.TextEngine == nil {
		o.TextEngine = BundledTextEngine()
	}
	h := &Harness{TextEngine: o.TextEngine}
	h.Stack = renderhost.New(renderhost.Opts{Size: o.Size, DevicePixelRatio: o.DevicePixelRatio, Debug: o.Debug,
		TextEngine: o.TextEngine, ReportError: func(err error) {
			if !o.CollectPanics && foundation.IsRecoveredPanic(err) {
				panic(err)
			}
			h.Errors = append(h.Errors, err)
		}})
	return h
}

// Pump runs one frame: layout, compositing bits, paint and composite.
func (h *Harness) Pump() { h.Stack.Pump() }

// Image renders the last scene with the software renderer.
func (h *Harness) Image() (*image.RGBA, error) {
	s := h.LastScene()
	if s == nil {
		return nil, errors.New(tagRenderingTestNoScene, "no frame was composited yet; call Pump first")
	}
	size := h.View.PhysicalSize()
	return soft.Render(s, int(size.Width), int(size.Height))
}

// BundledTextEngine is a text engine that has only the fonts bundled with
// Glutter, so that the text is laid out the same on every machine. The engine is
// made once per process, because registering the fonts is slow.
func BundledTextEngine() *layout.Engine { return bundledEngine() }

var bundledEngine = sync.OnceValue(func() *layout.Engine {
	c := text.NewCollection(text.CollectionOpts{Matcher: noSystemFonts{}})
	if err := fonts.Register(c); err != nil {
		panic(err)
	}
	return layout.NewEngine(context.Background(), c)
})

// Font is a font for NewTextEngine: the bytes of a TrueType or OpenType file under a
// family name (the name inside the font when Family is empty).
type Font struct {
	Family string
	Data   []byte
}

// NewTextEngine makes a text engine of its own, with the bundled fonts and the given
// ones and no system font, for a test that must not change the shared engine of
// BundledTextEngine. It fails with the tags of text.Collection.AddFont.
func NewTextEngine(extra ...Font) (*layout.Engine, error) {
	c := text.NewCollection(text.CollectionOpts{Matcher: noSystemFonts{}})
	if err := fonts.Register(c); err != nil {
		return nil, err
	}
	e := layout.NewEngine(context.Background(), c)
	for _, f := range extra {
		if err := e.LoadFontFromList(f.Data, f.Family); err != nil {
			return nil, err
		}
	}
	return e, nil
}

// noSystemFonts is a font matcher that finds nothing.
type noSystemFonts struct{}

func (noSystemFonts) Match(context.Context, string, int, bool, string) ([]text.FontFile, error) {
	return nil, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagRenderingTestNoScene = "RenderingTestNoScene"
)
