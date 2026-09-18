package uqmplanetgen

import (
	"encoding/binary"
	"errors"
	"image"
)

var errNilPalette = errors.New("uqmplanetgen: nil palette")

func checkPalettes(xtab *XTab, ctab CTab) error {
	switch {
	case xtab == nil && len(ctab) == 0:
		return errNilPalette
	case xtab == nil:
		return errors.New("uqmplanetgen: nil xtab")
	case len(ctab) == 0:
		return errors.New("uqmplanetgen: empty ctab")
	}
	return nil
}

// colorLUT builds the elevation-byte to color lookup used by Render, so the
// per-pixel loop is a single table read.
func colorLUT(gasGiant bool, baseElevation int, xtab *XTab, ctab CTab) *[256]uint32 {
	var lut [256]uint32
	for el := 0; el < 256; el++ {
		d := el // gas giants read the byte unsigned, making elevations non-negative
		if !gasGiant {
			d = int(int8(el)) + baseElevation // sign-extend, then add sea level
			if d < 0 {
				d = 0
			} else if d > 255 {
				d = 255
			}
		}
		lut[el] = ctab.At(int(xtab[d]))
	}
	return &lut
}

// Render converts an elevation map into an RGBA texture using xtab and ctab,
// which must be a matching pair from the same variant. gasGiant selects UQM's
// gas-giant color path, in which baseElevation is ignored; rocky worlds use
// baseElevation as their sea-level offset.
//
// An elevation whose translated palette index is not defined by ctab renders
// as that table's zero value, a fully transparent pixel.
func Render(e Elevation, gasGiant bool, baseElevation int,
	xtab *XTab, ctab CTab) (*image.RGBA, error) {
	if err := e.check(); err != nil {
		return nil, err
	}
	if err := checkPalettes(xtab, ctab); err != nil {
		return nil, err
	}

	lut := colorLUT(gasGiant, baseElevation, xtab, ctab)
	img := image.NewRGBA(image.Rect(0, 0, e.width, e.height))
	pix := img.Pix
	for i, el := range e.pix {
		binary.LittleEndian.PutUint32(pix[4*i:], lut[el])
	}
	return img, nil
}
