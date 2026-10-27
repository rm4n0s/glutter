package offscreen_test

import (
	"image"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/material"
	"github.com/rm4n0s/glutter/offscreen"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

const (
	counterW = 480
	counterH = 360
	// The center of the floating action button of the counter, in a view of counterW x
	// counterH: 16 pixels from the corner, 56 pixels wide.
	fabX = counterW - 16 - 28
	fabY = counterH - 16 - 28
	// pace is the time between two frames of a view of 30 frames a second, rounded up.
	pace = 34 * time.Millisecond
)

// counter is a Material counter whose state the test reaches (tap the button, or bump).
type counter struct {
	widgets.StatefulWidget
	probe *counterProbe
	// light is a counter of a line of text with no Material around it, which builds and
	// draws fast: the tests of a thousand frames use it.
	light bool
}

// counterProbe is how a test reaches the state of the counter that is mounted.
type counterProbe struct{ s *counterState }

// bump adds one to the counter, as the button does. Call it on the UI thread.
func (p *counterProbe) bump() { p.s.increment() }

func newCounter(probe *counterProbe) *counter {
	return &counter{StatefulWidget: widgets.NewStatefulWidgetBase(), probe: probe}
}

func newLightCounter(probe *counterProbe) *counter {
	c := newCounter(probe)
	c.light = true
	return c
}

func (c *counter) CreateState() widgets.State { return &counterState{} }

type counterState struct {
	widgets.StateOf[*counter]
	count int
}

func (s *counterState) InitState() {
	if p := s.Widget().probe; p != nil {
		p.s = s
	}
}

func (s *counterState) increment() { s.SetState(func() { s.count++ }) }

func (s *counterState) Build(bc widgets.BuildContext) widgets.Widget {
	if s.Widget().light {
		return widgets.NewDirectionality(ui.TextDirectionLTR, widgets.NewColoredBox(ui.NewColor(0xFFE8EAF6), widgets.NewCenter(
			widgets.NewText("Count "+strconv.Itoa(s.count)+" "+strings.Repeat("x", s.count%40), widgets.TextOpts{
				Style: &painting.TextStyle{Inherit: true, FontSize: foundation.Some(14.0), Color: foundation.Some(ui.NewColor(0xFF000000))}}))))
	}
	theme := material.NewThemeData(material.ThemeDataOpts{Platform: foundation.Some(foundation.TargetPlatformLinux),
		ColorSchemeSeed: foundation.Some(material.Colors.Blue.Primary)})
	return material.NewMaterialApp(material.MaterialAppOpts{Theme: theme, DebugShowCheckedModeBanner: foundation.Some(false),
		Home: material.NewScaffold(material.ScaffoldOpts{
			AppBar: material.NewAppBar(material.AppBarOpts{Title: widgets.NewText("Counter")}),
			Body: widgets.NewCenter(widgets.NewColumn([]widgets.Widget{
				widgets.NewText("You have pushed the button this many times:"),
				widgets.NewText(strconv.Itoa(s.count), widgets.TextOpts{Style: material.ThemeOf(bc).TextTheme.HeadlineMedium}),
			}, widgets.ColumnOpts{MainAxisAlignment: material.MainAxisAlignmentCenter})),
			FloatingActionButton: material.NewFloatingActionButton(s.increment, widgets.NewIcon(material.Icons.Add()),
				material.FloatingActionButtonOpts{Tooltip: "Increment"}),
		})})
}

// received is a frame as the callback saw it, with a copy of the pixels.
type received struct {
	number int
	damage []image.Rectangle
	img    *image.RGBA
}

// harness is a parent tester (the app's binding, on this goroutine as the UI thread) and a
// view on it, paced by the tester's fake clock. The frames of the view are recorded.
type harness struct {
	t      *testing.T
	parent *gluttertest.Tester
	view   *offscreen.View

	mu     sync.Mutex
	frames []received
	// onFrame, when set, runs after a frame is recorded, on the goroutine of the view.
	onFrame func(number int)
}

// newHarness makes a parent and a view of root. The view's callback copies the image.
func newHarness(t *testing.T, root widgets.Widget, size ui.Size, opts ...offscreen.ViewOpts) *harness {
	t.Helper()
	h := &harness{t: t, parent: gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 320, Height: 200}})}
	h.parent.PumpWidget(widgets.NewSizedBoxShrink(nil))
	v, err := offscreen.NewViewWithClock(h.parent.Binding, size, root, h.record, h.parent.Clock, opts...)
	if err != nil {
		t.Fatal(err)
	}
	h.view = v
	t.Cleanup(h.view.Close)
	return h
}

func (h *harness) record(f offscreen.Frame) {
	img := &image.RGBA{Pix: append([]uint8(nil), f.Image.Pix...), Stride: f.Image.Stride, Rect: f.Image.Rect}
	h.mu.Lock()
	h.frames = append(h.frames, received{f.Number, f.Damage, img})
	cb := h.onFrame
	h.mu.Unlock()
	if cb != nil {
		cb(f.Number)
	}
}

// received returns the frames so far.
func (h *harness) received() []received {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]received(nil), h.frames...)
}

// last is the newest frame, and fails the test when there is none.
func (h *harness) last() received {
	h.t.Helper()
	fr := h.received()
	if len(fr) == 0 {
		h.t.Fatal("the view made no frame")
	}
	return fr[len(fr)-1]
}

// step lets the fake clock run for d, runs what was posted to the UI thread (a frame of the
// view, if one is due), and waits for the goroutine of the view to draw it.
func (h *harness) step(d time.Duration) {
	h.t.Helper()
	h.parent.Pump(d)
	h.view.WaitIdle()
}

// settle runs frames until the view asks for no more (ink splashes end).
func (h *harness) settle() {
	h.t.Helper()
	for i := 0; i < 100; i++ {
		h.step(100 * time.Millisecond)
		if !h.view.Scheduled() {
			return
		}
	}
	h.t.Fatal("the view never settled")
}

// tap sends a touch down and up at the position through the dispatcher of the view, as an
// engine does.
func tap(v *offscreen.View, x, y float64) {
	d := v.Binding().PlatformDispatcher()
	p := ui.PointerData{Kind: ui.PointerDeviceKindTouch, Device: 1, PhysicalX: x, PhysicalY: y}
	down, up := p, p
	down.Change = ui.PointerChangeDown
	up.Change = ui.PointerChangeUp
	d.DispatchPointerDataPacket(ui.PointerDataPacket{Data: []ui.PointerData{down}})
	d.DispatchPointerDataPacket(ui.PointerDataPacket{Data: []ui.PointerData{up}})
}

func TestViewCounterTap(t *testing.T) {
	size := ui.Size{Width: counterW, Height: counterH}
	tests := []struct {
		name string
		taps int
	}{
		{"no tap", 0},
		{"one tap", 1},
		{"three taps", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, newCounter(nil), size)
			h.step(0)
			first := h.last()
			if first.number != 1 || len(first.damage) != 1 || first.damage[0] != first.img.Rect {
				t.Fatalf("the first frame is number %d with damage %v, want 1 and the whole image", first.number, first.damage)
			}
			for range tt.taps {
				tap(h.view, fabX, fabY)
				h.step(pace)
			}
			h.settle()

			ref := gluttertest.New(t, gluttertest.Opts{Size: size})
			ref.PumpWidget(newCounter(nil))
			for range tt.taps {
				ref.TapAt(ui.Offset{Dx: fabX, Dy: fabY})
				ref.Pump(pace)
			}
			ref.PumpAndSettle()
			samePixels(t, "view and Tester.Screenshot", h.last().img, ref.Screenshot())
			if tt.taps == 0 && len(h.received()) != 1 {
				t.Errorf("a view that changed nothing made %d frames, want 1", len(h.received()))
			}
		})
	}
}

func TestViewScaleAndSize(t *testing.T) {
	tests := []struct {
		name     string
		size     ui.Size
		ratio    float64
		resizeTo ui.Size
		wantW    int
		wantH    int
		thenW    int
		thenH    int
	}{
		{"ratio 1", ui.Size{Width: 320, Height: 200}, 0, ui.Size{Width: 400, Height: 240}, 320, 200, 400, 240},
		{"ratio 2", ui.Size{Width: 200, Height: 200}, 2, ui.Size{Width: 220, Height: 210}, 400, 400, 440, 420},
		{"ratio 1.5, rounded up", ui.Size{Width: 201, Height: 201}, 1.5, ui.Size{Width: 201, Height: 201}, 302, 302, 302, 302},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, newCounter(nil), tt.size, offscreen.ViewOpts{DevicePixelRatio: tt.ratio})
			h.step(0)
			if got := h.last().img.Rect; got != image.Rect(0, 0, tt.wantW, tt.wantH) {
				t.Fatalf("the first frame is %v, want %dx%d", got, tt.wantW, tt.wantH)
			}
			before := len(h.received())
			if err := h.view.SetSize(tt.resizeTo); err != nil {
				t.Fatal(err)
			}
			h.step(pace)
			fr := h.received()
			if tt.resizeTo == tt.size {
				if len(fr) != before {
					t.Fatalf("the same size made %d new frames", len(fr)-before)
				}
				return
			}
			got := fr[len(fr)-1]
			if got.img.Rect != image.Rect(0, 0, tt.thenW, tt.thenH) {
				t.Errorf("after SetSize the frame is %v, want %dx%d", got.img.Rect, tt.thenW, tt.thenH)
			}
			if len(got.damage) != 1 || got.damage[0] != got.img.Rect {
				t.Errorf("after SetSize the damage is %v, want the whole image", got.damage)
			}
			ref := gluttertest.New(t, gluttertest.Opts{Size: tt.resizeTo, DevicePixelRatio: max(tt.ratio, 1)})
			ref.PumpWidget(newCounter(nil))
			samePixels(t, "resized view and Tester.Screenshot", got.img, ref.Screenshot())
		})
	}
}

func TestViewSharesTheTextEngine(t *testing.T) {
	h := newHarness(t, newCounter(nil), ui.Size{Width: 100, Height: 100})
	got := h.view.Binding().PlatformDispatcher().TextEngine()
	want := h.parent.Binding.PlatformDispatcher().TextEngine()
	if got == nil || got != want {
		t.Errorf("the view's text engine is %v, want the parent's %v", got, want)
	}
}

func TestViewFramesArePaced(t *testing.T) {
	tests := []struct {
		name      string
		rate      int
		bumps     int
		maxFrames int
	}{
		{"default 30 a second", 0, 100, 3},
		{"10 a second", 10, 100, 3},
		{"one bump", 0, 1, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			probe := &counterProbe{}
			h := newHarness(t, newCounter(probe), ui.Size{Width: 320, Height: 200}, offscreen.ViewOpts{MaxFrameRate: tt.rate})
			h.step(0)
			for range tt.bumps {
				probe.bump()
				h.parent.Pump() // no time passes: the frame waits for the timer
			}
			h.view.WaitIdle()
			// Let the fake time pass, in steps shorter than a frame.
			for range 20 {
				h.step(10 * time.Millisecond)
			}
			h.step(time.Second)
			frames := h.received()
			if len(frames) > tt.maxFrames {
				t.Errorf("%d bumps made %d frames, want at most %d", tt.bumps, len(frames), tt.maxFrames)
			}
			// The last frame shows the last count.
			ref := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 320, Height: 200}})
			refProbe := &counterProbe{}
			ref.PumpWidget(newCounter(refProbe))
			for range tt.bumps {
				refProbe.bump()
			}
			ref.Pump()
			samePixels(t, "the last frame", frames[len(frames)-1].img, ref.Screenshot())
		})
	}
}

func TestViewPacingNeverMakesTwoFramesForOneRequest(t *testing.T) {
	probe := &counterProbe{}
	h := newHarness(t, newCounter(probe), ui.Size{Width: 320, Height: 200})
	h.step(0)
	probe.bump()
	h.parent.Pump() // a frame is asked for, the timer waits
	if !h.view.Scheduled() {
		t.Fatal("no frame is waiting for the timer")
	}
	probe.bump() // asked again while one waits
	h.parent.Pump()
	h.step(time.Second)
	if n := len(h.received()); n != 2 {
		t.Errorf("two requests made %d frames in all, want 2 (the first one and one more)", n)
	}
	if h.view.Scheduled() {
		t.Error("a frame is still scheduled")
	}
}

func TestViewBlinkingCaretDamagesLittle(t *testing.T) {
	size := ui.Size{Width: 400, Height: 300}
	root := scaffoldApp(widgets.NewColumn([]widgets.Widget{
		widgets.NewSizedBox(material.NewTextField(material.TextFieldOpts{Autofocus: true, Controller: widgets.NewTextEditingController("caret here")}),
			widgets.SizedBoxOpts{Height: foundation.Some(64.0)}),
		widgets.NewExpanded(textBody()),
	}))
	h := newHarness(t, root, size)
	h.step(0)
	// The field takes the focus and animates; once that is over, only the caret changes.
	for range 3 {
		h.step(500 * time.Millisecond)
	}
	settled := len(h.received())
	for range 6 {
		h.step(500 * time.Millisecond)
	}
	frames := h.received()[settled:]
	if len(frames) != 6 {
		t.Fatalf("six blinks of the caret made %d frames, want 6", len(frames))
	}
	whole := size.Width * size.Height
	for _, f := range frames {
		a := area(f.damage)
		if a == 0 || float64(a) > whole/20 {
			t.Errorf("frame %d damaged %d pixels (%v) of %v", f.number, a, f.damage, whole)
		}
	}
}

func TestViewSlowCallbackDropsFrames(t *testing.T) {
	probe := &counterProbe{}
	gate := make(chan struct{})
	entered := make(chan struct{}, 1)
	h := newHarness(t, newCounter(probe), ui.Size{Width: 320, Height: 200})
	h.mu.Lock()
	h.onFrame = func(n int) {
		if n == 1 {
			entered <- struct{}{}
			<-gate
		}
	}
	h.mu.Unlock()
	h.parent.Pump()
	<-entered // the goroutine of the view is in the callback of frame 1
	for range 5 {
		probe.bump()
		h.parent.Pump(pace) // a frame of the framework each time; the goroutine is busy
	}
	waiting, _ := h.view.Held()
	if waiting != 1 {
		t.Errorf("%d scenes wait for the goroutine, want 1 (the newest)", waiting)
	}
	close(gate)
	h.view.WaitIdle()
	frames := h.received()
	if len(frames) != 2 || frames[0].number != 1 || frames[1].number != 6 {
		numbers := []int{}
		for _, f := range frames {
			numbers = append(numbers, f.number)
		}
		t.Fatalf("the callback got frames %v, want the first and the last (1 and 6)", numbers)
	}
	ref := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 320, Height: 200}})
	refProbe := &counterProbe{}
	ref.PumpWidget(newCounter(refProbe))
	for range 5 {
		refProbe.bump()
	}
	ref.Pump()
	samePixels(t, "the drawn frame", frames[1].img, ref.Screenshot())
}

func TestViewCloseWaitsForTheCallback(t *testing.T) {
	probe := &counterProbe{}
	gate := make(chan struct{})
	entered := make(chan struct{}, 1)
	var calls int
	var mu sync.Mutex
	h := newHarness(t, newCounter(probe), ui.Size{Width: 320, Height: 200})
	h.mu.Lock()
	h.onFrame = func(int) {
		mu.Lock()
		calls++
		mu.Unlock()
		entered <- struct{}{}
		<-gate
	}
	h.mu.Unlock()
	h.parent.Pump()
	<-entered
	// A frame waits for the goroutine, and a timer waits for the next one.
	probe.bump()
	h.parent.Pump(pace)
	probe.bump()
	h.parent.Pump()
	base := h.parent.Clock.Pending()

	closed := make(chan struct{})
	go func() {
		h.parent.Binding.UIThread().Adopt()
		h.view.Close()
		close(closed)
	}()
	for !offscreen.IsClosed(h.view) {
		runtime.Gosched()
	}
	select {
	case <-closed:
		t.Fatal("Close returned while the callback was running")
	default:
	}
	close(gate)
	<-closed
	h.parent.Binding.UIThread().Adopt()

	if got := h.parent.Clock.Pending(); got != base-1 {
		t.Errorf("%d timers wait after Close, want %d (the pacing timer is stopped)", got, base-1)
	}
	mu.Lock()
	callsAtClose := calls
	mu.Unlock()
	// Nothing runs after Close: not a frame, not the callback, and nothing was posted.
	h.parent.Pump(time.Second)
	h.parent.Pump(time.Second)
	mu.Lock()
	defer mu.Unlock()
	if calls != callsAtClose || calls != 1 {
		t.Errorf("the callback was called %d times, %d at Close, want 1", calls, callsAtClose)
	}
	if h.view.Scheduled() {
		t.Error("a frame is scheduled after Close")
	}
	if waiting, _ := h.view.Held(); waiting != 0 {
		t.Errorf("%d scenes are held after Close", waiting)
	}
}

func TestViewFramePostedBeforeCloseDoesNothing(t *testing.T) {
	probe := &counterProbe{}
	h := newHarness(t, newCounter(probe), ui.Size{Width: 320, Height: 200})
	// The first frame is posted and has not run.
	if !h.view.Scheduled() {
		t.Fatal("the first frame was not asked for")
	}
	h.view.Close()
	h.parent.Pump(time.Second)
	h.parent.Pump(time.Second)
	if n := len(h.received()); n != 0 {
		t.Errorf("a frame posted before Close made %d frames after it", n)
	}
	h.view.Close() // twice is fine
}

func TestViewMemoryDoesNotGrow(t *testing.T) {
	probe := &counterProbe{}
	h := newHarness(t, newLightCounter(probe), ui.Size{Width: 320, Height: 200})
	h.step(0)
	var atTwenty int
	for i := 1; i <= 1000; i++ {
		probe.bump()
		h.step(pace)
		waiting, copies := h.view.Held()
		if waiting > 1 {
			t.Fatalf("frame %d: %d scenes wait", i, waiting)
		}
		if i == 20 {
			atTwenty = copies
		}
		if i == 1000 {
			t.Logf("copies at 20: %d, at 1000: %d", atTwenty, copies)
		}
		if i > 20 && copies > 2*atTwenty+4 {
			t.Fatalf("frame %d: the detacher keeps %d copies, after %d at frame 20", i, copies, atTwenty)
		}
	}
	if n := len(h.received()); n < 900 {
		t.Errorf("only %d of 1000 frames reached the callback", n)
	}
}

func TestViewPanicInCallbackIsReported(t *testing.T) {
	probe := &counterProbe{}
	h := newHarness(t, newCounter(probe), ui.Size{Width: 320, Height: 200})
	h.mu.Lock()
	h.onFrame = func(n int) {
		if n == 1 {
			panic("the callback broke")
		}
	}
	h.mu.Unlock()
	h.step(0)
	h.parent.Pump() // the report was posted to the parent's UI thread
	err := h.parent.TakeError()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("the parent got %v, want a tagged error", err)
	}
	if appErr.Tag != foundation.CallbackPanicked {
		t.Errorf("the tag is %s, want %s", appErr.Tag, foundation.CallbackPanicked)
	}
	if appErr.Metadata["panic"] != "the callback broke" {
		t.Errorf("the metadata is %v", appErr.Metadata)
	}
	probe.bump()
	h.step(pace)
	if fr := h.received(); len(fr) != 2 || fr[1].number != 2 {
		t.Errorf("the view stopped after the panic: %d frames", len(fr))
	}
}

func TestViewLeavesNoGoroutine(t *testing.T) {
	parent := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 320, Height: 200}})
	parent.PumpWidget(widgets.NewSizedBoxShrink(nil))
	before := runtime.NumGoroutine()
	frames := make(chan struct{}, 8)
	v, err := offscreen.NewView(parent.Binding, ui.Size{Width: 320, Height: 200}, newCounter(nil), func(offscreen.Frame) { frames <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}
	for len(frames) == 0 {
		parent.Pump()
		runtime.Gosched()
	}
	v.Close()
	v.Close()
	// The goroutine of the view returned before Close did; the runtime may need a moment to
	// take it off its count.
	for range 1000 {
		if runtime.NumGoroutine() <= before {
			break
		}
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("%d goroutines before NewView, %d after Close", before, after)
	}
}

func TestViewErrors(t *testing.T) {
	parent := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 320, Height: 200}})
	parent.PumpWidget(widgets.NewSizedBoxShrink(nil))
	size := ui.Size{Width: 100, Height: 100}
	noop := func(offscreen.Frame) {}
	tests := []struct {
		name  string
		make  func() error
		route string
	}{
		{"nil parent", func() error { _, err := offscreen.NewView(nil, size, newCounter(nil), noop); return err }, "NewView->newView.ViewParentNil"},
		{"nil root", func() error { _, err := offscreen.NewView(parent.Binding, size, nil, noop); return err }, "NewView->newView.ViewRootNil"},
		{"nil callback", func() error { _, err := offscreen.NewView(parent.Binding, size, newCounter(nil), nil); return err }, "NewView->newView.ViewCallbackNil"},
		{"zero size", func() error {
			_, err := offscreen.NewView(parent.Binding, ui.Size{}, newCounter(nil), noop)
			return err
		}, "NewView->newView.ViewSizeInvalid"},
		{"negative width", func() error {
			_, err := offscreen.NewView(parent.Binding, ui.Size{Width: -1, Height: 10}, newCounter(nil), noop)
			return err
		}, "NewView->newView.ViewSizeInvalid"},
		{"too large", func() error {
			_, err := offscreen.NewView(parent.Binding, ui.Size{Width: 1 << 20, Height: 10}, newCounter(nil), noop)
			return err
		}, "NewView->newView.ViewSizeInvalid"},
		{"negative ratio", func() error {
			_, err := offscreen.NewView(parent.Binding, size, newCounter(nil), noop, offscreen.ViewOpts{DevicePixelRatio: -1})
			return err
		}, "NewView->newView.ViewOptionInvalid"},
		{"negative rate", func() error {
			_, err := offscreen.NewView(parent.Binding, size, newCounter(nil), noop, offscreen.ViewOpts{MaxFrameRate: -1})
			return err
		}, "NewView->newView.ViewOptionInvalid"},
		{"SetSize not positive", func() error {
			v, err := offscreen.NewView(parent.Binding, size, newCounter(nil), noop)
			if err != nil {
				return err
			}
			defer v.Close()
			return v.SetSize(ui.Size{Width: 10})
		}, "View.SetSize.ViewSizeInvalid"},
		{"SetSize after Close", func() error {
			v, err := offscreen.NewView(parent.Binding, size, newCounter(nil), noop)
			if err != nil {
				return err
			}
			v.Close()
			return v.SetSize(size)
		}, "View.SetSize.ViewClosed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent.Binding.UIThread().Adopt()
			err := tt.make()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), tt.route)
			}
		})
	}
	t.Run("two options", func(t *testing.T) {
		defer func() {
			appErr, ok := errors.FromError(asError(recover()))
			if !ok || appErr.Tag != "MultipleOptions" {
				t.Errorf("recovered %v, want the tag MultipleOptions", appErr)
			}
		}()
		_, _ = offscreen.NewView(parent.Binding, size, newCounter(nil), noop, offscreen.ViewOpts{}, offscreen.ViewOpts{})
	})
}

func asError(r any) error {
	err, _ := r.(error)
	return err
}

// TestViewRace runs the framework on this goroutine, with text that changes and is laid
// out again each frame, while the goroutine of the view draws; run it with -race.
func TestViewRace(t *testing.T) {
	probe := &counterProbe{}
	h := newHarness(t, newLightCounter(probe), ui.Size{Width: 300, Height: 200})
	for i := range 400 {
		probe.bump()
		h.parent.Pump(pace)
		if i%7 == 0 {
			if err := h.view.SetSize(ui.Size{Width: float64(250 + i), Height: 200}); err != nil {
				t.Fatal(err)
			}
		}
	}
	h.view.WaitIdle()
	if len(h.received()) == 0 {
		t.Fatal("no frame")
	}
}
