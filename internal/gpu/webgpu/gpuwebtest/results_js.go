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
	// Frame is the frame the renderer drew and read back against what soft draws; Snapshot the
	// renderer's snapshot against soft (it is soft's: the two are the same pixels); Repeat the
	// frame drawn a second time against the first.
	Frame    diffStats `json:"frame"`
	Snapshot diffStats `json:"snapshot"`
	Repeat   diffStats `json:"repeat"`
	// Skipped is how many commands the renderer left out, and Unsupported says why.
	Skipped     int      `json:"skipped"`
	Unsupported []string `json:"unsupported"`
	Error       string   `json:"error"`
}

// parityResult is the result of the mode parity.
type parityResult struct {
	Scenes []sceneResult `json:"scenes"`
	// Renderer names the renderer and its adapter; Validation is every error the browser
	// reported for a call that broke a WebGPU rule.
	Renderer   string   `json:"renderer"`
	Validation []string `json:"validation"`
}

// tagResult is an error as the program saw it.
type tagResult struct {
	Tag   string `json:"tag"`
	Route string `json:"route"`
}

// unavailableResult is the result of the mode unavailable.
type unavailableResult struct {
	// NoGPU, NoAdapter, NoDevice, BadShader and Timeout are the errors of Acquire when the
	// browser has no navigator.gpu, answers null to requestAdapter, rejects requestDevice, rejects
	// a WGSL module, and does not answer in time.
	NoGPU     tagResult `json:"noGPU"`
	NoAdapter tagResult `json:"noAdapter"`
	NoDevice  tagResult `json:"noDevice"`
	BadShader tagResult `json:"badShader"`
	Timeout   tagResult `json:"timeout"`
	// OpenOfNil is the error of Open on no provider.
	OpenOfNil tagResult `json:"openOfNil"`
	// Taken is the error of a canvas that has a 2D context already, NoCanvas that of a view
	// without a canvas, WrongKind that of a handle that is not a canvas's.
	Taken     tagResult `json:"taken"`
	NoCanvas  tagResult `json:"noCanvas"`
	WrongKind tagResult `json:"wrongKind"`
	// Fresh is whether a new canvas gave a swapchain.
	Fresh bool `json:"fresh"`
	// Validation is every error the browser reported.
	Validation []string `json:"validation"`
}

// lossResult is the result of the mode loss.
type lossResult struct {
	// First is whether the first frame drew.
	First bool `json:"first"`
	// DuringLoss is the tag of what Draw returned right after the device was destroyed, and Pending
	// whether Draw drew nothing and returned no error; Ready is how many times the owner was told
	// that the new device had come (through Options.OnDeviceReady).
	DuringLoss string `json:"duringLoss"`
	Ready      int    `json:"ready"`
	// DrewDuringLoss is whether that frame delivered a capture, which it must not: nothing was drawn.
	DrewDuringLoss bool `json:"drewDuringLoss"`
	// AfterError is the error of the frame drawn on the new device, and AfterVsFirst how far
	// that frame is from the first, which must be none.
	AfterError   string    `json:"afterError"`
	AfterVsFirst diffStats `json:"afterVsFirst"`
	// SecondLoss is the tag that Draw returned for a loss less than ten seconds after the first.
	SecondLoss string   `json:"secondLoss"`
	Validation []string `json:"validation"`
}

// chooseResult is the result of the mode choose.
type chooseResult struct {
	// Error is the tag of what RunApp returned, empty while the app runs.
	Error string `json:"error"`
	// Canvases is how many canvases of a view the page has; WebGPU, WebGL2 and Context2D say which
	// kind of context the first has (a canvas answers null to the other kinds).
	Canvases  int  `json:"canvases"`
	WebGPU    bool `json:"webgpu"`
	WebGL2    bool `json:"webgl2"`
	Context2D bool `json:"context2d"`
}
