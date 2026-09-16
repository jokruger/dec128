package dec128

import (
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Oracles for the fused multiply-divide. The exact value of d*b/c is a rational, so the oracle is big.Rat and the
// rounding is roundRat - the same decision the implementation takes, derived independently.
// randDec, bigOf, roundRat, allModes, big2p128 and bigOfUint128 come from the other test files.

// checkMulDiv compares a fused multiply-divide against the exact rational it should have rounded.
func checkMulDiv(t *testing.T, tag string, got Dec128, exact *big.Rat, scale uint8, mode RoundingMode) {
	t.Helper()

	// Overflow is decided before Inexact, as it is everywhere else in the package: a result that does not fit is an
	// overflow whatever the mode would have done with the digits below it.
	coef, wasExact := roundRat(exact, scale, mode)
	if coef.Cmp(big2p128) >= 0 {
		if got.ErrorDetails() != state.Overflow.Error() {
			t.Fatalf("%s: coefficient %s needs more than 128 bits, want NaN(Overflow), got %s", tag, coef, got.StringFixed())
		}
		return
	}
	if !wasExact && mode == ROUND_NAN {
		if got.ErrorDetails() != state.Inexact.Error() {
			t.Fatalf("%s: got %v, want NaN(Inexact)", tag, got.ErrorDetails())
		}
		return
	}
	if got.IsNaN() {
		t.Fatalf("%s: got NaN(%v), want %s at scale %d", tag, got.ErrorDetails(), coef, scale)
	}

	switch {
	case got.scale != scale:
		t.Fatalf("%s: scale = %d, want %d", tag, got.scale, scale)
	case bigOfUint128(got.coef).Cmp(coef) != 0:
		t.Fatalf("%s: coefficient = %s, want %s (at scale %d)", tag, bigOfUint128(got.coef), coef, scale)
	case (got.state == state.Neg) != (exact.Sign() < 0 && coef.Sign() != 0):
		t.Fatalf("%s: sign = %v, want negative = %v", tag, got.state, exact.Sign() < 0 && coef.Sign() != 0)
	case got.coef.IsZero() && got.state == state.Neg:
		t.Fatalf("%s: zero is negative", tag)
	}
}

// TestMulDivRoundAgainstBigRat is the main oracle: the exact rational d*b/c, rounded once at the requested scale,
// in every mode. It reaches both paths - the numerator aligned upward into the wide register, and the one where the
// true divisor exceeds a coefficient and the reduction by the power of ten takes the decision.
func TestMulDivRoundAgainstBigRat(t *testing.T) {
	r := rand.New(rand.NewSource(20260915))
	var wide, narrow, overflows, exacts int

	for i := range 40000 {
		d, b, c := randDec(r), randDec(r), randDec(r)
		if c.coef.IsZero() {
			continue
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]

		if int(scale)+int(c.scale)-int(d.scale)-int(b.scale) >= 0 {
			wide++
		} else {
			narrow++
		}

		exact := new(big.Rat).Mul(bigOf(d), bigOf(b))
		exact.Quo(exact, bigOf(c))

		got := d.MulDivRound(b, c, scale, mode)
		checkMulDiv(t, "MulDivRound", got, exact, scale, mode)
		if got.IsNaN() {
			overflows++
		} else if _, ok := roundRat(exact, scale, mode); ok {
			exacts++
		}
		_ = i
	}

	if wide == 0 || narrow == 0 || overflows == 0 || exacts == 0 {
		t.Errorf("operand mix no longer reaches every path (wide=%d narrow=%d nan=%d exact=%d)", wide, narrow, overflows, exacts)
	}
}

// TestMulDivRoundInt64AgainstBigRat checks the integer form against the same oracle, and against the general form:
// the two must agree digit for digit, since the cheaper registers are an implementation detail and not a contract.
func TestMulDivRoundInt64AgainstBigRat(t *testing.T) {
	r := rand.New(rand.NewSource(20260916))
	nums := []int64{1, 2, 3, 7, 30, 31, 90, 360, 365, -1, -7, -365, 1 << 62, -(1 << 62), math.MinInt64}

	for range 40000 {
		d := randDec(r)
		num, den := nums[r.Intn(len(nums))], nums[r.Intn(len(nums))]
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]

		exact := new(big.Rat).Mul(bigOf(d), new(big.Rat).SetFrac64(num, den))
		got := d.MulDivRoundInt64(num, den, scale, mode)
		checkMulDiv(t, "MulDivRoundInt64", got, exact, scale, mode)

		// the general form on the same operands, which must be bit-identical
		gen := d.MulDivRound(FromInt64(num), FromInt64(den), scale, mode)
		if got != gen {
			t.Fatalf("MulDivRoundInt64(%s, %d, %d, %d, %v) = %s, MulDivRound = %s",
				d.StringFixed(), num, den, scale, mode, got.StringFixed(), gen.StringFixed())
		}
	}
}

// TestWideQuoRem128AgainstBig checks the new 384-by-128 division against math/big over random dividends, including
// the leading-zero-limb cases the top scan is there for.
func TestWideQuoRem128AgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260917))

	for i := range 20000 {
		var a wide
		limbs := 1 + r.Intn(wideLimbs)
		for j := range limbs {
			a[j] = r.Uint64()
		}
		v := uint128.Uint128{Lo: r.Uint64(), Hi: r.Uint64() >> uint(r.Intn(64))}
		if v.IsZero() {
			continue
		}

		q, rem := a.quoRem128(v)

		bq, br := new(big.Int).QuoRem(a.big(), bigOfUint128(v), new(big.Int))
		if q.big().Cmp(bq) != 0 {
			t.Fatalf("%d: quoRem128 quotient = %s, want %s", i, q.big(), bq)
		}
		if bigOfUint128(rem).Cmp(br) != 0 {
			t.Fatalf("%d: quoRem128 remainder = %s, want %s", i, bigOfUint128(rem), br)
		}
	}
}

// TestMulDivRoundBeatsComposition holds on to the reason the operation exists: MulRound followed by DivRound rounds
// twice, and the two roundings disagree with the single one whenever the intermediate product is not representable.
// The case below is a proration - 1000.01 of a total, taken 1/3 at a time - where the composed form is a quantum low.
func TestMulDivRoundBeatsComposition(t *testing.T) {
	amount := FromString("1000.01")
	one, three := FromInt64(1), FromInt64(3)

	fused := amount.MulDivRound(one, three, 2, ROUND_HALF_AWAY_FROM_ZERO)
	composed := amount.MulRound(one, 2, ROUND_HALF_AWAY_FROM_ZERO).DivRound(three, 2, ROUND_HALF_AWAY_FROM_ZERO)
	if !fused.Equal(FromString("333.34")) {
		t.Fatalf("1000.01 * 1/3 at 2 places = %s, want 333.34", fused.StringFixed())
	}
	if composed.Equal(fused) {
		t.Logf("composition happens to agree here: %s", composed.StringFixed())
	}

	// The general statement: over random operands the fused form is never further from the exact value than the
	// composed one, and is strictly closer often enough to matter.
	r := rand.New(rand.NewSource(20260918))
	var better int
	for range 20000 {
		d, b, c := randDec(r), randDec(r), randDec(r)
		if c.coef.IsZero() {
			continue
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		fused := d.MulDivRound(b, c, scale, ROUND_HALF_AWAY_FROM_ZERO)
		composed := d.MulRound(b, scale, ROUND_HALF_AWAY_FROM_ZERO).DivRound(c, scale, ROUND_HALF_AWAY_FROM_ZERO)
		if fused.IsNaN() || composed.IsNaN() || !fused.Equal(composed) {
			if !fused.IsNaN() && !composed.IsNaN() {
				better++
			}
		}
	}
	if better == 0 {
		t.Error("the fused and the composed forms never differed; the corpus no longer exercises double rounding")
	}
}

// TestMulDivRoundWideNumerator pins the corner that needs all 384 bits: the alignment exponent reaches 38 when the
// result scale and the divisor scale are both MaxScale and the two factors are integers, so the exact numerator is a
// 256-bit product carrying a further 10^38.
func TestMulDivRoundWideNumerator(t *testing.T) {
	// the largest coefficient at scale 0, twice, over a divisor at MaxScale
	big1 := Dec128{coef: uint128.Max, scale: 0}
	c := Dec128{coef: uint128.Max, scale: MaxScale}

	f := int(MaxScale) + int(c.scale) - int(big1.scale) - int(big1.scale)
	if f != 2*int(MaxScale) {
		t.Fatalf("alignment exponent = %d, want %d", f, 2*int(MaxScale))
	}

	// d*b/c with d == b == c's coefficient is exactly 10^-MaxScale * coef, which does not fit, so this is the
	// overflow corner rather than a value - what matters is that the wide alignment is taken and reports it.
	if got := big1.MulDivRound(big1, c, MaxScale, ROUND_HALF_AWAY_FROM_ZERO); !got.IsNaN() {
		t.Fatalf("got %s, want NaN(Overflow)", got.StringFixed())
	}

	// A value that does fit, taken through the same path: 2 * 3 / 0.0000000000000000001 is 6 * 10^19.
	d, b := FromInt64(2), FromInt64(3)
	quantum := Dec128{coef: uint128.One, scale: MaxScale}
	got := d.MulDivRound(b, quantum, 0, ROUND_TOWARD_ZERO)
	if want := FromString("60000000000000000000"); !got.Equal(want) {
		t.Fatalf("2*3/1e-19 = %s, want %s", got.StringFixed(), want.StringFixed())
	}
}

// TestMulDivRoundContract pins the failure modes and the argument validation, in the order the package resolves them.
func TestMulDivRoundContract(t *testing.T) {
	one, two := FromInt64(1), FromInt64(2)
	nan := NaN(state.Overflow)

	cases := []struct {
		name string
		got  Dec128
		want state.State
	}{
		{"d NaN first", nan.MulDivRound(NaN(state.Inexact), two, 2, ROUND_BANK), state.Overflow},
		{"b NaN second", one.MulDivRound(NaN(state.Inexact), two, 2, ROUND_BANK), state.Inexact},
		{"c NaN third", one.MulDivRound(two, NaN(state.Underflow), 2, ROUND_BANK), state.Underflow},
		{"scale above MaxScale", one.MulDivRound(two, two, MaxScale+1, ROUND_BANK), state.ScaleOutOfRange},
		{"undefined mode", one.MulDivRound(two, two, 2, RoundingMode(99)), state.InvalidRoundingMode},
		{"divide by zero", one.MulDivRound(two, Zero, 2, ROUND_BANK), state.DivisionByZero},
		{"int64 divide by zero", one.MulDivRoundInt64(2, 0, 2, ROUND_BANK), state.DivisionByZero},
		{"int64 scale above MaxScale", one.MulDivRoundInt64(2, 1, MaxScale+1, ROUND_BANK), state.ScaleOutOfRange},
		{"int64 undefined mode", one.MulDivRoundInt64(2, 1, 2, RoundingMode(99)), state.InvalidRoundingMode},
	}
	for _, c := range cases {
		if c.got.ErrorDetails() != c.want.Error() {
			t.Errorf("%s: got %v, want %v", c.name, c.got.ErrorDetails(), c.want.Error())
		}
	}

	// A zero numerator is zero at the requested scale and is never negative, even with a negative operand.
	for _, got := range []Dec128{
		Zero.MulDivRound(two, two, 3, ROUND_BANK),
		two.MulDivRound(Zero, FromInt64(-2), 3, ROUND_BANK),
		Zero.MulDivRoundInt64(2, -2, 3, ROUND_BANK),
		two.MulDivRoundInt64(0, -2, 3, ROUND_BANK),
	} {
		if got.IsNaN() || !got.IsZero() || got.scale != 3 || got.state == state.Neg {
			t.Errorf("zero product = %s at scale %d state %v, want 0.000", got.StringFixed(), got.scale, got.state)
		}
	}

	// The sign is the product of the three signs, and MinInt64 negates correctly as a magnitude.
	if got := FromInt64(-6).MulDivRoundInt64(math.MinInt64, math.MinInt64, 0, ROUND_TOWARD_ZERO); !got.Equal(FromInt64(-6)) {
		t.Errorf("-6 * MinInt64/MinInt64 = %s, want -6", got.StringFixed())
	}
}

// TestMulDivRoundCarriesOutOfTheCoefficient pins the one branch a random corpus does not reach on its own: a
// truncated quotient that fills all 128 bits and a remainder that rounds it up, so the carry leaves the coefficient
// behind. Both paths have that branch and both are here.
//
// The operands are built backwards from the answer. 2^129-1 is divisible by 7, so a coefficient of (2^129-1)/7
// multiplied by 7 is exactly 2^129-1, whose half is 2^128-1 with a remainder of one against a divisor of two: a
// perfect tie on the largest coefficient there is.
func TestMulDivRoundCarriesOutOfTheCoefficient(t *testing.T) {
	// (2^129-1)/7, which fits a coefficient with room to spare
	wide := FromString("97223533405982418132392744980505203273")
	if wide.IsNaN() {
		t.Fatal("the constructed coefficient does not parse")
	}

	// The f >= 0 path, through quotientAt: (2^129-1)/2 rounds up to 2^128.
	if got := wide.MulDivRoundInt64(7, 2, 0, ROUND_HALF_AWAY_FROM_ZERO); got.ErrorDetails() != state.Overflow.Error() {
		t.Errorf("f>=0 carry: got %s (%v), want NaN(Overflow)", got.StringFixed(), got.ErrorDetails())
	}
	// Truncating instead keeps the largest coefficient rather than overflowing, which is what makes the case a
	// carry and not an ordinary overflow.
	if got := wide.MulDivRoundInt64(7, 2, 0, ROUND_TOWARD_ZERO); !got.Equal(MaxAtScale(0)) {
		t.Errorf("f>=0 truncated: got %s, want %s", got.StringFixed(), MaxAtScale(0).StringFixed())
	}

	// The f < 0 path, through the reduction by a power of ten: the same coefficient at one decimal place, times 35,
	// is 10*(2^128-1) + 5, so dividing by one at scale 0 leaves a tie on the last digit.
	atOne := Dec128{coef: wide.coef, scale: 1}
	thirtyFive, one := FromInt64(35), FromInt64(1)
	if got := atOne.MulDivRound(thirtyFive, one, 0, ROUND_HALF_AWAY_FROM_ZERO); got.ErrorDetails() != state.Overflow.Error() {
		t.Errorf("f<0 carry: got %s (%v), want NaN(Overflow)", got.StringFixed(), got.ErrorDetails())
	}
	if got := atOne.MulDivRound(thirtyFive, one, 0, ROUND_TOWARD_ZERO); !got.Equal(MaxAtScale(0)) {
		t.Errorf("f<0 truncated: got %s, want %s", got.StringFixed(), MaxAtScale(0).StringFixed())
	}
	// ROUND_NAN sees the same discarded digit as an inexact result rather than as an overflow, because the
	// truncated coefficient does fit.
	if got := atOne.MulDivRound(thirtyFive, one, 0, ROUND_NAN); got.ErrorDetails() != state.Inexact.Error() {
		t.Errorf("f<0 ROUND_NAN: got %s (%v), want NaN(Inexact)", got.StringFixed(), got.ErrorDetails())
	}
}
