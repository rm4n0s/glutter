package openuri

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestCheckOpenURI(t *testing.T) {
	tests := []struct {
		uri string
		tag string
	}{
		{"https://example.com/a?b=c", ""},
		{"mailto:me@example.com", ""},
		{"file:///Users/a/picture.png", ""},
		{"file:///Users/a/My%20Folder/", ""},
		{"x-apple.systempreferences:com.apple.preference.security", ""},
		{"", tagOpenURIInvalid},
		{"no scheme", tagOpenURIInvalid},
		{"https://example.com/\nx", tagOpenURIInvalid},
		{"https://example.com/\x00", tagOpenURIInvalid},
		{"javascript:alert(1)", tagOpenURISchemeBlocked},
		{"vbscript:run", tagOpenURISchemeBlocked},
		{"file:///Applications/Safari.app", tagOpenURISchemeBlocked},
		{"file:///Applications/Safari.app/", tagOpenURISchemeBlocked},
		{"file:///Users/a/run.COMMAND", tagOpenURISchemeBlocked},
		{"file:///tmp/install.pkg", tagOpenURISchemeBlocked},
		{"file:///tmp/s%2Esh", tagOpenURISchemeBlocked},
	}
	for _, tc := range tests {
		err := Check(tc.uri)
		if tc.tag == "" {
			if err != nil {
				t.Errorf("Check(%q) = %v, want nil", tc.uri, err)
			}
			continue
		}
		appErr, ok := errors.FromError(err)
		if !ok || appErr.Tag != tc.tag || !appErr.HasRoute("Check."+tc.tag) {
			t.Errorf("Check(%q) = %v, want %s (route %v)", tc.uri, err, tc.tag, err)
		}
	}
}
