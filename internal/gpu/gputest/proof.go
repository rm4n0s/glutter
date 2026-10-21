package gputest

import "github.com/rm4n0s/glutter/ui"

// ProofTolerances is how far the GPU renderers may be from the software renderer in each proof
// scene (the scenes are the goldens of the proof tool, SPEC §9.2). A scene that is registered
// without an entry here fails the parity tests, so a step that adds one states its tolerance.
//
// Flutter: no counterpart
var ProofTolerances = map[string]Tolerance{
	"shapes-basic": {Over8Permille: 20, Over32Permille: 5},
	"shapes-clips": {Over8Permille: 30, Over32Permille: 3},
	"gradients":    {Over8Permille: 10, Over32Permille: 3},
	"blend-modes":  {Over8Permille: 15, Over32Permille: 5},
	"effects":      {Over8Permille: 1},
	"text-tour":    {Over8Permille: 1},
	// decorations: 1.0 permille over 8 levels (the edges of the rounded corners and
	// the shadow), one pixel over 32; rendering-tour: 0.1 permille over 8 levels.
	"decorations":    {Over8Permille: 3, Over32Permille: 1},
	"rendering-tour": {Over8Permille: 1},
}

// RetainedFrames makes the scenes of a retained layer: the layer is built in the first
// and added again in the others, as a RepaintBoundary that does not change.
//
// Flutter: no counterpart
func RetainedFrames(n int, clip func(b *ui.SceneBuilder)) []*ui.Scene {
	pic := func(fn func(c *ui.Canvas)) *ui.Picture { return PictureOf(fn) }
	inside := pic(func(c *ui.Canvas) {
		StripesAndDisc(c)
		c.DrawRRect(ui.NewRRectFromRectXY(ui.NewRectFromLTWH(20, 20, 60, 40), 8, 8), FillPaint(0x80FFFFFF))
	})
	under := pic(func(c *ui.Canvas) { c.DrawColor(ui.NewColor(0xFFEEEEEE), ui.BlendModeSrcOver) })
	var scenes []*ui.Scene
	var kept *ui.OffsetEngineLayer
	for i := 0; i < n; i++ {
		b := ui.NewSceneBuilder()
		b.AddPicture(ui.Offset{}, under)
		if clip != nil {
			clip(b)
		}
		b.PushOffset(6.5, 9)
		if kept == nil {
			kept = b.PushOffset(0, 0)
			b.AddPicture(ui.Offset{}, inside)
			b.Pop()
		} else {
			b.AddRetained(kept)
		}
		b.Pop()
		if clip != nil {
			b.Pop()
		}
		// Something that changes in every frame, over the retained layer.
		b.AddPicture(ui.Offset{}, pic(func(c *ui.Canvas) {
			c.DrawCircle(ui.Offset{Dx: 10 + float64(i)*12, Dy: 110}, 6, FillPaint(0xFFE53935))
		}))
		scenes = append(scenes, b.Build())
	}
	return scenes
}

// DamagePicture is a picture of one rectangle, for the tests of damage tracking.
//
// Flutter: no counterpart
func DamagePicture(rect ui.Rect) *ui.Picture {
	rec := ui.NewPictureRecorder()
	ui.NewCanvas(rec).DrawRect(rect, FillPaint(0xFF336699))
	return rec.EndRecording()
}

// DamageScene is a scene of the pictures, in order.
//
// Flutter: no counterpart
func DamageScene(pics ...*ui.Picture) *ui.Scene {
	b := ui.NewSceneBuilder()
	for _, p := range pics {
		b.AddPicture(ui.Offset{}, p)
	}
	return b.Build()
}
