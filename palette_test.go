package uqmplanetgen

import (
	"encoding/binary"
	"testing"
)

// These tests pin the observable palette behaviour: which bytes a .xlt maps to
// which palette index, and which pixel a palette index produces. They are
// written against rendered output rather than struct fields so that they
// survive changes to how XTab and CTab store their data.

// bintab assembles a BINTAB buffer holding the given items.
func bintab(items ...[]byte) []byte {
	out := make([]byte, 0, 64)
	out = binary.BigEndian.AppendUint32(out, 0xFFFFFFFF)
	out = binary.BigEndian.AppendUint32(out, uint32(len(items)))
	out = binary.BigEndian.AppendUint32(out, 0) // placeholder, in DWORD units
	for _, it := range items {
		out = binary.BigEndian.AppendUint32(out, uint32(len(it)))
	}
	for _, it := range items {
		out = append(out, it...)
	}
	return out
}

// xltItem builds one .xlt payload: three little-endian breakpoints followed by
// the 256-byte elevation-to-index table.
func xltItem(xlat []byte) []byte {
	item := make([]byte, 6, xtabFixtureSize)
	binary.LittleEndian.PutUint16(item[0:], 11)
	binary.LittleEndian.PutUint16(item[2:], 22)
	binary.LittleEndian.PutUint16(item[4:], 33)
	return append(item, xlat...)
}

const xtabFixtureSize = 6 + 256

// ctItem builds one .ct payload: first index, last index, then 6-bit triplets.
func ctItem(firstIndex byte, rgb ...byte) []byte {
	return append([]byte{firstIndex, firstIndex + byte(len(rgb)/3) - 1}, rgb...)
}

// fixturePalette is the shared probe palette. Elevation bytes 0..3 land on the
// four defined colors; byte 4 lands on palette index 200, which the table does
// not define.
func fixturePalette(t *testing.T) (*XTab, CTab) {
	t.Helper()

	xlat := make([]byte, 256)
	xlat[0] = 128
	xlat[1] = 129
	xlat[2] = 130
	xlat[3] = 131
	xlat[4] = 200
	xlat[250] = 129 // reachable through the rocky sign-extend path

	x, err := LoadXTab(bintab(xltItem(xlat)), 0)
	if err != nil {
		t.Fatalf("LoadXTab: %v", err)
	}

	ct, err := LoadCTab(bintab(
		ctItem(128, 0, 0, 0), // variant 0, deliberately a different shape
		ctItem(128,
			0, 0, 0, // 128: black
			63, 0, 0, // 129: full red
			0, 63, 0, // 130: full green
			32, 16, 8, // 131: a mixed value
		),
		ctItem(128, 63, 63, 63), // variant 2
	), 1)
	if err != nil {
		t.Fatalf("LoadCTab: %v", err)
	}
	return &x, ct
}

// probe renders a one-row map of the given elevation bytes and returns the
// resulting pixels. gasGiant selects the path that maps an elevation byte
// straight to a palette index.
func probe(t *testing.T, xtab *XTab, ctab CTab, gasGiant bool, base int, elev ...byte) [][4]byte {
	t.Helper()
	e, err := NewElevationFrom(elev, len(elev), 1)
	if err != nil {
		t.Fatalf("NewElevationFrom: %v", err)
	}
	img, err := Render(e, gasGiant, base, xtab, ctab)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := make([][4]byte, len(elev))
	for i := range out {
		copy(out[i][:], img.Pix[4*i:])
	}
	return out
}

func assertPixels(t *testing.T, got [][4]byte, want [][4]byte) {
	t.Helper()
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pixel %d = %v, want %v", i, got[i], want[i])
		}
	}
}

// TestPaletteMapping pins the elevation-to-color path for defined colors. The
// 6-bit channels drop their low bit and are expanded by cc5to8, so 63 becomes
// 255 and 32/16/8 become 132/66/33.
func TestPaletteMapping(t *testing.T) {
	xtab, ctab := fixturePalette(t)
	got := probe(t, xtab, ctab, true, 0, 0, 1, 2, 3)
	assertPixels(t, got, [][4]byte{
		{0, 0, 0, 255},
		{255, 0, 0, 255},
		{0, 255, 0, 255},
		{132, 66, 33, 255},
	})
}

// TestRockyBaseElevation pins the rocky path: the elevation byte is
// sign-extended, offset by the base elevation and clamped to [0,255] before
// the translation table is consulted.
func TestRockyBaseElevation(t *testing.T) {
	xtab, ctab := fixturePalette(t)

	// base 0: 0 -> 0, 1 -> 1, 0xFE (-2) clamps to 0.
	assertPixels(t, probe(t, xtab, ctab, false, 0, 0, 1, 0xFE), [][4]byte{
		{0, 0, 0, 255},
		{255, 0, 0, 255},
		{0, 0, 0, 255},
	})

	// base 3: 0 -> 3, 0xFF (-1) -> 2, 0xFE (-2) -> 1.
	assertPixels(t, probe(t, xtab, ctab, false, 3, 0, 0xFF, 0xFE), [][4]byte{
		{132, 66, 33, 255},
		{0, 255, 0, 255},
		{255, 0, 0, 255},
	})

	// The offset saturates rather than wrapping: 127 + 200 clamps to 255.
	assertPixels(t, probe(t, xtab, ctab, false, 200, 127, 50), [][4]byte{
		{0, 0, 0, 0},     // level 255 maps to palette index 0, which is undefined
		{255, 0, 0, 255}, // 50 + 200 = 250, which maps to 129
	})
}

// TestUndefinedPaletteIndex pins what happens when the translation table
// points outside the colormap. A CTab is indexed by absolute palette index and
// holds 0 where the source file defines nothing, so the pixel is fully
// transparent. The colormap-relative tables this replaced clamped to the
// nearest defined color instead.
func TestUndefinedPaletteIndex(t *testing.T) {
	xtab, ctab := fixturePalette(t)

	// Elevation 4 maps to palette index 200; the fixture defines 128..131.
	assertPixels(t, probe(t, xtab, ctab, true, 0, 4), [][4]byte{{0, 0, 0, 0}})

	if got := ctab.At(200); got != 0 {
		t.Errorf("At(200) = %#08x, want 0", got)
	}
	for _, i := range []int{-1, len(ctab), 1 << 20} {
		if got := ctab.At(i); got != 0 {
			t.Errorf("At(%d) = %#08x, want 0", i, got)
		}
	}
	if got, want := ctab.At(129), Color(255, 0, 0, 255); got != want {
		t.Errorf("At(129) = %#08x, want %#08x", got, want)
	}
}

// TestCTabIsAbsolutelyIndexed pins the layout: a loaded table covers the whole
// palette range and places its colors at the indexes the file names.
func TestCTabIsAbsolutelyIndexed(t *testing.T) {
	ct, err := LoadCTab(bintab(ctItem(200, 63, 0, 0, 0, 63, 0)), 0)
	if err != nil {
		t.Fatalf("LoadCTab: %v", err)
	}
	if len(ct) != 256 {
		t.Fatalf("len = %d, want 256", len(ct))
	}
	if ct[200] != Color(255, 0, 0, 255) || ct[201] != Color(0, 255, 0, 255) {
		t.Errorf("colors landed at the wrong indexes: %#08x %#08x", ct[200], ct[201])
	}
	for i, c := range ct {
		if i != 200 && i != 201 && c != 0 {
			t.Fatalf("index %d = %#08x, want 0", i, c)
		}
	}

	if _, err := LoadCTab(bintab(ctItem(255, 63, 0, 0, 0, 63, 0)), 0); err == nil {
		t.Error("LoadCTab accepted colors running past the end of the palette")
	}
}

// TestVariantSelection pins the modulo UQM applies to the item index.
func TestVariantSelection(t *testing.T) {
	data := bintab(
		ctItem(128, 63, 0, 0),
		ctItem(128, 0, 63, 0),
		ctItem(128, 0, 0, 63),
	)
	xlat := make([]byte, 256)
	xlat[0] = 128
	x, err := LoadXTab(bintab(xltItem(xlat)), 7) // one item: any index selects it
	if err != nil {
		t.Fatalf("LoadXTab: %v", err)
	}

	for _, tc := range []struct {
		variant int
		want    [4]byte
	}{
		{0, [4]byte{255, 0, 0, 255}},
		{1, [4]byte{0, 255, 0, 255}},
		{2, [4]byte{0, 0, 255, 255}},
		{4, [4]byte{0, 255, 0, 255}}, // 4 % 3 == 1
	} {
		ct, err := LoadCTab(data, tc.variant)
		if err != nil {
			t.Fatalf("LoadCTab variant %d: %v", tc.variant, err)
		}
		if got := probe(t, &x, ct, true, 0, 0)[0]; got != tc.want {
			t.Errorf("variant %d = %v, want %v", tc.variant, got, tc.want)
		}
	}
}

// TestLoadRejectsBadData pins the guards around malformed tables.
func TestLoadRejectsBadData(t *testing.T) {
	good := bintab(xltItem(make([]byte, 256)))

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"header only", good[:8]},
		{"truncated payload", good[:len(good)-10]},
		{"wrong prefix", append([]byte{0, 0, 0, 0}, good[4:]...)},
	} {
		if _, err := LoadXTab(tc.data, 0); err == nil {
			t.Errorf("LoadXTab accepted %s", tc.name)
		}
	}

	if _, err := LoadCTab(bintab(ctItem(128, 1, 2)), 0); err == nil {
		t.Error("LoadCTab accepted a payload that is not whole triplets")
	}
	if _, err := LoadCTab(nil, 0); err == nil {
		t.Error("LoadCTab accepted empty data")
	}
}

// TestIdentityXTab pins the identity table: every elevation level maps to the
// palette index of the same number.
func TestIdentityXTab(t *testing.T) {
	x := IdentityXTab()
	rgb := make([]byte, 256*3)
	for i := 0; i < 256; i++ {
		rgb[i*3] = byte(i % 64) // 6-bit red ramp
	}
	ct, err := NewCTab(0, rgb)
	if err != nil {
		t.Fatalf("NewCTab: %v", err)
	}

	got := probe(t, x, ct, true, 0, 0, 1, 2, 63, 64)
	want := [][4]byte{
		{0, 0, 0, 255},
		{0, 0, 0, 255},   // 6-bit 1 -> 5-bit 0
		{8, 0, 0, 255},   // 6-bit 2 -> 5-bit 1 -> cc5to8 8
		{255, 0, 0, 255}, // 6-bit 63 -> 5-bit 31 -> 255
		{0, 0, 0, 255},   // the ramp wraps
	}
	assertPixels(t, got, want)
}
