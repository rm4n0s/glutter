// Package gamepaddb implements the `glutter-gen gamepaddb` subcommand, which turns the two tables of gamepads in SDL 3 into Go: the
// mapping lines of src/joystick/SDL_gamepad_db.h and the vendor and product numbers of src/joystick/controller_list.h (SPEC §28.1).
//
// Flutter: no counterpart.
package gamepaddb

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

const (
	defaultReference = "reference/sdl"
	defaultOutDir    = "internal/gamepaddb"
	mappingsHeader   = "src/joystick/SDL_gamepad_db.h"
	controllersList  = "src/joystick/controller_list.h"
)

// Run executes `glutter-gen gamepaddb [-reference dir] [-out dir]`. It reads SDL 3's SDL_gamepad_db.h and controller_list.h under the
// reference directory and writes mappings_gen.go and controllers_gen.go to the output directory.
//
// Flutter: no counterpart
func Run(args []string) error {
	flags := flag.NewFlagSet("gamepaddb", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	reference := flags.String("reference", defaultReference, "directory with the SDL 3 sources (see scripts/fetch-reference.sh)")
	outDir := flags.String("out", defaultOutDir, "directory to write mappings_gen.go and controllers_gen.go to")
	if err := flags.Parse(args); err != nil {
		return errors.NewErr(tagGamepadDBBadFlags, err).SetMetadata("args", args)
	}
	mappingsText, err := os.ReadFile(filepath.Join(*reference, mappingsHeader))
	if err != nil {
		return errors.NewErr(tagGamepadDBReadFailed, err).SetMetadata("file", filepath.Join(*reference, mappingsHeader))
	}
	listText, err := os.ReadFile(filepath.Join(*reference, controllersList))
	if err != nil {
		return errors.NewErr(tagGamepadDBReadFailed, err).SetMetadata("file", filepath.Join(*reference, controllersList))
	}
	sections, err := parseMappings(string(mappingsText))
	if err != nil {
		return err
	}
	controllers, err := parseControllers(string(listText))
	if err != nil {
		return err
	}
	mappingsCode, err := generateMappings(sections)
	if err != nil {
		return err
	}
	controllersCode, err := generateControllers(controllers)
	if err != nil {
		return err
	}
	if err := writeIfChanged(filepath.Join(*outDir, "mappings_gen.go"), mappingsCode); err != nil {
		return err
	}
	return writeIfChanged(filepath.Join(*outDir, "controllers_gen.go"), controllersCode)
}

func writeIfChanged(path string, code []byte) error {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, code) {
		return nil
	}
	if err := os.WriteFile(path, code, 0o644); err != nil {
		return errors.NewErr(tagGamepadDBWriteFailed, err).SetMetadata("file", path)
	}
	return nil
}

// sectionOf names the Go variable that the lines under a C preprocessor condition of SDL_gamepad_db.h go to; "" skips them (a platform
// that Glutter does not run on, or the definitions of a private platform).
var sectionOf = map[string]string{
	"SDL_JOYSTICK_XINPUT":     "XInput",
	"SDL_JOYSTICK_WGI":        "WGI",
	"SDL_JOYSTICK_DINPUT":     "DInput",
	"SDL_JOYSTICK_LINUX":      "Linux",
	"SDL_PLATFORM_ANDROID":    "Android",
	"SDL_JOYSTICK_EMSCRIPTEN": "Web",
}

// sectionOrder is the order of the variables in the generated file.
var sectionOrder = []string{"XInput", "WGI", "DInput", "Linux", "Android", "Web"}

var (
	ifRE      = regexp.MustCompile(`^#\s*(?:ifdef|if defined\()\s*([A-Z0-9_]+)`)
	stringRE  = regexp.MustCompile(`^\s*"(.*)",\s*$`)
	endifRE   = regexp.MustCompile(`^#\s*endif`)
	anyCondRE = regexp.MustCompile(`^#\s*(?:if|ifdef|ifndef)\b`)
)

// parseMappings reads the mapping lines of SDL_gamepad_db.h and groups them by the platform of the preprocessor condition around them.
func parseMappings(text string) (map[string][]string, error) {
	out := map[string][]string{}
	var stack []string
	for n, line := range strings.Split(text, "\n") {
		switch {
		case anyCondRE.MatchString(line):
			section := ""
			if m := ifRE.FindStringSubmatch(line); m != nil {
				section = sectionOf[m[1]]
			}
			stack = append(stack, section)
		case endifRE.MatchString(line):
			if len(stack) == 0 {
				return nil, errors.New(tagGamepadDBSyntax, "an #endif has no #if", "line", n+1)
			}
			stack = stack[:len(stack)-1]
		default:
			m := stringRE.FindStringSubmatch(line)
			if m == nil || len(stack) == 0 {
				continue
			}
			section := stack[len(stack)-1]
			if section == "" {
				continue
			}
			s, err := strconv.Unquote(`"` + m[1] + `"`)
			if err != nil {
				return nil, errors.NewErr(tagGamepadDBSyntax, err).SetMetadata("line", n+1)
			}
			out[section] = append(out[section], s)
		}
	}
	for _, name := range sectionOrder {
		if len(out[name]) == 0 {
			return nil, errors.New(tagGamepadDBSyntax, "a section of the mapping table has no lines", "section", name)
		}
	}
	return out, nil
}

// controller is a line of controller_list.h.
type controller struct {
	vendor, product uint16
	typ             string // the name after k_eControllerType_
	name            string
}

var controllerRE = regexp.MustCompile(`^\s*\{\s*MAKE_CONTROLLER_ID\(\s*0x([0-9a-fA-F]+),\s*0x([0-9a-fA-F]+)\s*\),\s*k_eControllerType_([A-Za-z0-9]+),\s*(NULL|"[^"]*")\s*\}`)

// parseControllers reads the entries of controller_list.h, in file order (the first entry of a vendor and product wins, as in SDL).
func parseControllers(text string) ([]controller, error) {
	var out []controller
	for n, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		m := controllerRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		vendor, err := strconv.ParseUint(m[1], 16, 16)
		if err != nil {
			return nil, errors.NewErr(tagGamepadDBSyntax, err).SetMetadata("line", n+1)
		}
		product, err := strconv.ParseUint(m[2], 16, 16)
		if err != nil {
			return nil, errors.NewErr(tagGamepadDBSyntax, err).SetMetadata("line", n+1)
		}
		c := controller{vendor: uint16(vendor), product: uint16(product), typ: m[3]}
		if m[4] != "NULL" {
			c.name, err = strconv.Unquote(m[4])
			if err != nil {
				return nil, errors.NewErr(tagGamepadDBSyntax, err).SetMetadata("line", n+1)
			}
		}
		out = append(out, c)
	}
	if len(out) < 100 {
		return nil, errors.New(tagGamepadDBSyntax, "the controller list has too few entries", "entries", len(out))
	}
	return out, nil
}

const generatedNotice = `// Code generated by glutter-gen gamepaddb; DO NOT EDIT.
//
// This file is made from %s of SDL 3 (https://libsdl.org, zlib license), translated to Go and changed. SDL's notice is kept in
// third_party/sdl/LICENSE.

`

// formatCode formats the generated source; it fails only when the generator made source that is not Go.
func formatCode(src []byte) ([]byte, error) {
	out, err := format.Source(src)
	if err != nil {
		return nil, errors.NewErr(tagGamepadDBFormatFailed, err)
	}
	return out, nil
}

func generateMappings(sections map[string][]string) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, generatedNotice, mappingsHeader)
	b.WriteString("package gamepaddb\n\n")
	docs := map[string]string{
		"XInput":  "XInput is the mapping of the XInput pads on Windows (one line, its GUID is the name \"xinput\")",
		"WGI":     "WGI are the mappings of the pads that Windows.Gaming.Input knows",
		"DInput":  "DInput are the mappings of the pads that DirectInput and the raw HID of Windows know",
		"Linux":   "Linux are the mappings of the pads of the evdev joystick layer of Linux",
		"Android": "Android are the mappings of the pads of Android (by the GUID that the Android backend makes)",
		"Web":     "Web are the mappings that the browser's pads use when the browser's own mapping is not \"standard\"",
	}
	for _, name := range sectionOrder {
		fmt.Fprintf(&b, "// %s.\n//\n// Flutter: no counterpart\nvar %s = []string{\n", docs[name], name)
		for _, line := range sections[name] {
			fmt.Fprintf(&b, "\t%s,\n", strconv.Quote(line))
		}
		b.WriteString("}\n\n")
	}
	return formatCode(b.Bytes())
}

func generateControllers(list []controller) ([]byte, error) {
	// The first entry of a vendor and product wins: drop the later ones, then sort for a binary search.
	seen := map[uint32]bool{}
	var entries []controller
	for _, c := range list {
		key := uint32(c.vendor)<<16 | uint32(c.product)
		if seen[key] {
			continue
		}
		seen[key] = true
		entries = append(entries, c)
	}
	sort.Slice(entries, func(i, j int) bool {
		return uint32(entries[i].vendor)<<16|uint32(entries[i].product) < uint32(entries[j].vendor)<<16|uint32(entries[j].product)
	})
	typeSet := map[string]bool{}
	for _, c := range entries {
		typeSet[c.typ] = true
	}
	var types []string
	for t := range typeSet {
		types = append(types, t)
	}
	sort.Strings(types)

	var b bytes.Buffer
	fmt.Fprintf(&b, generatedNotice, controllersList)
	b.WriteString("package gamepaddb\n\n")
	b.WriteString("// ControllerType is what SDL 3's controller list says a pad with a vendor and a product number is (its k_eControllerType_ names).\n//\n// Flutter: no counterpart\ntype ControllerType uint8\n\nconst (\n\t// ControllerUnknown is a pad that is not in the list.\n\t//\n\t// Flutter: no counterpart\n\tControllerUnknown ControllerType = iota\n")
	for _, t := range types {
		fmt.Fprintf(&b, "\t// Controller%s is the list's %s.\n\t//\n\t// Flutter: no counterpart\n\tController%s\n", t, t, t)
	}
	b.WriteString(")\n\n")
	b.WriteString("// Controller is an entry of the list: Name is empty when SDL gives none.\n//\n// Flutter: no counterpart\ntype Controller struct {\n\tVendor, Product uint16\n\tType            ControllerType\n\tName            string\n}\n\n")
	b.WriteString("// Controllers is the list, sorted by vendor and then product.\n//\n// Flutter: no counterpart\nvar Controllers = []Controller{\n")
	for _, c := range entries {
		fmt.Fprintf(&b, "\t{0x%04x, 0x%04x, Controller%s, %s},\n", c.vendor, c.product, c.typ, strconv.Quote(c.name))
	}
	b.WriteString("}\n")
	return formatCode(b.Bytes())
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagGamepadDBBadFlags     = "GamepadDBBadFlags"
	tagGamepadDBReadFailed   = "GamepadDBReadFailed"
	tagGamepadDBWriteFailed  = "GamepadDBWriteFailed"
	tagGamepadDBSyntax       = "GamepadDBSyntax"
	tagGamepadDBFormatFailed = "GamepadDBFormatFailed"
)
