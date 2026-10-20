package widgets_test

import (
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests are ported from packages/flutter/test/widgets/autocomplete_test.dart.

var autocompleteOptions = []string{
	"aardvark", "bobcat", "chameleon", "dingo", "elephant", "flamingo", "goose", "hippopotamus",
	"iguana", "jaguar", "koala", "lemur", "mouse", "northern white rhinoceros",
}

func filterOptions(v services.TextEditingValue, deliver func([]string)) {
	var out []string
	for _, o := range autocompleteOptions {
		if strings.Contains(o, strings.ToLower(v.Text)) {
			out = append(out, o)
		}
	}
	deliver(out)
}

// testTextField is TestTextField: an EditableText with the test style.
func testTextField(key foundation.Key, controller *TextEditingController, focus *FocusNode, onSubmitted func(string)) Widget {
	return NewEditableText(controller, focus, plainEditableStyle(), ui.NewColor(0xFF000000), ui.NewColor(0xFF888888),
		EditableTextOpts{Key: key, OnSubmitted: onSubmitted})
}

func TestRawAutocompleteCanFilterAndSelectAListOfStringOptions(t *testing.T) {
	ts := gluttertest.New(t)
	fieldKey, optionsKey := foundation.NewValueKey("field"), foundation.NewValueKey("options")
	var lastOptions []string
	var lastOnSelected AutocompleteOnSelected[string]
	var focusNode *FocusNode
	var controller *TextEditingController
	pumpContainer(ts, NewRawAutocomplete(
		func(bc BuildContext, onSelected AutocompleteOnSelected[string], options []string) Widget {
			lastOptions, lastOnSelected = options, onSelected
			return NewSizedBox(nil, SizedBoxOpts{Key: optionsKey})
		}, filterOptions,
		RawAutocompleteOpts[string]{FieldViewBuilder: func(bc BuildContext, c *TextEditingController, f *FocusNode, onSubmitted foundation.VoidCallback) Widget {
			focusNode, controller = f, c
			return testTextField(fieldKey, c, f, nil)
		}}))
	if !ts.Exists(gluttertest.ByKey(fieldKey)) || ts.Exists(gluttertest.ByKey(optionsKey)) {
		t.Fatal("the field must show without the options")
	}
	focusNode.RequestFocus()
	ts.Pump()
	if !ts.Exists(gluttertest.ByKey(optionsKey)) || len(lastOptions) != len(autocompleteOptions) {
		t.Fatalf("focused: options shown=%v n=%d", ts.Exists(gluttertest.ByKey(optionsKey)), len(lastOptions))
	}
	controller.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: "ele", Selection: foundation.Some(services.NewTextSelectionCollapsed(3))}))
	ts.Pump()
	if len(lastOptions) != 2 || lastOptions[0] != "chameleon" || lastOptions[1] != "elephant" {
		t.Fatalf("options for ele = %v", lastOptions)
	}
	selection := lastOptions[1]
	lastOnSelected(selection)
	ts.Pump()
	if ts.Exists(gluttertest.ByKey(optionsKey)) || controller.Text() != selection {
		t.Fatalf("after select: options shown=%v text=%q", ts.Exists(gluttertest.ByKey(optionsKey)), controller.Text())
	}
	controller.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: "e", Selection: foundation.Some(services.NewTextSelectionCollapsed(1))}))
	ts.Pump()
	want := []string{"chameleon", "elephant", "goose", "lemur", "mouse", "northern white rhinoceros"}
	if !ts.Exists(gluttertest.ByKey(optionsKey)) || strings.Join(lastOptions, ",") != strings.Join(want, ",") {
		t.Fatalf("options for e = %v", lastOptions)
	}
}

func TestRawAutocompleteCanSplitTheFieldAndOptions(t *testing.T) {
	ts := gluttertest.New(t)
	fieldKey, optionsKey := foundation.NewValueKey("field"), foundation.NewValueKey("options")
	autocompleteKey := NewGlobalKey[State]()
	controller, focusNode := NewTextEditingController(), NewFocusNode()
	t.Cleanup(func() { controller.Dispose(); focusNode.Dispose() })
	var lastOptions []string
	var lastOnSelected AutocompleteOnSelected[string]
	pumpContainer(ts, NewColumn([]Widget{
		testTextField(fieldKey, controller, focusNode, func(string) { RawAutocompleteOnFieldSubmitted(autocompleteKey) }),
		NewExpanded(NewAlign(NewRawAutocomplete(
			func(bc BuildContext, onSelected AutocompleteOnSelected[string], options []string) Widget {
				lastOptions, lastOnSelected = options, onSelected
				return NewSizedBox(nil, SizedBoxOpts{Key: optionsKey, Width: foundation.Some(100.0), Height: foundation.Some(100.0)})
			}, filterOptions,
			RawAutocompleteOpts[string]{Key: autocompleteKey, FocusNode: focusNode, TextEditingController: controller}),
			AlignOpts{Alignment: painting.AlignmentTopLeft})),
	}))
	if ts.Exists(gluttertest.ByKey(optionsKey)) {
		t.Fatal("the options show before focus")
	}
	focusNode.RequestFocus()
	ts.Pump()
	if !ts.Exists(gluttertest.ByKey(optionsKey)) || len(lastOptions) != len(autocompleteOptions) {
		t.Fatalf("focused: n=%d", len(lastOptions))
	}
	controller.SetValue(services.NewTextEditingValue(services.TextEditingValueOpts{Text: "ele", Selection: foundation.Some(services.NewTextSelectionCollapsed(3))}))
	ts.Pump()
	if len(lastOptions) != 2 {
		t.Fatalf("options for ele = %v", lastOptions)
	}
	lastOnSelected(lastOptions[1])
	ts.Pump()
	if ts.Exists(gluttertest.ByKey(optionsKey)) || controller.Text() != "elephant" {
		t.Fatalf("after select: text=%q", controller.Text())
	}
}

func TestRawAutocompleteKeyboardNavigationHighlightsAndSelects(t *testing.T) {
	ts := gluttertest.New(t)
	var lastHighlight int
	var focusNode *FocusNode
	var controller *TextEditingController
	var selected string
	pumpContainer(ts, NewRawAutocomplete(
		func(bc BuildContext, onSelected AutocompleteOnSelected[string], options []string) Widget {
			lastHighlight = AutocompleteHighlightedOptionOf(bc)
			return NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(10.0), Height: foundation.Some(10.0)})
		}, filterOptions,
		RawAutocompleteOpts[string]{
			OnSelected: func(o string) { selected = o },
			FieldViewBuilder: func(bc BuildContext, c *TextEditingController, f *FocusNode, onSubmitted foundation.VoidCallback) Widget {
				focusNode, controller = f, c
				return testTextField(nil, c, f, func(string) { onSubmitted() })
			}}))
	focusNode.RequestFocus()
	ts.Pump()
	if lastHighlight != 0 {
		t.Fatalf("initial highlight = %d", lastHighlight)
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
	ts.SendKeyEvent(services.LogicalKeyboardKeyArrowDown)
	ts.Pump()
	if lastHighlight != 2 {
		t.Fatalf("highlight after two ArrowDown = %d, want 2", lastHighlight)
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyArrowUp)
	ts.Pump()
	if lastHighlight != 1 {
		t.Fatalf("highlight after ArrowUp = %d, want 1", lastHighlight)
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyEnter)
	ts.Pump()
	if selected != "bobcat" || controller.Text() != "bobcat" {
		t.Fatalf("selected = %q text = %q, want bobcat", selected, controller.Text())
	}
}

func TestRawAutocompleteEscapeHidesTheOptions(t *testing.T) {
	ts := gluttertest.New(t)
	optionsKey := foundation.NewValueKey("options")
	var focusNode *FocusNode
	pumpContainer(ts, NewRawAutocomplete(
		func(BuildContext, AutocompleteOnSelected[string], []string) Widget {
			return NewSizedBox(nil, SizedBoxOpts{Key: optionsKey})
		}, filterOptions,
		RawAutocompleteOpts[string]{FieldViewBuilder: func(bc BuildContext, c *TextEditingController, f *FocusNode, _ foundation.VoidCallback) Widget {
			focusNode = f
			return testTextField(nil, c, f, nil)
		}}))
	focusNode.RequestFocus()
	ts.Pump()
	if !ts.Exists(gluttertest.ByKey(optionsKey)) {
		t.Fatal("options not shown")
	}
	ts.SendKeyEvent(services.LogicalKeyboardKeyEscape)
	ts.Pump()
	if ts.Exists(gluttertest.ByKey(optionsKey)) {
		t.Fatal("options still shown after Escape")
	}
}

func TestRawAutocompleteAssertions(t *testing.T) {
	builder := func(BuildContext, AutocompleteOnSelected[string], []string) Widget { return NewSizedBox(nil) }
	fieldView := func(bc BuildContext, c *TextEditingController, f *FocusNode, _ foundation.VoidCallback) Widget {
		return NewSizedBox(nil)
	}
	controller, focus := NewTextEditingController(), NewFocusNode()
	t.Cleanup(func() { controller.Dispose(); focus.Dispose() })
	tests := []struct {
		name string
		tag  string
		fn   func()
	}{
		{"no field", "RawAutocompleteNoField", func() { NewRawAutocomplete(builder, filterOptions) }},
		{"focus node without controller", "RawAutocompleteFocusNodeMismatch", func() {
			NewRawAutocomplete(builder, filterOptions, RawAutocompleteOpts[string]{FieldViewBuilder: fieldView, FocusNode: focus})
		}},
		{"initial value and controller", "RawAutocompleteInitialValueAndController", func() {
			NewRawAutocomplete(builder, filterOptions, RawAutocompleteOpts[string]{
				FieldViewBuilder: fieldView, FocusNode: focus, TextEditingController: controller,
				InitialValue: foundation.Some(services.TextEditingValue{}),
			})
		}},
		{"key not mounted", "RawAutocompleteKeyNotMounted", func() { RawAutocompleteOnFieldSubmitted(NewGlobalKey[State]()) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { skipUnlessDebug(t); requireTag(t, tc.tag, tc.fn) })
	}
}
