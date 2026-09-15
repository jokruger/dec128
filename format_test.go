package dec128

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/jokruger/dec128/state"
)

func TestFormatVerbs(t *testing.T) {
	defer SetTrimOutput(TrimOutput())

	d := FromString("1.50")
	neg := FromString("-1.005")
	nan := NaN(state.Overflow)

	for _, c := range []struct {
		format string
		v      Dec128
		want   string
	}{
		// the string verbs, which must keep behaving as they did before Format existed
		{"%v", d, "1.5"},
		{"%s", d, "1.5"},
		{"%v", FromString("0.000"), "0"},
		{"%s", neg, "-1.005"},

		// the fixed verb: the value's own scale without a precision, exactly the precision with one
		{"%f", d, "1.50"},
		{"%F", d, "1.50"},
		{"%f", FromString("1.5"), "1.5"},
		{"%.2f", d, "1.50"},
		{"%.4f", d, "1.5000"},
		{"%.0f", d, "2"},
		{"%.1f", d, "1.5"},
		{"%.2f", neg, "-1.00"},
		{"%.3f", neg, "-1.005"},
		{"%.0f", neg, "-1"},
		{"%.1f", FromString("0"), "0.0"},
		{"%.25f", d, "1.5000000000000000000"}, // clamped to MaxScale
		{"%.2f", MaxAtScale(0), "NaN"},        // padding the largest coefficient does not fit

		// half to even, which is what Go's own %f does to a float64
		{"%.2f", FromString("1.005"), "1.00"},
		{"%.2f", FromString("1.015"), "1.02"},
		{"%.2f", FromString("2.665"), "2.66"},
		{"%.2f", FromString("2.675"), "2.68"},
		{"%.0f", FromString("0.5"), "0"},
		{"%.0f", FromString("1.5"), "2"},
		{"%.0f", FromString("-0.5"), "0"},

		// scientific
		{"%e", d, "1.5e+0"},
		{"%E", d, "1.5E+0"},
		{"%e", FromString("12345"), "1.2345e+4"},
		{"%E", FromString("-0.000123"), "-1.23E-4"},
		{"%e", Zero, "0e+0"},

		// the integer verb truncates toward zero
		{"%d", d, "1"},
		{"%d", neg, "-1"},
		{"%d", FromString("-1.9"), "-1"},
		{"%d", FromString("0.9"), "0"},

		// width and the flags
		{"%8.2f", d, "    1.50"},
		{"%-8.2f|", d, "1.50    |"},
		{"%08.2f", d, "00001.50"},
		{"%08.2f", neg, "-0001.00"},
		{"%+.2f", d, "+1.50"},
		{"%+.2f", neg, "-1.00"},
		{"%+d", d, "+1"},
		{"%+v", d, "+1.5"},
		{"%10v", d, "       1.5"},
		{"%-10v|", d, "1.5       |"},
		{"%+10.2f", d, "     +1.50"},
		{"%+010.2f", d, "+000001.50"},
		{"%2.2f", d, "1.50"}, // a width the value already exceeds
		{"%08e", d, "001.5e+0"},

		// a NaN is NaN under every verb, ignores the precision and the sign, and pads with spaces even under '0'
		{"%v", nan, "NaN"},
		{"%s", nan, "NaN"},
		{"%f", nan, "NaN"},
		{"%.2f", nan, "NaN"},
		{"%e", nan, "NaN"},
		{"%d", nan, "NaN"},
		{"%+v", nan, "NaN"},
		{"%8.2f", nan, "     NaN"},
		{"%08.2f", nan, "     NaN"},
		{"%-8v|", nan, "NaN     |"},
		{"%v", Null(), "NaN"},

		// an unsupported verb reports itself the way fmt does
		{"%x", d, "%!x(dec128.Dec128=1.5)"},
		{"%q", d, "%!q(dec128.Dec128=1.5)"},
		{"%t", nan, "%!t(dec128.Dec128=NaN)"},

		// Go syntax is an expression that rebuilds the value
		{"%#v", d, `dec128.FromString("1.50")`},
		{"%#v", neg, `dec128.FromString("-1.005")`},
	} {
		if got := fmt.Sprintf(c.format, c.v); got != c.want {
			t.Errorf("Sprintf(%q, %s) = %q, want %q", c.format, c.v.StringFixed(), got, c.want)
		}
	}
}

// A Dec128 inside a struct or a slice still prints through Format, which is what keeps existing log lines working.
func TestFormatNested(t *testing.T) {
	type row struct {
		Amount Dec128
		Rate   Dec128
	}
	r := row{FromString("1234.50"), FromString("0.0525")}

	for _, c := range []struct{ format, want string }{
		{"%v", "{1234.5 0.0525}"},
		{"%+v", "{Amount:+1234.5 Rate:+0.0525}"},
		{"%s", "{1234.5 0.0525}"},
	} {
		if got := fmt.Sprintf(c.format, r); got != c.want {
			t.Errorf("Sprintf(%q, struct) = %q, want %q", c.format, got, c.want)
		}
	}

	if got := fmt.Sprintf("%v", []Dec128{One, FromString("2.50")}); got != "[1 2.5]" {
		t.Errorf("slice = %q", got)
	}
	if got := fmt.Sprintf("%.2f", r.Amount); got != "1234.50" {
		t.Errorf("field = %q", got)
	}
}

// The string verbs must agree with the methods they stand for, whatever the global output setting is, because that is
// the whole of their contract.
func TestFormatAgreesWithTheMethods(t *testing.T) {
	defer SetTrimOutput(TrimOutput())

	r := rand.New(rand.NewSource(20261018))
	for _, trim := range []bool{false, true} {
		SetTrimOutput(trim)
		for range 20000 {
			d := randDec(r)
			if got := fmt.Sprintf("%v", d); got != d.String() {
				t.Fatalf("%%v = %q, String = %q (trim %v)", got, d.String(), trim)
			}
			if got := fmt.Sprintf("%f", d); got != d.StringFixed() {
				t.Fatalf("%%f = %q, StringFixed = %q (trim %v)", got, d.StringFixed(), trim)
			}
			if got := fmt.Sprintf("%e", d); got != d.StringSci() {
				t.Fatalf("%%e = %q, StringSci = %q (trim %v)", got, d.StringSci(), trim)
			}
			if got := fmt.Sprintf("%d", d); got != d.Trunc(0).String() {
				t.Fatalf("%%d = %q, Trunc(0) = %q (trim %v)", got, d.Trunc(0).String(), trim)
			}

			scale := uint8(r.Intn(int(MaxScale) + 1))
			want := d.RescaleRound(scale, ROUND_BANK).StringFixed()
			if got := fmt.Sprintf("%.*f", int(scale), d); got != want {
				t.Fatalf("%%.%df = %q, RescaleRound = %q", scale, got, want)
			}
		}
	}
}

// Width is honoured by padding, never by truncating, and the padding goes on the side the flags ask for.
func TestFormatWidth(t *testing.T) {
	d := FromString("-1.50")
	for w := 0; w <= 12; w++ {
		right := fmt.Sprintf("%*.2f", w, d)
		left := fmt.Sprintf("%-*.2f", w, d)
		zero := fmt.Sprintf("%0*.2f", w, d)

		for _, got := range []string{right, left, zero} {
			if len(got) != max(w, len("-1.50")) {
				t.Fatalf("width %d: %q has length %d", w, got, len(got))
			}
			if strings.TrimLeft(strings.TrimRight(got, " "), " 0-") != "1.50" {
				t.Fatalf("width %d: %q does not carry the value", w, got)
			}
		}
		if w > 5 {
			if right[0] != ' ' || left[0] != '-' || zero[0] != '-' {
				t.Fatalf("width %d: right %q, left %q, zero %q", w, right, left, zero)
			}
			// zero padding goes between the sign and the digits
			if zero[1] != '0' {
				t.Fatalf("width %d: zero padding before the sign: %q", w, zero)
			}
		}
	}
}
