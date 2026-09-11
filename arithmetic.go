package dec128

import (
	"math"
	"math/bits"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Add returns d + other.
// The exact sum has the larger of the two scales. If it fits in 128 bits at that scale that is the result; otherwise
// the scale is reduced to the largest at which the integer part fits and the discarded digits are rounded with the mode
// set by SetArithmeticRounding. NaN(Overflow) is returned only when the integer part does not fit even at scale 0. If
// either operand is NaN, the result is that NaN.
func (d Dec128) Add(other Dec128) Dec128 {
	// Fast path: same scale, same sign, no overflow
	if d.scale == other.scale && d.state == other.state && d.state < state.Error {
		coef, carry := d.coef.AddCarry(other.coef)
		if carry == 0 {
			return Dec128{coef: coef, scale: d.scale, state: d.state}
		}
	}
	return d.addSlow(other)
}

// AddInt returns the sum of the Dec128 and the int.
// If Dec128 is NaN, the result will be NaN. In case of overflow, the result will be NaN.
func (d Dec128) AddInt(other int) Dec128 {
	return d.AddInt64(int64(other))
}

// AddInt64 returns the sum of the Dec128 and the int64.
// If Dec128 is NaN, the result will be NaN. In case of overflow, the result will be NaN.
func (d Dec128) AddInt64(other int64) Dec128 {
	return d.Add(FromInt64(other))
}

// Sub returns d - other, with the same scale and overflow rules as Add.
func (d Dec128) Sub(other Dec128) Dec128 {
	// Fast path: same scale and same sign, so the result is the difference of the magnitudes and cannot overflow
	if d.scale == other.scale && d.state == other.state && d.state < state.Error {
		diff, borrow := d.coef.SubBorrow(other.coef)
		if borrow == 0 {
			if diff.IsZero() {
				return Dec128{scale: d.scale}
			}
			return Dec128{coef: diff, scale: d.scale, state: d.state}
		}
		// |other| > |d|: wrapped difference is the two's complement of the magnitude, and result takes opposite sign
		mag, _ := uint128.Zero.SubBorrow(diff)
		st := state.Neg
		if d.state == state.Neg {
			st = state.Default
		}
		return Dec128{coef: mag, scale: d.scale, state: st}
	}
	return d.subSlow(other)
}

// SubInt returns the difference of the Dec128 and the int.
// If Dec128 is NaN, the result will be NaN. In case of overflow the result will be NaN.
func (d Dec128) SubInt(other int) Dec128 {
	return d.SubInt64(int64(other))
}

// SubInt64 returns the difference of the Dec128 and the int64.
// If Dec128 is NaN, the result will be NaN. In case of overflow the result will be NaN.
func (d Dec128) SubInt64(other int64) Dec128 {
	return d.Sub(FromInt64(other))
}

// Mul returns d * other.
// The exact product has scale d.scale + other.scale. If it fits in 128 bits at that scale, or at MaxScale, that is the
// result. Otherwise the scale is reduced to the largest at which the integer part fits and the discarded digits are
// rounded with the mode set by SetArithmeticRounding; NaN(Overflow) is returned only when the integer part does not fit
// even at scale 0. If either operand is NaN, the result is that NaN.
// Use MulRound to choose the result scale and rounding mode per call.
func (d Dec128) Mul(other Dec128) Dec128 {
	// Fast path: one-limb coefficients whose product needs no reduction.
	if d.coef.Hi|other.coef.Hi == 0 && d.state < state.Error && other.state < state.Error && d.scale+other.scale <= MaxScale {
		hi, lo := bits.Mul64(d.coef.Lo, other.coef.Lo)
		st := signOf(d.state, other.state)
		if hi|lo == 0 {
			st = state.Default
		}
		return Dec128{coef: uint128.Uint128{Lo: lo, Hi: hi}, scale: d.scale + other.scale, state: st}
	}
	return d.mulSlow(other)
}

// MulRound returns d * other at exactly the given scale, rounding with mode.
// If the exact product needs no more than scale places it is returned exactly, padded with zeros up to scale; if that
// padded coefficient does not fit in 128 bits the result is NaN(Overflow). Otherwise the digits below scale are
// discarded with mode; under ROUND_NAN a nonzero discarded digit yields NaN(Inexact). scale above MaxScale yields
// NaN(ScaleOutOfRange) and an undefined mode NaN(InvalidRoundingMode).
func (d Dec128) MulRound(other Dec128, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case other.state >= state.Error:
		return other
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	}

	st := signOf(d.state, other.state)
	lo, hi := d.coef.MulCarry(other.coef)
	needed := d.scale + other.scale

	if scale >= needed {
		if !hi.IsZero() {
			return Dec128{state: state.Overflow}
		}
		if lo.IsZero() {
			return Dec128{scale: scale}
		}
		return Dec128{coef: lo, scale: needed, state: st}.Rescale(scale)
	}

	q, overflow, inexact := reduceWide(lo, hi, needed-scale, st, mode)
	switch {
	case overflow:
		return Dec128{state: state.Overflow}
	case inexact && mode == ROUND_NAN:
		return Dec128{state: state.Inexact}
	case q.IsZero():
		return Dec128{scale: scale}
	}

	return Dec128{coef: q, scale: scale, state: st}
}

// MulInt returns d * other.
// If Dec128 is NaN, the result will be NaN. In case of overflow, the result will be NaN.
func (d Dec128) MulInt(other int) Dec128 {
	return d.MulInt64(int64(other))
}

// MulInt64 returns d * other.
// If Dec128 is NaN, the result will be NaN. In case of overflow, the result will be NaN.
func (d Dec128) MulInt64(other int64) Dec128 {
	return d.Mul(FromInt64(other))
}

// Div returns d / other at the default scale (see SetDefaultScale), rounded with the mode set by SetArithmeticRounding.
// An exact quotient is returned at its ideal scale instead: trailing zeros are removed down to
// d.Scale() - other.Scale() (or 0), so 1/2 is 0.5, 1.00/2 is 0.50 and 1/3 keeps all its places. If the quotient does
// not fit in 128 bits at the default scale, the scale is reduced to the largest at which it does; NaN(Overflow) is
// returned only when the integer part does not fit even at scale 0. Division by zero yields NaN(DivisionByZero), and
// NaN operands propagate. Use DivRound to choose the scale and mode per call.
func (d Dec128) Div(other Dec128) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case other.state >= state.Error:
		return other
	case other.coef.IsZero():
		return Dec128{state: state.DivisionByZero}
	case d.coef.IsZero():
		return Dec128{scale: idealDivScale(d.scale, other.scale)}
	}

	st := signOf(d.state, other.state)
	mode := arithmeticRounding
	scale := defaultScale

	q, overflow, inexact := d.divAt(other, scale, st, mode)
	if overflow {
		// Estimate the largest fitting scale from the bit lengths: the quotient has at most
		// bitLen(d) + bitLen(10^f) - bitLen(other) + 1 bits. Then probe one above, because the estimate can be a digit
		// too cautious.
		bound := 127 + other.coef.BitLen() - d.coef.BitLen()
		for scale > 0 {
			f := int(scale) + int(other.scale) - int(d.scale)
			if f <= 0 || int(bitLen10[f]) <= bound {
				break
			}
			scale--
		}
		q, overflow, inexact = d.divAt(other, scale, st, mode)
		// A backstop loop here is unreachable: when the estimate accepts a scale, the quotient q satisfies
		// q < 2^a * 10^f / 2^(b-1) <= 2^(128-L) * (2^L - 1), so q <= 2^128 - 2 and even a round-up cannot carry out
		// of 128 bits. Overflow below is therefore only the scale-0 case, where the integer part itself does not fit.
		//for overflow && scale > 0 {
		//	scale--
		//	q, overflow, inexact = d.divAt(other, scale, st, mode)
		//}
		if overflow {
			return Dec128{state: state.Overflow}
		}
		if scale+1 < defaultScale {
			if q2, overflow2, inexact2 := d.divAt(other, scale+1, st, mode); !overflow2 {
				q, inexact, scale = q2, inexact2, scale+1
			}
		}
	}

	switch {
	case inexact:
		// The quotient lost a digit. A zero quotient means it lost all of them: neither operand was zero, so the
		// exact result is below one unit in the last place and rounding has erased it.
		if s := lossState(q.IsZero(), lossPolicy); s != state.OK {
			return Dec128{state: s}
		}
		if q.IsZero() {
			return Dec128{scale: scale}
		}
	default:
		// q cannot be zero here: the dividend is non-zero and the quotient is exact.
		// An exact quotient takes its ideal scale: trailing zeros go, but not below
		// d.scale - other.scale, so 1/2 = 0.5 and 1.00/2 = 0.50.
		q, scale = stripZeros(q, scale, idealDivScale(d.scale, other.scale))
	}

	return Dec128{coef: q, scale: scale, state: st}
}

// idealDivScale is the scale an exact quotient of operands with scales s1 and s2 takes:
// s1 - s2, or 0 if that is negative (the GDA ideal exponent, capped at MaxScale).
func idealDivScale(s1, s2 uint8) uint8 {
	if s1 > s2 {
		return min(s1-s2, MaxScale)
	}
	return 0
}

// DivRound returns d / other at exactly the given scale, rounding with mode.
// The quotient is computed in one step with its remainder, so the rounding decision is exact. If the coefficient at
// that scale does not fit in 128 bits the result is NaN(Overflow); under ROUND_NAN a nonzero remainder yields
// NaN(Inexact). Division by zero yields NaN(DivisionByZero), a scale above MaxScale NaN(ScaleOutOfRange), an
// undefined mode NaN(InvalidRoundingMode), and NaN operands propagate.
func (d Dec128) DivRound(other Dec128, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case other.state >= state.Error:
		return other
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case other.coef.IsZero():
		return Dec128{state: state.DivisionByZero}
	case d.coef.IsZero():
		return Dec128{scale: scale}
	}

	st := signOf(d.state, other.state)
	q, overflow, inexact := d.divAt(other, scale, st, mode)
	switch {
	case overflow:
		return Dec128{state: state.Overflow}
	case inexact && mode == ROUND_NAN:
		return Dec128{state: state.Inexact}
	case q.IsZero():
		return Dec128{scale: scale}
	}

	return Dec128{coef: q, scale: scale, state: st}
}

// DivInt returns d / other.
// If Dec128 is NaN, the result will be NaN. In case of overflow or division by zero the result will be NaN.
func (d Dec128) DivInt(other int) Dec128 {
	return d.DivInt64(int64(other))
}

// DivInt64 returns d / other.
// If Dec128 is NaN, the result will be NaN. In case of overflow or division by zero the result will be NaN.
func (d Dec128) DivInt64(other int64) Dec128 {
	return d.Div(FromInt64(other))
}

// Mod returns d % other: the remainder of the truncated division, with the sign of d and the larger scale of the two
// operands (see QuoRem). If any of the Dec128 is NaN, the result will be NaN. Division by zero yields
// NaN(DivisionByZero) and a quotient that does not fit in 128 bits NaN(Overflow).
func (d Dec128) Mod(other Dec128) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case other.state >= state.Error:
		return other
	case other.coef.IsZero():
		return Dec128{state: state.DivisionByZero}
	case d.coef.IsZero():
		return Dec128{scale: max(d.scale, other.scale)}
	}

	_, r, ok := d.tryQuoRem(other)
	if ok {
		return r
	}

	return Dec128{state: state.Overflow}
}

// ModInt returns d % other.
// If Dec128 is NaN, the result will be NaN. In case of overflow or division by zero the result will be NaN.
func (d Dec128) ModInt(other int) Dec128 {
	return d.ModInt64(int64(other))
}

// ModInt64 returns d % other.
// If Dec128 is NaN, the result will be NaN. In case of overflow or division by zero the result will be NaN.
func (d Dec128) ModInt64(other int64) Dec128 {
	return d.Mod(FromInt64(other))
}

// QuoRem returns the quotient and remainder of the division of Dec128 by other Dec128. The quotient is the integer
// part of d / other, truncated toward zero, and the remainder d - q*other carries the sign of d and the larger scale of
// the two operands; both are exact. If any of the Dec128 is NaN, the result will be NaN. Division by zero yields
// NaN(DivisionByZero) and a quotient that does not fit in 128 bits NaN(Overflow).
func (d Dec128) QuoRem(other Dec128) (Dec128, Dec128) {
	switch {
	case d.state >= state.Error:
		return d, d
	case other.state >= state.Error:
		return other, other
	case other.coef.IsZero():
		return Dec128{state: state.DivisionByZero}, Dec128{state: state.DivisionByZero}
	case d.coef.IsZero():
		return Zero, Dec128{scale: max(d.scale, other.scale)}
	}

	// tryQuoRem fails only when the quotient itself has 129 or more bits, which no rescaling of the operands changes
	q, r, ok := d.tryQuoRem(other)
	if ok {
		return q, r
	}

	return Dec128{state: state.Overflow}, Dec128{state: state.Overflow}
}

// QuoRemInt returns the quotient and remainder of the division of Dec128 by int.
// If Dec128 is NaN, the result will be NaN. In case of overflow or division by zero the result will be NaN.
func (d Dec128) QuoRemInt(other int) (Dec128, Dec128) {
	return d.QuoRemInt64(int64(other))
}

// QuoRemInt64 returns the quotient and remainder of the division of Dec128 by int64.
// If Dec128 is NaN, the result will be NaN. In case of overflow or division by zero the result will be NaN.
func (d Dec128) QuoRemInt64(other int64) (Dec128, Dec128) {
	return d.QuoRem(FromInt64(other))
}

// Abs returns |d|.
// If Dec128 is NaN, the result will be NaN.
func (d Dec128) Abs() Dec128 {
	if d.state >= state.Error {
		return d
	}
	return Dec128{coef: d.coef, scale: d.scale}
}

// Neg returns -d.
// If Dec128 is NaN, the result will be NaN.
func (d Dec128) Neg() Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case d.state == state.Neg:
		return Dec128{coef: d.coef, scale: d.scale}
	case d.coef.IsZero():
		return d
	default:
		return Dec128{coef: d.coef, scale: d.scale, state: state.Neg}
	}
}

// Sqrt returns the square root of d at the default scale, rounded with the mode set by SetArithmeticRounding.
// An exact root is returned at its ideal scale instead, half of d's rounded up: sqrt(4) is 2 and sqrt(4.00) is 2.0.
// The root of a negative value is NaN(SqrtNegative); NaN propagates.
// Use SqrtRound to choose the scale and mode per call.
func (d Dec128) Sqrt() Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case d.coef.IsZero():
		return Dec128{scale: min((d.scale+1)/2, defaultScale)}
	case d.state == state.Neg:
		return Dec128{state: state.SqrtNegative}
	}

	q, inexact := d.sqrtAt(defaultScale, arithmeticRounding)
	if inexact {
		// A root can reach zero only when the default scale is too coarse to hold it: sqrt(x) > x for 0 < x < 1, so
		// at the full MaxScale the smallest representable radicand still has a representable root.
		if s := lossState(q.IsZero(), lossPolicy); s != state.OK {
			return Dec128{state: s}
		}
		return Dec128{coef: q, scale: defaultScale}
	}
	// An exact root takes its ideal scale, half of d's rounded up: sqrt(4) is 2, sqrt(4.00) is 2.0,
	// while sqrt(2) keeps all its places.
	q, scale := stripZeros(q, defaultScale, min((d.scale+1)/2, defaultScale))
	return Dec128{coef: q, scale: scale}
}

// SqrtRound returns the square root of d at exactly the given scale, rounding with mode. The root at any scale up to
// MaxScale always fits in 128 bits, so the only NaN results are a negative d (SqrtNegative), a scale above MaxScale
// (ScaleOutOfRange), an undefined mode (InvalidRoundingMode), an inexact root under ROUND_NAN (Inexact), and
// a NaN operand.
func (d Dec128) SqrtRound(scale uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case d.coef.IsZero():
		return Dec128{scale: scale}
	case d.state == state.Neg:
		return Dec128{state: state.SqrtNegative}
	}

	q, inexact := d.sqrtAt(scale, mode)
	if inexact && mode == ROUND_NAN {
		return Dec128{state: state.Inexact}
	}
	return Dec128{coef: q, scale: scale}
}

// PowInt returns Dec128 raised to the power of n; see PowInt64.
func (d Dec128) PowInt(n int) Dec128 {
	return d.PowInt64(int64(n))
}

// PowInt64 returns Dec128 raised to the power of n, by repeated squaring with each product rounded to fit like Mul.
// d^0 is 1 for every d including 0. A negative n yields 1 / d^-n at the default scale, so it is inexact and rounded
// like Div; 0 to a negative power is NaN(DivisionByZero), and a power whose reciprocal does not fit is NaN(Overflow).
// NaN propagates.
func (d Dec128) PowInt64(n int64) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case n < 0:
		var r Dec128
		if n == math.MinInt64 {
			// -n does not fit an int64; d^(2^63) is d^(2^63-1) * d
			r = d.PowInt64(math.MaxInt64).Mul(d)
		} else {
			r = d.PowInt64(-n)
		}
		if r.IsZero() && !d.IsZero() {
			// the positive power rounded to zero, so its reciprocal does not fit
			return Dec128{state: state.Overflow}
		}
		return One.Div(r)
	case n == 0:
		return One
	case n == 1:
		return d
	case (n & 1) == 0:
		return d.Mul(d).PowInt64(n / 2)
	default:
		return d.Mul(d).PowInt64((n - 1) / 2).Mul(d)
	}
}
