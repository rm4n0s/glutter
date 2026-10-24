package ui

import "testing"

// serviceHost is a host that has one service.
type serviceHost struct{ fakeHost }

func (*serviceHost) PlatformService(name string) any {
	if name == "answer" {
		return 42
	}
	return nil
}

func TestPlatformService(t *testing.T) {
	tests := []struct {
		name string
		host PlatformHost
		ask  string
		want any
	}{
		{"a host with the service", &serviceHost{}, "answer", 42},
		{"a host that does not have that one", &serviceHost{}, "other", nil},
		{"a host without services", &fakeHost{}, "answer", nil},
		{"no host", nil, "answer", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewPlatformDispatcher(tt.host)
			if got := d.PlatformService(tt.ask); got != tt.want {
				t.Errorf("PlatformService(%q) = %v, want %v", tt.ask, got, tt.want)
			}
		})
	}
}
