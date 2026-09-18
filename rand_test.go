package uqmplanetgen

import "testing"

// TestRandKnownSequence pins the stream to the classic MINSTD values for seed 1.
func TestRandKnownSequence(t *testing.T) {
	r := NewRand(1)
	for i, want := range []uint32{16807, 282475249, 1622650073} {
		if got := r.Uint32(); got != want {
			t.Fatalf("draw %d = %d, want %d", i+1, got, want)
		}
	}
}

// TestSeedReduction checks UQM's seed normalisation: 0 becomes 1, and values
// above M are reduced by M exactly once.
func TestSeedReduction(t *testing.T) {
	for _, seed := range []uint32{0, 1, randM + 1} {
		if got := NewRand(seed).Uint32(); got != 16807 {
			t.Errorf("NewRand(%d): first draw = %d, want 16807", seed, got)
		}
	}
}

// TestRandDeterministicAndBounded checks that the stream is reproducible, stays
// inside [1, M-1], and differs between adjacent seeds.
func TestRandDeterministicAndBounded(t *testing.T) {
	draw := func(seed uint32, n int) []uint32 {
		r := NewRand(seed)
		out := make([]uint32, n)
		for i := range out {
			out[i] = r.Uint32()
		}
		return out
	}

	a := draw(987654321, 64)
	b := draw(987654321, 64)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("draw %d differs across identical seeds: %d vs %d", i, a[i], b[i])
		}
		if a[i] < 1 || a[i] >= randM {
			t.Fatalf("draw %d = %d outside [1, M-1]", i, a[i])
		}
	}

	c := draw(987654322, 64)
	same := true
	for i := range a {
		if a[i] != c[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("seeds 987654321 and 987654322 produced identical streams")
	}
}
