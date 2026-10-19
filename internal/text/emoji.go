package text

// IsEmoji reports whether the rune is one that a text normally shows as a color emoji: the
// pictographs of the emoji blocks and the symbols and dingbats that have an emoji
// presentation. A font that has such a rune in color is preferred over one that has it as a
// black and white symbol.
//
// Flutter: no counterpart
func IsEmoji(r rune) bool {
	switch {
	case r >= 0x1F300 && r <= 0x1FAFF, // pictographs, emoticons, transport, supplemental symbols
		r >= 0x1F000 && r <= 0x1F2FF, // mahjong, cards, enclosed alphanumerics
		r >= 0x2600 && r <= 0x27BF,   // miscellaneous symbols and dingbats
		r >= 0x231A && r <= 0x23FF,
		r == 0x2B50, r == 0x2B55, r == 0x2B1B, r == 0x2B1C,
		r == 0x203C, r == 0x2049, r == 0x2122, r == 0x2139,
		r >= 0x2194 && r <= 0x21AA,
		r == 0x24C2, r == 0x3030, r == 0x303D, r == 0x3297, r == 0x3299:
		return true
	}
	return false
}
