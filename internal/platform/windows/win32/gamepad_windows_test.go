//go:build windows

package win32

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// TestHidParserErrorsHaveTheirRoutes: the calls of the HID parser fail with HidParserFailed for a parsed report descriptor that is
// not one (nothing, or bytes that no device made), each from its own function.
func TestHidParserErrorsHaveTheirRoutes(t *testing.T) {
	garbage := make([]byte, 64)
	tests := []struct {
		name  string
		call  func() error
		route string
	}{
		{"caps of nothing", func() error { _, err := HidpGetCaps(nil); return err }, "HidpGetCaps.HidParserFailed"},
		{"caps of garbage", func() error { _, err := HidpGetCaps(garbage); return err }, "HidpGetCaps.HidParserFailed"},
		{"button caps of garbage", func() error { _, err := HidpInputButtonCaps(garbage, 1); return err }, "HidpInputButtonCaps.HidParserFailed"},
		{"value caps of garbage", func() error { _, err := HidpInputValueCaps(garbage, 1); return err }, "HidpInputValueCaps.HidParserFailed"},
		{"data of garbage", func() error { _, err := HidpGetData(garbage, make([]HidpData, 2), []byte{1, 2, 3}); return err }, "HidpGetData.HidParserFailed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			appErr, ok := errors.FromError(err)
			if !ok {
				t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
			}
			if appErr.Tag != TagHidParserFailed || !appErr.HasRoute(tt.route) {
				t.Errorf("unexpected failure: tag %s, path %s", appErr.Tag, appErr.Route())
			}
		})
	}
}
