package web

import (
	"maps"
	"testing"
)

// queryCases is the table of the rules of EnvFromQuery. The loader of glutter-web
// (internal/webtool/loader.js) follows the same rules, and its test runs the same
// table through Node: keep the two in step (internal/webtool/loader_test.go).
var queryCases = []struct {
	name   string
	search string
	want   map[string]string
}{
	{"empty", "", map[string]string{}},
	{"question mark only", "?", map[string]string{}},
	{"one", "?GLUTTER_RENDERER=webgl", map[string]string{"GLUTTER_RENDERER": "webgl"}},
	{"without the question mark", "GLUTTER_RENDERER=soft", map[string]string{"GLUTTER_RENDERER": "soft"}},
	{"other parameters are not copied", "?a=1&GLUTTER_X=2&glutter_y=3&XGLUTTER_Z=4", map[string]string{"GLUTTER_X": "2"}},
	{"percent decoding", "?GLUTTER_SCREENSHOT_AFTER=500%6Ds&GLUTTER_A%42=x%20y", map[string]string{"GLUTTER_SCREENSHOT_AFTER": "500ms", "GLUTTER_AB": "x y"}},
	{"plus is a space", "?GLUTTER_T=a+b", map[string]string{"GLUTTER_T": "a b"}},
	{"no value", "?GLUTTER_FLAG&GLUTTER_EMPTY=", map[string]string{"GLUTTER_FLAG": "", "GLUTTER_EMPTY": ""}},
	{"the last one wins", "?GLUTTER_A=1&GLUTTER_A=2", map[string]string{"GLUTTER_A": "2"}},
	{"a value may hold an equals sign", "?GLUTTER_A=b=c", map[string]string{"GLUTTER_A": "b=c"}},
	{"a name with an equals sign is dropped", "?GLUTTER_A%3DB=1&GLUTTER_C=2", map[string]string{"GLUTTER_C": "2"}},
	{"a bad escape is dropped", "?GLUTTER_A=%zz&GLUTTER_B=1", map[string]string{"GLUTTER_B": "1"}},
	{"a NUL is dropped", "?GLUTTER_A=%00&GLUTTER_B=1", map[string]string{"GLUTTER_B": "1"}},
	{"a fragment is not part of the query string", "?GLUTTER_A=1", map[string]string{"GLUTTER_A": "1"}},
}

func TestEnvFromQuery(t *testing.T) {
	for _, tc := range queryCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EnvFromQuery(tc.search); !maps.Equal(got, tc.want) {
				t.Errorf("EnvFromQuery(%q) = %v, want %v", tc.search, got, tc.want)
			}
		})
	}
}
