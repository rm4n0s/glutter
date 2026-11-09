package cupertino

import "github.com/rm4n0s/glutter/ui"

// CupertinoColorsType is the type of CupertinoColors: the colors of iOS and macOS, with the names of
// Flutter's CupertinoColors class. The system colors are CupertinoDynamicColor values; the others are plain colors.
//
// Flutter: cupertino/colors.dart CupertinoColors
type CupertinoColorsType struct {
	// ActiveBlue is the color activeBlue.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.activeBlue
	ActiveBlue CupertinoDynamicColor
	// ActiveGreen is the color activeGreen.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.activeGreen
	ActiveGreen CupertinoDynamicColor
	// ActiveOrange is the color activeOrange.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.activeOrange
	ActiveOrange CupertinoDynamicColor
	// White is the color white.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.white
	White ui.Color
	// Black is the color black.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.black
	Black ui.Color
	// Transparent is the color transparent.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.transparent
	Transparent ui.Color
	// LightBackgroundGray is the color lightBackgroundGray.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.lightBackgroundGray
	LightBackgroundGray ui.Color
	// ExtraLightBackgroundGray is the color extraLightBackgroundGray.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.extraLightBackgroundGray
	ExtraLightBackgroundGray ui.Color
	// DarkBackgroundGray is the color darkBackgroundGray.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.darkBackgroundGray
	DarkBackgroundGray ui.Color
	// InactiveGray is the color inactiveGray.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.inactiveGray
	InactiveGray CupertinoDynamicColor
	// DestructiveRed is the color destructiveRed.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.destructiveRed
	DestructiveRed CupertinoDynamicColor
	// SystemBlue is the color systemBlue.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemBlue
	SystemBlue CupertinoDynamicColor
	// SystemGreen is the color systemGreen.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemGreen
	SystemGreen CupertinoDynamicColor
	// SystemMint is the color systemMint.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemMint
	SystemMint CupertinoDynamicColor
	// SystemIndigo is the color systemIndigo.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemIndigo
	SystemIndigo CupertinoDynamicColor
	// SystemOrange is the color systemOrange.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemOrange
	SystemOrange CupertinoDynamicColor
	// SystemPink is the color systemPink.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemPink
	SystemPink CupertinoDynamicColor
	// SystemBrown is the color systemBrown.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemBrown
	SystemBrown CupertinoDynamicColor
	// SystemPurple is the color systemPurple.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemPurple
	SystemPurple CupertinoDynamicColor
	// SystemRed is the color systemRed.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemRed
	SystemRed CupertinoDynamicColor
	// SystemTeal is the color systemTeal.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemTeal
	SystemTeal CupertinoDynamicColor
	// SystemCyan is the color systemCyan.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemCyan
	SystemCyan CupertinoDynamicColor
	// SystemYellow is the color systemYellow.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemYellow
	SystemYellow CupertinoDynamicColor
	// SystemGrey is the color systemGrey.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemGrey
	SystemGrey CupertinoDynamicColor
	// SystemGrey2 is the color systemGrey2.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemGrey2
	SystemGrey2 CupertinoDynamicColor
	// SystemGrey3 is the color systemGrey3.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemGrey3
	SystemGrey3 CupertinoDynamicColor
	// SystemGrey4 is the color systemGrey4.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemGrey4
	SystemGrey4 CupertinoDynamicColor
	// SystemGrey5 is the color systemGrey5.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemGrey5
	SystemGrey5 CupertinoDynamicColor
	// SystemGrey6 is the color systemGrey6.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemGrey6
	SystemGrey6 CupertinoDynamicColor
	// Label is the color label.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.label
	Label CupertinoDynamicColor
	// SecondaryLabel is the color secondaryLabel.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.secondaryLabel
	SecondaryLabel CupertinoDynamicColor
	// TertiaryLabel is the color tertiaryLabel.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.tertiaryLabel
	TertiaryLabel CupertinoDynamicColor
	// QuaternaryLabel is the color quaternaryLabel.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.quaternaryLabel
	QuaternaryLabel CupertinoDynamicColor
	// SystemFill is the color systemFill.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemFill
	SystemFill CupertinoDynamicColor
	// SecondarySystemFill is the color secondarySystemFill.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.secondarySystemFill
	SecondarySystemFill CupertinoDynamicColor
	// TertiarySystemFill is the color tertiarySystemFill.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.tertiarySystemFill
	TertiarySystemFill CupertinoDynamicColor
	// QuaternarySystemFill is the color quaternarySystemFill.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.quaternarySystemFill
	QuaternarySystemFill CupertinoDynamicColor
	// PlaceholderText is the color placeholderText.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.placeholderText
	PlaceholderText CupertinoDynamicColor
	// SystemBackground is the color systemBackground.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemBackground
	SystemBackground CupertinoDynamicColor
	// SecondarySystemBackground is the color secondarySystemBackground.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.secondarySystemBackground
	SecondarySystemBackground CupertinoDynamicColor
	// TertiarySystemBackground is the color tertiarySystemBackground.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.tertiarySystemBackground
	TertiarySystemBackground CupertinoDynamicColor
	// SystemGroupedBackground is the color systemGroupedBackground.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.systemGroupedBackground
	SystemGroupedBackground CupertinoDynamicColor
	// SecondarySystemGroupedBackground is the color secondarySystemGroupedBackground.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.secondarySystemGroupedBackground
	SecondarySystemGroupedBackground CupertinoDynamicColor
	// TertiarySystemGroupedBackground is the color tertiarySystemGroupedBackground.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.tertiarySystemGroupedBackground
	TertiarySystemGroupedBackground CupertinoDynamicColor
	// Separator is the color separator.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.separator
	Separator CupertinoDynamicColor
	// OpaqueSeparator is the color opaqueSeparator.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.opaqueSeparator
	OpaqueSeparator CupertinoDynamicColor
	// Link is the color link.
	//
	// Flutter: cupertino/colors.dart CupertinoColors.link
	Link CupertinoDynamicColor
}

// CupertinoColors holds the colors of iOS and macOS. It is a read-only value.
//
// Flutter: cupertino/colors.dart CupertinoColors
var CupertinoColors = newCupertinoColors()

func newCupertinoColors() CupertinoColorsType {
	c := CupertinoColorsType{
		White:                            ui.NewColor(0xFFFFFFFF),
		Black:                            ui.NewColor(0xFF000000),
		Transparent:                      ui.NewColor(0x00000000),
		LightBackgroundGray:              ui.NewColor(0xFFE5E5EA),
		ExtraLightBackgroundGray:         ui.NewColor(0xFFEFEFF4),
		DarkBackgroundGray:               ui.NewColor(0xFF171717),
		InactiveGray:                     NewCupertinoDynamicColorWithBrightness(ui.NewColor(0xFF999999), ui.NewColor(0xFF757575), CupertinoDynamicColorOpts{DebugLabel: "inactiveGray"}),
		SystemBlue:                       NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFF007AFF), ui.NewColor(0xFF0A84FF), ui.NewColor(0xFF0040DD), ui.NewColor(0xFF409CFF), CupertinoDynamicColorOpts{DebugLabel: "systemBlue"}),
		SystemGreen:                      NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFF34C759), ui.NewColor(0xFF30D158), ui.NewColor(0xFF248A3D), ui.NewColor(0xFF30DB5B), CupertinoDynamicColorOpts{DebugLabel: "systemGreen"}),
		SystemMint:                       NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFF00C7BE), ui.NewColor(0xFF63E6E2), ui.NewColor(0xFF0C817B), ui.NewColor(0xFF66D4CF), CupertinoDynamicColorOpts{DebugLabel: "systemMint"}),
		SystemIndigo:                     NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFF5856D6), ui.NewColor(0xFF5E5CE6), ui.NewColor(0xFF3634A3), ui.NewColor(0xFF7D7AFF), CupertinoDynamicColorOpts{DebugLabel: "systemIndigo"}),
		SystemOrange:                     NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFFFF9500), ui.NewColor(0xFFFF9F0A), ui.NewColor(0xFFC93400), ui.NewColor(0xFFFFB340), CupertinoDynamicColorOpts{DebugLabel: "systemOrange"}),
		SystemPink:                       NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFFFF2D55), ui.NewColor(0xFFFF375F), ui.NewColor(0xFFD30F45), ui.NewColor(0xFFFF6482), CupertinoDynamicColorOpts{DebugLabel: "systemPink"}),
		SystemBrown:                      NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFFA2845E), ui.NewColor(0xFFAC8E68), ui.NewColor(0xFF7F6545), ui.NewColor(0xFFB59469), CupertinoDynamicColorOpts{DebugLabel: "systemBrown"}),
		SystemPurple:                     NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFFAF52DE), ui.NewColor(0xFFBF5AF2), ui.NewColor(0xFF8944AB), ui.NewColor(0xFFDA8FFF), CupertinoDynamicColorOpts{DebugLabel: "systemPurple"}),
		SystemRed:                        NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFFFF3B30), ui.NewColor(0xFFFF453A), ui.NewColor(0xFFD70015), ui.NewColor(0xFFFF6961), CupertinoDynamicColorOpts{DebugLabel: "systemRed"}),
		SystemTeal:                       NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFF5AC8FA), ui.NewColor(0xFF64D2FF), ui.NewColor(0xFF0071A4), ui.NewColor(0xFF70D7FF), CupertinoDynamicColorOpts{DebugLabel: "systemTeal"}),
		SystemCyan:                       NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFF32ADE6), ui.NewColor(0xFF64D2FF), ui.NewColor(0xFF0071A4), ui.NewColor(0xFF70D7FF), CupertinoDynamicColorOpts{DebugLabel: "systemCyan"}),
		SystemYellow:                     NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFFFFCC00), ui.NewColor(0xFFFFD60A), ui.NewColor(0xFFA05A00), ui.NewColor(0xFFFFD426), CupertinoDynamicColorOpts{DebugLabel: "systemYellow"}),
		SystemGrey:                       NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFF8E8E93), ui.NewColor(0xFF8E8E93), ui.NewColor(0xFF6C6C70), ui.NewColor(0xFFAEAEB2), CupertinoDynamicColorOpts{DebugLabel: "systemGrey"}),
		SystemGrey2:                      NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFFAEAEB2), ui.NewColor(0xFF636366), ui.NewColor(0xFF8E8E93), ui.NewColor(0xFF7C7C80), CupertinoDynamicColorOpts{DebugLabel: "systemGrey2"}),
		SystemGrey3:                      NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFFC7C7CC), ui.NewColor(0xFF48484A), ui.NewColor(0xFFAEAEB2), ui.NewColor(0xFF545456), CupertinoDynamicColorOpts{DebugLabel: "systemGrey3"}),
		SystemGrey4:                      NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFFD1D1D6), ui.NewColor(0xFF3A3A3C), ui.NewColor(0xFFBCBCC0), ui.NewColor(0xFF444446), CupertinoDynamicColorOpts{DebugLabel: "systemGrey4"}),
		SystemGrey5:                      NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFFE5E5EA), ui.NewColor(0xFF2C2C2E), ui.NewColor(0xFFD8D8DC), ui.NewColor(0xFF363638), CupertinoDynamicColorOpts{DebugLabel: "systemGrey5"}),
		SystemGrey6:                      NewCupertinoDynamicColorWithBrightnessAndContrast(ui.NewColor(0xFFF2F2F7), ui.NewColor(0xFF1C1C1E), ui.NewColor(0xFFEBEBF0), ui.NewColor(0xFF242426), CupertinoDynamicColorOpts{DebugLabel: "systemGrey6"}),
		Label:                            NewCupertinoDynamicColor(ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF), CupertinoDynamicColorOpts{DebugLabel: "label"}),
		SecondaryLabel:                   NewCupertinoDynamicColor(ui.NewColor(0x993C3C43), ui.NewColor(0x99EBEBF5), ui.NewColor(0xAD3C3C43), ui.NewColor(0xADEBEBF5), ui.NewColor(0x993C3C43), ui.NewColor(0x99EBEBF5), ui.NewColor(0xAD3C3C43), ui.NewColor(0xADEBEBF5), CupertinoDynamicColorOpts{DebugLabel: "secondaryLabel"}),
		TertiaryLabel:                    NewCupertinoDynamicColor(ui.NewColor(0x4C3C3C43), ui.NewColor(0x4CEBEBF5), ui.NewColor(0x603C3C43), ui.NewColor(0x60EBEBF5), ui.NewColor(0x4C3C3C43), ui.NewColor(0x4CEBEBF5), ui.NewColor(0x603C3C43), ui.NewColor(0x60EBEBF5), CupertinoDynamicColorOpts{DebugLabel: "tertiaryLabel"}),
		QuaternaryLabel:                  NewCupertinoDynamicColor(ui.NewColor(0x2D3C3C43), ui.NewColor(0x28EBEBF5), ui.NewColor(0x423C3C43), ui.NewColor(0x3DEBEBF5), ui.NewColor(0x2D3C3C43), ui.NewColor(0x28EBEBF5), ui.NewColor(0x423C3C43), ui.NewColor(0x3DEBEBF5), CupertinoDynamicColorOpts{DebugLabel: "quaternaryLabel"}),
		SystemFill:                       NewCupertinoDynamicColor(ui.NewColor(0x33787880), ui.NewColor(0x5B787880), ui.NewColor(0x47787880), ui.NewColor(0x70787880), ui.NewColor(0x33787880), ui.NewColor(0x5B787880), ui.NewColor(0x47787880), ui.NewColor(0x70787880), CupertinoDynamicColorOpts{DebugLabel: "systemFill"}),
		SecondarySystemFill:              NewCupertinoDynamicColor(ui.NewColor(0x28787880), ui.NewColor(0x51787880), ui.NewColor(0x3D787880), ui.NewColor(0x66787880), ui.NewColor(0x28787880), ui.NewColor(0x51787880), ui.NewColor(0x3D787880), ui.NewColor(0x66787880), CupertinoDynamicColorOpts{DebugLabel: "secondarySystemFill"}),
		TertiarySystemFill:               NewCupertinoDynamicColor(ui.NewColor(0x1E767680), ui.NewColor(0x3D767680), ui.NewColor(0x33767680), ui.NewColor(0x51767680), ui.NewColor(0x1E767680), ui.NewColor(0x3D767680), ui.NewColor(0x33767680), ui.NewColor(0x51767680), CupertinoDynamicColorOpts{DebugLabel: "tertiarySystemFill"}),
		QuaternarySystemFill:             NewCupertinoDynamicColor(ui.NewColor(0x14747480), ui.NewColor(0x2D767680), ui.NewColor(0x28747480), ui.NewColor(0x42767680), ui.NewColor(0x14747480), ui.NewColor(0x2D767680), ui.NewColor(0x28747480), ui.NewColor(0x42767680), CupertinoDynamicColorOpts{DebugLabel: "quaternarySystemFill"}),
		PlaceholderText:                  NewCupertinoDynamicColor(ui.NewColor(0x4C3C3C43), ui.NewColor(0x4CEBEBF5), ui.NewColor(0x603C3C43), ui.NewColor(0x60EBEBF5), ui.NewColor(0x4C3C3C43), ui.NewColor(0x4CEBEBF5), ui.NewColor(0x603C3C43), ui.NewColor(0x60EBEBF5), CupertinoDynamicColorOpts{DebugLabel: "placeholderText"}),
		SystemBackground:                 NewCupertinoDynamicColor(ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF000000), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF1C1C1E), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF242426), CupertinoDynamicColorOpts{DebugLabel: "systemBackground"}),
		SecondarySystemBackground:        NewCupertinoDynamicColor(ui.NewColor(0xFFF2F2F7), ui.NewColor(0xFF1C1C1E), ui.NewColor(0xFFEBEBF0), ui.NewColor(0xFF242426), ui.NewColor(0xFFF2F2F7), ui.NewColor(0xFF2C2C2E), ui.NewColor(0xFFEBEBF0), ui.NewColor(0xFF363638), CupertinoDynamicColorOpts{DebugLabel: "secondarySystemBackground"}),
		TertiarySystemBackground:         NewCupertinoDynamicColor(ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF2C2C2E), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF363638), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF3A3A3C), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF444446), CupertinoDynamicColorOpts{DebugLabel: "tertiarySystemBackground"}),
		SystemGroupedBackground:          NewCupertinoDynamicColor(ui.NewColor(0xFFF2F2F7), ui.NewColor(0xFF000000), ui.NewColor(0xFFEBEBF0), ui.NewColor(0xFF000000), ui.NewColor(0xFFF2F2F7), ui.NewColor(0xFF1C1C1E), ui.NewColor(0xFFEBEBF0), ui.NewColor(0xFF242426), CupertinoDynamicColorOpts{DebugLabel: "systemGroupedBackground"}),
		SecondarySystemGroupedBackground: NewCupertinoDynamicColor(ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF1C1C1E), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF242426), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF2C2C2E), ui.NewColor(0xFFFFFFFF), ui.NewColor(0xFF363638), CupertinoDynamicColorOpts{DebugLabel: "secondarySystemGroupedBackground"}),
		TertiarySystemGroupedBackground:  NewCupertinoDynamicColor(ui.NewColor(0xFFF2F2F7), ui.NewColor(0xFF2C2C2E), ui.NewColor(0xFFEBEBF0), ui.NewColor(0xFF363638), ui.NewColor(0xFFF2F2F7), ui.NewColor(0xFF3A3A3C), ui.NewColor(0xFFEBEBF0), ui.NewColor(0xFF444446), CupertinoDynamicColorOpts{DebugLabel: "tertiarySystemGroupedBackground"}),
		Separator:                        NewCupertinoDynamicColor(ui.NewColor(0x493C3C43), ui.NewColor(0x99545458), ui.NewColor(0x5E3C3C43), ui.NewColor(0xAD545458), ui.NewColor(0x493C3C43), ui.NewColor(0x99D2D2D2), ui.NewColor(0x5E3C3C43), ui.NewColor(0xAD545458), CupertinoDynamicColorOpts{DebugLabel: "separator"}),
		OpaqueSeparator:                  NewCupertinoDynamicColor(ui.NewColor(0xFFC6C6C8), ui.NewColor(0xFF38383A), ui.NewColor(0xFFC6C6C8), ui.NewColor(0xFF38383A), ui.NewColor(0xFFC6C6C8), ui.NewColor(0xFF38383A), ui.NewColor(0xFFC6C6C8), ui.NewColor(0xFF38383A), CupertinoDynamicColorOpts{DebugLabel: "opaqueSeparator"}),
		Link:                             NewCupertinoDynamicColor(ui.NewColor(0xFF007AFF), ui.NewColor(0xFF0984FF), ui.NewColor(0xFF007AFF), ui.NewColor(0xFF0984FF), ui.NewColor(0xFF007AFF), ui.NewColor(0xFF0984FF), ui.NewColor(0xFF007AFF), ui.NewColor(0xFF0984FF), CupertinoDynamicColorOpts{DebugLabel: "link"}),
	}
	c.ActiveBlue = c.SystemBlue
	c.ActiveGreen = c.SystemGreen
	c.ActiveOrange = c.SystemOrange
	c.DestructiveRed = c.SystemRed
	return c
}
