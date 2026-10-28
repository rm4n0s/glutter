package ui

import (
	"testing"

	"github.com/rm4n0s/errors"
)

// metricsEngine is a text engine that measures a style by its size and records the style it was given.
type metricsEngine struct {
	nopFactory
	got ResolvedTextStyle
}

func (e *metricsEngine) FontMetrics(style *ResolvedTextStyle) FontMetrics {
	e.got = *style
	return FontMetrics{Ascent: style.FontSize, Descent: style.FontSize / 4, Height: style.FontSize * 1.25, ZeroAdvance: style.FontSize / 2, Monospaced: len(style.FontFamilies) > 0}
}

func TestPlatformDispatcherFontMetrics(t *testing.T) {
	tests := []struct {
		name      string
		engine    ParagraphFactory
		style     *TextStyle
		want      FontMetrics
		wantSize  float64
		wantRoute string
	}{
		{"a style with a size", &metricsEngine{}, NewTextStyle(TextStyleOpts{FontFamily: "Mono", FontSize: Ptr(20.0)}),
			FontMetrics{Ascent: 20, Descent: 5, Height: 25, ZeroAdvance: 10, Monospaced: true}, 20, ""},
		{"a style without a size takes the paragraph's", &metricsEngine{}, NewTextStyle(),
			FontMetrics{Ascent: 14, Descent: 3.5, Height: 17.5, ZeroAdvance: 7}, 14, ""},
		{"no style is the default style", &metricsEngine{}, nil,
			FontMetrics{Ascent: 14, Descent: 3.5, Height: 17.5, ZeroAdvance: 7}, 14, ""},
		{"no engine", nil, nil, FontMetrics{}, 0, "PlatformDispatcher.FontMetrics.FontMetricsUnsupported"},
		{"an engine that cannot measure", nopFactory{}, nil, FontMetrics{}, 0, "PlatformDispatcher.FontMetrics.FontMetricsUnsupported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewPlatformDispatcher(nil)
			if tt.engine != nil {
				d.SetTextEngine(tt.engine)
			}
			got, err := d.FontMetrics(tt.style)
			if tt.wantRoute != "" {
				appErr, ok := errors.FromError(err)
				if !ok {
					t.Fatalf("expected *errors.Error, got %T (%v)", err, err)
				}
				if !appErr.HasRoute(tt.wantRoute) {
					t.Errorf("unexpected failure path: %s", appErr.Route())
				}
				return
			}
			if err != nil {
				t.Fatalf("FontMetrics: %v", err)
			}
			if got != tt.want {
				t.Errorf("FontMetrics = %+v, want %+v", got, tt.want)
			}
			if e := tt.engine.(*metricsEngine); e.got.FontSize != tt.wantSize {
				t.Errorf("the engine measured a style of size %v, want %v", e.got.FontSize, tt.wantSize)
			}
		})
	}
}
