//go:build linux && !android

package x11

import "testing"

func TestParseXftDPI(t *testing.T) {
	tests := []struct {
		name      string
		resources string
		want      float64
		ok        bool
	}{
		{"set", "Xft.antialias:\t1\nXft.dpi:\t144\nXft.hinting:\t1\n", 144, true},
		{"spaces", "  Xft.dpi :  192.5  \n", 192.5, true},
		{"missing", "Xft.antialias:\t1\n*.dpi: 200\n", 0, false},
		{"another resource with the name inside", "Foo.Xft.dpi: 300\n", 0, false},
		{"not a number", "Xft.dpi:\tbig\n", 0, false},
		{"zero", "Xft.dpi:\t0\n", 0, false},
		{"negative", "Xft.dpi:\t-96\n", 0, false},
		{"empty", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseXftDPI(tt.resources)
			if got != tt.want || ok != tt.ok {
				t.Errorf("parseXftDPI = %v, %v; want %v, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestScaleFor(t *testing.T) {
	// A 27" 4K monitor is 597 mm wide; a small laptop screen at 1920 is 330 mm, 148 dpi.
	fourK := monitor{width: 3840, height: 2160, mmWidth: 597, mmHeight: 336}
	laptop := monitor{width: 1920, height: 1080, mmWidth: 330, mmHeight: 186}
	desk := monitor{width: 1920, height: 1080, mmWidth: 527, mmHeight: 296}
	tests := []struct {
		name string
		xft  float64
		m    monitor
		want float64
	}{
		{"no setting, the usual monitor", 0, desk, 1},
		{"no setting, 4K from its size", 0, fourK, 1.75},
		{"no setting, a laptop from its size", 0, laptop, 1.5},
		{"no setting, no size", 0, monitor{width: 1920, height: 1080}, 1},
		{"no setting, no monitor", 0, monitor{}, 1},
		{"Xft.dpi wins over the size", 96, fourK, 1},
		{"Xft.dpi 192", 192, desk, 2},
		{"Xft.dpi 120", 120, desk, 1.25},
		{"Xft.dpi 100 is no change", 100, desk, 1},
		{"Xft.dpi below 96 is 1", 72, desk, 1},
		{"capped", 1000, desk, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scaleFor(tt.xft, tt.m); got != tt.want {
				t.Errorf("scaleFor(%v, %+v) = %v, want %v", tt.xft, tt.m, got, tt.want)
			}
		})
	}
}

func TestMonitorAtFallsBackToThePrimaryMonitor(t *testing.T) {
	p := &Platform{monitors: []monitor{
		{id: 1, x: 0, y: 0, width: 1920, height: 1080},
		{id: 2, x: 1920, y: 0, width: 3840, height: 2160, primary: true},
	}}
	tests := []struct {
		name string
		x, y int32
		want int
	}{
		{"on the first", 10, 10, 1},
		{"on the second", 2000, 100, 2},
		{"the edge belongs to the next", 1920, 0, 2},
		{"outside every monitor: the primary", -50, 5000, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, ok := p.monitorAt(tt.x, tt.y)
			if !ok || m.id != tt.want {
				t.Errorf("monitorAt(%d, %d) = %+v, %v; want monitor %d", tt.x, tt.y, m, ok, tt.want)
			}
		})
	}
	if _, ok := (&Platform{}).monitorAt(0, 0); ok {
		t.Error("monitorAt found a monitor on a screen without any")
	}
}
