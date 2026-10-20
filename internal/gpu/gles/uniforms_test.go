package gles

import (
	"testing"

	"github.com/rm4n0s/glutter/ui"
)

func TestUniformsAreSentOnlyWhenTheyChange(t *testing.T) {
	d, gl := newTestDevice(t)
	p, err := d.Program("solid")
	if err != nil {
		t.Fatal(err)
	}
	ramp := []float32{1, 0, 0, 1, 0, 1, 0, 1}
	tests := []struct {
		name string
		call string
		set  func(v int)
		// values are the arguments of the calls, in order; the GL calls made must be
		// the ones that change the value.
		values []int
		want   int
	}{
		{"int", "uniform1i", func(v int) { p.SetInt("uColor", int32(v)) }, []int{1, 1, 2, 2, 1}, 3},
		{"float", "uniform1f", func(v int) { p.SetFloat("uColor", float32(v)) }, []int{1, 1, 2, 1}, 3},
		{"vec2", "uniform2f", func(v int) { p.SetVec2("uColor", float32(v), 0) }, []int{3, 3, 4}, 2},
		{"vec4", "uniform4f", func(v int) { p.SetVec4("uColor", 0, 0, 0, float32(v)) }, []int{3, 3, 4, 4}, 2},
		{"float array", "uniform1fv", func(v int) { p.SetFloats("uColor", ramp[:v]) }, []int{4, 4, 8, 8, 4}, 3},
		{"vec4 array", "uniform4fv", func(v int) { p.SetVec4s("uColor", ramp[:v]) }, []int{4, 4, 8, 4}, 3},
		{"an unknown name", "uniform1i", func(v int) { p.SetInt("missingColor", int32(v)) }, []int{1, 2}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A value of another type at the location is a change too.
			p.forgetUniforms()
			gl.resetCalls()
			for _, v := range tt.values {
				tt.set(v)
			}
			if got := gl.calls[tt.call]; got != tt.want {
				t.Errorf("%s was called %d times for %v, want %d", tt.call, got, tt.values, tt.want)
			}
		})
	}
	s := d.Stats()
	if s.UniformsSet == 0 || s.UniformsSkipped == 0 {
		t.Errorf("the counters did not move: %+v", s)
	}
	// ResetState forgets what was set, so the next call reaches GL.
	p.SetInt("uColor", 5)
	d.ResetState()
	gl.resetCalls()
	p.SetInt("uColor", 5)
	if gl.calls["uniform1i"] != 1 {
		t.Errorf("a uniform after ResetState was sent %d times, want 1", gl.calls["uniform1i"])
	}
}

func TestClearColorIsSentOnlyWhenItChanges(t *testing.T) {
	d, gl := newTestDevice(t)
	gl.resetCalls()
	d.ClearColor(0, 0, 0, 0)
	d.ClearColor(0, 0, 0, 0)
	d.ClearColor(1, 0, 0, 1)
	d.ClearColor(1, 0, 0, 1)
	if gl.calls["clearColor"] != 2 {
		t.Errorf("clearColor was called %d times, want 2", gl.calls["clearColor"])
	}
	before := d.Stats().StateSkipped
	d.Viewport(0, 0, 10, 10)
	d.Viewport(0, 0, 10, 10)
	if d.Stats().StateSkipped != before+1 {
		t.Errorf("a repeated viewport was not counted as skipped")
	}
}

// TestASteadyFrameSendsFewerCallsThanTheFirst draws the same scene in two frames and checks that
// the second one sends no uniform that the first one set to the same value.
func TestASteadyFrameSendsFewerCallsThanTheFirst(t *testing.T) {
	r, _, gl := newTestRenderer()
	s, err := r.Attach(waylandHandle)
	if err != nil {
		t.Fatal(err)
	}
	b := ui.NewSceneBuilder()
	rec := ui.NewPictureRecorder()
	c := ui.NewCanvas(rec)
	for i := 0; i < 6; i++ {
		c.DrawRect(ui.NewRectFromLTWH(float64(i*10), 0, 8, 8), solidPaint(0xFF000000+uint32(i)*0x101010))
		c.DrawCircle(ui.Offset{Dx: float64(i * 10), Dy: 30}, 4, solidPaint(0xFF3366CC))
	}
	b.AddPicture(ui.Offset{}, rec.EndRecording())
	scene := b.Build()
	size := ui.Size{Width: 80, Height: 60}
	count := func() (uniforms int) {
		gl.resetCalls()
		if err := r.Render(s, scene, size, 1); err != nil {
			t.Fatal(err)
		}
		for name, n := range gl.calls {
			if len(name) > 7 && name[:7] == "uniform" {
				uniforms += n
			}
		}
		return uniforms
	}
	first, second := count(), count()
	if first == 0 || second >= first {
		t.Errorf("uniform calls: first frame %d, second frame %d; the second should send fewer", first, second)
	}
	if second > 4 {
		t.Errorf("a steady frame sent %d uniform calls", second)
	}
}
