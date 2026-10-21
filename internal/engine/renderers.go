package engine

import (
	"image"
	"log/slog"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/ui"
)

// RendererEnv is the environment variable that chooses the renderer, as
// Config.Renderer does (SPEC §9.6): "auto" (the default), or the name of a renderer.
// Linux has "vulkan" and "gles"; the web has "webgpu", "webgl" and "soft".
//
// Flutter: no counterpart
const RendererEnv = "GLUTTER_RENDERER"

// rendererAuto asks the engine to try the candidates of the OS in order.
const rendererAuto = "auto"

// knownRenderers are the names of every renderer Glutter has on any OS. A name in
// this list that the OS or the build does not have is RendererUnavailable; any other
// name is RendererUnknown.
var knownRenderers = []string{"vulkan", "gles", "webgpu", "webgl", "soft"}

// rendererCandidate is one renderer an OS can use. The backends_<goos>.go files list
// them in the order that auto tries them.
type rendererCandidate struct {
	// Name is what Config.Renderer and GLUTTER_RENDERER call it.
	Name string
	// New makes the renderer. It must not touch the GPU: the first Draw does, and its
	// error says whether the renderer works.
	New func(cfg *Config, log *slog.Logger) renderer
}

// describer is a renderer that can name itself and its device, for the log: "gles:
// llvmpipe (LLVM 19.1.7, 256 bits)".
type describer interface {
	Describe() string
}

// chooser is the renderer of the engine when no test replaces it: it chooses among
// the candidates at the first frame, for the whole process (SPEC §9.6), and from then
// on draws with the renderer it chose.
type chooser struct {
	cfg  *Config
	log  *slog.Logger
	cand []rendererCandidate // those still to try; the one that works is cur
	// named is whether the renderer was asked for by name, which never falls back.
	named bool
	cur   renderer
	name  string
}

// newChooser checks what the renderer is asked for (want is "" or "auto" for the
// first candidate that works, or a name) against the candidates of the OS. It fails
// with RendererUnknown for a name that is not a renderer of Glutter, and with
// RendererUnavailable for a renderer that this OS or build does not have.
func newChooser(cfg *Config, log *slog.Logger, want string, cands []rendererCandidate) (*chooser, error) {
	c := &chooser{cfg: cfg, log: log, cand: cands}
	if want == "" || want == rendererAuto {
		return c, nil
	}
	known := false
	for _, k := range knownRenderers {
		known = known || k == want
	}
	if !known {
		return nil, errors.New("RendererUnknown", "the renderer is not auto, vulkan, gles, webgpu, webgl or soft",
			"value", want)
	}
	for _, cand := range cands {
		if cand.Name == want {
			c.cand, c.named = []rendererCandidate{cand}, true
			return c, nil
		}
	}
	have := make([]string, len(cands))
	for i, cand := range cands {
		have[i] = cand.Name
	}
	return nil, errors.New("RendererUnavailable", "this OS or build does not have that renderer",
		"value", want, "available", strings.Join(have, ","))
}

// Draw draws with the renderer that was chosen. The first call tries the candidates:
// one is kept once it has presented its first frame; if its Draw fails, the error is
// logged once at Info level, the renderer is closed, and the next candidate is tried
// with the same frame. A renderer asked for by name returns its error; with auto the
// error of the last candidate is returned when every one failed.
func (c *chooser) Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	if c.cur != nil {
		return c.cur.Draw(h, width, height, scene, capture)
	}
	if len(c.cand) == 0 {
		return nil, errors.New("RendererUnavailable", "this OS or build has no renderer")
	}
	var last error
	for tried := 0; len(c.cand) > 0; tried++ {
		cand := c.cand[0]
		if tried > 0 && c.cfg.renewSurface != nil {
			// What the renderer before this one did to the native surface (the web's canvas
			// keeps the first kind of context it was asked for) must not reach this one.
			c.cfg.renewSurface(h)
		}
		r := cand.New(c.cfg, c.log)
		img, err := r.Draw(h, width, height, scene, capture)
		if err == nil {
			c.cur, c.name, c.cand = r, cand.Name, nil
			c.log.Info("renderer chosen", "renderer", c.Describe())
			return img, nil
		}
		r.Close()
		if c.named {
			return nil, err
		}
		attrs := []any{"renderer", cand.Name, "error", err}
		if appErr, ok := errors.FromError(err); ok {
			attrs = append(attrs, "tag", appErr.Tag)
		}
		c.log.Info("renderer not usable, trying the next", attrs...)
		last, c.cand = err, c.cand[1:]
	}
	return nil, last
}

// Describe names the renderer in use and its device, or "" before the first frame.
func (c *chooser) Describe() string {
	if c.cur == nil {
		return ""
	}
	if d, ok := c.cur.(describer); ok {
		return d.Describe()
	}
	return c.name
}

// Release implements renderer.
func (c *chooser) Release(h gpu.SurfaceHandle) {
	if c.cur != nil {
		c.cur.Release(h)
	}
}

// Invalidate implements renderer.
func (c *chooser) Invalidate(h gpu.SurfaceHandle) {
	if c.cur != nil {
		c.cur.Invalidate(h)
	}
}

// Close implements renderer.
func (c *chooser) Close() {
	if c.cur != nil {
		c.cur.Close()
		c.cur = nil
	}
}
