// Package keys implements the `glutter-gen keys` subcommand, which generates the key tables (XKB keycode to Flutter key ids).
//
// Flutter: no counterpart.
package keys

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/rm4n0s/errors"
)

const (
	defaultReference = "reference/flutter"
	defaultOut       = "internal/platform/linux/keymap_gen.go"
	defaultSvcOut    = "services/keyboard_key_gen.go"
	webLocationFile  = "web_logical_location_mapping.json"
	dataDir          = "dev/tools/gen_keycodes/data"
)

// Run executes `glutter-gen keys [-reference dir] [-out file] [-services-out file]
// [-web-out file]` with the arguments that follow the subcommand name. It reads
// Flutter's key data (physical_key_data.g.json, logical_key_data.g.json,
// layout_goals.json, synonyms.json and web_logical_location_mapping.json under
// dev/tools/gen_keycodes/data of the Flutter checkout in the reference directory)
// and writes the Go tables of the platform layer to the output file, the key
// constants of package services to the services file, and the tables of the web
// platform (the names of KeyboardEvent.code and KeyboardEvent.key) to the web file.
// An empty file name skips that file; the services file is written by default
// only when the platform file is, and the web file only when it is named.
//
// Flutter: dev/tools/gen_keycodes
func Run(args []string) error {
	flags := flag.NewFlagSet("keys", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	reference := flags.String("reference", defaultReference, "directory with the Flutter sources (see scripts/fetch-reference.sh)")
	out := flags.String("out", defaultOut, "file to write the platform tables to (empty: none)")
	svcOut := flags.String("services-out", "", "file to write the key constants of package services to (empty: none, or "+defaultSvcOut+" with the default -out)")
	webOut := flags.String("web-out", "", "file to write the tables of the web platform to (empty: none)")
	if err := flags.Parse(args); err != nil {
		return errors.NewErr(tagKeysBadFlags, err).SetMetadata("args", args)
	}
	if *svcOut == "" && *out == defaultOut {
		*svcOut = defaultSvcOut
	}
	src, err := loadData(filepath.Join(*reference, dataDir), *svcOut != "", *webOut != "")
	if err != nil {
		return err
	}
	if *out != "" {
		code, err := generate(src)
		if err != nil {
			return err
		}
		if err := writeIfChanged(*out, code); err != nil {
			return err
		}
	}
	if *webOut != "" {
		code, err := generateWeb(src)
		if err != nil {
			return err
		}
		if err := writeIfChanged(*webOut, code); err != nil {
			return err
		}
	}
	if *svcOut != "" {
		code, err := generateServices(src)
		if err != nil {
			return err
		}
		if err := writeIfChanged(*svcOut, code); err != nil {
			return err
		}
	}
	return nil
}

func writeIfChanged(path string, code []byte) error {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, code) {
		return nil
	}
	if err := os.WriteFile(path, code, 0o644); err != nil {
		return errors.NewErr(tagKeysWriteFailed, err).SetMetadata("file", path)
	}
	return nil
}

// physicalKey is an entry of physical_key_data.g.json.
type physicalKey struct {
	Names     map[string]string          `json:"names"`
	ScanCodes map[string]json.RawMessage `json:"scanCodes"`
	// OtherWebCodes are the older KeyboardEvent.code values of the key.
	OtherWebCodes []string `json:"otherWebCodes"`
}

// logicalKey is an entry of logical_key_data.g.json.
type logicalKey struct {
	Name   string              `json:"name"`
	Value  uint64              `json:"value"`
	Names  map[string][]string `json:"names"`
	Values map[string][]uint64 `json:"values"`
}

// data is Flutter's key data.
type data struct {
	physical map[string]physicalKey
	logical  map[string]logicalKey
	goals    map[string]bool
	// synonyms maps a pseudo-key (Shift) to the keys it stands for.
	synonyms map[string][]string
	// webLocations maps a KeyboardEvent.key to the names of the logical keys at the
	// locations standard, left, right and numpad (nil for none).
	webLocations map[string][]*string
}

func loadData(dir string, withSynonyms, withWeb bool) (*data, error) {
	d := &data{}
	files := []struct {
		name string
		into any
	}{
		{"physical_key_data.g.json", &d.physical},
		{"logical_key_data.g.json", &d.logical},
		{"layout_goals.json", &d.goals},
	}
	if withSynonyms {
		files = append(files, struct {
			name string
			into any
		}{"synonyms.json", &d.synonyms})
	}
	if withWeb {
		files = append(files, struct {
			name string
			into any
		}{webLocationFile, &d.webLocations})
	}
	for _, f := range files {
		path := filepath.Join(dir, f.name)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, errors.NewErr(tagKeysDataMissing, err).
				SetMetadata("file", path, "hint", "run scripts/fetch-reference.sh")
		}
		if err := json.Unmarshal(raw, f.into); err != nil {
			return nil, errors.NewErr(tagKeysDataInvalid, err).SetMetadata("file", path)
		}
	}
	return d, nil
}

// scanCode returns the integer scan code of a column (such as "usb" or "xkb")
// of the key called name, and false when the key has none there. It fails with
// KeysDataInvalid.
func (k physicalKey) scanCode(name, column string) (uint64, bool, error) {
	raw, ok := k.ScanCodes[column]
	if !ok {
		return 0, false, nil
	}
	var v uint64
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, false, errors.NewErr(tagKeysDataInvalid, err).SetMetadata("key", name, "column", column)
	}
	return v, true, nil
}

// goal is one key of the layout goals.
type goal struct {
	name      string
	xkb       uint64
	logical   uint64
	mandatory bool
}

func generate(d *data) ([]byte, error) {
	// XKB keycode -> physical key (the USB HID usage Flutter uses as the id).
	physical := map[uint64]uint64{}
	owner := map[uint64]string{}
	xkbOf := map[string]uint64{}
	for _, name := range sortedKeys(d.physical) {
		k := d.physical[name]
		xkb, hasXkb, err := k.scanCode(name, "xkb")
		if err != nil {
			return nil, err
		}
		usb, hasUSB, err := k.scanCode(name, "usb")
		if err != nil {
			return nil, err
		}
		if !hasXkb || !hasUSB {
			continue
		}
		if prev, dup := owner[xkb]; dup {
			return nil, errors.New(tagKeysDuplicate, "two physical keys share an XKB keycode",
				"xkb", xkb, "keys", []string{prev, name})
		}
		owner[xkb] = name
		physical[xkb] = usb
		xkbOf[name] = xkb
	}

	// GTK keyval (an X keysym) -> logical key.
	logical := map[uint64]uint64{}
	logicalOwner := map[uint64]string{}
	for _, name := range sortedKeys(d.logical) {
		k := d.logical[name]
		for _, sym := range k.Values["gtk"] {
			if prev, dup := logicalOwner[sym]; dup {
				return nil, errors.New(tagKeysDuplicate, "two logical keys share a GTK keyval",
					"keyval", sym, "keys", []string{prev, name})
			}
			logicalOwner[sym] = name
			logical[sym] = k.Value
		}
	}

	var goals []goal
	for _, name := range sortedKeys(d.goals) {
		xkb, ok := xkbOf[name]
		if !ok {
			return nil, errors.New(tagKeysGoalUnknown, "a layout goal names a physical key with no XKB keycode", "key", name)
		}
		l, ok := d.logical[name]
		if !ok {
			return nil, errors.New(tagKeysGoalUnknown, "a layout goal names a key that has no logical key", "key", name)
		}
		goals = append(goals, goal{name, xkb, l.Value, d.goals[name]})
	}
	sort.Slice(goals, func(i, j int) bool { return goals[i].xkb < goals[j].xkb })

	var b bytes.Buffer
	b.WriteString("// Code generated by glutter-gen keys; DO NOT EDIT.\n")
	b.WriteString("// Source: Flutter's dev/tools/gen_keycodes/data (physical_key_data.g.json,\n")
	b.WriteString("// logical_key_data.g.json, layout_goals.json). Flutter is BSD-3-Clause\n")
	b.WriteString("// licensed; see third_party/flutter/LICENSE.\n\n")
	b.WriteString("//go:build linux && !android\n\n")
	b.WriteString("package linux\n\n")

	b.WriteString("// xkbToPhysicalKey maps an XKB keycode (evdev scancode + 8) to the id of\n// Flutter's PhysicalKeyboardKey.\n")
	b.WriteString("var xkbToPhysicalKey = map[uint32]uint64{\n")
	for _, xkb := range sortedUint64(physical) {
		fmt.Fprintf(&b, "\t%d: 0x%x, // %s\n", xkb, physical[xkb], owner[xkb])
	}
	b.WriteString("}\n\n")

	b.WriteString("// keysymToLogicalKey maps an X keysym to the id of Flutter's\n// LogicalKeyboardKey, for the keys that are not printable characters.\n")
	b.WriteString("var keysymToLogicalKey = map[uint32]uint64{\n")
	for _, sym := range sortedUint64(logical) {
		fmt.Fprintf(&b, "\t0x%x: 0x%x, // %s\n", sym, logical[sym], logicalOwner[sym])
	}
	b.WriteString("}\n\n")

	b.WriteString("// layoutGoals lists the keys whose logical key does not follow the layout.\n")
	b.WriteString("var layoutGoals = []layoutGoal{\n")
	for _, g := range goals {
		fmt.Fprintf(&b, "\t{xkbKeycode: %d, logical: 0x%x, mandatory: %t}, // %s\n", g.xkb, g.logical, g.mandatory, g.name)
	}
	b.WriteString("}\n")

	out, err := format.Source(b.Bytes())
	if err != nil {
		return nil, errors.NewErr(tagKeysFormatFailed, err)
	}
	return out, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedUint64[V any](m map[uint64]V) []uint64 {
	keys := make([]uint64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagKeysBadFlags     = "KeysBadFlags"
	tagKeysDataInvalid  = "KeysDataInvalid"
	tagKeysDataMissing  = "KeysDataMissing"
	tagKeysDuplicate    = "KeysDuplicate"
	tagKeysFormatFailed = "KeysFormatFailed"
	tagKeysGoalUnknown  = "KeysGoalUnknown"
	tagKeysWriteFailed  = "KeysWriteFailed"
)
