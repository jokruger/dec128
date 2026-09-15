package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Moving a value onto a grid: a number of decimal places that may be negative, a multiple of some other value, or a
// power of ten. Rescale and the Round* methods only reach the grids that are a scale of the type itself, which leaves
// out the two rules core banking needs in every jurisdiction - rounding to the nearest hundred or thousand, and cash
// rounding to the nearest 0.05 - and makes a percent-to-fraction conversion go through a division.

// RoundToPlaces rounds d to the given number of decimal places with mode, where places may be negative.
//
// Positive places behave like Round: a value that already has no more than that many places is returned unchanged.
// Negative places clear integer digits and give a result at scale 0, so 1234.RoundToPlaces(-2, ROUND_BANK) is 1200.
// That is the rule for currencies with no minor unit and for the disclosure rounding several jurisdictions require.
//
// A result that does not fit is NaN(Overflow), an undefined mode NaN(InvalidRoundingMode), and under ROUND_NAN a
// discarded nonzero digit is NaN(Inexact). NaN propagates. It reads no process-global configuration.
func (d Dec128) RoundToPlaces(places int8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case places >= 0:
		if uint8(places) >= d.scale {
			return d
		}
		return d.Round(uint8(places), mode)
	case d.coef.IsZero():
		return Zero
	}

	n := -int(places) // the result is a multiple of 10^n
	if n > 2*int(MaxScale) {
		// 10^n is past the largest coefficient, so nothing is even half of it: the value rounds to zero, and a mode
		// that insists on moving away from it asks for a multiple that the type cannot hold.
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		if roundDecision(false, false, false, d.state, mode) {
			return Dec128{state: state.Overflow}
		}
		return Zero
	}

	// Drop d.scale + n digits, round the rest, and put the n back as zeros.
	qw, above, tie, inexact := wideFrom128(d.coef).quoCmpHalfPow10(uint8(int(d.scale) + n))
	q, _ := qw.uint128() // the quotient is at most the coefficient itself
	if inexact {
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		if roundDecision(above, tie, q.Lo&1 == 1, d.state, mode) {
			q, _ = q.AddCarry(uint128.One) // below 10^38 after the division, so it cannot carry out
		}
	}
	if q.IsZero() {
		return Zero // a zero is never negative
	}

	coef, s := q.Mul(Pow10Uint128[n])
	if s >= state.Error {
		return Dec128{state: state.Overflow}
	}
	return Dec128{coef: coef, state: d.state}
}

// RoundToMultiple returns the multiple of m nearest to d under mode: Swiss and Swedish cash rounding to 0.05, the
// "round the payment up to the next whole 10" of retail lending, and any other grid that is not a power of ten.
//
// The quotient d/m is formed exactly, with its remainder, so there is one rounding and it is exact; the result is
// then that integer count of m and carries m's scale, because it is a whole number of them. m must be strictly
// positive: zero or negative is NaN(DomainError).
//
// NaN(Overflow) when the result does not fit, and also when the count of multiples does not, which is the case of an
// m so much finer than d that d/m needs more than 38 digits. An undefined mode is NaN(InvalidRoundingMode), and under
// ROUND_NAN a d that is not already a multiple of m is NaN(Inexact). NaN propagates. It reads no process-global
// configuration.
func (d Dec128) RoundToMultiple(m Dec128, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case m.state >= state.Error:
		return m
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case m.coef.IsZero() || m.state == state.Neg:
		return Dec128{state: state.DomainError}
	case d.coef.IsZero():
		return Dec128{scale: m.scale}
	}

	// The exact integer quotient and its remainder, both at the common scale of the two operands.
	q, rem, ok := d.tryQuoRem(m)
	if !ok {
		return Dec128{state: state.Overflow}
	}

	if !rem.IsZero() {
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		// The rounding decision compares twice the remainder with m at that same common scale; both are magnitudes
		// below 2^128, so doubling is exact in the 192-bit register.
		a, b, _ := alignOperands(rem.Abs(), m)
		twice, carry := a.lo.AddCarry(a.lo)
		c := compareWidened(widened{lo: twice, hi: a.hi*2 + carry}, b)
		if roundDecision(c > 0, c == 0, q.coef.Lo&1 == 1, d.state, mode) {
			var out uint64
			if q.coef, out = q.coef.AddCarry(uint128.One); out != 0 {
				return Dec128{state: state.Overflow}
			}
			q.state = d.state
		}
	}

	if q.coef.IsZero() {
		return Dec128{scale: m.scale} // a zero is never negative
	}
	// An exact multiplication: the count has scale 0, so the product has m's scale and only its width can fail.
	return q.MulRound(m, m.scale, ROUND_TOWARD_ZERO)
}

// ScaleByPow10 returns d * 10^k, exactly or not at all.
//
// It is the conversion between a percentage, a basis point, a per-mille and a fraction, and it should never go
// through a division: moving the decimal point is a change of scale and leaves the coefficient alone whenever the new
// scale is one the type has. Where it is not, the coefficient is multiplied or divided by the difference, which is
// still exact or NaN and never rounded.
//
// NaN(Overflow) when the value has grown past the largest coefficient, NaN(Underflow) when it has shrunk past the
// smallest quantum or would need places the type does not have, and a NaN operand propagates. It reads no
// process-global configuration.
func (d Dec128) ScaleByPow10(k int) Dec128 {
	if d.state >= state.Error || k == 0 || d.coef.IsZero() {
		return d
	}

	// The reachable window is narrow, and deciding the outside of it first is what keeps the subtraction below from
	// wrapping for a k near the ends of an int. Growing needs coef * 10^(k-scale) to fit 128 bits, so k is at most
	// 38 + scale <= 3*MaxScale; shrinking needs scale + k places, so -k is at most 38 + (MaxScale - scale) <=
	// 3*MaxScale. One step past either end is an overflow or an underflow whatever the coefficient is.
	switch {
	case k > 3*int(MaxScale):
		return Dec128{state: state.Overflow}
	case k < -3*int(MaxScale):
		return Dec128{state: state.Underflow}
	}

	t := int(d.scale) - k // the scale the result wants
	switch {
	case t >= 0 && t <= int(MaxScale):
		return Dec128{coef: d.coef, scale: uint8(t), state: d.state}

	case t < 0:
		// more integer digits than the coefficient carries: multiply it up, at scale 0
		if -t > 2*int(MaxScale) {
			return Dec128{state: state.Overflow}
		}
		coef, s := d.coef.Mul(Pow10Uint128[-t])
		if s >= state.Error {
			return Dec128{state: state.Overflow}
		}
		return Dec128{coef: coef, state: d.state}

	default:
		// More decimal places than the type has. The value is still representable if the digits that fall off the end
		// are zeros - 1200 * 10^-21 is 0.0000000000000000012 - and unrepresentable rather than merely long if they
		// are not.
		j := t - int(MaxScale)
		if j > 2*int(MaxScale) {
			return Dec128{state: state.Underflow}
		}
		q, _, _, inexact := wideFrom128(d.coef).quoCmpHalfPow10(uint8(j))
		coef, _ := q.uint128() // the quotient is at most the coefficient itself
		if inexact || coef.IsZero() {
			return Dec128{state: state.Underflow}
		}
		return Dec128{coef: coef, scale: MaxScale, state: d.state}
	}
}

// RoundToSignificant rounds d to at most the given number of significant digits with mode.
//
// It is the grid a rate is quoted on rather than the one an amount is held on: an FX rate to five significant digits
// is 1.0987 at parity and 0.0000109 for a currency with a small unit, which no number of decimal places describes.
// It is also the precision-based rounding of IEEE 754 decimal arithmetic, where dec128 otherwise works to a scale.
//
// A value that already has no more significant digits than asked for is returned unchanged, as the Round* methods
// leave a shorter value alone. Zero has no significant digits and is returned unchanged.
//
// What the result is, exactly, is the value rounded at the decimal position that leaves the requested number of
// digits - and that is all a scale-based type can promise, because rounding can carry into a new digit that no scale
// can then drop: 9.99 to two significant digits is 10.0 and 999 to one is 1000, since neither 10 nor 1000 can be
// written with a leading digit and an exponent here. Use Canonical if the trailing zeros of such a result are
// unwanted.
//
// digits == 0 is NaN(DomainError), an undefined mode NaN(InvalidRoundingMode), a result that no longer fits
// NaN(Overflow), and under ROUND_NAN a discarded nonzero digit NaN(Inexact). NaN propagates. It reads no
// process-global configuration.
func (d Dec128) RoundToSignificant(digits uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case digits == 0:
		return Dec128{state: state.DomainError}
	}

	p := d.SignificantDigits()
	if p <= int(digits) {
		return d
	}

	// The digits to drop, expressed as the number of decimal places to keep, which RoundToPlaces takes negative when
	// the cut falls to the left of the point.
	places := int(d.scale) - (p - int(digits))
	if places >= 0 {
		return d.Round(uint8(places), mode)
	}

	return d.RoundToPlaces(int8(places), mode)
}
