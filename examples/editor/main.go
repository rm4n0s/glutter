// Command editor is a small text editor on a WidgetsApp, without Material. The
// left panel is a multi-line EditableText with undo and redo (Ctrl+Z, Shift+Ctrl+Z
// and Ctrl+Y). Ctrl+S, a Shortcuts binding that maps to a SaveIntent, writes the
// text to $XDG_DATA_HOME/glutter-editor/notes.txt (or ~/.local/share/...) in a
// goroutine with a context.Context; a failure shows in the status bar. A web page
// has no data directory, so on the web (foundation.IsWeb) the notes are kept in
// memory for as long as the page is open, and the status bar says so. The right
// panel is a read-only SelectableRegion with the saved file, which you can select
// and copy. The status bar shows the line and column of the caret, the number of
// lines, words and characters, and whether the text has unsaved changes. It uses
// only the public packages.
//
// Run it with: go run ./examples/editor
// On the web: go run ./cmd/glutter-web serve ./examples/editor
//
// Environment: GLUTTER_BACKEND=x11|wayland, GLUTTER_SCREENSHOT=<file.png>,
// GLUTTER_SCREENSHOT_AFTER=<duration>. In debug builds:
// GLUTTER_DEV_DATA_DIR=<dir> keeps the notes in that directory (instead of the
// data directory of the user), GLUTTER_DEV_TEXT=<text> replaces the text of the
// editor once the saved file is loaded (\n is a new line), and
// GLUTTER_DEV_SELECT=<start,end> selects that byte range of the text.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/animation"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/material/fonts"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/widgets"
)

// The Material 3 baseline colors that the example paints with.
var (
	primary   = ui.NewColor(0xFF6750A4)
	surface   = ui.NewColor(0xFFFFFBFE)
	panel     = ui.NewColor(0xFFF3EDF7)
	outline   = ui.NewColor(0xFFCAC4D0)
	onSurface = ui.NewColor(0xFF1C1B1F)
	muted     = ui.NewColor(0xFF625B71)
	errorRed  = ui.NewColor(0xFFB3261E)
	selection = ui.NewColor(0x666750A4)
	white     = ui.NewColor(0xFFFFFFFF)
	barColor  = ui.NewColor(0xFFE8DEF8)
)

const (
	// dirName is the directory of the app inside the data directory.
	dirName = "glutter-editor"
	// fileName is the file that Ctrl+S writes.
	fileName = "notes.txt"
	// memoryStatus tells that the notes live in memory (the web has no data directory).
	memoryStatus = "in memory (this page has no data directory)"
)

func textStyle(size float64, color ui.Color, weight ui.FontWeight) *painting.TextStyle {
	return painting.NewTextStyle(painting.TextStyleOpts{
		FontFamily: fonts.RobotoFamily, FontSize: foundation.Some(size), Color: foundation.Some(color), FontWeight: weight,
	})
}

func label(s string, size float64, color ui.Color, weight ui.FontWeight) widgets.Widget {
	return widgets.NewText(s, widgets.TextOpts{Style: textStyle(size, color, weight)})
}

// oneLine is a label that is cut with an ellipsis instead of wrapping, for the
// status bar, where a long error message must not push the rest away.
func oneLine(s string, size float64, color ui.Color, weight ui.FontWeight) widgets.Widget {
	return widgets.NewText(s, widgets.TextOpts{
		Style: textStyle(size, color, weight), Overflow: painting.TextOverflowEllipsis, MaxLines: foundation.Some(1),
	})
}

// dataDir is the directory that holds the notes: $XDG_DATA_HOME/glutter-editor,
// or ~/.local/share/glutter-editor when XDG_DATA_HOME is not set.
func dataDir(getenv func(string) string) (string, error) {
	if base := getenv("XDG_DATA_HOME"); base != "" {
		return filepath.Join(base, dirName), nil
	}
	if home := getenv("HOME"); home != "" {
		return filepath.Join(home, ".local", "share", dirName), nil
	}
	return "", errors.New(tagEditorNoDataDir, "neither XDG_DATA_HOME nor HOME is set")
}

// memoryNotes are the notes of an app that has no directory to keep them in, such as a web
// page: they last until the page is closed. The saved text is read and written by the
// goroutines of the state, so a mutex guards it.
type memoryNotes struct {
	mu    sync.Mutex
	text  string
	found bool
}

// load returns the saved text; found is false when nothing was saved yet.
func (m *memoryNotes) load(ctx context.Context) (text string, found bool, err error) {
	if err := ctx.Err(); err != nil {
		return "", false, errors.NewErr(tagEditorLoadCanceled, err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.text, m.found, nil
}

// save keeps the text.
func (m *memoryNotes) save(ctx context.Context, text string) error {
	if err := ctx.Err(); err != nil {
		return errors.NewErr(tagEditorSaveCanceled, err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.text, m.found = text, true
	return nil
}

// loadNotes reads the saved file; found is false when there is none yet.
func loadNotes(ctx context.Context, dir string) (text string, found bool, err error) {
	if err := ctx.Err(); err != nil {
		return "", false, errors.NewErr(tagEditorLoadCanceled, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, fileName))
	switch {
	case err == nil:
		return string(data), true, nil
	case errors.Is(err, os.ErrNotExist):
		return "", false, nil
	}
	return "", false, errors.NewErr(tagEditorReadFailed, err).SetMetadata("dir", dir)
}

// saveNotes writes the text to the file, through a temporary file, so that a
// failure leaves the previous version in place.
func saveNotes(ctx context.Context, dir, text string) error {
	if err := ctx.Err(); err != nil {
		return errors.NewErr(tagEditorSaveCanceled, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errors.NewErr(tagEditorCreateDirFailed, err).SetMetadata("dir", dir)
	}
	path := filepath.Join(dir, fileName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(text), 0o644); err != nil {
		return errors.NewErr(tagEditorWriteFailed, err).SetMetadata("file", tmp)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return errors.NewErr(tagEditorRenameFailed, err).SetMetadata("file", path)
	}
	return nil
}

// stats are the numbers of the status bar.
type stats struct {
	line, column         int // of the caret, from 1
	lines, words, chars  int
	selected             int // characters in the selection
	hasSelection, hasPos bool
}

// statsOf counts the text. The column and the number of selected characters
// count characters, not bytes; the offsets of the selection are byte offsets.
func statsOf(text string, sel services.TextSelection) stats {
	s := stats{
		lines: strings.Count(text, "\n") + 1,
		words: len(strings.Fields(text)),
		chars: utf8.RuneCountInString(text),
	}
	if sel.BaseOffset < 0 || sel.ExtentOffset < 0 || sel.BaseOffset > len(text) || sel.ExtentOffset > len(text) {
		return s
	}
	s.hasPos = true
	before := text[:sel.ExtentOffset]
	s.line = strings.Count(before, "\n") + 1
	s.column = utf8.RuneCountInString(before[strings.LastIndex(before, "\n")+1:]) + 1
	if !sel.IsCollapsed() {
		s.hasSelection = true
		s.selected = utf8.RuneCountInString(text[sel.Start():sel.End()])
	}
	return s
}

// describe is the left part of the status bar.
func (s stats) describe() string {
	var b strings.Builder
	if s.hasPos {
		fmt.Fprintf(&b, "Ln %d, Col %d", s.line, s.column)
		if s.hasSelection {
			fmt.Fprintf(&b, " (%d selected)", s.selected)
		}
		b.WriteString("   ·   ")
	}
	fmt.Fprintf(&b, "%d %s   ·   %d %s   ·   %d %s",
		s.lines, plural(s.lines, "line"), s.words, plural(s.words, "word"), s.chars, plural(s.chars, "character"))
	return b.String()
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// options are what the environment of a debug build can set.
type options struct {
	// dir is the directory of the notes; empty means the data directory of the
	// user.
	dir string
	// memory, when it is set, keeps the notes in memory instead of a directory: a
	// web page has no data directory (main sets it when foundation.IsWeb).
	memory *memoryNotes
	// text replaces the text of the editor after the saved file is loaded.
	text foundation.Opt[string]
	// selectStart and selEnd select a range of the text when selectRange is set.
	selectRange         bool
	selectStart, selEnd int
}

// optionsFromEnv reads the GLUTTER_DEV_ variables. Release builds ignore them.
func optionsFromEnv(getenv func(string) string) (options, error) {
	var o options
	if !foundation.DebugMode {
		return o, nil
	}
	o.dir = getenv("GLUTTER_DEV_DATA_DIR")
	if v := getenv("GLUTTER_DEV_TEXT"); v != "" {
		o.text = foundation.Some(strings.ReplaceAll(v, `\n`, "\n"))
	}
	if v := getenv("GLUTTER_DEV_SELECT"); v != "" {
		start, end, ok := strings.Cut(v, ",")
		a, errA := strconv.Atoi(start)
		b, errB := strconv.Atoi(end)
		if !ok || errA != nil || errB != nil || a < 0 || b < a {
			return o, errors.New(tagEditorSelectInvalid, `GLUTTER_DEV_SELECT must be "start,end" with 0 <= start <= end`, "value", v)
		}
		o.selectRange, o.selectStart, o.selEnd = true, a, b
	}
	return o, nil
}

// saveIntent asks to save the text.
type saveIntent struct {
	widgets.IntentBase
}

// newApp is the whole app: a WidgetsApp with one route, whose page is the editor.
// The route gives the page the Overlay that the EditableText needs.
func newApp(o options) widgets.Widget {
	return widgets.NewWidgetsApp(surface, widgets.WidgetsAppOpts{
		Title: "Glutter editor",
		// The window is a proof image: it has no debug ribbon.
		DebugShowCheckedModeBanner: foundation.Some(false),
		TextStyle:                  textStyle(16, onSurface, ui.FontWeightW400),
		OnGenerateRoute: func(settings widgets.RouteSettings) widgets.Route {
			return widgets.NewPageRouteBuilder(
				func(widgets.BuildContext, animation.Animation[float64], animation.Animation[float64]) widgets.Widget {
					return newEditorPage(o)
				},
				widgets.PageRouteBuilderOpts{PageRouteOpts: widgets.PageRouteOpts{RouteOpts: widgets.RouteOpts{Settings: settings}}})
		},
	})
}

// editorPage is the whole window: the editor, the saved file and the status bar.
type editorPage struct {
	widgets.StatefulWidget
	o options
}

func newEditorPage(o options) *editorPage {
	return &editorPage{StatefulWidget: widgets.NewStatefulWidgetBase(), o: o}
}

func (w *editorPage) CreateState() widgets.State { return &editorState{} }

type editorState struct {
	widgets.StateOf[*editorPage]
	controller *widgets.TextEditingController
	focus      *widgets.FocusNode
	// fieldKey finds the EditableText, and gestures turns the mouse into
	// selections: a click puts the caret, a double click selects a word, a triple
	// click a line, a drag a range, and a middle click pastes the primary
	// selection. TextField does this in Flutter; this app has no Material.
	fieldKey *widgets.GlobalKey[*widgets.EditableTextState]
	gestures *widgets.TextSelectionGestureDetectorBuilder
	// dir is the directory of the notes; dirErr is why there is none.
	dir    string
	dirErr error
	// saved is the text of the file as it was last loaded or saved, and loaded
	// says whether that has happened.
	saved  string
	loaded bool
	// saving is set while a save runs, so that two saves do not overlap.
	saving bool
	// status is the message of the status bar; failed makes it an error.
	status string
	failed bool
}

func (s *editorState) InitState() {
	s.controller = widgets.NewTextEditingController()
	s.focus = widgets.NewFocusNode()
	s.fieldKey = widgets.NewGlobalKey[*widgets.EditableTextState]()
	s.gestures = widgets.NewTextSelectionGestureDetectorBuilder(s)
	s.dir = s.Widget().o.dir
	if s.dir == "" && s.Widget().o.memory == nil {
		s.dir, s.dirErr = dataDir(os.Getenv)
	}
	if s.dirErr != nil {
		s.fail("Cannot find the data directory", s.dirErr)
		return
	}
	dir, o := s.dir, s.Widget().o
	s.Go(func(ctx context.Context) {
		var (
			text  string
			found bool
			err   error
		)
		if o.memory != nil {
			text, found, err = o.memory.load(ctx)
		} else {
			text, found, err = loadNotes(ctx, dir)
		}
		s.SetState(func() {
			s.loaded = true
			if err != nil {
				s.fail("Cannot load "+fileName, err)
				return
			}
			if o.memory != nil && s.status == "" {
				s.status = "Notes are kept " + memoryStatus
			}
			if found {
				s.saved = text
				s.status = "Loaded " + fileName
				if o.memory != nil {
					s.status = "Loaded the notes " + memoryStatus
				}
				s.controller.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{
					Text: text, Selection: foundation.Some(services.NewTextSelectionCollapsed(len(text))),
				}))
			}
			if t, ok := o.text.Get(); ok {
				s.controller.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{
					Text: t, Selection: foundation.Some(services.NewTextSelectionCollapsed(len(t))),
				}))
			}
			if o.selectRange {
				n := len(s.controller.Text())
				s.controller.SetSelection(services.NewTextSelection(min(o.selectStart, n), min(o.selEnd, n)))
			}
		})
	})
}

// EditableTextKey, ForcePressEnabled and SelectionEnabled make the state the
// delegate of the gesture builder.
func (s *editorState) EditableTextKey() *widgets.GlobalKey[*widgets.EditableTextState] {
	return s.fieldKey
}
func (s *editorState) ForcePressEnabled() bool { return false }
func (s *editorState) SelectionEnabled() bool  { return true }

func (s *editorState) Dispose() {
	s.controller.Dispose()
	s.focus.Dispose()
}

// fail shows the error in the status bar.
func (s *editorState) fail(what string, err error) {
	s.status, s.failed = what+": "+describeError(err), true
}

// describeError is the message of an error and the error under it.
func describeError(err error) string {
	appErr, ok := errors.FromError(err)
	if !ok {
		return err.Error()
	}
	if appErr.OriginalErr != nil {
		return appErr.Message + " (" + appErr.OriginalErr.Error() + ")"
	}
	return appErr.Message
}

// save writes the text of the editor to the file in a goroutine and shows the
// result in the status bar.
func (s *editorState) save() {
	if s.saving || s.dirErr != nil {
		return
	}
	text, dir, memory := s.controller.Text(), s.dir, s.Widget().o.memory
	s.SetState(func() { s.saving, s.status, s.failed = true, "Saving...", false })
	s.Go(func(ctx context.Context) {
		var err error
		if memory != nil {
			err = memory.save(ctx, text)
		} else {
			err = saveNotes(ctx, dir, text)
		}
		s.SetState(func() {
			s.saving = false
			if err != nil {
				s.fail("Save failed", err)
				return
			}
			s.saved = text
			if memory != nil {
				s.status = "Saved " + memoryStatus
				return
			}
			s.status = "Saved " + filepath.Join(dir, fileName)
		})
	})
}

func (s *editorState) Build(bc widgets.BuildContext) widgets.Widget {
	body := widgets.NewColumn([]widgets.Widget{
		widgets.NewExpanded(widgets.NewPadding(painting.NewEdgeInsetsAll(16), s.panels())),
		s.statusBar(),
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
	// Clear of the system bars and the cutout, and above the soft keyboard (nothing, on the desktop).
	padded := widgets.NewPadding(painting.NewEdgeInsetsFromLTRB(0, 0, 0, widgets.MediaQueryViewInsetsOf(bc).Bottom), widgets.NewSafeArea(body))
	return widgets.NewShortcuts(widgets.ShortcutMap{
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyKeyS, widgets.SingleActivatorOpts{Control: true}), Intent: saveIntent{}},
		{Activator: widgets.NewSingleActivator(services.LogicalKeyboardKeyKeyY, widgets.SingleActivatorOpts{Control: true}), Intent: widgets.RedoTextIntent{Cause: services.SelectionChangedCauseKeyboard}},
	}, widgets.NewActions(widgets.ActionMap{
		widgets.IntentTypeOf[saveIntent](): widgets.NewCallbackAction(func(saveIntent) any {
			s.save()
			return nil
		}),
	}, widgets.NewColoredBox(surface, padded)))
}

// panels are the editor on the left and the saved file on the right.
func (s *editorState) panels() widgets.Widget {
	return widgets.NewRow([]widgets.Widget{
		widgets.NewExpanded(titled("Editor", s.editor()), widgets.ExpandedOpts{Flex: foundation.Some(3)}),
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(16.0)}),
		widgets.NewExpanded(titled("Saved copy (read-only)", s.savedPanel()), widgets.ExpandedOpts{Flex: foundation.Some(2)}),
	}, widgets.RowOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
}

// titled puts a heading over the body.
func titled(title string, body widgets.Widget) widgets.Widget {
	return widgets.NewColumn([]widgets.Widget{
		label(title, 13, muted, ui.FontWeightW500),
		widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Height: foundation.Some(6.0)}),
		widgets.NewExpanded(body),
	}, widgets.ColumnOpts{CrossAxisAlignment: rendering.CrossAxisAlignmentStretch})
}

func box(color ui.Color, child widgets.Widget) widgets.Widget {
	return widgets.NewContainer(child, widgets.ContainerOpts{
		Padding: painting.NewEdgeInsetsAll(12),
		Decoration: painting.NewBoxDecoration(painting.BoxDecorationOpts{
			Color:        foundation.Some(color),
			BorderRadius: painting.NewBorderRadiusCircular(8),
			Border:       painting.NewBorderAll(painting.BorderSideOpts{Color: foundation.Some(outline)}),
		}),
	})
}

// editor is the multi-line EditableText. It fills its panel and scrolls itself.
func (s *editorState) editor() widgets.Widget {
	field := widgets.NewEditableText(s.controller, s.focus, textStyle(16, onSurface, ui.FontWeightW400), primary, outline,
		widgets.EditableTextOpts{
			Key:                    s.fieldKey,
			Autofocus:              true,
			Expands:                true,
			MaxLines:               foundation.Some(rendering.MaxLinesUnlimited),
			RendererIgnoresPointer: true,
			SelectionColor:         foundation.Some(selection),
			SelectionControls:      widgets.NewEmptyTextSelectionControls(),
			KeyboardType:           foundation.Some(services.TextInputTypeMultiline),
			TextInputAction:        services.TextInputActionNewline,
			MouseCursor:            services.SystemMouseCursors.Text,
		})
	return box(white, s.gestures.BuildGestureDetector(field, widgets.TextSelectionGestureDetectorBuildOpts{
		Behavior: rendering.HitTestBehaviorTranslucent,
	}))
}

// savedPanel shows the saved file in a read-only SelectableRegion.
func (s *editorState) savedPanel() widgets.Widget {
	var content widgets.Widget
	switch {
	case !s.loaded:
		content = label("Loading...", 16, muted, ui.FontWeightW400)
	case s.saved == "":
		content = label("Nothing saved yet. Press Ctrl+S.", 16, muted, ui.FontWeightW400)
	default:
		content = widgets.NewSelectableRegion(widgets.NewEmptyTextSelectionControls(),
			widgets.NewText(s.saved, widgets.TextOpts{Style: textStyle(16, onSurface, ui.FontWeightW400)}),
			widgets.SelectableRegionOpts{})
	}
	return box(panel, widgets.NewSingleChildScrollView(content))
}

// statusBar shows the numbers of the text and the status. It follows the
// controller, so that it changes with every key.
func (s *editorState) statusBar() widgets.Widget {
	return widgets.NewListenableBuilder(s.controller, func(widgets.BuildContext, widgets.Widget) widgets.Widget {
		st := statsOf(s.controller.Text(), s.controller.Selection())
		right, color := s.status, muted
		switch {
		case s.failed:
			color = errorRed
		case s.controller.Text() != s.saved && !s.saving:
			right = "Modified"
		}
		return widgets.NewColoredBox(barColor, widgets.NewPadding(painting.NewEdgeInsetsSymmetric(8, 16), widgets.NewRow([]widgets.Widget{
			widgets.NewExpanded(oneLine(st.describe(), 13, onSurface, ui.FontWeightW400)),
			widgets.NewSizedBox(nil, widgets.SizedBoxOpts{Width: foundation.Some(16.0)}),
			widgets.NewConstrainedBox(rendering.NewBoxConstraints(rendering.BoxConstraintsOpts{MaxWidth: foundation.Some(360.0)}),
				oneLine(right, 13, color, ui.FontWeightW500)),
		})))
	}, nil)
}

func main() {
	o, err := optionsFromEnv(os.Getenv)
	if err == nil {
		if foundation.IsWeb && o.dir == "" {
			o.memory = &memoryNotes{}
		}
		err = widgets.RunApp(newApp(o), widgets.RunAppOpts{
			AppID:  "dev.glutter.editor",
			Window: widgets.WindowConfig{Title: "Glutter – editor", Size: ui.Size{Width: 820, Height: 460}},
		})
	}
	if err != nil {
		if appErr, ok := errors.FromError(err); ok {
			fmt.Fprintf(os.Stderr, "%s: %s %v\n", appErr.Tag, appErr.Message, appErr.Metadata)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagEditorCreateDirFailed = "EditorCreateDirFailed"
	tagEditorLoadCanceled    = "EditorLoadCanceled"
	tagEditorNoDataDir       = "EditorNoDataDir"
	tagEditorReadFailed      = "EditorReadFailed"
	tagEditorRenameFailed    = "EditorRenameFailed"
	tagEditorSaveCanceled    = "EditorSaveCanceled"
	tagEditorSelectInvalid   = "EditorSelectInvalid"
	tagEditorWriteFailed     = "EditorWriteFailed"
)
