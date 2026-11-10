//go:build ios

package ios

import (
	"github.com/ebitengine/purego/objc"
	"github.com/rm4n0s/glutter/internal/platform/darwin/objcx"
)

// uiPasteboard is pasteBackend over UIPasteboard's general pasteboard (SPEC §30.6). UIPasteboard may be used from any thread, which the
// reads need: they wait for the paste prompt of iOS 16 and later when another app copied what they read. Each call has its own
// autorelease pool, since the goroutine that makes it has none.
//
// Flutter: shell/platform/darwin/ios/framework/Source/FlutterPlatformPlugin.mm getClipboardData, setClipboardData, clipboardHasStrings
type uiPasteboard struct{}

var _ pasteBackend = uiPasteboard{}

func (uiPasteboard) board() objc.ID { return class("UIPasteboard").Send(sel("generalPasteboard")) }

// valueFor is the object of the pasteboard for the type, 0 for none.
func (b uiPasteboard) valueFor(selector, typ string) objc.ID {
	t := objcx.NewString(typ)
	defer objcx.Release(t)
	return send[objc.ID](b.board(), selector, t)
}

// HasStrings says that the pasteboard has text. It does not ask the user.
func (b uiPasteboard) HasStrings() (has bool) {
	objcx.WithPool(func() { has = send[bool](b.board(), "hasStrings") })
	return has
}

// String reads the text; iOS asks the user first when it came from another app, and the call returns when the user has answered.
func (b uiPasteboard) String() (text string, ok bool) {
	objcx.WithPool(func() {
		if s := send[objc.ID](b.board(), "string"); s != 0 {
			text, ok = objcx.String(s), true
		}
	})
	return text, ok
}

// HTML reads the HTML: the pasteboard has it as a string or as data.
func (b uiPasteboard) HTML() (html string, ok bool) {
	objcx.WithPool(func() {
		v := b.valueFor("valueForPasteboardType:", typeHTML)
		switch {
		case v == 0:
		case send[bool](v, "isKindOfClass:", class("NSData")):
			html, ok = string(objcx.DataBytes(v)), true
		default:
			html, ok = objcx.String(v), true
		}
	})
	return html, ok
}

// PNG reads the image as PNG: the data if the pasteboard has PNG, else the image converted.
func (b uiPasteboard) PNG() (png []byte) {
	objcx.WithPool(func() {
		if data := b.valueFor("dataForPasteboardType:", typePNG); data != 0 {
			png = objcx.DataBytes(data)
			return
		}
		if img := send[objc.ID](b.board(), "image"); img != 0 {
			png = objcx.DataBytes(send[objc.ID](img, "pngData"))
		}
	})
	return png
}

// URLs reads the URLs of the items.
func (b uiPasteboard) URLs() (urls []string) {
	objcx.WithPool(func() {
		list := send[objc.ID](b.board(), "URLs")
		if list == 0 {
			return
		}
		for i := 0; i < objcx.ArrayCount(list); i++ {
			urls = append(urls, objcx.URLString(objcx.ArrayAt(list, i)))
		}
	})
	return urls
}

// Write replaces the items of the pasteboard.
func (b uiPasteboard) Write(items []pasteItem) bool {
	objcx.WithPool(func() {
		dicts := make([]objc.ID, 0, len(items))
		var owned []objc.ID
		for _, item := range items {
			var keys, vals []objc.ID
			for _, v := range item {
				k := objcx.NewString(v.UTI)
				var val objc.ID
				switch v.Kind {
				case pasteData:
					val = objcx.NewData(v.Data)
				case pasteURL:
					if val = objcx.NewURL(v.Text); val == 0 {
						objcx.Release(k)
						continue
					}
				default:
					val = objcx.NewString(v.Text)
				}
				keys, vals = append(keys, k), append(vals, val)
				owned = append(owned, k, val)
			}
			d := objcx.NewDictionary(keys, vals)
			dicts = append(dicts, d)
			owned = append(owned, d)
		}
		arr := objcx.NewArray(dicts...)
		b.board().Send(sel("setItems:"), arr)
		objcx.Release(arr)
		for _, o := range owned {
			objcx.Release(o)
		}
	})
	return true
}
