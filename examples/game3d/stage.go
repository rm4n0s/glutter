package main

import (
	"context"
	"math"
	"strings"

	"github.com/rm4n0s/glutter/foundation"
	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/ui"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// stage is what the player sees: the scene with the level and the character, and the camera that follows the character. The
// world is where the game is; the stage shows it (sync), once for each frame.
type stage struct {
	scene  *viewport3d.Scene
	camera *viewport3d.Camera

	level, character *viewport3d.Uploaded
	// charItems are the items of the character, kept so that a frame does not ask for them again.
	charItems []*viewport3d.Item
	anims     map[string]*viewport3d.Animation
	pose      *viewport3d.Pose

	// coins are the items of the coins, in the order of the world's coins, and spin is their turn.
	coins []*viewport3d.Item
	spin  float64

	// yaw and pitch are the camera's: yaw is the direction it looks in (0 looks along -z), pitch how far down.
	yaw, pitch float64
	eye        vecmath.Vec3 // the point that the camera follows, which trails the character a little
}

// The camera: how far behind the character, how high its target is above the feet, and how far it may look up and down.
const (
	cameraDistance = 7.0
	cameraHeight   = 1.4
	minPitch       = -0.35
	maxPitch       = 1.2
)

// newStage makes the scene, with its sky and its sun, and the camera.
func newStage() *stage {
	s := &stage{scene: viewport3d.NewScene(), pose: viewport3d.NewPose(), yaw: math.Pi, pitch: 0.42}
	s.camera = viewport3d.NewCamera(viewport3d.CameraOpts{Distance: cameraDistance, FieldOfView: 0.8, Near: 0.1})
	s.scene.SetStyle(viewport3d.SceneStyle{
		Background: foundation.Some(ui.NewColorFromARGB(255, 142, 190, 235)),
		Lights: []viewport3d.DirectionalLight{{
			Direction: vecmath.Vec3{X: -0.45, Y: 1, Z: 0.3}, Color: ui.NewColorFromARGB(255, 255, 244, 224), Intensity: 1.8,
		}},
		Shadow:               viewport3d.ShadowStyle{Enabled: true, Size: 2048, Strength: foundation.Some(0.8)},
		EnvironmentIntensity: foundation.Some(0.55),
	})
	return s
}

// load makes the level and the character on the device and puts them in the scene. It fails with what Upload fails with.
func (s *stage) load(ctx context.Context, dev *gpu.Device, a *assets) error {
	s.release()
	level, err := viewport3d.Upload(ctx, dev, a.level)
	if err != nil {
		return err
	}
	character, err := viewport3d.Upload(ctx, dev, a.character)
	if err != nil {
		level.Release()
		return err
	}
	s.level, s.character = level, character
	s.anims = map[string]*viewport3d.Animation{}
	for _, an := range a.character.Animations {
		s.anims[an.Name] = viewport3d.NewAnimation(an)
	}
	s.coins = s.coins[:0]
	for _, p := range level.Parts {
		if strings.HasPrefix(p.Node.Name, "Coin") {
			s.coins = append(s.coins, p.Item)
		}
	}
	s.scene.Add(level.Items()...)
	s.charItems = character.Items()
	s.scene.Add(s.charItems...)
	return nil
}

// release takes the level and the character out of the scene and frees them.
func (s *stage) release() {
	for _, u := range []*viewport3d.Uploaded{s.level, s.character} {
		if u != nil {
			s.scene.Remove(u.Items()...)
			u.Release()
		}
	}
	s.level, s.character, s.charItems, s.coins = nil, nil, nil, s.coins[:0]
}

// dispose frees everything of the stage.
func (s *stage) dispose() {
	s.release()
	s.camera.Dispose()
}

// look turns the camera by the motion of a mouse, a stick or the keys: radians to the right and down.
func (s *stage) look(dyaw, dpitch float64) {
	s.yaw = math.Mod(s.yaw-dyaw, 2*math.Pi)
	s.pitch = math.Max(minPitch, math.Min(maxPitch, s.pitch+dpitch))
}

// forward is the direction the camera looks in, on the ground plane, and right the direction to its right: the move of the
// player's input is turned by them.
func (s *stage) forward() vecmath.Vec3 { return vecmath.Vec3{X: -math.Sin(s.yaw), Z: -math.Cos(s.yaw)} }
func (s *stage) right() vecmath.Vec3   { f := s.forward(); return vecmath.Vec3{X: -f.Z, Z: f.X} }

// move is the move that the player asked for (x to the right, y forward, both from -1 to 1) in the world's plane.
func (s *stage) move(x, forward float64) vecmath.Vec3 {
	return s.right().Scale(x).Add(s.forward().Scale(forward))
}

// sync shows the world: the character at its place with the pose of its animation, the coins turning (and gone when they were
// picked up), the character blinking while it cannot be hurt, and the camera behind it. dt is the time of the frame.
func (s *stage) sync(w *world, dt float64) {
	if s.character == nil {
		return
	}
	s.character.SetTransform(vecmath.NewMat4Translation(w.pos).Mul(vecmath.NewMat4RotationY(w.yaw)))
	if an := s.anims[w.anim]; an != nil {
		// A run and an idle go round; a jump and a hit play once and stay at their last pose.
		t, dur := w.animTime*s.animRate(w), float64(an.Duration())
		if w.anim == "idle" || w.anim == "run" {
			t = math.Mod(t, dur)
		} else {
			t = math.Min(t, dur)
		}
		s.pose.Clear()
		an.Sample(float32(t), s.pose)
		s.character.SetPose(s.pose)
	}
	visible := w.hurt == 0 || int(w.hurt*14)%2 == 0
	for _, it := range s.charItems {
		it.SetVisible(visible)
	}

	s.spin = math.Mod(s.spin+2.4*dt, 2*math.Pi)
	for i, it := range s.coins {
		if i >= len(w.coins) {
			break
		}
		c := w.coins[i]
		it.SetVisible(!c.taken)
		if !c.taken {
			bobY := 0.08 * math.Sin(2*s.spin+float64(i))
			it.SetTransform(vecmath.NewMat4Translation(c.at.Add(vecmath.Vec3{Y: bobY})).Mul(vecmath.NewMat4RotationY(s.spin + float64(i))))
		}
	}

	// The camera trails the character's middle a little and looks down on it from behind.
	want := w.pos.Add(vecmath.Vec3{Y: cameraHeight})
	if s.eye == (vecmath.Vec3{}) {
		s.eye = want
	}
	s.eye = s.eye.Lerp(want, math.Min(1, 12*dt))
	s.camera.SetTarget(s.eye)
	s.camera.SetDistance(cameraDistance)
	s.camera.SetOrientation(vecmath.NewQuatAxisAngle(vecmath.Vec3{Y: 1}, s.yaw).Mul(vecmath.NewQuatAxisAngle(vecmath.Vec3{X: 1}, -s.pitch)))
}

// animRate is how fast the animation of the world's character plays: a run goes faster the faster the character moves.
func (s *stage) animRate(w *world) float64 {
	if w.anim == "run" {
		return math.Max(0.6, math.Hypot(w.vel.X, w.vel.Z)/runSpeed)
	}
	return 1
}
