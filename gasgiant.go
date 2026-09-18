package uqmplanetgen

const (
	maxStorms     = 8
	bandDeltaMask = (1<<rangeShift)*numBandColors - 1

	// maxStormTries bounds the placement search. UQM re-rolls until a storm
	// misses every other one, which never terminates on a map too small to
	// hold them all. The limit is far above what any real map needs, so it
	// cannot change the output of a run that would have finished.
	maxStormTries = 4096
)

// stormRect is the COORD/SIZE rectangle UQM uses for storm placement.
type stormRect struct {
	x, y int16
	w, h int16
}

// GasGiant fills e with a banded gas giant texture: numBands horizontal bands
// of random height, each textured by fault passes on a five-row strip, then
// nested-ring storms and a dither pass. As in UQM, height/numBands must be at
// least 3: the band-edge jitter takes a modulo by bandHeight-2.
func (e Elevation) GasGiant(r *Rand, numBands, depthDelta int) error {
	if err := e.check(); err != nil {
		return err
	}
	if err := e.checkBands(numBands); err != nil {
		return err
	}
	e.gasGiant(r, numBands, depthDelta)
	return nil
}

// gasGiant ports UQM's plangen.c MakeGasGiant. The map is filled top-down;
// every band edge except the bottom one gets 50 fault passes on a five-row
// strip centred on it, then the band's rows are painted with the current
// bandDelta (a +/-64 random walk masked into [0,255]).
func (e Elevation) gasGiant(r *Rand, numBands, depthDelta int) {
	w := int16(e.width)
	h := int16(e.height)

	bandHeight := uint16(int(h) / numBands)
	bandBump := int16(int(h) % numBands)
	bandError := int16(numBands >> 1)

	loword := uint16(r.Uint32())
	bandDelta := int16(int(loword&(numBandColors-1))<<rangeShift + 1<<(rangeShift-1))

	lp := 0
	var lastY, nextY int16
	for i := numBands; i > 0; i-- {
		rv := r.Uint32()
		loword = uint16(rv)
		hiword := uint16(rv >> 16)

		nextY += int16(bandHeight)
		if bandError -= bandBump; bandError < 0 {
			nextY++
			bandError += int16(numBands)
		}

		var curY int16
		if i == 1 {
			curY = h // bottom edge of the map: no jitter, no faults
		} else {
			bh2 := int(bandHeight) - 2
			curY = int16(int(nextY) + (bh2 >> 1) - (int(hiword&0xFF)%bh2 + 1))

			e.strip(int(curY)-2, 5).faults(r, 50, depthDelta)
		}

		// COUNT wraps mod 2^16 if cur_y < last_y, as in C.
		pix := e.pix
		for j := uint16(int(curY) - int(lastY)); j > 0; j-- {
			for k := int(w); k > 0; k-- {
				pix[lp] += byte(bandDelta) // SBYTE += int16: mod-256
				lp++
			}
		}

		lastY = curY
		bandDelta = int16((int32(bandDelta) + (int32(loword&1)<<1-1)*(1<<rangeShift)) & bandDeltaMask)
	}

	e.makeStorms(r, uint16(4+r.Uint32()&3+1)) // 5..8 storms
	e.dither(r)
}

// makeStorms ports UQM's plangen.c MakeStorms. Storms are placed from the
// highest index down; each candidate position is re-rolled until it misses
// every storm already placed by at least a 4px margin. A storm is stamped as
// up to five nested crater rings: an outer 6/6 ring, then inward rings whose
// delta starts at HIBYTE(loword) & ((3 << rangeShift) + 20) and grows by
// +2/+2/+4, each zeroing its row segments first.
func (e Elevation) makeStorms(r *Rand, stormCount uint16) {
	w := int16(e.width)
	h := int16(e.height)

	var storms [maxStorms]stormRect

	for i := int(stormCount); i > 0; {
		i--
		ps := &storms[i]

		intersect := false
		var loword uint16 // survives the roll loop: reused for bandDelta below
		for tries := 0; ; tries++ {
			rv := r.Uint32()
			loword = uint16(rv)
			hiword := uint16(rv >> 16)

			switch hiword >> 8 & 31 {
			case 0:
				ps.h = int16(int(hiword&0xFF)%int(h>>2) + int(h>>2))
			case 1, 2, 3, 4:
				ps.h = int16(int(hiword&0xFF)%int(h>>3) + int(h>>3))
			default:
				ps.h = int16(int(hiword&0xFF)%int(h>>4) + 4) // literal 4, as in UQM
			}
			if ps.h <= 4 { // keep the ellipse drawable
				ps.h += 4
			}

			// Second roll: width and position. C splits off a hiword here too
			// but never uses it.
			loword = uint16(r.Uint32())
			ps.w = ps.h + int16(int(loword&0xFF)%int(ps.h))
			ps.x = int16(int((loword>>8)&0xFF) % int(w-ps.w))
			ps.y = int16(int(loword&0xFF) % int(h-ps.h))

			for j := i + 1; j < int(stormCount); j++ {
				dx := storms[j].x - ps.x
				dy := storms[j].y - ps.y
				intersect = int16(int(dx)+int(storms[j].w)+4) > 0 &&
					int16(int(dy)+int(storms[j].h)+4) > 0 &&
					int(dx) < int(ps.w)+4 && int(dy) < int(ps.h)+4
				if intersect {
					break
				}
			}
			if !intersect || tries >= maxStormTries {
				break
			}
		}

		e.crater(int(ps.x), int(ps.y), int(ps.w), int(ps.h), 6, 6, false)
		ps.shrink()

		bandDelta := int16(int(loword>>8&0xFF) & ((3 << rangeShift) + 20))
		e.crater(int(ps.x), int(ps.y), int(ps.w), int(ps.h),
			int(bandDelta), int(bandDelta), true)
		ps.shrink()

		for _, step := range [...]int16{2, 2} {
			bandDelta += step
			if ps.w > 2 && ps.h > 2 {
				e.crater(int(ps.x), int(ps.y), int(ps.w), int(ps.h),
					int(bandDelta), int(bandDelta), true)
				ps.shrink()
			}
		}

		bandDelta += 4
		e.crater(int(ps.x), int(ps.y), int(ps.w), int(ps.h),
			int(bandDelta), int(bandDelta), true) // innermost ring, always stamped
	}
}

// shrink steps one ring inward.
func (s *stormRect) shrink() {
	s.x++
	s.y++
	s.w -= 2
	s.h -= 2
}
