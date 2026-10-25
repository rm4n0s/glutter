package keys

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rm4n0s/errors"
)

// pinnedFlutter names the Flutter commit that the reference directory holds, from the
// reference/VERSION file that scripts/fetch-reference.sh writes next to it (the reference
// directory is reference/flutter). It returns "an unpinned checkout" when the file is
// missing or has no commit.
func pinnedFlutter(reference string) string {
	raw, err := os.ReadFile(filepath.Join(reference, "..", "VERSION"))
	if err != nil {
		return "an unpinned checkout"
	}
	var commit, version string
	for _, line := range strings.Split(string(raw), "\n") {
		if v, ok := strings.CutPrefix(line, "flutter_commit="); ok {
			commit = strings.TrimSpace(v)
		}
		if v, ok := strings.CutPrefix(line, "flutter_version="); ok {
			version = strings.TrimSpace(v)
		}
	}
	if commit == "" {
		return "an unpinned checkout"
	}
	if version != "" {
		return fmt.Sprintf("commit %s (Flutter %s)", commit, version)
	}
	return "commit " + commit
}

// sortGoals orders the layout goals by scan code.
func sortGoals(goals []goal) {
	sort.Slice(goals, func(i, j int) bool { return goals[i].xkb < goals[j].xkb })
}

// windowsVirtualKeys returns the virtual-key codes and their names (VK_ without the prefix) of
// a logical key; the two lists are the same length. It fails with KeysDataInvalid when they
// are not.
func (k logicalKey) windowsVirtualKeys(name string) (codes []uint64, names []string, err error) {
	codes, names = k.Values["windows"], k.Names["windows"]
	if len(codes) != len(names) {
		return nil, nil, errors.New(tagKeysDataInvalid, "a logical key has a different number of Windows virtual keys and names",
			"key", name, "codes", len(codes), "names", len(names))
	}
	return codes, names, nil
}

// generateWindows writes the tables of the Windows platform: the physical key of each scan code
// (windowsToPhysicalMap_), the logical key of each virtual key (windowsToLogicalMap_) and of the
// scan codes that decide the logical key (scanCodeToLogicalMap_), the names of the virtual keys,
// and the layout goals with their scan codes. It ports windows_flutter_key_map_cc.tmpl of
// dev/tools/gen_keycodes, whose output is flutter_key_map.g.cc.
func generateWindows(d *data, pinned string) ([]byte, error) {
	// Windows scan code -> physical key (the USB HID usage Flutter uses as the id).
	physical := map[uint64]uint64{}
	physicalOwner := map[uint64]string{}
	scanOf := map[string]uint64{}
	for _, name := range sortedKeys(d.physical) {
		k := d.physical[name]
		scan, hasScan, err := k.scanCode(name, "windows")
		if err != nil {
			return nil, err
		}
		usb, hasUSB, err := k.scanCode(name, "usb")
		if err != nil {
			return nil, err
		}
		if !hasScan || !hasUSB {
			continue
		}
		if prev, dup := physicalOwner[scan]; dup {
			return nil, errors.New(tagKeysDuplicate, "two physical keys share a Windows scan code",
				"scanCode", scan, "keys", []string{prev, name})
		}
		physicalOwner[scan] = name
		physical[scan] = usb
		scanOf[name] = scan
	}

	// Windows virtual key -> logical key.
	logical := map[uint64]uint64{}
	logicalOwner := map[uint64]string{}
	vkNames := map[string]uint64{}
	for _, name := range sortedKeys(d.logical) {
		k := d.logical[name]
		codes, names, err := k.windowsVirtualKeys(name)
		if err != nil {
			return nil, err
		}
		for i, vk := range codes {
			if prev, dup := logicalOwner[vk]; dup {
				return nil, errors.New(tagKeysDuplicate, "two logical keys share a Windows virtual key",
					"virtualKey", vk, "keys", []string{prev, name})
			}
			logicalOwner[vk] = name
			logical[vk] = k.Value
			if prev, dup := vkNames[names[i]]; dup {
				return nil, errors.New(tagKeysDuplicate, "two Windows virtual keys share a name",
					"name", names[i], "virtualKeys", []uint64{prev, vk})
			}
			vkNames[names[i]] = vk
		}
	}

	// The scan codes that decide the logical key (the numpad, and the right-hand Control and Alt).
	byScan := map[uint64]uint64{}
	byScanOwner := map[uint64]string{}
	for _, name := range sortedKeys(d.windowsScan) {
		scan, ok := scanOf[name]
		if !ok {
			return nil, errors.New(tagKeysGoalUnknown, "a Windows scan code mapping names a physical key with no Windows scan code", "key", name)
		}
		l, ok := d.logical[d.windowsScan[name]]
		if !ok {
			return nil, errors.New(tagKeysGoalUnknown, "a Windows scan code mapping names a key that has no logical key",
				"key", name, "logical", d.windowsScan[name])
		}
		byScan[scan] = l.Value
		byScanOwner[scan] = d.windowsScan[name]
	}

	// The layout goals.
	var goals []goal
	for _, name := range sortedKeys(d.goals) {
		scan, ok := scanOf[name]
		if !ok {
			return nil, errors.New(tagKeysGoalUnknown, "a layout goal names a physical key with no Windows scan code", "key", name)
		}
		l, ok := d.logical[name]
		if !ok {
			return nil, errors.New(tagKeysGoalUnknown, "a layout goal names a key that has no logical key", "key", name)
		}
		goals = append(goals, goal{name, scan, l.Value, d.goals[name]})
	}
	sortGoals(goals)

	var b bytes.Buffer
	b.WriteString("// Code generated by glutter-gen keys; DO NOT EDIT.\n")
	fmt.Fprintf(&b, "// Source: Flutter's dev/tools/gen_keycodes/data, %s:\n", pinned)
	b.WriteString("// physical_key_data.g.json, logical_key_data.g.json, windows_scancode_logical_map.json and\n")
	b.WriteString("// layout_goals.json, the content of windows_flutter_key_map_cc.tmpl. Flutter is BSD-3-Clause\n")
	b.WriteString("// licensed; see third_party/flutter/LICENSE.\n\n")
	b.WriteString("package windows\n\n")

	b.WriteString("// scanCodeToPhysical maps a Windows scan code (bits 16 to 23 of the lParam of a key message, with\n")
	b.WriteString("// the prefix 0xE000 for an extended key) to the id of Flutter's PhysicalKeyboardKey (the USB HID usage).\n//\n")
	b.WriteString("// Flutter: engine/src/flutter/shell/platform/windows/flutter_key_map.g.cc KeyboardKeyEmbedderHandler::windowsToPhysicalMap_\n")
	b.WriteString("var scanCodeToPhysical = map[uint32]uint64{\n")
	for _, c := range sortedUint64(physical) {
		fmt.Fprintf(&b, "\t0x%x: 0x%x, // %s\n", c, physical[c], physicalOwner[c])
	}
	b.WriteString("}\n\n")

	b.WriteString("// virtualKeyToLogical maps a Windows virtual-key code to the id of Flutter's LogicalKeyboardKey.\n//\n")
	b.WriteString("// Flutter: engine/src/flutter/shell/platform/windows/flutter_key_map.g.cc KeyboardKeyEmbedderHandler::windowsToLogicalMap_\n")
	b.WriteString("var virtualKeyToLogical = map[uint32]uint64{\n")
	for _, c := range sortedUint64(logical) {
		fmt.Fprintf(&b, "\t0x%x: 0x%x, // %s\n", c, logical[c], logicalOwner[c])
	}
	b.WriteString("}\n\n")

	b.WriteString("// scanCodeToLogical maps the scan codes that decide the logical key (the numpad, whose virtual keys\n")
	b.WriteString("// depend on Num Lock, and the right-hand Control and Alt) to the id of Flutter's LogicalKeyboardKey.\n//\n")
	b.WriteString("// Flutter: engine/src/flutter/shell/platform/windows/flutter_key_map.g.cc KeyboardKeyEmbedderHandler::scanCodeToLogicalMap_\n")
	b.WriteString("var scanCodeToLogical = map[uint32]uint64{\n")
	for _, c := range sortedUint64(byScan) {
		fmt.Fprintf(&b, "\t0x%x: 0x%x, // %s\n", c, byScan[c], byScanOwner[c])
	}
	b.WriteString("}\n\n")

	b.WriteString("// virtualKeyNames maps the name of a virtual key (VK_ without the prefix, as Flutter's key data has\n")
	b.WriteString("// it) to its code; the scripted events use it.\n//\n")
	b.WriteString("// Flutter: dev/tools/gen_keycodes/data/windows_logical_to_window_vk.json\n")
	b.WriteString("var virtualKeyNames = map[string]uint32{\n")
	for _, n := range sortedKeys(vkNames) {
		fmt.Fprintf(&b, "\t%q: 0x%x,\n", n, vkNames[n])
	}
	b.WriteString("}\n\n")

	b.WriteString("// layoutGoals lists the keys whose logical key must not depend on the layout, so that shortcuts work on\n")
	b.WriteString("// every layout: a layout that has no Latin character on one of them gives it the logical key of the\n")
	b.WriteString("// US layout when the goal is mandatory.\n//\n")
	b.WriteString("// Flutter: dev/tools/gen_keycodes/data/layout_goals.json\n")
	b.WriteString("var layoutGoals = []layoutGoal{\n")
	for _, g := range goals {
		fmt.Fprintf(&b, "\t{scanCode: 0x%x, logical: 0x%x, mandatory: %t}, // %s\n", g.xkb, g.logical, g.mandatory, g.name)
	}
	b.WriteString("}\n")

	out, err := format.Source(b.Bytes())
	if err != nil {
		return nil, errors.NewErr(tagKeysFormatFailed, err)
	}
	return out, nil
}
