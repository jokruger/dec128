package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// indexExp returns the index of the exponent marker in s, or -1 if there is none.
func indexExp[S string | []byte](s S) int {
	for i := range len(s) {
		if c := s[i]; c == 'e' || c == 'E' {
			return i
		}
	}
	return -1
}

// maxExpDigits caps the exponent accumulator. Anything at or beyond this magnitude is out of range for a Dec128
// regardless of the mantissa, so the value is saturated instead of being allowed to overflow int.
const maxExpDigits = 1_000_000

// parseExp parses the exponent part of a decimal in scientific notation, i.e. everything following the marker.
func parseExp[S string | []byte](s S) (int, bool) {
	sz := len(s)
	if sz == 0 {
		return 0, false
	}

	i := 0
	neg := false

	switch s[0] {
	case '+':
		i++
	case '-':
		neg = true
		i++
	}

	if i >= sz {
		return 0, false
	}

	exp := 0
	for ; i < sz; i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		if exp < maxExpDigits {
			exp = exp*10 + int(c-'0')
		}
	}

	if neg {
		return -exp, true
	}

	return exp, true
}

// fromSciString parses a decimal in scientific notation, where k is the index of the exponent marker. Only FromString
// reaches this, so the format is always checked.
func fromSciString[S string | []byte](s S, k int) Dec128 {
	exp, ok := parseExp(s[k+1:])
	if !ok {
		return Dec128{state: state.InvalidFormat}
	}

	i := 0
	st := state.Default

	switch s[0] {
	case '+':
		i++
	case '-':
		st = state.Neg
		i++
	}

	// locate the decimal point inside the mantissa
	j := k
	for m := i; m < k; m++ {
		if s[m] == '.' {
			j = m
			break
		}
	}

	digits := k - i
	if j < k {
		// the decimal point is not a digit
		digits--
	}
	if digits == 0 {
		// the mantissa holds no digits at all
		return Dec128{state: state.InvalidFormat}
	}

	// Assemble the mantissa's coefficient. A mantissa may be written with more digits than a coefficient holds, the
	// surplus being trailing zeros that the exponent absorbs: 1e0, 10.0e-1 and 100e-2 all name 1. Give up the smallest
	// number of those zeros that lets the digits fit, so such a mantissa still names its value rather than being
	// rejected; padding that already fits is left alone, so 1.50e0 keeps its scale of 2. This is the reduction
	// applyExp performs below for a scale above MaxScale, and the one DecodePgNumeric, DecodeInt128 and DecodeIEEE
	// perform on import.
	//
	// iEnd and fEnd bound the integer and fractional digits still in play. Dropping a fractional zero lowers the
	// number of decimal places the mantissa carries; once the fraction is gone, dropping an integer zero raises the
	// exponent instead.
	iEnd, fEnd := j, k
	for {
		coef, frac, e := sciCoef(s, i, iEnd, j, fEnd)
		if e < state.Error {
			return applyExp(coef, frac, exp, st)
		}
		if e != state.Overflow {
			return Dec128{state: e}
		}
		switch {
		case fEnd > j+1 && s[fEnd-1] == '0':
			fEnd--
		case fEnd <= j+1 && iEnd > i && s[iEnd-1] == '0':
			iEnd--
			exp++
		default:
			return Dec128{state: state.Overflow}
		}
	}
}

// sciCoef assembles the coefficient of a mantissa whose integer digits are s[i:iEnd] and whose fractional digits are
// s[j+1:fEnd], the decimal point sitting at j; there is no fraction when fEnd is not past j+1. It returns the
// coefficient and the number of decimal places it carries.
func sciCoef[S string | []byte](s S, i, iEnd, j, fEnd int) (uint128.Uint128, int, state.State) {
	var fpart uint128.Uint128
	frac := 0
	if fEnd > j+1 {
		frac = fEnd - j - 1
		var e state.State
		if fpart, e = uint128.FromString(s[j+1 : fEnd]); e >= state.Error {
			return uint128.Zero, 0, e
		}
	}

	ipart, e := uint128.FromString(s[i:iEnd])
	if e >= state.Error {
		return uint128.Zero, 0, e
	}
	if ipart.IsZero() {
		// no integer digits to shift the fraction past, so the fraction is the whole coefficient however long it is
		return fpart, frac, state.OK
	}
	if frac >= len(Pow10Uint128) {
		// shifting a nonzero integer part by this many places cannot fit whatever the digits are
		return uint128.Zero, 0, state.Overflow
	}

	coef, e := ipart.Mul(Pow10Uint128[frac])
	if e >= state.Error {
		return uint128.Zero, 0, e
	}
	if coef, e = coef.Add(fpart); e >= state.Error {
		return uint128.Zero, 0, e
	}

	return coef, frac, state.OK
}

// applyExp builds a Dec128 from a mantissa coefficient, the number of fractional digits the mantissa carried and the
// parsed exponent.
func applyExp(coef uint128.Uint128, frac int, exp int, st state.State) Dec128 {
	// zero is zero at any exponent, and never negative
	if coef.IsZero() {
		return Zero
	}

	scale := frac - exp

	if scale < 0 {
		// the value has more integer digits than the mantissa: shift the coefficient up
		if -scale >= len(Pow10Uint128) {
			return Dec128{state: state.Overflow}
		}
		c, s := coef.Mul(Pow10Uint128[-scale])
		if s >= state.Error {
			return Dec128{state: s}
		}
		return Dec128{coef: c, scale: 0, state: st}
	}

	// trailing zeros in the coefficient can often bring an out of range scale back in
	coef, s, ok := shedScale(coef, scale)
	if !ok {
		return Dec128{state: state.ScaleOutOfRange}
	}

	return Dec128{coef: coef, scale: s, state: st}
}

// StringSci returns the scientific notation representation of the Dec128, with the trailing zeros of the mantissa
// removed. If the Dec128 is zero, the string "0e+0" is returned. If the Dec128 is NaN, the string "NaN" is returned.
func (d Dec128) StringSci() string {
	buf := [MaxSciStrLen]byte{}
	return string(d.StringSciToBuf(buf[:]))
}

// StringSciToBuf returns the scientific notation representation of the Dec128, with the trailing zeros of the mantissa
// removed. If the Dec128 is zero, the string "0e+0" is returned. If the Dec128 is NaN, the string "NaN" is returned.
func (d Dec128) StringSciToBuf(buf []byte) []byte {
	buf = buf[:0]

	switch {
	case d.state >= state.Error:
		return append(buf, NaNStr...)
	case d.coef.IsZero():
		return append(buf, ZeroSciStr...)
	}

	return d.appendStringSci(buf)
}

// appendStringSci appends the scientific notation representation of the decimal to sb. called only when d is not NaN
// and d.coef is not zero
func (d Dec128) appendStringSci(sb []byte) []byte {
	buf := [uint128.MaxStrLen]byte{}
	coef := d.coef.StringToBuf(buf[:])

	if d.state == state.Neg {
		sb = append(sb, '-')
	}

	// the coefficient never has leading zeros, so the exponent of its leading digit does not depend on how many
	// trailing zeros are dropped below
	exp := len(coef) - 1 - int(d.scale)

	n := len(coef)
	for n > 1 && coef[n-1] == '0' {
		n--
	}

	sb = append(sb, coef[0])
	if n > 1 {
		sb = append(sb, '.')
		sb = append(sb, coef[1:n]...)
	}

	sb = append(sb, 'e')
	if exp < 0 {
		sb = append(sb, '-')
		exp = -exp
	} else {
		sb = append(sb, '+')
	}

	// the coefficient holds at most uint128.MaxStrLen digits and the scale is at most MaxScale, so the exponent is
	// always in [-19, 38] and needs at most two digits
	if exp >= 10 {
		sb = append(sb, byte('0'+exp/10))
		exp %= 10
	}

	return append(sb, byte('0'+exp))
}
