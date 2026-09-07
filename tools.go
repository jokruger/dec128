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

	var acc sums
	acc.add(a, scale)
	for _, d := range b {
		acc.add(d, scale)
	}
	pos, neg := acc.pos, acc.neg

	st := state.Default
	lo, hi := pos.lo, pos.hi
	if compare256(pos.lo, pos.hi, neg.lo, neg.hi) < 0 {
		st = state.Neg
		lo, hi, pos, neg = neg.lo, neg.hi, neg, pos
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

	return fitWide(lo, hi, scale, st, arithmeticRounding)
}

// accumulator is a 256-bit unsigned total of aligned magnitudes. Each term is below 2^128 * 10^19 < 2^192 and there are
// fewer than 2^63 of them, so it cannot overflow.
type accumulator struct {
	lo, hi uint128.Uint128
}

func (acc *accumulator) add(d Dec128, scale uint8) {
	lo, hi := d.coef.Mul64Carry(Pow10Uint64[scale-d.scale])
	var carry uint64
	acc.lo, carry = acc.lo.AddCarry(lo)
	acc.hi, _ = acc.hi.AddCarry(uint128.Uint128{Lo: hi + carry}) // hi < 2^64 - 1, no wrap
}

// sums keeps the positive and negative terms apart so that cancellation happens once, exactly, at the end.
type sums struct {
	pos, neg accumulator
}

func (s *sums) add(d Dec128, scale uint8) {
	if d.state == state.Neg {
		s.neg.add(d, scale)
	} else {
		s.pos.add(d, scale)
	}
}

// Avg returns the arithmetic mean of its arguments: the exact Sum divided by their count with Div, so the default
// scale, rounding mode and ideal-scale rule apply.
func Avg(a Dec128, b ...Dec128) Dec128 {
	return Sum(a, b...).DivInt64(int64(len(b) + 1))
}
