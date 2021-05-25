// Public Domain (-) 2010-present, The Web4 Authors.
// See the Web4 UNLICENSE file for details.

// Package lexinum provides lexicographically sortable encoding of numbers.
package lexinum

import (
	"fmt"
)

const maxSmall = 8258175

// DecodeHeight decodes the height value from the given byte slice.
func DecodeHeight(d []byte) (uint32, error) {
	if len(d) < 2 {
		return 0, fmt.Errorf("lexinum: invalid encoding")
	}
	if d[0] < 0x80 {
		return 0, fmt.Errorf("lexinum: invalid first byte %q", d[0])
	}
	// Handle big integers.
	if d[0] == 0xff {
		var (
			rem uint32
			v   uint32
		)
		ridx := len(d) - 4
		for i, char := range d[2:] {
			if char == 0x01 {
				if i != ridx {
					return 0, fmt.Errorf("lexinum: invalid encoding")
				}
				rem = uint32(d[i+3])
				break
			}
			v += (v * 252) + uint32(char-2)
		}
		return (v * 255) + rem + maxSmall, nil
	}
	// Handle small integers.
	if len(d) < 3 {
		return 0, fmt.Errorf("lexinum: invalid encoding")
	}
	return (((uint32(d[0]-128) * 255) + uint32(d[1]-1)) * 255) + uint32(d[2]-1), nil
}

// EncodeHeight encodes the given height so that it is lexicographically
// comparable in numeric order.
func EncodeHeight(v uint32) []byte {
	if v == 0 {
		return []byte{0x80, 0x01, 0x01}
	}
	// Handle small integers.
	if v < maxSmall {
		enc := []byte{0x80, 0x01, 0x01}
		div, mod := v/255, v%255
		enc[2] = byte(mod + 1)
		if div > 0 {
			div, mod = div/255, div%255
			enc[1] = byte(mod + 1)
			if div > 0 {
				enc[0] = byte(div + 128)
			}
		}
		return enc
	}
	// Handle big integers.
	v -= maxSmall
	enc := []byte{0xff, 0xff}
	lead, rem := v/255, v%255
	n := uint32(1)
	for lead/pow(253, n) > 0 {
		n++
	}
	enc[1] = byte(n) + 1
	var (
		chars []byte
		mod   uint32
	)
	for {
		if lead == 0 {
			break
		}
		lead, mod = lead/253, lead%253
		chars = append(chars, byte(mod+2))
	}
	for i := len(chars) - 1; i >= 0; i-- {
		enc = append(enc, chars[i])
	}
	if rem > 0 {
		enc = append(enc, 0x01, byte(rem))
	}
	return enc
}

func pow(a, b uint32) uint32 {
	n := uint32(1)
	for b > 0 {
		if b&1 != 0 {
			n *= a
		}
		b >>= 1
		a *= a
	}
	return n
}
