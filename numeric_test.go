package dec128

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
)

func TestSignificantAndIntegerDigits(t *testing.T) {
	for _, c := range []struct {
		in       string
		sig, int int
	}{
		{"0", 0, 0},
		{"0.00", 0, 0},
		{"5", 1, 1},
		{"0.05", 1, 0},
		{"0.5", 1, 0}, // below one, so no integer digits: the zero before the point is not one
		{"1.50", 3, 1},
		{"1.5", 2, 1},
		{"-123.45", 5, 3},
		{"0.000000000000000000", 0, 0},
		{"0.0000000000000000001", 1, 0},
		{"9999999999999999999", 19, 19},
		{"340282366920938463463374607431768211455", 39, 39},
		{"34028236692093846346.3374607431768211455", 39, 20},
		{"-0.9999999999999999999", 19, 0},
	} {
		d := FromString(c.in)
		if got := d.SignificantDigits(); got != c.sig {
			t.Errorf("SignificantDigits(%s) = %d, want %d", c.in, got, c.sig)
		}
		if got := d.IntegerDigits(); got != c.int {
			t.Errorf("IntegerDigits(%s) = %d, want %d", c.in, got, c.int)
		}
	}

	if got := NaN(state.Overflow).SignificantDigits(); got != 0 {
		t.Errorf("SignificantDigits(NaN) = %d", got)
	}
	if got := NaN(state.Overflow).IntegerDigits(); got != 0 {
		t.Errorf("IntegerDigits(NaN) = %d", got)
	}

	// Against the text, which is the definition: the digits of the coefficient, and those before the point.
	r := rand.New(rand.NewSource(20261013))
	for range 40000 {
		d := randDec(r)
		want := len(d.coef.String())
		if d.coef.IsZero() {
			want = 0
		}
		if got := d.SignificantDigits(); got != want {
			t.Fatalf("SignificantDigits(%s) = %d, want %d (coefficient %s)", d.StringFixed(), got, want, d.coef)
		}
		wantInt := max(0, want-int(d.scale))
		if got := d.IntegerDigits(); got != wantInt {
			t.Fatalf("IntegerDigits(%s) = %d, want %d", d.StringFixed(), got, wantInt)
		}
	}
}

func TestFitsNumeric(t *testing.T) {
	for _, c := range []struct {
		in         string
		p, s       uint8
		want       bool
		commentary string
	}{
		{"1.50", 3, 1, true, "a trailing zero is not a place the column has to hold"},
		{"1.50", 2, 1, true, "one integer digit and one place"},
		{"1.50", 1, 1, false, "no room for the integer digit"},
		{"1.5", 2, 1, true, ""},
		{"0.99", 2, 2, true, "the NUMERIC(2,2) case: no integer digits at all"},
		{"0.990", 2, 2, true, ""},
		{"1.00", 2, 2, false, "one integer digit, and the column has none"},
		{"0", 2, 2, true, "zero fits anything with room"},
		{"0", 1, 0, true, ""},
		{"123.45", 5, 2, true, ""},
		{"123.45", 4, 2, false, ""},
		{"123.456", 5, 2, false, "three places, and the column has two"},
		{"1234567890123456.7890", 20, 4, true, "the shape of a money column"},
		{"1234567890123456.7890", 19, 4, false, ""},
		{"0.5", 45, 19, true, "a column wider than the type"},
		{"340282366920938463463374607431768211455", 39, 0, true, ""},
		{"340282366920938463463374607431768211455", 38, 0, false, ""},
		{"1.5", 2, 3, false, "a scale above the precision is not a column"},
		{"1.5", 0, 0, false, "a precision of zero is not a column"},
	} {
		if got := FromString(c.in).FitsNumeric(c.p, c.s); got != c.want {
			t.Errorf("FitsNumeric(%s, %d, %d) = %v, want %v %s", c.in, c.p, c.s, got, c.want, c.commentary)
		}
	}

	if NaN(state.Overflow).FitsNumeric(20, 4) {
		t.Error("a NaN fits no column")
	}

	// Against the definition in math/big: the value fits when it is exact at the column's scale and the coefficient
	// there is below 10^precision.
	r := rand.New(rand.NewSource(20261014))
	for range 40000 {
		d := randDec(r)
		s := uint8(r.Intn(int(MaxScale) + 2))
		p := uint8(r.Intn(45))

		got := d.FitsNumeric(p, s)
		want := false
		if p > 0 && s <= p {
			exact := new(big.Int).Abs(bigAtScale(d, d.scale))
			if s >= d.scale {
				exact.Mul(exact, bigPow10(s-d.scale))
				want = exact.Cmp(bigPow10(p)) < 0
			} else {
				q, rem := new(big.Int).QuoRem(exact, bigPow10(d.scale-s), new(big.Int))
				want = rem.Sign() == 0 && q.Cmp(bigPow10(p)) < 0
			}
		}
		if got != want {
			t.Fatalf("FitsNumeric(%s, %d, %d) = %v, want %v", d.StringFixed(), p, s, got, want)
		}
	}
}

// The three of them together are the pre-insert check they exist for. Note which failure rounding can fix: a value
// with too many places becomes one that fits, and RescaleRoundInexact says what that cost; a value with too many
// integer digits does not fit at any scale, and no rounding to a scale will change that.
func TestNumericShapeWorkflow(t *testing.T) {
	const p, s = 19, 4 // a money column

	for _, c := range []struct {
		in      string
		fits    bool
		fixable bool
	}{
		{"1234.5", true, false},
		{"1234.56789", false, true},             // five places, and the column has four
		{"0.00005", false, true},                // below the column's quantum
		{"1234567890123456.7890", false, false}, // sixteen integer digits, and the column has fifteen
		{"340282366920938463463374607431768211455", false, false},
	} {
		d := FromString(c.in)
		if got := d.FitsNumeric(p, s); got != c.fits {
			t.Errorf("FitsNumeric(%s) = %v, want %v", c.in, got, c.fits)
			continue
		}
		if c.fits {
			// a value that fits survives the trip through the column's scale unchanged and loses nothing
			got, inexact := d.RescaleRoundInexact(s, ROUND_BANK)
			if inexact || !got.Equal(d) {
				t.Errorf("%s fits but rescaling to %d gave %s (inexact %v)", c.in, s, got.StringFixed(), inexact)
			}
			continue
		}

		got, inexact := d.RescaleRoundInexact(s, ROUND_BANK)
		// Rounding reports a loss exactly when there were places to lose, which is also exactly when it helps.
		if want := d.Canonical().scale > s; inexact != want {
			t.Errorf("%s rescaled to %d: inexact %v, want %v", c.in, s, inexact, want)
		}
		if fixed := !got.IsNaN() && got.FitsNumeric(p, s); fixed != c.fixable {
			t.Errorf("%s rescaled to %d is %s, which fits: %v, want %v", c.in, s, got.StringFixed(), fixed, c.fixable)
		}
		if !c.fixable && got.IntegerDigits() <= int(p)-int(s) && !got.IsNaN() {
			t.Errorf("%s was called unfixable but its integer part fits", c.in)
		}
	}
}

func TestIsInteger(t *testing.T) {
	for _, c := range []struct {
		in   string
		want bool
	}{
		{"0", true},
		{"0.000", true},
		{"5", true},
		{"1.000", true},
		{"1.5", false},
		{"-1.000", true},
		{"-1.5", false},
		{"0.0000000000000000001", false},
		{"340282366920938463463374607431768211455", true},
		{"34028236692093846346.3374607431768211455", false},
	} {
		if got := FromString(c.in).IsInteger(); got != c.want {
			t.Errorf("IsInteger(%s) = %v, want %v", c.in, got, c.want)
		}
	}
	if NaN(state.Overflow).IsInteger() {
		t.Error("a NaN is not an integer")
	}

	// The definition: a value is an integer when truncating it changes nothing.
	r := rand.New(rand.NewSource(20261015))
	for range 40000 {
		d := randDec(r)
		if got, want := d.IsInteger(), d.Trunc(0).Equal(d); got != want {
			t.Fatalf("IsInteger(%s) = %v, but Trunc(0) says %v", d.StringFixed(), got, want)
		}
	}
}

func TestIntFrac(t *testing.T) {
	for _, c := range []struct {
		in, ip, fp string
	}{
		{"1.25", "1", "0.25"},
		{"-1.25", "-1", "-0.25"},
		{"0.25", "0", "0.25"},
		{"-0.25", "0", "-0.25"},
		{"5", "5", "0"},
		{"-5", "-5", "0"},
		{"1.000", "1", "0.000"},
		{"-1.000", "-1", "0.000"},
		{"0", "0", "0"},
		{"34028236692093846346.3374607431768211455", "34028236692093846346", "0.3374607431768211455"},
	} {
		ip, fp := FromString(c.in).IntFrac()
		if ip.StringFixed() != c.ip || fp.StringFixed() != c.fp {
			t.Errorf("IntFrac(%s) = %s, %s; want %s, %s", c.in, ip.StringFixed(), fp.StringFixed(), c.ip, c.fp)
		}
		if ip.state == state.Neg && ip.coef.IsZero() {
			t.Errorf("IntFrac(%s): the integer part is a negative zero", c.in)
		}
		if fp.state == state.Neg && fp.coef.IsZero() {
			t.Errorf("IntFrac(%s): the fractional part is a negative zero", c.in)
		}
	}

	ip, fp := NaN(state.DomainError).IntFrac()
	if ip.state != state.DomainError || fp.state != state.DomainError {
		t.Errorf("IntFrac(NaN) = %v, %v", ip, fp)
	}

	// The post-condition: the parts add back up exactly, the integer part has no places, the fraction is below one,
	// and neither part disagrees with the sign of the value.
	r := rand.New(rand.NewSource(20261016))
	for range 40000 {
		d := randDec(r)
		ip, fp := d.IntFrac()
		if sum := ip.Add(fp); !sum.Equal(d) {
			t.Fatalf("IntFrac(%s): %s + %s = %s", d.StringFixed(), ip.StringFixed(), fp.StringFixed(), sum.StringFixed())
		}
		if ip.scale != 0 {
			t.Fatalf("IntFrac(%s): the integer part has scale %d", d.StringFixed(), ip.scale)
		}
		if fp.scale != d.scale {
			t.Fatalf("IntFrac(%s): the fractional part has scale %d", d.StringFixed(), fp.scale)
		}
		if fp.Abs().Compare(One) >= 0 {
			t.Fatalf("IntFrac(%s): the fractional part %s is not below one", d.StringFixed(), fp.StringFixed())
		}
		if !ip.IsZero() && ip.IsNegative() != d.IsNegative() {
			t.Fatalf("IntFrac(%s): the integer part has the wrong sign", d.StringFixed())
		}
		if !fp.IsZero() && fp.IsNegative() != d.IsNegative() {
			t.Fatalf("IntFrac(%s): the fractional part has the wrong sign", d.StringFixed())
		}
		if !ip.IsInteger() {
			t.Fatalf("IntFrac(%s): the integer part is not an integer", d.StringFixed())
		}
	}
}

func TestClamp(t *testing.T) {
	lo, hi := Zero, FromString("2.00")
	for _, c := range []struct {
		in, want string
	}{
		{"1.5", "1.5"},
		{"3", "2.00"},
		{"-1", "0"},
		{"0", "0"},
		{"2", "2"},
		{"2.000001", "2.00"},
		{"1.999999", "1.999999"},
	} {
		if got := FromString(c.in).Clamp(lo, hi); got.StringFixed() != c.want {
			t.Errorf("Clamp(%s, 0, 2.00) = %s, want %s", c.in, got.StringFixed(), c.want)
		}
	}

	// lo == hi pins the value
	if got := FromString("5").Clamp(One, One); !got.Equal(One) {
		t.Errorf("Clamp to a point = %v", got)
	}
	// an inverted range is refused rather than resolved
	if got := One.Clamp(FromString("2"), One); got.state != state.DomainError {
		t.Errorf("Clamp with lo above hi = %v", got)
	}
	// NaN propagates, and in argument order
	if got := NaN(state.Overflow).Clamp(NaN(state.Underflow), hi); got.state != state.Overflow {
		t.Errorf("Clamp of a NaN = %v", got)
	}
	if got := One.Clamp(NaN(state.Underflow), NaN(state.Inexact)); got.state != state.Underflow {
		t.Errorf("Clamp with a NaN lo = %v", got)
	}
	if got := One.Clamp(lo, NaN(state.Inexact)); got.state != state.Inexact {
		t.Errorf("Clamp with a NaN hi = %v", got)
	}

	// The post-condition over random ranges: the result is inside the range and is one of the three candidates.
	r := rand.New(rand.NewSource(20261017))
	for range 40000 {
		d, a, b := randDec(r), randDec(r), randDec(r)
		if a.Compare(b) > 0 {
			a, b = b, a
		}
		got := d.Clamp(a, b)
		if got.IsNaN() {
			t.Fatalf("Clamp(%s, %s, %s) = NaN(%v)", d.StringFixed(), a.StringFixed(), b.StringFixed(), got.ErrorDetails())
		}
		if got.Compare(a) < 0 || got.Compare(b) > 0 {
			t.Fatalf("Clamp(%s, %s, %s) = %s is outside the range",
				d.StringFixed(), a.StringFixed(), b.StringFixed(), got.StringFixed())
		}
		if got != d && got != a && got != b {
			t.Fatalf("Clamp(%s, %s, %s) = %s is none of its arguments",
				d.StringFixed(), a.StringFixed(), b.StringFixed(), got.StringFixed())
		}
		// and it only moves a value that was outside
		if d.Compare(a) >= 0 && d.Compare(b) <= 0 && got != d {
			t.Fatalf("Clamp moved %s, which was already inside [%s, %s]", d.StringFixed(), a.StringFixed(), b.StringFixed())
		}
	}
}
