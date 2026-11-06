//go:build js && wasm

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"sync"
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
	Scenes []viewport3DScene `json:"scenes"`
	// Picks are the picks of internal/scenetest (SPEC §27.8), each with what was wrong with it, or nothing.
	Picks      []viewport3DPick `json:"picks"`
	Validation []string         `json:"validation"`
}

// viewport3DPick is the result of one pick check: its name, what was wrong ("" for nothing) and the error that kept it from running.
type viewport3DPick struct {
	Name    string `json:"name"`
	Problem string `json:"problem"`
	Error   string `json:"error"`
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
	// The browser calls back on a goroutine of its own: what it posts (a finished read) runs on the goroutine of the test, in wait,
	// as in an app, where the binding runs it on the UI thread. The pick checks release their textures in such a callback.
	var postMu sync.Mutex
	var posted []func()
	post := func(fn func()) {
		postMu.Lock()
		posted = append(posted, fn)
		postMu.Unlock()
	}
	runPosted := func() {
		postMu.Lock()
		fns := posted
		posted = nil
		postMu.Unlock()
		for _, fn := range fns {
			fn()
		}
	}
	prov, err := webgpu.Acquire(context.Background(), webgpu.Options{
		Timeout:       time.Minute,
		OnDeviceReady: func(igpu.SurfaceHandle) {},
		Post:          post,
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
	wait := func(done func() bool) {
		for i := 0; i < 600; i++ {
			runPosted()
			app.RunPosted()
			if done() {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
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
	for _, c := range scenetest.PickChecks() {
		pick := viewport3DPick{Name: c.Name}
		problem, err := scenetest.RunPickCheck(d, wait, c)
		if err != nil {
			pick.Error = describe(err)
		}
		pick.Problem = problem
		res.Picks = append(res.Picks, pick)
	}
	res.Validation = prov.ValidationErrors()
	return res
}
