package dec128

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"strings"
	"testing"

	"github.com/jokruger/dec128/state"
)

// Random-operand oracles for the operations that arith_crosscheck_test.go does not cover: the exact division pair
// (QuoRem, Mod), the explicit rounding methods, the order relation, and the text and interchange round trips.
// randDec, bigAtScale, bigPow10, roundBig, big2p128 and allModes come from arith_crosscheck_test.go.

// exactCoef returns the signed coefficient of d at its own scale.
func exactCoef(d Dec128) *big.Int {
	return bigAtScale(d, d.scale)
}

// TestQuoRemModAgainstBig checks the exact division pair against math/big: the quotient is the truncated integer
// quotient, the remainder carries the sign of the dividend and the larger scale of the two operands, and
// q*other + r reproduces the dividend exactly. Mod must return the same remainder QuoRem does.
func TestQuoRemModAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260913))
	var overflows, zeroRems int

	for i := range 100000 {
		a, b := randDec(r), randDec(r)
		q, rem := a.QuoRem(b)

		if b.coef.IsZero() {
			if q.ErrorDetails() != state.DivisionByZero.Error() || rem.ErrorDetails() != state.DivisionByZero.Error() {
				t.Fatalf("%d: %s / %s: want NaN(DivisionByZero), got %v / %v", i, a.StringFixed(), b.StringFixed(), q.ErrorDetails(), rem.ErrorDetails())
			}
			continue
		}
		if q.IsNaN() {
			// The only other failure is a quotient of 129 bits or more, which no rescaling can fix.
			if q.ErrorDetails() != state.Overflow.Error() {
				t.Fatalf("%d: %s / %s = NaN(%v), want NaN(Overflow)", i, a.StringFixed(), b.StringFixed(), q.ErrorDetails())
			}
			overflows++
			continue
		}

		scale := max(a.scale, b.scale)
		ba, bb := bigAtScale(a, scale), bigAtScale(b, scale)
		wantQ, wantR := new(big.Int).QuoRem(ba, bb, new(big.Int))

		switch {
		case exactCoef(q).Cmp(wantQ) != 0:
			t.Fatalf("%d: QuoRem(%s, %s) quotient = %s, want %s", i, a.StringFixed(), b.StringFixed(), q.StringFixed(), wantQ)
		case q.scale != 0:
			t.Fatalf("%d: QuoRem(%s, %s) quotient scale = %d, want 0", i, a.StringFixed(), b.StringFixed(), q.scale)
		case rem.scale != scale:
			t.Fatalf("%d: QuoRem(%s, %s) remainder scale = %d, want %d", i, a.StringFixed(), b.StringFixed(), rem.scale, scale)
		case bigAtScale(rem, scale).Cmp(wantR) != 0:
			t.Fatalf("%d: QuoRem(%s, %s) remainder = %s, want %s at scale %d", i, a.StringFixed(), b.StringFixed(), rem.StringFixed(), wantR, scale)
		}

		// a = q*b + r, exactly
		back := new(big.Int).Mul(wantQ, bb)
		back.Add(back, wantR)
		if back.Cmp(ba) != 0 {
			t.Fatalf("%d: QuoRem(%s, %s) does not reconstruct the dividend", i, a.StringFixed(), b.StringFixed())
		}

		if m := a.Mod(b); !m.Equal(rem) || m.scale != rem.scale {
			t.Fatalf("%d: Mod(%s, %s) = %s at scale %d, QuoRem remainder = %s at scale %d",
				i, a.StringFixed(), b.StringFixed(), m.StringFixed(), m.scale, rem.StringFixed(), rem.scale)
		}
		if rem.coef.IsZero() {
			if rem.state == state.Neg {
				t.Fatalf("%d: zero remainder is negative", i)
			}
			zeroRems++
		}
	}

	if overflows == 0 || zeroRems == 0 {
		t.Errorf("operand mix no longer reaches the edge cases (overflows=%d zero remainders=%d)", overflows, zeroRems)
	}
}

// roundOracle rounds the exact coefficient v of a value at scale from to the target scale with mode, in math/big.
func roundOracle(v *big.Int, from, target uint8, mode RoundingMode) (*big.Int, bool) {
	den := bigPow10(from - target)
	neg := v.Sign() < 0
	q, r := new(big.Int).QuoRem(new(big.Int).Abs(v), den, new(big.Int))
	out := roundBig(q, r, den, neg, mode)
	if neg && out.Sign() != 0 {
		out = new(big.Int).Neg(out)
	}
	return out, r.Sign() != 0
}

// TestRoundAllModesAgainstBig checks Round, every Round* method it dispatches to, and RescaleRound against math/big
// for random values and every mode. It also pins the documented difference between the two families: Round* and Round
// leave a value that is already shorter alone, while RescaleRound produces exactly the scale asked for.
func TestRoundAllModesAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260914))
	for i := range 40000 {
		d := randDec(r)
		target := uint8(r.Intn(int(MaxScale) + 1))

		for _, mode := range allModes {
			got := d.Round(target, mode)

			if target >= d.scale {
				// unchanged, whatever the mode
				if got != d {
					t.Fatalf("%d: Round(%s, %d, %v) = %s, want the value unchanged", i, d.StringFixed(), target, mode, got.StringFixed())
				}
				// RescaleRound pads instead of leaving it alone
				if rr := d.RescaleRound(target, mode); !rr.IsNaN() && rr.scale != target {
					t.Fatalf("%d: RescaleRound(%s, %d, %v) = %s at scale %d, want scale %d",
						i, d.StringFixed(), target, mode, rr.StringFixed(), rr.scale, target)
				}
				continue
			}

			want, inexact := roundOracle(exactCoef(d), d.scale, target, mode)

			if inexact && mode == ROUND_NAN {
				if got.ErrorDetails() != state.Inexact.Error() {
					t.Fatalf("%d: Round(%s, %d, ROUND_NAN) = %v, want NaN(Inexact)", i, d.StringFixed(), target, got.ErrorDetails())
				}
				continue
			}
			switch {
			case got.IsNaN():
				t.Fatalf("%d: Round(%s, %d, %v) = NaN(%v)", i, d.StringFixed(), target, mode, got.ErrorDetails())
			case got.scale != target:
				t.Fatalf("%d: Round(%s, %d, %v) = %s at scale %d", i, d.StringFixed(), target, mode, got.StringFixed(), got.scale)
			case exactCoef(got).Cmp(want) != 0:
				t.Fatalf("%d: Round(%s, %d, %v) = %s, want coefficient %s", i, d.StringFixed(), target, mode, got.StringFixed(), want)
			case got.coef.IsZero() && got.state == state.Neg:
				t.Fatalf("%d: Round(%s, %d, %v) produced a negative zero", i, d.StringFixed(), target, mode)
			}

			// RescaleRound must agree with Round when it is lowering the scale.
			if rr := d.RescaleRound(target, mode); rr != got {
				t.Fatalf("%d: RescaleRound(%s, %d, %v) = %s, Round = %s", i, d.StringFixed(), target, mode, rr.StringFixed(), got.StringFixed())
			}
		}

		// The mode-named methods must match the mode they are named after.
		if target < d.scale {
			pairs := []struct {
				mode RoundingMode
				got  Dec128
			}{
				{ROUND_TOWARD_ZERO, d.RoundTowardZero(target)},
				{ROUND_TOWARD_ZERO, d.Trunc(target)},
				{ROUND_DOWN, d.RoundDown(target)},
				{ROUND_UP, d.RoundUp(target)},
				{ROUND_AWAY_FROM_ZERO, d.RoundAwayFromZero(target)},
				{ROUND_HALF_TOWARD_ZERO, d.RoundHalfTowardZero(target)},
				{ROUND_HALF_AWAY_FROM_ZERO, d.RoundHalfAwayFromZero(target)},
				{ROUND_BANK, d.RoundBank(target)},
			}
			for _, p := range pairs {
				want, _ := roundOracle(exactCoef(d), d.scale, target, p.mode)
				if p.got.IsNaN() || p.got.scale != target || exactCoef(p.got).Cmp(want) != 0 {
					t.Fatalf("%d: %v method on %s to %d = %s, want coefficient %s", i, p.mode, d.StringFixed(), target, p.got.StringFixed(), want)
				}
			}
		}
	}
}

// TestTextRoundTripFuzz checks that every representable value survives its own text forms.
func TestTextRoundTripFuzz(t *testing.T) {
	r := rand.New(rand.NewSource(20260915))
	for i := range 100000 {
		d := randDec(r)

		// The fixed form carries the scale, so it must come back bit-identical through both parsers.
		fixed := d.StringFixed()
		if back := FromString(fixed); back != d {
			t.Fatalf("%d: FromString(%q) = %v, want %v", i, fixed, back, d)
		}
		if back := FromSafeString(fixed); back != d {
			t.Fatalf("%d: FromSafeString(%q) = %v, want %v", i, fixed, back, d)
		}
		// The trimmed and scientific forms drop trailing zeros, so they come back numerically equal
		// and identical to the canonical value.
		canon := d.Canonical()
		if back := FromString(d.String()); back != canon {
			t.Fatalf("%d: FromString(String(%v) = %q) = %v, want %v", i, d, d.String(), back, canon)
		}
		if back := FromString(d.StringSci()); back != canon {
			t.Fatalf("%d: FromString(StringSci(%v) = %q) = %v, want %v", i, d, d.StringSci(), back, canon)
		}
		// And the buffered forms must produce the same bytes as the allocating ones.
		buf := make([]byte, MaxSciStrLen)
		if s := string(d.StringToBuf(buf)); s != d.String() {
			t.Fatalf("%d: StringToBuf = %q, String = %q", i, s, d.String())
		}
		if s := string(d.StringFixedToBuf(buf)); s != fixed {
			t.Fatalf("%d: StringFixedToBuf = %q, StringFixed = %q", i, s, fixed)
		}
		if s := string(d.StringSciToBuf(buf)); s != d.StringSci() {
			t.Fatalf("%d: StringSciToBuf = %q, StringSci = %q", i, s, d.StringSci())
		}
	}
}

// TestCodecRoundTripFuzz checks the four binary interchange formats over random values. Each is exact for the values
// it claims to cover, so anything that goes out must come back identical - with one documented exception, noted below.
func TestCodecRoundTripFuzz(t *testing.T) {
	r := rand.New(rand.NewSource(20260916))
	var int128Fits, ieeeFits int

	for i := range 60000 {
		d := randDec(r)

		// The package's own format, which is bit-identical for every finite value including a zero with a scale.
		b, err := d.MarshalBinary()
		if err != nil {
			t.Fatalf("%d: MarshalBinary(%v): %v", i, d, err)
		}
		if n := d.BinarySize(); n != len(b) {
			t.Fatalf("%d: BinarySize(%v) = %d, MarshalBinary wrote %d", i, d, n, len(b))
		}
		var bin Dec128
		if err := bin.UnmarshalBinary(b); err != nil {
			t.Fatalf("%d: UnmarshalBinary(%v): %v", i, d, err)
		}
		if bin != d {
			t.Fatalf("%d: binary round trip of %v = %v", i, d, bin)
		}

		// PostgreSQL numeric: total for every finite value, and it does carry a zero's scale.
		pb, err := d.AppendPgNumeric(nil)
		if err != nil {
			t.Fatalf("%d: AppendPgNumeric(%v): %v", i, d, err)
		}
		var pg Dec128
		if err := pg.DecodePgNumeric(pb); err != nil {
			t.Fatalf("%d: DecodePgNumeric(%v): %v", i, d, err)
		}
		if pg != d {
			t.Fatalf("%d: pg round trip of %s = %s", i, d.StringFixed(), pg.StringFixed())
		}

		// int128: one bit narrower than Dec128, so FitsInt128 gates it.
		if d.FitsInt128(d.scale) {
			int128Fits++
			for _, order := range []binary.ByteOrder{binary.BigEndian, binary.LittleEndian} {
				ib, err := d.AppendInt128(nil, d.scale, order)
				if err != nil {
					t.Fatalf("%d: AppendInt128(%v): %v", i, d, err)
				}
				var got Dec128
				if err := got.DecodeInt128(ib, d.scale, order); err != nil {
					t.Fatalf("%d: DecodeInt128(%v): %v", i, d, err)
				}
				if got != d {
					t.Fatalf("%d: int128 round trip of %s = %s", i, d.StringFixed(), got.StringFixed())
				}
			}
		} else if _, err := d.EncodeInt128(make([]byte, Int128Bytes), d.scale, binary.BigEndian); err == nil {
			t.Fatalf("%d: FitsInt128 says no but EncodeInt128 succeeded for %v", i, d)
		}

		// IEEE decimal128: 34 significant digits, so FitsIEEE gates it.
		if d.FitsIEEE() {
			ieeeFits++
			eb, err := d.AppendIEEE(nil)
			if err != nil {
				t.Fatalf("%d: AppendIEEE(%v): %v", i, d, err)
			}
			var got Dec128
			if err := got.DecodeIEEE(eb); err != nil {
				t.Fatalf("%d: DecodeIEEE(%v): %v", i, d, err)
			}
			if got != d {
				t.Fatalf("%d: ieee round trip of %s = %s", i, d.StringFixed(), got.StringFixed())
			}
		}
	}

	if int128Fits == 0 || ieeeFits == 0 {
		t.Errorf("operand mix never fits the narrower formats (int128=%d ieee=%d)", int128Fits, ieeeFits)
	}
}

// TestSciParsingAgainstBigRat parses random scientific-notation literals and compares the value against big.Rat.
// The mantissa is allowed to run past what a coefficient can hold and the exponent to swing either way, so this walks
// the padding and exponent-folding paths in fromSciString and applyExp.
func TestSciParsingAgainstBigRat(t *testing.T) {
	r := rand.New(rand.NewSource(20260917))
	var accepted int

	check := func(s string) {
		t.Helper()
		want, ok := new(big.Rat).SetString(s)
		if !ok {
			t.Fatalf("the test generated a literal big.Rat rejects: %q", s)
		}
		d := FromString(s)
		if d.IsNaN() {
			return // out of range for this type; the accept/reject boundary is not what this test is about
		}
		accepted++
		got := new(big.Rat).SetFrac(exactCoef(d), bigPow10(d.scale))
		if got.Cmp(want) != 0 {
			t.Fatalf("FromString(%q) = %s at scale %d, want %s", s, got.RatString(), d.scale, want.RatString())
		}
	}

	for _, s := range []string{
		"5e-3", "5E-3", "5e+3", "0.005e0", "1.50e0", "1.5e1", "15e-1", "-1.5e-2", "+1.5e2",
		"0e100", "0e-100", "-0e5", "1e19", "1e-19", "1e0",
		"0.0000000000000000000000000000000000000000005e40",
		"0.000000000000000000000000000000000000000000000005e45",
		"3.40282366920938463463374607431768211455e38",
		"3.40282366920938463463374607431768211455e-1",
		"9.9999999999999999999999999999999999999e37",
	} {
		check(s)
	}

	for range 100000 {
		digits := 1 + r.Intn(41)
		var sb strings.Builder
		if r.Intn(2) == 0 {
			sb.WriteByte('-')
		}
		point := r.Intn(digits + 1)
		for i := range digits {
			if i == point && i != 0 {
				sb.WriteByte('.')
			}
			sb.WriteByte(byte('0' + r.Intn(10)))
		}
		check(sb.String() + fmt.Sprintf("e%+d", r.Intn(90)-45))
	}

	if accepted < 1000 {
		t.Errorf("only %d of the generated literals were accepted; the test is not exercising much", accepted)
	}
}

// TestCompareIsTotalOrder checks that Compare is a total order consistent with Equal, including across scales and with
// NaN values mixed in, and that sorting, Min and Max all agree with it.
func TestCompareIsTotalOrder(t *testing.T) {
	nanReasons := []state.State{state.Error, state.NaN, state.DivisionByZero, state.Overflow, state.InvalidFormat, state.Null}
	r := rand.New(rand.NewSource(20260918))

	for range 2000 {
		xs := make([]Dec128, 12)
		for i := range xs {
			if r.Intn(8) == 0 {
				xs[i] = NaN(nanReasons[r.Intn(len(nanReasons))])
			} else {
				xs[i] = randDec(r)
			}
		}

		for i := range xs {
			for j := range xs {
				cij, cji := xs[i].Compare(xs[j]), xs[j].Compare(xs[i])
				if cij != -cji {
					t.Fatalf("Compare is not antisymmetric for %v and %v: %d and %d", xs[i], xs[j], cij, cji)
				}
				if (cij == 0) != xs[i].Equal(xs[j]) {
					t.Fatalf("Compare and Equal disagree for %s and %s: %d and %v",
						xs[i].StringFixed(), xs[j].StringFixed(), cij, xs[i].Equal(xs[j]))
				}
				// the other relations are defined in terms of Compare and must not drift from it
				if xs[i].LessThan(xs[j]) != (cij < 0) || xs[i].GreaterThan(xs[j]) != (cij > 0) ||
					xs[i].LessThanOrEqual(xs[j]) != (cij <= 0) || xs[i].GreaterThanOrEqual(xs[j]) != (cij >= 0) {
					t.Fatalf("relational methods disagree with Compare for %v and %v", xs[i], xs[j])
				}
			}
		}

		sort.SliceStable(xs, func(i, j int) bool { return xs[i].Compare(xs[j]) < 0 })
		for i := 1; i < len(xs); i++ {
			if xs[i-1].Compare(xs[i]) > 0 {
				t.Fatalf("sort produced a descending pair: %v then %v", xs[i-1], xs[i])
			}
		}
		// Min and Max return the first NaN in argument order, so they only track the sorted ends
		// when no argument is NaN.
		anyNaN := false
		for _, x := range xs {
			anyNaN = anyNaN || x.IsNaN()
		}
		if !anyNaN {
			if lo := Min(xs[0], xs[1:]...); !lo.Equal(xs[0]) {
				t.Fatalf("Min = %s, smallest = %s", lo.StringFixed(), xs[0].StringFixed())
			}
			if hi := Max(xs[0], xs[1:]...); !hi.Equal(xs[len(xs)-1]) {
				t.Fatalf("Max = %s, largest = %s", hi.StringFixed(), xs[len(xs)-1].StringFixed())
			}
		}
	}
}

// TestAvgWithinOperandRange checks the mean against math/big and asserts the property that catches a bad Sum or a bad
// division by the count: the mean of a set never falls outside it.
func TestAvgWithinOperandRange(t *testing.T) {
	// Avg divides with Div, so its result depends on the default scale; pin it rather than inherit whatever an
	// earlier test left behind.
	defer SetDefaultScale(DefaultScale())
	defer SetArithmeticRounding(ArithmeticRounding())
	SetDefaultScale(MaxScale)
	SetArithmeticRounding(ROUND_TOWARD_ZERO)

	r := rand.New(rand.NewSource(20260919))
	var checked int

	for range 30000 {
		n := 1 + r.Intn(5)
		ds := make([]Dec128, n)
		scale := uint8(0)
		for i := range ds {
			ds[i] = randDec(r)
			scale = max(scale, ds[i].scale)
		}

		avg := Avg(ds[0], ds[1:]...)
		if avg.IsNaN() {
			continue
		}

		// Avg is Sum followed by Div, so it must be exactly that.
		total := Sum(ds[0], ds[1:]...)
		if !avg.Equal(total.DivInt64(int64(n))) {
			t.Fatalf("Avg = %s but Sum/n = %s", avg.StringFixed(), total.DivInt64(int64(n)).StringFixed())
		}

		// The remaining properties hold only when the total itself needed no reduction: once Sum has had to drop
		// digits the mean inherits that error, and neither the range nor the one-quantum bound is guaranteed.
		if total.IsNaN() || total.scale != scale {
			continue
		}

		lo, hi := Min(ds[0], ds[1:]...), Max(ds[0], ds[1:]...)
		if avg.LessThan(lo) || avg.GreaterThan(hi) {
			t.Fatalf("Avg = %s falls outside [%s, %s]", avg.StringFixed(), lo.StringFixed(), hi.StringFixed())
		}
		exact := new(big.Rat)
		for _, d := range ds {
			exact.Add(exact, new(big.Rat).SetFrac(bigAtScale(d, scale), bigPow10(scale)))
		}
		exact.Quo(exact, new(big.Rat).SetInt64(int64(n)))
		got := new(big.Rat).SetFrac(exactCoef(avg), bigPow10(avg.scale))
		diff := new(big.Rat).Abs(new(big.Rat).Sub(got, exact))
		if diff.Cmp(new(big.Rat).SetFrac(big.NewInt(1), bigPow10(avg.scale))) >= 0 {
			t.Fatalf("Avg = %s is %s away from the exact mean %s", avg.StringFixed(), diff.FloatString(25), exact.FloatString(25))
		}
		checked++
	}

	if checked == 0 {
		t.Error("no mean was checked")
	}
}

// TestPowIntAgainstBig checks the exact powers against math/big. A power whose exact form does not fit is rounded to
// fit like Mul and is covered by the fit oracle instead, so only the exact cases are compared here.
func TestPowIntAgainstBig(t *testing.T) {
	r := rand.New(rand.NewSource(20260920))
	var exactCases int

	for i := range 40000 {
		d := randDec(r)
		n := r.Intn(8)
		got := d.PowInt64(int64(n))

		if n == 0 {
			if !got.Equal(One) {
				t.Fatalf("%d: (%s)^0 = %s, want 1", i, d.StringFixed(), got.StringFixed())
			}
			continue
		}
		if got.IsNaN() || int(d.scale)*n > int(MaxScale) || got.scale != uint8(int(d.scale)*n) {
			continue // rounded to fit, or overflowed
		}
		want := new(big.Int).Exp(exactCoef(d), big.NewInt(int64(n)), nil)
		if exactCoef(got).Cmp(want) != 0 {
			t.Fatalf("%d: (%s)^%d = %s, want coefficient %s at scale %d", i, d.StringFixed(), n, got.StringFixed(), want, got.scale)
		}
		exactCases++

		// PowInt must agree with PowInt64, and squaring must agree with Mul.
		if !d.PowInt(n).Equal(got) {
			t.Fatalf("%d: PowInt and PowInt64 disagree for (%s)^%d", i, d.StringFixed(), n)
		}
		if n == 2 && !d.Mul(d).Equal(got) {
			t.Fatalf("%d: (%s)^2 = %s, Mul = %s", i, d.StringFixed(), got.StringFixed(), d.Mul(d).StringFixed())
		}
	}

	if exactCases == 0 {
		t.Error("no exact power was compared")
	}
}
