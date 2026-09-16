package dec128

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/jokruger/dec128/state"
)

// A value this type can hold exactly must not be rejected because of how it was written. Trailing zeros are padding,
// not information: the parsers give up the smallest number of them needed, the same reduction DecodePgNumeric,
// DecodeInt128, DecodeIEEE and applyExp perform on import. These tests are the regression net for that, and for the
// silent-NaN path it used to open - the reason carried was ScaleOutOfRange or Overflow rather than InvalidFormat, so
// Scan, UnmarshalJSON and UnmarshalText returned a nil error and handed back a NaN.

func TestPaddedInputIsAccepted(t *testing.T) {
	accept := []struct{ in, want string }{
		// more written places than MaxScale, the surplus all zeros
		{"1.50000000000000000000", "1.5"},
		{"0.00000000000000000000", "0"},
		{"0.0000000000000000000000000000000000000000000", "0"},
		{"123.45000000000000000000", "123.45"},
		// the scale is in range but the coefficient does not fit at it
		{"340282366920938463463.3746074317682114550", "340282366920938463463.374607431768211455"}, // MaxAtScale(18) plus one zero
		{"100000000000000000000.0000000000000000000", "100000000000000000000"},
		{"123456789012345678901234567890.1234567890", "123456789012345678901234567890.123456789"},
		// the scientific path: padding in the fraction, and padding in the integer part that the exponent absorbs
		{"1234567890123456789012345678901234567890e-2", "12345678901234567890123456789012345678.9"},
		{"9510064090660540200010051006500084882000e-2", "95100640906605402000100510065000848820"},
		{"516050697300749091019000860024085230210.0e-3", "516050697300749091019000860024085230.21"},
		{"1.5000000000000000000000000000000000000000e5", "150000"},
	}
	for _, c := range accept {
		got := FromString(c.in)
		if got.IsNaN() {
			t.Errorf("FromString(%q) = NaN(%v), want %s", c.in, got.ErrorDetails(), c.want)
			continue
		}
		if !got.Equal(FromString(c.want)) {
			t.Errorf("FromString(%q) = %s, want %s", c.in, got.StringFixed(), c.want)
		}
		// the same input through the paths that route to FromString must not error either
		var d Dec128
		if err := d.Scan(c.in); err != nil || d.IsNaN() {
			t.Errorf("Scan(%q): err %v, value %s", c.in, err, d.StringFixed())
		}
		if err := d.UnmarshalText([]byte(c.in)); err != nil || d.IsNaN() {
			t.Errorf("UnmarshalText(%q): err %v, value %s", c.in, err, d.StringFixed())
		}
	}

	// Padding that already fits keeps the scale it was written with, so 1.50 is not silently canonicalized.
	for _, c := range []struct{ in, want string }{
		{"1.50", "1.50"},
		{"1.50e0", "1.50"},
		{"1.5000000000000000000", "1.5000000000000000000"}, // exactly MaxScale places
	} {
		if got := FromString(c.in).StringFixed(); got != c.want {
			t.Errorf("FromString(%q) = %s, want %s", c.in, got, c.want)
		}
	}

	// And a value that genuinely needs more than MaxScale places is still refused.
	reject := []struct {
		in string
		st state.State
	}{
		{"1.50000000000000000001", state.ScaleOutOfRange},
		{"0.00000000000000000001", state.ScaleOutOfRange},
		{"1.000000000000000000000000000000001", state.ScaleOutOfRange},
		{"340282366920938463463374607431768211455.5", state.Overflow},
		{"340282366920938463463374607431768211456", state.Overflow},
		{"1.99999999999999999999999999999999999999999e5", state.Overflow},
	}
	for _, c := range reject {
		if got := FromString(c.in); got.state != c.st {
			t.Errorf("FromString(%q) = %v (%s), want %s", c.in, got, got.state, c.st)
		}
	}
}

// TestPaddedInputMalformed covers what the reduction does with input that is not a well-formed decimal. The padding
// retry re-reads digits the fast path never validated, so it is the first place a bad character in a long fraction is
// noticed - which makes it an InvalidFormat, the one reason Scan and the Unmarshal methods report as an error.
func TestPaddedInputMalformed(t *testing.T) {
	cases := []struct {
		in string
		st state.State
	}{
		// the integer part does not fit however much of the fraction goes
		{"340282366920938463463374607431768211456.00000000000000000000", state.Overflow},
		// a bad character among the digits that would be kept
		{"1.x0000000000000000000", state.InvalidFormat},
		// a bad character in a scientific mantissa, reached through the exponent marker
		{"1#00000000000000000000e5", state.InvalidFormat},
		// a bad character among the digits that have to go: the scale cannot come down
		{"1.0000000000000000000x0", state.ScaleOutOfRange},
	}
	for _, c := range cases {
		if got := FromString(c.in); got.state != c.st {
			t.Errorf("FromString(%q) = %s, want %s", c.in, got.state, c.st)
		}
	}

	// An invalid format is the only reason these report as an error; the others are values.
	var d Dec128
	if err := d.Scan("1.x0000000000000000000"); err == nil {
		t.Error("Scan of a malformed long form returned no error")
	}
	if err := d.Scan("340282366920938463463374607431768211456.00000000000000000000"); err != nil {
		t.Errorf("Scan of an out-of-range value returned %v, want a NaN value and no error", err)
	} else if !d.IsNaN() {
		t.Errorf("Scan of an out-of-range value = %s, want NaN", d.StringFixed())
	}
}

// TestFromStringRejectsOnlyUnrepresentable is the exhaustive form: for random well-formed literals, FromString must
// accept exactly those values the type can hold, and the value it returns must be the one that was written. This is
// the check that found the padding defect, so it stays.
func TestFromStringRejectsOnlyUnrepresentable(t *testing.T) {
	// representable reports whether v can be held exactly: some scale up to MaxScale makes it an integer that fits
	// in 128 bits.
	representable := func(v *big.Rat) bool {
		for s := 0; s <= int(MaxScale); s++ {
			scaled := new(big.Rat).Mul(v, new(big.Rat).SetInt(bigPow10(uint8(s))))
			if scaled.IsInt() {
				return new(big.Int).Abs(scaled.Num()).Cmp(big2p128) < 0
			}
		}
		return false
	}

	check := func(s string) {
		t.Helper()
		want, ok := new(big.Rat).SetString(s)
		if !ok {
			t.Fatalf("the generator produced a literal big.Rat rejects: %q", s)
		}
		d := FromString(s)
		switch {
		case d.IsNaN() && representable(want):
			t.Fatalf("FromString(%q) = NaN(%v) but %s is representable", s, d.ErrorDetails(), want.FloatString(25))
		case !d.IsNaN() && !representable(want):
			t.Fatalf("FromString(%q) = %s but %s is not representable", s, d.StringFixed(), want.FloatString(45))
		case d.IsNaN():
			return
		}
		if got := new(big.Rat).SetFrac(exactCoef(d), bigPow10(d.scale)); got.Cmp(want) != 0 {
			t.Fatalf("FromString(%q) = %s, want %s", s, got.RatString(), want.RatString())
		}
	}

	r := rand.New(rand.NewSource(20260921))
	for range 60000 {
		digits := 1 + r.Intn(46)
		var sb strings.Builder
		if r.Intn(2) == 0 {
			sb.WriteByte('-')
		}
		point := r.Intn(digits + 1)
		for i := range digits {
			if i == point && i != 0 {
				sb.WriteByte('.')
			}
			if r.Intn(3) == 0 {
				sb.WriteByte('0') // bias toward padding, which is what the reduction is about
			} else {
				sb.WriteByte(byte('0' + r.Intn(10)))
			}
		}
		s := sb.String()
		check(s)
		if r.Intn(2) == 0 {
			check(s + fmt.Sprintf("e%+d", r.Intn(80)-40))
		}
		// FromSafeString applies the same reduction, so on well-formed regular-form input the two must agree.
		if d, f := FromString(s), FromSafeString(s); d != f {
			t.Fatalf("FromString(%q) = %v, FromSafeString = %v", s, d, f)
		}
	}
}

// TestBinaryZeroKeepsScale pins the one place the package's own format used to differ from every other encoder: a zero
// now carries its scale, in the presence bit the format already had, so a decoder of any version reads it.
func TestBinaryZeroKeepsScale(t *testing.T) {
	z := FromString("0.00")
	if z.IsNaN() || z.Scale() != 2 {
		t.Fatalf("0.00 parsed as %v", z)
	}

	b, err := z.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	// flag byte with only the scale-presence bit, then the scale
	if want := []byte{0b0010_0000, 2}; string(b) != string(want) {
		t.Errorf("MarshalBinary(0.00) = %v, want %v", b, want)
	}
	if n := z.BinarySize(); n != len(b) {
		t.Errorf("BinarySize = %d, MarshalBinary wrote %d", n, len(b))
	}

	var back Dec128
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if back != z {
		t.Errorf("round trip of 0.00 = %v (scale %d), want scale 2", back, back.Scale())
	}

	// A zero at scale 0 and a NaN stay one byte: neither has a scale worth recording.
	for _, d := range []Dec128{Zero, NaN(state.Overflow), Null()} {
		b, _ := d.MarshalBinary()
		if len(b) != 1 || d.BinarySize() != 1 {
			t.Errorf("%v: encoded to %d bytes, BinarySize %d, want 1", d, len(b), d.BinarySize())
		}
	}

	// Every other encoder already agreed; check they still do, so the four cannot drift apart again.
	for _, s := range []string{"0", "0.0", "0.00", "0.0000000000000000000"} {
		d := FromString(s)
		var bin, pg, ieee, i128 Dec128

		bb, _ := d.MarshalBinary()
		if err := bin.UnmarshalBinary(bb); err != nil || bin != d {
			t.Errorf("%s: binary round trip = %v, %v", s, bin, err)
		}
		pb, _ := d.AppendPgNumeric(nil)
		if err := pg.DecodePgNumeric(pb); err != nil || pg != d {
			t.Errorf("%s: pg round trip = %v, %v", s, pg, err)
		}
		eb, _ := d.AppendIEEE(nil)
		if err := ieee.DecodeIEEE(eb); err != nil || ieee != d {
			t.Errorf("%s: ieee round trip = %v, %v", s, ieee, err)
		}
		ib, _ := d.AppendInt128(nil, d.Scale(), binary.BigEndian)
		if err := i128.DecodeInt128(ib, d.Scale(), binary.BigEndian); err != nil || i128 != d {
			t.Errorf("%s: int128 round trip = %v, %v", s, i128, err)
		}
		if v, _ := d.Value(); v != s {
			t.Errorf("%s: Value = %v", s, v)
		}
	}
}

// TestGlobalConfigurationRestored is the canary for the process-global settings. Every test that changes one restores
// it, so whenever this runs the four must be at their package defaults; run the suite with -shuffle=on and a test that
// forgets will be caught here.
func TestGlobalConfigurationRestored(t *testing.T) {
	if got := DefaultScale(); got != MaxScale {
		t.Errorf("DefaultScale = %d, want %d: a test changed it without restoring it", got, MaxScale)
	}
	if got := ArithmeticRounding(); got != ROUND_TOWARD_ZERO {
		t.Errorf("ArithmeticRounding = %v, want ROUND_TOWARD_ZERO: a test changed it without restoring it", got)
	}
	if TrimOutput() {
		t.Error("TrimOutput is set: a test changed it without restoring it")
	}
	if got := NullValue(); got != Zero {
		t.Errorf("NullValue = %v, want Zero: a test changed it without restoring it", got)
	}
}
