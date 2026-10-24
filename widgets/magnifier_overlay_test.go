package widgets_test

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gluttertest"
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/rendering"
	"github.com/rm4n0s/glutter/services"
	"github.com/rm4n0s/glutter/ui"
	. "github.com/rm4n0s/glutter/widgets"
)

// These tests cover the magnifier of SelectionOverlay and EditableText: 'can show
// magnifier when no handles exist' of the SelectionOverlay group of
// packages/flutter/test/widgets/text_selection_test.dart, and 'should build nothing
// by default' and 'magnifierConfiguration can be updated to display a new magnifier'
// of packages/flutter/test/widgets/editable_text_test.dart. 'magnifier is in correct
// position when EditableText is scaled' uses TextMagnifier and is in the tests of
// package material.

// magnifierOverlayFixture is a SelectionOverlay with the configuration, in the
// fixture of the other tests of the overlay.
type magnifierOverlayFixture struct {
	ts      *gluttertest.Tester
	bc      BuildContext
	overlay *SelectionOverlay
}

func newMagnifierOverlayFixture(t *testing.T, config *TextMagnifierConfiguration) *magnifierOverlayFixture {
	t.Helper()
	f := &magnifierOverlayFixture{ts: gluttertest.New(t)}
	start, end, toolbar := rendering.NewLayerLink(), rendering.NewLayerLink(), rendering.NewLayerLink()
	anchor := func(link *rendering.LayerLink, x, y float64) Widget {
		return NewPositioned(NewCompositedTransformTarget(link, NewSizedBox(nil, SizedBoxOpts{Width: foundation.Some(1.0), Height: foundation.Some(1.0)})),
			PositionedOpts{Left: foundation.Some(x), Top: foundation.Some(y)})
	}
	pumpContainer(f.ts, NewStack([]Widget{
		anchor(start, 100, 100), anchor(end, 300, 100), anchor(toolbar, 200, 50),
		NewBuilder(func(bc BuildContext) Widget { f.bc = bc; return NewSizedBox(nil) }),
	}))
	f.overlay = NewSelectionOverlay(f.bc, rendering.TextSelectionHandleTypeLeft, 14, rendering.TextSelectionHandleTypeRight, 14,
		[]rendering.TextSelectionPoint{{Point: ui.Offset{Dx: 100, Dy: 114}}, {Point: ui.Offset{Dx: 300, Dy: 114}}},
		handleControls{}, start, end, toolbar, SelectionOverlayOpts{MagnifierConfiguration: config})
	t.Cleanup(f.overlay.Dispose)
	return f
}

func TestSelectionOverlayMagnifier(t *testing.T) {
	magnifierKey := NewGlobalKey[State]()
	var builtGlobalGesturePosition ui.Offset
	var builtFieldBounds ui.Rect
	config := NewTextMagnifierConfiguration(TextMagnifierConfigurationOpts{
		HideHandlesInMagnifier: true,
		MagnifierBuilder: func(_ BuildContext, _ *MagnifierController, notifier *foundation.ValueNotifier[MagnifierInfo]) Widget {
			builtGlobalGesturePosition = notifier.Value().GlobalGesturePosition
			builtFieldBounds = notifier.Value().FieldBounds
			return NewSizedBoxShrink(nil, WidgetOpts{Key: magnifierKey})
		},
	})
	magnifier := gluttertest.ByKey(magnifierKey)
	globalGesturePosition := ui.Offset{Dx: 10, Dy: 10}
	fieldBounds := ui.Offset{}.And(ui.Size{Width: 200, Height: 50})
	info := MagnifierInfo{
		GlobalGesturePosition: globalGesturePosition,
		CaretRect:             ui.Offset{}.And(ui.Size{Width: 5, Height: 20}),
		FieldBounds:           fieldBounds,
		CurrentLineBoundaries: ui.Offset{}.And(ui.Size{Width: 200, Height: 50}),
	}

	t.Run("can show magnifier when no handles exist", func(t *testing.T) {
		f := newMagnifierOverlayFixture(t, config)
		if f.ts.Exists(magnifier) {
			t.Fatal("the magnifier is there before it is shown")
		}

		f.overlay.ShowMagnifier(info)
		f.ts.Pump()

		if err := f.ts.TakeError(); err != nil {
			t.Fatalf("error: %v", err)
		}
		if !f.ts.Exists(magnifier) {
			t.Fatal("the magnifier is not shown")
		}
		if builtFieldBounds != fieldBounds {
			t.Errorf("field bounds = %v, want %v", builtFieldBounds, fieldBounds)
		}
		if builtGlobalGesturePosition != globalGesturePosition {
			t.Errorf("gesture position = %v, want %v", builtGlobalGesturePosition, globalGesturePosition)
		}
		if !f.overlay.MagnifierExists() || !f.overlay.MagnifierIsVisible() {
			t.Errorf("exists %v, visible %v, want both", f.overlay.MagnifierExists(), f.overlay.MagnifierIsVisible())
		}
		if f.overlay.MagnifierConfiguration() != config {
			t.Error("MagnifierConfiguration is not the one given")
		}
	})

	t.Run("a disabled configuration builds nothing, and is the default", func(t *testing.T) {
		for name, config := range map[string]*TextMagnifierConfiguration{"disabled": TextMagnifierConfigurationDisabled, "default": nil} {
			t.Run(name, func(t *testing.T) {
				f := newMagnifierOverlayFixture(t, config)
				f.overlay.ShowMagnifier(info)
				f.ts.Pump()
				if f.overlay.MagnifierExists() || f.overlay.MagnifierIsVisible() {
					t.Error("a magnifier exists")
				}
				// The other calls do nothing without a magnifier.
				f.overlay.UpdateMagnifier(info)
				f.overlay.HideMagnifier()
				f.ts.Pump()
				if f.overlay.MagnifierConfiguration() != TextMagnifierConfigurationDisabled {
					t.Error("the configuration is not the disabled one")
				}
			})
		}
	})

	t.Run("updating and hiding", func(t *testing.T) {
		var info *foundation.ValueNotifier[MagnifierInfo]
		f := newMagnifierOverlayFixture(t, NewTextMagnifierConfiguration(TextMagnifierConfigurationOpts{
			MagnifierBuilder: func(_ BuildContext, _ *MagnifierController, notifier *foundation.ValueNotifier[MagnifierInfo]) Widget {
				info = notifier
				return NewSizedBoxShrink(nil, WidgetOpts{Key: magnifierKey})
			},
		}))
		f.overlay.UpdateMagnifier(MagnifierInfo{GlobalGesturePosition: ui.Offset{Dx: 1}})
		if info != nil {
			t.Fatal("updating without a magnifier built one")
		}

		first := MagnifierInfo{GlobalGesturePosition: ui.Offset{Dx: 1, Dy: 2}}
		f.overlay.ShowMagnifier(first)
		f.ts.Pump()
		if !f.ts.Exists(magnifier) || info.Value() != first {
			t.Fatalf("the magnifier is shown: %v, its info %v, want %v", f.ts.Exists(magnifier), info.Value(), first)
		}

		// Showing again does not build another one.
		built := info
		f.overlay.ShowMagnifier(MagnifierInfo{GlobalGesturePosition: ui.Offset{Dx: 9}})
		if info != built || info.Value() != first {
			t.Error("showing a second time built a magnifier or changed the info")
		}

		second := MagnifierInfo{GlobalGesturePosition: ui.Offset{Dx: 5, Dy: 6}}
		f.overlay.UpdateMagnifier(second)
		if info.Value() != second {
			t.Errorf("info = %v, want %v", info.Value(), second)
		}

		f.overlay.HideMagnifier()
		f.ts.Pump()
		if f.ts.Exists(magnifier) || f.overlay.MagnifierExists() {
			t.Error("the magnifier is still there after HideMagnifier")
		}
		// Hiding again does nothing.
		f.overlay.HideMagnifier()
	})

	t.Run("showing the magnifier hides the toolbar, and hiding the overlay hides the magnifier", func(t *testing.T) {
		f := newMagnifierOverlayFixture(t, config)
		f.overlay.ShowToolbar(f.bc, menuBuilder("Cut"))
		f.ts.Pump()
		if !f.overlay.ToolbarIsVisible() {
			t.Fatal("the toolbar is not shown")
		}
		f.overlay.ShowMagnifier(info)
		f.ts.Pump()
		if f.overlay.ToolbarIsVisible() || f.ts.Exists(gluttertest.ByText("Cut")) {
			t.Error("the toolbar stayed")
		}
		if !f.ts.Exists(magnifier) {
			t.Fatal("the magnifier is not shown")
		}
		f.overlay.Hide()
		f.ts.Pump()
		if f.ts.Exists(magnifier) {
			t.Error("Hide left the magnifier")
		}
	})

	t.Run("the magnifier is below the handles when it does not show them, above otherwise", func(t *testing.T) {
		tests := []struct {
			name             string
			hideHandles      bool
			wantMagnifierTop bool
		}{
			{"it shows the handles", false, true},
			{"it hides the handles", true, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				f := newMagnifierOverlayFixture(t, NewTextMagnifierConfiguration(TextMagnifierConfigurationOpts{
					HideHandlesInMagnifier: tt.hideHandles,
					MagnifierBuilder: func(BuildContext, *MagnifierController, *foundation.ValueNotifier[MagnifierInfo]) Widget {
						return NewSizedBoxShrink(nil, WidgetOpts{Key: magnifierKey})
					},
				}))
				f.overlay.ShowHandles()
				f.overlay.ShowMagnifier(info)
				f.ts.Pump()

				// The order of the entries of the overlay is the order of the tree, from the
				// bottom.
				index := func(key foundation.Key) int {
					for i, e := range f.ts.Elements(gluttertest.Everything()) {
						if e.Widget().Key() == key {
							return i
						}
					}
					t.Fatalf("no widget with the key %v", key)
					return -1
				}
				magnifierIndex, handleIndex := index(magnifierKey), index(handleKey(rendering.TextSelectionHandleTypeLeft))
				if got := magnifierIndex > handleIndex; got != tt.wantMagnifierTop {
					t.Errorf("magnifier at %d, handle at %d: the magnifier is above the handles: %v, want %v", magnifierIndex, handleIndex, got, tt.wantMagnifierTop)
				}
			})
		}
	})
}

func magnifierEditableText(controller *TextEditingController, focus *FocusNode, config *TextMagnifierConfiguration) *EditableText {
	style := painting.NewTextStyle(painting.TextStyleOpts{FontSize: foundation.Some(16.0), Color: foundation.Some(ui.NewColor(0xFF000000))})
	return NewEditableText(controller, focus, style, ui.NewColor(0xFF2196F3), ui.NewColor(0xFF9E9E9E), EditableTextOpts{
		ShowSelectionHandles: true, Autofocus: true, SelectionControls: handleControls{}, KeyboardType: foundation.Some(services.TextInputTypeText),
		TextAlign: ui.TextAlignRight, MagnifierConfiguration: config,
	})
}

func TestEditableTextMagnifierConfiguration(t *testing.T) {
	t.Run("should build nothing by default", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller, focus := NewTextEditingController(), NewFocusNode()
		t.Cleanup(controller.Dispose)
		t.Cleanup(focus.Dispose)
		editableText := magnifierEditableText(controller, focus, nil)
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: editableText}))

		bc := ts.Element(gluttertest.ByType[*EditableText]())
		notifier := foundation.NewValueNotifier(MagnifierInfoEmpty)
		defer notifier.Dispose()

		if built := editableText.MagnifierConfiguration().MagnifierBuilder()(bc, NewMagnifierController(), notifier); built != nil {
			t.Errorf("built %T, want nothing", built)
		}
		if editableText.MagnifierConfiguration() != TextMagnifierConfigurationDisabled {
			t.Error("the default configuration is not the disabled one")
		}
	})

	t.Run("magnifierConfiguration can be updated to display a new magnifier", func(t *testing.T) {
		// Regression test for https://github.com/flutter/flutter/issues/142077.
		ts := gluttertest.New(t)
		controller, focus := NewTextEditingController(), NewFocusNode()
		t.Cleanup(controller.Dispose)
		t.Cleanup(focus.Dispose)
		keyOne, keyTwo := foundation.NewUniqueKey(), foundation.NewUniqueKey()
		key := foundation.Key(keyOne)
		var setState StateSetter

		magnifierConfiguration := NewTextMagnifierConfiguration(TextMagnifierConfigurationOpts{
			MagnifierBuilder: func(BuildContext, *MagnifierController, *foundation.ValueNotifier[MagnifierInfo]) Widget {
				return NewPlaceholder(nil, PlaceholderOpts{Key: key})
			},
		})

		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewAlign(NewSizedBox(NewStatefulBuilder(func(_ BuildContext, localSetState StateSetter) Widget {
			setState = localSetState
			return magnifierEditableText(controller, focus, magnifierConfiguration)
		}), SizedBoxOpts{Width: foundation.Some(400.0)}), AlignOpts{Alignment: painting.AlignmentTopLeft})}))

		ts.Pump() // Wait for autofocus to take effect.

		checkMagnifierKey := func(testKey foundation.Key) {
			t.Helper()
			editableText := gluttertest.WidgetAs[*EditableText](ts, gluttertest.ByType[*EditableText]())
			bc := ts.Element(gluttertest.ByType[*EditableText]())
			magnifierInfo := foundation.NewValueNotifier(MagnifierInfoEmpty)
			defer magnifierInfo.Dispose()
			built := editableText.MagnifierConfiguration().MagnifierBuilder()(bc, NewMagnifierController(), magnifierInfo)
			if built == nil || built.Key() != testKey {
				t.Errorf("built magnifier key = %v, want %v", built, testKey)
			}
		}

		checkMagnifierKey(keyOne)

		setState(func() { key = keyTwo })
		ts.PumpAndSettle()

		checkMagnifierKey(keyTwo)
	})

	t.Run("a new configuration recreates the selection overlay, the same one does not", func(t *testing.T) {
		ts := gluttertest.New(t)
		controller, focus := NewTextEditingController("hello"), NewFocusNode()
		t.Cleanup(controller.Dispose)
		t.Cleanup(focus.Dispose)
		builder := func(BuildContext, *MagnifierController, *foundation.ValueNotifier[MagnifierInfo]) Widget {
			return NewPlaceholder(nil)
		}
		one := NewTextMagnifierConfiguration(TextMagnifierConfigurationOpts{MagnifierBuilder: builder})
		two := NewTextMagnifierConfiguration(TextMagnifierConfigurationOpts{MagnifierBuilder: builder})
		config := one
		var rebuild StateSetter
		ts.PumpWidget(testWidgetsApp(testWidgetsAppOpts{home: NewAlign(NewSizedBox(NewStatefulBuilder(func(_ BuildContext, localSetState StateSetter) Widget {
			rebuild = localSetState
			return magnifierEditableText(controller, focus, config)
		}), SizedBoxOpts{Width: foundation.Some(400.0)}), AlignOpts{Alignment: painting.AlignmentTopLeft})}))
		ts.Pump() // Wait for autofocus to take effect.
		state := gluttertest.StateOf[*EditableTextState](ts, gluttertest.ByType[*EditableText]())
		ts.TapAt(ts.Center(gluttertest.ByType[*EditableText]()))
		ts.PumpAndSettle()
		before := state.SelectionOverlay()
		if before == nil {
			t.Fatal("there is no selection overlay after a tap")
		}

		rebuild(func() {})
		ts.PumpAndSettle()
		if state.SelectionOverlay() != before {
			t.Error("the overlay was recreated for the same configuration")
		}

		rebuild(func() { config = two })
		ts.PumpAndSettle()
		if state.SelectionOverlay() == before {
			t.Error("the overlay was not recreated for another configuration")
		}
		if state.SelectionOverlay() == nil {
			t.Error("no new overlay")
		}
	})
}
