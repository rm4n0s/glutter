package win32

// The clipboard formats, the system parameters and the other values of step 28.6 (SPEC §25.7,
// §25.8). They carry no OS tag, so that the layout test checks them against the headers.
const (
	// CFDIB is CF_DIB.
	//
	// Flutter: no counterpart
	CFDIB = 8
	// CFUnicodeText is CF_UNICODETEXT: the text of the clipboard, UTF-16 with a terminating NUL.
	//
	// Flutter: no counterpart
	CFUnicodeText = 13
	// CFHDrop is CF_HDROP: the files of the clipboard, a DROPFILES followed by the paths.
	//
	// Flutter: no counterpart
	CFHDrop = 15
	// CFDIBV5 is CF_DIBV5: a bitmap with a BITMAPV5HEADER.
	//
	// Flutter: no counterpart
	CFDIBV5 = 17

	// BIBitfields is BI_BITFIELDS.
	//
	// Flutter: no counterpart
	BIBitfields = 3
	// LCSsRGB is LCS_sRGB, the color space "sRGB" of a BITMAPV5HEADER.
	//
	// Flutter: no counterpart
	LCSsRGB = 0x73524742
	// LCSGMImages is LCS_GM_IMAGES, the rendering intent that keeps the match of the colors.
	//
	// Flutter: no counterpart
	LCSGMImages = 4

	// SPIGetNonClientMetrics is SPI_GETNONCLIENTMETRICS.
	//
	// Flutter: no counterpart
	SPIGetNonClientMetrics = 0x0029
	// SPIGetHighContrast is SPI_GETHIGHCONTRAST.
	//
	// Flutter: no counterpart
	SPIGetHighContrast = 0x0042
	// SPIGetFontSmoothing is SPI_GETFONTSMOOTHING.
	//
	// Flutter: no counterpart
	SPIGetFontSmoothing = 0x004A
	// SPIGetClientAreaAnimation is SPI_GETCLIENTAREAANIMATION.
	//
	// Flutter: no counterpart
	SPIGetClientAreaAnimation = 0x1042
	// SPIGetFontSmoothingType is SPI_GETFONTSMOOTHINGTYPE.
	//
	// Flutter: no counterpart
	SPIGetFontSmoothingType = 0x200A
	// SPIGetFontSmoothingOrientation is SPI_GETFONTSMOOTHINGORIENTATION.
	//
	// Flutter: no counterpart
	SPIGetFontSmoothingOrientation = 0x2012
	// FEFontSmoothingStandard is FE_FONTSMOOTHINGSTANDARD (gray levels).
	//
	// Flutter: no counterpart
	FEFontSmoothingStandard = 1
	// FEFontSmoothingClearType is FE_FONTSMOOTHINGCLEARTYPE (sub-pixel).
	//
	// Flutter: no counterpart
	FEFontSmoothingClearType = 2
	// HCFHighContrastOn is HCF_HIGHCONTRASTON.
	//
	// Flutter: no counterpart
	HCFHighContrastOn = 0x00000001

	// SMCxCursor is SM_CXCURSOR.
	//
	// Flutter: no counterpart
	SMCxCursor = 13
	// MUILanguageName is MUI_LANGUAGE_NAME: the languages as BCP 47 names.
	//
	// Flutter: no counterpart
	MUILanguageName = 0x8
	// LocaleSShortTime is LOCALE_SSHORTTIME: the short time pattern of the user's locale.
	//
	// Flutter: no counterpart
	LocaleSShortTime = 0x00000079

	// EndSessionCloseApp is ENDSESSION_CLOSEAPP (the application is being closed for an update).
	//
	// Flutter: no counterpart
	EndSessionCloseApp = 0x00000001
	// EndSessionCritical is ENDSESSION_CRITICAL (the application is forced to end).
	//
	// Flutter: no counterpart
	EndSessionCritical = 0x40000000
	// EndSessionLogoff is ENDSESSION_LOGOFF (the user is logging off).
	//
	// Flutter: no counterpart
	EndSessionLogoff = 0x80000000

	// KeyRead is KEY_READ.
	//
	// Flutter: no counterpart
	KeyRead = 0x00020019
	// RRFRTRegSZ is RRF_RT_REG_SZ.
	//
	// Flutter: no counterpart
	RRFRTRegSZ = 0x00000002
	// RRFRTRegExpandSZ is RRF_RT_REG_EXPAND_SZ.
	//
	// Flutter: no counterpart
	RRFRTRegExpandSZ = 0x00000004
)

// HKeyLocalMachine is HKEY_LOCAL_MACHINE.
//
// Flutter: no counterpart
const HKeyLocalMachine uintptr = 0x80000002
