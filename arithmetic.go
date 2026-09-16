package dec128

import (
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

// AddRound returns d + other at exactly the given scale, rounding with mode.
// The exact sum is formed at the larger of the two scales and brought to scale in one step, so the rounding decision
// is exact and is taken once. A coefficient that does not fit in 128 bits at that scale yields NaN(Overflow); under
// ROUND_NAN a discarded nonzero digit yields NaN(Inexact). A scale above MaxScale yields NaN(ScaleOutOfRange), an
// undefined mode NaN(InvalidRoundingMode), and NaN operands propagate.
//
// Unlike Add it reads no process-global configuration; see "The global-free subset" in the package documentation.
func (d Dec128) AddRound(other Dec128, scale uint8, mode RoundingMode) Dec128 {
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

	sum, needed, st := d.addExact(other)
	return sum.at(needed, scale, st, mode)
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

// SubRound returns d - other at exactly the given scale, rounding with mode; see AddRound for the rules.
// Unlike Sub it reads no process-global configuration.
func (d Dec128) SubRound(other Dec128, scale uint8, mode RoundingMode) Dec128 {
	// d - other == d + (-other): Neg of a NaN is the same NaN and Neg of zero is zero, so the identity holds for
	// every input, and AddRound validates the scale and the mode.
	return d.AddRound(other.Neg(), scale, mode)
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

// MulAddRound returns d*b + c at exactly the given scale, rounding with mode: the fused multiply-add.
//
// The product is formed exactly, the addend is aligned into the same intermediate, and the sum is brought to scale in
// one step, so the result is rounded once. Writing it as d.Mul(b).Add(c).Round(scale, mode) rounds up to three times
// and reads the process-global configuration on the way; this reads none of it. A chain of multiply-accumulate is
// what almost every financial formula is - P*r*t, balance*(1+i) - pmt, a Newton step, Horner evaluation of a cashflow
// polynomial - so the difference compounds.
//
// The failure modes are those of MulRound: NaN(Overflow) when the coefficient does not fit at scale, NaN(Inexact)
// under ROUND_NAN when a nonzero digit would be discarded, NaN(ScaleOutOfRange) above MaxScale,
// NaN(InvalidRoundingMode) for an undefined mode, and a NaN operand propagates - d first, then b, then c.
func (d Dec128) MulAddRound(b, c Dec128, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case b.state >= state.Error:
		return b
	case c.state >= state.Error:
		return c
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	}

	lo, hi := d.coef.MulCarry(b.coef)
	needed := d.scale + b.scale
	st := signOf(d.state, b.state)
	if lo.IsZero() && hi.IsZero() {
		st = state.Default
	}

	// Fast path: the product is itself a representable value, so the fused operation is the exact product added to c
	// in the 192-bit register that Add already uses, and the single rounding is the one AddRound takes.
	if hi.IsZero() && needed <= MaxScale {
		sum, sumScale, sumSt := Dec128{coef: lo, scale: needed, state: st}.addExact(c)
		return sum.at(sumScale, scale, sumSt, mode)
	}

	return mulAddSlow(lo, hi, needed, st, c, scale, mode)
}

// MulDivRound returns d * b / c at exactly the given scale, rounding with mode: the fused multiply-divide.
//
// The product is formed exactly and the numerator and the divisor are kept apart until one rounding decision is made
// at the end. MulRound followed by DivRound rounds twice, and the first rounding perturbs the second whenever the
// product is not representable at the intermediate scale. The case that has no other answer at all is scaling by an
// exact rational whose decimal expansion does not terminate - 1/3, 2/7, 31/365 - because such a factor cannot be
// written as a decimal, so "convert the factor, then multiply" is not available. Proration of a total across parts, a
// percentage of a total, unit conversion by a rational factor and a weighted average are all that shape. It is the
// multiplicative counterpart of MulAddRound.
//
// The exact numerator reaches 2^383, the 256-bit product carrying a further 10^38 when the scales are at their
// extremes, so the intermediate is held in the wide register and the division is exact against the full divisor.
//
// A zero c is NaN(DivisionByZero), a quotient that does not fit NaN(Overflow), a scale above MaxScale
// NaN(ScaleOutOfRange), an undefined mode NaN(InvalidRoundingMode), and under ROUND_NAN a nonzero remainder is
// NaN(Inexact). A NaN operand propagates: d first, then b, then c. It reads no process-global configuration.
func (d Dec128) MulDivRound(b, c Dec128, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case b.state >= state.Error:
		return b
	case c.state >= state.Error:
		return c
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case c.coef.IsZero():
		return Dec128{state: state.DivisionByZero}
	case d.coef.IsZero() || b.coef.IsZero():
		return Dec128{scale: scale} // a zero is never negative
	}

	st := signOf(signOf(d.state, b.state), c.state)
	lo, hi := d.coef.MulCarry(b.coef)

	// The exact quotient is d.coef * b.coef * 10^f / c.coef.
	f := int(scale) + int(c.scale) - int(d.scale) - int(b.scale)

	if f >= 0 {
		// The alignment cannot carry out of the register: the product is below 2^256 and f is at most twice MaxScale,
		// so 10^f is below 2^127 and the numerator stays under 2^383.
		n, _ := wideFrom256(lo, hi).mulPow10(uint8(f))
		q, r := n.quoRem128(c.coef)
		return quotientAt(q, r, c.coef, scale, st, mode)
	}

	// f < 0: the true divisor is c.coef * 10^-f, which reaches 2^254 and is not a coefficient. Divide by the two
	// factors in turn instead, and let the reduction by the power of ten take the decision: the remainder of the
	// division by c.coef is worth less than one unit in that quotient's last place, which is exactly the sticky bit
	// quoCmpHalfPow10Sticky folds into the comparison.
	q, rem := wideFrom256(lo, hi).quoRem128(c.coef)
	qw, above, tie, inexact := q.quoCmpHalfPow10Sticky(uint8(-f), !rem.IsZero())
	// Overflow before Inexact, as reduceAt and every other reduction in the package order them.
	coef, ok := qw.uint128()
	if !ok {
		return Dec128{state: state.Overflow}
	}
	if inexact && mode == ROUND_NAN {
		return Dec128{state: state.Inexact}
	}
	if inexact && roundDecision(above, tie, coef.Lo&1 == 1, st, mode) {
		var carry uint64
		if coef, carry = coef.AddCarry(uint128.One); carry != 0 {
			return Dec128{state: state.Overflow}
		}
	}
	if coef.IsZero() {
		return Dec128{scale: scale} // a zero is never negative
	}

	return Dec128{coef: coef, scale: scale, state: st}
}

// MulDivRoundInt64 returns d * num / den at exactly the given scale, rounding with mode.
//
// It is MulDivRound for the common case where the rational factor is a pair of integers - a day count over a year
// basis, a share over a total, a period over a term - and it saves the caller two conversions. It is not a wrapper:
// with integer operands the numerator stays below 2^255 and the divisor below 2^127 whatever the scales are, so the
// numerator is a 192-bit magnitude rather than a 256-bit product and the divisor is usually a single limb, which
// makes the division a hardware divide per occupied limb instead of a 192-by-128 reciprocal step. That is worth
// about a tenth of the running time at MaxScale and rather less at a money scale, where both forms are dominated by
// the same work; the reason to reach for it is the operands, not the speed.
//
// It fails as MulDivRound does, with a zero den giving NaN(DivisionByZero).
func (d Dec128) MulDivRoundInt64(num, den int64, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case den == 0:
		return Dec128{state: state.DivisionByZero}
	case d.coef.IsZero() || num == 0:
		return Dec128{scale: scale} // a zero is never negative
	}

	// magnitudes: negating in uint64 is correct for math.MinInt64 as well
	n, m := uint64(num), uint64(den)
	neg := d.state == state.Neg
	if num < 0 {
		n, neg = -n, !neg
	}
	if den < 0 {
		m, neg = -m, !neg
	}
	st := state.Default
	if neg {
		st = state.Neg
	}

	// d.coef * |num| is below 2^191 and cannot carry out of the register.
	w, _ := wideFrom128(d.coef).mul64(n)
	div := uint128.FromUint64(m)

	if f := int(scale) - int(d.scale); f >= 0 {
		// below 2^191 * 10^19 < 2^255
		w, _ = w.mulPow10(uint8(f))
	} else {
		// |den| * 10^-f is below 2^63 * 2^64 = 2^127, so the scaled divisor is still a coefficient
		div, _ = div.Mul64(Pow10Uint64[-f])
	}

	if div.Hi == 0 {
		// a one-limb divisor is one 128-by-64 division per limb instead of one 192-by-128
		q, r := w.quoRem64(div.Lo)
		return quotientAt(q, uint128.FromUint64(r), div, scale, st, mode)
	}

	q, r := w.quoRem128(div)
	return quotientAt(q, r, div, scale, st, mode)
}

// AddQuoRound returns d + e/f at exactly the given scale, rounding with mode. It is the fused add-and-divide, and it
// makes one rounding decision where the written-out form makes two.
//
// It is the shape a per-unit figure takes: a base plus a share, a fixed leg plus an accrual over a day count, a
// running total plus the next instalment of a division that does not terminate. Written as d.Add(e.DivRound(f, ...))
// the quotient is rounded to a scale before it is added, and whatever that rounding discarded is gone; here the whole
// of d + e/f is formed exactly first - the identity is (d*f + e)/f, whose numerator is an exact 384-bit magnitude -
// and the single decision is made on the true value.
//
// MulAddRound and MulDivRound are the other two fused forms; this is the one whose inexactness is in the divisor
// rather than the product. For d - e/f, negate e: there is no SubQuoRound, because Neg is exact and a second method
// would only be a second place for the sign to go wrong.
//
// A zero f is NaN(DivisionByZero), a scale above MaxScale NaN(ScaleOutOfRange), an undefined mode
// NaN(InvalidRoundingMode), a result too large for the coefficient NaN(Overflow), and under ROUND_NAN a result that is
// not exact is NaN(Inexact). NaN propagates: d first, then e, then f.
//
// It reads no process-global configuration and allocates nothing.
func (d Dec128) AddQuoRound(e, f Dec128, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case e.state >= state.Error:
		return e
	case f.state >= state.Error:
		return f
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case f.coef.IsZero():
		return Dec128{state: state.DivisionByZero}
	}

	// The exact numerator is d*f + e, formed at the scale of whichever term has more places. The product occupies 256
	// bits and the alignment at most 10^19 more, so the sum is under 2^321 and the register never fills.
	ps := int(d.scale) + int(f.scale)
	ns := max(ps, int(e.scale))

	lo, hi := d.coef.MulCarry(f.coef)
	pw, over := wideFrom256(lo, hi).mulPow10(uint8(ns - ps))
	if over {
		return Dec128{state: state.Overflow}
	}
	ew, over := wideFrom128(e.coef).mulPow10(uint8(ns - int(e.scale)))
	if over {
		return Dec128{state: state.Overflow}
	}

	// The two terms carry their own signs, so the sum is a magnitude and a sign rather than a signed register. A zero
	// coefficient is never negative, which is what makes the state comparison the whole of the sign test.
	pst := signOf(d.state, f.state)
	n, nst := pw, pst
	if pst == e.state {
		var carry bool
		if n, carry = pw.add(ew); carry {
			return Dec128{state: state.Overflow}
		}
	} else if pw.compare(ew) >= 0 {
		n = pw.sub(ew)
	} else {
		n, nst = ew.sub(pw), e.state
	}

	// The quotient's coefficient is n * 10^(f.scale + scale - ns) / f.coef, and that exponent is within one MaxScale
	// of zero in either direction: above, ns is the product's own scale; below, it is e's.
	st := signOf(nst, f.state)
	g := int(f.scale) + int(scale) - ns

	if g >= 0 {
		n2, over := n.mulPow10(uint8(g))
		if over {
			return Dec128{state: state.Overflow}
		}
		q, r := n2.quoRem128(f.coef)
		return quotientAt(q, r, f.coef, scale, st, mode)
	}

	// g < 0: divide by the two factors in turn and let the reduction by the power of ten take the decision, with the
	// remainder of the first division folded in as the sticky bit - the same split MulDivRound makes, for the same
	// reason.
	q, rem := n.quoRem128(f.coef)
	qw, above, tie, inexact := q.quoCmpHalfPow10Sticky(uint8(-g), !rem.IsZero())
	// Overflow before Inexact, as every other reduction in the package orders them.
	coef, ok := qw.uint128()
	if !ok {
		return Dec128{state: state.Overflow}
	}
	if inexact && mode == ROUND_NAN {
		return Dec128{state: state.Inexact}
	}
	if inexact && roundDecision(above, tie, coef.Lo&1 == 1, st, mode) {
		var carry uint64
		if coef, carry = coef.AddCarry(uint128.One); carry != 0 {
			return Dec128{state: state.Overflow}
		}
	}
	if coef.IsZero() {
		return Dec128{scale: scale} // a zero is never negative
	}

	return Dec128{coef: coef, scale: scale, state: st}
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
// Use DivRoundInexact to learn whether the quotient was exact without losing it.
func (d Dec128) DivRound(other Dec128, scale uint8, mode RoundingMode) Dec128 {
	q, _ := d.DivRoundInexact(other, scale, mode)
	return q
}

// DivRoundInexact is DivRound with the fact it already knows: whether a nonzero remainder had to be discarded.
//
// ROUND_NAN and LossNaNOnInexact can tell a caller that a quotient was not exact, but only by throwing the quotient
// away. A solver needs both - the rounded value to carry on from, and the knowledge that it is not the exact one - and
// the alternative is to divide twice under two modes and compare, which doubles the cost of every step. The flag is
// false whenever the result is NaN, including for a NaN operand and for an argument the method rejects.
func (d Dec128) DivRoundInexact(other Dec128, scale uint8, mode RoundingMode) (Dec128, bool) {
	switch {
	case d.state >= state.Error:
		return d, false
	case other.state >= state.Error:
		return other, false
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}, false
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}, false
	case other.coef.IsZero():
		return Dec128{state: state.DivisionByZero}, false
	case d.coef.IsZero():
		return Dec128{scale: scale}, false
	}

	st := signOf(d.state, other.state)
	q, overflow, inexact := d.divAt(other, scale, st, mode)
	switch {
	case overflow:
		return Dec128{state: state.Overflow}, false
	case inexact && mode == ROUND_NAN:
		return Dec128{state: state.Inexact}, false
	case q.IsZero():
		return Dec128{scale: scale}, inexact
	}

	return Dec128{coef: q, scale: scale, state: st}, inexact
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

// PowInt64 returns Dec128 raised to the power of n, at the scale the scale rule gives the result and rounded with the
// mode set by SetArithmeticRounding.
//
// It shares the guarded core of PowIntRound, so the running product carries up to 57 decimal places and the result is
// rounded once rather than at every squaring; only the choice of the result scale differs, this one taking the
// largest scale at which the result fits rather than a scale given per call. d^0 is 1 for every d including 0. A
// negative n is the power of the reciprocal; 0 to a negative power is NaN(DivisionByZero), and a result whose integer
// part does not fit is NaN(Overflow). NaN propagates.
//
// Prefer PowIntRound in new code: it takes the scale and the rounding mode per call and reads no process-global
// configuration.
func (d Dec128) PowInt64(n int64) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case n == 0:
		return One
	case d.coef.IsZero() && n < 0:
		return Dec128{state: state.DivisionByZero}
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

	// The sticky flag carries the digits the core dropped while squaring, so the loss policy rules on the whole
	// computation and not only on the final reduction.
	return coef.fit(workScale, st, arithmeticRounding, lossPolicy, inexact || dropped)
}
