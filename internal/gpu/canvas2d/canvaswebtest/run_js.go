//go:build js && wasm

package main

import (
	"encoding/json"
	"image"
	"image/color"
	"log/slog"
	"os"
	"strconv"
	"syscall/js"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/gpu"
	"github.com/rm4n0s/glutter/internal/gpu/canvas2d"
	"github.com/rm4n0s/glutter/internal/issues"
	"github.com/rm4n0s/glutter/internal/platform/web"
	"github.com/rm4n0s/glutter/ui"
)

// viewAttribute names the view a canvas shows (internal/platform/web ViewAttribute).
const viewAttribute = "data-glutter-view"

func main() {
	switch mode := param("mode"); mode {
	case "issue":
		postResult(issue(param("issue")))
	default:
		err := errors.New(tagCanvasWebTestModeUnknown, "the query parameter mode is not issue", "mode", mode)
		slog.Error("canvaswebtest", "err", err)
		os.Exit(1)
	}
	// The result is posted by an asynchronous call: the program stays until the test has it, so
	// that its ending is not taken for the program's failure.
	select {}
}

// param reads a query parameter of the page.
func param(name string) string {
	q := js.Global().Get("URLSearchParams").New(js.Global().Get("location").Get("search"))
	if v := q.Call("get", name); v.Type() == js.TypeString {
		return v.String()
	}
	return ""
}

// postResult posts the result to the test server; the answer is not waited for.
func postResult(data any) {
	body, err := json.Marshal(map[string]any{"kind": "canvasweb", "data": data})
	if err != nil {
		panic(err)
	}
	init := map[string]any{"method": "POST", "headers": map[string]any{"Content-Type": "application/json"}, "body": string(body)}
	js.Global().Call("fetch", web.ResultsPath, init)
}

// issueResult is the result of the mode issue.
type issueResult struct {
	// Lines are the lines of the issue's report (internal/issues).
	Lines []string `json:"lines"`
	// Shown lists the pixels of the canvas that differ from the frames the renderer drew.
	Shown []string `json:"shown"`
	Error string   `json:"error"`
}

// issue runs the reproduction of a reported issue with the canvas2d renderer, on a canvas of the
// size the reproduction asks for, and reads a few pixels of each frame back from the canvas. The
// reproduction of guard-allocs draws nothing: it runs in the page's Go, with no canvas.
func issue(name string) issueResult {
	var res issueResult
	if name == issues.GuardAllocsName {
		res.Lines = issues.RunGuardAllocs().Lines()
		return res
	}
	if name != issues.SoftMipmapsName {
		res.Error = "the query parameter issue is not " + issues.SoftMipmapsName + " or " + issues.GuardAllocsName + ": " + name
		return res
	}
	doc := js.Global().Get("document")
	canvas := doc.Call("createElement", "canvas")
	canvas.Call("setAttribute", viewAttribute, "0")
	canvas.Set("width", issues.SoftMipmapsWidth)
	canvas.Set("height", issues.SoftMipmapsHeight)
	doc.Get("body").Call("appendChild", canvas)
	defer canvas.Call("remove")
	r := canvas2d.New(canvas2d.Options{Background: ui.NewColor(0xFFFFFFFF)})
	defer func() { _ = r.Close() }()
	s, err := r.Attach(gpu.SurfaceHandle{Kind: gpu.SurfaceKindWeb, Window: 0})
	if err != nil {
		res.Error = "Attach: " + err.Error()
		return res
	}
	defer s.Destroy()
	frame := 0
	rep, err := issues.RunSoftMipmaps(func(scene *ui.Scene, width, height int) (*image.RGBA, issues.SoftMipmapsStats, error) {
		img, err := r.RenderCapture(s, scene, ui.Size{Width: float64(width), Height: float64(height)}, 1)
		if err != nil {
			return nil, issues.SoftMipmapsStats{}, err
		}
		// The canvas has the renderer's context (2D, without alpha): what the page shows.
		ctx := canvas.Call("getContext", "2d", map[string]any{"alpha": false})
		for _, pt := range [][2]int{{4, 4}, {256, 192}, {600, 400}} {
			data := ctx.Call("getImageData", pt[0], pt[1], 1, 1).Get("data")
			got := color.RGBA{R: uint8(data.Index(0).Int()), G: uint8(data.Index(1).Int()), B: uint8(data.Index(2).Int()), A: uint8(data.Index(3).Int())}
			if want := img.RGBAAt(pt[0], pt[1]); got != want {
				res.Shown = append(res.Shown, "frame "+strconv.Itoa(frame)+": pixel "+strconv.Itoa(pt[0])+","+strconv.Itoa(pt[1])+" of the canvas is "+rgba(got)+", the frame has "+rgba(want))
			}
		}
		frame++
		st := r.MipmapStats()
		return img, issues.SoftMipmapsStats{Chains: st.Chains, Built: st.Built}, nil
	})
	if err != nil {
		res.Error = "the frames: " + err.Error()
		if appErr, ok := errors.FromError(err); ok {
			res.Error += " [" + appErr.Tag + " " + appErr.Route() + "]"
		}
		return res
	}
	res.Lines = rep.Lines()
	return res
}

func rgba(c color.RGBA) string {
	return strconv.Itoa(int(c.R)) + "," + strconv.Itoa(int(c.G)) + "," + strconv.Itoa(int(c.B)) + "," + strconv.Itoa(int(c.A))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagCanvasWebTestModeUnknown = "CanvasWebTestModeUnknown"
)
