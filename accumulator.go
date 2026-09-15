package dec128

import (
	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
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
	floor    uint8       // the scale it was created with, which Reset returns it to
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
	a := &Accumulator{floor: scale}
	if scale > MaxScale {
		a.state = state.ScaleOutOfRange
		return a
	}
	a.scale = scale
	return a
}

// Reset empties the accumulator and returns it to the working scale it was created with, so that one can be reused
// for the next batch instead of allocating another. The terms and any failure recorded along the way are both
// cleared - except for the ScaleOutOfRange of an accumulator created with a scale it can never have, which emptying
// it cannot fix.
func (a *Accumulator) Reset() {
	bad := a.floor > MaxScale
	*a = Accumulator{scale: min(a.floor, MaxScale), floor: a.floor}
	if bad {
		a.state = state.ScaleOutOfRange
	}
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

// Mean returns the arithmetic mean of the terms added so far at exactly the given scale, rounding with mode. The
// exact total is divided by Count and the quotient is brought to scale, and the two together take one rounding
// decision rather than one each, so the result is the correctly rounded mean of the terms and not the rounded mean of
// a rounded total.
//
// An empty accumulator is NaN(DivisionByZero): the mean of nothing is not zero. Otherwise it fails as Total does,
// and also with NaN(Overflow) when a total that fills the register is asked for at a scale that needs padding.
//
// Terms that were added with AddMul count as one term each, so the mean of a weighted sum is that sum over the number
// of products in it.
func (a *Accumulator) Mean(scale uint8, mode RoundingMode) Dec128 {
	switch {
	case a.state >= state.Error:
		return Dec128{state: a.state}
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case a.count == 0:
		return Dec128{state: state.DivisionByZero}
	}

	m, st, ws := a.total()

	// Pad before dividing when the mean is wanted at a finer scale than the terms were kept at, so that the division
	// produces those places instead of the reduction having to discard them.
	//
	// The padding cannot carry out of the register. A term is below 2^256 and there are fewer than 2^63 of them,
	// because the count is an int, so the total at working scale zero is below 2^319; a working scale of ws carries
	// a further 10^ws, and padding to scale multiplies by 10^(scale-ws), so what is held is below 2^319 * 10^scale,
	// and scale is at most MaxScale: 2^319 * 10^19 < 2^383.
	if scale > ws {
		m, _ = m.mulPow10(scale - ws)
		ws = scale
	}

	q, r := m.quoRem64(uint64(a.count))

	// What the rounding decision has to weigh is r/count of the register's last place, plus the ws-scale digits the
	// reduction drops. When there are no digits to drop the remainder is the whole of it and decides on its own;
	// otherwise those digits are the more significant part and the remainder is no more than a sticky bit.
	var qw wide
	var above, tie, inexact bool
	if k := ws - scale; k == 0 {
		half := uint64(a.count) / 2
		qw, above, tie, inexact = q, r > half, uint64(a.count)%2 == 0 && r == half, r != 0
	} else {
		qw, above, tie, inexact = q.quoCmpHalfPow10Sticky(k, r != 0)
	}

	coef, ok := qw.uint128()
	if !ok {
		return Dec128{state: state.Overflow}
	}
	if inexact {
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		if roundDecision(above, tie, coef.Lo&1 == 1, st, mode) {
			var carry uint64
			if coef, carry = coef.AddCarry(uint128.One); carry != 0 {
				return Dec128{state: state.Overflow}
			}
		}
	}
	if coef.IsZero() {
		return Dec128{scale: scale} // a zero is never negative
	}

	return Dec128{coef: coef, scale: scale, state: st}
}
