package macos

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/internal/platform"
	"github.com/rm4n0s/glutter/ui"
)

func TestStyleForTheConfig(t *testing.T) {
	no := foundation.Some(false)
	tests := []struct {
		name       string
		cfg        platform.WindowConfig
		mask       uint64
		deco       ui.DecorationMode
		caps       ui.WindowCapabilities
		dialog     bool
		modal      bool
		fullScreen bool
	}{
		{"default", platform.WindowConfig{}, maskTitled | maskClosable | maskMiniaturizable | maskResizable, ui.DecorationServerSide,
			ui.WindowCapabilities{WindowMenu: true, Maximize: true, Fullscreen: true, Minimize: true}, false, false, true},
		{"not resizable", platform.WindowConfig{Resizable: no}, maskTitled | maskClosable | maskMiniaturizable, ui.DecorationServerSide,
			ui.WindowCapabilities{WindowMenu: true, Minimize: true}, false, false, false},
		{"not decorated", platform.WindowConfig{Decorated: no}, maskBorderless | maskResizable, ui.DecorationNone,
			ui.WindowCapabilities{}, false, false, false},
		{"borderless kind", platform.WindowConfig{Kind: platform.WindowKindBorderless, Resizable: no}, maskBorderless, ui.DecorationNone,
			ui.WindowCapabilities{}, false, false, false},
		{"app decorations keep the traffic lights over the content", platform.WindowConfig{Decorations: platform.DecorationsApp},
			maskTitled | maskClosable | maskMiniaturizable | maskResizable | maskFullSizeContent, ui.DecorationApp,
			ui.WindowCapabilities{WindowMenu: true, Maximize: true, Fullscreen: true, Minimize: true}, false, false, true},
		{"dialog has no minimize button", platform.WindowConfig{Kind: platform.WindowKindDialog, HasParent: true},
			maskTitled | maskClosable | maskResizable, ui.DecorationServerSide,
			ui.WindowCapabilities{WindowMenu: true, Maximize: true}, true, false, false},
		{"modal dialog", platform.WindowConfig{Kind: platform.WindowKindDialog, HasParent: true, Modal: true, Resizable: no},
			maskTitled | maskClosable, ui.DecorationServerSide, ui.WindowCapabilities{WindowMenu: true}, true, true, false},
		{"a modal dialog with no parent is not modal", platform.WindowConfig{Kind: platform.WindowKindDialog, Modal: true},
			maskTitled | maskClosable | maskResizable, ui.DecorationServerSide, ui.WindowCapabilities{WindowMenu: true, Maximize: true}, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := styleFor(tt.cfg)
			if got := s.mask(); got != tt.mask {
				t.Errorf("mask = %#x, want %#x", got, tt.mask)
			}
			if got := s.decoration(); got != tt.deco {
				t.Errorf("decoration = %v, want %v", got, tt.deco)
			}
			if got := s.capabilities(); got != tt.caps {
				t.Errorf("capabilities = %+v, want %+v", got, tt.caps)
			}
			if s.Dialog != tt.dialog || s.Modal != tt.modal || s.FullScreen != tt.fullScreen {
				t.Errorf("dialog/modal/fullscreen = %v/%v/%v, want %v/%v/%v", s.Dialog, s.Modal, s.FullScreen, tt.dialog, tt.modal, tt.fullScreen)
			}
		})
	}
}
