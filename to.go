package dec128

import (
	"strconv"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// EncodeToInt64 returns d * 10^exp as a signed int64, truncated toward zero: the coefficient the value would have at
// scale exp, with its sign. It returns an error carrying the NaN's state for a NaN, state.ScaleOutOfRange for exp above
// MaxScale, and state.Overflow when the result does not fit an int64. Use EncodeInt128 for a conversion that refuses to
// drop digits.
func (d Dec128) EncodeToInt64(exp uint8) (int64, error) {
	if d.state < state.Error && d.coef.IsZero() {
		return 0, nil
	}

	t := d.Rescale(exp)
	if t.state >= state.Error {
		return 0, t.state.Error()
	}

	i, s := t.coef.Uint64()
	if s >= state.Error {
		return 0, s.Error()
	}

	if t.state == state.Neg {
		if i > 9223372036854775808 {
			return 0, state.Overflow.Error()
		}
		return -int64(i), nil
	}

	if i > 9223372036854775807 {
		return 0, state.Overflow.Error()
	}

	return int64(i), nil
}

// EncodeToUint64 returns the Dec128 encoded as uint64 coefficient with requested exponent.
// Negative and too large values are not allowed.
func (d Dec128) EncodeToUint64(exp uint8) (uint64, error) {
	switch {
	case d.state < state.Error && d.coef.IsZero():
		return 0, nil
	case d.state == state.Neg:
		return 0, state.NegativeInUnsignedOp.Error()
	}

	t := d.Rescale(exp)
	if t.state >= state.Error {
		return 0, t.state.Error()
	}

	i, s := t.coef.Uint64()
	if s >= state.Error {
		return 0, s.Error()
	}

	return i, nil
}

// EncodeToUint128 returns the Dec128 encoded as uint128 coefficient with requested exponent.
// Negative values are not allowed.
func (d Dec128) EncodeToUint128(exp uint8) (uint128.Uint128, error) {
	switch {
	case d.state < state.Error && d.coef.IsZero():
		return uint128.Zero, nil
	case d.state == state.Neg:
		return uint128.Zero, state.NegativeInUnsignedOp.Error()
	}

	t := d.Rescale(exp)
	if t.state >= state.Error {
		return uint128.Zero, t.state.Error()
	}

	return t.coef, nil
}

// String returns the string representation of the Dec128 with the trailing zeros removed.
// If the Dec128 is zero, the string "0" is returned. If the Dec128 is NaN, the string "NaN" is returned.
func (d Dec128) String() string {
	buf := [MaxStrLen]byte{}
	return string(d.StringToBuf(buf[:]))
}

// StringToBuf returns the string representation of the Dec128 with the trailing zeros removed.
// If the Dec128 is zero, the string "0" is returned. If the Dec128 is NaN, the string "NaN" is returned.
func (d Dec128) StringToBuf(buf []byte) []byte {
	buf = buf[:0]

	switch {
	case d.state >= state.Error:
		return append(buf, NaNStr...)
	case d.coef.IsZero():
		return append(buf, ZeroStr...)
	}

	sb, trim := d.appendString(buf)
	if trim {
		return trimTrailingZeros(sb)
	}

	return sb
}

// StringFixedToBuf writes the fixed form of the Dec128 (trailing zeros preserved, as StringFixed) into buf and returns
// the slice holding it. buf needs room for MaxStrLen bytes to hold every value. This is the zero-allocation form of the
// text that Value, MarshalJSON and MarshalText emit by default.
func (d Dec128) StringFixedToBuf(buf []byte) []byte {
	buf = buf[:0]

	switch {
	case d.state >= state.Error:
		return append(buf, NaNStr...)
	case d.coef.IsZero():
		return append(buf, zeroStrs[d.scale]...)
	}

	sb, _ := d.appendString(buf)

	return sb
}

// StringFixed returns the string representation of the Dec128 with the trailing zeros preserved.
// If the Dec128 is NaN, the string "NaN" is returned.
func (d Dec128) StringFixed() string {
	switch {
	case d.state >= state.Error:
		return NaNStr
	case d.coef.IsZero():
		return zeroStrs[d.scale]
	}

	buf := [MaxStrLen]byte{}
	sb, _ := d.appendString(buf[:0])

	return string(sb)
}

// Int returns the integer part of the Dec128 as int, truncated toward zero (1.99 is 1, -1.99 is -1). It returns an
// error carrying state.Overflow when the value does not fit an int of the platform, and the NaN's state for a NaN.
func (d Dec128) Int() (int, error) {
	i, err := d.EncodeToInt64(0)
	if err != nil {
		return 0, err
	}
	return int(i), intRangeError(i)
}

// Int64 returns the integer part of the Dec128 as int64, truncated toward zero (1.99 is 1, -1.99 is -1). It returns an
// error carrying state.Overflow when the value does not fit, and the NaN's state for a NaN.
func (d Dec128) Int64() (int64, error) {
	return d.EncodeToInt64(0)
}

// InexactFloat64 returns the float64 representation of the decimal.
// The result may not be 100% accurate due to the limitation of float64.
func (d Dec128) InexactFloat64() (float64, error) {
	if d.state >= state.Error {
		return 0, d.state.Error()
	}
	return strconv.ParseFloat(d.String(), 64)
}
