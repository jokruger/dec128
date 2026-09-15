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

// Sum returns the exact sum of its arguments, rounded to fit only once, at the end. The terms are accumulated
// exactly, in an Accumulator, with the positive and negative ones kept apart so that the result does not depend on
// the order of the arguments and no intermediate is ever rounded or overflowed; the total is then reduced by the
// scale rule like any other result. The first NaN argument, in order, is returned as is.
//
// Use an Accumulator directly for a sum built up over time, or for a sum of products.
func Sum(a Dec128, b ...Dec128) Dec128 {
	// The accumulator does not escape, so this is still allocation-free.
	var acc Accumulator
	acc.Add(a)
	for _, d := range b {
		acc.Add(d)
	}
	if acc.state >= state.Error {
		return Dec128{state: acc.state}
	}

	m, st, scale := acc.total()
	if coef, ok := m.uint128(); ok {
		// exact and in range: every term came from Add, so the working scale is at most MaxScale
		if coef.IsZero() {
			st = state.Default
		}
		return Dec128{coef: coef, scale: scale, state: st}
	}

	return m.fit(scale, st, arithmeticRounding, lossPolicy, false)
}

// Avg returns the arithmetic mean of its arguments: the exact Sum divided by their count with Div, so the default
// scale, rounding mode and ideal-scale rule apply. For a global-free mean, total an Accumulator and divide it with
// DivRound.
func Avg(a Dec128, b ...Dec128) Dec128 {
	return Sum(a, b...).DivInt64(int64(len(b) + 1))
}
