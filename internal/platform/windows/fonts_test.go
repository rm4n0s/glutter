package windows

import (
	"reflect"
	"strings"
	"testing"
)

func TestFontFiles(t *testing.T) {
	listing := map[string][]string{
		`C:\Windows\Fonts`: {"segoeui.ttf", "SEGOEUIB.TTF", "msgothic.ttc", "vgasys.fon", "desktop.ini", "arial.ttf", "cour.pfb"},
		`C:\Users\me\AppData\Local\Microsoft\Windows\Fonts`: {"Mine.otf"},
	}
	list := func(dir string) []string { return listing[dir] }
	tests := []struct {
		name    string
		entries []fontEntry
		dirs    []string
		want    []string
	}{
		{"relative files are in the fonts directory", []fontEntry{{"Segoe UI (TrueType)", "segoeui.ttf"}}, nil,
			[]string{`C:\Windows\Fonts\segoeui.ttf`}},
		{"absolute files stay", []fontEntry{{"Mine (OpenType)", `C:\Users\me\AppData\Local\Microsoft\Windows\Fonts\Mine.otf`}, {"Z", `Z:\usr\share\fonts\z.ttf`}}, nil,
			[]string{`C:\Users\me\AppData\Local\Microsoft\Windows\Fonts\Mine.otf`, `Z:\usr\share\fonts\z.ttf`}},
		{"a network path stays", []fontEntry{{"Shared", `\\server\fonts\s.ttf`}}, nil, []string{`\\server\fonts\s.ttf`}},
		{"bitmap and Type 1 fonts are left out", []fontEntry{{"VGA", "vgasys.fon"}, {"Courier", "cour.pfb"}, {"Arial (TrueType)", "arial.ttf"}}, nil,
			[]string{`C:\Windows\Fonts\arial.ttf`}},
		{"the vertical names repeat a file", []fontEntry{{"@MS Gothic", "msgothic.ttc"}, {"MS Gothic & MS UI Gothic (TrueType)", "msgothic.ttc"}}, nil,
			[]string{`C:\Windows\Fonts\msgothic.ttc`}},
		{"the directories add the files the registry lacks, once, without regard to case",
			[]fontEntry{{"Segoe UI (TrueType)", "SegoeUI.ttf"}}, []string{`C:\Windows\Fonts`, `C:\Users\me\AppData\Local\Microsoft\Windows\Fonts`},
			[]string{`C:\Windows\Fonts\SegoeUI.ttf`, `C:\Windows\Fonts\SEGOEUIB.TTF`, `C:\Windows\Fonts\msgothic.ttc`, `C:\Windows\Fonts\arial.ttf`,
				`C:\Users\me\AppData\Local\Microsoft\Windows\Fonts\Mine.otf`}},
		{"nothing", nil, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fontFiles(tt.entries, `C:\Windows\Fonts`, tt.dirs, list)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("files:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

func TestAbsoluteWindowsPaths(t *testing.T) {
	for p, want := range map[string]bool{
		`C:\a.ttf`: true, `c:/a.ttf`: true, `\\server\share\a.ttf`: true, `a.ttf`: false, `Fonts\a.ttf`: false, `C:a.ttf`: false, ``: false,
	} {
		if got := isAbsoluteWindowsPath(p); got != want {
			t.Errorf("isAbsoluteWindowsPath(%q) = %v", p, got)
		}
	}
}
