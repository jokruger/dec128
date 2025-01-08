package dec128

import (
	"math"
	"strconv"

	"github.com/jokruger/dec128/errors"
	"github.com/jokruger/dec128/uint128"
)

// EncodeToUint64 returns the Dec128 encoded as uint64 coefficient with requested exponent.
// Negative and too large values are not allowed.
func (self Dec128) EncodeToUint64(exp uint8) (uint64, error) {
	if self.neg {
		return 0, errors.Negative.Value()
	}

	d := self.Rescale(exp)

	if d.err != errors.None {
		return 0, d.err.Value()
	}

	i, err := d.coef.Uint64()
	if err != errors.None {
		return 0, err.Value()
	}

	return i, nil
}

// EncodeToUint128 returns the Dec128 encoded as uint128 coefficient with requested exponent.
// Negative values are not allowed.
func (self Dec128) EncodeToUint128(exp uint8) (uint128.Uint128, error) {
	if self.neg {
		return uint128.Zero, errors.Negative.Value()
	}

	d := self.Rescale(exp)

	if d.err != errors.None {
		return uint128.Zero, d.err.Value()
	}

	return d.coef, nil
}

// String returns the string representation of the Dec128 with the trailing zeros removed.
// If the Dec128 is zero, the string "0" is returned.
// If the Dec128 is NaN, the string "NaN" is returned.
func (self Dec128) String() string {
	if self.err != errors.None {
		return NaNStr
	}

	if self.IsZero() {
		return ZeroStr
	}

	buf := [MaxStrLen]byte{}
	sb, trim := self.stringToBuf(buf[:])
	if trim {
		return string(trimTrailingZeros(sb))
	}

	return string(sb)
}

// StringFixed returns the string representation of the Dec128 with the trailing zeros preserved.
// If the Dec128 is NaN, the string "NaN" is returned.
func (self Dec128) StringFixed() string {
	if self.err != errors.None {
		return NaNStr
	}

	if self.IsZero() {
		return zeroStrs[self.exp]
	}

	buf := [MaxStrLen]byte{}
	sb, _ := self.stringToBuf(buf[:])

	return string(sb)
}

// Int returns the integer part of the Dec128 as int.
func (self Dec128) Int() (int, error) {
	t := self.Rescale(0)
	if t.err != errors.None {
		return 0, t.err.Value()
	}
	if t.coef.Hi != 0 {
		return 0, errors.Overflow.Value()
	}
	if t.coef.Lo > math.MaxInt {
		return 0, errors.Overflow.Value()
	}

	if t.neg {
		return -int(t.coef.Lo), nil
	}

	return int(t.coef.Lo), nil
}

// Int64 returns the integer part of the Dec128 as int64.
func (self Dec128) Int64() (int64, error) {
	t := self.Rescale(0)
	if t.err != errors.None {
		return 0, t.err.Value()
	}
	if t.coef.Hi != 0 {
		return 0, errors.Overflow.Value()
	}
	if t.coef.Lo > math.MaxInt64 {
		return 0, errors.Overflow.Value()
	}

	if t.neg {
		return -int64(t.coef.Lo), nil
	}

	return int64(t.coef.Lo), nil
}

// InexactFloat64 returns the float64 representation of the decimal.
// The result may not be 100% accurate due to the limitation of float64 (less decimal precision).
func (self Dec128) InexactFloat64() (float64, error) {
	if self.err != errors.None {
		return 0, self.err.Value()
	}
	return strconv.ParseFloat(self.String(), 64)
}
