package services

import (
	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

// TextSelectionDelegate is what a text selection toolbar and handles need from
// the editable text they work on.
//
// Flutter: services/text_input.dart TextSelectionDelegate
type TextSelectionDelegate interface {
	// TextEditingValue is the current value of the field.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.textEditingValue
	TextEditingValue() TextEditingValue
	// UserUpdateTextEditingValue sets the value of the field because of the user,
	// with the reason the selection changed.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.userUpdateTextEditingValue
	UserUpdateTextEditingValue(value TextEditingValue, cause SelectionChangedCause)
	// HideToolbar hides the toolbar, and the handles too when hideHandles is
	// true.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.hideToolbar
	HideToolbar(hideHandles bool)
	// BringIntoView scrolls the position into view.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.bringIntoView
	BringIntoView(position ui.TextPosition)
	// CutEnabled is whether the field can cut the selection.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.cutEnabled
	CutEnabled() bool
	// CopyEnabled is whether the field can copy the selection.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.copyEnabled
	CopyEnabled() bool
	// PasteEnabled is whether the field can paste.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.pasteEnabled
	PasteEnabled() bool
	// SelectAllEnabled is whether the field can select all its text.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.selectAllEnabled
	SelectAllEnabled() bool
	// LookUpEnabled is whether the field can look the selection up.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.lookUpEnabled
	LookUpEnabled() bool
	// SearchWebEnabled is whether the field can search the web for the selection.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.searchWebEnabled
	SearchWebEnabled() bool
	// ShareEnabled is whether the field can share the selection.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.shareEnabled
	ShareEnabled() bool
	// LiveTextInputEnabled is whether the field can take its text from the
	// camera.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.liveTextInputEnabled
	LiveTextInputEnabled() bool
	// CutSelection cuts the selection to the clipboard.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.cutSelection
	CutSelection(cause SelectionChangedCause)
	// PasteText pastes the text of the clipboard; the paste finishes later,
	// when the clipboard answers.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.pasteText
	PasteText(cause SelectionChangedCause)
	// SelectAll selects all the text.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.selectAll
	SelectAll(cause SelectionChangedCause)
	// CopySelection copies the selection to the clipboard.
	//
	// Flutter: services/text_input.dart TextSelectionDelegate.copySelection
	CopySelection(cause SelectionChangedCause)
}

// TextSelectionDelegateDefaults gives the methods of TextSelectionDelegate that
// have a default in Flutter: cut, copy, paste and select all are enabled; the
// others are not (Live Text input is off, look up, web search and share are on).
// Embed it in a delegate.
//
// Flutter: services/text_input.dart TextSelectionDelegate
type TextSelectionDelegateDefaults struct{}

// CutEnabled is true.
func (TextSelectionDelegateDefaults) CutEnabled() bool { return true }

// CopyEnabled is true.
func (TextSelectionDelegateDefaults) CopyEnabled() bool { return true }

// PasteEnabled is true.
func (TextSelectionDelegateDefaults) PasteEnabled() bool { return true }

// SelectAllEnabled is true.
func (TextSelectionDelegateDefaults) SelectAllEnabled() bool { return true }

// LookUpEnabled is true.
func (TextSelectionDelegateDefaults) LookUpEnabled() bool { return true }

// SearchWebEnabled is true.
func (TextSelectionDelegateDefaults) SearchWebEnabled() bool { return true }

// ShareEnabled is true.
func (TextSelectionDelegateDefaults) ShareEnabled() bool { return true }

// LiveTextInputEnabled is false.
func (TextSelectionDelegateDefaults) LiveTextInputEnabled() bool { return false }

// TextInputClient is an interface for a client of the text input of the
// platform: a text field that the input method edits.
//
// Flutter: services/text_input.dart TextInputClient
type TextInputClient interface {
	// CurrentTextEditingValue is the value of the field right now; ok is false
	// when the client has none.
	//
	// Flutter: services/text_input.dart TextInputClient.currentTextEditingValue
	CurrentTextEditingValue() (TextEditingValue, bool)
	// CurrentAutofillScope is the scope of autofill the field is in, or nil.
	//
	// Flutter: services/text_input.dart TextInputClient.currentAutofillScope
	CurrentAutofillScope() AutofillScope
	// UpdateEditingValue is called when the platform changes the value of the
	// field.
	//
	// Flutter: services/text_input.dart TextInputClient.updateEditingValue
	UpdateEditingValue(value TextEditingValue)
	// PerformAction is called when the user does the action of the Enter key.
	//
	// Flutter: services/text_input.dart TextInputClient.performAction
	PerformAction(action TextInputAction)
	// InsertContent is called when the input method inserts a content, such as
	// an image.
	//
	// Flutter: services/text_input.dart TextInputClient.insertContent
	InsertContent(content KeyboardInsertedContent)
	// PerformPrivateCommand is called when the input method sends a command only
	// it and the app know.
	//
	// Flutter: services/text_input.dart TextInputClient.performPrivateCommand
	PerformPrivateCommand(action string, data map[string]any)
	// UpdateFloatingCursor is called when the floating cursor moves.
	//
	// Flutter: services/text_input.dart TextInputClient.updateFloatingCursor
	UpdateFloatingCursor(point RawFloatingCursorPoint)
	// ShowAutocorrectionPromptRect is called when the platform wants the app to
	// mark the text from start to end as autocorrected.
	//
	// Flutter: services/text_input.dart TextInputClient.showAutocorrectionPromptRect
	ShowAutocorrectionPromptRect(start, end int)
	// OnFocusReceived is called when the platform asks the field to take focus;
	// it returns whether it did.
	//
	// Flutter: services/text_input.dart TextInputClient.onFocusReceived
	OnFocusReceived() bool
	// ConnectionClosed is called when the platform closes the connection.
	//
	// Flutter: services/text_input.dart TextInputClient.connectionClosed
	ConnectionClosed()
	// DidChangeInputControl is called when the input control of the TextInput
	// changes while the client is attached.
	//
	// Flutter: services/text_input.dart TextInputClient.didChangeInputControl
	DidChangeInputControl(oldControl, newControl TextInputControl)
	// ShowToolbar is called when the platform asks the app to show its
	// selection toolbar.
	//
	// Flutter: services/text_input.dart TextInputClient.showToolbar
	ShowToolbar()
	// InsertTextPlaceholder is called when the platform asks for room for a
	// placeholder of the size in the text.
	//
	// Flutter: services/text_input.dart TextInputClient.insertTextPlaceholder
	InsertTextPlaceholder(size ui.Size)
	// RemoveTextPlaceholder is called when the placeholder is not needed.
	//
	// Flutter: services/text_input.dart TextInputClient.removeTextPlaceholder
	RemoveTextPlaceholder()
	// PerformSelector is called when the platform asks for an editing command by
	// the name of its selector (such as "moveLeft:").
	//
	// Flutter: services/text_input.dart TextInputClient.performSelector
	PerformSelector(selectorName string)
}

// TextInputClientDefaults gives the methods of TextInputClient that have a
// default in Flutter, which do nothing (OnFocusReceived returns false). Embed it
// in a client.
//
// Flutter: services/text_input.dart TextInputClient
type TextInputClientDefaults struct{}

// InsertContent does nothing.
func (TextInputClientDefaults) InsertContent(KeyboardInsertedContent) {}

// OnFocusReceived returns false.
func (TextInputClientDefaults) OnFocusReceived() bool { return false }

// DidChangeInputControl does nothing.
func (TextInputClientDefaults) DidChangeInputControl(oldControl, newControl TextInputControl) {}

// ShowToolbar does nothing.
func (TextInputClientDefaults) ShowToolbar() {}

// InsertTextPlaceholder does nothing.
func (TextInputClientDefaults) InsertTextPlaceholder(ui.Size) {}

// RemoveTextPlaceholder does nothing.
func (TextInputClientDefaults) RemoveTextPlaceholder() {}

// PerformSelector does nothing.
func (TextInputClientDefaults) PerformSelector(string) {}

// DeltaTextInputClient is a TextInputClient that gets the changes of the
// platform as TextEditingDeltas, when TextInputConfiguration.EnableDeltaModel is
// set.
//
// Flutter: services/text_input.dart DeltaTextInputClient
type DeltaTextInputClient interface {
	TextInputClient
	// UpdateEditingValueWithDeltas is called when the platform changes the value
	// of the field, with the changes in order.
	//
	// Flutter: services/text_input.dart DeltaTextInputClient.updateEditingValueWithDeltas
	UpdateEditingValueWithDeltas(textEditingDeltas []TextEditingDelta)
}

// TextInputControl is a controller of the text input: the platform input
// method is one, and an app can add its own (an on-screen keyboard that is
// part of the app, for example). Controls must be comparable, like pointers.
//
// Flutter: services/text_input.dart TextInputControl
type TextInputControl interface {
	// Attach is called when a client connects to the text input.
	//
	// Flutter: services/text_input.dart TextInputControl.attach
	Attach(client TextInputClient, configuration TextInputConfiguration)
	// Detach is called when the client disconnects.
	//
	// Flutter: services/text_input.dart TextInputControl.detach
	Detach(client TextInputClient)
	// Show is called when the client asks for the input method to show.
	//
	// Flutter: services/text_input.dart TextInputControl.show
	Show()
	// Hide is called when the input method should hide.
	//
	// Flutter: services/text_input.dart TextInputControl.hide
	Hide()
	// UpdateConfig is called when the configuration of the client changes.
	//
	// Flutter: services/text_input.dart TextInputControl.updateConfig
	UpdateConfig(configuration TextInputConfiguration)
	// SetEditingState is called when the value of the client changes.
	//
	// Flutter: services/text_input.dart TextInputControl.setEditingState
	SetEditingState(value TextEditingValue)
	// SetEditableSizeAndTransform is called when the size of the editable or its
	// transform to the view changes.
	//
	// Flutter: services/text_input.dart TextInputControl.setEditableSizeAndTransform
	SetEditableSizeAndTransform(editableBoxSize ui.Size, transform *ui.Matrix4)
	// SetComposingRect is called when the rectangle of the composing text, in the
	// coordinates of the editable, changes.
	//
	// Flutter: services/text_input.dart TextInputControl.setComposingRect
	SetComposingRect(rect ui.Rect)
	// SetCaretRect is called when the rectangle of the caret, in the coordinates
	// of the editable, changes.
	//
	// Flutter: services/text_input.dart TextInputControl.setCaretRect
	SetCaretRect(rect ui.Rect)
	// SetSelectionRects is called when the rectangles of the characters change.
	//
	// Flutter: services/text_input.dart TextInputControl.setSelectionRects
	SetSelectionRects(selectionRects []SelectionRect)
	// UpdateStyle is called when the style of the text changes.
	//
	// Flutter: services/text_input.dart TextInputControl.updateStyle
	UpdateStyle(style TextInputStyle)
	// RequestAutofill is called when the client asks for autofill.
	//
	// Flutter: services/text_input.dart TextInputControl.requestAutofill
	RequestAutofill()
	// FinishAutofillContext is called when the autofill context ends.
	//
	// Flutter: services/text_input.dart TextInputControl.finishAutofillContext
	FinishAutofillContext(shouldSave bool)
}

// TextInputControlDefaults gives the methods of TextInputControl, which all do
// nothing. Embed it in a control, and write the methods it needs.
//
// Flutter: services/text_input.dart TextInputControl
type TextInputControlDefaults struct{}

// Attach does nothing.
func (TextInputControlDefaults) Attach(TextInputClient, TextInputConfiguration) {}

// Detach does nothing.
func (TextInputControlDefaults) Detach(TextInputClient) {}

// Show does nothing.
func (TextInputControlDefaults) Show() {}

// Hide does nothing.
func (TextInputControlDefaults) Hide() {}

// UpdateConfig does nothing.
func (TextInputControlDefaults) UpdateConfig(TextInputConfiguration) {}

// SetEditingState does nothing.
func (TextInputControlDefaults) SetEditingState(TextEditingValue) {}

// SetEditableSizeAndTransform does nothing.
func (TextInputControlDefaults) SetEditableSizeAndTransform(ui.Size, *ui.Matrix4) {}

// SetComposingRect does nothing.
func (TextInputControlDefaults) SetComposingRect(ui.Rect) {}

// SetCaretRect does nothing.
func (TextInputControlDefaults) SetCaretRect(ui.Rect) {}

// SetSelectionRects does nothing.
func (TextInputControlDefaults) SetSelectionRects([]SelectionRect) {}

// UpdateStyle does nothing.
func (TextInputControlDefaults) UpdateStyle(TextInputStyle) {}

// RequestAutofill does nothing.
func (TextInputControlDefaults) RequestAutofill() {}

// FinishAutofillContext does nothing.
func (TextInputControlDefaults) FinishAutofillContext(bool) {}

// TextInputBackend is where the input method of the platform is: the platform
// dispatcher (ui.PlatformDispatcher) is one. It takes the state of the focused
// text field.
//
// Flutter: no counterpart (the flutter/textinput channel)
type TextInputBackend interface {
	// SetTextInput gives the input method the state of the focused field.
	SetTextInput(state ui.TextInputState)
}

// TextInputOpts are the optional parameters of NewTextInput.
//
// Flutter: no counterpart
type TextInputOpts struct {
	// Backend is the input method of the platform. Without one, nothing is sent
	// to a platform, but the controls and the clients work.
	Backend TextInputBackend
	// Post runs a function on the UI thread after the work that runs now: it
	// stands for Dart's scheduleMicrotask. Without it, the function runs at
	// once.
	Post func(fn func())
}

// TextInput is the text input service: it connects the text fields of the
// framework (the clients) to the controls (the input method of the platform and
// the controls that an app adds). The binding has one.
//
// Deviation: Flutter's TextInput is a singleton with static methods that talk
// to the engine over the flutter/textinput channel; here it is a value the
// binding owns and the engine side is the platform control, which sends
// ui.TextInputState to the platform and takes ui.TextInputEvent from it. The
// scribble (stylus handwriting on iOS) and system context menu parts are not
// ported (SPEC §12.10).
//
// Flutter: services/text_input.dart TextInput
type TextInput struct {
	post func(fn func())

	platform       *platformTextInputControl
	currentControl TextInputControl
	inputControls  []TextInputControl

	currentConnection    *TextInputConnection
	currentConfiguration TextInputConfiguration
	lastConnection       *TextInputConnection
	nextID               int
	hidePending          bool
}

// NewTextInput creates a text input service.
//
// Flutter: services/text_input.dart TextInput._
func NewTextInput(o ...TextInputOpts) *TextInput {
	op := first(o)
	t := &TextInput{post: op.Post, nextID: 1}
	t.platform = &platformTextInputControl{input: t, backend: op.Backend}
	t.currentControl = t.platform
	t.inputControls = []TextInputControl{t.platform}
	return t
}

func (t *TextInput) addInputControl(control TextInputControl) {
	if control == TextInputControl(t.platform) {
		return
	}
	for _, c := range t.inputControls {
		if c == control {
			return
		}
	}
	t.inputControls = append(t.inputControls, control)
}

func (t *TextInput) removeInputControl(control TextInputControl) {
	if control == TextInputControl(t.platform) {
		return
	}
	for i, c := range t.inputControls {
		if c == control {
			t.inputControls = append(t.inputControls[:i:i], t.inputControls[i+1:]...)
			return
		}
	}
}

// SetInputControl sets the control that the app uses in place of the input
// method of the platform; nil means none, so that the text input has no current
// control. The platform control still gets all the calls. The attached client
// is told (DidChangeInputControl).
//
// Flutter: services/text_input.dart TextInput.setInputControl
func (t *TextInput) SetInputControl(newControl TextInputControl) {
	oldControl := t.currentControl
	if newControl == oldControl {
		return
	}
	if newControl != nil {
		t.addInputControl(newControl)
	}
	if oldControl != nil {
		t.removeInputControl(oldControl)
	}
	t.currentControl = newControl
	if t.currentConnection != nil {
		t.currentConnection.client.DidChangeInputControl(oldControl, newControl)
	}
}

// RestorePlatformInputControl makes the input method of the platform the
// current control again.
//
// Flutter: services/text_input.dart TextInput.restorePlatformInputControl
func (t *TextInput) RestorePlatformInputControl() { t.SetInputControl(t.platform) }

// Attach connects the client to the text input, with the configuration, and
// returns the connection. The client is the only one connected: the earlier
// connection is no longer attached. The input method shows when the client
// calls Show on the connection.
//
// Flutter: services/text_input.dart TextInput.attach
func (t *TextInput) Attach(client TextInputClient, configuration TextInputConfiguration) *TextInputConnection {
	c := &TextInputConnection{input: t, client: client, id: t.nextID}
	t.nextID++
	t.attach(c, configuration)
	return c
}

// attach tells the controls about the client; it is Attach and the answer to a
// request of the existing input state.
func (t *TextInput) attach(connection *TextInputConnection, configuration TextInputConfiguration) {
	t.currentConnection = connection
	t.currentConfiguration = configuration
	t.lastConnection = connection
	for _, c := range t.controls() {
		c.Attach(connection.client, configuration)
	}
}

// controls returns a copy of the list of controls, so that a control can add or
// remove controls while it is called.
func (t *TextInput) controls() []TextInputControl {
	return append([]TextInputControl(nil), t.inputControls...)
}

func (t *TextInput) scheduleHide() {
	if t.hidePending {
		return
	}
	t.hidePending = true
	hide := func() {
		t.hidePending = false
		if t.currentConnection == nil {
			t.hide()
		}
	}
	// Schedule a deferred task that hides the text input. If someone else shows
	// the keyboard during this update cycle, then the task will do nothing.
	if t.post == nil {
		hide()
		return
	}
	t.post(hide)
}

func (t *TextInput) clearClient() {
	client := t.currentConnection.client
	for _, c := range t.controls() {
		c.Detach(client)
	}
	t.currentConnection = nil
	t.scheduleHide()
}

func (t *TextInput) updateConfig(configuration TextInputConfiguration) {
	t.currentConfiguration = configuration
	for _, c := range t.controls() {
		c.UpdateConfig(configuration)
	}
}

func (t *TextInput) setEditingState(value TextEditingValue) {
	for _, c := range t.controls() {
		c.SetEditingState(value)
	}
}

func (t *TextInput) show() {
	for _, c := range t.controls() {
		c.Show()
	}
}

func (t *TextInput) hide() {
	for _, c := range t.controls() {
		c.Hide()
	}
}

func (t *TextInput) setEditableSizeAndTransform(size ui.Size, transform *ui.Matrix4) {
	for _, c := range t.controls() {
		c.SetEditableSizeAndTransform(size, transform)
	}
}

func (t *TextInput) setComposingTextRect(rect ui.Rect) {
	for _, c := range t.controls() {
		c.SetComposingRect(rect)
	}
}

func (t *TextInput) setCaretRect(rect ui.Rect) {
	for _, c := range t.controls() {
		c.SetCaretRect(rect)
	}
}

func (t *TextInput) setSelectionRects(rects []SelectionRect) {
	for _, c := range t.controls() {
		c.SetSelectionRects(rects)
	}
}

func (t *TextInput) updateStyle(style TextInputStyle) {
	for _, c := range t.controls() {
		c.UpdateStyle(style)
	}
}

func (t *TextInput) requestAutofill() {
	for _, c := range t.controls() {
		c.RequestAutofill()
	}
}

// updateEditingValue gives the value to the other controls and to the client;
// the control that exclude names already has it.
func (t *TextInput) updateEditingValue(value TextEditingValue, exclude TextInputControl) {
	if t.currentConnection == nil {
		return
	}
	for _, c := range t.controls() {
		if c != exclude {
			c.SetEditingState(value)
		}
	}
	t.currentConnection.client.UpdateEditingValue(value)
}

// UpdateEditingValue changes the value of the attached client from the code
// of the app, as the platform would: the controls other than the current one
// are told, and then the client. Nothing happens without a client.
//
// Flutter: services/text_input.dart TextInput.updateEditingValue
func (t *TextInput) UpdateEditingValue(value TextEditingValue) {
	t.updateEditingValue(value, t.currentControl)
}

// FinishAutofillContext ends the autofill context of the controls; shouldSave
// says whether the platform should keep what was filled.
//
// Flutter: services/text_input.dart TextInput.finishAutofillContext
func (t *TextInput) FinishAutofillContext(shouldSave bool) {
	for _, c := range t.controls() {
		c.FinishAutofillContext(shouldSave)
	}
}

// HandleTextInput is the answer to a change that the input method of the
// platform asks of the field (ui.PlatformDispatcher.OnTextInput). The binding
// installs it. Without a client, the event is ignored.
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleTextInput(event ui.TextInputEvent) {
	if t.currentConnection == nil {
		return
	}
	t.platform.handleEvent(event)
}

// clientFor returns the client the platform's message is for: the current
// client if the ids are the same. A message for another client is dropped, as
// Flutter drops it; the id -1 is accepted in a debug build so tests don't have
// to know the id.
func (t *TextInput) clientFor(clientID int) TextInputClient {
	if t.currentConnection == nil {
		return nil
	}
	if clientID != t.currentConnection.id && (!foundation.DebugMode || clientID != -1) {
		return nil
	}
	return t.currentConnection.client
}

// HandleUpdateEditingState is the platform changing the value of the field
// (TextInputClient.updateEditingState): the other controls and the client get
// it.
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleUpdateEditingState(clientID int, value TextEditingValue) {
	if t.clientFor(clientID) == nil {
		return
	}
	t.updateEditingValue(value, t.platform)
}

// HandleUpdateEditingStateWithDeltas is the platform changing the value of the
// field as deltas (TextInputClient.updateEditingStateWithDeltas). The client
// must be a DeltaTextInputClient, or the call panics in a debug build with the
// tag TextInputClientNotDelta.
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleUpdateEditingStateWithDeltas(clientID int, deltas []TextEditingDelta) {
	client := t.clientFor(clientID)
	if client == nil {
		return
	}
	d, ok := client.(DeltaTextInputClient)
	if !ok {
		panic(errors.New("TextInputClientNotDelta", "you must be using a DeltaTextInputClient if TextInputConfiguration.EnableDeltaModel is set to true"))
	}
	d.UpdateEditingValueWithDeltas(deltas)
}

// HandlePerformAction is the platform doing the action of the Enter key
// (TextInputClient.performAction).
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandlePerformAction(clientID int, action TextInputAction) {
	if client := t.clientFor(clientID); client != nil {
		client.PerformAction(action)
	}
}

// HandleCommitContent is the input method inserting a content
// (TextInputClient.performAction with commitContent).
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleCommitContent(clientID int, content KeyboardInsertedContent) {
	if client := t.clientFor(clientID); client != nil {
		client.InsertContent(content)
	}
}

// HandlePerformSelectors is the platform asking for editing commands by the
// names of their selectors (TextInputClient.performSelectors).
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandlePerformSelectors(clientID int, selectors []string) {
	if client := t.clientFor(clientID); client != nil {
		for _, s := range selectors {
			client.PerformSelector(s)
		}
	}
}

// HandlePerformPrivateCommand is the platform sending a private command
// (TextInputClient.performPrivateCommand). A nil data is an empty map.
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandlePerformPrivateCommand(clientID int, action string, data map[string]any) {
	if client := t.clientFor(clientID); client != nil {
		if data == nil {
			data = map[string]any{}
		}
		client.PerformPrivateCommand(action, data)
	}
}

// HandleUpdateFloatingCursor is the platform moving the floating cursor
// (TextInputClient.updateFloatingCursor).
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleUpdateFloatingCursor(clientID int, point RawFloatingCursorPoint) {
	if client := t.clientFor(clientID); client != nil {
		client.UpdateFloatingCursor(point)
	}
}

// HandleConnectionClosed is the platform closing the connection
// (TextInputClient.onConnectionClosed).
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleConnectionClosed(clientID int) {
	if client := t.clientFor(clientID); client != nil {
		client.ConnectionClosed()
	}
}

// HandleShowAutocorrectionPromptRect is the platform asking to mark an
// autocorrection (TextInputClient.showAutocorrectionPromptRect).
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleShowAutocorrectionPromptRect(clientID, start, end int) {
	if client := t.clientFor(clientID); client != nil {
		client.ShowAutocorrectionPromptRect(start, end)
	}
}

// HandleShowToolbar is the platform asking for the selection toolbar
// (TextInputClient.showToolbar).
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleShowToolbar(clientID int) {
	if client := t.clientFor(clientID); client != nil {
		client.ShowToolbar()
	}
}

// HandleInsertTextPlaceholder is the platform asking for room for a placeholder
// (TextInputClient.insertTextPlaceholder).
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleInsertTextPlaceholder(clientID int, size ui.Size) {
	if client := t.clientFor(clientID); client != nil {
		client.InsertTextPlaceholder(size)
	}
}

// HandleRemoveTextPlaceholder is the platform taking its placeholder away
// (TextInputClient.removeTextPlaceholder).
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleRemoveTextPlaceholder(clientID int) {
	if client := t.clientFor(clientID); client != nil {
		client.RemoveTextPlaceholder()
	}
}

// HandleFocusReceived is the platform asking the field of the last connection
// to take the focus (TextInputClient.onFocusReceived). It returns what the
// client returns, and false when the id isn't the one of the last connection.
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleFocusReceived(clientID int) bool {
	if t.lastConnection != nil && t.lastConnection.id == clientID {
		return t.lastConnection.client.OnFocusReceived()
	}
	return false
}

// HandleRequestExistingInputState is the platform asking for the state of the
// field again (TextInputClient.requestExistingInputState): the current client
// is attached again and its value is sent.
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleRequestExistingInputState() {
	if t.currentConnection == nil {
		return
	}
	t.attach(t.currentConnection, t.currentConfiguration)
	if value, ok := t.currentConnection.client.CurrentTextEditingValue(); ok {
		t.setEditingState(value)
	}
}

// HandleUpdateEditingStateWithTag is the autofill of the platform
// (TextInputClient.updateEditingStateWithTag): each value goes to the field of
// the scope with that id, if it enables autofill.
//
// Flutter: services/text_input.dart TextInput._handleTextInputInvocation
func (t *TextInput) HandleUpdateEditingStateWithTag(values map[string]TextEditingValue) {
	if t.currentConnection == nil {
		return
	}
	scope := t.currentConnection.client.CurrentAutofillScope()
	if scope == nil {
		return
	}
	for tag, value := range values {
		if c := scope.GetAutofillClient(tag); c != nil && c.TextInputConfiguration().AutofillConfiguration.Enabled {
			c.Autofill(value)
		}
	}
}

// TextInputConnection is the connection of a client to the text input, made by
// TextInput.Attach. The client uses it to tell the controls about its state.
//
// Flutter: services/text_input.dart TextInputConnection
type TextInputConnection struct {
	input  *TextInput
	client TextInputClient
	id     int

	cachedSize          *ui.Size
	cachedTransform     *ui.Matrix4
	cachedRect          *ui.Rect
	cachedCaretRect     *ui.Rect
	cachedSelectionRect []SelectionRect
}

// ID is the identifier of the connection; the platform's messages carry the
// id of the client they are for.
//
// Flutter: services/text_input.dart TextInputConnection._id
func (c *TextInputConnection) ID() int { return c.id }

// Attached is whether this connection is the current one of the text input.
//
// Flutter: services/text_input.dart TextInputConnection.attached
func (c *TextInputConnection) Attached() bool { return c.input.currentConnection == c }

func (c *TextInputConnection) mustBeAttached() {
	if foundation.DebugMode && !c.Attached() {
		panic(errors.New("TextInputConnectionDetached", "the text input connection is no longer attached"))
	}
}

// Show asks for the input method to show. The connection must be attached, or
// the call panics in a debug build with the tag TextInputConnectionDetached.
//
// Flutter: services/text_input.dart TextInputConnection.show
func (c *TextInputConnection) Show() {
	c.mustBeAttached()
	c.input.show()
}

// RequestAutofill asks for autofill.
//
// Flutter: services/text_input.dart TextInputConnection.requestAutofill
func (c *TextInputConnection) RequestAutofill() {
	c.mustBeAttached()
	c.input.requestAutofill()
}

// UpdateConfig tells the controls the configuration of the client changed.
//
// Flutter: services/text_input.dart TextInputConnection.updateConfig
func (c *TextInputConnection) UpdateConfig(configuration TextInputConfiguration) {
	c.mustBeAttached()
	c.input.updateConfig(configuration)
}

// SetEditingState tells the controls the value of the client changed.
//
// Flutter: services/text_input.dart TextInputConnection.setEditingState
func (c *TextInputConnection) SetEditingState(value TextEditingValue) {
	c.mustBeAttached()
	c.input.setEditingState(value)
}

// SetEditableSizeAndTransform tells the controls the size of the editable and
// the transform from its coordinates to the view's. Nothing is sent if neither
// changed.
//
// Flutter: services/text_input.dart TextInputConnection.setEditableSizeAndTransform
func (c *TextInputConnection) SetEditableSizeAndTransform(editableBoxSize ui.Size, transform *ui.Matrix4) {
	if c.cachedSize != nil && *c.cachedSize == editableBoxSize && c.cachedTransform != nil && c.cachedTransform.Equals(transform) {
		return
	}
	c.cachedSize = &editableBoxSize
	c.cachedTransform = transform.Clone()
	c.input.setEditableSizeAndTransform(editableBoxSize, transform)
}

// finiteRect is the rectangle that the controls get: the one given, or the
// (0, 0, -1, -1) rectangle of Flutter if it isn't finite.
func finiteRect(rect ui.Rect) ui.Rect {
	if rect.IsFinite() {
		return rect
	}
	return ui.NewRectFromLTWH(0, 0, -1, -1)
}

// SetComposingRect tells the controls the rectangle of the composing text, in
// the coordinates of the editable. Nothing is sent if it didn't change.
//
// Flutter: services/text_input.dart TextInputConnection.setComposingRect
func (c *TextInputConnection) SetComposingRect(rect ui.Rect) {
	if c.cachedRect != nil && *c.cachedRect == rect {
		return
	}
	c.cachedRect = &rect
	c.input.setComposingTextRect(finiteRect(rect))
}

// SetCaretRect tells the controls the rectangle of the caret, in the
// coordinates of the editable. Nothing is sent if it didn't change.
//
// Flutter: services/text_input.dart TextInputConnection.setCaretRect
func (c *TextInputConnection) SetCaretRect(rect ui.Rect) {
	if c.cachedCaretRect != nil && *c.cachedCaretRect == rect {
		return
	}
	c.cachedCaretRect = &rect
	c.input.setCaretRect(finiteRect(rect))
}

// SetSelectionRects tells the controls the rectangles of the characters.
// Nothing is sent if they didn't change.
//
// Flutter: services/text_input.dart TextInputConnection.setSelectionRects
func (c *TextInputConnection) SetSelectionRects(selectionRects []SelectionRect) {
	if equalSelectionRects(c.cachedSelectionRect, selectionRects) {
		return
	}
	c.cachedSelectionRect = append([]SelectionRect(nil), selectionRects...)
	c.input.setSelectionRects(selectionRects)
}

func equalSelectionRects(a, b []SelectionRect) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// UpdateStyle tells the controls the style of the text.
//
// Flutter: services/text_input.dart TextInputConnection.updateStyle
func (c *TextInputConnection) UpdateStyle(style TextInputStyle) {
	c.mustBeAttached()
	c.input.updateStyle(style)
}

// Close ends the connection: the controls are told the client is gone, and the
// input method hides once the update cycle ends, unless a client attaches
// before.
//
// Flutter: services/text_input.dart TextInputConnection.close
func (c *TextInputConnection) Close() {
	if c.Attached() {
		c.input.clearClient()
	}
}

// ConnectionClosedReceived is the platform closing the connection: it is no
// longer attached, and the controls are not told.
//
// Flutter: services/text_input.dart TextInputConnection.connectionClosedReceived
func (c *TextInputConnection) ConnectionClosedReceived() {
	c.input.currentConnection = nil
}
