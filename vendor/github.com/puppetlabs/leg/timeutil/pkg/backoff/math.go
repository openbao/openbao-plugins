package backoff

import "math"

// mul64Checked multiplies x and y, returning either math.MinInt64 or
// math.MaxInt64 if the multiplication would overflow.
func mul64Checked(x, y int64) int64 {
	// If the high bit of x^y is set, that means either x or y is negative, but
	// not both. So adding that single bit to math.MaxInt64 will wrap around to
	// math.MinInt64 if the result should be negative based on the sign of the
	// inputs.
	lim := math.MaxInt64 + ((x ^ y) >> 63)

	switch {
	case x == 0 || y == 0:
		return 0
	case x == math.MinInt64 || y == math.MinInt64:
		return lim
	default:
		z := x * y
		if z/y != x {
			return lim
		}

		return z
	}
}
