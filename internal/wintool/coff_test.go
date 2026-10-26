package wintool

import "testing"

func TestFindArch(t *testing.T) {
	tests := []struct {
		name    string
		machine uint16
		addrNB  uint16
		ok      bool
	}{
		{"amd64", 0x8664, 3, true},
		{"arm64", 0xAA64, 2, true},
		{"386", 0, 0, false},
		{"", 0, 0, false},
	}
	for _, tt := range tests {
		a, ok := FindArch(tt.name)
		if ok != tt.ok || a.Machine != tt.machine || a.AddrNB != tt.addrNB {
			t.Errorf("FindArch(%q) = %+v, %v", tt.name, a, ok)
		}
	}
}
