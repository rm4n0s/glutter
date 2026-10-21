package web

import (
	"net/url"
	"strings"
)

// EnvPrefix starts the name of every query parameter that the loader copies into the
// Go program's environment.
//
// Flutter: no counterpart
const EnvPrefix = "GLUTTER_"

// EnvFromQuery returns the environment that the query string of a page's URL
// (location.search, with or without the leading "?") gives the Go program: the
// parameters whose names start with GLUTTER_, so that ?GLUTTER_RENDERER=webgl works as
// the environment variable does on Linux (SPEC §23.1). Names and values are
// percent-decoded; the last of a repeated name wins; a parameter without "=" has an
// empty value. Parameters that fail to decode, or whose name has a "=" or a NUL, are
// left out.
//
// The loader in index.html does the same in JavaScript before the program starts, because
// the engine reads its environment at startup; this is the specification it follows,
// and the loader's tests check it against this function's table.
//
// Flutter: no counterpart
func EnvFromQuery(search string) map[string]string {
	env := map[string]string{}
	search = strings.TrimPrefix(search, "?")
	if search == "" {
		return env
	}
	for _, pair := range strings.Split(search, "&") {
		rawName, rawValue, _ := strings.Cut(pair, "=")
		name, err := url.QueryUnescape(rawName)
		if err != nil || !strings.HasPrefix(name, EnvPrefix) || strings.ContainsAny(name, "=\x00") {
			continue
		}
		value, err := url.QueryUnescape(rawValue)
		if err != nil || strings.Contains(value, "\x00") {
			continue
		}
		env[name] = value
	}
	return env
}
