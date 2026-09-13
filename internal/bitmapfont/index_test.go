package bitmapfont

import "testing"

func TestIndex(t *testing.T) {
	tests := []struct {
		char  rune
		index int
		found bool
	}{
		{char: 'A', index: 0, found: true},
		{char: 'z', index: 25, found: true},
		{char: '0', index: 26, found: true},
		{char: '9', index: 35, found: true},
		{char: '(', index: 36, found: true},
		{char: ')', index: 37, found: true},
		{char: ',', index: 38, found: true},
		{char: '.', index: 39, found: true},
		{char: '!', index: 40, found: true},
		{char: ' ', index: -1, found: false},
		{char: 'é', index: -1, found: false},
	}

	for _, test := range tests {
		index, found := Index(test.char)
		if index != test.index || found != test.found {
			t.Errorf("Index(%q) = (%d, %v), want (%d, %v)", test.char, index, found, test.index, test.found)
		}
	}
}
