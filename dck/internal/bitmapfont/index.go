// Package bitmapfont maps characters to the Bilizir bitmap font atlas.
package bitmapfont

// Index returns the glyph index for a supported character.
func Index(char rune) (int, bool) {
	// Rows: ABCDEFGHIJ, KLMNOPQRST, UVWXYZ0123, 456789(),., !
	if char >= 'a' && char <= 'z' {
		char -= 'a' - 'A'
	}

	switch {
	case char >= 'A' && char <= 'Z':
		return int(char - 'A'), true
	case char >= '0' && char <= '9':
		return 26 + int(char-'0'), true
	}

	switch char {
	case '(':
		return 36, true
	case ')':
		return 37, true
	case ',':
		return 38, true
	case '.':
		return 39, true
	case '!':
		return 40, true
	default:
		return -1, false
	}
}
