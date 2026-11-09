package macos

import (
	"net/url"
	"path"
	"strings"

	"github.com/rm4n0s/errors"
	"github.com/rm4n0s/glutter/internal/platform/mobile"
)

// This file is the plain-Go part of opening a URI (SPEC §29.9): the checks that come before NSWorkspace, which also
// launches programs.

// launchExtensions are the types of file whose file: URI OpenURI refuses: NSWorkspace would run them or open them
// with the program that runs them (an application bundle, a script, an installer).
var launchExtensions = map[string]bool{
	".app": true, ".command": true, ".tool": true, ".sh": true, ".bash": true, ".zsh": true, ".csh": true,
	".pkg": true, ".mpkg": true, ".dmg": true, ".scpt": true, ".scptd": true, ".applescript": true, ".workflow": true,
	".action": true, ".terminal": true, ".jar": true, ".pl": true, ".py": true, ".rb": true,
	".exe": true, ".bat": true, ".cmd": true, ".ps1": true, ".js": true, ".vbs": true, ".prefpane": true, ".saver": true,
}

// checkOpenURI is the validation before the URI is opened (SPEC §29.9), after Linux's and Windows': an empty URI,
// or one with a control character, is OpenURIInvalid; a URI with no scheme is OpenURIInvalid; the schemes
// javascript: and vbscript:, and a file: URI of a program, a script, an installer or an application bundle are
// OpenURISchemeBlocked. Anything else goes to NSWorkspace.
func checkOpenURI(uri string) error {
	if uri == "" || strings.IndexFunc(uri, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return errors.New(tagOpenURIInvalid, "a URI is not empty and has no control character", "uri", uri)
	}
	scheme := mobile.URIScheme(uri)
	switch {
	case scheme == "":
		return errors.New(tagOpenURIInvalid, "a URI needs a scheme, such as https: or file:", "uri", uri)
	case scheme == "javascript" || scheme == "vbscript":
		return errors.New(tagOpenURISchemeBlocked, "a URI that runs script is not opened", "uri", uri, "scheme", scheme)
	case scheme == "file":
		name := fileURIName(uri)
		if launchExtensions[strings.ToLower(path.Ext(name))] {
			return errors.New(tagOpenURISchemeBlocked, "a file: URI of a program, a script or an installer is not opened", "uri", uri, "scheme", scheme, "name", name)
		}
	}
	return nil
}

// fileURIName is the file name of a file: URI, decoded, without the trailing slash of a bundle's directory.
func fileURIName(raw string) string {
	rest := raw[strings.IndexByte(raw, ':')+1:]
	if i := strings.IndexAny(rest, "?#"); i >= 0 {
		rest = rest[:i]
	}
	if u, err := url.PathUnescape(rest); err == nil {
		rest = u
	}
	return path.Base(strings.TrimRight(rest, "/"))
}

// The tags of the errors that this file makes (rm4n0s/errors, R5); package desktop matches them by value.
const (
	tagOpenURIInvalid       = "OpenURIInvalid"
	tagOpenURISchemeBlocked = "OpenURISchemeBlocked"
	tagOpenURIUnavailable   = "OpenURIUnavailable"
)
