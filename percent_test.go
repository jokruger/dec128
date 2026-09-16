package dec128

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
)

func bigPow10Int(n int) *big.Int {
	return new(big.Int).Exp(bigTen, big.NewInt(int64(n)), nil)
}

// exactProduct returns |d.coef * f.coef| and the sign of the product, which is the exact value of d*f at the scale
// d.scale + f.scale.
func exactProduct(d, f Dec128) (*big.Int, bool) {
	p := new(big.Int).Mul(new(big.Int).Abs(bigAtScale(d, d.scale)), new(big.Int).Abs(bigAtScale(f, f.scale)))
	return p, (d.state == state.Neg) != (f.state == state.Neg) && p.Sign() != 0
}

// scaledRoundOracle is the reference for MulScaledRound: the exact product times 10^k brought to exactly scale, with
// the implementation's ordering of the failures - a coefficient that does not fit is an overflow before the rounding
// mode gets to object to the loss.
func scaledRoundOracle(d, f Dec128, k int, scale uint8, mode RoundingMode) (coef *big.Int, neg bool, nan state.State) {
	prod, neg := exactProduct(d, f)
	if prod.Sign() == 0 {
		return big.NewInt(0), false, state.OK
	}

	g := k + int(scale) - int(d.scale) - int(f.scale)
	q := new(big.Int)
	if g >= 0 {
		q.Mul(prod, bigPow10Int(g))
		if q.Cmp(big2p128) >= 0 {
			return nil, false, state.Overflow
		}
	} else {
		div := bigPow10Int(-g)
		r := new(big.Int)
		q.QuoRem(prod, div, r)
		if q.Cmp(big2p128) >= 0 {
			return nil, false, state.Overflow
		}
		if r.Sign() != 0 && mode == ROUND_NAN {
			return nil, false, state.Inexact
		}
		q = roundBig(q, r, div, neg, mode)
		if q.Cmp(big2p128) >= 0 {
			return nil, false, state.Overflow
		}
	}
	return q, neg && q.Sign() != 0, state.OK
}

func checkScaledRound(t *testing.T, op string, d, f Dec128, k int, scale uint8, mode RoundingMode, got Dec128) {
	t.Helper()
	wantCoef, wantNeg, wantNaN := scaledRoundOracle(d, f, k, scale, mode)
	if wantNaN != state.OK {
		if got.state != wantNaN {
			t.Errorf("%s(%s, %s, k=%d, scale=%d, %s) = %v, want NaN(%s)", op, d.StringFixed(), f.StringFixed(), k, scale, mode, got, wantNaN)
		}
		return
	}
	if got.IsNaN() {
		t.Errorf("%s(%s, %s, k=%d, scale=%d, %s) = NaN(%v), want %s", op, d.StringFixed(), f.StringFixed(), k, scale, mode, got.ErrorDetails(), wantCoef)
		return
	}
	if got.scale != scale {
		t.Errorf("%s(%s, %s, k=%d, scale=%d, %s) = %s at scale %d", op, d.StringFixed(), f.StringFixed(), k, scale, mode, got.StringFixed(), got.scale)
		return
	}
	if gotAbs := new(big.Int).Abs(bigAtScale(got, got.scale)); gotAbs.Cmp(wantCoef) != 0 {
		t.Errorf("%s(%s, %s, k=%d, scale=%d, %s) = %s, want coefficient %s", op, d.StringFixed(), f.StringFixed(), k, scale, mode, got.StringFixed(), wantCoef)
		return
	}
	if (got.state == state.Neg) != wantNeg {
		t.Errorf("%s(%s, %s, k=%d, scale=%d, %s) = %s: wrong sign", op, d.StringFixed(), f.StringFixed(), k, scale, mode, got.StringFixed())
	}
}

func TestMulPercentExamples(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())
	SetArithmeticRounding(ROUND_TOWARD_ZERO)
	SetLossPolicy(LossRound)

	tests := []struct {
		amount, rate string
		want         string // the exact product, with its scale
		wantBank2    string // and what the caller presents
	}{
		{"100.00", "7.5", "7.50000", "7.50"},
		{"1000", "20", "200.00", "200.00"},
		{"1234.56", "19", "234.5664", "234.57"},
		{"-1234.56", "19", "-234.5664", "-234.57"},
		{"1234.56", "-19", "-234.5664", "-234.57"},
		{"0.01", "50", "0.0050", "0.00"}, // half to even, and the tie is exact
		{"1", "0.5", "0.005", "0.00"},
		{"12.345", "100", "12.34500", "12.34"},
		{"0", "7.5", "0.000", "0.00"},
		{"250.00", "0", "0.0000", "0.00"},
	}

	for _, tt := range tests {
		a, r := FromString(tt.amount), FromString(tt.rate)
		got := a.MulPercent(r)
		if got.StringFixed() != tt.want {
			t.Errorf("%s.MulPercent(%s) = %s, want %s", tt.amount, tt.rate, got.StringFixed(), tt.want)
		}
		if p := got.RescaleRound(2, ROUND_BANK); p.StringFixed() != tt.wantBank2 {
			t.Errorf("%s.MulPercent(%s).RescaleRound(2, ROUND_BANK) = %s, want %s", tt.amount, tt.rate, p.StringFixed(), tt.wantBank2)
		}
		// the global-free form at the presented scale agrees with rounding the exact product once
		if pr := a.MulPercentRound(r, 2, ROUND_BANK); pr.StringFixed() != tt.wantBank2 {
			t.Errorf("%s.MulPercentRound(%s, 2, ROUND_BANK) = %s, want %s", tt.amount, tt.rate, pr.StringFixed(), tt.wantBank2)
		}
	}
}

func TestMulScaledExamples(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())
	SetArithmeticRounding(ROUND_TOWARD_ZERO)
	SetLossPolicy(LossRound)

	tests := []struct {
		d, f string
		k    int
		want string
	}{
		{"1000", "25", -4, "2.5000"}, // basis points
		{"1000", "25", -3, "25.000"}, // per mille
		{"1000", "25", -2, "250.00"}, // per cent
		{"1000", "25", 0, "25000"},   // plain multiplication
		{"1000", "25", 2, "2500000"}, // and the other direction
		{"1.5", "2.5", -2, "0.0375"}, // scales add, then the point moves
		{"-1.5", "2.5", -2, "-0.0375"},
		{"1.5", "-2.5", 2, "-375"},
		{"0", "2.5", -2, "0.000"},
	}

	for _, tt := range tests {
		got := FromString(tt.d).MulScaled(FromString(tt.f), tt.k)
		if got.StringFixed() != tt.want {
			t.Errorf("%s.MulScaled(%s, %d) = %s, want %s", tt.d, tt.f, tt.k, got.StringFixed(), tt.want)
		}
	}
}

// MulScaled with k = 0 is Mul, in the exact case and in the case that has to be made to fit, under every combination
// of the two settings Mul reads.
func TestMulScaledIsMulAtZero(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	r := rand.New(rand.NewSource(20260916))
	for _, mode := range allModes {
		for _, policy := range []LossPolicy{LossRound, LossNaNOnUnderflow, LossNaNOnInexact} {
			SetArithmeticRounding(mode)
			SetLossPolicy(policy)
			for i := 0; i < 2000; i++ {
				a, b := randDec(r), randDec(r)
				want, got := a.Mul(b), a.MulScaled(b, 0)
				if want != got {
					t.Fatalf("[%s/%v] %s.MulScaled(%s, 0) = %v, Mul = %v", mode, policy, a.StringFixed(), b.StringFixed(), got, want)
				}
			}
		}
	}
}

// An exact result of MulScaled is the exact product with the point moved, which is what ScaleByPow10 does, so the two
// must agree wherever ScaleByPow10 can express the answer at all.
func TestMulScaledAgreesWithScaleByPow10(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())
	SetArithmeticRounding(ROUND_TOWARD_ZERO)
	SetLossPolicy(LossNaNOnInexact) // so that an inexact MulScaled is a NaN rather than a rounded value

	r := rand.New(rand.NewSource(42))
	for i := 0; i < 5000; i++ {
		a, b := randDec(r), randDec(r)
		k := r.Intn(9) - 4
		prod := a.Mul(b)
		if prod.IsNaN() {
			continue
		}
		want := prod.ScaleByPow10(k)
		got := a.MulScaled(b, k)
		if want.IsNaN() || got.IsNaN() {
			continue // the two reject different things: ScaleByPow10 is exact or nothing, MulScaled applies the rule
		}
		if !want.Equal(got) {
			t.Fatalf("%s.MulScaled(%s, %d) = %s, Mul then ScaleByPow10 = %s", a.StringFixed(), b.StringFixed(), k, got.StringFixed(), want.StringFixed())
		}
	}
}

func TestMulScaledAgainstBig(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	r := rand.New(rand.NewSource(7))
	for _, mode := range allModes {
		for _, policy := range []LossPolicy{LossRound, LossNaNOnUnderflow} {
			SetArithmeticRounding(mode)
			SetLossPolicy(policy)
			for i := 0; i < 3000; i++ {
				a, b := randDec(r), randDec(r)
				k := r.Intn(13) - 6
				exact, neg := exactProduct(a, b)
				if neg {
					exact.Neg(exact)
				}
				// the scale rule works on the exact product at the scale it stands at; a negative scale is the
				// coefficient multiplied up, at scale 0
				needed := int(a.scale) + int(b.scale) - k
				if needed < 0 {
					exact.Mul(exact, bigPow10Int(-needed))
					needed = 0
				}
				checkFit(t, "MulScaled", a, b, a.MulScaled(b, k), exact, uint8(needed), mode)
			}
		}
	}
}

func TestMulScaledRoundAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	for _, mode := range allModes {
		for i := 0; i < 4000; i++ {
			a, b := randDec(r), randDec(r)
			k := r.Intn(13) - 6
			scale := uint8(r.Intn(int(MaxScale) + 1))
			checkScaledRound(t, "MulScaledRound", a, b, k, scale, mode, a.MulScaledRound(b, k, scale, mode))
		}
	}
}

// MulPercentRound is MulDivRound by a hundred, and the point of both is that it is one rounding: the two must be
// bit-identical, at every scale and in every mode.
func TestMulPercentRoundMatchesMulDivRound(t *testing.T) {
	r := rand.New(rand.NewSource(1907))
	for _, mode := range allModes {
		for i := 0; i < 4000; i++ {
			a, b := randDec(r), randDec(r)
			scale := uint8(r.Intn(int(MaxScale) + 1))
			got := a.MulPercentRound(b, scale, mode)
			want := a.MulDivRound(b, Decimal100, scale, mode)
			if got != want {
				t.Fatalf("[%s] %s.MulPercentRound(%s, %d) = %v, MulDivRound by 100 = %v",
					mode, a.StringFixed(), b.StringFixed(), scale, got, want)
			}
		}
	}
}

func TestMulScaledExtremeK(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())
	SetArithmeticRounding(ROUND_TOWARD_ZERO)
	SetLossPolicy(LossRound)

	a, b := FromString("1.5"), FromString("2.5")

	// Far past either end the answer is settled by k alone, and it must be the one the general path would give.
	for _, k := range []int{100, 1000, 1 << 30, int(^uint(0) >> 1)} {
		if got := a.MulScaled(b, k); got.state != state.Overflow {
			t.Errorf("MulScaled(k=%d) = %v, want NaN(Overflow)", k, got)
		}
		if got := a.MulScaledRound(b, k, 2, ROUND_BANK); got.state != state.Overflow {
			t.Errorf("MulScaledRound(k=%d) = %v, want NaN(Overflow)", k, got)
		}
	}
	for _, k := range []int{-200, -1000, -(1 << 30), -int(^uint(0)>>1) - 1} {
		if got := a.MulScaled(b, k); !got.IsZero() || got.scale != MaxScale {
			t.Errorf("MulScaled(k=%d) = %s at scale %d, want zero at MaxScale", k, got.StringFixed(), got.scale)
		}
		if got := a.MulScaledRound(b, k, 2, ROUND_BANK); !got.IsZero() || got.scale != 2 {
			t.Errorf("MulScaledRound(k=%d) = %s at scale %d, want zero at scale 2", k, got.StringFixed(), got.scale)
		}
		// a directed mode still reaches the first unit of the scale, and ROUND_NAN refuses the loss
		if got := a.MulScaledRound(b, k, 2, ROUND_AWAY_FROM_ZERO); got.StringFixed() != "0.01" {
			t.Errorf("MulScaledRound(k=%d, ROUND_AWAY_FROM_ZERO) = %s, want 0.01", k, got.StringFixed())
		}
		if got := a.MulScaledRound(b, k, 2, ROUND_NAN); got.state != state.Inexact {
			t.Errorf("MulScaledRound(k=%d, ROUND_NAN) = %v, want NaN(Inexact)", k, got)
		}
		if got := a.Neg().MulScaledRound(b, k, 2, ROUND_DOWN); got.StringFixed() != "-0.01" {
			t.Errorf("MulScaledRound(k=%d, ROUND_DOWN, negative) = %s, want -0.01", k, got.StringFixed())
		}
	}

	// The boundary between the short circuit and the general path, where a product of 39 digits still leaves
	// something behind: 10^38 * 10^38 = 10^76 taken down 95 places is gone, taken down 57 places is not.
	big1 := Dec128{coef: Pow10Uint128[38]}
	if got := big1.MulScaledRound(big1, -57, MaxScale, ROUND_TOWARD_ZERO); got.IsZero() {
		t.Errorf("10^38 * 10^38 * 10^-57 = %s, want a nonzero value", got.StringFixed())
	}
	if got := big1.MulScaledRound(big1, -114, MaxScale, ROUND_TOWARD_ZERO); !got.IsZero() {
		t.Errorf("10^38 * 10^38 * 10^-114 = %s, want zero", got.StringFixed())
	}
}

// The reason the operation exists rather than being composed: the exact product is held in a wider register than a
// coefficient, so an intermediate that the result does not need cannot fail it. 3e37 at 50 per cent is NaN(Overflow)
// when the product is formed first and 1.5e37 here.
func TestMulPercentIntermediateDoesNotOverflow(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetDefaultScale(DefaultScale())
	defer SetLossPolicy(CurrentLossPolicy())
	SetArithmeticRounding(ROUND_TOWARD_ZERO)
	SetDefaultScale(MaxScale)
	SetLossPolicy(LossRound)

	a := FromString("30000000000000000000000000000000000000") // 3e37
	half := FromInt64(50)
	const want = "15000000000000000000000000000000000000"

	if composed := a.Mul(half).Div(Decimal100); composed.state != state.Overflow {
		t.Fatalf("Mul then Div = %v, want NaN(Overflow) - the premise of this test has changed", composed)
	}
	// the scale rule keeps what it can: 1.5e39 at the product's scale of 2 does not fit, so the result is that value
	// at scale 1, which is the same number
	if got := a.MulPercent(half); got.StringFixed() != want+".0" {
		t.Errorf("MulPercent = %s, want %s", got.StringFixed(), want+".0")
	}
	if got := a.MulPercentRound(half, 0, ROUND_BANK); got.StringFixed() != want {
		t.Errorf("MulPercentRound = %s, want %s", got.StringFixed(), want)
	}
}

func TestMulScaledArgumentErrors(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())
	SetArithmeticRounding(ROUND_TOWARD_ZERO)
	SetLossPolicy(LossRound)

	a, b := FromString("1.5"), FromString("2.5")
	nan1, nan2 := NaN(state.DivisionByZero), NaN(state.Underflow)

	// a NaN operand propagates, d first and then f, before any argument is looked at
	if got := nan1.MulScaledRound(nan2, 0, 200, ROUND_BANK); got.state != state.DivisionByZero {
		t.Errorf("NaN propagation: got %v", got)
	}
	if got := a.MulScaledRound(nan2, 0, 200, ROUND_BANK); got.state != state.Underflow {
		t.Errorf("NaN propagation of f: got %v", got)
	}
	if got := nan1.MulScaled(nan2, -2); got.state != state.DivisionByZero {
		t.Errorf("NaN propagation: got %v", got)
	}
	if got := a.MulPercent(nan2); got.state != state.Underflow {
		t.Errorf("NaN propagation of f: got %v", got)
	}

	if got := a.MulScaledRound(b, -2, MaxScale+1, ROUND_BANK); got.state != state.ScaleOutOfRange {
		t.Errorf("scale above MaxScale: got %v", got)
	}
	if got := a.MulPercentRound(b, MaxScale+1, ROUND_BANK); got.state != state.ScaleOutOfRange {
		t.Errorf("scale above MaxScale: got %v", got)
	}
	if got := a.MulScaledRound(b, -2, 2, RoundingMode(200)); got.state != state.InvalidRoundingMode {
		t.Errorf("undefined mode: got %v", got)
	}
	if got := a.MulPercentRound(b, 2, RoundingMode(200)); got.state != state.InvalidRoundingMode {
		t.Errorf("undefined mode: got %v", got)
	}

	// a zero operand is a zero at the requested scale, and never negative
	if got := Zero.MulPercentRound(a.Neg(), 4, ROUND_BANK); got.StringFixed() != "0.0000" || got.state == state.Neg {
		t.Errorf("zero operand: got %s, state %v", got.StringFixed(), got.state)
	}
	if got := a.Neg().MulPercent(Zero); got.state == state.Neg {
		t.Errorf("zero product is negative: %s", got.StringFixed())
	}

	// overflow: the largest value scaled up by a hundred has nowhere to go
	if got := MaxAtScale(0).MulScaled(FromInt64(1), 2); got.state != state.Overflow {
		t.Errorf("overflow: got %v", got)
	}
	if got := MaxAtScale(0).MulScaledRound(FromInt64(1), 2, 0, ROUND_BANK); got.state != state.Overflow {
		t.Errorf("overflow: got %v", got)
	}
}
