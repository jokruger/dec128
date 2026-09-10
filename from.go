package dec128

import (
	"math"
	"strconv"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// FromString creates a new Dec128 from a string.
// In case of empty string, it returns Zero. In case of errors, it returns NaN with the corresponding error.
func FromString[S string | []byte](s S) Dec128 {
	sz := len(s)

	switch sz {
	case 0:
		return Zero
	case 1:
		switch s[0] {
		case '0':
			return Zero
		case '+', '-', '.':
			return Dec128{state: state.InvalidFormat}
		}
	case 2:
		if (s[0] == '+' || s[0] == '-') && s[1] == '.' {
			return Dec128{state: state.InvalidFormat}
		}
	}

	var i, scale int
	var st state.State

	switch s[0] {
	case '+':
		i++
	case '-':
		st = state.Neg
		i++
	}

	if sz <= uint128.MaxSafeStrLen64 {
		// safe to parse with uint64 as coef
		var u uint64
		for ; i < sz; i++ {
			c := s[i]
			if c == '.' {
				if scale != 0 {
					return Dec128{state: state.InvalidFormat}
				}
				scale = sz - i - 1
				continue
			}
			if c < '0' || c > '9' {
				if c == 'e' || c == 'E' {
					return fromSciString(s, i)
				}
				// Only reached by input that is not a number, so valid input never pays for the check: the three
				// spellings PostgreSQL emits for its special values are accepted here.
				return fromSpecial(s)
			}
			u = u*10 + uint64(c-'0')
		}
		if u == 0 {
			return Dec128{coef: uint128.Zero, scale: uint8(scale)}
		}
		return Dec128{coef: uint128.FromUint64(u), scale: uint8(scale), state: st}
	}

	// locate the decimal point, stopping early if an exponent marker comes first
	j := 0
	for ; j < sz; j++ {
		c := s[j]
		if c == '.' {
			break
		}
		if c > '9' {
			// digits, the sign and the decimal point are all <= '9'
			if c == 'e' || c == 'E' {
				return fromSciString(s, j)
			}
			return Dec128{state: state.InvalidFormat}
		}
	}

	if j >= sz-1 {
		// an integer, with or without a trailing point ("123." is accepted, as PostgreSQL and the short path accept it)
		coef, e := uint128.FromString(s[i:j])
		if e >= state.Error {
			return Dec128{state: e}
		}
		if coef.IsZero() {
			return Zero
		}
		return Dec128{coef: coef, scale: 0, state: st}
	}

	// exponent part, if any, is counted into the scale here, which is what pushes a scientific mantissa over the limit
	scale = sz - j - 1
	if scale > uint128.MaxSafeStrLen64 {
		if m := indexExp(s[j+1:]); m >= 0 {
			return fromSciString(s, j+1+m)
		}
		return trimFraction(s, i, j, st)
	}

	ipart, ei := uint128.FromString(s[i:j])
	if ei >= state.Error {
		// The integer part does not fit on its own, so no reduction of the fraction can help - but a negative
		// exponent still can, and the marker is not a digit, so it lands here rather than parsing.
		if m := indexExp(s[j+1:]); m >= 0 {
			return fromSciString(s, j+1+m)
		}
		return Dec128{state: ei}
	}

	fpart, ef := uint128.FromString(s[j+1:])
	if ef >= state.Error {
		// the marker is not a digit, so it lands here rather than parsing
		if m := indexExp(s[j+1:]); m >= 0 {
			return fromSciString(s, j+1+m)
		}
		return Dec128{state: ef}
	}

	// max scale is 19, so the fpart.Hi is always 0 and scale is always <= len(pow10)
	coef, e := ipart.MulAdd64(Pow10Uint64[scale], fpart.Lo)
	if e >= state.Error {
		return trimFraction(s, i, j, st)
	}

	if coef.IsZero() {
		return Dec128{coef: uint128.Zero, scale: uint8(scale)}
	}

	return Dec128{coef: coef, scale: uint8(scale), state: st}
}

// FromSafeString creates a new Dec128 from safe string (no format checks are applied).
// In case of errors, it returns NaN with the corresponding error. Only the regular form is supported here: scientific
// notation is not recognized and would be parsed as if the exponent marker were a digit. Use FromString for input
// that may carry an exponent; it accepts both forms and costs nothing extra to do so.
func FromSafeString[S string | []byte](s S) Dec128 {
	sz := len(s)

	if sz == 0 || (sz == 1 && s[0] == '0') {
		return Zero
	}

	var i, scale int
	var st state.State

	switch s[0] {
	case '+':
		i++
	case '-':
		st = state.Neg
		i++
	}

	if sz <= uint128.MaxSafeStrLen64 {
		// safe to parse with uint64 as coef
		var u uint64
		for ; i < sz; i++ {
			c := s[i]
			if c == '.' {
				scale = sz - i - 1
				continue
			}
			u = u*10 + uint64(c-'0')
		}
		if u == 0 {
			return Dec128{coef: uint128.Zero, scale: uint8(scale)}
		}
		return Dec128{coef: uint128.FromUint64(u), scale: uint8(scale), state: st}
	}

	j := 0
	for j < sz && s[j] != '.' {
		j++
	}

	if j == sz {
		coef, e := uint128.FromSafeString(s[i:])
		if e >= state.Error {
			return Dec128{state: e}
		}
		if coef.IsZero() {
			return Zero
		}
		return Dec128{coef: coef, scale: 0, state: st}
	}

	scale = sz - j - 1
	if scale > uint128.MaxSafeStrLen64 {
		return trimFraction(s, i, j, st)
	}

	ipart, ei := uint128.FromSafeString(s[i:j])
	if ei >= state.Error {
		return Dec128{state: ei}
	}

	// scale is <= MaxSafeStrLen64, so the fractional part fits into uint64
	fpart, _ := uint128.FromSafeString(s[j+1:])

	// unreachable because FromSafeString cannot be error for <= MaxSafeStrLen64 digits
	//if ef >= state.Error {
	//	return Dec128{state: ef}
	//}

	// max scale is 19, so the fpart.Hi is always 0 and scale is always <= len(pow10)
	coef, e := ipart.MulAdd64(Pow10Uint64[scale], fpart.Lo)
	if e >= state.Error {
		return trimFraction(s, i, j, st)
	}

	if coef.IsZero() {
		return Dec128{coef: uint128.Zero, scale: uint8(scale)}
	}

	return Dec128{coef: coef, scale: uint8(scale), state: st}
}

// trimFraction re-parses a regular-form decimal that cannot be held at its written scale, dropping the smallest number
// of trailing zeros from the fraction that makes it fit: the scale has to come down to MaxScale, and the coefficient
// has to fit in 128 bits. s[i:j] is the integer part and s[j+1:] the fraction, and there is at least one fractional
// digit. It returns NaN(ScaleOutOfRange) when the scale cannot come down and NaN(Overflow) when the coefficient still
// does not fit.
//
// Trailing zeros are padding, not information, so a value that is exactly representable at a lower scale is accepted
// rather than rejected - the same reduction DecodePgNumeric, DecodeInt128, DecodeIEEE and applyExp all perform. Only
// input that would otherwise have failed reaches this, so valid input pays nothing for it.
func trimFraction[S string | []byte](s S, i, j int, st state.State) Dec128 {
	frac := s[j+1:]
	scale := len(frac)

	// Digits beyond MaxScale have to go, so they all have to be zeros.
	drop := 0
	if scale > int(MaxScale) {
		drop = scale - int(MaxScale)
	}
	for d := 1; d <= drop; d++ {
		if frac[scale-d] != '0' {
			return Dec128{state: state.ScaleOutOfRange}
		}
	}

	ipart, e := uint128.FromString(s[i:j])
	if e >= state.Error {
		return Dec128{state: e}
	}

	// Widen the drop one zero at a time until the coefficient fits. The kept fraction is at most MaxScale digits from
	// here on, so it always fits a uint64 and one parse per attempt is cheap; the loop runs at most MaxScale+1 times.
	for {
		f, e := uint128.FromString(frac[:scale-drop])
		if e >= state.Error {
			return Dec128{state: e}
		}
		coef, e := ipart.MulAdd64(Pow10Uint64[scale-drop], f.Lo)
		if e < state.Error {
			if coef.IsZero() {
				return Dec128{coef: uint128.Zero, scale: uint8(scale - drop)}
			}
			return Dec128{coef: coef, scale: uint8(scale - drop), state: st}
		}
		if drop == scale || frac[scale-drop-1] != '0' {
			return Dec128{state: state.Overflow}
		}
		drop++
	}
}

// DecodeFromUint128 decodes a Dec128 from an unsigned coefficient and an exponent: the value is coef * 10^-exp. An
// exponent above MaxScale yields NaN(ScaleOutOfRange).
func DecodeFromUint128(coef uint128.Uint128, exp uint8) Dec128 {
	if exp > MaxScale {
		return Dec128{state: state.ScaleOutOfRange}
	}
	return Dec128{coef: coef, scale: exp}
}

// DecodeFromUint64 decodes a Dec128 from a uint64 coefficient and an exponent: the value is coef * 10^-exp. An
// exponent above MaxScale yields NaN(ScaleOutOfRange).
func DecodeFromUint64(coef uint64, exp uint8) Dec128 {
	if exp > MaxScale {
		return Dec128{state: state.ScaleOutOfRange}
	}
	return Dec128{coef: uint128.FromUint64(coef), scale: exp}
}

// DecodeFromInt64 decodes a Dec128 from an int64 coefficient and an exponent: the value is coef * 10^-exp. An
// exponent above MaxScale yields NaN(ScaleOutOfRange).
func DecodeFromInt64(coef int64, exp uint8) Dec128 {
	switch {
	case exp > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case coef == 0:
		return Dec128{coef: uint128.Zero, scale: exp}
	case coef == -9223372036854775808:
		return Dec128{coef: uint128.FromUint64(9223372036854775808), scale: exp, state: state.Neg}
	case coef < 0:
		return Dec128{coef: uint128.FromUint64(uint64(-coef)), scale: exp, state: state.Neg}
	default:
		return Dec128{coef: uint128.FromUint64(uint64(coef)), scale: exp}
	}
}

// FromInt creates a new Dec128 from an int.
func FromInt(i int) Dec128 {
	return DecodeFromInt64(int64(i), 0)
}

// FromInt64 creates a new Dec128 from an int64.
func FromInt64(i int64) Dec128 {
	return DecodeFromInt64(i, 0)
}

// FromFloat64 returns a decimal from float64, using the shortest decimal string that round-trips the float (1.0/3 is
// 0.3333333333333333). NaN and the infinities yield NaN(state.NaN); a magnitude with more than 39 digits yields
// NaN(Overflow), and one that needs more than MaxScale fractional digits, such as 1e-20, yields NaN(ScaleOutOfRange).
func FromFloat64(f float64) Dec128 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return Dec128{state: state.NaN}
	}

	// 'f' never emits an exponent, and every float64 that fits into a Dec128 formats within MaxStrLen bytes, so the
	// scratch array covers every representable value. Anything longer grows the slice and then fails to fit anyway.
	buf := [MaxStrLen]byte{}
	return FromSafeString(strconv.AppendFloat(buf[:0], f, 'f', -1, 64))
}

// fromSpecial parses the special values PostgreSQL emits for a numeric column and nothing else: "NaN" becomes a NaN
// carrying state.NaN, and "Infinity" and "-Infinity" become NaN(Overflow), the nearest thing a fixed-width type has to
// an infinite magnitude. Any other input is an invalid format. Comparison is exact: no other spelling, case or sign
// is accepted.
func fromSpecial[S string | []byte](s S) Dec128 {
	switch string(s) {
	case NaNStr:
		return Dec128{state: state.NaN}
	case "Infinity", "-Infinity":
		return Dec128{state: state.Overflow}
	}
	return Dec128{state: state.InvalidFormat}
}
