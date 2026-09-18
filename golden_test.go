package uqmplanetgen

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestGoldenElevation pins the port to elevation maps captured from the
// original C implementation. Regenerate the reference data whenever the C
// sources change.
func TestGoldenElevation(t *testing.T) {
	for _, s := range []struct {
		file   string
		w, h   int
		algo   Algo
		seed   uint32
		faults int
		depth  int
		blem   int
	}{
		{"golden_oolite.elev", 242, 75, AlgoCratered, 42, 230, 2, 200},
		{"golden_rugged.elev", 242, 75, AlgoTopo, 7, 300, 3, 0},
		{"golden_gasgiant.elev", 242, 75, AlgoGasGiant, 99, 10, 2, 0},
		{"golden_cratered_small.elev", 37, 19, AlgoCratered, 5, 40, 1, 12},
		{"golden_topo_small.elev", 37, 19, AlgoTopo, 5, 40, 1, 0},
	} {
		t.Run(s.file, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("testdata", s.file))
			if err != nil {
				t.Fatalf("reading reference data: %v", err)
			}

			e := NewElevation(s.w, s.h)
			if err := e.Generate(s.algo, s.seed, s.faults, s.depth, s.blem); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got := e.Pix(); !bytes.Equal(got, want) {
				i := 0
				for got[i] == want[i] {
					i++
				}
				t.Fatalf("pixel %d = 0x%02X, want 0x%02X", i, got[i], want[i])
			}
		})
	}
}
