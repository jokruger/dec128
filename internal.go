package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
	"math"
	"math/bits"
)

var (
	// precalculated StringFixed values for 0 Dec128 in all possible scales
	zeroStrs = [...]string{
		"0",                     // 10^0
		"0.0",                   // 10^1
		"0.00",                  // 10^2
		"0.000",                 // 10^3
		"0.0000",                // 10^4
		"0.00000",               // 10^5
		"0.000000",              // 10^6
		"0.0000000",             // 10^7
		"0.00000000",            // 10^8
		"0.000000000",           // 10^9
		"0.0000000000",          // 10^10
		"0.00000000000",         // 10^11
		"0.000000000000",        // 10^12
		"0.0000000000000",       // 10^13
		"0.00000000000000",      // 10^14
		"0.000000000000000",     // 10^15
		"0.0000000000000000",    // 10^16
		"0.00000000000000000",   // 10^17
		"0.000000000000000000",  // 10^18
		"0.0000000000000000000", // 10^19
	}

	// precalculated array of zero characters
	zeros = [...]byte{'0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0'}
)

// mulSlow is the general case of Mul: NaN propagation, two-limb coefficients and products that must be reduced to fit.
// Kept out of Mul so that the fast path stays lean.
func (d Dec128) mulSlow(other Dec128) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case other.state >= state.Error:
		return other
	}

	lo, hi := d.coef.MulCarry(other.coef)
	scale := d.scale + other.scale
	st := signOf(d.state, other.state)
	if hi.IsZero() && scale <= MaxScale {
		// exact and in range
		if lo.IsZero() {
			st = state.Default
		}
		return Dec128{coef: lo, scale: scale, state: st}
	}

	return fitWide(lo, hi, scale, st, arithmeticRounding, lossPolicy)
}

// called only when both are not NaN
func (d Dec128) tryQuoRem(other Dec128) (Dec128, Dec128, bool) {
	var factor uint8
	var u uint128.Uint128
	var c uint128.Uint128
	var dv uint128.Uint128
	var s state.State

	if d.scale == other.scale {
		factor = d.scale
		u = d.coef
		dv = other.coef
	} else {
		factor = max(d.scale, other.scale)
		u, c = d.coef.MulCarry(Pow10Uint128[factor-d.scale])
		dv, s = other.coef.Mul(Pow10Uint128[factor-other.scale])
		if s >= state.Error {
			// the aligned divisor exceeds 128 bits, so it exceeds |d| (aligned, factor == d.scale here): the quotient
			// is zero and the remainder is d itself
			return Zero, Dec128{coef: d.coef, scale: factor, state: d.state}, true
		}
	}

	q1, r1, s := uint128.QuoRem256By128(u, c, dv)
	if s >= state.Error {
		return Dec128{state: s}, Dec128{state: s}, false
	}

	q := Dec128{coef: q1}
	if d.state != other.state && !q1.IsZero() {
		q.state = state.Neg
	}
	r := Dec128{coef: r1, scale: factor}
	if !r1.IsZero() {
		r.state = d.state // the remainder takes the sign of the dividend; a zero is never negative
	}
	return q, r, true
}

// appendString appends the string representation of the decimal to sb. Returns the new slice and whether the decimal
// contains a decimal point. Called only when d is not NaN.
func (d Dec128) appendString(sb []byte) ([]byte, bool) {
	buf := [uint128.MaxStrLen]byte{}
	coef := d.coef.StringToBuf(buf[:])

	if d.state == state.Neg {
		sb = append(sb, '-')
	}

	scale := int(d.scale)
	if scale == 0 {
		return append(sb, coef...), false
	}

	sz := len(coef)
	if scale > sz {
		sb = append(sb, '0', '.')
		sb = append(sb, zeros[:scale-sz]...)
		sb = append(sb, coef...)
	} else if scale == sz {
		sb = append(sb, '0', '.')
		sb = append(sb, coef...)
	} else {
		sb = append(sb, coef[:sz-scale]...)
		sb = append(sb, '.')
		sb = append(sb, coef[sz-scale:]...)
	}

	return sb, true
}

func trimTrailingZeros(sb []byte) []byte {
	i := len(sb)

	for i > 0 && sb[i-1] == '0' {
		i--
	}

	if i > 0 && sb[i-1] == '.' {
		i--
	}

	return sb[:i]
}

// addSlow is the general case of Add: NaN propagation, differing scales, opposite signs and results that must be
// reduced to fit. It is kept out of Add so that Add's fast path stays a few instructions (it is still above the
// compiler's inlining budget, so the split buys a smaller, better-predicted body rather than inlining).
func (d Dec128) addSlow(other Dec128) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case other.state >= state.Error:
		return other
	}

	sum, scale, st := d.addExact(other)
	if sum.hi == 0 {
		// exact and in range: the common case for two-limb or unaligned operands
		if sum.lo.IsZero() {
			st = state.Default
		}
		return Dec128{coef: sum.lo, scale: scale, state: st}
	}

	return fitWide(sum.lo, uint128.Uint128{Lo: sum.hi}, scale, st, arithmeticRounding, lossPolicy)
}

// subSlow is the general case of Sub. d - other == d + (-other); Neg of a NaN is the same NaN and Neg of zero is zero,
// so the identity holds for every input.
func (d Dec128) subSlow(other Dec128) Dec128 {
	return d.addSlow(other.Neg())
}

// widened is an operand aligned to a common scale without the risk of overflowing:
// the coefficient is held in 192 bits as (lo, hi) with hi the bits above 2^128.
type widened struct {
	lo uint128.Uint128
	hi uint64
}

// alignOperands brings d and other to their common scale max(d.scale, other.scale), widening the one that needs scaling
// to 192 bits. Alignment therefore never fails; only a result that does not fit back into 128 bits does.
func alignOperands(d, other Dec128) (a, b widened, scale uint8) {
	switch {
	case d.scale == other.scale:
		return widened{lo: d.coef}, widened{lo: other.coef}, d.scale
	case d.scale < other.scale:
		lo, hi := d.coef.Mul64Carry(Pow10Uint64[other.scale-d.scale])
		return widened{lo: lo, hi: hi}, widened{lo: other.coef}, other.scale
	default:
		lo, hi := other.coef.Mul64Carry(Pow10Uint64[d.scale-other.scale])
		return widened{lo: d.coef}, widened{lo: lo, hi: hi}, d.scale
	}
}

// compareWidened compares two 192-bit magnitudes.
func compareWidened(a, b widened) int {
	switch {
	case a.hi < b.hi:
		return -1
	case a.hi > b.hi:
		return 1
	}
	return a.lo.Compare(b.lo)
}

// addExact returns the exact sum of d and other as a 192-bit magnitude at their common scale, with the sign of the
// result. Called only when neither operand is NaN. A widened operand is below 2^128 * 10^19 < 2^192, so neither the sum
// nor the difference of two such magnitudes can exceed 192 bits.
func (d Dec128) addExact(other Dec128) (sum widened, scale uint8, st state.State) {
	a, b, scale := alignOperands(d, other)

	if d.state == other.state {
		lo, carry := a.lo.AddCarry(b.lo)
		return widened{lo: lo, hi: a.hi + b.hi + carry}, scale, d.state
	}

	// Opposite signs: the difference of the magnitudes, signed by the larger one.
	switch compareWidened(a, b) {
	case 0:
		return widened{}, scale, state.Default
	case 1:
		lo, borrow := a.lo.SubBorrow(b.lo)
		return widened{lo: lo, hi: a.hi - b.hi - borrow}, scale, d.state
	default:
		lo, borrow := b.lo.SubBorrow(a.lo)
		return widened{lo: lo, hi: b.hi - a.hi - borrow}, scale, other.state
	}
}

// at brings the exact 192-bit magnitude a, which stands at the scale needed, to exactly the scale target, rounding
// with mode for a result of sign st. It is the tail of AddRound and SubRound: the target is at most MaxScale and the
// needed scale is one of the operands' own, so the reduction is never by more than 10^MaxScale.
func (a widened) at(needed, target uint8, st state.State, mode RoundingMode) Dec128 {
	if target >= needed {
		// padding with zeros, which only a value that already needs all 128 bits can fail
		if a.hi != 0 {
			return Dec128{state: state.Overflow}
		}
		if a.lo.IsZero() {
			return Dec128{scale: target}
		}
		coef, s := a.lo.Mul64(Pow10Uint64[target-needed])
		if s >= state.Error {
			return Dec128{state: state.Overflow}
		}
		return Dec128{coef: coef, scale: target, state: st}
	}

	q, overflow, inexact := reduceWide(a.lo, uint128.Uint128{Lo: a.hi}, needed-target, st, mode)
	switch {
	case overflow:
		return Dec128{state: state.Overflow}
	case inexact && mode == ROUND_NAN:
		return Dec128{state: state.Inexact}
	case q.IsZero():
		return Dec128{scale: target} // a zero is never negative
	}

	return Dec128{coef: q, scale: target, state: st}
}

// divAt returns |d| / |other| scaled to exactly the given scale and rounded with mode, for a result of sign
// st. overflow reports that the quotient does not fit in 128 bits; inexact that a nonzero remainder was discarded.
// Requires d.coef != 0, other.coef != 0 and scale <= MaxScale.
func (d Dec128) divAt(other Dec128, scale uint8, st state.State, mode RoundingMode) (q uint128.Uint128, overflow bool, inexact bool) {
	// The quotient is d.coef * 10^f / other.coef with f = scale + other.scale - d.scale.
	f := int(scale) + int(other.scale) - int(d.scale)

	if f >= 0 {
		// f <= 2*MaxScale, so the numerator fits in 256 bits.
		lo, hi := d.coef.MulCarry(Pow10Uint128[f])
		q, r, s := uint128.QuoRem256By128(lo, hi, other.coef)
		if s >= state.Error {
			return uint128.Zero, true, false
		}
		up, inexact := roundUp(q, r, other.coef, st, mode)
		if up {
			var carry uint64
			if q, carry = q.AddCarry(uint128.One); carry != 0 {
				return uint128.Zero, true, inexact
			}
		}
		return q, false, inexact
	}

	// f < 0: scale the divisor up instead (-f <= d.scale <= MaxScale). The quotient is then at most d.coef / 10 and can
	// neither overflow nor carry when rounded up.
	dlo, dhi := other.coef.Mul64Carry(Pow10Uint64[-f])
	if dhi == 0 {
		q, r, _ := d.coef.QuoRem(dlo)
		up, inexact := roundUp(q, r, dlo, st, mode)
		if up {
			q, _ = q.AddCarry(uint128.One)
		}
		return q, false, inexact
	}

	// The scaled divisor exceeds 128 bits, so the quotient is 0 and the remainder is |d| itself
	// (nonzero by precondition); only the rounding decision remains, against a 192-bit half.
	half := widened{lo: dlo.Rsh(1), hi: dhi >> 1}
	half.lo.Hi |= dhi << 63
	c := compareWidened(widened{lo: d.coef}, half)
	if roundDecision(c > 0, c == 0 && dlo.Lo&1 == 0, false, st, mode) {
		return uint128.One, false, true
	}

	return uint128.Zero, false, true
}

// top64 returns the 64 bits of the 256-bit value (lo, hi) starting at bit e, for an even e in 0..192 chosen so that
// the value's highest set bit is within them.
func top64(lo, hi uint128.Uint128, e int) uint64 {
	switch {
	case e == 0:
		return lo.Lo
	case e < 64:
		return lo.Lo>>e | lo.Hi<<(64-e)
	case e == 64:
		return lo.Hi
	case e < 128:
		return lo.Hi>>(e-64) | hi.Lo<<(128-e)
	case e == 128:
		return hi.Lo
	case e < 192:
		return hi.Lo>>(e-128) | hi.Hi<<(192-e)
	default:
		return hi.Hi
	}
}

// isqrt256 returns floor(sqrt(n)) for the 256-bit value n = hi*2^128 + lo, by Newton-Raphson from an initial guess that
// is at or above the root.
func isqrt256(lo, hi uint128.Uint128) uint128.Uint128 {
	// unreachable for a zero radicand: sqrtAt requires d > 0
	//if lo.IsZero() && hi.IsZero() {
	//	return uint128.Zero
	//}

	// Initial guess from the floating-point root of the top 64 bits: n = top * 2^e + rest with e even, so
	// sqrt(n) < sqrt(top+1) * 2^(e/2). float64 loses at most 2^10 of top (below 2^64) and math.Sqrt is correctly rounded,
	// which together move the root by less than one, so floor(s)+2 is above sqrt(top+1). The guess is then correct to
	// about 52 bits and Newton-Raphson needs three steps instead of the eight or so from a power of two.
	bl := bitLen256(lo, hi)
	e := 0
	if bl > 64 {
		e = (bl - 64 + 1) &^ 1 // round up to even
	}
	x0 := uint64(math.Sqrt(float64(top64(lo, hi, e)))) + 2
	x := uint128.Uint128{Lo: ^uint64(0), Hi: ^uint64(0)}
	if bits.Len64(x0)+e/2 <= 128 {
		// otherwise the shifted guess would not fit; the largest coefficient is still at or above the root, n < 2^256
		x = uint128.Uint128{Lo: x0}.Lsh(uint(e / 2))
	}

	for {
		// n / x fits in 128 bits whenever x >= sqrt(n). The one exception is a radicand above (2^128-1)^2, whose root
		// is the largest coefficient itself: x already holds it and the division reports overflow. sqrtAt never gets
		// there (its radicands are below 2^255), so this only makes the function total.
		y, _, s := uint128.QuoRem256By128(lo, hi, x)
		if s >= state.Error {
			return x
		}

		// x1 = (x + y) / 2, computed without overflowing 128 bits
		x1, carry := x.AddCarry(y)
		x1 = x1.Rsh(1)
		x1.Hi |= carry << 63

		// x starts at or above the root and decreases monotonically -> first non-decreasing step is the fixed point
		if x1.Compare(x) >= 0 {
			return x
		}
		x = x1
	}
}

// sqrtAt returns sqrt(d) scaled to exactly the given scale and rounded with mode, and whether the result is inexact.
// Requires d > 0, not NaN, scale <= MaxScale. The root is taken at a working scale w with 2w >= d.scale, so that the
// radicand coef * 10^(2w - d.scale) is an integer (below 2^255, so it fits in 256 bits and its root fits in 128).
// floor(sqrt(n)) never overflows. When w is above the requested scale the result is reduced with the exactness of the
// root as the sticky bit, which keeps the rounding decision correct: the true value is q + e with 0 <= e < 1 and
// e > 0 iff the root was inexact.
func (d Dec128) sqrtAt(scale uint8, mode RoundingMode) (q uint128.Uint128, inexact bool) {
	work := scale
	if int(work)*2 < int(d.scale) {
		work = (d.scale + 1) / 2
	}
	lo, hi := d.coef.MulCarry(Pow10Uint128[int(work)*2-int(d.scale)])

	r := isqrt256(lo, hi)
	sqLo, sqHi := r.MulCarry(r)
	sticky := sqLo != lo || sqHi != hi

	if work == scale {
		if !sticky {
			return r, false
		}
		// Above half iff n > r*r + r (an exact tie would need n = r*r + r + 1/4).
		tLo, c := sqLo.AddCarry(r)
		tHi, _ := sqHi.AddCarry(uint128.Uint128{Lo: c})
		above := compare256(lo, hi, tLo, tHi) > 0
		if roundDecision(above, false, r.Lo&1 == 1, state.Default, mode) {
			r, _ = r.AddCarry(uint128.One) // r < 2^128, since n < 2^255
		}
		return r, true
	}

	// Reduce from the working scale to the requested one; k <= (MaxScale+1)/2.
	k := work - scale
	q, rem, _ := r.QuoRemPow10(k)
	half := Pow10Uint64[k] / 2
	inexact = rem != 0 || sticky
	if !inexact {
		return q, false
	}
	above := rem > half || (rem == half && sticky)
	tie := rem == half && !sticky
	if roundDecision(above, tie, q.Lo&1 == 1, state.Default, mode) {
		q, _ = q.AddCarry(uint128.One)
	}

	return q, true
}
