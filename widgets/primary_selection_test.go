package widgets_test

import (
	"context"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gestures"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are written for Glutter: the primary selection and the rich
// clipboard are the Linux desktop's, and Flutter has neither.

var middleClick = gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse, Buttons: foundation.Some(gestures.TertiaryButton)}

func TestSelectingTextInAFieldSetsThePrimarySelection(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformLinux)
	f := newSelectionFixture(t, "hello brave world")
	f.click(f.pointAt(8), 2) // double click selects "brave"
	if text, ok := f.ts.PrimaryText(); !ok || text != "brave" {
		t.Fatalf("primary = %q (%v), want brave", text, ok)
	}
	// The clipboard is not touched.
	if _, ok := f.ts.ClipboardText(); ok {
		t.Error("selecting set the clipboard")
	}
	// A drag sets it once, at the end of the frame, with the final selection.
	g := f.ts.StartGesture(f.pointAt(0), gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
	f.ts.Pump()
	g.MoveTo(f.pointAt(10))
	f.ts.Pump()
	g.Up()
	f.ts.Pump(500 * time.Millisecond)
	want := f.controller.Text()[f.controller.Selection().Start():f.controller.Selection().End()]
	if text, _ := f.ts.PrimaryText(); text != want || len(want) < 9 {
		t.Errorf("primary = %q, want the selected %q", text, want)
	}
	// A caret leaves the primary selection as it was.
	f.click(f.pointAt(3), 1)
	if text, _ := f.ts.PrimaryText(); text != want {
		t.Errorf("primary after a caret = %q, want %q", text, want)
	}
}

func TestSelectingTextDoesNotSetThePrimarySelectionOffLinux(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformMacOS)
	f := newSelectionFixture(t, "hello brave world")
	f.click(f.pointAt(8), 2)
	if f.controller.Selection().IsCollapsed() {
		t.Fatal("the word was not selected")
	}
	if text, ok := f.ts.PrimaryText(); ok {
		t.Errorf("primary = %q, want none", text)
	}
}

func TestObscuredTextIsNotOnThePrimarySelection(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformLinux)
	ts := gluttertest.New(t)
	controller, focus := NewTextEditingController("secret"), NewFocusNode()
	t.Cleanup(func() { controller.Dispose(); focus.Dispose() })
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewEditableText(controller, focus, plainEditableStyle(), ui.NewColor(0xFF000000), ui.NewColor(0xFF888888),
		EditableTextOpts{ObscureText: true})}))
	state := gluttertest.StateOf[*EditableTextState](ts, gluttertest.ByType[*EditableText]())
	state.RequestKeyboard()
	ts.Pump()
	state.SelectAll(services.SelectionChangedCauseKeyboard)
	ts.Pump()
	ts.Pump(100 * time.Millisecond)
	if text, ok := ts.PrimaryText(); ok {
		t.Errorf("primary = %q, want none", text)
	}
}

func TestMiddleClickPastesThePrimarySelection(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformLinux)
	f := newSelectionFixture(t, "hello world")
	f.ts.SetPrimaryText("BRAVE ")
	at := f.pointAt(6)
	f.ts.TapAt(at, middleClick)
	f.ts.Pump(500 * time.Millisecond)
	text := f.controller.Text()
	// The text went in at the pointer, between "hello " and "world".
	if text != "hello BRAVE world" && text != "hello wBRAVE orld" {
		t.Errorf("text = %q, want the primary selection pasted at the pointer", text)
	}
	if sel := f.controller.Selection(); !sel.IsCollapsed() || sel.BaseOffset != len("hello BRAVE ") && sel.BaseOffset != len("hello wBRAVE ") {
		t.Errorf("selection = %v, want a caret after the pasted text", sel)
	}
	// The clipboard stays as it was.
	if _, ok := f.ts.ClipboardText(); ok {
		t.Error("the middle click touched the clipboard")
	}
}

func TestMiddleClickWithoutAPrimarySelectionDoesNothing(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformLinux)
	f := newSelectionFixture(t, "hello world")
	f.ts.TapAt(f.pointAt(6), middleClick)
	f.ts.Pump(500 * time.Millisecond)
	if f.controller.Text() != "hello world" {
		t.Errorf("text = %q", f.controller.Text())
	}
}

func TestMiddleClickDoesNothingOffLinux(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformWindows)
	f := newSelectionFixture(t, "hello world")
	f.ts.SetPrimaryText("X")
	f.ts.TapAt(f.pointAt(6), middleClick)
	f.ts.Pump(500 * time.Millisecond)
	if f.controller.Text() != "hello world" {
		t.Errorf("text = %q", f.controller.Text())
	}
}

func TestMiddleClickDoesNotPasteIntoAReadOnlyField(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformLinux)
	ts := gluttertest.New(t)
	controller, focus := NewTextEditingController("read only"), NewFocusNode()
	t.Cleanup(func() { controller.Dispose(); focus.Dispose() })
	key := NewGlobalKey[*EditableTextState]()
	builder := NewTextSelectionGestureDetectorBuilder(&testSelectionField{key: key})
	field := NewEditableText(controller, focus, plainEditableStyle(), ui.NewColor(0xFF000000), ui.NewColor(0xFF888888),
		EditableTextOpts{Key: key, ReadOnly: true, RendererIgnoresPointer: true})
	ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewAlign(NewSizedBox(
		builder.BuildGestureDetector(field, TextSelectionGestureDetectorBuildOpts{Behavior: rendering.HitTestBehaviorTranslucent}),
		SizedBoxOpts{Width: foundation.Some(300.0), Height: foundation.Some(100.0)}), AlignOpts{Alignment: painting.AlignmentTopLeft})}))
	ts.SetPrimaryText("X")
	ts.TapAt(ts.Center(gluttertest.ByType[*EditableText]()), middleClick)
	ts.Pump(500 * time.Millisecond)
	if controller.Text() != "read only" {
		t.Errorf("text = %q", controller.Text())
	}
}

func TestSelectableRegionSetsThePrimarySelection(t *testing.T) {
	usePlatform(t, foundation.TargetPlatformLinux)
	ts := gluttertest.New(t)
	pumpSelectable(ts, NewCenter(ltrText("How are you")))
	p := paragraphOf(ts, "How are you")
	g := ts.StartGesture(paragraphOffsetToPosition(p, 2), mouseKind)
	ts.Pump()
	g.MoveTo(paragraphOffsetToPosition(p, 6))
	ts.Pump()
	if _, ok := ts.PrimaryText(); ok {
		t.Error("the primary selection was set before the selection was final")
	}
	g.Up()
	ts.Pump()
	if text, ok := ts.PrimaryText(); !ok || text != "w ar" {
		t.Errorf("primary = %q (%v), want %q", text, ok, "w ar")
	}
}

func TestClipboardHoldsImagesAndFiles(t *testing.T) {
	ts := gluttertest.New(t)
	clip := ts.Binding.Clipboard()
	ctx := context.Background()
	png := []byte("\x89PNG fake")
	uris := []string{"file:///home/me/a.txt"}

	var setErr error
	clip.SetData(ctx, services.ClipboardData{Text: "a.txt", PNG: png, URIs: uris}, func(err error) { setErr = err })
	ts.FlushPosted()
	if setErr != nil {
		t.Fatal(setErr)
	}
	if string(ts.ClipboardPNG()) != string(png) || len(ts.ClipboardURIs()) != 1 {
		t.Fatalf("host clipboard: %q %v", ts.ClipboardPNG(), ts.ClipboardURIs())
	}
	get := func(format string) *services.ClipboardData {
		var got *services.ClipboardData
		var gerr error
		clip.GetData(ctx, format, func(d *services.ClipboardData, err error) { got, gerr = d, err })
		ts.FlushPosted()
		if gerr != nil {
			t.Fatalf("GetData(%s): %v", format, gerr)
		}
		return got
	}
	if d := get(services.ClipboardKImagePNG); d == nil || string(d.PNG) != string(png) {
		t.Errorf("image = %+v", d)
	}
	if d := get(services.ClipboardKURIList); d == nil || len(d.URIs) != 1 || d.URIs[0] != uris[0] {
		t.Errorf("uris = %+v", d)
	}
	if d := get(services.ClipboardKTextPlain); d == nil || d.Text != "a.txt" {
		t.Errorf("text = %+v", d)
	}
	// Another application copies files only: no text, no image.
	ts.ClearClipboard()
	ts.SetClipboardURIs(uris)
	if d := get(services.ClipboardKURIList); d == nil || len(d.URIs) != 1 {
		t.Errorf("uris of another app = %+v", d)
	}
	if d := get(services.ClipboardKImagePNG); d != nil {
		t.Errorf("image of another app = %+v, want none", d)
	}
	if d := get(services.ClipboardKTextPlain); d != nil {
		t.Errorf("text of another app = %+v, want none", d)
	}
	if d := get("application/x-unknown"); d != nil {
		t.Errorf("unknown format = %+v", d)
	}
}

func TestPrimaryClipboardData(t *testing.T) {
	ts := gluttertest.New(t)
	clip := ts.Binding.Clipboard()
	ctx := context.Background()
	var setErr error
	clip.SetPrimaryData(ctx, services.NewClipboardData("selected"), func(err error) { setErr = err })
	ts.FlushPosted()
	if text, ok := ts.PrimaryText(); setErr != nil || !ok || text != "selected" {
		t.Fatalf("primary = %q (%v), err %v", text, ok, setErr)
	}
	var got *services.ClipboardData
	clip.GetPrimaryData(ctx, services.ClipboardKTextPlain, func(d *services.ClipboardData, err error) { got = d })
	ts.FlushPosted()
	if got == nil || got.Text != "selected" {
		t.Errorf("GetPrimaryData = %+v", got)
	}
	clip.GetPrimaryData(ctx, "application/x-unknown", func(d *services.ClipboardData, err error) { got = d })
	if got != nil {
		t.Errorf("unknown format = %+v", got)
	}
	// The system clipboard is another selection.
	if _, ok := ts.ClipboardText(); ok {
		t.Error("SetPrimaryData set the clipboard")
	}
}

func TestClipboardWithoutABackendFailsTheRichCalls(t *testing.T) {
	clip := services.NewClipboard(nil)
	check := func(name string, err error) {
		t.Helper()
		appErr, ok := errors.FromError(err)
		if !ok || !appErr.HasRoute("ClipboardUnavailable") {
			t.Errorf("%s: err = %v, want ClipboardUnavailable", name, err)
		}
	}
	ctx := context.Background()
	clip.SetPrimaryData(ctx, services.NewClipboardData("x"), func(err error) { check("SetPrimaryData", err) })
	clip.SetData(ctx, services.ClipboardData{PNG: []byte("p")}, func(err error) { check("SetData", err) })
	clip.GetPrimaryData(ctx, services.ClipboardKTextPlain, func(_ *services.ClipboardData, err error) { check("GetPrimaryData", err) })
	clip.GetData(ctx, services.ClipboardKImagePNG, func(_ *services.ClipboardData, err error) { check("GetData", err) })
}
