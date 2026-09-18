package uqmplanetgen

import "fmt"

// numPaletteIndexes is the range an [XTab] entry can address, and so the length
// of every CTab this package builds.
const numPaletteIndexes = 256

// CTab is a planet-surface colormap (a .ct file) indexed by absolute palette
// index: CTab[XTab[level]] is the color for that elevation level. Indexes the
// source table does not define hold 0.
//
// Each entry packs one color in image.RGBA's memory order — red in the low
// byte, then green, blue and alpha — so rendering is a single 32-bit store.
// Build entries with [Color] rather than assembling them by hand.
type CTab []uint32

// Color packs 8-bit channels into a CTab entry.
func Color(r, g, b, a byte) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16 | uint32(a)<<24
}

// At returns the color at a palette index, or 0 for one the table does not
// define.
func (c CTab) At(index int) uint32 {
	if index < 0 || index >= len(c) {
		return 0
	}
	return c[index]
}

// cc5to8 is gfxlib.h's 5-bit to 8-bit channel expansion.
func cc5to8(c int) int { return c<<3 | c>>2 }

// color6 converts one UQM colormap triplet. The stored channels are 6-bit and
// drop their low bit before expansion. The three results are packed and then
// truncated per byte, which is how the original handles a synthetic palette
// whose channels run past 6 bits.
func color6(r, g, b byte) uint32 {
	v := uint32(cc5to8(int(r)>>1))<<16 | uint32(cc5to8(int(g)>>1))<<8 | uint32(cc5to8(int(b)>>1))
	return Color(byte(v>>16), byte(v>>8), byte(v), 0xFF)
}

// newCTabFromRGB places len(rgb)/3 converted triplets at consecutive palette
// indexes starting at firstIndex.
func newCTabFromRGB(firstIndex int, rgb []byte) (CTab, error) {
	if len(rgb)%3 != 0 {
		return nil, fmt.Errorf("uqmplanetgen: rgb length %d is not a multiple of 3", len(rgb))
	}
	n := len(rgb) / 3
	if n == 0 {
		return nil, fmt.Errorf("uqmplanetgen: color table is empty")
	}
	if firstIndex < 0 || firstIndex+n > numPaletteIndexes {
		return nil, fmt.Errorf("uqmplanetgen: %d colors at index %d overflow the %d-entry palette",
			n, firstIndex, numPaletteIndexes)
	}

	c := make(CTab, numPaletteIndexes)
	for i := 0; i < n; i++ {
		c[firstIndex+i] = color6(rgb[i*3], rgb[i*3+1], rgb[i*3+2])
	}
	return c, nil
}

// NewCTab builds a colormap from 6-bit RGB triplets placed at consecutive
// palette indexes starting at firstIndex, for synthetic palettes.
func NewCTab(firstIndex int, rgb []byte) (CTab, error) {
	return newCTabFromRGB(firstIndex, rgb)
}

// LoadCTab parses item variant of a BINTAB .ct buffer. Shipped .ct files hold
// all three temperature variants: 0 cold, 1 normal, 2 hot. See [TempVariant].
// The payload is a first palette index, a last index this package recomputes
// from the payload size, then the triplets.
func LoadCTab(data []byte, variant int) (CTab, error) {
	item, ok := bintabVariant(data, variant)
	if !ok || len(item) < 2+3 || (len(item)-2)%3 != 0 {
		return nil, loadErr("color table", variant)
	}
	c, err := newCTabFromRGB(int(item[0]), item[2:])
	if err != nil {
		return nil, fmt.Errorf("uqmplanetgen: color table variant %d: %w", variant, err)
	}
	return c, nil
}
