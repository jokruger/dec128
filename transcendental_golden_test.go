package dec128

import (
	"encoding/csv"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/jokruger/dec128/state"
)

// The oracle for Exp, Ln, Ln1p, Expm1 and Pow. testdata/transcendental_golden.csv holds each value to 80 significant
// digits, produced by Python's decimal module - an implementation of the General Decimal Arithmetic specification, in
// which exp and ln are correctly rounded - so it is an independent reference and not a second copy of the series
// under test.
//
// These are the only operations in the package that are not exact, so what these tests measure is the error in units
// of the last place, rather than asserting an exact value.

type goldenRow struct {
	fn         string
	arg1, arg2 Dec128
	exact      *big.Rat
	digits     string
}

func loadTranscendentalGolden(t *testing.T) []goldenRow {
	t.Helper()

	f, err := os.Open("testdata/transcendental_golden.csv")
	if err != nil {
		t.Fatal("failed to open golden file: testdata/transcendental_golden.csv")
	}
	defer func() { _ = f.Close() }()

	recs, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("reading the golden data: %v", err)
	}

	rows := make([]goldenRow, 0, len(recs))
	for i, rec := range recs[1:] {
		if len(rec) != 4 {
			t.Fatalf("row %d has %d fields, want 4", i+2, len(rec))
		}
		exact, ok := new(big.Rat).SetString(rec[3])
		if !ok {
			t.Fatalf("row %d: cannot parse the value %q", i+2, rec[3])
		}
		r := goldenRow{fn: rec[0], arg1: FromString(rec[1]), exact: exact, digits: rec[3]}
		if rec[2] != "" {
			r.arg2 = FromString(rec[2])
		}
		if r.arg1.IsNaN() || (rec[2] != "" && r.arg2.IsNaN()) {
			t.Fatalf("row %d: an argument does not parse as a Dec128: %q %q", i+2, rec[1], rec[2])
		}
		rows = append(rows, r)
	}

	return rows
}

func (r goldenRow) call(scale uint8, mode RoundingMode) Dec128 {
	switch r.fn {
	case "Ln":
		return r.arg1.Ln(scale, mode)
	case "Exp":
		return r.arg1.Exp(scale, mode)
	case "Ln1p":
		return r.arg1.Ln1p(scale, mode)
	case "Expm1":
		return r.arg1.Expm1(scale, mode)
	case "Log10":
		return r.arg1.Log10(scale, mode)
	case "Log2":
		return r.arg1.Log2(scale, mode)
	default:
		return r.arg1.Pow(r.arg2, scale, mode)
	}
}

func (r goldenRow) String() string {
	if r.fn == "Pow" {
		return fmt.Sprintf("%s(%s, %s)", r.fn, r.arg1.StringFixed(), r.arg2.StringFixed())
	}
	return fmt.Sprintf("%s(%s)", r.fn, r.arg1.StringFixed())
}

// ambiguousAt reports whether the 80 digits of the golden value are too few to decide the rounding at this scale,
// which needs a run of zeros or nines just below the last kept digit some sixty digits long. It has never fired; it
// is here so that a case which did would be skipped rather than reported as an error in the implementation.
func (r goldenRow) ambiguousAt(scale uint8) bool {
	dot := strings.IndexByte(r.digits, '.')
	if dot < 0 {
		return false
	}
	frac := r.digits[dot+1:]
	if int(scale) >= len(frac) {
		return false
	}
	tail := strings.TrimRight(frac[scale:], "0")
	if tail == "" {
		return false // an exact value at this scale, which is decidable and must be compared
	}
	return strings.HasPrefix(tail, strings.Repeat("0", 40)) || strings.HasPrefix(tail, strings.Repeat("9", 40))
}

// TestTranscendentalUlpError is the measurement the documented guarantee rests on: over every golden value, at every
// scale and in every rounding mode, how far is the result from the correctly rounded one? The claim in the method
// documentation is faithful rounding - at most one unit in the last place - and the assertion here is that claim.
// The largest error actually observed is reported, so that a regression shows up as a number and not only as a pass.
func TestTranscendentalUlpError(t *testing.T) {
	rows := loadTranscendentalGolden(t)

	var worst int64
	var worstCase string
	var compared, skipped int

	for _, r := range rows {
		for _, scale := range []uint8{0, 2, 6, 10, MaxScale} {
			if r.ambiguousAt(scale) {
				skipped++
				continue
			}
			for _, mode := range allModes {
				if mode == ROUND_NAN {
					continue // refuses every inexact result by design; covered separately
				}

				want, _ := roundRat(r.exact, scale, mode)
				got := r.call(scale, mode)

				if want.CmpAbs(big2p128) >= 0 {
					if !got.IsNaN() {
						t.Errorf("%s at scale %d: want NaN(Overflow), got %s", r, scale, got.StringFixed())
					}
					continue
				}
				if got.IsNaN() {
					t.Errorf("%s at scale %d under %v: got NaN(%v), want %s", r, scale, mode, got.ErrorDetails(), want)
					continue
				}

				diff := new(big.Int).Sub(bigAtScale(got, scale), signedGolden(want, r.exact))
				ulp := diff.Abs(diff).Int64()
				compared++
				if ulp > worst {
					worst, worstCase = ulp, fmt.Sprintf("%s at scale %d under %v", r, scale, mode)
				}
				if ulp > 1 {
					t.Errorf("%s at scale %d under %v: got %s, want %s (off by %d units in the last place)",
						r, scale, mode, got.StringFixed(), want, ulp)
				}
			}
		}
	}

	if compared == 0 {
		t.Fatal("no golden rows were compared")
	}
	t.Logf("compared %d results (%d skipped as ambiguous); largest error %d ulp%s",
		compared, skipped, worst, worstSuffix(worst, worstCase))
}

// signedGolden restores the sign roundRat drops: it returns the rounded magnitude, and the oracle value carries the
// sign separately.
func signedGolden(mag *big.Int, exact *big.Rat) *big.Int {
	if exact.Sign() < 0 {
		return new(big.Int).Neg(mag)
	}
	return mag
}

func worstSuffix(worst int64, where string) string {
	if worst == 0 {
		return ""
	}
	return ", at " + where
}

// TestTranscendentalExactCases pins the arguments where the value is exact, because those are the ones where the
// result must be exact in every mode rather than merely close, and where ROUND_NAN must return a value.
func TestTranscendentalExactCases(t *testing.T) {
	for _, mode := range allModes {
		for _, c := range []struct {
			name string
			got  Dec128
			want string
		}{
			{"Ln(1)", One.Ln(4, mode), "0.0000"},
			{"Ln1p(0)", Zero.Ln1p(4, mode), "0.0000"},
			{"Exp(0)", Zero.Exp(4, mode), "1.0000"},
			{"Expm1(0)", Zero.Expm1(4, mode), "0.0000"},
			{"Pow(x, 0)", FromString("7.5").Pow(Zero, 4, mode), "1.0000"},
			{"Pow(1, x)", One.Pow(FromString("0.3333333333"), 4, mode), "1.0000"},
			{"Pow(4, 0.5)", FromInt64(4).Pow(FromString("0.5"), 4, mode), "2.0000"},
			{"Pow(8, 2)", FromInt64(8).Pow(FromInt64(2), 4, mode), "64.0000"},
		} {
			if c.got.IsNaN() || c.got.StringFixed() != c.want {
				t.Errorf("%s under %v = %s (%v), want %s", c.name, mode, c.got.StringFixed(), c.got.ErrorDetails(), c.want)
			}
		}
	}
}

// TestLnTenConstant checks the one hardcoded constant against the golden data, so that the digits in the source are
// held to a reference from outside this package.
func TestLnTenConstant(t *testing.T) {
	rows := loadTranscendentalGolden(t)
	for _, r := range rows {
		if r.fn != "Ln" || !r.arg1.Equal(FromInt64(10)) {
			continue
		}
		// the constant truncated to 70 places must match the oracle truncated to 70 places
		got := lnTen(70)
		want := ratAtScale(r.exact, 70)
		if got.Cmp(want) != 0 {
			t.Fatalf("lnTenCoef is wrong:\n got  %s\n want %s", got, want)
		}
		return
	}
	t.Fatal("the golden data has no Ln(10) row to check the constant against")
}

// TestTranscendentalRoundNan pins the ROUND_NAN behavior: an inexact transcendental refuses, an exact one does not.
func TestTranscendentalRoundNan(t *testing.T) {
	if got := FromInt64(2).Ln(MaxScale, ROUND_NAN); got.ErrorDetails() != state.Inexact.Error() {
		t.Errorf("Ln(2) under ROUND_NAN = %v, want NaN(Inexact)", got.ErrorDetails())
	}
	if got := One.Ln(4, ROUND_NAN); got.IsNaN() || !got.IsZero() {
		t.Errorf("Ln(1) under ROUND_NAN = %v, want an exact zero", got.ErrorDetails())
	}
	if got := FromInt64(1).Exp(MaxScale, ROUND_NAN); got.ErrorDetails() != state.Inexact.Error() {
		t.Errorf("Exp(1) under ROUND_NAN = %v, want NaN(Inexact)", got.ErrorDetails())
	}
}

// TestPowExpLnAgainstPowRational is the large-sample error measurement, and the one that does not depend on the
// golden file. PowRational is correctly rounded by construction, so for any exponent it can take it is the true
// answer; powExpLn is the approximation. Comparing them over random operands measures the error of the series and
// the argument reduction against a reference that shares none of their code.
//
// Pow itself routes these exponents to PowRational, so the comparison has to call the general arm directly - which
// is exactly why it is a separate method.
func TestPowExpLnAgainstPowRational(t *testing.T) {
	r := rand.New(rand.NewSource(20260928))

	var worst int64
	var worstCase string
	var compared int

	for range 4000 {
		d := randPowBase(r).Abs()
		if d.IsNaN() || d.coef.IsZero() {
			continue
		}
		p := int64(1 + r.Intn(24))
		if r.Intn(2) == 0 {
			p = -p
		}
		q := int64(1 + r.Intn(24))
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]
		if mode == ROUND_NAN {
			continue // the exact arm returns a value where the approximate one cannot certify exactness
		}

		// the exponent as a Dec128, which has to be exact for the comparison to mean anything
		e := FromInt64(p).DivRound(FromInt64(q), MaxScale, ROUND_TOWARD_ZERO)
		if e.IsNaN() || !e.MulRound(FromInt64(q), MaxScale, ROUND_TOWARD_ZERO).Equal(FromInt64(p)) {
			continue // p/q does not terminate at MaxScale, so the two are not being asked the same question
		}

		want := d.PowRational(p, q, scale, mode)
		got := d.powExpLn(e, scale, mode)
		if want.IsNaN() || got.IsNaN() {
			if want.IsNaN() != got.IsNaN() {
				// only an overflow that lands either side of the boundary, which the ulp check below cannot express
				continue
			}
			continue
		}

		diff := new(big.Int).Sub(bigAtScale(got, scale), bigAtScale(want, scale))
		ulp := diff.Abs(diff).Int64()
		compared++
		if ulp > worst {
			worst, worstCase = ulp, fmt.Sprintf("%s^(%d/%d) at scale %d under %v", d.StringFixed(), p, q, scale, mode)
		}
		if ulp > 1 {
			t.Errorf("%s^(%d/%d) at scale %d under %v: exp/ln gives %s, exact gives %s (off by %d ulp)",
				d.StringFixed(), p, q, scale, mode, got.StringFixed(), want.StringFixed(), ulp)
		}
	}

	if compared < 500 {
		t.Fatalf("only %d comparisons; the corpus no longer exercises the general arm", compared)
	}
	t.Logf("compared %d results; largest error %d ulp%s", compared, worst, worstSuffix(worst, worstCase))
}

// TestExpLnRoundTrip is the cheap identity check over a wide corpus: exp(ln(x)) returns to x, and ln(a*b) is
// ln(a)+ln(b). It catches a reduction that is wrong in a range the golden corpus does not name.
func TestExpLnRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(20260929))
	var checked int

	for range 3000 {
		d := randPowBase(r).Abs()
		if d.IsNaN() || d.coef.IsZero() {
			continue
		}

		ln := d.Ln(MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
		if ln.IsNaN() {
			continue
		}
		back := ln.Exp(MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
		if back.IsNaN() {
			continue
		}

		// The round trip cannot be exact: ln is rounded to MaxScale before exp sees it, and for a large d one unit
		// in its last place is a relative 10^-19 that exp multiplies back up. Compare relatively.
		diff := back.SubRound(d, MaxScale, ROUND_TOWARD_ZERO).Abs()
		tol := d.Abs().MulRound(FromString("0.000000000000000001"), MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
		if tol.IsZero() {
			tol = QuantumAtScale(MaxScale)
		}
		if diff.Compare(tol) > 0 {
			t.Fatalf("exp(ln(%s)) = %s, off by %s (tolerance %s)", d.StringFixed(), back.StringFixed(), diff.StringFixed(), tol.StringFixed())
		}
		checked++
	}

	if checked == 0 {
		t.Fatal("the corpus never produced a usable round trip")
	}
}

// TestLn1pMatchesComposition and TestExpm1MatchesComposition pin what these two actually do on this type, which is
// not what the usual argument for them says. A Dec128 carries 39 significant digits, so 1+x for a small x is exact
// and the composition loses nothing: the functions agree, and the documentation says they agree rather than claiming
// digits back that were never lost. The test exists so that the claim stays true if the type's digit budget ever
// changes - if it narrows, these will start to differ and this will say so.
func TestLn1pMatchesComposition(t *testing.T) {
	r := rand.New(rand.NewSource(20260930))
	var checked int

	for range 3000 {
		d := randPowBase(r)
		if d.IsNaN() {
			continue
		}
		sum := One.Add(d)
		if sum.IsNaN() || sum.scale != d.scale || !sum.IsPositive() {
			continue // only where 1+d is exact, which is the claim
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]

		got, want := d.Ln1p(scale, mode), sum.Ln(scale, mode)
		if got != want {
			t.Fatalf("Ln1p(%s) = %s (%v), Ln(1+%s) = %s (%v)",
				d.StringFixed(), got.StringFixed(), got.state, d.StringFixed(), want.StringFixed(), want.state)
		}
		checked++
	}

	if checked == 0 {
		t.Fatal("the corpus never produced an exact 1+d")
	}

	// The smallest value the type has, where the cancellation argument would bite hardest in a narrower format.
	x := QuantumAtScale(MaxScale)
	if got := x.Ln1p(MaxScale, ROUND_HALF_AWAY_FROM_ZERO); !got.Equal(x) {
		t.Errorf("Ln1p(%s) = %s, want %s", x.StringFixed(), got.StringFixed(), x.StringFixed())
	}
	if got := One.Add(x).Ln(MaxScale, ROUND_HALF_AWAY_FROM_ZERO); !got.Equal(x) {
		t.Errorf("Ln(1+%s) = %s, want %s - 1+x is exact at this width, so the composition must agree",
			x.StringFixed(), got.StringFixed(), x.StringFixed())
	}

	// A basis point, against the oracle.
	if got, want := FromString("0.0001").Ln1p(16, ROUND_HALF_AWAY_FROM_ZERO), FromString("0.0000999950003333"); !got.Equal(want) {
		t.Errorf("Ln1p(0.0001) at 16 places = %s, want %s", got.StringFixed(), want.StringFixed())
	}
}

func TestExpm1MatchesComposition(t *testing.T) {
	r := rand.New(rand.NewSource(20260931))
	var checked int

	for range 3000 {
		d := randPowBase(r)
		if d.IsNaN() || d.Abs().Compare(FromInt64(40)) > 0 {
			continue
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		mode := allModes[r.Intn(len(allModes))]
		if mode == ROUND_NAN {
			continue // the composition rounds twice, so it refuses in places the fused form does not
		}

		e := d.Exp(scale, mode)
		if e.IsNaN() {
			continue
		}
		got, want := d.Expm1(scale, mode), e.Sub(One)
		// They agree wherever exp(d) was exact at this scale; where it was not, the composition rounded first and
		// the two may differ by a unit, which is the ordinary two-roundings-versus-one difference.
		diff := got.SubRound(want, scale, ROUND_TOWARD_ZERO).Abs()
		if diff.Compare(QuantumAtScale(scale)) > 0 {
			t.Fatalf("Expm1(%s) = %s, Exp(%s)-1 = %s at scale %d", d.StringFixed(), got.StringFixed(), d.StringFixed(), want.StringFixed(), scale)
		}
		checked++
	}

	if checked == 0 {
		t.Fatal("the corpus never produced a usable comparison")
	}

	x := QuantumAtScale(MaxScale)
	if got := x.Expm1(MaxScale, ROUND_HALF_AWAY_FROM_ZERO); !got.Equal(x) {
		t.Errorf("Expm1(%s) = %s, want %s", x.StringFixed(), got.StringFixed(), x.StringFixed())
	}
	if got := x.Exp(MaxScale, ROUND_HALF_AWAY_FROM_ZERO).Sub(One); !got.Equal(x) {
		t.Errorf("Exp(%s)-1 = %s, want %s - exp(x) is exact at this width, so the composition must agree",
			x.StringFixed(), got.StringFixed(), x.StringFixed())
	}
}

// TestPowDispatch checks that each arm answers, by comparing against the method it should have reached.
func TestPowDispatch(t *testing.T) {
	d := FromString("1.05")

	// an integer exponent is PowIntRound, bit for bit
	if got, want := d.Pow(FromInt64(360), 12, ROUND_BANK), d.PowIntRound(360, 12, ROUND_BANK); got != want {
		t.Errorf("Pow with an integer exponent = %s, PowIntRound = %s", got.StringFixed(), want.StringFixed())
	}
	// a short fraction is PowRational, bit for bit, so a square root arrives exactly
	if got, want := d.Pow(FromString("0.5"), 12, ROUND_BANK), d.PowRational(1, 2, 12, ROUND_BANK); got != want {
		t.Errorf("Pow with a short fraction = %s, PowRational = %s", got.StringFixed(), want.StringFixed())
	}
	if got, want := FromInt64(9).Pow(FromString("0.5"), 12, ROUND_BANK), FromInt64(9).SqrtRound(12, ROUND_BANK); got != want {
		t.Errorf("Pow(9, 0.5) = %s, SqrtRound(9) = %s", got.StringFixed(), want.StringFixed())
	}
	// 0.0625 is 1/16 however many places it is written at
	if got, want := d.Pow(FromString("0.06250000000"), 12, ROUND_BANK), d.PowRational(1, 16, 12, ROUND_BANK); got != want {
		t.Errorf("Pow(0.0625...) = %s, PowRational(1, 16) = %s", got.StringFixed(), want.StringFixed())
	}
	// and a long exponent goes the general way
	long := FromString("0.3333333333")
	if got, want := d.Pow(long, 12, ROUND_BANK), d.powExpLn(long, 12, ROUND_BANK); got != want {
		t.Errorf("Pow with a long exponent = %s, powExpLn = %s", got.StringFixed(), want.StringFixed())
	}
}

// TestTranscendentalContract pins the domains and the special values.
func TestTranscendentalContract(t *testing.T) {
	two, nan := FromInt64(2), NaN(state.Underflow)

	cases := []struct {
		name string
		got  Dec128
		want state.State
	}{
		{"Ln of zero", Zero.Ln(2, ROUND_BANK), state.DomainError},
		{"Ln of a negative", FromInt64(-1).Ln(2, ROUND_BANK), state.DomainError},
		{"Ln1p of -1", FromInt64(-1).Ln1p(2, ROUND_BANK), state.DomainError},
		{"Ln1p below -1", FromString("-1.5").Ln1p(2, ROUND_BANK), state.DomainError},
		{"Ln NaN propagates", nan.Ln(2, ROUND_BANK), state.Underflow},
		{"Ln scale", two.Ln(MaxScale+1, ROUND_BANK), state.ScaleOutOfRange},
		{"Ln mode", two.Ln(2, RoundingMode(99)), state.InvalidRoundingMode},
		{"Exp overflow", FromInt64(89).Exp(0, ROUND_BANK), state.Overflow},
		{"Exp scale", two.Exp(MaxScale+1, ROUND_BANK), state.ScaleOutOfRange},
		{"Expm1 overflow", FromInt64(89).Expm1(0, ROUND_BANK), state.Overflow},
		{"Pow d NaN first", nan.Pow(NaN(state.Inexact), 2, ROUND_BANK), state.Underflow},
		{"Pow e NaN second", two.Pow(NaN(state.Inexact), 2, ROUND_BANK), state.Inexact},
		{"Pow negative base, fractional exponent", FromInt64(-2).Pow(FromString("0.3333333333"), 2, ROUND_BANK), state.DomainError},
		{"Pow zero base, negative exponent", Zero.Pow(FromString("-0.3333333333"), 2, ROUND_BANK), state.DivisionByZero},
		{"Pow scale", two.Pow(two, MaxScale+1, ROUND_BANK), state.ScaleOutOfRange},
		{"Pow mode", two.Pow(two, 2, RoundingMode(99)), state.InvalidRoundingMode},
	}
	for _, c := range cases {
		if c.got.ErrorDetails() != c.want.Error() {
			t.Errorf("%s: got %v, want %v", c.name, c.got.ErrorDetails(), c.want.Error())
		}
	}

	// A negative base with an integer exponent is fine, and keeps its sign by parity.
	if got := FromInt64(-2).Pow(FromInt64(3), 0, ROUND_BANK); !got.Equal(FromInt64(-8)) {
		t.Errorf("(-2)^3 = %s, want -8", got.StringFixed())
	}
	// Exp of a very negative argument rounds to zero, or to one unit away from it.
	if got := FromInt64(-60).Exp(4, ROUND_TOWARD_ZERO); got.IsNaN() || !got.IsZero() {
		t.Errorf("exp(-60) at 4 places = %s, want zero", got.StringFixed())
	}
	if got := FromInt64(-60).Exp(4, ROUND_AWAY_FROM_ZERO); !got.Equal(QuantumAtScale(4)) {
		t.Errorf("exp(-60) at 4 places away from zero = %s, want 0.0001", got.StringFixed())
	}
	// A zero base with a positive fractional exponent is zero, and is never negative.
	if got := Zero.Pow(FromString("0.3333333333"), 3, ROUND_BANK); got.IsNaN() || !got.IsZero() || got.state == state.Neg {
		t.Errorf("0^(1/3) = %s state %v, want 0.000", got.StringFixed(), got.state)
	}
}

// TestTranscendentalRangeEnds covers the branches that only the far ends of the argument range reach: the short
// circuits that refuse before the series runs, and the carry out of a coefficient at the very top of Exp's range.
func TestTranscendentalRangeEnds(t *testing.T) {
	// Far enough out that the reduction alone settles it, without running the series.
	for _, mode := range allModes {
		if got := FromInt64(200).Exp(4, mode); got.ErrorDetails() != state.Overflow.Error() {
			t.Errorf("exp(200) under %v = %v, want NaN(Overflow)", mode, got.ErrorDetails())
		}
		if got := FromInt64(200).Expm1(4, mode); got.ErrorDetails() != state.Overflow.Error() {
			t.Errorf("expm1(200) under %v = %v, want NaN(Overflow)", mode, got.ErrorDetails())
		}
		if got := FromInt64(2).Pow(FromString("700.3333333333"), 4, mode); got.ErrorDetails() != state.Overflow.Error() {
			t.Errorf("2^700.33 under %v = %v, want NaN(Overflow)", mode, got.ErrorDetails())
		}
	}

	// And far enough the other way that the result is below the quantum of any scale.
	for _, c := range []struct {
		name string
		got  Dec128
		want string
	}{
		{"exp(-200) truncated", FromInt64(-200).Exp(4, ROUND_TOWARD_ZERO), "0.0000"},
		{"exp(-200) away from zero", FromInt64(-200).Exp(4, ROUND_AWAY_FROM_ZERO), "0.0001"},
		{"exp(-200) toward positive", FromInt64(-200).Exp(4, ROUND_UP), "0.0001"},
		{"2^-700.33 truncated", FromInt64(2).Pow(FromString("-700.3333333333"), 4, ROUND_TOWARD_ZERO), "0.0000"},
		{"2^-700.33 away from zero", FromInt64(2).Pow(FromString("-700.3333333333"), 4, ROUND_AWAY_FROM_ZERO), "0.0001"},
	} {
		if c.got.IsNaN() || c.got.StringFixed() != c.want {
			t.Errorf("%s = %s (%v), want %s", c.name, c.got.StringFixed(), c.got.ErrorDetails(), c.want)
		}
	}
	// ROUND_NAN refuses rather than choosing a direction, on both of those paths.
	if got := FromInt64(-200).Exp(4, ROUND_NAN); got.ErrorDetails() != state.Inexact.Error() {
		t.Errorf("exp(-200) under ROUND_NAN = %v, want NaN(Inexact)", got.ErrorDetails())
	}
	if got := FromInt64(2).Pow(FromString("-700.3333333333"), 4, ROUND_NAN); got.ErrorDetails() != state.Inexact.Error() {
		t.Errorf("2^-700.33 under ROUND_NAN = %v, want NaN(Inexact)", got.ErrorDetails())
	}
	// expm1 of a very negative argument is -1: exp(d) is indistinguishable from zero there.
	if got := FromInt64(-200).Expm1(4, ROUND_BANK); !got.Equal(FromInt64(-1)) {
		t.Errorf("expm1(-200) = %s, want -1", got.StringFixed())
	}

	// The top of Exp's range: ln(2^128) is 88.7228391116729996054..., so an argument just below it is the largest
	// value the function returns and one just above it overflows. Note what is *not* tested here, because it cannot
	// be: the carry out of a coefficient when the rounded-up result leaves the range. For that the true value would
	// have to lie within one unit in the last place below 2^128, a window 3e-48 wide in the argument, where the
	// finest argument this type can express moves in steps of 1e-19. No representable argument lands there, for Exp
	// or for any of the others, which is why that branch of fixedToDec has no test.
	if got := FromString("88.7228391").Exp(0, ROUND_TOWARD_ZERO); got.IsNaN() {
		t.Errorf("exp(88.7228391) at scale 0 = NaN(%v), want a value near the top of the range", got.ErrorDetails())
	}
	if got := FromString("88.7228392").Exp(0, ROUND_TOWARD_ZERO); got.ErrorDetails() != state.Overflow.Error() {
		t.Errorf("exp(88.7228392) = %s, want NaN(Overflow)", got.StringFixed())
	}
}

// TestPowHugeIntegerExponent covers the arm for an integer exponent too large for an int64, which no real
// calculation reaches but which the type's range allows a caller to write.
func TestPowHugeIntegerExponent(t *testing.T) {
	huge := FromString("10000000000000000000")     // 10^19, past the int64 range
	negHuge := FromString("-10000000000000000000") // and the same with a sign

	// |d| > 1 with a positive exponent leaves the range, and with a negative one collapses to zero.
	if got := FromInt64(2).Pow(huge, 2, ROUND_BANK); got.ErrorDetails() != state.Overflow.Error() {
		t.Errorf("2^1e19 = %v, want NaN(Overflow)", got.ErrorDetails())
	}
	if got := FromInt64(2).Pow(negHuge, 2, ROUND_TOWARD_ZERO); got.IsNaN() || !got.IsZero() {
		t.Errorf("2^-1e19 = %s, want zero", got.StringFixed())
	}
	if got := FromInt64(2).Pow(negHuge, 2, ROUND_AWAY_FROM_ZERO); !got.Equal(QuantumAtScale(2)) {
		t.Errorf("2^-1e19 away from zero = %s, want 0.01", got.StringFixed())
	}
	if got := FromInt64(2).Pow(negHuge, 2, ROUND_NAN); got.ErrorDetails() != state.Inexact.Error() {
		t.Errorf("2^-1e19 under ROUND_NAN = %v, want NaN(Inexact)", got.ErrorDetails())
	}
	// |d| < 1 is the mirror image.
	if got := FromString("0.5").Pow(huge, 2, ROUND_TOWARD_ZERO); got.IsNaN() || !got.IsZero() {
		t.Errorf("0.5^1e19 = %s, want zero", got.StringFixed())
	}
	if got := FromString("0.5").Pow(negHuge, 2, ROUND_BANK); got.ErrorDetails() != state.Overflow.Error() {
		t.Errorf("0.5^-1e19 = %v, want NaN(Overflow)", got.ErrorDetails())
	}
	// A base of one in magnitude is the only one whose power at that exponent is representable, and its sign is
	// still decided by the parity of the exponent.
	if got := One.Pow(huge, 3, ROUND_BANK); !got.Equal(One) || got.scale != 3 {
		t.Errorf("1^1e19 = %s at scale %d, want 1.000", got.StringFixed(), got.scale)
	}
	if got := FromInt64(-1).Pow(huge, 3, ROUND_BANK); !got.Equal(One) {
		t.Errorf("(-1)^1e19 = %s, want 1 (the exponent is even)", got.StringFixed())
	}
	odd := FromString("10000000000000000001")
	if got := FromInt64(-1).Pow(odd, 3, ROUND_BANK); !got.Equal(FromInt64(-1)) {
		t.Errorf("(-1)^(1e19+1) = %s, want -1 (the exponent is odd)", got.StringFixed())
	}
	// Zero to a huge power is zero, and to a huge negative power is a division by zero.
	if got := Zero.Pow(huge, 2, ROUND_BANK); got.IsNaN() || !got.IsZero() {
		t.Errorf("0^1e19 = %s, want zero", got.StringFixed())
	}
	if got := Zero.Pow(negHuge, 2, ROUND_BANK); got.ErrorDetails() != state.DivisionByZero.Error() {
		t.Errorf("0^-1e19 = %v, want NaN(DivisionByZero)", got.ErrorDetails())
	}
}

// TestPowWideExponent covers the exponent that is too wide to be a short fraction whatever it reduces to: a
// coefficient past 64 bits cannot be below maxRootDegree, so the dispatch goes straight to the general arm without
// attempting the reduction.
func TestPowWideExponent(t *testing.T) {
	// 39 digits, so the coefficient needs more than 64 bits, and a fractional part, so it is not an integer.
	e := FromString("1234567890123456789.0123456789012345678")
	if e.IsNaN() || e.coef.Hi == 0 {
		t.Fatalf("the exponent is not the wide one this test needs: %s", e.StringFixed())
	}
	if _, _, ok := rationalExponent(e); ok {
		t.Error("an exponent with a coefficient past 64 bits must not be taken as a short fraction")
	}

	// A base just above one keeps the result in range at an exponent that size.
	d := FromString("1.0000000000000000001")
	got := d.Pow(e, MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	if got.IsNaN() {
		t.Fatalf("Pow with a wide exponent = NaN(%v), want a value", got.ErrorDetails())
	}
	// ln(d) is about 1e-19, so the result is about e^0.1234...
	if got.Compare(One) <= 0 || got.Compare(FromInt64(2)) >= 0 {
		t.Errorf("Pow with a wide exponent = %s, which is not between 1 and 2", got.StringFixed())
	}
	if want := d.powExpLn(e, MaxScale, ROUND_HALF_AWAY_FROM_ZERO); got != want {
		t.Errorf("the dispatch did not reach the general arm: %s vs %s", got.StringFixed(), want.StringFixed())
	}
}
