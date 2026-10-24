package android

import (
	"math"
	"reflect"
	"testing"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/ui"
)

func le32(v uint32) []byte { return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)} }

func TestDecodeAction(t *testing.T) {
	tests := []struct {
		name   string
		action ui.SemanticsAction
		args   []byte
		want   any
	}{
		{"tap", ui.SemanticsActionTap, nil, nil},
		{"long press", ui.SemanticsActionLongPress, []byte{}, nil},
		{"scroll up", ui.SemanticsActionScrollUp, nil, nil},
		{"scroll down", ui.SemanticsActionScrollDown, nil, nil},
		{"scroll left", ui.SemanticsActionScrollLeft, nil, nil},
		{"scroll right", ui.SemanticsActionScrollRight, nil, nil},
		{"increase", ui.SemanticsActionIncrease, nil, nil},
		{"decrease", ui.SemanticsActionDecrease, nil, nil},
		{"show on screen", ui.SemanticsActionShowOnScreen, nil, nil},
		{"copy", ui.SemanticsActionCopy, nil, nil},
		{"cut", ui.SemanticsActionCut, nil, nil},
		{"paste", ui.SemanticsActionPaste, nil, nil},
		{"did gain focus", ui.SemanticsActionDidGainAccessibilityFocus, nil, nil},
		{"did lose focus", ui.SemanticsActionDidLoseAccessibilityFocus, nil, nil},
		{"dismiss", ui.SemanticsActionDismiss, nil, nil},
		{"focus", ui.SemanticsActionFocus, nil, nil},
		{"expand", ui.SemanticsActionExpand, nil, nil},
		{"collapse", ui.SemanticsActionCollapse, nil, nil},
		{"cursor forward by character", ui.SemanticsActionMoveCursorForwardByCharacter, []byte{1}, true},
		{"cursor backward by character", ui.SemanticsActionMoveCursorBackwardByCharacter, []byte{0}, false},
		{"cursor forward by word", ui.SemanticsActionMoveCursorForwardByWord, []byte{1}, true},
		{"cursor backward by word", ui.SemanticsActionMoveCursorBackwardByWord, []byte{0}, false},
		{"set selection", ui.SemanticsActionSetSelection, append(le32(2), le32(5)...), map[string]int{"base": 2, "extent": 5}},
		{"set selection backwards", ui.SemanticsActionSetSelection, append(le32(5), le32(2)...), map[string]int{"base": 5, "extent": 2}},
		{"set text", ui.SemanticsActionSetText, []byte("héllo"), "héllo"},
		{"set text to nothing", ui.SemanticsActionSetText, nil, ""},
		{"custom action", ui.SemanticsActionCustomAction, le32(7), 7},
		{"scroll to offset", ui.SemanticsActionScrollToOffset, append(le32(math.Float32bits(1.5)), le32(math.Float32bits(-2))...), []float64{1.5, -2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeAction(12, int(tt.action), tt.args)
			if err != nil {
				t.Fatal(err)
			}
			want := ui.SemanticsActionEvent{ViewID: ui.ImplicitViewID, NodeID: 12, Type: tt.action, Arguments: tt.want}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("decodeAction = %+v, want %+v", got, want)
			}
		})
	}
}

func TestDecodeActionFailsWithAndroidSemanticsActionInvalid(t *testing.T) {
	tests := []struct {
		name   string
		action int
		args   []byte
	}{
		{"no action", 0, nil},
		{"two actions", int(ui.SemanticsActionTap | ui.SemanticsActionLongPress), nil},
		{"an action that does not exist", 1 << 30, nil},
		{"a negative action", -1, nil},
		{"arguments for a tap", int(ui.SemanticsActionTap), []byte{1}},
		{"no extend flag", int(ui.SemanticsActionMoveCursorForwardByWord), nil},
		{"a short selection", int(ui.SemanticsActionSetSelection), []byte{1, 2, 3}},
		{"a short custom action", int(ui.SemanticsActionCustomAction), []byte{1}},
		{"a short offset", int(ui.SemanticsActionScrollToOffset), make([]byte, 4)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeAction(1, tt.action, tt.args)
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if !appErr.HasRoute("decodeAction." + tagAndroidSemanticsActionInvalid) {
				t.Errorf("unexpected failure path: %s", appErr.Route())
			}
		})
	}
}
