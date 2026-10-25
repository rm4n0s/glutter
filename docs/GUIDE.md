# Glutter for Flutter developers

Glutter has Flutter's framework API on its own engine, so what you know carries over: the
widget tree, `BuildContext`, `setState`, `Theme.of`, `Navigator`, animation controllers,
`CustomPaint`. This page shows eleven things side by side, in Dart and in Go. The rules behind
them are in [SPEC §12](SPEC.md#12-api-conventions-flutter--go); this is the short version.

**The rules in one paragraph.** A widget is made by `NewX(required..., opts ...XOpts)`. The
parameters that Flutter marks `required` are positional, in Flutter's order, except `child`
and `children`, which come last; the rest are fields of the options struct `XOpts`, named as
in Flutter. A nullable parameter whose `null` is not Go's zero value is `m.Some(v)`
(`foundation.Opt[T]`). Every enum's zero value is `XUnspecified`, which means "the widget's
default". A named constructor `ListView.builder` is `NewListViewBuilder`; a static method
`Theme.of` is `ThemeOf`; `maybeOf` returns `(value, ok)`. Widgets are immutable and their
fields are read through methods (`w.Title()`). Apps import one package:

```go
import m "github.com/rm4n0s/glutter/material"
```

Each Go snippet below is checked by `go vet` in the test of this directory
(`docs/docs_test.go`). The imports that they use, and that are left out for brevity, are
`m` (`material`), `foundation`, `ui`, `painting`, `rendering`, `animation`, `context`,
`strconv` and `time`.

## 1. A widget without state

Flutter:

```dart
class Greeting extends StatelessWidget {
  const Greeting({super.key, required this.name});
  final String name;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Text('Hello, $name', style: Theme.of(context).textTheme.titleLarge),
    );
  }
}
```

Glutter:

```go
type Greeting struct {
	m.StatelessWidget
	name string
}

func NewGreeting(name string, opts ...m.WidgetOpts) *Greeting {
	return &Greeting{StatelessWidget: m.NewStatelessWidgetBase(opts...), name: name}
}

func (g *Greeting) Name() string { return g.name }

func (g *Greeting) Build(bc m.BuildContext) m.Widget {
	return m.NewPadding(m.NewEdgeInsetsAll(16),
		m.NewText("Hello, "+g.name, m.TextOpts{Style: m.ThemeOf(bc).TextTheme.TitleLarge}))
}
```

The widget embeds `m.StatelessWidget` (Flutter's superclass), and `key` is the `Key` field of
`m.WidgetOpts`, which the constructor passes on. `Theme.of(context).textTheme.titleLarge` is
`m.ThemeOf(bc).TextTheme.TitleLarge`: a function for the static method, an exported field for
the property.

## 2. A widget with state

Flutter:

```dart
class Counter extends StatefulWidget {
  const Counter({super.key});
  @override
  State<Counter> createState() => _CounterState();
}

class _CounterState extends State<Counter> {
  int _count = 0;

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: () => setState(() => _count++),
      child: Text('Pushed $_count times'),
    );
  }
}
```

Glutter:

```go
type Counter struct{ m.StatefulWidget }

func NewCounter(opts ...m.WidgetOpts) *Counter {
	return &Counter{StatefulWidget: m.NewStatefulWidgetBase(opts...)}
}

func (c *Counter) CreateState() m.State { return &counterState{} }

type counterState struct {
	m.StateOf[*Counter]
	count int
}

func (s *counterState) Build(bc m.BuildContext) m.Widget {
	return m.NewTextButton(
		func() { s.SetState(func() { s.count++ }) },
		m.NewText("Pushed "+strconv.Itoa(s.count)+" times"))
}
```

`State<Counter>` is `m.StateOf[*Counter]`, which gives `s.Widget()` (Flutter's `widget`),
`s.Context()`, `s.SetState(...)` and the lifecycle methods you override (`InitState`,
`DidUpdateWidget`, `Dispose`). `onPressed` is a required parameter, so it is positional, and
`nil` disables the button, as `null` does in Flutter.

## 3. Layout, with options

Flutter:

```dart
Column(
  crossAxisAlignment: CrossAxisAlignment.start,
  children: [
    const Text('Title'),
    const SizedBox(height: 8),
    Row(children: [
      const Expanded(child: Text('Left')),
      Container(width: 80, color: Colors.amber, child: const Text('Right')),
    ]),
  ],
)
```

Glutter:

```go
func layout() m.Widget {
	return m.NewColumn([]m.Widget{
		m.NewText("Title"),
		m.NewSizedBox(nil, m.SizedBoxOpts{Height: m.Some(8.0)}),
		m.NewRow([]m.Widget{
			m.NewExpanded(m.NewText("Left")),
			m.NewContainer(m.NewText("Right"), m.ContainerOpts{
				Width: m.Some(80.0),
				Color: m.Some(m.Colors.Amber.Primary),
			}),
		}),
	}, m.ColumnOpts{CrossAxisAlignment: m.CrossAxisAlignmentStart})
}
```

`children` is positional and the first argument because `Column` has nothing required before
it; `child` of `Expanded` and `Container` is the first positional argument as well, and `nil`
stands for "no child". `width: 80` is `m.Some(80.0)`, since a `null` width means "not
constrained" and `0` is a width. The enum is `m.CrossAxisAlignmentStart`.

## 4. Named constructors and builders

Flutter:

```dart
ListView.builder(
  itemCount: names.length,
  itemBuilder: (context, i) => ListTile(
    leading: const Icon(Icons.person),
    title: Text(names[i]),
    onTap: () => debugPrint('tapped $i'),
  ),
)
```

Glutter:

```go
func nameList(names []string) m.Widget {
	return m.NewListViewBuilder(func(bc m.BuildContext, i int) m.Widget {
		return m.NewListTile(m.ListTileOpts{
			Leading: m.NewIcon(m.Icons.Person()),
			Title:   m.NewText(names[i]),
			OnTap:   func() { println("tapped", i) },
		})
	}, m.ListViewOpts{ItemCount: m.Some(len(names))})
}
```

`ListView.builder` is `NewListViewBuilder`; the builder is a required parameter and so is
positional. `Icons.person` is the method `m.Icons.Person()` (SPEC §12.10 says why: 8,800 icons
as fields make the Go compiler run out of memory).

## 5. Inherited data and `of`

Flutter:

```dart
Widget build(BuildContext context) {
  final scheme = Theme.of(context).colorScheme;
  final width = MediaQuery.sizeOf(context).width;
  final brightness = Theme.maybeBrightnessOf(context);
  return Container(
    color: width > 600 ? scheme.primaryContainer : scheme.surface,
    child: TextButton(
      onPressed: () => Navigator.of(context).pop(),
      child: const Text('Close'),
    ),
  );
}
```

Glutter:

```go
func buildResponsive(bc m.BuildContext) m.Widget {
	scheme := m.ThemeOf(bc).ColorScheme
	width := m.MediaQuerySizeOf(bc).Width
	brightness, haveTheme := m.ThemeMaybeBrightnessOf(bc)
	_, _ = brightness, haveTheme
	color := scheme.Surface()
	if width > 600 {
		color = scheme.PrimaryContainer()
	}
	return m.NewContainer(
		m.NewTextButton(func() { m.NavigatorPop(bc) }, m.NewText("Close")),
		m.ContainerOpts{Color: m.Some(color)})
}
```

The colors of a `ColorScheme` are methods (`scheme.Surface()`), because the scheme computes the
ones that were not set from the others. Static methods are functions named after the class (`ThemeOf`, `NavigatorPop`), `of` and the
aspect-specific `sizeOf` keep their meaning, and `maybeOf` returns a second value that says
whether there was an ancestor, so there is no `null` to check. `Navigator.of(context).pop()`
has the shortcut `NavigatorPop(bc)`; the result of a route, which in Dart is the `Future` of
the push, is a channel (next example).

## 6. Navigation and results

Flutter:

```dart
Future<void> pick(BuildContext context) async {
  final color = await Navigator.of(context).push<String>(
    MaterialPageRoute(builder: (context) => const PickerPage()),
  );
  if (color != null) setState(() => picked = color);
}
```

Glutter:

```go
type Picker struct{ m.StatefulWidget }

type pickerState struct {
	m.StateOf[*Picker]
	picked string
}

func (s *pickerState) pick() {
	result := m.NavigatorPush(s.Context(), m.NewMaterialPageRoute(func(bc m.BuildContext) m.Widget {
		return m.NewScaffold(m.ScaffoldOpts{
			Body: m.NewTextButton(func() { m.NavigatorPop(bc, "blue") }, m.NewText("Blue")),
		})
	}))
	s.Go(func(ctx context.Context) {
		select {
		case v := <-result:
			if color, ok := v.(string); ok {
				s.SetState(func() { s.picked = color })
			}
		case <-ctx.Done():
		}
	})
}
```

A route's result is not generic: `NavigatorPush` returns a `<-chan any` that gets the value
once and is closed. Waiting for it is waiting on a channel, and `State.Go` runs the goroutine
for you: its `ctx` ends when the state is disposed, so nothing outlives the widget. Only the
UI goroutine may touch the framework, so the goroutine hands its result over with
`SetState` (SPEC §13).

## 7. Animation

Flutter:

```dart
class _PulseState extends State<Pulse> with SingleTickerProviderStateMixin {
  late final AnimationController _controller = AnimationController(
    vsync: this,
    duration: const Duration(seconds: 1),
  )..repeat(reverse: true);
  late final Animation<double> _scale = Tween(begin: 0.7, end: 1.2)
      .animate(CurvedAnimation(parent: _controller, curve: Curves.easeInOut));

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) =>
      ScaleTransition(scale: _scale, child: widget.child);
}
```

Glutter:

```go
type Pulse struct {
	m.StatefulWidget
	child m.Widget
}

func NewPulse(child m.Widget, opts ...m.WidgetOpts) *Pulse {
	return &Pulse{StatefulWidget: m.NewStatefulWidgetBase(opts...), child: child}
}

func (p *Pulse) CreateState() m.State { return &pulseState{} }

type pulseState struct {
	m.StateOf[*Pulse]
	m.SingleTickerProvider
	controller *animation.AnimationController
	curved     *animation.CurvedAnimation
	scale      animation.Animation[float64]
}

func (s *pulseState) InitState() {
	s.controller = animation.NewAnimationController(s, animation.AnimationControllerOpts{
		Duration: m.Some(time.Second),
	})
	s.controller.Repeat(animation.RepeatOpts{Reverse: true})
	s.curved = animation.NewCurvedAnimation(s.controller, animation.Curves.EaseInOut)
	s.scale = animation.NewTween(0.7, 1.2).Animate(s.curved)
}

func (s *pulseState) Dispose() {
	s.curved.Dispose()
	s.controller.Dispose()
}

func (s *pulseState) Build(bc m.BuildContext) m.Widget {
	return m.NewScaleTransition(s.scale, s.Widget().child)
}
```

The mixin `SingleTickerProviderStateMixin` is the embedded `m.SingleTickerProvider`, and
`vsync: this` is the state itself, the first argument. Go has no `late final`, so the fields
are made in `InitState`. A `Tween` is generic (`NewTween(0.7, 1.2)` is a `*Tween[float64]`),
and `Curves.easeInOut` is the read-only value `animation.Curves.EaseInOut`. You dispose what
you made, as in Flutter; the debug build tells you about a controller that is left over.

## 8. Asynchronous data

Flutter:

```dart
FutureBuilder<String>(
  future: fetchGreeting(),
  builder: (context, snapshot) {
    if (snapshot.hasError) return Text('Error: ${snapshot.error}');
    if (!snapshot.hasData) return const CircularProgressIndicator();
    return Text(snapshot.data!);
  },
)
```

Glutter:

```go
func greeting() m.Widget {
	return m.NewFutureBuilder(func(ctx context.Context) (string, error) {
		return fetchGreeting(ctx)
	}, func(bc m.BuildContext, snapshot m.AsyncSnapshot[string]) m.Widget {
		if snapshot.HasError() {
			return m.NewText("Error: " + snapshot.Error().Error())
		}
		if !snapshot.HasData() {
			return m.NewCircularProgressIndicator()
		}
		return m.NewText(snapshot.Data())
	})
}

func fetchGreeting(ctx context.Context) (string, error) {
	select {
	case <-time.After(time.Second):
		return "Hello", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
```

A `Future` is a function that takes a `context.Context` and returns a value and an error (a
`Stream` is a `<-chan T`). The widget runs the function on its own goroutine, cancels the
context when it is disposed, and builds with the snapshot on the UI thread. Everything that
does I/O takes a `ctx` first, as in the rest of Go.

## 9. Text input and forms

Flutter:

```dart
final _formKey = GlobalKey<FormState>();
final _controller = TextEditingController();

Form(
  key: _formKey,
  child: Column(children: [
    TextFormField(
      controller: _controller,
      decoration: const InputDecoration(labelText: 'Name'),
      validator: (v) => (v == null || v.isEmpty) ? 'Required' : null,
    ),
    ElevatedButton(
      onPressed: () {
        if (_formKey.currentState!.validate()) save(_controller.text);
      },
      child: const Text('Save'),
    ),
  ]),
)
```

Glutter:

```go
type nameForm struct {
	m.StatefulWidget
}

func (f *nameForm) CreateState() m.State { return &nameFormState{} }

type nameFormState struct {
	m.StateOf[*nameForm]
	formKey    *m.GlobalKey[*m.FormState]
	controller *m.TextEditingController
}

func (s *nameFormState) InitState() {
	s.formKey = m.NewGlobalKey[*m.FormState]()
	s.controller = m.NewTextEditingController()
}

func (s *nameFormState) Dispose() { s.controller.Dispose() }

func (s *nameFormState) Build(bc m.BuildContext) m.Widget {
	return m.NewForm(m.NewColumn([]m.Widget{
		m.NewTextFormField(m.TextFormFieldOpts{
			Controller: s.controller,
			Decoration: ptr(m.NewInputDecoration(m.InputDecorationOpts{LabelText: m.Some("Name")})),
			Validator: func(v m.Opt[string]) m.Opt[string] {
				if text, ok := v.Get(); !ok || text == "" {
					return m.Some("Required")
				}
				return m.None[string]()
			},
		}),
		m.NewElevatedButton(func() {
			if form, ok := s.formKey.CurrentState(); ok && form.Validate() {
				println("saved", s.controller.Text())
			}
		}, m.NewText("Save")),
	}), m.FormOpts{Key: s.formKey})
}

func ptr[T any](v T) *T { return &v }
```

`TextEditingController()` is `m.NewTextEditingController()`; the optional initial text is
the one variadic parameter. `InputDecoration` is an immutable value class with many fields:
it is passed as a pointer (`nil` is `null`), made by `NewInputDecoration`. A global key is generic in the
state it finds (`m.GlobalKey[*m.FormState]`), and `CurrentState()` returns the state and whether
there is one. A validator gets the value and returns the error text, both as `m.Opt[string]`
in place of a nullable `String`. A
controller is a `Listenable` that you dispose, as in Dart.

## 10. Custom painting

Flutter:

```dart
class DiscPainter extends CustomPainter {
  DiscPainter(this.color);
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()..color = color;
    canvas.drawCircle(size.center(Offset.zero), size.shortestSide / 2, paint);
  }

  @override
  bool shouldRepaint(DiscPainter old) => old.color != color;
}

CustomPaint(painter: DiscPainter(Colors.teal), size: const Size(100, 100))
```

Glutter:

```go
type DiscPainter struct {
	rendering.CustomPainterBase
	color ui.Color
}

func NewDiscPainter(color ui.Color) *DiscPainter {
	return &DiscPainter{CustomPainterBase: rendering.NewCustomPainterBase(nil), color: color}
}

func (p *DiscPainter) Paint(canvas *ui.Canvas, size ui.Size) {
	paint := ui.NewPaint()
	paint.SetColor(p.color)
	canvas.DrawCircle(ui.Offset{Dx: size.Width / 2, Dy: size.Height / 2}, min(size.Width, size.Height)/2, paint)
}

func (p *DiscPainter) ShouldRepaint(old rendering.CustomPainter) bool {
	o, ok := old.(*DiscPainter)
	return !ok || o.color != p.color
}

func disc() m.Widget {
	return m.NewCustomPaint(nil, m.CustomPaintOpts{
		Painter: NewDiscPainter(m.Colors.Teal.Primary),
		Size:    ui.Size{Width: 100, Height: 100},
	})
}
```

`CustomPainter` is an interface, and `rendering.CustomPainterBase` is the part you embed to
get the `Listenable` (the `repaint` argument) and the defaults for `HitTest`. `Paint()` with
its cascade `..color =` is `ui.NewPaint()` and a setter, `paint.SetColor(...)`. `canvas` is a `*ui.Canvas`
with Flutter's methods (`DrawCircle`, `DrawPath`, `Save`, `Translate`, ...).

## 11. Fonts

Flutter declares fonts in `pubspec.yaml` and uses them by family name; `FontLoader` loads one
at run time. Glutter has no manifest: you give the bytes of a TrueType or OpenType file under a
family name, in `RunAppOpts.Fonts` (before the first frame) or later with `FontLoader`. A
code editor's monospace font is the usual case:

```go
package main

import (
	"context"
	"os"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
)

func editorText(code string) m.Widget {
	return m.NewText(code, m.TextOpts{Style: m.NewTextStyle(m.TextStyleOpts{
		FontFamily:         "JetBrains Mono",
		FontFamilyFallback: []string{"monospace"},
		FontSize:           m.Some(13.0),
	})})
}

func main() {
	err := m.RunApp(editorText("fmt.Println()"), m.RunAppOpts{
		Assets: os.DirFS("assets"), // or an embed.FS
		Fonts: []m.FontSource{
			{Family: "JetBrains Mono", Asset: "fonts/JetBrainsMono-Regular.ttf"},
			{Family: "JetBrains Mono", Asset: "fonts/JetBrainsMono-Bold.ttf"},
			{Family: "JetBrains Mono", Asset: "fonts/JetBrainsMono-Italic.ttf"},
		},
	})
	if err != nil {
		os.Exit(1)
	}
}

// loadFromDisk adds a font while the app runs, on the UI thread: the text that is on the
// screen is laid out again.
func loadFromDisk(bc m.BuildContext, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	loader := services.NewFontLoader("My Mono")
	loader.AddFontBytes(data)
	return loader.Load(context.Background(), m.WidgetsBindingOf(bc).PlatformDispatcher())
}
```

Fonts of one family make a family: each file tells its own weight and slant (the OS/2 table; a
variable font counts for every weight of its `wght` axis), so the three files above serve
`FontWeight.w400`, `w700` and `FontStyle.italic`, and the closest one is used for a weight in
between. The loaded family is also a fallback: it works in `FontFamilyFallback` the same way.
The fonts of a style are tried in the order `FontFamily`, then each `FontFamilyFallback`
entry, then the system's default font; a name that no installed or loaded font has gives
nothing, so `FontFamilyFallback: []string{"monospace"}` reaches the user's monospace font
(`monospace`, `sans-serif` and `serif` stand for the system's own). In a widget test, give the
fonts to `gluttertest.Opts.Fonts`.

Flutter gives a tab no stops: it is a white space. A code editor wants it to advance to the next
tab stop, so `TextPainterOpts.TabSize` (and `ui.ParagraphStyleOpts.TabSize` under it) is the
distance between stops as a count of the width of a space in the text's font, as CSS's `tab-size`
is: `TabSize: 4` puts the stops 4 spaces apart, measured from the start of each line (the left edge,
or the right edge of a right-to-left paragraph), and a line that wraps starts its stops again. The
default, 0, is Flutter's behavior. A tab at the end of a line hangs like other white space, the
boxes, positions and carets after a tab are at the stop, and nothing is drawn for the tab itself.
`Text` and `RichText` do not have the option yet; use a `TextPainter`.

## For Flutter web developers

If you know Flutter on the web, this is what is the same and what is not.

- **Build and run.** There is no `flutter build web`: `go run ./cmd/glutter-web serve ./your/app`
  builds and serves it, and `glutter-web build -release -o dist ./your/app` makes the directory to
  deploy (`index.html`, `wasm_exec.js`, `app.wasm`, `app.wasm.gz`). It is one WebAssembly module and one
  canvas; there is no HTML renderer and no DOM widgets, only the hidden text field that the
  browser needs for typing, the clipboard and input methods.
- **Renderers.** Flutter web has CanvasKit and Skwasm. Glutter has WebGPU, then WebGL 2, then a
  software renderer on a 2D canvas, chosen at the first frame (`?GLUTTER_RENDERER=webgl` in the
  address chooses by hand). The first two are the same engine as on Linux, so a frame looks the same
  there and here.
- **Fonts.** There are no system fonts on the web. Text is drawn with the bundled Roboto and
  Material Icons and the fonts the app loads (`ui.LoadFontFromList`, `FontLoader`); a character that
  none has is a missing-glyph box, as in Flutter web without a fallback font.
- **`kIsWeb`** is `foundation.IsWeb`, a constant. `defaultTargetPlatform` is the browser's OS, found
  as Flutter web finds it. Use `IsWeb` for what the page does not have, in your own code too:

```go
package main

import "github.com/rm4n0s/glutter/foundation"

// notesPlace says where an app keeps its notes: a page has no data directory.
func notesPlace() string {
	if foundation.IsWeb {
		return "in memory"
	}
	return "in the data directory"
}

func main() { println(notesPlace()) }
```

  `examples/editor` does this: on the web its notes live in memory until the page is closed, and the
  status bar says so.
- **Not on the web:** a second window, hiding the window, the tray, file dialogs, drag and drop
  out of the app, accessibility, system fonts, closing the page (`SystemNavigator.pop` does
  nothing). Services that are missing return their `...Unavailable` error; `desktop.OpenURI` and
  notifications work.
- **Files and images.** `os` has no real files in a page. Assets come from `embed.FS` as on Linux;
  `http` goes through `fetch`, so the browser's CORS rules apply.
- **Size and speed.** The page loads one large module (the counter is 8.8 MB gzipped in a release
  build) and runs on one thread, so a click that builds a whole new screen takes tens of
  milliseconds. Build with `-release` for anything you measure, and read
  [PERFORMANCE.md](PERFORMANCE.md).

## For Flutter Android developers

If you build Flutter apps for Android, this is what is the same and what is not.

- **Build and run.** There is no `flutter run` or `flutter build apk`: `go run ./cmd/glutter-android run ./your/app`
  builds the APK (one program for arm64 and one for x86_64), installs it on the connected phone or
  the running emulator, starts it and streams its log until you press Ctrl+C. `build -release` makes the
  release APK (`-key` and `-cert` sign it; a debug build is signed with a key the tool makes once), and
  `shot -screen -o file.png` takes a screenshot. It needs the Android SDK and NDK (`ANDROID_HOME`) and
  Java for `apksigner`; a missing piece says which `sdkmanager` package to install. There is no Gradle
  and no `android/` folder: the manifest is generated (one activity with the `configChanges` that
  Flutter declares), and `-app-id`, `-label`, `-icon` and `-permission INTERNET` change it.
- **`--dart-define` and arguments.** `-env GLUTTER_X=v` (repeatable) on `run` and `shot` gives the
  activity an intent extra that is copied into the program's environment before `main` runs:
  `os.Getenv("GLUTTER_X")`. `adb shell am start --es GLUTTER_X v` does the same by hand.
- **Files and logs.** `HOME` is the app's files directory (`getApplicationDocumentsDirectory`), and
  `os` works there as on any Linux. `slog` goes to `logcat` with the level as the priority and the app
  ID as the tag, so `adb logcat -s <app id>` shows only your app; `debugPrint` has no counterpart,
  use `slog`.
- **No plugins.** There is no platform channel and no pub.dev: what a plugin does for you is the
  standard library (`net/http`, `os`), the `desktop` package (`OpenURI`, notifications, which ask for
  the permission), or your own Java and NDK code if you need it. The framework's own platform services
  (clipboard, text input, system UI, haptic feedback, the back gesture, settings) are `services` as in
  Flutter.
- **Back.** `PopScope` and the `Navigator` work as in Flutter: the activity asks the system for the back
  gesture only while something can handle it, so the system's own back (leaving the app) happens when
  nothing can pop. Material's pages use the predictive back transition on Android 14 and later, with
  `PredictiveBackPageTransitionsBuilder` as the theme's default for Android, as in Flutter.
- **Platform checks.** `foundation.DefaultTargetPlatform()` is `TargetPlatformAndroid`, so the framework takes
  the Android branches it takes in Flutter (clamping scroll physics and the overscroll glow or stretch,
  the Android text selection toolbar, tap target sizes, the typography). `GOOS=android` is a tag of the
  Go toolchain: the framework does not check `runtime.GOOS`, and neither should your app.
- **Renderers.** Flutter has Impeller. Glutter has Vulkan, OpenGL ES and a CPU renderer into the
  window, chosen at the first frame (Vulkan on Android 10 and later where the driver is good);
  `-env GLUTTER_RENDERER=gles` chooses by hand, and the log line `renderer chosen` says which one runs.
- **Not on Android:** a second window, the tray, file dialogs, autofill, an Android App Bundle (an APK
  only), and the plugins' channels. Accessibility (TalkBack), the soft keyboard and input methods, the
  system fonts and emoji, dark mode, text scale, locales, reduced motion and the system bars work.
- **Size and start.** The program is a Go binary: a release counter is about 26 MiB per ABI and the
  gallery 37 MiB, and it starts in about 0.6 s on the emulator with the CPU renderer. Build with
  `-release` for anything you measure, and read [PERFORMANCE.md](PERFORMANCE.md).

## What is not the same

- **No `const` widgets.** Go has no const constructors. Build a widget once and reuse it
  (a package-level value, or a field of the state), which is what `const` achieves.
- **Threads.** The framework runs on one goroutine, the UI thread. Other goroutines hand
  work over with `SetState` or the binding's `Post`; `State.Go` starts a goroutine that is
  cancelled with the state. There is no `async` and `await`: blocking is fine on your own
  goroutines.
- **Errors.** Flutter's exceptions are panics only for programmer errors (a debug-build
  `assert` is a check that exists in debug builds, `-tags glutter_release` removes it). A
  failure that can happen at run time is an `error` value.
- **Hot reload** is not possible in Go; `go run` is fast and the examples take no state.
- **Platform code.** `dart:io`, plugins and platform channels are the packages of the
  standard library, and `desktop` (file dialogs, notifications, the tray, opening a URL).
- **Closing to the tray.** Where a Flutter app reaches for the `window_manager` plugin to hide
  its window, a Glutter app calls `SetHidden(true)` on the window's controller
  (`WindowingOwner().ImplicitWindowController()` for the first window), usually from
  `AppLifecycleListener.OnExitRequested`, which then returns `AppExitResponseCancel`. The window
  leaves the screen and the task bar, the app keeps running, and `SetHidden(false)` or
  `Activate()` brings it back as it was; `examples/tray` does this.
- **The window you are in.** `WindowControllerOf(bc)` is the controller of the window that hosts
  `bc` (the first window included) with `Title`, `SetTitle`, `Activate`, `IsActivated` and
  `SetMinimized`; a regular window is also a `MaximizableWindowController` (`SetMaximized`,
  `SetFullscreen`), asked with a type assertion. `RegularWindowControllerOpts.State` and
  `WindowConfig.State` open a window maximized or fullscreen, so it never shows small first.
- **Closing the first window.** By default closing it is a request to exit the app. Give it a
  delegate with `WindowingOwner().SetImplicitWindowDelegate(d)` and it closes like any window;
  `ImplicitWindowController().Destroy()` removes it, its content leaves the tree, and the app goes
  on while other windows are open. For that the other windows must not be inside the first
  window's `View`: use `RunWidget` with a `WindowManager` around the `View`
  (`examples/multiwindow` does).
- **Where the list of every difference is.** SPEC §12.10, one row per deviation, kept up to
  date.
