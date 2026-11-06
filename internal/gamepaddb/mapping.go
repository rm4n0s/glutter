package gamepaddb

import (
	"os"
	"strconv"
	"strings"

	"github.com/rm4n0s/errors"
)

// This file is translated to Go and changed from src/joystick/SDL_gamepad.c of SDL 3 (zlib license, third_party/sdl/LICENSE):
// the format of a mapping line, its parser, and the table of mappings with the order in which it matches a pad.

// Button is a button of the standard layout. Its numbers are the order of the names in a mapping line.
//
// Flutter: no counterpart
type Button uint8

// The buttons of the standard layout.
const (
	// South is the bottom face button (A on an Xbox pad, cross on a PlayStation one).
	//
	// Flutter: no counterpart
	South Button = iota
	// East is the right face button (B, circle).
	//
	// Flutter: no counterpart
	East
	// West is the left face button (X, square).
	//
	// Flutter: no counterpart
	West
	// North is the top face button (Y, triangle).
	//
	// Flutter: no counterpart
	North
	// Back is the button on the left of the middle (View, Select, Share).
	//
	// Flutter: no counterpart
	Back
	// Guide is the button in the middle (Xbox, PS, Home).
	//
	// Flutter: no counterpart
	Guide
	// Start is the button on the right of the middle (Menu, Options).
	//
	// Flutter: no counterpart
	Start
	// LeftStick is the press of the left stick.
	//
	// Flutter: no counterpart
	LeftStick
	// RightStick is the press of the right stick.
	//
	// Flutter: no counterpart
	RightStick
	// LeftShoulder is the left bumper.
	//
	// Flutter: no counterpart
	LeftShoulder
	// RightShoulder is the right bumper.
	//
	// Flutter: no counterpart
	RightShoulder
	// DPadUp is up on the D-pad.
	//
	// Flutter: no counterpart
	DPadUp
	// DPadDown is down on the D-pad.
	//
	// Flutter: no counterpart
	DPadDown
	// DPadLeft is left on the D-pad.
	//
	// Flutter: no counterpart
	DPadLeft
	// DPadRight is right on the D-pad.
	//
	// Flutter: no counterpart
	DPadRight
	// Misc1 is an extra button (Share on an Xbox Series pad, the microphone button on a DualSense).
	//
	// Flutter: no counterpart
	Misc1
	// Paddle1 is the upper right paddle on the back.
	//
	// Flutter: no counterpart
	Paddle1
	// Paddle2 is the upper left paddle on the back.
	//
	// Flutter: no counterpart
	Paddle2
	// Paddle3 is the lower right paddle on the back.
	//
	// Flutter: no counterpart
	Paddle3
	// Paddle4 is the lower left paddle on the back.
	//
	// Flutter: no counterpart
	Paddle4
	// Touchpad is the press of the touchpad.
	//
	// Flutter: no counterpart
	Touchpad
	// Misc2 is another extra button.
	//
	// Flutter: no counterpart
	Misc2
	// Misc3 is another extra button.
	//
	// Flutter: no counterpart
	Misc3
	// Misc4 is another extra button.
	//
	// Flutter: no counterpart
	Misc4
	// Misc5 is another extra button.
	//
	// Flutter: no counterpart
	Misc5
	// Misc6 is another extra button.
	//
	// Flutter: no counterpart
	Misc6
	// ButtonCount is the number of buttons.
	//
	// Flutter: no counterpart
	ButtonCount
)

// Axis is an axis of the standard layout.
//
// Flutter: no counterpart
type Axis uint8

// The axes of the standard layout.
const (
	// LeftX is the left stick from left (−32768) to right (32767).
	//
	// Flutter: no counterpart
	LeftX Axis = iota
	// LeftY is the left stick from up (−32768) to down (32767).
	//
	// Flutter: no counterpart
	LeftY
	// RightX is the right stick from left to right.
	//
	// Flutter: no counterpart
	RightX
	// RightY is the right stick from up to down.
	//
	// Flutter: no counterpart
	RightY
	// LeftTrigger is the left trigger from 0 to 32767.
	//
	// Flutter: no counterpart
	LeftTrigger
	// RightTrigger is the right trigger from 0 to 32767.
	//
	// Flutter: no counterpart
	RightTrigger
	// AxisCount is the number of axes.
	//
	// Flutter: no counterpart
	AxisCount
)

// The range of a raw axis and of an axis of the layout.
const (
	// AxisMin is the smallest value of an axis.
	//
	// Flutter: no counterpart
	AxisMin = -32768
	// AxisMax is the largest value of an axis.
	//
	// Flutter: no counterpart
	AxisMax = 32767
)

// The positions of a hat, as the bits of a mask.
const (
	// HatCentered is a hat that is not pushed.
	//
	// Flutter: no counterpart
	HatCentered = 0x00
	// HatUp is a hat pushed up.
	//
	// Flutter: no counterpart
	HatUp = 0x01
	// HatRight is a hat pushed right.
	//
	// Flutter: no counterpart
	HatRight = 0x02
	// HatDown is a hat pushed down.
	//
	// Flutter: no counterpart
	HatDown = 0x04
	// HatLeft is a hat pushed left.
	//
	// Flutter: no counterpart
	HatLeft = 0x08
)

// buttonNames are the names of the buttons in a mapping line.
var buttonNames = [ButtonCount]string{
	"a", "b", "x", "y", "back", "guide", "start", "leftstick", "rightstick", "leftshoulder", "rightshoulder",
	"dpup", "dpdown", "dpleft", "dpright", "misc1", "paddle1", "paddle2", "paddle3", "paddle4", "touchpad",
	"misc2", "misc3", "misc4", "misc5", "misc6",
}

// axisNames are the names of the axes in a mapping line.
var axisNames = [AxisCount]string{"leftx", "lefty", "rightx", "righty", "lefttrigger", "righttrigger"}

// Name is the name of the button in a mapping line.
func (b Button) Name() string {
	if b < ButtonCount {
		return buttonNames[b]
	}
	return ""
}

// Name is the name of the axis in a mapping line.
func (a Axis) Name() string {
	if a < AxisCount {
		return axisNames[a]
	}
	return ""
}

func buttonFromName(s string) (Button, bool) {
	for i, n := range buttonNames {
		if strings.EqualFold(s, n) {
			return Button(i), true
		}
	}
	return 0, false
}

func axisFromName(s string) (Axis, bool) {
	s = strings.TrimLeft(s, "+-")
	for i, n := range axisNames {
		if strings.EqualFold(s, n) {
			return Axis(i), true
		}
	}
	return 0, false
}

// BindKind says what a binding reads from the pad or writes to the layout.
type BindKind uint8

const (
	bindNone BindKind = iota
	bindButton
	bindAxis
	bindHat
)

// Binding connects one input of a pad to one output of the layout. It is a comparable value.
//
// Flutter: no counterpart
type Binding struct {
	In  BindKind
	Out BindKind
	// InIndex is the number of the pad's button, axis or hat; HatMask the bit of the hat; InMin and InMax the values of the
	// axis that are the ends of the binding (InMin > InMax for an inverted axis, and one of them 0 for a half axis).
	InIndex      int
	HatMask      uint8
	InMin, InMax int
	// OutButton or OutAxis is the output; OutMin and OutMax the values that the ends of an axis give.
	OutButton      Button
	OutAxis        Axis
	OutMin, OutMax int
}

// ParseBindings reads the part of a mapping line that follows the name ("a:b0,b:b1,leftx:a0,type:xbox360,...") into bindings.
// Fields that are not buttons or axes (crc:, type:, face:, platform:, hint:, sdk>=:) and names this layout does not have are
// skipped, as SDL does. A name that is too long is an error.
//
// Flutter: no counterpart
func ParseBindings(text string) ([]Binding, error) {
	var out []Binding
	var game, joy strings.Builder
	inGame := true
	flush := func() {
		if b, ok := parseElement(game.String(), joy.String()); ok {
			for _, o := range out {
				if o == b {
					game.Reset()
					joy.Reset()
					return
				}
			}
			out = append(out, b)
		}
		game.Reset()
		joy.Reset()
	}
	for i := 0; i < len(text); i++ {
		switch c := text[i]; {
		case c == ':':
			inGame = false
			// The joystick side restarts at each colon, as in SDL, which resets its index there.
			joy.Reset()
		case c == ' ':
		case c == ',':
			inGame = true
			flush()
		case inGame:
			if game.Len() >= 19 {
				return nil, errors.New(TagMappingInvalid, "a button name of the mapping is too long", "name", game.String())
			}
			game.WriteByte(c)
		default:
			if joy.Len() >= 127 {
				return nil, errors.New(TagMappingInvalid, "an input name of the mapping is too long", "name", joy.String())
			}
			joy.WriteByte(c)
		}
	}
	if game.Len() > 0 || joy.Len() > 0 {
		flush()
	}
	return out, nil
}

// parseElement reads one "name:input" pair.
func parseElement(gameName, joyName string) (Binding, bool) {
	var b Binding
	var halfOut, halfIn byte
	if gameName != "" && (gameName[0] == '+' || gameName[0] == '-') {
		halfOut = gameName[0]
		gameName = gameName[1:]
	}
	axis, isAxis := axisFromName(gameName)
	button, isButton := buttonFromName(gameName)
	switch {
	case isAxis:
		b.Out = bindAxis
		b.OutAxis = axis
		switch {
		case axis == LeftTrigger || axis == RightTrigger:
			b.OutMin, b.OutMax = 0, AxisMax
		case halfOut == '+':
			b.OutMin, b.OutMax = 0, AxisMax
		case halfOut == '-':
			b.OutMin, b.OutMax = 0, AxisMin
		default:
			b.OutMin, b.OutMax = AxisMin, AxisMax
		}
	case isButton:
		b.Out = bindButton
		b.OutButton = button
	default:
		return Binding{}, false
	}

	if joyName != "" && (joyName[0] == '+' || joyName[0] == '-') {
		halfIn = joyName[0]
		joyName = joyName[1:]
	}
	invert := joyName != "" && joyName[len(joyName)-1] == '~'
	switch {
	case len(joyName) > 1 && joyName[0] == 'a' && isDigit(joyName[1]):
		b.In = bindAxis
		b.InIndex = atoi(joyName[1:])
		switch halfIn {
		case '+':
			b.InMin, b.InMax = 0, AxisMax
		case '-':
			b.InMin, b.InMax = 0, AxisMin
		default:
			b.InMin, b.InMax = AxisMin, AxisMax
		}
		if invert {
			b.InMin, b.InMax = b.InMax, b.InMin
		}
	case len(joyName) > 1 && joyName[0] == 'b' && isDigit(joyName[1]):
		b.In = bindButton
		b.InIndex = atoi(joyName[1:])
	case len(joyName) > 3 && joyName[0] == 'h' && isDigit(joyName[1]) && joyName[2] == '.' && isDigit(joyName[3]):
		b.In = bindHat
		b.InIndex = atoi(joyName[1:])
		b.HatMask = uint8(atoi(joyName[3:]))
	default:
		return Binding{}, false
	}
	return b, true
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// atoi reads the digits at the start of s, as C's atoi does.
func atoi(s string) int {
	n := 0
	for i := 0; i < len(s) && isDigit(s[i]); i++ {
		n = n*10 + int(s[i]-'0')
		if n > 1<<20 {
			break
		}
	}
	return n
}

// Priority says where a mapping came from; a mapping replaces one of the same GUID only when its priority is the same or higher.
//
// Flutter: no counterpart
type Priority uint8

const (
	// PriorityDefault is a mapping of the built-in table or one made from what the pad reports.
	//
	// Flutter: no counterpart
	PriorityDefault Priority = iota
	// PriorityAPI is a mapping that the app added with AddMappings.
	//
	// Flutter: no counterpart
	PriorityAPI
	// PriorityUser is a mapping that the user gave (an environment variable).
	//
	// Flutter: no counterpart
	PriorityUser
)

// Mapping is a line of the table: the pad it is for and the text that says which input is which button.
//
// Flutter: no counterpart
type Mapping struct {
	// GUID names the pad; its CRC is kept in Text (crc:) and is zero here.
	GUID GUID
	// Name is the name that the mapping gives the pad.
	Name string
	// Text is the part after the name: "a:b0,b:b1,...".
	Text     string
	Priority Priority
}

// Field returns the value of the field name in the mapping text ("type" gives "xbox360" for "type:xbox360"), or "".
//
// Flutter: no counterpart
func (m *Mapping) Field(name string) string {
	v, _ := field(m.Text, name)
	return v
}

// field finds ",name:" or a leading "name:" in a mapping text.
func field(text, name string) (string, bool) {
	key := name + ":"
	i := -1
	if strings.HasPrefix(text, key) {
		i = 0
	} else if j := strings.Index(text, ","+key); j >= 0 {
		i = j + 1
	}
	if i < 0 {
		return "", false
	}
	v := text[i+len(key):]
	if c := strings.IndexByte(v, ','); c >= 0 {
		v = v[:c]
	}
	return v, true
}

// Kind returns the kind the mapping's type: field names, or KindUnknown.
//
// Flutter: no counterpart
func (m *Mapping) Kind() Kind { return KindFromString(m.Field("type")) }

// FaceStyle returns the style of face buttons the mapping's face: field names, then the hints the community's files use, or
// FaceUnknown.
//
// Flutter: no counterpart
func (m *Mapping) FaceStyle() FaceStyle {
	if s := faceStyleFromString(m.Field("face")); s != FaceUnknown {
		return s
	}
	if strings.Contains(m.Text, "SDL_GAMECONTROLLER_USE_GAMECUBE_LABELS") {
		return FaceAXBY
	}
	if strings.Contains(m.Text, "SDL_GAMECONTROLLER_USE_BUTTON_LABELS") {
		return FaceBAYX
	}
	return FaceUnknown
}

// crc returns the CRC the mapping's crc: field names, and whether it has one.
func (m *Mapping) crc() (uint16, bool) {
	v, ok := field(m.Text, "crc")
	if !ok {
		return 0, false
	}
	n, _ := strconv.ParseUint(leadingHex(v), 16, 16)
	return uint16(n), true
}

func leadingHex(s string) string {
	i := 0
	for i < len(s) && (isDigit(s[i]) || s[i] >= 'a' && s[i] <= 'f' || s[i] >= 'A' && s[i] <= 'F') {
		i++
	}
	return s[:i]
}

// Table is the set of mappings known to the program, in the order they were added. It is not safe for use from several
// goroutines; the pad manager owns one and uses it on the UI thread.
//
// Flutter: no counterpart
type Table struct {
	mappings []*Mapping
	def      *Mapping
	xinput   *Mapping
	// SDKVersion is the version of Android that the program runs on (0 elsewhere); the mappings with sdk>= and sdk<= fields
	// that it does not satisfy are skipped.
	SDKVersion int
	// Getenv reads the environment variable that a hint: field names; nil means os.Getenv.
	Getenv func(string) string
}

// NewTable makes a table with the mappings of SDL's built-in table that belong to the given sections (see Builtin).
//
// Flutter: no counterpart
func NewTable(sections ...[]string) *Table {
	t := &Table{}
	for _, s := range sections {
		for _, line := range s {
			// The lines of the built-in table are valid; one that is not is left out, as SDL does.
			_, _ = t.add(line, PriorityDefault)
		}
	}
	return t
}

// Len returns the number of mappings.
//
// Flutter: no counterpart
func (t *Table) Len() int { return len(t.mappings) }

// Mappings returns the mappings in the order they were added.
//
// Flutter: no counterpart
func (t *Table) Mappings() []*Mapping { return t.mappings }

// Default returns the mapping that a line with the GUID "default" gave, or nil.
//
// Flutter: no counterpart
func (t *Table) Default() *Mapping { return t.def }

// Add adds the mapping line "GUID,name,a:b0,b:b1,..." with the priority of an app's mapping. It returns 1 when the mapping is
// new, 0 when it replaced or kept an old one for the same pad (or when its hint: field turned it off), and an error tagged
// GamepadMappingInvalid when the line cannot be read.
//
// Flutter: no counterpart
func (t *Table) Add(line string) (int, error) { return t.add(line, PriorityAPI) }

// AddText adds the mappings of a text in the community's format (one line each; lines with a platform: field that is not
// platform are skipped, and lines without one are skipped too, as in SDL's gamecontrollerdb.txt). It returns the number of
// new mappings.
//
// Flutter: no counterpart
func (t *Table) AddText(text, platform string) (int, error) {
	n := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		i := strings.Index(line, "platform:")
		if i < 0 {
			continue
		}
		v := line[i+len("platform:"):]
		c := strings.IndexByte(v, ',')
		if c < 0 {
			continue
		}
		v = v[:c]
		if !strings.EqualFold(v, platform) {
			continue
		}
		added, err := t.add(line, PriorityAPI)
		if err != nil {
			return n, err
		}
		n += added
	}
	return n, nil
}

func (t *Table) add(line string, priority Priority) (int, error) {
	first := strings.IndexByte(line, ',')
	if first < 0 {
		return 0, errors.New(TagMappingInvalid, "the mapping has no GUID", "line", line)
	}
	guidText := line[:first]
	isDefault := strings.EqualFold(guidText, "default")
	isXInput := strings.EqualFold(guidText, "xinput")
	guid := ParseGUID(guidText)
	if second := strings.IndexByte(line[first+1:], ','); second < 0 {
		return 0, errors.New(TagMappingInvalid, "the mapping has no name", "line", line)
	}

	if info := guid.Info(); IsGameCube(info.Vendor, info.Product) && !strings.Contains(line, "SDL_GAMECONTROLLER_USE_GAMECUBE_LABELS") {
		line += "hint:SDL_GAMECONTROLLER_USE_GAMECUBE_LABELS:=1,"
	}

	if i := strings.Index(line, "hint:"); i >= 0 {
		skip, converted := t.applyHint(line, i)
		if skip {
			return 0, nil
		}
		line = converted
	}

	if t.SDKVersion != 0 {
		if v, ok := field(line, "sdk>="); ok && t.SDKVersion < atoi(v) {
			return 0, errors.New(TagMappingInvalid, "the mapping is for a newer version of Android", "line", line)
		}
		if v, ok := field(line, "sdk<="); ok && t.SDKVersion > atoi(v) {
			return 0, errors.New(TagMappingInvalid, "the mapping is for an older version of Android", "line", line)
		}
	}

	first = strings.IndexByte(line, ',')
	second := first + 1 + strings.IndexByte(line[first+1:], ',')
	name := line[first+1 : second]
	text := strings.TrimSpace(strings.TrimLeft(line[second+1:], " \t\r\n\v\f"))
	if _, err := ParseBindings(text); err != nil {
		return 0, err
	}

	// Fix up the GUID and the mapping with the CRC, as SDL does.
	crc := guid.Info().CRC
	if crc != 0 {
		var before, after string
		if i := strings.Index(text, "crc:"); i >= 0 {
			before = text[:i]
			if c := strings.IndexByte(text[i:], ','); c >= 0 {
				after = text[i+c+1:]
			}
		} else {
			before = text
		}
		comma := ","
		if before == "" || strings.HasSuffix(before, ",") {
			comma = ""
		}
		text = before + comma + "crc:" + hex4(crc) + "," + after
		guid = guid.WithCRC(0)
	} else if v, ok := field(text, "crc"); ok {
		if n, _ := strconv.ParseUint(leadingHex(v), 16, 16); n != 0 {
			guid = guid.WithCRC(uint16(n))
		}
	}

	if old := t.find(guid, true); old != nil {
		if old.Priority <= priority {
			old.Name, old.Text, old.Priority = name, text, priority
		}
		return 0, nil
	}
	m := &Mapping{GUID: guid.WithCRC(0), Name: name, Text: text, Priority: priority}
	t.mappings = append(t.mappings, m)
	switch {
	case isDefault:
		t.def = m
	case isXInput:
		t.xinput = m
	}
	return 1, nil
}

// applyHint reads the hint: field at index i of the line. It returns skip when the hint turns the mapping off, and the line
// with the face buttons put in their positions when the hint is about labels.
func (t *Table) applyHint(line string, i int) (skip bool, converted string) {
	rest := line[i+len("hint:"):]
	negate := false
	if strings.HasPrefix(rest, "!") {
		negate = true
		rest = rest[1:]
	}
	end := strings.IndexAny(rest, ",:")
	if end < 0 {
		end = len(rest)
	}
	hint := rest[:end]
	rest = rest[end:]
	def := false
	if strings.HasPrefix(rest, ":=") {
		def = atoi(rest[2:]) != 0
	}
	switch hint {
	case "SDL_GAMECONTROLLER_USE_GAMECUBE_LABELS":
		if negate {
			return false, line
		}
		return false, positional(line, hint, [][2]string{{",b:", "x"}, {",x:", "b"}})
	case "SDL_GAMECONTROLLER_USE_BUTTON_LABELS":
		if negate {
			return false, line
		}
		return false, positional(line, hint, [][2]string{{",a:", "b"}, {",b:", "a"}, {",x:", "y"}, {",y:", "x"}})
	}
	getenv := t.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	value := def
	if v := getenv(hint); v != "" {
		value = v != "0" && !strings.EqualFold(v, "false")
	}
	if negate {
		value = !value
	}
	return !value, line
}

// positional rewrites the first occurrence of each ",x:" key to the name in the pair, all at once on the original line, and
// puts a '!' in front of the hint so that the line is not converted again.
func positional(line, hint string, swaps [][2]string) string {
	b := []byte(line)
	for _, s := range swaps {
		if i := strings.Index(line, s[0]); i >= 0 {
			b[i+1] = s[1][0]
		}
	}
	out := string(b)
	return strings.Replace(out, "hint:"+hint, "hint:!"+hint, 1)
}

// find returns the mapping for a GUID by SDL's order: an exact match of the version and the CRC, then without the CRC, then
// without the version, and (when adding is false) the XInput mapping for an XInput pad. adding asks only for an exact match.
func (t *Table) find(guid GUID, adding bool) *Mapping {
	if m := t.match(guid, true, true); m != nil {
		return m
	}
	if adding {
		return nil
	}
	if m := t.match(guid, true, false); m != nil {
		return m
	}
	if guid.usesVersion() {
		if m := t.match(guid, false, true); m != nil {
			return m
		}
		if m := t.match(guid, false, false); m != nil {
			return m
		}
	}
	return nil
}

// Find returns the mapping for a pad's GUID, or nil. A pad that speaks XInput gets the XInput mapping when none matches.
//
// Flutter: no counterpart
func (t *Table) Find(guid GUID) *Mapping {
	if m := t.find(guid, false); m != nil {
		return m
	}
	if guid.isXInput() {
		return t.xinput
	}
	return nil
}

func (t *Table) match(guid GUID, matchVersion, exactCRC bool) *Mapping {
	crc := guid.Info().CRC
	guid = guid.WithCRC(0)
	if !matchVersion {
		guid = guid.WithVersion(0)
	}
	var best *Mapping
	for _, m := range t.mappings {
		if m.GUID == (GUID{}) {
			continue
		}
		g := m.GUID
		if !matchVersion {
			g = g.WithVersion(0)
		}
		if g != guid {
			continue
		}
		if mc, ok := m.crc(); ok {
			if mc != crc {
				continue
			}
			return m
		} else if crc != 0 && exactCRC {
			continue
		}
		if best == nil {
			best = m
		}
	}
	return best
}

// SetAuto adds the mapping that a backend made from what a pad reports (its text only: "a:b0,..."), under the pad's GUID and
// name, with the lowest priority, and returns it. A mapping that is there already for the GUID is kept.
//
// Flutter: no counterpart
func (t *Table) SetAuto(guid GUID, name, text string) *Mapping {
	guid = guid.WithCRC(0)
	name = strings.ReplaceAll(name, ",", " ")
	if len(name) > 127 {
		name = name[:127]
	}
	if m := t.find(guid, true); m != nil {
		return m
	}
	m := &Mapping{GUID: guid, Name: name, Text: text, Priority: PriorityDefault}
	t.mappings = append(t.mappings, m)
	return m
}

// TagMappingInvalid is the tag of the error of a mapping line that cannot be read.
//
// Flutter: no counterpart
const TagMappingInvalid = "GamepadMappingInvalid"
