// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package widgets

import (
	"context"
	"io/fs"
	"log/slog"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/engine"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/text"
	"github.com/rm4n0s/glutter/internal/text/layout"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"

	textfontconfig "github.com/rm4n0s/glutter/internal/text/fontconfig"
)

// WindowState is the initial state of a window.
//
// Flutter: widgets/_window.dart WindowController
type WindowState int

// The values of WindowState.
const (
	// WindowStateUnspecified is the zero value and means WindowStateNormal.
	//
	// Flutter: no counterpart
	WindowStateUnspecified WindowState = iota
	// WindowStateNormal is a window that is neither maximized, minimized nor
	// fullscreen.
	//
	// Flutter: widgets/_window.dart WindowController.isMaximized
	WindowStateNormal
	// WindowStateMaximized is a maximized window.
	//
	// Flutter: widgets/_window.dart WindowController.isMaximized
	WindowStateMaximized
	// WindowStateMinimized is a minimized window.
	//
	// Flutter: widgets/_window.dart WindowController.isMinimized
	WindowStateMinimized
	// WindowStateFullscreen is a fullscreen window.
	//
	// Flutter: widgets/_window.dart WindowController.isFullscreen
	WindowStateFullscreen
)

// WindowConfig describes the first window of an app (RunAppOpts.Window). The
// zero value is a regular, resizable, decorated window of a size the platform
// chooses.
//
// Flutter: widgets/_window.dart WindowController
type WindowConfig struct {
	// Title is the title of the window; the default is "Glutter".
	Title string
	// Size is the initial size in logical pixels; zero lets the platform choose.
	Size ui.Size
	// MinSize is the smallest size in logical pixels; zero means no minimum.
	MinSize ui.Size
	// MaxSize is the largest size in logical pixels; zero means no maximum.
	MaxSize ui.Size
	// Resizable is whether the user can resize the window; the default is true.
	Resizable foundation.Opt[bool]
	// Decorated is whether the window has a title bar and borders; the default
	// is true.
	Decorated foundation.Opt[bool]
	// State is the initial state of the window.
	State WindowState
}

// RunAppOpts are the optional parameters of RunApp and RunWidget (SPEC §13).
//
// Flutter: widgets/binding.dart runApp
type RunAppOpts struct {
	// Assets is the file system the AssetBundle reads from; without it the
	// bundle finds nothing.
	Assets fs.FS
	// AppID is the application ID: the Wayland app_id and the X11 WM_CLASS. It
	// must match the app's .desktop file.
	AppID string
	// Window is the config of the first window.
	Window WindowConfig
	// Backend, when not empty, is "wayland" or "x11" and overrides
	// GLUTTER_BACKEND.
	Backend string
	// ErrorReporter receives the errors the framework recovers from or reports
	// (SPEC §14.2). The default logs each one, as JSON, through Logger.
	ErrorReporter foundation.ErrorReporter
	// ErrorWidgetBuilder builds the widget that replaces a widget that panicked
	// in Build; the default is a red box with the message in debug builds and a
	// gray box in release builds.
	ErrorWidgetBuilder func(err error) Widget
	// Logger gets the log of the engine and of the default error reporter;
	// slog.Default() when nil.
	Logger *slog.Logger
	// Debug are the debug flags (SPEC §14.3); the default is a fresh set, all
	// off.
	Debug *DebugOptions
}

// RunApp runs the app: it opens the first window, shows the widget in it, and
// runs the event loop until the last window is closed or the app asks to exit.
// It must be called on the main goroutine (SPEC §8.4, §13). The widget is
// wrapped in a View for the window; use RunWidget to provide the View yourself.
//
// The failures of the environment come back as tagged errors (SPEC §14.1):
// NoDisplay, BackendUnknown, the platform's and the renderer's.
//
// Flutter: widgets/binding.dart runApp
func RunApp(app Widget, opts ...RunAppOpts) error {
	return run(app, true, firstOpts(opts))
}

// RunWidget is RunApp that does not wrap the widget in a View: the widget must
// be, or contain, a View.
//
// Flutter: widgets/binding.dart runWidget
func RunWidget(app Widget, opts ...RunAppOpts) error {
	return run(app, false, firstOpts(opts))
}

func run(app Widget, wrap bool, o RunAppOpts) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := engine.Config{
		Backend: o.Backend,
		Window:  o.platformWindow(),
		Logger:  o.Logger,
		Start: func(d *ui.PlatformDispatcher) error {
			d.SetTextEngine(systemTextEngine(ctx, o.Logger))
			_, err := startApp(ctx, d, app, wrap, o)
			return err
		},
	}
	return engine.Run(ctx, cfg)
}

// platformWindow translates the window config of the app into the platform's.
func (o RunAppOpts) platformWindow() platform.WindowConfig {
	w := o.Window
	title := w.Title
	if title == "" {
		title = "Glutter"
	}
	state := platform.WindowStateUnspecified
	switch w.State {
	case WindowStateNormal:
		state = platform.WindowStateNormal
	case WindowStateMaximized:
		state = platform.WindowStateMaximized
	case WindowStateMinimized:
		state = platform.WindowStateMinimized
	case WindowStateFullscreen:
		state = platform.WindowStateFullscreen
	}
	return platform.WindowConfig{Title: title, AppID: o.AppID, Size: w.Size, MinSize: w.MinSize, MaxSize: w.MaxSize,
		Resizable: w.Resizable, Decorated: w.Decorated, State: state}
}

// startApp creates the widgets binding on the dispatcher and attaches the app:
// what RunApp does once the engine has the window.
func startApp(ctx context.Context, d *ui.PlatformDispatcher, app Widget, wrap bool, o RunAppOpts) (*WidgetsBinding, error) {
	po := painting.PaintingBindingOpts{}
	if o.Assets != nil {
		po.AssetBundle = services.NewFSAssetBundle(o.Assets)
	}
	b := NewWidgetsBinding(ctx, d, WidgetsBindingOpts{
		Debug: o.Debug, ErrorReporter: o.ErrorReporter, ErrorWidgetBuilder: o.ErrorWidgetBuilder,
		Logger: o.Logger, Painting: po,
	})
	d.IsIdle = b.IsIdle
	if wrap && d.ImplicitView() == nil {
		return nil, errors.New("ImplicitViewMissing", "the platform did not provide the implicit view that RunApp shows the app in")
	}
	root := app
	if wrap {
		root = b.WrapWithDefaultView(app)
	}
	b.AttachRootWidget(root)
	return b, nil
}

// IsIdle is whether the app has nothing more to show: the root widget is
// attached, no frame is scheduled, no animation is running and no image is
// loading. The self-capture of GLUTTER_SCREENSHOT waits for it.
//
// Flutter: no counterpart
func (b *WidgetsBinding) IsIdle() bool {
	return b.rootElement != nil && !b.scheduler.HasScheduledFrame() && b.scheduler.TransientCallbackCount() == 0 &&
		b.painting.ImageCache().PendingImageCount() == 0
}

// systemTextEngine makes the text engine of an app: the bundled fonts and the
// fonts of the machine, found through fontconfig if it is there and by scanning
// the font directories if not.
func systemTextEngine(ctx context.Context, logger *slog.Logger) *layout.Engine {
	if logger == nil {
		logger = slog.Default()
	}
	opts := text.CollectionOpts{}
	if m, err := textfontconfig.Open(ctx); err == nil {
		opts.Matcher = m
	} else {
		logger.Info("fontconfig is not available; scanning the font directories", "error", err)
	}
	c := text.NewCollection(opts)
	if err := fonts.Register(c); err != nil {
		logger.Error("the bundled fonts could not be registered", "error", err)
	}
	return layout.NewEngine(ctx, c, layout.EngineOpts{OnFontError: func(err error) {
		logger.Warn("a font could not be loaded", "error", err)
	}})
}
