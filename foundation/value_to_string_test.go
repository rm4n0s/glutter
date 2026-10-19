package foundation

import "testing"

func TestDiagnosticsPropertyValueToStringOption(t *testing.T) {
	p := NewDiagnosticsProperty("value", 3, DiagnosticsPropertyOpts{ValueToString: func(parent *TextTreeConfiguration) string {
		if parent != nil && !parent.LineBreakProperties {
			return "single"
		}
		return "multi"
	}})
	tests := []struct {
		name   string
		parent *TextTreeConfiguration
		want   string
	}{
		{"line breaks", &TextTreeConfiguration{LineBreakProperties: true}, "multi"},
		{"single line", &TextTreeConfiguration{LineBreakProperties: false}, "single"},
		{"no parent", nil, "multi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.ValueToString(tt.parent); got != tt.want {
				t.Errorf("ValueToString = %q, want %q", got, tt.want)
			}
		})
	}
}
