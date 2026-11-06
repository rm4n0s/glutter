package evdev

import (
	"strconv"
	"strings"
)

// A device is looked at through sysfs before it is opened, so that no keyboard or mouse is opened (SPEC §28.2): the kernel
// tells what a device can do in /sys/class/input/eventN/device/capabilities/*, which any user can read, and the node in /dev/input
// is opened only when the device is a pad.

// parseCapabilities reads a capabilities file of sysfs: hexadecimal words separated by spaces, the highest word first, each of
// the size of the machine's unsigned long (the files print it without leading zeros).
func parseCapabilities(s string) bits {
	words := strings.Fields(s)
	var b bits
	wordBits := strconv.IntSize
	for i, w := range words {
		v, err := strconv.ParseUint(w, 16, 64)
		if err != nil {
			continue
		}
		base := (len(words) - 1 - i) * wordBits
		for bit := 0; bit < wordBits; bit++ {
			if v>>bit&1 != 0 {
				b = b.set(base + bit)
			}
		}
	}
	return b
}

// parseHex16 reads the 4 hexadecimal digits of the id files of sysfs (bustype, vendor, product, version).
func parseHex16(s string) (uint16, bool) {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 16, 16)
	return uint16(v), err == nil
}

// sysfsFiles are the files of sysfs that describe a device, by the directory of the device under /sys/class/input/eventN.
type sysfsFiles interface {
	// read returns the content of the file under the device's directory, e.g. "device/capabilities/key".
	read(rel string) (string, error)
}

// probe reads the description of a device from sysfs. ok is false when a file is missing: the device may be too new for sysfs
// to have its attributes, and the caller tries again.
func probe(f sysfsFiles) (c caps, ok bool) {
	get := func(rel string) (string, bool) {
		s, err := f.read(rel)
		return s, err == nil
	}
	var parts [4]uint16
	for i, name := range []string{"bustype", "vendor", "product", "version"} {
		s, ok := get("device/id/" + name)
		if !ok {
			return c, false
		}
		if parts[i], ok = parseHex16(s); !ok {
			return c, false
		}
	}
	c.ID = inputID{Bus: parts[0], Vendor: parts[1], Product: parts[2], Version: parts[3]}
	name, ok := get("device/name")
	if !ok {
		return c, false
	}
	c.Name = strings.TrimSpace(name)
	for _, p := range []struct {
		file string
		dst  *bits
	}{{"key", &c.Key}, {"abs", &c.Abs}, {"rel", &c.Rel}, {"ff", &c.FF}} {
		s, ok := get("device/capabilities/" + p.file)
		if !ok {
			return c, false
		}
		*p.dst = parseCapabilities(s)
	}
	// The properties are a newer file: a kernel without it has none.
	if s, ok := get("device/properties"); ok {
		c.Prop = parseCapabilities(s)
	}
	return c, true
}
