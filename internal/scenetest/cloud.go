package scenetest

import (
	"bytes"
	"context"
	"fmt"
	"math"

	"github.com/rm4n0s/glutter/gpu"
	"github.com/rm4n0s/glutter/model/ply"
	"github.com/rm4n0s/glutter/vecmath"
	"github.com/rm4n0s/glutter/viewport3d"
)

// cloudPLY is the text of a PLY file of n points on a sphere of radius 1 (the golden angle spiral), each with a color that follows
// its place: red with x, green with y, blue with z.
func cloudPLY(n int) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "ply\nformat ascii 1.0\nelement vertex %d\nproperty float x\nproperty float y\nproperty float z\nproperty uchar red\nproperty uchar green\nproperty uchar blue\nend_header\n", n)
	golden := math.Pi * (3 - math.Sqrt(5))
	for i := 0; i < n; i++ {
		y := 1 - 2*(float64(i)+0.5)/float64(n)
		r := math.Sqrt(1 - y*y)
		x, z := r*math.Cos(golden*float64(i)), r*math.Sin(golden*float64(i))
		fmt.Fprintf(&b, "%.6f %.6f %.6f %d %d %d\n", x, y, z, int((x+1)*127.5), int((y+1)*127.5), int((z+1)*127.5))
	}
	return b.Bytes()
}

// cloudCases are the scenes of the point clouds (step 31.10): a cloud that a PLY file gives, with its colors and with one color, as
// round and as square points.
func cloudCases() []Case {
	return []Case{{Name: "point-cloud", Width: 480, Height: 240, Build: func(d *gpu.Device) (*viewport3d.Scene, *viewport3d.Camera, func()) {
		m, err := ply.Decode(context.Background(), bytes.NewReader(cloudPLY(3000)), ply.Options{Name: "sphere"})
		if err != nil {
			panic(err)
		}
		u, err := viewport3d.Upload(context.Background(), d, m, viewport3d.UploadOpts{})
		if err != nil {
			panic(err)
		}
		s := viewport3d.NewScene()
		left := u.Parts[0].Item
		left.SetTransform(vecmath.NewMat4Translation(vecmath.Vec3{X: -1.3}))
		right := viewport3d.NewItem(left.Mesh(), viewport3d.NewPointMaterial(viewport3d.PointOpts{Color: color32(0xFFE08030), Size: 3.5, Shape: viewport3d.PointShapeSquare}),
			viewport3d.ItemOpts{Transform: translate(1.3, 0, 0)})
		s.Add(left, right)
		cam := frontCamera(s, 480, 240)
		cam.Orbit(0.5, 0.25)
		return s, cam, u.Release
	}}}
}
