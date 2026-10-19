package services_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
)

func TestTextInputTypes(t *testing.T) {
	values := services.TextInputTypeValues()
	if len(values) != 13 {
		t.Fatalf("%d types, want 13", len(values))
	}
	for i, v := range values {
		if v.Index() != i {
			t.Errorf("type %d has the index %d", i, v.Index())
		}
	}
	if services.TextInputTypeNumber != services.NewTextInputTypeNumberWithOptions() {
		t.Error("number is the number without options")
	}
	signed := services.NewTextInputTypeNumberWithOptions(services.TextInputTypeNumberOpts{Signed: true})
	if signed == services.TextInputTypeNumber || signed.Index() != services.TextInputTypeNumber.Index() {
		t.Errorf("signed = %v", signed)
	}
	if got := services.TextInputTypeText.String(); got != "TextInputType(name: TextInputType.text, signed: null, decimal: null)" {
		t.Errorf("String = %q", got)
	}
	if got := signed.String(); got != "TextInputType(name: TextInputType.number, signed: true, decimal: false)" {
		t.Errorf("String = %q", got)
	}
	if got := services.TextInputTypeStreetAddress.String(); !strings.Contains(got, "TextInputType.address") {
		t.Errorf("String = %q, want the name address", got)
	}
	if (services.TextInputType{}) != services.TextInputTypeText {
		t.Error("the zero TextInputType is text")
	}
}

func TestTextInputConfigurationDefaults(t *testing.T) {
	c := services.NewTextInputConfiguration()
	want := services.TextInputConfiguration{
		InputType: services.TextInputTypeText, Autocorrect: true,
		SmartDashesType: services.SmartDashesTypeEnabled, SmartQuotesType: services.SmartQuotesTypeEnabled,
		EnableSuggestions: true, EnableInteractiveSelection: true, InputAction: services.TextInputActionDone,
		KeyboardAppearance: ui.BrightnessLight, TextCapitalization: services.TextCapitalizationNone,
		AutofillConfiguration: services.AutofillConfigurationDisabled, EnableIMEPersonalizedLearning: true,
	}
	if !c.Equal(want) {
		t.Errorf("configuration = %v\nwant %v", c, want)
	}
	if c.ReadOnly || c.ObscureText || c.EnableDeltaModel || c.ActionLabel.IsSome() || c.EnableInlinePrediction.IsSome() {
		t.Errorf("configuration = %v", c)
	}
	obscured := services.NewTextInputConfiguration(services.TextInputConfigurationOpts{ObscureText: true})
	if obscured.SmartDashesType != services.SmartDashesTypeDisabled || obscured.SmartQuotesType != services.SmartQuotesTypeDisabled {
		t.Errorf("obscured text keeps smart dashes and quotes: %v", obscured)
	}
	explicit := services.NewTextInputConfiguration(services.TextInputConfigurationOpts{
		ObscureText: true, SmartDashesType: services.SmartDashesTypeEnabled, SmartQuotesType: services.SmartQuotesTypeEnabled,
		DisableAutocorrect: true, DisableSuggestions: true, DisableInteractiveSelection: true, DisableIMEPersonalizedLearning: true,
	})
	if explicit.SmartDashesType != services.SmartDashesTypeEnabled || explicit.SmartQuotesType != services.SmartQuotesTypeEnabled ||
		explicit.Autocorrect || explicit.EnableSuggestions || explicit.EnableInteractiveSelection || explicit.EnableIMEPersonalizedLearning {
		t.Errorf("explicit = %v", explicit)
	}
}

func TestTextInputConfigurationEqualityAndCopyWith(t *testing.T) {
	base := services.NewTextInputConfiguration(services.TextInputConfigurationOpts{
		ViewID: foundation.Some(1), ActionLabel: foundation.Some("label1"),
		SmartDashesType: services.SmartDashesTypeEnabled, SmartQuotesType: services.SmartQuotesTypeEnabled,
		AllowedMimeTypes: []string{"text/plain", "application/pdf"},
	})
	same := base.CopyWith(services.TextInputConfigurationCopy{})
	if !base.Equal(same) {
		t.Errorf("a copy with nothing changed differs: %v vs %v", base, same)
	}
	changes := map[string]services.TextInputConfigurationCopy{
		"viewId":                        {ViewID: foundation.Some(2)},
		"inputType":                     {InputType: foundation.Some(services.TextInputTypeEmailAddress)},
		"readOnly":                      {ReadOnly: foundation.Some(true)},
		"obscureText":                   {ObscureText: foundation.Some(true)},
		"autocorrect":                   {Autocorrect: foundation.Some(false)},
		"smartDashesType":               {SmartDashesType: services.SmartDashesTypeDisabled},
		"smartQuotesType":               {SmartQuotesType: services.SmartQuotesTypeDisabled},
		"enableSuggestions":             {EnableSuggestions: foundation.Some(false)},
		"enableInteractiveSelection":    {EnableInteractiveSelection: foundation.Some(false)},
		"actionLabel":                   {ActionLabel: foundation.Some("label2")},
		"inputAction":                   {InputAction: services.TextInputActionSearch},
		"keyboardAppearance":            {KeyboardAppearance: ui.BrightnessDark},
		"textCapitalization":            {TextCapitalization: services.TextCapitalizationWords},
		"enableIMEPersonalizedLearning": {EnableIMEPersonalizedLearning: foundation.Some(false)},
		"allowedMimeTypes":              {AllowedMimeTypes: foundation.Some([]string{"image/png"})},
		"autofillConfiguration": {AutofillConfiguration: foundation.Some(services.NewAutofillConfiguration("id",
			[]string{services.AutofillHintEmail}, services.TextEditingValueEmpty))},
		"enableDeltaModel":       {EnableDeltaModel: foundation.Some(true)},
		"hintLocales":            {HintLocales: foundation.Some([]ui.Locale{{LanguageCode: "el"}})},
		"enableInlinePrediction": {EnableInlinePrediction: foundation.Some(true)},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			got := base.CopyWith(change)
			if got.Equal(base) || base.Equal(got) {
				t.Errorf("changing %s changed nothing: %v", name, got)
			}
		})
	}
	// Fields that are not given are kept.
	got := base.CopyWith(services.TextInputConfigurationCopy{InputAction: services.TextInputActionSend})
	if v, _ := got.ViewID.Get(); v != 1 || got.ActionLabel != foundation.Some("label1") || got.InputAction != services.TextInputActionSend {
		t.Errorf("copy = %v", got)
	}
}

func TestTextInputConfigurationString(t *testing.T) {
	c := services.NewTextInputConfiguration(services.TextInputConfigurationOpts{
		ViewID: foundation.Some(3), ActionLabel: foundation.Some("go"), HintLocales: []ui.Locale{{LanguageCode: "el"}},
		EnableInlinePrediction: foundation.Some(true),
	})
	s := c.String()
	for _, want := range []string{"TextInputConfiguration(", "viewId: 3", "actionLabel: go", "inputAction: TextInputActionDone",
		"hintLocales:", "enableInlinePrediction: true", "keyboardAppearance: BrightnessLight"} {
		if !strings.Contains(s, want) {
			t.Errorf("String lacks %q: %s", want, s)
		}
	}
}

func TestEnumNames(t *testing.T) {
	if got := services.TextInputActionDone.String(); got != "TextInputActionDone" {
		t.Errorf("String = %q", got)
	}
	if got := services.TextInputAction(99).String(); got != "TextInputAction(99)" {
		t.Errorf("String = %q", got)
	}
}

func TestTextInputStyle(t *testing.T) {
	a := services.NewTextInputStyle(ui.TextDirectionLTR, ui.TextAlignLeft, services.TextInputStyleOpts{
		FontFamily: foundation.Some("Roboto"), FontSize: foundation.Some(14.0), FontWeight: ui.FontWeightW500,
		LetterSpacing: foundation.Some(1.0), WordSpacing: foundation.Some(2.0), LineHeight: foundation.Some(1.5),
	})
	b := a
	if a != b {
		t.Error("a copy of a style differs")
	}
	b.FontSize = foundation.Some(15.0)
	if a == b {
		t.Error("styles with another font size are equal")
	}
	if !foundation.DebugMode {
		return // the diagnostics are for debug builds
	}
	builder := foundation.NewDiagnosticPropertiesBuilder()
	a.DebugFillProperties(builder)
	var got []string
	for _, n := range builder.Properties {
		if !n.IsFiltered(foundation.DiagnosticLevelInfo) {
			got = append(got, n.String())
		}
	}
	for _, want := range []string{`fontFamily: "Roboto"`, "fontSize: 14.0", "letterSpacing: 1.0", "wordSpacing: 2.0", "lineHeight: 1.5"} {
		found := false
		for _, g := range got {
			found = found || g == want
		}
		if !found {
			t.Errorf("properties %q lack %q", got, want)
		}
	}
	empty := services.NewTextInputStyle(ui.TextDirectionRTL, ui.TextAlignRight)
	builder = foundation.NewDiagnosticPropertiesBuilder()
	empty.DebugFillProperties(builder)
	for _, n := range builder.Properties {
		if !n.IsFiltered(foundation.DiagnosticLevelInfo) && strings.HasPrefix(n.String(), "fontFamily") {
			t.Errorf("a style without a font family prints %q", n.String())
		}
	}
}

func TestRawFloatingCursorPointAndSelectionRect(t *testing.T) {
	p := services.NewRawFloatingCursorPoint(services.FloatingCursorDragStateUpdate, services.RawFloatingCursorPointOpts{Offset: foundation.Some(ui.Offset{Dx: 1, Dy: 2})})
	if o, _ := p.Offset.Get(); o != (ui.Offset{Dx: 1, Dy: 2}) || p.State != services.FloatingCursorDragStateUpdate {
		t.Errorf("point = %+v", p)
	}
	if foundation.DebugMode {
		if tag := panicTag(t, func() { services.NewRawFloatingCursorPoint(services.FloatingCursorDragStateUpdate) }); tag != "RawFloatingCursorPointOffsetMissing" {
			t.Errorf("an update without an offset panicked with %q", tag)
		}
	}
	services.NewRawFloatingCursorPoint(services.FloatingCursorDragStateStart) // needs no offset
	r := services.NewSelectionRect(3, ui.NewRectFromLTWH(1, 2, 3, 4))
	if r.Position != 3 || r.Direction != ui.TextDirectionUnspecified || r.String() != "SelectionRect(3, "+r.Bounds.String()+")" {
		t.Errorf("rect = %v", r)
	}
	if r2 := services.NewSelectionRect(3, r.Bounds, ui.TextDirectionRTL); r2.Direction != ui.TextDirectionRTL {
		t.Errorf("rect = %v", r2)
	}
}

func TestKeyboardInsertedContent(t *testing.T) {
	a := services.NewKeyboardInsertedContent("image/png", "content://x", []byte{1, 2})
	if !a.HasData() || services.NewKeyboardInsertedContent("image/png", "u", nil).HasData() {
		t.Error("HasData is wrong")
	}
	if !a.Equal(services.NewKeyboardInsertedContent("image/png", "content://x", []byte{1, 2})) ||
		a.Equal(services.NewKeyboardInsertedContent("image/png", "content://x", []byte{1})) ||
		a.Equal(services.NewKeyboardInsertedContent("image/gif", "content://x", []byte{1, 2})) {
		t.Error("Equal is wrong")
	}
	if got := a.String(); got != "KeyboardInsertedContent(image/png, content://x, [1 2])" {
		t.Errorf("String = %q", got)
	}
}

func TestTextLayoutMetricsCharacterClasses(t *testing.T) {
	for _, r := range []rune{'\t', '\n', ' ', 0xA0, 0x2003, 0x3000} {
		if !services.TextLayoutMetricsIsWhitespace(r) {
			t.Errorf("%U is whitespace", r)
		}
	}
	for _, r := range []rune{'a', '0', 0x85, 0x2028, '-'} {
		if services.TextLayoutMetricsIsWhitespace(r) {
			t.Errorf("%U is not whitespace", r)
		}
	}
	for _, r := range []rune{'\n', 0x0B, 0x0C, '\r', 0x85, 0x2028, 0x2029} {
		if !services.TextLayoutMetricsIsLineTerminator(r) {
			t.Errorf("%U terminates a line", r)
		}
	}
	for _, r := range []rune{'a', ' ', '\t'} {
		if services.TextLayoutMetricsIsLineTerminator(r) {
			t.Errorf("%U does not terminate a line", r)
		}
	}
}

// later is a Post whose tasks have not run yet.
func later(func()) {}

func TestTextInputControlGetsAttachedAndDetached(t *testing.T) {
	ti := services.NewTextInput(services.TextInputOpts{Post: later})
	control := &fakeControl{}
	ti.SetInputControl(control)
	conn := ti.Attach(newFakeClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration())
	want := []string{"attach"}
	if !reflect.DeepEqual(control.calls, want) {
		t.Fatalf("calls = %v, want %v", control.calls, want)
	}
	conn.Close()
	want = append(want, "detach")
	if !reflect.DeepEqual(control.calls, want) {
		t.Errorf("calls = %v, want %v", control.calls, want)
	}
}

func TestTextInputControlReceivesStateChanges(t *testing.T) {
	ti := services.NewTextInput(services.TextInputOpts{Post: later})
	control := &fakeControl{}
	ti.SetInputControl(control)
	conn := ti.Attach(newFakeClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration())
	control.calls = nil

	conn.UpdateConfig(services.NewTextInputConfiguration())
	conn.SetEditingState(services.TextEditingValueEmpty)
	conn.Show()
	conn.RequestAutofill()
	conn.UpdateStyle(services.NewTextInputStyle(ui.TextDirectionLTR, ui.TextAlignLeft))
	conn.SetEditableSizeAndTransform(ui.Size{Width: 10, Height: 10}, ui.NewMatrix4Identity())
	conn.SetComposingRect(ui.NewRectFromLTWH(0, 0, 1, 1))
	conn.SetCaretRect(ui.NewRectFromLTWH(0, 0, 1, 1))
	conn.SetSelectionRects([]services.SelectionRect{services.NewSelectionRect(0, ui.NewRectFromLTWH(0, 0, 1, 1))})
	ti.FinishAutofillContext(true)
	conn.Close()
	want := []string{"updateConfig", "setEditingState", "show", "requestAutofill", "updateStyle",
		"setEditableSizeAndTransform", "setComposingRect", "setCaretRect", "setSelectionRects", "finishAutofillContext true", "detach"}
	if !reflect.DeepEqual(control.calls, want) {
		t.Errorf("calls = %v\nwant %v", control.calls, want)
	}
}

func TestTextInputControlDoesNotInterfereWithThePlatformInput(t *testing.T) {
	backend := &fakeBackend{}
	ti := services.NewTextInput(services.TextInputOpts{Backend: backend})
	control := &fakeControl{}
	ti.SetInputControl(control)
	client := newFakeClient(services.TextEditingValueEmpty)
	conn := ti.Attach(client, services.NewTextInputConfiguration())
	backend.states = nil

	ti.HandleUpdateEditingState(conn.ID(), services.TextEditingValueEmpty)
	if client.last != "updateEditingValue" {
		t.Errorf("the client's last call is %q", client.last)
	}
	if want := []string{"attach", "setEditingState"}; !reflect.DeepEqual(control.calls, want) {
		t.Errorf("control calls = %v, want %v", control.calls, want)
	}
	// The change came from the platform, so the platform is not told.
	if len(backend.states) != 0 {
		t.Errorf("the platform got %d states back", len(backend.states))
	}
}

func TestBothInputControlsReceiveRequests(t *testing.T) {
	backend := &fakeBackend{}
	ti := services.NewTextInput(services.TextInputOpts{Backend: backend})
	control := &fakeControl{}
	ti.SetInputControl(control)
	client := newFakeClient(services.TextEditingValueEmpty)
	conn := ti.Attach(client, services.NewTextInputConfiguration())
	conn.Show()
	// The custom control is current, so the platform's input method stays off.
	if backend.last().Enabled {
		t.Error("the input method of the platform is on while the app has its own control")
	}
	ti.RestorePlatformInputControl()
	conn2 := ti.Attach(client, services.NewTextInputConfiguration())
	conn2.Show()
	if !backend.last().Enabled {
		t.Error("the input method of the platform is off after it was restored")
	}
	if got := control.calls; len(got) != 2 || got[0] != "attach" || got[1] != "show" {
		t.Errorf("the removed control got %v after the restore, want only attach and show", got)
	}
}

func TestTextInputNotifiesTheAttachedClientOfAControlChange(t *testing.T) {
	ti := services.NewTextInput()
	control := &fakeControl{}
	ti.SetInputControl(control)
	client := newFakeClient(services.TextEditingValueEmpty)
	conn := ti.Attach(client, services.NewTextInputConfiguration())
	ti.SetInputControl(nil)
	if client.last != "didChangeInputControl" || client.controlOld != services.TextInputControl(control) || client.controlNew != nil {
		t.Errorf("client = %+v", client)
	}
	conn.Show()
	if client.last != "didChangeInputControl" {
		t.Errorf("the client's last call is %q", client.last)
	}
	// Setting the same control again changes nothing.
	n := len(client.calls)
	ti.SetInputControl(nil)
	if len(client.calls) != n {
		t.Error("the client was told of a control that did not change")
	}
}

func TestTextInputUpdateEditingValueFromTheApp(t *testing.T) {
	ti := services.NewTextInput()
	control := &fakeControl{}
	ti.SetInputControl(control)
	client := newFakeClient(services.TextEditingValueEmpty)
	ti.UpdateEditingValue(val("x", services.NewTextSelectionCollapsed(1))) // no client yet: ignored
	if len(client.values) != 0 {
		t.Fatal("a value reached a client that is not attached")
	}
	ti.Attach(client, services.NewTextInputConfiguration())
	control.calls = nil
	ti.UpdateEditingValue(val("x", services.NewTextSelectionCollapsed(1)))
	if len(client.values) != 1 || client.values[0].Text != "x" {
		t.Errorf("client values = %v", client.values)
	}
	// The current control is the custom one, so it is the one that already
	// has the value, and the platform control gets it.
	if len(control.calls) != 0 {
		t.Errorf("the current control was told %v", control.calls)
	}
}

func TestTextInputHandlersGoToTheCurrentClient(t *testing.T) {
	ti := services.NewTextInput()
	client := newFakeClient(services.TextEditingValueEmpty)
	conn := ti.Attach(client, services.NewTextInputConfiguration())
	id := conn.ID()

	ti.HandlePerformAction(id, services.TextInputActionSearch)
	ti.HandleCommitContent(id, services.NewKeyboardInsertedContent("image/png", "u", nil))
	ti.HandlePerformSelectors(id, []string{"moveLeft:", "deleteBackward:"})
	ti.HandlePerformPrivateCommand(id, "cmd", map[string]any{"k": 1.5})
	ti.HandlePerformPrivateCommand(id, "none", nil)
	ti.HandleUpdateFloatingCursor(id, services.NewRawFloatingCursorPoint(services.FloatingCursorDragStateEnd))
	ti.HandleShowAutocorrectionPromptRect(id, 1, 4)
	ti.HandleShowToolbar(id)
	ti.HandleInsertTextPlaceholder(id, ui.Size{Width: 3, Height: 4})
	ti.HandleRemoveTextPlaceholder(id)
	ti.HandleConnectionClosed(id)

	want := []string{"performAction", "insertContent", "performSelector moveLeft:", "performSelector deleteBackward:",
		"performPrivateCommand cmd", "performPrivateCommand none", "updateFloatingCursor", "showAutocorrectionPromptRect 1 4",
		"showToolbar", "insertTextPlaceholder Size(3.0, 4.0)", "removeTextPlaceholder", "connectionClosed"}
	if !reflect.DeepEqual(client.calls, want) {
		t.Errorf("calls = %q\nwant %q", client.calls, want)
	}
	if client.actions[0] != services.TextInputActionSearch {
		t.Errorf("action = %v", client.actions[0])
	}
	if client.private[0]["k"] != 1.5 || client.private[1] == nil || len(client.private[1]) != 0 {
		t.Errorf("private data = %v", client.private)
	}
}

func TestTextInputIgnoresMessagesForAnotherClient(t *testing.T) {
	ti := services.NewTextInput()
	client := newFakeClient(services.TextEditingValueEmpty)
	conn := ti.Attach(client, services.NewTextInputConfiguration())
	other := conn.ID() + 5
	ti.HandlePerformAction(other, services.TextInputActionDone)
	ti.HandleUpdateEditingState(other, services.TextEditingValueEmpty)
	ti.HandleShowToolbar(other)
	if len(client.calls) != 0 {
		t.Errorf("calls = %q, want none", client.calls)
	}
	// In a debug build the id -1 reaches the current client.
	ti.HandlePerformAction(-1, services.TextInputActionDone)
	if foundation.DebugMode && len(client.calls) != 1 {
		t.Errorf("calls = %q, want the one for -1", client.calls)
	}
	// After the connection closed there is no client.
	n := len(client.calls)
	conn.Close()
	ti.HandlePerformAction(conn.ID(), services.TextInputActionDone)
	if len(client.calls) != n {
		t.Error("a closed connection got a call")
	}
}

func TestTextInputDeltasNeedADeltaClient(t *testing.T) {
	ti := services.NewTextInput()
	plain := newFakeClient(services.TextEditingValueEmpty)
	conn := ti.Attach(plain, services.NewTextInputConfiguration())
	if tag := panicTag(t, func() { ti.HandleUpdateEditingStateWithDeltas(conn.ID(), nil) }); tag != "TextInputClientNotDelta" {
		t.Errorf("panicked with %q", tag)
	}
	deltaClient := fakeDeltaClient{newFakeClient(services.TextEditingValueEmpty)}
	conn = ti.Attach(deltaClient, services.NewTextInputConfiguration())
	d := services.NewTextEditingDeltaNonTextUpdate("", services.NewTextSelectionCollapsed(0), ui.TextRangeEmpty)
	ti.HandleUpdateEditingStateWithDeltas(conn.ID(), []services.TextEditingDelta{d})
	if len(deltaClient.deltas) != 1 || len(deltaClient.deltas[0]) != 1 {
		t.Errorf("deltas = %v", deltaClient.deltas)
	}
	// Another client's message is dropped before the check.
	ti.HandleUpdateEditingStateWithDeltas(conn.ID()+3, nil)
}

func TestTextInputFocusReceived(t *testing.T) {
	ti := services.NewTextInput()
	client := newFakeClient(services.TextEditingValueEmpty)
	client.focus = true
	conn := ti.Attach(client, services.NewTextInputConfiguration())
	if !ti.HandleFocusReceived(conn.ID()) {
		t.Error("the client takes the focus")
	}
	if ti.HandleFocusReceived(conn.ID() + 1) {
		t.Error("another id got the focus")
	}
	// The last connection is asked even when it isn't attached any more.
	conn.Close()
	if !ti.HandleFocusReceived(conn.ID()) {
		t.Error("the last connection is still asked")
	}
	client.focus = false
	if ti.HandleFocusReceived(conn.ID()) {
		t.Error("the client refused the focus")
	}
}

func TestTextInputRequestExistingInputState(t *testing.T) {
	ti := services.NewTextInput()
	control := &fakeControl{}
	ti.SetInputControl(control)
	ti.HandleRequestExistingInputState() // no client: nothing happens
	if len(control.calls) != 0 {
		t.Fatalf("calls = %v", control.calls)
	}
	client := newFakeClient(val("abc", services.NewTextSelectionCollapsed(3)))
	ti.Attach(client, services.NewTextInputConfiguration())
	control.calls = nil
	ti.HandleRequestExistingInputState()
	if want := []string{"attach", "setEditingState"}; !reflect.DeepEqual(control.calls, want) {
		t.Errorf("calls = %v, want %v", control.calls, want)
	}
	client.hasValue = false
	control.calls = nil
	ti.HandleRequestExistingInputState()
	if want := []string{"attach"}; !reflect.DeepEqual(control.calls, want) {
		t.Errorf("calls = %v, want %v", control.calls, want)
	}
}

func TestTextInputConnectionDetached(t *testing.T) {
	ti := services.NewTextInput()
	first := ti.Attach(newFakeClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration())
	if !first.Attached() {
		t.Fatal("a new connection is attached")
	}
	second := ti.Attach(newFakeClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration())
	if first.Attached() || !second.Attached() {
		t.Fatal("the second connection replaces the first")
	}
	calls := map[string]func(){
		"Show":            first.Show,
		"RequestAutofill": first.RequestAutofill,
		"UpdateConfig":    func() { first.UpdateConfig(services.NewTextInputConfiguration()) },
		"SetEditingState": func() { first.SetEditingState(services.TextEditingValueEmpty) },
		"UpdateStyle":     func() { first.UpdateStyle(services.TextInputStyle{}) },
	}
	for name, call := range calls {
		if foundation.DebugMode {
			if tag := panicTag(t, call); tag != "TextInputConnectionDetached" {
				t.Errorf("%s on a detached connection panicked with %q", name, tag)
			}
		}
	}
	first.Close() // closing a connection that is not attached does nothing
	if !second.Attached() {
		t.Error("closing the first connection closed the second")
	}
	second.ConnectionClosedReceived()
	if second.Attached() {
		t.Error("the platform closed the connection, yet it is attached")
	}
}

func TestTextInputConnectionCachesWhatDidNotChange(t *testing.T) {
	ti := services.NewTextInput()
	control := &fakeControl{}
	ti.SetInputControl(control)
	conn := ti.Attach(newFakeClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration())
	control.calls = nil

	size, m := ui.Size{Width: 100, Height: 20}, ui.NewMatrix4Identity()
	conn.SetEditableSizeAndTransform(size, m)
	conn.SetEditableSizeAndTransform(size, ui.NewMatrix4Identity()) // an equal transform
	m.Translate(5, 0, 0)
	conn.SetEditableSizeAndTransform(size, m)
	conn.SetEditableSizeAndTransform(ui.Size{Width: 50, Height: 20}, m)

	rect := ui.NewRectFromLTWH(1, 2, 3, 4)
	conn.SetComposingRect(rect)
	conn.SetComposingRect(rect)
	conn.SetCaretRect(rect)
	conn.SetCaretRect(rect)
	conn.SetCaretRect(ui.NewRectFromLTWH(1, 2, 3, 5))

	rects := []services.SelectionRect{services.NewSelectionRect(0, rect)}
	conn.SetSelectionRects(rects)
	conn.SetSelectionRects([]services.SelectionRect{services.NewSelectionRect(0, rect)})
	conn.SetSelectionRects(nil)

	want := []string{"setEditableSizeAndTransform", "setEditableSizeAndTransform", "setEditableSizeAndTransform",
		"setComposingRect", "setCaretRect", "setCaretRect", "setSelectionRects", "setSelectionRects"}
	if !reflect.DeepEqual(control.calls, want) {
		t.Errorf("calls = %v\nwant %v", control.calls, want)
	}
}

func TestTextInputConnectionSendsAFiniteRect(t *testing.T) {
	ti := services.NewTextInput()
	got := &rectControl{}
	ti.SetInputControl(got)
	conn := ti.Attach(newFakeClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration())
	conn.SetCaretRect(ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 4})
	conn.SetCaretRect(ui.Rect{Left: 0, Top: 0, Right: 1, Bottom: 1}.Shift(ui.Offset{Dx: math.Inf(1), Dy: 0}))
	conn.SetComposingRect(ui.Rect{Left: 0, Top: 0, Right: 1, Bottom: 1}.Shift(ui.Offset{Dx: math.Inf(1), Dy: 0}))
	if len(got.caret) != 2 || got.caret[0] != (ui.Rect{Left: 1, Top: 2, Right: 3, Bottom: 4}) {
		t.Fatalf("caret rects = %v", got.caret)
	}
	if want := ui.NewRectFromLTWH(0, 0, -1, -1); got.caret[1] != want || got.composing[0] != want {
		t.Errorf("a rect that is not finite became %v and %v, want %v", got.caret[1], got.composing[0], want)
	}
}

type rectControl struct {
	services.TextInputControlDefaults
	caret, composing []ui.Rect
}

func (c *rectControl) SetCaretRect(r ui.Rect)     { c.caret = append(c.caret, r) }
func (c *rectControl) SetComposingRect(r ui.Rect) { c.composing = append(c.composing, r) }

func TestTextInputHidesAfterTheUpdateCycle(t *testing.T) {
	var queue []func()
	post := func(fn func()) { queue = append(queue, fn) }
	run := func() {
		q := queue
		queue = nil
		for _, fn := range q {
			fn()
		}
	}
	backend := &fakeBackend{}
	ti := services.NewTextInput(services.TextInputOpts{Backend: backend, Post: post})
	conn := ti.Attach(newFakeClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration())
	conn.Show()
	if !backend.last().Enabled {
		t.Fatal("the input method is off after Show")
	}
	conn.Close()
	// The platform control is told of the detach at once.
	if backend.last().Enabled {
		t.Error("the input method is on after the connection closed")
	}
	if len(queue) != 1 {
		t.Fatalf("%d tasks are waiting, want the one that hides", len(queue))
	}
	// A client that attaches in the same cycle keeps the input method: the
	// hide does nothing.
	conn2 := ti.Attach(newFakeClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration())
	conn2.Show()
	n := len(backend.states)
	run()
	if len(backend.states) != n {
		t.Error("the hide ran although a client is attached")
	}
	// With no client at the end of the cycle, Hide reaches the controls.
	conn2.Close()
	control := &fakeControl{}
	ti.SetInputControl(control)
	run()
	if got := control.calls; len(got) == 0 || got[len(got)-1] != "hide" {
		t.Errorf("control calls = %v, want hide last", got)
	}
	// Two closes in a cycle schedule one hide.
	conn3 := ti.Attach(newFakeClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration())
	conn3.Close()
	conn4 := ti.Attach(newFakeClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration())
	conn4.Close()
	if len(queue) != 1 {
		t.Errorf("%d tasks are waiting, want one", len(queue))
	}
}

func TestTextInputHidesAtOnceWithoutAPoster(t *testing.T) {
	ti := services.NewTextInput()
	control := &fakeControl{}
	ti.SetInputControl(control)
	ti.Attach(newFakeClient(services.TextEditingValueEmpty), services.NewTextInputConfiguration()).Close()
	if got := control.calls; len(got) != 3 || got[2] != "hide" {
		t.Errorf("calls = %v, want attach, detach, hide", got)
	}
}

func TestAutofill(t *testing.T) {
	enabled := services.NewAutofillConfiguration("user", []string{services.AutofillHintUsername}, val("bob", services.NewTextSelectionCollapsed(3)),
		services.AutofillConfigurationOpts{HintText: foundation.Some("Username")})
	if !enabled.Enabled || services.AutofillConfigurationDisabled.Enabled {
		t.Error("Enabled is wrong")
	}
	if !enabled.Equal(enabled) || enabled.Equal(services.AutofillConfigurationDisabled) {
		t.Error("Equal is wrong")
	}
	if got := enabled.String(); !strings.Contains(got, "uniqueIdentifier: user") || !strings.Contains(got, "hintText: Username") {
		t.Errorf("String = %q", got)
	}
	if services.AutofillHintPassword != "password" || services.AutofillHintOneTimeCode != "oneTimeCode" || services.AutofillHintURL != "url" {
		t.Error("the hints are wrong")
	}

	on := &fakeAutofillClient{id: "a", config: services.NewTextInputConfiguration(services.TextInputConfigurationOpts{AutofillConfiguration: foundation.Some(enabled)})}
	off := &fakeAutofillClient{id: "b", config: services.NewTextInputConfiguration()}
	scope := &fakeScope{clients: []services.AutofillClient{on, off}}
	ti := services.NewTextInput()
	client := newFakeClient(services.TextEditingValueEmpty)
	client.scope = scope
	ti.Attach(client, services.NewTextInputConfiguration())
	ti.HandleUpdateEditingStateWithTag(map[string]services.TextEditingValue{
		"a": val("filled", services.NewTextSelectionCollapsed(6)), "b": val("no", services.NewTextSelectionCollapsed(2)), "c": val("none", services.NewTextSelectionCollapsed(4)),
	})
	if len(on.filled) != 1 || on.filled[0].Text != "filled" || len(off.filled) != 0 {
		t.Errorf("filled = %v and %v", on.filled, off.filled)
	}
	// A client with no scope ignores it.
	client.scope = nil
	ti.HandleUpdateEditingStateWithTag(map[string]services.TextEditingValue{"a": services.TextEditingValueEmpty})
	if len(on.filled) != 1 {
		t.Error("a field was filled without a scope")
	}
}

func TestAutofillScopeAttachNeedsEnabledClients(t *testing.T) {
	enabled := services.NewAutofillConfiguration("x", nil, services.TextEditingValueEmpty)
	good := &fakeAutofillClient{id: "a", config: services.NewTextInputConfiguration(services.TextInputConfigurationOpts{AutofillConfiguration: foundation.Some(enabled)})}
	bad := &fakeAutofillClient{id: "b", config: services.NewTextInputConfiguration()}
	ti := services.NewTextInput()
	conn := (&fakeScope{clients: []services.AutofillClient{good}}).Attach(ti, newFakeClient(services.TextEditingValueEmpty), good.config)
	if conn == nil || !conn.Attached() {
		t.Error("the scope did not attach the client")
	}
	if foundation.DebugMode {
		if tag := panicTag(t, func() {
			(&fakeScope{clients: []services.AutofillClient{good, bad}}).Attach(ti, newFakeClient(services.TextEditingValueEmpty), good.config)
		}); tag != "AutofillClientDisabled" {
			t.Errorf("panicked with %q", tag)
		}
	}
}

func TestTextSelectionDelegateDefaults(t *testing.T) {
	var d services.TextSelectionDelegateDefaults
	if !d.CutEnabled() || !d.CopyEnabled() || !d.PasteEnabled() || !d.SelectAllEnabled() || !d.LookUpEnabled() ||
		!d.SearchWebEnabled() || !d.ShareEnabled() || d.LiveTextInputEnabled() {
		t.Error("the defaults of the delegate are wrong")
	}
	var c services.TextInputClientDefaults
	c.InsertContent(services.KeyboardInsertedContent{})
	c.ShowToolbar()
	c.PerformSelector("x")
	c.InsertTextPlaceholder(ui.Size{})
	c.RemoveTextPlaceholder()
	c.DidChangeInputControl(nil, nil)
	if c.OnFocusReceived() {
		t.Error("a client takes the focus by default")
	}
	var ctl services.TextInputControlDefaults
	ctl.Attach(nil, services.TextInputConfiguration{})
	ctl.Detach(nil)
	ctl.Show()
	ctl.Hide()
	ctl.UpdateConfig(services.TextInputConfiguration{})
	ctl.SetEditingState(services.TextEditingValue{})
	ctl.SetEditableSizeAndTransform(ui.Size{}, nil)
	ctl.SetComposingRect(ui.Rect{})
	ctl.SetCaretRect(ui.Rect{})
	ctl.SetSelectionRects(nil)
	ctl.UpdateStyle(services.TextInputStyle{})
	ctl.RequestAutofill()
	ctl.FinishAutofillContext(false)
}
