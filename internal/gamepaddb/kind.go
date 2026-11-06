package gamepaddb

import (
	"sort"
	"strings"
)

// This file is translated to Go and changed from src/joystick/SDL_joystick.c (the kind of a pad from its vendor and product),
// src/joystick/controller_type.c and src/joystick/usb_ids.h of SDL 3 (zlib license, third_party/sdl/LICENSE).

// Kind is the model of pad that a vendor and product number say it is; it picks the picture of the buttons an app shows, and
// the names of the face buttons.
//
// Flutter: no counterpart
type Kind uint8

// The kinds of pads.
const (
	// KindUnknown is a pad whose kind is not known; a mapping's type: field is not set either.
	//
	// Flutter: no counterpart
	KindUnknown Kind = iota
	// KindStandard is a pad with the standard layout and no known model.
	//
	// Flutter: no counterpart
	KindStandard
	// KindXbox360 is an Xbox 360 pad.
	//
	// Flutter: no counterpart
	KindXbox360
	// KindXboxOne is an Xbox One, Xbox Series or Elite pad.
	//
	// Flutter: no counterpart
	KindXboxOne
	// KindPS3 is a PlayStation 3 pad.
	//
	// Flutter: no counterpart
	KindPS3
	// KindPS4 is a PlayStation 4 pad (DualShock 4).
	//
	// Flutter: no counterpart
	KindPS4
	// KindPS5 is a PlayStation 5 pad (DualSense).
	//
	// Flutter: no counterpart
	KindPS5
	// KindSwitchPro is a Nintendo Switch Pro pad or one that copies it.
	//
	// Flutter: no counterpart
	KindSwitchPro
	// KindJoyConLeft is a left Joy-Con on its own.
	//
	// Flutter: no counterpart
	KindJoyConLeft
	// KindJoyConRight is a right Joy-Con on its own.
	//
	// Flutter: no counterpart
	KindJoyConRight
	// KindJoyConPair is a pair of Joy-Cons used as one pad.
	//
	// Flutter: no counterpart
	KindJoyConPair
	// KindGameCube is a GameCube pad.
	//
	// Flutter: no counterpart
	KindGameCube
	// KindSteam is a Steam pad.
	//
	// Flutter: no counterpart
	KindSteam
	kindCount
)

// kindNames are the values of the type: field of a mapping, in the order of the kinds.
var kindNames = [kindCount]string{"unknown", "standard", "xbox360", "xboxone", "ps3", "ps4", "ps5", "switchpro", "joyconleft", "joyconright", "joyconpair", "gamecube", "steam"}

// String is the name of the kind as a mapping's type: field spells it.
func (k Kind) String() string {
	if k < kindCount {
		return kindNames[k]
	}
	return "unknown"
}

// KindFromString reads the type: field of a mapping; an unknown name is KindUnknown.
//
// Flutter: no counterpart
func KindFromString(s string) Kind {
	s = strings.TrimLeft(s, "+-")
	for i, n := range kindNames {
		if strings.EqualFold(s, n) {
			return Kind(i)
		}
	}
	return KindUnknown
}

// The vendor and product numbers that decide a kind by themselves.
const (
	vendorNintendo             = 0x057e
	productJoyConGrip          = 0x200e
	productJoyConLeft          = 0x2006
	productJoyConPair          = 0x2008
	productJoyConRight         = 0x2007
	productSwitch2JoyConLeft   = 0x2067
	productSwitch2JoyConPair   = 0x2068
	productSwitch2JoyConRight  = 0x2066
	productSwitch2GameCubePad  = 0x2073
	productNintendoGameCubeAda = 0x0337
)

// gameCubeDevices are the adapters and pads that SDL knows as GameCube pads (vendor and product).
var gameCubeDevices = [][2]uint16{
	{0x0079, 0x1843}, // DragonRise GameCube Controller Adapter
	{0x0079, 0x1844}, // DragonRise GameCube Controller Adapter
	{0x0079, 0x1846}, // DragonRise GameCube Controller Adapter
	{vendorNintendo, productNintendoGameCubeAda},
	{vendorNintendo, productSwitch2GameCubePad},
	{0x05e3, 0x0681}, // Austgame GameCube to USB convertor
	{0x0926, 0x8888}, // Cyber Gadget GameCube Controller
	{0x0e6f, 0x0185}, // PDP Wired Fight Pad Pro for Nintendo Switch
	{0x1a34, 0xf705}, // GameCube {HuiJia USB box}
	{0x20d6, 0xa711}, // PowerA Wired Controller Nintendo GameCube Style
}

// IsGameCube says whether the vendor and product are a GameCube pad or adapter.
//
// Flutter: no counterpart
func IsGameCube(vendor, product uint16) bool {
	for _, d := range gameCubeDevices {
		if d[0] == vendor && d[1] == product {
			return true
		}
	}
	return false
}

// Lookup finds the entry of the controller list for a vendor and product.
//
// Flutter: no counterpart
func Lookup(vendor, product uint16) (Controller, bool) {
	key := uint32(vendor)<<16 | uint32(product)
	i := sort.Search(len(Controllers), func(i int) bool {
		return uint32(Controllers[i].Vendor)<<16|uint32(Controllers[i].Product) >= key
	})
	if i < len(Controllers) && Controllers[i].Vendor == vendor && Controllers[i].Product == product {
		return Controllers[i], true
	}
	return Controller{}, false
}

// ControllerName returns the name that the controller list gives a vendor and product ("" when it gives none).
//
// Flutter: no counterpart
func ControllerName(vendor, product uint16) string {
	c, _ := Lookup(vendor, product)
	return c.Name
}

// KindOf says what kind of pad a vendor and product number are. A few pads are known by their name alone (vendor and product
// zero), and the Joy-Cons by their names too. forUI asks for the kind to show to the user: a GameCube adapter is then a GameCube
// pad, and a PlayStation 4 pad that speaks XInput is a PlayStation 4 pad, where for the mapping it is a standard one.
//
// Flutter: no counterpart
func KindOf(vendor, product uint16, name string, forUI bool) Kind {
	switch {
	case vendor == 0 && product == 0:
		// Some devices are only identifiable by their name: clones of the Switch Pro pad.
		if name == "Lic Pro Controller" || name == "Nintendo Wireless Gamepad" || name == "Wireless Gamepad" {
			return KindSwitchPro
		}
		return KindStandard
	case vendor == 1 && product == 1:
		return KindStandard
	case vendor == vendorNintendo && (product == productJoyConLeft || product == productSwitch2JoyConLeft):
		return KindJoyConLeft
	case vendor == vendorNintendo && (product == productJoyConRight || product == productSwitch2JoyConRight):
		if strings.Contains(name, "NES Controller") {
			// There is no kind for the Nintendo Online NES pad.
			return KindStandard
		}
		return KindJoyConRight
	case vendor == vendorNintendo && product == productJoyConGrip:
		switch {
		case strings.Contains(name, "(L)"):
			return KindJoyConLeft
		case strings.Contains(name, "L+R"):
			return KindJoyConPair
		}
		return KindJoyConRight
	case vendor == vendorNintendo && (product == productJoyConPair || product == productSwitch2JoyConPair):
		return KindJoyConPair
	case forUI && IsGameCube(vendor, product):
		return KindGameCube
	}
	c, _ := Lookup(vendor, product)
	switch c.Type {
	case ControllerXBox360Controller:
		return KindXbox360
	case ControllerXBoxOneController, ControllerXBoxEliteController:
		return KindXboxOne
	case ControllerPS3Controller:
		return KindPS3
	case ControllerPS4Controller:
		return KindPS4
	case ControllerPS5Controller, ControllerPS5EdgeController:
		return KindPS5
	case ControllerXInputPS4Controller:
		if forUI {
			return KindPS4
		}
	case ControllerSwitchProController, ControllerSwitchInputOnlyController:
		return KindSwitchPro
	case ControllerXInputSwitchController:
		if forUI {
			return KindSwitchPro
		}
		return KindXbox360
	case ControllerSteamController, ControllerSteamControllerV2, ControllerSteamControllerNeptune, ControllerSteamControllerTriton,
		ControllerHoriSteamController:
		return KindSteam
	}
	return KindStandard
}

// KindOfGUID is KindOf for the vendor and product of a GUID, as the user sees it: a pad that speaks XInput and is of no known
// model is an Xbox One pad.
//
// Flutter: no counterpart
func KindOfGUID(g GUID, name string) Kind {
	i := g.Info()
	k := KindOf(i.Vendor, i.Product, name, true)
	if k == KindStandard && g.isXInput() {
		return KindXboxOne
	}
	return k
}

// FaceStyle is how the four face buttons are labelled on a pad.
//
// Flutter: no counterpart
type FaceStyle uint8

// The styles of face buttons.
const (
	// FaceUnknown is a style that is not known.
	//
	// Flutter: no counterpart
	FaceUnknown FaceStyle = iota
	// FaceABXY is A at the bottom, B on the right, X on the left, Y at the top (Xbox).
	//
	// Flutter: no counterpart
	FaceABXY
	// FaceAXBY is A at the bottom, X on the right, B on the left, Y at the top (GameCube).
	//
	// Flutter: no counterpart
	FaceAXBY
	// FaceBAYX is B at the bottom, A on the right, Y on the left, X at the top (Nintendo).
	//
	// Flutter: no counterpart
	FaceBAYX
	// FaceSony is cross at the bottom, circle on the right, square on the left, triangle at the top (PlayStation).
	//
	// Flutter: no counterpart
	FaceSony
)

// FaceStyleOf is the style of the face buttons of a kind of pad.
//
// Flutter: no counterpart
func FaceStyleOf(k Kind) FaceStyle {
	switch k {
	case KindPS3, KindPS4, KindPS5:
		return FaceSony
	case KindSwitchPro, KindJoyConLeft, KindJoyConRight, KindJoyConPair:
		return FaceBAYX
	case KindGameCube:
		return FaceAXBY
	}
	return FaceABXY
}

func faceStyleFromString(s string) FaceStyle {
	switch s {
	case "abxy":
		return FaceABXY
	case "axby":
		return FaceAXBY
	case "bayx":
		return FaceBAYX
	case "sony":
		return FaceSony
	}
	return FaceUnknown
}
