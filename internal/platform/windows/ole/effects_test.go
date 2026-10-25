package ole

import (
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/windows/win32"
	"github.com/rm4n0s/glutter/ui"
)

func TestActionsAndEffectsBothWays(t *testing.T) {
	tests := []struct {
		name    string
		effect  uint32
		actions ui.DropActions
	}{
		{"none", win32.DropEffectNone, ui.DropActionNone},
		{"copy", win32.DropEffectCopy, ui.DropActionCopy},
		{"move", win32.DropEffectMove, ui.DropActionMove},
		{"link", win32.DropEffectLink, ui.DropActionLink},
		{"copy and move", win32.DropEffectCopy | win32.DropEffectMove, ui.DropActionCopy | ui.DropActionMove},
		{"all three", win32.DropEffectCopy | win32.DropEffectMove | win32.DropEffectLink, ui.DropActionCopy | ui.DropActionMove | ui.DropActionLink},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ActionsOf(tt.effect); got != tt.actions {
				t.Errorf("ActionsOf(%#x) = %v, want %v", tt.effect, got, tt.actions)
			}
			if got := EffectOf(tt.actions); got != tt.effect {
				t.Errorf("EffectOf(%v) = %#x, want %#x", tt.actions, got, tt.effect)
			}
		})
	}
	// DROPEFFECT_SCROLL is not an action.
	if got := ActionsOf(win32.DropEffectScroll | win32.DropEffectCopy); got != ui.DropActionCopy {
		t.Errorf("the scroll bit changed the actions: %v", got)
	}
}

func TestOfferedActionsFollowTheKeys(t *testing.T) {
	all := ui.DropActionCopy | ui.DropActionMove | ui.DropActionLink
	tests := []struct {
		name    string
		key     uint32
		allowed ui.DropActions
		want    ui.DropActions
	}{
		{"no key offers all", 0, all, all},
		{"a button is not a modifier", win32.MKLButton, all, all},
		{"control asks for copy", win32.MKControl, all, ui.DropActionCopy},
		{"shift asks for move", win32.MKShift, all, ui.DropActionMove},
		{"control and shift ask for link", win32.MKControl | win32.MKShift, all, ui.DropActionLink},
		{"alt asks for link", win32.MKAlt, all, ui.DropActionLink},
		{"the key asks for what the source does not allow", win32.MKShift, ui.DropActionCopy, ui.DropActionCopy},
		{"link asked, only copy and move allowed", win32.MKAlt, ui.DropActionCopy | ui.DropActionMove, ui.DropActionCopy | ui.DropActionMove},
		{"nothing allowed", win32.MKControl, ui.DropActionNone, ui.DropActionNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OfferedActions(tt.key, tt.allowed); got != tt.want {
				t.Errorf("OfferedActions(%#x, %v) = %v, want %v", tt.key, tt.allowed, got, tt.want)
			}
		})
	}
}

func TestEffectForTheAcceptedAction(t *testing.T) {
	cm := uint32(win32.DropEffectCopy | win32.DropEffectMove)
	tests := []struct {
		name     string
		accepted ui.DropActions
		allowed  uint32
		want     uint32
	}{
		{"refused", ui.DropActionNone, cm, win32.DropEffectNone},
		{"the accepted action is allowed", ui.DropActionMove, cm, win32.DropEffectMove},
		{"the accepted action is not allowed", ui.DropActionLink, cm, win32.DropEffectNone},
		{"several accepted, copy first", ui.DropActionMove | ui.DropActionCopy, cm, win32.DropEffectCopy},
		{"several accepted, the allowed one", ui.DropActionCopy | ui.DropActionLink, win32.DropEffectLink, win32.DropEffectLink},
		{"nothing allowed", ui.DropActionCopy, win32.DropEffectNone, win32.DropEffectNone},
		{"the scroll bit is not an effect to return", ui.DropActionCopy, win32.DropEffectCopy | win32.DropEffectScroll, win32.DropEffectCopy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EffectFor(tt.accepted, tt.allowed); got != tt.want {
				t.Errorf("EffectFor(%v, %#x) = %#x, want %#x", tt.accepted, tt.allowed, got, tt.want)
			}
		})
	}
}
