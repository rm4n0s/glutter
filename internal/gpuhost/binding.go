package gpuhost

import (
	"context"

	"github.com/rm4n0s/glutter/internal/gpu/explicit"
	"github.com/rm4n0s/glutter/internal/renderhost"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// TB is the part of testing.TB that NewBinding needs.
//
// Flutter: no counterpart
type TB interface {
	Helper()
	Cleanup(func())
}

// App is a widgets binding over a fake platform whose GPU device is the one a test gives: a real Vulkan, Direct3D
// 12 or WebGPU device, for the tests that run package gpu on a GPU.
//
// Flutter: no counterpart
type App struct {
	// Binding is the widgets binding; package gpu reaches the device through it.
	Binding *widgets.WidgetsBinding
	// Host is the device host: it can lose the device and end the app.
	Host *Host

	platform *renderhost.Host
}

type (
	renderHost = renderhost.Host
	deviceHost = Host
)

// appHost is the fake platform with the GPU device.
type appHost struct {
	*renderHost
	*deviceHost
}

// NewBinding makes the app. The goroutine of the caller is its UI thread. The host ends with the test.
//
// Flutter: no counterpart
func NewBinding(tb TB, dev explicit.Device) *App {
	tb.Helper()
	platform := &renderhost.Host{DiscardScenes: true}
	h := New(dev)
	d := ui.NewPlatformDispatcher(&appHost{platform, h})
	ctx, cancel := context.WithCancel(context.Background())
	tb.Cleanup(cancel)
	b := widgets.NewWidgetsBinding(ctx, d, widgets.WidgetsBindingOpts{})
	b.UIThread().Adopt()
	tb.Cleanup(h.End)
	return &App{Binding: b, Host: h, platform: platform}
}

// RunPosted runs the functions that were posted to the UI thread, such as the done function of a read.
//
// Flutter: no counterpart
func (a *App) RunPosted() { a.platform.RunPosted() }
