package services_test

import (
	"fmt"
	"slices"

	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

// fakeBackend records the states the platform control sends to the platform.
type fakeBackend struct{ states []ui.TextInputState }

func (b *fakeBackend) SetTextInput(s ui.TextInputState) { b.states = append(b.states, s) }

func (b *fakeBackend) last() ui.TextInputState {
	if len(b.states) == 0 {
		return ui.TextInputState{}
	}
	return b.states[len(b.states)-1]
}

// fakeControl is a TextInputControl that records its calls.
type fakeControl struct {
	services.TextInputControlDefaults
	calls []string
}

func (c *fakeControl) Attach(services.TextInputClient, services.TextInputConfiguration) {
	c.calls = append(c.calls, "attach")
}
func (c *fakeControl) Detach(services.TextInputClient) { c.calls = append(c.calls, "detach") }
func (c *fakeControl) Show()                           { c.calls = append(c.calls, "show") }
func (c *fakeControl) Hide()                           { c.calls = append(c.calls, "hide") }
func (c *fakeControl) UpdateConfig(services.TextInputConfiguration) {
	c.calls = append(c.calls, "updateConfig")
}
func (c *fakeControl) SetEditingState(services.TextEditingValue) {
	c.calls = append(c.calls, "setEditingState")
}
func (c *fakeControl) SetEditableSizeAndTransform(ui.Size, *ui.Matrix4) {
	c.calls = append(c.calls, "setEditableSizeAndTransform")
}
func (c *fakeControl) SetComposingRect(ui.Rect) { c.calls = append(c.calls, "setComposingRect") }
func (c *fakeControl) SetCaretRect(ui.Rect)     { c.calls = append(c.calls, "setCaretRect") }
func (c *fakeControl) SetSelectionRects([]services.SelectionRect) {
	c.calls = append(c.calls, "setSelectionRects")
}
func (c *fakeControl) UpdateStyle(services.TextInputStyle) { c.calls = append(c.calls, "updateStyle") }
func (c *fakeControl) RequestAutofill()                    { c.calls = append(c.calls, "requestAutofill") }
func (c *fakeControl) FinishAutofillContext(shouldSave bool) {
	c.calls = append(c.calls, fmt.Sprintf("finishAutofillContext %v", shouldSave))
}

// fakeClient is a TextInputClient that records its calls.
type fakeClient struct {
	services.TextInputClientDefaults
	value      services.TextEditingValue
	hasValue   bool
	scope      services.AutofillScope
	calls      []string
	last       string
	focus      bool
	values     []services.TextEditingValue
	deltas     [][]services.TextEditingDelta
	actions    []services.TextInputAction
	selectors  []string
	private    []map[string]any
	contents   []services.KeyboardInsertedContent
	floating   []services.RawFloatingCursorPoint
	controlOld services.TextInputControl
	controlNew services.TextInputControl
}

func newFakeClient(v services.TextEditingValue) *fakeClient {
	return &fakeClient{value: v, hasValue: true}
}

func (c *fakeClient) record(name string) { c.calls = append(c.calls, name); c.last = name }

func (c *fakeClient) CurrentTextEditingValue() (services.TextEditingValue, bool) {
	return c.value, c.hasValue
}
func (c *fakeClient) CurrentAutofillScope() services.AutofillScope { return c.scope }
func (c *fakeClient) UpdateEditingValue(v services.TextEditingValue) {
	c.record("updateEditingValue")
	c.values = append(c.values, v)
}
func (c *fakeClient) PerformAction(a services.TextInputAction) {
	c.record("performAction")
	c.actions = append(c.actions, a)
}
func (c *fakeClient) InsertContent(k services.KeyboardInsertedContent) {
	c.record("insertContent")
	c.contents = append(c.contents, k)
}
func (c *fakeClient) PerformPrivateCommand(action string, data map[string]any) {
	c.record("performPrivateCommand " + action)
	c.private = append(c.private, data)
}
func (c *fakeClient) UpdateFloatingCursor(p services.RawFloatingCursorPoint) {
	c.record("updateFloatingCursor")
	c.floating = append(c.floating, p)
}
func (c *fakeClient) ShowAutocorrectionPromptRect(start, end int) {
	c.record(fmt.Sprintf("showAutocorrectionPromptRect %d %d", start, end))
}
func (c *fakeClient) OnFocusReceived() bool { c.record("onFocusReceived"); return c.focus }
func (c *fakeClient) ConnectionClosed()     { c.record("connectionClosed") }
func (c *fakeClient) DidChangeInputControl(o, n services.TextInputControl) {
	c.record("didChangeInputControl")
	c.controlOld, c.controlNew = o, n
}
func (c *fakeClient) ShowToolbar() { c.record("showToolbar") }
func (c *fakeClient) InsertTextPlaceholder(s ui.Size) {
	c.record(fmt.Sprintf("insertTextPlaceholder %v", s))
}
func (c *fakeClient) RemoveTextPlaceholder() { c.record("removeTextPlaceholder") }
func (c *fakeClient) PerformSelector(name string) {
	c.record("performSelector " + name)
	c.selectors = append(c.selectors, name)
}

// fakeDeltaClient is a fakeClient that takes deltas.
type fakeDeltaClient struct{ *fakeClient }

func (c fakeDeltaClient) UpdateEditingValueWithDeltas(d []services.TextEditingDelta) {
	c.record("updateEditingValueWithDeltas")
	c.deltas = append(c.deltas, slices.Clone(d))
}

// fakeScope is an AutofillScope over some clients.
type fakeScope struct {
	clients []services.AutofillClient
}

func (s *fakeScope) GetAutofillClient(id string) services.AutofillClient {
	for _, c := range s.clients {
		if c.AutofillID() == id {
			return c
		}
	}
	return nil
}
func (s *fakeScope) AutofillClients() []services.AutofillClient { return s.clients }
func (s *fakeScope) Attach(ti *services.TextInput, trigger services.TextInputClient, cfg services.TextInputConfiguration) *services.TextInputConnection {
	return services.AutofillScopeAttach(s, ti, trigger, cfg)
}

type fakeAutofillClient struct {
	id     string
	config services.TextInputConfiguration
	filled []services.TextEditingValue
}

func (c *fakeAutofillClient) AutofillID() string { return c.id }
func (c *fakeAutofillClient) TextInputConfiguration() services.TextInputConfiguration {
	return c.config
}
func (c *fakeAutofillClient) Autofill(v services.TextEditingValue) { c.filled = append(c.filled, v) }
