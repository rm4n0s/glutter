package engine

import (
	"image"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/shadercache"
	"github.com/rm4n0s/glutter/ui"
)

// RendererEnv is the environment variable that chooses the renderer, as
// Config.Renderer does (SPEC §9.6): "auto" (the default), or the name of a renderer.
// Linux has "vulkan" and "gles"; the web has "webgpu", "webgl" and "soft"; Windows has "d3d12" (from step 28.10) and "soft".
//
// Flutter: no counterpart
const RendererEnv = "GLUTTER_RENDERER"

// rendererAuto asks the engine to try the candidates of the OS in order.
const rendererAuto = "auto"

// knownRenderers are the names of every renderer Glutter has on any OS. A name in
// this list that the OS or the build does not have is RendererUnavailable; any other
// name is RendererUnknown.
var knownRenderers = []string{"vulkan", "gles", "webgpu", "webgl", "d3d12", "soft"}

// rendererCandidate is one renderer an OS can use. The backends_<goos>.go files list
// them in the order that auto tries them.
type rendererCandidate struct {
	// Name is what Config.Renderer and GLUTTER_RENDERER call it.
	Name string
	// New makes the renderer. It must not touch the GPU: the first Draw does, and its
	// error says whether the renderer works.
	New func(cfg *Config, log *slog.Logger) renderer
	// SwitchOn, when set, says whether an error from Draw of this renderer, once it was
	// chosen, is its last resort: the engine then closes it and goes on to the next candidate
	// of the list, as if this one had failed its first frame (SPEC §9.6). WebGPU has it for a
	// device that was lost twice in ten seconds or cannot be had again; Vulkan does not. A
	// renderer asked for by name never switches.
	SwitchOn func(err error) bool
}

// newShaderCache makes the cache of what the GPU drivers compile (SPEC §9.7): in Config.ShaderCacheDir, or the
// directory of the operating system (defaultShaderCacheDir). It is nil when the cache is off (GLUTTER_SHADER_CACHE=0) or there is no place for
// it (the web). A cache that cannot be read or written is logged once and then ignored.
func newShaderCache(cfg *Config, getenv func(string) string, log *slog.Logger) *shadercache.Cache {
	if getenv(shadercache.EnvVar) == "0" {
		return nil
	}
	dir := cfg.ShaderCacheDir
	if dir == "" {
		dir = defaultShaderCacheDir(getenv)
	}
	if dir == "" {
		return nil
	}
	return shadercache.New(dir, func(err error) {
		attrs := []any{"dir", dir, "err", err}
		if appErr, ok := errors.FromError(err); ok {
			attrs = append(attrs, "tag", appErr.Tag)
		}
		log.Warn("the shader cache cannot be used; the shaders are compiled at each start", attrs...)
	})
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
	// chosen is the candidate that cur came from, and after the candidates that follow it,
	// which a switch (rendererCandidate.SwitchOn) goes on with.
	chosen rendererCandidate
	after  []rendererCandidate
	// renew is whether the next try must start with a new native surface: the renderer that
	// is given up has already used the current one.
	renew bool
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
		return nil, errors.New(tagRendererUnknown, "the renderer is not auto, vulkan, gles, webgpu, webgl, d3d12 or soft",
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
	return nil, errors.New(tagRendererUnavailable, "this OS or build does not have that renderer",
		"value", want, "available", strings.Join(have, ","))
}

// Draw draws with the renderer that was chosen. The first call tries the candidates:
// one is kept once it has presented its first frame; if its Draw fails, the error is
// logged once at Info level, the renderer is closed, and the next candidate is tried
// with the same frame. A renderer asked for by name returns its error; with auto the
// error of the last candidate is returned when every one failed.
//
// A renderer that was chosen can still give up for good: when its candidate has a SwitchOn
// that accepts the error of a frame (WebGPU after a second lost device or a device that
// cannot be had again), the renderer is closed and the candidates after it are tried with
// the same frame, on a new native surface (SPEC §9.5, §9.6).
func (c *chooser) Draw(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	if c.cur == nil {
		if len(c.cand) == 0 {
			return nil, errors.New(tagRendererUnavailable, "this OS or build has no renderer")
		}
		return c.try(h, width, height, scene, capture)
	}
	img, err := c.cur.Draw(h, width, height, scene, capture)
	if err == nil || c.named || c.chosen.SwitchOn == nil || !c.chosen.SwitchOn(err) || len(c.after) == 0 {
		return img, err
	}
	attrs := []any{"renderer", c.chosen.Name, "err", err}
	if appErr, ok := errors.FromError(err); ok {
		attrs = append(attrs, "tag", appErr.Tag)
	}
	c.log.Info("renderer given up, switching to the next", attrs...)
	c.cur.Close()
	c.cur, c.cand, c.after, c.renew = nil, c.after, nil, true
	return c.try(h, width, height, scene, capture)
}

// try draws the frame with the first candidate that can.
func (c *chooser) try(h gpu.SurfaceHandle, width, height int, scene *ui.Scene, capture bool) (*image.RGBA, error) {
	var last error
	for tried := 0; len(c.cand) > 0; tried++ {
		cand := c.cand[0]
		if (tried > 0 || c.renew) && c.cfg.renewSurface != nil {
			// What the renderer before this one did to the native surface (the web's canvas
			// keeps the first kind of context it was asked for) must not reach this one.
			c.cfg.renewSurface(h)
		}
		start := time.Now()
		r := cand.New(c.cfg, c.log)
		img, err := r.Draw(h, width, height, scene, capture)
		took := time.Since(start)
		if err == nil {
			c.cur, c.name, c.chosen, c.after, c.cand, c.renew = r, cand.Name, cand, c.cand[1:], nil, false
			c.log.Info("renderer chosen", "renderer", c.Describe())
			c.logStartup(cand.Name, took, r)
			return img, nil
		}
		r.Close()
		if c.frameStats() {
			c.log.Info("startup", "stage", "renderer tried", "renderer", cand.Name, "took", took.Round(time.Millisecond), "err", err)
		}
		if c.named {
			return nil, err
		}
		attrs := []any{"renderer", cand.Name, "err", err}
		if appErr, ok := errors.FromError(err); ok {
			attrs = append(attrs, "tag", appErr.Tag)
		}
		c.log.Info("renderer not usable, trying the next", attrs...)
		last, c.cand = err, c.cand[1:]
	}
	return nil, last
}

// frameStats is whether GLUTTER_FRAME_STATS=1 asks for the parts of the start in the log.
func (c *chooser) frameStats() bool {
	getenv := c.cfg.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	return getenv(FrameStatsEnv) == "1"
}

// logStartup splits the first draw of the renderer that was chosen into the parts the renderer measured: opening
// the GPU, making the shader programs or pipelines (built, or taken from the shader cache), and the rest, which is
// the frame itself.
func (c *chooser) logStartup(name string, took time.Duration, r renderer) {
	if !c.frameStats() {
		return
	}
	round := func(d time.Duration) time.Duration { return d.Round(time.Microsecond * 100) }
	c.log.Info("startup", "stage", "first frame of the renderer", "renderer", name, "took", round(took))
	rep, ok := r.(gpu.StartupReporter)
	if !ok {
		return
	}
	s := rep.StartupReport()
	c.log.Info("startup", "stage", "GPU context or device", "renderer", name, "took", round(s.Context))
	c.log.Info("startup", "stage", "shaders", "renderer", name, "took", round(s.Shaders),
		"built", s.Built, "from_cache", s.Cached, "refused", s.Refused, "pipeline_cache_bytes", s.CacheBytes)
	c.log.Info("startup", "stage", "frame without the above", "renderer", name, "took", round(max(0, took-s.Context-s.Shaders)))
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

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagRendererUnavailable = "RendererUnavailable"
	tagRendererUnknown     = "RendererUnknown"
)
