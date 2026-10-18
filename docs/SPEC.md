# Glutter: Specification

| | |
|---|---|
| Status | Draft 0.1 |
| Date | 2026-10-18 |
| Owner | Manos Ragiadakos |
| Module | `github.com/rm4n0s/glutter` |
| License | BSD-3-Clause |

Glutter is a GUI toolkit for Go that reproduces Flutter's framework API on its own Go engine. An app is a normal Go program built with `CGO_ENABLED=0 go build`: no Dart, no Skia, no C compiler, no development packages. Version 1.0 is a complete desktop toolkit for Linux (Wayland and X11), including a system tray. Everything specific to an OS or GPU sits behind two internal interfaces, so Windows, macOS, web and mobile can be added later without changing the framework.

---

## 1. Goals

| # | Goal |
|---|---|
| G1 | **Faithful to Flutter.** It has the same three trees (Widget → Element → RenderObject), the same layout protocol (constraints go down, sizes go up), and the same lifecycle, names, defaults and behavior. A Flutter developer can follow Flutter's documentation and translate it mechanically using the rules in §12 |
| G2 | **Required parameters are checked at compile time.** Flutter's `required` becomes a positional constructor argument (§12.1) |
| G3 | **Easy to build.** You need only Go: no C compiler, no `pkg-config`, no `-devel` packages. Cross-compiling with `GOOS`/`GOARCH` works (§16) |
| G4 | **A complete desktop toolkit** with the scope in §3 |
| G5 | **Portable by construction.** OS, windowing and GPU code is isolated behind the interfaces in §6.3 |
| G6 | **Fast.** The targets are in §15. When nothing changes, CPU use is close to zero |
| G7 | **Testable without a display.** Widget tests and golden images run without a display server or GPU (§17) |

## 2. Non-goals for 1.0

- Hot reload, or a hot restart that keeps state. Go can't do this; we offer only a fast rebuild-and-run.
- Cupertino widgets. Planned after 1.0.
- Windows, macOS, web, Android and iOS backends. The design supports them (§6.4), but they aren't implemented.
- Dart interop, Flutter plugins, and platform channels as a message codec. Go APIs are exposed directly instead.
- Custom fragment shaders (`FragmentProgram`), 3D, video playback, WebView, SVG, printing.
- DevTools and the widget inspector UI. Debug paint flags and tree dumps are in scope (§14.3).
- Material 2 visuals. Only Material 3 is supported.
- The XEmbed system tray. Only StatusNotifierItem is supported (§8.9).
- State restoration (`RestorationMixin`, `RestorationScope`).
- Internationalization beyond the mechanism. `Localizations` works, but only English is shipped for `MaterialLocalizations` and `WidgetsLocalizations`.

## 3. Scope of 1.0 (Linux)

| Area | In scope |
|---|---|
| Windowing | Wayland (primary) and X11. Multiple windows, dialog windows, fullscreen/maximize/minimize, min/max size, window icon, app id, title bars drawn by the app when the compositor doesn't draw them |
| Input | Mouse, high-resolution scrolling, touchpad pan/zoom gestures, touchscreens, keyboard (any XKB layout, compose/dead keys), IME (Wayland text-input-v3; on X11, IBus or Fcitx5 over D-Bus) |
| Data exchange | Clipboard (text, images, files), primary selection, drag and drop with other apps and within the app |
| Rendering | Antialiased paths, clips, transforms, all of Flutter's gradients, images, opacity and layers, blur, shadows, color filters, blend modes, backdrop filters, shader masks |
| Text | System fonts and fallback, rich text, bidi, line breaking, ellipsis, decorations, inline widgets, full text editing with IME |
| Framework | foundation, scheduler, animation, physics, gestures, painting, rendering, semantics, services and widgets, covering everything listed in §11 |
| Material 3 | The full component set listed in §11.2, plus theming including `ColorScheme.fromSeed` |
| Desktop services | System tray with a menu, notifications, file open/save dialogs, opening URIs, dark/light mode and accent color from system settings |
| Accessibility | The semantics tree exposed to screen readers through AT-SPI2 (Orca) |

## 4. Decisions log

| # | Decision | Notes |
|---|---|---|
| D1 | Flutter's API is translated with **positional required arguments plus a trailing options struct** ("Version 2") | §12. Rejected: struct literals (no compile-time required check), constructors returning `(Widget, error)`, and constructors returning error widgets |
| D2 | **No cgo.** The system's C libraries are called through `github.com/ebitengine/purego` (Apache-2.0, from the Ebiten team): `dlopen`, C calls and C→Go callbacks without a C compiler. They are libwayland-client/-egl/-cursor, libxkbcommon(-x11), libX11 (+X11-xcb, Xi, Xrandr, Xcursor), libEGL and libGLESv2 | Supersedes the earlier "cgo allowed". Same libraries as Gio uses, called without cgo, as Ebiten does. §8.1, §16 |
| D3 | **Every system library is loaded at runtime** with `dlopen`, nothing is linked at build time | A missing library becomes a tagged error (`LibraryLoadFailed`, `EGLLoadFailed`) instead of a loader failure before `main`, and a missing Wayland library falls back to X11. Lookup rules are in §8.1 |
| D4 | `github.com/go-text/typesetting` is an allowed dependency (fonts, shaping, bidi, segmentation, system font discovery) | §10 |
| D5 | Scope is **Widgets + Material 3**. Cupertino comes later | §11 |
| D6 | Linux first: **Wayland primary, X11 supported**, selected at runtime | §8 |
| D7 | Renderer: **OpenGL ES 3.0 through EGL**, with GLSL ES 3.00 shaders | The same shaders run on WebGL2, ANGLE (Windows, macOS) and Android. §9 |
| D8 | Desktop extras in 1.0: system tray, notifications, file dialogs, drag and drop, multiple windows, accessibility | §8.7–8.10 |
| D9 | Errors use `github.com/rm4n0s/errors` | §14 |
| D10 | Our own engine. Gio and Ebiten aren't used | Glutter owns every layer above the system libraries |
| D11 | Go 1.27 minimum | Generic type aliases (1.24+) are used for re-exports |
| D12 | `golang.org/x/image` is an allowed dependency (v0.47.0 at the time of writing) | WebP and BMP decoding (§7) |
| D13 | Navigator ships **both** the imperative API and the Pages/Router API in 1.0 | M10 |
| D14 | D-Bus through `github.com/godbus/dbus/v5` (pure Go, BSD-2; Gio and Fyne use it too) | Calls are converted to Go types automatically, and Go objects can be served directly, which the tray, its menu and AT-SPI2 need. `internal/dbus` needs no cgo |
| D15 | Bugs panic, and **the library recovers** per element/render object/callback, reports through `ErrorReporter` and shows `ErrorWidgetBuilder`'s widget | §14.2. Rejected: returning errors from `Build`, and crashing like Gio and Ebiten |
| D16 | **Wayland must work flawlessly** (§8.11). Two more C libraries, loaded through purego and optional: **libdecor** (native title bars on GNOME) and **libfontconfig** (font matching like Flutter's Linux engine) | Without them, Glutter falls back to its own title bar and to `fontscan` |

## 5. Size estimate

These are production lines of Go, excluding tests (expect about as many test lines). They're calibrated against Flutter's framework (330k lines of Dart), Gio (40k lines of Go) and go-text (58k lines of Go).

| Layer | Lines |
|---|---|
| Platform (Wayland, X11, input, IME, clipboard, drag and drop, window decorations, purego bindings and generator) | 13–19k |
| Renderer (GLES and software) | 15–25k |
| Text layout and editing support (on go-text) | 5–8k |
| Framework core (foundation … widgets) | 80–110k |
| Material 3 | 40–60k |
| Desktop services (D-Bus, tray, notifications, portals, AT-SPI2) | 15–25k |
| **Total** | **~175–245k** |

---

## 6. Architecture

### 6.1 Layers

```
 app code (Go)
 ───────────────────────────────────────────────────────────────────────
 material             Material 3 widgets, theming, ink, icons
 widgets              Widget/Element/State, every core widget, Navigator, focus, editing
 rendering semantics  RenderObject tree, layout and paint, layers, semantics tree
 painting gestures services animation physics scheduler foundation
 desktop              tray, notifications, file dialogs, open URI
 ─────────────────────────── public engine API ────────────────────────
 ui                   Canvas, Paint, Path, Picture, Scene, Paragraph, Image,
                      PlatformDispatcher, View, pointer/key data, semantics updates
 ─────────────────────────── internal ─────────────────────────────────
 internal/engine      implements the ui contracts using the three below
 internal/platform    Platform interface ── linux/wayland, linux/x11, fake
 internal/gpu         Renderer interface ── gles, soft
 internal/text        paragraph layout on go-text
 internal/dbus        D-Bus connection and services (tray, notify, portals, AT-SPI2)
 ───────────────────────────────────────────────────────────────────────
 system, all loaded with dlopen through purego (no cgo):
   libwayland-client/-egl/-cursor, libxkbcommon(-x11), libX11 (+X11-xcb, Xi, Xrandr, Xcursor),
   libEGL, libGLESv2, libdecor, libfontconfig;  session D-Bus through godbus (pure Go)
```

`ui` plays the role of Flutter's `dart:ui`, and `internal/engine` plays the role of Flutter's C++ engine. The framework packages talk only to `ui`.

### 6.2 Package layout

```
glutter/
  ui/                    engine API (dart:ui)
  foundation/            Key, Opt[T], ChangeNotifier, TargetPlatform, diagnostics, debug flags
  scheduler/             SchedulerBinding, Ticker, frame phases
  animation/             AnimationController, Tween, Curves, ...
  physics/               simulations
  gestures/              pointer events, hit testing, gesture arena, recognizers
  painting/              EdgeInsets, BoxDecoration, borders, TextPainter, ImageProvider, ...
  rendering/             RenderObject, RenderBox, slivers, layers, PipelineOwner
  semantics/             SemanticsNode, actions, flags
  services/              keyboard, text input, clipboard, mouse cursors, assets
  widgets/               the widgets library
  material/              Material 3 (re-exports widgets; §12.11)
  desktop/               tray, notifications, file dialogs, open URI
  gluttertest/           WidgetTester, finders, golden comparison (flutter_test)
  internal/
    engine/              wires platform + gpu + text into the ui contracts
    platform/            Platform interface, shared types
      linux/             event loop, settings, selection of Wayland or X11
        wayland/         purego; Go protocol bindings generated from protocols/*.xml
        x11/             purego
      fake/              scripted platform for tests
    gpu/                 Renderer interface, Scene flattening, glyph and path atlases
      gles/              purego (EGL + GLES 3.0)
      soft/              pure-Go rasterizer (goldens, headless)
    text/                paragraph layout, font collection, glyph outlines
    dbus/                pure Go (godbus): tray, notifications, portals, settings, AT-SPI2
  cmd/glutter-gen/       generators: Wayland protocols, key tables, re-exports, CopyWith
  examples/              counter, gallery, tray, multiwindow, editor
  third_party/flutter/   Flutter's LICENSE (we port from BSD-3 code)
  scripts/               fetch-reference.sh (pinned Flutter sources into reference/)
  reference/             git-ignored Flutter sources for porting (§20)
  docs/                  SPEC.md, IMPLEMENTATION.md, IMPLEMENTED.md
```

### 6.3 Portability contract

The two internal interfaces are the only places where an OS, windowing system or GPU API can appear. Here is a sketch of the method sets; the details are decided at milestone M1:

```go
// internal/platform
type Platform interface {
	Run(ctx context.Context, h Handler) error             // owns the UI thread's event loop
	Post(fn func())                                       // safe from any goroutine; wakes the loop
	NewWindow(cfg WindowConfig) (Window, error)
	Displays() []Display
	Clipboard() Clipboard
	Settings() Settings                                   // brightness, accent color, text scale, reduce motion, ...
	Services() Services                                   // optional capabilities; nil when absent
}

type Window interface {
	ID() ViewID
	NativeSurface() gpu.SurfaceHandle                     // opaque to everything except gpu backends
	SetTitle(string); SetSize(ui.Size); SetState(WindowState); /* ... */
	SetCursor(ui.MouseCursor)
	SetTextInput(ui.TextInputState)                        // IME: enable, cursor rectangle, surrounding text
	StartDrag(DragData) error
	Close()
}

type Handler interface { // implemented by internal/engine
	OnPointer([]ui.PointerData); OnKey(ui.KeyData); OnTextInput(ui.TextInputEvent)
	OnFrame(ViewID, time.Duration); OnMetrics(ViewID, ui.ViewMetrics); OnLifecycle(ui.AppLifecycleState)
	OnDrop(ViewID, DropEvent); OnSettings(Settings); OnCloseRequest(ViewID) bool
}

// internal/gpu
type Renderer interface {
	Attach(gpu.SurfaceHandle) (Surface, error)
	Render(s Surface, scene *ui.Scene, size ui.Size, dpr float32) error
	Snapshot(scene *ui.Scene, size ui.Size) (*image.RGBA, error) // toImage, raster cache
	Close() error
}
```

Rules:
1. Public packages never import `internal/platform/<os>`, `internal/gpu/gles` or `internal/dbus`. Only `internal/engine` chooses backends.
2. No `runtime.GOOS` and no OS build tags outside the backend packages. Framework behavior that differs by platform (scroll physics, shortcuts such as Ctrl vs Cmd, menu conventions, typography) is driven by `foundation.TargetPlatform`. That value comes from the platform, and tests can override it, as with Flutter's `debugDefaultTargetPlatformOverride`.
3. Desktop services (`Services`: tray, notifications, file chooser, open URI, accessibility bridge) are optional capabilities. A missing one returns a tagged error such as `TrayUnavailable`. It never panics.
4. Shaders are GLSL ES 3.00 only. Any GLES call outside `internal/gpu/gles` is a bug.
5. Backend selection on Linux: `GLUTTER_BACKEND=wayland|x11` if set. Otherwise Wayland when `WAYLAND_DISPLAY` connects, then X11 through `DISPLAY`, then the error `NoDisplay`.
6. Nothing is linked at build time, so one binary contains both backends. A missing library only disables the backend that needs it.
7. There is no `import "C"` anywhere. `CGO_ENABLED=0 go build ./...` must always succeed.

### 6.4 What a future port has to implement

| Target | Platform backend | GPU backend | Services |
|---|---|---|---|
| Windows | Win32 through `syscall` (no cgo): windows, IME (IMM32/TSF), OLE drag and drop, clipboard | `gles` on bundled ANGLE (`libEGL.dll`, `libGLESv2.dll`), the same approach as Flutter's Windows engine | Shell_NotifyIcon tray, toast notifications, IFileDialog, UI Automation |
| macOS | Cocoa through purego's `objc` package (no cgo) | `gles` on ANGLE/Metal; a native Metal backend later | NSStatusItem, UNUserNotificationCenter, NSOpenPanel, NSAccessibility |
| Web | `syscall/js` (canvas element, DOM events, hidden textarea for IME) | `gles` on WebGL2 (the same API and shaders) | Notifications API, `<input type=file>`, ARIA tree |
| Android / iOS | NDK `ANativeActivity` / UIKit through cgo (the only targets where cgo is unavoidable: purego requires cgo on Android, and iOS apps must link statically) | `gles` (Android); ANGLE/Metal (iOS) | platform equivalents |

None of these ports needs changes above `internal/`.

**GPU libraries on Windows and macOS.** Neither OS provides EGL/GLES, so a `gles`-based port **ships ANGLE with the app**:
- **Windows**: `libEGL.dll` and `libGLESv2.dll` next to the `.exe`, loaded with `LoadLibraryEx(…, LOAD_LIBRARY_SEARCH_DEFAULT_DIRS)` (the app directory, then System32; never the current directory or `PATH`), as Gio does. `d3dcompiler_47.dll` comes with Windows 10+.
- **macOS**: the `.dylib`s in `Contents/Frameworks/`, loaded by an absolute path derived from the executable and code-signed with the app.

The alternative is native backends (Direct3D 11, Metal) behind `internal/gpu`. That's Gio's default, needs no bundled files, and costs one renderer and shader language per OS. Choose when the port starts.

---

## 7. Engine API (`ui`)

This is the Go translation of `dart:ui`. Values are immutable unless the type is a builder or recorder.

| Group | Types and functions |
|---|---|
| Geometry | `Offset`, `Size`, `Rect`, `RRect`, `RSuperellipse`, `Radius`, `Matrix4` (float32 column-major), `Rect.Intersect`/`ExpandToInclude`/`Inflate`, lerps |
| Color | `Color` (float32 ARGB in extended sRGB, as in current Flutter), `ColorSpace`, `BlendMode` (all 29 of Flutter's modes), `FilterQuality` |
| Paint | `Paint` (color, shader, blend mode, style, stroke width/cap/join/miter limit, `MaskFilter`, `ColorFilter`, `ImageFilter`, anti-alias, invert colors) |
| Shaders | `GradientLinear`, `GradientRadial` (including two-point conical), `GradientSweep`, `ImageShader`, `TileMode` |
| Filters | `ColorFilterMode`, `ColorFilterMatrix`, `ColorFilterLinearToSrgbGamma`/`SrgbToLinearGamma`, `ImageFilterBlur`, `ImageFilterMatrix`, `ImageFilterCompose`, `ImageFilterDilate`/`Erode`, `MaskFilterBlur` (`BlurStyle`) |
| Path | `Path` (MoveTo, LineTo, QuadraticBezierTo, CubicTo, ConicTo, ArcTo, ArcToPoint, relative variants, AddRect/RRect/Oval/Arc/Polygon/Path, Close, FillType, Transform, Shift, GetBounds, Contains, `PathCombine` with `PathOperation`), `PathMetrics` (length, extract path, tangent) |
| Canvas | `PictureRecorder`, `Canvas` (Save/SaveLayer/Restore/RestoreToCount, Translate/Scale/Rotate/Skew/Transform, ClipRect/RRect/RSuperellipse/Path, DrawColor/Paint/Line/Rect/RRect/DRRect/Oval/Circle/Arc/Path/Points/Vertices/Atlas/Image/ImageRect/ImageNine/Paragraph/Picture/Shadow), `Picture` (`ToImage`) |
| Compositing | `SceneBuilder` (PushOffset, PushTransform, PushClipRect/RRect/Path, PushOpacity, PushColorFilter, PushImageFilter, PushBackdropFilter, PushShaderMask, AddPicture, AddRetained, Pop), `Scene`, `EngineLayer` handles for retained layers |
| Images | `Image`, `InstantiateImageCodec(ctx, []byte)` → `Codec` (PNG, JPEG and GIF including animation through the standard library; WebP and BMP through `golang.org/x/image`), `DecodeImageFromPixels` |
| Text | `ParagraphBuilder`, `ParagraphStyle`, `TextStyle`, `StrutStyle`, `Paragraph` (Layout, Width, Height, MinIntrinsicWidth, MaxIntrinsicWidth, AlphabeticBaseline, IdeographicBaseline, LongestLine, DidExceedMaxLines, GetBoxesForRange, GetBoxesForPlaceholders, GetPositionForOffset, GetWordBoundary, GetLineBoundary, ComputeLineMetrics, GetGlyphInfoAt), `TextDirection`, `TextAlign`, `TextDecoration`, `FontWeight`, `FontStyle`, `FontFeature`, `FontVariation`, `PlaceholderAlignment`, `LoadFontFromList` |
| Platform | `PlatformDispatcher` (views, frame callbacks `OnBeginFrame`/`OnDrawFrame`, `ScheduleFrame`, pointer data packets, key data, locales, text scale factor, platform brightness, accessibility features, `OnSemanticsActionEvent`, app lifecycle, `RequestAppExit`), `FlutterView` → `View` (ID, physical size, device pixel ratio, view padding/insets, display, `Render(scene)`, `UpdateSemantics`) |
| Input data | `PointerData` (change, kind: mouse/touch/trackpad/stylus, device, position, buttons, scroll deltas, pan/zoom), `KeyData` (type, physical, logical, character, synthesized), `TextInputEvent` |
| Semantics | `SemanticsUpdateBuilder`, `SemanticsAction`, `SemanticsFlag`, `SemanticsRole` |

Numbers follow Dart's types: `Offset`, `Size`, `Rect`, `Radius`, `PointerData` and other geometry-like values use `float64` (Dart's `double`), and only `Matrix4` and `Color` use `float32`. Enums that Flutter declares start with `<Enum>Unspecified` as the zero value, followed by Flutter's values in declaration order (§12.3).

`ui` holds no global state. `RunApp` creates the `PlatformDispatcher` and hands it to the binding (§13).

---

## 8. Linux platform layer

### 8.1 System libraries

Everything is loaded with `purego.Dlopen` at startup (D2, D3). The versioned name is tried first, because the unversioned symlink exists only when the `-devel` package is installed.

| Library | Names tried | Needed for | If missing |
|---|---|---|---|
| libwayland-client | `libwayland-client.so.0` | Wayland | fall back to X11 |
| libwayland-egl | `libwayland-egl.so.1` | Wayland + EGL | fall back to X11 |
| libwayland-cursor | `libwayland-cursor.so.0` | cursors without `wp_cursor_shape_v1` | default cursor only |
| libxkbcommon | `libxkbcommon.so.0` | keymaps and compose, both backends | `LibraryLoadFailed` |
| libxkbcommon-x11 | `libxkbcommon-x11.so.0` | X11 keymaps | `LibraryLoadFailed` on X11 |
| libX11, libX11-xcb | `libX11.so.6`, `libX11-xcb.so.1` | X11 | X11 unavailable |
| libXi, libXrandr, libXcursor | `libXi.so.6`, `libXrandr.so.2`, `libXcursor.so.1` | X11 input, monitors, cursors | that X11 feature is disabled |
| libEGL | `libEGL.so.1`, `libEGL.so` | rendering | `EGLLoadFailed` |
| libGLESv2 | `libGLESv2.so.2`, `libGLESv2.so` | rendering | `EGLLoadFailed` |
| libdecor | `libdecor-0.so.0` (loads its own GTK or cairo plugin) | native title bars on Wayland compositors without server-side decorations (GNOME), §8.2 | Glutter draws the title bar itself |
| libfontconfig | `libfontconfig.so.1` | font matching and fallback that follow the user's fontconfig settings, as Flutter's Linux engine does, §10 | go-text's `fontscan` |

**How the libraries are found.**
- `dlopen` gets a bare name with no path, so the dynamic linker (`ld.so`) searches exactly as it does for linked libraries: `LD_LIBRARY_PATH`, then `/etc/ld.so.cache` (built by `ldconfig` from `/etc/ld.so.conf.d/`), then `/lib` and `/usr/lib`. That covers every distro's library directory (`/usr/lib64`, `/usr/lib/x86_64-linux-gnu`, `/usr/lib`).
- `GLUTTER_EGL_LIBRARY` and `GLUTTER_GLES_LIBRARY` (absolute paths) override the lookup for unusual setups, e.g. NixOS without `LD_LIBRARY_PATH`.
- On failure, the error carries the names tried and `dlerror()` as metadata.

**Rules for calling C through purego.**
1. **No C variadic functions.** purego can't call them. Use the non-variadic variant: Wayland requests go through `wl_proxy_marshal_array_flags`, not `wl_proxy_marshal_flags`. This is also why X11 IME uses D-Bus instead of XIM (§8.3).
2. **Callbacks are created once, at startup.** A process can have about 2,000 C→Go callbacks, and they are never freed. Each callback dispatches by its user-data pointer. For example, all Wayland events go through **one** dispatcher installed with `wl_proxy_add_dispatcher`. Callback arguments must fit in a pointer.
3. **C structs mirrored in Go** (`wl_interface`, `wl_message`, `wl_argument`, `XEvent` and friends, xkb and EGL types) each have a layout test. It checks `unsafe.Sizeof`/`Offsetof` against values taken from the C headers and recorded in the test. Data that C keeps (interface tables, strings) lives in C memory, allocated with libc `malloc` through purego, never in Go memory.
4. Only the UI thread calls into these libraries.

**Wayland protocol bindings.** `cmd/glutter-gen wayland` reads the protocol XML files kept in `internal/platform/linux/wayland/protocols/` (MIT licensed) and writes Go code (`*_gen.go`): typed request methods, decoded events, and the `wl_interface` tables. Neither users nor contributors need `wayland-scanner` or `wayland-protocols`.

On Linux, libEGL and libGLESv2 come from libglvnd, which dispatches to Mesa or NVIDIA using the vendor files in `/usr/share/glvnd/egl_vendor.d/`. Every Wayland desktop has them, because the compositors draw through EGL themselves. GPU-less machines get Mesa's llvmpipe (GLES 3.2 in software).


### 8.2 Wayland

| Protocol | Required? | Without it |
|---|---|---|
| `wl_compositor`, `wl_subcompositor`, `wl_shm`, `wl_seat`, `wl_output`, `xdg_wm_base` | required | error `WaylandProtocolMissing` |
| `zxdg_decoration_manager_v1` | optional | libdecor, else Glutter's own title bar (see "Window decorations" below) |
| `wp_fractional_scale_v1` + `wp_viewporter` | optional | integer `wl_surface.set_buffer_scale` |
| `wp_cursor_shape_v1` | optional | XCursor theme through wayland-cursor |
| `zwp_text_input_manager_v3` | optional | no IME; raw keys plus xkb compose only |
| `wl_data_device_manager` | optional (always present in practice) | no clipboard or drag and drop |
| `zwp_primary_selection_device_manager_v1` | optional | no primary selection |
| `zwp_pointer_gestures_v1` | optional | no touchpad pinch/rotate (`PointerPanZoom*` events) |
| `xdg_activation_v1` | optional | focus requests ignored |
| `wp_presentation` | optional | frame timing from frame callbacks only |
| `xdg_toplevel_icon_manager_v1` | optional | window icon comes only from the `.desktop` file |
| `zxdg_exporter_v2` (`xdg_foreign`) | optional | portal dialogs not parented to our window |
| `xdg_dialog_v1` | optional | modal dialogs not marked as modal for the compositor |
| `xdg_wm_base` v6+ (`suspended` state, `wm_capabilities`, `configure_bounds`) | optional (lower versions work) | no suspended state (rendering stops only when frame callbacks stop); every title-bar button is shown; the first size isn't limited to the screen |
| `wl_surface` v6 (`preferred_buffer_scale`) | optional | the scale comes from the outputs the surface is on |
| `zxdg_output_manager_v1`, `wl_output` v4 | optional | logical monitor sizes computed from mode and scale |
| `zwp_tablet_manager_v2` | optional | no stylus input (`PointerDeviceKind.stylus`) |
| `zwp_idle_inhibit_manager_v1` | optional | `desktop.KeepScreenOn` reports `IdleInhibitUnavailable` |
| `wp_content_type_v1` | optional | no content-type hint |

**Window decorations.** Glutter picks the first of these that works, per window:
1. **Server-side**, through `zxdg_decoration_manager_v1`: KDE, wlroots compositors (Sway, Hyprland), COSMIC.
2. **libdecor**, on compositors without server-side decorations, i.e. GNOME. Its GTK plugin draws a native Adwaita title bar, matching what Flutter's GTK-based Linux engine shows. libdecor draws in its own subsurfaces around our EGL surface and reports configure, close and commit through callbacks (§8.1 rule 2).
3. **Glutter's `WindowDecoration` widget** (Material defaults, replaceable by the app). It handles title, buttons limited by `wm_capabilities`, double-click to maximize, and right-click for the window menu (`xdg_toplevel.show_window_menu`). Edge resizing goes through `xdg_toplevel.resize` and dragging through `.move`. The shadow and resize borders sit inside the surface and are excluded with `xdg_surface.set_window_geometry`.

An app that wants its own title bar (tabs in the title bar, for example) sets `WindowConfig.Decorations = DecorationsApp`. That skips steps 1 and 2, and the app places `WindowDecoration` or its own widget.

### 8.3 X11

Xlib (through purego, as Ebiten does) with EGL (`EGL_PLATFORM_X11_KHR`). The pieces:
- **XInput2** for smooth scrolling, touch and precise pointer input
- **libxkbcommon-x11** for keymaps, so key handling is identical to Wayland
- **IME over D-Bus**: IBus (`org.freedesktop.IBus`, as Gio does on X11) or Fcitx5's D-Bus input-method frontend, whichever is running. XIM isn't used, because its API is C-variadic (§8.1 rule 1)
- **ICCCM/EWMH**: `_NET_WM_NAME`, `_NET_WM_STATE`, `_NET_WM_PING`, `_NET_WM_ICON`, `_NET_WM_WINDOW_TYPE`, `_MOTIF_WM_HINTS`, `WM_CLASS` set to the app id
- **Selections**: `CLIPBOARD` and `PRIMARY`, with `INCR` for large transfers
- **XDND v5** for drag and drop
- **Xcursor** themes for cursors
- **Xrandr** for monitors and refresh rates
- **XSETTINGS / `Xft.dpi`** for scaling

Frames are paced by `eglSwapInterval(0)` plus a timer at the monitor's refresh rate (XRandR's `XRRConfigCurrentRate`, 60 Hz when libXrandr is missing or says nothing): the first frame is drawn when the window is mapped and after each resize, other frames come one interval after the previous one, and requests in between are merged. (`eglSwapInterval(1)` would make the swap block on a window that is hidden, the same reason as on Wayland, §8.11.)

How the X11 backend does the rest:
- The window uses the X visual of the EGL config (`gles.X11Visual`), with a colormap of its own; `eglCreatePlatformWindowSurface` gets a pointer to the window ID.
- Keyboard, pointer and touch all come from XInput2 events of the master devices, and from nothing else: key events carry the modifier and group state, which goes straight into xkbcommon, so no XKB event handling is needed. The server's auto-repeat arrives as key events flagged `XIKeyRepeat` and becomes `KeyRepeatEvent` (SPEC §8.5 repeat timing is the server's). Without libXi or XInput2 on the server the backend has no keyboard and pointer input.
- Scrolling uses the scroll valuators of the device (the change in notches, 53 logical pixels each, as a scroll signal); the legacy buttons 4 to 7 count only on devices that have no scroll valuators. There is no pan/zoom gesture on X11, so a touchpad scrolls as scroll signals and pinch does not exist.
- Focus follows `XI_FocusIn`/`XI_FocusOut`; the held keys are released when the window loses it.
- An X IO error (the server went away) makes Xlib exit the process, as in every Xlib program; there is no connection-lost error like Wayland's.

### 8.4 Event loop and threading

- `RunApp` must be called from the main goroutine. `internal/platform` calls `runtime.LockOSThread()` in an `init` function, which keeps the main goroutine on the main OS thread. That thread is the **UI thread**.
- The loop waits with `poll(2)` on:
  - the Wayland/X11 connection
  - the D-Bus connection
  - an `eventfd` (wakeups from `Post`)
  - a `timerfd` (timers and the X11 frame pacer)
- On each wakeup it does the following, in order:
  1. Dispatch display events and translate them into `ui` events.
  2. Run posted functions.
  3. On a frame callback, run the frame: `OnBeginFrame` (animations), then `OnDrawFrame` (build, layout, paint, compositing, semantics), then `Scene`, then `Renderer.Render`, then swap.
- Frames are produced only when scheduled (`ScheduleFrame`). An idle app sleeps in `poll` with no timers armed.
- Rendering stays on the UI thread for 1.0. `ui.Scene` is immutable, so a separate raster thread can be added later without API changes.

### 8.5 Input

- **Pointer**: `wl_pointer` (axis value120 high-resolution scrolling, axis source, stop events → `PointerScrollEvent` and `PointerPanZoom*` for touchpads), `wl_touch`, XI2 equivalents. Device kinds map to Flutter's `PointerDeviceKind`.
  - Wayland pointer: the events between two `wl_pointer.frame` events become one batch for `Handler.OnPointer` (without `frame`, before `wl_pointer` version 5, each event is a batch). Enter is an add, motion a hover or, with a button down, a move; the first button down is a down, the last one up an up, and the buttons in between are moves with the new button state. Leaving with buttons down releases them (synthesized) before the remove. Linux buttons map to Flutter's bits (left 1, right 2, middle 4, side/back 8, extra/forward 16).
  - Scrolling: a wheel (or tilt, or no axis source) is a scroll signal, 53 logical pixels per notch (Flutter's Linux embedder), taken from `axis_value120` (or `axis_discrete`), else from the pixel value; a continuous source scrolls by its pixels. A touchpad (`axis_source` finger) is a pan/zoom gesture of the trackpad device: it starts with the first scroll event, its pan follows the fingers (the opposite of the axis values), and `axis_stop` ends it. Pinch and swipe (`zwp_pointer_gestures_v1`) are pan/zoom gestures with the compositor's translation, scale and rotation (degrees to radians).
  - Device ids: the mouse is 0, the trackpad 1, and a touch point is 1000 plus its touch id (Flutter tells pointers apart by device). A touch point is an add and a down at touch-down, moves with the primary button, and an up and a remove at touch-up; `wl_touch.cancel` cancels and removes every point.
  - Cursors: `wp_cursor_shape_v1` when the compositor has it, otherwise the XCursor theme through `libwayland-cursor` (`XCURSOR_THEME`, `XCURSOR_SIZE`; the first image of an animated cursor), with each cursor's names tried in order (CSS names first). `MouseCursorNone` hides the cursor. The cursor is set again each time the pointer enters a window.
- **Keyboard**:
  - The evdev scancode maps to `PhysicalKeyboardKey`, and the xkb keysym to `LogicalKeyboardKey`. Both tables are generated by `cmd/glutter-gen` from Flutter's key data.
  - The logical key follows Flutter's GTK embedder. A keysym in the generated table (Return, Shift_L, F1, ...) gives that key's id. Any other keysym with a Unicode character gives the lower-case character (plane 0). A keysym with neither (a dead key) gives the keysym in the GTK plane (`0x15_0000_0000 | keysym`), as does an unknown physical key with its XKB keycode. The keys of Flutter's layout goals (letters, digits, and a few punctuation keys) take the first Latin character that any layout of the keymap has on that key, at level 0, so shortcuts work on every layout and Shift or AltGr do not change the logical key; on a keymap with no Latin character there, the mandatory goals use the US key and the others keep the layout's own.
  - The character of a key-down event comes from its keysym, not from the modifiers (Control+a is "a"), and goes through xkbcommon's compose state, so a dead key gives no text and the next key gives the composed one. Key-up events carry no character and reuse the ids of the key-down.
  - Repeat rate and delay come from the compositor (Wayland) or XKB (X11), and repeats are synthesized as `KeyRepeatEvent`. Wayland: 25 keys per second after 600 ms until `repeat_info` arrives, a rate of 0 turns repeating off, and only the key pressed last repeats.
  - Dead keys and compose use xkbcommon's compose API when no IME is active.
- `HardwareKeyboard` state (pressed keys, lock modes) is kept consistent across focus changes. Keys are synthesized up/down on focus loss/gain, as Flutter's embedders do.

### 8.6 Text input (IME)

This is Flutter's text-input model (`TextInputConnection`, `TextEditingValue`, composing range) mapped to:
- **Wayland** `zwp_text_input_v3`: enable/disable, surrounding text, cursor rectangle, content type, preedit, commit, delete surrounding text
- **X11**: IBus or Fcitx5 over D-Bus (§8.3)

IBus and Fcitx5 work through both.

### 8.7 Windows (multiple windows)

- Every top-level window is a `ui.View` with its own EGL surface. All windows share one EGL context, which is made current per surface.
- **Kinds**: regular, dialog (parent and modal flags), and borderless/tooltip-style regular windows. Menus and popups render inside the window's `Overlay`, as in Flutter. Native `xdg_popup` is not used in 1.0.
- **Properties**: title, size, min/max size, resizable, decorated, fullscreen, maximized, minimized, icon, app id (Wayland `app_id` / X11 `WM_CLASS`). The app id must match the app's `.desktop` file for the tray and notifications to work.
- **Widget API**: `RunApp` shows the first window. Other windows are created with Flutter's experimental windowing API (`widgets/_window.dart`): `WindowController` with the `Window` widget, and `DialogWindowController` with `DialogWindow`, inside a `ViewCollection`. Flutter's `PopupWindow`, `TooltipWindow` and `SatelliteWindow` (native popups placed by `WindowPositioner`) are deferred until after 1.0. Flutter's API is still experimental, so re-check it against the pinned `reference/flutter` at M14 and follow it.
- **Closing**: closing the last window ends `RunApp`. An app can intercept closing with `AppLifecycleListener.OnExitRequested`, mirroring Flutter's `AppExitResponse`.

### 8.8 System settings

These come from the xdg-desktop-portal Settings interface:
- `org.freedesktop.appearance` `color-scheme` → `PlatformBrightness`
- `accent-color`
- `contrast` → high contrast
- `org.gnome.desktop.interface`: `text-scaling-factor` → text scale, `enable-animations` → `DisableAnimations`, `cursor-size`, `cursor-theme`

Changes are followed live through the `SettingChanged` signal. If the portal is missing, the fallbacks are XSETTINGS (X11), environment variables (`XCURSOR_SIZE`, `XCURSOR_THEME`), and then defaults. The locale comes from `LC_ALL`/`LC_MESSAGES`/`LANG`.

### 8.9 D-Bus services (`internal/dbus`, exposed through `desktop`)

| Service | Protocol | When unavailable |
|---|---|---|
| **System tray** | `org.kde.StatusNotifierItem` registered with `org.kde.StatusNotifierWatcher`; menu through `com.canonical.dbusmenu`; icon as ARGB32 pixmaps or a theme icon name; activate, secondary activate, scroll, tooltip, status (active/passive/needs attention) | `TrayUnavailable`. GNOME needs the AppIndicator extension; this must be documented |
| **Notifications** | `org.freedesktop.Notifications` (summary, body, icon, actions, urgency, replaces-id, timeout, `ActionInvoked`/`NotificationClosed`); inside Flatpak, `org.freedesktop.portal.Notification` | `NotificationsUnavailable` |
| **File dialogs** | `org.freedesktop.portal.FileChooser` (`OpenFile` with multiple selection, directory mode and filters; `SaveFile`; `SaveFiles`; current folder and name), parented by `xdg_foreign` handle or X11 window id | `FileChooserUnavailable`. There is no GTK fallback. Cancelling is a result, not an error |
| **Open URI** | `org.freedesktop.portal.OpenURI` | fall back to `xdg-open`, else `OpenURIUnavailable` |
| **Settings** | `org.freedesktop.portal.Settings` | §8.8 fallbacks |
| **Accessibility** | AT-SPI2 on the accessibility bus (§8.10) | logged at debug level; the app keeps working |

Calls into these services happen on a background goroutine with a `context.Context`. Results and signals are delivered to the UI thread through `Post`.

Desktop API sketch (Version 2 style):

```go
// declarative: the tray lives in the widget tree and updates when it rebuilds
desktop.NewSystemTray(icon, child, desktop.SystemTrayOpts{
	Tooltip:    "Glutter",
	OnActivate: s.showWindow,
	Menu: []desktop.TrayMenuEntry{
		desktop.NewTrayMenuItem("Open", s.showWindow),
		desktop.NewTrayMenuSeparator(),
		desktop.NewTrayMenuItem("Quit", s.quit),
	},
})

// imperative: call from a goroutine, never block the UI thread
files, err := desktop.OpenFiles(ctx, bc, desktop.OpenFilesOpts{
	Title: "Open image", Filters: []desktop.FileFilter{{Name: "Images", Patterns: []string{"*.png", "*.jpg"}}},
})
id, err := desktop.ShowNotification(ctx, bc, "Download finished", desktop.NotificationOpts{Body: name})
```

### 8.10 Accessibility (AT-SPI2)

The framework's semantics tree (`ui.SemanticsUpdate`) is mirrored as AT-SPI2 objects on the accessibility bus (the address comes from `org.a11y.Bus.GetAddress`). Each object implements the matching interfaces: `Accessible`, `Component`, `Action`, `Text`, `EditableText`, `Value`, `Selection`. Roles and states are mapped from `SemanticsFlag` and `SemanticsRole`. Events are sent for focus, state changes, text changes, caret moves, child changes and visible data changes. The bridge is enabled only while a screen reader is active (`org.a11y.Status.ScreenReaderEnabled`), so the semantics tree costs nothing otherwise. This mirrors Flutter's `SemanticsBinding.semanticsEnabled`. It is tested by hand with Orca and automatically against a fake bus.

### 8.11 Wayland quality requirements

Wayland is the primary backend, and these behaviors are requirements, not nice-to-haves. Each one gets a `glutter_display` test or an entry in the manual checklist (§17).

**Frame pacing**
- `eglSwapInterval(0)`. Frames are paced only by `wl_surface.frame` callbacks, so `eglSwapBuffers` never blocks, even when the window is hidden or occluded.
- When the toplevel is `suspended`, or no frame callback arrives, nothing is rendered. Tickers are resumed with the next frame's timestamp, so animations jump to the right point instead of replaying.
- At most one frame is in flight per window. Input arriving between frames is batched into the next one.

**Resizing and scaling**
- On `configure`: lay out and render at the new size and commit it in the same iteration, without waiting for a frame callback. Interactive resizing tracks the pointer. With EGL the buffer swap is the commit, so the backend resizes the `wl_egl_window`, sends `ack_configure`, and then renders and swaps; the first commit after the ack therefore already has the new size.
- While resizing, subsurfaces (libdecor) are in synchronized mode, so the decoration and the content never get out of step.
- Fractional scale: the buffer is `ceil(logical size × scale)`, and `wp_viewport` maps it back to the logical size. Text and shapes are rasterized at that buffer resolution, so they stay sharp at 125%/150%/175%.
- Moving a window between monitors with different scales re-renders at the new scale in the next frame, with no blurry frame in between.

**Input**
- Keyboard: the keymap is read from the compositor's fd per seat; repeat follows `wl_keyboard.repeat_info`; modifiers come from the `modifiers` event; pressed keys are released when focus leaves (§8.5).
- IME: `zwp_text_input_v3` (GNOME, KDE, wlroots). If the compositor lacks it, Glutter falls back to IBus/Fcitx5 over D-Bus, as on X11.
- Pointer: frame-grouped events (`wl_pointer.frame`), `axis_value120` for high-resolution wheels, `axis_source` + `axis_stop` for kinetic scrolling, and touchpad pinch/rotate through `zwp_pointer_gestures_v1`.
- Cursors: `wp_cursor_shape_v1` when available. Otherwise the XCursor theme and size from the settings portal, loaded at the scale of the monitor the pointer is on.

**Data exchange**
- Clipboard, primary selection and drag and drop read and write through pipes on a goroutine, never on the UI thread.
- MIME types offered: `text/plain;charset=utf-8`, `text/plain`, `UTF8_STRING`, `text/uri-list`, `image/png`, plus app-defined types.
- Drag and drop supports copy, move and ask actions and a drag icon surface.

**Lifecycle and correctness**
- `XDG_ACTIVATION_TOKEN` from the environment is consumed at startup, so launchers' startup feedback ends. Activation requests (e.g. clicking a notification) use `xdg_activation_v1`.
- A protocol error from the compositor ends `RunApp` with `WaylandProtocolError`, carrying the interface, object id and code (`wl_display_get_protocol_error`).
- The app id is set before the first commit, so the compositor matches the `.desktop` file (icon, name, tray, notifications).

**GPU drivers**
- Mesa and NVIDIA (through `libnvidia-egl-wayland`, which the NVIDIA driver installs) both work through `EGL_KHR_platform_wayland`. Explicit sync and buffer management are handled inside the EGL driver, so Glutter does nothing driver-specific.

---

## 9. Renderer

### 9.1 GLES backend (`internal/gpu/gles`)

- **Context**: EGL 1.4+, an OpenGL ES 3.0 context (`EGL_OPENGL_ES3_BIT`), one context shared by all windows. The libraries are opened with `purego.Dlopen`, and GL function pointers are resolved through `eglGetProcAddress` and bound with `purego.RegisterFunc`.
- **Pipeline**: the `ui.Scene` layer tree is flattened into draw lists. Commands are then batched by pipeline, texture and blend mode, and issued as instanced or batched draws.
- **Color**: premultiplied alpha, with blending in non-linear sRGB to match Flutter's output.

| Primitive | Technique |
|---|---|
| Rect, RRect, RSuperellipse, oval, circle, line, DRRect | **Analytic antialiasing**: a signed distance field in the fragment shader, with no multisampling |
| BoxShadow / `drawShadow` on rects and rrects | Analytic Gaussian-blurred rrect in closed form, with no offscreen pass |
| General paths | Flattened on the CPU and rasterized into a **coverage atlas** (8-bit, cached by path + transform), using the same rasterizer as the software backend; then composited on the GPU. Paths larger than an atlas tile use a tiled mask. A GPU tessellation path may be added later if profiling shows a need |
| Text | Glyph outlines from go-text, rasterized into a **glyph atlas** with grayscale antialiasing and 4 horizontal subpixel positions. Color glyphs: COLR and bitmap emoji (CBDT/sbix), as far as go-text supports them |
| Images | Textures with mipmaps for downscaling (`FilterQuality`). Nine-patch and atlas draws are batched |
| Gradients | Linear, radial, two-point conical and sweep in the shader; all tile modes |
| Clips | Rects through the scissor test. RRects and paths through a stencil buffer, or antialiased through coverage masks (Flutter clips are antialiased by default) |
| Layers | `SaveLayer`, opacity, color/image filters, backdrop filter and shader mask render into pooled offscreen framebuffers |
| Blur | Separable Gaussian with downsampling for large sigma |
| Blend modes | Porter-Duff through `glBlendFunc`. The advanced modes use `KHR_blend_equation_advanced` when present, else a shader that reads a copy of the destination |
| Raster cache | A `RepaintBoundary` subtree that stays unchanged for 3 frames is cached as a texture, as Flutter does |

**Errors** (returned from `RunApp`): `EGLLoadFailed`, `EGLInitFailed`, `EGLNoConfig`, `GLES3Unsupported`, `ShaderCompileFailed` (an internal bug, reported with the log).

### 9.2 Software backend (`internal/gpu/soft`)

A pure-Go scanline rasterizer computing exact area coverage. It renders the same `ui.Scene` into `*image.RGBA` and implements every feature in §9.1. It serves as:
1. the renderer for golden tests (deterministic, no GPU)
2. the backend for `Picture.ToImage` in headless tests
3. the shared path and glyph rasterizer used by `gles` for its atlases

Tests tagged `glutter_display` compare `gles` against `soft` within a per-pixel tolerance.

---

## 10. Text

- **go-text/typesetting** is used for:
  - font parsing (TrueType, OpenType, CFF, variable fonts, bitmap fonts)
  - shaping (its HarfBuzz port)
  - bidi
  - UAX #14 line breaking and UAX #29 graphemes and words
  - system font discovery and fallback (`fontscan`, cached in `$XDG_CACHE_HOME/glutter/fonts`) when fontconfig is unavailable
- **fontconfig** (`libfontconfig.so.1`, §8.1), when present, decides family matching and fallback order, as in Flutter's Linux engine. That means `sans-serif`, the user's UI font, emoji and CJK preferences, and per-language rules. go-text then parses and shapes the files fontconfig picks. Antialiasing and hinting follow GNOME's `font-antialiasing` and `font-hinting` settings from the settings portal.
- **Bundled fonts** (in `material/fonts`, through `embed`):
  - Roboto. It's the first family in Flutter's Linux typography; the system fallbacks Ubuntu, Cantarell, DejaVu Sans and Liberation Sans follow it, as in Flutter
  - Material Icons (Apache-2.0)
- **Paragraph layout** (`internal/text`), in this order:
  1. Spans become runs, segmented by script, direction and font, with fallback.
  2. Runs are shaped.
  3. Lines are broken, honoring `softWrap`, `maxLines` and ellipsis.
  4. Each line is reordered for bidi.
  5. Text is aligned and justified, with strut, height, leading distribution, letter and word spacing.
  6. Placeholders for `WidgetSpan` are laid out.
  7. Decorations are added: underline, overline and line-through in solid/double/dotted/dashed/wavy styles, with thickness.
  8. Metrics are computed: baselines, line metrics and intrinsics.
- **Queries for editing**: boxes for a range, position for an offset, word and line boundaries, caret metrics, glyph info.
- **Text scaling** follows Flutter's `TextScaler`, including clamping.
- **Font features and variations** are passed through to shaping.

---

## 11. Framework scope

### 11.1 Core libraries

| Go package | Flutter library | 1.0 scope |
|---|---|---|
| `foundation` | foundation | `Key`, `ValueKey[T]`, `ObjectKey`, `UniqueKey`, `GlobalKey[T]` (declared here, used by widgets), `Opt[T]`, `Listenable`, `ChangeNotifier`, `ValueNotifier[T]`, `TargetPlatform`, `DiagnosticsNode` and tree dumps, `DebugMode`/`ProfileMode`/`ReleaseMode` constants, debug flags |
| `scheduler` | scheduler | `SchedulerBinding` phases (transient, persistent, post-frame callbacks), `Ticker`, `TickerProvider`, `TickerFuture` (as a channel/callback), `SchedulerPhase`, `TimeDilation` (debug) |
| `animation` | animation | `Animation[T]`, `AnimationController` (forward, reverse, repeat, fling, animateTo/With), `Tween[T]` and typed tweens (`ColorTween`, `RectTween`, `IntTween`, `StepTween`, `ConstantTween`), `CurvedAnimation`, `CurveTween`, `TweenSequence`, `ReverseAnimation`, `ProxyAnimation`, `AnimationMin`/`Max`/`Mean`, `TrainHoppingAnimation`, every curve in `Curves`, `Cubic`, `Interval`, `Threshold`, `ElasticCurves` |
| `physics` | physics | `SpringSimulation`, `SpringDescription`, `FrictionSimulation`, `BoundedFrictionSimulation`, `GravitySimulation`, `ClampingScrollSimulation`, `BouncingScrollSimulation`, tolerances |
| `gestures` | gestures | every `PointerEvent` (including hover, scroll, scroll inertia cancel, pan/zoom start/update/end, signals), hit testing, `PointerRouter`, `GestureArenaManager`, `PointerSignalResolver`, recognizers (Tap, DoubleTap, SerialTap, LongPress, Pan, Horizontal/VerticalDrag, Scale, MultiTap, Eager), `VelocityTracker` (including the IOS/MacOS variants), `MouseTracker` (enter/exit/hover), per-device drag settings |
| `painting` | painting | `EdgeInsets`(`Directional`), `Alignment`(`Directional`), `BorderRadius`(`Directional`), `BorderSide`, `Border`, `BoxBorder`, `ShapeBorder` family (RoundedRectangle, RoundedSuperellipse, Stadium, Circle, Oval, BeveledRectangle, ContinuousRectangle, Star, LinearBorder), `BoxDecoration`, `ShapeDecoration`, `Gradient`s, `BoxShadow`, `DecorationImage`, `ImageProvider` (Asset, File, Memory, Network through `net/http`, Resize), `ImageCache`, `ImageStream`, `TextPainter`, `InlineSpan`/`TextSpan`/`PlaceholderSpan`, `TextStyle`, `StrutStyle`, `TextScaler`, `ColorSwatch`, `HSVColor`/`HSLColor`, `MatrixUtils`, `FlutterLogoDecoration`, every `lerp` |
| `rendering` | rendering | `RenderObject`, `RenderBox`, `BoxConstraints`, `PipelineOwner` (layout, compositing bits, paint, semantics), relayout/repaint boundaries, the layer tree (`ContainerLayer`, `OffsetLayer`, `ClipRect/RRect/RSuperellipse/PathLayer`, `OpacityLayer`, `TransformLayer`, `ColorFilterLayer`, `ImageFilterLayer`, `BackdropFilterLayer`, `ShaderMaskLayer`, `PictureLayer`, `LeaderLayer`/`FollowerLayer`), every RenderX behind the widgets below (Flex, Stack, IndexedStack, Wrap, Flow, Table, ListBody, Padding, Align, Positioned, ConstrainedBox, LimitedBox, FractionallySizedBox, IntrinsicWidth/Height, AspectRatio, FittedBox, Baseline, Transform, FractionalTranslation, RotatedBox, Clip*, Opacity, DecoratedBox, CustomPaint, Paragraph, EditableText, Image, MouseRegion, Listener, IgnorePointer, AbsorbPointer, Offstage, RepaintBoundary, AnnotatedRegion), `Viewport`/`ShrinkWrappingViewport`, slivers (List, FixedExtentList, VariedExtentList, Grid, Padding, ToBoxAdapter, PersistentHeader, FillViewport, FillRemaining, MainAxisGroup, CrossAxisGroup, Opacity, IgnorePointer, Offstage), `SliverGridDelegate`s, `RenderAbstractViewport`, `SelectionRegistrar` |
| `semantics` | semantics | `SemanticsNode`, `SemanticsConfiguration`, `SemanticsOwner`, actions, flags, roles, `SemanticsProperties`, sort keys, update building |
| `services` | services | `HardwareKeyboard`, `KeyEvent` (down/up/repeat), `LogicalKeyboardKey`/`PhysicalKeyboardKey` (generated), `KeyboardLockMode`, `TextInput`/`TextInputConnection`/`TextInputClient`, `TextEditingValue`, `TextSelection`, `TextRange`, `TextInputFormatter`s (FilteringTextInputFormatter, LengthLimitingTextInputFormatter), `Clipboard` (text, plus image/file extensions), `MouseCursor`/`SystemMouseCursors`, `AssetBundle` over `fs.FS`, `SystemSound` (no-op), `SystemNavigator.Pop` → close window |
| `widgets` | widgets | **Framework**: `Widget`, `Element`, `BuildContext`, `BuildOwner`, `StatelessWidget`, `StatefulWidget`, `State[T]`, `InheritedWidget`, `InheritedNotifier`, `InheritedModel`, `InheritedTheme`, `ProxyWidget`, `ParentDataWidget`, `RenderObjectWidget` family, `GlobalKey`, `ErrorWidget`, `WidgetsBinding`, `AppLifecycleListener`. **Basic**: Container, Padding, Align, Center, SizedBox, ConstrainedBox, UnconstrainedBox, OverflowBox, LimitedBox, FractionallySizedBox, FittedBox, AspectRatio, Baseline, Row, Column, Flex, Expanded, Flexible, Spacer, Stack, Positioned(Directional), IndexedStack, Wrap, Flow, Table, ListBody, Offstage, Visibility, Opacity, Transform, RotatedBox, FractionalTranslation, Clip*, CustomPaint, DecoratedBox, ColoredBox, Placeholder, Text, RichText, DefaultTextStyle, Icon, IconTheme, ImageIcon, Image, RawImage, GestureDetector, RawGestureDetector, Listener, MouseRegion, AbsorbPointer, IgnorePointer, Builder, StatefulBuilder, LayoutBuilder, OrientationBuilder, ValueListenableBuilder, ListenableBuilder, AnimatedBuilder, FutureBuilder, StreamBuilder, MediaQuery, Directionality, Localizations, Title, SafeArea, KeyedSubtree, RepaintBoundary, Semantics, MergeSemantics, ExcludeSemantics, BlockSemantics. **Scrolling**: Scrollable, ScrollController, ScrollPosition, ScrollPhysics (Clamping, Bouncing, NeverScrollable, AlwaysScrollable, PageScrollPhysics), ScrollConfiguration/ScrollBehavior (desktop: scrollbars visible, no mouse drag-scroll), ScrollNotification, SingleChildScrollView, ListView (+ builder, separated, custom), GridView (+ count, extent, builder), CustomScrollView, PageView, NestedScrollView, ReorderableListView/ReorderableList, ListWheelScrollView, RawScrollbar, AutomaticKeepAlive, KeepAlive, InteractiveViewer, two-dimensional scrolling (TableView basis). **Navigation**: Navigator (imperative API and Pages API), Route, PageRoute, ModalRoute, PopupRoute, RawDialogRoute, PageRouteBuilder, Router, RouteInformationParser, BackButtonListener, PopScope, Overlay, OverlayEntry, OverlayPortal, Hero. **Focus and input**: FocusNode, FocusScopeNode, Focus, FocusScope, FocusTraversalGroup, FocusTraversalPolicy family, Shortcuts, Actions, Intent, CallbackShortcuts, DefaultTextEditingShortcuts (Linux bindings), EditableText, TextEditingController, UndoHistory, SelectableRegion, SelectionContainer, SelectionArea (basis), context-menu infrastructure, Draggable, LongPressDraggable, DragTarget, platform drag and drop bridge, Form, FormField, RawAutocomplete, RawMenuAnchor, Tooltip (basis). **Animation**: ImplicitlyAnimatedWidget family (AnimatedContainer, AnimatedPadding, AnimatedAlign, AnimatedPositioned(Directional), AnimatedOpacity, AnimatedRotation, AnimatedScale, AnimatedSlide, AnimatedDefaultTextStyle, AnimatedPhysicalModel, AnimatedFractionallySizedBox, AnimatedTheme in material), TweenAnimationBuilder, AnimatedSwitcher, AnimatedCrossFade, AnimatedSize, AnimatedList, SliverAnimatedList, AnimatedGrid, AnimatedModalBarrier, every XTransition (Fade, Scale, Rotation, Slide, Size, Position, RelativePosition, DecoratedBox, Align, DefaultTextStyle, MatrixTransition). **Windows**: View, ViewCollection, WindowController + Window, DialogWindowController + DialogWindow, and `WindowDecoration` (Glutter-specific: the title bar drawn by the app, §8.2) |

### 11.2 Material 3 (`material`)

- **Theming**:
  - `ThemeData` (seeded and explicit), `Theme`, `ThemeOf`, `AnimatedTheme`
  - `ColorScheme`, including `ColorSchemeFromSeed` through a port of material-color-utilities (HCT, dynamic scheme variants, contrast levels)
  - `TextTheme`, `Typography` (2021), `VisualDensity` (compact on desktop), `MaterialTapTargetSize`
  - every `XThemeData` component theme, `WidgetState`/`WidgetStateProperty`, `Material`, elevation with surface tint, `Ink`, `InkWell`, `InkResponse`, `InkRipple`, `InkHighlight`
  - `Icons` (Material Icons font), `MaterialLocalizations` (English)
- **App structure**: MaterialApp, Scaffold, AppBar, SliverAppBar (+ medium and large), BottomAppBar, NavigationBar, NavigationRail, NavigationDrawer, Drawer, TabBar, TabBarView, TabController, BottomSheet (+ modal), SnackBar, MaterialBanner, ScaffoldMessenger.
- **Actions**: ElevatedButton, FilledButton (+ tonal), OutlinedButton, TextButton, IconButton (all variants), FloatingActionButton (small, large, extended), SegmentedButton, MenuAnchor, MenuBar, SubmenuButton, MenuItemButton, CheckboxMenuButton, RadioMenuButton, PopupMenuButton, DropdownButton, DropdownMenu.
- **Inputs**: TextField, TextFormField, InputDecoration, Checkbox, Radio, RadioGroup, Switch, Slider, RangeSlider, Chips (Input, Filter, Choice, Action, Raw), Autocomplete, SearchAnchor, SearchBar.
- **Pickers and dialogs**: `ShowDatePicker`, `ShowDateRangePicker`, `ShowTimePicker`, CalendarDatePicker, YearPicker, AlertDialog (+ adaptive), SimpleDialog, Dialog (+ fullscreen), `ShowDialog`, `ShowModalBottomSheet`, `ShowMenu`.
- **Display**: Card (+ filled, outlined), ListTile, CheckboxListTile, SwitchListTile, RadioListTile, ExpansionTile, ExpansionPanelList, DataTable, PaginatedDataTable, Divider, VerticalDivider, Badge, Tooltip, LinearProgressIndicator, CircularProgressIndicator, RefreshIndicator, Scrollbar, Stepper, CarouselView, Chip, CircleAvatar, SelectionArea, adaptive text-selection toolbar and desktop context menus.

The order in which these are implemented is given in §18.

---

## 12. API conventions (Flutter → Go)

### 12.1 Constructors (D1)

Every widget and every public framework class is created by a constructor function:

```go
func NewX(<required parameters...>, opts ...XOpts) *X
```

- Flutter's `required` parameters become **positional parameters**, so leaving one out is a compile error.
- Optional parameters live in `XOpts`. You can omit it entirely. Passing more than one `XOpts` value panics with tag `MultipleOptions`.
- Widgets are **immutable after construction**. Their fields are unexported and read through methods: Flutter's `widget.title` becomes `w.Title()`.
- Constructors with **more than 6 required parameters** (e.g. `ThemeData.raw` has 82, `SemanticsData` has 32) are reviewed case by case. By default they aren't public API; the friendlier constructor plus `CopyWith` is offered instead.

### 12.2 Positional order

1. Flutter's required parameters, in declaration order, **except** `child`/`children`/`sliver`/`slivers`, which always come last
2. then `opts ...XOpts`

```go
m.NewPadding(m.NewEdgeInsetsAll(8), child)
m.NewElevatedButton(s.save, m.NewText("Save"))       // onPressed, child
m.NewElevatedButton(nil, m.NewText("Save"))          // nil onPressed = disabled, as in Flutter
m.NewAnimatedContainer(300*time.Millisecond, child, m.AnimatedContainerOpts{Curve: m.Curves.EaseOut})
```

A required-but-nullable Flutter parameter (`required VoidCallback? onPressed`) is still positional. Writing `nil` is the explicit choice Flutter requires.

### 12.3 Options and defaults

- Fields have Flutter's names in Go case (`mainAxisAlignment` → `MainAxisAlignment`). The `key` parameter is the `Key` field of every `XOpts`.
- **A zero value means "Flutter's default"**. Where that isn't possible:
  - A nullable parameter whose `null` behaves differently from Go's zero value becomes `foundation.Opt[T]` (`m.Some(v)`). Examples: `Container.Width` (null = unconstrained, 0 = zero), `TextStyle.Color` (null = inherit), `Text.MaxLines`.
  - A `bool` whose Flutter default is `true` becomes `Opt[bool]` (e.g. `SoftWrap`). Names are never inverted.
  - Every enum's zero value is `<Enum>Unspecified`, which means "this widget's default". Flutter's values follow in declaration order: `MainAxisAlignmentStart`, `MainAxisAlignmentEnd`, ...
- Immutable value classes with many fields (`TextStyle`, `ThemeData`, `BoxDecoration`, `InputDecoration`, `ButtonStyle`) are passed as pointers from their constructors; `nil` means `null`. `CopyWith(XCopy)` takes a generated struct whose fields are all `Opt[T]`, so "not given" and "set to zero" stay distinguishable.
- Value types without required parameters (`Offset`, `Size`, `Rect`, `EdgeInsets`, `Alignment`, `Color`) are plain structs and may be written as literals: `m.EdgeInsets{Left: 8, Right: 8}`.

### 12.4 Naming

| Flutter | Glutter |
|---|---|
| `Text('hi')` | `m.NewText("hi")` |
| named constructor `ListView.builder(...)`, `EdgeInsets.all(8)` | `m.NewListViewBuilder(...)` returning `*ListView`; `m.NewEdgeInsetsAll(8)` |
| static method `Theme.of(context)`, `Navigator.push(context, r)`, `TextStyle.lerp(a, b, t)` | `m.ThemeOf(bc)`, `m.NavigatorPush(bc, r)`, `m.TextStyleLerp(a, b, t)` |
| `maybeOf` | `XMaybeOf(bc) (T, bool)` |
| enum `MainAxisAlignment.center` | `m.MainAxisAlignmentCenter` |
| constants `Colors.blue`, `Icons.add`, `Curves.easeIn` | read-only package values `m.Colors.Blue`, `m.Icons.Add`, `m.Curves.EaseIn` |
| typedefs `VoidCallback`, `ValueChanged<T>`, `WidgetBuilder` | `func()`, `func(T)`, `func(bc BuildContext) Widget` with the same names as types |
| `BuildContext context` | `bc BuildContext` (`ctx` is reserved for `context.Context`) |

### 12.5 Writing your own widgets

Go embedding isn't inheritance, so the framework decides an element's kind from the **interfaces the outer type implements**. The embedded base struct carries the key and a marker:

```go
type CounterPage struct {
	m.StatefulWidget          // marker + key
	title string
}

func NewCounterPage(title string, opts ...m.WidgetOpts) *CounterPage {
	return &CounterPage{StatefulWidget: m.NewStatefulWidgetBase(opts...), title: title}
}

func (p *CounterPage) CreateState() m.State { return &counterPageState{} }

type counterPageState struct {
	m.StateOf[*CounterPage]   // Widget(), SetState, Mounted, Lifetime, Context
	count int
}

func (s *counterPageState) Build(bc m.BuildContext) m.Widget { /* ... */ }
```

- **Lifecycle hooks** are optional interfaces: `InitState()`, `DidChangeDependencies()`, `DidUpdateWidget(old W)`, `Deactivate()`, `Activate()`, `Dispose()`, `Reassemble()`. The framework does its own bookkeeping, so there's nothing like Flutter's `super.initState()` to call.
- **Mixins become embedded structs**: `m.SingleTickerProvider`, `m.TickerProvider`, `m.AutomaticKeepAliveClient`.
- **Render objects** embed `rendering.RenderBox` and call `rendering.InitRenderBox(&r.RenderBox, r)` in their constructor. That gives the base a pointer to the outer object for its "virtual" calls (`PerformLayout`, `Paint`, `HitTestChildren`, ...).
- **Inherited data** uses generics: `m.DependOn[*AppConfig](bc)` and `m.GetInherited[*AppConfig](bc)`. You don't write an `of` function.

### 12.6 Callbacks, async and goroutines

- Futures become goroutines plus `SetState`.
  - `SetState` is safe from any goroutine. On the UI thread it runs immediately, as in Flutter. Elsewhere it's queued for the UI thread. It is silently dropped if the state is no longer mounted.
  - `s.Lifetime()` is a `context.Context` cancelled at `Dispose`.
  - `s.Go(func(ctx context.Context))` starts a goroutine with that context and with panic recovery (§14.2). Prefer it to a bare `go`.
- `FutureBuilder` takes `func(ctx context.Context) (T, error)`, and `StreamBuilder` takes a `<-chan T`. Their snapshots carry `error`.
- Flutter APIs that return a `Future` take a callback or return a channel: `NavigatorPush` returns `<-chan any`, and `ShowDialog[T]` returns `<-chan T`. Each of these is listed in §12.10.

### 12.7 Generics

`Animation[T]`, `Tween[T]` (built-in lerp for `float64`, `int`, `Color`, `Offset`, `Size`, `Rect`, `EdgeInsets`, `BorderRadius`, `Alignment`, `TextStyle`, `Decoration`; any `T` with `Lerp(T, float64) T`), `ValueNotifier[T]`, `ValueKey[T]`, `GlobalKey[S]`, `StateOf[W]`, `FormField[T]`, `DropdownButton[T]`, `PopupMenuButton[T]`, `WidgetStateProperty[T]`.

### 12.8 Operators and cascades

- Operators become methods: `a.Add(b)`, `a.Sub(b)`, `a.Mul(k)`, `a.Div(k)`, `a.TruncDiv(k)`, `a.Mod(k)`, `a.Neg()`, `rect.Shift(o)`, `m.Multiply(n)`. The comparison operators of `Offset` and `Size` are `Lt`, `Le`, `Gt` and `Ge`.
  - `Offset.Scale(sx, sy)` is Flutter's `Offset.scale`, not the `*` operator; the operator is `Mul(k)`.
  - `Size - Size` is `Sub` (gives an `Offset`) and `Size - Offset` is `SubOffset` (gives a `Size`).
  - `Matrix4.Multiply(n)` returns the product `m * n` and leaves `m` alone (Dart's `*`). `MultiplyBy(n)` sets `m = m * n` (vector_math's `multiply`).
  - The statics `Offset.lerp(a, b, t)` with nullable arguments become `OffsetLerp(a, b *Offset, t) *Offset` (and `SizeLerp`, `RectLerp`, `RadiusLerp`, `RRectLerp`, `RSuperellipseLerp`); the non-null case is the method `a.Lerp(b, t)`.
- Cascades (`..`) become separate statements.

### 12.9 Debug-only checks

Flutter's `assert` checks become checks that exist only in debug builds (`foundation.DebugMode`; release builds use the build tag `glutter_release`). What happens when one fails is in §14.2.

### 12.10 Deliberate deviations (kept up to date)

| Flutter | Glutter | Why |
|---|---|---|
| hot reload | none | not possible in Go |
| `const` widgets | none; reuse package-level values | Go has no const constructors |
| `Future` APIs | goroutine/channel/callback (§12.6) | idiomatic Go |
| `widget.x` | `s.Widget().X()` | widgets have unexported fields (§12.1) |
| `Offset.fromDirection(direction, [distance = 1.0])` | `NewOffsetFromDirection(direction, distance)` | package `ui` cannot import `foundation.Opt`; pass `1` for the default |
| `~/` on `Offset`, `Size`, `Radius` throws for infinity and NaN | `TruncDiv` returns the infinity or NaN | no exceptions in Go |
| `Matrix4` is `vector_math_64` (float64) | `ui.Matrix4` has float32 storage (§7); `TransformPoint` and `TransformRect` compute in float64 from it | smaller, and what the GPU wants |
| `RSuperellipse.contains` asks Impeller's `RoundSuperellipse` | an approximation: superellipse of exponent 5 in a box 1.2924 × the radius | Impeller's geometry is not in `reference/`; revisit with the software renderer (step 2.7) |
| Flutter records a C++ `DisplayList` | `ui.DisplayList`: a list of exported `Op...` structs (`OpDrawRect`, `OpClipPath`, ...) that the renderers type-switch on; `internal/displaylist` does not exist | an `internal` package cannot hold ops that use `ui` types without an import cycle |
| `Canvas.drawRawPoints`, `drawRawAtlas` (typed-data variants) | none | Go has slices of `Offset`, `RSTransform` and `Rect` |
| `Picture.toImage` returns a `Future` | `ToImage(width, height) (*Image, error)` and `ToImageSync` (the same) | both renderers draw on the calling goroutine |
| `Canvas.clipRect(rect, doAntiAlias: true)` | `ClipRect(rect, ClipRectOpts{DoAntiAlias: ui.Ptr(false)})`; nil means true | no `foundation.Opt` in `ui` |
| the DisplayListBuilder drops draws outside the clip | every draw is recorded | the renderers clip; culling can be added without changing the API |
| `Path` wraps `SkPath` (float32 points) | `ui.Path` follows `SkPath`'s rules (injected moves, close, arcs, conic building) with float64 points | no float32 round trips; same structure |
| `Path.getBounds` | `GetBounds` is `SkPath::computeFiniteBounds`: the box of all points, control points included, and the empty rectangle if a point is not finite. `GetTightBounds` is the exact box | Flutter's behavior; the exact one is also needed |
| `Path.addRSuperellipse` asks Impeller | corners are 8 cubics per corner on the same superellipse as `RSuperellipse.Contains` | same approximation |
| `PathMetrics` is a lazy `Iterable` | `ComputeMetrics` returns a slice of all contours with a length | simpler; contours are measured when called |
| `Path.arcToPoint(clockwise: true)`, `PathMetric.extractPath(startWithMoveTo: true)` | `Clockwise *bool`, `StartWithMoveTo *bool` in the options; nil means true | no `foundation.Opt` in `ui` |
| (none) | `Path.Flatten`, `Path.Segments`/`ForEachSegment`, `GetTightBounds` | what the renderers need |
| `ColorSpace` starts at `sRGB` | `ColorSpaceSRGB` is the zero value; there is no `ColorSpaceUnspecified` | `Color` is a literal-able value type, and the zero value must be sRGB |
| `Paint` properties (`paint.color = c`) | accessors `p.Color()` / `p.SetColor(c)`; `Paint{}` is Flutter's `Paint()` | the defaults (opaque black, anti-alias on, srcOver, miter limit 4) are not Go zero values, so the fields are unexported |
| `Scene` is a native C++ layer tree | `ui.Scene` holds an immutable tree of exported `Layer` types (`ContainerLayer`, `TransformLayer`, `ClipRectLayer`, `OpacityLayer`, `PictureLayer`, ...) that the renderers type-switch on | same reason as `ui.DisplayList`; Flutter's `pushOffset` and `pushTransform` both give a `TransformLayer` here |
| `SceneBuilder.pushX(..., oldLayer: x, clipBehavior: Clip.antiAlias)` | `PushX(args, PushXOpts{OldLayer, ClipBehavior})`; `OldLayer` is the typed handle (`*OpacityEngineLayer`, ...) and the zero `ClipBehavior` is `ClipAntiAlias` | options structs (§12) |
| `SceneBuilder.addPicture` | a nil picture is ignored; `Pop` at the root is ignored; using the builder after `Build` panics with `SceneBuilderBuilt`; `Scene.ToImage` is synchronous and `NotImplemented` until step 2.10 | programmer errors follow §14 |
| Skia's AA rasterizer unions overlapping shapes exactly | `soft` accumulates signed area per pixel and folds it with the fill rule, so coverage is exact where one edge crosses a pixel and approximate where several do (the coverage of coincident edges adds up, e.g. a stroke that retraces itself) | the golden renderer needs deterministic, simple, fast coverage; the difference is below the golden tolerance for real scenes |
| `Paint.strokeWidth == 0` is a hairline | a hairline is one device pixel wide with the paint's caps and joins (no special hairline rasterizer) | simple and deterministic |
| `Image` wraps an engine texture; `decodeImageFromPixels` takes a callback | `ui.Image` holds premultiplied RGBA8 pixels shared by its clones (`Width`, `Height`, `Pixels`, `Clone`, `Dispose`); `DecodeImageFromPixels(ctx, pixels, w, h, format, opts)` returns the image, with `rgba8888` and `bgra8888` only; `NewImageFromRGBA` wraps pixels for the renderers | the software renderer needs the pixels (step 2.7 introduced the core of what the plan gives to 2.10) |
| Skia clips are rasterized per primitive | `soft` keeps a clip as a pixel rectangle plus an optional coverage mask; an anti-aliased clip multiplies coverage (so two anti-aliased edges in one pixel multiply instead of taking the exact intersection); a clip without anti-aliasing keeps the pixels whose center is inside, with a center on the edge counting as inside | simple and deterministic |
| `drawVertices` blends the vertex colors with the paint by `blendMode` | the same, with the paint (its shader at the interpolated texture coordinates, or its color) as the source and the vertex colors as the destination, then the paint's color filters and its own blend mode; triangles are not anti-aliased | Skia's behavior as documented in `Canvas.drawVertices` |
| Skia dithers gradients when asked | `soft` always dithers gradient shaders with an 8x8 ordered matrix (at most one level of 8 bits) | smooth ramps; deterministic for goldens; the GLES shaders do the same |
| `drawAtlas` colors and `blendMode` | the sprite is the source and the color the destination (so `srcIn` with a transparent white scales the sprite's alpha); the default `src` leaves the sprite as it is | Flutter's documented example |
| `Paint.shader` on `drawImage*` | ignored: the image is the source, as in Skia | the paint's alpha, color filter, invert, blend mode, filter quality and anti-alias still apply |
| `Canvas.drawShadow` is Skia's `SkShadowUtils` (tessellated umbra and penumbra) | `soft` blurs the shape's coverage: an ambient shadow (blur radius `0.5 * elevation`, alpha 0.039) and a spot shadow (the shape grown by `600 / (600 - e)`, moved down by `e * (600 + h/2) / (600 - e)`, blur radius `800 * e / (600 - e)`, alpha 0.25), both in device pixels with `e` scaled by the transform's scale; Skia's sigma from radius `0.57735 r + 0.5`; an opaque occluder is cut out of its shadow | same extent and falloff as Flutter's Linux look, not the same pixels; Material elevation shadows are tuned in step 12.3 |
| `MaskFilter.blur` and image-filter sigmas are in logical pixels | `soft` scales them by the transform's scale (separately per axis for image filters); blurs are exact Gaussians cut at 3 sigma, with no downsampling | simple and deterministic; the GLES renderer downsamples large sigmas (step 3.x) |
| An image filter on a `Paint` filters the shape's layer | the shape is drawn into a layer without the filter, the layer is filtered, and only the pixels the filtered layer has something in are composited with the paint's blend mode | Skia's layer is as big as the shape and its blur, so `clear` does not wipe the whole clip |
| `Picture.toImage` and `Scene.toImage` draw with the engine's native rasterizer | `ToImage(width, height, ToImageOpts{Rasterizer})`: a `ui.Rasterizer` (`RasterizeScene(scene, w, h) (*Image, error)`) is given by the caller, which gets it from its binding; `soft.ImageRasterizer` is the software one; with none the error is `RasterizerMissing` | `ui` cannot import the renderers (they import `ui`), and a global registry would be global mutable state |
| `instantiateImageCodec` is asynchronous and decodes lazily; JPEG EXIF orientation is applied | `InstantiateImageCodec(ctx, data, opts)` decodes every frame at once and `GetNextFrame` returns them in turn (wrapping); EXIF orientation is not applied; an image over 67 million pixels fails with `ImageDecodeFailed`; a GIF frame lasting 10 ms or less lasts 100 ms; WebP is not animated | decoders from the standard library and `x/image`; bounded memory for hostile files |
| `ImageByteFormat.rawExtendedRgba128` | not provided; `Image` is 8 bits per channel | there is no wide-gamut image yet |
| nullable `double`/`Color` arguments, as in `Color.withValues(alpha: ...)` | `ColorWithValuesOpts{Alpha: ui.Ptr(0.5)}` | package `ui` cannot import `foundation.Opt`; `ui.Ptr` makes the pointer |
| `ColorFilter`, `ImageFilter` and `MaskFilter` compare with `==` | `a.Equal(b)` | pointers to immutable values |
| `Gradient`'s and `ImageShader`'s `ArgumentError`s | a panic with the tag `GradientColorCountInvalid` or `GradientStopCountMismatch` | Go has no exceptions (§14.2) |
| Skia draws shapes with exact analytic coverage | `gles` draws rect, rrect, oval, circle, superellipse, DRRect, lines and points from a signed distance field: coverage is `0.5 - d` in pixels, exact along straight edges and a first-order estimate at elliptical and superellipse corners; an axis-aligned rectangle uses the exact box area; the GPU differs from `soft` by at most about 60 of 255 on a few edge pixels (the display tests state the bounds) | no multisampling, one draw call per batch (§9.1) |
| A stroke thinner than a pixel is a hairline with the width as alpha | the same: one pixel wide, alpha scaled by the width; a hairline is one full pixel | Skia does this too; `soft` draws the geometry |
| `drawDRRect` is the even-odd fill of both outlines | `gles` draws the outer rounded rectangle minus the inner one, which differs only when the inner one is not inside the outer one | one shader branch; the case is not used by Material |
| Nested anti-aliased rectangle clips multiply their coverage | `gpu.Flatten` folds axis-aligned rectangle clips into one rectangle with fractional edges (the intersection), applied exactly in the shader and as the scissor box; a clip that is not an axis-aligned rectangle stays a `ClipShape` for the stencil or mask clips of step 3.3 | one scissor and one clip rectangle per draw |
| The engine draws whatever the scene has | until steps 3.3 to 3.5 the GPU renderer returns `GLNotImplemented` (naming the feature) for paths, images, shaders, color and image filters, mask filters, layers, shape clips and blend modes other than srcOver, dstOver and plus; `RendererOpts.SkipUnsupported` leaves them out and lists them in `LastFrame` | development examples show what works step by step; the option stays for them |
| The engine clears the frame to the scene's background (the embedder's clear color) | `ui.Scene` has no background; `gles.NewRenderer(RendererOpts{Background, HasBackground})` clears each frame to one color (default opaque white), and `Snapshot` is `GLNotImplemented` until step 3.6 | the scene is a layer tree only; the window background is the renderer's setting |

### 12.11 One import for apps

Flutter's `material.dart` re-exports `widgets.dart`. Glutter gets the same effect with `cmd/glutter-gen reexport`, which writes `material/widgets_reexport_gen.go`:
- type aliases (generic aliases included)
- thin wrapper functions, which the compiler inlines

Apps import only `m "github.com/rm4n0s/glutter/material"`. The widgets layer itself never imports `material`.

### 12.12 Reference example

```go
package main

import (
	"log"
	"strconv"

	m "github.com/rm4n0s/glutter/material"
)

func main() {
	if err := m.RunApp(NewMyApp()); err != nil {
		log.Fatal(err)
	}
}

type MyApp struct{ m.StatelessWidget }

func NewMyApp() *MyApp { return &MyApp{} }

func (a *MyApp) Build(bc m.BuildContext) m.Widget {
	return m.NewMaterialApp(m.MaterialAppOpts{
		Title: "Counter",
		Theme: m.NewThemeData(m.ThemeDataOpts{ColorSchemeSeed: m.Some(m.Colors.Blue)}),
		Home:  NewCounterPage("Counter"),
	})
}

type CounterPage struct {
	m.StatefulWidget
	title string
}

func NewCounterPage(title string, opts ...m.WidgetOpts) *CounterPage {
	return &CounterPage{StatefulWidget: m.NewStatefulWidgetBase(opts...), title: title}
}

func (p *CounterPage) Title() string     { return p.title }
func (p *CounterPage) CreateState() m.State { return &counterPageState{} }

type counterPageState struct {
	m.StateOf[*CounterPage]
	count int
}

func (s *counterPageState) increment() { s.SetState(func() { s.count++ }) }

func (s *counterPageState) Build(bc m.BuildContext) m.Widget {
	return m.NewScaffold(m.ScaffoldOpts{
		AppBar: m.NewAppBar(m.AppBarOpts{Title: m.NewText(s.Widget().Title())}),
		Body: m.NewCenter(
			m.NewColumn([]m.Widget{
				m.NewText("You have pushed the button this many times:"),
				m.NewText(strconv.Itoa(s.count), m.TextOpts{Style: m.ThemeOf(bc).TextTheme().HeadlineMedium()}),
			}, m.ColumnOpts{MainAxisAlignment: m.MainAxisAlignmentCenter}),
		),
		FloatingActionButton: m.NewFloatingActionButton(s.increment, m.NewIcon(m.Icons.Add),
			m.FloatingActionButtonOpts{Tooltip: "Increment"}),
	})
}
```

---

## 13. Threading and lifecycle

- `m.RunApp(app Widget, opts ...RunAppOpts) error` must run on the main goroutine (§8.4). It returns when the last window closes or after `SystemNavigatorPop`. Environment failures come back as tagged errors (§14.1).
- `RunAppOpts` include:
  - `Assets fs.FS` for the `AssetBundle`
  - `AppID`, `Window` (the first window's config)
  - `Backend` (overrides `GLUTTER_BACKEND`)
  - `ErrorReporter` (§14.2)
  - `Logger *slog.Logger`
- The UI thread owns the element tree, render tree, `ChangeNotifier`s, controllers and GL. From other goroutines, only these are safe:
  - `State.SetState`
  - `WidgetsBindingOf(bc).Post(fn)`
  - the `desktop` service calls (they take a `context.Context` and deliver results with `Post`)
- `debug` builds detect calls made from the wrong goroutine on the main framework entry points and report them with tag `WrongThread`.
- App lifecycle maps to `AppLifecycleState`:
  - `Resumed`: focused
  - `Inactive`: visible but unfocused
  - `Hidden`: minimized or all windows hidden
  - `Detached`: shutting down
  - `Paused` is not used on desktop, as in Flutter's desktop embedders.

## 14. Errors and diagnostics

### 14.1 Environment errors

These are returned from `RunApp`, image/font loading and `desktop` calls, always with tags from `rm4n0s/errors`. The initial tags are:
- `LibraryLoadFailed`, `NoDisplay`, `BackendUnknown`, `BackendUnavailable`, `WaylandConnectFailed`, `WaylandProtocolMissing`, `WaylandProtocolError`, `X11OpenDisplayFailed`, `IdleInhibitUnavailable`
- `EGLLoadFailed`, `EGLInitFailed`, `EGLNoConfig`, `GLES3Unsupported`, `ShaderCompileFailed`
- `AssetNotFound`, `ImageDecodeFailed`, `FontLoadFailed`
- `DBusConnectFailed`, `TrayUnavailable`, `NotificationsUnavailable`, `FileChooserUnavailable`, `OpenURIUnavailable`
- `MultipleOptions`, `WrongThread`

Every tag has a route test.

### 14.2 Programmer errors and panics in app code (D15)

Glutter follows Flutter. Flutter reports failed asserts and exceptions through `FlutterError.onError`, replaces the failing subtree with `ErrorWidget`, and keeps the app running.

- **Signaling**: `Build`, layout, paint and callbacks never return errors. Bugs panic: Go runtime panics, and failed debug checks, which panic with a tagged `*errors.Error` (e.g. `SetStateAfterDispose`).
- **Recovery is done by the library.** App code never writes `recover()`. The framework recovers at the same points where Flutter catches exceptions:

  | Recovery point | Flutter equivalent |
  |---|---|
  | `Build` of every element | `ComponentElement.performRebuild` |
  | layout and paint of every render object | `RenderObject._reportException` |
  | gesture callbacks | `GestureRecognizer.invokeCallback` |
  | `Listenable` and animation listeners | `ChangeNotifier.notifyListeners` |
  | scheduler frame/post-frame callbacks and timers | `SchedulerBinding` |
  | image and font stream listeners | `ImageStreamCompleter` |

- **Reporting**: each recovered panic becomes a tagged error (`BuildPanicked`, `LayoutPanicked`, `PaintPanicked`, `CallbackPanicked`) carrying the panic value, the stack and the element path. It goes to `RunAppOpts.ErrorReporter func(error)`. The default reporter logs `appErr.ToJson()` through the app's `slog.Logger`. An app can forward errors to its own service, or call `os.Exit` to crash instead.
- **Display**: the failing subtree is replaced by `RunAppOpts.ErrorWidgetBuilder func(error) Widget`. The default is Flutter's: a red box with the message in debug builds, a plain gray box in release builds.
- **Goroutines**: Go can't catch a panic in a goroutine started with a bare `go` statement, so it crashes the program. Use `State.Go(func(ctx context.Context))`, which runs the function with the state's `Lifetime()` context under the same recovery and reporting.
- **Tests**: `gluttertest` fails a test on any reported error unless the test takes it with `tester.TakeError()`, Flutter's `takeException()`.
- **Constructors** never return errors or error widgets (D1).

### 14.3 Debugging aids

- Flags in `rendering`/`widgets`: `DebugPaintSizeEnabled`, `DebugPaintBaselinesEnabled`, `DebugPaintLayerBordersEnabled`, `DebugRepaintRainbowEnabled`, `DebugPrintRebuildDirtyWidgets`, `DebugPrintLayouts`, `DebugProfileBuildsEnabled`. These are fields of a `DebugOptions` value passed through `RunAppOpts`, not global variables.
- `DebugDumpApp`, `DebugDumpRenderTree`, `DebugDumpLayerTree`, `DebugDumpSemanticsTree`, `DebugDumpFocusTree`.
- A performance overlay (frame build and raster time graphs).
- `runtime/trace` regions around the frame phases.

---

## 15. Performance targets

These are measured on a 2020 laptop with an Intel Iris Xe or similar GPU and Mesa, at 1920×1080 and a device pixel ratio of 1.25.

| Metric | Target |
|---|---|
| Idle CPU (no animation, no input) | ~0%: no frames scheduled, no timers armed |
| Frame rate (gallery screen, ~500 visible widgets, one animation) | the display's refresh rate with no dropped frames |
| UI thread CPU per frame, animation-only frame (no rebuild) | < 2 ms |
| UI thread CPU per frame, full rebuild of the gallery screen | < 8 ms |
| Steady-state allocations, animation-only frame | ≤ 1 KB per frame |
| Cold start to first frame | < 150 ms (fonts cached) |
| Resident memory, counter example | < 60 MB |

Benchmarks (`go test -bench`) in `rendering`, `widgets` and `internal/gpu/soft` track these numbers. Regressions above 10% need a justification.

## 16. Build and runtime requirements

**Building an app**: Go 1.27+ and nothing else. `CGO_ENABLED=0 go build` produces a binary that loads the system libraries at runtime. It's dynamically linked against glibc's loader, which is how purego reaches `dlopen`. Cross-compiling for linux/amd64 and linux/arm64 from any OS works.

**Contributing**: Go 1.27+ plus a C compiler, **only** because `go test -race` requires cgo. When cgo is enabled, purego uses it transparently, so the same code is tested both ways (§17). No `-devel` packages, `wayland-scanner` or `wayland-protocols` are needed.

**Runtime**:
- **Required**:
  - a Wayland compositor with `xdg_wm_base`, or an X server
  - an EGL driver with OpenGL ES 3.0 (Mesa ≥ 20, NVIDIA through glvnd, or llvmpipe)
  - the runtime libraries in §8.1. They're installed on every Linux desktop (Wayland compositors and Xwayland depend on them), and a missing one is reported as a tagged error.
- **Optional**:
  - a D-Bus session bus, which every desktop session has; all the services below use it
  - xdg-desktop-portal plus a backend, for file dialogs, settings and open URI
  - a StatusNotifierItem host, for the tray (KDE, XFCE, Cinnamon, and GNOME with the AppIndicator extension)
  - a notification daemon
  - at-spi2-core, for accessibility

## 17. Testing strategy

- **Unit tests** are table-driven and sit next to the code (CLAUDE.md).
- **Widget tests** (`gluttertest`):
  - `t := gluttertest.New(tb)`, then `PumpWidget`, `Pump(d)`, `PumpAndSettle`, `Tap`, `LongPress`, `Drag`, `Fling`, `Hover`, `Scroll`, `EnterText`, `SendKeyEvent`, `SendKeyDownEvent`/`UpEvent`, `PumpFrames`
  - finders: `ByText`, `ByKey`, `ByType[T]`, `ByIcon`, `ByTooltip`, `BySemanticsLabel`, `Descendant`, `Ancestor`
  - assertions are plain functions returning errors for `t.Error`
  - a fake clock drives tickers and timers
  - the `fake` platform scripts windows, input and settings
- **Golden tests** compare against PNGs in `testdata/goldens/`, rendered by `internal/gpu/soft`. `-update-goldens` rewrites them.
- **Ported Flutter tests**: for each widget and render object, the relevant cases from Flutter's `packages/flutter/test` are ported as table tests. This is the main way to check fidelity.
- **Platform tests** (`-tags glutter_display`) open real Wayland/X11 windows and GL contexts (they can run under a headless compositor such as `weston --backend=headless` or `Xvfb`), and compare `gles` against `soft`.
- **Wayland compatibility matrix.**
  - Every milestone that touches the platform is checked against:
    - compositors: GNOME/Mutter, KDE/KWin, Sway, Hyprland, Weston and COSMIC
    - drivers: Mesa (Intel, AMD), NVIDIA proprietary, and llvmpipe
    - scales: 100%, 125%, 150% and 200%
    - a two-monitor setup with mixed scales
  - The automated part runs under headless compositors (`weston --backend=headless`, `sway` with `WLR_BACKENDS=headless`, `mutter --headless`, `kwin_wayland --virtual`).
  - The rest is a manual checklist in `docs/IMPLEMENTATION.md` covering every item in §8.11.
- **D-Bus services** are tested against an in-process fake bus that records method calls and emits signals. Real-desktop checks are listed per milestone in `docs/IMPLEMENTATION.md`.
- **Both build modes**:
  - `go test -race ./...` runs with cgo (purego uses the real cgo runtime)
  - `CGO_ENABLED=0 go test ./...` runs purego's cgo-free path, which users get
  - both must pass
- **C struct layout tests** for every Go mirror of a C struct (§8.1 rule 3).
- **Fuzz tests** cover the X11/Wayland event decoders, the XKB compose handling, the paragraph layout, and the image and font loaders.

## 18. Milestones

Each milestone is broken into steps in `docs/IMPLEMENTATION.md` (written after this spec is approved) and follows the workflow in CLAUDE.md.

| # | Milestone | Done when |
|---|---|---|
| M0 | Skeleton: `go.mod`, package layout, `cmd/glutter-gen` stub, `reference/` fetch script, `third_party/flutter/LICENSE` | `go build ./...` and all four checks pass |
| M1 | Platform core: purego library loaders and C struct mirrors, Wayland binding generator, Wayland and X11 window, event loop, EGL, GLES clear, frame callbacks, pointer and keyboard (xkb), fake platform | a window opens and closes and logs input on both backends |
| M2 | `ui` engine + software renderer: geometry, Path, Paint, Canvas, Picture, SceneBuilder, analytic shapes, shadows, gradients, clips, layers, golden harness | goldens for every `Canvas` operation |
| M3 | GLES renderer to parity with `soft`, plus atlases and batching | `gles` matches the `soft` goldens within tolerance |
| M4 | Text: font collection, fontscan, Roboto embedded, Paragraph with every query, glyph atlas | text goldens, including bidi, ellipsis and decorations |
| M5 | foundation, scheduler, physics, gestures (hit testing, arena, every recognizer) | ported Flutter gesture tests pass |
| M6 | rendering core: RenderObject, RenderBox, PipelineOwner, layers, compositing, every box render object | ported Flutter rendering tests pass |
| M7 | widgets core: Element tree, State, keys, GlobalKey, InheritedWidget family, basic widgets, MediaQuery, `RunApp`, `gluttertest` | counter example runs on Linux |
| M8 | animation library, implicit and explicit animations, transitions, Hero | ported animation tests pass |
| M9 | scrolling: Scrollable, physics, ListView/GridView/CustomScrollView, slivers, PageView, NestedScrollView, scrollbars, reorderable lists | ported scrolling tests pass |
| M10 | navigation and focus: Navigator (both APIs), Router, Overlay, OverlayPortal, focus system, Shortcuts/Actions | ported tests pass |
| M11 | text editing: EditableText, selection, Linux shortcuts, undo, IME (text-input-v3 on Wayland; IBus/Fcitx5 over D-Bus on X11), clipboard, SelectableRegion | editing works with IBus and Fcitx5 on both backends |
| M12 | Material foundation: ThemeData, ColorScheme (+ HCT port), Typography, Material, Ink, icons, localizations | theme goldens |
| M13 | Material components, in batches: app structure → actions → inputs → dialogs and pickers → display | gallery example covers §11.2 |
| M14 | desktop windowing: multiple windows, dialog windows, decorations (server-side, libdecor, app-drawn), cursor shapes, fractional scale, settings, drag and drop, every item in §8.11 | multiwindow example; drag and drop with Files/Dolphin; the Wayland matrix (§17) passes |
| M15 | D-Bus services: tray, notifications, file dialogs, open URI | tray example on KDE and GNOME+AppIndicator |
| M16 | accessibility: semantics → AT-SPI2 | gallery usable with Orca |
| M17 | effects and performance: backdrop filter, advanced blend modes, raster cache, performance overlay, benchmarks meeting §15 | targets met |
| M18 | 1.0: documentation, examples, API review against §12 | release |

## 19. Examples (`examples/`)

`counter`, `layout` (Row/Column/Stack/Wrap), `animations`, `scrolling` (lists, grids, slivers), `navigation`, `editor` (text editing, clipboard, IME), `gallery` (all Material components, light/dark), `multiwindow`, `tray` (tray + notifications + file dialog), `custompaint`.

## 20. Licensing and third-party code

| Component | License | Notes |
|---|---|---|
| Glutter | BSD-3-Clause | |
| Flutter framework (API, algorithms and tests ported from it) | BSD-3-Clause | `third_party/flutter/LICENSE` plus a NOTICE entry; ported files carry `// Flutter:` references |
| go-text/typesetting | Unlicense OR BSD-3-Clause | Go dependency |
| golang.org/x/image | BSD-3-Clause | Go dependency (WebP, BMP) |
| godbus/dbus/v5 | BSD-2-Clause | Go dependency (D-Bus) |
| ebitengine/purego | Apache-2.0 | Go dependency (calling C libraries without cgo); NOTICE entry |
| rm4n0s/errors | (author's) | Go dependency |
| Roboto, Material Icons | Apache-2.0 / OFL | embedded in `material/fonts` with their licenses |
| material-color-utilities (ported) | Apache-2.0 | NOTICE entry |
| Wayland protocol XML | MIT | kept with the generated code |
| Flutter key data (generated key tables) | BSD-3-Clause | NOTICE entry |

**Reference sources.** `reference/flutter` holds a sparse checkout of `packages/flutter/lib` and `packages/flutter/test`, pinned to the latest stable Flutter tag when M0 starts (the tag is recorded in `reference/VERSION`). It is git-ignored and fetched by `scripts/fetch-reference.sh`.

## 21. Open decisions

None at the moment. New ones are added here with a recommendation and moved to §4 once decided.

## 22. Risks

| Risk | Mitigation |
|---|---|
| Without cgo, purego imitates the cgo runtime (`internal/fakecgo`), which depends on Go runtime internals and can break with a new Go release | Pin the purego version. Run both test modes (§17) on every new Go release before raising the minimum Go version. Ebiten depends on the same mechanism, so breakage is found and fixed upstream quickly |
| A wrong Go mirror of a C struct corrupts memory silently | layout tests (§8.1 rule 3); fuzz tests on event decoding |
| The C→Go callback limit (~2,000 per process) | callbacks are created once at startup and dispatch by user data (§8.1 rule 2); a test counts them |
| C variadic APIs can't be called (XIM, some Xlib calls) | non-variadic variants; IME over D-Bus on X11 |
| purego callbacks can't take structs by value | none of the libraries used need that (checked per library in M1) |
| libdecor's GTK plugin loads GTK 3 into the process (more memory, slower first window) | load it only on compositors without server-side decorations; `DecorationsApp` skips it; measure against §15 in M14 |
