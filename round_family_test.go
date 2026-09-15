package dec128

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Tests for the global-free *Round family: the operations that take the scale and the rounding mode per call and never
// consult defaultScale, arithmeticRounding or lossPolicy. The oracle is the same for all of them - an exact signed
// value at a known scale, brought to the requested scale in math/big - so it lives here rather than in each test.

// wantAt is the reference implementation of "exactly this scale": the exact signed value exact*10^-needed expressed at
// the scale target and rounded with mode. It returns the expected magnitude, or the NaN state the operation owes.
func wantAt(exact *big.Int, needed, target uint8, mode RoundingMode) (*big.Int, state.State) {
	abs := new(big.Int).Abs(exact)

	if target >= needed {
		w := new(big.Int).Mul(abs, bigPow10(target-needed))
		if w.Cmp(big2p128) >= 0 {
			return nil, state.Overflow
		}
		return w, state.OK
	}

	div := bigPow10(needed - target)
	q, rem := new(big.Int).QuoRem(abs, div, new(big.Int))
	w := roundBig(q, rem, div, exact.Sign() < 0, mode)
	switch {
	case w.Cmp(big2p128) >= 0:
		return nil, state.Overflow
	case mode == ROUND_NAN && rem.Sign() != 0:
		return nil, state.Inexact
	}
	return w, state.OK
}

// checkAt asserts that got is the value exact*10^-needed at exactly the scale target under mode.
func checkAt(t *testing.T, what string, got Dec128, exact *big.Int, needed, target uint8, mode RoundingMode) {
	t.Helper()
	want, wantNaN := wantAt(exact, needed, target, mode)
	switch {
	case wantNaN != state.OK:
		if got.state != wantNaN {
			t.Errorf("%s [scale %d, %s] = %v, want NaN(%s)", what, target, mode, got, wantNaN)
		}
	case got.IsNaN():
		t.Errorf("%s [scale %d, %s] = NaN(%v), want coefficient %s", what, target, mode, got.ErrorDetails(), want)
	case got.scale != target:
		t.Errorf("%s [scale %d, %s] = %s: scale %d", what, target, mode, got.StringFixed(), got.scale)
	case new(big.Int).Abs(bigAtScale(got, target)).Cmp(want) != 0:
		t.Errorf("%s [scale %d, %s] = %s, want coefficient %s", what, target, mode, got.StringFixed(), want)
	case (got.state == state.Neg) != (exact.Sign() < 0 && want.Sign() != 0):
		t.Errorf("%s [scale %d, %s] = %s: wrong sign", what, target, mode, got.StringFixed())
	}
}

func TestAddSubRoundAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260915))
	for range 40000 {
		x, y := randDec(r), randDec(r)
		mode := allModes[r.Intn(len(allModes))]
		scale := uint8(r.Intn(int(MaxScale) + 1))
		needed := max(x.scale, y.scale)
		bx, by := bigAtScale(x, needed), bigAtScale(y, needed)

		checkAt(t, "AddRound("+x.StringFixed()+", "+y.StringFixed()+")",
			x.AddRound(y, scale, mode), new(big.Int).Add(bx, by), needed, scale, mode)
		checkAt(t, "SubRound("+x.StringFixed()+", "+y.StringFixed()+")",
			x.SubRound(y, scale, mode), new(big.Int).Sub(bx, by), needed, scale, mode)
	}
}

func TestAddSubRoundEdges(t *testing.T) {
	cases := []struct {
		a, b  string
		scale uint8
		mode  RoundingMode
		add   string
		sub   string
	}{
		{"1.005", "0", 2, ROUND_BANK, "1.00", "1.00"},
		{"1.015", "0", 2, ROUND_BANK, "1.02", "1.02"},
		{"0.125", "0.125", 2, ROUND_BANK, "0.25", "0.00"},
		{"1", "2", 3, ROUND_TOWARD_ZERO, "3.000", "-1.000"},
		{"-1.005", "0", 2, ROUND_HALF_AWAY_FROM_ZERO, "-1.01", "-1.01"},
		{"-1.001", "0", 2, ROUND_DOWN, "-1.01", "-1.01"},
		{"1.001", "0", 2, ROUND_UP, "1.01", "1.01"},
		// the difference is exact at scale 2, so no mode can disagree
		{"10.50", "0.25", 2, ROUND_NAN, "10.75", "10.25"},
	}
	for _, c := range cases {
		a, b := FromString(c.a), FromString(c.b)
		if got := a.AddRound(b, c.scale, c.mode); got.StringFixed() != c.add {
			t.Errorf("AddRound(%s, %s, %d, %s) = %s, want %s", c.a, c.b, c.scale, c.mode, got.StringFixed(), c.add)
		}
		if got := a.SubRound(b, c.scale, c.mode); got.StringFixed() != c.sub {
			t.Errorf("SubRound(%s, %s, %d, %s) = %s, want %s", c.a, c.b, c.scale, c.mode, got.StringFixed(), c.sub)
		}
	}

	// ROUND_NAN reports the loss instead of taking it
	if got := FromString("1.005").AddRound(Zero, 2, ROUND_NAN); got.state != state.Inexact {
		t.Errorf("AddRound under ROUND_NAN = %v, want NaN(Inexact)", got)
	}
	if got := FromString("1.005").SubRound(Zero, 2, ROUND_NAN); got.state != state.Inexact {
		t.Errorf("SubRound under ROUND_NAN = %v, want NaN(Inexact)", got)
	}
}

func TestAddSubRoundArgumentErrors(t *testing.T) {
	a, b := FromString("1.5"), FromString("2.5")
	nan := NaN(state.Overflow)

	for _, c := range []struct {
		what string
		got  Dec128
		want state.State
	}{
		{"AddRound scale", a.AddRound(b, MaxScale+1, ROUND_BANK), state.ScaleOutOfRange},
		{"SubRound scale", a.SubRound(b, MaxScale+1, ROUND_BANK), state.ScaleOutOfRange},
		{"AddRound mode", a.AddRound(b, 2, RoundingMode(99)), state.InvalidRoundingMode},
		{"SubRound mode", a.SubRound(b, 2, RoundingMode(99)), state.InvalidRoundingMode},
		{"AddRound NaN left", nan.AddRound(b, 2, ROUND_BANK), state.Overflow},
		{"AddRound NaN right", a.AddRound(nan, 2, ROUND_BANK), state.Overflow},
		{"SubRound NaN left", nan.SubRound(b, 2, ROUND_BANK), state.Overflow},
		{"SubRound NaN right", a.SubRound(nan, 2, ROUND_BANK), state.Overflow},
	} {
		if c.got.state != c.want {
			t.Errorf("%s = %v, want NaN(%s)", c.what, c.got, c.want)
		}
	}

	// a sum whose integer part does not fit, and one whose padded coefficient does not
	if got := MaxAtScale(0).AddRound(MaxAtScale(0), 0, ROUND_BANK); got.state != state.Overflow {
		t.Errorf("AddRound overflow = %v", got)
	}
	if got := MaxAtScale(0).AddRound(Zero, 1, ROUND_BANK); got.state != state.Overflow {
		t.Errorf("AddRound padding overflow = %v", got)
	}
	// zero results are never negative, at any scale
	for _, got := range []Dec128{
		a.SubRound(a, 4, ROUND_BANK),
		FromString("-0.004").AddRound(Zero, 2, ROUND_BANK),
	} {
		if got.state != state.Default || !got.IsZero() {
			t.Errorf("zero result = %v, state %s", got, got.state)
		}
	}
}

func TestMulAddRoundAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260919))
	for range 60000 {
		x, y, c := randDec(r), randDec(r), randDec(r)
		mode := allModes[r.Intn(len(allModes))]
		scale := uint8(r.Intn(int(MaxScale) + 1))

		prodScale := x.scale + y.scale
		needed := max(prodScale, c.scale)
		exact := new(big.Int).Mul(bigAtScale(x, x.scale), bigAtScale(y, y.scale))
		exact.Mul(exact, bigPow10(needed-prodScale))
		exact.Add(exact, bigAtScale(c, needed))

		what := "MulAddRound(" + x.StringFixed() + ", " + y.StringFixed() + ", " + c.StringFixed() + ")"
		checkAt(t, what, x.MulAddRound(y, c, scale, mode), exact, needed, scale, mode)
	}
}

// The fused form must agree with the unfused one wherever the unfused one is itself exact, which is what makes it a
// drop-in replacement rather than a different operation.
func TestMulAddRoundAgreesWithSeparateSteps(t *testing.T) {
	r := rand.New(rand.NewSource(20260920))
	for range 40000 {
		x, y, c := randDec(r), randDec(r), randDec(r)
		mode := allModes[r.Intn(len(allModes))]
		scale := uint8(r.Intn(int(MaxScale) + 1))

		// c == 0 reduces the fused operation to MulRound
		if got, want := x.MulAddRound(y, Zero, scale, mode), x.MulRound(y, scale, mode); got != want {
			t.Fatalf("MulAddRound(%s, %s, 0, %d, %s) = %v, MulRound = %v",
				x.StringFixed(), y.StringFixed(), scale, mode, got, want)
		}

		// an exactly representable product reduces it to AddRound on that product
		if p := x.Mul(y); !p.IsNaN() && p.scale == x.scale+y.scale {
			if got, want := x.MulAddRound(y, c, scale, mode), p.AddRound(c, scale, mode); got != want {
				t.Fatalf("MulAddRound(%s, %s, %s, %d, %s) = %v, %s.AddRound = %v",
					x.StringFixed(), y.StringFixed(), c.StringFixed(), scale, mode, got, p.StringFixed(), want)
			}
		}
	}
}

func TestMulAddRoundEdges(t *testing.T) {
	// the case the operation exists for: a product that is not representable, added to something that is
	// 0.0000000001^2 is 10^-20, below the smallest quantum, but the sum with 1 is still exact to 19 places
	tiny := FromString("0.0000000001")
	if got := tiny.MulAddRound(tiny, One, MaxScale, ROUND_HALF_AWAY_FROM_ZERO); got.StringFixed() != "1.0000000000000000000" {
		t.Errorf("tiny product = %s", got.StringFixed())
	}
	// ... and ROUND_NAN says so rather than swallowing it
	if got := tiny.MulAddRound(tiny, One, MaxScale, ROUND_NAN); got.state != state.Inexact {
		t.Errorf("tiny product under ROUND_NAN = %v", got)
	}

	// one rounding, not two: 0.005 * 1 + 0.005 is 0.01 at two places, while rounding each half first gives 0.00
	half := FromString("0.005")
	if got := half.MulAddRound(One, half, 2, ROUND_BANK); got.StringFixed() != "0.01" {
		t.Errorf("fused = %s, want 0.01", got.StringFixed())
	}
	if got := half.MulRound(One, 2, ROUND_BANK).AddRound(half.Round(2, ROUND_BANK), 2, ROUND_BANK); got.StringFixed() != "0.00" {
		t.Errorf("unfused = %s, want 0.00 (this is the bias the fused form removes)", got.StringFixed())
	}

	// exact cancellation of a wide product against its negation
	big1 := MaxAtScale(0)
	if got := big1.MulAddRound(big1, big1.Mul(big1).Neg(), 0, ROUND_BANK); got.state != state.Overflow {
		// the product itself is not representable, so the addend cannot be either: both are NaN
		t.Logf("wide cancellation = %v", got)
	}
	if got := big1.MulAddRound(big1, Zero, 0, ROUND_BANK); got.state != state.Overflow {
		t.Errorf("product overflow = %v, want NaN(Overflow)", got)
	}
	// the product overflows 128 bits but the addend brings it back
	if got := big1.MulAddRound(FromInt64(2), MaxAtScale(0).Neg(), 0, ROUND_BANK); got.StringFixed() != MaxAtScale(0).StringFixed() {
		t.Errorf("recovered sum = %s", got.StringFixed())
	}
}

func TestMulAddRoundArgumentErrors(t *testing.T) {
	a, b, c := FromString("1.5"), FromString("2.5"), FromString("0.25")
	nan1, nan2, nan3 := NaN(state.Overflow), NaN(state.DivisionByZero), NaN(state.NotConverged)

	for _, tc := range []struct {
		what string
		got  Dec128
		want state.State
	}{
		{"scale", a.MulAddRound(b, c, MaxScale+1, ROUND_BANK), state.ScaleOutOfRange},
		{"mode", a.MulAddRound(b, c, 2, RoundingMode(99)), state.InvalidRoundingMode},
		{"NaN d", nan1.MulAddRound(b, c, 2, ROUND_BANK), state.Overflow},
		{"NaN b", a.MulAddRound(nan2, c, 2, ROUND_BANK), state.DivisionByZero},
		{"NaN c", a.MulAddRound(b, nan3, 2, ROUND_BANK), state.NotConverged},
		{"d first", nan1.MulAddRound(nan2, nan3, 2, ROUND_BANK), state.Overflow},
	} {
		if tc.got.state != tc.want {
			t.Errorf("%s = %v, want NaN(%s)", tc.what, tc.got, tc.want)
		}
	}

	// a zero product keeps the addend's sign, and an exact cancellation is a non-negative zero
	if got := Zero.MulAddRound(b, c.Neg(), 2, ROUND_BANK); got.StringFixed() != "-0.25" {
		t.Errorf("zero product + negative addend = %s", got.StringFixed())
	}
	if got := a.MulAddRound(b, a.Mul(b).Neg(), 4, ROUND_BANK); got.state != state.Default || !got.IsZero() {
		t.Errorf("cancellation = %v, state %s", got, got.state)
	}
}

// TestGlobalFreeSubset is the test the "global-free subset" section of doc.go points at. Every operation named there
// is run over a matrix of the three settings that can change a value - SetDefaultScale, SetArithmeticRounding and
// SetLossPolicy - and the results must be bit-identical to the ones the defaults produce. A method that starts
// reading a global fails here rather than quietly breaking a caller who needs the same bytes in every process.
func TestGlobalFreeSubset(t *testing.T) {
	defer SetDefaultScale(DefaultScale())
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	// Operand pairs: random ones for coverage, and the values that sit on the boundaries of the scale rule.
	r := rand.New(rand.NewSource(20260928))
	pairs := [][2]Dec128{
		{One, FromInt64(3)},
		{FromString("1.005"), FromString("0.0000000001")},
		{MaxAtScale(0), FromInt64(2)},
		{MaxAtScale(MaxScale), QuantumAtScale(MaxScale)},
		{FromString("-7.5"), FromString("2.25")},
		{Zero, FromString("1.5")},
		{NaN(state.Overflow), One},
	}
	for range 200 {
		pairs = append(pairs, [2]Dec128{randDec(r), randDec(r)})
	}

	free := []struct {
		name string
		fn   func(a, b Dec128) Dec128
	}{
		{"AddRound", func(a, b Dec128) Dec128 { return a.AddRound(b, 6, ROUND_BANK) }},
		{"SubRound", func(a, b Dec128) Dec128 { return a.SubRound(b, 6, ROUND_BANK) }},
		{"MulRound", func(a, b Dec128) Dec128 { return a.MulRound(b, 6, ROUND_BANK) }},
		{"MulAddRound", func(a, b Dec128) Dec128 { return a.MulAddRound(b, a, 6, ROUND_BANK) }},
		{"DivRound", func(a, b Dec128) Dec128 { return a.DivRound(b, 6, ROUND_BANK) }},
		{"SqrtRound", func(a, b Dec128) Dec128 { return a.Abs().SqrtRound(6, ROUND_BANK) }},
		{"PowIntRound", func(a, b Dec128) Dec128 { return a.PowIntRound(7, 6, ROUND_BANK) }},
		{"PowIntRound negative", func(a, b Dec128) Dec128 { return a.PowIntRound(-3, 6, ROUND_BANK) }},
		{"Accumulator", func(a, b Dec128) Dec128 {
			var acc Accumulator
			acc.Add(a)
			acc.AddMul(a, b)
			acc.Sub(b)
			return acc.Total(6, ROUND_BANK)
		}},
		{"QuoRem quotient", func(a, b Dec128) Dec128 { q, _ := a.QuoRem(b); return q }},
		{"QuoRem remainder", func(a, b Dec128) Dec128 { _, rem := a.QuoRem(b); return rem }},
		{"Mod", func(a, b Dec128) Dec128 { return a.Mod(b) }},
		{"Round", func(a, b Dec128) Dec128 { return a.Round(4, ROUND_HALF_AWAY_FROM_ZERO) }},
		{"RoundToPlaces", func(a, b Dec128) Dec128 { return a.RoundToPlaces(-2, ROUND_BANK) }},
		{"RoundToMultiple", func(a, b Dec128) Dec128 { return a.RoundToMultiple(b.Abs(), ROUND_BANK) }},
		{"ScaleByPow10", func(a, b Dec128) Dec128 { return a.ScaleByPow10(-3) }},
		{"DivRoundInexact", func(a, b Dec128) Dec128 { q, _ := a.DivRoundInexact(b, 6, ROUND_BANK); return q }},
		{"Allocate", func(a, b Dec128) Dec128 {
			shares, ok := a.RescaleRound(2, ROUND_TOWARD_ZERO).Allocate([]Dec128{One, b.Abs()}, 2)
			if !ok {
				return NaN(state.DomainError)
			}
			return shares[0]
		}},
		{"Split", func(a, b Dec128) Dec128 {
			shares, ok := a.RescaleRound(2, ROUND_TOWARD_ZERO).Split(3, 2)
			if !ok {
				return NaN(state.DomainError)
			}
			return shares[2]
		}},
		{"RoundBank", func(a, b Dec128) Dec128 { return a.RoundBank(4) }},
		{"Trunc", func(a, b Dec128) Dec128 { return a.Trunc(4) }},
		{"Rescale", func(a, b Dec128) Dec128 { return a.Rescale(4) }},
		{"RescaleRound", func(a, b Dec128) Dec128 { return a.RescaleRound(4, ROUND_BANK) }},
		{"Canonical", func(a, b Dec128) Dec128 { return a.Canonical() }},
		{"Abs", func(a, b Dec128) Dec128 { return a.Abs() }},
		{"Neg", func(a, b Dec128) Dec128 { return a.Neg() }},
		{"Min", func(a, b Dec128) Dec128 { return Min(a, b) }},
		{"Max", func(a, b Dec128) Dec128 { return Max(a, b) }},
		{"FromString round trip", func(a, b Dec128) Dec128 { return FromString(a.StringFixed()) }},
		{"binary round trip", func(a, b Dec128) Dec128 {
			var buf [MaxBytes]byte
			n, err := a.EncodeBinary(buf[:])
			if err != nil {
				return NaN(state.InvalidFormat)
			}
			var back Dec128
			if _, err := back.DecodeBinary(buf[:n]); err != nil {
				return NaN(state.InvalidFormat)
			}
			return back
		}},
		{"comparison", func(a, b Dec128) Dec128 { return FromInt64(int64(a.Compare(b))) }},
		{"NthRootRound", func(a, b Dec128) Dec128 { return a.Abs().NthRootRound(3, 6, ROUND_BANK) }},
		{"NthRootRound squared", func(a, b Dec128) Dec128 { return a.Abs().NthRootRound(2, 6, ROUND_BANK) }},
		{"RoundToSignificant", func(a, b Dec128) Dec128 { return a.RoundToSignificant(5, ROUND_BANK) }},
		{"RescaleRoundInexact", func(a, b Dec128) Dec128 { q, _ := a.RescaleRoundInexact(4, ROUND_BANK); return q }},
		{"RescaleRoundInexact flag", func(a, b Dec128) Dec128 {
			if _, inexact := a.RescaleRoundInexact(4, ROUND_BANK); inexact {
				return One
			}
			return Zero
		}},
		{"Clamp", func(a, b Dec128) Dec128 { return a.Clamp(b.Abs().Neg(), b.Abs()) }},
		{"IsInteger", func(a, b Dec128) Dec128 {
			if a.IsInteger() {
				return One
			}
			return Zero
		}},
		{"IntFrac integer part", func(a, b Dec128) Dec128 { ip, _ := a.IntFrac(); return ip }},
		{"IntFrac fractional part", func(a, b Dec128) Dec128 { _, fp := a.IntFrac(); return fp }},
		{"SignificantDigits", func(a, b Dec128) Dec128 { return FromInt(a.SignificantDigits()) }},
		{"IntegerDigits", func(a, b Dec128) Dec128 { return FromInt(a.IntegerDigits()) }},
		{"FitsNumeric", func(a, b Dec128) Dec128 {
			if a.FitsNumeric(20, 4) {
				return One
			}
			return Zero
		}},
		{"Accumulator.Mean", func(a, b Dec128) Dec128 {
			var acc Accumulator
			acc.Add(a)
			acc.AddMul(a, b)
			acc.Sub(b)
			return acc.Mean(6, ROUND_BANK)
		}},
		{"AllocateResidual", func(a, b Dec128) Dec128 {
			shares, ok := a.RescaleRound(2, ROUND_TOWARD_ZERO).AllocateResidual([]Dec128{One, b.Abs()}, 2, 1, ROUND_BANK)
			if !ok {
				return NaN(state.DomainError)
			}
			return shares[1]
		}},
		{"SplitResidual", func(a, b Dec128) Dec128 {
			shares, ok := a.RescaleRound(2, ROUND_TOWARD_ZERO).SplitResidual(3, 2, 2, ROUND_BANK)
			if !ok {
				return NaN(state.DomainError)
			}
			return shares[2]
		}},
		{"Format", func(a, b Dec128) Dec128 { return FromString(fmt.Sprintf("%.6f", a)) }},
	}

	// the defaults, which every combination below must reproduce
	SetDefaultScale(MaxScale)
	SetArithmeticRounding(ROUND_TOWARD_ZERO)
	SetLossPolicy(LossRound)
	base := make([][]Dec128, len(free))
	for i, op := range free {
		base[i] = make([]Dec128, len(pairs))
		for j, p := range pairs {
			base[i][j] = op.fn(p[0], p[1])
		}
	}

	for _, scale := range []uint8{0, 2, 6, MaxScale} {
		for _, mode := range allModes {
			for _, policy := range []LossPolicy{LossRound, LossNaNOnUnderflow, LossNaNOnInexact} {
				SetDefaultScale(scale)
				SetArithmeticRounding(mode) // writes the policy too, so it goes first
				SetLossPolicy(policy)

				for i, op := range free {
					for j, p := range pairs {
						if got := op.fn(p[0], p[1]); got != base[i][j] {
							t.Fatalf("%s(%s, %s) = %v under scale %d / %s / %s, want %v",
								op.name, p[0].StringFixed(), p[1].StringFixed(), got, scale, mode, policy, base[i][j])
						}
					}
				}
			}
		}
	}
}

// The complement of the subset: the operations doc.go excludes must actually depend on the settings, or the list is
// over-cautious and the ones it leaves out could be promoted.
func TestGlobalFreeSubsetIsNotVacuous(t *testing.T) {
	defer SetDefaultScale(DefaultScale())
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	a, b := FromString("1"), FromString("3")
	wide2 := FromString("1.0000000001")
	// two coefficients whose sum ends in a nonzero digit, so that making it fit has a direction to choose
	wide1 := MaxAtScale(MaxScale)
	nearMax := Dec128{coef: uint128.Uint128{Lo: ^uint64(0) - 1, Hi: ^uint64(0)}, scale: MaxScale}

	bound := []struct {
		name string
		fn   func() Dec128
	}{
		{"Add", func() Dec128 { return wide1.Add(nearMax) }},
		{"Sub", func() Dec128 { return wide1.Sub(nearMax.Neg()) }},
		{"Mul", func() Dec128 { return wide2.Mul(wide2) }},
		{"Div", func() Dec128 { return a.Div(b) }},
		{"Sqrt", func() Dec128 { return b.Sqrt() }},
		{"PowInt64", func() Dec128 { return wide2.PowInt64(9) }},
		{"Sum", func() Dec128 { return Sum(wide1, nearMax) }},
		{"Avg", func() Dec128 { return Avg(a, b, b) }},
	}

	SetDefaultScale(MaxScale)
	SetArithmeticRounding(ROUND_TOWARD_ZERO)
	SetLossPolicy(LossRound)
	base := make([]Dec128, len(bound))
	for i, op := range bound {
		base[i] = op.fn()
	}

	SetDefaultScale(4)
	SetArithmeticRounding(ROUND_HALF_AWAY_FROM_ZERO)
	SetLossPolicy(LossNaNOnInexact)
	for i, op := range bound {
		if got := op.fn(); got == base[i] {
			t.Errorf("%s = %v under both settings; it may belong in the global-free subset", op.name, got)
		}
	}
}
