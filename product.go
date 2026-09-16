package dec128

import (
	"math/big"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// The product family: Sum's multiplicative twin, exact in the middle and rounded once at the end.
//
// Chain-linking is the operation these exist for - a growth factor over a series of periods is the product of
// (1 + r) over the series, and a fund's return over a year is the product of its daily factors. Folding Mul over the
// series rounds at every step, and those roundings are not noise: they all point the same way under a directed mode,
// and under a nearest mode they still accumulate as a random walk over several hundred terms. Prod forms the whole
// product exactly and makes one decision on it, so the answer does not depend on where the series was cut or in what
// order the factors arrived.
//
// # Why these allocate when Sum does not
//
// A sum of n terms is bounded by n times the largest of them, so 256 bits hold any sum this type can be asked for and
// Sum keeps its register on the stack. A product has no such bound: the exact product of n coefficients has the sum
// of their digit counts, which is 39n, and the exact scale is the sum of their scales. Twenty money-scale factors
// already need more than 384 bits, and a year of daily factors needs about 1500 decimal places - all of which must be
// carried, because the whole point is that the rounding decision is made on the exact value.
//
// So the width is inherent rather than incidental, and there is no fixed-width form to fall back to. What there is
// instead is Mul, which rounds every step and allocates nothing, and MulRound, which does the same with the scale and
// the mode per call. Use those when the factors are few and the intermediate rounding is acceptable; use these when
// the series is long and it is not.

// prodTotals returns the exact product of the operands as a magnitude and a total scale, with the sign the product
// carries. The bool is false when an operand is NaN, and the Dec128 is then the first such operand in order, which is
// what every variadic operation in the package returns.
func prodTotals(a Dec128, b []Dec128) (*big.Int, int, state.State, Dec128, bool) {
	// Every operand is scanned for a NaN before anything is multiplied, so that a NaN late in the series is reported
	// even when an earlier operand was zero. Sum has the same property and for the same reason: the argument list is
	// the caller's data, and a NaN in it is a fact about the data rather than about the arithmetic.
	if a.state >= state.Error {
		return nil, 0, state.Default, a, false
	}
	for _, d := range b {
		if d.state >= state.Error {
			return nil, 0, state.Default, d, false
		}
	}

	st := a.state
	ts := int(a.scale)
	zero := a.coef.IsZero()
	for _, d := range b {
		st = signOf(st, d.state)
		ts += int(d.scale)
		zero = zero || d.coef.IsZero()
	}
	if zero {
		// A zero factor makes the product zero, and a zero is never negative. The scale still accumulates, because it
		// is the scale the exact product would have had.
		return new(big.Int), ts, state.Default, Dec128{}, true
	}

	mag := a.coef.BigInt()
	for _, d := range b {
		mag.Mul(mag, d.coef.BigInt())
	}

	return mag, ts, st, Dec128{}, true
}

// prodAt reduces an exact magnitude at scale ts to a Dec128 at exactly the given scale, taking the one rounding. It
// returns NaN(Overflow) when the coefficient does not fit, which is what prodFit retries on.
func prodAt(mag *big.Int, ts int, st state.State, scale uint8, mode RoundingMode, policy LossPolicy) Dec128 {
	if ts <= int(scale) {
		// Padding, which is exact: there is nothing below the target scale to discard.
		q := new(big.Int).Mul(mag, pow10big(int(scale)-ts))
		coef, s := uint128.FromBigInt(q)
		if s >= state.Error {
			return Dec128{state: state.Overflow}
		}
		if coef.IsZero() {
			return Dec128{scale: scale} // a zero is never negative
		}
		return Dec128{coef: coef, scale: scale, state: st}
	}

	den := pow10big(ts - int(scale))
	q, rem := new(big.Int).QuoRem(mag, den, new(big.Int))

	// Overflow before Inexact, as every other reduction in the package orders them.
	coef, s := uint128.FromBigInt(q)
	if s >= state.Error {
		return Dec128{state: state.Overflow}
	}

	if rem.Sign() != 0 {
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		twice := new(big.Int).Lsh(rem, 1)
		c := twice.Cmp(den)
		if roundDecision(c > 0, c == 0, q.Bit(0) == 1, st, mode) {
			if coef, s = uint128.FromBigInt(q.Add(q, bigOne)); s >= state.Error {
				return Dec128{state: state.Overflow}
			}
		}
		// The quotient is zero only when every significant digit went; the operands were non-zero, because a zero
		// factor never reaches here.
		if ls := lossState(coef.IsZero(), policy); ls != state.OK {
			return Dec128{state: ls}
		}
	}

	if coef.IsZero() {
		return Dec128{scale: scale} // a zero is never negative
	}

	return Dec128{coef: coef, scale: scale, state: st}
}

// prodFit applies the scale rule to an exact magnitude: the largest scale at or below min(ts, MaxScale) whose
// coefficient fits 128 bits, with the digits below it discarded by mode and policy. It is fitWide's contract, over a
// magnitude too wide for fitWide's register.
func prodFit(mag *big.Int, ts int, st state.State, mode RoundingMode, policy LossPolicy) Dec128 {
	// The mode here comes from SetArithmeticRounding, where ROUND_NAN is the deprecated spelling of
	// SetLossPolicy(LossNaNOnInexact) rather than a direction of its own: refusing the loss is the policy's decision,
	// and a program that sets both - policy second, as the documentation requires - means the policy. fitWide and
	// wide.fit take it the same way, through roundDecision, which gives ROUND_NAN the direction of truncation.
	// prodAt's own ROUND_NAN branch is for the per-call ProdRound, where the mode does mean refusal.
	if mode == ROUND_NAN {
		mode = ROUND_TOWARD_ZERO
	}

	for s := min(ts, int(MaxScale)); s >= 0; s-- {
		d := prodAt(mag, ts, st, uint8(s), mode, policy)
		if d.state != state.Overflow {
			return d
		}
	}

	return Dec128{state: state.Overflow}
}

// Prod returns the exact product of its arguments, rounded to fit only once, at the end.
//
// The factors are multiplied exactly - no intermediate is rounded, whatever the scales - and the product is then
// reduced by the scale rule like any other result: it keeps the sum of the operands' scales when that is at most
// MaxScale and the coefficient fits, and otherwise the scale falls to the largest at which the integer part fits, with
// the discarded digits rounded by the mode set by SetArithmeticRounding. NaN(Overflow) comes back only when the
// integer part does not fit even at scale 0. The first NaN argument, in order, is returned as is.
//
// Prod(x) is x, and a zero factor anywhere makes the result zero - after the rest of the arguments have been checked
// for a NaN, so that bad data is reported rather than absorbed.
//
// It allocates; see the note at the top of this file for why there is no fixed-width form. Like Mul it reads
// SetArithmeticRounding and SetLossPolicy, and SetDefaultScale is not involved because the scale rule decides the
// scale. ProdRound is the global-free spelling.
func Prod(a Dec128, b ...Dec128) Dec128 {
	mag, ts, st, nan, ok := prodTotals(a, b)
	if !ok {
		return nan
	}
	if mag.Sign() == 0 {
		return Dec128{scale: uint8(min(ts, int(MaxScale)))}
	}

	return prodFit(mag, ts, st, arithmeticRounding, lossPolicy)
}

// ProdRound returns the product of its arguments at exactly the given scale, rounding with mode: Prod with the scale
// and the mode per call instead of the scale rule and the process-global configuration.
//
// The scale and the mode come first because the factors are variadic, as they do in SumRound and for the same reason.
// A product that does not fit at scale is NaN(Overflow), a scale above MaxScale NaN(ScaleOutOfRange), an undefined
// mode NaN(InvalidRoundingMode), and under ROUND_NAN a discarded nonzero digit is NaN(Inexact). The first NaN
// argument, in order, is returned as is.
//
// It allocates, and it reads no process-global configuration.
func ProdRound(scale uint8, mode RoundingMode, a Dec128, b ...Dec128) Dec128 {
	switch {
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	}

	mag, ts, st, nan, ok := prodTotals(a, b)
	if !ok {
		return nan
	}
	if mag.Sign() == 0 {
		return Dec128{scale: scale}
	}

	return prodAt(mag, ts, st, scale, mode, LossRound)
}

// ProdSlice is Prod over a slice, and returns One for an empty one.
//
// One is the empty product, as Zero is the empty sum: it is the identity of the operation, and it is what makes
// chain-linking a series that turned out to have no periods give the factor that changes nothing rather than the one
// that annihilates. Prod takes its first factor separately, so Prod(xs[0], xs[1:]...) panics in the caller when xs is
// empty, which is exactly where a series is most likely to have come from a query that returned no rows.
func ProdSlice(xs []Dec128) Dec128 {
	if len(xs) == 0 {
		return One
	}

	return Prod(xs[0], xs[1:]...)
}

// ProdSliceRound is ProdRound over a slice, and returns one at the requested scale for an empty one.
//
// The slice comes first here and the scale and the mode last, as they do everywhere else in the package; ProdRound
// takes them first only because its factors are variadic.
func ProdSliceRound(xs []Dec128, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case len(xs) == 0:
		return One.RescaleRound(scale, mode)
	}

	return ProdRound(scale, mode, xs[0], xs[1:]...)
}
