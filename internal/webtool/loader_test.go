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
