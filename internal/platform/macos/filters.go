package macos

import (
	"path"
	"strings"

	"github.com/rm4n0s/glutter/internal/platform"
)

// This file is the plain-Go part of the file dialogs (SPEC §29.9): the filters of the dialog as the extensions that
// the allowed content types are made from. The calls to AppKit are in filedialog_darwin.go.

// filterGroup is a filter of the dialog: its name and the extensions (lowercase, without the dot) of its patterns.
type filterGroup struct {
	Name       string
	Extensions []string
}

// filterPlan is what the dialog allows: every file, or the files of the groups.
type filterPlan struct {
	// AllFiles says that a filter allows every file ("*" or "*.*"), so no type restricts the dialog.
	AllFiles bool
	Groups   []filterGroup
}

// extensionOfPattern is the extension that a pattern such as "*.png" or "report.TXT" stands for, lowercase.
// all is true for "*" and "*.*", which stand for every file; ext is empty, and all false, for a pattern that names
// no extension (a name with no dot).
func extensionOfPattern(pattern string) (ext string, all bool) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "*" || pattern == "*.*" {
		return "", true
	}
	ext = strings.ToLower(strings.TrimPrefix(path.Ext(pattern), "."))
	if strings.ContainsAny(ext, "*?[]") {
		return "", false
	}
	return ext, false
}

// planFilters turns the filters of a dialog into the plan. A filter with a pattern for every file makes the plan
// allow every file; the extensions of a group are listed once, in the order of its patterns.
func planFilters(filters []platform.FileFilter) filterPlan {
	var plan filterPlan
	for _, f := range filters {
		g := filterGroup{Name: f.Name}
		for _, p := range f.Patterns {
			ext, all := extensionOfPattern(p)
			switch {
			case all:
				plan.AllFiles = true
			case ext != "":
				dup := false
				for _, e := range g.Extensions {
					dup = dup || e == ext
				}
				if !dup {
					g.Extensions = append(g.Extensions, ext)
				}
			}
		}
		plan.Groups = append(plan.Groups, g)
	}
	return plan
}

// extensions are the extensions that a dialog that opens allows: those of every group, once each; nil when a filter
// allows every file or there is no filter with an extension.
func (p filterPlan) extensions() []string {
	if p.AllFiles {
		return nil
	}
	var out []string
	for _, g := range p.Groups {
		for _, e := range g.Extensions {
			dup := false
			for _, o := range out {
				dup = dup || o == e
			}
			if !dup {
				out = append(out, e)
			}
		}
	}
	return out
}

// saveExtensions are the extensions that a dialog that saves allows: those of the first filter (the one that is
// chosen), nil when it allows every file.
func (p filterPlan) saveExtensions() []string {
	if len(p.Groups) == 0 {
		return nil
	}
	return p.Groups[0].Extensions
}

// filterNameFor is the name of the first filter that has the extension of the file, "" when none has.
func (p filterPlan) filterNameFor(file string) string {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(file), "."))
	if ext == "" {
		return ""
	}
	for _, g := range p.Groups {
		for _, e := range g.Extensions {
			if e == ext {
				return g.Name
			}
		}
	}
	return ""
}
