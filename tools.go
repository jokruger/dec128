package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// NaN returns a Dec128 with the given error.
func NaN(reason state.State) Dec128 {
	if reason < state.Error {
		reason = state.Error
	}
	return Dec128{state: reason}
}

// MaxAtScale returns the largest finite decimal representable at the given scale, or NaN(ScaleOutOfRange) when the
// scale is above MaxScale.
func MaxAtScale(scale uint8) Dec128 {
	if scale > MaxScale {
		return Dec128{state: state.ScaleOutOfRange}
	}
	return Dec128{coef: uint128.Max, scale: scale, state: state.OK}
}

// MinAtScale returns the smallest finite (most negative) decimal representable at the given scale, or
// NaN(ScaleOutOfRange) when the scale is above MaxScale.
func MinAtScale(scale uint8) Dec128 {
	if scale > MaxScale {
		return Dec128{state: state.ScaleOutOfRange}
	}
	return Dec128{coef: uint128.Max, scale: scale, state: state.Neg}
}

// QuantumAtScale returns the quantum (unit in last place, or granularity) for the given scale. It represents the
// smallest positive increment distinguishable at that scale, i.e. 10^-scale. A scale above MaxScale yields
// NaN(ScaleOutOfRange).
func QuantumAtScale(scale uint8) Dec128 {
	if scale > MaxScale {
		return Dec128{state: state.ScaleOutOfRange}
	}
	return Dec128{coef: uint128.One, scale: scale, state: state.OK}
}

// Max returns the largest Dec128 value from the input list. The first NaN argument, in order, is returned as is, so a
// failed calculation cannot hide behind a valid one.
func Max(a Dec128, b ...Dec128) Dec128 {
	if a.state >= state.Error {
		return a
	}
	for _, d := range b {
		if d.state >= state.Error {
			return d
		}
		if d.GreaterThan(a) {
			a = d
		}
	}
	return a
}

// Min returns the smallest Dec128 value from the input list. The first NaN argument, in order, is returned as is.
func Min(a Dec128, b ...Dec128) Dec128 {
	if a.state >= state.Error {
		return a
	}
	for _, d := range b {
		if d.state >= state.Error {
			return d
		}
		if d.LessThan(a) {
			a = d
		}
	}
	return a
}

// Sum returns the exact sum of its arguments, rounded to fit only once, at the end. The terms are aligned to the
// largest scale and accumulated in 256 bits as separate positive and negative totals, so the result does not depend on
// the order of the arguments and no intermediate is ever rounded or overflowed; the total is then reduced by the scale
// rule like any other result. The first NaN argument, in order, is returned as is.
//
// It keeps a register of its own rather than using an Accumulator: every term here arrives through Add, so the aligned
// magnitudes fit 256 bits, and the wider register an Accumulator needs for AddMul costs several times the time. The
// two agree on every sum either can take. Use an Accumulator for a sum built up over time, or for a sum of products.
func Sum(a Dec128, b ...Dec128) Dec128 {
	if a.state >= state.Error {
		return a
	}
	scale := a.scale
	for _, d := range b {
		if d.state >= state.Error {
			return d
		}
		scale = max(scale, d.scale)
	}
	var acc sums256
	acc.add(a, scale)
	for _, d := range b {
		acc.add(d, scale)
	}
	pos, neg := acc.pos, acc.neg
	st := state.Default
	lo, hi := pos.lo, pos.hi
	if compare256(pos.lo, pos.hi, neg.lo, neg.hi) < 0 {
		st = state.Neg
		lo, hi, neg = neg.lo, neg.hi, pos
	}
	var borrow uint64
	lo, borrow = lo.SubBorrow(neg.lo)
	hi, _ = hi.SubBorrow(neg.hi)
	hi, _ = hi.SubBorrow(uint128.Uint128{Lo: borrow})

	if hi.IsZero() {
		if lo.IsZero() {
			st = state.Default
		}
		return Dec128{coef: lo, scale: scale, state: st}
	}

	return fitWide(lo, hi, scale, st, arithmeticRounding, lossPolicy)
}

// sumTotal is the exact part of a sum for the global-free twins: the terms aligned at the largest of their scales
// and totalled in the 256-bit register, returned as a magnitude, its sign and that scale. Positive and negative
// terms are kept apart and cancelled once, at the end, so the total does not depend on the order the terms arrived
// in. ok is false when a term is a NaN, and firstNaN then says which.
//
// Sum repeats this rather than calling it, which is the one piece of duplication in the package and is deliberate.
// Go does not inline a function containing a loop, so routing Sum through a call costs it 45.2ns to 47.0ns per
// eight-term sum - a measured 4%, repeatable and well outside the noise - and Sum is the hot path of exactly the
// batch reconciliation this library exists for. Narrowing the return did not recover it; the cost is the call.
// TestSumRoundMatchesSum pins the two against each other so the copies cannot drift.
func sumTotal(a Dec128, b []Dec128) (lo, hi uint128.Uint128, st state.State, scale uint8, ok bool) {
	if a.state >= state.Error {
		return uint128.Zero, uint128.Zero, state.Default, 0, false
	}

	scale = a.scale
	for _, d := range b {
		if d.state >= state.Error {
			return uint128.Zero, uint128.Zero, state.Default, 0, false
		}
		scale = max(scale, d.scale)
	}

	var acc sums256
	acc.add(a, scale)
	for _, d := range b {
		acc.add(d, scale)
	}
	pos, neg := acc.pos, acc.neg

	st = state.Default
	lo, hi = pos.lo, pos.hi
	if compare256(pos.lo, pos.hi, neg.lo, neg.hi) < 0 {
		st = state.Neg
		lo, hi, neg = neg.lo, neg.hi, pos
	}
	var borrow uint64
	lo, borrow = lo.SubBorrow(neg.lo)
	hi, _ = hi.SubBorrow(neg.hi)
	hi, _ = hi.SubBorrow(uint128.Uint128{Lo: borrow})

	return lo, hi, st, scale, true
}

// firstNaN returns the first NaN among the terms, in order. It is only called once sumTotal has reported that there
// is one, so the loop always finds one when a is not it, and the final return is the a-is-the-NaN case rather than a
// fallback.
func firstNaN(a Dec128, b []Dec128) Dec128 {
	if a.state < state.Error {
		for _, d := range b {
			if d.state >= state.Error {
				return d
			}
		}
	}
	return a
}

// SumRound returns the sum of its arguments at exactly the given scale, rounding with mode: Sum with the scale and
// the mode per call instead of the scale rule and the process-global configuration.
//
// The terms are aligned at the largest of their scales and totalled exactly, and only the final reduction to scale
// rounds, so the result does not depend on the order the terms arrived in. It is the one-shot form of an Accumulator
// totalled with Total, and it keeps the narrower register that a sum of terms - rather than a sum of products - needs.
//
// The scale and the mode come first because the terms are variadic. A total that does not fit at scale is
// NaN(Overflow), a scale above MaxScale NaN(ScaleOutOfRange), an undefined mode NaN(InvalidRoundingMode), and under
// ROUND_NAN a discarded nonzero digit is NaN(Inexact). The first NaN argument, in order, is returned as is. It reads
// no process-global configuration.
func SumRound(scale uint8, mode RoundingMode, a Dec128, b ...Dec128) Dec128 {
	switch {
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	}

	lo, hi, st, ws, ok := sumTotal(a, b)
	if !ok {
		return firstNaN(a, b)
	}

	return wideFrom256(lo, hi).at(ws, scale, st, mode)
}

// SumSliceRound is SumRound over a slice, and returns zero at the requested scale for an empty one.
//
// The slice comes first here and the scale and the mode last, as they do everywhere else in the package; SumRound
// takes them first only because its terms are variadic.
func SumSliceRound(xs []Dec128, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case len(xs) == 0:
		return Dec128{scale: scale}
	}

	return SumRound(scale, mode, xs[0], xs[1:]...)
}

// SumSlice is Sum over a slice, and returns Zero for an empty one. Sum takes its first term separately, so
// Sum(xs[0], xs[1:]...) panics in the caller when xs is empty, which is exactly where a total is most likely to come
// from a query that returned no rows.
func SumSlice(xs []Dec128) Dec128 {
	if len(xs) == 0 {
		return Zero
	}
	return Sum(xs[0], xs[1:]...)
}

// total256 is a 256-bit unsigned total of aligned magnitudes. Each term is below 2^128 * 10^19 < 2^192 and there are
// fewer than 2^63 of them, so it cannot overflow.
type total256 struct {
	lo, hi uint128.Uint128
}

func (acc *total256) add(d Dec128, scale uint8) {
	lo, hi := d.coef.Mul64Carry(Pow10Uint64[scale-d.scale])
	var carry uint64
	acc.lo, carry = acc.lo.AddCarry(lo)
	acc.hi, _ = acc.hi.AddCarry(uint128.Uint128{Lo: hi + carry}) // hi < 2^64 - 1, no wrap
}

// sums256 keeps the positive and negative terms apart so that cancellation happens once, exactly, at the end.
type sums256 struct {
	pos, neg total256
}

func (s *sums256) add(d Dec128, scale uint8) {
	if d.state == state.Neg {
		s.neg.add(d, scale)
	} else {
		s.pos.add(d, scale)
	}
}

// AvgRound returns the arithmetic mean of its arguments at exactly the given scale, rounding with mode.
//
// The exact total is divided by the number of terms and the quotient is brought to scale, and the two take one
// rounding decision together rather than one each, so the result is the correctly rounded mean of the terms and not
// the rounded mean of a rounded total. Avg by contrast sums under the scale rule and then divides with Div, so it
// rounds twice and reads the process-global configuration both times.
//
// It fails as SumRound does. It reads no process-global configuration.
func AvgRound(scale uint8, mode RoundingMode, a Dec128, b ...Dec128) Dec128 {
	switch {
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	}

	lo, hi, st, ws, ok := sumTotal(a, b)
	if !ok {
		return firstNaN(a, b)
	}

	return meanFromWide(wideFrom256(lo, hi), ws, len(b)+1, st, scale, mode)
}

// Avg returns the arithmetic mean of its arguments: the exact Sum divided by their count with Div, so the default
// scale, rounding mode and ideal-scale rule apply. Use AvgRound for the scale and the mode per call, and for the
// single rounding: this form rounds the total and then rounds the quotient.
func Avg(a Dec128, b ...Dec128) Dec128 {
	return Sum(a, b...).DivInt64(int64(len(b) + 1))
}
