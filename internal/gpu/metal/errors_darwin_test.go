//go:build darwin && !ios

package metal

import (
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/gpu/explicit/devicetest"
)

// routeOf checks that err is an error of this module with the route.
func wantRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("got route %s, want %s", appErr.Route(), route)
	}
}

// mustPanic runs fn and returns the error it panicked with.
func mustPanic(t *testing.T, fn func()) (err error) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Error("the call did not panic")
			return
		}
		e, ok := r.(error)
		if !ok {
			t.Errorf("the call panicked with %v, not an error", r)
			return
		}
		err = e
	}()
	fn()
	return nil
}

func TestOpenWithoutADeviceIsUnavailable(t *testing.T) {
	_, err := Open(Options{noDevice: true})
	wantRoute(t, err, "Open.MetalDeviceUnavailable")
}

func TestErrorForCode(t *testing.T) {
	tests := []struct {
		code  int64
		route string
	}{
		{cbErrorOutOfMemory, "errorForCode.GPUOutOfMemory"},
		{cbErrorDeviceRemoved, "errorForCode.GPUDeviceLost"},
		{cbErrorTimeout, "errorForCode.GPUDeviceLost"},
		{cbErrorPageFault, "errorForCode.GPUDeviceLost"},
		{cbErrorAccessRevoked, "errorForCode.GPUDeviceLost"},
		{1, "errorForCode.MetalCallFailed"},
	}
	for _, tt := range tests {
		wantRoute(t, errorForCode(tt.code, "x"), tt.route)
	}
}

func TestShaderThatMetalRefuses(t *testing.T) {
	d := openTestDevice(t, Options{})
	_, err := d.library(&explicit.Shader{Name: "bad", MSL: "this is not MSL"})
	wantRoute(t, err, "Device.library.MetalShaderCompileFailed")
	_, err = d.library(&explicit.Shader{Name: "empty"})
	wantRoute(t, err, "Device.library.MetalShaderCompileFailed")
}

func TestPipelineErrors(t *testing.T) {
	d := openTestDevice(t, Options{})
	sh := devicetest.ShaderFill
	// A fragment function that the compiled library does not have.
	lib, err := d.library(sh)
	if err != nil {
		t.Fatal(err)
	}
	_, err = lib.function("nothing")
	wantRoute(t, err, "library.function.MetalPipelineFailed")
}

func TestBeginTwiceIsAnError(t *testing.T) {
	d := openTestDevice(t, Options{})
	enc, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Begin()
	wantRoute(t, err, "Device.Begin.MetalFrameOpen")
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
}

func TestSwapchainOfAnotherSurface(t *testing.T) {
	d := openTestDevice(t, Options{})
	_, err := d.NewSwapchain(gpu.SurfaceHandle{Kind: gpu.SurfaceKindX11, Window: 1})
	wantRoute(t, err, "Device.NewSwapchain->surfaceLayer.MetalSwapchainFailed")
}

func TestForeignResourcesPanic(t *testing.T) {
	d1 := openTestDevice(t, Options{})
	d2 := openTestDevice(t, Options{})
	b, err := d2.NewBuffer(16, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Destroy()
	err = mustPanic(t, func() { d1.WriteBuffer(b, 0, []byte{1}) })
	wantRoute(t, err, "Device.ownBuffer.MetalForeignResource")
	tex, err := d2.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureSampled})
	if err != nil {
		t.Fatal(err)
	}
	defer tex.Destroy()
	err = mustPanic(t, func() { d1.WriteTexture(tex, 0, imageRect(0, 0, 1, 1), []byte{1, 2, 3, 4}, 4) })
	wantRoute(t, err, "Device.ownTexture.MetalForeignResource")
}

func TestWriteOutsideABufferPanics(t *testing.T) {
	d := openTestDevice(t, Options{})
	b, err := d.NewBuffer(16, explicit.BufferUniform)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Destroy()
	err = mustPanic(t, func() { d.WriteBuffer(b, 8, make([]byte, 16)) })
	wantRoute(t, err, "Device.WriteBuffer.MetalUsage")
}

func TestUsageErrorsOfTheEncoder(t *testing.T) {
	d := openTestDevice(t, Options{})
	enc, err := d.Begin()
	if err != nil {
		t.Fatal(err)
	}
	tex, err := d.NewTexture(explicit.TextureDesc{Width: 4, Height: 4, Format: explicit.TextureFormatRGBA8Unorm, Usage: explicit.TextureRenderTarget | explicit.TextureCopySource})
	if err != nil {
		t.Fatal(err)
	}
	defer tex.Destroy()
	pass := enc.BeginRenderPass(explicit.PassDesc{Color: []explicit.ColorAttachment{{Target: tex}}})
	err = mustPanic(t, func() { pass.Draw(3, 1, 0, 0) })
	wantRoute(t, err, "pass.requirePipeline.MetalUsage")
	err = mustPanic(t, func() { enc.CopyTexture(tex, imageRect(0, 0, 1, 1), tex, imagePoint(1, 1)) })
	wantRoute(t, err, "encoder.blit.MetalUsage")
	pass.End()
	err = mustPanic(t, func() { enc.CopyTexture(tex, imageRect(0, 0, 9, 9), tex, imagePoint(0, 0)) })
	wantRoute(t, err, "encoder.CopyTexture.MetalUsage")
	if err := enc.Submit(); err != nil {
		t.Fatal(err)
	}
}

func TestQuerySetsAreUnsupported(t *testing.T) {
	d := openTestDevice(t, Options{})
	_, err := d.NewQuerySet(2)
	wantRoute(t, err, "Device.NewQuerySet.GPUQueryUnsupported")
}
