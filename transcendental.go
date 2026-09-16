package dec128

import (
	"math/big"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// The completeness tier: Exp, Ln, their cancellation-free companions Ln1p and Expm1, and Pow with a decimal exponent.
//
// These are the only operations in the package that are not exact. Everything else here computes an exact
// intermediate and makes one rounding decision on it, so "correctly rounded" is a property of the construction;
// a transcendental has no exact intermediate to make a decision on, because its value is irrational for every
// argument the type can hold except the handful where it is trivially an integer. What they do instead is carry
// transGuard decimal places beyond the caller's scale, bound the error of the series and the argument reduction well
// inside that, and round once at the end.
//
// # The guarantee, and what was measured
//
// These are faithfully rounded: the result is one of the two representable values either side of the exact one, so it
// is at most one unit in the last place away from correct. That is also what the General Decimal Arithmetic
// specification asks of exp, ln, log10 and power - "should be correctly rounded, but may be up to 1 ulp in error" -
// so the claim here is the reference standard's own, and not a weaker one. Three measurements stand behind it.
//
// Against testdata/transcendental_golden.csv - 80-digit values from an implementation of the General Decimal
// Arithmetic specification, in which exp and ln are correctly rounded - the public methods were compared at five
// scales in every rounding mode, 6097 results in all, and every one came back correctly rounded: the largest error
// was zero units in the last place. TestTranscendentalUlpError is that comparison.
//
// A fixed corpus can only hold the arguments somebody thought to write down, so the second measurement is randomized
// and independent. TestExpAgainstIndependentSeries and TestLnAgainstIndependentSeries recompute both functions in
// 512-bit binary floating point by the plain Taylor series with no argument reduction at all, and obtain the
// logarithm by inverting that series with Newton's method rather than taking a logarithm of its own. The reference
// therefore shares no constant, no reduction and no arithmetic with the code here. Over 28000 results the largest
// error was zero units in the last place.
//
// The third is the honest one for Pow, because the first two partly measure the dispatch: Pow sends a short rational
// exponent to PowRational, which is exact, so those rows never reach the series at all.
// TestPowExpLnAgainstPowRational calls the general arm directly, on the very exponents the dispatch would have routed
// away from it, and compares it with PowRational over random operands: 1515 results, largest error one unit in the
// last place.
//
// That one unit is not a shortage of guard digits, and no number of them would remove it. It appears where the exact
// value is itself representable - 19 to the power 24/8 is exactly 6859 - and a series converging on it from below is
// then truncated by a directed rounding mode. Deciding those correctly means proving the value exact, which is the
// table-maker's dilemma, and it is why correct rounding is not claimed. Where an exact answer matters and the
// exponent is a short fraction, PowRational gives one, and Pow reaches PowRational by itself.
//
// math/big supplies no transcendental functions - Float has Sqrt and nothing else - so the series and the reduction
// are written here. They work in fixed point over big.Int rather than in big.Float, because an integer at a known
// scale makes the error a countable number of units in the last place rather than something to reason about through
// a floating mantissa, and because that is how the rest of the package reasons.

// transGuard is how many decimal places beyond the caller's scale a transcendental carries while it works.
//
// The binding case is Exp. The squarings that undo its argument reduction multiply the relative error by 2^expHalvings,
// about 10^3, and a result may be as large as 10^38, so an absolute error of 10^-w in the reduced value becomes
// 10^(41-w) in the result. At w = scale + 60 that leaves 19 decimal places of slack below the caller's last digit,
// which is why a result landing close enough to a rounding boundary to be decided wrongly has never been observed.
// Ln is cheaper: its error is about 2^lnSqrts units in the last place of the working scale, and nothing amplifies it.
const transGuard = 60

// lnSqrts is how many times Ln halves the exponent of its reduced argument by taking a square root, and expHalvings
// how many times Exp halves its reduced argument before the series. Each reduction step buys faster convergence and
// costs accuracy - a square root adds a unit in the last place and every halving doubles the relative error of what
// follows - so both counts are a measured optimum rather than a round number, and transGuard is sized to absorb what
// they cost.
//
// The two are not the same size because the steps are not the same price. A halving in Exp is a shift and the
// squaring that undoes it is one multiplication, so ten of them are cheap and the count sits on a flat optimum
// anywhere from eight to twelve. A square root in Ln is a full big.Int Sqrt, which costs far more than the series
// terms it saves: profiled at MaxScale, ten of them were 84 per cent of the running time of Ln. Measured over a
// spread of mantissas across [1, 10) and of decimal exponents - a single benchmark argument will not show this, since
// a mantissa already close to one makes the roots pure overhead - the cost is flat between three and four and rises
// on both sides, by about a third at ten. Four is also the more accurate: the error is roughly 2^lnSqrts units in the
// last place of the working scale, so sixteen rather than a thousand.
const (
	lnSqrts     = 4
	expHalvings = 10
)

// lnTenScale and lnTenCoef hold ln(10) as an exact integer at that scale: the one constant these functions need, for
// Ln's decimal reduction and for Exp's. TestLnTenConstant checks it against the golden data, which comes from an
// implementation of the General Decimal Arithmetic specification and not from this package.
const lnTenScale = 130

const lnTenCoef = "2302585092994045684017991454684364207601101488628772976033327900967572609677352480235997205089598298341967784042286248633409525465" +
	"0"

// tenBig is the base every scaling here is in.
var tenBig = big.NewInt(10)

// PROTO: powers of ten are computed once and shared. Callers must not mutate the returned value.
const pow10CacheMax = 400

var pow10Cache = func() (t [pow10CacheMax + 1]*big.Int) {
	p := big.NewInt(1)
	for i := range t {
		t[i] = new(big.Int).Set(p)
		p.Mul(p, big.NewInt(10))
	}
	return
}()

// pow10big returns 10^n as a big.Int.
func pow10big(n int) *big.Int {
	if n >= 0 && n <= pow10CacheMax {
		return pow10Cache[n]
	}
	return new(big.Int).Exp(tenBig, big.NewInt(int64(n)), nil)
}

// fixedOne returns the fixed-point representation of one at scale w.
func fixedOne(w int) *big.Int {
	return new(big.Int).Set(pow10big(w))
}

// fixedMulTo is fixedMul writing into z, so the inner loops allocate nothing.
func fixedMulTo(z, a, b, p10 *big.Int) *big.Int {
	z.Mul(a, b)
	return z.Quo(z, p10)
}

// fixedSqrtTo returns the square root of a at the scale whose factor is p10, for a >= 0 - sqrt(a/10^w) is
// sqrt(a*10^w)/10^w - writing the result into z so that the reduction loop allocates nothing.
func fixedSqrtTo(z, a, p10 *big.Int) *big.Int {
	z.Mul(a, p10)
	return z.Sqrt(z)
}

// The scale factor 10^w is passed to these rather than recomputed inside them: they are called from the inner loop
// of every series here, and an exponentiation per iteration was most of what the first version of this file spent
// its time and its allocations on.

// fixedMul returns a*b at the scale whose factor is p10, truncating the digits below it.
func fixedMul(a, b, p10 *big.Int) *big.Int {
	t := new(big.Int).Mul(a, b)
	return t.Quo(t, p10)
}

// fixedDiv returns a/b at the scale whose factor is p10, truncating the digits below it. b must be nonzero.
func fixedDiv(a, b, p10 *big.Int) *big.Int {
	t := new(big.Int).Mul(a, p10)
	return t.Quo(t, b)
}

// lnTwoCoef holds ln(2) as an exact integer at lnTenScale, for Log2. It is the second and last constant these
// functions need. TestLnTwoConstant checks it two ways: against the golden data, and against what lnFixed computes
// for 2 at a far higher working scale, so a mistyped digit cannot survive in agreement with itself.
const lnTwoCoef = "69314718055994530941723212145817656807550013436025525412068000949339362196969471560586332699641868754200148102057068573368552" +
	"02357"

// lnTenBig and lnTwoBig are the stored constants parsed once, rather than on every call that needs them.
var (
	lnTenBig, _ = new(big.Int).SetString(lnTenCoef, 10)
	lnTwoBig, _ = new(big.Int).SetString(lnTwoCoef, 10)
)

// lnTen returns ln(10) at scale w, truncated from the stored constant. w is always far below lnTenScale.
func lnTen(w int) *big.Int {
	return new(big.Int).Quo(lnTenBig, pow10big(lnTenScale-w))
}

// lnTwo returns ln(2) at scale w, truncated from the stored constant, as lnTen does.
func lnTwo(w int) *big.Int {
	return new(big.Int).Quo(lnTwoBig, pow10big(lnTenScale-w))
}

// lnFixed returns ln(x) at scale w for a fixed-point x > 0 at scale w.
//
// The argument is reduced twice. First by the decimal exponent, x = m * 10^k with m in [1, 10), which costs one
// multiplication by ln(10) and is exact in the constant. Then by lnSqrts square roots, which bring m below
// 10^(1/2^lnSqrts) and so the series argument below about 0.072; the series is the odd-power expansion of
// 2*atanh((m-1)/(m+1)), which gains a little over two decimal digits a term at that argument and converges the faster
// the closer m is to one. The square roots are where the error comes from: each adds a unit in the last place, the errors above it are
// halved by every root below, and the total is multiplied back by 2^lnSqrts at the end - some sixteen units in the
// last place of the working scale, which transGuard covers many times over.
func lnFixed(x *big.Int, w int) *big.Int {
	p10 := pow10big(w)
	one := new(big.Int).Set(p10)

	// x = m * 10^k with m in [1, 10): k is the position of the leading digit relative to the scale.
	k := len(x.String()) - 1 - w
	m := new(big.Int).Set(x)
	if k > 0 {
		m.Quo(m, pow10big(k))
	} else if k < 0 {
		m.Mul(m, pow10big(-k))
	}

	// m <- m^(1/2^lnSqrts)
	for range lnSqrts {
		fixedSqrtTo(m, m, p10)
	}

	// z = (m-1)/(m+1), and ln(m) = 2*(z + z^3/3 + z^5/5 + ...)
	num := new(big.Int).Sub(m, one)
	den := new(big.Int).Add(m, one)
	z := fixedDiv(num, den, p10)
	zsq := fixedMul(z, z, p10)

	sum := new(big.Int).Set(z)
	term := new(big.Int).Set(z)
	tmp := new(big.Int)
	div := new(big.Int)
	for i := int64(3); ; i += 2 {
		fixedMulTo(term, term, zsq, p10)
		if term.Sign() == 0 {
			break
		}
		sum.Add(sum, tmp.Quo(term, div.SetInt64(i)))
	}
	sum.Lsh(sum, 1)       // the 2 of 2*atanh
	sum.Lsh(sum, lnSqrts) // undo the square roots
	if k != 0 {
		t := new(big.Int).Mul(lnTen(w), big.NewInt(int64(k)))
		sum.Add(sum, t)
	}

	return sum
}

// expFixed returns exp(v) at scale w for a signed fixed-point v at scale w, and whether the result is in a range
// worth computing. It is out of range when the result cannot fit any Dec128 at any scale, which the caller turns
// into NaN(Overflow) or into a zero.
//
// v is reduced to k*ln(10) + r with |r| <= ln(10)/2, so that the 10^k is an exact shift of the decimal point, and r
// is then halved expHalvings times to bring the Taylor argument below 1.2e-3. The squarings that undo that halving
// are where the error comes from: each doubles the relative error, so the series result is good to about 2^expHalvings
// units in the last place and the shift by 10^k adds nothing.
func expFixed(v *big.Int, w int) (*big.Int, int, bool) {
	p10 := pow10big(w)

	// k = round(v / ln 10), bounded before anything is computed: a Dec128 spans 10^-19 to 3.4*10^38.
	l10 := lnTen(w)
	q := fixedDiv(v, l10, p10)
	k64 := new(big.Int).Quo(q, p10).Int64()
	if rem := new(big.Int).Rem(q, p10); rem.Sign() != 0 {
		// round to nearest, which only needs to be approximate: any k within one of the best leaves |r| below ln(10)
		half := new(big.Int).Quo(p10, big.NewInt(2))
		if rem.Abs(rem).Cmp(half) > 0 {
			if v.Sign() < 0 {
				k64--
			} else {
				k64++
			}
		}
	}
	if k64 > 40 || k64 < -60 {
		return nil, int(k64), false
	}

	r := new(big.Int).Mul(l10, big.NewInt(k64))
	r.Sub(v, r)

	// r <- r / 2^expHalvings
	r.Rsh(r, expHalvings) // r may be negative; Rsh on a negative big.Int floors, which is a unit in the last place

	// exp(r) = 1 + r + r^2/2! + ...
	sum := new(big.Int).Set(p10)
	term := new(big.Int).Set(p10)
	div := new(big.Int)
	for i := int64(1); ; i++ {
		fixedMulTo(term, term, r, p10)
		term.Quo(term, div.SetInt64(i))
		if term.Sign() == 0 {
			break
		}
		sum.Add(sum, term)
	}

	// undo the halving
	for range expHalvings {
		fixedMulTo(sum, sum, sum, p10)
	}

	return sum, int(k64), true
}

// fixedToDec turns a signed fixed-point value at scale w into a Dec128 at exactly the requested scale, taking the one
// rounding the operation performs. shift is a power of ten still to be applied, which Exp leaves for here so that its
// argument reduction costs no precision: the value is sum * 10^shift / 10^w.
func fixedToDec(v *big.Int, w int, shift int, scale uint8, mode RoundingMode) Dec128 {
	st := state.Default
	if v.Sign() < 0 {
		st = state.Neg
	}
	mag := new(big.Int).Abs(v)

	// The coefficient is mag / 10^(w - shift - scale), and that exponent is always positive: w is scale + transGuard
	// and shift is the decimal exponent Exp separated out, which expFixed bounds well below transGuard. So this is
	// always a reduction and never a padding, and there is no second case to write.
	den := pow10big(w - shift - int(scale))
	q, rem := new(big.Int).QuoRem(mag, den, new(big.Int))

	// Overflow before Inexact, as every other reduction in the package orders them.
	coef, sc := uint128.FromBigInt(q)
	if sc >= state.Error {
		return Dec128{state: state.Overflow}
	}

	if rem.Sign() != 0 {
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		twice := new(big.Int).Lsh(rem, 1)
		c := twice.Cmp(den)
		if roundDecision(c > 0, c == 0, q.Bit(0) == 1, st, mode) {
			// The increment can in principle leave the range, and the check stays for that reason, but no argument
			// this type can hold reaches it: the carry needs the true value to sit within one unit in the last place
			// below 2^128, a window about 3e-48 wide in the argument of any of these functions, where the finest
			// argument expressible moves in steps of 1e-19. It is the one branch in the package with no test, and
			// TestTranscendentalRangeEnds says so where the test would otherwise be.
			if coef, sc = uint128.FromBigInt(q.Add(q, big.NewInt(1))); sc >= state.Error {
				return Dec128{state: state.Overflow}
			}
		}
	}

	if coef.IsZero() {
		return Dec128{scale: scale} // a zero is never negative
	}

	return Dec128{coef: coef, scale: scale, state: st}
}

// transArgs validates the arguments every one of these methods takes.
func transArgs(d Dec128, scale uint8, mode RoundingMode) (Dec128, bool) {
	switch {
	case d.state >= state.Error:
		return d, false
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}, false
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}, false
	}
	return Dec128{}, true
}

// fixedFromDec returns |d| as a fixed-point magnitude at scale w, which is exact because w is far above MaxScale.
func fixedFromDec(d Dec128, w int) *big.Int {
	m := new(big.Int).SetBytes(bigBytes(d.coef))
	return m.Mul(m, pow10big(w-int(d.scale)))
}

// Ln returns the natural logarithm of d at exactly the given scale, rounding with mode.
//
// It is faithfully rounded and not correctly rounded: the value is irrational for every argument but one, so there
// is no exact intermediate to make the decision on. Over the golden corpus the largest error observed was zero units
// in the last place, at every scale and in every mode; see "The guarantee, and what was measured" in this file for
// what that figure does and does not cover.
//
// Ln(1) is exactly zero. A d of zero or less is NaN(DomainError), a scale above MaxScale NaN(ScaleOutOfRange), an
// undefined mode NaN(InvalidRoundingMode), and under ROUND_NAN anything but an exact result is NaN(Inexact). NaN
// propagates. Use Ln1p when the argument is one plus a small value; see that method for why.
//
// It allocates, and it reads no process-global configuration.
func (d Dec128) Ln(scale uint8, mode RoundingMode) Dec128 {
	if bad, ok := transArgs(d, scale, mode); !ok {
		return bad
	}
	if d.state == state.Neg || d.coef.IsZero() {
		return Dec128{state: state.DomainError}
	}

	w := int(scale) + transGuard
	return fixedToDec(lnFixed(fixedFromDec(d, w), w), w, 0, scale, mode)
}

// Ln1p returns the natural logarithm of 1+d at exactly the given scale, rounding with mode.
//
// It is the entry point every decimal and floating-point library provides for this shape, and it is here for the
// same reason - a daily rate, a basis point or a log return is written as one plus something small - but the usual
// justification for it does not apply to this type, and it is worth saying so rather than repeating the claim.
//
// That justification is cancellation: in binary floating point, and in any format with fewer digits than this one,
// forming 1+d rounds away the low digits of a small d before the logarithm ever sees them. A Dec128 carries 39
// significant digits and at most 19 of them after the point, so 1+d needs 20 digits at worst and is exact. Ln1p and
// Ln(One.Add(d)) therefore agree here for every d where the sum does not overflow, which TestLn1pMatchesComposition
// holds them to.
//
// What Ln1p does give is a total function and one less thing to get right at the top of the range, where 1+d can
// exceed a coefficient and be rounded: this forms 1+d at the working scale, where it is always exact. Prefer it for
// that, and for saying what is meant; do not reach for it expecting digits back that Ln would have lost.
//
// Ln1p(0) is exactly zero. A d of -1 or less is NaN(DomainError); otherwise it fails and is rounded as Ln is.
//
// It allocates, and it reads no process-global configuration.
func (d Dec128) Ln1p(scale uint8, mode RoundingMode) Dec128 {
	if bad, ok := transArgs(d, scale, mode); !ok {
		return bad
	}

	w := int(scale) + transGuard
	x := fixedOne(w)
	if d.state == state.Neg {
		x.Sub(x, fixedFromDec(d, w))
	} else {
		x.Add(x, fixedFromDec(d, w))
	}
	if x.Sign() <= 0 {
		return Dec128{state: state.DomainError}
	}

	return fixedToDec(lnFixed(x, w), w, 0, scale, mode)
}

// pow5Uint64 holds 5^0..5^19, which is what tells a power of two from a decimal that merely looks like one: a value
// is 2^k exactly when its coefficient divided by 5^scale is a power of two, because 2^-n is 5^n over 10^n.
var pow5Uint64 = func() (t [MaxScale + 1]uint64) {
	t[0] = 1
	for i := 1; i <= int(MaxScale); i++ {
		t[i] = t[i-1] * 5
	}
	return
}()

// pow10Exponent reports whether d is exactly an integral power of ten and, if so, which one. A Dec128 is 10^k exactly
// when its coefficient is itself a power of ten, whatever scale it is written at: 100, 1.00 and 0.001 all qualify.
func (d Dec128) pow10Exponent() (int, bool) {
	j := d.coef.Len10() - 1
	if j < 0 || j >= len(Pow10Uint128) || !d.coef.Equal(Pow10Uint128[j]) {
		return 0, false
	}
	return j - int(d.scale), true
}

// pow2Exponent reports whether d is exactly an integral power of two and, if so, which one.
func (d Dec128) pow2Exponent() (int, bool) {
	c := d.coef
	if d.scale > 0 {
		q, r, st := c.QuoRem64(pow5Uint64[d.scale])
		if st >= state.Error || r != 0 {
			return 0, false
		}
		c = q
	}
	if c.IsZero() || c.NonZeroBitsCount() != 1 {
		return 0, false
	}
	return c.BitLen() - 1 - int(d.scale), true
}

// logBase is the shared body of Log10 and Log2: the natural logarithm at the working scale, divided by the stored
// natural logarithm of the base. One division is added to Ln's error, which transGuard absorbs along with the rest.
func (d Dec128) logBase(lnBase func(int) *big.Int, scale uint8, mode RoundingMode) Dec128 {
	w := int(scale) + transGuard
	v := lnFixed(fixedFromDec(d, w), w)

	neg := v.Sign() < 0
	v.Abs(v)
	v.Mul(v, pow10big(w))
	v.Quo(v, lnBase(w))
	if neg {
		v.Neg(v)
	}

	return fixedToDec(v, w, 0, scale, mode)
}

// Log10 returns the base-ten logarithm of d at exactly the given scale, rounding with mode.
//
// An exact power of ten gives an exact integer - Log10(1000) is 3, Log10(0.001) is -3, and 1.00 counts as 10^0 - which
// the General Decimal Arithmetic specification requires and which a series cannot be relied on to produce: a value
// converging on 3 from below, truncated by a directed rounding mode, would give 2.999... instead. Those are detected
// and answered exactly rather than computed. Everything else is Ln divided by the stored ln(10), and is faithfully
// rounded on the same terms as Ln.
//
// It is the operation to reach for when what is wanted is an order of magnitude with a fraction. When the integer part
// alone will do, Logb gives it exactly and for nothing.
//
// A d of zero or less is NaN(DomainError). It otherwise fails and is rounded as Ln does. NaN propagates.
//
// It allocates unless the argument is a power of ten, and it reads no process-global configuration.
func (d Dec128) Log10(scale uint8, mode RoundingMode) Dec128 {
	if bad, ok := transArgs(d, scale, mode); !ok {
		return bad
	}
	if d.state == state.Neg || d.coef.IsZero() {
		return Dec128{state: state.DomainError}
	}
	if k, ok := d.pow10Exponent(); ok {
		return FromInt64(int64(k)).RescaleRound(scale, mode)
	}

	return d.logBase(lnTen, scale, mode)
}

// Log2 returns the base-two logarithm of d at exactly the given scale, rounding with mode.
//
// An exact power of two gives an exact integer, on the same reasoning as Log10 and by the same mechanism: every power
// of two is a terminating decimal, because 2^-n is 5^n over 10^n, so Log2(0.25) is -2 exactly and not -1.999... under a
// mode that rounds toward zero. The specification says nothing about this - it has no log2 - but a Log2 that disagreed
// with Log10 about whether an exact answer is worth detecting would be the harder thing to explain.
//
// Everything else is Ln divided by the stored ln(2), faithfully rounded as Ln is. It fails as Log10 does, and NaN
// propagates.
//
// It allocates unless the argument is a power of two, and it reads no process-global configuration.
func (d Dec128) Log2(scale uint8, mode RoundingMode) Dec128 {
	if bad, ok := transArgs(d, scale, mode); !ok {
		return bad
	}
	if d.state == state.Neg || d.coef.IsZero() {
		return Dec128{state: state.DomainError}
	}
	if k, ok := d.pow2Exponent(); ok {
		return FromInt64(int64(k)).RescaleRound(scale, mode)
	}

	return d.logBase(lnTwo, scale, mode)
}

// Exp returns e raised to the power d at exactly the given scale, rounding with mode.
//
// It is faithfully rounded on the same terms as Ln and measured the same way: over the golden corpus the largest
// error observed was zero units in the last place. Exp(0) is exactly one.
//
// A result too large for a Dec128 is NaN(Overflow) - which begins a little above 88.72 - and one too small for the
// requested scale rounds to zero, or to one unit in the last place under a mode that rounds away from it. The
// argument validation is that of Ln. NaN propagates. Use Expm1 when what is wanted is exp(d)-1 for a small d; see
// that method for why.
//
// It allocates, and it reads no process-global configuration.
func (d Dec128) Exp(scale uint8, mode RoundingMode) Dec128 {
	if bad, ok := transArgs(d, scale, mode); !ok {
		return bad
	}
	if d.coef.IsZero() {
		return Dec128{coef: Pow10Uint128[scale], scale: scale} // exp(0) is exactly one
	}

	w := int(scale) + transGuard
	v := fixedFromDec(d, w)
	if d.state == state.Neg {
		v.Neg(v)
	}

	sum, k, ok := expFixed(v, w)
	if !ok {
		if k > 0 {
			return Dec128{state: state.Overflow}
		}
		// smaller than anything the requested scale can hold, and the sign is positive, so only a mode that rounds
		// away from zero reaches the first unit
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		if roundDecision(false, false, false, state.Default, mode) {
			return Dec128{coef: uint128.One, scale: scale}
		}
		return Dec128{scale: scale}
	}

	return fixedToDec(sum, w, k, scale, mode)
}

// Expm1 returns e raised to the power d, minus one, at exactly the given scale, rounding with mode.
//
// It is to Exp what Ln1p is to Ln, and the same caveat applies: the cancellation this function exists to avoid in a
// binary format does not arise here, because exp(d) for a small d is one plus something that a 39-digit coefficient
// holds exactly, so Exp(d).Sub(One) keeps the digits too. TestExpm1MatchesComposition holds the two together.
//
// It is still the right thing to write, and it subtracts at the working scale rather than after rounding, which is
// what keeps it exact at the top of the range where exp(d) itself is wide. It is the conversion a
// continuously-compounded rate needs in the direction Ln1p does not cover.
//
// Expm1(0) is exactly zero. It fails as Exp does.
//
// It allocates, and it reads no process-global configuration.
func (d Dec128) Expm1(scale uint8, mode RoundingMode) Dec128 {
	if bad, ok := transArgs(d, scale, mode); !ok {
		return bad
	}
	if d.coef.IsZero() {
		return Dec128{scale: scale} // exp(0)-1 is exactly zero
	}

	w := int(scale) + transGuard
	v := fixedFromDec(d, w)
	if d.state == state.Neg {
		v.Neg(v)
	}

	sum, k, ok := expFixed(v, w)
	if !ok {
		if k > 0 {
			return Dec128{state: state.Overflow}
		}
		// exp(d) is indistinguishable from zero here, so exp(d)-1 is -1 at the requested scale
		return Dec128{coef: Pow10Uint128[scale], scale: scale, state: state.Neg}
	}

	// exp(d) - 1 at the working scale, with the shift applied first so that the one is subtracted from the value and
	// not from its mantissa
	if k >= 0 {
		sum.Mul(sum, pow10big(k))
	} else {
		sum.Quo(sum, pow10big(-k))
	}
	sum.Sub(sum, fixedOne(w))

	return fixedToDec(sum, w, 0, scale, mode)
}

// rationalExponent expresses e as a reduced fraction p/q, reporting whether both halves are small enough for
// PowRational to take. A decimal is always a rational - e at scale s is its coefficient over 10^s - so this is only
// ever a question of size, and the reduction is what makes it worth asking: 0.5 is 1/2 and 0.0625 is 1/16 whatever
// scale they are written at.
func rationalExponent(e Dec128) (p, q int64, ok bool) {
	if e.coef.Hi != 0 {
		return 0, 0, false // a coefficient past 64 bits is past maxRootDegree whatever it reduces to
	}

	c, den := e.coef.Lo, Pow10Uint64[e.scale]
	g := gcdUint64(c, den)
	c, den = c/g, den/g
	if c > maxRootDegree || den > maxRootDegree {
		return 0, 0, false
	}

	p = int64(c)
	if e.state == state.Neg {
		p = -p
	}

	return p, int64(den), true
}

// hugeIntegerPow handles an integer exponent too large for an int64. No such power is representable unless the base
// is one in magnitude, which the caller has already dealt with, so the only question is which end of the range it
// leaves by: away from zero when the magnitudes push the same way, toward it when they oppose.
func hugeIntegerPow(d Dec128, negExp bool, scale uint8, mode RoundingMode) Dec128 {
	if (d.Abs().Compare(One) > 0) != negExp {
		return Dec128{state: state.Overflow}
	}
	if mode == ROUND_NAN {
		return Dec128{state: state.Inexact}
	}
	// smaller than the quantum of any scale; only a mode that rounds away from zero reaches the first unit, and the
	// sign of a power of a negative base at an exponent this size is not knowable, so it is taken as positive
	if roundDecision(false, false, false, state.Default, mode) {
		return Dec128{coef: uint128.One, scale: scale}
	}

	return Dec128{scale: scale}
}

// Pow returns d raised to the power e at exactly the given scale, rounding with mode. It is the general power, and it
// dispatches to the cheapest form that can answer exactly the question asked:
//
//   - an integer e goes to PowIntRound, which is fixed-width and allocates nothing, and is faithfully rounded;
//   - an e that reduces to a fraction both of whose halves are at most maxRootDegree goes to PowRational, which is
//     correctly rounded, so that d.Pow(Half, ...) is bit-identical to d.SqrtRound(...);
//   - anything else is exp(e * ln d), which is faithfully rounded as Exp and Ln are.
//
// Which of the three answered is not observable except in the guarantee and the cost, and the boundary is a property
// of the exponent rather than of the value: 0.5 and 0.0625 are short fractions at any scale they are written at,
// while 0.3333333333 is not.
//
// d^0 is one for every d. A negative d has a power only for an integer e, and is NaN(DomainError) otherwise, because
// the value is not a real number. A zero d is zero for a positive e and NaN(DivisionByZero) for a negative one. The
// argument validation and the remaining failures are those of Exp. NaN propagates: d first, then e.
//
// It allocates unless the dispatch reaches PowIntRound, and it reads no process-global configuration.
func (d Dec128) Pow(e Dec128, scale uint8, mode RoundingMode) Dec128 {
	switch {
	case d.state >= state.Error:
		return d
	case e.state >= state.Error:
		return e
	case scale > MaxScale:
		return Dec128{state: state.ScaleOutOfRange}
	case !mode.IsValid():
		return Dec128{state: state.InvalidRoundingMode}
	case e.coef.IsZero():
		// one at exactly the requested scale, for every d, as PowIntRound and PowRational both have it
		return Dec128{coef: Pow10Uint128[scale], scale: scale}
	}

	if e.IsInteger() {
		if n, err := e.Int64(); err == nil {
			return d.PowIntRound(n, scale, mode)
		}
		if d.Abs().Equal(One) {
			// the only base whose power at this exponent is representable; the parity of an integer this large is
			// still knowable, because a value at scale 0 carries every one of its digits
			if d.state == state.Neg && e.coef.Lo&1 == 1 {
				return Dec128{coef: Pow10Uint128[scale], scale: scale, state: state.Neg}
			}
			return Dec128{coef: Pow10Uint128[scale], scale: scale}
		}
		if d.coef.IsZero() {
			if e.state == state.Neg {
				return Dec128{state: state.DivisionByZero}
			}
			return Dec128{scale: scale}
		}
		return hugeIntegerPow(d, e.state == state.Neg, scale, mode)
	}

	switch {
	case d.coef.IsZero():
		if e.state == state.Neg {
			return Dec128{state: state.DivisionByZero}
		}
		return Dec128{scale: scale} // a zero is never negative
	case d.state == state.Neg:
		// a non-integer power of a negative base is not a real number
		return Dec128{state: state.DomainError}
	}

	if p, q, ok := rationalExponent(e); ok {
		return d.PowRational(p, q, scale, mode)
	}

	return d.powExpLn(e, scale, mode)
}

// powExpLn is the general power, exp(e * ln d), for a positive d and a nonzero e. It is the arm of Pow that the
// dispatch reaches last and the only one that is not exact or fixed-width, and it is separate so that
// TestPowExpLnAgainstPowRational can put it against the correctly rounded PowRational on the very exponents the
// dispatch would otherwise route away from it.
func (d Dec128) powExpLn(e Dec128, scale uint8, mode RoundingMode) Dec128 {
	// The logarithm is taken with enough extra places to absorb the multiplication: an absolute error in ln d is
	// multiplied by |e|, so the working scale grows by the integer digits of the exponent.
	w := int(scale) + transGuard
	wLn := w + max(e.IntegerDigits(), 0) + 2

	v := lnFixed(fixedFromDec(d, wLn), wLn)
	v.Mul(v, new(big.Int).SetBytes(bigBytes(e.coef)))
	v.Quo(v, pow10big(int(e.scale)))
	v.Quo(v, pow10big(wLn-w))
	if e.state == state.Neg {
		v.Neg(v)
	}

	sum, k, ok := expFixed(v, w)
	if !ok {
		if k > 0 {
			return Dec128{state: state.Overflow}
		}
		if mode == ROUND_NAN {
			return Dec128{state: state.Inexact}
		}
		if roundDecision(false, false, false, state.Default, mode) {
			return Dec128{coef: uint128.One, scale: scale}
		}
		return Dec128{scale: scale}
	}

	return fixedToDec(sum, w, k, scale, mode)
}
