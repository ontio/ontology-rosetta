// Public Domain (-) 2010-present, The Web4 Authors.
// See the Web4 UNLICENSE file for details.

package lexinum

import (
	"bytes"
	"math"
	"testing"
)

func TestEncoding(t *testing.T) {
	prev := []byte{}
	for i := uint32(0); i < math.MaxUint32; i++ {
		enc := EncodeHeight(i)
		cmp := bytes.Compare(enc, prev)
		if cmp != 1 {
			t.Fatalf("Encoding of %d is not lexicographically greater than the previous value", i)
		}
		prev = enc
		v, err := DecodeHeight(enc)
		if err != nil {
			t.Fatalf("Failed to decode %d: %s", i, err)
		}
		if i != v {
			t.Fatalf("Failed to decode %d: got %d", i, v)
		}
		if i%10000000 == 0 {
			t.Logf("Testing encoding %d...", i)
		}
	}
}
