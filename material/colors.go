// Copyright 2014 The Flutter Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in third_party/flutter/LICENSE.
//
// This file is ported from the Flutter framework (see NOTICE).

package material

import (
	"github.com/rm4n0s/glutter/painting"
	"github.com/rm4n0s/glutter/ui"
)

// MaterialColor defines a single color as well as a color swatch with ten shades
// of the color. The color's shades are referred to by index: 50, 100, 200 to
// 900. Dart's MaterialColor extends Color; ui.Color is a plain value, so the
// primary color is the Primary field (SPEC §12.10).
//
// Flutter: material/colors.dart MaterialColor
type MaterialColor struct {
	*painting.ColorSwatch[int]
}

// NewMaterialColor creates a swatch of the primary color and the shades, which
// are given in order as pairs of the index and the color.
//
// Flutter: material/colors.dart MaterialColor.new
func NewMaterialColor(primary uint32, shades ...Shade) MaterialColor {
	return MaterialColor{newSwatch(primary, shades)}
}

// Shade is one color of a swatch: its index and its value.
//
// Flutter: material/colors.dart MaterialColor.new
type Shade struct {
	// Index is the index of the shade: 50, 100, 200...
	Index int
	// Value is the color as 0xAARRGGBB.
	Value uint32
}

func newSwatch(primary uint32, shades []Shade) *painting.ColorSwatch[int] {
	keys := make([]int, len(shades))
	colors := make(map[int]ui.Color, len(shades))
	for i, s := range shades {
		keys[i] = s.Index
		colors[s.Index] = ui.NewColor(s.Value)
	}
	return painting.NewColorSwatch(ui.NewColor(primary), keys, colors)
}

func (c MaterialColor) shade(index int) ui.Color {
	col, _ := c.At(index)
	return col
}

// Shade50 is the shade 50 of the swatch.
//
// Flutter: material/colors.dart MaterialColor.shade50
func (c MaterialColor) Shade50() ui.Color { return c.shade(50) }

// Shade100 is the shade 100 of the swatch.
//
// Flutter: material/colors.dart MaterialColor.shade100
func (c MaterialColor) Shade100() ui.Color { return c.shade(100) }

// Shade200 is the shade 200 of the swatch.
//
// Flutter: material/colors.dart MaterialColor.shade200
func (c MaterialColor) Shade200() ui.Color { return c.shade(200) }

// Shade300 is the shade 300 of the swatch.
//
// Flutter: material/colors.dart MaterialColor.shade300
func (c MaterialColor) Shade300() ui.Color { return c.shade(300) }

// Shade400 is the shade 400 of the swatch.
//
// Flutter: material/colors.dart MaterialColor.shade400
func (c MaterialColor) Shade400() ui.Color { return c.shade(400) }

// Shade500 is the shade 500 of the swatch.
//
// Flutter: material/colors.dart MaterialColor.shade500
func (c MaterialColor) Shade500() ui.Color { return c.shade(500) }

// Shade600 is the shade 600 of the swatch.
//
// Flutter: material/colors.dart MaterialColor.shade600
func (c MaterialColor) Shade600() ui.Color { return c.shade(600) }

// Shade700 is the shade 700 of the swatch.
//
// Flutter: material/colors.dart MaterialColor.shade700
func (c MaterialColor) Shade700() ui.Color { return c.shade(700) }

// Shade800 is the shade 800 of the swatch.
//
// Flutter: material/colors.dart MaterialColor.shade800
func (c MaterialColor) Shade800() ui.Color { return c.shade(800) }

// Shade900 is the shade 900 of the swatch.
//
// Flutter: material/colors.dart MaterialColor.shade900
func (c MaterialColor) Shade900() ui.Color { return c.shade(900) }

// MaterialAccentColor defines a single accent color as well as a color swatch
// with four shades of the color: 100, 200, 400 and 700.
//
// Flutter: material/colors.dart MaterialAccentColor
type MaterialAccentColor struct {
	*painting.ColorSwatch[int]
}

// NewMaterialAccentColor creates an accent swatch.
//
// Flutter: material/colors.dart MaterialAccentColor.new
func NewMaterialAccentColor(primary uint32, shades ...Shade) MaterialAccentColor {
	return MaterialAccentColor{newSwatch(primary, shades)}
}

func (c MaterialAccentColor) shade(index int) ui.Color {
	col, _ := c.At(index)
	return col
}

// Shade100 is the shade 100 of the swatch.
//
// Flutter: material/colors.dart MaterialAccentColor.shade100
func (c MaterialAccentColor) Shade100() ui.Color { return c.shade(100) }

// Shade200 is the shade 200 of the swatch.
//
// Flutter: material/colors.dart MaterialAccentColor.shade200
func (c MaterialAccentColor) Shade200() ui.Color { return c.shade(200) }

// Shade400 is the shade 400 of the swatch.
//
// Flutter: material/colors.dart MaterialAccentColor.shade400
func (c MaterialAccentColor) Shade400() ui.Color { return c.shade(400) }

// Shade700 is the shade 700 of the swatch.
//
// Flutter: material/colors.dart MaterialAccentColor.shade700
func (c MaterialAccentColor) Shade700() ui.Color { return c.shade(700) }

// ColorsType is the type of Colors: the colors and the swatches of the Material
// design color palette, with the names of Flutter's Colors class.
//
// Flutter: material/colors.dart Colors
type ColorsType struct {
	// Transparent is the color transparent.
	//
	// Flutter: material/colors.dart Colors.transparent
	Transparent ui.Color
	// Black is the color black.
	//
	// Flutter: material/colors.dart Colors.black
	Black ui.Color
	// Black87 is the color black87.
	//
	// Flutter: material/colors.dart Colors.black87
	Black87 ui.Color
	// Black54 is the color black54.
	//
	// Flutter: material/colors.dart Colors.black54
	Black54 ui.Color
	// Black45 is the color black45.
	//
	// Flutter: material/colors.dart Colors.black45
	Black45 ui.Color
	// Black38 is the color black38.
	//
	// Flutter: material/colors.dart Colors.black38
	Black38 ui.Color
	// Black26 is the color black26.
	//
	// Flutter: material/colors.dart Colors.black26
	Black26 ui.Color
	// Black12 is the color black12.
	//
	// Flutter: material/colors.dart Colors.black12
	Black12 ui.Color
	// White is the color white.
	//
	// Flutter: material/colors.dart Colors.white
	White ui.Color
	// White70 is the color white70.
	//
	// Flutter: material/colors.dart Colors.white70
	White70 ui.Color
	// White60 is the color white60.
	//
	// Flutter: material/colors.dart Colors.white60
	White60 ui.Color
	// White54 is the color white54.
	//
	// Flutter: material/colors.dart Colors.white54
	White54 ui.Color
	// White38 is the color white38.
	//
	// Flutter: material/colors.dart Colors.white38
	White38 ui.Color
	// White30 is the color white30.
	//
	// Flutter: material/colors.dart Colors.white30
	White30 ui.Color
	// White24 is the color white24.
	//
	// Flutter: material/colors.dart Colors.white24
	White24 ui.Color
	// White12 is the color white12.
	//
	// Flutter: material/colors.dart Colors.white12
	White12 ui.Color
	// White10 is the color white10.
	//
	// Flutter: material/colors.dart Colors.white10
	White10 ui.Color
	// Red is the swatch red.
	//
	// Flutter: material/colors.dart Colors.red
	Red MaterialColor
	// RedAccent is the swatch redAccent.
	//
	// Flutter: material/colors.dart Colors.redAccent
	RedAccent MaterialAccentColor
	// Pink is the swatch pink.
	//
	// Flutter: material/colors.dart Colors.pink
	Pink MaterialColor
	// PinkAccent is the swatch pinkAccent.
	//
	// Flutter: material/colors.dart Colors.pinkAccent
	PinkAccent MaterialAccentColor
	// Purple is the swatch purple.
	//
	// Flutter: material/colors.dart Colors.purple
	Purple MaterialColor
	// PurpleAccent is the swatch purpleAccent.
	//
	// Flutter: material/colors.dart Colors.purpleAccent
	PurpleAccent MaterialAccentColor
	// DeepPurple is the swatch deepPurple.
	//
	// Flutter: material/colors.dart Colors.deepPurple
	DeepPurple MaterialColor
	// DeepPurpleAccent is the swatch deepPurpleAccent.
	//
	// Flutter: material/colors.dart Colors.deepPurpleAccent
	DeepPurpleAccent MaterialAccentColor
	// Indigo is the swatch indigo.
	//
	// Flutter: material/colors.dart Colors.indigo
	Indigo MaterialColor
	// IndigoAccent is the swatch indigoAccent.
	//
	// Flutter: material/colors.dart Colors.indigoAccent
	IndigoAccent MaterialAccentColor
	// Blue is the swatch blue.
	//
	// Flutter: material/colors.dart Colors.blue
	Blue MaterialColor
	// BlueAccent is the swatch blueAccent.
	//
	// Flutter: material/colors.dart Colors.blueAccent
	BlueAccent MaterialAccentColor
	// LightBlue is the swatch lightBlue.
	//
	// Flutter: material/colors.dart Colors.lightBlue
	LightBlue MaterialColor
	// LightBlueAccent is the swatch lightBlueAccent.
	//
	// Flutter: material/colors.dart Colors.lightBlueAccent
	LightBlueAccent MaterialAccentColor
	// Cyan is the swatch cyan.
	//
	// Flutter: material/colors.dart Colors.cyan
	Cyan MaterialColor
	// CyanAccent is the swatch cyanAccent.
	//
	// Flutter: material/colors.dart Colors.cyanAccent
	CyanAccent MaterialAccentColor
	// Teal is the swatch teal.
	//
	// Flutter: material/colors.dart Colors.teal
	Teal MaterialColor
	// TealAccent is the swatch tealAccent.
	//
	// Flutter: material/colors.dart Colors.tealAccent
	TealAccent MaterialAccentColor
	// Green is the swatch green.
	//
	// Flutter: material/colors.dart Colors.green
	Green MaterialColor
	// GreenAccent is the swatch greenAccent.
	//
	// Flutter: material/colors.dart Colors.greenAccent
	GreenAccent MaterialAccentColor
	// LightGreen is the swatch lightGreen.
	//
	// Flutter: material/colors.dart Colors.lightGreen
	LightGreen MaterialColor
	// LightGreenAccent is the swatch lightGreenAccent.
	//
	// Flutter: material/colors.dart Colors.lightGreenAccent
	LightGreenAccent MaterialAccentColor
	// Lime is the swatch lime.
	//
	// Flutter: material/colors.dart Colors.lime
	Lime MaterialColor
	// LimeAccent is the swatch limeAccent.
	//
	// Flutter: material/colors.dart Colors.limeAccent
	LimeAccent MaterialAccentColor
	// Yellow is the swatch yellow.
	//
	// Flutter: material/colors.dart Colors.yellow
	Yellow MaterialColor
	// YellowAccent is the swatch yellowAccent.
	//
	// Flutter: material/colors.dart Colors.yellowAccent
	YellowAccent MaterialAccentColor
	// Amber is the swatch amber.
	//
	// Flutter: material/colors.dart Colors.amber
	Amber MaterialColor
	// AmberAccent is the swatch amberAccent.
	//
	// Flutter: material/colors.dart Colors.amberAccent
	AmberAccent MaterialAccentColor
	// Orange is the swatch orange.
	//
	// Flutter: material/colors.dart Colors.orange
	Orange MaterialColor
	// OrangeAccent is the swatch orangeAccent.
	//
	// Flutter: material/colors.dart Colors.orangeAccent
	OrangeAccent MaterialAccentColor
	// DeepOrange is the swatch deepOrange.
	//
	// Flutter: material/colors.dart Colors.deepOrange
	DeepOrange MaterialColor
	// DeepOrangeAccent is the swatch deepOrangeAccent.
	//
	// Flutter: material/colors.dart Colors.deepOrangeAccent
	DeepOrangeAccent MaterialAccentColor
	// Brown is the swatch brown.
	//
	// Flutter: material/colors.dart Colors.brown
	Brown MaterialColor
	// Grey is the swatch grey.
	//
	// Flutter: material/colors.dart Colors.grey
	Grey MaterialColor
	// BlueGrey is the swatch blueGrey.
	//
	// Flutter: material/colors.dart Colors.blueGrey
	BlueGrey MaterialColor
	// Primaries are the swatches that an app can pick a primary color from. The
	// grey swatch is left out on purpose: picking grey at random for an app makes
	// it look disabled.
	//
	// Flutter: material/colors.dart Colors.primaries
	Primaries []MaterialColor
	// Accents are the accent swatches.
	//
	// Flutter: material/colors.dart Colors.accents
	Accents []MaterialAccentColor
}

// Colors is the Material color palette. It is a read-only value.
//
// Flutter: material/colors.dart Colors
var Colors = newColors()

func newColors() ColorsType {
	c := ColorsType{
		Transparent:      ui.NewColor(0x00000000),
		Black:            ui.NewColor(0xFF000000),
		Black87:          ui.NewColor(0xDD000000),
		Black54:          ui.NewColor(0x8A000000),
		Black45:          ui.NewColor(0x73000000),
		Black38:          ui.NewColor(0x61000000),
		Black26:          ui.NewColor(0x42000000),
		Black12:          ui.NewColor(0x1F000000),
		White:            ui.NewColor(0xFFFFFFFF),
		White70:          ui.NewColor(0xB3FFFFFF),
		White60:          ui.NewColor(0x99FFFFFF),
		White54:          ui.NewColor(0x8AFFFFFF),
		White38:          ui.NewColor(0x62FFFFFF),
		White30:          ui.NewColor(0x4DFFFFFF),
		White24:          ui.NewColor(0x3DFFFFFF),
		White12:          ui.NewColor(0x1FFFFFFF),
		White10:          ui.NewColor(0x1AFFFFFF),
		Red:              NewMaterialColor(0xFFF44336, Shade{50, 0xFFFFEBEE}, Shade{100, 0xFFFFCDD2}, Shade{200, 0xFFEF9A9A}, Shade{300, 0xFFE57373}, Shade{400, 0xFFEF5350}, Shade{500, 0xFFF44336}, Shade{600, 0xFFE53935}, Shade{700, 0xFFD32F2F}, Shade{800, 0xFFC62828}, Shade{900, 0xFFB71C1C}),
		RedAccent:        NewMaterialAccentColor(0xFFFF5252, Shade{100, 0xFFFF8A80}, Shade{200, 0xFFFF5252}, Shade{400, 0xFFFF1744}, Shade{700, 0xFFD50000}),
		Pink:             NewMaterialColor(0xFFE91E63, Shade{50, 0xFFFCE4EC}, Shade{100, 0xFFF8BBD0}, Shade{200, 0xFFF48FB1}, Shade{300, 0xFFF06292}, Shade{400, 0xFFEC407A}, Shade{500, 0xFFE91E63}, Shade{600, 0xFFD81B60}, Shade{700, 0xFFC2185B}, Shade{800, 0xFFAD1457}, Shade{900, 0xFF880E4F}),
		PinkAccent:       NewMaterialAccentColor(0xFFFF4081, Shade{100, 0xFFFF80AB}, Shade{200, 0xFFFF4081}, Shade{400, 0xFFF50057}, Shade{700, 0xFFC51162}),
		Purple:           NewMaterialColor(0xFF9C27B0, Shade{50, 0xFFF3E5F5}, Shade{100, 0xFFE1BEE7}, Shade{200, 0xFFCE93D8}, Shade{300, 0xFFBA68C8}, Shade{400, 0xFFAB47BC}, Shade{500, 0xFF9C27B0}, Shade{600, 0xFF8E24AA}, Shade{700, 0xFF7B1FA2}, Shade{800, 0xFF6A1B9A}, Shade{900, 0xFF4A148C}),
		PurpleAccent:     NewMaterialAccentColor(0xFFE040FB, Shade{100, 0xFFEA80FC}, Shade{200, 0xFFE040FB}, Shade{400, 0xFFD500F9}, Shade{700, 0xFFAA00FF}),
		DeepPurple:       NewMaterialColor(0xFF673AB7, Shade{50, 0xFFEDE7F6}, Shade{100, 0xFFD1C4E9}, Shade{200, 0xFFB39DDB}, Shade{300, 0xFF9575CD}, Shade{400, 0xFF7E57C2}, Shade{500, 0xFF673AB7}, Shade{600, 0xFF5E35B1}, Shade{700, 0xFF512DA8}, Shade{800, 0xFF4527A0}, Shade{900, 0xFF311B92}),
		DeepPurpleAccent: NewMaterialAccentColor(0xFF7C4DFF, Shade{100, 0xFFB388FF}, Shade{200, 0xFF7C4DFF}, Shade{400, 0xFF651FFF}, Shade{700, 0xFF6200EA}),
		Indigo:           NewMaterialColor(0xFF3F51B5, Shade{50, 0xFFE8EAF6}, Shade{100, 0xFFC5CAE9}, Shade{200, 0xFF9FA8DA}, Shade{300, 0xFF7986CB}, Shade{400, 0xFF5C6BC0}, Shade{500, 0xFF3F51B5}, Shade{600, 0xFF3949AB}, Shade{700, 0xFF303F9F}, Shade{800, 0xFF283593}, Shade{900, 0xFF1A237E}),
		IndigoAccent:     NewMaterialAccentColor(0xFF536DFE, Shade{100, 0xFF8C9EFF}, Shade{200, 0xFF536DFE}, Shade{400, 0xFF3D5AFE}, Shade{700, 0xFF304FFE}),
		Blue:             NewMaterialColor(0xFF2196F3, Shade{50, 0xFFE3F2FD}, Shade{100, 0xFFBBDEFB}, Shade{200, 0xFF90CAF9}, Shade{300, 0xFF64B5F6}, Shade{400, 0xFF42A5F5}, Shade{500, 0xFF2196F3}, Shade{600, 0xFF1E88E5}, Shade{700, 0xFF1976D2}, Shade{800, 0xFF1565C0}, Shade{900, 0xFF0D47A1}),
		BlueAccent:       NewMaterialAccentColor(0xFF448AFF, Shade{100, 0xFF82B1FF}, Shade{200, 0xFF448AFF}, Shade{400, 0xFF2979FF}, Shade{700, 0xFF2962FF}),
		LightBlue:        NewMaterialColor(0xFF03A9F4, Shade{50, 0xFFE1F5FE}, Shade{100, 0xFFB3E5FC}, Shade{200, 0xFF81D4FA}, Shade{300, 0xFF4FC3F7}, Shade{400, 0xFF29B6F6}, Shade{500, 0xFF03A9F4}, Shade{600, 0xFF039BE5}, Shade{700, 0xFF0288D1}, Shade{800, 0xFF0277BD}, Shade{900, 0xFF01579B}),
		LightBlueAccent:  NewMaterialAccentColor(0xFF40C4FF, Shade{100, 0xFF80D8FF}, Shade{200, 0xFF40C4FF}, Shade{400, 0xFF00B0FF}, Shade{700, 0xFF0091EA}),
		Cyan:             NewMaterialColor(0xFF00BCD4, Shade{50, 0xFFE0F7FA}, Shade{100, 0xFFB2EBF2}, Shade{200, 0xFF80DEEA}, Shade{300, 0xFF4DD0E1}, Shade{400, 0xFF26C6DA}, Shade{500, 0xFF00BCD4}, Shade{600, 0xFF00ACC1}, Shade{700, 0xFF0097A7}, Shade{800, 0xFF00838F}, Shade{900, 0xFF006064}),
		CyanAccent:       NewMaterialAccentColor(0xFF18FFFF, Shade{100, 0xFF84FFFF}, Shade{200, 0xFF18FFFF}, Shade{400, 0xFF00E5FF}, Shade{700, 0xFF00B8D4}),
		Teal:             NewMaterialColor(0xFF009688, Shade{50, 0xFFE0F2F1}, Shade{100, 0xFFB2DFDB}, Shade{200, 0xFF80CBC4}, Shade{300, 0xFF4DB6AC}, Shade{400, 0xFF26A69A}, Shade{500, 0xFF009688}, Shade{600, 0xFF00897B}, Shade{700, 0xFF00796B}, Shade{800, 0xFF00695C}, Shade{900, 0xFF004D40}),
		TealAccent:       NewMaterialAccentColor(0xFF64FFDA, Shade{100, 0xFFA7FFEB}, Shade{200, 0xFF64FFDA}, Shade{400, 0xFF1DE9B6}, Shade{700, 0xFF00BFA5}),
		Green:            NewMaterialColor(0xFF4CAF50, Shade{50, 0xFFE8F5E9}, Shade{100, 0xFFC8E6C9}, Shade{200, 0xFFA5D6A7}, Shade{300, 0xFF81C784}, Shade{400, 0xFF66BB6A}, Shade{500, 0xFF4CAF50}, Shade{600, 0xFF43A047}, Shade{700, 0xFF388E3C}, Shade{800, 0xFF2E7D32}, Shade{900, 0xFF1B5E20}),
		GreenAccent:      NewMaterialAccentColor(0xFF69F0AE, Shade{100, 0xFFB9F6CA}, Shade{200, 0xFF69F0AE}, Shade{400, 0xFF00E676}, Shade{700, 0xFF00C853}),
		LightGreen:       NewMaterialColor(0xFF8BC34A, Shade{50, 0xFFF1F8E9}, Shade{100, 0xFFDCEDC8}, Shade{200, 0xFFC5E1A5}, Shade{300, 0xFFAED581}, Shade{400, 0xFF9CCC65}, Shade{500, 0xFF8BC34A}, Shade{600, 0xFF7CB342}, Shade{700, 0xFF689F38}, Shade{800, 0xFF558B2F}, Shade{900, 0xFF33691E}),
		LightGreenAccent: NewMaterialAccentColor(0xFFB2FF59, Shade{100, 0xFFCCFF90}, Shade{200, 0xFFB2FF59}, Shade{400, 0xFF76FF03}, Shade{700, 0xFF64DD17}),
		Lime:             NewMaterialColor(0xFFCDDC39, Shade{50, 0xFFF9FBE7}, Shade{100, 0xFFF0F4C3}, Shade{200, 0xFFE6EE9C}, Shade{300, 0xFFDCE775}, Shade{400, 0xFFD4E157}, Shade{500, 0xFFCDDC39}, Shade{600, 0xFFC0CA33}, Shade{700, 0xFFAFB42B}, Shade{800, 0xFF9E9D24}, Shade{900, 0xFF827717}),
		LimeAccent:       NewMaterialAccentColor(0xFFEEFF41, Shade{100, 0xFFF4FF81}, Shade{200, 0xFFEEFF41}, Shade{400, 0xFFC6FF00}, Shade{700, 0xFFAEEA00}),
		Yellow:           NewMaterialColor(0xFFFFEB3B, Shade{50, 0xFFFFFDE7}, Shade{100, 0xFFFFF9C4}, Shade{200, 0xFFFFF59D}, Shade{300, 0xFFFFF176}, Shade{400, 0xFFFFEE58}, Shade{500, 0xFFFFEB3B}, Shade{600, 0xFFFDD835}, Shade{700, 0xFFFBC02D}, Shade{800, 0xFFF9A825}, Shade{900, 0xFFF57F17}),
		YellowAccent:     NewMaterialAccentColor(0xFFFFFF00, Shade{100, 0xFFFFFF8D}, Shade{200, 0xFFFFFF00}, Shade{400, 0xFFFFEA00}, Shade{700, 0xFFFFD600}),
		Amber:            NewMaterialColor(0xFFFFC107, Shade{50, 0xFFFFF8E1}, Shade{100, 0xFFFFECB3}, Shade{200, 0xFFFFE082}, Shade{300, 0xFFFFD54F}, Shade{400, 0xFFFFCA28}, Shade{500, 0xFFFFC107}, Shade{600, 0xFFFFB300}, Shade{700, 0xFFFFA000}, Shade{800, 0xFFFF8F00}, Shade{900, 0xFFFF6F00}),
		AmberAccent:      NewMaterialAccentColor(0xFFFFD740, Shade{100, 0xFFFFE57F}, Shade{200, 0xFFFFD740}, Shade{400, 0xFFFFC400}, Shade{700, 0xFFFFAB00}),
		Orange:           NewMaterialColor(0xFFFF9800, Shade{50, 0xFFFFF3E0}, Shade{100, 0xFFFFE0B2}, Shade{200, 0xFFFFCC80}, Shade{300, 0xFFFFB74D}, Shade{400, 0xFFFFA726}, Shade{500, 0xFFFF9800}, Shade{600, 0xFFFB8C00}, Shade{700, 0xFFF57C00}, Shade{800, 0xFFEF6C00}, Shade{900, 0xFFE65100}),
		OrangeAccent:     NewMaterialAccentColor(0xFFFFAB40, Shade{100, 0xFFFFD180}, Shade{200, 0xFFFFAB40}, Shade{400, 0xFFFF9100}, Shade{700, 0xFFFF6D00}),
		DeepOrange:       NewMaterialColor(0xFFFF5722, Shade{50, 0xFFFBE9E7}, Shade{100, 0xFFFFCCBC}, Shade{200, 0xFFFFAB91}, Shade{300, 0xFFFF8A65}, Shade{400, 0xFFFF7043}, Shade{500, 0xFFFF5722}, Shade{600, 0xFFF4511E}, Shade{700, 0xFFE64A19}, Shade{800, 0xFFD84315}, Shade{900, 0xFFBF360C}),
		DeepOrangeAccent: NewMaterialAccentColor(0xFFFF6E40, Shade{100, 0xFFFF9E80}, Shade{200, 0xFFFF6E40}, Shade{400, 0xFFFF3D00}, Shade{700, 0xFFDD2C00}),
		Brown:            NewMaterialColor(0xFF795548, Shade{50, 0xFFEFEBE9}, Shade{100, 0xFFD7CCC8}, Shade{200, 0xFFBCAAA4}, Shade{300, 0xFFA1887F}, Shade{400, 0xFF8D6E63}, Shade{500, 0xFF795548}, Shade{600, 0xFF6D4C41}, Shade{700, 0xFF5D4037}, Shade{800, 0xFF4E342E}, Shade{900, 0xFF3E2723}),
		Grey:             NewMaterialColor(0xFF9E9E9E, Shade{50, 0xFFFAFAFA}, Shade{100, 0xFFF5F5F5}, Shade{200, 0xFFEEEEEE}, Shade{300, 0xFFE0E0E0}, Shade{350, 0xFFD6D6D6}, Shade{400, 0xFFBDBDBD}, Shade{500, 0xFF9E9E9E}, Shade{600, 0xFF757575}, Shade{700, 0xFF616161}, Shade{800, 0xFF424242}, Shade{850, 0xFF303030}, Shade{900, 0xFF212121}),
		BlueGrey:         NewMaterialColor(0xFF607D8B, Shade{50, 0xFFECEFF1}, Shade{100, 0xFFCFD8DC}, Shade{200, 0xFFB0BEC5}, Shade{300, 0xFF90A4AE}, Shade{400, 0xFF78909C}, Shade{500, 0xFF607D8B}, Shade{600, 0xFF546E7A}, Shade{700, 0xFF455A64}, Shade{800, 0xFF37474F}, Shade{900, 0xFF263238}),
	}
	c.Primaries = []MaterialColor{c.Red, c.Pink, c.Purple, c.DeepPurple, c.Indigo, c.Blue, c.LightBlue, c.Cyan, c.Teal, c.Green, c.LightGreen, c.Lime, c.Yellow, c.Amber, c.Orange, c.DeepOrange, c.Brown, c.BlueGrey}
	c.Accents = []MaterialAccentColor{c.RedAccent, c.PinkAccent, c.PurpleAccent, c.DeepPurpleAccent, c.IndigoAccent, c.BlueAccent, c.LightBlueAccent, c.CyanAccent, c.TealAccent, c.GreenAccent, c.LightGreenAccent, c.LimeAccent, c.YellowAccent, c.AmberAccent, c.OrangeAccent, c.DeepOrangeAccent}
	return c
}
