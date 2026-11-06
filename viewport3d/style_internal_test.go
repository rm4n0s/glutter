package viewport3d

import (
	"testing"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/ui"
)

func TestTheStyleResolvesItsDefaults(t *testing.T) {
	light := ui.NewColor(0xFFF4F4F4)
	red := ui.NewColor(0xFFFF0000)
	tests := []struct {
		name      string
		style     SceneStyle
		wantMode  DisplayMode
		wantEdge  ui.Color
		wantWidth float64
		wantLamps int
	}{
		{"the zero style", SceneStyle{}, DisplayModeShaded, defaultEdgeColor, 1, 0},
		{"edges over faces are dark", SceneStyle{DisplayMode: DisplayModeShadedWithEdges}, DisplayModeShadedWithEdges, defaultEdgeColor, 1, 0},
		{"a wireframe on a dark background has light edges", SceneStyle{DisplayMode: DisplayModeWireframe}, DisplayModeWireframe, lightEdgeColor, 1, 0},
		{"a hidden line view on a dark background has light edges", SceneStyle{DisplayMode: DisplayModeHiddenLine}, DisplayModeHiddenLine, lightEdgeColor, 1, 0},
		{"a wireframe on a light background has dark edges", SceneStyle{DisplayMode: DisplayModeWireframe, Background: foundation.Some(light)}, DisplayModeWireframe, defaultEdgeColor, 1, 0},
		{"an edge color that was given", SceneStyle{DisplayMode: DisplayModeWireframe, EdgeColor: foundation.Some(red), EdgeWidth: 3}, DisplayModeWireframe, red, 3, 0},
		{"more lights than there are", SceneStyle{Lights: make([]DirectionalLight, 7)}, DisplayModeShaded, defaultEdgeColor, 1, maxLights},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.style.resolve()
			if r.mode != tt.wantMode || r.edgeColor != tt.wantEdge || r.edgeWidth != tt.wantWidth || len(r.lights) != tt.wantLamps {
				t.Errorf("mode %v, edge %v, width %v, lights %d; want %v, %v, %v, %d", r.mode, r.edgeColor, r.edgeWidth, len(r.lights), tt.wantMode, tt.wantEdge, tt.wantWidth, tt.wantLamps)
			}
			if r.ambient != defaultAmbient || r.headLight != defaultHeadLight || r.edgeAngle != defaultEdgeAngle {
				t.Errorf("ambient %v, head light %v, edge angle %v", r.ambient, r.headLight, r.edgeAngle)
			}
		})
	}
}
