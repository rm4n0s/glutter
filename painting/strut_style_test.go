package painting

import (
	"reflect"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func TestStrutStyleFromTextStyle(t *testing.T) {
	ts := NewTextStyle(TextStyleOpts{FontFamily: "Roboto", FontFamilyFallback: []string{"Noto"}, FontSize: f64(12), Height: f64(1.5),
		LeadingDistribution: ui.TextLeadingDistributionEven, FontWeight: ui.FontWeightW500, FontStyle: ui.FontStyleItalic, DebugLabel: "ts"})
	s := NewStrutStyleFromTextStyle(ts, StrutStyleOpts{Leading: f64(0.5), ForceStrutHeight: foundation.Some(true)})
	if s.FontFamily != "Roboto" || !reflect.DeepEqual(s.FontFamilyFallback(), []string{"Noto"}) || s.FontSize.Value() != 12 ||
		s.Height.Value() != 1.5 || s.LeadingDistribution != ui.TextLeadingDistributionEven || s.FontWeight != ui.FontWeightW500 ||
		s.FontStyle != ui.FontStyleItalic || s.Leading.Value() != 0.5 || !s.ForceStrutHeight.Value() || s.DebugLabel != "ts" {
		t.Errorf("got %+v", s)
	}
	o := NewStrutStyleFromTextStyle(ts, StrutStyleOpts{FontFamily: "Other", FontSize: f64(20)})
	if o.FontFamily != "Other" || o.FontSize.Value() != 20 {
		t.Errorf("overrides: %+v", o)
	}
}

func TestStrutStyleMergeAndInherit(t *testing.T) {
	a := NewStrutStyle(StrutStyleOpts{FontFamily: "A", FontSize: f64(10), Leading: f64(1)})
	b := NewStrutStyle(StrutStyleOpts{FontSize: f64(20), FontWeight: ui.FontWeightW700})
	m := a.Merge(b)
	if m.FontFamily != "A" || m.FontSize.Value() != 20 || m.FontWeight != ui.FontWeightW700 || m.Leading.Value() != 1 {
		t.Errorf("merge: %+v", m)
	}
	if a.Merge(nil) != a {
		t.Error("merging nil returns the receiver")
	}
	ts := NewTextStyle(TextStyleOpts{FontFamily: "T", FontSize: f64(7), Height: f64(2), LeadingDistribution: ui.TextLeadingDistributionEven})
	in := a.InheritFromTextStyle(ts)
	if in.FontFamily != "A" || in.FontSize.Value() != 10 || in.Height.Value() != 2 || in.LeadingDistribution != ui.TextLeadingDistributionEven || in.Leading.Value() != 1 {
		t.Errorf("inherit: %+v", in)
	}
	noHeight := NewStrutStyle().InheritFromTextStyle(NewTextStyle(TextStyleOpts{LeadingDistribution: ui.TextLeadingDistributionEven}))
	if noHeight.LeadingDistribution != ui.TextLeadingDistributionUnspecified {
		t.Error("the leading distribution is only inherited with a height")
	}
	if a.InheritFromTextStyle(nil) != a {
		t.Error("inheriting from nil returns the receiver")
	}
}

func TestStrutStyleCompareAndEqual(t *testing.T) {
	base := NewStrutStyle(StrutStyleOpts{FontSize: f64(10), Height: f64(1)})
	tests := []struct {
		name  string
		other *StrutStyle
		cmp   RenderComparison
		equal bool
	}{
		{"same", base, RenderComparisonIdentical, true},
		{"equal", NewStrutStyle(StrutStyleOpts{FontSize: f64(10), Height: f64(1)}), RenderComparisonIdentical, true},
		{"size", NewStrutStyle(StrutStyleOpts{FontSize: f64(11), Height: f64(1)}), RenderComparisonLayout, false},
		{"force", NewStrutStyle(StrutStyleOpts{FontSize: f64(10), Height: f64(1), ForceStrutHeight: foundation.Some(true)}), RenderComparisonLayout, false},
		{"distribution matters with a height", NewStrutStyle(StrutStyleOpts{FontSize: f64(10), Height: f64(1), LeadingDistribution: ui.TextLeadingDistributionEven}), RenderComparisonLayout, false},
		{"label is ignored", NewStrutStyle(StrutStyleOpts{FontSize: f64(10), Height: f64(1), DebugLabel: "x"}), RenderComparisonIdentical, true},
	}
	for _, tt := range tests {
		if got := base.CompareTo(tt.other); got != tt.cmp {
			t.Errorf("%s: CompareTo = %v, want %v", tt.name, got, tt.cmp)
		}
		if got := base.Equal(tt.other); got != tt.equal {
			t.Errorf("%s: Equal = %v, want %v", tt.name, got, tt.equal)
		}
	}
	noHeight := NewStrutStyle(StrutStyleOpts{FontSize: f64(10)})
	if !noHeight.Equal(NewStrutStyle(StrutStyleOpts{FontSize: f64(10), LeadingDistribution: ui.TextLeadingDistributionEven})) {
		t.Error("without a height the distribution is ignored")
	}
	if !StrutStyleDisabled().Equal(NewStrutStyle(StrutStyleOpts{Height: f64(0), Leading: f64(0)})) {
		t.Error("disabled is a strut with no height and no leading")
	}
}

func TestStrutStyleDiagnostics(t *testing.T) {
	if !foundation.DebugMode {
		t.Skip("release and profile builds print only short names")
	}
	tests := []struct {
		name string
		s    *StrutStyle
		want string
	}{
		{"empty", NewStrutStyle(), "StrutStyle(forceStrutHeight: null)"},
		{"forced", NewStrutStyle(StrutStyleOpts{ForceStrutHeight: foundation.Some(true)}), "StrutStyle(<strut height forced>)"},
		{"some", NewStrutStyle(StrutStyleOpts{FontFamily: "Roboto", FontSize: f64(12), FontWeight: ui.FontWeightW700, Height: f64(2), LeadingDistribution: ui.TextLeadingDistributionEven}),
			"StrutStyle(family: Roboto, size: 12.0, weight: w700, height: 2.0x, forceStrutHeight: null, leadingDistribution: even)"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("%s: String = %q, want %q", tt.name, got, tt.want)
		}
	}
}
