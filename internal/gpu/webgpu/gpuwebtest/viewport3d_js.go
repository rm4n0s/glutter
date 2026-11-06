//go:build js && wasm

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"time"

	"github.com/rm4n0s/glutter/gpu"
	igpu "github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/webgpu"
	"github.com/rm4n0s/glutter/internal/gpuhost"
	"github.com/rm4n0s/glutter/internal/scenetest"
)

// viewport3DResult is the result of the mode viewport3d.
type viewport3DResult struct {
	// Error says what failed before the scenes could be drawn.
	Error string `json:"error"`
	// Renderer is the adapter the scenes were drawn on.
	Renderer string `json:"renderer"`
	// Scenes are the pictures, one for each case of internal/scenetest, in the order of scenetest.Cases.
	Scenes     []viewport3DScene `json:"scenes"`
	Validation []string          `json:"validation"`
}

// viewport3DScene is one picture: the name of the case, a PNG of it in base64, or what failed.
type viewport3DScene struct {
	Name  string `json:"name"`
	PNG   string `json:"png"`
	Error string `json:"error"`
}

// viewport3D draws the scenes of internal/scenetest (the 3D view of step 30.9) through package gpu on the browser's WebGPU,
// and gives the pictures to the test, which compares them with the goldens.
func viewport3D() viewport3DResult {
	var res viewport3DResult
	eval(patches)
	newCanvas(0, 64, 64)
	prov, err := webgpu.Acquire(context.Background(), webgpu.Options{
		Timeout:       time.Minute,
		OnDeviceReady: func(igpu.SurfaceHandle) {},
	})
	if err != nil {
		res.Error = "Acquire: " + describe(err)
		return res
	}
	dev, err := prov.Open(handleOf(0))
	if err != nil {
		res.Error = "Open: " + describe(err)
		return res
	}
	var tb cleanups
	app := gpuhost.NewBinding(&tb, dev)
	defer func() {
		for i := len(tb) - 1; i >= 0; i-- {
			tb[i]()
		}
	}()
	d, err := gpu.DeviceOfBinding(app.Binding)
	if err != nil {
		res.Error = "DeviceOfBinding: " + describe(err)
		return res
	}
	if named, ok := dev.(interface{ Description() string }); ok {
		res.Renderer = named.Description()
	}
	wait := func(done func() bool) { pump(app, done) }
	for _, c := range scenetest.Cases() {
		scene := viewport3DScene{Name: c.Name}
		img, err := scenetest.Render(d, wait, c)
		if err != nil {
			scene.Error = describe(err)
		} else {
			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil {
				scene.Error = err.Error()
			} else {
				scene.PNG = base64.StdEncoding.EncodeToString(buf.Bytes())
			}
		}
		res.Scenes = append(res.Scenes, scene)
	}
	res.Validation = prov.ValidationErrors()
	return res
}
