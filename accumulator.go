package dec128

import (
	"github.com/jokruger/dec128/state"
)

// Accumulator is a running total held in a register wider than a Dec128, so that a sum, or a sum of products, stays
// exact until the single rounding in Total.
//
// It is the one mutable type in the package and the one with pointer receivers: everything else is an immutable value.
// That is deliberate. NPV, weighted averages, schedule totals and the reconciliation checks that a core-banking
// system runs over them are sums over an unbounded number of terms, and rounding each term before adding it is both
// biased and dependent on the order of the summation. With AddMul the whole sum is exact until the end, so
// "the parts add up to the whole" is an exact assertion rather than an epsilon check.
//
// The zero value is a usable empty accumulator at working scale 0; NewAccumulator only sets a higher floor. Positive
// and negative terms are kept apart and cancelled once, at the end, so the total does not depend on the order in
// which the terms arrived. Nothing here reads the process-global configuration.
//
// An Accumulator is not safe for concurrent use.
type Accumulator struct {
	pos, neg wide
	count    int
	scale    uint8       // working scale: the largest natural scale any term has needed
	state    state.State // sticky failure - the first NaN term, or a register that overflowed
}

// NewAccumulator returns an empty accumulator whose working scale starts at the given scale, which must not exceed
// MaxScale. The working scale is a floor rather than a fixed setting: a term that needs more places than the
// accumulator currently carries raises it, up to twice MaxScale, so that every Add and every AddMul is exact (a product
// of two values at MaxScale needs twice MaxScale). Give it the scale the total is expected to have, so that Total at
// that scale needs no padding.
//
// A scale above MaxScale makes the accumulator a NaN(ScaleOutOfRange) that stays one, in keeping with the package's
// rule that a failure is a value rather than a panic.
func NewAccumulator(scale uint8) *Accumulator {
	a := &Accumulator{}
	if scale > MaxScale {
		a.state = state.ScaleOutOfRange
		return a
	}
	a.scale = scale
	return a
}

// Count returns the number of terms added, whether or not any of them failed.
func (a *Accumulator) Count() int {
	return a.count
}

// IsNaN reports whether the accumulator has failed: a NaN term was added, or the total went outside the register.
// The failure is sticky and Total returns it.
func (a *Accumulator) IsNaN() bool {
	return a.state >= state.Error
}

// ErrorDetails returns the error the accumulator has failed with, or nil.
func (a *Accumulator) ErrorDetails() error {
	if a.state >= state.Error {
		return a.state.Error()
	}
	return nil
}

// grow raises the working scale to s, scaling the running totals to match. It reports whether the accumulator is
// still usable.
func (a *Accumulator) grow(s uint8) bool {
	if s <= a.scale {
		return true
	}
	k := s - a.scale
	pos, over := a.pos.mulPow10(k)
	if over {
		a.state = state.Overflow
		return false
	}
	neg, over := a.neg.mulPow10(k)
	if over {
		a.state = state.Overflow
		return false
	}
	a.pos, a.neg, a.scale = pos, neg, s
	return true
}

// addWide adds the magnitude term, which stands at the scale ts, to the positive or negative total according to st.
func (a *Accumulator) addWide(term wide, ts uint8, st state.State) {
	if !a.grow(ts) {
		return
	}
	// The alignment cannot carry out: a coefficient below 2^256 shifted by at most 10^38 stays below 2^383.
	term, _ = term.mulPow10(a.scale - ts)

	dst := &a.pos
	if st == state.Neg {
		dst = &a.neg
	}
	sum, over := dst.add(term)
	if over {
		a.state = state.Overflow
		return
	}
	*dst = sum
}

// Add adds d to the running total, exactly. A NaN term makes the accumulator NaN with that reason, and the first such
// term is the one that is kept.
func (a *Accumulator) Add(d Dec128) {
	a.count++
	if a.state >= state.Error {
		return
	}
	if d.state >= state.Error {
		a.state = d.state
		return
	}
	a.addWide(wideFrom128(d.coef), d.scale, d.state)
}

// Sub subtracts d from the running total, exactly.
func (a *Accumulator) Sub(d Dec128) {
	a.Add(d.Neg())
}

// AddMul adds the exact product x*y to the running total. The product is never rounded on the way in, which is the
// whole point of the type: a sum of products is one rounding rather than one per term.
func (a *Accumulator) AddMul(x, y Dec128) {
	a.count++
	if a.state >= state.Error {
		return
	}
	switch {
	case x.state >= state.Error:
		a.state = x.state
		return
	case y.state >= state.Error:
		a.state = y.state
		return
	}
	lo, hi := x.coef.MulCarry(y.coef)
	a.addWide(wideFrom256(lo, hi), x.scale+y.scale, signOf(x.state, y.state))
}

// SubMul subtracts the exact product x*y from the running total.
func (a *Accumulator) SubMul(x, y Dec128) {
	a.AddMul(x.Neg(), y)
}

// total returns the exact accumulated magnitude, its sign and the working scale it stands at.
func (a *Accumulator) total() (wide, state.State, uint8) {
	if a.pos.compare(a.neg) < 0 {
		return a.neg.sub(a.pos), state.Neg, a.scale
	}
	return a.pos.sub(a.neg), state.Default, a.scale
}

// Total returns the accumulated value at exactly the given scale, rounding with mode. This is the only rounding the
// accumulator performs, and it does not change the running total, so an accumulator can be totalled at several scales
// or carried on with afterwards.
//
// A total that does not fit in 128 bits at that scale is NaN(Overflow); under ROUND_NAN a discarded nonzero digit is
// NaN(Inexact). A scale above MaxScale is NaN(ScaleOutOfRange) and an undefined mode NaN(InvalidRoundingMode). A
// failure recorded earlier is returned as it was recorded.
func (a *Accumulator) Total(scale uint8, mode RoundingMode) Dec128 {
	switch {
	case a.state >= state.Error:
		return Dec128{state: a.state}
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	}
	m, st, ws := a.total()
	return m.at(ws, scale, st, mode)
}
