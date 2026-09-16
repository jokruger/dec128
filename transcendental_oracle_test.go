package dec128

import (
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/uint128"
)

// A randomized oracle for Exp and Ln, to sit beside the golden corpus.
//
// testdata/transcendental_golden.csv is an independent reference but it is a fixed list, and a fixed list can only
// find the mistakes someone thought to put an argument in for. What it cannot find is a blind spot shared between the
// corpus and the implementation: an argument class nobody wrote down, in which the series happens to be wrong. So
// these tests compute the same two functions a second time, over random operands, by a construction that has nothing
// in common with the one under test.
//
// The reference is binary floating point at 512 bits - big.Float, not fixed point over big.Int - so none of
// transcendental.go's machinery is reachable from it. refExp is the plain Taylor series with no argument reduction at
// all, which is the definition of the function written out; the implementation reduces by k*ln(10) and then halves ten
// times, so the two share no constant, no reduction and no arithmetic. refLn does not use a series of its own: it
// inverts refExp by Newton's method, which means the logarithm oracle is the exponential oracle and an error would
// have to be in both, in opposite directions, to cancel.
//
// 512 bits is about 154 decimal digits. The worst case for the reference is exp of a large negative argument, where
// the direct series cancels catastrophically; refExp computes the reciprocal of exp(|x|) instead, so no case loses
// more than the few digits the peak term costs, and 154 digits leaves well over a hundred for a result the test then
// compares at 19.

const refPrec = 512

func refFloat(v float64) *big.Float {
	return new(big.Float).SetPrec(refPrec).SetFloat64(v)
}

// refExp returns e**x by the defining series, sum of x^i/i!, with no argument reduction. A negative x is taken as the
// reciprocal of the positive case, which is what keeps the cancellation out of it.
func refExp(x *big.Float) *big.Float {
	one := refFloat(1)
	ax := new(big.Float).SetPrec(refPrec).Abs(x)

	sum := new(big.Float).SetPrec(refPrec).Set(one)
	term := new(big.Float).SetPrec(refPrec).Set(one)
	den := new(big.Float).SetPrec(refPrec)
	for i := int64(1); i < 20000; i++ {
		term.Mul(term, ax)
		term.Quo(term, den.SetInt64(i))
		sum.Add(sum, term)
		// Stop when the term can no longer move the sum at this precision.
		if term.Sign() == 0 || term.MantExp(nil) < sum.MantExp(nil)-int(refPrec)-8 {
			break
		}
	}

	if x.Sign() < 0 {
		return new(big.Float).SetPrec(refPrec).Quo(one, sum)
	}

	return sum
}

// refLn returns the natural logarithm of a positive x by Newton's method on refExp: y <- y + x/exp(y) - 1, which
// doubles the correct digits each step. The seed is float64's own logarithm, so five steps take sixteen digits past
// the precision the reference keeps.
func refLn(x *big.Float) *big.Float {
	f, _ := x.Float64()
	y := refFloat(math.Log(f))
	one := refFloat(1)
	t := new(big.Float).SetPrec(refPrec)

	for range 6 {
		t.Quo(x, refExp(y))
		t.Sub(t, one)
		y.Add(y, t)
	}

	return y
}

// refRat turns a reference value into the exact rational the rounding helpers take. The big.Float is not exact, but it
// carries a hundred digits more than the comparison needs, which is the same footing the golden corpus stands on.
func refRat(f *big.Float) *big.Rat {
	r, _ := f.Rat(nil)
	return r
}

// decToFloat is the operand conversion, exact because a Dec128 is a rational and 512 bits hold any of them.
func decToFloat(d Dec128) *big.Float {
	r, err := d.Rat()
	if err != nil {
		panic(err)
	}
	return new(big.Float).SetPrec(refPrec).SetRat(r)
}

// ulpAgainstRef compares one result with the correctly rounded reference and returns the error in units of the last
// place.
func ulpAgainstRef(t *testing.T, got Dec128, exact *big.Rat, scale uint8, mode RoundingMode, what string) int64 {
	t.Helper()

	want, _ := roundRat(exact, scale, mode)
	if exact.Sign() < 0 {
		want = new(big.Int).Neg(want)
	}

	if new(big.Int).Abs(want).Cmp(big2p128) >= 0 {
		if !got.IsNaN() {
			t.Errorf("%s at scale %d under %v: want NaN(Overflow), got %s", what, scale, mode, got.StringFixed())
		}
		return 0
	}
	if got.IsNaN() {
		t.Errorf("%s at scale %d under %v: got NaN(%v), want %s", what, scale, mode, got.ErrorDetails(), want)
		return 0
	}

	diff := new(big.Int).Sub(bigAtScale(got, scale), want)

	return diff.Abs(diff).Int64()
}

// randExpArg returns an argument for Exp spread over the range where the result is representable, which runs from
// about -43.6 (the smallest nonzero result at MaxScale) to about 88.7 (the largest that fits at scale 0).
func randExpArg(r *rand.Rand) Dec128 {
	switch r.Intn(4) {
	case 0:
		// tiny, where the series is nearly the argument itself
		return Dec128{coef: uint128.FromUint64(uint64(1 + r.Int63n(1_000_000))), scale: MaxScale}
	case 1:
		// the money-rate range, which is what the function is actually used on
		return DecodeFromUint64(uint64(r.Int63n(500_000)), 6).Neg()
	case 2:
		// anywhere in range, with the sign chosen separately
		d := DecodeFromUint64(uint64(r.Int63n(880_000_000)), 7)
		if r.Intn(2) == 0 {
			return d.Neg()
		}
		return d
	default:
		// close to the ends, where the reduction has the most work to do
		d := DecodeFromUint64(uint64(430_000_000+r.Int63n(10_000_000)), 7)
		if r.Intn(2) == 0 {
			return d.Neg()
		}
		return d
	}
}

// randLnArg returns a positive argument for Ln, weighted toward one - where the reduction cancels - and toward the
// ends of the range, where it does the most.
func randLnArg(r *rand.Rand) Dec128 {
	switch r.Intn(4) {
	case 0:
		// just either side of one, where ln(m) and ln(10) very nearly cancel
		d := One.Rescale(MaxScale)
		off := uint64(r.Int63n(1_000_000))
		if r.Intn(2) == 0 {
			d.coef, _ = d.coef.Add64(off)
		} else {
			d.coef, _ = d.coef.Sub64(off)
		}
		return d
	case 1:
		// a rate factor
		return DecodeFromUint64(uint64(1_000_000_000+r.Int63n(200_000_000)), 9)
	case 2:
		// somewhere in the middle of the range
		return DecodeFromUint64(uint64(1+r.Int63n(1_000_000_000_000_000)), uint8(r.Intn(int(MaxScale)+1)))
	default:
		// the ends
		if r.Intn(2) == 0 {
			return Dec128{coef: uint128.FromUint64(uint64(1 + r.Int63n(1000))), scale: MaxScale}
		}
		return DecodeFromUint64(uint64(r.Int63n(1_000_000_000_000_000_000)), 0)
	}
}

// TestExpAgainstIndependentSeries measures Exp against the plain Taylor series in binary floating point, over random
// arguments spread across the representable range. The documented guarantee is faithful rounding; this is the
// randomized half of the evidence for it, the golden corpus being the fixed half.
func TestExpAgainstIndependentSeries(t *testing.T) {
	defer SetDefaultScale(DefaultScale())
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	r := rand.New(rand.NewSource(20260916))
	cases := 400
	if testing.Short() {
		cases = 20
	}

	var worst int64
	var worstCase string
	compared := 0

	for range cases {
		d := randExpArg(r)
		exact := refRat(refExp(decToFloat(d)))
		for _, scale := range []uint8{0, 2, 6, 10, MaxScale} {
			for _, mode := range allModes {
				if mode == ROUND_NAN {
					continue // refuses every inexact result by design
				}
				what := "Exp(" + d.StringFixed() + ")"
				ulp := ulpAgainstRef(t, d.Exp(scale, mode), exact, scale, mode, what)
				compared++
				if ulp > worst {
					worst, worstCase = ulp, what+" at scale "+FromInt(int(scale)).String()+" under "+mode.String()
				}
				if ulp > 1 {
					t.Errorf("%s at scale %d under %v: off by %d units in the last place", what, scale, mode, ulp)
				}
			}
		}
	}

	if compared == 0 {
		t.Fatal("no results were compared")
	}
	t.Logf("compared %d results; largest error %d ulp%s", compared, worst, worstSuffix(worst, worstCase))
}

// TestLnAgainstIndependentSeries is the same measurement for Ln, against a reference that inverts the exponential
// rather than taking a logarithm of its own.
func TestLnAgainstIndependentSeries(t *testing.T) {
	defer SetDefaultScale(DefaultScale())
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	r := rand.New(rand.NewSource(20260917))
	cases := 400
	if testing.Short() {
		cases = 20
	}

	var worst int64
	var worstCase string
	compared := 0

	for range cases {
		d := randLnArg(r)
		if d.IsZero() || d.IsNegative() {
			continue
		}
		exact := refRat(refLn(decToFloat(d)))
		for _, scale := range []uint8{0, 2, 6, 10, MaxScale} {
			for _, mode := range allModes {
				if mode == ROUND_NAN {
					continue
				}
				what := "Ln(" + d.StringFixed() + ")"
				ulp := ulpAgainstRef(t, d.Ln(scale, mode), exact, scale, mode, what)
				compared++
				if ulp > worst {
					worst, worstCase = ulp, what+" at scale "+FromInt(int(scale)).String()+" under "+mode.String()
				}
				if ulp > 1 {
					t.Errorf("%s at scale %d under %v: off by %d units in the last place", what, scale, mode, ulp)
				}
			}
		}
	}

	if compared == 0 {
		t.Fatal("no results were compared")
	}
	t.Logf("compared %d results; largest error %d ulp%s", compared, worst, worstSuffix(worst, worstCase))
}

// TestLnTwoConstant holds the stored ln(2) to the same standard TestLnTenConstant holds ln(10) to, and then to a
// second one: the constant is checked against what the package's own lnFixed computes for 2 at a working scale far
// above where the constant is used. A mistyped digit cannot pass both, because the two come from different places -
// the first from Python's decimal module by way of the golden corpus, the second from the series in this package.
func TestLnTwoConstant(t *testing.T) {
	stored, ok := new(big.Int).SetString(lnTwoCoef, 10)
	if !ok {
		t.Fatal("lnTwoCoef does not parse")
	}
	if got := len(lnTwoCoef); got != lnTenScale {
		t.Fatalf("lnTwoCoef has %d digits, want %d", got, lnTenScale)
	}

	// Against the package's own series, computed well above the scale the constant is stored at.
	const w = lnTenScale + 20
	own := lnFixed(new(big.Int).Mul(big.NewInt(2), pow10big(w)), w)
	own.Quo(own, pow10big(w-lnTenScale))
	if diff := new(big.Int).Sub(own, stored); diff.CmpAbs(big.NewInt(2)) > 0 {
		t.Errorf("lnTwoCoef differs from lnFixed(2) by %s units at scale %d", diff, lnTenScale)
	}

	// And against the golden corpus, which comes from outside this package entirely. The corpus carries 80
	// significant digits, so the comparison is made there and not at the constant's own scale.
	const goldenScale = 79
	for _, row := range loadTranscendentalGolden(t) {
		if row.fn != "Ln" || !row.arg1.Equal(FromInt64(2)) {
			continue
		}
		want := new(big.Int).Quo(new(big.Int).Mul(row.exact.Num(), pow10big(goldenScale)), row.exact.Denom())
		at := new(big.Int).Quo(stored, pow10big(lnTenScale-goldenScale))
		if diff := new(big.Int).Sub(want, at); diff.CmpAbs(big.NewInt(1)) > 0 {
			t.Errorf("lnTwoCoef differs from the golden ln(2) by %s units at scale %d", diff, goldenScale)
		}
		return
	}
	t.Fatal("the golden corpus has no Ln(2) row to check the constant against")
}
