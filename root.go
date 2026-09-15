package dec128

import (
	"math"
	"math/big"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// The n-th root, which is the one operation in the package whose exact intermediate does not fit a fixed register.
//
// Converting an effective rate to a periodic one - (1+i)^(1/12) - 1 - is the inverse of the compounding PowIntRound
// does, and a schedule needs both. The rounding decision for a root has to compare R^n with the radicand exactly, and
// R carries up to 39 digits, so R^n reaches 39*n digits: 117 for a cube root and thousands for a rate over a daily
// schedule. That is what math/big is for, and it is why this is the only operation here that allocates.

// maxRootDegree is the largest n NthRootRound computes. The exact comparison grows with n, so a degree beyond any
// real period count is refused rather than computed for minutes.
const maxRootDegree = 1024

// NthRootRound returns the n-th root of d at exactly the given scale, rounding with mode.
//
// It is the inverse of PowIntRound and the primitive of rate conversion: the monthly factor of an annual one is
// factor.NthRootRound(12, scale, mode). n == 1 is RescaleRound and n == 2 agrees with SqrtRound digit for digit.
//
// The result is correctly rounded. The root is taken as an exact integer root and the decision compares R^n with the
// radicand in full, so a value exactly half way between two representable results is a genuine tie and is broken by
// mode like any other - 0.25 to the power one half is 0.5, which at scale 0 is 0 under ROUND_BANK and 1 under
// ROUND_HALF_AWAY_FROM_ZERO.
//
// A negative d has a root only for an odd n, and it is the negated root of the magnitude; an even n on a negative d
// is NaN(DomainError), as are n <= 0 and an n above 1024. A zero is zero at the requested scale, a scale above
// MaxScale is NaN(ScaleOutOfRange), an undefined mode NaN(InvalidRoundingMode), and under ROUND_NAN a root that is
// not exact is NaN(Inexact). NaN propagates. NaN(Overflow) is reachable only through n == 1, where the operation is
// a rescale rather than a root.
//
// Unlike everything else in the package this allocates, because the exact comparison is wider than any fixed
// register: one root costs a few dozen math/big values. It reads no process-global configuration.
func (d Dec128) NthRootRound(n int, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case n <= 0 || n > maxRootDegree:
		return Dec128{state: state.DomainError}
	case n == 1:
		return d.RescaleRound(scale, mode)
	case d.state == state.Neg && n%2 == 0:
		return Dec128{state: state.DomainError}
	case d.coef.IsZero():
		return Dec128{scale: scale}
	}

	// The root of |d| at the requested scale is the n-th root of coef * 10^(n*scale - d.scale). A negative exponent
	// there would make the radicand a fraction, so it is kept as the pair A/B and every comparison below is
	// multiplied out: R is the largest integer with R^n * B <= A.
	ten, one := big.NewInt(10), big.NewInt(1)
	a, b := new(big.Int).SetBytes(bigBytes(d.coef)), big.NewInt(1)
	if e := n*int(scale) - int(d.scale); e >= 0 {
		a.Mul(a, new(big.Int).Exp(ten, big.NewInt(int64(e)), nil))
	} else {
		b.Exp(ten, big.NewInt(int64(-e)), nil)
	}

	// floor(nthroot(floor(A/B))) is floor(nthroot(A/B)): both are the largest R with R^n <= A/B, and for an integer
	// R^n that holds against the truncated value exactly as it does against the true one.
	r := intNthRoot(new(big.Int).Quo(a, b), n)

	nb := big.NewInt(int64(n))
	rn := new(big.Int).Exp(r, nb, nil)
	rn.Mul(rn, b)
	inexact := rn.Cmp(a) != 0

	st := d.state
	if inexact {
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		// Above half means (R + 1/2)^n < A/B, which cleared of both fractions is (2R+1)^n * B < 2^n * A.
		lhs := new(big.Int).Lsh(r, 1)
		lhs.Add(lhs, one)
		lhs.Exp(lhs, nb, nil)
		lhs.Mul(lhs, b)
		rhs := new(big.Int).Lsh(a, uint(n))
		c := lhs.Cmp(rhs)
		if roundDecision(c < 0, c == 0, r.Bit(0) == 1, st, mode) {
			r.Add(r, one)
		}
	}

	// The root fits a coefficient for every n >= 2, and n == 1 returned above: the largest value is 3.4x10^38 and
	// the finest scale 10^19, so even the square root reaches only 1.845x10^19 * 10^19 = 1.845x10^38, and adding
	// the rounding step's unit leaves it below 2^128. TestNthRootFitsACoefficient measures that bound.
	coef, _ := uint128.FromBigInt(r)
	if coef.IsZero() {
		return Dec128{scale: scale} // a zero is never negative
	}

	return Dec128{coef: coef, scale: scale, state: st}
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
