package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Integer powers with guard digits.
//
// PowInt64 raises a value by repeated squaring with every intermediate product put back into a Dec128, so a long
// compounding chain rounds a dozen times at the published scale, every time in the same direction under the default
// ROUND_TOWARD_ZERO. PowIntRound keeps the running product at a working scale of up to powGuard places instead - as
// many digits as a coefficient will hold - and rounds once, at the end, to the scale the caller asked for.
// (1+r)^360 is the most-used primitive in consumer lending and it is worth the extra digits.

// powGuard is the widest working scale the running product is kept at. The product is held in 192 bits rather than
// the 128 of a coefficient, which is 57 decimal digits: a value near 1 carries 57 places and one near 10^k carries
// 57-k, so the relative error of a reduction is around 10^-57 whatever the magnitude, and 19 digits are left over
// when the result is written back at the full width of a coefficient. The reductions round to nearest, so the error
// does not accumulate in one direction the way truncation would.
const powGuard = 3 * int(MaxScale)

// powReduce brings a 384-bit exact product, standing at the scale needed, back to 192 bits at the largest working
// scale not above powGuard at which it fits, rounding to nearest with ties away from zero. ok is false when the
// integer part does not fit even at scale 0.
func powReduce(p wide, needed int) (wide, uint8, bool, bool) {
	k := 0
	if needed > powGuard {
		k = needed - powGuard
	}
	// The quotient by 10^k has at least bitLen - bits(10^k) bits and bits(10^k) is about 3.3219*k, so no k below
	// (bitLen-192)/3.3219 can fit. 1000/3322 is just under 1/3.3219, which keeps the estimate on the safe side; the
	// loop below makes up the digit or two it can be short by.
	if kw := (p.bitLen() - 192) * 1000 / 3322; kw > k {
		k = kw
	}

	for k <= needed {
		q, above, tie, inexact := p.quoCmpHalfPow10(uint8(k))
		if inexact && roundDecision(above, tie, q[0]&1 == 1, state.Default, ROUND_HALF_AWAY_FROM_ZERO) {
			q, _ = q.add(wide{1})
		}
		if q.fits192() {
			return q, uint8(needed - k), inexact, true
		}
		k++
	}

	return wide{}, 0, false, false
}

// powMul multiplies two magnitudes held at working scales and reduces the product back to the working width.
func powMul(a wide, s1 uint8, b wide, s2 uint8) (wide, uint8, bool, bool) {
	needed := int(s1) + int(s2)
	p := a.mul192(b)
	if needed <= powGuard && p.fits192() {
		// exact and in range, which is the whole of a short chain over short operands
		return p, uint8(needed), false, true
	}
	return powReduce(p, needed)
}

// powMagnitude raises the magnitude coef * 10^-scale to the power m > 0 by repeated squaring, keeping the running
// product at its own working scale throughout.
func powMagnitude(coef wide, scale uint8, m uint64) (wide, uint8, bool, bool) {
	base, baseScale := coef, scale
	var acc wide
	var accScale uint8
	inexact, started := false, false

	for m > 0 {
		var dropped, ok bool
		if m&1 == 1 {
			if started {
				if acc, accScale, dropped, ok = powMul(acc, accScale, base, baseScale); !ok {
					return wide{}, 0, false, false
				}
				inexact = inexact || dropped
			} else {
				acc, accScale, started = base, baseScale, true
			}
		}
		if m >>= 1; m > 0 {
			// Squaring the base cannot overflow where the power itself does not: the largest square taken is
			// base^(2^floor(log2 m)), which is at most the result when the base is at or above one, and below one a
			// square only shrinks.
			if base, baseScale, dropped, ok = powMul(base, baseScale, base, baseScale); !ok {
				return wide{}, 0, false, false
			}
			inexact = inexact || dropped
		}
	}

	return acc, accScale, inexact, true
}

// reciprocalGuard returns 1 / (coef * 10^-scale) at the largest working scale not above powGuard at which it fits the
// working width, rounded to nearest with ties away from zero, and whether that was inexact.
//
// A negative power inverts first and raises afterwards, rather than raising and inverting, because the power of a
// small base underflows the working scale long before its reciprocal stops being representable: 0.5^100 is 8x10^-31,
// which keeps only a handful of digits at any fixed scale, while 0.5^-100 is 2^100 and is exact.
func reciprocalGuard(coef uint128.Uint128, scale uint8) (wide, uint8, bool) {
	// 1/d at working scale w is 10^(w+scale) / coef, whose bit length is at most bits(10^(w+scale)) - bits(coef) + 1.
	// Choosing the exponent at or below (190 + bits(coef)) / log2(10) - and 1000/3322 is just under 1/log2(10) -
	// bounds that at 191 bits, so the quotient fits the working width with a bit to spare for the rounding step and
	// no search is needed. Capping w at powGuard only lowers the exponent, which keeps the bound. The working scale
	// never has to go below zero: the estimate is at least 38 even for a one-bit coefficient at MaxScale places.
	w := min((190+coef.BitLen())*1000/3322-int(scale), powGuard)
	q, rem := divPow10ByCoef(w+int(scale), coef)

	if rem.IsZero() {
		return q, uint8(w), false
	}
	// Nearest, ties away from zero. As in roundUp, the remainder is never doubled: it is compared with
	// floor(coef/2), and an odd coef cannot produce an exact tie.
	if c := rem.Compare(coef.Rsh(1)); c > 0 || (c == 0 && coef.Lo&1 == 0) {
		q, _ = q.add(wide{1})
	}

	return q, uint8(w), true
}

// divPow10ByCoef returns 10^e / c and its remainder, by long division in base 10^MaxScale: the numerator's digits in
// that base are 10^(e mod MaxScale) followed by zeros, so each step is one 192-by-128 division whose quotient digit
// fits a limb. The quotient is exact and may be wider than a coefficient, which is why it comes back in the register.
func divPow10ByCoef(e int, c uint128.Uint128) (wide, uint128.Uint128) {
	q, rem, _ := Pow10Uint128[e%int(MaxScale)].QuoRem(c)
	w := wideFrom128(q)

	for range e / int(MaxScale) {
		lo, hi := rem.MulCarry(Pow10Uint128[MaxScale])
		// rem < c, so the quotient digit is below 10^MaxScale and the division cannot report overflow
		d, r, _ := uint128.QuoRem256By128(lo, hi, c)
		w, _ = w.mul64(Pow10Uint64[MaxScale])
		w, _ = w.add(wideFrom128(d))
		rem = r
	}

	return w, rem
}

// PowIntRound returns d raised to the power n at exactly the given scale, rounding with mode.
//
// The running product is kept with guard digits - 192 bits and up to 57 decimal places, half as many again as a
// coefficient holds - and each reduction rounds to nearest, so a long chain neither loses digits early nor drifts in
// one direction. Only the result is rounded, to the caller's scale and mode. PowInt64 by contrast puts every
// intermediate back into a Dec128 at the published scale with the process-global rounding mode, which for the default
// ROUND_TOWARD_ZERO is a systematic downward bias over the dozen roundings of a (1+r)^360.
//
// The result is faithfully rounded: it is one of the two representable values adjacent to the exact power. It was
// also the correctly rounded one in every case measured - TestPowIntRoundAgainstBigRat compares it against an exact
// math/big.Rat oracle over random bases and exponents in all eight rounding modes, and the largest error observed is
// zero units in the last place. That is a measurement and not a proof: the guard digits leave about 19 decimal places
// of slack at the full width of a coefficient, so a near-tie closer to the boundary than that could round the other
// way, and a caller who depends on the correctly rounded value of one should verify it independently.
//
// For scale, the ten-year daily-rest factor 1.000164383561643836^3650 comes back exact at all 19 places, where
// PowInt64 under the default truncating mode is 2262 units in the last place low.
//
// d^0 is 1 for every d, including 0 and including a d whose powers would overflow. A negative n is the power of the
// reciprocal: 1/d is taken first, at the guard scale, because the power of a small base underflows long before its
// reciprocal stops being representable. 0 to a negative power is NaN(DivisionByZero). A power whose integer part does
// not fit is NaN(Overflow), a scale above MaxScale NaN(ScaleOutOfRange), an undefined mode NaN(InvalidRoundingMode),
// and under ROUND_NAN any discarded nonzero digit, intermediate or final, is NaN(Inexact). NaN propagates.
//
// It reads no process-global configuration.
func (d Dec128) PowIntRound(n int64, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case n == 0:
		// one at exactly the requested scale; the padding always fits
		return Dec128{coef: Pow10Uint128[scale], scale: scale}
	case d.coef.IsZero():
		if n < 0 {
			return Dec128{state: state.DivisionByZero}
		}
		return Dec128{scale: scale}
	}

	// |n| as a magnitude: negating in uint64 is correct for math.MinInt64 as well
	m := uint64(n)
	if n < 0 {
		m = -m
	}

	st := state.Default
	if d.state == state.Neg && m&1 == 1 {
		st = state.Neg
	}

	coef, workScale := wideFrom128(d.coef), d.scale
	inexact := false
	if n < 0 {
		coef, workScale, inexact = reciprocalGuard(d.coef, d.scale)
	}

	coef, workScale, dropped, ok := powMagnitude(coef, workScale, m)
	if !ok {
		return Dec128{state: state.Overflow}
	}
	inexact = inexact || dropped
	if inexact && mode == ROUND_NAN {
		return Dec128{state: state.Inexact}
	}

	if coef == (wide{}) {
		// The power is a nonzero value below the quantum of the working scale, so it is smaller than anything the
		// requested scale can hold: only a directed mode reaches the first unit.
		if inexact && roundDecision(false, false, false, st, mode) {
			return Dec128{coef: uint128.One, scale: scale, state: st}
		}
		return Dec128{scale: scale} // a zero is never negative
	}

	return coef.at(workScale, scale, st, mode)
}
