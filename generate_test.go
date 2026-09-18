package uqmplanetgen

import (
	"bytes"
	"image"
	"sync"
	"testing"
)

// TestGenerateDeterministic checks that the same seed reproduces bit-identical
// maps. Gas giants need height/bands >= 3, so each algorithm gets its own
// plausible parameters.
func TestGenerateDeterministic(t *testing.T) {
	const w, h = 64, 48

	for _, tc := range []struct {
		algo   Algo
		faults int // fault passes, or band count for gas giants
	}{
		{AlgoTopo, 50},
		{AlgoCratered, 50},
		{AlgoGasGiant, 8},
	} {
		t.Run(tc.algo.String(), func(t *testing.T) {
			a, b := NewElevation(w, h), NewElevation(w, h)
			if err := a.Generate(tc.algo, 42, tc.faults, 2, 30); err != nil {
				t.Fatalf("first Generate: %v", err)
			}
			if err := b.Generate(tc.algo, 42, tc.faults, 2, 30); err != nil {
				t.Fatalf("second Generate: %v", err)
			}
			if !bytes.Equal(a.Pix(), b.Pix()) {
				t.Fatal("the same seed produced different maps")
			}
		})
	}
}

// TestGenerateConcurrent checks that generation keeps no shared state: maps
// produced in parallel must match ones produced one at a time.
func TestGenerateConcurrent(t *testing.T) {
	const w, h, n = 64, 48, 8

	want := make([]Elevation, n)
	for i := range want {
		want[i] = NewElevation(w, h)
		if err := want[i].Generate(AlgoCratered, uint32(i+1), 50, 2, 30); err != nil {
			t.Fatalf("Generate: %v", err)
		}
	}

	got := make([]Elevation, n)
	var wg sync.WaitGroup
	for i := range got {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got[i] = NewElevation(w, h)
			got[i].Generate(AlgoCratered, uint32(i+1), 50, 2, 30)
		}(i)
	}
	wg.Wait()

	for i := range got {
		if !bytes.Equal(got[i].Pix(), want[i].Pix()) {
			t.Fatalf("map %d differs when generated concurrently", i)
		}
	}
}

// grayPalette returns an identity translation table and a 6-bit grayscale ramp
// (UQM colormaps store 6-bit channels).
func grayPalette(t *testing.T) (*XTab, CTab) {
	t.Helper()
	rgb := make([]byte, 256*3)
	for d := range rgb {
		rgb[d] = byte(d / 3 * 63 / 255)
	}
	ct, err := NewCTab(0, rgb)
	if err != nil {
		t.Fatalf("NewCTab: %v", err)
	}
	return IdentityXTab(), ct
}

func assertNotBlack(t *testing.T, img *image.RGBA) {
	t.Helper()
	for i := 0; i+2 < len(img.Pix); i += 4 {
		if img.Pix[i]|img.Pix[i+1]|img.Pix[i+2] != 0 {
			return
		}
	}
	t.Fatal("rendered image is entirely black")
}

// TestRenderSmoke renders a rocky world and a gas giant end to end.
func TestRenderSmoke(t *testing.T) {
	const w, h = 64, 20
	xtab, ctab := grayPalette(t)

	for _, tc := range []struct {
		algo          Algo
		faults        int
		blemishes     int
		gasGiant      bool
		baseElevation int
	}{
		{AlgoCratered, 80, 40, false, 150},
		{AlgoGasGiant, 4, 0, true, 29},
	} {
		t.Run(tc.algo.String(), func(t *testing.T) {
			elev := NewElevation(w, h)
			if err := elev.Generate(tc.algo, 7, tc.faults, 2, tc.blemishes); err != nil {
				t.Fatalf("Generate: %v", err)
			}
			img, err := Render(elev, tc.gasGiant, tc.baseElevation, xtab, ctab)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			if got := img.Bounds(); got.Dx() != w || got.Dy() != h {
				t.Fatalf("bounds %v, want %dx%d", got, w, h)
			}
			assertNotBlack(t, img)
		})
	}
}

// TestPresets checks that every embedded colormap parses and renders.
func TestPresets(t *testing.T) {
	presets := Presets()
	if len(presets) == 0 {
		t.Fatal("no presets")
	}
	for i := range presets {
		p := &presets[i]
		if p.XTab == nil || p.CTab == nil {
			t.Errorf("%s: missing colormap", p.Name)
			continue
		}
		if _, err := p.Render(DefaultWidth, DefaultHeight); err != nil {
			t.Errorf("%s: %v", p.Name, err)
		}
	}

	if _, err := PresetPlanet("water_world"); err != nil {
		t.Errorf("case-insensitive lookup: %v", err)
	}
	if _, err := PresetPlanet("NO_SUCH_WORLD"); err == nil {
		t.Error("PresetPlanet of an unknown name returned no error")
	}
}

// TestElevationDims checks the dimensions a map carries and the guards around
// a buffer that does not match them.
func TestElevationDims(t *testing.T) {
	e := NewElevation(7, 5)
	if e.Width() != 7 || e.Height() != 5 || len(e.Pix()) != 35 {
		t.Fatalf("got %dx%d with %d bytes, want 7x5 with 35", e.Width(), e.Height(), len(e.Pix()))
	}
	e.Set(3, 2, -9)
	if got := e.At(3, 2); got != -9 {
		t.Errorf("At(3,2) = %d, want -9", got)
	}
	if e.Pix()[2*7+3] != 0xF7 { // -9 as two's-complement
		t.Error("Set did not write the row-major position")
	}

	if _, err := NewElevationFrom(make([]byte, 34), 7, 5); err == nil {
		t.Error("NewElevationFrom accepted an undersized buffer")
	}
	if _, err := NewElevationFrom(make([]byte, 35), 7, 5); err != nil {
		t.Errorf("NewElevationFrom: %v", err)
	}
	if err := NewElevation(0, 5).Generate(AlgoTopo, 1, 10, 1, 0); err == nil {
		t.Error("Generate accepted a zero-width map")
	}
}

// TestGenerateRejectsUnusableSizes pins the guards around parameters whose
// modulo arithmetic would divide by zero. These used to panic, which in a
// WebAssembly build traps the whole module.
func TestGenerateRejectsUnusableSizes(t *testing.T) {
	for _, tc := range []struct {
		name      string
		w, h      int
		algo      Algo
		faults    int
		blemishes int
	}{
		{"gas giant with more bands than rows", 242, 75, AlgoGasGiant, 37, 0},
		{"gas giant with one row per band", 242, 75, AlgoGasGiant, 75, 0},
		{"gas giant with no bands", 242, 75, AlgoGasGiant, 0, 0},
		{"gas giant with a negative band count", 242, 75, AlgoGasGiant, -4, 0},
		{"gas giant too short for storms", 40, 12, AlgoGasGiant, 4, 0},
		{"craters on a map under 16 rows", 40, 12, AlgoCratered, 20, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewElevation(tc.w, tc.h)
			if err := e.Generate(tc.algo, 1, tc.faults, 2, tc.blemishes); err == nil {
				t.Error("Generate accepted it")
			}
		})
	}

	// Craters are only a constraint when some are asked for.
	if err := NewElevation(40, 12).Generate(AlgoCratered, 1, 20, 2, 0); err != nil {
		t.Errorf("a short map with no craters: %v", err)
	}
	// A crater too wide for the map is skipped rather than dividing by zero.
	if err := NewElevation(20, 64).Generate(AlgoCratered, 1, 20, 2, 40); err != nil {
		t.Errorf("a narrow map with craters: %v", err)
	}
	if err := NewElevation(242, 75).GasGiant(NewRand(1), 37, 2); err == nil {
		t.Error("GasGiant accepted more bands than rows")
	}
}

// TestTempVariant pins UQM's palette-variant thresholds.
func TestTempVariant(t *testing.T) {
	for _, tc := range []struct{ temp, want int }{
		{200, 2},
		{101, 2},
		{100, 1}, // not strictly above the hot threshold
		{0, 1},
		{-39, 1},
		{-40, 0}, // not strictly above the cold threshold
		{-80, 0},
	} {
		if got := TempVariant(tc.temp); got != tc.want {
			t.Errorf("TempVariant(%d) = %d, want %d", tc.temp, got, tc.want)
		}
	}
}

func BenchmarkRenderPreset(b *testing.B) {
	p, err := PresetPlanet("WATER_WORLD")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := p.Render(DefaultWidth, DefaultHeight); err != nil {
			b.Fatal(err)
		}
	}
}
