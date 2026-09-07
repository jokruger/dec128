package dec128

import (
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
