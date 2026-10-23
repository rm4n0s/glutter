package gles

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/internal/gpu"
)

// androidEGL is a fakeEGL that is also a windowPreparer, as the real one is on Android.
type androidEGL struct {
	*fakeEGL
	prepared []uintptr
}

func (a *androidEGL) prepareWindow(_, _, window uintptr) {
	a.call("prepareWindow")
	a.prepared = append(a.prepared, window)
}

var androidHandle = gpu.SurfaceHandle{Kind: gpu.SurfaceKindAndroid, Window: 0xA0}

func TestOpenOnAndroid(t *testing.T) {
	egl := &androidEGL{fakeEGL: newFakeEGL()}
	gl := &fakeGL{versionString: "OpenGL ES 3.2 Adreno"}
	c, _, err := openWith(egl, fakeGLLoader(gl), androidHandle)
	if err != nil {
		t.Fatalf("openWith: %v", err)
	}
	if egl.platform != eglPlatformAndroid || egl.nativeDisplay != 0 || egl.nativeWindow != 0xA0 || egl.surfaceX11 {
		t.Errorf("platform %#x, display %#x, window %#x, x11 %v", egl.platform, egl.nativeDisplay, egl.nativeWindow, egl.surfaceX11)
	}
	// The window gets the format of the config before the surface is made.
	want := []string{"getPlatformDisplay", "initialize", "bindAPI", "chooseConfig", "createContext", "prepareWindow", "createWindowSurface", "makeCurrent", "swapInterval"}
	if !reflect.DeepEqual(egl.calls, want) {
		t.Errorf("EGL calls = %v, want %v", egl.calls, want)
	}
	if len(egl.prepared) != 1 || egl.prepared[0] != 0xA0 {
		t.Errorf("prepared windows %v", egl.prepared)
	}
	c.Close()
}

func TestThePreparationIsForAndroidWindowsOnly(t *testing.T) {
	egl := &androidEGL{fakeEGL: newFakeEGL()}
	if _, _, err := openWith(egl, fakeGLLoader(&fakeGL{versionString: "OpenGL ES 3.2 Mesa"}), waylandHandle); err != nil {
		t.Fatal(err)
	}
	if len(egl.prepared) != 0 {
		t.Errorf("a Wayland window was prepared: %v", egl.prepared)
	}
}

// TestTheContextOutlivesTheSurface checks what Android needs when the activity's surface is
// destroyed and made again (SPEC §24.4): destroying the surface that is current leaves the
// context current with no surface, and the same context, device and textures draw into the new
// surface.
func TestTheContextOutlivesTheSurface(t *testing.T) {
	egl := &androidEGL{fakeEGL: newFakeEGL()}
	c, first, err := openWith(egl, fakeGLLoader(&fakeGL{versionString: "OpenGL ES 3.2 Adreno"}), androidHandle)
	if err != nil {
		t.Fatal(err)
	}
	dev := c.Device()
	egl.calls = nil
	first.Destroy()
	if want := []string{"makeCurrent(surfaceless)", "destroySurface"}; !reflect.DeepEqual(egl.calls, want) {
		t.Errorf("destroying the current surface called %v, want %v", egl.calls, want)
	}
	first.Destroy() // twice is fine
	if len(egl.calls) != 2 {
		t.Errorf("a second Destroy called EGL again: %v", egl.calls)
	}
	next, err := c.NewSurface(gpu.SurfaceHandle{Kind: gpu.SurfaceKindAndroid, Window: 0xB0})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.MakeCurrent(next); err != nil {
		t.Fatal(err)
	}
	if c.Device() != dev {
		t.Error("the device was made again with the surface")
	}
	// A surface that is not the current one is destroyed without touching the context.
	other, err := c.NewSurface(androidHandle)
	if err != nil {
		t.Fatal(err)
	}
	egl.calls = nil
	other.Destroy()
	if want := []string{"destroySurface"}; !reflect.DeepEqual(egl.calls, want) {
		t.Errorf("destroying a surface that is not current called %v, want %v", egl.calls, want)
	}
	c.Close()
}
