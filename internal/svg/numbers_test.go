package svg

import (
	"math"
	"testing"
)

func TestScanNumber(t *testing.T) {
	tests := []struct {
		in   string
		want float64
		next int
		ok   bool
	}{
		{"12", 12, 2, true},
		{"-12.5", -12.5, 5, true},
		{"+.5", 0.5, 3, true},
		{".5.5", 0.5, 2, true},
		{"1.5.5", 1.5, 3, true},
		{"-1-2", -1, 2, true},
		{"1e2", 100, 3, true},
		{"1E-2x", 0.01, 4, true},
		{"1em", 1, 1, true},
		{"1ex", 1, 1, true},
		{"1e+", 1, 1, true},
		{"5.", 5, 2, true},
		{"1234567890123456789", 1234567890123456789, 19, true},
		{"abc", 0, 0, false},
		{"-", 0, 0, false},
		{".", 0, 0, false},
		{"inf", 0, 0, false},
		{"nan", 0, 0, false},
		{"0x10", 0, 1, true},
		{"1e999", 0, 0, false},
	}
	for _, tt := range tests {
		got, next, ok := scanNumber(tt.in, 0)
		if ok != tt.ok {
			t.Errorf("scanNumber(%q) ok = %v, want %v", tt.in, ok, tt.ok)
			continue
		}
		if ok && (math.Abs(got-tt.want) > 1e-9*math.Max(1, math.Abs(tt.want)) || next != tt.next) {
			t.Errorf("scanNumber(%q) = %v, %d; want %v, %d", tt.in, got, next, tt.want, tt.next)
		}
	}
}

func TestParseNumberList(t *testing.T) {
	tests := []struct {
		in   string
		want []float64
		ok   bool
	}{
		{"1 2 3", []float64{1, 2, 3}, true},
		{"1,2,3", []float64{1, 2, 3}, true},
		{" 1 , 2  3 ", []float64{1, 2, 3}, true},
		{"1-2", []float64{1, -2}, true},
		{"", nil, true},
		{"1 a", nil, false},
		{"1,,2", nil, false},
	}
	for _, tt := range tests {
		got, ok := ParseNumberList(tt.in)
		if ok != tt.ok || len(got) != len(tt.want) {
			t.Errorf("ParseNumberList(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ParseNumberList(%q) = %v; want %v", tt.in, got, tt.want)
			}
		}
	}
}

func TestParseLengthUnits(t *testing.T) {
	tests := []struct {
		in   string
		want Length
		ok   bool
		px   float64 // the length in user units with the default context (font 10, x-height 4, viewport 200x100)
	}{
		{"10", Length{10, UnitPx}, true, 10},
		{"10px", Length{10, UnitPx}, true, 10},
		{" 10PX ", Length{10, UnitPx}, true, 10},
		{"9pt", Length{9, UnitPt}, true, 12},
		{"2pc", Length{2, UnitPc}, true, 32},
		{"25.4mm", Length{25.4, UnitMm}, true, 96},
		{"2.54cm", Length{2.54, UnitCm}, true, 96},
		{"1in", Length{1, UnitIn}, true, 96},
		{"2em", Length{2, UnitEm}, true, 20},
		{"3ex", Length{3, UnitEx}, true, 12},
		{"1.5rem", Length{1.5, UnitRem}, true, 21},
		{"50%", Length{50, UnitPercent}, true, 100},
		{"1e1px", Length{10, UnitPx}, true, 10},
		{"10 px", Length{}, false, 0},
		{"10foo", Length{}, false, 0},
		{"", Length{}, false, 0},
		{"px", Length{}, false, 0},
	}
	lc := lenCtx{fontSize: 10, rootFont: 14, xRatio: 0.4, vw: 200, vh: 100, vpKnown: true}
	for _, tt := range tests {
		got, ok := ParseLength(tt.in)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("ParseLength(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
			continue
		}
		if ok {
			px, _ := lc.px(got, axisX)
			if math.Abs(px-tt.px) > 1e-9 {
				t.Errorf("%q resolves to %v, want %v", tt.in, px, tt.px)
			}
		}
	}
}

func TestPercentagesAreRelativeToTheirAxis(t *testing.T) {
	lc := lenCtx{fontSize: 10, vw: 200, vh: 100, vpKnown: true}
	tests := []struct {
		axis axis
		want float64
	}{
		{axisX, 100},
		{axisY, 50},
		{axisXY, 0.5 * math.Sqrt((200*200+100*100)/2.0)},
	}
	for _, tt := range tests {
		got, ok := lc.px(Length{50, UnitPercent}, tt.axis)
		if !ok || math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("axis %d: %v, %v; want %v", tt.axis, got, ok, tt.want)
		}
	}
	unknown := lenCtx{fontSize: 10}
	if _, ok := unknown.px(Length{50, UnitPercent}, axisX); ok {
		t.Error("a percentage without a viewport resolved")
	}
	if got := unknown.keep(Length{50, UnitPercent}); got != (Length{50, UnitPercent}) {
		t.Errorf("keep(percent) = %v", got)
	}
	if got := lc.keep(Length{2, UnitEm}); got != (Length{20, UnitPx}) {
		t.Errorf("keep(2em) = %v", got)
	}
}

func TestParseViewBoxAndPreserveAspectRatio(t *testing.T) {
	vbTests := []struct {
		in   string
		want ViewBox
		ok   bool
	}{
		{"0 0 24 24", ViewBox{0, 0, 24, 24}, true},
		{"-5,-5,10,10", ViewBox{-5, -5, 10, 10}, true},
		{"0 0 0 10", ViewBox{0, 0, 0, 10}, true},
		{"0 0 24", ViewBox{}, false},
		{"a b c d", ViewBox{}, false},
	}
	for _, tt := range vbTests {
		got, ok, err := parseViewBox(tt.in)
		if err != nil || ok != tt.ok || got != tt.want {
			t.Errorf("parseViewBox(%q) = %v, %v, %v; want %v, %v", tt.in, got, ok, err, tt.want, tt.ok)
		}
	}
	parTests := []struct {
		in   string
		want PreserveAspectRatio
		ok   bool
	}{
		{"xMidYMid", PreserveAspectRatio{AlignXMidYMid, false}, true},
		{"xMidYMid meet", PreserveAspectRatio{AlignXMidYMid, false}, true},
		{"xMinYMin slice", PreserveAspectRatio{AlignXMinYMin, true}, true},
		{"defer xMaxYMax meet", PreserveAspectRatio{AlignXMaxYMax, false}, true},
		{"none", PreserveAspectRatio{AlignNone, false}, true},
		{"xMidYMin", PreserveAspectRatio{AlignXMidYMin, false}, true},
		{"xMaxYMin", PreserveAspectRatio{AlignXMaxYMin, false}, true},
		{"xMinYMid", PreserveAspectRatio{AlignXMinYMid, false}, true},
		{"xMaxYMid", PreserveAspectRatio{AlignXMaxYMid, false}, true},
		{"xMinYMax", PreserveAspectRatio{AlignXMinYMax, false}, true},
		{"xMidYMax", PreserveAspectRatio{AlignXMidYMax, false}, true},
		{"xMidYMid cover", PreserveAspectRatio{}, false},
		{"nonsense", PreserveAspectRatio{}, false},
		{"", PreserveAspectRatio{}, false},
	}
	for _, tt := range parTests {
		got, ok := ParsePreserveAspectRatio(tt.in)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("ParsePreserveAspectRatio(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}
