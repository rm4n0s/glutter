package web

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]any
		want   Config
	}{
		{"nothing", nil, Config{}},
		{"empty object", map[string]any{}, Config{}},
		{"null host", map[string]any{"hostElement": nil}, Config{}},
		{"selector", map[string]any{"hostElement": "#app"}, Config{HostElement: "#app"}},
		{"selector is trimmed", map[string]any{"hostElement": "  .stage > div \n"}, Config{HostElement: ".stage > div"}},
		{"unknown fields are ignored", map[string]any{"renderer": "soft", "hostElement": "body"}, Config{HostElement: "body"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseConfig(tc.fields)
			if err != nil {
				t.Fatalf("ParseConfig() failed: %v", err)
			}
			if got != tc.want {
				t.Errorf("ParseConfig() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseConfigRejectsAWrongType(t *testing.T) {
	for _, v := range []any{3.0, true, []any{"#a"}, map[string]any{}} {
		_, err := ParseConfig(map[string]any{"hostElement": v})
		appErr, ok := errors.FromError(err)
		if !ok {
			t.Fatalf("hostElement %v: expected *errors.Error, got %T", v, err)
		}
		if appErr.Tag != "WebConfigInvalid" || !appErr.HasRoute("TestParseConfigRejectsAWrongType->ParseConfig.WebConfigInvalid") {
			t.Errorf("hostElement %v: got %s via %s, want WebConfigInvalid via ParseConfig", v, appErr.Tag, appErr.Route())
		}
	}
}
