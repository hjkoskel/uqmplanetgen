package uqmplanetgen

// Crater stamps an elliptical crater whose bounding box has its top-left
// corner at (x,y) and size rw x rh. rimDelta is added to the rim ring,
// craterDelta (usually negative) fills the interior; setDepth zeroes each
// touched row segment first. The caller must guarantee the bounding box fits
// in the map, as UQM's dispatch code does.
func (e Elevation) Crater(x, y, rw, rh, rimDelta, craterDelta int, setDepth bool) error {
	if err := e.check(); err != nil {
		return err
	}
	e.crater(x, y, rw, rh, rimDelta, craterDelta, setDepth)
	return nil
}

// crater ports UQM's plangen.c MakeCrater midpoint-ellipse rasteriser.
// Elevation updates here are plain mod-256 additions, unlike the clamped fault
// paths. Indices are deliberately COUNT-sized (uint16) and wrap like the
// original.
func (e Elevation) crater(xCorner, yCorner, rw, rh, rimDelta, craterDelta int, setDepth bool) {
	width := e.width // the height is not needed: the caller guarantees the fit
	pix := e.pix

	var x, lfX, rtX int16
	a := int16(int32(rw) >> 1)
	b := int16(int32(rh) >> 1)
	y := b

	// For A,B <= 16383 the squares stay below 2^31, so int64 math is exact;
	// dy is a genuine 32-bit 'long' multiply and is truncated to match.
	asq := int64(uint32(a)) * int64(uint32(a))
	bsq := int64(uint32(b)) * int64(uint32(b))
	twoAsq := asq << 1
	twoBsq := bsq << 1

	dx := int64(0)
	dy := int64(int32(twoAsq * int64(b)))
	d := bsq - (dy >> 1) + (asq >> 2)

	a += int16(xCorner)
	b += int16(yCorner)
	topIdx := uint16((int32(b) - int32(y)) * int32(width))
	botIdx := uint16((int32(b) + int32(y)) * int32(width))
	rimPixels := uint16(1)

	// Quadrant 1: each step stamps the top and bottom rows together.
	for dx < dy {
		if d > 0 {
			lfX, rtX = a-x, a+x
			if setDepth {
				n := int(rtX) - int(lfX) + 1
				clear(pix[int(topIdx)+int(lfX):][:n])
				clear(pix[int(botIdx)+int(lfX):][:n])
			}
			if lfX == rtX {
				pix[int(topIdx)+int(lfX)] += byte(rimDelta)
				pix[int(botIdx)+int(lfX)] += byte(rimDelta)
				rimPixels = 0
			} else {
				for ; rimPixels > 0; rimPixels-- {
					pix[int(topIdx)+int(lfX)] += byte(rimDelta)
					pix[int(botIdx)+int(lfX)] += byte(rimDelta)
					if lfX != rtX {
						pix[int(topIdx)+int(rtX)] += byte(rimDelta)
						pix[int(botIdx)+int(rtX)] += byte(rimDelta)
					}
					lfX++
					rtX--
				}
				for lfX < rtX {
					pix[int(topIdx)+int(lfX)] += byte(craterDelta)
					pix[int(botIdx)+int(lfX)] += byte(craterDelta)
					pix[int(topIdx)+int(rtX)] += byte(craterDelta)
					pix[int(botIdx)+int(rtX)] += byte(craterDelta)
					lfX++
					rtX--
				}
				if lfX == rtX {
					pix[int(topIdx)+int(lfX)] += byte(craterDelta)
					pix[int(botIdx)+int(lfX)] += byte(craterDelta)
				}
			}

			y--
			topIdx += uint16(width)
			botIdx -= uint16(width)
			dy -= twoAsq
			d -= dy // the updated dy, as in C
		}

		rimPixels++ // outside the if, as in C
		x++
		dx += twoBsq
		d += bsq + dx
	}

	d += ((((asq - bsq) * 3) >> 1) - (dx + dy)) >> 1

	// Quadrant 2.
	for y > 0 {
		lfX, rtX = a-x, a+x
		if setDepth {
			n := int(rtX) - int(lfX) + 1
			clear(pix[int(topIdx)+int(lfX):][:n])
			clear(pix[int(botIdx)+int(lfX):][:n])
		}
		if lfX == rtX {
			pix[int(topIdx)+int(lfX)] += byte(rimDelta)
			pix[int(botIdx)+int(lfX)] += byte(rimDelta)
		} else {
			for ; rimPixels > 0; rimPixels-- {
				pix[int(topIdx)+int(lfX)] += byte(rimDelta)
				pix[int(botIdx)+int(lfX)] += byte(rimDelta)
				if lfX != rtX {
					pix[int(topIdx)+int(rtX)] += byte(rimDelta)
					pix[int(botIdx)+int(rtX)] += byte(rimDelta)
				}
				lfX++
				rtX--
			}
			for lfX < rtX {
				pix[int(topIdx)+int(lfX)] += byte(craterDelta)
				pix[int(botIdx)+int(lfX)] += byte(craterDelta)
				pix[int(topIdx)+int(rtX)] += byte(craterDelta)
				pix[int(botIdx)+int(rtX)] += byte(craterDelta)
				lfX++
				rtX--
			}
			if lfX == rtX {
				pix[int(topIdx)+int(lfX)] += byte(craterDelta)
				pix[int(botIdx)+int(lfX)] += byte(craterDelta)
			}
		}

		if d < 0 {
			x++
			dx += twoBsq
			d += dx
		}

		rimPixels = 1 // reset every row, as in C
		y--
		topIdx += uint16(width)
		botIdx -= uint16(width)
		dy -= twoAsq
		d += asq - dy
	}

	// Final row (y == 0): top only.
	lfX, rtX = a-x, a+x
	if setDepth {
		clear(pix[int(topIdx)+int(lfX):][:int(rtX)-int(lfX)+1])
	}
	if lfX == rtX {
		pix[int(topIdx)+int(lfX)] += byte(rimDelta)
		return
	}
	for ; rimPixels > 0; rimPixels-- {
		pix[int(topIdx)+int(lfX)] += byte(rimDelta)
		if lfX != rtX {
			pix[int(topIdx)+int(rtX)] += byte(rimDelta)
		}
		lfX++
		rtX--
	}
	for lfX < rtX {
		pix[int(topIdx)+int(lfX)] += byte(craterDelta)
		pix[int(topIdx)+int(rtX)] += byte(craterDelta)
		lfX++
		rtX--
	}
	if lfX == rtX {
		pix[int(topIdx)+int(lfX)] += byte(craterDelta)
	}
}
