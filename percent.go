package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Multiplying by a factor that is quoted against a power of ten. A percentage, a per-mille, a basis point and a price
// quoted per hundred units are all "multiply, then move the point", and the obvious spelling of that - d.Mul(p)
// followed by a division by a hundred - makes two decisions where one will do: the product is brought to a scale
// first, and the quotient is rounded again afterwards. It also fails on a product that the result does not need,
// because an intermediate that overflows is an overflow even when the division would have brought it back:
// 3e37 * 50 per cent is NaN that way round and 1.5e37 here.
//
// Moving the decimal point is not a division, so it should not cost a rounding. The operations below fold it into the
// same reduction the exact product needs, which leaves exactly one.

// MulScaled returns d * f * 10^k under the scale rule.
//
// It is Mul with the decimal point moved: the exact product stands at scale d.Scale() + f.Scale() - k, and that is the
// result whenever it is a scale this type has and the coefficient fits. Where the scale is negative the coefficient is
// multiplied up and the result has scale 0; where it is above MaxScale, or the coefficient does not fit, the scale is
// reduced to the largest at which the integer part fits and the digits below it are discarded with the mode set by
// SetArithmeticRounding, which SetLossPolicy may refuse. Trailing zeros are kept, as Mul keeps them: 1.00 scaled by
// 10^-2 is 0.0100 and not 0.01.
//
// k is the count of places to move the point to the left of the product, negated: 2 for a per-cent, 3 for a per-mille,
// 4 for a basis point, and a negative k multiplies by a power of ten instead. MulPercent is the k = -2 case, and
// MulScaledRound is the global-free form that takes the scale and the mode per call.
//
// NaN(Overflow) when the integer part does not fit even at scale 0, and a NaN operand propagates: d first, then f.
// It reads SetArithmeticRounding and SetLossPolicy, and only when the exact result does not fit - exactly as Mul does.
func (d Dec128) MulScaled(f Dec128, k int) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case f.state >= state.Error:
		return f
	}

	lo, hi := d.coef.MulCarry(f.coef)
	if lo.IsZero() && hi.IsZero() {
		// A zero is never negative, and it keeps the scale the exact product would stand at wherever that is a scale.
		return Dec128{scale: uint8(min(max(int(d.scale)+int(f.scale)-k, 0), int(MaxScale)))}
	}

	st := signOf(d.state, f.state)

	// Settle the unreachable ends of k before the arithmetic, so that a k near the ends of an int cannot wrap the
	// subtraction below. The product is at least one unit, so 39 zeros added to it pass the largest coefficient
	// whatever the scales are, and 115 digits taken off it leave nothing whatever the scales are.
	switch {
	case k > 4*int(MaxScale):
		return Dec128{state: state.Overflow}
	case k < -6*int(MaxScale):
		return wide{}.fit(MaxScale, st, arithmeticRounding, lossPolicy, true)
	}

	needed := int(d.scale) + int(f.scale) - k // the scale the exact product stands at
	switch {
	case needed < 0:
		// More integer digits than the product carries: multiply it up, at scale 0.
		if -needed > 2*int(MaxScale) {
			return Dec128{state: state.Overflow}
		}
		n, over := wideFrom256(lo, hi).mulPow10(uint8(-needed))
		coef, ok := n.uint128()
		if over || !ok {
			return Dec128{state: state.Overflow}
		}
		return Dec128{coef: coef, state: st}

	case needed <= int(MaxScale) && hi.IsZero():
		// exact and in range, which is the whole of the money-sized case
		return Dec128{coef: lo, scale: uint8(needed), state: st}
	}

	return wideFrom256(lo, hi).fit(uint8(needed), st, arithmeticRounding, lossPolicy, false)
}

// MulScaledRound returns d * f * 10^k at exactly the given scale, rounding with mode.
//
// The product is formed exactly and the power of ten is folded into the single reduction that brings it to scale, so
// there is one rounding decision and it is made on the exact value. It is the global-free form of MulScaled and the
// general form of MulPercentRound: k is 2 for a per-cent, 3 for a per-mille, 4 for a basis point, and negative to
// multiply by a power of ten instead.
//
// A coefficient that does not fit at scale is NaN(Overflow), a scale above MaxScale NaN(ScaleOutOfRange), an undefined
// mode NaN(InvalidRoundingMode), and under ROUND_NAN a discarded nonzero digit NaN(Inexact). A NaN operand propagates,
// d first, then f. It reads no process-global configuration.
func (d Dec128) MulScaledRound(f Dec128, k int, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case f.state >= state.Error:
		return f
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case d.coef.IsZero() || f.coef.IsZero():
		return Dec128{scale: scale} // a zero is never negative
	}

	st := signOf(d.state, f.state)

	// The same two ends as MulScaled, for the same reason.
	switch {
	case k > 4*int(MaxScale):
		return Dec128{state: state.Overflow}
	case k < -6*int(MaxScale):
		return roundedFromNothing(scale, st, mode)
	}

	lo, hi := d.coef.MulCarry(f.coef)

	// g is the power of ten the exact product must be multiplied by to stand at scale.
	g := k + int(scale) - int(d.scale) - int(f.scale)
	if g >= 0 {
		// Scaling up is exact or it does not fit at all.
		if g > 2*int(MaxScale) {
			return Dec128{state: state.Overflow}
		}
		n, over := wideFrom256(lo, hi).mulPow10(uint8(g))
		coef, ok := n.uint128()
		if over || !ok {
			return Dec128{state: state.Overflow}
		}
		return Dec128{coef: coef, scale: scale, state: st}
	}

	// Scaling down: one reduction over the exact product. Overflow before Inexact, as reduceAt and every other
	// reduction in the package order them.
	qw, above, tie, inexact := wideFrom256(lo, hi).quoCmpHalfPow10(uint8(-g))
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

// MulPercent returns d * f percent, that is d * f / 100, under the scale rule.
//
// It is the MulScaled(f, -2) case, and it is here because it is the one every schedule of fees, rates and taxes is
// made of. The exact product of an amount and a rate carries two more places than the two of them together - 100.00
// times 7.5 per cent is 7.50000 - so a money-sized calculation is exact and the rounding to the presented scale is the
// caller's, at the end, where it belongs:
//
//	vat := net.MulPercent(rate).RoundBank(2)
//
// It fails as MulScaled does, and reads SetArithmeticRounding and SetLossPolicy only when the exact result does not
// fit. MulPercentRound is the global-free form.
func (d Dec128) MulPercent(f Dec128) Dec128 {
	return d.MulScaled(f, -2)
}

// MulPercentRound returns d * f percent, that is d * f / 100, at exactly the given scale, rounding with mode.
//
// It is MulScaledRound(f, -2, scale, mode): the product is exact and the division by a hundred is a move of the
// decimal point inside the single reduction, so the result is rounded once. Writing it as
// d.Mul(f).DivRound(Decimal100, scale, mode) reduces the product under the process-global mode before the quotient is
// rounded at all, and fails outright on a product that does not fit where the quotient would have.
//
// It fails as MulScaledRound does.
func (d Dec128) MulPercentRound(f Dec128, scale uint8, mode RoundingMode) Dec128 {
	return d.MulScaledRound(f, -2, scale, mode)
}

// roundedFromNothing is the result of a reduction that discarded every digit the value had: zero, or the first unit of
// the scale under a mode directed away from it. It is the wide.fit total-loss case without the loss policy, for the
// global-free operations that never consult it.
func roundedFromNothing(scale uint8, st state.State, mode RoundingMode) Dec128 {
	if mode == ROUND_NAN {
		return Dec128{state: state.Inexact}
	}
	if roundDecision(false, false, false, st, mode) {
		return Dec128{coef: uint128.One, scale: scale, state: st}
	}
	return Dec128{scale: scale} // a zero is never negative
}
