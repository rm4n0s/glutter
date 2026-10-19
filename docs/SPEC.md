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
| `wp_fractional_scale_v1` + `wp_viewporter` | optional | integer `wl_surface.set_buffer_scale` (from `wl_surface.preferred_buffer_scale`, else the scale of the outputs the surface is on) |
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
| BoxShadow / `drawShadow` on rects and rrects | Analytic Gaussian-blurred rrect in closed form, with no offscreen pass. That is the shape of every `BoxShadow` (`MaskFilter.blur` on a rect, rrect, oval or circle under a transform that scales and moves). `Canvas.drawShadow` and every other shape with a mask filter use the blurred layer below (see §12.10) |
| General paths | Flattened on the CPU and rasterized into a **coverage atlas** (8-bit, cached by path + transform), using the same rasterizer as the software backend; then composited on the GPU. Paths larger than an atlas tile use a tiled mask. A GPU tessellation path may be added later if profiling shows a need |
| Text | Glyph outlines from go-text as paths with their origin on a quarter of a pixel horizontally and a pixel vertically (when the transform only scales and moves), so each glyph at a size has four coverage tiles in the path coverage atlas (§12.10). Color glyphs: COLR v0 and v1 as pictures; CBDT, sbix and SVG are not drawn |
| Images | Textures with mipmaps for downscaling (`FilterQuality`). Nine-patch and atlas draws are batched |
| Gradients | Linear, radial, two-point conical and sweep in the shader; all tile modes |
| Clips | Rects through the scissor test. RRects and paths through a stencil buffer, or antialiased through coverage masks (Flutter clips are antialiased by default) |
| Layers | `SaveLayer`, opacity, color/image filters, backdrop filter and shader mask render into pooled offscreen framebuffers |
| Blur | Separable Gaussian with downsampling for large sigma |
| Blend modes | `srcOver`, `dstOver`, `plus` and `dstOut` through `glBlendFunc`; every other mode in a shader that reads a copy of the destination (`KHR_blend_equation_advanced` is not used, see §12.10) |
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
| `MaskFilter.blur` and image-filter sigmas are in logical pixels | `soft` scales them by the transform's scale (separately per axis for image filters); blurs are exact Gaussians cut at 3 sigma, with no downsampling. `gles` does the same sums in two passes, one along each axis, over the layer's pixels; when a sigma in pixels is over 8 the picture is first averaged down by a power of two along that axis, blurred there and scaled up bilinearly, so the kernel never has more than 49 taps (the result differs from the exact one by at most a few levels, see the display tests) | simple and deterministic; large blurs stay cheap on the GPU |
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
| The engine draws whatever the scene has | the GPU renderer returns `GLNotImplemented` (naming the feature) for gradients with more than 32 colors, perspective transforms, blend modes that do not exist, and (without the caches of a renderer) the shapes that need them; `RendererOpts.SkipUnsupported` leaves them out and lists them in `LastFrame` | development examples show what works step by step; the option stays for them |
| The advanced blend modes use `KHR_blend_equation_advanced` when present | not used: `srcOver`, `dstOver` and `plus` go through `glBlendFunc`, and every other mode (all 29 of `BlendMode`, for paints, layers, color filters, vertices and atlas sprites) is computed in the fragment shader (`shaders/blend.glsl`, a port of `soft/blend.go`) from a copy of the pixels under the draw, made with `glCopyTexSubImage2D` into a pooled texture; such a draw is its own call, so that it sees the draws before it | one path that gives the same pixels on every driver, which the extension may not (it is not on every driver and each rounds in its own way); the cost is a copy of the draw's bounds per draw, which only the rare modes pay |
| Gradients and image shaders are shader programs of the engine | the paint's shader, color filter, inverted colors and blend mode are uniforms of the draw call (`paintSpec`, interned so that draws with one paint share a call): up to 32 color stops (Skia's ramp rules, interpolated without premultiplied alpha), linear, radial, two-point conical (a port of `soft`'s `conicalFunc`), sweep, and image shaders (`texelFetch` with every tile mode, nearest, bilinear or Mitchell bicubic, the mip level chosen on the CPU); gradients get the same 8x8 ordered dither as `soft`; color filters (mode, matrix, gamma curves) and the inversion are applied to the paint's color after its alpha, as `soft` does | no per-paint shader programs to compile; `soft` and `gles` run the same formulas |
| Layers composite with the paint of the `SaveLayer`, the alpha of an opacity layer or the filter of a color filter layer | each layer is drawn into a pooled RGBA8 framebuffer that covers the part of the surface it can touch (the clip's bounds, the `SaveLayer` bounds and the layer it is in), cleared to transparent when it starts, and drawn into its parent by the `layer` pipeline with the alpha, color filter, inverted colors and blend mode, through the clip the layer was begun with (a clip layer with a save layer clips the composite, and its children too, as `soft` does); a layer that shows nothing has no framebuffer | pooling by size keeps the cost to one clear per layer |
| Image filters, backdrop filters and shader masks are Skia's | `gles` draws them with the same rules as `soft`: an `ImageFilterLayer`, a `SaveLayer` or a paint with an image filter filters the layer's pixels in passes (`filters.go`, `shaders/filter.*`: Gaussian blur, dilate and erode, matrix, color filter, compose) before compositing them, over the clip's bounds (a paint's filtered layer composites only where it has drawn something); a blur with no tile mode is transparent beyond the edge for an `ImageFilterLayer`, clamps for a `SaveLayer` or a paint, and mirrors for a backdrop, as in `soft`; a `BackdropFilterLayer` starts as a copy of the pixels under it (`glCopyTexSubImage2D`) filtered with the mirror tile mode by default; a `ShaderMaskLayer` paints its shader over the mask rectangle into the layer with its blend mode before compositing; every pass reads the texture before it with `texelFetch` and draws into a pooled framebuffer | one code path for every filter; the layer is as big as the clip, as in `soft` |
| `MaskFilter.blur` and `Canvas.drawShadow` are drawn by Skia's blur and `SkShadowUtils` | `gles` draws a `MaskFilter.blur` on a filled rect, rrect, oval or circle under a transform that only scales and moves in one pass, from the closed form of the Gaussian blur of a rounded rectangle (`shaders/shadow.glsl`: normal distribution functions across the rows, the integral down the rows exact for the straight sides and 16 samples over each corner zone); any other shape (paths, strokes, rotated shapes) is drawn white into a layer that covers the shape's bounds plus 3 sigma, which is blurred, and whose alpha is then the coverage that the paint (its shader, filters and blend mode) fills, in the blur style (normal, solid, outer, inner). `Canvas.drawShadow` is drawn as `soft` does it, from two such layers (ambient and spot) in a group from which an opaque shape is cut out with `dstOut`; it does not use the closed form, because the path is not known to be a rounded rectangle | the closed form covers `BoxShadow`; the rest keeps one rule for every shape; Material shadows are tuned in step 12.3 |
| The scale of a window is its output's scale | a window's scale is, in order of preference: `wp_fractional_scale_v1`'s preferred scale (in 120ths, used with a `wp_viewport` whose destination is the logical size; the buffer is `ceil(size * scale / 120)` pixels), the surface's `preferred_buffer_scale` (`wl_surface` v6), the largest `wl_output.scale` of the outputs the surface has entered, then 1; the integer scales use `wl_surface.set_buffer_scale`. When the scale changes (a window moved to another monitor, or the compositor's first `preferred_scale`, which comes after the first buffer on some compositors) the window resizes its `wl_egl_window` and draws a frame at once, in the same commit as the new viewport, so no frame of the old scale is shown after the event. `ViewMetrics` has the buffer's size and the exact scale | Flutter's Linux embedder is integer-only; SPEC §8.11 asks for fractional sharpness |
| Input is delivered as it arrives | the engine queues the input that arrives between two frames (and asks for a frame) and handles all of it, in order, at the start of the next frame, before the scene is built (`internal/engine/frames.go`); the shared-memory mode, which has no frames, handles it at once | a burst of input costs one frame (SPEC §8.11, "Frame pacing") |
| A hidden window keeps its frame callbacks | a suspended Wayland window (the `suspended` state of `xdg_toplevel` v6) renders nothing and keeps a frame request pending; when it is shown again a frame is drawn at once with the time of now, and the frame callback that was in flight before, which may never come or come late, is ignored | tickers resume at the time of the next frame (SPEC §8.11) |
| `Picture.toImage` rasterizes with the engine, with no window | `gles.Renderer.Snapshot` (and `RasterizeScene`, which makes `Renderer` a `ui.Rasterizer`) draws the scene over transparent pixels into a pooled framebuffer and reads it back, premultiplied, top row first; it needs the renderer's GL context, so a surface must have been attached once (it is made current with a live surface, or without any, by `EGL_KHR_surfaceless_context`); without an attach it fails with `RendererNotAttached` | the context is created from a window's display (§8.2), and every window has one |
| A `RepaintBoundary` subtree that is unchanged for 3 frames is cached as a texture | the groundwork: `RendererOpts.RasterCacheAfter` (0, the default, is off) caches a `TransformLayer` or `ContainerLayer` (the layers a `RepaintBoundary` makes, and never the scene's root) that has been in that many consecutive scenes, as `SceneBuilder.AddRetained` keeps it. `gpu.FlattenOpts.Boundary` makes such a layer a group whose children are drawn under a clip of whole pixels with none of the outer clip's shapes, into a framebuffer that the cache keeps instead of returning to the pool, keyed by the layer, the transform of its contents and the pixels its clip's bounds cover; later frames with the same key draw that texture as one instance, and the commands inside it count as drawn. The group is composited under the clip it is in, so the clip's shapes and fractional edges apply to the group as a whole, as with a save layer (a rounded clip's anti-aliased edge differs from the uncached drawing where content overlaps there); an entry is evicted after 120 unused frames or, least recently used first, when the entries hold over 64 MiB; snapshots do not use it. The 3-frame rule for `RepaintBoundary`, and the rest of the policy, come with step 17.2 | the mechanism comes first, the rule that decides what is cached is a policy of its own |
| General paths are drawn by Skia's scan converter or tessellated by Impeller | `gles` rasterizes a path with the software rasterizer (`soft.OpMask`, the same coverage as `soft`) into 8-bit tiles of at most 256 x 256 pixels of a coverage atlas (2048 x 2048 pages, at most 8, slots of 32 to 256 pixels, least recently used evicted but never what the frame uses; `AtlasFull` when a frame needs more), cached by the geometry, the transform's linear part, its fractional translation and the stroke, so a path drawn again at another whole-pixel offset, or only partly visible, finds its tiles; coverage is quantized to 8 bits | the plan's coverage atlas (SPEC §9.1); the CPU work is paid once per shape and transform |
| Clips are applied by Skia per primitive | a clip that is not an axis-aligned rectangle (rounded rectangles, superellipses, paths, rotated rectangles, differences) is rasterized with `soft.ClipMask` for each shape, multiplied, and cached as an 8-bit texture over the clip's bounds; draws multiply their coverage by it, as `soft` does, so two anti-aliased edges in one pixel multiply | clips antialiased through coverage masks (SPEC §9.1); no stencil buffer |
| Images are sampled by Skia with mipmaps from the engine's own downscaler | the texture's mipmaps are made by `glGenerateMipmap`, the level for medium and high quality is chosen on the CPU as in `soft`, and bicubic filtering (Mitchell) is done in the shader; odd-sized images differ from `soft` in the last level by a rounding | the GPU makes the chain; the pick matches `soft` |
| The engine clears the frame to the scene's background (the embedder's clear color) | `ui.Scene` has no background; `gles.NewRenderer(RendererOpts{Background, HasBackground})` clears each frame to one color (default opaque white), and `Snapshot` is `GLNotImplemented` until step 3.6 | the scene is a layer tree only; the window background is the renderer's setting |
| Flutter's Linux engine asks fontconfig through Skia's `SkFontMgr_fontconfig` | `internal/platform/linux/fontconfig` binds `libfontconfig.so.1` through purego (`FcFontSort` with the user's configuration substituted, trimmed, per family, OpenType weight converted as `FcWeightFromOpenType` does, slant and language) and `internal/text/fontconfig` adapts it to `text.Matcher`; `text.Collection` takes any `Matcher`, and without one scans the font directories with go-text's `fontscan` (index in `$XDG_CACHE_HOME/glutter/fonts`), ordering the family's own fonts by weight and slant, then fontscan's substitution for the family, then the language's font | the plan's `internal/text/fontconfig/` would call C from outside `internal/platform` (CLAUDE.md, SPEC §8.1); the binding lives in the platform tree and the text tree only adapts it |
| Fonts are whatever the engine's font manager has | `Collection.Match` returns TrueType and OpenType files only (`.ttf .otf .ttc .otc`): fontconfig also lists Type 1 and bitmap fonts, which go-text cannot read. The high 16 bits of fontconfig's index (a named instance of a variable font) are ignored; the caller sets the variable axes, as Flutter does for the weight, with `Variation` | go-text has no Type 1 or PCF reader; instances are a subset of what axis values express |
| Roboto is the static family of Material Design | the bundle has one variable file, Roboto 3's `Roboto[ital,wdth,wght].ttf` (v3.016, renamed `Roboto-Variable.ttf`), whose axes give every weight, width and the italic; Material Icons is `MaterialIcons-Regular.ttf` (`material/fonts/VERSIONS` has the pins and hashes) | one file instead of eighteen; text layout (step 4.2) sets the `wght` and `ital` axes from the style |
| Text offsets (`TextPosition.offset`, `TextRange`, `TextBox` queries) are UTF-16 code units, as in Dart strings | UTF-8 byte offsets into the Go string, everywhere in `ui`, `painting` and the text-editing code | Go strings are UTF-8, `len(s)` and `s[i:j]` are in bytes, and Wayland's `text-input-v3` also uses bytes; ported Flutter tests that index non-ASCII text are adjusted |
| `ParagraphBuilder(style)` uses the engine's global font collection | `ui.NewParagraphBuilder(factory, style)`: the text engine (`ui.ParagraphFactory`, implemented by `internal/text/layout.Engine` over a `text.Collection`) is a parameter, reached by the framework through the `PlatformDispatcher` (step 5.2); `ui.LoadFontFromList` becomes a method of the engine's font collection | no global mutable state (CLAUDE.md); `ui.Paragraph` wraps a `ParagraphLayout` that the engine makes |
| `TextStyle`, `ParagraphStyle`, `StrutStyle` take nullable named parameters | `NewTextStyle(TextStyleOpts)` and the like; a nullable number or color is a pointer (`ui.Ptr(14.0)`) because `ui` cannot use `foundation.Opt`; an unset enum or `FontWeight` is its zero value; a style pushed on a `ParagraphBuilder` inherits what it leaves unset (`ResolvedTextStyle` is the result); `Color` together with `Foreground` panics with `TextStyleColorAndForeground` | same inheritance as `ParagraphBuilder::pushStyle` |
| `TextDirection` is `rtl, ltr` and `FontWeight` is an object | `TextDirectionRTL` before `TextDirectionLTR` as in Flutter's declaration order, with the unspecified zero value (left to right) first; `FontWeight` is an `int` (0 unspecified, 1 to 1000) with the nine conventional constants | enums follow §12.3 |
| Line height is computed by Skia's paragraph module | the same rules, ported from `skparagraph` as documented: half of the font's line gap goes on each side of the glyphs; a `height` then scales ascent and descent proportionally, or adds the missing space evenly with `TextLeadingDistribution.even`; the strut merges as a minimum (or sets the height with `forceStrutHeight`); a line's height is `round(ascent + descent)`; `TextHeightBehavior` can leave the first ascent and the last descent unscaled; the baseline is the first line's ascent from the top, and the ideographic baseline the bottom of the first line; the empty paragraph and the line after a final newline have the height of one line of the paragraph's style | Flutter's engine C++ is not in `reference/`; the formulas are those of `Run::calculateMetrics` and `InternalLineMetrics` |
| Lines break as Skia's paragraph module does | greedy breaking at UAX #14 opportunities with white space hanging at the end of a line (it does not count for fitting, alignment or `LongestLine`, but does for `MaxIntrinsicWidth`); a word wider than the line is cut at cluster boundaries (never inside a ligature or a combining sequence); an infinite width never breaks softly (that is how `softWrap: false` is laid out); when `maxLines` is exceeded the last kept line is cut at the last cluster where its text and the ellipsis fit and the ellipsis is shaped in the style of the text before it; justification widens the spaces of lines that do not end the paragraph or a hard break; bidi runs take levels from their direction and the paragraph's (nesting deeper than one embedding is not distinguished) | the plan's task list; Skia's source is not available to port |
| Text is shaped at the font size | runs are shaped at 128 pixels with HarfBuzz (go-text) and scaled to the size of the style, because go-text rounds the shaping size up to whole pixels; advances are exact to 1/64 pixel at 128 pixels; variable fonts get `wght` from the font weight and `ital` from the italic style, then the style's `FontVariation`s, and each variation set has its own shaper (go-text reads the coordinates from the first face it sees) | exact fractional sizes |
| Font fallback follows the engine's font manager | the families of the style in order, then `sans-serif`; each family expands to the fonts the `Matcher` (fontconfig, or fontscan) gives, and a character that none has asks the `Matcher` for a font by character (fontscan's script-based fallback; fontconfig's sorted list already covers it); a character no font has is drawn with the first font's missing-glyph box | the plan's "script, direction and font fallback through 4.1" |
| `getBoxesForRange` comes from Skia's `TextLine::getRectsForRange` | boxes are made from the position of every rune of a line (a cluster's width is shared equally by its runes, so a range inside a ligature has a box of its share); neighbouring boxes of one direction and height are merged; the vertical extent is: `tight`, the font's ascent and descent of the run (no line gap, no line height); `max`, the line's ascent and descent; `includeLineSpacingTop`/`Bottom`/`Middle`, the tight box extended by the whole, or half each way, of the space to the neighbouring line's tight box (none above the first line or below the last); `strut`, the strut's metrics (the tight box without a strut); `BoxWidthStyle.max` adds boxes at the start and end of each line up to the widest left and right edges. Line break characters have no box | the engine's C++ is not in `reference/`; this follows the documentation of the styles, and gives boxes with no gaps for the "include line spacing" styles |
| `getPositionForOffset` | the nearest grapheme cluster (UAX #29) on the line at the y coordinate (the first or last line above or below the text), snapping to its nearer edge, so a position never falls inside a cluster (it can fall inside a ligature, between its characters); a point beyond a line's text is its start or end, the end of a line that ends in a soft break being `upstream` and before the newline on a line that ends in one | caret positions for editing |
| `getWordBoundary` uses ICU | UAX #29 word segments from go-text; anything that is not a word (space, punctuation, symbol) is one grapheme cluster; an ideograph is a word of its own (no dictionary segmentation); the affinity picks the character before (`upstream`) or at the offset | no ICU |
| `getBoxesForPlaceholders` has a box for each placeholder | one box per placeholder that is on a visible line, in order; a placeholder cut away by `maxLines` has none | there is no box to place its widget at |
| Placeholders align per `PlaceholderAlignment` | `baseline` puts the box's baseline (`baselineOffset`, default the bottom edge; for the ideographic baseline, lower by the font's descent) on the text's; `aboveBaseline` and `belowBaseline` put the bottom or top edge on the baseline; `top`, `bottom` and `middle` align to the line's box and make the line taller if the box is taller than the text | Skia's `Run::updateMetrics` for placeholders |
| An `ellipsis` with no `maxLines` ellipsizes the first line that is wider than the width | the first line that does not fit is not wrapped: it is ellipsized, the text after it dropped, and `didExceedMaxLines` is true (with `maxLines` set, a wider word is cut across lines); the reference `paragraph_test.dart` expects it (step 6.6) | `TextPainter.ellipsis` documents it |
| Decorations are painted by Skia's paragraph module | `layout` gives the decoration lines as data (`Decorations()`: kind, style, color, extent, vertical position and thickness) for the renderers; the thickness is the font's underline or strikeout thickness (a 14th of the font size if it has none) times `decorationThickness`, the underline and line through sit at the font's positions, the overline at the top of the glyphs; hanging spaces are not decorated; a `double` line is `DoubleDecorationSpacing` (3 px) below the first | the renderers draw them as rectangles (solid, double, dotted and dashed: the dots are `thickness` long with a gap of the same, the dashes `4 x thickness` with a gap of `2 x`) and a stroked path (wavy: a quarter wave as long as the thickness), without snapping, in step 4.4 |
| ICU resolves the bidi levels of a paragraph (UAX #9, with the paragraph's base direction) | `layout` resolves the levels itself from the `x/text` class table: the base level is the given `TextDirection`, not the first strong character's (go-text's resolver ignores it and gives numbers after Arabic or Hebrew the wrong level); rules W1 to W7, N1, N2, I1, I2 and L1 are implemented, and a line break ends a paragraph; explicit embeddings, overrides and isolates (LRE, RLE, LRO, RLO, PDF, LRI, RLI, FSI, PDI) and paired brackets (N0) are ignored | go-text's resolver cannot take a paragraph direction; the rare explicit controls can follow when an app needs them |
| Skia draws text from a glyph atlas of its own | `ParagraphPainter` (`ui`) lowers a laid-out paragraph into ordinary display operations (`ParagraphDraw`: a backgrounds rectangle, shadow paths, glyph paths, decorations, color glyph pictures, in the order of `TextLine::paint`), which `internal/gpu/textdraw.Expand` places under the transform; `soft` draws them and `gpu.Flatten` hands them to `gles` as draws of paths, which the path coverage atlas (§9.1) already caches by outline, linear transform and fractional translation, so there is no second atlas. Glyph origins snap to a quarter pixel horizontally and a whole pixel vertically when the transform only scales and translates (not under a rotation or perspective); a glyph path is made once per face, glyph and size | one cache and one code path for glyphs and shapes; the snapping is what makes the atlas hit |
| Skia draws `TextStyle.shadows` as blurred glyph draws | each shadow is the glyph paths again in the shadow color, displaced, with a normal mask blur of `BlurSigma` | the same effect through the paths' mask filter |
| Skia and FreeType draw every color glyph format | COLR v0 and v1 (solid, linear, radial and sweep gradients, transforms, composites, glyph clips) are recorded into a `ui.Picture` per face, glyph and text color (palette 0, entry `0xFFFF` is the text color), cached; CBDT and sbix bitmaps and SVG glyphs are not drawn (a font that has only those shows no emoji; Noto Color Emoji with COLR does) | go-text decodes the COLR tables only; bitmap glyphs follow when an app needs them |
| `removeListener(listener)` compares the function | `AddListener` returns a `foundation.ListenerHandle`; `RemoveListener(handle)` undoes that registration. The zero handle does nothing, and `MergeListenables` hands out its own handles | Go functions cannot be compared, and a method value made twice is two values |
| `ChangeNotifier` is a mixin; `notifyListeners` is `@protected` | `foundation.ChangeNotifier` is a struct to embed (the zero value works); `NotifyListeners` is exported. A panic in a listener is not caught yet: step 7.3 reports it and goes on, as Flutter does | Go embedding; recovery points are wired in 7.3 (§14.2) |
| `ValueNotifier<T>` compares with `==` | `ValueNotifier[T any]` compares with Go's `==` for comparable values and by identity for slices, maps and functions (what Dart does for lists and closures) | `T` can be any type, as in Dart |
| `ValueKey<T>`, `ObjectKey` are classes | `ValueKey[T comparable]` and `ObjectKey` are plain struct values (`NewValueKey` returns a value, not a pointer), so `==` and map lookups compare them as Dart's `==` and `hashCode` do; `UniqueKey` is a pointer (equal only to itself). A key type of your own embeds `foundation.LocalKeyBase`. `NewObjectKey` of a value that cannot be compared panics with `ObjectKeyNotComparable` | keys are map keys in reconciliation |
| `defaultTargetPlatform` reads the operating system; `debugDefaultTargetPlatformOverride` is a global | `foundation.DefaultTargetPlatform()` is Linux (the only backend, §3) unless a debug build sets `SetDebugDefaultTargetPlatformOverride(Some(p))` (an atomic value, and a panic with `TargetPlatformOverrideNotDebug` in other builds). The engine supplies the host platform when a second OS is supported | `foundation` has no build tags; §3 forbids `runtime.GOOS` |
| `Diagnosticable` is a mixin with overridable `toString`, `toStringShort`, `toDiagnosticsNode` | `Diagnosticable` has one method, `DebugFillProperties`; `DebugDescribeChildren` makes it a `DiagnosticableTree`. `ToStringShort`, `ToDiagnosticsNode`, `DiagnosticableString`, `ToStringShallow` and `ToStringDeep` are functions that use the optional `ShortDescriber` and `DiagnosticsNodeProvider` interfaces. There is no `super.debugFillProperties`: call the embedded type's method | Go has no mixins or virtual `super` |
| `StringProperty`, `DoubleProperty`, `IntProperty`, `PercentProperty`, `FlagProperty`, `IterableProperty`, `EnumProperty`, `ObjectFlagProperty`, `FlagsSummary`, `MessageProperty` are subclasses of `DiagnosticsProperty` | one `*DiagnosticsProperty` type; the `NewStringProperty` family configure it, and `TypeName()` says which class it stands for. A nullable value is `Opt[T]`; a nullable string option (`ifNull`, `ifEmpty`, `tooltip`, `description`, `unit`, `ifTrue`, ...) is a `string` where `""` means null. `defaultValue: null` is `DefaultNull`, and a nil `DefaultValue` is "no default". A nil slice given to `NewIterableProperty` is null; pass an empty one for an empty list | one type avoids a virtual-call framework for ten classes |
| `debugDoublePrecision` | not ported: `DebugFormatDouble` always has one decimal | a global that tests set; add it with an options value if a test needs it |
| `DiagnosticsNode.toJsonMap`, `DiagnosticsSerializationDelegate`, `toTimelineArguments`, `DiagnosticsStackTrace` | none | DevTools and the timeline are out of scope (§2); `DiagnosticsStackTrace` comes with the error types in step 7.3 |
| the `opts.First` helper of §12.1 | `internal/opts.First`; packages with a parameter called `opts` wrap it (`oneOpts` in `ui`, `internal/text` and `internal/text/layout`, `firstOpts` in `foundation`) because the parameter hides the package | the name `opts` is the convention for the trailing parameter |
| `window.resizable` and `decorated` default to true | `platform.WindowConfig.Resizable` and `Decorated` are `foundation.Opt[bool]`; read them with `IsResizable()` and `IsDecorated()`. `DefaultWindowConfig()` is the zero value | the zero value is the default (§12.3) |
| `PlatformDispatcher`'s `onBeginFrame`, `onDrawFrame`, `onPointerDataPacket`, ... are properties set by assignment | exported function fields (`d.OnBeginFrame = func(frameTime time.Duration) {...}`); the methods that report what happened (`BeginFrame`, `DrawFrame`, `DispatchPointerDataPacket`, `DispatchKeyData`, `UpdateViewConfiguration`, `UpdatePlatformConfiguration`, `SetAppLifecycleState`, `AddView`, `RemoveView`, `SetDisplays`) are exported too, for the engine in another package; they are documented "Called by the engine" and an app does not call them | the engine lives in `internal/engine`, and Dart's private hooks have no Go equivalent |
| `FlutterView.render(scene)` returns nothing; the engine rasterizes after `onDrawFrame` | `View.Render(scene) error` draws at once, inside the call, on the UI thread, and returns the renderer's error (`EGLContextLost` and the like), which also ends `Run`. A frame in which the framework renders nothing draws the previous scene again (the clear color before the first), because the platform needs exactly one presented frame for each `OnFrame` it asked for (`platform.Window.RequestFrame`) | Wayland's frame callbacks fire on commit; a frame that is not presented stops all later frames |
| the lifecycle arrives on the `flutter/lifecycle` channel; `requestAppExit` is the `System.exitApplication` message | `PlatformDispatcher.OnAppLifecycleStateChanged`, `AppLifecycleState()` and `RequestAppExit()` on the dispatcher | no platform channels (§11.2) |
| `ViewConfiguration`, `PlatformConfiguration` are private to `dart:ui` | exported `ui.ViewConfiguration` and `ui.PlatformConfiguration` structs, which the engine passes to the dispatcher; `Padding` is the part of `ViewPadding` that `ViewInsets` do not cover | the engine and `ui` are different packages |
| the platform tells the locales as `Locale` objects | `platform.Settings.Locales` is `[]ui.Locale`, parsed by `ui.ParseLocale` from BCP 47 tags and POSIX names such as `en_US.UTF-8` | the Linux backends read tags |
| `Brightness` is in `dart:ui` | `ui.Brightness` with the unspecified zero value, which the dispatcher reads as light | enum convention (§12.3) |
| the embedder starts the framework by calling `main` | `engine.Config.Start(d *ui.PlatformDispatcher) error` runs once, on the UI thread, after the window exists and before the loop; the dev examples' `Config.Scene` is built on the dispatcher (an `OnDrawFrame` that renders its scene) and is used only when `Start` is nil | `RunApp` (step 7.4) attaches the binding in `Start` |
| `SchedulerBinding` is a mixin with `SchedulerBinding.instance`, and `timeDilation` is a global | `scheduler.SchedulerBinding` is a struct made by `NewSchedulerBinding(dispatcher)` (the framework's bindings embed it); `TimeDilation()`/`SetTimeDilation(v)` are on the binding | no singletons (CLAUDE.md); `Ticker` takes the binding in `NewTicker` |
| `scheduleTask<T>` returns a `Future<T>`; tasks wait for `Timer.run` | `ScheduleTask[T](b, task, priority)` returns a `<-chan T` and asks the event loop through `PlatformDispatcher.Post`; tasks of one priority run in the order scheduled (Dart's heap does not promise it) | futures are channels (§12.6) |
| `lockEvents(callback)` unlocks when the callback's `Future` completes | `LockEvents()` returns the function that unlocks; `AddUnlockedCallback` stands for overriding `unlocked()` | no futures |
| `scheduleWarmUpFrame`, `onReportTimings`/`FrameTiming`, performance mode, service extensions, `debugPrint*Banner` flags | not ported yet | the engine has no warm-up frame or timings (the performance overlay is step 17.3); the flags go in `DebugOptions` (step 7.4) |
| `TickerFuture` is a `Future<void>` | a struct with `Done()` (closed on completion only), `Then(fn)`, `OrCancel()` (a channel that gets nil, or a `TickerCanceled` error) and `WhenCompleteOrCancel(fn)`; callbacks run at once on the UI thread, not in a later microtask | futures are channels and callbacks (§12.6) |
| callbacks that throw are reported and the frame goes on | a panic in a frame callback or task propagates for now | recovery points are wired in step 7.3 (§14.2) |
| `Simulation` is an abstract class with a mutable `tolerance` | `physics.Simulation` is an interface (`X`, `Dx`, `IsDone`, `Tolerance`, `SetTolerance`); a simulation embeds `SimulationBase`, whose tolerance is the default until set. `Tolerance{}` has all fields 0, so `NewTolerance(ToleranceOpts{...})` is the constructor that gives Flutter's 1e-3 defaults | Go embedding; a zero value cannot be the default (§12.3) |
| `BouncingScrollSimulation` and `ClampingScrollSimulation` are in `widgets/scroll_simulation.dart` | they are in package `physics`, as the §11.1 row lists them (step 5.4); step 9.2 does not port that file again | the scroll simulations depend on nothing but the physics |
| `BoundedFrictionSimulation extends FrictionSimulation` and `timeAtX`, `finalX` call the overridden `x` | `BoundedFrictionSimulation` embeds `*FrictionSimulation` and has its own `X`, `FinalX`, `TimeAtX` | Go embedding has no virtual calls |
| `nearEqual(double? a, double? b, epsilon)` | `NearEqual(a, b, epsilon float64)` and `NearEqualOpt(a, b Opt[float64], epsilon)` | no nullable numbers |
| `FrictionSimulation.through`, `SpringDescription.withDampingRatio` and `.withDurationAndBounce` | `NewFrictionSimulationThrough`, `NewSpringDescriptionWithDampingRatio` and `NewSpringDescriptionWithDurationAndBounce` | §12.4 |
| `PointerEvent` is an abstract class with a subclass per event; each constructor has its own named parameters, and `_Transformed*` subclasses defer to the original | `gestures.PointerEvent` is an interface and each event is a struct (`*PointerDownEvent`, ...) made by `NewPointerDownEvent(PointerEventOpts)`. The options struct is shared: a constructor reads the fields Flutter's has and ignores the others (`NewPointerAddedEvent` ignores `Buttons`). `Transformed(m)` returns a copy of the same type with `Original()` and `Transform()` set and `LocalPosition()`/`LocalDelta()` computed at once, not lazily. `CopyWith(PointerEventCopyOpts)` copies only the fields that event's `copyWith` lists, as Flutter does | one options struct instead of 15, and no virtual dispatch to subclass; `is` becomes a type switch |
| `kPrimaryButton`, `kSecondaryButton`, `kTouchContact`, ... | `PrimaryButton`, `SecondaryButton`, `TouchContact`, ... | §12.4: no `k` prefix (`kDebugMode` is `DebugMode`) |
| `PointerEvent.toString`, `toStringFull` | `String()` and `gestures.ToStringFull(e)` | Go has no `toString(minLevel:)` |
| `PointerRouter.addRoute(pointer, route, [transform])` and `removeRoute(pointer, route)` compare functions | `AddRoute(pointer, route, transform)` returns a `PointerRouteHandle` that `RemoveRoute(pointer, handle)` takes; likewise `AddGlobalRoute` and `RemoveGlobalRoute`. Removing an unknown route panics with `PointerRouteNotFound` in debug builds | Go functions can't be compared (as for listeners, step 5.1) |
| `HitTestEntry<T>` is a class that `BoxHitTestEntry` extends; `HitTestable.hitTest` is deprecated; `NativeHitTestTarget` is a mixin | `HitTestEntry` is an interface and `HitTestEntryBase` the struct to create (`NewHitTestEntry`) or embed. `HitTestResult.pushTransform`, `pushOffset` and `popTransform` are exported, for `rendering`'s `BoxHitTestResult`, which wraps a result with `NewHitTestResultWrap`. `hitTest` is not ported; `NativeHitTestTarget` is an interface with one marker method and nothing uses it | no protected members or generic subclasses; the deprecated method and `PlatformDispatcher.onHitTest` (platform views, which Glutter doesn't have) have no use |
| `GestureBinding` is a mixin on `BindingBase` with `GestureBinding.instance`, and `RendererBinding` overrides `hitTestInView` and `dispatchEvent`; tests override `handleEvent` and `debugSamplingClock` | `gestures.GestureBinding` is a struct made by `NewGestureBinding(dispatcher, scheduler)` that sets the dispatcher's `OnPointerDataPacket`. `OnHitTestInView` (runs before the binding adds itself) and `OnDispatchEvent` (runs before the targets get the event) stand for the overrides; `GestureBindingOpts.Clock` is the clock that the resampler, the recognizers' timers (`NewTimer`) and the velocity trackers read. `GestureScheduler` is the part of the scheduler it needs (`*scheduler.SchedulerBinding` implements it) | no singletons or mixins (CLAUDE.md) |
| `SamplingClock` has `now()` and `stopwatch()`; the resampler uses `Timer.periodic` | `SamplingClock` has `Now()` and `AfterFunc`; the binding re-arms the timer and runs each tick on the UI thread through `Post` | Go has no periodic timer that runs on the UI thread |
| `GestureArenaManager` calls `scheduleMicrotask` | `NewGestureArenaManager(scheduleMicrotask)`; the binding passes the dispatcher's `Post`. A member that joins a closed arena panics with `GestureArenaClosed` | no microtask queue in Go |
| `debugPrintHitTestResults`, `debugPrintGestureArenaDiagnostics`, `debugPrintMouseHoverEvents`, `debugPrintResamplingMargin`, `debugPrintRecognizerCallbacksTrace` are globals | the fields of `gestures.DebugOptions`, held by the binding (`GestureBindingOpts.Debug`) and the arena (`GestureArenaManagerOpts.Debug`), with a `Print` function. `debugAssertAllGesturesVarsUnset` is not ported | no global mutable state |
| `PointerRouter`, `PointerSignalResolver` and `GestureBinding` catch exceptions of routes, callbacks and targets and report them with `FlutterError.reportError` | a panic in a route, callback or target propagates for now | recovery points are wired in step 7.3 (§14.2) |
| `PointerData()` has `change: PointerChange.cancel` and `kind: touch` | the zero `ui.PointerData` converts as cancel and touch: `PointerChangeUnspecified` and `PointerDeviceKindUnspecified` | Go's zero value is the default |
| `PointerEventConverter.expand` returns an `Iterable` and `DevicePixelRatioGetter` returns `double?` | `PointerEventConverterExpand` returns a slice and the getter returns `Opt[float64]` | §12.10 conventions for nullable values || Recognizers use `GestureBinding.instance` (the arena, the pointer router, `Timer`) | every recognizer constructor takes the `*GestureBinding` first: `NewTapGestureRecognizer(binding, opts...)`. Timers are `binding.NewTimer(d, fn)`, which runs `fn` on the UI thread and reads the binding's clock (`GestureBindingOpts.Clock`) | no singletons (CLAUDE.md); tests use a fake clock |
| Recognizers are classes that override `addAllowedPointer`, `handleEvent`, `acceptGesture`, `didStopTrackingLastPointer`, `handlePrimaryPointer`, … and mixins (`_TapStatusTrackerMixin`) | a recognizer embeds `GestureRecognizerBase`, `OneSequenceGestureRecognizer` or `PrimaryPointerGestureRecognizer` and passes itself to `Init`, `InitOneSequence` or `InitPrimaryPointer`; the base calls the overridable methods through that `self` (`GestureRecognizerHooks`, `OneSequenceHooks`, `PrimaryPointerHooks`). The tap status tracker is part of `BaseTapAndDragGestureRecognizer`. `BaseTapGestureRecognizer`, `DragGestureRecognizer` (axis strategies for vertical, horizontal and pan) and `MultiDragPointerState` work the same way | Go has no virtual methods on embedded structs |
| Callbacks (`onTap`, `onDragUpdate`, …) are properties, and recognizers are configured with a cascade | the callbacks are exported func fields (`OnTap`) that the owner sets after the constructor; the constructor takes the other parameters in an options struct. `InvokeCallback` runs a callback (it prints under `DebugOptions.PrintRecognizerCallbacksTrace`); recovery from a panic in it is `TODO(step 7.3)` | the options-struct convention (§12) |
| `VelocityTracker`, `IOSScrollViewFlingVelocityTracker` and `MacOSScrollViewFlingVelocityTracker` are three classes; `VelocityTracker` uses a `Stopwatch` | one `VelocityTracker` struct with a mode, made by `NewVelocityTrackerWithKind`, `NewIOSScrollViewFlingVelocityTracker` or `NewMacOSScrollViewFlingVelocityTracker`, each with the `SamplingClock` that replaces the stopwatch. `GetVelocityEstimate` returns an `Opt` | the subclasses only change two methods; no `Stopwatch` in Go |
| `GestureArenaEntry` is a class | `GestureArenaEntry` is an interface with `Resolve`; the arena and the teams return their own entries | a team's entry resolves a combined member |
| `preAcceptSlopTolerance` and `postAcceptSlopTolerance` are `double?` with defaults | `SlopTolerance`: the zero value is the default, `SlopToleranceNone` is `null` and `NewSlopTolerance(v)` a limit | `Opt` can't tell the default from `null` |
| `DragStartBehavior.start` is the default | `DragStartBehaviorUnspecified` (the zero value) means `start` | the enum rule (§12) |
| `eagerVictoryOnDrag` defaults to `true` | the option is `NoEagerVictoryOnDrag`; the field is `EagerVictoryOnDrag` | the zero value of a bool is false |
| `PlatformDispatcher` view `gestureSettings` is `ui.GestureSettings(physicalTouchSlop, physicalDoubleTapSlop)` with `null` fields | `ui.GestureSettings` has `float64` fields, where 0 means unset; `ViewConfiguration.GestureSettings` and `View.GestureSettings()` hold it, and `NewDeviceGestureSettingsFromView(view)` divides by the pixel ratio | the zero value (§12) |
| `ForcePressGestureRecognizer` | not ported | desktop pointers report no pressure (§11.1 lists no force press) |
| `TapAndDragGestureRecognizer` (deprecated) | not ported; use `TapAndPanGestureRecognizer` | deprecated in Flutter |
| `MouseTracker`, `MouseCursor`, `MouseCursorManager`, `MouseCursorSession` and `MouseTrackerAnnotation` | `rendering.MouseTracker` (`NewMouseTracker(hitTestInView, fallbackCursor)`) and, in `services`, `MouseCursor`, `MouseCursorSession`, `MouseCursorManager` and the annotation, which step 10.4 completes with `SystemMouseCursors` and the platform wiring. `MouseCursorDefer` and `MouseCursorUncontrolled` are variables. `MouseTrackerAnnotation` is an interface (the render object is the annotation) and `MouseTrackerAnnotationData` the plain struct; `MouseCursorSession.Activate` is synchronous; `DebugDeviceActiveCursor` returns a cursor and a bool | `MouseTracker` needs them before 10.4; no `Future` in the UI thread's model |
| `RendererBinding` calls `mouseTracker.updateWithEvent` and a post-frame callback calls `updateAllDevices` | the tracker has `UpdateWithEvent(event, hitTestResult)` (nil hit test result means hit test again) and `UpdateAllDevices()`; the renderer binding that calls them is step 6.3 and 6.7 | no renderer binding yet |
| `EdgeInsetsGeometry`, `AlignmentGeometry` and `BorderRadiusGeometry` are abstract classes with subclasses and a private mixed class | interfaces (`EdgeInsetsGeometry`, ...) with an unexported parts accessor, value structs (`EdgeInsets`, `EdgeInsetsDirectional`, `Alignment`, ...) and an unexported mixed type. The operators with covariant returns (`-x`, `*`, `/`, `~/`, `%`, `flipped`) are methods on the concrete types (`Neg`, `Mul`, `Div`, `TruncDiv`, `Mod`) and package functions for the interface (`EdgeInsetsGeometryMul`, `AlignmentGeometryLerp`, ...); `+` and `-` are `Plus`/`Minus` on the concrete types and `Add`/`Subtract` on the interface. `EdgeInsets.lerp` and the other typed lerps take and return pointers, nil being null | Go has no abstract classes, operators or covariant return types |
| `EdgeInsetsGeometry.all`, `.only`, `.directional`, `AlignmentGeometry.topLeft`, `BorderRadiusGeometry.all`, ... (factories and constants on the abstract class) | not ported; use `NewEdgeInsetsAll`, `EdgeInsets{...}`, `AlignmentTopLeft`, `NewBorderRadiusAll`, ... | an interface has no static members |
| `copyWith` with named parameters on the geometry, border, shadow and decoration classes | `CopyWith(XCopy{...})` with a hand-written `XCopy` struct of `foundation.Opt` fields (a nil interface or an unspecified enum keeps the old value); the generator of step 11 does not make these | the options struct convention (§12.1) applied to the method; written by hand until the generator exists |
| `ShapeBorder` is abstract with protected `add`, `lerpFrom`, `lerpTo` and named `textDirection` | the interface `ShapeBorder` exports `Add(other, reversed)`, `LerpFrom` and `LerpTo`, and takes `textDirection ui.TextDirection` as the last positional argument of `GetOuterPath`, `GetInnerPath`, `PaintInterior` and `Paint` (`ui.TextDirectionUnspecified` is null). Embed `ShapeBorderDefaults` for the defaults `Add`, `PaintInterior` and `PreferPaintInterior`; `ShapeBorderDefaultLerpFrom/To(self, x, t)` are `super.lerpFrom/To`. `ShapeBorder.+` is `ShapeBorderPlus` and `ShapeBorder.lerp` is `ShapeBorderLerp` | other packages must implement `ShapeBorder`, and Go has no protected |
| `BoxBorder.top` and `.bottom`; `BoxBorder.paint(canvas, rect, {shape, borderRadius, textDirection})` | the interface has `TopSide()`, `BottomSide()` and `PaintBox(canvas, rect, BoxPaintOpts)`, because `Border` and `BorderDirectional` have fields named `Top` and `Bottom`; `Paint(canvas, rect, textDirection)` is `ShapeBorder`'s. A zero `BorderSide{}` in a `Border` literal is `BorderSideNone`, as the missing sides of a Dart `Border` are | a Go struct cannot have a field and a method of the same name; the zero value should mean what an omitted side means |
| `OutlinedBorder` is an abstract class with `side` and `copyWith(side)` | the interface has `Side()` and `CopyWithSide(side)`; the concrete borders (`RoundedRectangleBorder`, `CircleBorder`, ...) are comparable value structs with unexported sides, built by `NewX(XOpts)` and copied by `CopyWith(XCopy)`. `OvalBorder` embeds `CircleBorder` and is a `CircleBorder` to the lerps that test for one; the `_ShapeToCircleBorder`, `_StadiumTo...` classes are unexported structs | embedding stands for `OvalBorder extends CircleBorder`; the sides are read through the method |
| `BorderSide` constructor with defaults and `BorderSide.none` | `NewBorderSide(BorderSideOpts)` (color black, width 1.0, solid, inside) and the variable `BorderSideNone`; the zero value `BorderSide{}` is treated as none by the borders (`normalize`) | Go has no constructor defaults |
| `StarBorder.polygon`, `LinearBorder.start/end/top/bottom` and the other named constructors | `NewStarBorderPolygon`, `NewLinearBorderStart`, `NewLinearBorderEnd`, `NewLinearBorderTop`, `NewLinearBorderBottom` | naming rule |
| `Gradient` is an abstract class with `colors`, `stops`, `transform`, protected `lerpFrom`/`lerpTo` and `scale` | the interface `Gradient` has `GetColors`, `GetStops`, `GetTransform`, `CreateShader`, `Scale`, `WithOpacity`, `FromColor`, exported `LerpFrom` and `LerpTo` and `Equal`; `GradientLerp(a, b, t)` is `Gradient.lerp`; `LinearGradient.lerp` and the others are `LinearGradientLerp`, ... on pointers. The radial radius defaults to 0.5 and the sweep end angle to 2 pi through `foundation.Opt` | a Go struct cannot have a field and a method of the same name; `Opt` for defaults other than zero |
| `BoxDecoration.image` and `ShapeDecoration.image` are `DecorationImage` | the interface `DecorationImageSource` (a painter factory, `Blended` and `Equal`), which `*DecorationImage` of step 6.2 implements; `DecorationImagePainter` is an interface | `DecorationImage` needs `ImageProvider`, step 6.2 |
| `Decoration` is an abstract class | the interface `Decoration`, with `DecorationDefaults` to embed for the defaults (zero padding, no lerps, `HitTest` true, `GetClipPath` panics with `DecorationNoClipPath`); `Decoration.lerp` is `DecorationLerp`, and `LerpFrom`/`LerpTo` are exported. `BoxPainter` is an interface (`Paint`, `Dispose`); `onChanged` is the argument of `CreateBoxPainter`. `hitTest(size, position, {textDirection})` is `HitTest(size, position, textDirection)` | Go has no abstract classes or protected methods |
| `ImageConfiguration` (with `bundle`) | `ImageConfiguration` in `image_provider.go`, with `Bundle services.AssetBundle`, `Binding *PaintingBinding` (see the `PaintingBinding.instance` row below), `Opt` fields and `TextDirection` (unspecified for null) | no singletons; unset values are `Opt` |
| `FlutterLogoDecoration` paints its label with a `TextPainter` | it builds a `ui.Paragraph` with the engine given in `FlutterLogoDecorationOpts.TextEngine`; with no engine the label is not painted | there is no global font collection (§8.5); `TextPainter` is step 6.2 |
| `ColorSwatch<T> extends Color` | `ColorSwatch[T]` with the field `Primary` (a `ui.Color`), `At(key)`, `Keys()` and `ColorSwatchLerp` | `ui.Color` is a value struct; it cannot be extended |
| `MatrixUtils` (static methods) | the methods of `ui.Matrix4` made by step 2 (`AsTranslation`, `AsScale`, `Equals`, `TransformPoint`, `TransformRect`, `InverseTransformRect`, `NewMatrix4CylindricalProjection`, `NewMatrix4ForceToPoint`, `MatrixMultiplyInPlace`); `painting` has `DebugDescribeTransform` and `NewTransformProperty` | the geometry lives with the matrix in `ui` |
| `ClipContext` mixin | the interface `ClipContext` (`Canvas()`) and the functions `ClipPathAndPaint`, `ClipRRectAndPaint`, `ClipRSuperellipseAndPaint` and `ClipRectAndPaint` | a mixin becomes functions over an interface |
| `debugDisableShadows` and `BoxShadow.toPaint` reading it | not ported; shadows are always drawn | a global flag (no global state, CLAUDE.md) |
| asserts and `FlutterError.fromParts` in `painting` (`BorderSide.merge`, `BoxBorder`, `BoxDecoration`, the star, circle and linear borders, colors, gradients, text direction) | panics in debug builds, with the tags `BorderSideCannotMerge`, `BorderSideWidthNegative`, `BoxBorderCircleRadius`, `BoxBorderHairlineRadius`, `BoxBorderNonUniformRadius`, `BoxBorderNonUniformShape`, `BoxBorderStrokeAlignNonUniform`, `BoxBorderLerpInvalid`, `TextDirectionRequired`, `BoxDecoration...`, `ShapeDecoration...`, `StarBorderInvalid`, `BorderEccentricityOutOfRange`, `LinearBorderEdgeSizeOutOfRange`, `ColorComponentOutOfRange`, `GradientColorsTooFew`, `GradientInterpolationInvalid`, `FlutterLogo...`, `DecorationHitTestOutside` and `DecorationNoClipPath` | §14.1 |
| `DiagnosticsProperty.valueToString` overridden by a subclass (`TransformProperty`) | `foundation.DiagnosticsPropertyOpts.ValueToString` | Go has no subclassing of the property |
| `rounded_superellipse_border.dart` | in `rounded_rectangle_border.dart` of the pinned Flutter; ported there | the pinned version |
| `box_fit.dart` (`applyBoxFit`, `BoxFit`) is not in step 6.1's list | ported with `FlutterLogoDecoration`, which uses it, in `painting/box_fit.go` | `FlutterLogoDecoration` needs it; step 6.2's `DecorationImage` does too |
| `TextStyle` has nullable fields, `copyWith`, `apply`, `merge`, `lerp`, and the `textScaleFactor` parameters | `NewTextStyle(TextStyleOpts)` with `foundation.Opt` for nullable numbers, colors, `Locale` and `TextDecoration`, `ui` enums and `ui.FontWeight` at their zero value for null, `*ui.Paint` for `foreground`/`background`, nil slices for null; `CopyWith(TextStyleCopy)` where `TextStyleCopy` is an alias of `TextStyleOpts`; `Apply(TextStyleApply)` with `Opt` factors (default 1) and plain deltas (default 0); `TextStyleLerp`; `GetTextStyle`/`GetParagraphStyle` take the `TextScaler` only (the deprecated `textScaleFactor` is dropped); `ui.TextStyleOpts.Decoration` became `*ui.TextDecoration` so an explicit `none` overrides an underline of the parent | Go has no nullable value types; `TextDecoration.none` is the zero value; `textScaleFactor` is deprecated in Flutter |
| `StrutStyle`, `StrutStyle.fromTextStyle`, `StrutStyle.disabled` | `NewStrutStyle(StrutStyleOpts)`, `NewStrutStyleFromTextStyle(textStyle, opts)` (which prefixes the family twice with a package, as Flutter does) and `StrutStyleDisabled()` | named constructors; a constant that holds a pointer is a function |
| `TextScaler` is an abstract class with `linear`, `noScaling`, `clamp` | the interface `TextScaler` (`Scale`, `TextScaleFactor`, `Clamp`, `String`); `NewTextScalerLinear`, the variable `TextScalerNoScaling`, `TextScalerClamp(s, opts)` for an implementation's `Clamp`, `TextScalerEqual` for `==` | interfaces; equality of values |
| Text offsets in spans, `TextPainter` and `WordBoundary` are UTF-16 code units | UTF-8 byte offsets (§12.10 above): `InlineSpanCodeUnitAt` returns a byte, `TextSpan` text length is `len(text)`, a placeholder takes the 3 bytes of U+FFFC, `TextPainter.GetOffsetAfter`/`GetOffsetBefore` step by the encoded length of a character, `WordBoundary` decodes runes (`\p{Space_Separator}` and `\p{Punctuation}` are `unicode.Zs` and `unicode.P`); `TextPainterIsHighSurrogate`/`IsLowSurrogate` stay for UTF-16 values | the same decision as `ui` |
| `InlineSpan` is an abstract class with concrete methods (`getSpanForPosition`, `toPlainText`, `getSemanticsInformation`, `codeUnitAt`) | the interface `InlineSpan` plus the embeddable `InlineSpanBase` (the style) and the functions `InlineSpanGetSpanForPosition`, `InlineSpanToPlainText`, `InlineSpanGetSemanticsInformation`, `InlineSpanCodeUnitAt`; `Build(builder, SpanBuildOpts{TextScaler, Dimensions})`; `PlaceholderSpan` is an interface with `PlaceholderSpanBase` to embed (`DebugAssertIsValid` panics with `PlaceholderSpanNotImplemented`, so `WidgetSpan` must override it); `InlineSpanSemanticsInformation.SemanticsLabelSet` says the label is not null; `ui.StringAttribute`, `SpellOutStringAttribute` and `LocaleStringAttribute` are new in `ui` | no inheritance; the mixed-in default methods are functions |
| `TextSpan` implements `HitTestTarget` and `MouseTrackerAnnotation`; its mouse cursor defaults to `SystemMouseCursors.click` with a recognizer | the same interfaces (`OnEnter()` and `OnExit()` are methods, the callbacks are given in `TextSpanOpts`); the cursor is `services.MouseCursorDefer` unless given (TODO(step 10.4): the click cursor); two spans with an `OnEnter` or `OnExit` callback are never `Equal` unless they are the same span; invalid UTF-8 in the text is replaced by U+FFFD instead of reporting an `ArgumentError` | `SystemMouseCursors` arrives in step 10.4; function values can not be compared |
| `TextPainter` takes no text engine, `textScaleFactor`, `TextSelection`; `dispose` may be left out | `TextPainterOpts.TextEngine` (the `ui.ParagraphFactory` the framework gets from `PlatformDispatcher.TextEngine()`, new in `ui`); `GetBoxesForSelection` takes a `ui.TextRange`; `Layout(minWidth, maxWidth)` takes both; the errors of `StateError` and of the debug asserts are panics with the tags `TextPainterTextNotSet`, `TextPainterDirectionNotSet`, `TextPainterPaintBeforeLayout`, `TextPainterLayoutNotAvailable`, `TextPainterNeedsRelayout`, `TextPainterDisposed`, `TextPainterMaxLinesInvalid`, `TextPainterConstraintsNaN`, `TextPainterPlaceholderCountMismatch`, `TextPainterSelectionInvalid`; `ui.Paragraph` has no `dispose`, so the painter only drops its references; `WordBoundary` builds on `services.TextBoundary` (`TextBoundaryDefaults` to embed, since the three methods default to each other) | no global font collection; `TextSelection` is step 11.2; no explicit paragraph release |
| `TextOverflow`, `TextWidthBasis`, `ImageRepeat` and `ResizeImagePolicy` | enums with the zero value `…Unspecified` (null, or the default) | §12.1 |
| `PaintingBinding.instance`, `rootBundle`, `imageCache` | `NewPaintingBinding(ctx, schedulerBinding, opts)` with `ImageCache()`, `AssetBundle()` (the root bundle, from `RunAppOpts.Assets` in step 7.4), `HTTPClient()`, `Post`, `AfterFunc`, `Context()`; the functions that Flutter gives to `ImageProvider` (`resolve`, `obtainCacheStatus`, `evict`, `resolveStreamForKey`) take the binding: `ImageProviderResolve(binding, provider, configuration)` and so on, and `ObtainKey(binding, configuration)`; `DecorationImage` painters take it from `ImageConfiguration.Binding`, and panic with `ImageConfigurationNoBinding` without; `ImageCache` takes the `SchedulerBinding` for `addPostFrameCallback` | no singletons (CLAUDE.md) |
| `ImageProvider<T>.obtainKey` returns a `Future<T>`; `loadBuffer` is deprecated; `ImmutableBuffer`, `TargetImageSizeCallback`, `Future<ui.Codec>` | `ObtainKey` is synchronous (the asset variants come from `fs.FS`); `LoadImage(binding, key, decode)` returns the completer; keys are `any` and must be comparable; `ImageDecoderCallback(ctx, data []byte, getTargetSize)` with `TargetImageSize{Width, Height Opt[int]}`; the file, asset and network reads run in a goroutine and hand the result to the UI thread through `PaintingBinding.Post`; `loadBuffer` and the web strategies (`WebHtmlElementStrategy`) are not ported | futures are goroutines and channels (§12.6); `[]byte` stands for the buffer |
| `ImageStreamListener` compares `onImage`, `onChunk` and `onError` | `*ImageStreamListener`, compared by pointer: add and remove with the same pointer; `ImageListener`, `ImageChunkListener` and `ImageErrorListener` get an `error` (its stack is in the error) | function values can not be compared |
| `ImageStreamCompleter` is an abstract class; `OneFrameImageStreamCompleter` and `MultiFrameImageStreamCompleter` extend it; they take `Future`s and a `Stream` | one concrete `*ImageStreamCompleter` (`NewImageStreamCompleter`, `SetImage`, `ReportError`, `ReportImageChunkEvent`, `SetOnDisposed`), with `NewOneFrameImageStreamCompleter(binding, <-chan ImageResult, …)` and `NewMultiFrameImageStreamCompleter(binding, opts)` taking channels; `AddOnLastListenerRemovedCallback` returns the function that removes the callback; `reportError`'s `FlutterErrorDetails` is `ImageErrorDetails`; an error nobody handled goes to `PaintingBinding.ReportError` (TODO(step 7.3): the framework's error reporter); a panic in a listener propagates (step 7.3 recovers it); the frame after `getNextFrame` runs through `Post`, as Dart's `await` would; `ImageStreamCompleter.addListener` after dispose panics with `ImageStreamDisposed` | no subclasses; futures are channels |
| `ImageProvider` implementations: `NetworkImage` (dart:io `HttpClient`), `FileImage(File)`, `MemoryImage(Uint8List)`, `AssetImage` with `AssetManifest` | `NewNetworkImage` (`net/http` with the binding's context and client; errors `NetworkImageLoadFailed`, `NetworkImageEmpty`, `NetworkImageRequestFailed`), `NewFileImage(path)` (`ImageFileEmpty`, `ImageFileReadFailed`), `NewMemoryImage(bytes)` (equal when the slices share the first element), `NewAssetImage` and `NewExactAssetImage` over `services.AssetBundle`, whose `Variants` finds `<dir>/<scale>x/<file>` siblings; the main asset is variant 1.0; `services.FSAssetBundle` (errors `AssetNotFound`, `AssetReadFailed`, `AssetReadCanceled`) and `services.AssetBundle` are a first version that step 10.4 completes | the asset manifest is the file system |
| `ResizeImage` asserts when composed with another provider that applies a size | the load fails with `ResizeImageComposed` (the check runs in the load goroutine, where a panic would end the process); `NewResizeImage` without a size panics with `ResizeImageNoSize` in debug builds | a panic in a goroutine is fatal |
| `paintImage` reports oversized images (`debugInvertOversizedImages`, `ImageSizeInfo`, `Flutter.ImageSizesForFrame`) | not ported; `PaintImage(canvas, rect, image, PaintImageOpts)` has the drawing only, with the asserts as `PaintImageDisposed`, `PaintImageCenterSliceFit` and `PaintImageCenterSliceCropped` | developer tooling for the Dart VM |
| `DecorationImagePainter.paint(…, {blend, blendMode})` | `Paint(canvas, rect, clipPath, configuration, opts ...DecorationImagePaintOpts)`; `*DecorationImage` implements `DecorationImageSource`; the blend of two images is an unexported type made by `Blended`; `DecorationImage.matchTextDirection` without a direction panics with `DecorationImageNoTextDirection` | options struct |
| `FractionalOffset extends Alignment` | `FractionalOffset` embeds `Alignment` (so it is an `AlignmentGeometry`), its operators are methods that return `FractionalOffset` (`Plus`, `Minus` take a `FractionalOffset`), `FractionalOffsetLerp` takes pointers | covariant operators (§12.10 above) |
| `AutomaticNotchedShape` (uses `Path.combine`) | not ported: `ui.Path` has no boolean operations yet; `CircularNotchedRectangle` and the `NotchedShape` interface are | path operations are not in the engine API of step 2 |
| `FlutterLogoDecoration` paints its label with a `TextPainter` (row above) | unchanged: it keeps the `ui.Paragraph` built from `FlutterLogoDecorationOpts.TextEngine` (6.1); the output is the same as through a `TextPainter` | `TextPainter` adds nothing to a one-line label |
| `shader_warm_up.dart`, `image_decoder.dart` (`ImageDecoder`, web), `PaintingBinding.shaderWarmUp`, `PaintingBinding.systemFonts` through `handleSystemMessage` | no shader warm-up (the GPU backends compile their programs at start); `PaintingBinding.SystemFonts()` is a `Listenable` that `HandleSystemFontsChange()` notifies; `geometry.dart` is `PositionDependentBox(opts)` | no Skia shader cache; the platform has no system message channel |
| `RenderObject` is an abstract class; render objects subclass it and override `performLayout`, `paint` and others; mixins (`RenderObjectWithChildMixin`, `ContainerRenderObjectMixin`, `RenderBoxContainerDefaultsMixin`) | `RenderObject` is an interface of the hooks; `RenderObjectBase` and `RenderBox` are embedded structs that keep `self` (`InitRenderObject(&r.RenderObjectBase, r)`, `InitRenderBox(&r.RenderBox, r)`) and call the hooks through it; the mixins are generic embedded structs (`RenderObjectWithChild[C]`, `ContainerRenderObject[C]`, `ContainerBoxParentData[C]`, `RenderBoxContainerDefaults`) that register a child model with the base so `Attach`, `Detach`, `VisitChildren` and `RedepthChildren` reach the children; `ChildLayoutHelper` is a set of functions | Go has no inheritance; embedding with a back pointer is the §12.1 pattern |
| `debugDoingLayout`, `debugPrintLayouts`, `debugPaintSizeEnabled` and the other global debug flags of `rendering/debug.dart`; `debugCheckIntrinsicSizes` | the flags are fields of `DebugOptions`, held by the `PipelineOwner` (children share the parent's) and the `RendererBinding`, so tests can change them without a global; the transient state (`activeLayout`, `activePaint`, `checkingIntrinsics`, `doingBaseline`, `dryLayoutCalculationValid`) is one shared `renderDebugState` per owner tree; `DebugAssertAllRenderVarsUnset` is a method | no global mutable state (CLAUDE.md) |
| `RendererBinding.instance`, `RendererBinding` as a mixin of `WidgetsFlutterBinding` | `NewRendererBinding(dispatcher, scheduler, gestures, RendererBindingOpts{Debug, ReportError})`; `AddRenderView`, `RemoveRenderView`, `RenderViews`, `RootPipelineOwner`, `DrawFrame() error`, `DeferFirstFrame`, `AllowFirstFrame`, `ResetFirstFrameSent`, `Reassemble`, `ForceRepaint`, `HitTestInView` and the three dumps are methods; `DrawFrame` returns the first error of a view's compositing and the persistent-frame callback reports it through `ReportError` (TODO(step 7.3): the framework's error reporter and the recovery of panics in layout and paint) | no singletons |
| `AllowFirstFrame` schedules a warm-up frame (`scheduleWarmUpFrame`) | schedules a normal frame; the scheduler has no warm-up frame | the engine here draws on demand |
| `RenderView` takes a `FlutterView` and a `ViewConfiguration` with `BoxConstraints` and `devicePixelRatio`; `compositeFrame` builds a scene and calls `view.render` | `NewRenderView(view *ui.View, opts)`; `ViewConfiguration` is a comparable struct built by `NewViewConfiguration` (with `FromView` through `NewViewConfigurationFromView`); `CompositeFrame() error` returns the render error; `PrepareInitialFrame` is a method; system chrome (`SystemChrome.setSystemUIOverlayStyle`) is not ported (TODO: step 10.4) | `ui.View.Render` returns an error; the services library arrives later |
| `Layer` is an abstract class with `addToScene`, `findAnnotations`, subclasses `ContainerLayer`, `OffsetLayer`, `ClipRectLayer`, … | `Layer` is an interface; `LayerBase`, `ContainerLayer` and `OffsetLayer` are embedded structs (`InitLayer`, `InitContainerLayer`, `InitOffsetLayer`), and `ContainerLayerLike` and `OffsetLayerLike` are the interfaces that accept any layer with those bases; `LayerHandle[T]` counts references like Flutter's and disposes at zero; `AnnotationResult` is not generic (it stores `any` values and entries with the offset); `BackdropKey` is made from an atomic counter | Go has no inheritance; generics on a method set are not possible |
| `TextureLayer`, `PlatformViewLayer`, `PerformanceOverlayLayer` | not ported | no external textures, platform views or Skia overlay in this engine |
| `OffsetLayer.toImage(bounds, pixelRatio)` rasterizes through `Scene.toImage` | `ToImage(bounds, pixelRatio, ui.ToImageOpts{Rasterizer})` and `ToImageSync` take the rasterizer (the `soft` or GPU backend) in the options and return `(*ui.Image, error)`; without one they fail with `RasterizerMissing` | the engine API has no global rasterizer |
| `PaintingContext` records through `PictureRecorder` and `Canvas` in `_startRecording` | `PaintingContext` uses `ui.PictureRecorder` and `ui.Canvas` directly; `PushClipRect`, `PushClipRRect`, `PushClipRSuperellipse`, `PushClipPath` (with `PushClipRectOpts` etc. for `ClipBehavior` and `OldLayer`), `PushColorFilter`, `PushTransform`, `PushOpacity` (with the old layer positional) and `PushLayer` take the painting callback as a positional argument; `PaintChild` is a method and `RepaintCompositedChild` a function; `AddCompositionCallback` returns the function that removes it | Dart named arguments become positional arguments (§12) |
| `InformationCollector` returns `Iterable<DiagnosticsNode>`; `assert` in `BoxConstraints.debugAssertIsValid` and `RenderBox` | `InformationCollector` is `func() []string`; the asserts are panics with tags (`BoxNotLaidOut`, `BoxSizeNotSet`, `BoxSizeViolatesConstraints`, `LayerDisposedTwice`, …), each with a route test in `rendering/errors_debug_test.go`; the checks run only when `foundation.DebugMode` is true | §14 |
| `RenderBox._DebugSize` (a `Size` that remembers its owner), `Size` access checks in `debugAssertDoesMeetConstraints` | not ported; `RenderBox.Size()` panics with `BoxNotLaidOut` or `BoxSizeAccessInvalid` in debug builds and returns the size in release builds | Go has no subclass of a value type |
| `RenderObject.markNeedsSemanticsUpdate`, `SemanticsConfiguration`, `PipelineOwner.flushSemantics` | `FlushSemantics` is a stub and the semantics hooks are no-ops (TODO(step 6.7)) | semantics arrive in step 6.7 |
| `MouseTracker` fallback cursor from `SystemMouseCursors.basic` | the tracker passes no cursor when no region wants one (TODO(step 10.4)) | `services` has no cursors yet |
| `clipPath` tests use a point on the outline | the ported test uses a point just inside the outline, because the software path hit test treats the outline as outside | the soft path containment test differs on the boundary |
| `RenderProxyBox`, `RenderShiftedBox`, `RenderAligningShiftedBox`, `RenderProxyBoxWithHitTestBehavior` are classes (`RenderProxyBoxMixin` and `RenderObjectWithChildMixin` mix the behavior into other bases) | embedded structs: `RenderProxyBox{RenderBox; RenderObjectWithChild[Box]}` with `InitRenderProxyBox(&r.RenderProxyBox, self, child)`, `InitRenderShiftedBox`, `InitRenderAligningShiftedBox(…, alignment, textDirection, child)` and `InitRenderProxyBoxWithHitTestBehavior(…, behavior, child)`; a subclass embeds the one it extends and its methods shadow the base's, with `ComputeSizeForNoChild` called through `self`; `RenderProxyBoxMixin` on a foreign base is not ported (embed `RenderProxyBox`); `DebugPaintSize` is called through `self` by `RenderBox.DebugPaint` | no inheritance; the self pattern (§12.5) |
| `RenderX({required …, RenderBox? child})` | `NewRenderX(required…, child Box, opts ...RenderXOpts) *RenderX`: the child is positional and last before the options, `nil` for none (§12.2); properties are `X()` and `SetX(v)`; `final` fields that Dart makes mutable and public (`RenderCustomPaint.isComplex`, `RenderPointerListener.onPointerDown`, `RenderTransform.transformHitTests`, `RenderMetaData.metaData`) are exported fields | §12.1–12.3 |
| `HitTestBehavior? behavior` of `RenderProxyBoxWithHitTestBehavior` | `HitTestBehavior` with `HitTestBehaviorUnspecified` as the zero value, which is `deferToChild`, except in `RenderMouseRegion` where it is `opaque`; `Name()`/`String()` like the other enums | §12.3 |
| `CustomClipper<T>`, `CustomPainter`, `SingleChildLayoutDelegate` are abstract classes that extend `Listenable` | interfaces (`CustomClipper[T]`, `CustomPainter`, `SingleChildLayoutDelegate`) with `foundation.Listenable` embedded, and `CustomClipperBase[T]`, `CustomPainterBase`, `SingleChildLayoutDelegateBase` to embed for the `reclip`/`repaint`/`relayout` listenable and the defaults (`NewCustomClipperBase(reclip)`); `shouldReclip`, `shouldRepaint` and `shouldRelayout` take the old object as the interface; `runtimeType` comparisons use `foundation.RuntimeTypeName`; `ShapeBorderClipper` compares shapes with `reflect.DeepEqual` | no inheritance; `ShapeBorder` has no `Equal` |
| `shaderCallback` and `constraintsTransform` setters skip an equal function | the setters always apply the new function (`RenderShaderMask.SetShaderCallback` repaints, `RenderConstraintsTransformBox.SetConstraintsTransform` compares the new child constraints) | Go functions can not be compared |
| `RenderBackdropFilter({filter, filterConfig, …})` with `ImageFilterConfig` | `NewRenderBackdropFilter(filter ui.ImageFilter, child, opts)`; `ImageFilterConfig` is not ported (it exists to make a filter from the bounds, for shader-based blurs), so `filter` and `filterConfig` are one property | the engine's filters take no bounds |
| `RenderAnimatedOpacity`, `RenderAnimatedOpacityMixin`, `RenderSliverAnimatedOpacity` | not ported yet: they listen to an `Animation<double>` (TODO(step 8.1), with the animation library); `RenderPerformanceOverlay` is step 17.3 | `animation` is empty until M8 |
| `RenderSemanticsGestureHandler`, `RenderSemanticsAnnotations`, `RenderBlockSemantics`, `RenderMergeSemantics`, `RenderExcludeSemantics`, `RenderIndexedSemantics`, `describeSemanticsConfiguration`, `visitChildrenForSemantics` and `assembleSemanticsNode` of the render objects, `CustomPainter.semanticsBuilder` and `CustomPainterSemantics`, `RenderIgnorePointer`'s and `RenderAbsorbPointer`'s `isBlockingUserActions` | not ported yet (TODO(step 6.7)); the setters call `MarkNeedsSemanticsUpdate` (a no-op until then) | semantics is step 6.7 |
| `RenderIgnorePointer.ignoringSemantics`, `RenderAbsorbPointer.ignoringSemantics` (deprecated) | not ported | deprecated; `ExcludeSemantics` replaces it |
| `RenderClip*.debugPaintSize` draws the dashed outline and a scissors glyph with a `TextPainter` | draws the outline only | a render object has no text engine |
| `DebugOverflowIndicatorMixin` draws stripes and labels ("RIGHT OVERFLOWED BY 25 PIXELS") and reports a `FlutterError` | `DebugOverflowIndicator` (embedded; `PaintOverflowIndicator(self, ctx, offset, containerRect, childRect, hints...)`, `ResetOverflowReport` from `Reassemble`) draws the stripes only; the report is an error tagged `RenderOverflow`, delivered once to `DebugOptions.ReportError` (set from `RendererBindingOpts.ReportError`), or printed with `DebugPrint` when none is set | no text engine in render objects; the error reporter is step 7.3 |
| `RenderCustomPaint` / `CustomPainter.paint` asserts on unbalanced `save`/`restore` with a `FlutterError` | panics `CustomPainterSaveUnbalanced` or `CustomPainterRestoreUnbalanced`; `RenderDecoratedBox` with a painter that leaves the canvas unbalanced panics `DecorationPainterSaveMismatch` | §14 |
| `RenderMouseRegion implements MouseTrackerAnnotation`, with public mutable `onEnter`, `onHover`, `onExit` | implements `services.MouseTrackerAnnotation` with `OnEnter()`, `OnExit()`, `Cursor()` and `ValidForMouseTracker()` and has `SetOnEnter`, `SetOnHover`, `SetOnExit`, `SetCursor`, `SetOpaque`, `SetHitTestBehavior` | the interface of step 5.6 |
| `RenderAnnotatedRegion<T extends Object>` | `RenderAnnotatedRegion` holds an `any` (like `AnnotatedRegionLayer`) | see §12.10 above |
| `RenderTransform.transform` is a `Matrix4` of doubles; `translate(x, [y, z])`, `scale(x, [y, z])` | the matrix holds float32 (`ui.Matrix4`); `Translate(x, y, z)` and `Scale(x, y, z)` take all three; `filterQuality` is `Opt[ui.FilterQuality]` and `origin` is `Opt[ui.Offset]` | the matrix type of step 2; no optional positional parameters |
| `RelativeRect.lerp(a, b, t)` with a null `b` reads `b!` | `RelativeRectLerp(a, b *RelativeRect, t)` scales `a` when `b` is nil | evidently intended |
| `RenderFlex`, `RenderStack`, `RenderIndexedStack`, `RenderWrap`, `RenderFlow`, `RenderTable`, `RenderListBody`, `RenderCustomMultiChildLayoutBox` mix `ContainerRenderObjectMixin` and `RenderBoxContainerDefaultsMixin` into `RenderBox` | structs that embed `RenderBox` and `ContainerRenderObject[Box]` (`NewRenderX(children []Box, opts ...RenderXOpts)`, `children` is positional and `nil` for none); the defaults are the functions `DefaultPaint`, `DefaultHitTestChildren` and the baseline functions; `RenderTable` keeps a flat list of cells and implements the child model itself | the self pattern (§12.5); `RenderTable`'s children are not a linked list |
| `FlexParentData.flex` is `int?` and `fit` is `FlexFit?`; `StackParentData`'s `top`, `right`, `bottom`, `left`, `width`, `height` are `double?`; `TableCellParentData.x`, `y` are `int?` | `Flex`, `X`, `Y` and the stack's six values are `foundation.Opt`; `Fit`, `VerticalAlignment` are enums whose zero value (`FlexFitUnspecified`, `TableCellVerticalAlignmentUnspecified`) is the null | §12.1 |
| `RenderFlex(textDirection: null)`, `textBaseline: null` | `ui.TextDirectionUnspecified` and `ui.TextBaselineUnspecified` are the null; the enums `MainAxisAlignment`, `MainAxisSize`, `CrossAxisAlignment`, `FlexFit`, `WrapAlignment`, `WrapCrossAlignment`, `StackFit`, `TableCellVerticalAlignment` have an `Unspecified` zero value that the constructors and setters turn into the default | §12.1 |
| `_debugCheckConstraints`, `_debugHasNecessaryDirections` of `RenderFlex` and `RenderWrap`, `_debugCheckConstraints` of `RenderListBody`, the `layoutChild`/`positionChild` asserts of `MultiChildLayoutDelegate`, the asserts of `RenderTable` | panics with tags: `RenderFlexUnboundedFlex`, `FlexDirectionMissing`, `FlexBaselineMissing`, `FlexSpacingNegative`, `WrapDirectionMissing`, `StackUnboundedConstraints`, `StackChildNotPositioned`, `ListBodyBoundedMainAxis`, `ListBodyUnboundedCrossAxis`, `FlowChildPaintedTwice`, `MultiChildLayout…`, `Table…`; an error of a layout is not caught by the pipeline yet (step 7.3) | §14; the message is one string, without `ErrorHint` parts |
| `RenderIndexedStack(index: null)`; the default `index` is 0 | `RenderIndexedStackOpts.Index` is an `Opt` where `None` means the default 0; `SetIndex(None)` is the null | `Opt` of an `Opt` would be needed otherwise |
| `RenderFlow.paintChild(i, {transform, opacity = 1.0})` | `PaintChildAt(i, FlowPaintChildOpts{Transform, Opacity})`; a `nil` transform is the identity and a `None` opacity is 1 | options struct (§12.1); the name says that it takes an index |
| `FlowDelegate`, `MultiChildLayoutDelegate` are abstract classes with `Listenable? repaint/relayout` | interfaces with the base structs `FlowDelegateBase` and `MultiChildLayoutDelegateBase` (the base keeps the state of a layout and has `LayoutChild`, `PositionChild`, `HasChild`) | §12.5 |
| `MultiChildLayoutParentData.id` is an `Object?` | `ID any`, which must be comparable | it is a map key |
| `MultiChildLayoutDelegate.layoutChild` wraps an invalid constraints assertion in a `FlutterError` | the panic of `BoxConstraints.DebugAssertIsValid` passes through | one tag fewer |
| `TableColumnWidth` classes are `const` and compare by identity | `FixedColumnWidth`, `FractionColumnWidth`, `FlexColumnWidth`, `IntrinsicColumnWidth`, `MaxColumnWidth`, `MinColumnWidth` are comparable values behind the `TableColumnWidth` interface | map and equality checks work |
| `RenderTable.rowDecorations =` skips an identical list | always recreates the painters | slices can not be compared |
| `RenderTable.describeSemanticsConfiguration`, `assembleSemanticsNode`, `clearSemantics`; `RenderIndexedStack.visitChildrenForSemantics` | not ported yet | TODO(step 6.7); the `toStringDeep` of a table lacks the `semantic boundary` line until then |
| `TableBorder` is an immutable class with `const` constructors and `BorderSide.none` defaults | `TableBorder` is a comparable struct; `NewTableBorder`, `NewTableBorderAll`, `NewTableBorderSymmetric` set the zero sides to `BorderSideNone`; `TableBorderLerp` takes pointers (`nil` is null) | §12.1 |
| `RenderWrap.debugFillProperties` reports `runSpacing` under the name `crossAxisAlignment` | the same, so that the dump matches | kept for fidelity |
| `ListBody` quirk: `computeMin/MaxIntrinsicHeight` of a vertical list takes the maximum of the children | the same | kept for fidelity |
| `RenderImage.opacity` is an `Animation<double>?` | a `foundation.ValueListenable[float64]`; `RenderImage` listens while attached | `Animation` is step 8.1; an `Animation` will be a `ValueListenable` then |
| `getBoxesForSelection(TextSelection, ...)` and `RenderParagraph.selectionColor` are `TextSelection`/`Color` | `GetBoxesForSelection(ui.TextRange, ...)`; positions are byte offsets, like everything in the text engine | `TextSelection` is step 11.2 |
| `_SelectableFragment` with the selection logic | a fragment that registers with the `SelectionRegistrar` and reports its content length, bounding boxes and size; the selection events are not handled | TODO(step 11.5) |
| `RenderParagraph.assembleSemanticsNode`, `describeSemanticsConfiguration`, `visitChildrenForSemantics`, `clearSemantics` | not ported yet | TODO(step 6.7) |
| `RenderParagraph` gets the font collection from the `dart:ui` engine | `RenderParagraphOpts.TextEngine` (and `painting.TextPainterOpts.TextEngine`), else `ui.PlatformDispatcher.TextEngine()` | the same as `TextPainter` |
| `RenderParagraph` relayouts when the system fonts change (`SystemFontsDidChange`) | the method exists and clears the layout; nothing calls it yet | TODO(step 7.4) wires it to the binding |
| `RenderParagraph.hitTestChildren` adds a `TextSpan` as a `HitTestTarget` | the same; the span at a position is found with `ui.NewTextPosition`, whose affinity is downstream | the zero `ui.TextPosition` is upstream |
| `ParagraphStyle.ellipsis` with no `maxLines` | the first line that would wrap is ellipsized and the text after it dropped (`paragraph_test.dart` expects it) | updates the row on `ellipsis` above; `internal/text/layout` |
| `renderingtest` builds the binding itself | `internal/renderhost` builds the fake engine and the rendering binding; `renderingtest.Harness` embeds it | the proof scenes need it too and `internal/gpu/soft`'s tests import the proof scenes |

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
| golang.org/x/text | BSD-3-Clause | Go dependency that go-text already requires; `internal/text/layout` uses its `unicode/bidi` class table for UAX #9 (§12.10), added in step 4.4. Flagged for review |
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
