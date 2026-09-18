// Package uqmplanetgen generates planet surface textures with the fractal
// algorithms from Star Control II / The Ur-Quan Masters.
//
// An [Elevation] is a row-major map of signed 8-bit heights, one byte per
// pixel, with 0 as sea level. [Elevation.Generate] fills one using one of the
// three UQM algorithms; [Render] turns it into an image using a matching
// colormap pair: an [XTab] elevation-to-index table and a [CTab] palette.
// [Planet] bundles both steps, and the preset table covers the world types UQM
// ships.
//
// Output is bit-identical to the original C. Integer widths follow the
// original types (SBYTE, COUNT, COORD/SIZE, DWORD) so C's implicit 16- and
// 32-bit wraparound is reproduced exactly; the arithmetic in elevation.go,
// crater.go and gasgiant.go is deliberate and is pinned by the golden tests.
//
// Generation draws on a caller-owned [Rand] rather than a package-level
// stream, so planets can be rendered concurrently.
package uqmplanetgen

// DefaultWidth and DefaultHeight are the surface dimensions UQM uses at its
// 320x200 video mode. Generation is resolution independent, but crater and
// storm sizes are tuned for these values.
const (
	DefaultWidth  = 242
	DefaultHeight = 75
)

// Algo selects a generation algorithm (UQM's PLANALGO).
type Algo int

const (
	AlgoTopo     Algo = iota // fault-displacement terrain only
	AlgoCratered             // faults, craters and dithering
	AlgoGasGiant             // banded gas giant with storms
)

func (a Algo) String() string {
	switch a {
	case AlgoTopo:
		return "topo"
	case AlgoCratered:
		return "cratered"
	case AlgoGasGiant:
		return "gasgiant"
	}
	return "unknown"
}

// Palette variant thresholds, in degrees Celsius. See [TempVariant].
const (
	HotThreshold  = 100
	ColdThreshold = -40
)

// TempVariant returns the colormap variant index UQM picks for a surface
// temperature: 2 above [HotThreshold], 0 at or below [ColdThreshold], 1 in
// between.
func TempVariant(tempCelsius int) int {
	switch {
	case tempCelsius > HotThreshold:
		return 2
	case tempCelsius > ColdThreshold:
		return 1
	}
	return 0
}
