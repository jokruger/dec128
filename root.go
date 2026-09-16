package dec128

import (
	"math"
	"math/big"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Roots and rational powers: the two operations in the package whose exact intermediate does not fit any fixed
// register, and therefore the two that allocate.
//
// Converting an effective rate to a periodic one - (1+i)^(1/12) - 1 - is the inverse of the compounding PowIntRound
// does, and a schedule needs both. The rounding decision for a root has to compare R^n with the radicand exactly, and
// R carries up to 39 digits, so R^n reaches 39*n digits: 117 for a cube root and thousands for a rate over a daily
// schedule. A rational power is wider again, 39*|p| + MaxScale*q digits. That is what math/big is for.
//
// What the width buys is a stronger guarantee than the rest of the package can offer: both are correctly rounded,
// because nothing is approximated on the way to the decision. PowIntRound, which stays fixed-width, is faithfully
// rounded and says so.

// maxRootDegree is the largest degree NthRootRound computes, and the largest magnitude PowRational accepts for
// either half of its exponent once the fraction is reduced. Beyond it the operation is refused rather than computed.
//
// The work grows with the degree, so the bound is a measurement and not a caution. With the worst operand there is -
// a full-width 38-digit coefficient at MaxScale, taken to a scale of MaxScale - a degree of 1024 costs 1ms with a
// numerator of one and 3ms with p and q coprime, which is what the reduction leaves; 10950 costs 23ms and 89ms;
// 16384 costs 44ms and 151ms. 32768 was measured outside the package at 734ms for the coprime pair, which is what
// put the bound below it. TestPowRationalCost holds the top of the range to those figures.
//
// 16384 covers every schedule a real instrument produces: 30 years of monthly rests is 360, 30 years of daily rests
// is 10950 and 40 years of them is 14600. It refuses a century of daily rests, which is not an instrument.
//
// The bound is on p and q and never on elapsed time. A time-based bail-out would accept an input on a fast machine
// and refuse it on a slow one, which is the architecture-dependent behavior the amd64/arm64 matrix exists to catch;
// an input-size bound accepts and refuses identically everywhere.
const maxRootDegree = 16384

// NthRootRound returns the n-th root of d at exactly the given scale, rounding with mode.
//
// It is the inverse of PowIntRound and the primitive of rate conversion: the monthly factor of an annual one is
// factor.NthRootRound(12, scale, mode). n == 1 is RescaleRound and n == 2 agrees with SqrtRound digit for digit. It
// is PowRational with a numerator of one, and shares that method's machinery.
//
// The result is correctly rounded. The root is taken as an exact integer root and the decision compares R^n with the
// radicand in full, so a value exactly half way between two representable results is a genuine tie and is broken by
// mode like any other - 0.25 to the power one half is 0.5, which at scale 0 is 0 under ROUND_BANK and 1 under
// ROUND_HALF_AWAY_FROM_ZERO.
//
// A negative d has a root only for an odd n, and it is the negated root of the magnitude; an even n on a negative d
// is NaN(DomainError), as are n <= 0 and an n above maxRootDegree. A zero is zero at the requested scale, a scale
// above MaxScale is NaN(ScaleOutOfRange), an undefined mode NaN(InvalidRoundingMode), and under ROUND_NAN a root
// that is not exact is NaN(Inexact). NaN propagates. NaN(Overflow) is reachable only through n == 1, where the
// operation is a rescale rather than a root.
//
// Unlike most of the package this allocates, because the exact comparison is wider than any fixed register: one root
// costs a few dozen math/big values. It reads no process-global configuration.
func (d Dec128) NthRootRound(n int, scale uint8, mode RoundingMode) Dec128 {
	if n == 1 && d.state < state.Error && scale <= MaxScale && mode.IsValid() {
		// a rescale rather than a root, and worth taking without touching math/big
		return d.RescaleRound(scale, mode)
	}
	return d.PowRational(1, int64(n), scale, mode)
}

// PowRational returns d raised to the power p/q at exactly the given scale, rounding with mode.
//
// It is the companion of NthRootRound, which is the case p == 1, and of PowIntRound, which is q == 1: the fractional
// exponent of a compound change spread over a number of periods, and the geometric-mean family. Taking the root and
// the power separately rounds twice, and the root is where the significant digits are lost, so that error is then
// amplified by the power; here the fraction is applied in one step.
//
// The result is correctly rounded, which is a stronger guarantee than PowIntRound's, and it holds because the
// decision is made exactly rather than with guard digits. d^(p/q) is the q-th root of d^p, and d^p is an exact
// rational, so the whole computation is one integer root and two integer comparisons - R^q against the radicand for
// exactness, and (2R+1)^q against it for the half-way test - with nothing rounded on the way.
//
// The fraction is reduced before anything else, so 2/4 is 1/2 and 2/6 is 1/3. That is the convention that makes
// d^(p/q) a single-valued continuous function of its exponent, and for a negative d it is the whole of the answer:
// the reduced denominator decides, so (-8)^(2/6) is (-8)^(1/3) and thus -2, not (64)^(1/6) and thus 2, and
// (-4)^(2/4) is NaN(DomainError) rather than 2. A caller who wants the other reading should raise first and take the
// root afterwards. A negative d has a value only for an odd reduced q, and it is negative when the reduced p is odd;
// an even reduced q on a negative d is NaN(DomainError). q <= 0 is NaN(DomainError), as is a
// reduced |p| or q above maxRootDegree - see that constant for the measurements the bound comes from. p == 0 is 1
// for every d, including zero and including a d whose powers would overflow. A zero d is zero at the requested scale
// for a positive p and NaN(DivisionByZero) for a negative one. A result that does not fit is NaN(Overflow), a scale
// above MaxScale NaN(ScaleOutOfRange), an undefined mode NaN(InvalidRoundingMode), and under ROUND_NAN a result that
// is not exact is NaN(Inexact). NaN propagates.
//
// Like NthRootRound this allocates: the radicand reaches 39*|p| + MaxScale*q decimal digits, far wider than any
// fixed register. It reads no process-global configuration.
func (d Dec128) PowRational(p, q int64, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case q <= 0:
		return Dec128{state: state.DomainError}
	case p == 0:
		// one at exactly the requested scale; the padding always fits
		return Dec128{coef: Pow10Uint128[scale], scale: scale}
	}

	// |p| as a magnitude: negating in uint64 is correct for math.MinInt64 as well.
	pm, qm := uint64(p), uint64(q)
	if p < 0 {
		pm = -pm
	}

	// Reduce before the bound is applied, so that a fraction written large but meaning little - 2/4, or 16384/16384 -
	// is accepted on the value it actually has.
	g := gcdUint64(pm, qm)
	pm, qm = pm/g, qm/g

	switch {
	case pm > maxRootDegree || qm > maxRootDegree:
		return Dec128{state: state.DomainError}
	case d.state == state.Neg && qm%2 == 0:
		// an even root of a negative value is not a real number
		return Dec128{state: state.DomainError}
	case d.coef.IsZero():
		if p < 0 {
			return Dec128{state: state.DivisionByZero}
		}
		return Dec128{scale: scale} // a zero is never negative
	case p > 0 && pm == 1 && qm == 1:
		// the fraction reduced to one: a rescale rather than a power, and worth taking without math/big
		return d.RescaleRound(scale, mode)
	}

	st := state.Default
	if d.state == state.Neg && pm%2 == 1 {
		st = state.Neg
	}

	return powRatAt(d.coef, d.scale, p < 0, pm, qm, scale, st, mode)
}

// gcdUint64 returns the greatest common divisor of a and b, with gcd(a, 0) == a.
func gcdUint64(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// log10Coef returns log10 of a nonzero coefficient. It is only ever used a full decade clear of a decision boundary,
// so the float64 rounding in it cannot reach a returned value.
func log10Coef(c uint128.Uint128) float64 {
	return math.Log10(math.Ldexp(float64(c.Hi), 64) + float64(c.Lo))
}

// powRatAt computes (coef * 10^-ds) ** (+/-pm/qm) at exactly the scale s, for a result of sign st, rounding with
// mode. pm and qm are coprime and both in 1..maxRootDegree, and coef is nonzero.
//
// The coefficient of the result is the qm-th root of coef^(+/-pm) * 10^(s*qm -/+ ds*pm), which is an exact rational
// and is kept as the pair A/B so that a negative exponent needs no fraction: every comparison below is multiplied
// out by B.
func powRatAt(
	coef uint128.Uint128,
	ds uint8,
	negExp bool,
	pm, qm uint64,
	s uint8,
	st state.State,
	mode RoundingMode,
) Dec128 {
	// How many decimal digits the result will have, estimated before anything is built. A large |p| is nearly always
	// invalid input rather than expensive input - the result still has to fit 39 digits - and without this an
	// exponent well outside the representable range would build a radicand of hundreds of thousands of digits only
	// to report that it does not fit. The test fires only a full decade clear of either boundary, so everything that
	// is even close falls through to the exact path below: this is nthRootGuess's rule again, a float64 that chooses
	// a shortcut and never a returned value, which is what keeps the result identical on every architecture.
	e10 := float64(pm) / float64(qm)
	if negExp {
		e10 = -e10
	}
	switch lg := e10*(log10Coef(coef)-float64(ds)) + float64(s); {
	case lg > 40:
		// a coefficient holds at most 39 digits, so this is out of range by more than a decade
		return Dec128{state: state.Overflow}
	case lg < -2:
		// below a hundredth of the last place, so the coefficient truncates to zero and what is discarded is nowhere
		// near half: only a directed mode reaches the first unit
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		if roundDecision(false, false, false, st, mode) {
			return Dec128{coef: uint128.One, scale: s, state: st}
		}
		return Dec128{scale: s} // a zero is never negative
	}

	return powRatExact(coef, ds, negExp, pm, qm, s, st, mode)
}

// powRatExact is powRatAt without the magnitude shortcut: the whole computation in exact integers. It is separate so
// that TestPowRationalShortCircuitAgrees can put the two side by side and require the shortcut never to change a
// returned value.
func powRatExact(
	coef uint128.Uint128,
	ds uint8,
	negExp bool,
	pm, qm uint64,
	s uint8,
	st state.State,
	mode RoundingMode,
) Dec128 {
	ten, one := big.NewInt(10), big.NewInt(1)
	a, b := big.NewInt(1), big.NewInt(1)

	// coef^(+/-pm): the numerator for a positive exponent, the denominator for a negative one.
	cp := new(big.Int).Exp(new(big.Int).SetBytes(bigBytes(coef)), big.NewInt(int64(pm)), nil)
	if negExp {
		b = cp
	} else {
		a = cp
	}

	// 10^(s*qm -/+ ds*pm). Both terms are below MaxScale*maxRootDegree, so the exponent cannot overflow an int64.
	e := int64(s) * int64(qm)
	if negExp {
		e += int64(ds) * int64(pm)
	} else {
		e -= int64(ds) * int64(pm)
	}
	if e >= 0 {
		a.Mul(a, new(big.Int).Exp(ten, big.NewInt(e), nil))
	} else {
		b.Mul(b, new(big.Int).Exp(ten, big.NewInt(-e), nil))
	}

	// floor(nthroot(floor(A/B))) is floor(nthroot(A/B)): both are the largest R with R^qm <= A/B, and for an integer
	// R^qm that holds against the truncated value exactly as it does against the true one.
	t := new(big.Int).Quo(a, b)
	r := t
	if qm > 1 {
		r = intNthRoot(t, int(qm))
	}

	nb := big.NewInt(int64(qm))
	rn := new(big.Int).Exp(r, nb, nil)
	rn.Mul(rn, b)
	inexact := rn.Cmp(a) != 0

	// Overflow before Inexact, as reduceAt, quotientAt and every other reduction in the package order them: a result
	// that does not fit is an overflow whatever the mode would have done with what lies below it. Unlike a root, a
	// power can leave the range - only qm >= 2 with pm == 1 is bounded by the argument in
	// TestNthRootFitsACoefficient - so this is reachable here where it is not in NthRootRound.
	c128, sc := uint128.FromBigInt(r)
	if sc >= state.Error {
		return Dec128{state: state.Overflow}
	}

	if inexact {
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		// Above half means (R + 1/2)^qm < A/B, which cleared of both fractions is (2R+1)^qm * B < 2^qm * A.
		lhs := new(big.Int).Lsh(r, 1)
		lhs.Add(lhs, one)
		lhs.Exp(lhs, nb, nil)
		lhs.Mul(lhs, b)
		rhs := new(big.Int).Lsh(a, uint(qm))
		c := lhs.Cmp(rhs)
		if roundDecision(c < 0, c == 0, r.Bit(0) == 1, st, mode) {
			// the increment can itself leave the range, exactly as a carry out of a rounded quotient does
			if c128, sc = uint128.FromBigInt(r.Add(r, one)); sc >= state.Error {
				return Dec128{state: state.Overflow}
			}
		}
	}

	if c128.IsZero() {
		return Dec128{scale: s} // a zero is never negative
	}

	return Dec128{coef: c128, scale: s, state: st}
}

// bigBytes returns the coefficient's big-endian bytes, for big.Int.SetBytes.
func bigBytes(u uint128.Uint128) []byte {
	b := u.BytesBigEndian()
	return b[:]
}

// nthRootGuess returns a value above v ** (1/n), close enough that the iteration that follows converges
// quadratically from it.
//
// A guess within a factor of two, which the bit length alone gives, is not close enough: the iteration contracts by
// (n-1)/n per step while it is far from the root, so a degree in the hundreds would take hundreds of steps over
// numbers thousands of digits wide. The root is taken in float64 logarithms instead - v as m * 2^k, so that the
// logarithm is in range whatever the width - which is accurate to a relative 1e-16 and therefore well inside the
// 1/n the quadratic regime needs. The margin added afterwards covers that error and puts the guess on the high
// side, which is the side the iteration descends from.
func nthRootGuess(v *big.Int, n int) *big.Int {
	k := v.BitLen()
	m, _ := new(big.Float).SetMantExp(new(big.Float).SetInt(v), -k).Float64() // m in [0.5, 1)
	g := math.Exp((math.Log(m) + float64(k)*math.Ln2) / float64(n))
	if g <= 0 || math.IsInf(g, 0) || math.IsNaN(g) {
		// Unreachable for a v and an n this package passes, and SetFloat64 panics on a NaN, so the coarse guess
		// stands as the fallback.
		return bitLenGuess(v, n)
	}
	x, _ := new(big.Float).SetFloat64(g * (1 + 1e-9)).Int(nil)
	return x.Add(x, big.NewInt(2))
}

// bitLenGuess returns 2^(bitlen(v)/n + 1), which is above v ** (1/n) for every v and n because v is below
// 2^bitlen(v). It is correct but coarse, and only the fallback: see nthRootGuess for why the distance matters.
func bitLenGuess(v *big.Int, n int) *big.Int {
	return new(big.Int).Lsh(big.NewInt(1), uint(v.BitLen()/n+1))
}

// intNthRoot returns floor(v ** (1/n)) for v >= 0 and n >= 2.
func intNthRoot(v *big.Int, n int) *big.Int {
	if v.Sign() == 0 {
		return new(big.Int)
	}
	if v.BitLen() <= n {
		return big.NewInt(1) // v < 2^n, so the root is between 1 and 2
	}
	return intNthRootFrom(v, n, nthRootGuess(v, n))
}

// intNthRootFrom returns floor(v ** (1/n)) for v >= 1 and n >= 2, by Newton from the starting value x.
//
// The descent cannot stop above the root, whatever x is. While x is above it, x^n exceeds v, so v/x^(n-1) is below
// x and its floor is at most x-1; the next value is then at most ((n-1)x + x-1)/n, whose floor is x-1. So the step
// strictly decreases until it reaches the root, and the only correction the result can need is upward - which the
// loop below takes, for a start below the root and for the rare landing that truncation leaves one short.
func intNthRootFrom(v *big.Int, n int, x *big.Int) *big.Int {
	one := big.NewInt(1)
	nb, nm1 := big.NewInt(int64(n)), big.NewInt(int64(n-1))
	t, q, next := new(big.Int), new(big.Int), new(big.Int)

	for {
		// x <- ((n-1)*x + v/x^(n-1)) / n
		t.Exp(x, nm1, nil)
		if t.Sign() == 0 {
			break // x is zero, so the root is found by climbing
		}
		q.Quo(v, t)
		next.Mul(x, nm1)
		next.Add(next, q)
		next.Quo(next, nb)
		if next.Cmp(x) >= 0 {
			break
		}
		x.Set(next)
	}

	for next.Add(x, one); t.Exp(next, nb, nil).Cmp(v) <= 0; next.Add(x, one) {
		x.Set(next)
	}

	return x
}
