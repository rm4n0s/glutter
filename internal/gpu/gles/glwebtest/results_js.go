//go:build js && wasm

package main

// diffStats is how far two images are from each other (gputest.Diff).
type diffStats struct {
	Pixels int `json:"pixels"`
	Over8  int `json:"over8"`
	Over32 int `json:"over32"`
	Over96 int `json:"over96"`
	Max    int `json:"max"`
}

// sceneResult is what the program saw drawing one proof scene.
type sceneResult struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	// Frame is the frame the renderer drew into the canvas and read back, against what soft
	// draws; Snapshot is the snapshot (an offscreen framebuffer) against soft; SnapshotVsFrame
	// the snapshot against the frame; Repeat the frame drawn a second time against the first.
	Frame           diffStats `json:"frame"`
	Snapshot        diffStats `json:"snapshot"`
	SnapshotVsFrame diffStats `json:"snapshotVsFrame"`
	Repeat          diffStats `json:"repeat"`
	// Skipped is how many commands the renderer left out, and Unsupported says why.
	Skipped     int      `json:"skipped"`
	Unsupported []string `json:"unsupported"`
	Error       string   `json:"error"`
}

// parityResult is the result of the mode parity.
type parityResult struct {
	Scenes []sceneResult `json:"scenes"`
	// Version and Renderer are what the first context reported.
	Version  string `json:"version"`
	Renderer string `json:"renderer"`
}

// tagResult is an error as the program saw it.
type tagResult struct {
	Tag   string `json:"tag"`
	Route string `json:"route"`
}

// unavailableResult is the result of the mode unavailable.
type unavailableResult struct {
	// Taken is the error of a canvas that has a 2D context already.
	Taken tagResult `json:"taken"`
	// NoCanvas is the error of a view that has no canvas, and WrongKind that of a handle that
	// is not a canvas's.
	NoCanvas  tagResult `json:"noCanvas"`
	WrongKind tagResult `json:"wrongKind"`
	// Fresh is whether a new canvas, which has had no context, gave one.
	Fresh bool `json:"fresh"`
	// Second is the error of a second view on a context that has a canvas already.
	Second tagResult `json:"second"`
}

// lossResult is the result of the mode loss.
type lossResult struct {
	// First is whether the first frame drew. While the context was lost: the tag of the
	// read-back's error, whether Render returned no error, and the tag of the snapshot's error.
	// Restored is how many times the browser's restoration reached the renderer's callback,
	// and after it the errors of the read-back and of Render, and how far the frames after the
	// restoration are from the first.
	First            bool      `json:"first"`
	LostCapture      string    `json:"lostCapture"`
	LostRenderOK     bool      `json:"lostRenderOK"`
	LostSnapshot     string    `json:"lostSnapshot"`
	Restored         int       `json:"restored"`
	AfterError       string    `json:"afterError"`
	AfterVsFirst     diffStats `json:"afterVsFirst"`
	SnapshotVsFirst  diffStats `json:"snapshotVsFirst"`
	AfterRenderError string    `json:"afterRenderError"`
}

// chooseResult is the result of the modes choose and restore.
type chooseResult struct {
	// Error is the tag of what RunApp returned, empty while the app runs.
	Error string `json:"error"`
	// Canvases is how many canvases of a view the page has, Connected whether the one that
	// is the view's is in the page, WebGL2 and Context2D which kind of context it has.
	Canvases  int  `json:"canvases"`
	WebGL2    bool `json:"webgl2"`
	Context2D bool `json:"context2d"`
	// The animation frames asked for since the page began, before the context was lost, while
	// it was lost, and after it was restored (restore only).
	FramesBefore   int `json:"framesBefore"`
	FramesWhileOff int `json:"framesWhileOff"`
	FramesAfter    int `json:"framesAfter"`
}
