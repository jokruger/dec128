package dec128

import (
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
)

func TestRoundDispatchMatchesMethods(t *testing.T) {
	values := []string{
		"0", "1.235", "1.245", "-1.235", "-1.245", "1.2351", "-1.2349", "2.5", "3.5", "-2.5",
		"0.005", "-0.005", "9.995", "-9.995", "123456789012345678.123456789012345678",
		"-170141183460469231731.687303715884105727", "1.000", "-1.000",
	}
	modes := []struct {
		mode RoundingMode
		fn   func(Dec128, uint8) Dec128
	}{
		{ROUND_TOWARD_ZERO, Dec128.RoundTowardZero},
		{ROUND_DOWN, Dec128.RoundDown},
		{ROUND_UP, Dec128.RoundUp},
		{ROUND_AWAY_FROM_ZERO, Dec128.RoundAwayFromZero},
		{ROUND_HALF_TOWARD_ZERO, Dec128.RoundHalfTowardZero},
		{ROUND_HALF_AWAY_FROM_ZERO, Dec128.RoundHalfAwayFromZero},
		{ROUND_BANK, Dec128.RoundBank},
	}
	for _, v := range values {
		d := FromString(v)
		if d.IsNaN() {
			t.Fatalf("bad test value %q", v)
		}
		for scale := uint8(0); scale <= 4; scale++ {
			for _, m := range modes {
				got, want := d.Round(scale, m.mode), m.fn(d, scale)
				if got != want {
					t.Errorf("%s.Round(%d, %s) = %s, want %s", v, scale, m.mode, got.StringFixed(), want.StringFixed())
				}
			}
		}
	}

	nan := NaN(state.Overflow)
	for _, m := range modes {
		if got := nan.Round(2, m.mode); got.state != state.Overflow {
			t.Errorf("NaN.Round(2, %s) must propagate the NaN, got %s", m.mode, got.ErrorDetails())
		}
	}
}

func TestRoundNaNMode(t *testing.T) {
	cases := []struct {
		in      string
		scale   uint8
		want    string
		inexact bool
	}{
		{"1.2300", 2, "1.23", false},
		{"1.2300", 4, "1.2300", false},
		{"1.2300", 6, "1.2300", false},
		{"-1.2300", 1, "", true},
		{"1.2301", 2, "", true},
		{"0.000", 1, "0.0", false},
		{"100", 0, "100", false},
	}
	for _, c := range cases {
		got := FromString(c.in).Round(c.scale, ROUND_NAN)
		switch {
		case c.inexact && got.state != state.Inexact:
			t.Errorf("%s.Round(%d, ROUND_NAN) = %v, want NaN(Inexact)", c.in, c.scale, got)
		case !c.inexact && got.StringFixed() != c.want:
			t.Errorf("%s.Round(%d, ROUND_NAN) = %s, want %s", c.in, c.scale, got.StringFixed(), c.want)
		}
	}
	if err := FromString("1.5").Round(0, ROUND_NAN).ErrorDetails(); err == nil || err.Error() != "inexact result" {
		t.Errorf("ErrorDetails for an inexact result = %v", err)
	}
}

func TestRoundInvalidMode(t *testing.T) {
	got := FromString("1.5").Round(0, RoundingMode(200))
	if got.state != state.InvalidRoundingMode {
		t.Errorf("invalid mode must yield NaN(InvalidRoundingMode), got %v", got)
	}
	if got.ErrorDetails() == nil || got.ErrorDetails().Error() != "invalid rounding mode" {
		t.Errorf("ErrorDetails = %v", got.ErrorDetails())
	}
	if RoundingMode(200).IsValid() || !ROUND_BANK.IsValid() {
		t.Error("IsValid is wrong")
	}
	if s := RoundingMode(200).String(); s != "RoundingMode(200)" {
		t.Errorf("String of undefined mode = %q", s)
	}
	if s := ROUND_HALF_AWAY_FROM_ZERO.String(); s != "ROUND_HALF_AWAY_FROM_ZERO" {
		t.Errorf("String = %q", s)
	}
}

func TestRescaleRound(t *testing.T) {
	d := FromString("1.235")
	if got := d.RescaleRound(5, ROUND_BANK); got.StringFixed() != "1.23500" || got.Scale() != 5 {
		t.Errorf("raise: got %s scale %d", got.StringFixed(), got.Scale())
	}
	if got := d.RescaleRound(3, ROUND_BANK); got != d {
		t.Errorf("same scale must return d unchanged, got %v", got)
	}
	if got := d.RescaleRound(2, ROUND_BANK); got.StringFixed() != "1.24" {
		t.Errorf("lower with bank: got %s, want 1.24", got.StringFixed())
	}
	if got := d.RescaleRound(2, ROUND_TOWARD_ZERO); got.StringFixed() != "1.23" {
		t.Errorf("lower with truncate: got %s, want 1.23", got.StringFixed())
	}
	if got := d.RescaleRound(2, ROUND_NAN); got.state != state.Inexact {
		t.Errorf("lower with ROUND_NAN must be NaN(Inexact), got %v", got)
	}
	if got := d.RescaleRound(MaxScale+1, ROUND_BANK); got.state != state.ScaleOutOfRange {
		t.Errorf("scale above MaxScale must be NaN(ScaleOutOfRange), got %v", got)
	}
	if got := NaN(state.Overflow).RescaleRound(2, ROUND_BANK); got.state != state.Overflow {
		t.Errorf("NaN must propagate, got %v", got)
	}
	// Rescale itself keeps truncating.
	if got := d.Rescale(2); got.StringFixed() != "1.23" {
		t.Errorf("Rescale must truncate, got %s", got.StringFixed())
	}
}

func TestSetArithmeticRounding(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	if ArithmeticRounding() != ROUND_TOWARD_ZERO {
		t.Fatalf("default mode = %s, want ROUND_TOWARD_ZERO", ArithmeticRounding())
	}
	SetArithmeticRounding(ROUND_BANK)
	if ArithmeticRounding() != ROUND_BANK {
		t.Errorf("mode = %s after SetArithmeticRounding(ROUND_BANK)", ArithmeticRounding())
	}
	defer func() {
		if recover() == nil {
			t.Error("SetArithmeticRounding with an undefined mode must panic")
		}
	}()
	SetArithmeticRounding(RoundingMode(99))
}

func TestRescaleRoundInexact(t *testing.T) {
	for _, c := range []struct {
		in      string
		scale   uint8
		mode    RoundingMode
		want    string
		inexact bool
	}{
		{"1.005", 2, ROUND_BANK, "1.00", true},
		{"1.500", 2, ROUND_BANK, "1.50", false},
		{"1.5", 2, ROUND_BANK, "1.50", false}, // padding is exact
		{"1.5", 4, ROUND_BANK, "1.5000", false},
		{"1.5", 1, ROUND_BANK, "1.5", false}, // the same scale is exact
		{"1.999", 2, ROUND_HALF_AWAY_FROM_ZERO, "2.00", true},
		{"-1.005", 2, ROUND_AWAY_FROM_ZERO, "-1.01", true},
		{"0.00", 0, ROUND_BANK, "0", false},
		{"1.00", 0, ROUND_BANK, "1", false},
		{"0.001", 2, ROUND_BANK, "0.00", true},
	} {
		got, inexact := FromString(c.in).RescaleRoundInexact(c.scale, c.mode)
		if got.IsNaN() || got.StringFixed() != c.want || inexact != c.inexact {
			t.Errorf("RescaleRoundInexact(%s, %d, %s) = %s, %v; want %s, %v",
				c.in, c.scale, c.mode, got.StringFixed(), inexact, c.want, c.inexact)
		}
	}

	// The value is whatever RescaleRound gives, everywhere, and the flag is the answer to
	// "does padding it back out return the original", which is the definition of having lost nothing.
	r := rand.New(rand.NewSource(20261027))
	for range 40000 {
		d := randDec(r)
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]

		got, inexact := d.RescaleRoundInexact(scale, mode)
		if want := d.RescaleRound(scale, mode); got != want {
			t.Fatalf("RescaleRoundInexact and RescaleRound disagree on %s at scale %d: %v vs %v",
				d.StringFixed(), scale, got, want)
		}
		if got.IsNaN() {
			if inexact {
				t.Fatalf("a NaN result must not be reported as inexact: %v", got)
			}
			continue
		}
		if scale >= d.scale && inexact {
			t.Fatalf("raising the scale of %s to %d is exact, but it was reported inexact", d.StringFixed(), scale)
		}
		if !inexact && !got.Rescale(d.scale).Equal(d) {
			t.Fatalf("%s at scale %d is %s, reported exact, but padding it back gives %s",
				d.StringFixed(), scale, got.StringFixed(), got.Rescale(d.scale).StringFixed())
		}
		// and when it is inexact, the discarded part really was nonzero
		if inexact {
			if _, rem, _ := d.coef.QuoRemPow10(d.scale - scale); rem == 0 {
				t.Fatalf("%s at scale %d was reported inexact but nothing was discarded", d.StringFixed(), scale)
			}
		}
	}

	// the argument errors report nothing
	for _, c := range []struct {
		what  string
		d     Dec128
		scale uint8
		mode  RoundingMode
	}{
		{"NaN", NaN(state.DomainError), 2, ROUND_BANK},
		{"scale above MaxScale", One, MaxScale + 1, ROUND_BANK},
		{"undefined mode", FromString("1.005"), 2, RoundingMode(99)},
		{"ROUND_NAN on an inexact rescale", FromString("1.005"), 2, ROUND_NAN},
		{"padding overflow", MaxAtScale(0), 1, ROUND_BANK},
	} {
		got, inexact := c.d.RescaleRoundInexact(c.scale, c.mode)
		if !got.IsNaN() || inexact {
			t.Errorf("%s = %v, inexact %v", c.what, got, inexact)
		}
	}
}
