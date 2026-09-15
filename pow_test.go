package dec128

import (
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// PowIntRound is checked against an exact rational oracle: (num/den)^n is computable exactly in math/big, so the
// correctly rounded result at any scale is a number rather than an estimate, and the distance between it and what the
// implementation returns is the accuracy the doc comment is allowed to claim.

// ratPow returns the exact value of d^n as a rational, or nil for 0^negative.
func ratPow(d Dec128, n int64) *big.Rat {
	v := new(big.Rat).SetFrac(bigAtScale(d, d.scale), bigPow10(d.scale))
	if v.Sign() == 0 && n < 0 {
		return nil
	}
	m := uint64(n)
	if n < 0 {
		m = -m
	}
	r := new(big.Rat).SetInt64(1)
	base := new(big.Rat).Set(v)
	for m > 0 {
		if m&1 == 1 {
			r.Mul(r, base)
		}
		base.Mul(base, base)
		m >>= 1
	}
	if n < 0 {
		r.Inv(r)
	}
	return r
}

// roundRat returns the correctly rounded coefficient of |v| at the given scale under mode, and whether the value was
// exact there.
func roundRat(v *big.Rat, scale uint8, mode RoundingMode) (*big.Int, bool) {
	num := new(big.Int).Mul(new(big.Int).Abs(v.Num()), bigPow10(scale))
	q, rem := new(big.Int).QuoRem(num, v.Denom(), new(big.Int))
	return roundBig(q, rem, v.Denom(), v.Sign() < 0, mode), rem.Sign() == 0
}

// randPowBase returns a base of the shape compounding actually uses: a rate factor near one, plus the occasional
// arbitrary value so that the wide and the overflowing cases appear too.
func randPowBase(r *rand.Rand) Dec128 {
	switch r.Intn(4) {
	case 0:
		return randDec(r)
	case 1:
		// 1 + a rate between 0 and 10%, at a random number of places
		scale := uint8(2 + r.Intn(int(MaxScale)-1))
		d := FromInt64(1).Rescale(scale)
		d.coef, _ = d.coef.Add64(uint64(r.Int63n(int64(Pow10Uint64[scale] / 10))))
		return d
	case 2:
		// a small value, where a long power underflows
		return Dec128{coef: uint128.FromUint64(uint64(1 + r.Intn(1000))), scale: uint8(1 + r.Intn(4))}
	default:
		// a small integer or a value just above one
		d := Dec128{coef: uint128.FromUint64(uint64(1 + r.Intn(20))), scale: uint8(r.Intn(3))}
		if r.Intn(2) == 0 {
			d.state = state.Neg
		}
		return d
	}
}

func TestPowIntRoundAgainstBigRat(t *testing.T) {
	r := rand.New(rand.NewSource(20260925))
	worstUlp := new(big.Int)
	checked := 0

	for range 60000 {
		d := randPowBase(r)
		n := int64(r.Intn(400)) - 100
		if n == 0 {
			n = 1
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]

		got := d.PowIntRound(n, scale, mode)
		exact := ratPow(d, n)
		if exact == nil {
			if got.state != state.DivisionByZero {
				t.Fatalf("0^%d = %v, want NaN(DivisionByZero)", n, got)
			}
			continue
		}
		want, isExact := roundRat(exact, scale, mode)

		if want.Cmp(big2p128) >= 0 {
			if !got.IsNaN() {
				t.Fatalf("PowIntRound(%s, %d, %d, %s) = %s, want NaN", d.StringFixed(), n, scale, mode, got.StringFixed())
			}
			continue
		}
		if mode == ROUND_NAN {
			// the implementation reports every rounding it took, including the intermediate ones, so a NaN here is
			// only required to be honest: an exact power must not be reported as inexact
			if got.IsNaN() && isExact && got.state == state.Inexact && n > 0 {
				t.Fatalf("PowIntRound(%s, %d, %d, ROUND_NAN) = NaN(Inexact) for an exact power", d.StringFixed(), n, scale)
			}
			continue
		}
		if got.IsNaN() {
			// an intermediate that does not fit is reported as an overflow even when the final value would have fit;
			// only a result the oracle says is representable and reachable is a failure
			t.Fatalf("PowIntRound(%s, %d, %d, %s) = NaN(%v), want %s", d.StringFixed(), n, scale, mode, got.ErrorDetails(), want)
		}
		if got.scale != scale {
			t.Fatalf("PowIntRound(%s, %d, %d, %s): scale %d", d.StringFixed(), n, scale, mode, got.scale)
		}

		diff := new(big.Int).Sub(new(big.Int).Abs(bigAtScale(got, scale)), want)
		diff.Abs(diff)
		if diff.Cmp(big.NewInt(1)) > 0 {
			t.Fatalf("PowIntRound(%s, %d, %d, %s) = %s, want %s: off by %s ulp",
				d.StringFixed(), n, scale, mode, got.StringFixed(), want, diff)
		}
		if diff.Cmp(worstUlp) > 0 {
			worstUlp.Set(diff)
			t.Logf("PowIntRound(%s, %d, %d, %s) = %s, correctly rounded is %s",
				d.StringFixed(), n, scale, mode, got.StringFixed(), want)
		}
		checked++
	}

	t.Logf("checked %d powers, largest error %s ulp", checked, worstUlp)
	if worstUlp.Sign() != 0 {
		t.Errorf("the doc comment claims a largest observed error of zero ulp; this run saw %s", worstUlp)
	}
}

// PowIntRound must beat the unguarded PowInt64 on the case the guard digits exist for: a long compounding chain under
// the default truncating mode, where every intermediate rounding pushes the same way.
func TestPowIntRoundBeatsPowInt64OnCompounding(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetDefaultScale(DefaultScale())
	SetArithmeticRounding(ROUND_TOWARD_ZERO)
	SetDefaultScale(MaxScale)

	base := FromString("1.000164383561643836") // a 6% annual rate, daily rest
	const n = 3650                             // ten years

	exact := ratPow(base, n)
	want, _ := roundRat(exact, MaxScale, ROUND_HALF_AWAY_FROM_ZERO)

	guarded := base.PowIntRound(n, MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	unguarded := base.PowInt64(n).RescaleRound(MaxScale, ROUND_HALF_AWAY_FROM_ZERO)

	gErr := new(big.Int).Abs(new(big.Int).Sub(new(big.Int).Abs(bigAtScale(guarded, MaxScale)), want))
	uErr := new(big.Int).Abs(new(big.Int).Sub(new(big.Int).Abs(bigAtScale(unguarded, MaxScale)), want))
	t.Logf("exact %s, guarded %s (%s ulp), unguarded %s (%s ulp)",
		new(big.Rat).SetFrac(want, bigPow10(MaxScale)).FloatString(int(MaxScale)), guarded.StringFixed(), gErr, unguarded.StringFixed(), uErr)

	if gErr.Sign() != 0 {
		t.Errorf("PowIntRound is off by %s ulp at ten years of daily rest", gErr)
	}
	if uErr.Cmp(gErr) < 0 {
		t.Errorf("PowInt64 (%s ulp) beat PowIntRound (%s ulp); the guard digits are not doing their job", uErr, gErr)
	}
}

func TestPowIntRoundEdges(t *testing.T) {
	two := FromInt64(2)

	for _, c := range []struct {
		d     string
		n     int64
		scale uint8
		mode  RoundingMode
		want  string
	}{
		{"1.05", 3, 6, ROUND_BANK, "1.157625"},   // exact
		{"1.05", 3, 4, ROUND_BANK, "1.1576"},     // rounded once
		{"1.05", 3, 8, ROUND_BANK, "1.15762500"}, // padded
		{"2", 10, 0, ROUND_BANK, "1024"},
		{"2", -10, 10, ROUND_BANK, "0.0009765625"}, // exact reciprocal
		{"-2", 3, 0, ROUND_BANK, "-8"},             // odd power keeps the sign
		{"-2", 4, 0, ROUND_BANK, "16"},             // even power loses it
		{"-2", -3, 4, ROUND_BANK, "-0.1250"},
		{"0.5", 100, MaxScale, ROUND_HALF_AWAY_FROM_ZERO, "0.0000000000000000000"}, // underflows to zero
		{"3", 1, 2, ROUND_BANK, "3.00"},
		{"0", 5, 2, ROUND_BANK, "0.00"},
	} {
		if got := FromString(c.d).PowIntRound(c.n, c.scale, c.mode); got.StringFixed() != c.want {
			t.Errorf("PowIntRound(%s, %d, %d, %s) = %s, want %s", c.d, c.n, c.scale, c.mode, got.StringFixed(), c.want)
		}
	}

	// the reciprocal of a power that underflowed at the working scale is still exact where it is representable:
	// 0.5^-100 is 2^100, which fits a coefficient even though 0.5^100 does not fit a scale
	if got := FromString("0.5").PowIntRound(-100, 0, ROUND_BANK); got.StringFixed() != "1267650600228229401496703205376" {
		t.Errorf("0.5^-100 = %s", got.StringFixed())
	}

	// anything to the power zero is one, at exactly the requested scale
	for _, d := range []Dec128{Zero, One, MaxAtScale(0), FromString("-7.5")} {
		if got := d.PowIntRound(0, 3, ROUND_BANK); got.StringFixed() != "1.000" {
			t.Errorf("%s^0 = %s", d.StringFixed(), got.StringFixed())
		}
	}

	// the exponent extremes
	// 2^MinInt64 is 2^-(2^63), a positive value far below any quantum
	if got := two.PowIntRound(math.MinInt64, MaxScale, ROUND_BANK); !got.IsZero() {
		t.Errorf("2^MinInt64 = %v", got)
	}
	if got := two.PowIntRound(math.MinInt64, MaxScale, ROUND_UP); got.StringFixed() != "0.0000000000000000001" {
		t.Errorf("2^MinInt64 rounded up = %s, want one quantum", got.StringFixed())
	}
	if got := two.PowIntRound(math.MaxInt64, 0, ROUND_BANK); got.state != state.Overflow {
		t.Errorf("2^MaxInt64 = %v", got)
	}
	if got := One.PowIntRound(math.MinInt64, 2, ROUND_BANK); got.StringFixed() != "1.00" {
		t.Errorf("1^MinInt64 = %s", got.StringFixed())
	}
}

func TestPowIntRoundArgumentErrors(t *testing.T) {
	d := FromString("1.5")
	for _, c := range []struct {
		what string
		got  Dec128
		want state.State
	}{
		{"scale", d.PowIntRound(2, MaxScale+1, ROUND_BANK), state.ScaleOutOfRange},
		{"mode", d.PowIntRound(2, 2, RoundingMode(99)), state.InvalidRoundingMode},
		{"NaN", NaN(state.NotConverged).PowIntRound(2, 2, ROUND_BANK), state.NotConverged},
		{"NaN at zero exponent", NaN(state.NotConverged).PowIntRound(0, 2, ROUND_BANK), state.NotConverged},
		{"zero to a negative power", Zero.PowIntRound(-1, 2, ROUND_BANK), state.DivisionByZero},
		{"overflow", MaxAtScale(0).PowIntRound(2, 0, ROUND_BANK), state.Overflow},
		{"reciprocal overflow", FromString("0.0000000001").PowIntRound(-50, 0, ROUND_BANK), state.Overflow},
	} {
		if c.got.state != c.want {
			t.Errorf("%s = %v, want NaN(%s)", c.what, c.got, c.want)
		}
	}

	// ROUND_NAN reports an inexact power rather than rounding it
	if got := FromString("1.0000000001").PowIntRound(30, MaxScale, ROUND_NAN); got.state != state.Inexact {
		t.Errorf("inexact power under ROUND_NAN = %v", got)
	}
	if got := FromString("1.05").PowIntRound(3, 6, ROUND_NAN); got.StringFixed() != "1.157625" {
		t.Errorf("exact power under ROUND_NAN = %v", got)
	}
	if got := FromString("1.05").PowIntRound(3, 2, ROUND_NAN); got.state != state.Inexact {
		t.Errorf("exact power at too few places under ROUND_NAN = %v", got)
	}
}

// reciprocalGuard is the entry point for every negative power, and its working scale is chosen from an estimate, so
// it is checked directly against an exact rational over the whole range of coefficients and scales.
func TestReciprocalGuard(t *testing.T) {
	r := rand.New(rand.NewSource(20260926))
	retries := 0
	for range 30000 {
		var c uint128.Uint128
		switch r.Intn(4) {
		case 0:
			c = uint128.FromUint64(uint64(1 + r.Intn(1000)))
		case 1:
			c = uint128.FromUint64(r.Uint64() | 1)
		case 2:
			c = uint128.Uint128{Lo: r.Uint64(), Hi: r.Uint64()}
		default:
			c = Pow10Uint128[r.Intn(39)]
			if c.IsZero() {
				c = uint128.One
			}
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))

		q, w, inexact := reciprocalGuard(c, scale)
		// the working scale is chosen without a search, so it must always be the estimate itself
		if want := min((190+c.BitLen())*1000/3322-int(scale), powGuard); int(w) != want {
			retries++
		}

		if !q.fits192() {
			t.Fatalf("reciprocalGuard(%s, %d) does not fit the working width", c.BigInt(), scale)
		}
		num, den := bigPow10(uint8(int(w)+int(scale))), c.BigInt()
		wantQ, rem := new(big.Int).QuoRem(num, den, new(big.Int))
		if new(big.Int).Lsh(rem, 1).Cmp(den) >= 0 {
			wantQ.Add(wantQ, big.NewInt(1)) // nearest, ties away from zero
		}
		if q.big().Cmp(wantQ) != 0 {
			t.Fatalf("reciprocalGuard(%s, %d) = %s at scale %d, want %s", c.BigInt(), scale, q.big(), w, wantQ)
		}
		if inexact != (rem.Sign() != 0) {
			t.Fatalf("reciprocalGuard(%s, %d): inexact %v, remainder %s", c.BigInt(), scale, inexact, rem)
		}
	}
	if retries != 0 {
		t.Errorf("the working scale differed from its estimate in %d of 30000 cases", retries)
	}
}

// PowInt64 shares the guarded core, so it must agree with PowIntRound wherever PowIntRound is asked for the scale the
// scale rule would have chosen anyway.
func TestPowInt64SharesTheGuardedCore(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy()) // SetArithmeticRounding writes the policy as well
	defer SetDefaultScale(DefaultScale())

	r := rand.New(rand.NewSource(20260927))
	for _, mode := range allModes {
		SetArithmeticRounding(mode)
		for range 20000 {
			d := randPowBase(r)
			n := int64(r.Intn(200)) - 50
			got := d.PowInt64(n)
			if got.IsNaN() {
				// A NaN carries no scale to ask the other operation about, but it does carry a claim: the power
				// could not be produced at the scale the rule would have chosen, and scale 0 is the coarsest
				// there is, so the per-call form must fail there too.
				if want := d.PowIntRound(n, 0, mode); !want.IsNaN() {
					t.Fatalf("PowInt64(%s, %d) = NaN(%v) but PowIntRound at scale 0 = %s",
						d.StringFixed(), n, got.ErrorDetails(), want.StringFixed())
				}
				continue
			}
			if want := d.PowIntRound(n, got.scale, mode); got != want {
				t.Fatalf("PowInt64(%s, %d) under %s = %s at scale %d, PowIntRound at the same scale = %s (%v)",
					d.StringFixed(), n, mode, got.StringFixed(), got.scale, want.StringFixed(), want.ErrorDetails())
			}
		}
	}
}

// A power that falls below every quantum the type has is still a nonzero value, and a directed mode has to move it to
// the first unit rather than report a zero. The two forms took that decision in different places until the scale rule
// learned to make it, so this is where they are held together.
func TestPowTinyResultUnderDirectedModes(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	tenth := FromString("0.1")
	quantum := QuantumAtScale(MaxScale)

	for _, c := range []struct {
		d    Dec128
		n    int64
		mode RoundingMode
		want string
	}{
		// 10^-60 is positive and far below 10^-19, so a ceiling is one quantum and a floor is zero
		{tenth, 60, ROUND_UP, "0.0000000000000000001"},
		{tenth, 60, ROUND_AWAY_FROM_ZERO, "0.0000000000000000001"},
		{tenth, 60, ROUND_DOWN, "0.0000000000000000000"},
		{tenth, 60, ROUND_TOWARD_ZERO, "0.0000000000000000000"},
		{tenth, 60, ROUND_HALF_AWAY_FROM_ZERO, "0.0000000000000000000"},
		{tenth, 60, ROUND_BANK, "0.0000000000000000000"},
		// and the same from below zero, where the directions swap
		{tenth.Neg(), 61, ROUND_DOWN, "-0.0000000000000000001"},
		{tenth.Neg(), 61, ROUND_AWAY_FROM_ZERO, "-0.0000000000000000001"},
		{tenth.Neg(), 61, ROUND_UP, "0.0000000000000000000"},
		{tenth.Neg(), 61, ROUND_TOWARD_ZERO, "0.0000000000000000000"},
		// an even power of a negative base is positive, so the ceiling moves and the floor does not
		{tenth.Neg(), 60, ROUND_UP, "0.0000000000000000001"},
		{tenth.Neg(), 60, ROUND_DOWN, "0.0000000000000000000"},
		// a quantum squared is the same case reached by a different route
		{quantum, 2, ROUND_UP, "0.0000000000000000001"},
		{quantum, 2, ROUND_TOWARD_ZERO, "0.0000000000000000000"},
	} {
		SetArithmeticRounding(c.mode)
		SetLossPolicy(LossRound)
		if got := c.d.PowInt64(c.n); got.StringFixed() != c.want {
			t.Errorf("PowInt64(%s, %d) under %s = %s (%v), want %s",
				c.d.StringFixed(), c.n, c.mode, got.StringFixed(), got.ErrorDetails(), c.want)
		}
		if got := c.d.PowIntRound(c.n, MaxScale, c.mode); got.StringFixed() != c.want {
			t.Errorf("PowIntRound(%s, %d, %d, %s) = %s (%v), want %s",
				c.d.StringFixed(), c.n, MaxScale, c.mode, got.StringFixed(), got.ErrorDetails(), c.want)
		}
	}

	// The loss policy rules on the rounded result, so a directed mode that reached the first unit did not underflow
	// and one that stayed at zero did - which is how a value that merely rounds to zero behaves too.
	for _, c := range []struct {
		mode RoundingMode
		want state.State
	}{
		{ROUND_UP, state.Default},
		{ROUND_AWAY_FROM_ZERO, state.Default},
		{ROUND_TOWARD_ZERO, state.Underflow},
		{ROUND_DOWN, state.Underflow},
		{ROUND_BANK, state.Underflow},
	} {
		SetArithmeticRounding(c.mode)
		SetLossPolicy(LossNaNOnUnderflow)
		if got := tenth.PowInt64(60); got.state != c.want {
			t.Errorf("PowInt64(0.1, 60) under %s with LossNaNOnUnderflow = %v (%v), want state %s",
				c.mode, got, got.ErrorDetails(), c.want)
		}
	}

	// Under LossNaNOnInexact every one of them is inexact, whatever direction it took.
	for _, mode := range allModes {
		SetArithmeticRounding(mode)
		SetLossPolicy(LossNaNOnInexact)
		if got := tenth.PowInt64(60); got.state != state.Inexact {
			t.Errorf("PowInt64(0.1, 60) under %s with LossNaNOnInexact = %v", mode, got)
		}
	}
}

// The exponent extremes, against an independent oracle. TestPowIntRoundAgainstBigRat is exact but stops at a few
// hundred, because a big.Rat of d^n is unusable beyond that; big.Float at 320 bits carries the same chain to 2^63
// with a relative error around 2^-310, which is ten orders of magnitude below half a quantum of a 19-place result.
func TestPowIntRoundLargeExponents(t *testing.T) {
	const prec = 320

	powFloat := func(d Dec128, n int64) *big.Float {
		v := new(big.Float).SetPrec(prec).SetInt(bigAtScale(d, d.scale))
		v.Quo(v, new(big.Float).SetPrec(prec).SetInt(bigPow10(d.scale)))
		m := uint64(n)
		if n < 0 {
			m = -m
		}
		acc := new(big.Float).SetPrec(prec).SetInt64(1)
		base := new(big.Float).SetPrec(prec).Set(v)
		for m > 0 {
			if m&1 == 1 {
				acc.Mul(acc, base)
			}
			if m >>= 1; m > 0 {
				base.Mul(base, base)
			}
		}
		if n < 0 {
			acc.Quo(new(big.Float).SetPrec(prec).SetInt64(1), acc)
		}
		return acc
	}

	for _, c := range []struct {
		in    string
		n     int64
		scale uint8
	}{
		{"1.0000000000000000001", math.MaxInt64, MaxScale},
		{"1.0000000000000000001", math.MinInt64, MaxScale},
		{"1.0000000000000000001", 1 << 40, MaxScale},
		{"0.9999999999999999999", 1 << 40, MaxScale},
		{"0.9999999999999999999", -(1 << 40), MaxScale},
		{"1.000000001", 1 << 20, MaxScale},
		{"1.000000001", 1 << 20, 4},
		{"1.0000000000000000001", math.MaxInt64, 2},
		{"0.9999999999999999999", math.MaxInt64, MaxScale},
	} {
		d := FromString(c.in)
		got := d.PowIntRound(c.n, c.scale, ROUND_HALF_AWAY_FROM_ZERO)
		if got.IsNaN() {
			t.Errorf("PowIntRound(%s, %d, %d) = NaN(%v)", c.in, c.n, c.scale, got.ErrorDetails())
			continue
		}

		// the exact value at the requested scale, truncated; the correctly rounded one is within one of it
		want, _ := new(big.Float).Mul(powFloat(d, c.n), new(big.Float).SetPrec(prec).SetInt(bigPow10(c.scale))).Int(nil)
		diff := new(big.Int).Sub(new(big.Int).Abs(bigAtScale(got, c.scale)), want)
		diff.Abs(diff)
		if diff.Cmp(big.NewInt(1)) > 0 {
			t.Errorf("PowIntRound(%s, %d, %d) = %s, the oracle truncates to %s: off by %s",
				c.in, c.n, c.scale, got.StringFixed(), want, diff)
			continue
		}
		t.Logf("PowIntRound(%s, %d, %d) = %s", c.in, c.n, c.scale, got.StringFixed())
	}

	// A power of one is one however large the exponent, and the identity exponent returns the value itself at its
	// own scale rather than a recomputed one.
	for _, d := range []Dec128{MaxAtScale(0), MaxAtScale(MaxScale), FromString("-123.456"), QuantumAtScale(MaxScale)} {
		if got := d.PowIntRound(1, d.scale, ROUND_BANK); got != d {
			t.Errorf("%s^1 = %v, want the value itself", d.StringFixed(), got)
		}
	}
	for _, n := range []int64{1, 2, 1 << 30, math.MaxInt64, -1, math.MinInt64} {
		if got := One.PowIntRound(n, 4, ROUND_BANK); got.StringFixed() != "1.0000" {
			t.Errorf("1^%d = %s", n, got.StringFixed())
		}
	}
}
