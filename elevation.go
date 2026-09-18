package uqmplanetgen

import (
	"errors"
	"fmt"
)

const (
	numBandColors = 4
	rangeShift    = 6

	ditherVariance = 1 << (rangeShift - 3)
)

var errEmptyElevation = errors.New("uqmplanetgen: empty elevation buffer")

// Elevation is a row-major map of signed 8-bit elevations carrying its own
// dimensions: pixel (x,y) lives at pix[y*width+x], and 0 is sea level. The
// bytes hold two's-complement int8 values; the generation passes rely on
// mod-256 wraparound, so they are stored as bytes rather than int8.
//
// An Elevation is a view over its pixels, not a copy of them: assigning one
// shares the storage, as a slice does.
type Elevation struct {
	pix           []byte
	width, height int
}

// NewElevation allocates a zeroed map. A non-positive width or height yields a
// map that every method rejects.
func NewElevation(width, height int) Elevation {
	if width <= 0 || height <= 0 {
		return Elevation{width: width, height: height}
	}
	return Elevation{pix: make([]byte, width*height), width: width, height: height}
}

// NewElevationFrom wraps an existing buffer, which must hold exactly
// width*height bytes and is not copied.
func NewElevationFrom(pix []byte, width, height int) (Elevation, error) {
	e := Elevation{pix: pix, width: width, height: height}
	if err := e.check(); err != nil {
		return Elevation{}, err
	}
	return e, nil
}

// Width and Height are the dimensions fixed at construction.
func (e Elevation) Width() int  { return e.width }
func (e Elevation) Height() int { return e.height }

// Pix returns the underlying pixels, in row-major order. Writes through it are
// visible to e.
func (e Elevation) Pix() []byte { return e.pix }

// At returns the elevation at (x,y).
func (e Elevation) At(x, y int) int8 { return int8(e.pix[y*e.width+x]) }

// Set writes the elevation at (x,y).
func (e Elevation) Set(x, y int, v int8) { e.pix[y*e.width+x] = byte(v) }

func (e Elevation) check() error {
	if e.width <= 0 || e.height <= 0 {
		return fmt.Errorf("uqmplanetgen: invalid map size %dx%d", e.width, e.height)
	}
	if len(e.pix) != e.width*e.height {
		return fmt.Errorf("uqmplanetgen: elevation holds %d bytes, want %d (%dx%d)",
			len(e.pix), e.width*e.height, e.width, e.height)
	}
	return nil
}

// strip returns a map sharing e's storage that starts at the given row and is
// height rows tall. Its buffer runs to the end of e rather than stopping at
// the strip, matching what the band-edge fault passes in gasGiant expect, so
// the result does not satisfy check and is for internal use only.
func (e Elevation) strip(row, height int) Elevation {
	if row < 0 {
		row = 0
	}
	return Elevation{pix: e.pix[row*e.width:], width: e.width, height: height}
}

// Generate zeroes e and runs the full elevation pass for algo, mirroring UQM's
// GeneratePlanetSurface. numFaults is the fault iteration count for rocky
// worlds or the band count for gas giants; numBlemishes applies only to
// [AlgoCratered]. Gas giants need Height/numFaults >= 3 (see
// [Elevation.GasGiant]).
func (e Elevation) Generate(algo Algo, seed uint32, numFaults, faultDepth, numBlemishes int) error {
	if err := e.check(); err != nil {
		return err
	}
	if algo == AlgoGasGiant {
		if err := e.checkBands(numFaults); err != nil {
			return err
		}
	} else if numBlemishes > 0 && e.height < minBlemishHeight {
		return fmt.Errorf("uqmplanetgen: craters need a height of at least %d, have %d",
			minBlemishHeight, e.height)
	}
	e.generate(NewRand(seed), algo, numFaults, faultDepth, numBlemishes)
	return nil
}

// minBlemishHeight is the smallest map the crater and storm size rolls work on:
// both take a modulo by height>>4.
const minBlemishHeight = 16

// checkBands rejects band counts the gas giant pass cannot use. The band-edge
// jitter takes a modulo by bandHeight-2, and the storms need the same height
// floor as craters.
func (e Elevation) checkBands(numBands int) error {
	if numBands <= 0 {
		return fmt.Errorf("uqmplanetgen: need at least one band, have %d", numBands)
	}
	if e.height/numBands < 3 {
		return fmt.Errorf("uqmplanetgen: %d bands need a height of at least %d, have %d",
			numBands, 3*numBands, e.height)
	}
	if e.height < minBlemishHeight {
		return fmt.Errorf("uqmplanetgen: gas giant storms need a height of at least %d, have %d",
			minBlemishHeight, e.height)
	}
	return nil
}

func (e Elevation) generate(r *Rand, algo Algo, numFaults, faultDepth, numBlemishes int) {
	width, height := e.width, e.height
	clear(e.pix)

	switch algo {
	case AlgoGasGiant:
		e.gasGiant(r, numFaults, faultDepth)

	case AlgoTopo, AlgoCratered:
		if numFaults != 0 {
			e.faults(r, numFaults, faultDepth)
		}

		for i := 0; i < numBlemishes; i++ {
			loword := uint16(r.Uint32())
			var rw int
			switch (loword >> 8) & 31 {
			case 0:
				rw = int(loword&0xFF)%(height>>2) + height>>2
			case 1, 2, 3, 4:
				rw = int(loword&0xFF)%(height>>3) + height>>3
			default:
				rw = int(loword&0xFF)%(height>>4) + 4 // UQM adds a literal 4, not height>>4
			}

			loword = uint16(r.Uint32())
			if rw >= width || rw >= height {
				continue // no room; the draw above still has to happen
			}
			x := int((loword>>8)&0xFF) % (width - rw)
			y := int(loword&0xFF) % (height - rw)

			e.crater(x, y, rw, rw, faultDepth<<2, -(faultDepth << 2), false)
		}

		if algo == AlgoCratered {
			e.dither(r)
		}
		e.validate()

	default:
		// As in the original switch: an unknown algorithm leaves a flat map.
	}
}

// Faults runs iterations fault-formation passes: each pass raises the region
// between two random DDA lines by +/-depthDelta and lowers the wrapping
// complement. Consumes two values from r per iteration.
func (e Elevation) Faults(r *Rand, iterations, depthDelta int) error {
	if err := e.check(); err != nil {
		return err
	}
	e.faults(r, iterations, depthDelta)
	return nil
}

// Dither adds per-pixel noise to every element; one value from r is shared by
// four consecutive pixels, as in UQM.
func (e Elevation) Dither(r *Rand) error {
	if len(e.pix) == 0 {
		return errEmptyElevation
	}
	e.dither(r)
	return nil
}

// Validate flattens sign-flip discontinuities between horizontally adjacent
// pixels.
func (e Elevation) Validate() error {
	if err := e.check(); err != nil {
		return err
	}
	e.validate()
	return nil
}

// addClamped adds delta to n pixels starting at index lp, clamping into the
// signed 8-bit range as DeltaTopography does, and returns the index just past
// them.
func (e Elevation) addClamped(lp int, n uint16, delta int32) int {
	pix := e.pix // hoisted: this is the hottest loop in the package
	for ; n > 0; n-- {
		if v := int32(int8(pix[lp])) + delta; v >= -128 && v <= 127 {
			pix[lp] = byte(v)
		}
		lp++
	}
	return lp
}

// faults ports UQM's gentopo.c DeltaTopography. Each pass walks two random DDA
// lines down from the top row; pixels between them get +depthDelta and the
// wrapping complement gets -depthDelta. The dimensions must fit in int16, as
// they do for every map size UQM uses.
func (e Elevation) faults(r *Rand, numIterations, depthDelta int) {
	w := int16(e.width)
	h := int16(e.height)

	var line0, line1 lineDDA
	deltaY := (h - 1) << 1
	dd := depthDelta // the original reassigns its parameter each pass

	for ; numIterations > 0; numIterations-- {
		if r.Uint32()&1 == 0 {
			dd = -dd
		}

		rv := r.Uint32()
		w1 := uint16(rv)
		w2 := uint16(rv >> 16)

		line0.init(int16(uint16(w1&0xFF)%uint16(w)), int16(uint16((w1>>8)&0xFF)%uint16(w)), deltaY)
		// The right-hand sides are computed in C's full int, then truncated.
		line1.init(
			int16(int32(uint16(w2&0xFF)%uint16(w-1))+int32(line0.xTop)+1),
			int16(int32(uint16((w2>>8)&0xFF)%uint16(w-1))+int32(line0.xBot)+1),
			deltaY)

		lp := int(line0.xTop)
		for row := uint16(h); row > 0; row-- {
			w1 = uint16(line1.xTop - line0.xTop)
			w2 = uint16(int32(w) - int32(w1))

			// Inside the region: +dd, from the pen position, wrapping to column 0.
			var n uint16
			if int(line0.xTop)+int(w1) > int(w) {
				n = uint16(int32(w) - int32(line0.xTop))
			} else {
				n = w1
				line0.xTop += int16(w1)
			}
			w1 -= n
			lp = e.addClamped(lp, n, int32(dd))
			if w1 == 0 {
				if line0.xTop == w { // the pen ran off the end of the row
					line0.xTop = 0
					lp -= int(w)
				}
			} else {
				line0.xTop = int16(w1)
				lp -= int(w)
				lp = e.addClamped(lp, w1, int32(dd))
			}

			// Outside (the wrapping complement): -dd.
			if int(line0.xTop)+int(w2) > int(w) {
				n = uint16(int32(w) - int32(line0.xTop))
			} else {
				n = w2
				line0.xTop += int16(w2)
			}
			w2 -= n
			lp = e.addClamped(lp, n, -int32(dd))
			if w2 == 0 {
				if line0.xTop == w {
					line0.xTop = 0
					lp -= int(w)
				}
			} else {
				line0.xTop = int16(w2)
				lp -= int(w)
				lp = e.addClamped(lp, w2, -int32(dd))
			}

			lp += int(w) // next row

			if deltaY >= line0.deltaX {
				if line0.errorTerm += line0.deltaX; line0.errorTerm >= 0 {
					lp += int(line0.xIncr)
					line0.xTop += line0.xIncr
					line0.errorTerm -= deltaY
				}
			} else {
				for {
					lp += int(line0.xIncr)
					line0.xTop += line0.xIncr
					if line0.errorTerm += deltaY; line0.errorTerm >= 0 {
						break
					}
				}
				line0.errorTerm -= line0.deltaX
			}

			if deltaY >= line1.deltaX {
				if line1.errorTerm += line1.deltaX; line1.errorTerm >= 0 {
					line1.xTop += line1.xIncr
					line1.errorTerm -= deltaY
				}
			} else {
				for {
					line1.xTop += line1.xIncr
					if line1.errorTerm += deltaY; line1.errorTerm >= 0 {
						break
					}
				}
				line1.errorTerm -= line1.deltaX
			}
		}
	}
}

// lineDDA mirrors the C LineDDA struct. Every field is int16_t in the
// original, and Go's fixed-width arithmetic wraps mod 2^16 the same way.
type lineDDA struct {
	xTop      int16
	xBot      int16
	deltaX    int16
	xIncr     int16 // +1 or -1
	errorTerm int16
}

func (l *lineDDA) init(xTop, xBot, deltaY int16) {
	l.xTop = xTop
	l.xBot = xBot
	l.deltaX = (xBot - xTop) << 1
	l.xIncr = 1
	if l.deltaX < 0 {
		l.xIncr = -1
		l.deltaX = -l.deltaX
	}
	if l.deltaX > deltaY {
		l.errorTerm = -(l.deltaX >> 1)
	} else {
		l.errorTerm = -(deltaY >> 1)
	}
}

// validate ports UQM's plangen.c ValidateMap. Pass one scans the first row to
// decide which sign-flip state dominates; pass two rewrites every pixel whose
// signed difference from its left neighbour exceeds a full byte range.
func (e Elevation) validate() {
	var state uint8
	var pixelCount [2]uint8 // wraps at 256, like the original BYTE counters
	var lb [2]int8          // first byte seen in each state

	lastByte := e.pix[0]
	for i := 1; i < e.width; i++ {
		if pixelCount[state] == 0 {
			lb[state] = int8(lastByte)
		}
		pixelCount[state]++

		if signFlip(lastByte, e.pix[i]) {
			state ^= 1
		}
		lastByte = e.pix[i]
	}

	if pixelCount[0] > pixelCount[1] {
		lastByte = byte(lb[0])
	} else {
		lastByte = byte(lb[1])
	}
	for i := range e.pix {
		if signFlip(lastByte, e.pix[i]) {
			e.pix[i] = lastByte
		}
		lastByte = e.pix[i]
	}
}

// signFlip reports whether two adjacent elevations differ by more than half the
// signed range. C promotes both SBYTEs to int before subtracting.
func signFlip(a, b byte) bool {
	d := int32(int8(a)) - int32(int8(b))
	return d > 128 || d < -128
}

func (e Elevation) dither(r *Rand) {
	var rv uint32
	pix := e.pix
	for i := range pix {
		if i&3 == 0 {
			rv = r.Uint32()
		} else {
			rv >>= 8 // logical shift, like C's DWORD
		}
		pix[i] += byte(ditherVariance/2 - int(rv&(ditherVariance-1)))
	}
}
