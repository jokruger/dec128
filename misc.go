package dec128

import (
	"github.com/jokruger/dec128/state"
)

// The small operations the General Decimal Arithmetic specification names and this package had not: copy-sign, logb,
// the reciprocal, compare-total and remainder-near.
//
// None of them is hard, and that is rather the point. Each is a line or two that every caller would otherwise write
// for themselves, and each has a corner - the sign of a zero, the exponent of a zero, the tie in a nearest-integer
// quotient - that is easy to get wrong once and never notice.

// CopySign returns d with the sign of e: the magnitude of the first operand and the sign of the second.
//
// It is the GDA copy-sign, and the use is transferring a direction onto a magnitude that was computed without one -
// a signed adjustment derived from an unsigned tolerance, a cashflow whose direction comes from the leg rather than
// the amount.
//
// This type has no negative zero, so CopySign of a zero is that zero whatever e's sign, and a value made zero by
// this operation loses the sign it was given. NaN propagates: d first, then e.
//
// It reads no process-global configuration and allocates nothing.
func (d Dec128) CopySign(e Dec128) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case e.state >= state.Error:
		return e
	case d.coef.IsZero():
		return Dec128{scale: d.scale} // a zero is never negative
	case e.state == state.Neg:
		return Dec128{coef: d.coef, scale: d.scale, state: state.Neg}
	}

	return Dec128{coef: d.coef, scale: d.scale, state: state.Default}
}

// Logb returns the exponent of the most significant digit of d, as an integer at scale 0.
//
// It is the GDA logb, and it answers "how big is this, in powers of ten" without taking a logarithm: Logb(250) is 2,
// Logb(2.50) is 0 and Logb(0.03) is -2. The value is exact and the operation is a digit count, not a series, so it
// costs nothing and is available where Log10 would be too expensive or too approximate.
//
// It is not IntegerDigits, which counts the digits before the point and is 0 for everything below one; Logb goes
// negative there, which is what makes it useful for scaling. The two agree only in that Logb is IntegerDigits-1 for a
// value of one or more.
//
// Logb(0) is NaN(DivisionByZero), which is the specification's -Infinity in the vocabulary this type has. NaN
// propagates.
//
// It reads no process-global configuration and allocates nothing.
func (d Dec128) Logb() Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case d.coef.IsZero():
		return Dec128{state: state.DivisionByZero}
	}

	return FromInt64(int64(d.coef.Len10() - 1 - int(d.scale)))
}

// Inv returns 1/d at the default scale, rounding with the arithmetic rounding mode.
//
// It is One.Div(d) named, and it is here because the reciprocal is common enough in rate work - an annual factor
// inverted to a discount factor, a price inverted to a yield basis - that writing the one is clearer than writing the
// division. Inv(0) is NaN(DivisionByZero). NaN propagates.
//
// Like Div it reads SetDefaultScale and SetArithmeticRounding; InvRound is the global-free spelling.
func (d Dec128) Inv() Dec128 {
	return One.Div(d)
}

// InvRound returns 1/d at exactly the given scale, rounding with mode.
//
// It is the global-free Inv, and it fails as DivRound does: NaN(DivisionByZero) for a zero d, NaN(ScaleOutOfRange)
// above MaxScale, NaN(InvalidRoundingMode) for an undefined mode, and NaN(Inexact) under ROUND_NAN when the
// reciprocal does not terminate at the requested scale - which is most of the time, 1/3 among them.
//
// It reads no process-global configuration and allocates nothing.
func (d Dec128) InvRound(scale uint8, mode RoundingMode) Dec128 {
	return One.DivRound(d, scale, mode)
}

// CmpTotal compares d and other by representation rather than by value, and returns -1, 0 or 1.
//
// It is the GDA compare-total: a total order in which numerically equal values written at different scales are still
// ordered, and only two values with the same sign, the same coefficient and the same scale compare equal. 1.5 and
// 1.50 are equal under Compare and Equal; under CmpTotal the one with more places is the lower, because the
// specification orders by exponent and 1.50 has the smaller one. For a negative pair the order reverses, as it does
// for the values themselves.
//
// Reach for it when a set of values has to have a deterministic order that a round trip through a wider format cannot
// change - sorting for a digest, ordering the keys of a report - and for that use Canonical first if what is wanted
// is the value order with the representations collapsed rather than separated.
//
// It diverges from the specification in one place, deliberately. GDA puts NaN above every number; this package's
// Compare puts NaN below, and having the two disagree about where NaN sits would be a worse trap than disagreeing
// with the specification. So NaN sorts below every value here too, and two NaNs are ordered by their state code, so
// that the order is total over NaNs as well.
//
// It reads no process-global configuration and allocates nothing.
func (d Dec128) CmpTotal(other Dec128) int {
	dNaN, oNaN := d.state >= state.Error, other.state >= state.Error
	switch {
	case dNaN && oNaN:
		switch {
		case d.state < other.state:
			return -1
		case d.state > other.state:
			return 1
		}
		return 0
	case dNaN:
		return -1
	case oNaN:
		return 1
	}

	if c := d.Compare(other); c != 0 {
		return c
	}

	// Numerically equal, so the exponent decides: the larger (more positive) exponent is higher, and this type's
	// exponent is -scale, so the smaller scale is higher. The order reverses for a negative pair, which here means
	// for two negative values - a zero is never negative, and two equal zeros are both non-negative.
	if d.scale == other.scale {
		return 0
	}

	higher := 1
	if d.state == state.Neg {
		higher = -1
	}
	if d.scale < other.scale {
		return higher
	}

	return -higher
}

// RemainderNear returns d - other * n, where n is the integer nearest to d/other and a tie goes to the even one.
//
// It is the GDA remainder-near and IEEE 754's remainder, and it differs from Mod in which multiple of the divisor is
// taken away: Mod truncates the quotient toward zero and leaves a remainder with the sign of d and a magnitude below
// |other|, while this one rounds the quotient to nearest and leaves a remainder in [-|other|/2, |other|/2] whose sign
// is whichever way the nearest multiple fell. RemainderNear(10, 6) is -2 where Mod(10, 6) is 4, and RemainderNear(10, 3)
// is 1 where Mod is 1.
//
// That is the operation to reach for when what matters is the distance to the nearest grid point rather than the
// position within a cell: how far a price is from the nearest tick, how far a date count is from the nearest whole
// period. RoundToMultiple rounds to that grid point; this returns what is left over, and the two agree - d is
// RoundToMultiple(other, ROUND_BANK) plus RemainderNear(other).
//
// The result is exact: no digits are discarded and no rounding mode applies, because the remainder of a terminating
// division is representable whenever the quotient is. A zero other is NaN(DivisionByZero), and a quotient too wide
// for a coefficient is NaN(Overflow), which are the conditions QuoRem fails under. A zero result is a zero, never
// negative. NaN propagates.
//
// It reads no process-global configuration and allocates nothing.
func (d Dec128) RemainderNear(other Dec128) Dec128 {
	q, r := d.QuoRem(other)
	if r.state >= state.Error || r.coef.IsZero() {
		return r
	}

	// QuoRem leaves d = other*q + r with |r| < |other| and r carrying the sign of d. The nearest-integer quotient is
	// q, or one further from zero in the direction of r when |r| is more than half of |other|; on a tie the even q
	// wins, which is q itself when q is even.
	//
	// The comparison is made in the wide register rather than on Dec128 values, because neither 2|r| nor |other| at
	// the common scale need fit in a coefficient even though r and the answer both do: a remainder of 1E-12 against a
	// divisor of 1E+30 aligns to a 42-digit divisor, and doubling a remainder that already fills its coefficient
	// carries out of it. Aligning costs at most 10^MaxScale and the doubling one bit, so 384 bits cover both.
	cs := max(r.scale, other.scale)
	ra, oa := wideFrom128(r.coef), wideFrom128(other.coef)
	if r.scale < cs {
		ra, _ = ra.mulPow10(cs - r.scale)
	}
	if other.scale < cs {
		oa, _ = oa.mulPow10(cs - other.scale)
	}
	twice, _ := ra.add(ra)

	switch c := twice.compare(oa); {
	case c < 0:
		return r
	case c == 0 && !q.odd():
		return r
	}

	// One step further out: the remainder becomes |other| - |r| carrying the opposite sign to r. Because |r| is more
	// than half of |other|, that difference is strictly below |r|, so it fits wherever r did.
	coef, ok := oa.sub(ra).uint128()
	if !ok {
		return Dec128{state: state.Overflow}
	}
	if coef.IsZero() {
		return Dec128{scale: cs} // a zero is never negative
	}

	st := state.Neg
	if r.state == state.Neg {
		st = state.Default
	}

	return Dec128{coef: coef, scale: cs, state: st}
}

// odd reports whether a value that is known to be an exact integer is odd. The quotient QuoRem returns always is.
func (d Dec128) odd() bool {
	if d.scale == 0 {
		return d.coef.Lo&1 == 1
	}
	t := d.Rescale(0)
	return t.state < state.Error && t.coef.Lo&1 == 1
}
