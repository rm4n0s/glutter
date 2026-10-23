package webtool

import (
	"bytes"
	_ "embed" // the loader
	"encoding/json"
	"html"
	"strings"

	"github.com/rm4n0s/errors"
)

//go:embed loader.js
var loaderJS string

// LoaderJS is the JavaScript of the loader that index.html contains.
//
// Flutter: no counterpart (flutter_bootstrap.js)
func LoaderJS() string { return loaderJS }

// Page describes the index.html of an app.
//
// Flutter: no counterpart
type Page struct {
	// Title is the title of the page until the app sets its own.
	Title string
	// HostElement is the CSS selector for globalThis.glutterConfig.hostElement; empty
	// leaves the view to fill the page (SPEC §23.2).
	HostElement string
	// Body is HTML put into the body before the loader: the host element of a page that
	// embeds the app in an element of its own (SPEC §23.2).
	Body string
}

// IndexHTML makes the index.html of the app: the viewport meta tag, globalThis.glutterConfig,
// wasm_exec.js and the loader (SPEC §23.1). A message inside <noscript> is for a browser
// without JavaScript.
//
// Flutter: no counterpart (flutter build web's index.html)
func IndexHTML(p Page) ([]byte, error) {
	cfg := map[string]string{}
	if p.HostElement != "" {
		cfg["hostElement"] = p.HostElement
	}
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, errors.NewErr(tagWebPageFailed, err)
	}
	// The JSON sits inside a script element: nothing in it may end the element.
	safe := strings.NewReplacer("<", "\\u003c", ">", "\\u003e", "&", "\\u0026").Replace(string(cfgJSON))
	title := p.Title
	if title == "" {
		title = "Glutter"
	}
	var b bytes.Buffer
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no\">\n")
	b.WriteString("<title>" + html.EscapeString(title) + "</title>\n")
	b.WriteString("<script>globalThis.glutterConfig = " + safe + ";</script>\n")
	b.WriteString("<script src=\"wasm_exec.js\"></script>\n")
	b.WriteString("</head>\n<body>\n<noscript>This app needs JavaScript.</noscript>\n")
	b.WriteString(p.Body)
	b.WriteString("<script>\n" + strings.ReplaceAll(loaderJS, "</script", `<\/script`) + "</script>\n")
	b.WriteString("</body>\n</html>\n")
	return b.Bytes(), nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWebPageFailed = "WebPageFailed"
)
