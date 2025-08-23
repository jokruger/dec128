package dec128

import (
	"math"
	"strconv"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// FromString creates a new Dec128 from a string.
// The string must be in the format of [+-][0-9]+(.[0-9]+)?
// In case of empty string, it returns Zero.
// In case of errors, it returns NaN with the corresponding error.
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
				return Dec128{state: state.InvalidFormat}
			}
			u = u*10 + uint64(c-'0')
		}
		if u == 0 && scale == 0 {
			return Zero
		}
		return Dec128{coef: uint128.FromUint64(u), exp: uint8(scale), state: st}
	}

	j := 0
	for ; j < sz; j++ {
		if s[j] == '.' {
			break
		}
	}

	if j == sz {
		coef, e := uint128.FromString(s[i:])
		if e >= state.Error {
			return Dec128{state: e}
		}
		return Dec128{coef: coef, exp: 0, state: st}
	}

	if j == sz-1 {
		return Dec128{state: state.InvalidFormat}
	}

	scale = sz - j - 1
	if scale > uint128.MaxSafeStrLen64 {
		return Dec128{state: state.ScaleOutOfRange}
	}

	ipart, ei := uint128.FromString(s[i:j])
	if ei >= state.Error {
		return Dec128{state: ei}
	}

	fpart, ef := uint128.FromString(s[j+1:])
	if ef >= state.Error {
		return Dec128{state: ef}
	}

	// max scale is 19, so the fpart.Hi is always 0 and scale is always <= len(pow10)
	coef, e := ipart.MulAdd64(Pow10Uint64[scale], fpart.Lo)
	if e >= state.Error {
		return Dec128{state: e}
	}

	if coef.IsZero() && scale == 0 {
		return Zero
	}

	return Dec128{coef: coef, exp: uint8(scale), state: st}
}

// FromSafeString creates a new Dec128 from safe string (no format checks are applied).
// In case of errors, it returns NaN with the corresponding error.
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
		if u == 0 && scale == 0 {
			return Zero
		}
		return Dec128{coef: uint128.FromUint64(u), exp: uint8(scale), state: st}
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
		return Dec128{coef: coef, exp: 0, state: st}
	}

	scale = sz - j - 1
	if scale > uint128.MaxSafeStrLen64 {
		return Dec128{state: state.ScaleOutOfRange}
	}

	ipart, ei := uint128.FromSafeString(s[i:j])
	if ei >= state.Error {
		return Dec128{state: ei}
	}

	fpart, ef := uint128.FromSafeString(s[j+1:])
	if ef >= state.Error {
		return Dec128{state: ef}
	}

	// max scale is 19, so the fpart.Hi is always 0 and scale is always <= len(pow10)
	coef, e := ipart.MulAdd64(Pow10Uint64[scale], fpart.Lo)
	if e >= state.Error {
		return Dec128{state: e}
	}

	if coef.IsZero() && scale == 0 {
		return Zero
	}

	return Dec128{coef: coef, exp: uint8(scale), state: st}
}

// DecodeFromUint128 decodes a Dec128 from a Uint128 and an exponent.
func DecodeFromUint128(coef uint128.Uint128, exp uint8) Dec128 {
	return Dec128{coef: coef, exp: exp}
}

// DecodeFromUint64 decodes a Dec128 from a uint64 and an exponent.
func DecodeFromUint64(coef uint64, exp uint8) Dec128 {
	return Dec128{coef: uint128.FromUint64(coef), exp: exp}
}

// DecodeFromInt64 decodes a Dec128 from a int64 and an exponent.
func DecodeFromInt64(coef int64, exp uint8) Dec128 {
	switch {
	case coef == -9223372036854775808:
		return Dec128{coef: uint128.FromUint64(9223372036854775808), exp: exp, state: state.Neg}
	case coef < 0:
		return Dec128{coef: uint128.FromUint64(uint64(-coef)), exp: exp, state: state.Neg}
	default:
		return Dec128{coef: uint128.FromUint64(uint64(coef)), exp: exp}
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

// FromFloat64 returns a decimal from float64.
func FromFloat64(f float64) Dec128 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return Dec128{state: state.NaN}
	}
	return FromString(strconv.FormatFloat(f, 'f', -1, 64))
}
