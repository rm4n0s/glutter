//go:build (linux && glutter_display) || (darwin && !ios && glutter_display) || (windows && glutter_wine)

package viewport3d_test

import (
	"math"
	"testing"
	"time"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/internal/scenetest"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

func halfToFloat(h uint16) float64 {
	sign := 1.0
	if h&0x8000 != 0 {
		sign = -1
	}
	e := int(h>>10) & 31
	m := float64(h & 1023)
	switch e {
	case 0:
		return sign * m / 1024 * math.Pow(2, -14)
	case 31:
		if m == 0 {
			return sign * math.Inf(1)
		}
		return math.NaN()
	}
	return sign * (1 + m/1024) * math.Pow(2, float64(e-15))
}

// cpuSky is a port of sky() of env.wgsl for the direction of the texel (x, y) of a face of the given size.
func cpuSky(face, x, y, size int) [3]float64 {
	u := 2*(float64(x)+0.5)/float64(size) - 1
	v := 2*(float64(y)+0.5)/float64(size) - 1
	var d vecmath.Vec3
	switch face {
	case 0:
		d = vecmath.Vec3{X: 1, Y: -v, Z: -u}
	case 1:
		d = vecmath.Vec3{X: -1, Y: -v, Z: u}
	case 2:
		d = vecmath.Vec3{X: u, Y: 1, Z: v}
	case 3:
		d = vecmath.Vec3{X: u, Y: -1, Z: -v}
	case 4:
		d = vecmath.Vec3{X: u, Y: -v, Z: 1}
	default:
		d = vecmath.Vec3{X: -u, Y: -v, Z: -1}
	}
	d = d.Normalize()
	smooth := func(a, b, x float64) float64 {
		t := math.Min(math.Max((x-a)/(b-a), 0), 1)
		return t * t * (3 - 2*t)
	}
	mix := func(a, b, t float64) float64 { return a + (b-a)*t }
	up := d.Y
	horizon, zenith, floor := [3]float64{0.62, 0.64, 0.67}, [3]float64{0.75, 0.80, 0.90}, [3]float64{0.10, 0.10, 0.11}
	key := math.Pow(math.Max(d.Dot(vecmath.Vec3{X: 0.55, Y: 0.75, Z: 0.45}.Normalize()), 0), 24)
	fill := math.Pow(math.Max(d.Dot(vecmath.Vec3{X: -0.7, Y: 0.35, Z: -0.5}.Normalize()), 0), 10)
	keyC, fillC := [3]float64{1, 0.97, 0.92}, [3]float64{0.75, 0.85, 1}
	var out [3]float64
	for i := 0; i < 3; i++ {
		c := mix(horizon[i], zenith[i], smooth(0, 1, up))
		c = mix(c, floor[i], 1-smooth(-0.45, 0, up))
		out[i] = c + keyC[i]*key*6 + fillC[i]*fill*0.8
	}
	return out
}

// readFace reads one face of one level of a cube of RGBA16Float as floats.
func readFace(t *testing.T, d *gpu.Device, wait func(func() bool), cube *gpu.Texture, level, face int) []float64 {
	t.Helper()
	size := cube.Size().Width >> level
	if size < 1 {
		size = 1
	}
	rowAlign := d.Limits().BytesPerRowAlignment
	row := (size*8 + rowAlign - 1) / rowAlign * rowAlign
	buf, err := d.NewBuffer(gpu.BufferDesc{Label: "env readback", Size: row * size, Usage: gpu.BufferUsageCopySrc | gpu.BufferUsageCopyDst})
	if err != nil {
		t.Fatal(err)
	}
	defer buf.Release()
	enc := d.NewCommandEncoder()
	enc.CopyTextureToBuffer(gpu.TextureCopy{Texture: cube, MipLevel: level, Origin: gpu.Origin3D{Z: face}}, gpu.BufferCopy{Buffer: buf, Layout: gpu.DataLayout{BytesPerRow: row}},
		gpu.Extent3D{Width: size, Height: size, DepthOrArrayLayers: 1})
	if err := d.Queue().Submit(enc.Finish()); err != nil {
		t.Fatal(err)
	}
	var data []byte
	done := false
	buf.Read(0, row*size, func(b []byte, err error) {
		if err != nil {
			t.Error(err)
		}
		data, done = b, true
	})
	wait(func() bool { return done })
	if !done {
		t.Fatal("the read did not finish")
	}
	out := make([]float64, 0, size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			for c := 0; c < 4; c++ {
				i := y*row + x*8 + c*2
				out = append(out, halfToFloat(uint16(data[i])|uint16(data[i+1])<<8))
			}
		}
	}
	return out
}

// TestTheDefaultEnvironmentIsTheSky reads back the cubes that the compute passes make for the default environment and compares them
// with a port of the sky to Go. It found that Wine's d3dcompiler_47 gives a wrong float for a thread id turned into one (step 31.9).
func TestTheDefaultEnvironmentIsTheSky(t *testing.T) {
	app, d := deviceApp(t)
	wait := waitFor(app)
	scene := viewport3d.NewScene()
	scene.Add(viewport3d.NewItem(scenetest.Sphere(0.5, 16, 8), viewport3d.NewPBRMaterial()))
	cam := viewport3d.NewCamera()
	r := viewport3d.NewRenderer(scene, cam)
	r.Init(d)
	defer r.Dispose()
	size := gpu.Extent3D{Width: 64, Height: 64, DepthOrArrayLayers: 1}
	color, err := d.NewTexture(gpu.TextureDesc{Label: "color", Size: size, Format: gpu.TextureFormatRGBA8Unorm, Usage: gpu.TextureUsageRenderAttachment | gpu.TextureUsageCopySrc})
	if err != nil {
		t.Fatal(err)
	}
	defer color.Release()
	depth, err := d.NewTexture(gpu.TextureDesc{Label: "depth", Size: size, Format: gpu.TextureFormatDepth32Float, Usage: gpu.TextureUsageRenderAttachment})
	if err != nil {
		t.Fatal(err)
	}
	defer depth.Release()
	enc := d.NewCommandEncoder()
	frame := &gpu.ViewFrame{Device: d, Encoder: enc, Color: color, Depth: depth, Width: 64, Height: 64, DevicePixelRatio: 1, Time: time.Second}
	r.Draw(frame)
	if err := frame.Submit(); err != nil {
		t.Fatal(err)
	}
	sky, spec, irr := r.EnvironmentCubes()
	if sky == nil || spec == nil || irr == nil {
		t.Fatalf("the cubes are missing: %v %v %v", sky, spec, irr)
	}
	// mean is the mean of the three color channels of a face, and fails the test for a NaN or an infinity.
	mean := func(name string, v []float64) float64 {
		sum := 0.0
		for i, x := range v {
			if i%4 == 3 {
				continue
			}
			if math.IsNaN(x) || math.IsInf(x, 0) {
				t.Fatalf("%s has %v at the value %d", name, x, i)
			}
			sum += x
		}
		return sum / float64(len(v)*3/4)
	}
	for face := 0; face < 6; face++ {
		got := readFace(t, d, wait, sky, 0, face)
		worst, cpuMean := 0.0, 0.0
		for y := 0; y < 128; y++ {
			for x := 0; x < 128; x++ {
				want := cpuSky(face, x, y, 128)
				for c := 0; c < 3; c++ {
					g := got[(y*128+x)*4+c]
					worst = math.Max(worst, math.Abs(g-want[c])/(math.Abs(want[c])+0.05))
					cpuMean += want[c] / (128 * 128 * 3)
				}
			}
		}
		if worst > 0.02 {
			t.Errorf("the sky of face %d differs from the CPU's by %.3f (relative)", face, worst)
		}
		// The mirror level of the specular cube is the sky. The coarser levels and the irradiance blur it, so a face may be brighter or
		// darker than the sky was there; a compute pass that writes none of a level, or only a part, shows as a mean far from the sky's.
		name := "face " + string(rune('0'+face))
		if m := mean("specular level 0 "+name, readFace(t, d, wait, spec, 0, face)); math.Abs(m-cpuMean) > 0.05*cpuMean+0.01 {
			t.Errorf("the mean of level 0 of the specular cube on %s is %.3f, the sky's %.3f", name, m, cpuMean)
		}
		for _, l := range []int{2, 4} {
			if m := mean("specular level "+string(rune('0'+l))+" "+name, readFace(t, d, wait, spec, l, face)); m < 0.5*cpuMean || m > 3*cpuMean+0.1 {
				t.Errorf("the mean of level %d of the specular cube on %s is %.3f, the sky's %.3f", l, name, m, cpuMean)
			}
		}
		if m := mean("irradiance "+name, readFace(t, d, wait, irr, 0, face)); m < 0.5*cpuMean || m > 3*cpuMean+0.1 {
			t.Errorf("the mean of the irradiance on %s is %.3f, the sky's %.3f", name, m, cpuMean)
		}
	}
}
