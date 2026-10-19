//go:build !glutter_release && !glutter_profile

package painting

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestEdgeInsetsResolveNeedsTextDirection(t *testing.T) {
	a := EdgeInsets{Top: 5, Left: 5}
	b := EdgeInsetsDirectional{Top: 15, Start: 15}
	tests := []struct {
		name string
		call func()
	}{
		{"directional", func() { NewEdgeInsetsDirectionalAll(10).Resolve(ui.TextDirectionUnspecified) }},
		{"mixed add", func() { a.Add(b).Resolve(ui.TextDirectionUnspecified) }},
		{"mixed subtract", func() { b.Subtract(a).Resolve(ui.TextDirectionUnspecified) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireTaggedPanic(t, "TextDirectionRequired", tt.call)
		})
	}
}

func TestBorderRadiusResolveNeedsTextDirection(t *testing.T) {
	tests := []struct {
		name string
		call func()
	}{
		{"directional", func() { NewBorderRadiusDirectionalCircular(1).Resolve(ui.TextDirectionUnspecified) }},
		{"mixed", func() {
			BorderRadius{TopLeft: ui.NewRadiusCircular(1)}.Add(BorderRadiusDirectional{TopStart: ui.NewRadiusCircular(1)}).Resolve(ui.TextDirectionUnspecified)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireTaggedPanic(t, "TextDirectionRequired", tt.call)
		})
	}
}
