//go:build js && wasm

package web

import "syscall/js"

// ViewAttribute is the attribute that names the view a canvas shows. The renderer that
// draws into the canvas (internal/gpu/canvas2d) finds it with it, through the Window of
// the gpu.SurfaceHandle.
//
// Flutter: lib/web_ui/lib/src/engine/view_embedder/global_html_attributes.dart GlobalHtmlAttributes
const ViewAttribute = "data-glutter-view"

// global is globalThis.
func global() js.Value { return js.Global() }

// isObject says whether v is an object or a function: not undefined, null or a primitive.
func isObject(v js.Value) bool {
	return v.Type() == js.TypeObject || v.Type() == js.TypeFunction
}

// try runs f and reports whether it finished without a JavaScript exception, which
// syscall/js turns into a panic.
func try(f func()) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()
	f()
	return true
}

// setStyle sets a CSS property of an element.
func setStyle(el js.Value, name, value string) {
	el.Get("style").Call("setProperty", name, value)
}

// matchMedia makes the MediaQueryList of a media query.
func matchMedia(query string) js.Value { return global().Call("matchMedia", query) }

// number reads v as a float64, and def when v is not a number.
func number(v js.Value, def float64) float64 {
	if v.Type() == js.TypeNumber {
		return v.Float()
	}
	return def
}
