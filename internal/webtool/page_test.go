package webtool

import (
	"strings"
	"testing"
)

func TestIndexHTML(t *testing.T) {
	tests := []struct {
		name     string
		page     Page
		contains []string
		absent   []string
	}{
		{"defaults", Page{}, []string{
			"<!doctype html>", `<meta charset="utf-8">`, "<title>Glutter</title>", "globalThis.glutterConfig = {};",
			`<script src="wasm_exec.js"></script>`, "WebAssembly.instantiateStreaming", `fetch("app.wasm")`,
			"glutterEnvFromQuery(location.search)", `name="viewport"`, "<noscript>",
		}, []string{"hostElement"}},
		{"title is escaped", Page{Title: `a <b> & "c"`}, []string{"<title>a &lt;b&gt; &amp; &#34;c&#34;</title>"}, nil},
		{"host element", Page{HostElement: "#app"}, []string{`globalThis.glutterConfig = {"hostElement":"#app"};`}, nil},
		{"selector cannot end the script", Page{HostElement: `</script><b>"&`}, []string{"\\u003c/script\\u003e\\u003cb\\u003e"}, []string{`{"hostElement":"</script>`}},
		{"body comes before the loader", Page{Body: `<div id="stage"></div>`}, []string{`<div id="stage"></div>`}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := IndexHTML(tc.page)
			if err != nil {
				t.Fatal(err)
			}
			html := string(out)
			for _, want := range tc.contains {
				if !strings.Contains(html, want) {
					t.Errorf("index.html lacks %q", want)
				}
			}
			for _, bad := range tc.absent {
				if strings.Contains(html, bad) {
					t.Errorf("index.html has %q", bad)
				}
			}
		})
	}
}

func TestIndexHTMLOrder(t *testing.T) {
	out, err := IndexHTML(Page{Body: "<p id=host></p>"})
	if err != nil {
		t.Fatal(err)
	}
	html := string(out)
	cfg, exec, host, loader := strings.Index(html, "glutterConfig ="), strings.Index(html, "wasm_exec.js"), strings.Index(html, "<p id=host>"), strings.Index(html, "function glutterBoot")
	if cfg >= exec || exec >= host || host >= loader {
		t.Errorf("the page has config at %d, wasm_exec.js at %d, host at %d, loader at %d: want that order", cfg, exec, host, loader)
	}
	if strings.Count(html, "</script>") != 3 {
		t.Errorf("%d script elements, want 3 (config, wasm_exec.js, loader)", strings.Count(html, "</script>"))
	}
}
