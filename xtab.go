package uqmplanetgen

// XTab is UQM's elevation-to-palette-index translation table (a .xlt file):
// entry d is the absolute palette index for elevation level d.
type XTab [256]byte

// IdentityXTab maps every elevation level straight to the palette index of the
// same number.
func IdentityXTab() *XTab {
	var x XTab
	for d := range x {
		x[d] = byte(d)
	}
	return &x
}

// xtabItemSize is the on-disk XLAT_DESC size: three int16 elevation
// breakpoints, which nothing here uses, followed by the table itself.
const (
	xtabHeaderSize = 3 * 2
	xtabItemSize   = xtabHeaderSize + 256
)

// LoadXTab parses item variant of a BINTAB .xlt buffer. The index is taken
// modulo the item count, as UQM does; shipped .xlt files hold a single item,
// so temperature never changes the elevation-to-index mapping.
func LoadXTab(data []byte, variant int) (XTab, error) {
	item, ok := bintabVariant(data, variant)
	if !ok || len(item) < xtabItemSize {
		return XTab{}, loadErr("xlat table", variant)
	}

	var x XTab
	copy(x[:], item[xtabHeaderSize:xtabItemSize])
	return x, nil
}
