package dec128

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
)

// Tests for the operations added alongside the read-across of the General Decimal Arithmetic specification: the
// math/big bridge, the decomposer codec, the five small GDA operations, the two logarithms and the fused
// add-and-divide, and the product family.
//
// Where a case is an example from the specification the comment says so. The expected values for everything else are
// derived here from math/big, in the manner the rest of this suite uses: an oracle, not a hand-computed number.

// ---------------------------------------------------------------------------------------------------------------
// The math/big bridge

func TestRatIsExactAndSigned(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1.75", "7/4"},
		{"-1.75", "-7/4"},
		{"0", "0"},
		{"0.00", "0"},
		{"-0.0000000000000000001", "-1/10000000000000000000"},
		{"340282366920938463463374607431768211455", "340282366920938463463374607431768211455"},
		{"1.50", "3/2"},
	}
	for _, c := range cases {
		r, err := FromString(c.in).Rat()
		if err != nil {
			t.Fatalf("Rat(%s): %v", c.in, err)
		}
		if got := r.RatString(); got != c.want {
			t.Errorf("Rat(%s) = %s, want %s", c.in, got, c.want)
		}
	}

	if _, err := NaN(state.Overflow).Rat(); err == nil {
		t.Error("Rat of a NaN should return the NaN's error")
	}
}

func TestBigIntIsTheTruncatedIntegerPart(t *testing.T) {
	cases := [][2]string{
		{"1.75", "1"},
		{"-1.75", "-1"},
		{"0.999", "0"},
		{"-0.999", "0"},
		{"0", "0"},
		// Past the int64 range, where Int64 fails and this is the only exit.
		{"340282366920938463463374607431768211455", "340282366920938463463374607431768211455"},
		{"-34028236692093846346337460743176821145.5", "-34028236692093846346337460743176821145"},
	}
	for _, c := range cases {
		n, err := FromString(c[0]).BigInt()
		if err != nil {
			t.Fatalf("BigInt(%s): %v", c[0], err)
		}
		if got := n.String(); got != c[1] {
			t.Errorf("BigInt(%s) = %s, want %s", c[0], got, c[1])
		}
	}

	// It is the integer part, not the coefficient: those differ for every value with a fractional part.
	d := FromString("1.75")
	n, _ := d.BigInt()
	if n.String() == d.Coefficient().BigInt().String() {
		t.Error("BigInt returned the coefficient rather than the integer part")
	}

	if _, err := NaN(state.DivisionByZero).BigInt(); err == nil {
		t.Error("BigInt of a NaN should return the NaN's error")
	}
}

func TestFromRatRoundsOnce(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	// Exactly representable rationals come back exact whatever the mode.
	for _, c := range []struct {
		num, den int64
		scale    uint8
		want     string
	}{
		{1, 4, 2, "0.25"},
		{-1, 8, 3, "-0.125"},
		{3, 2, 1, "1.5"},
		{7, 1, 0, "7"},
		{0, 5, 4, "0"},
	} {
		for _, mode := range allModes {
			got := FromRat(big.NewRat(c.num, c.den), c.scale, mode)
			if got.String() != c.want {
				t.Errorf("FromRat(%d/%d, %d, %v) = %s, want %s", c.num, c.den, c.scale, mode, got, c.want)
			}
		}
	}

	// A rational that does not terminate is rounded, and refused under ROUND_NAN.
	third := big.NewRat(1, 3)
	if got := FromRat(third, 19, ROUND_BANK).String(); got != "0.3333333333333333333" {
		t.Errorf("FromRat(1/3, 19, bank) = %s", got)
	}
	if got := FromRat(third, 19, ROUND_AWAY_FROM_ZERO).String(); got != "0.3333333333333333334" {
		t.Errorf("FromRat(1/3, 19, away) = %s", got)
	}
	if got := FromRat(third, 19, ROUND_NAN); !got.IsNaN() || got.ErrorDetails() != state.Inexact.Error() {
		t.Errorf("FromRat(1/3, 19, ROUND_NAN) = %v, want NaN(Inexact)", got)
	}

	// The failures.
	if got := FromRat(nil, 2, ROUND_BANK); !got.IsNaN() {
		t.Error("FromRat(nil) should be NaN")
	}
	if got := FromRat(big.NewRat(1, 2), MaxScale+1, ROUND_BANK); got.ErrorDetails() != state.ScaleOutOfRange.Error() {
		t.Errorf("FromRat above MaxScale = %v, want NaN(ScaleOutOfRange)", got.ErrorDetails())
	}
	if got := FromRat(big.NewRat(1, 2), 2, RoundingMode(99)); got.ErrorDetails() != state.InvalidRoundingMode.Error() {
		t.Errorf("FromRat with a bad mode = %v", got.ErrorDetails())
	}
	huge := new(big.Rat).SetInt(new(big.Int).Lsh(big.NewInt(1), 200))
	if got := FromRat(huge, 0, ROUND_BANK); got.ErrorDetails() != state.Overflow.Error() {
		t.Errorf("FromRat of 2^200 = %v, want NaN(Overflow)", got.ErrorDetails())
	}
}

// TestRatRoundTripIsIdentity is the property that makes the bridge worth having: every value goes out and comes back
// unchanged, at its own scale, without the caller having to think about the sign.
func TestRatRoundTripIsIdentity(t *testing.T) {
	r := rand.New(rand.NewSource(20260916))
	for range 5000 {
		d := randDec(r)
		if d.IsNaN() {
			continue
		}
		rat, err := d.Rat()
		if err != nil {
			t.Fatalf("Rat(%s): %v", d, err)
		}
		back := FromRat(rat, d.Scale(), ROUND_NAN)
		if back.IsNaN() {
			t.Fatalf("FromRat(Rat(%s)) was inexact", d)
		}
		if !back.Equal(d) || back.Scale() != d.Scale() {
			t.Errorf("round trip of %s at scale %d gave %s at scale %d", d, d.Scale(), back, back.Scale())
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------
// The decomposer codec

func TestDecomposeComposeRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(20260917))
	buf := make([]byte, 16)
	for range 5000 {
		d := randDec(r)
		form, neg, coef, exp := d.Decompose(buf[:0])
		var back Dec128
		if err := back.Compose(form, neg, coef, exp); err != nil {
			t.Fatalf("Compose after Decompose(%v): %v", d, err)
		}
		if d.IsNaN() {
			if !back.IsNaN() {
				t.Errorf("a NaN decomposed to %v", back)
			}
			continue
		}
		if !back.Equal(d) || back.Scale() != d.Scale() {
			t.Errorf("round trip of %s at scale %d gave %s at scale %d", d, d.Scale(), back, back.Scale())
		}
	}
}

func TestDecomposeParts(t *testing.T) {
	buf := make([]byte, 16)

	// The exponent is always -Scale, and zero yields an empty coefficient with its scale kept.
	form, neg, coef, exp := FromString("0.00").Decompose(buf[:0])
	if form != 0 || neg || len(coef) != 0 || exp != -2 {
		t.Errorf("Decompose(0.00) = %d %v %x %d, want 0 false '' -2", form, neg, coef, exp)
	}

	// A NaN is form 2 and carries nothing.
	form, _, coef, _ = NaN(state.Overflow).Decompose(buf[:0])
	if form != 2 || len(coef) != 0 {
		t.Errorf("Decompose(NaN) = form %d, coefficient %x; want form 2 and nothing", form, coef)
	}

	// The coefficient is minimal big-endian, as big.Int.Bytes would give it.
	_, neg, coef, exp = FromString("-1.75").Decompose(buf[:0])
	if !neg || len(coef) != 1 || coef[0] != 175 || exp != -2 {
		t.Errorf("Decompose(-1.75) = %v %x %d, want true af -2", neg, coef, exp)
	}
}

func TestComposeNormalizesAndRefuses(t *testing.T) {
	var d Dec128

	// A positive exponent is applied to the coefficient.
	if err := d.Compose(0, false, []byte{123}, 2); err != nil || d.String() != "12300" {
		t.Errorf("Compose(123, e+2) = %s, %v", d, err)
	}

	// An exponent below -MaxScale is cancelled against factors of ten in the coefficient where it can be.
	// 15000000E-25 is 1.5E-18, which scale 19 holds.
	if err := d.Compose(0, false, new(big.Int).SetInt64(15000000).Bytes(), -25); err != nil {
		t.Fatalf("Compose(15000000, e-25): %v", err)
	}
	if d.String() != "0.0000000000000000015" || d.Scale() != MaxScale {
		t.Errorf("Compose(15000000, e-25) = %s at scale %d", d, d.Scale())
	}

	// What cannot be cancelled is refused rather than rounded: 15E-25 is 1.5E-24.
	if err := d.Compose(0, false, []byte{15}, -25); err == nil {
		t.Error("Compose(15, e-25) should be refused")
	} else if err.Error() != state.Inexact.Error().Error() {
		t.Errorf("Compose(15, e-25) = %v, want inexact", err)
	}

	// Infinity has no representation here, and is refused rather than turned into a NaN.
	before := d
	if err := d.Compose(1, false, nil, 0); err == nil {
		t.Error("Compose of an infinity should be refused")
	}
	if d != before {
		t.Error("a refused Compose modified the receiver")
	}

	// An unknown form byte, and a coefficient too wide.
	if err := d.Compose(7, false, nil, 0); err == nil {
		t.Error("Compose of an unknown form should be refused")
	}
	wide := new(big.Int).Lsh(big.NewInt(1), 200).Bytes()
	if err := d.Compose(0, false, wide, 0); err == nil {
		t.Error("Compose of a 200-bit coefficient should be refused")
	}

	// Form 2 is a NaN and is not an error, because a NaN is a value here.
	if err := d.Compose(2, false, nil, 0); err != nil || !d.IsNaN() {
		t.Errorf("Compose of a NaN = %s, %v", d, err)
	}
}

// ---------------------------------------------------------------------------------------------------------------
// The five small GDA operations. The tabulated cases are the examples given in the specification itself.

func TestCopySign(t *testing.T) {
	for _, c := range [][3]string{
		{"5", "-1", "-5"},
		{"-5", "1", "5"},
		{"-5", "-1", "-5"},
		{"1.50", "-2", "-1.50"},
		// This type has no negative zero, so a zero keeps its sign whatever it is given.
		{"0", "-1", "0"},
		{"0.00", "-1", "0.00"},
	} {
		got := FromString(c[0]).CopySign(FromString(c[1]))
		if got.String() != FromString(c[2]).String() || got.Scale() != FromString(c[2]).Scale() {
			t.Errorf("CopySign(%s, %s) = %s at scale %d, want %s", c[0], c[1], got, got.Scale(), c[2])
		}
	}

	if got := NaN(state.Overflow).CopySign(One); got.ErrorDetails() != state.Overflow.Error() {
		t.Error("CopySign should propagate a NaN receiver")
	}
	if got := One.CopySign(NaN(state.Underflow)); got.ErrorDetails() != state.Underflow.Error() {
		t.Error("CopySign should propagate a NaN argument")
	}
}

func TestLogb(t *testing.T) {
	// The first three are the examples in the specification.
	for _, c := range [][2]string{
		{"250", "2"},
		{"2.50", "0"},
		{"0.03", "-2"},
		{"1", "0"},
		{"9.99", "0"},
		{"-1234.5", "3"},
		{"0.0000000000000000001", "-19"},
		{"340282366920938463463374607431768211455", "38"},
	} {
		if got := FromString(c[0]).Logb().String(); got != c[1] {
			t.Errorf("Logb(%s) = %s, want %s", c[0], got, c[1])
		}
	}

	// Zero is the specification's -Infinity, which in this type's vocabulary is NaN(DivisionByZero).
	if got := Zero.Logb(); got.ErrorDetails() != state.DivisionByZero.Error() {
		t.Errorf("Logb(0) = %v, want NaN(DivisionByZero)", got.ErrorDetails())
	}
	if got := NaN(state.Overflow).Logb(); got.ErrorDetails() != state.Overflow.Error() {
		t.Error("Logb should propagate a NaN")
	}

	// Logb is IntegerDigits-1 for every value of one or more, and goes negative below it where IntegerDigits stops.
	r := rand.New(rand.NewSource(20260918))
	for range 2000 {
		d := randDec(r)
		if d.IsNaN() || d.IsZero() {
			continue
		}
		lb, err := d.Logb().Int64()
		if err != nil {
			t.Fatalf("Logb(%s) does not fit an int64", d)
		}
		if d.Abs().GreaterThanOrEqual(One) {
			if want := int64(d.IntegerDigits() - 1); lb != want {
				t.Errorf("Logb(%s) = %d, want IntegerDigits-1 = %d", d, lb, want)
			}
		} else if lb >= 0 {
			t.Errorf("Logb(%s) = %d, want a negative exponent for a value below one", d, lb)
		}
	}
}

func TestInvAndInvRound(t *testing.T) {
	defer SetDefaultScale(DefaultScale())
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	if got, want := FromString("4").InvRound(4, ROUND_BANK).String(), "0.25"; got != want {
		t.Errorf("InvRound(4) = %s, want %s", got, want)
	}
	if got := FromString("3").InvRound(19, ROUND_BANK).String(); got != "0.3333333333333333333" {
		t.Errorf("InvRound(3) = %s", got)
	}
	if got := Zero.InvRound(4, ROUND_BANK); got.ErrorDetails() != state.DivisionByZero.Error() {
		t.Errorf("InvRound(0) = %v, want NaN(DivisionByZero)", got.ErrorDetails())
	}

	// Inv and InvRound agree when InvRound is given what Inv reads from the process-global configuration.
	SetDefaultScale(12)
	SetArithmeticRounding(ROUND_HALF_AWAY_FROM_ZERO)
	r := rand.New(rand.NewSource(20260919))
	for range 2000 {
		d := randDec(r)
		if d.IsNaN() || d.IsZero() {
			continue
		}
		if got, want := d.Inv(), One.Div(d); got != want {
			t.Errorf("Inv(%s) = %s, want %s", d, got, want)
		}
	}
}

func TestCmpTotalIsATotalOrder(t *testing.T) {
	// The tabulated cases are the examples in the specification, save the last two, which fix the reversal for a
	// negative pair and the position of a NaN.
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"12.73", "127.9", -1},
		{"-127", "12", -1},
		{"12.30", "12.3", -1},
		{"12.30", "12.30", 0},
		{"12.3", "12.300", 1},
		{"-1.00", "-1", 1},
		{"0.000", "0", -1},
	} {
		if got := FromString(c.a).CmpTotal(FromString(c.b)); got != c.want {
			t.Errorf("CmpTotal(%s, %s) = %d, want %d", c.a, c.b, got, c.want)
		}
	}

	// This package puts NaN below every value, as Compare does, rather than above it as the specification does.
	if got := NaN(state.Overflow).CmpTotal(MinAtScale(0)); got != -1 {
		t.Errorf("CmpTotal(NaN, min) = %d, want -1 (NaN sorts low here, as it does under Compare)", got)
	}
	if got := NaN(state.Overflow).CmpTotal(NaN(state.Overflow)); got != 0 {
		t.Errorf("CmpTotal of two identical NaNs = %d, want 0", got)
	}
	if got := NaN(state.NaN).CmpTotal(NaN(state.Overflow)); got != -1 {
		t.Errorf("CmpTotal orders NaNs by state code; got %d", got)
	}

	// The property: anti-symmetry, and a zero result only for identical representations.
	r := rand.New(rand.NewSource(20260920))
	vals := make([]Dec128, 0, 400)
	for range 400 {
		vals = append(vals, randDec(r))
	}
	vals = append(vals, FromString("1.5"), FromString("1.50"), FromString("1.500"), Zero, FromString("0.0"),
		NaN(state.Overflow), NaN(state.Inexact))
	for _, a := range vals {
		for _, b := range vals {
			ab, ba := a.CmpTotal(b), b.CmpTotal(a)
			if ab != -ba {
				t.Fatalf("CmpTotal(%v, %v) = %d but CmpTotal(%v, %v) = %d", a, b, ab, b, a, ba)
			}
			if ab == 0 && a != b {
				t.Fatalf("CmpTotal called %v and %v equal, but their representations differ", a, b)
			}
		}
	}
}

func TestRemainderNear(t *testing.T) {
	// The tabulated cases are the examples in the specification.
	for _, c := range [][3]string{
		{"2.1", "3", "-0.9"},
		{"10", "6", "-2"},
		{"10", "3", "1"},
		{"-10", "3", "-1"},
		{"10.2", "1", "0.2"},
		{"10", "0.3", "0.1"},
		{"3.6", "1.3", "-0.3"},
		// A tie goes to the even quotient: 7/2 is 3.5, and 4 is the even neighbor.
		{"7", "2", "-1"},
		{"5", "2", "1"},
		{"9", "2", "1"},
		{"-7", "2", "1"},
		// A remainder far smaller than a divisor that does not fit at the remainder's scale.
		{"0.000000000001", "1000000000000000000000000000000", "0.000000000001"},
	} {
		got := FromString(c[0]).RemainderNear(FromString(c[1]))
		if !got.Equal(FromString(c[2])) {
			t.Errorf("RemainderNear(%s, %s) = %s, want %s", c[0], c[1], got, c[2])
		}
	}

	if got := One.RemainderNear(Zero); got.ErrorDetails() != state.DivisionByZero.Error() {
		t.Errorf("RemainderNear by zero = %v, want NaN(DivisionByZero)", got.ErrorDetails())
	}
	if got := NaN(state.Overflow).RemainderNear(One); !got.IsNaN() {
		t.Error("RemainderNear should propagate a NaN")
	}

	// The properties, over random operands: the result is exact, its magnitude is at most half the divisor's, and
	// RoundToMultiple plus RemainderNear reconstructs the value.
	r := rand.New(rand.NewSource(20260921))
	checked := 0
	for range 20000 {
		d, o := randDec(r), randDec(r)
		if d.IsNaN() || o.IsNaN() || o.IsZero() {
			continue
		}
		rem := d.RemainderNear(o)
		if rem.IsNaN() {
			continue // the quotient did not fit, which QuoRem reports the same way
		}

		// Exact against math/big: d - o*round-half-even(d/o).
		dr, _ := d.Rat()
		or, _ := o.Rat()
		q := new(big.Rat).Quo(dr, or)
		n, _ := roundRat(q, 0, ROUND_BANK)
		if q.Sign() < 0 {
			n.Neg(n)
		}
		want := new(big.Rat).Sub(dr, new(big.Rat).Mul(or, new(big.Rat).SetInt(n)))
		gotRat, err := rem.Rat()
		if err != nil {
			t.Fatalf("RemainderNear(%s, %s) = %s: %v", d, o, rem, err)
		}
		if gotRat.Cmp(want) != 0 {
			t.Fatalf("RemainderNear(%s, %s) = %s, want %s", d, o, gotRat.RatString(), want.RatString())
		}

		// |rem| <= |o|/2.
		twice := new(big.Rat).Add(new(big.Rat).Abs(gotRat), new(big.Rat).Abs(gotRat))
		if twice.Cmp(new(big.Rat).Abs(or)) > 0 {
			t.Fatalf("RemainderNear(%s, %s) = %s, which is more than half of the divisor", d, o, rem)
		}
		checked++
	}
	if checked < 1000 {
		t.Fatalf("only %d operand pairs were usable; the generator is not exercising this", checked)
	}
}

// ---------------------------------------------------------------------------------------------------------------
// The two logarithms

func TestLog10AndLog2AreExactOnPowersOfTheBase(t *testing.T) {
	// An exact power of ten gives an exact integer under every mode, which is what the specification requires of
	// log10 and what a series alone would not give: converging from below, a directed mode would return 2.999...
	for _, c := range [][2]string{
		{"1", "0"}, {"10", "1"}, {"100", "2"}, {"1000", "3"},
		{"0.1", "-1"}, {"0.001", "-3"}, {"1.00", "0"}, {"10.0", "1"},
		{"0.0000000000000000001", "-19"},
		{"10000000000000000000", "19"},
	} {
		for _, mode := range allModes {
			if mode == ROUND_NAN {
				continue // an exact result is not refused, but the comparison below is simpler without it
			}
			got := FromString(c[0]).Log10(6, mode)
			if !got.Equal(FromString(c[1])) {
				t.Errorf("Log10(%s) under %v = %s, want exactly %s", c[0], mode, got, c[1])
			}
		}
	}

	// Every power of two is a terminating decimal, so the same holds there.
	for _, c := range [][2]string{
		{"1", "0"}, {"2", "1"}, {"4", "2"}, {"8", "3"}, {"1024", "10"},
		{"0.5", "-1"}, {"0.25", "-2"}, {"0.125", "-3"}, {"0.0625", "-4"},
		{"1.0", "0"}, {"2.0", "1"}, {"536870912", "29"},
	} {
		for _, mode := range allModes {
			if mode == ROUND_NAN {
				continue
			}
			got := FromString(c[0]).Log2(6, mode)
			if !got.Equal(FromString(c[1])) {
				t.Errorf("Log2(%s) under %v = %s, want exactly %s", c[0], mode, got, c[1])
			}
		}
	}

	// And a value that merely looks like one is not treated as exact: 2.5 is not a power of two, 1.5 not a power of
	// ten, and 0.2 is neither although 0.2 * 5 is 1.
	for _, s := range []string{"2.5", "1.5", "0.2", "3", "0.3"} {
		if got := FromString(s).Log2(19, ROUND_BANK); got.IsInteger() {
			t.Errorf("Log2(%s) = %s, which was taken as exact", s, got)
		}
	}
	for _, s := range []string{"2", "1.5", "0.5", "11"} {
		if got := FromString(s).Log10(19, ROUND_BANK); got.IsInteger() {
			t.Errorf("Log10(%s) = %s, which was taken as exact", s, got)
		}
	}
}

func TestLog10AndLog2Failures(t *testing.T) {
	for _, s := range []string{"0", "-1", "-0.5"} {
		if got := FromString(s).Log10(6, ROUND_BANK); got.ErrorDetails() != state.DomainError.Error() {
			t.Errorf("Log10(%s) = %v, want NaN(DomainError)", s, got.ErrorDetails())
		}
		if got := FromString(s).Log2(6, ROUND_BANK); got.ErrorDetails() != state.DomainError.Error() {
			t.Errorf("Log2(%s) = %v, want NaN(DomainError)", s, got.ErrorDetails())
		}
	}
	if got := One.Log10(MaxScale+1, ROUND_BANK); got.ErrorDetails() != state.ScaleOutOfRange.Error() {
		t.Error("Log10 above MaxScale should be NaN(ScaleOutOfRange)")
	}
	if got := FromString("3").Log2(6, RoundingMode(99)); got.ErrorDetails() != state.InvalidRoundingMode.Error() {
		t.Error("Log2 with an undefined mode should be NaN(InvalidRoundingMode)")
	}
	if got := NaN(state.Overflow).Log10(6, ROUND_BANK); got.ErrorDetails() != state.Overflow.Error() {
		t.Error("Log10 should propagate a NaN")
	}
}

// TestLogsAgreeWithLn holds the two logarithms to the identity that defines them, at a scale low enough that the one
// unit the division may cost cannot reach the last digit.
func TestLogsAgreeWithLn(t *testing.T) {
	r := rand.New(rand.NewSource(20260922))
	ln10 := FromString("2.302585092994045684")
	ln2 := FromString("0.693147180559945309")

	for range 2000 {
		d := randDec(r)
		if d.IsNaN() || d.IsZero() || d.IsNegative() {
			continue
		}
		if k, ok := d.pow10Exponent(); ok {
			_ = k
			continue // answered exactly, and not by the identity
		}
		want := d.Ln(MaxScale, ROUND_BANK).DivRound(ln10, 12, ROUND_BANK)
		if got := d.Log10(12, ROUND_BANK); got.Sub(want).Abs().GreaterThan(QuantumAtScale(12)) {
			t.Errorf("Log10(%s) = %s, but Ln/ln(10) is %s", d, got, want)
		}
		if _, ok := d.pow2Exponent(); ok {
			continue
		}
		want2 := d.Ln(MaxScale, ROUND_BANK).DivRound(ln2, 12, ROUND_BANK)
		if got := d.Log2(12, ROUND_BANK); got.Sub(want2).Abs().GreaterThan(QuantumAtScale(12)) {
			t.Errorf("Log2(%s) = %s, but Ln/ln(2) is %s", d, got, want2)
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------
// The fused add-and-divide

func TestAddQuoRoundAgainstBigRat(t *testing.T) {
	r := rand.New(rand.NewSource(20260923))
	checked := 0
	for range 30000 {
		d, e, f := randDec(r), randDec(r), randDec(r)
		if d.IsNaN() || e.IsNaN() || f.IsNaN() || f.IsZero() {
			continue
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]
		if mode == ROUND_NAN {
			mode = ROUND_BANK
		}

		got := d.AddQuoRound(e, f, scale, mode)

		dr, _ := d.Rat()
		er, _ := e.Rat()
		fr, _ := f.Rat()
		want := FromRat(new(big.Rat).Add(dr, new(big.Rat).Quo(er, fr)), scale, mode)

		if got.IsNaN() != want.IsNaN() || (!got.IsNaN() && got != want) {
			t.Fatalf("%s + %s/%s at scale %d under %v: got %s, want %s", d, e, f, scale, mode, got, want)
		}
		checked++
	}
	if checked < 10000 {
		t.Fatalf("only %d operand triples were usable", checked)
	}
}

func TestAddQuoRoundIsBetterThanTheWrittenOutForm(t *testing.T) {
	// The point of the fused form is that the quotient is not rounded before the addition. This finds the cases where
	// that changes the answer and holds the fused form to the exact value in each of them - and fails if there are
	// none, because then the operation would not be worth having.
	r := rand.New(rand.NewSource(20260927))
	separated := 0

	for range 20000 {
		d, e, f := randDec(r), randDec(r), randDec(r)
		if d.IsNaN() || e.IsNaN() || f.IsNaN() || f.IsZero() {
			continue
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))

		fused := d.AddQuoRound(e, f, scale, ROUND_BANK)
		naive := d.AddRound(e.DivRound(f, scale, ROUND_BANK), scale, ROUND_BANK)
		if fused.IsNaN() || naive.IsNaN() || fused == naive {
			continue
		}

		// They differ, so exactly one of them is the correctly rounded value of d + e/f.
		dr, _ := d.Rat()
		er, _ := e.Rat()
		fr, _ := f.Rat()
		exact := FromRat(new(big.Rat).Add(dr, new(big.Rat).Quo(er, fr)), scale, ROUND_BANK)
		if fused != exact {
			t.Fatalf("%s + %s/%s at scale %d: fused gave %s, the exact value rounds to %s", d, e, f, scale, fused, exact)
		}
		if separated == 0 {
			t.Logf("for example %s + %s/%s at scale %d: fused %s, written out %s", d, e, f, scale, fused, naive)
		}
		separated++
	}

	if separated == 0 {
		t.Fatal("no operand triple separated the fused form from the written-out one; the operation buys nothing")
	}
	t.Logf("%d of the triples separated the two forms, and the fused form was right in every one", separated)
}

func TestAddQuoRoundFailures(t *testing.T) {
	a := FromString("1.5")
	if got := a.AddQuoRound(a, Zero, 2, ROUND_BANK); got.ErrorDetails() != state.DivisionByZero.Error() {
		t.Error("a zero divisor should be NaN(DivisionByZero)")
	}
	if got := a.AddQuoRound(a, a, MaxScale+1, ROUND_BANK); got.ErrorDetails() != state.ScaleOutOfRange.Error() {
		t.Error("a scale above MaxScale should be NaN(ScaleOutOfRange)")
	}
	if got := a.AddQuoRound(a, a, 2, RoundingMode(99)); got.ErrorDetails() != state.InvalidRoundingMode.Error() {
		t.Error("an undefined mode should be NaN(InvalidRoundingMode)")
	}
	if got := a.AddQuoRound(One, FromString("3"), 2, ROUND_NAN); got.ErrorDetails() != state.Inexact.Error() {
		t.Errorf("ROUND_NAN on an inexact result = %v, want NaN(Inexact)", got.ErrorDetails())
	}
	// NaN propagates in argument order.
	if got := NaN(state.Overflow).AddQuoRound(NaN(state.Underflow), a, 2, ROUND_BANK); got.ErrorDetails() != state.Overflow.Error() {
		t.Error("the receiver's NaN should win")
	}
	if got := a.AddQuoRound(NaN(state.Underflow), NaN(state.Inexact), 2, ROUND_BANK); got.ErrorDetails() != state.Underflow.Error() {
		t.Error("the second argument's NaN should win over the third's")
	}
	// Overflow.
	if got := MaxAtScale(0).AddQuoRound(MaxAtScale(0), QuantumAtScale(MaxScale), 0, ROUND_BANK); got.ErrorDetails() != state.Overflow.Error() {
		t.Errorf("a result past the coefficient = %v, want NaN(Overflow)", got.ErrorDetails())
	}
}

// ---------------------------------------------------------------------------------------------------------------
// The product family

func TestProdRoundAgainstBigRat(t *testing.T) {
	r := rand.New(rand.NewSource(20260924))
	checked := 0
	for range 4000 {
		n := 1 + r.Intn(6)
		xs := make([]Dec128, n)
		want := new(big.Rat).SetInt64(1)
		nan := false
		for j := range xs {
			xs[j] = randDec(r)
			if xs[j].IsNaN() {
				nan = true
				break
			}
			q, _ := xs[j].Rat()
			want.Mul(want, q)
		}
		if nan {
			continue
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]
		if mode == ROUND_NAN {
			mode = ROUND_BANK
		}

		got := ProdSliceRound(xs, scale, mode)
		exp := FromRat(want, scale, mode)
		if got.IsNaN() != exp.IsNaN() || (!got.IsNaN() && got != exp) {
			t.Fatalf("Prod%v at scale %d under %v: got %s, want %s", xs, scale, mode, got, exp)
		}
		checked++
	}
	if checked < 1000 {
		t.Fatalf("only %d slices were usable", checked)
	}
}

// TestProdOfTwoAgreesWithMul pins the product family to the operation it generalizes: for two factors there is no
// intermediate rounding to differ over, so Prod must be Mul exactly, scale included.
func TestProdOfTwoAgreesWithMul(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	// Prod applies the scale rule over math/big where Mul applies it over the 256-bit product: a third
	// implementation of the one rule, and this is what holds it to the other two. The whole matrix, because the
	// policies are where the three differ if they ever do - LossNaNOnInexact turns a dropped digit into a NaN and
	// LossNaNOnUnderflow only a total loss, and each has to fire on the same operands in both.
	r := rand.New(rand.NewSource(20260925))
	for _, mode := range allModes {
		for _, policy := range []LossPolicy{LossRound, LossNaNOnUnderflow, LossNaNOnInexact} {
			SetArithmeticRounding(mode) // writes the loss policy too, so it goes first
			SetLossPolicy(policy)
			for range 5000 {
				a, b := randDec(r), randDec(r)
				if a.IsNaN() || b.IsNaN() {
					continue
				}
				got, want := Prod(a, b), a.Mul(b)
				if got != want {
					t.Fatalf("under %v/%v: Prod(%s, %s) = %s at scale %d, Mul = %s at scale %d",
						mode, policy, a, b, got, got.Scale(), want, want.Scale())
				}
			}
		}
	}
}

func TestProdOneRoundingBeatsAFold(t *testing.T) {
	// Chain-linking a year of daily factors. The fold rounds 250 times and drifts; the product rounds once.
	xs := make([]Dec128, 250)
	for i := range xs {
		xs[i] = FromString("1.0004")
	}

	once := ProdSliceRound(xs, MaxScale, ROUND_BANK)

	fold := One
	for _, x := range xs {
		fold = fold.MulRound(x, MaxScale, ROUND_BANK)
	}

	// The exact product, to settle which of the two is right.
	want := new(big.Rat).SetInt64(1)
	for _, x := range xs {
		q, _ := x.Rat()
		want.Mul(want, q)
	}
	exact := FromRat(want, MaxScale, ROUND_BANK)

	if once != exact {
		t.Errorf("one rounding gave %s, the exact product rounds to %s", once, exact)
	}
	if fold == exact {
		t.Skip("the fold happened to agree; the example no longer separates them")
	}
	t.Logf("one rounding %s, fold of MulRound %s, both against the exact %s", once, fold, exact)
}

func TestProdEdges(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	// The empty product is one, as the empty sum is zero.
	if got := ProdSlice(nil); !got.Equal(One) {
		t.Errorf("ProdSlice(nil) = %s, want 1", got)
	}
	if got := ProdSliceRound(nil, 4, ROUND_BANK); !got.Equal(One) || got.Scale() != 4 {
		t.Errorf("ProdSliceRound(nil, 4) = %s at scale %d, want 1 at scale 4", got, got.Scale())
	}

	// One factor is itself.
	if got := Prod(FromString("1.50")); got != FromString("1.50") {
		t.Errorf("Prod of one factor = %s", got)
	}

	// A zero factor makes the product zero, and the zero is not negative.
	got := ProdRound(4, ROUND_BANK, FromString("-2"), Zero, FromString("3"))
	if !got.IsZero() || got.IsNegative() {
		t.Errorf("a zero factor gave %s", got)
	}

	// A NaN is reported even when an earlier factor was zero, because it is a fact about the data.
	if got := ProdRound(4, ROUND_BANK, Zero, NaN(state.Overflow)); got.ErrorDetails() != state.Overflow.Error() {
		t.Errorf("a NaN after a zero = %v, want NaN(Overflow)", got.ErrorDetails())
	}
	// The first NaN in order wins.
	if got := ProdRound(4, ROUND_BANK, NaN(state.Underflow), NaN(state.Overflow)); got.ErrorDetails() != state.Underflow.Error() {
		t.Error("the first NaN in order should be returned")
	}

	// The failures.
	if got := ProdRound(MaxScale+1, ROUND_BANK, One, One); got.ErrorDetails() != state.ScaleOutOfRange.Error() {
		t.Error("a scale above MaxScale should be NaN(ScaleOutOfRange)")
	}
	if got := ProdRound(4, RoundingMode(99), One, One); got.ErrorDetails() != state.InvalidRoundingMode.Error() {
		t.Error("an undefined mode should be NaN(InvalidRoundingMode)")
	}
	if got := ProdRound(0, ROUND_BANK, FromString("1e30"), FromString("1e30")); got.ErrorDetails() != state.Overflow.Error() {
		t.Error("a product past the coefficient should be NaN(Overflow)")
	}
	if got := ProdRound(2, ROUND_NAN, FromString("1.005"), FromString("1.005")); got.ErrorDetails() != state.Inexact.Error() {
		t.Errorf("ROUND_NAN on an inexact product = %v, want NaN(Inexact)", got.ErrorDetails())
	}

	// Order does not change the result, which is the property one rounding buys.
	xs := []Dec128{FromString("1.0001"), FromString("0.9999"), FromString("1.05"), FromString("0.000001"), FromString("7.25")}
	base := ProdSliceRound(xs, MaxScale, ROUND_BANK)
	r := rand.New(rand.NewSource(20260926))
	for range 200 {
		perm := append([]Dec128(nil), xs...)
		r.Shuffle(len(perm), func(i, j int) { perm[i], perm[j] = perm[j], perm[i] })
		if got := ProdSliceRound(perm, MaxScale, ROUND_BANK); got != base {
			t.Fatalf("order changed the product: %s versus %s", got, base)
		}
	}
}
