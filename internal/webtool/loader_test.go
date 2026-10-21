package webtool

import (
	"encoding/json"
	"maps"
	"os/exec"
	"strings"
	"testing"

	"github.com/rm4n0s/glutter/internal/platform/web"
)

// loaderSearches are query strings that the loader's glutterEnvFromQuery and Go's
// web.EnvFromQuery must read the same way. It is the table of web's query_test.go, plus the
// forms a URL really has.
var loaderSearches = []string{
	"", "?", "?GLUTTER_RENDERER=webgl", "GLUTTER_RENDERER=soft",
	"?a=1&GLUTTER_X=2&glutter_y=3&XGLUTTER_Z=4", "?GLUTTER_SCREENSHOT_AFTER=500%6Ds&GLUTTER_A%42=x%20y",
	"?GLUTTER_T=a+b", "?GLUTTER_FLAG&GLUTTER_EMPTY=", "?GLUTTER_A=1&GLUTTER_A=2", "?GLUTTER_A=b=c",
	"?GLUTTER_A%3DB=1&GLUTTER_C=2", "?GLUTTER_A=%zz&GLUTTER_B=1", "?GLUTTER_A=%00&GLUTTER_B=1",
	"?GLUTTER_DEV_TAPS=3&glutter_report=%2F__glutter%2Fresults&GLUTTER_SCREENSHOT=1",
	"?GLUTTER_A=%E2%82%AC", "&&GLUTTER_A=1&&",
}

// TestLoaderReadsTheQueryAsGoDoes runs the loader's function under Node and compares it with
// web.EnvFromQuery.
func TestLoaderReadsTheQueryAsGoDoes(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is not installed (sudo dnf install nodejs)")
	}
	input, err := json.Marshal(loaderSearches)
	if err != nil {
		t.Fatal(err)
	}
	script := `const fs = require("fs");
eval(process.argv[1] + "\n;globalThis.envOf = glutterEnvFromQuery;");
const searches = JSON.parse(fs.readFileSync(0, "utf8"));
process.stdout.write(JSON.stringify(searches.map((s) => globalThis.envOf(s))));`
	cmd := exec.Command(node, "-e", script, LoaderJS())
	cmd.Stdin = strings.NewReader(string(input))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("node failed: %v", err)
	}
	var got []map[string]string
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("node printed %q: %v", out, err)
	}
	if len(got) != len(loaderSearches) {
		t.Fatalf("node answered %d searches, want %d", len(got), len(loaderSearches))
	}
	for i, search := range loaderSearches {
		if want := web.EnvFromQuery(search); !maps.Equal(got[i], want) {
			t.Errorf("search %q: the loader gives %v, Go gives %v", search, got[i], want)
		}
	}
}

func TestLoaderReportURL(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is not installed (sudo dnf install nodejs)")
	}
	script := `eval(process.argv[1] + "\n;globalThis.f = glutterReportURL;");
process.stdout.write(JSON.stringify(["", "?a=1", "?glutter_report=%2Fr", "?a=1&glutter_report=/x&b=2", "?xglutter_report=/no"].map(globalThis.f)));`
	out, err := exec.Command(node, "-e", script, LoaderJS()).Output()
	if err != nil {
		t.Fatalf("node failed: %v", err)
	}
	if want := `["","","/r","/x",""]`; string(out) != want {
		t.Errorf("glutterReportURL gives %s, want %s", out, want)
	}
}

func TestLoaderLogEvery(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is not installed (sudo dnf install nodejs)")
	}
	script := `eval(process.argv[1] + "\n;globalThis.f = glutterLogEvery;");
process.stdout.write(JSON.stringify(["", "?a=1", "?glutter_log_every=1000", "?a=1&glutter_log_every=250&b=2", "?glutter_log_every=x", "?xglutter_log_every=5", "?glutter_log_every=-5", "?glutter_log_every="].map(globalThis.f)));`
	out, err := exec.Command(node, "-e", script, LoaderJS()).Output()
	if err != nil {
		t.Fatalf("node failed: %v", err)
	}
	if want := `[0,0,1000,250,0,0,0,0]`; string(out) != want {
		t.Errorf("glutterLogEvery gives %s, want %s", out, want)
	}
}

// runLoaderFunctions evaluates the loader under Node with a document that is just enough for
// glutterShow, runs the script and returns what it prints as JSON.
func runLoaderFunctions(t *testing.T, script string) []byte {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is not installed (sudo dnf install nodejs)")
	}
	prelude := `const elements = [];
globalThis.document = {
  body: { appendChild(e) { elements.push(e); e.parent = this; } },
  documentElement: { appendChild(e) { elements.push(e); } },
  createElement(tag) { return { tag, style: {}, attrs: {}, setAttribute(k, v) { this.attrs[k] = v; } }; },
  getElementById(id) { return elements.find((e) => e.id === id) || null; },
};
eval(process.argv[1]);
`
	out, err := exec.Command(node, "-e", prelude+script, LoaderJS()).Output()
	if err != nil {
		t.Fatalf("node failed: %v", err)
	}
	return out
}

// TestLoaderShowsTheFailureInThePage: a program that stops with an error (a renderer asked for by
// name that the browser lacks, for example) is told to the user in the page, over whatever the
// program drew, with the last lines of what it wrote; one message is kept and replaced (SPEC §23.1).
func TestLoaderShowsTheFailureInThePage(t *testing.T) {
	log := "2026/10/21 12:00:00 INFO startup stage=a\n2026/10/21 12:00:01 INFO renderer not usable\n\n" +
		"2026/10/21 12:00:02 WebGPUUnavailable: this browser has no usable WebGPU reason=\"no navigator.gpu\"\n"
	input, err := json.Marshal(map[string]string{"log": log, "long": strings.Repeat("x", 5000), "many": strings.Repeat("line\n", 20) + "last\n"})
	if err != nil {
		t.Fatal(err)
	}
	out := runLoaderFunctions(t, `const input = `+string(input)+`;
glutterShow(glutterFailureText(1, input.log));
const first = document.getElementById("glutter-message");
glutterShow("again");
process.stdout.write(JSON.stringify({
  text: glutterFailureText(1, input.log),
  empty: glutterFailureText(2, ""),
  long: glutterFailureText(3, input.long).length,
  many: glutterFailureText(4, input.many).split("\n").length,
  count: elements.length,
  sameElement: first === document.getElementById("glutter-message"),
  finalText: document.getElementById("glutter-message").textContent,
  role: first.attrs.role,
  fixed: first.style.cssText.includes("position:fixed"),
}));`)
	var got struct {
		Text, Empty, FinalText, Role string
		Long, Many, Count            int
		SameElement, Fixed           bool
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("node printed %q: %v", out, err)
	}
	if !strings.Contains(got.Text, "exit code 1") || !strings.Contains(got.Text, "WebGPUUnavailable: this browser has no usable WebGPU") {
		t.Errorf("the message is %q: it should have the exit code and the error of the log", got.Text)
	}
	if got.Empty != "The app stopped with an error (exit code 2)." {
		t.Errorf("without a log the message is %q", got.Empty)
	}
	if got.Long > 400 {
		t.Errorf("a long line gave a message of %d characters, want it cut", got.Long)
	}
	if want := 2 + 4; got.Many != want { // the first line, a blank one, and the last four lines
		t.Errorf("the message has %d lines, want %d: only the last four lines of the log", got.Many, want)
	}
	if strings.Contains(got.Text, "INFO") {
		t.Errorf("the message has Info lines, which only hide the error: %q", got.Text)
	}
	if got.Count != 1 || !got.SameElement || got.FinalText != "again" {
		t.Errorf("%d elements, same element %v, text %q: the message should be one element, replaced", got.Count, got.SameElement, got.FinalText)
	}
	if got.Role != "alert" || !got.Fixed {
		t.Errorf("role %q, fixed %v: the message should be an alert over the page", got.Role, got.Fixed)
	}
}
