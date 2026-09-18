package uqmplanetgen

// Park-Miller (MINSTD) constants: M = 2^31-1, A = 16807, Q = M/A, R = M%A.
const (
	randM uint32 = 1<<31 - 1
	randA uint32 = 16807
	randQ uint32 = 127773
	randR uint32 = 2836
)

// Rand is UQM's global PRNG (TFB_Random in libs/math/random.c) as a value, so
// that each generation run owns its stream. It is not safe for concurrent use;
// give every goroutine its own. Always build one with [NewRand]: the zero
// value has no seed.
type Rand struct {
	state uint32
}

// NewRand returns a generator seeded as TFB_SeedRandom would.
func NewRand(seed uint32) *Rand {
	r := new(Rand)
	r.Seed(seed)
	return r
}

// Seed restarts the stream. Seeds above M are reduced by M once, and 0 becomes 1.
func (r *Rand) Seed(seed uint32) {
	if seed > randM {
		seed -= randM
	}
	if seed == 0 {
		seed = 1
	}
	r.state = seed
}

// Uint32 returns the next value of the stream, in [1, M-1].
func (r *Rand) Uint32() uint32 {
	// Wraps mod 2^32, exactly like the original DWORD arithmetic.
	r.state = randA*(r.state%randQ) - randR*(r.state/randQ)
	switch {
	case r.state > randM:
		r.state -= randM
	case r.state == 0:
		r.state = 1
	}
	return r.state
}
