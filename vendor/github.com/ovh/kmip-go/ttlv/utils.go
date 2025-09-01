package ttlv

import (
	"fmt"
	"math/big"
	"math/bits"
	"strconv"
	"strings"
)

func padForLen(l, padSize int) int {
	return (padSize - l%padSize) % padSize
}

func bigIntToBytes(value *big.Int, padding int) (b []byte, padVal byte, padLen int) {
	if padding < 1 {
		padding = 1
	}
	b = value.Bytes()
	padVal = byte(0)
	padLen = padForLen(len(b), padding)
	if value.Sign() < 0 {
		padVal = 0xFF
		carry := byte(1)
		for i := len(b) - 1; i >= 0; i-- {
			b[i] = ^b[i] + carry
			if carry > 0 && b[i] != 0 {
				carry = 0
			}
		}
	} else if value.Sign() == 0 {
		return []byte{}, 0, padding
	}
	if (b[0]>>7)&1 != (padVal&1) && padLen == 0 {
		padLen = padding
	}
	return b, padVal, padLen
}

func bytesToBigInt(v []byte) *big.Int {
	if bits.LeadingZeros8(v[0]) > 0 {
		// Positive integer
		bv := big.NewInt(0).SetBytes(v)
		return bv
	}
	// Negative integer
	bv := big.NewInt(0)
	carry := byte(1)
	for i := len(v) - 1; i >= 0; i-- {
		v[i] = ^(v[i] - carry)
		if carry > 0 && v[i] != 0 {
			carry = 0
		}
	}
	bv.SetBytes(v)
	bv.Neg(bv)
	return bv
}

// revMap reverses the given map. Values in the map must be unique or the function will panic.
func revMap[K, V comparable](m map[K]V) map[V]K {
	res := make(map[V]K, len(m))
	for k, v := range m {
		if _, ok := res[v]; ok {
			panic(fmt.Sprintf("Duplicate map key: %+v", v))
		}
		res[v] = k
	}
	return res
}

func parseInt(val string, bits int) (int64, error) {
	if strings.HasPrefix(val, "0x") {
		ui, err := strconv.ParseUint(val[2:], 16, bits)
		return int64(ui), err
	} else {
		return strconv.ParseInt(val, 10, bits)
	}
}

func parseUint(val string, bits int) (uint64, error) {
	if strings.HasPrefix(val, "0x") {
		ui, err := strconv.ParseUint(val[2:], 16, bits)
		return ui, err
	} else {
		return strconv.ParseUint(val, 10, bits)
	}
}
