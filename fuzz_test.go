package dec128

import (
	"encoding/binary"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Fuzz targets for the two untrusted surfaces: the text parsers and the interchange decoders. Nothing here may panic -
// the package's contract is that a failure is a NaN value, never a panic and never an error from arithmetic - and a
// value that parses must be the value that was written.
//
// During a normal `go test` these run over the seed corpus and testdata/fuzz only; use
// `go test -fuzz FuzzFromString -fuzztime 60s` to search.

func FuzzFromString(f *testing.F) {
	for _, s := range []string{
		"", "0", "-0", "1.5", "1.", ".5", "+1", "1e5", "1E-5", "NaN", "Infinity", "-Infinity",
		"340282366920938463463374607431768211455", "1e-20", "0.0000000000000000000000001e25",
		"1.50000000000000000000", "1234567890123456789012345678901234567890e-2",
		"+.", "--1", "1e", "1e+", "1.2.3", "1e5e6",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		d := FromString(s)

		// every output form has to work on whatever came back
		_ = d.String()
		_ = d.StringFixed()
		_ = d.StringSci()
		_, _ = d.MarshalJSON()
		_, _ = d.MarshalText()
		_, _ = d.MarshalBinary()
		_, _ = d.Value()
		_ = d.Canonical()
		if d.IsNaN() {
			return
		}

		// a parsed value round-trips through its own fixed text, bit for bit
		if back := FromString(d.StringFixed()); back != d {
			t.Fatalf("FromString(%q) = %v, whose StringFixed %q reparses to %v", s, d, d.StringFixed(), back)
		}
		// and it is the number that was written
		if r, ok := new(big.Rat).SetString(strings.TrimSuffix(s, ".")); ok {
			got := new(big.Rat).SetFrac(bigAtScale(d, d.Scale()), bigPow10(d.Scale()))
			if got.Cmp(r) != 0 {
				t.Fatalf("FromString(%q) = %s, want %s", s, got.RatString(), r.RatString())
			}
		}
	})
}

func FuzzDecoders(f *testing.F) {
	f.Add([]byte{0}, []byte{0, 0, 0, 0, 0, 0, 0, 0}, uint8(0))
	f.Add([]byte{0x20, 2}, []byte("1.5"), uint8(2))

	f.Fuzz(func(t *testing.T, own []byte, other []byte, scale uint8) {
		var d Dec128
		_, _ = d.DecodeBinary(own)
		_ = d.UnmarshalBinary(own)
		_ = d.DecodePgNumeric(other)
		_ = d.DecodeIEEE(other)
		_ = d.DecodeInt128(other, scale, binary.BigEndian)
		_ = d.DecodeInt128(other, scale, binary.LittleEndian)
		_ = d.UnmarshalJSON(other)
		_ = d.UnmarshalText(other)
		_ = d.Scan(other)

		// whatever a decoder left behind must be usable by everything downstream
		_ = d.String()
		_ = d.StringFixed()
		_ = d.StringSci()
		_ = d.Add(One)
		_ = d.Sub(Decimal2)
		_ = d.Mul(Decimal3)
		_ = d.Div(Decimal7)
		_ = d.Mod(Decimal3)
		_, _ = d.QuoRem(Decimal3)
		_ = d.Sqrt()
		_ = d.PowInt64(3)
		_ = d.Canonical()
		_ = d.Round(2, ROUND_BANK)
		_ = d.Rescale(MaxScale)
		_ = d.NextUp()
		_ = d.NextDown()
		_, _ = d.Int64()
		_, _ = d.MarshalJSON()
		_, _ = d.EncodePgNumeric(make([]byte, MaxPgNumericBytes))
		_, _ = d.EncodeIEEE(make([]byte, IEEEBytes))
		_, _ = d.EncodeInt128(make([]byte, Int128Bytes), scale, binary.BigEndian)
	})
}

// Per-operation targets. The two above cover the untrusted surfaces; these carry an oracle for each operation, so the
// engine's coverage guidance drives it into the branch it has not reached yet rather than into a generic smoke test.
// udecimal and zerodecimal both fuzz per operation this way.

// fuzzDec builds a value from fuzzer-supplied limbs. The scale is folded into range rather than rejected, so no input
// is wasted, and a zero is never negative.
func fuzzDec(hi, lo uint64, scale uint8, neg bool) Dec128 {
	d := Dec128{coef: uint128.Uint128{Lo: lo, Hi: hi}, scale: scale % (MaxScale + 1)}
	if neg && !d.coef.IsZero() {
		d.state = state.Neg
	}
	return d
}

func fuzzMode(b uint8) RoundingMode {
	return RoundingMode(b % (uint8(ROUND_BANK) + 1))
}

func addOperandSeeds(f *testing.F, extra ...any) {
	limbs := []uint64{0, 1, 2, 9, math.MaxUint64, math.MaxUint64 - 1, 1 << 63, 10000000000000000000}
	for _, hi := range limbs[:4] {
		for _, lo := range limbs {
			for _, s := range []uint8{0, 2, 6, MaxScale} {
				args := append([]any{hi, lo, s, true, lo, hi, s, false}, extra...)
				f.Add(args...)
			}
		}
	}
}

func FuzzAddSub(f *testing.F) {
	addOperandSeeds(f, uint8(0))
	f.Fuzz(func(t *testing.T, xh, xl uint64, xs uint8, xn bool, yh, yl uint64, ys uint8, yn bool, m uint8) {
		x, y := fuzzDec(xh, xl, xs, xn), fuzzDec(yh, yl, ys, yn)
		mode := fuzzMode(m)

		defer SetArithmeticRounding(ArithmeticRounding())
		SetArithmeticRounding(mode)

		scale := max(x.scale, y.scale)
		bx, by := bigAtScale(x, scale), bigAtScale(y, scale)
		checkFit(t, "Add", x, y, x.Add(y), new(big.Int).Add(bx, by), scale, mode)
		checkFit(t, "Sub", x, y, x.Sub(y), new(big.Int).Sub(bx, by), scale, mode)

		// Add is commutative whatever the mode: the exact sum is the same value with the same sign, so it reduces
		// the same way.
		if a, b := x.Add(y), y.Add(x); !a.Equal(b) {
			t.Fatalf("Add is not commutative for %s and %s: %s vs %s", x.StringFixed(), y.StringFixed(), a.StringFixed(), b.StringFixed())
		}
		// x-y == -(y-x) needs the two to agree on which way to round, so it holds for the sign-symmetric modes
		// always, and for the directed ones only when nothing was discarded. ROUND_UP on a difference that had to be
		// reduced truncates one side's magnitude and grows the other's, which is correct and not an identity.
		symmetric := mode != ROUND_DOWN && mode != ROUND_UP
		a, b := x.Sub(y), y.Sub(x).Neg()
		if !a.IsNaN() && !b.IsNaN() && (symmetric || a.scale == scale) && !a.Equal(b) {
			t.Fatalf("x-y != -(y-x) for %s and %s under %v: %s vs %s",
				x.StringFixed(), y.StringFixed(), mode, a.StringFixed(), b.StringFixed())
		}
	})
}

func FuzzMulModes(f *testing.F) {
	addOperandSeeds(f, uint8(0), uint8(0))
	f.Fuzz(func(t *testing.T, xh, xl uint64, xs uint8, xn bool, yh, yl uint64, ys uint8, yn bool, m, target uint8) {
		x, y := fuzzDec(xh, xl, xs, xn), fuzzDec(yh, yl, ys, yn)
		mode := fuzzMode(m)

		defer SetArithmeticRounding(ArithmeticRounding())
		SetArithmeticRounding(mode)

		exact := new(big.Int).Mul(bigAtScale(x, x.scale), bigAtScale(y, y.scale))
		checkFit(t, "Mul", x, y, x.Mul(y), exact, x.scale+y.scale, mode)

		// MulRound at an exact scale, against the same exact product
		scale := target % (MaxScale + 1)
		got := x.MulRound(y, scale, mode)
		want, inexact := roundQuot(exact, bigPow10(x.scale+y.scale-min(x.scale+y.scale, scale)), mode)
		if scale >= x.scale+y.scale {
			want = new(big.Int).Mul(exact, bigPow10(scale-(x.scale+y.scale)))
			inexact = false
		}
		switch {
		case new(big.Int).Abs(want).Cmp(big2p128) >= 0:
			if got.state != state.Overflow {
				t.Fatalf("MulRound(%s, %s, %d, %v) = %v, want NaN(Overflow)", x.StringFixed(), y.StringFixed(), scale, mode, got)
			}
		case inexact && mode == ROUND_NAN:
			if got.state != state.Inexact {
				t.Fatalf("MulRound(%s, %s, %d, ROUND_NAN) = %v, want NaN(Inexact)", x.StringFixed(), y.StringFixed(), scale, got)
			}
		case got.IsNaN():
			t.Fatalf("MulRound(%s, %s, %d, %v) = NaN(%v)", x.StringFixed(), y.StringFixed(), scale, mode, got.ErrorDetails())
		case got.scale != scale || exactCoef(got).Cmp(want) != 0:
			t.Fatalf("MulRound(%s, %s, %d, %v) = %s at scale %d, want coefficient %s",
				x.StringFixed(), y.StringFixed(), scale, mode, got.StringFixed(), got.scale, want)
		}
	})
}

func FuzzDivQuoRem(f *testing.F) {
	addOperandSeeds(f, uint8(0), uint8(0))
	f.Fuzz(func(t *testing.T, xh, xl uint64, xs uint8, xn bool, yh, yl uint64, ys uint8, yn bool, m, target uint8) {
		x, y := fuzzDec(xh, xl, xs, xn), fuzzDec(yh, yl, ys, yn)
		mode := fuzzMode(m)
		scale := target % (MaxScale + 1)

		if y.coef.IsZero() {
			for _, got := range []Dec128{x.Div(y), x.DivRound(y, scale, mode), x.Mod(y)} {
				if got.ErrorDetails() != state.DivisionByZero.Error() {
					t.Fatalf("dividing %s by zero = %v, want DivisionByZero", x.StringFixed(), got.ErrorDetails())
				}
			}
			return
		}

		// DivRound against the exact quotient at that scale
		want, neg, inexact := divOracle(x, y, scale, mode)
		checkQuotient(t, "DivRound", x, y, x.DivRound(y, scale, mode), scale, mode, want, neg, inexact, false)

		// QuoRem is exact, and reconstructs the dividend
		q, r := x.QuoRem(y)
		if !q.IsNaN() {
			common := max(x.scale, y.scale)
			bx, by := bigAtScale(x, common), bigAtScale(y, common)
			wq, wr := new(big.Int).QuoRem(bx, by, new(big.Int))
			if exactCoef(q).Cmp(wq) != 0 || bigAtScale(r, common).Cmp(wr) != 0 || r.scale != common {
				t.Fatalf("QuoRem(%s, %s) = (%s, %s), want (%s, %s at scale %d)",
					x.StringFixed(), y.StringFixed(), q.StringFixed(), r.StringFixed(), wq, wr, common)
			}
			if m := x.Mod(y); !m.Equal(r) || m.scale != r.scale {
				t.Fatalf("Mod and QuoRem disagree for %s and %s", x.StringFixed(), y.StringFixed())
			}
		}
	})
}

func FuzzSqrtRound(f *testing.F) {
	for _, hi := range []uint64{0, 1, math.MaxUint64} {
		for _, lo := range []uint64{0, 1, 2, 4, 100, math.MaxUint64} {
			for _, s := range []uint8{0, 2, 6, MaxScale} {
				f.Add(hi, lo, s, uint8(0), uint8(0))
			}
		}
	}
	f.Fuzz(func(t *testing.T, hi, lo uint64, s, target, m uint8) {
		d := fuzzDec(hi, lo, s, false)
		scale := target % (MaxScale + 1)
		got := d.SqrtRound(scale, fuzzMode(m))
		if got.IsNaN() {
			if fuzzMode(m) != ROUND_NAN {
				t.Fatalf("SqrtRound(%s, %d, %v) = NaN(%v)", d.StringFixed(), scale, fuzzMode(m), got.ErrorDetails())
			}
			return
		}
		// The result is within one quantum of the true root, whichever way the mode moved the last digit:
		// (q-1)^2 <= d <= (q+1)^2, compared at a common scale as k^2 * 10^d.scale against coef * 10^(2*scale).
		bq, bd := exactCoef(got), exactCoef(d)
		radicand := new(big.Int).Mul(bd, bigPow10(2*scale))
		square := func(k *big.Int) *big.Int {
			return new(big.Int).Mul(new(big.Int).Mul(k, k), bigPow10(d.scale))
		}
		if bq.Sign() > 0 {
			if lower := square(new(big.Int).Sub(bq, big.NewInt(1))); lower.Cmp(radicand) > 0 {
				t.Fatalf("SqrtRound(%s, %d, %v) = %s is more than one quantum above the root",
					d.StringFixed(), scale, fuzzMode(m), got.StringFixed())
			}
		}
		if upper := square(new(big.Int).Add(bq, big.NewInt(1))); upper.Cmp(radicand) < 0 {
			t.Fatalf("SqrtRound(%s, %d, %v) = %s is more than one quantum below the root",
				d.StringFixed(), scale, fuzzMode(m), got.StringFixed())
		}
	})
}

func FuzzRoundModes(f *testing.F) {
	for _, hi := range []uint64{0, 1, math.MaxUint64} {
		for _, lo := range []uint64{0, 1, 5, 50, math.MaxUint64} {
			for _, s := range []uint8{0, 2, 6, MaxScale} {
				f.Add(hi, lo, s, true, uint8(0), uint8(0))
			}
		}
	}
	f.Fuzz(func(t *testing.T, hi, lo uint64, s uint8, neg bool, target, m uint8) {
		d := fuzzDec(hi, lo, s, neg)
		places := target % (MaxScale + 1)
		mode := fuzzMode(m)
		got := d.Round(places, mode)

		if places >= d.scale {
			if got != d {
				t.Fatalf("Round(%s, %d, %v) = %v, want the value unchanged", d.StringFixed(), places, mode, got)
			}
			return
		}
		want, inexact := roundOracle(exactCoef(d), d.scale, places, mode)
		if inexact && mode == ROUND_NAN {
			if got.state != state.Inexact {
				t.Fatalf("Round(%s, %d, ROUND_NAN) = %v, want NaN(Inexact)", d.StringFixed(), places, got)
			}
			return
		}
		if got.IsNaN() || got.scale != places || exactCoef(got).Cmp(want) != 0 {
			t.Fatalf("Round(%s, %d, %v) = %v, want coefficient %s at scale %d", d.StringFixed(), places, mode, got, want, places)
		}
		if got.coef.IsZero() && got.state == state.Neg {
			t.Fatalf("Round(%s, %d, %v) produced a negative zero", d.StringFixed(), places, mode)
		}
		if rr := d.RescaleRound(places, mode); rr != got {
			t.Fatalf("RescaleRound and Round disagree for %s at %d places under %v", d.StringFixed(), places, mode)
		}
	})
}

// FuzzInvariants checks what has to hold for any single value, whatever it is: the text and binary forms round-trip,
// the sign predicates agree with Sign, Canonical preserves the value, and one step up then down returns.
func FuzzInvariants(f *testing.F) {
	for _, hi := range []uint64{0, 1, math.MaxUint64} {
		for _, lo := range []uint64{0, 1, math.MaxUint64} {
			for _, s := range []uint8{0, 1, 6, MaxScale} {
				f.Add(hi, lo, s, true)
				f.Add(hi, lo, s, false)
			}
		}
	}
	f.Fuzz(func(t *testing.T, hi, lo uint64, s uint8, neg bool) {
		d := fuzzDec(hi, lo, s, neg)

		if back := FromString(d.StringFixed()); back != d {
			t.Fatalf("%v does not survive its own fixed text %q (got %v)", d, d.StringFixed(), back)
		}
		if back := FromString(d.StringSci()); back != d.Canonical() {
			t.Fatalf("%v does not survive its scientific form %q", d, d.StringSci())
		}
		b, _ := d.MarshalBinary()
		var bin Dec128
		if err := bin.UnmarshalBinary(b); err != nil || bin != d {
			t.Fatalf("%v does not survive its binary form: %v, %v", d, bin, err)
		}
		if n := d.BinarySize(); n != len(b) {
			t.Fatalf("%v: BinarySize %d, wrote %d", d, n, len(b))
		}
		if c := d.Canonical(); !c.Equal(d) {
			t.Fatalf("Canonical changed the value of %v to %v", d, c)
		}

		switch sign := d.Sign(); {
		case sign < 0 && !d.IsNegative(),
			sign > 0 && !d.IsPositive(),
			sign == 0 && !d.IsZero():
			t.Fatalf("%v: Sign is %d but the predicates disagree", d, sign)
		}
		if d.IsNegative() && d.IsPositive() {
			t.Fatalf("%v is both negative and positive", d)
		}
		if up := d.NextUp(); !up.IsNaN() {
			if up.Compare(d) <= 0 {
				t.Fatalf("NextUp(%s) = %s is not greater", d.StringFixed(), up.StringFixed())
			}
			if back := up.NextDown(); !back.IsNaN() && !back.Equal(d) {
				t.Fatalf("NextUp then NextDown of %s gave %s", d.StringFixed(), back.StringFixed())
			}
		}
		if !d.Neg().Neg().Equal(d) || d.Abs().Sign() < 0 {
			t.Fatalf("Neg/Abs are inconsistent for %v", d)
		}
	})
}
