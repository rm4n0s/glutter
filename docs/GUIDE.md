# Glutter for Flutter developers

Glutter has Flutter's framework API on its own engine, so what you know carries over: the
widget tree, `BuildContext`, `setState`, `Theme.of`, `Navigator`, animation controllers,
`CustomPaint`. This page shows sixteen things, most of them side by side, in Dart and in Go. The rules behind
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
`m` (`material`), `foundation`, `ui`, `painting`, `rendering`, `animation`, `svg`, `offscreen`, `context`,
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

A list that lets the user read while items arrive at its start (a live log, a feed) has one more
option, which Flutter does not have: `MaintainVisibleContentPosition`. Flutter moves the item the
user reads by the height of the items inserted before it, even when the items have keys. With the
option on, the list keeps that item in place when the list is not at its start (the new items are
built when the user scrolls up to them), and at its start it still follows the newest item, as
Flutter does. It needs a key on each item and a `FindChildIndexCallback` that finds an item by its
key:

```go
type logLine struct {
	id   int
	text string
}

// logList shows the newest line at offset 0 (the bottom). New lines are inserted at index 0.
func logList(lines []logLine, ctl m.ScrollController) m.Widget {
	index := func(key foundation.Key) m.Opt[int] {
		for i, l := range lines {
			if key == foundation.Key(foundation.NewValueKey(l.id)) {
				return m.Some(i)
			}
		}
		return m.None[int]()
	}
	return m.NewListViewBuilder(func(bc m.BuildContext, i int) m.Widget {
		return m.NewSizedBox(m.NewText(lines[i].text), m.SizedBoxOpts{
			Key: foundation.NewValueKey(lines[i].id), Height: m.Some(24.0),
		})
	}, m.ListViewOpts{
		Reverse:                        true,
		Controller:                     ctl,
		ItemCount:                      m.Some(len(lines)),
		FindChildIndexCallback:         index,
		MaintainVisibleContentPosition: true,
	})
}
```

A list of items of different heights (a Markdown preview of thousands of blocks) has another
option that Flutter does not have: `EstimateItemOffsets`. Flutter finds the first item at a
scroll offset by walking from the items it has, building and laying out every item in between, so
a click on the scroll bar or a `JumpTo` far away costs the whole list in one frame. With the
option on, a jump of more than a viewport and the cache extent builds only the items near the new
offset: the list divides the offset by the average height of the items it has measured and starts
there. The items shown are laid out exactly; only the offsets far from them are estimates, and
the list corrects them when the user scrolls back to the start. It applies to `ListView` without
`ItemExtent`, `ItemExtentBuilder` and `PrototypeItem` (those find an item at once), and to
`SliverList`:

```go
func preview(blocks []string, ctl m.ScrollController) m.Widget {
	return m.NewListViewBuilder(func(bc m.BuildContext, i int) m.Widget {
		return m.NewText(blocks[i])
	}, m.ListViewOpts{
		Controller:          ctl,
		ItemCount:           m.Some(len(blocks)),
		EstimateItemOffsets: true,
	})
}
```

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
fonts to `gluttertest.Opts.Fonts`. When neither the style nor the system has a font (a widget
test has no system fonts, so `monospace` is nothing there), the bundled Roboto draws the text,
as Flutter's default font does; before 1.16.4 such text was laid out empty and invisible.

Flutter gives a tab no stops: it is a white space. A code editor wants it to advance to the next
tab stop, so `TextPainterOpts.TabSize` (and `ui.ParagraphStyleOpts.TabSize` under it) is the
distance between stops as a count of the width of a space in the text's font, as CSS's `tab-size`
is: `TabSize: 4` puts the stops 4 spaces apart, measured from the start of each line (the left edge,
or the right edge of a right-to-left paragraph), and a line that wraps starts its stops again. The
default, 0, is Flutter's behavior. A tab at the end of a line hangs like other white space, the
boxes, positions and carets after a tab are at the stop, and nothing is drawn for the tab itself.
`Text` and `RichText` do not have the option yet; use a `TextPainter`.

Flutter's `runApp` has no way to end the app from outside the widget tree: the embedder ends the
process. `RunAppContext` takes a `context.Context` first, like any Go function that waits, and
returns nil when the context ends, after it closed the windows (the observers and
`OnExitRequested` are not asked, as for `ExitApplication`). Glutter installs no signal handlers;
`signal.NotifyContext` makes the context of SIGINT and SIGTERM:

```go
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	m "github.com/rm4n0s/glutter/material"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := m.RunAppContext(ctx, m.NewText("hello")); err != nil {
		os.Exit(1)
	}
}
```

## 12. A widget that fails

When a widget's `build` throws, Flutter shows `ErrorWidget` in its place. The builder is one for the whole app:

```dart
ErrorWidget.builder = (FlutterErrorDetails details) =>
    const Center(child: Text('Something went wrong'));
```

Glutter does the same when a `Build` panics: the panic is recovered at the element that failed, `ErrorWidget` takes
its place, and the error goes to the app's error reporter. It also has what Flutter does not: a boundary that takes
the failures of its whole subtree. This is for an app that shows widgets someone else wrote, such as plugins:

```go
func pluginView(name string, view m.Widget, report func(err error)) m.Widget {
	return m.NewErrorBoundary(func(bc m.BuildContext, err error) m.Widget {
		return m.NewCenter(m.NewText(name + " failed"))
	}, view, m.ErrorBoundaryOpts{OnError: report})
}
```

When anything below the boundary fails to build, the fallback takes the place of the boundary's whole child, in
the same frame. That covers a `Build`, a state's `InitState`, `DidUpdateWidget` or `DidChangeDependencies`, a
`LayoutBuilder`, or an item of a `ListView.builder`. The failed widgets are never painted, and their states are
disposed.
- **Who is told.** `OnError` is called first, then the app's error reporter.
- **The fallback.** It gets the first error; a `nil` fallback shows `ErrorWidget`.
- **Trying again.** A boundary that failed stays failed. Give it a new `Key` to try the child again.
- **Nesting.** Boundaries nest, and the nearest one takes the error. A fallback that fails goes to the next
  boundary up.
- **Inside `OnError`.** It runs inside the frame: don't call `SetState` on an ancestor there. Hand the work to the
  binding's `Post`.

## 13. SVG images

Flutter's framework draws no SVG: apps add the package `flutter_svg`.

```dart
SvgPicture.asset(
  'assets/logo.svg',
  width: 48,
  semanticsLabel: 'Logo',
  colorFilter: const ColorFilter.mode(Colors.teal, BlendMode.srcIn),
)
```

Glutter has the same widget in its package `svg` (`github.com/rm4n0s/glutter/svg`), which `material` does not
re-export:

```go
func logo() m.Widget {
	return svg.NewSvgPictureAsset("assets/logo.svg", svg.SvgPictureAssetOpts{
		SvgPictureOpts: svg.SvgPictureOpts{
			Width:          m.Some(48.0),
			SemanticsLabel: m.Some("Logo"),
			ColorFilter:    m.NewColorFilterMode(m.Colors.Teal.Primary, m.BlendModeSrcIn),
		},
	})
}

func badge() m.Widget {
	return svg.NewSvgPictureString(`<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="currentColor"/></svg>`,
		svg.SvgPictureOpts{Width: m.Some(24.0), Theme: m.Some(svg.SvgTheme{CurrentColor: ui.NewColor(0xFFE53935)})})
}
```

- **The constructors:** `.asset`, `.string`, `.memory`, `.file` and `.network` are `NewSvgPictureAsset`, `NewSvgPictureString`, `NewSvgPictureMemory`, `NewSvgPictureFile` and `NewSvgPictureNetwork`.
- **Options:** the asset's `bundle` and `package` are in `SvgPictureAssetOpts`, and the network's `headers` in `SvgPictureNetworkOpts`.
- **`currentColor`:** `SvgTheme` gives it, either per widget (`Theme`) or for a subtree (`NewDefaultSvgTheme`).
- **Loading:** assets, files and URLs load on a goroutine. A document is parsed once and cached for the app (`CacheOf(bc)`, or your own `NewSvgCacheScope`).
- **What is drawn:** shapes, gradients, patterns, clip paths, masks, text, markers and the common filters. What is not drawn is listed in [SPEC §26](SPEC.md#26-svg-images-19).
- **Without a widget:** to draw an SVG in your own painter, `svg.Parse` gives a `Document`, whose `Picture()` is a `ui.Picture`.

## 14. Drawing a view off screen

Flutter can draw a widget into an image (`RenderRepaintBoundary.toImage`) and can show a second view in a window the
embedder made, but it has no way to run a tree that nobody sees and read its pixels as they change. Glutter's package
`offscreen` (`github.com/rm4n0s/glutter/offscreen`, which `material` does not re-export) does:

```go
// startView runs root in a view of 1280x800 pixels. Call it on the UI thread, where parent
// is the binding of the app (WidgetsBindingOf(bc)).
func startView(parent *m.WidgetsBinding, root m.Widget, send func(pixels []byte, width int)) (*offscreen.View, error) {
	return offscreen.NewView(parent, m.Size{Width: 1280, Height: 800}, root, func(f offscreen.Frame) {
		// This runs on the goroutine of the view. f.Image is drawn into again as soon as it
		// returns, so copy what you send; f.Damage says which rectangles changed.
		send(append([]byte(nil), f.Image.Pix...), f.Image.Rect.Dx())
	}, offscreen.ViewOpts{DevicePixelRatio: 1, MaxFrameRate: 30})
}

// tap sends a touch to the view, as an engine does. Call it on the UI thread.
func tap(v *offscreen.View, x, y float64) {
	d := v.Binding().PlatformDispatcher()
	p := ui.PointerData{Kind: ui.PointerDeviceKindTouch, PhysicalX: x, PhysicalY: y}
	down, up := p, p
	down.Change, up.Change = ui.PointerChangeDown, ui.PointerChangeUp
	d.DispatchPointerDataPacket(ui.PointerDataPacket{Data: []ui.PointerData{down}})
	d.DispatchPointerDataPacket(ui.PointerDataPacket{Data: []ui.PointerData{up}})
}
```

- **A binding of its own.** The tree is built, laid out and painted on the app's UI thread, but the view has its own
  `WidgetsBinding`, so its frame rate is its own (at most `MaxFrameRate`, 30 by default) and does not depend on the
  app's windows. It shares the app's text engine (fonts you loaded work), error reporter and assets. Reach its binding
  with `View.Binding()`.
- **The drawing is on another goroutine.** A frame's scene is copied on the UI thread (`NewDetacher`) and drawn into a
  `NewSurface`, which redraws only what changed. A newer frame replaces one that waits, so a slow callback makes the view
  drop frames and never queue them. A frame that changed no pixel calls nothing.
- **Size and end.** `SetSize` changes the size in logical pixels. `Close` unmounts the tree (every `State.Dispose`
  runs), waits for the goroutine, and may be called twice; do not call it from the callback.
- **Without a view.** To turn a picture into an image once, give `offscreen.NewRasterizer()` to
  `ToImageOpts{Rasterizer: ...}`. It draws with the software renderer, with no window and no GPU, on any goroutine.
- **Not in the view.** It has no window, no accessibility tree and no input method; the clipboard is the app's.

## 15. Glyphs in an image, and the metrics of a font

A terminal draws a grid of characters, and the fastest way to do it is Flutter's `Canvas.drawAtlas`: one image with every
character, and one draw call that places thousands of its parts and tints each one. Flutter has no way to get that image;
you lay out a paragraph for each character and read a picture back. Glutter has two calls for it.
`PlatformDispatcher.FontMetrics(style)` measures the font of a style without a paragraph: the ascent, descent, leading,
line height, the advance of "0" (the width of a cell) and whether the font is monospaced. `offscreen.NewGlyphSheet` draws
grapheme clusters into one image, with the same fonts, fallback and outlines as a paragraph, and says where each one is:

```go
// newSheet measures the font and draws the characters the grid shows, for the device pixel ratio
// of the widget's screen. Call it on the UI thread.
func newSheet(bc m.BuildContext, characters []string) (*offscreen.GlyphSheet, ui.FontMetrics, error) {
	d := m.WidgetsBindingOf(bc).PlatformDispatcher()
	ratio := m.MediaQueryDevicePixelRatioOf(bc)
	style := ui.NewTextStyle(ui.TextStyleOpts{FontFamily: "monospace", FontSize: ui.Ptr(14.0)})
	metrics, err := d.FontMetrics(style)
	if err != nil {
		return nil, ui.FontMetrics{}, err
	}
	sheet, err := offscreen.NewGlyphSheet(d.TextEngine(), style, ratio, characters)
	return sheet, metrics, err
}

// drawGrid draws the rows of the grid, in logical pixels, with one DrawAtlas call. Dispose the
// sheet's Image when the grid is done with it.
func drawGrid(c *ui.Canvas, sheet *offscreen.GlyphSheet, metrics ui.FontMetrics, rows [][]string, color ui.Color) {
	ratio := sheet.DevicePixelRatio
	byCluster := make(map[string]offscreen.Glyph, len(sheet.Glyphs))
	for _, g := range sheet.Glyphs {
		byCluster[g.Cluster] = g
	}
	var transforms []ui.RSTransform
	var rects []ui.Rect
	var colors []ui.Color
	for row, cells := range rows {
		// The pen of a cell is on the baseline, on a whole pixel, so that the glyphs stay sharp.
		baseline := float64(int((float64(row)*metrics.Height+metrics.Ascent)*ratio+0.5)) / ratio
		for col, cluster := range cells {
			g := byCluster[cluster]
			if g.Rect.IsEmpty() {
				continue // a space draws nothing
			}
			x := float64(int(float64(col)*metrics.ZeroAdvance*ratio+0.5)) / ratio
			transforms = append(transforms, ui.RSTransform{
				Scos: float32(1 / ratio), // the sheet is in pixels, the canvas in logical pixels
				Tx:   float32(x + g.Offset.Dx/ratio),
				Ty:   float32(baseline + g.Offset.Dy/ratio),
			})
			rects = append(rects, g.Rect)
			colors = append(colors, color)
		}
	}
	c.DrawAtlas(sheet.Image, transforms, rects, colors, ui.BlendModeModulate, nil, nil)
}
```

- **The image.** Premultiplied RGBA: an outline glyph is white with its coverage as the alpha, so `BlendModeModulate` with a
  color paints the glyph in that color. A color emoji keeps its colors (`Glyph.Colored`). The pixels are the ones a
  paragraph draws with its origin on a whole pixel (same paths, same quarter-pixel positions, same coverage).
- **The place of a cluster.** `Rect` is its part of the image, `Offset` the vector from the pen (the origin on the baseline)
  to the top left corner of `Rect`, and `Advance` how far the pen moves; all in pixels of the image. A space has an empty
  `Rect`. A cluster given twice is drawn once.
- **The metrics** are logical pixels at the style's size and equal what a paragraph of one line in the style measures:
  `Ascent` and `Descent` include half of `Leading` each, `Height` is a line, and `ZeroAdvance` is the cell width of a
  monospaced font. A cell grid wants whole device pixels, so round `ZeroAdvance*ratio` and `Height*ratio` yourself, as above.
- **Calls and errors.** Call both on the UI thread. `NewGlyphSheet` fails with `GlyphClustersMissing` for an empty list and
  `GlyphPixelRatioInvalid` for a ratio that is not positive, `FontMetrics` with `FontMetricsUnsupported` when the text
  engine cannot measure (SPEC §9.9).

## 16. Shaders for the GPU

An app that draws with the GPU itself (package `gpu`) writes its shaders in WGSL, the language of WebGPU, and compiles
them when it changes them, not when it runs. Put the files in a directory and generate:

```text
//go:generate go run github.com/rm4n0s/glutter/cmd/glutter-gen gpushaders ./shaders
```

For each `shaders/<name>.wgsl` the command writes `shaders/<name>.spv` and adds a variable to `shaders/shaders_gen.go`,
named after the file (`blur_pass.wgsl` is `shaders.BlurPass`). It is a `*gpu.Shader` that holds the WGSL (the web takes
it as it is), the SPIR-V (Vulkan, through `embed`), the HLSL for shader model 5.1 (Direct3D 12) and the reflection.
Check the generated files in: building the app needs only Go. The generator needs `naga` 30.0.1
(`cargo install naga-cli --version 30.0.1 --locked`); without it the command fails with `GPUShadersNagaMissing` and
prints that line. A shader that naga refuses is `GPUShadersCompileFailed`, with naga's messages.

- **A shader can include another.** A line `//#include "common.wgsl"` is replaced by the text of the file, which is
  how shaders share a struct or a function.
- **What a binding can be.** `var<uniform>`, `var<storage, read>` and `var<storage, read_write>`; `texture_2d`,
  `texture_2d_array`, `texture_cube` and `texture_3d` of `f32` or `u32`; `texture_storage_2d`, `_2d_array` and `_3d` of
  `rgba8unorm`, `rgba16float`, `r32float` and `r32uint` that are written; `sampler`. Anything else (a depth texture, a
  multisampled texture, a comparison sampler, a texture of `i32`) is `GPUShadersBindingUnsupported`, with the name of the
  variable. Entry points may be `@vertex`, `@fragment` and `@compute` (with a `@workgroup_size` of literals).
- **The reflection** says what the pipeline needs to know, so that you do not write it twice:

```go
package main

import "github.com/rm4n0s/glutter/gpu"

// uniformSize is the bytes of the uniform buffer at a binding of an entry point, by the layout rules of WGSL.
// For a storage buffer that ends in an array without a length, Size counts one element of the array.
func uniformSize(s *gpu.Shader, entry string, group, binding int) (int, bool) {
	e, ok := s.Entry(entry)
	if !ok {
		return 0, false
	}
	for _, b := range e.Bindings {
		if b.Group == group && b.Binding == binding && b.Kind == gpu.BindingUniformBuffer {
			return b.Size, true
		}
	}
	return 0, false
}

func main() {}
```

  Each `gpu.EntryPoint` has its stage, its bindings (group, binding, kind, dimension, sample type, storage format, size),
  the `@location` inputs of a vertex entry point with their types, and the workgroup size of a compute one.
- **The first instance.** A vertex shader that reads `@builtin(instance_index)` or `@builtin(vertex_index)` says so
  (`ReadsInstanceIndex`, `ReadsVertexIndex`). Direct3D 12 does not count the first instance or the first vertex of a draw
  in them, so such a shader must be drawn with both at 0 (debug builds panic with `GPUFirstInstanceRead` otherwise).
  Pass an offset in a uniform when you need one.
- **Checks.** If `spirv-val` is installed, each module is validated with it (`GPUShadersSPIRVInvalid`). In a checkout of
  Glutter with Wine installed, the HLSL of each entry point is compiled with the real `D3DCompile`
  (`GPUShadersHLSLInvalid`); `-wine=off` skips it and `-wine=on` fails when it cannot run.
- **In tests.** `go run github.com/rm4n0s/glutter/cmd/glutter-gen gpushaders -check ./shaders` writes nothing and fails
  with `GPUShadersOutOfDate` when the checked-in files differ from a new generation, which keeps a forgotten
  `go generate` out of the repository.

## 17. Drawing with the GPU

Package `gpu` gives an app the GPU device of the renderer that draws its windows: buffers, textures, samplers, pipelines
made from the shaders of section 16, and commands. The names are WebGPU's, in Go form (`NewBuffer`, `NewRenderPipeline`,
`CommandEncoder`, `Queue.Submit`). Everything is used on the UI thread; an app prepares bytes on any goroutine and writes
them from the UI thread.

```go
package main

import (
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/widgets"
)

// drawOnce draws one triangle into a texture with the pipeline and returns the texture. The shader is one that
// glutter-gen gpushaders made; the pipeline has no vertex buffer, as the vertex function builds the triangle.
func drawOnce(d *gpu.Device, shader *gpu.Shader) (*gpu.Texture, error) {
	module, err := d.NewShaderModule(shader)
	if err != nil {
		return nil, err
	}
	defer module.Release()
	pipeline, err := d.NewRenderPipeline(gpu.RenderPipelineDesc{
		Vertex:   gpu.VertexState{Module: module, EntryPoint: "vs_main"},
		Fragment: gpu.FragmentState{Module: module, EntryPoint: "fs_main", Targets: []gpu.ColorTargetState{{Format: gpu.TextureFormatRGBA8Unorm}}},
	})
	if err != nil {
		return nil, err
	}
	defer pipeline.Release()
	target, err := d.NewTexture(gpu.TextureDesc{
		Size:   gpu.Extent3D{Width: 256, Height: 256},
		Format: gpu.TextureFormatRGBA8Unorm,
		Usage:  gpu.TextureUsageRenderAttachment | gpu.TextureUsageTextureBinding,
	})
	if err != nil {
		return nil, err
	}
	encoder := d.NewCommandEncoder()
	pass := encoder.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{
		View: target.View(), ClearValue: gpu.Color{A: 1},
	}}})
	pass.SetPipeline(pipeline)
	pass.Draw(3, 1, 0, 0)
	pass.End()
	if err := d.Queue().Submit(encoder.Finish()); err != nil {
		target.Release()
		return nil, err
	}
	return target, nil
}

// onDevice makes the resources of the app each time there is a device: at the first frame, and again after the
// device was lost.
func onDevice(b *widgets.WidgetsBinding, shader *gpu.Shader) {
	var texture *gpu.Texture
	gpu.OnDevice(b, func(d *gpu.Device) {
		var err error
		if texture, err = drawOnce(d, shader); err != nil {
			return // GPUDeviceLost, GPUInvalidDescriptor, ...: the error says which
		}
		d.OnLost(func(error) { texture.Release() }) // forgets the texture: the device that had it is gone
	})
}

func main() {}
```

- **Getting the device.** `gpu.DeviceOf(bc)` returns it, or `GPUUnavailable` when the app runs on the OpenGL ES,
  WebGL or software renderer (an app that needs a GPU says so; one that can do without draws another way), or
  `GPUDeviceNotReady` before the first frame, when the renderer has not opened its device. `gpu.OnDevice` is the
  way that never meets the second: it calls your function from the event loop, with the device, and again with the
  next one after a loss.
- **Limits and features.** `d.Limits()` has the design limits that every backend honours (a texture of 8192, four
  bind groups, 256 bytes between dynamic offsets and so on); `d.Features()` says whether timestamp queries exist.
- **Errors you return, mistakes you panic on.** A description that cannot be made is a returned error with the field
  in its metadata (`GPUInvalidDescriptor`, `GPULimitExceeded`). A mistake while recording (a draw with no pipeline, a
  bind group of another layout, a draw past the end of a buffer, a pass not ended) panics with `GPUEncodingInvalid` in
  debug builds and is skipped by release builds; the recording device of `gluttertest` runs the same checks.
- **Release what you make.** Every buffer, texture, sampler, pipeline and bind group has `Release`; the GPU frees it
  when the work that uses it is done. A debug build logs each resource that is alive when the app ends, with the line
  that made it (`GPUResourceLeaked`).
- **Reading back.** `Buffer.Read(offset, size, done)` calls `done` from the event loop after the GPU finished, never
  inside the call: copy a texture into a buffer first (`CopyTextureToBuffer`, rows a multiple of
  `Limits().BytesPerRowAlignment` apart), then read it. A web page cannot wait for the GPU, so the same code runs there.
- **Device loss.** A driver reset or a lost WebGPU device calls the `OnLost` functions once; the old `Device` and its
  resources then do nothing, and the calls that return an error return `GPUDeviceLost`. Make the resources again in
  the `OnDevice` function.
- **Showing a texture in the widget tree.** `id := d.RegisterTexture(texture)` gives the id (never 0; the texture needs
  `TextureUsageTextureBinding` and the format `RGBA8Unorm`, `BGRA8Unorm` or `RGBA16Float`), and `widgets.NewTexture(id)`
  shows it, scaled to its box with `FilterQuality` (low by default), with no copy: it is clipped, transformed and
  faded like any other widget. After you draw into the texture, call `d.MarkTextureFrameAvailable(id)` to make a frame;
  `Freeze` ignores that call and shows the texture as it is. An `RGBA8Unorm` texture holds sRGB-encoded values and an
  `RGBA16Float` one linear values, which the renderer encodes. `d.UnregisterTexture(id)` (or `Release` of the texture)
  ends it. On the OpenGL ES and software renderers the widget draws nothing, and the log says so once.
- **In tests.** `gluttertest.New(t, gluttertest.Opts{GPU: true})` gives the app a recording device
  (`tester.GPUDevice()` keeps what was encoded: `Calls()`, `Ops()`, `Logs()`; `Lose()` loses the device). Without
  the option the app has no GPU device, as with the software renderer.

### A view of your own

`gpu.NewView(onPaint, opts)` is a widget that owns a render target the size of its box in pixels and calls your function
to draw into it. `examples/gpu-triangle` is a lit cube that turns, drawn this way.

```go
package main

import (
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/widgets"
)

func newView() *gpu.View {
	return gpu.NewView(func(f *gpu.ViewFrame) {
		// f.Color is the target (f.Resolve the texture it is resolved into when SampleCount is 4, f.Depth the depth).
		pass := f.Encoder.BeginRenderPass(gpu.RenderPassDesc{ColorAttachments: []gpu.RenderPassColorAttachment{{
			View: f.Color.View(), ClearValue: gpu.Color{A: 1},
		}}})
		// ... SetPipeline, SetBindGroup, Draw ...
		pass.End() // the view finishes and submits f.Encoder when the function returns
	}, gpu.ViewOpts{
		DepthFormat: gpu.TextureFormatDepth32Float, // none by default
		SampleCount: 4,                             // 1 by default
		Continuous:  true,                          // draw at every frame; without it, draw when created, resized or marked
		OnInit:      func(d *gpu.Device) { /* make buffers, pipelines */ },
		OnDispose:   func() { /* release them */ },
		Fallback:    widgets.NewText("This needs a GPU renderer."),
	})
}

func main() { _ = newView() }
```

- **When it draws.** In the frame, after the widgets are painted and before the frame is shown. A view that is not
  `Continuous` draws once when it appears, again when it is resized (or the device pixel ratio changes) and whenever
  `Controller.MarkNeedsPaint()` asks; between those it costs nothing and schedules no frame.
- **Size.** The targets are the box times the device pixel ratio, rounded: `f.Width` and `f.Height` are pixels,
  `f.DevicePixelRatio` is the ratio. Size the box as any widget (`Expanded`, `SizedBox`, `AspectRatio`); the view
  takes all the room it is given. A resize makes new targets, so do not keep `f.Color` between frames.
- **Resources.** Make yours in `OnInit` and release them in `OnDispose`. `OnInit` runs again with the new device
  after a loss, and `OnDispose` runs for the old one first. The app ending runs it too.
- **Without a GPU.** On the OpenGL ES, WebGL and software renderers the view shows `Fallback`. (On a Linux machine
  without a GPU, `auto` draws the whole app with `soft` from 1.12 (`GLUTTER_RENDERER=soft` asks for it): on the CPU
  into the window's shared memory, redrawing only what changed, which is much cheaper than OpenGL on llvmpipe.
  `GLUTTER_RENDERER=vulkan` on lavapipe still runs a GPU view.)
- **Input.** The view is an ordinary box: wrap it in `Listener` or `GestureDetector`.
- **Tests.** With `gluttertest.Opts{GPU: true}` the view draws on the recording device; check what the function encoded.

## 18. Locking the pointer

A game's mouse look, or a model that you orbit without the pointer stopping at the edge of the screen, needs the pointer
locked: the cursor is hidden, the pointer stays where it was, and the mouse's pointer events carry the motion in `Delta`.
Flutter has no such call; `PointerLockOf(bc)` is Glutter's.

```go
package main

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	m "github.com/rm4n0s/glutter/material"
)

type lookState struct {
	m.StateOf[*look]
	yaw, pitch float64
	handle     foundation.ListenerHandle
}

type look struct{ m.StatefulWidget }

func newLook() *look { return &look{StatefulWidget: m.NewStatefulWidgetBase()} }

func (w *look) CreateState() m.State { return &lookState{} }

func (s *lookState) InitState() {
	// The lock can end without the app asking: Alt+Tab, the compositor, Escape on the web.
	s.handle = m.PointerLockOf(s.Context()).AddListener(func() { s.SetState(func() {}) })
}

func (s *lookState) Dispose() {
	m.PointerLockOf(s.Context()).RemoveListener(s.handle)
}

func (s *lookState) Build(bc m.BuildContext) m.Widget {
	return m.NewListener(m.NewGestureDetector(m.NewSizedBox(m.NewText("click to look around")), m.GestureDetectorOpts{
		OnTap: func() {
			// Lock from a click or a key: the web refuses it otherwise (PointerLockDenied).
			if err := m.PointerLockOf(bc).Lock(m.ViewOf(bc)); err != nil {
				if appErr, ok := errors.FromError(err); ok && appErr.Tag == m.TagPointerLockUnavailable {
					return // this platform cannot lock the pointer: use the drag instead
				}
			}
		},
	}), m.ListenerOpts{
		OnPointerMove: func(e *gestures.PointerMoveEvent) {
			if m.PointerLockOf(bc).Locked() {
				s.SetState(func() { s.yaw += e.Delta().Dx * 0.005; s.pitch += e.Delta().Dy * 0.005 })
			}
		},
		OnPointerHover: func(e *gestures.PointerHoverEvent) {
			if m.PointerLockOf(bc).Locked() {
				s.SetState(func() { s.yaw += e.Delta().Dx * 0.005; s.pitch += e.Delta().Dy * 0.005 })
			}
		},
	})
}

func main() { _ = newLook() }
```

- **Where it works.** Wayland (a compositor with `pointer-constraints` and `relative-pointer`), X11, Windows, the web
  and Android 8 or later. A platform that cannot gives `PointerLockUnavailable` (`TagPointerLockUnavailable`); one that
  refuses gives `PointerLockDenied`: the web outside a click or a key, a window that is not the active one.
- **While locked.** The cursor is hidden. The mouse's pointer events keep the `Position` they had when the lock began,
  are hovers (moves while a button is down) and carry the motion in `Delta`, unaccelerated where the platform gives
  it (physical pixels). Buttons still make down and up events there.
- **When it ends.** `Unlock()`, the window losing the focus, the compositor or the system taking it back, closing the
  window, Escape on the web. A lock that ended by itself calls the listeners; `Unlock()` does not. It does not come
  back by itself: lock again from the next click or key.
- **Tests.** `gluttertest` has `FailPointerLock(err)` (the next `Lock` fails), `EndPointerLock(view)` (the platform ends the
  lock), `MoveLockedMouse(delta)` and `PointerLocked(view)` for the fake platform.

## 19. 3D math and a camera

Package `vecmath` is the 3D math: vectors, quaternions, matrices, rays, planes, boxes, spheres and frustums in `float64`,
as plain values that methods return (nothing allocates). Package `viewport3d` has the `Camera` of a CAD view and
`Navigation`, the widget that moves it with the mouse, the touch screen, a trackpad and the keyboard. The camera orbits a
target at a distance, and gives the GPU `float32` matrices taken relative to the eye, so that a scene 50 km wide with a
detail of 0.1 mm draws without jitter. `viewport3d.NewView(scene, camera)` is the widget that draws a `Scene` as the
camera sees it, inside a `Navigation`; the camera and the navigation also work with any other widget, such as a
`gpu.View` of your own.

```go
package main

import (
	"log"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

func main() {
	camera := viewport3d.NewCamera(viewport3d.CameraOpts{Distance: 8})
	// What the camera shows: the near plane and "fit" are made from it.
	scene := vecmath.NewBox3(vecmath.Vec3{X: -1, Y: -1, Z: -1}, vecmath.Vec3{X: 1, Y: 1, Z: 1})
	camera.SetBounds(scene)
	camera.AddListener(func() {
		// The picture changes: draw again. A model matrix is in float64 and goes to the GPU relative to the eye.
		world := vecmath.NewMat4Translation(vecmath.Vec3{X: 50000})
		_ = camera.ModelViewProjection(world) // [16]float32, for a uniform buffer
	})

	view := viewport3d.NewNavigation(camera, m.NewSizedBoxExpand(m.NewText("drag to orbit")), viewport3d.NavigationOpts{
		LockPointer: true, // an orbit with the mouse hides the cursor, and does not stop at the screen's edge
	})
	err := m.RunApp(m.NewMaterialApp(m.MaterialAppOpts{Home: m.NewScaffold(m.ScaffoldOpts{Body: view})}))
	if err != nil {
		log.Fatal(err)
	}
}
```

A scene is a flat list of items that you own and change: each item has a mesh, a material, a transform in `float64`, and
the view draws again whenever the scene or the camera changes (an idle view costs nothing).

```go
package main

import (
	"log"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// A square pyramid: five vertices and no normals, so it is drawn flat.
func pyramid() *viewport3d.Mesh {
	mesh, err := viewport3d.NewMesh(viewport3d.MeshData{
		Positions: []float32{-1, 0, -1, 1, 0, -1, 1, 0, 1, -1, 0, 1, 0, 1.5, 0},
		Indices16: []uint16{0, 1, 4, 1, 2, 4, 2, 3, 4, 3, 0, 4, 0, 3, 2, 0, 2, 1},
	})
	if err != nil {
		log.Fatal(err) // viewport3d.TagMeshInvalid: the data does not make a mesh
	}
	return mesh
}

func main() {
	scene := viewport3d.NewScene()
	scene.SetStyle(viewport3d.SceneStyle{DisplayMode: viewport3d.DisplayModeShadedWithEdges})
	mesh := pyramid()
	steel := viewport3d.NewPhongMaterial(viewport3d.PhongOpts{Color: ui.NewColor(0xFFB8C4D0)})
	for i := 0; i < 5; i++ {
		// Items that share a mesh and a material are drawn by one instanced call.
		at := vecmath.NewMat4Translation(vecmath.Vec3{X: float64(i) * 3})
		scene.Add(viewport3d.NewItem(mesh, steel, viewport3d.ItemOpts{Transform: m.Some(at)}))
	}
	camera := viewport3d.NewCamera(viewport3d.CameraOpts{Up: vecmath.Vec3Y})
	camera.SetStandardView(viewport3d.StandardViewIsometric)
	camera.FitTo(scene.Bounds())

	view := viewport3d.NewView(scene, camera) // 4x multisampling; the mouse and the touch move the camera
	err := m.RunApp(m.NewMaterialApp(m.MaterialAppOpts{Home: m.NewScaffold(m.ScaffoldOpts{Body: view})}))
	if err != nil {
		log.Fatal(err)
	}
}
```

- **Meshes.** `NewMesh(MeshData)` checks the data (`MeshInvalid`) and keeps it; the view uploads it once and `Release`
  frees it. Positions are `float32` offsets from `MeshData.Origin`, a `float64` point: a part that sits 50 km from the
  world's origin keeps its offsets small, and its items still draw without jitter. A mesh is triangles, lines
  (`PrimitiveLines`, pairs of indices) or points (`PrimitivePoints`); normals, texture coordinates and colors are
  optional, and a mesh of triangles without normals is drawn flat.
- **Materials.** `NewUnlitMaterial` and `NewPhongMaterial` for triangles (a color, a texture, the vertex colors, double
  sided, transparent), `NewLineMaterial` (a width in logical pixels, a dash pattern, anti-aliased) and `NewPointMaterial`
  (round or square, a size in logical pixels). A color with alpha, or `Transparent`, makes an item transparent: such items
  are drawn after the opaque ones, back to front. A change of a material, of an item or of the scene's style reaches the
  views at once.
- **Display modes and edges.** `SceneStyle.DisplayMode` is shaded, shaded with edges, wireframe or hidden line. The edges
  are `FeatureEdges(mesh, angle)`: the boundary edges and those where two triangles meet at more than the angle (30° by
  default); vertices at the same place count as one, so a mesh whose faces do not share vertices has its edges where they
  meet. A smooth sphere has none. The light is at the camera, with up to four more directional lights and an ambient term
  (`SceneStyle`).
- **Your own view.** `viewport3d.NewRenderer(scene, camera)` is what the view uses: call `Init(device)` and `Dispose()`
  from the `OnInit` and `OnDispose` of your own `gpu.View` (with a `Depth32Float` depth target) and `Draw(frame)` from
  its paint function, to draw a scene and your own shaders in one view. `Renderer.Stats()` says what the last frame drew.

**Section planes, picking and selection.** A scene has up to six section planes (`Scene.SetSectionPlanes`), in world
coordinates in `float64`; each cuts away what its normal points to, in the faces, the lines and the points. Where a closed
mesh is cut, a cap closes it and an outline marks the cut (`SceneStyle.Section`: no cap, the cap's color, the outline's
color and width). A tap asks what is under the pointer with `ViewController.Pick`; the answer comes in the frame after the
pick's own, with the item, the triangle, the world point in `float64` and the normal. `Scene.Select` highlights items with
a color and an outline (`SceneStyle.Selection`).

```go
package main

import (
	"log"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// A cube of 8 shared vertices: closed, so a plane that cuts it gets a cap.
func cube() *viewport3d.Mesh {
	mesh, err := viewport3d.NewMesh(viewport3d.MeshData{
		Positions: []float32{-1, -1, -1, 1, -1, -1, 1, 1, -1, -1, 1, -1, -1, -1, 1, 1, -1, 1, 1, 1, 1, -1, 1, 1},
		Indices16: []uint16{4, 5, 6, 4, 6, 7, 1, 0, 3, 1, 3, 2, 5, 1, 2, 5, 2, 6, 0, 4, 7, 0, 7, 3, 7, 6, 2, 7, 2, 3, 0, 1, 5, 0, 5, 4},
	})
	if err != nil {
		log.Fatal(err)
	}
	return mesh
}

func main() {
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(cube(), viewport3d.NewPhongMaterial(), viewport3d.ItemOpts{PickID: 1}))
	// Cut away what is at the right of x = 0.2.
	scene.SetSectionPlanes(vecmath.NewPlaneFromPointNormal(vecmath.Vec3{X: 0.2}, vecmath.Vec3X))
	camera := viewport3d.NewCamera(viewport3d.CameraOpts{Up: vecmath.Vec3Y})
	camera.SetStandardView(viewport3d.StandardViewIsometric)
	camera.FitTo(scene.Bounds())

	ctl := viewport3d.NewViewController()
	pick := func(at ui.Offset) {
		ctl.Pick(at, func(r viewport3d.PickResult, err error) {
			switch {
			case err != nil: // viewport3d.TagPickUnavailable, or a failed read of the GPU
				log.Println(err)
			case r.Hit:
				scene.Select(r.Item)
				log.Println("picked", r.PickID, "triangle", r.Index, "at", r.Point, "normal", r.Normal)
			default:
				scene.ClearSelection()
			}
		})
	}
	view := m.NewGestureDetector(viewport3d.NewView(scene, camera, viewport3d.ViewOpts{Controller: ctl}),
		m.GestureDetectorOpts{OnTapUp: func(d m.TapUpDetails) { pick(d.LocalPosition) }})
	err := m.RunApp(m.NewMaterialApp(m.MaterialAppOpts{Home: m.NewScaffold(m.ScaffoldOpts{Body: view})}))
	if err != nil {
		log.Fatal(err)
	}
}
```

- **Caps.** The cap is drawn with the stencil, so the view's depth target has one (`NewView` does that while a plane cuts
  the scene). A mesh is closed when each of its edges belongs to two triangles; `Mesh.Closed()` says. A transparent item has
  no cap, and neither has a scene in wireframe or hidden line mode.
- **Picking.** `Pick(at, done, PickOpts{Radius})` finds the nearest primitive within `Radius` pixels (2 by default), so a line
  of one pixel can be hit; `PickRect(rect, done)` gives the items in a box. A pick is drawn only when asked, into a target of
  the size of the rectangle. `Scene.Raycast(ray)` answers at once on the CPU, for a question now and then.
- **Selection.** `Item.SetSelected`, `Scene.Select`, `ClearSelection` and `Selection`. The outline is not hidden by what is in
  front of the item.

**Overlays.** A scene draws an infinite grid (`SceneStyle.Grid`), and a view can show the axes of the camera, a view cube
that goes to the standard views, and widgets that follow points of the world, over the picture.

```go
package main

import (
	"log"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

func main() {
	scene := viewport3d.NewScene()
	// A grid on the plane y = 0: a line every half unit, a stronger one every fifth.
	scene.SetStyle(viewport3d.SceneStyle{Grid: viewport3d.GridStyle{Show: true, Spacing: 0.5, MajorEvery: 5}})
	camera := viewport3d.NewCamera(viewport3d.CameraOpts{Up: vecmath.Vec3Y})
	camera.SetStandardView(viewport3d.StandardViewIsometric)

	view := viewport3d.NewView(scene, camera, viewport3d.ViewOpts{
		Axes:     &viewport3d.AxesOpts{},     // the bottom left
		ViewCube: &viewport3d.ViewCubeOpts{}, // the top right: tap a face, an edge or a corner
		Children: []m.Widget{
			// A label that follows a point of the world, centered above it. It is hidden while the point is behind the camera, outside
			// the view, or, with HideWhenCovered, behind another item.
			viewport3d.NewAnchored(vecmath.Vec3{Y: 1}, m.NewText("Origin"),
				viewport3d.AnchoredOpts{Alignment: m.AlignmentBottomCenter, HideWhenCovered: true}),
		},
	})
	err := m.RunApp(m.NewMaterialApp(m.MaterialAppOpts{Home: m.NewScaffold(m.ScaffoldOpts{Body: view})}))
	if err != nil {
		log.Fatal(err)
	}
}
```

- **The grid.** It is on the plane through `Origin` that is perpendicular to `Normal` (+y by default; use `Vec3Z` in a world with
  +z up). The lines are as wide at the horizon as they are near, fade with the distance (`FadeDistance`) and disappear where
  they would be closer than a few pixels. Far from the origin it is as sharp as at the origin. It is drawn under transparent
  items and over opaque ones, and is not picked.
- **The view cube.** A face goes to the view that looks at it, an edge to the view between two faces and a corner to the view
  between three, with the same animation as `AnimateTo` (`Duration` changes it); the target and the distance stay.
  `Camera.ViewDirectionPose(dir)` is the pose of a view from any direction. The cube and the axes are widgets: `NewViewCube(camera)`
  and `NewAxes(camera)` go anywhere, with or without a view.
- **Anchored widgets.** The child of an `Anchored` has its natural size and is put on `Camera.Project(point)`; a widget in
  `Children` that is not an `Anchored` fills the view. They get the pointer before the camera's navigation, and a hidden one is
  neither painted, nor tapped, nor in the semantics tree. `View.Project(p)` gives the same point for a `CustomPainter` that draws
  dimension lines over the view.

- **Models.** `stl.Load(ctx, fsys, name)` and `obj.Load(ctx, fsys, name)` (and `Decode` for a reader) read a file into a
  `model.Model`: meshes, materials, images and a tree of nodes, as plain Go with no GPU. `viewport3d.Upload` makes the
  meshes, the materials, the textures and the items of it on a device; add the items to a scene and release the result when
  the scene no longer shows them.
  `gltf.Load(ctx, fsys, name)` reads `.gltf` (its buffers and images from the same directory, or `data:` URIs) and `.glb`
  files, with materials of the metallic-roughness model and their textures, skins, morph targets, animations (as data),
  cameras, lights and scenes (`gltf.TagGLTFInvalid`, with the JSON pointer of the failure in `path` and a `reason`;
  `gltf.TagGLTFExtensionUnsupported` for a required extension that is not read, such as Draco or KTX2). A model that came from
  glTF has `Material.PBR` set; `Upload` makes a `viewport3d.PBRMaterial` of it (see "PBR, lighting and animation" below).
  `ply.Load(ctx, fsys, name)` reads PLY files (ASCII, and binary in both byte orders; positions, normals, colors, texture
  coordinates and polygons, which it cuts into triangles). A PLY file with no faces is a point cloud: its mesh is of points,
  and when it has colors `Upload` makes a `viewport3d.PointMaterial` with `VertexColors`, so the cloud shows its own colors
  (set `PointOpts.Size` for the size of a point). `threemf.Load(ctx, fsys, name)` reads 3MF files: objects, components with
  their transforms, build items, the base materials and color groups of the materials extension, and the unit of the file in
  `Model.Units` (positions are not scaled). Textures, beam lattices, slices and the production extension are not read; a
  file that requires one fails with `threemf.TagThreeMFInvalid` (`ply.TagPLYInvalid` is the error of a PLY file), whose
  `reason` says what is wrong. `Options.MaxModelBytes` limits what a 3MF file may unpack to.

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/model/obj"
	"github.com/rm4n0s/glutter/viewport3d"
)

// show reads models/name and puts it in the scene. Call it on the UI thread with the device of the view (gpu.OnDevice).
func show(ctx context.Context, device *gpu.Device, scene *viewport3d.Scene, camera *viewport3d.Camera, name string) (*viewport3d.Uploaded, error) {
	// The material library and the images of the file are read from the same directory. Float64 keeps the digits of a model
	// that is far from its origin; without it the positions are float32.
	m, err := obj.Load(ctx, os.DirFS("models"), name, obj.Options{Float64: true})
	if err != nil {
		return nil, err // obj.TagOBJInvalid (with "line" and "reason"), or model.TagModelReadFailed
	}
	up, err := viewport3d.Upload(ctx, device, m, viewport3d.UploadOpts{FirstPickID: 1}) // pick ids 1, 2, ... for the parts
	if err != nil {
		return nil, err
	}
	scene.Add(up.Items()...)
	camera.FitTo(up.Bounds)
	return up, nil // up.Parts say which node and mesh of the model an item is made of
}

func main() {
	if _, err := os.Stat("models"); err != nil {
		log.Println("no models directory:", err)
	}
	_ = show
}
```

- **STL.** Binary and ASCII are told apart by the size, not the header. Vertices that are equal are welded, and a normal is
  smooth across an edge unless its two faces differ by more than `CreaseAngle` (30° by default); `Flat: true` keeps every
  triangle's own normal and `WeldTolerance` welds vertices that are close. Binary files may give a color for each triangle (the
  VisCAM and SolidView convention); the mesh then has vertex colors, and `Upload` makes a material that uses them.
- **OBJ.** Polygons are cut into triangles, `o` and `g` make the nodes, `usemtl` splits a node into a mesh for each material,
  and `l` and `p` make meshes of lines and points. The `mtllib` file is read through the same `fs.FS`: `Kd`, `Ks`, `Ns`, `d`
  and `map_Kd` are used, and a library or an image that is not there leaves the default material and one line in the `slog`
  log. The texture coordinate `v` is turned, as the model's origin is at the top left.

- **PBR, lighting and animation.** `viewport3d.NewPBRMaterial(viewport3d.PBROpts{...})` is glTF's metallic-roughness
  material: `Color`, `Metallic`, `Roughness`, `Emissive`, the five textures (`BaseColorTexture`, `MetallicRoughnessTexture`,
  `NormalTexture`, `OcclusionTexture`, `EmissiveTexture`, each a `PBRTexture` with an optional sampler, texture coordinate set
  and `TextureTransform`), `AlphaMode`, `DoubleSided`, `VertexColors` and `Unlit`. A scene with a PBR item is drawn into a float
  target and tone mapped over the view: `SceneStyle.ToneMapping` (`ToneMappingNeutral` by default, or `ToneMappingACES`) and
  `Exposure`. The light is the style's directional lights, the headlight and the environment: `SceneStyle.Environment` is
  `viewport3d.NewEnvironment(cube)` of a cube texture of yours (`RGBA16Float` for radiance above 1, or `RGBA8Unorm` for sRGB
  colors; it must live as long as the scene uses it), and without one the view makes a neutral studio; the renderer prefilters it
  once. `SceneStyle.Shadow` (`ShadowStyle{Enabled: true}`) turns on one shadow map for the first directional light (`Light` is
  an index in `Lights`; `Size` 2048 by default), fitted to the scene's bounds so that it does not shimmer when the camera moves.
  `Upload` of a glTF model plays animations through `viewport3d.NewAnimation(model.Animations[0])`: `Sample(t, pose)` writes
  the value of every channel at `t` seconds into a `Pose` that you keep and reuse (looping is yours), and `up.SetPose(pose)`
  moves the nodes, the skins and the morph weights. A mesh that you build yourself is skinned with `MeshData.Joints` and
  `Weights` and `item.SetSkin(viewport3d.NewSkin(inverseBindMatrices))` (then `skin.SetJointWorld(i, m)`), and morphed with
  `MeshData.MorphTargets` and `item.SetMorphWeights(w)`.

```go
package main

import (
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/model"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// play shows what Animation.Sample gives at the time since the start, looping, in a model that Upload made.
func play(up *viewport3d.Uploaded, m *model.Model, pose *viewport3d.Pose, since time.Duration) {
	anim := viewport3d.NewAnimation(m.Animations[0])
	t := float32(since.Seconds())
	if d := anim.Duration(); d > 0 {
		for t >= d {
			t -= d
		}
	}
	anim.Sample(t, pose) // writes into the pose: nothing is allocated when the pose is reused
	up.SetPose(pose)
}

// lit is a scene with a shadow and a PBR sphere of rough gold, tone mapped with ACES.
func lit(sphere *viewport3d.Mesh) *viewport3d.Scene {
	scene := viewport3d.NewScene()
	scene.SetStyle(viewport3d.SceneStyle{
		Lights:      []viewport3d.DirectionalLight{{Direction: vecmath.Vec3{X: -0.6, Y: 1, Z: 0.35}}},
		Shadow:      viewport3d.ShadowStyle{Enabled: true},
		ToneMapping: viewport3d.ToneMappingACES,
	})
	gold := viewport3d.NewPBRMaterial(viewport3d.PBROpts{Metallic: foundation.Some(1.0), Roughness: foundation.Some(0.4)})
	scene.Add(viewport3d.NewItem(sphere, gold))
	return scene
}

func main() { _, _ = play, lit }
```

- **The math.** Matrices are column-major, as WGSL reads them, and `a.Mul(b)` applies `b` first. The clip space is
  WebGPU's, and the projections are reversed-Z (`Perspective(fovY, aspect, near)` has the far plane at infinity): the
  depth is 1 at the near plane and 0 far away, so a pipeline tests `Greater` and clears the depth to 0.
  `Mat4.Float32()` and `Vec3.Float32()` go to the GPU, `Mat4.Matrix4()` to a `Canvas`. `Inverse` says when a matrix is
  singular.
- **The camera.** `Orbit`, `Pan`, `Zoom`, `ZoomAt` (keeps the point under a pixel where it is), `Roll`, `FitTo(box)`,
  `SetStandardView(viewport3d.StandardViewTop)` and `AnimateTo(pose, vsync)` (a move of 300 ms on a ticker provider, such
  as the `TickerProvider` a `State` embeds) change it, and its listeners are told. `Ray(at)`, `Project(p)` and
  `Unproject(at, depth)` go between a point of the view in logical pixels and the world. A perspective and an
  orthographic camera show what is at the target at the same size, so `SetKind` does not make the model jump.
- **The navigation.** The defaults: the left button orbits (around the point under the pointer when `PivotAt` knows it,
  else the target); the middle button, or Shift with the left, pans; the wheel zooms toward the pointer; a double click
  fits the scene; a trackpad's pinch zooms; one finger orbits and two pan, pinch and twist; the arrows orbit and `+` and
  `-` zoom. Every binding is an option of `NavigationOpts` (`LeftDrag: viewport3d.NavigationActionPan`, `Wheel:
  viewport3d.NavigationActionNone`, ...), and so are the speeds. The recognizers are in the gesture arena, so a button or
  a drag of a parent, and a `ListView` around the view, win and lose as for any two recognizers: inside a vertical list a
  finger that moves along the list scrolls it, and one that moves across it orbits the view.
- **Keys.** The keys are `Shortcuts` and `Actions` of the intents `OrbitIntent`, `ZoomIntent`, `PanIntent`, `FitIntent`
  and `StandardViewIntent`; `NavigationOpts.Shortcuts` replaces the keys, and `Actions` replaces what an intent does.
- **Tests.** Navigation is a widget, so `gluttertest` drives it: `StartGesture` with a mouse or touch `Kind`, `ScrollAt`
  for the wheel, `StartTrackpadGesture`, `SendKeyEvent`, and, for the locked pointer, `MoveLockedMouse` and
  `PointerLocked`.

- **A whole viewer.** `examples/cad` puts the chapter together in one file: a part made by code (`meshBuilder`, boxes, a
  cylinder and a tube) or an STL or OBJ opened with the `desktop` file dialog (on the web, the built-in sample), the
  navigation, the view cube and the grid, the four display modes, a section plane that a `Slider` moves, a pick (`ViewController.Pick`)
  whose item and world point go to a side panel, and a distance between two picks that a `CustomPainter` draws over the view
  from `Camera.Project`, with an `Anchored` label at its middle. Read it before you write your own.
- **Speed.** A frame of a scene that did not change allocates nothing in `viewport3d` and `gpu` (the allocation test runs on the
  recording device), and an idle view draws no frame. On a Radeon RX 6800M a part of a million triangles with its edges, 4x
  multisampled at 1920x1080, draws at about 400 frames a second with 0.12 ms of CPU; 10,000 items that share a mesh cost 0.95 ms.
  Items that share a mesh and a material are one draw, so give them the same `*Mesh` and `Material`; 10,000 items that each have
  a mesh of their own cost about 20 ms of CPU, because each is three calls into the driver (`docs/PERFORMANCE.md`, "The 3D view").
  `go test -run '^$' -bench 'Frame|Display' ./viewport3d` measures your machine (`GLUTTER_BENCH_GPU=hardware` with `-tags
  glutter_display` for the GPU).
- **Memory.** A model that is opened and closed 100 times, a view resized 1,000 times and a device that is lost over and over
  leave the heap flat and nothing on the device (`go test -run Leaks ./gpu ./viewport3d`). Call `Uploaded.Release()` when you
  take a model out of the scene: the meshes, the materials and the textures it made are released then, and `Scene.Remove` alone
  does not release them.

## 20. Gamepads

`gamepad.Of(bc)` gives the pads that are connected. Whatever a pad reports, its buttons and axes are the same: `South` is
the bottom face button (A on an Xbox pad, cross on a PlayStation one), and the sticks and triggers are numbers you can
use as they are. The mapping from the pad's own numbers is SDL 3's table of mappings plus the Linux Gamepad
Specification, so common pads need nothing from you. Flutter has no gamepad API; this one is Glutter's.

```go
package main

import (
	"time"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gamepad"
	m "github.com/rm4n0s/glutter/material"
)

type padView struct{ m.StatefulWidget }

func newPadView() *padView { return &padView{StatefulWidget: m.NewStatefulWidgetBase()} }

func (w *padView) CreateState() m.State { return &padViewState{} }

type padViewState struct {
	m.StateOf[*padView]
	pads    *gamepad.Gamepads
	handle  foundation.ListenerHandle
	removes map[*gamepad.Pad]func()
	x, y    float64
}

func (s *padViewState) InitState() {
	s.pads = gamepad.Of(s.Context())
	s.removes = map[*gamepad.Pad]func(){}
	// The set of pads is a ChangeNotifier: listen to it for pads that come and go.
	s.handle = s.pads.AddListener(s.follow)
	s.follow()
}

func (s *padViewState) Dispose() {
	s.pads.RemoveListener(s.handle)
	for _, remove := range s.removes {
		remove()
	}
}

func (s *padViewState) follow() {
	for _, pad := range s.pads.Pads() {
		if _, ok := s.removes[pad]; ok {
			continue
		}
		s.removes[pad] = pad.AddListener(func(e gamepad.Event) {
			// Events arrive on the UI thread, in order, with the time the pad gave them.
			if e.Kind == gamepad.ButtonDown && e.Button == gamepad.South {
				_ = e.Pad.Rumble(0.5, 0.5, 200*time.Millisecond) // GamepadRumbleUnavailable when it cannot
			}
		})
	}
}

// Read the state once a frame for a continuous input like a stick.
func (s *padViewState) Build(bc m.BuildContext) m.Widget {
	for _, pad := range s.pads.Pads() {
		x, y := gamepad.ApplyStickDeadZone(pad.State().Axis(gamepad.LeftX), pad.State().Axis(gamepad.LeftY), 0.15)
		s.x, s.y = s.x+x, s.y+y
	}
	return m.NewText("pads: " + itoa(len(s.pads.Pads())))
}

func itoa(n int) string { return string(rune('0' + n%10)) }

func main() { _ = newPadView() }
```

- **No dead zone is applied.** A stick that rests off centre is the app's to correct: `ApplyDeadZone(v, dead)` for one
  axis, `ApplyStickDeadZone(x, y, dead)` for a stick (a circle, so that a diagonal is not cut by a square). Sticks are from
  -1 to 1 (down and right are positive), triggers from 0 to 1.
- **Pads move the focus, too.** By default a pad's buttons are also key events: South is `GameButtonA` (Flutter's default
  shortcuts activate the focused button with it), the D-pad is the arrow keys (the focus moves), the shoulders and
  triggers, Start, Back and Guide are the other game-button keys. Only the window that has the focus gets them. A game that
  reads the pads itself calls `gamepad.Of(bc).SetKeyEvents(false)`. Flutter does this on Android only (SPEC §12.10).
- **Turning the pads off, or starting them late.** By default the service starts with the app, so that a pad moves the focus,
  but it only reads: nothing is written to a pad (no light, no enhanced mode) until the app first calls `gamepad.Of(bc)`. An
  app that wants no pad at all sets `RunAppOpts.Gamepads` to `"off"`: no goroutine, no watch on `/dev/input`, no device
  opened, no pad key events, and `Available()` is false. `"ondemand"` starts nothing until the app first uses the package.
  `GLUTTER_GAMEPADS=off` (or `ondemand`, or `auto`) in the environment overrides the option for one run of any app. A
  PlayStation, Switch Pro or Bluetooth Xbox pad that has a HID driver disconnects and connects again once, when the app first
  uses the package (it moves from its event device to its driver); `"ondemand"` has no such step.

```go
func runWithoutPads(app m.Widget) error {
	return m.RunApp(app, m.RunAppOpts{Gamepads: "off"})
}
```

- **A pad that nothing is known of.** A pad that has no mapping in the table, and that does not follow the Linux Gamepad
  Specification, is not in `Pads()`. `AddMappings` takes lines in the community's format (the lines of SDL's
  `gamecontrollerdb.txt`, `GUID,name,a:b0,b:b1,...,platform:Linux`; `Pad.GUID()` is the GUID to put at the start) and the pad
  becomes a gamepad when a line is for it. The `GamepadMappingInvalid` error (`gamepad.TagMappingInvalid`) says a line that
  cannot be read; the lines before it are added.
- **Kinds and labels.** `Pad.Kind()` tells an Xbox pad from a PlayStation or a Switch one by the vendor and product
  numbers, and `Pad.FaceStyle()` how its face buttons are labelled (`FaceSony`: cross, circle, square, triangle), for the
  picture of a button you show.
- **On Linux** the pads are read from `/dev/input/event*`. On a desktop, systemd's rules (`70-uaccess.rules`) give the user
  at the seat the joysticks that are plugged in, so there is nothing to do. A pad that cannot be read is reported once in the log
  (`slog`, Info) and read when the permissions change. Without a seat (a service, a container, a remote session), put the
  user in the `input` group, or give the group the joysticks with a udev rule in `/etc/udev/rules.d/70-glutter-joystick.rules`:
  `SUBSYSTEM=="input", ENV{ID_INPUT_JOYSTICK}=="1", GROUP="input", MODE="0660"`, then `sudo udevadm control --reload &&
  sudo udevadm trigger`. Only a joystick or a gamepad is opened: keyboards and mice are left alone. If the system's inotify
  instances are used up, new pads are looked for every two seconds.
- **Rumble.** `Pad.Rumble(low, high, d)` makes the pad's two motors turn for `d` (low and high are the strengths from 0
  to 1 of the heavy and the light motor; a duration of 0 stops them) and `Pad.CanRumble()` says whether the pad can. It works
  on Linux (force feedback: the pad's node must be writable, which the seat's rules give with the read access; it is opened for writing at the first rumble), on Windows for
  the Xbox pads (XInput), on the web where the browser has `vibrationActuator` (Chrome and Edge; at most 5 seconds), and on
  Android 12 and later; everywhere else it returns `GamepadRumbleUnavailable`. `Pad.RumbleTriggers(left, right, d)` turns the
  motors in the triggers (`CanRumbleTriggers()`: the DualSense and the Xbox One and Series pads over Bluetooth) and
  `Pad.SetLED(color)` sets the light bar of a PlayStation pad or the Home light of a Switch Pro controller (`HasLED()`).
- **PlayStation, Switch Pro and Bluetooth Xbox pads** have a HID driver on Linux and Windows. It gives what the system's own
  driver does not show: the light, the trigger motors, the touchpad (`Pad.Touchpads()`, `Pad.Touches()` and
  `Pad.AddTouchListener(fn)`: fingers with a position from 0 to 1 and a pressure), the battery (`Pad.Battery()` and
  `Pad.Power()` also say whether the pad charges or is full) and the gyroscope and accelerometer (`Pad.OpenSensor`, section 22). A pad that cannot be reached this way works as any other pad. **On Linux** that means the user must be
  allowed to open `/dev/hidraw*`; on a desktop the seat's rules may not give hidraw access for every pad, and then the log says
  once (`slog`, Info) that a pad is read through its event device, and a udev rule in
  `/etc/udev/rules.d/71-glutter-gamepads.rules` gives the access:
  `KERNEL=="hidraw*", ATTRS{idVendor}=="054c", MODE="0660", TAG+="uaccess"` (Sony), the same with `057e` (Nintendo) and `045e`
  (Microsoft), and for pads that are connected over Bluetooth `KERNELS=="0005:054C:*", MODE="0660", TAG+="uaccess"` (and
  `0005:057E:*`, `0005:045E:*`); then `sudo udevadm control --reload && sudo udevadm trigger`. **On Windows** there is nothing to
  do; XInput keeps serving Xbox pads while it is there. The drivers start when the app first uses package `gamepad`, not
  with the app. `GLUTTER_GAMEPAD_HID=0` in the environment turns the drivers off, and every pad is read the old way.
- **On Windows** Xbox pads and the others that Windows serves through XInput are read at 125 Hz while they are connected (no
  work is done for an empty slot), and every other game controller through Raw Input and the HID parser, so a generic USB
  pad needs the line of the table that SDL has for it (`AddMappings` for a pad it does not know). A pad that only reports
  when something changes is at rest in the app's eyes until it first does.
- **On the web** the browser shows a pad to a page only after the user presses one of its buttons (a privacy rule of the
  browsers), and the page reads the pads only while one is connected. A pad with the browser's `standard` mapping is
  already in the layout. On a page without HTTPS (other than localhost) the browser may hide the pads.
- **On Android** a pad's buttons arrive once: as the gamepad's key events (`GameButtonA` and the others) and through
  `gamepad.Of(bc)`, never as keyboard keys as well. The `input gamepad keyevent KEYCODE_BUTTON_A` command of the shell makes
  a pad that Glutter calls "Virtual Gamepad" and is the way to try a pad on the emulator.
- **Tests.** `gluttertest` has `ConnectPad(PadSpec)`, which returns a `TestPad` with `Press`, `Release`, `Tap`, `SetAxis`,
  `Disconnect` and `Rumbles()` for a pad that reports the standard layout, or, with `Axes`, `Buttons`, `Hats` and `Mapping`,
  a raw pad with `RawButton`, `RawAxis` and `RawHat` that goes through the table of mappings.

## 21. Audio

`audio.Of(bc)` is the sound of the app. Open a device, make a stream for each sound, and put samples into it: the streams of
a device are mixed, converted to the device's format and clipped for you. Glutter reads WAV files (`audio/wav`); other codecs
(Ogg Vorbis, MP3, Opus) are not in Glutter: decode them yourself and put the samples. Flutter has no audio of its own, so this
package is Glutter's. On Linux the sound goes through PulseAudio (which PipeWire serves on current desktops), or through ALSA
where no PulseAudio server answers (`GLUTTER_AUDIO_BACKEND=pulse` or `alsa` picks one); nothing is loaded until the app lists or
opens a device, and an app never starts a sound server. On Windows it is WASAPI (shared mode, the default device followed when
the user changes it), on Android AAudio, and on the web an AudioWorklet that `glutter-web` puts in the build. Where the system has no
sound (or a browser has no audio worklet) `audio.Of(bc).Available()` is false and every open fails with `AudioUnavailable`.

```go
package main

import (
	"context"
	"math"

	"github.com/rm4n0s/glutter/audio"
	m "github.com/rm4n0s/glutter/material"
)

type beeper struct {
	dev *audio.Device
	st  *audio.Stream
}

func (b *beeper) start(bc m.BuildContext) error {
	dev, err := audio.Of(bc).OpenPlayback(context.Background()) // the default device, which it follows
	if err != nil {
		return err // audio.TagUnavailable, audio.TagDeviceOpenFailed, ...
	}
	b.dev = dev
	b.st = dev.NewStream(audio.Format{Rate: 44100, Channels: 1}) // converted to the device's format
	return nil
}

// beep puts half a second of a tone: the samples are a []float32 from -1 to 1 (also []int16 and []int32), one for each channel
// of a frame.
func (b *beeper) beep(freq float64) error {
	const rate = 44100
	tone := make([]float32, rate/2)
	for i := range tone {
		tone[i] = float32(0.3 * math.Sin(2*math.Pi*freq*float64(i)/rate))
	}
	if err := b.st.Put(tone); err != nil {
		return err
	}
	b.st.Flush() // a sound that ends: play its last frames too
	return nil
}

func main() { _ = &beeper{} }
```

- **Streams and the mix.** Each stream has its own format, gain (`SetGain`) and speed (`SetSpeed`: 2 plays it twice as fast, an
  octave higher), and the device has a gain of its own. `Put` and `Get` may be called from any goroutine; everything else is for
  the UI thread. A sound that is played again is a new `Put`. `Queued()` says how much is waiting, `Clear()` drops it.
- **Generated sound.** `dev.NewCallbackStream(format, fill)` calls `fill(out)` on the audio thread when the device needs
  frames: write interleaved float32 frames into `out` (what you leave alone is silence). `fill` must not touch the widgets, wait
  for the UI thread or allocate, or the sound breaks up.
- **Flush.** A stream whose rate or speed differs from the device's keeps its last few frames until more come (the resampler
  needs them); `Flush()` after the last `Put` of a sound plays them out. A stream that has the device's rate and speed 1 does not
  wait.
- **Recording.** `OpenCapture` gives a device whose streams (`NewStream(format)` with the format you want) are read with
  `Get(out)`, which returns the number of frames and does not wait. A stream that is not read keeps ten seconds.
- **Devices.** `Devices(audio.Playback)` lists them (`DeviceInfo.ID` goes into `DeviceOpts.ID`), `AddListener` tells when
  the list or the default changes. `Device.State()` is running, paused (`Pause`/`Resume`), suspended (a browser lets a page
  make sound only after a click or a key: the device runs by itself afterwards, and `Device.AddListener` tells you), lost
  (unplugged) or closed. A `Put` on a lost device fails with `AudioDeviceLost`.
- **Android.** A recording needs the permission `android.permission.RECORD_AUDIO` in the manifest: build with
  `glutter-android build -permission android.permission.RECORD_AUDIO` (an app that only plays must not ask for it). The first
  `OpenCapture` shows the system's question and waits for the answer; a refusal is `AudioCapturePermissionDenied`. Playback
  pauses when the app is hidden and goes on when it is shown (`DeviceOpts.PauseInBackground`, true by default there).
- **The web.** The page must be served from `https` or `localhost`, and `glutter-web serve` and `build` give you
  `glutter-audio-worklet.js` next to `index.html` (a deploy needs both files). A browser lets a page make sound only after a
  click, a touch or a key, so a device opens as `Suspended` and runs by itself at the first one. `glutter-web serve` sends the
  headers that make the page cross-origin isolated (`Cross-Origin-Opener-Policy: same-origin`, `Cross-Origin-Embedder-Policy:
  require-corp`), and then the app and the worklet share memory; a site that cannot send them still plays, with a little more
  work for the page (the frames travel as messages). The buffer is 100 ms (`DeviceOpts.BufferFrames` changes it). `OpenCapture`
  waits for the user's answer to the browser's question: call it from a goroutine that is not the UI one. A page plays on the
  browser's default output.
- **WAV files.** `wav.Decode(ctx, r)` gives `Audio{Format, Samples}` for PCM of 8, 16, 24 and 32 bits and float, ready for
  `Put`; `Format` is the format to make the stream with. A file that is not one gives `WAVInvalid` with a message.
- **Tests.** `gluttertest` has `tester.Audio()`: a speaker and a microphone that are the defaults, `AddDevice`, `RemoveDevice`,
  `SetDefault`, `SetUnavailable` and `FailOpen`, and the devices the app opened (`Playback()`, `Capture()`), on which the test
  plays the audio thread: `Pull(frames, chunk)` runs the callback and returns what would have been heard, `Push(frames)` is
  what a microphone recorded, and `Lose`, `Suspend` and `Unsuspend` change the state. Nothing plays by itself, so a test says
  when and for how long.

## 22. Motion sensors

`sensors.Of(bc)` gives the accelerometer and the gyroscope of a phone, of a browser on a phone, and of a gamepad that has
them. Flutter has none in its framework (the `sensors_plus` plugin does it); the names here are Glutter's. A reading is three
numbers on SDL's axes for a device in its natural orientation: x to the right, y up, z out of the screen. The accelerometer
reads in m/s² **with gravity**, so a phone that lies flat on its back reads about 9.8 on z; the gyroscope reads in rad/s,
counterclockwise as seen from the positive end of an axis.

```go
package main

import (
	"context"

	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/sensors"
)

type tilt struct {
	gravity [3]float64
}

// start opens the accelerometer; update is called on the UI thread, once a frame, with the newest reading.
func (t *tilt) start(bc m.BuildContext, update func()) {
	sys := sensors.Of(bc)
	if !sys.Has(sensors.Accelerometer) {
		return // a desktop, or a phone without one
	}
	// Open may wait for the user (a browser on iOS asks), so it is not called on the UI goroutine.
	go func() {
		accel, err := sys.Open(context.Background(), sensors.Accelerometer, sensors.Opts{Rate: 60})
		if err != nil {
			return // sensors: SensorUnavailable, SensorPermissionDenied, SensorOpenFailed
		}
		accel.AddListener(func(batch []sensors.Reading) {
			t.gravity = batch[len(batch)-1].Values // all the readings since the last frame, oldest first
			update()
		})
	}()
}

func main() { _ = &tilt{} }
```

- **Batches.** The listeners of a sensor are called once a frame, on the UI thread, with every reading that came since the last
  frame (oldest first; the slice is yours only during the call). A sensor at 200 readings a second on a 60 Hz screen gives about
  3 or 4 readings to each call; `Latest()` gives the newest reading at once, from any goroutine. A sensor with no listener makes
  no frames. While the app is hidden no frame runs, and the sensor keeps the newest 4096 readings (`Dropped()` counts the rest).
- **Rate.** `Opts.Rate` is the readings a second that you would like; the default is 60 (or the sensor's slowest allowed rate).
  `Sensor.Rate()` is what the sensor was set to, and 0 where the system does not say (a browser). Android 12 and later give an
  app at most 200 Hz unless its manifest has `HIGH_SAMPLING_RATE_SENSORS` (`glutter-android build -permission
  android.permission.HIGH_SAMPLING_RATE_SENSORS`).
- **Errors.** `Open` fails with `SensorUnavailable` (`platform.TagSensorUnavailable`: the desktop, a device without the sensor,
  a page that is not served over https or from localhost), `SensorPermissionDenied` (the user refused) or `SensorOpenFailed`.
  `Has(kind)` is false where the device has no such sensor; a browser cannot know before its first reading and says yes.
- **On the web** the browser sends `devicemotion` events at its own rate. Safari on iOS asks the user first, and only in the
  answer to a tap: the first `Open` waits for the first click or touch, asks, and goes on with the answer; Chrome and Firefox ask
  nothing. A phone's browser vibrates for `HapticFeedback` (`navigator.vibrate`, 5 to 50 ms for the five kinds); a desktop's
  does nothing.
- **A gamepad's sensors** are the same type: `pad.OpenSensor(sensors.Gyroscope)` turns the pad's sensors on (they cost battery,
  so a pad reports them only while one is open) and returns a `*sensors.Sensor`; closing the last one turns them off. Call
  both on the UI thread. It fails with `GamepadSensorsUnavailable` for a pad that has none.
- **Tests.** `gluttertest` has `tester.Sensors()`: `SetAvailable`, `SetDenied` and `FailOpen`, and the sensors that the app
  opened (`Open(kind)` gives a `TestSensor` with `Send` and `SendValues`). The readings reach the listeners in the next
  `Pump`. For a pad, `PadSpec{Sensors: true}` and `TestPad.Sensor(kind, x, y, z)`. On the Android emulator the virtual sensors
  are set with `adb emu sensor set acceleration 0:9.81:0` and `adb emu sensor set gyroscope 1:0:0`.

## 23. A game, from the pieces

`examples/game3d` (Coin Run) is a small 3D game that uses chapters 18 to 22 together: a skinned character that runs and jumps
through a glTF level with PBR materials and a shadow, played with a pad, the keyboard, a locked mouse or a phone's touch, with
music and effects, rumble on a hit and a Material pause menu. Its files: `world.go` (the rules, with no window, no GPU and no
sound, so a table test plays the game), `stage.go` (the scene, the character and the camera: it shows what the world holds),
`sound.go`, `assets.go` (what is built into the program) and `main.go` (the page). `internal/genassets` makes the character, the
level and the sounds from numbers, so they are public domain; a game of yours loads its own files the same way.

The frame is a ticker. A `view` with `Continuous: true` draws at every frame, and the state's ticker plays the game, with the
time since the last frame (a frame that took long is cut short, so that a window that was dragged does not throw the
character):

```go
package main

import (
	"math"

	"github.com/rm4n0s/glutter/gamepad"
	m "github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

type player struct {
	pads      *gamepad.Gamepads
	keyboard  *services.HardwareKeyboard
	list      []*gamepad.Pad // reused, so that a frame allocates nothing
	pos       vecmath.Vec3
	character *viewport3d.Uploaded // what viewport3d.Upload made of the character's glTF file
	run       *viewport3d.Animation
	pose      *viewport3d.Pose
	clock     float64
}

func newPlayer(bc m.BuildContext) *player {
	b := m.WidgetsBindingOf(bc)
	return &player{pads: gamepad.ForBinding(b), keyboard: b.Keyboard(), pose: viewport3d.NewPose()}
}

// frame is called by a ticker once a frame, with the seconds since the last one.
func (p *player) frame(dt float64) {
	x, z := 0.0, 0.0
	if p.keyboard.IsLogicalKeyPressed(services.LogicalKeyboardKeyKeyW) { // the keys that are down now: no key handler to keep
		z--
	}
	if p.keyboard.IsLogicalKeyPressed(services.LogicalKeyboardKeyKeyS) {
		z++
	}
	p.list = p.pads.AppendPads(p.list[:0]) // Pads() makes a slice each time
	for _, pad := range p.list {
		st := pad.State()
		lx, ly := gamepad.ApplyStickDeadZone(st.Axis(gamepad.LeftX), st.Axis(gamepad.LeftY), 0.18)
		x, z = x+lx, z+ly
	}
	p.pos = p.pos.Add(vecmath.Vec3{X: x, Z: z}.Scale(5 * dt))
	p.clock += dt
	p.pose.Clear() // a pose keeps what the last animation set
	p.run.Sample(float32(math.Mod(p.clock, float64(p.run.Duration()))), p.pose)
	p.character.SetPose(p.pose)
	p.character.SetTransform(vecmath.NewMat4Translation(p.pos)) // the whole model, in the world
}

func main() { _ = newPlayer }
```

- **Input.** The sticks and the held keys are read in the frame; the discrete actions (pause, a menu) are key events, which a pad makes
  too (chapter 20): Start is `GameButtonStart`, B is `GameButtonB`, the D-pad is the arrows and A activates the focused button, so
  a Material menu works with a pad and no code of yours. A pad's jump is the press of a button in this frame, not the button's
  state (a held button is not a second jump). The mouse is locked by a click (`PointerLockOf(bc).Lock(ViewOf(bc))`; chapter 18); the
  lock ends by itself when the window loses the focus, and then the game pauses. Touch is a `Listener` that tells which half of the
  view a finger began in: the left one moves, the right one looks.
- **The model.** `viewport3d.Upload` of a glTF file gives items; `Uploaded.SetTransform` moves the whole model and `SetPose` shows a
  pose of an animation (chapter 19). The pose of an animation and a jump are the app's choice (the example's `world` picks
  `idle`, `run`, `jump` or `hit` from what the character does); there are no blend trees. A coin is an item of the level that
  the game turns and hides.
- **Sound.** Open the device once and keep it. A looping music is a callback stream (the function that fills `out` runs on the
  audio thread and touches nothing of the app); an effect is a stream of its own, started again with `Clear` and `Put`:

```go
package main

import (
	"context"

	"github.com/rm4n0s/glutter/audio"
	"github.com/rm4n0s/glutter/audio/wav"
	m "github.com/rm4n0s/glutter/material"
)

func startMusic(ctx context.Context, bc m.BuildContext, loop *wav.Audio) (*audio.Device, error) {
	dev, err := audio.Of(bc).OpenPlayback(ctx)
	if err != nil {
		return nil, err // AudioUnavailable: a program with no sound plays on
	}
	pos := 0
	music := dev.NewCallbackStream(loop.Format, func(out []float32) {
		for i := range out {
			out[i] = loop.Samples[pos]
			if pos++; pos == len(loop.Samples) {
				pos = 0
			}
		}
	})
	music.SetGain(0.3)
	return dev, nil
}

func main() { _ = startMusic }
```

- **A hit** makes `pad.Rumble(0.9, 0.7, 350*time.Millisecond)` on every pad that `CanRumble`, and
  `HapticFeedback().HeavyImpact()` on a phone.
- **Tests.** The rules are a plain Go type: `world_test.go` plays frames of it. The page is tested with `gluttertest`: `ConnectPad`,
  `SetAxis`, `Tap(gamepad.Start)`, `Rumbles()`, `SendKeyDownEvent`, `MoveLockedMouse`, `EndPointerLock`, and the sound that reaches
  the device (`tester.Audio().Playback().Pull(frames, chunk)`), with `Opts{GPU: true}` for the view.
- **Speed.** The renderer draws a frame of the example at 1920x1080 with 4x multisampling in under 2 ms on a Radeon RX 6800M (the
  target of SPEC §28.6 is 60 frames a second); the program's own part of a frame (input, rules, animation, sync) takes about 0.4 ms and
  allocates nothing, and a frame of the renderer with a moving skinned item, turning items and a moving camera allocates nothing
  (`viewport3d`'s `TestAFrameOfMovingThingsAllocatesNothing`).

## 24. The Cupertino look

Package `cupertino` is Flutter's `cupertino` library: the widgets of iOS and macOS, ported file by file and drawn the same
on every platform. It is plain Go over `widgets`, imported as `c` (it re-exports `widgets`, as `material` does, so a Cupertino
app imports only it):

```go
package main

import (
	"log"

	c "github.com/rm4n0s/glutter/cupertino"
)

func main() {
	err := c.RunApp(c.NewCupertinoApp(c.CupertinoAppOpts{
		Title: "Settings",
		Theme: c.NewCupertinoThemeData(c.CupertinoThemeDataOpts{Brightness: c.BrightnessLight}),
		Home: c.NewCupertinoPageScaffold(c.NewCustomScrollView([]c.Widget{
			c.NewCupertinoSliverNavigationBar(c.CupertinoSliverNavigationBarOpts{LargeTitle: c.NewText("Settings")}),
			c.NewSliverToBoxAdapter(c.NewCupertinoListSectionInsetGrouped(c.CupertinoListSectionOpts{
				Header: c.NewText("Connections"),
				Children: []c.Widget{
					c.NewCupertinoListTileNotched(c.NewText("Airplane Mode"), c.CupertinoListTileOpts{
						Trailing: c.NewCupertinoSwitch(false, func(bool) {}),
					}),
				},
			})),
		})),
	}))
	if err != nil {
		log.Fatal(err)
	}
}
```

- **What is there.** Everything of Flutter's `cupertino` at the pinned commit: `CupertinoApp` and `CupertinoApp.router`, the page route
  with the parallax slide and the back swipe, the sheet, the modal popup and the dialog routes, the tab scaffold with a navigator
  for each tab, the navigation bars (with the large title that collapses and flies between two routes), the button, the switch,
  checkbox, radio, slider, activity indicators and both segmented controls, the text field, the search field and the form rows,
  the list sections and tiles, the expansion tile, the pull to refresh sliver, the alert dialog, the action sheet, the context menu
  and the menu anchor, the pickers (the wheel, the date and time picker, the countdown timer), the scrollbar, the text selection
  with its handles, toolbars and magnifier, and the colors (`CupertinoColors`, `CupertinoDynamicColor`, which resolves by brightness,
  contrast and interface level) and `CupertinoIcons`. `examples/cupertino` shows each of them on the pages of a tab scaffold, with a
  dark mode switch.
- **The font.** The text styles name the families `CupertinoSystemText` and `CupertinoSystemDisplay`. On macOS and iOS they are San
  Francisco, found through Core Text; San Francisco is never bundled. Elsewhere the machine's default font draws, and on the web
  (which has no system fonts) the bundled Roboto does.
- **Material and Cupertino together.** `material` imports `cupertino` (never the other way), so a Material app has the adaptive
  widgets of Flutter: `m.NewSwitchAdaptive`, `NewCheckboxAdaptive`, `NewRadioAdaptive`, `NewSliderAdaptive`,
  `NewCircularProgressIndicatorAdaptive`, `NewRefreshIndicatorAdaptive`, `NewAlertDialogAdaptive` with `ShowAdaptiveDialog`, and the
  adaptive scrollbar, list tiles and text selection toolbar look like iOS when the theme's platform is iOS or macOS and like Material
  elsewhere. `Theme` also gives the widgets below it a `CupertinoTheme` made from the Material theme
  (`m.MaterialBasedCupertinoThemeData`, with `ThemeData.CupertinoOverrideTheme` to change it), and a `TextField` on iOS or macOS takes the
  cursor and selection of that theme, its Cupertino handles and toolbar, and the spell check of iOS.
- **Tests.** `gluttertest` runs a Cupertino app as any other. The goldens of the package use Go Mono for the two system families, so
  that they are the same on every host, and `foundation.SetDebugDefaultTargetPlatformOverride` picks the platform of a test.
- **Speed.** A frame of the page transition takes about 60 microseconds of the UI thread on average (the first one, which builds
  the page, about 2 ms), a frame of the large title collapsing 25 microseconds, and a frame of a flinging picker 40 microseconds
  (`cupertino/bench_test.go`, `docs/PERFORMANCE.md`).

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

## For Flutter Windows developers

If you build Flutter apps for Windows, this is how a Glutter app becomes an `.exe`.

- **Build and run.** There is no `flutter build windows` and no `windows/` folder with a C++ runner:
  `go run ./cmd/glutter-windows build ./your/app` makes `app.exe`, a Go program with no DLL next to
  it (`-arch amd64,arm64` builds both, `-release` strips it and builds with `glutter_release`, `-o`
  names the file). On Linux `run` builds it and starts it under Wine (`-headless` on a hidden X
  server instead of your screen; it needs Wine and Xvfb and must be started in a checkout of
  Glutter), on Windows it starts the `.exe` itself, and `shot -o file.png` saves the frame the app
  draws. A plain `GOOS=windows CGO_ENABLED=0 go build` also gives an `.exe` that runs, but without
  the resources below.
- **The manifest.** The tool generates the application manifest that Flutter's runner keeps in
  `runner.exe.manifest`: per-monitor DPI awareness v2, so the window is sharp at 150% and follows a
  move between monitors; the UTF-8 code page; long paths; version 6 of the Common Controls, so the
  system dialogs look current; Windows 10 and 11 as supported systems; and `asInvoker`. `-name`,
  `-app-id` and `-version 1.2.3.4` set the description, the assembly name and the file version that
  Explorer's Properties dialog shows. Without a manifest the program still makes itself per-monitor
  aware when it starts, which is a little later than the manifest does it.
- **The icon.** `-icon logo.png` takes a square PNG of at least 16 pixels and makes the icon of the
  `.exe` (16, 24, 32, 48, 64, 128 and 256 pixels) that Explorer, the task bar and the title bar show;
  without it the icon is Glutter's. Give a PNG of 256 pixels or more so that the large sizes are
  sharp.
- **The GUI subsystem and the log.** A build is a GUI program, so double-clicking it opens no console
  window. Start it from a terminal and its `slog` lines appear there; started from Explorer, they go
  to `OutputDebugString`, where DebugView or a debugger shows them. `-console` builds a console
  program instead, for a tool that prints to its terminal.
- **No plugins.** A Flutter Windows app reaches the system through plugins (C++ code in
  `windows/flutter/ephemeral` and the plugin registrant) and platform channels. A Glutter app is one
  Go program and has neither: use the standard library, call Win32 yourself with `syscall`
  (`syscall.NewLazyDLL("user32.dll").NewProc("MessageBeepW").Call(0)`; callbacks need
  `syscall.NewCallback`, made once), or use `desktop` for what Flutter leaves to plugins (the file
  dialogs, notifications, the tray, `OpenURI`). Where your code must differ, ask
  `foundation.DefaultTargetPlatform() == foundation.TargetPlatformWindows`, as in Flutter, and never
  `runtime.GOOS`. The framework takes Flutter's Windows branches by itself: the Windows text
  editing shortcuts, the scrollbar and the Material text selection controls.
- **Fonts and text.** The Material theme asks for Segoe UI on Windows, as Flutter's typography does,
  and the matcher finds it in the registry's list of system fonts (`sans-serif` and `system-ui` are
  Segoe UI, `monospace` is Consolas); color emoji fall back to Segoe UI Emoji and the other scripts
  to Microsoft YaHei, Yu Gothic, Malgun Gothic and the rest of the chain of SPEC §25.8. The bundled
  Roboto and Material Icons stay first, as on every platform.
- **What a Flutter Windows app has that Glutter does not (yet).** Real toast notifications (a
  notification is a tray balloon, which Windows shows as a toast but with one action: a click on
  it); the snap layouts of Windows 11 over the maximize button of a title bar that the app draws
  itself (`WindowDecoration`; the native bar has them); `DirectManipulation`, which gives a
  precision touchpad its inertia (Glutter scrolls by the wheel messages the touchpad makes); the
  touch keyboard's handwriting panel; an installer or MSIX package (you ship the `.exe`; an
  unsigned one makes SmartScreen ask once).
- **Testing and measuring without a PC.** `go run ./cmd/glutter-windows run ./your/app` on Linux
  starts the `.exe` under Wine, and the repository's Wine tests run the whole Windows platform
  there. Wine's Direct3D 12 is vkd3d on the host's Vulkan (a CPU rasterizer unless you have a
  driver that presents on Xvfb), so frame times, the first frame (2 to 12 seconds) and the memory
  of `scripts/perf-windows.sh` are for noticing regressions, not what a GPU does; measure on a PC
  with Task Manager and `GLUTTER_FRAME_STATS=1`.
- **Unsigned.** There is no installer, MSIX or code signing: the first start of an unsigned `.exe`
  makes SmartScreen ask once. Signing is the app author's.

## For Flutter macOS developers

If you build Flutter apps for macOS, this is how a Glutter app becomes an `.app`.

- **Build and run.** There is no `flutter build macos` and no `macos/` folder with a Swift runner or an Xcode project:
  `go run ./cmd/glutter-macos build ./your/app` makes `app.app`, a signed bundle, and needs only Go and the
  `codesign` of the Command Line Tools. `-o` names the bundle, `-name`, `-id` and `-version` set what the Dock, the
  bundle identifier and Finder's Get Info show, `-release` strips the program and builds with `glutter_release`,
  and `-universal` makes one executable for Apple silicon and Intel (a Mac builds both by itself, no `lipo`).
  `run` builds the bundle in a temporary directory and starts it with its log on your terminal, `shot -bundle -o
  file.png` saves the frame the app draws (no screen recording permission: the app saves its own frame), and
  `perf` measures it. A plain `GOOS=darwin CGO_ENABLED=0 go build` also gives a program that runs, but it is not in
  a bundle, so it has no identifier, no icon of its own and no notifications.
- **The plist.** The tool writes the `Info.plist` that Flutter keeps in `macos/Runner/Info.plist`: the bundle
  identifier (`-id dev.example.app`; `dev.glutter.<name>` without it), the name, the version
  (`CFBundleShortVersionString` and `CFBundleVersion`), the icon, macOS 12 as the minimum, `NSHighResolutionCapable`,
  and the usage string of the microphone. Another usage string (`NSCameraUsageDescription`, `NSLocationUsageDescription`
  and the rest) is `-usage KEY="text"`, repeatable; macOS shows it the first time the app asks.
- **The icon.** `-icon logo.png` takes a square PNG of at least 16 pixels, and the tool scales it to the ten sizes of an
  `.icns` file (16 to 1024 pixels). Give a PNG of 1024 pixels so that the large sizes are sharp, with the margin that
  macOS icons have (an icon that fills its square looks larger than the others in the Dock). Without `-icon` the icon is
  Glutter's.
- **Signing, notarizing and Gatekeeper.** The tool signs ad hoc (`codesign --sign -`), which gives the bundle a
  signature that binds its plist and resources to the program; that is enough to run an app you built on your own Mac,
  and an app that someone downloads is stopped by Gatekeeper once ("an unidentified developer": Control-click, Open).
  To ship to others, sign with your Developer ID and notarize it, which are yours to do and need an Apple Developer
  account. Build with `-hardened` (the hardened runtime, with the entitlement of the microphone), then:
  ```sh
  codesign --force --sign "Developer ID Application: Your Name (TEAMID)" --options runtime --timestamp \
      --entitlements entitlements.plist App.app          # entitlements.plist: com.apple.security.device.audio-input, if you record
  ditto -c -k --keepParent App.app App.zip
  xcrun notarytool submit App.zip --keychain-profile "notary" --wait
  xcrun stapler staple App.app
  ```
  The Mac App Store needs the App Sandbox and a provisioning profile, which this tool does not make.
- **Frames and displays.** A window follows its display: on macOS 14 and later the view's `CADisplayLink` gives
  the display's own rate (120 frames a second on a ProMotion Mac), on macOS 12 and 13 a `CVDisplayLink` does; a window
  on a second display with another scale is redrawn at that scale. The renderer is Metal (`GLUTTER_RENDERER=metal`),
  and `soft` draws into IOSurfaces on a Mac that has no Metal device (a virtual machine without GPU passthrough).
  A `CADisplayLink` gives no tick to a window on a display that is asleep, which is how a Mac looks through Screen
  Sharing with its own monitor off; since 1.16.3 the link is watched, and a `CVDisplayLink` paces the frames while it
  is silent, so the app draws and answers clicks there too. `GLUTTER_DISPLAYLINK=cv` or `ca` uses one of the two
  only, to find which one a problem belongs to. A screenshot that the app takes of itself (`GLUTTER_SCREENSHOT`,
  `glutter-macos shot`) shows what the app draws and not what macOS draws over it (the window buttons), and with it
  the frames come from the `CVDisplayLink` alone unless `GLUTTER_DISPLAYLINK=auto` is set: it proves the drawing,
  not that a person can use the window.
- **Your own title bar.** With `WindowConfig.Decorations` set to `m.DecorationsApp`, a macOS window has no title bar of
  the system but keeps the system's red, yellow and green buttons at its top left (top right in a right-to-left
  language). `m.NewWindowDecoration` knows: it draws no buttons, border or resize strips of its own there (macOS
  resizes the window), starts its `Leading` widgets after the buttons, and tells the window how high its bar is so
  that the buttons sit in the middle of it. A bar that you draw without `WindowDecoration` does the same with two
  things: `m.ViewOf(bc).Window().SystemButtons` is the area the buttons take, in logical pixels of the view (empty
  where the system has none, so the same code runs on Linux and Windows), and
  `view.PerformWindowAction(ui.WindowAction{Kind: ui.WindowActionTitleBarHeight, Height: h})`, sent after a frame,
  centers them in a bar of that height. In full screen macOS hides the buttons and the area is empty.
- **No plugins.** A Flutter macOS app reaches the system through plugins (Swift or Objective-C code and the plugin
  registrant) and platform channels. A Glutter app is one Go program and has neither: use the standard library, call
  AppKit yourself through `github.com/ebitengine/purego/objc` (`objc.GetClass("NSSound").Send(...)`; a Go callback cannot
  return a structure such as an `NSRect`, and a callback is made once and found by its context), or use `desktop` for what
  Flutter leaves to plugins (the file dialogs, notifications, the tray, `OpenURI`). Where your code must differ, ask
  `foundation.DefaultTargetPlatform() == foundation.TargetPlatformMacOS`, as in Flutter, and never `runtime.GOOS`.
  The framework takes Flutter's macOS branches by itself: the Command key for shortcuts, the macOS text editing
  shortcuts, the bouncing scroll physics of a trackpad.
- **The menu bar.** `PlatformMenuBar` is the menu bar of the app (Flutter's own widget), and the application menu that
  macOS expects (About, Services, Hide, Hide Others, Show All, Quit) is made for it, so `Cmd+Q` and `Cmd+H` work. An app run
  with `go run` (no bundle) has the name of its program in the menu bar. The Dock icon's menu and badge, and the pointer
  lock (`widgets.PointerLock`), are not done on macOS.
- **Notifications and the microphone need a bundle.** `desktop` notifications use `UNUserNotificationCenter`, which
  needs a bundle identifier: from a bundle that Finder or `open` started they work (and ask the user once); from
  `go run` they fail with `NotificationsUnavailable`, which says why. The first recording asks for the microphone with
  the bundle's usage string; an app that is not in a bundle is asked about as the terminal that started it.
- **Gamepads and sound.** `gamepad` reads pads through IOKit, and a PlayStation, Switch Pro or Xbox pad over Bluetooth
  through its own HID driver (the light, the touchpad, the motion sensors, the trigger motors); a pad that no HID
  driver serves rumbles through the Force Feedback framework since 1.16 (written after SDL's, and tested against a fake
  of the framework: tell us if yours does not). `audio` plays and records through Core Audio, follows the system's default device when it changes,
  and reports a device you named as lost when it is unplugged.
- **What a Flutter macOS app has that Glutter does not (yet).** The App Sandbox and the Mac App Store (an app is shipped
  as a notarized `.app` or a disk image of your own), a Touch Bar, Handoff and iCloud, the share sheet, the native
  `NSTextView` features that Flutter's text field does not have either, and platform views (`AppKitView`).
- **Testing and measuring.** The repository's tests run the whole platform on a Mac with `-tags glutter_display`
  (real windows, never taking your focus: they are accessory apps, with Metal's validation layer on). `scripts/perf-macos.sh`
  measures a bundle on your Mac; the numbers are that machine's, and `docs/PERFORMANCE.md` has the ones of the Mac that
  made the first table.

## For Flutter iOS developers

If you build Flutter apps for iOS, this is how a Glutter app becomes an `.app` for the Simulator and for a device.

- **Build and run.** There is no `flutter build ios` and no `ios/` folder with a Swift runner or an Xcode project:
  `go run ./cmd/glutter-ios build ./your/app` makes `app.app`, and `run` builds one for the Simulator, installs it on a
  device of the Simulator (an iPhone by default, `-device ipad` for an iPad), launches it and shows its log on your
  terminal. `shot -o file.png` saves the frame the app draws. You need the full Xcode (the SDK and the Simulator's
  runtime are in it), not an Xcode project. `run` and `shot` use your own Simulator devices (`-set dir` takes a device set
  of another directory, which is what the repository's tests do, so that they never touch yours).
- **Cgo.** Unlike every other target of Glutter, `GOOS=ios` needs cgo: Go links an iOS program with Xcode's `clang`,
  and the tool sets `CGO_ENABLED=1`, the SDK and the target for you. Your own code stays Go and needs no C compiler of its
  own; `go build ./...` for iOS (`scripts/check-ios.sh`) does what `GOOS=ios CGO_ENABLED=1 CC=... go build` does.
- **The plist.** The tool writes what `ios/Runner/Info.plist` has in Flutter: the identifier (`-id dev.example.app`;
  `dev.glutter.<name>` without it), the name, the version, iOS 15 as the minimum, a launch screen as a dictionary, the
  scene manifest, `UISupportedInterfaceOrientations` (`-orientations portrait,landscape-left`), and
  `CADisableMinimumFrameDurationOnPhone`, so that a ProMotion phone runs at 120 frames a second. A usage string
  is `-usage NSCameraUsageDescription="text"`, repeatable: iOS shows it the first time the app asks, and an app with no
  string for a sensor is ended by the system when it asks.
- **The icon.** `-icon logo.png` takes a square PNG of at least 167 pixels; the tool writes the sizes that iOS reads (no
  asset catalog, so no `actool`). Give a PNG of 1024 pixels with no transparency: iOS rounds the corners itself.
- **Signing for a device.** The Simulator takes an ad hoc signature. A device needs your team and a provisioning profile
  that has the device and the app's identifier (make it in Xcode or at developer.apple.com):
  ```sh
  go run ./cmd/glutter-ios build -team ABCDE12345 -profile Gallery.mobileprovision -id dev.example.gallery -o Gallery.app ./examples/gallery
  xcrun devicectl device install app --device <UDID> Gallery.app
  ```
  `-identity` names the certificate (`Apple Development` by default); a build for the App Store is signed with a
  distribution identity and profile, and uploading it (Transporter or `xcrun altool`) is yours to do.
- **Frames and the background.** A `CADisplayLink` gives the display's rate and Metal draws (`GLUTTER_RENDERER=soft`
  draws on the CPU into IOSurfaces). iOS ends an app that commits GPU work while it is in the background, so Glutter draws no
  frame there and waits for the GPU before the app leaves the screen. `AppLifecycleState` follows the state of the app's
  scene, as in Flutter's embedding.
- **No plugins.** A Flutter iOS app reaches the system through plugins (Swift or Objective-C, the plugin registrant) and
  platform channels. A Glutter app is one Go program and has neither: use the standard library, call UIKit yourself
  through `github.com/ebitengine/purego/objc`, or use what the framework gives: `services.Clipboard`, `HapticFeedback`,
  `SystemSound`, `SystemChrome` (the status bar's style, the orientations, the home indicator), spell check, `desktop`
  notifications and `OpenURI`, `gamepad`, `audio` and `sensors`. Where your code must differ, ask
  `foundation.DefaultTargetPlatform() == foundation.TargetPlatformIOS`, as in Flutter, and never `runtime.GOOS`.
- **The soft keyboard and the text.** A field gets the soft keyboard and its insets as in Flutter (`MediaQuery`'s
  `viewInsets`), with the keyboard type and the action of `TextInputConfiguration`, autocorrect and the spell check, the
  composing of Chinese, Japanese and Korean input methods, a hardware keyboard, and Cupertino's selection handles and
  magnifier. iOS asks the user before an app pastes what another app copied, so a read of the clipboard can wait for the
  user: it is asynchronous and never blocks the UI.
- **Dynamic Type and accessibility.** The text scale is the user's Dynamic Type size (Flutter's table of sizes, so
  `MediaQuery.textScaler` is what a Flutter iOS app sees, and a font scales as `UIFontMetrics` scales it). VoiceOver,
  Switch Control and Voice Control read a tree of elements made from the semantics, with traits, values, hints, custom
  actions and the adjust, scroll and escape gestures.
- **What a Flutter iOS app has that Glutter does not (yet).** Several scenes on an iPad, drag and drop between apps,
  platform views (`UiKitView`), the share sheet, widgets and Live Activities, background tasks and push notifications,
  the document picker, and Xcode's tools for a Flutter project (`flutter run` with hot reload).
- **Testing and measuring.** The repository's tests run on the Simulator with `-tags glutter_ios`
  (`go test -p 1 -tags glutter_ios ./internal/iostest ./examples`): they start and shut down their own devices and leave
  none booted. `docs/PERFORMANCE.md` has the numbers of the Simulator, and `docs/MANUAL-CHECKS.md` the list of what only
  a phone shows.

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
- **Your own title bar.** Set `WindowConfig.Decorations` to `m.DecorationsApp` and put
  `m.NewWindowDecoration(child, m.WindowDecorationOpts{Title: ..., Leading: ..., Actions: ...})` around your
  app. It draws the title, the window buttons and the resize edges. On macOS the window buttons and the resizing
  are the system's, at the left of the bar ("For Flutter macOS developers" above).
  - Only the title and the empty space around it move the window, maximize it on a double click and open the
    window menu on a right click. The `Leading` and `Actions` widgets get their own gestures, so tabs in the bar
    can be dragged, double-clicked and right-clicked like any widget.
  - The `Actions` are laid out first. The `Leading` widgets come next, as one row that may take all the room the
    `Actions` leave: a loose `m.NewFlexible` of tabs takes what it needs, an `m.NewExpanded` takes all of it. The
    title gets the rest.
- **Where the list of every difference is.** SPEC §12.10, one row per deviation, kept up to
  date.
