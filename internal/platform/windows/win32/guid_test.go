package win32

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestParseGUID(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    GUID
		wantTag string
	}{
		{"IUnknown with braces", "{00000000-0000-0000-C000-000000000046}", GUID{Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}, ""},
		{"without braces", "00000000-0000-0000-C000-000000000046", GUID{Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}, ""},
		{"lower case", "{189819f1-1db6-4b57-be54-1821339b85f7}", GUID{0x189819f1, 0x1db6, 0x4b57, [8]byte{0xbe, 0x54, 0x18, 0x21, 0x33, 0x9b, 0x85, 0xf7}}, ""},
		{"IDXGIFactory2", "{50C83A1C-E072-4C48-87B0-3630FA36A6D0}", GUID{0x50c83a1c, 0xe072, 0x4c48, [8]byte{0x87, 0xb0, 0x36, 0x30, 0xfa, 0x36, 0xa6, 0xd0}}, ""},
		{"empty", "", GUID{}, TagGUIDInvalid},
		{"too short", "{00000000-0000-0000-C000-00000000004}", GUID{}, TagGUIDInvalid},
		{"too long", "{00000000-0000-0000-C000-0000000000466}", GUID{}, TagGUIDInvalid},
		{"an opening brace only", "{00000000-0000-0000-C000-000000000046", GUID{}, TagGUIDInvalid},
		{"a closing brace only", "00000000-0000-0000-C000-000000000046}", GUID{}, TagGUIDInvalid},
		{"a lone brace", "{", GUID{}, TagGUIDInvalid},
		{"a dash in the wrong place", "{0000000-00000-0000-C000-000000000046}", GUID{}, TagGUIDInvalid},
		{"a digit that is not hexadecimal", "{0000000G-0000-0000-C000-000000000046}", GUID{}, TagGUIDInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGUID(tt.text)
			if tt.wantTag != "" {
				appErr, ok := errors.FromError(err)
				if !ok {
					t.Fatalf("expected *errors.Error, got %T: %v", err, err)
				}
				if appErr.Tag != tt.wantTag || !appErr.HasRoute("ParseGUID.GUIDInvalid") {
					t.Errorf("got %s via %s, want %s via ParseGUID", appErr.Tag, appErr.Route(), tt.wantTag)
				}
				if appErr.Metadata["text"] != tt.text {
					t.Errorf("metadata = %v, want the text", appErr.Metadata)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("ParseGUID = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGUIDStringRoundTrips(t *testing.T) {
	for _, text := range []string{"{00000000-0000-0000-C000-000000000046}", "{50C83A1C-E072-4C48-87B0-3630FA36A6D0}", "{FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF}"} {
		if got := MustParseGUID(text).String(); got != text {
			t.Errorf("String = %q, want %q", got, text)
		}
	}
	if IIDUnknown.String() != "{00000000-0000-0000-C000-000000000046}" {
		t.Errorf("IIDUnknown = %v", IIDUnknown)
	}
}

func TestMustParseGUIDPanicsWithTheTaggedError(t *testing.T) {
	defer func() {
		appErr, ok := errors.FromError(asError(recover()))
		if !ok || appErr.Tag != TagGUIDInvalid {
			t.Errorf("recovered %v, want the GUIDInvalid error", appErr)
		}
	}()
	MustParseGUID("nonsense")
}

// asError turns a recovered value into an error, or nil.
func asError(r any) error {
	err, _ := r.(error)
	return err
}
