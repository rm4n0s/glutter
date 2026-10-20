package fonts

import (
	"strings"
	"testing"
)

type recorder struct {
	families []string
	sizes    []int
	failOn   string
}

func (r *recorder) AddBundled(family string, data []byte) error {
	if family == r.failOn {
		return errFail
	}
	r.families = append(r.families, family)
	r.sizes = append(r.sizes, len(data))
	return nil
}

type failure string

func (f failure) Error() string { return string(f) }

const errFail = failure("add failed")

func TestFontsAreEmbedded(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		min  int
	}{
		{"Roboto", Roboto(), 1 << 20},
		{"Material Icons", MaterialIcons(), 100 << 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.data) < tt.min {
				t.Fatalf("%d bytes, want at least %d", len(tt.data), tt.min)
			}
			// TrueType files start with 0x00010000 or "true", OpenType with "OTTO".
			head := string(tt.data[:4])
			if head != "\x00\x01\x00\x00" && head != "true" && head != "OTTO" {
				t.Errorf("not an sfnt font, starts with %q", head)
			}
		})
	}
}

func TestLicenses(t *testing.T) {
	tests := []struct {
		name, text, want string
	}{
		{"Roboto", RobotoLicense(), "SIL OPEN FONT LICENSE"},
		{"Material Icons", MaterialIconsLicense(), "Creative Commons"},
	}
	for _, tt := range tests {
		if !strings.Contains(tt.text, tt.want) {
			t.Errorf("%s license does not contain %q", tt.name, tt.want)
		}
	}
}

func TestRegister(t *testing.T) {
	r := new(recorder)
	if err := Register(r); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.families, ","); got != "Roboto,MaterialIcons" {
		t.Errorf("families = %s", got)
	}
	if err := Register(&recorder{failOn: MaterialIconsFamily}); err != errFail {
		t.Errorf("Register error = %v, want the AddBundled error", err)
	}
}
