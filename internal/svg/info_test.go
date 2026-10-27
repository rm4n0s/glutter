package svg

import (
	"strings"
	"testing"
)

func TestScaleDependent(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"a plain shape", `<rect width="10" height="10"/>`, false},
		{"a gradient", `<defs><linearGradient id="g"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient></defs><rect width="10" height="10" fill="url(#g)"/>`, false},
		{"a clip path and a mask", `<defs><clipPath id="c"><rect width="5" height="5"/></clipPath><mask id="m"><rect width="9" height="9" fill="white"/></mask></defs><rect width="10" height="10" clip-path="url(#c)" mask="url(#m)"/>`, false},
		{"text", `<text x="1" y="10">Hi</text>`, false},
		{"a pattern", `<defs><pattern id="p" width="4" height="4" patternUnits="userSpaceOnUse"><rect width="2" height="2"/></pattern></defs><rect width="10" height="10" fill="url(#p)"/>`, true},
		{"a filter", `<defs><filter id="f"><feGaussianBlur stdDeviation="1"/></filter></defs><rect width="10" height="10" filter="url(#f)"/>`, true},
		{"a stroke that does not scale", `<path d="M0 0L10 10" stroke="black" vector-effect="non-scaling-stroke"/>`, true},
		{"an svg image", `<image width="4" height="4" href="data:image/svg+xml;utf8,%3Csvg xmlns='http://www.w3.org/2000/svg' width='1' height='1'/%3E"/>`, true},
		{"a pattern that nothing uses", `<defs><pattern id="p" width="4" height="4"><rect width="2" height="2"/></pattern></defs><rect width="10" height="10"/>`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, svgOpen+tt.body+`</svg>`)
			if got := d.ScaleDependent(); got != tt.want {
				t.Errorf("ScaleDependent = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApproximateBytes(t *testing.T) {
	small := mustParse(t, svgOpen+`<rect width="10" height="10"/></svg>`)
	var many strings.Builder
	for range 200 {
		many.WriteString(`<path d="M0 0L10 10L20 0Z"/>`)
	}
	large := mustParse(t, svgOpen+many.String()+`</svg>`)
	image := mustParse(t, svgOpen+`<image width="4" height="4" href="data:image/png;base64,`+strings.Repeat("QUJD", 2000)+`"/></svg>`)
	tests := []struct {
		name     string
		got      int
		min, max int
	}{
		{"a small document", small.ApproximateBytes(), 512, 4096},
		{"a large document", large.ApproximateBytes(), 200 * 300, 200 * 3000},
		{"an image counts its data", image.ApproximateBytes(), 6000, 20000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got < tt.min || tt.got > tt.max {
				t.Errorf("ApproximateBytes = %d, want %d to %d", tt.got, tt.min, tt.max)
			}
		})
	}
	if small.ApproximateBytes() >= large.ApproximateBytes() {
		t.Errorf("a small document (%d) is not smaller than a large one (%d)", small.ApproximateBytes(), large.ApproximateBytes())
	}
}
