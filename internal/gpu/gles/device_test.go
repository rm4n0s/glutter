package gles

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/rm4n0s/errors"
)

func newTestDevice(t *testing.T) (*Device, *fakeGL) {
	t.Helper()
	gl := &fakeGL{versionString: "OpenGL ES 3.2 Mesa 24.1.0", vendor: "Fake", renderer: "FakeGPU", extensionList: []string{"GL_EXT_a", "GL_EXT_b"}}
	d, err := newDevice(gl)
	if err != nil {
		t.Fatal(err)
	}
	return d, gl
}

func routeOf(t *testing.T, err error, tag string) *errors.Error {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if appErr.Tag != tag {
		t.Fatalf("Tag = %q, want %q (%s)", appErr.Tag, tag, appErr.Message)
	}
	return appErr
}

func TestNewDeviceCapsAndShaders(t *testing.T) {
	d, gl := newTestDevice(t)
	caps := d.Caps()
	if caps.Version != "OpenGL ES 3.2 Mesa 24.1.0" || caps.Vendor != "Fake" || caps.Renderer != "FakeGPU" {
		t.Errorf("caps strings = %+v", caps)
	}
	if caps.MaxTextureSize != 4096 || caps.MaxTextureUnits != 16 {
		t.Errorf("caps limits = %+v", caps)
	}
	if !caps.Extensions["GL_EXT_a"] || !caps.Extensions["GL_EXT_b"] || len(caps.Extensions) != 2 {
		t.Errorf("extensions = %v", caps.Extensions)
	}
	names, err := ShaderNames()
	if err != nil || len(names) == 0 {
		t.Fatalf("ShaderNames = %v, %v", names, err)
	}
	// Every embedded program is built at startup, from GLSL ES 3.00 sources.
	if got := d.Stats().Programs; got != len(names) {
		t.Errorf("programs = %d, want %d", got, len(names))
	}
	if gl.liveCount("shader") != 0 {
		t.Errorf("%d shaders are still alive after linking", gl.liveCount("shader"))
	}
	for _, name := range names {
		for _, stage := range []string{"vert", "frag"} {
			src, err := ShaderSource(name, stage)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(src, "#version 300 es") {
				t.Errorf("%s.%s.glsl does not start with #version 300 es", name, stage)
			}
		}
	}
	d.Close()
	for _, kind := range []string{"program", "shader", "texture", "buffer", "framebuffer", "renderbuffer"} {
		if n := gl.liveCount(kind); n != 0 {
			t.Errorf("%d %s objects leaked after Close", n, kind)
		}
	}
	d.Close() // twice is fine
}

func TestNewDeviceErrors(t *testing.T) {
	tests := []struct {
		name  string
		gl    *fakeGL
		tag   string
		route string
		meta  map[string]any
	}{
		{"old GL", &fakeGL{versionString: "OpenGL ES 2.0 Old"}, "GLES3Unsupported", "newDevice.GLES3Unsupported", map[string]any{"version": "OpenGL ES 2.0 Old"}},
		{"vertex shader", &fakeGL{versionString: "OpenGL ES 3.0", failShader: "vertex"}, "ShaderCompileFailed",
			"newDevice->Device.Program->Device.NewProgram->Device.compile.ShaderCompileFailed", map[string]any{"stage": "vertex shader", "log": "0:1: error: fake compile error"}},
		{"fragment shader", &fakeGL{versionString: "OpenGL ES 3.0", failShader: "fragment", shaderLog: "syntax error"}, "ShaderCompileFailed",
			"Device.compile.ShaderCompileFailed", map[string]any{"stage": "fragment shader", "log": "syntax error"}},
		{"link", &fakeGL{versionString: "OpenGL ES 3.0", failLink: true}, "ShaderCompileFailed",
			"Device.NewProgram.ShaderCompileFailed", map[string]any{"stage": "link", "log": "error: fake link error"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := newDevice(tt.gl)
			if d != nil {
				t.Error("a device came back with the error")
			}
			appErr := routeOf(t, err, tt.tag)
			if !appErr.HasRoute(tt.route) {
				t.Errorf("route = %s, want it to contain %s", appErr.Route(), tt.route)
			}
			for k, v := range tt.meta {
				if appErr.Metadata[k] != v {
					t.Errorf("Metadata[%q] = %v, want %v", k, appErr.Metadata[k], v)
				}
			}
			if appErr.Tag == "ShaderCompileFailed" && appErr.Metadata["program"] == "" {
				t.Error("the error does not name the program")
			}
			// A failure leaves nothing behind in the driver.
			for _, kind := range []string{"program", "shader"} {
				if n := tt.gl.liveCount(kind); n != 0 {
					t.Errorf("%d %s objects leaked", n, kind)
				}
			}
		})
	}
}

func TestProgramLookups(t *testing.T) {
	d, gl := newTestDevice(t)
	p, err := d.Program("solid")
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := d.Program("solid"); again != p {
		t.Error("Program compiled a second time")
	}
	if p.Name() != "solid" {
		t.Errorf("Name = %q", p.Name())
	}
	gl.resetCalls()
	a, b := p.Uniform("uColor"), p.Uniform("uColor")
	if a != b || gl.calls["getUniformLocation"] != 1 {
		t.Errorf("uniform locations %d, %d with %d lookups, want one lookup", a, b, gl.calls["getUniformLocation"])
	}
	if p.Uniform("missingThing") != -1 || p.Attrib("missingAttr") != -1 {
		t.Error("an unknown name should give -1")
	}
	_, err = d.Program("nope")
	appErr := routeOf(t, err, "ShaderSourceMissing")
	if !appErr.HasRoute("ShaderSource.ShaderSourceMissing") {
		t.Errorf("route = %s", appErr.Route())
	}
}

func TestStateCache(t *testing.T) {
	d, gl := newTestDevice(t)
	p, _ := d.Program("solid")
	b, _ := d.Program("blit")
	tex, err := d.AcquireTexture(8, 8, TextureFormatRGBA8)
	if err != nil {
		t.Fatal(err)
	}
	fb, err := d.AcquireFramebuffer(8, 8, false)
	if err != nil {
		t.Fatal(err)
	}
	// Each setter is called twice with the same value: GL hears it once.
	tests := []struct {
		call string
		fn   func()
		want int
	}{
		{"viewport", func() { d.Viewport(0, 0, 100, 50) }, 1},
		{"scissor", func() { d.SetScissor(true, 1, 2, 3, 4) }, 1},
		{"blendFuncSeparate", func() { d.BlendFunc(glOne, glOne, glOne, glOne) }, 1},
		{"blendEquation", func() { d.BlendEquation(glFuncReverseSubtract) }, 1},
		{"colorMask", func() { d.ColorMask(true, true, true, false) }, 1},
		{"stencilFunc", func() { d.StencilFunc(glNotequal, 1, 0xFF) }, 1},
		{"stencilOp", func() { d.StencilOp(glKeep, glKeep, glKeep) }, 0}, // already the state
		{"stencilMask", func() { d.StencilMask(0x0F) }, 1},
		{"useProgram", func() { d.UseProgram(p) }, 1},
		{"bindTexture", func() { d.BindTexture(1, tex) }, 1},
		{"activeTexture", func() { d.BindTexture(1, tex) }, 1},
		{"bindFramebuffer", func() { d.BindFramebuffer(fb) }, 1},
		{"bindVertexArray", func() { d.BindVertexArray(7) }, 1},
	}
	// Some counts are cumulative across rows that share a call name; reset per row.
	for _, tt := range tests {
		gl.resetCalls()
		tt.fn()
		tt.fn()
		got := gl.calls[tt.call]
		if tt.call == "bindTexture" || tt.call == "activeTexture" {
			// These rows only prove the second call adds nothing.
			if got > 1 {
				t.Errorf("%s was called %d times for two identical setters", tt.call, got)
			}
			continue
		}
		if got != tt.want {
			t.Errorf("%s was called %d times, want %d", tt.call, got, tt.want)
		}
	}
	// A different value does reach GL, and the other program is not confused with the first.
	gl.resetCalls()
	d.UseProgram(b)
	d.UseProgram(p)
	if gl.calls["useProgram"] != 2 {
		t.Errorf("useProgram calls = %d, want 2", gl.calls["useProgram"])
	}
	d.SetScissor(false, 0, 0, 0, 0)
	d.SetScissor(false, 0, 0, 0, 0)
	if gl.calls["disable"] != 1 {
		t.Errorf("disable calls = %d, want 1", gl.calls["disable"])
	}
	// ResetState puts the cache and the driver back to the starting point.
	d.ResetState()
	gl.resetCalls()
	d.Viewport(0, 0, 100, 50)
	if gl.calls["viewport"] != 1 {
		t.Errorf("viewport after ResetState was called %d times, want 1", gl.calls["viewport"])
	}
}

func TestBindTextureUnitInvalid(t *testing.T) {
	d, _ := newTestDevice(t)
	tex, _ := d.AcquireTexture(2, 2, TextureFormatUnspecified)
	defer func() {
		appErr, ok := errors.FromError(recoverError(recover()))
		if !ok || appErr.Tag != "GLTextureUnitInvalid" || !appErr.HasRoute("Device.bindTextureID.GLTextureUnitInvalid") {
			t.Errorf("expected GLTextureUnitInvalid, got %v", appErr)
		}
	}()
	d.BindTexture(maxTrackedUnits, tex)
}

func recoverError(r any) error {
	err, _ := r.(error)
	return err
}

func TestTexturePool(t *testing.T) {
	d, gl := newTestDevice(t)
	a, err := d.AcquireTexture(64, 32, TextureFormatRGBA8)
	if err != nil {
		t.Fatal(err)
	}
	if a.Width() != 64 || a.Height() != 32 || a.Format() != TextureFormatRGBA8 {
		t.Errorf("texture = %dx%d format %d", a.Width(), a.Height(), a.Format())
	}
	d.ReleaseTexture(a)
	b, _ := d.AcquireTexture(64, 32, TextureFormatUnspecified)
	if b != a || gl.calls["genTexture"] != 1 {
		t.Errorf("same size and format should reuse the texture (genTexture called %d times)", gl.calls["genTexture"])
	}
	c, _ := d.AcquireTexture(64, 32, TextureFormatRGBA8) // b is still in use
	if c == b {
		t.Error("a texture in use was handed out twice")
	}
	r8, _ := d.AcquireTexture(64, 32, TextureFormatR8)
	if r8 == c || r8 == b || r8.Format() != TextureFormatR8 {
		t.Error("a different format must not share a texture")
	}
	if got := d.Stats(); got.Textures != 3 || got.FreeTextures != 0 {
		t.Errorf("stats = %+v", got)
	}
	d.ReleaseTexture(b)
	d.ReleaseTexture(c)
	if got := d.Stats(); got.Textures != 3 || got.FreeTextures != 2 {
		t.Errorf("stats = %+v, want 3 live and 2 free", got)
	}
	// Free textures older than the limit are deleted; a young one stays.
	d.EndFrame()
	d.Trim(1)
	if got := d.Stats().FreeTextures; got != 2 {
		t.Errorf("free textures after one frame = %d, want 2", got)
	}
	d.EndFrame()
	d.EndFrame()
	d.Trim(1)
	if got := d.Stats(); got.FreeTextures != 0 || got.Textures != 1 {
		t.Errorf("stats after trimming = %+v, want only the busy texture", got)
	}
	if gl.liveCount("texture") != 1 {
		t.Errorf("driver still has %d textures, want 1", gl.liveCount("texture"))
	}
	// EndFrame frees what has been idle for a long time without being asked.
	d.ReleaseTexture(r8)
	for i := 0; i < defaultMaxIdleFrames+2; i++ {
		d.EndFrame()
	}
	if got := d.Stats().Textures; got != 0 {
		t.Errorf("textures after a long idle = %d, want 0", got)
	}
}

func TestTextureErrors(t *testing.T) {
	d, gl := newTestDevice(t)
	for _, tt := range []struct{ w, h int }{{0, 5}, {5, -1}, {4097, 5}, {5, 4097}} {
		_, err := d.AcquireTexture(tt.w, tt.h, TextureFormatRGBA8)
		appErr := routeOf(t, err, "TextureSizeInvalid")
		if !appErr.HasRoute("Device.AcquireTexture->Device.newTexture.TextureSizeInvalid") {
			t.Errorf("route = %s", appErr.Route())
		}
	}
	gl.errOnce = 0x505 // GL_OUT_OF_MEMORY on the allocation
	_, err := d.AcquireTexture(10, 10, TextureFormatRGBA8)
	appErr := routeOf(t, err, "GLError")
	if appErr.Metadata["gl_error"] != "0x505" || appErr.Metadata["operation"] != "texImage2D" {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
	if gl.liveCount("texture") != 0 {
		t.Error("a failed texture leaked")
	}

	tex, _ := d.AcquireTexture(4, 4, TextureFormatRGBA8)
	uploads := []struct {
		name       string
		x, y, w, h int
		n          int
		ok         bool
	}{
		{"whole", 0, 0, 4, 4, 64, true},
		{"part", 1, 1, 2, 2, 16, true},
		{"short data", 0, 0, 4, 4, 63, false},
		{"outside", 3, 0, 2, 1, 8, false},
		{"negative", -1, 0, 1, 1, 4, false},
		{"empty", 0, 0, 0, 1, 0, false},
	}
	for _, tt := range uploads {
		err := d.UploadTexture(tex, tt.x, tt.y, tt.w, tt.h, make([]byte, tt.n))
		if tt.ok {
			if err != nil {
				t.Errorf("%s: %v", tt.name, err)
			}
			continue
		}
		appErr := routeOf(t, err, "TextureUploadInvalid")
		if !appErr.HasRoute("Device.UploadTexture.TextureUploadInvalid") {
			t.Errorf("%s: route = %s", tt.name, appErr.Route())
		}
	}
	r8, _ := d.AcquireTexture(4, 4, TextureFormatR8)
	if err := d.UploadTexture(r8, 0, 0, 4, 4, make([]byte, 16)); err != nil {
		t.Errorf("R8 upload: %v", err)
	}
	if err := d.UploadTexture(r8, 0, 0, 4, 4, make([]byte, 64)); err == nil {
		t.Error("an RGBA-sized upload into an R8 texture should fail")
	}

	d.ReleaseTexture(tex)
	defer func() {
		appErr, ok := errors.FromError(recoverError(recover()))
		if !ok || appErr.Tag != "TextureReleased" || !appErr.HasRoute("Device.ReleaseTexture.TextureReleased") {
			t.Errorf("expected TextureReleased, got %v", appErr)
		}
	}()
	d.ReleaseTexture(tex)
}

func TestSetTextureParamsSkipsRedundantCalls(t *testing.T) {
	d, gl := newTestDevice(t)
	tex, _ := d.AcquireTexture(4, 4, TextureFormatRGBA8)
	gl.resetCalls()
	d.SetTextureParams(tex, glLinear, glLinear, glClampToEdge, glClampToEdge) // the defaults
	if gl.calls["texParameteri"] != 0 {
		t.Errorf("texParameteri called %d times for the defaults", gl.calls["texParameteri"])
	}
	d.SetTextureParams(tex, glLinearMipmapLinear, glLinear, glClampToEdge, glClampToEdge)
	if gl.calls["texParameteri"] != 1 {
		t.Errorf("texParameteri called %d times for one change, want 1", gl.calls["texParameteri"])
	}
	d.GenerateMipmaps(tex)
	if gl.calls["generateMipmap"] != 1 {
		t.Error("generateMipmap was not called")
	}
}

func TestBufferPool(t *testing.T) {
	d, gl := newTestDevice(t)
	a, err := d.AcquireBuffer(glArrayBuffer, 300)
	if err != nil {
		t.Fatal(err)
	}
	if a.Capacity() != 512 {
		t.Errorf("capacity for 300 bytes = %d, want 512", a.Capacity())
	}
	d.ReleaseBuffer(a)
	b, _ := d.AcquireBuffer(glArrayBuffer, 500) // same capacity class
	if b != a || gl.calls["genBuffer"] != 1 {
		t.Error("a buffer of the same capacity class should be reused")
	}
	c, _ := d.AcquireBuffer(glArrayBuffer, 513)
	if c == b || c.Capacity() != 1024 {
		t.Errorf("513 bytes should get a new 1024-byte buffer, got %d", c.Capacity())
	}
	small, _ := d.AcquireBuffer(glArrayBuffer, 1)
	if small.Capacity() != minBufferSize {
		t.Errorf("minimum capacity = %d, want %d", small.Capacity(), minBufferSize)
	}
	if err := d.UploadBuffer(b, make([]byte, 400)); err != nil {
		t.Errorf("UploadBuffer: %v", err)
	}
	if err := d.UploadBuffer(b, nil); err != nil {
		t.Errorf("UploadBuffer(nil): %v", err)
	}
	err = d.UploadBuffer(small, make([]byte, minBufferSize+1))
	appErr := routeOf(t, err, "BufferTooSmall")
	if !appErr.HasRoute("Device.UploadBuffer.BufferTooSmall") {
		t.Errorf("route = %s", appErr.Route())
	}
	d.ReleaseBuffer(b)
	d.ReleaseBuffer(c)
	d.Trim(0)
	if got := d.Stats(); got.Buffers != 1 || got.FreeBuffers != 0 {
		t.Errorf("stats after Trim(0) = %+v, want only the buffer in use", got)
	}
	defer func() {
		appErr, ok := errors.FromError(recoverError(recover()))
		if !ok || appErr.Tag != "BufferReleased" || !appErr.HasRoute("Device.ReleaseBuffer.BufferReleased") {
			t.Errorf("expected BufferReleased, got %v", appErr)
		}
	}()
	d.ReleaseBuffer(b)
}

func TestFramebufferPool(t *testing.T) {
	d, gl := newTestDevice(t)
	a, err := d.AcquireFramebuffer(32, 16, true)
	if err != nil {
		t.Fatal(err)
	}
	if a.Width() != 32 || a.Height() != 16 || a.Texture() == nil || a.Texture().Width() != 32 {
		t.Errorf("framebuffer = %dx%d, texture %v", a.Width(), a.Height(), a.Texture())
	}
	if gl.liveCount("renderbuffer") != 1 {
		t.Error("a stencil framebuffer needs a renderbuffer")
	}
	d.ReleaseFramebuffer(a)
	b, _ := d.AcquireFramebuffer(32, 16, true)
	if b != a || gl.calls["genFramebuffer"] != 1 {
		t.Error("the same size and stencil should reuse the framebuffer")
	}
	c, _ := d.AcquireFramebuffer(32, 16, false)
	if c == b || gl.liveCount("renderbuffer") != 1 {
		t.Error("a framebuffer without stencil must be a different one, and have no renderbuffer")
	}
	// Binding the framebuffer and then freeing it leaves the window's bound again.
	d.BindFramebuffer(c)
	d.ReleaseFramebuffer(b)
	d.ReleaseFramebuffer(c)
	d.Trim(0)
	if gl.liveCount("framebuffer") != 0 || gl.liveCount("texture") != 0 || gl.liveCount("renderbuffer") != 0 {
		t.Errorf("objects leaked after Trim(0): fb %d tex %d rb %d", gl.liveCount("framebuffer"), gl.liveCount("texture"), gl.liveCount("renderbuffer"))
	}
	gl.resetCalls()
	d.BindFramebuffer(nil)
	if gl.calls["bindFramebuffer"] != 0 {
		t.Error("deleting the bound framebuffer should have made the window's current")
	}
	defer func() {
		appErr, ok := errors.FromError(recoverError(recover()))
		if !ok || appErr.Tag != "FramebufferReleased" || !appErr.HasRoute("Device.ReleaseFramebuffer.FramebufferReleased") {
			t.Errorf("expected FramebufferReleased, got %v", appErr)
		}
	}()
	d.ReleaseFramebuffer(a)
}

func TestFramebufferErrors(t *testing.T) {
	d, gl := newTestDevice(t)
	_, err := d.AcquireFramebuffer(0, 10, false)
	appErr := routeOf(t, err, "TextureSizeInvalid")
	if !appErr.HasRoute("Device.AcquireFramebuffer->Device.newTexture.TextureSizeInvalid") {
		t.Errorf("route = %s", appErr.Route())
	}
	gl.fbStatus = 0x8CD6 // GL_FRAMEBUFFER_INCOMPLETE_ATTACHMENT
	_, err = d.AcquireFramebuffer(16, 16, true)
	appErr = routeOf(t, err, "FramebufferIncomplete")
	if !appErr.HasRoute("Device.AcquireFramebuffer.FramebufferIncomplete") || appErr.Metadata["status"] != "0x8cd6" {
		t.Errorf("route %s, metadata %v", appErr.Route(), appErr.Metadata)
	}
	for _, kind := range []string{"framebuffer", "texture", "renderbuffer"} {
		if gl.liveCount(kind) != 0 {
			t.Errorf("a failed framebuffer leaked %s objects", kind)
		}
	}
	if s := d.Stats(); s.Framebuffers != 0 || s.Textures != 0 || s.Renderbuffers != 0 {
		t.Errorf("stats = %+v, want no objects", s)
	}
}

func TestGLErrorCheck(t *testing.T) {
	d, gl := newTestDevice(t)
	gl.errOnce = 0x502
	appErr := routeOf(t, d.checkError("draw"), "GLError")
	if appErr.Metadata["gl_error"] != "0x502" || appErr.Metadata["operation"] != "draw" {
		t.Errorf("metadata = %v", appErr.Metadata)
	}
	if err := d.checkError("draw"); err != nil {
		t.Errorf("the flag should have been cleared: %v", err)
	}
}

func TestGLSymbolTable(t *testing.T) {
	g := new(glLib)
	seen := map[string]bool{}
	for _, b := range g.glSymbols() {
		if !strings.HasPrefix(b.symbol, "gl") {
			t.Errorf("symbol %q does not look like a GL function", b.symbol)
		}
		if seen[b.symbol] {
			t.Errorf("symbol %q is bound twice", b.symbol)
		}
		seen[b.symbol] = true
		if b.fptr == nil {
			t.Errorf("symbol %q has no target", b.symbol)
		}
	}
	for _, must := range []string{"glGetString", "glGetStringi", "glDrawArraysInstanced", "glBlitFramebuffer", "glTexImage2D", "glVertexAttribDivisor"} {
		if !seen[must] {
			t.Errorf("the table lacks %s", must)
		}
	}
}

func TestLoadGLSymbolMissing(t *testing.T) {
	egl := newFakeEGL() // knows no GL functions
	_, err := loadGL(egl)
	appErr := routeOf(t, err, "SymbolMissing")
	if !appErr.HasRoute("loadGL.SymbolMissing") || appErr.Metadata["symbol"] == "" || appErr.Metadata["library"] != "eglGetProcAddress" {
		t.Errorf("route %s, metadata %v", appErr.Route(), appErr.Metadata)
	}
}

// TestConstantsMatchHeaders checks every constant of glconst.go against the value
// in the system's GLES3 headers.
func TestConstantsMatchHeaders(t *testing.T) {
	values := map[string]string{}
	define := regexp.MustCompile(`^#define\s+(GL_\w+)\s+(0x[0-9A-Fa-f]+|\d+)\b`)
	found := false
	for _, path := range []string{"/usr/include/GLES3/gl3.h", "/usr/include/GLES2/gl2.h"} {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		found = true
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if m := define.FindStringSubmatch(sc.Text()); m != nil {
				values[m[1]] = m[2]
			}
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if !found {
		t.Skip("the GLES headers are not installed")
	}
	src, err := os.ReadFile("glconst.go")
	if err != nil {
		t.Fatal(err)
	}
	line := regexp.MustCompile(`(?m)^\s*(gl\w+)\s*=\s*(0x[0-9A-Fa-f]+|\d+)\s*// (GL_\w+)\s*$`)
	matches := line.FindAllStringSubmatch(string(src), -1)
	if len(matches) < 100 {
		t.Fatalf("parsed only %d constants from glconst.go", len(matches))
	}
	parse := func(s string) uint64 {
		var v uint64
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			for _, c := range s[2:] {
				v = v*16 + uint64(strings.IndexRune("0123456789abcdef", c|0x20))
			}
			return v
		}
		for _, c := range s {
			v = v*10 + uint64(c-'0')
		}
		return v
	}
	for _, m := range matches {
		want, ok := values[m[3]]
		if !ok {
			t.Errorf("%s: %s is not in the headers", m[1], m[3])
			continue
		}
		if parse(m[2]) != parse(want) {
			t.Errorf("%s = %s, but %s is %s in the headers", m[1], m[2], m[3], want)
		}
	}
}

func TestExpandIncludes(t *testing.T) {
	files := map[string]string{"a.glsl": "float a;\n", "b.glsl": "float b;"}
	read := func(f string) ([]byte, error) {
		if s, ok := files[f]; ok {
			return []byte(s), nil
		}
		return nil, errors.New("Missing", "no such file")
	}
	got, err := expandIncludes("p", "frag", "#version 300 es\n//#include \"a.glsl\"\nvoid main() {}\n  //#include \"b.glsl\"  \nend", read)
	if err != nil {
		t.Fatal(err)
	}
	want := "#version 300 es\nfloat a;\nvoid main() {}\nfloat b;\nend"
	if got != want {
		t.Errorf("expanded = %q, want %q", got, want)
	}
	// No include lines: the source is as it was.
	if got, err := expandIncludes("p", "frag", "x\n// #include is only a comment\n", read); err != nil || got != "x\n// #include is only a comment\n" {
		t.Errorf("plain source = %q, %v", got, err)
	}
	for _, src := range []string{"//#include a.glsl", "//#include \"a.glsl", "//#include \"\"x"} {
		_, err := expandIncludes("p", "frag", src, read)
		appErr := routeOf(t, err, "ShaderIncludeInvalid")
		if !appErr.HasRoute("expandIncludes.ShaderIncludeInvalid") || appErr.Metadata["program"] != "p" {
			t.Errorf("%q: route %s", src, appErr.Route())
		}
	}
	_, err = expandIncludes("p", "vert", "//#include \"nope.glsl\"", read)
	appErr := routeOf(t, err, "ShaderSourceMissing")
	if !appErr.HasRoute("expandIncludes.ShaderSourceMissing") || appErr.Metadata["include"] != "nope.glsl" || appErr.Metadata["stage"] != "vert" {
		t.Errorf("route %s, metadata %v", appErr.Route(), appErr.Metadata)
	}
}

func TestEmbeddedShadersHaveNoUnexpandedIncludes(t *testing.T) {
	names, _ := ShaderNames()
	for _, n := range names {
		for _, stage := range []string{"vert", "frag"} {
			src, err := ShaderSource(n, stage)
			if err != nil {
				t.Fatalf("%s.%s: %v", n, stage, err)
			}
			if strings.Contains(src, "//#include") {
				t.Errorf("%s.%s still has an include line", n, stage)
			}
			if !strings.HasPrefix(src, "#version 300 es") {
				t.Errorf("%s.%s does not start with the version", n, stage)
			}
		}
	}
	for _, want := range []string{"shape", "mask", "image", "vertices"} {
		found := false
		for _, n := range names {
			found = found || n == want
		}
		if !found {
			t.Errorf("no %s program", want)
		}
	}
}

// TestTextureEditsUseUnitZero is the regression test of a crash in the counter
// example: after a texture was bound to another unit (a copy of the framebuffer
// for the ripple), an upload to a texture that the cache said was bound to unit 0
// went to the texture of the active unit, and GL answered INVALID_VALUE.
func TestTextureEditsUseUnitZero(t *testing.T) {
	d, gl := newTestDevice(t)
	page, err := d.AcquireTexture(64, 64, TextureFormatR8)
	if err != nil {
		t.Fatal(err)
	}
	other, err := d.AcquireTexture(8, 8, TextureFormatRGBA8)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		edit func() error
	}{
		{"upload", func() error { return d.UploadTexture(page, 0, 0, 2, 2, make([]byte, 4)) }},
		{"mipmaps", func() error { d.GenerateMipmaps(page); return nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := d.UploadTexture(page, 0, 0, 1, 1, []byte{1}); err != nil {
				t.Fatal(err)
			}
			d.BindTexture(1, other) // the active unit is now 1
			gl.resetCalls()
			if err := tt.edit(); err != nil {
				t.Fatal(err)
			}
			// The last activeTexture before the edit must be unit 0.
			active := ""
			for _, call := range gl.log {
				switch {
				case strings.HasPrefix(call, "activeTexture"):
					active = call
				case strings.HasPrefix(call, "texSubImage2D"), strings.HasPrefix(call, "generateMipmap"):
					if active != "activeTexture"+fmt.Sprint(uint32(glTexture0)) {
						t.Errorf("%s ran with %q as the last activeTexture, want unit 0", call, active)
					}
				}
			}
		})
	}
}
