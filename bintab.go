package uqmplanetgen

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// bintabUncompressed is the only prefix UQM uses for planet tables; the format
// also allows RLE, which these files never use.
const bintabUncompressed = 0xFFFFFFFF

var errLoadFailed = errors.New("bad BINTAB header or undersized payload")

func loadErr(kind string, variant int) error {
	return fmt.Errorf("uqmplanetgen: %s variant %d: %w", kind, variant, errLoadFailed)
}

// bintabCount returns the number of items a BINTAB buffer declares, or 0 if
// the prefix or size is wrong. All header words are big-endian.
func bintabCount(p []byte) int {
	if len(p) < 8 || binary.BigEndian.Uint32(p[0:4]) != bintabUncompressed {
		return 0
	}
	return int(binary.BigEndian.Uint32(p[4:8]))
}

// bintabItem returns the payload of the given item. The layout is a 4-byte
// 0xFFFFFFFF prefix, the item count, a placeholder in DWORD units, N payload
// lengths, then the payloads back to back; the first one starts at byte
// 12 + 4*N + 4*placeholder.
func bintabItem(p []byte, variant int) ([]byte, bool) {
	if len(p) < 12 || variant < 0 {
		return nil, false
	}
	if binary.BigEndian.Uint32(p[0:4]) != bintabUncompressed {
		return nil, false
	}
	count := int(binary.BigEndian.Uint32(p[4:8]))
	placeholder := int(binary.BigEndian.Uint32(p[8:12]))

	off := 12 + 4*count + 4*placeholder
	if variant >= count || off > len(p) {
		return nil, false
	}

	for i := 0; i <= variant; i++ {
		if 16+4*i > len(p) {
			return nil, false
		}
		length := int(binary.BigEndian.Uint32(p[12+4*i : 16+4*i]))
		if length > len(p)-off {
			return nil, false
		}
		if i == variant {
			return p[off : off+length], true
		}
		off += length
	}
	return nil, false
}

// bintabVariant picks item variant out of a BINTAB buffer, taking the index
// modulo the item count exactly as UQM's SetAbsStringTableIndex does.
func bintabVariant(data []byte, variant int) ([]byte, bool) {
	count := bintabCount(data)
	if count <= 0 {
		return nil, false
	}
	return bintabItem(data, variant%count)
}
