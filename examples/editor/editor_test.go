package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// useLinux makes the framework behave as on the Linux desktop, which the example
// is for: the tests default to Android.
func useLinux(t *testing.T) {
	t.Helper()
	if !foundation.DebugMode {
		t.Skip("the platform override is a debug-only feature")
	}
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformLinux))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
}

func newTester(t *testing.T, o options) *gluttertest.Tester {
	t.Helper()
	useLinux(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 820, Height: 460}})
	ts.PumpWidget(newApp(o))
	settle(t, ts, func() bool { return editorOf(ts).loaded })
	return ts
}

// settle pumps frames until the goroutine that reads or writes the file has handed
// its result to the UI thread.
func settle(t *testing.T, ts *gluttertest.Tester, done func() bool) {
	t.Helper()
	for range 400 {
		ts.Pump(10 * time.Millisecond)
		if done() {
			ts.Pump(10 * time.Millisecond)
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("the file operation did not finish")
}

func editorOf(ts *gluttertest.Tester) *editorState {
	return gluttertest.StateOf[*editorState](ts, gluttertest.ByType[*editorPage]())
}

func fieldOf(ts *gluttertest.Tester) *widgets.EditableTextState {
	return gluttertest.StateOf[*widgets.EditableTextState](ts, gluttertest.ByType[*widgets.EditableText]())
}

func noErrors(t *testing.T, ts *gluttertest.Tester) {
	t.Helper()
	if errs := ts.TakeErrors(); len(errs) != 0 {
		t.Errorf("the example reported errors: %v", errs)
	}
}

// type_ types the text at the caret, as the input method sends it: the editing
// value gets the characters, and the caret moves after them.
func type_(ts *gluttertest.Tester, text string) {
	field := fieldOf(ts)
	value := editorOf(ts).controller.Value()
	sel := value.Selection
	typed := value.Text[:sel.Start()] + text + value.Text[sel.End():]
	field.UpdateEditingValue(services.NewTextEditingValue(services.TextEditingValueOpts{
		Text: typed, Selection: foundation.Some(services.NewTextSelectionCollapsed(sel.Start() + len(text))),
	}))
	ts.Pump()
}

// pressWithControl presses the key while the Control key is down.
func pressWithControl(ts *gluttertest.Tester, key services.LogicalKeyboardKey, shift bool) {
	ts.SendKeyDownEvent(services.LogicalKeyboardKeyControlLeft)
	if shift {
		ts.SendKeyDownEvent(services.LogicalKeyboardKeyShiftLeft)
	}
	ts.SendKeyEvent(key)
	if shift {
		ts.SendKeyUpEvent(services.LogicalKeyboardKeyShiftLeft)
	}
	ts.SendKeyUpEvent(services.LogicalKeyboardKeyControlLeft)
	ts.Pump()
}

// statusBar is the text of the left part of the status bar.
func statusBar(t *testing.T, ts *gluttertest.Tester) string {
	t.Helper()
	f := gluttertest.ByTextContaining(" line")
	if ts.Count(f) == 0 {
		t.Fatal("no status bar")
	}
	return ts.Widget(f).(*widgets.Text).Data().Value()
}

// statusMessage is the right part of the status bar.
func statusMessage(ts *gluttertest.Tester) string {
	for _, w := range []string{"Modified", "Saved /", "Save failed", "Loaded", "Saving", "Cannot"} {
		if f := gluttertest.ByTextContaining(w); ts.Count(f) > 0 {
			return ts.Widget(f).(*widgets.Text).Data().Value()
		}
	}
	return ""
}

func TestDataDir(t *testing.T) {
	useLinux(t)
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"xdg", map[string]string{"XDG_DATA_HOME": "/data", "HOME": "/home/me"}, "/data/glutter-editor"},
		{"home", map[string]string{"HOME": "/home/me"}, "/home/me/.local/share/glutter-editor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dataDir(func(k string) string { return tt.env[k] })
			if err != nil || got != tt.want {
				t.Errorf("dataDir = %q, %v, want %q", got, err, tt.want)
			}
		})
	}
	_, err := dataDir(func(string) string { return "" })
	requireRoute(t, err, "dataDir.EditorNoDataDir")
}

func TestDataDirOnWindows(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("the platform override is a debug-only feature")
	}
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformWindows))
	t.Cleanup(func() { foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]()) })
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"appdata", map[string]string{"APPDATA": `C:\Users\me\AppData\Roaming`, "HOME": "/home/me", "XDG_DATA_HOME": "/data"}, filepath.Join(`C:\Users\me\AppData\Roaming`, "glutter-editor")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dataDir(func(k string) string { return tt.env[k] })
			if err != nil || got != tt.want {
				t.Errorf("dataDir = %q, %v, want %q", got, err, tt.want)
			}
		})
	}
	// The HOME and XDG_DATA_HOME that Wine passes through are not used.
	_, err := dataDir(func(k string) string { return map[string]string{"HOME": "/home/me", "XDG_DATA_HOME": "/data"}[k] })
	requireRoute(t, err, "dataDir.EditorNoDataDir")
}

func requireRoute(t *testing.T, err error, route string) {
	t.Helper()
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
	}
	if !appErr.HasRoute(route) {
		t.Errorf("unexpected failure path: %s, want %s", appErr.Route(), route)
	}
}

func TestSaveAndLoadNotes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "made", "on", "demand")
	ctx := context.Background()
	if text, found, err := loadNotes(ctx, dir); err != nil || found || text != "" {
		t.Fatalf("loadNotes of nothing = %q, %v, %v", text, found, err)
	}
	for _, text := range []string{"first", "second\nline ✓", ""} {
		if err := saveNotes(ctx, dir, text); err != nil {
			t.Fatalf("saveNotes(%q): %v", text, err)
		}
		got, found, err := loadNotes(ctx, dir)
		if err != nil || !found || got != text {
			t.Errorf("loadNotes = %q, %v, %v, want %q", got, found, err, text)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, fileName+".tmp")); !os.IsNotExist(err) {
		t.Errorf("the temporary file is still there: %v", err)
	}
}

func TestFileFailuresHaveTheirOwnRoutes(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	blocked := t.TempDir() // holds a file where the directory should be
	if err := os.WriteFile(filepath.Join(blocked, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	noWrite := t.TempDir() // the temporary file is a directory
	if err := os.Mkdir(filepath.Join(noWrite, fileName+".tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	noRename := t.TempDir() // the file is a directory that has something in it
	if err := os.MkdirAll(filepath.Join(noRename, fileName, "inside"), 0o755); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		run   func() error
		route string
	}{
		{"load canceled", func() error { _, _, err := loadNotes(canceled, t.TempDir()); return err }, "loadNotes.EditorLoadCanceled"},
		{"load unreadable", func() error { _, _, err := loadNotes(context.Background(), noRename); return err }, "loadNotes.EditorReadFailed"},
		{"save canceled", func() error { return saveNotes(canceled, t.TempDir(), "x") }, "saveNotes.EditorSaveCanceled"},
		{"save no directory", func() error { return saveNotes(context.Background(), filepath.Join(blocked, "file", "sub"), "x") }, "saveNotes.EditorCreateDirFailed"},
		{"save no temporary file", func() error { return saveNotes(context.Background(), noWrite, "x") }, "saveNotes.EditorWriteFailed"},
		{"save no rename", func() error { return saveNotes(context.Background(), noRename, "x") }, "saveNotes.EditorRenameFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireRoute(t, tt.run(), tt.route)
		})
	}
	// A failed rename takes its temporary file away.
	if _, err := os.Stat(filepath.Join(noRename, fileName+".tmp")); !os.IsNotExist(err) {
		t.Errorf("the temporary file stayed after the failed rename: %v", err)
	}
}

func TestOptionsFromEnv(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("release builds ignore the environment")
	}
	env := map[string]string{
		"GLUTTER_DEV_DATA_DIR": "/tmp/x", "GLUTTER_DEV_TEXT": `a\nb`, "GLUTTER_DEV_SELECT": "1,3",
	}
	o, err := optionsFromEnv(func(k string) string { return env[k] })
	if err != nil || o.dir != "/tmp/x" || o.text.OrElse("") != "a\nb" || !o.selectRange || o.selectStart != 1 || o.selEnd != 3 {
		t.Errorf("options = %+v, %v", o, err)
	}
	for _, bad := range []string{"x", "3,1", "-1,2", "1"} {
		_, err := optionsFromEnv(func(k string) string {
			if k == "GLUTTER_DEV_SELECT" {
				return bad
			}
			return ""
		})
		requireRoute(t, err, "optionsFromEnv.EditorSelectInvalid")
	}
}

func TestStats(t *testing.T) {
	tests := []struct {
		name string
		text string
		sel  services.TextSelection
		want string
	}{
		{"empty", "", services.NewTextSelectionCollapsed(0), "Ln 1, Col 1   ·   1 line   ·   0 words   ·   0 characters"},
		{"caret on the second line", "ab cd\nef", services.NewTextSelectionCollapsed(8), "Ln 2, Col 3   ·   2 lines   ·   3 words   ·   8 characters"},
		{"selection", "ab cd\nef", services.NewTextSelection(3, 8), "Ln 2, Col 3 (5 selected)   ·   2 lines   ·   3 words   ·   8 characters"},
		{"columns count characters", "héllo wörld", services.NewTextSelectionCollapsed(len("héllo")), "Ln 1, Col 6   ·   1 line   ·   2 words   ·   11 characters"},
		{"no selection yet", "one", services.NewTextSelectionCollapsed(-1), "1 line   ·   1 word   ·   3 characters"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statsOf(tt.text, tt.sel).describe(); got != tt.want {
				t.Errorf("describe = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestTypeSelectUndoAndSave types into the editor, selects a word with a double
// click, undoes the typing with Ctrl+Z, and saves with Ctrl+S into a temporary
// directory.
func TestTypeSelectUndoAndSave(t *testing.T) {
	dir := t.TempDir()
	ts := newTester(t, options{dir: dir})
	state := editorOf(ts)
	field := fieldOf(ts)
	if !state.focus.HasFocus() {
		t.Fatal("the editor did not take the focus")
	}

	// Typing shows in the status bar, which counts it.
	type_(ts, "hello brave world")
	if got := state.controller.Text(); got != "hello brave world" {
		t.Fatalf("text = %q", got)
	}
	if got, want := statusBar(t, ts), "Ln 1, Col 18   ·   1 line   ·   3 words   ·   17 characters"; got != want {
		t.Errorf("status bar = %q, want %q", got, want)
	}
	if got := statusMessage(ts); got != "Modified" {
		t.Errorf("status = %q, want Modified", got)
	}

	// A double click selects the word under the pointer.
	ts.Pump(600 * time.Millisecond)
	r := field.RenderEditable()
	at := r.LocalToGlobal(r.GetLocalRectForCaret(ui.NewTextPosition(8)).Center(), nil)
	for i := range 2 {
		ts.TapAt(at, gluttertest.GestureOpts{Kind: ui.PointerDeviceKindMouse})
		if i == 0 {
			ts.Pump(50 * time.Millisecond)
		}
	}
	ts.Pump(500 * time.Millisecond)
	if sel := state.controller.Selection(); sel.Start() != 6 || sel.End() != 11 {
		t.Fatalf("selection = %v, want the word brave (6, 11)", sel)
	}
	if got, want := statusBar(t, ts), "Ln 1, Col 12 (5 selected)   ·   1 line   ·   3 words   ·   17 characters"; got != want {
		t.Errorf("status bar = %q, want %q", got, want)
	}

	// Typing replaces the selection; Ctrl+Z brings the word back, Shift+Ctrl+Z
	// (and Ctrl+Y) type it again.
	type_(ts, "small")
	if got := state.controller.Text(); got != "hello small world" {
		t.Fatalf("text = %q", got)
	}
	ts.Pump(600 * time.Millisecond)
	pressWithControl(ts, services.LogicalKeyboardKeyKeyZ, false)
	if got := state.controller.Text(); got != "hello brave world" {
		t.Errorf("after Ctrl+Z text = %q, want the word back", got)
	}
	pressWithControl(ts, services.LogicalKeyboardKeyKeyZ, true)
	if got := state.controller.Text(); got != "hello small world" {
		t.Errorf("after Shift+Ctrl+Z text = %q, want the redone word", got)
	}
	pressWithControl(ts, services.LogicalKeyboardKeyKeyZ, false)
	pressWithControl(ts, services.LogicalKeyboardKeyKeyY, false)
	if got := state.controller.Text(); got != "hello small world" {
		t.Errorf("after Ctrl+Y text = %q, want the redone word", got)
	}

	// Ctrl+S writes the file in a goroutine, and the panel shows the saved text.
	if ts.Count(gluttertest.ByText("Nothing saved yet. Press Ctrl+S.")) != 1 {
		t.Error("the panel of the saved file does not tell that nothing is saved")
	}
	pressWithControl(ts, services.LogicalKeyboardKeyKeyS, false)
	settle(t, ts, func() bool { return !state.saving })
	data, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil || string(data) != "hello small world" {
		t.Fatalf("saved file = %q, %v", data, err)
	}
	if got := statusMessage(ts); got != "Saved "+filepath.Join(dir, fileName) {
		t.Errorf("status = %q, want the saved path", got)
	}
	if ts.Count(gluttertest.ByType[*widgets.SelectableRegion]()) != 1 {
		t.Fatal("the saved file has no SelectableRegion")
	}
	if ts.Count(gluttertest.ByText("hello small world")) == 0 {
		t.Error("the panel does not show the saved text")
	}
	// The next key makes the text modified again.
	type_(ts, "!")
	if got := statusMessage(ts); got != "Modified" {
		t.Errorf("status = %q, want Modified after a change", got)
	}
	noErrors(t, ts)
}

func TestSavedFileIsLoadedAtStartAndCanBeSelected(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("one two\nthree"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := newTester(t, options{dir: dir})
	state := editorOf(ts)
	if got := state.controller.Text(); got != "one two\nthree" {
		t.Errorf("editor text = %q, want the saved file", got)
	}
	if got := statusMessage(ts); got != "Loaded "+fileName {
		t.Errorf("status = %q", got)
	}
	// The panel is a SelectableRegion: select all and copy through the clipboard.
	region := gluttertest.StateOf[*widgets.SelectableRegionState](ts, gluttertest.ByType[*widgets.SelectableRegion]())
	region.SelectAll()
	ts.Pump()
	region.CopySelection()
	ts.Pump()
	if text, ok := ts.ClipboardText(); !ok || text != "one two\nthree" {
		t.Errorf("clipboard = %q, %v, want the saved text", text, ok)
	}
	noErrors(t, ts)
}

func TestFailedSaveShowsInTheStatusBar(t *testing.T) {
	blocked := t.TempDir()
	if err := os.WriteFile(filepath.Join(blocked, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := newTester(t, options{dir: filepath.Join(blocked, "file", "sub")})
	state := editorOf(ts)
	type_(ts, "text")
	pressWithControl(ts, services.LogicalKeyboardKeyKeyS, false)
	settle(t, ts, func() bool { return !state.saving })
	if !state.failed || !strings.HasPrefix(state.status, "Save failed: ") {
		t.Fatalf("status = %q (failed %v), want a save error", state.status, state.failed)
	}
	if ts.Count(gluttertest.ByTextContaining("Save failed")) != 1 {
		t.Error("the error is not in the status bar")
	}
	// The text was not lost, and a later save to a good place would work.
	if got := state.controller.Text(); got != "text" {
		t.Errorf("text = %q", got)
	}
	noErrors(t, ts)
}

func TestWithoutADataDirectoryTheStatusBarSaysSo(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "")
	useLinux(t)
	ts := gluttertest.New(t, gluttertest.Opts{Size: ui.Size{Width: 820, Height: 460}})
	ts.PumpWidget(newApp(options{}))
	ts.Pump(50 * time.Millisecond)
	state := editorOf(ts)
	if !state.failed || !strings.Contains(state.status, "Cannot find the data directory") {
		t.Errorf("status = %q", state.status)
	}
	// Ctrl+S does nothing without a directory, and the editor still works.
	type_(ts, "abc")
	pressWithControl(ts, services.LogicalKeyboardKeyKeyS, false)
	if state.saving || state.controller.Text() != "abc" {
		t.Errorf("saving = %v, text = %q", state.saving, state.controller.Text())
	}
	noErrors(t, ts)
}

// TestEditorGolden draws the editor with a selection that spans two lines, a
// saved file that is older than the text, and the status bar.
func TestEditorGolden(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("Shopping list\n- bread\n- oat milk\n- coffee beans\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := newTester(t, options{
		dir:  dir,
		text: foundation.Some("Shopping list\n- bread\n- oat milk\n- coffee beans\n- dark chocolate\n\nRemember: the quick brown fox jumps over the lazy dog."),
		// From "offee beans" to "dark choc".
		selectRange: true, selectStart: 36, selEnd: 59,
	})
	ts.Pump(100 * time.Millisecond)
	ts.MatchesGolden("editor")
	noErrors(t, ts)
}

// TestNotesInMemoryWhereThereIsNoDataDirectory covers the web, where a page has no directory:
// the notes are kept in memory, the status bar says so (and not that a directory is missing),
// Ctrl+S saves them, and a new editor on the same store loads them.
func TestNotesInMemoryWhereThereIsNoDataDirectory(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "")
	memory := &memoryNotes{}
	ts := newTester(t, options{memory: memory})
	state := editorOf(ts)
	if state.failed || state.dirErr != nil || !strings.Contains(state.status, "kept in memory") {
		t.Errorf("status = %q (failed %v, dirErr %v)", state.status, state.failed, state.dirErr)
	}
	type_(ts, "web notes")
	pressWithControl(ts, services.LogicalKeyboardKeyKeyS, false)
	settle(t, ts, func() bool { return !state.saving })
	if state.failed || !strings.HasPrefix(state.status, "Saved in memory") {
		t.Errorf("status = %q (failed %v)", state.status, state.failed)
	}
	if ts.Count(gluttertest.ByText("web notes")) == 0 {
		t.Error("the panel does not show the saved text")
	}
	noErrors(t, ts)

	ts2 := newTester(t, options{memory: memory})
	if got := editorOf(ts2).controller.Text(); got != "web notes" {
		t.Errorf("a new editor loaded %q, want the saved notes", got)
	}
}

func TestMemoryNotesCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := &memoryNotes{}
	_, _, err := m.load(ctx)
	requireRoute(t, err, "load.EditorLoadCanceled")
	requireRoute(t, m.save(ctx, "x"), "save.EditorSaveCanceled")
}
