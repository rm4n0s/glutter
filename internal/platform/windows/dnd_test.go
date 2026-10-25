package windows

import (
	"bytes"
	"io"
	"log/slog"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/internal/platform/windows/ole"
	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

const (
	testPID    = 4242
	fmtURLW    = 0xC010
	fmtURL     = 0xC011
	fmtPNG     = 0xC012
	fmtMarker1 = 0xC020 // the marker of drag request 1 of this process
	fmtOther   = 0xC030 // the marker of a drag of another process
)

var testNames = map[uint32]string{
	fmtURLW:    win32.FormatNameURLW,
	fmtURL:     win32.FormatNameURL,
	fmtPNG:     "PNG",
	fmtMarker1: ole.LocalMarker(testPID, 1),
	fmtOther:   ole.LocalMarker(testPID+1, 9),
	0xC040:     "Something Else",
}

func testRegister(name string) (uint32, error) {
	for f, n := range testNames {
		if n == name {
			return f, nil
		}
	}
	return 0xC0FF, nil
}

// fakeSource is a data object with formats and bytes.
type fakeSource struct {
	formats []uint32
	data    map[uint32][]byte
	// failFormats fail to read; slow advances the clock by this much at each read.
	failFormats map[uint32]bool
	clock       *fakeClock
	slow        time.Duration
	formatsErr  error
	reads       []uint32
	closed      int
}

func (f *fakeSource) Formats() ([]uint32, error) { return f.formats, f.formatsErr }
func (f *fakeSource) Read(format uint32) ([]byte, error) {
	f.reads = append(f.reads, format)
	if f.clock != nil {
		f.clock.t = f.clock.t.Add(f.slow)
	}
	if f.failFormats[format] {
		return nil, errors.New(tagTestReadFailed, "the read failed")
	}
	return f.data[format], nil
}
func (f *fakeSource) Close() { f.closed++ }

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

// dropRecorder is a handler that records the DropEvents and the end of the drags, and answers the
// action in answer.
type dropRecorder struct {
	platform.NopHandler
	events []ui.DropEvent
	ends   []ui.DragResult
	answer ui.DropActions
}

func (h *dropRecorder) OnDrop(e ui.DropEvent) ui.DropActions {
	h.events = append(h.events, e)
	return h.answer
}
func (h *dropRecorder) OnDragEnd(r ui.DragResult) { h.ends = append(h.ends, r) }

func newDropWindow(h platform.Handler, clock *fakeClock) *dropWindow {
	return &dropWindow{
		view:    3,
		handler: func() platform.Handler { return h },
		toClient: func(pt win32.Point) (ui.Offset, bool) {
			return ui.Offset{Dx: float64(pt.X - 100), Dy: float64(pt.Y - 50)}, true
		},
		nameOf: func(f uint32) string { return testNames[f] },
		pid:    testPID, now: clock.now, budget: dropReadBudget,
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func utf16Bytes(s string) []byte { return textBytes(textToClipboard(s)) }

func TestDropTargetEnterOverLeave(t *testing.T) {
	all := uint32(win32.DropEffectCopy | win32.DropEffectMove | win32.DropEffectLink)
	src := &fakeSource{formats: []uint32{win32.CFHDrop, win32.CFUnicodeText, win32.CFText, fmtURLW, fmtURL, fmtPNG, 0xC040, win32.CFDIB, fmtMarker1}}
	h := &dropRecorder{answer: ui.DropActionCopy}
	d := newDropWindow(h, &fakeClock{t: time.Unix(0, 0)})

	// Enter: the types, the place, the actions and the local drag are reported; the effect is the action.
	if got := d.DragEnter(src, 0, win32.Point{X: 130, Y: 80}, all); got != win32.DropEffectCopy {
		t.Errorf("DragEnter effect = %#x, want copy", got)
	}
	// Over with a changed key state: shift narrows the offer to move; the framework accepts move.
	h.answer = ui.DropActionMove
	if got := d.DragOver(win32.MKShift, win32.Point{X: 140, Y: 90}, all); got != win32.DropEffectMove {
		t.Errorf("DragOver effect = %#x, want move", got)
	}
	d.DragLeave()
	d.DragLeave() // a second leave does nothing
	if got := d.DragOver(0, win32.Point{}, all); got != win32.DropEffectNone {
		t.Errorf("DragOver after the leave = %#x, want none", got)
	}

	wantTypes := []string{ole.MIMEURIList, ole.MIMEPlainUTF8, ole.MIMEPlain, ole.MIMEPNG, "Something Else"}
	want := []ui.DropEvent{
		{ViewID: 3, Phase: ui.DropPhaseEnter, Position: ui.Offset{Dx: 30, Dy: 30}, Types: wantTypes, Actions: ui.DropActionCopy | ui.DropActionMove | ui.DropActionLink, LocalID: 1},
		{ViewID: 3, Phase: ui.DropPhaseMove, Position: ui.Offset{Dx: 40, Dy: 40}, Types: wantTypes, Actions: ui.DropActionMove, LocalID: 1},
		{ViewID: 3, Phase: ui.DropPhaseLeave, Types: wantTypes, LocalID: 1},
	}
	if !reflect.DeepEqual(h.events, want) {
		t.Errorf("events:\n got %+v\nwant %+v", h.events, want)
	}
}

func TestDropTargetEffectsFollowTheFrameworkAndTheSource(t *testing.T) {
	tests := []struct {
		name    string
		answer  ui.DropActions
		allowed uint32
		want    uint32
	}{
		{"accepted copy", ui.DropActionCopy, win32.DropEffectCopy | win32.DropEffectMove, win32.DropEffectCopy},
		{"accepted move", ui.DropActionMove, win32.DropEffectCopy | win32.DropEffectMove, win32.DropEffectMove},
		{"refused", ui.DropActionNone, win32.DropEffectCopy, win32.DropEffectNone},
		{"accepted but the source does not allow it", ui.DropActionLink, win32.DropEffectCopy, win32.DropEffectNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dropRecorder{answer: tt.answer}
			d := newDropWindow(h, &fakeClock{})
			src := &fakeSource{formats: []uint32{win32.CFUnicodeText}}
			if got := d.DragEnter(src, 0, win32.Point{}, tt.allowed); got != tt.want {
				t.Errorf("DragEnter = %#x, want %#x", got, tt.want)
			}
			if got := d.Drop(src, 0, win32.Point{}, tt.allowed); got != tt.want {
				t.Errorf("Drop = %#x, want %#x", got, tt.want)
			}
		})
	}
}

func TestDropReadsTheTextAndTheFiles(t *testing.T) {
	files := dropFilesFromPaths([]string{`C:\Users\me\a b.txt`, `\\server\share\é.png`, `relative\x.txt`})
	tests := []struct {
		name     string
		formats  []uint32
		data     map[uint32][]byte
		wantText string
		wantURIs []string
	}{
		{"files", []uint32{win32.CFHDrop}, map[uint32][]byte{win32.CFHDrop: files}, "",
			[]string{"file:///C:/Users/me/a%20b.txt", "file://server/share/%C3%A9.png", "file:///relative/x.txt"}},
		{"text with CRLF", []uint32{win32.CFUnicodeText}, map[uint32][]byte{win32.CFUnicodeText: utf16Bytes("one\r\ntwo\rthree")}, "one\ntwo\nthree", nil},
		{"unicode text wins over ansi", []uint32{win32.CFText, win32.CFUnicodeText},
			map[uint32][]byte{win32.CFText: []byte("ansi\x00"), win32.CFUnicodeText: utf16Bytes("wide")}, "wide", nil},
		{"ansi text alone", []uint32{win32.CFText}, map[uint32][]byte{win32.CFText: []byte("caf\xe9\r\nx\x00junk")}, "café\nx", nil},
		{"oem text alone", []uint32{win32.CFOEMText}, map[uint32][]byte{win32.CFOEMText: []byte("abc\x00")}, "abc", nil},
		{"a URL from a browser: text and the URL", []uint32{win32.CFUnicodeText, fmtURLW},
			map[uint32][]byte{win32.CFUnicodeText: utf16Bytes("the page"), fmtURLW: ole.StringData("https://example.com/a?b=1", true)},
			"the page", []string{"https://example.com/a?b=1"}},
		{"a URL and no text: the URL is the text", []uint32{fmtURL},
			map[uint32][]byte{fmtURL: ole.StringData("http://example.com/", false)}, "http://example.com/", []string{"http://example.com/"}},
		{"the wide URL is preferred", []uint32{fmtURL, fmtURLW},
			map[uint32][]byte{fmtURL: ole.StringData("http://old/", false), fmtURLW: ole.StringData("http://new/", true)}, "http://new/", []string{"http://new/"}},
		{"files and a text together", []uint32{win32.CFHDrop, win32.CFUnicodeText},
			map[uint32][]byte{win32.CFHDrop: dropFilesFromPaths([]string{`D:\x.txt`}), win32.CFUnicodeText: utf16Bytes("x.txt")}, "x.txt", []string{"file:///D:/x.txt"}},
		{"a data object that offers nothing we know", []uint32{win32.CFDIB, 0xC040}, nil, "", nil},
		{"a damaged file list is left out", []uint32{win32.CFHDrop, win32.CFUnicodeText},
			map[uint32][]byte{win32.CFHDrop: {1, 2, 3}, win32.CFUnicodeText: utf16Bytes("still the text")}, "still the text", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dropRecorder{answer: ui.DropActionCopy}
			d := newDropWindow(h, &fakeClock{})
			src := &fakeSource{formats: tt.formats, data: tt.data}
			d.DragEnter(src, 0, win32.Point{}, win32.DropEffectCopy)
			d.Drop(src, 0, win32.Point{X: 101, Y: 51}, win32.DropEffectCopy)
			last := h.events[len(h.events)-1]
			if last.Phase != ui.DropPhaseDrop || last.Text != tt.wantText || !slices.Equal(last.URIs, tt.wantURIs) {
				t.Errorf("drop = %+v; want text %q, URIs %v", last, tt.wantText, tt.wantURIs)
			}
			if last.Position != (ui.Offset{Dx: 1, Dy: 1}) {
				t.Errorf("position = %v", last.Position)
			}
		})
	}
}

func TestDropTheAppRefusesIsAnswerWithNoEffect(t *testing.T) {
	h := &dropRecorder{answer: ui.DropActionCopy}
	d := newDropWindow(h, &fakeClock{})
	src := &fakeSource{formats: []uint32{win32.CFUnicodeText}, data: map[uint32][]byte{win32.CFUnicodeText: utf16Bytes("x")}}
	if got := d.DragEnter(src, 0, win32.Point{}, win32.DropEffectCopy); got != win32.DropEffectCopy {
		t.Fatalf("DragEnter = %#x", got)
	}
	h.answer = ui.DropActionNone // the target changed its mind by the time of the drop
	if got := d.Drop(src, 0, win32.Point{}, win32.DropEffectCopy); got != win32.DropEffectNone {
		t.Errorf("Drop = %#x, want none", got)
	}
	if h.events[len(h.events)-1].Phase != ui.DropPhaseDrop {
		t.Errorf("the app was not told of the drop: %+v", h.events)
	}
}

func TestDropAfterTheBudgetReadsNoMore(t *testing.T) {
	clock := &fakeClock{t: time.Unix(0, 0)}
	h := &dropRecorder{answer: ui.DropActionCopy}
	d := newDropWindow(h, clock)
	// Each read takes 3 s: the text is read at 0 s, the files would start at 3 s (in time), the URL at
	// 6 s (too late).
	src := &fakeSource{
		formats: []uint32{win32.CFUnicodeText, win32.CFHDrop, fmtURLW},
		data: map[uint32][]byte{
			win32.CFUnicodeText: utf16Bytes("t"), win32.CFHDrop: dropFilesFromPaths([]string{`C:\a`}), fmtURLW: ole.StringData("http://x/", true),
		},
		clock: clock, slow: 3 * time.Second,
	}
	d.Drop(src, 0, win32.Point{}, win32.DropEffectCopy)
	last := h.events[len(h.events)-1]
	if last.Text != "t" || !slices.Equal(last.URIs, []string{"file:///C:/a"}) {
		t.Errorf("drop = text %q, URIs %v", last.Text, last.URIs)
	}
	if want := []uint32{win32.CFUnicodeText, win32.CFHDrop}; !slices.Equal(src.reads, want) {
		t.Errorf("the source was read for %v, want %v", src.reads, want)
	}
}

func TestDropFormatsThatFailAreLeftOut(t *testing.T) {
	h := &dropRecorder{answer: ui.DropActionCopy}
	d := newDropWindow(h, &fakeClock{})
	src := &fakeSource{
		formats:     []uint32{win32.CFUnicodeText, win32.CFText},
		data:        map[uint32][]byte{win32.CFText: []byte("ansi")},
		failFormats: map[uint32]bool{win32.CFUnicodeText: true},
	}
	d.Drop(src, 0, win32.Point{}, win32.DropEffectCopy)
	if last := h.events[len(h.events)-1]; last.Text != "ansi" {
		t.Errorf("text = %q: the failing unicode text should fall back to the ansi one", last.Text)
	}
	// A source that cannot list its formats still drops, with nothing to read.
	h2 := &dropRecorder{answer: ui.DropActionCopy}
	d2 := newDropWindow(h2, &fakeClock{})
	d2.Drop(&fakeSource{formatsErr: errors.New(tagTestListFailed, "no list")}, 0, win32.Point{}, win32.DropEffectCopy)
	if last := h2.events[len(h2.events)-1]; last.Phase != ui.DropPhaseDrop || last.Text != "" || len(last.URIs) != 0 {
		t.Errorf("drop = %+v", last)
	}
}

func TestDropOfAnotherProcessIsNotLocal(t *testing.T) {
	h := &dropRecorder{answer: ui.DropActionCopy}
	d := newDropWindow(h, &fakeClock{})
	d.DragEnter(&fakeSource{formats: []uint32{win32.CFUnicodeText, fmtOther}}, 0, win32.Point{}, win32.DropEffectCopy)
	if got := h.events[0]; got.LocalID != 0 || slices.Contains(got.Types, testNames[fmtOther]) {
		t.Errorf("enter = %+v: the marker of another process is no local drag and no type", got)
	}
}

func TestDropWithNoHandlerAcceptsNothing(t *testing.T) {
	d := newDropWindow(nil, &fakeClock{})
	d.handler = func() platform.Handler { return nil }
	src := &fakeSource{formats: []uint32{win32.CFUnicodeText}}
	if got := d.DragEnter(src, 0, win32.Point{}, win32.DropEffectCopy); got != win32.DropEffectNone {
		t.Errorf("DragEnter = %#x", got)
	}
	d.DragLeave() // must not panic
	if got := d.Drop(src, 0, win32.Point{}, win32.DropEffectCopy); got != win32.DropEffectNone {
		t.Errorf("Drop = %#x", got)
	}
}

func TestAnsiText(t *testing.T) {
	tests := []struct{ in, want string }{
		{"abc", "abc"}, {"abc\x00def", "abc"}, {"a\r\nb", "a\nb"}, {"a\rb", "a\nb"}, {"caf\xe9", "café"}, {"", ""},
	}
	for _, tt := range tests {
		if got := ansiText([]byte(tt.in)); got != tt.want {
			t.Errorf("ansiText(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBuildDragData(t *testing.T) {
	req := ui.DragRequest{ID: 1, Text: "line 1\nline 2", URIs: []string{"file:///C:/x/a%20b.txt", "https://example.com/p", "file://server/s/c.txt", "mailto:a@b.c"}}
	set, err := buildDragData(req, testPID, testRegister)
	if err != nil {
		t.Fatal(err)
	}
	wantFormats := []uint32{win32.CFUnicodeText, win32.CFText, win32.CFHDrop, fmtURLW, fmtURL, fmtMarker1}
	if got := set.Formats(); !slices.Equal(got, wantFormats) {
		t.Fatalf("formats = %v, want %v", got, wantFormats)
	}
	text, _ := set.Get(win32.CFUnicodeText)
	if got := textFromClipboard(text); got != "line 1\nline 2" {
		t.Errorf("unicode text = %q", got)
	}
	ansi, _ := set.Get(win32.CFText)
	if !bytes.Equal(ansi, []byte("line 1\r\nline 2\x00")) {
		t.Errorf("ansi text = %q", ansi)
	}
	hdrop, _ := set.Get(win32.CFHDrop)
	paths, err := pathsFromDropFiles(hdrop)
	if err != nil || !slices.Equal(paths, []string{`C:\x\a b.txt`, `\\server\s\c.txt`}) {
		t.Errorf("files = %q, %v", paths, err)
	}
	urlw, _ := set.Get(fmtURLW)
	if got := ole.StringFromData(urlw, true); got != "https://example.com/p" {
		t.Errorf("wide URL = %q", got)
	}
	url, _ := set.Get(fmtURL)
	if got := ole.StringFromData(url, false); got != "https://example.com/p" {
		t.Errorf("ANSI URL = %q", got)
	}
}

func TestBuildDragDataOffersOnlyWhatTheRequestHas(t *testing.T) {
	tests := []struct {
		name string
		req  ui.DragRequest
		want []uint32
	}{
		{"only text", ui.DragRequest{ID: 1, Text: "x"}, []uint32{win32.CFUnicodeText, win32.CFText, fmtMarker1}},
		{"only files", ui.DragRequest{ID: 1, URIs: []string{"file:///C:/a"}}, []uint32{win32.CFHDrop, fmtMarker1}},
		{"only a link", ui.DragRequest{ID: 1, URIs: []string{"https://x/"}}, []uint32{fmtURLW, fmtURL, fmtMarker1}},
		{"nothing: the marker alone", ui.DragRequest{ID: 1}, []uint32{fmtMarker1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := buildDragData(tt.req, testPID, testRegister)
			if err != nil {
				t.Fatal(err)
			}
			if got := set.Formats(); !slices.Equal(got, tt.want) {
				t.Errorf("formats = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildDragDataFailsWhenAFormatCannotBeRegistered(t *testing.T) {
	failing := func(string) (uint32, error) { return 0, errors.New(tagTestRegisterFailed, "no") }
	_, err := buildDragData(ui.DragRequest{ID: 1, Text: "x"}, testPID, failing)
	assertTagRoute(t, err, tagTestRegisterFailed, "TestBuildDragDataFailsWhenAFormatCannotBeRegistered")
	if appErr, _ := errors.FromError(err); appErr != nil && appErr.Route() == "" {
		t.Error("the error lost its route")
	}
}

// ---- the drags the app starts ----

// runnerFixture is a dragRunner over fakes: the posted functions wait in posted until a test runs them.
type runnerFixture struct {
	r        *dragRunner
	buttons  uint32
	posted   []func()
	loops    []loopCall
	released []ui.ViewID
	ends     []ui.DragResult
	// the answer of the loop
	effect  uint32
	dropped bool
	loopErr error
	// onLoop runs inside the loop (the test plays the part of OLE).
	onLoop func(m *ole.SourceMachine)
}

type loopCall struct {
	view    ui.ViewID
	data    *ole.DataSet
	allowed uint32
	buttons uint32
}

func newRunner() *runnerFixture {
	f := &runnerFixture{buttons: win32.MKLButton}
	f.r = &dragRunner{
		pid: testPID, log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		buttons:  func() uint32 { return f.buttons },
		register: testRegister,
		post:     func(fn func()) { f.posted = append(f.posted, fn) },
		loop: func(view ui.ViewID, data *ole.DataSet, m *ole.SourceMachine, allowed uint32) (uint32, bool, error) {
			f.loops = append(f.loops, loopCall{view, data, allowed, m.Buttons})
			if f.onLoop != nil {
				f.onLoop(m)
			}
			return f.effect, f.dropped, f.loopErr
		},
		released: func(v ui.ViewID) { f.released = append(f.released, v) },
		end:      func(r ui.DragResult) { f.ends = append(f.ends, r) },
	}
	return f
}

func (f *runnerFixture) runPosted() {
	posted := f.posted
	f.posted = nil
	for _, fn := range posted {
		fn()
	}
}

func TestDragStartsInAPostedFunctionAndReportsTheEnd(t *testing.T) {
	tests := []struct {
		name    string
		actions ui.DropActions
		effect  uint32
		dropped bool
		want    ui.DropActions
		allowed uint32
	}{
		{"dropped as a copy", ui.DropActionCopy, win32.DropEffectCopy, true, ui.DropActionCopy, win32.DropEffectCopy},
		{"dropped as a move", ui.DropActionCopy | ui.DropActionMove, win32.DropEffectMove, true, ui.DropActionMove, win32.DropEffectCopy | win32.DropEffectMove},
		{"no actions means copy", ui.DropActionNone, win32.DropEffectCopy, true, ui.DropActionCopy, win32.DropEffectCopy},
		{"dropped where nothing takes it", ui.DropActionCopy, win32.DropEffectNone, true, ui.DropActionNone, win32.DropEffectCopy},
		{"cancelled", ui.DropActionCopy, win32.DropEffectCopy, false, ui.DropActionNone, win32.DropEffectCopy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunner()
			f.effect, f.dropped = tt.effect, tt.dropped
			if err := f.r.Start(7, ui.DragRequest{ID: 5, Text: "x", Actions: tt.actions}); err != nil {
				t.Fatal(err)
			}
			if len(f.loops) != 0 || len(f.ends) != 0 {
				t.Fatal("the loop ran before the handler that asked had returned")
			}
			if len(f.posted) != 1 || !f.r.Active() {
				t.Fatalf("posted = %d, active = %v", len(f.posted), f.r.Active())
			}
			f.runPosted()
			if len(f.loops) != 1 || f.loops[0].view != 7 || f.loops[0].allowed != tt.allowed || f.loops[0].buttons != win32.MKLButton {
				t.Errorf("loops = %+v", f.loops)
			}
			if !slices.Equal(f.released, []ui.ViewID{7}) {
				t.Errorf("released = %v: the pointer is lifted once, for the window of the drag", f.released)
			}
			if want := []ui.DragResult{{ID: 5, Action: tt.want}}; !reflect.DeepEqual(f.ends, want) {
				t.Errorf("ends = %+v, want %+v", f.ends, want)
			}
			if f.r.Active() {
				t.Error("the drag is still active after it ended")
			}
		})
	}
}

func TestDragStartRefusals(t *testing.T) {
	t.Run("a drag is going on", func(t *testing.T) {
		f := newRunner()
		if err := f.r.Start(1, ui.DragRequest{ID: 1}); err != nil {
			t.Fatal(err)
		}
		err := f.r.Start(1, ui.DragRequest{ID: 2})
		assertTagRoute(t, err, "DragInProgress", "dragRunner.Start.DragInProgress")
	})
	for _, tt := range []struct {
		name    string
		buttons uint32
	}{{"no button", 0}, {"only shift", win32.MKShift}, {"only control and alt", win32.MKControl | win32.MKAlt}} {
		t.Run(tt.name, func(t *testing.T) {
			f := newRunner()
			f.buttons = tt.buttons
			err := f.r.Start(1, ui.DragRequest{ID: 1})
			assertTagRoute(t, err, "DragNoButton", "dragRunner.Start.DragNoButton")
			if f.r.Active() || len(f.posted) != 0 {
				t.Error("a refused drag left a session or a posted function")
			}
		})
	}
	t.Run("a format cannot be registered", func(t *testing.T) {
		f := newRunner()
		f.r.register = func(string) (uint32, error) { return 0, errors.New(tagTestRegisterFailed, "no") }
		err := f.r.Start(1, ui.DragRequest{ID: 1})
		assertTagRoute(t, err, tagTestRegisterFailed, "dragRunner.Start->")
		if f.r.Active() {
			t.Error("a refused drag left a session")
		}
	})
	t.Run("the next drag can start after the first ended", func(t *testing.T) {
		f := newRunner()
		if err := f.r.Start(1, ui.DragRequest{ID: 1}); err != nil {
			t.Fatal(err)
		}
		f.runPosted()
		if err := f.r.Start(1, ui.DragRequest{ID: 2}); err != nil {
			t.Errorf("the second drag: %v", err)
		}
	})
}

func TestDragWhoseButtonWasReleasedBeforeTheLoopEndsWithoutOne(t *testing.T) {
	f := newRunner()
	if err := f.r.Start(1, ui.DragRequest{ID: 4, Text: "x"}); err != nil {
		t.Fatal(err)
	}
	f.buttons = 0 // the button went up before the posted function ran
	f.runPosted()
	if len(f.loops) != 0 {
		t.Error("DoDragDrop would run with no button down")
	}
	if want := []ui.DragResult{{ID: 4}}; !reflect.DeepEqual(f.ends, want) {
		t.Errorf("ends = %+v, want %+v", f.ends, want)
	}
	if !slices.Equal(f.released, []ui.ViewID{1}) {
		t.Errorf("released = %v", f.released)
	}
}

func TestDragCancelledByEscapeOrTheWindowClosing(t *testing.T) {
	t.Run("escape in the loop", func(t *testing.T) {
		f := newRunner()
		f.onLoop = func(m *ole.SourceMachine) {
			// OLE asks: the button is down, then Escape is pressed.
			if d := m.Query(false, win32.MKLButton); d != ole.Continue {
				t.Errorf("first query = %v", d)
			}
			if d := m.Query(true, win32.MKLButton); d != ole.Cancel {
				t.Errorf("escape query = %v", d)
			}
		}
		f.effect, f.dropped = win32.DropEffectCopy, false // OLE reports the cancel
		if err := f.r.Start(1, ui.DragRequest{ID: 9, Text: "x"}); err != nil {
			t.Fatal(err)
		}
		f.runPosted()
		if want := []ui.DragResult{{ID: 9}}; !reflect.DeepEqual(f.ends, want) {
			t.Errorf("ends = %+v: a cancelled drag is not accepted", f.ends)
		}
	})
	t.Run("the window closes while the loop runs", func(t *testing.T) {
		f := newRunner()
		f.onLoop = func(m *ole.SourceMachine) {
			f.r.WindowClosed(1)
			if d := m.Query(false, win32.MKLButton); d != ole.Cancel {
				t.Errorf("after the window closed, query = %v", d)
			}
		}
		if err := f.r.Start(1, ui.DragRequest{ID: 9}); err != nil {
			t.Fatal(err)
		}
		f.runPosted()
		if len(f.ends) != 1 || f.ends[0].Action != ui.DropActionNone {
			t.Errorf("ends = %+v", f.ends)
		}
	})
	t.Run("another window closing changes nothing", func(t *testing.T) {
		f := newRunner()
		if err := f.r.Start(1, ui.DragRequest{ID: 9}); err != nil {
			t.Fatal(err)
		}
		f.r.WindowClosed(2)
		f.dropped, f.effect = true, win32.DropEffectCopy
		f.runPosted()
		if len(f.loops) != 1 || len(f.ends) != 1 || f.ends[0].Action != ui.DropActionCopy {
			t.Errorf("loops = %d, ends = %+v", len(f.loops), f.ends)
		}
	})
	t.Run("the window closes before the loop ran", func(t *testing.T) {
		f := newRunner()
		if err := f.r.Start(1, ui.DragRequest{ID: 9}); err != nil {
			t.Fatal(err)
		}
		f.r.WindowClosed(1)
		f.runPosted()
		if len(f.loops) != 0 || len(f.ends) != 1 || f.ends[0].Action != ui.DropActionNone {
			t.Errorf("loops = %d, ends = %+v", len(f.loops), f.ends)
		}
	})
	t.Run("the platform closing cancels", func(t *testing.T) {
		f := newRunner()
		if err := f.r.Start(1, ui.DragRequest{ID: 9}); err != nil {
			t.Fatal(err)
		}
		f.r.Cancel()
		f.runPosted()
		if len(f.loops) != 0 || len(f.ends) != 1 {
			t.Errorf("loops = %d, ends = %+v", len(f.loops), f.ends)
		}
	})
}

func TestDragThatFailsToRunEndsAsNotAccepted(t *testing.T) {
	f := newRunner()
	f.loopErr = errors.New(tagTestLoopFailed, "DoDragDrop failed")
	f.effect, f.dropped = win32.DropEffectCopy, true
	if err := f.r.Start(1, ui.DragRequest{ID: 3}); err != nil {
		t.Fatal(err)
	}
	f.runPosted()
	if want := []ui.DragResult{{ID: 3}}; !reflect.DeepEqual(f.ends, want) {
		t.Errorf("ends = %+v, want %+v", f.ends, want)
	}
	if f.r.Active() {
		t.Error("the drag is still active")
	}
}

// A drag that returns to the app's own window: the target hears of it with the marker, and the
// framework gets the ID of the request as LocalID.
func TestDragReturningToTheAppsOwnWindowCarriesTheLocalID(t *testing.T) {
	f := newRunner()
	var set *ole.DataSet
	f.onLoop = func(*ole.SourceMachine) { set = f.loops[0].data }
	if err := f.r.Start(1, ui.DragRequest{ID: 1, Text: "hello"}); err != nil {
		t.Fatal(err)
	}
	f.runPosted()
	// What the loop would hand to the app's own target: a data object that offers the set.
	own := &fakeSource{formats: set.Formats(), data: map[uint32][]byte{}}
	for _, fmt := range set.Formats() {
		own.data[fmt], _ = set.Get(fmt)
	}
	h := &dropRecorder{answer: ui.DropActionCopy}
	d := newDropWindow(h, &fakeClock{})
	d.DragEnter(own, 0, win32.Point{}, win32.DropEffectCopy)
	d.Drop(own, 0, win32.Point{}, win32.DropEffectCopy)
	if h.events[0].LocalID != 1 {
		t.Errorf("LocalID = %d, want 1", h.events[0].LocalID)
	}
	if last := h.events[len(h.events)-1]; last.Text != "hello" || last.LocalID != 1 {
		t.Errorf("drop = %+v", last)
	}
}

// The tags of the errors that the fakes of these tests make.
const (
	tagTestReadFailed     = "TestReadFailed"
	tagTestListFailed     = "TestListFailed"
	tagTestRegisterFailed = "TestRegisterFailed"
	tagTestLoopFailed     = "TestLoopFailed"
)
