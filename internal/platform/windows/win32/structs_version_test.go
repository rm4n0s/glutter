package win32

import "testing"

func TestFileVersionOf(t *testing.T) {
	tests := []struct {
		name string
		info VSFixedFileInfo
		want FileVersion
		text string
	}{
		{"d3dcompiler_47 on Windows 10", VSFixedFileInfo{FileVersionMS: 10 << 16, FileVersionLS: 19041<<16 | 1}, FileVersion{10, 0, 19041, 1}, "10.0.19041.1"},
		{"the highest numbers", VSFixedFileInfo{FileVersionMS: 0xFFFFFFFF, FileVersionLS: 0xFFFFFFFF}, FileVersion{65535, 65535, 65535, 65535}, "65535.65535.65535.65535"},
		{"nothing", VSFixedFileInfo{}, FileVersion{}, "0.0.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FileVersionOf(tt.info)
			if got != tt.want || got.String() != tt.text {
				t.Errorf("FileVersionOf = %+v (%s), want %+v (%s)", got, got, tt.want, tt.text)
			}
		})
	}
}
