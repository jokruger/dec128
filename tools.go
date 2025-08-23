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

// MaxAtScale returns the largest finite decimal representable at the given scale.
func MaxAtScale(scale uint8) Dec128 {
	return Dec128{coef: uint128.Max, scale: scale, state: state.OK}
}

// MinAtScale returns the smallest finite (most negative) decimal representable at the given scale.
func MinAtScale(scale uint8) Dec128 {
	return Dec128{coef: uint128.Max, scale: scale, state: state.Neg}
}

// QuantumAtScale returns the quantum (unit in last place, or granularity) for the given scale. It represents the smallest positive increment distinguishable at that scale, i.e. 10^-scale.
func QuantumAtScale(scale uint8) Dec128 {
	return Dec128{coef: uint128.One, scale: scale, state: state.OK}
}

// Max returns the largest Dec128 value from the input list.
func Max(a Dec128, b ...Dec128) Dec128 {
	for _, d := range b {
		if d.GreaterThan(a) {
			a = d
		}
	}
	return a
}

// Min returns the smallest Dec128 value from the input list.
func Min(a Dec128, b ...Dec128) Dec128 {
	for _, d := range b {
		if d.LessThan(a) {
			a = d
		}
	}
	return a
}

// Sum returns the sum of the Dec128 values in the input list.
func Sum(a Dec128, b ...Dec128) Dec128 {
	for _, d := range b {
		a = a.Add(d)
	}
	return a
}

// Avg returns the average of the Dec128 values in the input list.
func Avg(a Dec128, b ...Dec128) Dec128 {
	return Sum(a, b...).DivInt64(int64(len(b) + 1))
}
