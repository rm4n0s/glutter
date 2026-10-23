package web

import (
	"strings"

	"github.com/rm4n0s/errors"
)

// Config is what the page tells the app through globalThis.glutterConfig, which the
// index.html of glutter-web sets (SPEC §23.2).
//
// Flutter: lib/web_ui/lib/src/engine/configuration.dart FlutterConfiguration
type Config struct {
	// HostElement is the CSS selector of the element the view fills. Empty means the
	// view fills the page.
	HostElement string
}

// ParseConfig reads the fields of glutterConfig, given as the plain values of a
// JavaScript object: a nil map is the zero Config. A field of the wrong type fails with
// WebConfigInvalid. Fields it does not know are ignored, so that a later step can add
// some.
//
// Flutter: lib/web_ui/lib/src/engine/configuration.dart FlutterConfiguration.fromJs
func ParseConfig(fields map[string]any) (Config, error) {
	var c Config
	if v, ok := fields["hostElement"]; ok && v != nil {
		s, isString := v.(string)
		if !isString {
			return Config{}, errors.New(tagWebConfigInvalid, "glutterConfig.hostElement must be a CSS selector string")
		}
		c.HostElement = strings.TrimSpace(s)
	}
	return c, nil
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWebConfigInvalid = "WebConfigInvalid"
)
