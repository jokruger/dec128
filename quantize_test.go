package dec128

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
)

func TestRoundToPlaces(t *testing.T) {
	for _, c := range []struct {
		in     string
		places int8
		mode   RoundingMode
		want   string
	}{
		// positive places behave like Round: a shorter value is left alone
		{"1.2345", 2, ROUND_BANK, "1.23"},
		{"1.2", 4, ROUND_BANK, "1.2"},
		{"1.235", 2, ROUND_HALF_AWAY_FROM_ZERO, "1.24"},
		{"-1.235", 2, ROUND_HALF_AWAY_FROM_ZERO, "-1.24"},
		// negative places clear integer digits and land at scale 0
		{"1234", -2, ROUND_BANK, "1200"},
		{"1250", -2, ROUND_BANK, "1200"},
		{"1350", -2, ROUND_BANK, "1400"},
		{"1250", -2, ROUND_HALF_AWAY_FROM_ZERO, "1300"},
		{"-1250", -2, ROUND_HALF_AWAY_FROM_ZERO, "-1300"},
		{"1234", -4, ROUND_HALF_AWAY_FROM_ZERO, "0"},
		{"5678", -4, ROUND_HALF_AWAY_FROM_ZERO, "10000"},
		{"1234.56", -1, ROUND_TOWARD_ZERO, "1230"},
		{"-1234.56", -1, ROUND_DOWN, "-1240"},
		{"-1234.56", -1, ROUND_UP, "-1230"},
		{"1", -1, ROUND_UP, "10"},
		{"0", -5, ROUND_UP, "0"},
		{"999", -3, ROUND_HALF_AWAY_FROM_ZERO, "1000"},
	} {
		if got := FromString(c.in).RoundToPlaces(c.places, c.mode); got.String() != c.want {
			t.Errorf("RoundToPlaces(%s, %d, %s) = %s, want %s", c.in, c.places, c.mode, got.String(), c.want)
		}
	}

	// the grid can be coarser than the type: everything rounds to zero, or asks for a multiple that does not fit
	for _, c := range []struct {
		places int8
		mode   RoundingMode
		want   string
		st     state.State
	}{
		{-39, ROUND_HALF_AWAY_FROM_ZERO, "0", state.Default},
		{-100, ROUND_TOWARD_ZERO, "0", state.Default},
		{-39, ROUND_UP, "", state.Overflow},
		{-39, ROUND_AWAY_FROM_ZERO, "", state.Overflow},
		{-39, ROUND_NAN, "", state.Inexact},
	} {
		got := One.RoundToPlaces(c.places, c.mode)
		if got.state != c.st || (c.st < state.Error && got.String() != c.want) {
			t.Errorf("1.RoundToPlaces(%d, %s) = %v (state %s), want %q/%s", c.places, c.mode, got, got.state, c.want, c.st)
		}
	}
	// 10^38 is the coarsest grid a coefficient can land on
	if got := MaxAtScale(0).RoundToPlaces(-38, ROUND_HALF_AWAY_FROM_ZERO); got.String() != "300000000000000000000000000000000000000" {
		t.Errorf("rounding to 10^38 = %s", got.String())
	}
	if got := MaxAtScale(0).RoundToPlaces(-38, ROUND_UP); got.state != state.Overflow {
		t.Errorf("rounding up to 10^38 = %v, want NaN(Overflow)", got)
	}

	// argument errors and NaN
	if got := One.RoundToPlaces(2, RoundingMode(99)); got.state != state.InvalidRoundingMode {
		t.Errorf("bad mode = %v", got)
	}
	if got := NaN(state.NotConverged).RoundToPlaces(-2, ROUND_BANK); got.state != state.NotConverged {
		t.Errorf("NaN propagation = %v", got)
	}
	if got := FromString("1.5").RoundToPlaces(-1, ROUND_NAN); got.state != state.Inexact {
		t.Errorf("ROUND_NAN on an inexact grid = %v", got)
	}
	if got := FromString("20").RoundToPlaces(-1, ROUND_NAN); got.String() != "20" {
		t.Errorf("ROUND_NAN on an exact grid = %v", got)
	}
}

// RoundToPlaces with negative places must agree with the general grid, and both with math/big.
func TestRoundToPlacesAndMultipleAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260929))
	for range 40000 {
		d := randDec(r)
		places := int8(r.Intn(45) - 25)
		mode := allModes[r.Intn(len(allModes))]

		got := d.RoundToPlaces(places, mode)
		if places >= 0 {
			if want := d.Round(min(uint8(places), d.scale), mode); got != want {
				t.Fatalf("RoundToPlaces(%s, %d, %s) = %v, want %v", d.StringFixed(), places, mode, got, want)
			}
			continue
		}

		// the exact value at scale 0 on a grid of 10^n, in math/big
		n := -int(places)
		exact := bigAtScale(d, d.scale)
		div := new(big.Int).Mul(bigPow10(d.scale), bigPow10(uint8(min(n, 2*int(MaxScale)))))
		if n > 2*int(MaxScale) {
			continue // the grid is past the type; covered by the table above
		}
		q, rem := new(big.Int).QuoRem(new(big.Int).Abs(exact), div, new(big.Int))
		q = roundBig(q, rem, div, exact.Sign() < 0, mode)
		want := new(big.Int).Mul(q, bigPow10(uint8(n)))

		switch {
		case mode == ROUND_NAN && rem.Sign() != 0:
			if got.state != state.Inexact {
				t.Fatalf("RoundToPlaces(%s, %d, ROUND_NAN) = %v, want NaN(Inexact)", d.StringFixed(), places, got)
			}
		case want.Cmp(big2p128) >= 0:
			if got.state != state.Overflow {
				t.Fatalf("RoundToPlaces(%s, %d, %s) = %v, want NaN(Overflow)", d.StringFixed(), places, mode, got)
			}
		case got.IsNaN():
			t.Fatalf("RoundToPlaces(%s, %d, %s) = NaN(%v), want %s", d.StringFixed(), places, mode, got.ErrorDetails(), want)
		default:
			if got.scale != 0 || new(big.Int).Abs(bigAtScale(got, 0)).Cmp(want) != 0 {
				t.Fatalf("RoundToPlaces(%s, %d, %s) = %s, want %s", d.StringFixed(), places, mode, got.StringFixed(), want)
			}
			if wantNeg := exact.Sign() < 0 && want.Sign() != 0; (got.state == state.Neg) != wantNeg {
				t.Fatalf("RoundToPlaces(%s, %d, %s) = %s: wrong sign", d.StringFixed(), places, mode, got.StringFixed())
			}
		}
	}
}

func TestRoundToMultiple(t *testing.T) {
	for _, c := range []struct {
		in, m string
		mode  RoundingMode
		want  string
	}{
		// Swiss cash rounding
		{"2.37", "0.05", ROUND_HALF_AWAY_FROM_ZERO, "2.35"},
		{"2.38", "0.05", ROUND_HALF_AWAY_FROM_ZERO, "2.40"},
		{"2.375", "0.05", ROUND_HALF_AWAY_FROM_ZERO, "2.40"},
		{"2.375", "0.05", ROUND_HALF_TOWARD_ZERO, "2.35"},
		{"2.375", "0.05", ROUND_BANK, "2.40"},
		{"2.425", "0.05", ROUND_BANK, "2.40"},
		{"-2.375", "0.05", ROUND_HALF_AWAY_FROM_ZERO, "-2.40"},
		// the next whole ten, as retail lending states it
		{"183.47", "10", ROUND_UP, "190"},
		{"183.47", "10", ROUND_DOWN, "180"},
		{"-183.47", "10", ROUND_UP, "-180"},
		{"-183.47", "10", ROUND_AWAY_FROM_ZERO, "-190"},
		// already a multiple
		{"2.40", "0.05", ROUND_NAN, "2.40"},
		{"0", "0.05", ROUND_BANK, "0.00"},
		// the result takes the multiple's scale, because it is a whole number of them
		{"7", "0.25", ROUND_BANK, "7.00"},
		{"7.6", "0.25", ROUND_BANK, "7.50"},
	} {
		if got := FromString(c.in).RoundToMultiple(FromString(c.m), c.mode); got.StringFixed() != c.want {
			t.Errorf("RoundToMultiple(%s, %s, %s) = %s, want %s", c.in, c.m, c.mode, got.StringFixed(), c.want)
		}
	}

	for _, c := range []struct {
		what string
		got  Dec128
		want state.State
	}{
		{"zero multiple", One.RoundToMultiple(Zero, ROUND_BANK), state.DomainError},
		{"negative multiple", One.RoundToMultiple(FromString("-0.05"), ROUND_BANK), state.DomainError},
		{"bad mode", One.RoundToMultiple(One, RoundingMode(99)), state.InvalidRoundingMode},
		{"NaN d", NaN(state.NotConverged).RoundToMultiple(One, ROUND_BANK), state.NotConverged},
		{"NaN m", One.RoundToMultiple(NaN(state.Underflow), ROUND_BANK), state.Underflow},
		{"not a multiple under ROUND_NAN", FromString("2.37").RoundToMultiple(FromString("0.05"), ROUND_NAN), state.Inexact},
		{"result too wide", MaxAtScale(0).RoundToMultiple(FromString("0.5"), ROUND_UP), state.Overflow},
		// The count of multiples is the largest a coefficient holds and the remainder rounds it up, so the count
		// itself carries out even though the product of the truncated count with the multiple would have fit.
		{"count carries out", FromString("238197656844656924424362225202237748019").
			RoundToMultiple(FromString("0.7"), ROUND_HALF_AWAY_FROM_ZERO), state.Overflow},
	} {
		if c.got.state != c.want {
			t.Errorf("%s = %v, want NaN(%s)", c.what, c.got, c.want)
		}
	}
}

// RoundToMultiple must be Round(d/m)*m with a single rounding, which is what dividing exactly and rounding the
// integer quotient gives; math/big says what that is.
func TestRoundToMultipleAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260930))
	for range 40000 {
		d := randDec(r)
		m := randDec(r).Abs()
		if m.coef.IsZero() {
			continue
		}
		mode := allModes[r.Intn(len(allModes))]
		got := d.RoundToMultiple(m, mode)

		s := max(d.scale, m.scale)
		bd, bm := bigAtScale(d, s), bigAtScale(m, s)
		trunc, rem := new(big.Int).QuoRem(new(big.Int).Abs(bd), bm, new(big.Int))
		if trunc.Cmp(big2p128) >= 0 {
			// the count of multiples does not fit a coefficient, which is reported before anything else
			if got.state != state.Overflow {
				t.Fatalf("RoundToMultiple(%s, %s, %s) = %v, want NaN(Overflow) for a count of %s",
					d.StringFixed(), m.StringFixed(), mode, got, trunc)
			}
			continue
		}
		q := roundBig(trunc, rem, bm, bd.Sign() < 0, mode)
		// the result is that many multiples, at the multiple's own scale
		want := new(big.Int).Mul(q, bigAtScale(m, m.scale))

		switch {
		case q.Cmp(big2p128) >= 0:
			if got.state != state.Overflow {
				t.Fatalf("RoundToMultiple(%s, %s, %s) = %v, want NaN(Overflow)", d.StringFixed(), m.StringFixed(), mode, got)
			}
		case mode == ROUND_NAN && rem.Sign() != 0:
			if got.state != state.Inexact {
				t.Fatalf("RoundToMultiple(%s, %s, ROUND_NAN) = %v", d.StringFixed(), m.StringFixed(), got)
			}
		case want.Cmp(big2p128) >= 0:
			if got.state != state.Overflow {
				t.Fatalf("RoundToMultiple(%s, %s, %s) = %v, want NaN(Overflow)", d.StringFixed(), m.StringFixed(), mode, got)
			}
		case got.IsNaN():
			t.Fatalf("RoundToMultiple(%s, %s, %s) = NaN(%v), want %s", d.StringFixed(), m.StringFixed(), mode, got.ErrorDetails(), want)
		default:
			if got.scale != m.scale || new(big.Int).Abs(bigAtScale(got, got.scale)).Cmp(want) != 0 {
				t.Fatalf("RoundToMultiple(%s, %s, %s) = %s at scale %d, want %s at scale %d",
					d.StringFixed(), m.StringFixed(), mode, got.StringFixed(), got.scale, want, m.scale)
			}
			if wantNeg := bd.Sign() < 0 && want.Sign() != 0; (got.state == state.Neg) != wantNeg {
				t.Fatalf("RoundToMultiple(%s, %s, %s) = %s: wrong sign", d.StringFixed(), m.StringFixed(), mode, got.StringFixed())
			}
		}
	}
}

func TestScaleByPow10(t *testing.T) {
	for _, c := range []struct {
		in   string
		k    int
		want string
	}{
		{"5.25", -2, "0.0525"}, // percent to fraction
		{"25", -4, "0.0025"},   // basis points to fraction
		{"0.0525", 2, "5.25"},  // and back
		{"1.5", 1, "15"},       // the scale runs out, so the coefficient grows
		{"1.5", 2, "150"},
		{"1", 3, "1000"},
		{"1", 0, "1"},
		{"0", 5, "0"},
		{"-1.5", -1, "-0.15"},
		{"1200", -21, "0.0000000000000000012"}, // past MaxScale, but the low digits are zeros
	} {
		got := FromString(c.in).ScaleByPow10(c.k)
		if got.IsNaN() || got.String() != c.want {
			t.Errorf("ScaleByPow10(%s, %d) = %v, want %s", c.in, c.k, got, c.want)
		}
	}

	for _, c := range []struct {
		what string
		got  Dec128
		want state.State
	}{
		{"grown past the coefficient", MaxAtScale(0).ScaleByPow10(1), state.Overflow},
		{"grown far past it", One.ScaleByPow10(39), state.Overflow},
		{"needs places that do not exist", One.ScaleByPow10(-20), state.Underflow},
		{"shrunk far past the quantum", One.ScaleByPow10(-100), state.Underflow},
		{"NaN", NaN(state.NotConverged).ScaleByPow10(2), state.NotConverged},
	} {
		if c.got.state != c.want {
			t.Errorf("%s = %v, want NaN(%s)", c.what, c.got, c.want)
		}
	}

	// exactness and the round trip, against math/big
	r := rand.New(rand.NewSource(20261001))
	for range 40000 {
		d := randDec(r)
		k := r.Intn(90) - 45
		got := d.ScaleByPow10(k)
		if got.IsNaN() {
			continue
		}
		want := new(big.Rat).SetFrac(bigAtScale(d, d.scale), bigPow10(d.scale))
		p := new(big.Rat).SetFrac(bigPow10(uint8(max(k, -k))), big.NewInt(1))
		if k >= 0 {
			want.Mul(want, p)
		} else {
			want.Quo(want, p)
		}
		gotRat := new(big.Rat).SetFrac(bigAtScale(got, got.scale), bigPow10(got.scale))
		if gotRat.Cmp(want) != 0 {
			t.Fatalf("ScaleByPow10(%s, %d) = %s, want %s", d.StringFixed(), k, got.StringFixed(), want.FloatString(40))
		}
	}
}

func TestDivRoundInexact(t *testing.T) {
	for _, c := range []struct {
		a, b    string
		scale   uint8
		want    string
		inexact bool
	}{
		{"1", "2", 2, "0.50", false},
		{"1", "3", 2, "0.33", true},
		{"10", "5", 0, "2", false},
		{"0", "3", 2, "0.00", false},
		{"1", "3", 19, "0.3333333333333333333", true},
	} {
		got, inexact := FromString(c.a).DivRoundInexact(FromString(c.b), c.scale, ROUND_TOWARD_ZERO)
		if got.StringFixed() != c.want || inexact != c.inexact {
			t.Errorf("DivRoundInexact(%s, %s, %d) = %s, %v; want %s, %v", c.a, c.b, c.scale, got.StringFixed(), inexact, c.want, c.inexact)
		}
	}

	// it must agree with DivRound everywhere, and never report a rounding it did not take
	r := rand.New(rand.NewSource(20261002))
	for range 40000 {
		x, y := randDec(r), randDec(r)
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]
		got, inexact := x.DivRoundInexact(y, scale, mode)
		if want := x.DivRound(y, scale, mode); got != want {
			t.Fatalf("DivRoundInexact and DivRound disagree: %v vs %v", got, want)
		}
		if got.IsNaN() && inexact {
			t.Fatalf("a NaN result must not be reported as inexact: %v", got)
		}
		if !got.IsNaN() && !inexact {
			// an exact quotient multiplied back must give the dividend
			if back := got.MulRound(y, max(got.scale+y.scale, 0), ROUND_TOWARD_ZERO); !back.IsNaN() && !back.Equal(x) {
				t.Fatalf("%s / %s = %s reported exact, but multiplying back gives %s",
					x.StringFixed(), y.StringFixed(), got.StringFixed(), back.StringFixed())
			}
		}
	}

	// the argument errors report nothing
	for _, c := range []struct {
		what  string
		a, b  Dec128
		scale uint8
		mode  RoundingMode
	}{
		{"scale", One, One, MaxScale + 1, ROUND_BANK},
		{"mode", One, One, 2, RoundingMode(99)},
		{"division by zero", One, Zero, 2, ROUND_BANK},
		{"NaN", NaN(state.NotConverged), One, 2, ROUND_BANK},
		{"ROUND_NAN", One, FromInt64(3), 2, ROUND_NAN},
	} {
		got, inexact := c.a.DivRoundInexact(c.b, c.scale, c.mode)
		if !got.IsNaN() || inexact {
			t.Errorf("%s = %v, inexact %v", c.what, got, inexact)
		}
	}
}
