package webtest

import (
	"encoding/json"
	"html"
	"strconv"
	"strings"
	"time"
)

// FramePath is where Shot serves the page that holds the app in a frame.
//
// Flutter: no counterpart
const FramePath = "/__glutter/frame.html"

// Resize is a change of the size of the frame that the app is in, some time after the app has
// put its canvas into the page, which is when it starts drawing. The app sees it as the browser
// window getting that size.
//
// Flutter: no counterpart
type Resize struct {
	// After is the time after the app's canvas is in the page.
	After time.Duration
	// Width and Height are the new size in CSS pixels.
	Width, Height int
}

// FramePage makes the page that shows the app (src, a URL on the same server) in a frame of
// exactly width x height CSS pixels, and changes the size of the frame at the times of the
// resizes. A browser window cannot be made an exact size (Chromium and Firefox keep a minimum
// width and show toolbars even when headless), but the app only sees the viewport of its page,
// and a frame gives it the one that is asked for.
//
// Flutter: no counterpart
func FramePage(src string, width, height int, resizes []Resize) []byte {
	var script strings.Builder
	for _, r := range resizes {
		script.WriteString("whenCanvas(function(){setTimeout(function(){f.style.width=" + strconv.Quote(strconv.Itoa(r.Width)+"px") + ";f.style.height=" +
			strconv.Quote(strconv.Itoa(r.Height)+"px") + ";}," + strconv.Itoa(int(r.After.Milliseconds())) + ");});\n")
	}
	srcJSON, _ := json.Marshal(src) // a string always marshals
	safe := strings.NewReplacer("<", "\\u003c", ">", "\\u003e", "&", "\\u0026").Replace(string(srcJSON))
	return []byte("<!doctype html>\n<html><head><meta charset=\"utf-8\"><title>" + html.EscapeString("glutter frame") + "</title>\n" +
		"<style>html,body{margin:0;background:#808080}iframe{border:0;display:block;width:" + strconv.Itoa(width) + "px;height:" +
		strconv.Itoa(height) + "px}</style></head>\n<body><iframe id=\"app\"></iframe>\n<script>\nvar f=document.getElementById(\"app\");\nf.src=" +
		safe + ";\n" + whenCanvasJS + script.String() + "</script></body></html>\n")
}

// whenCanvasJS defines whenCanvas(fn), which calls fn once the app's page has a canvas of a view.
const whenCanvasJS = "function whenCanvas(fn){var t=setInterval(function(){try{if(f.contentDocument&&" +
	"f.contentDocument.querySelector(\"canvas[data-glutter-view]\")){clearInterval(t);fn();}}catch(e){}},20);}\n"
