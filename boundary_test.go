package dec128

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Corner cases the random oracles reach only by luck: a curated corpus of boundary values crossed with itself, the
// zero value of the type used directly, every uint8 a scale parameter can take, and a handful of named properties
// the documentation promises.

// boundaryCorpus returns the values worth crossing with each other: the ends of the coefficient range, the limb
// boundary, the powers of ten and their neighbors, at the scales a caller actually picks, in both signs.
func boundaryCorpus() []Dec128 {
	coefs := []uint128.Uint128{
		{},                   // 0
		{Lo: 1},              // 1
		{Lo: 2},              // 2
		{Lo: 9},              // 9
		{Lo: 10},             // 10
		{Lo: math.MaxUint64}, // the largest one-limb value
		{Hi: 1},              // 2^64, the smallest two-limb value
		{Lo: math.MaxUint64, Hi: math.MaxUint64 >> 1}, // 2^127 - 1
		{Hi: 1 << 63}, // 2^127
		uint128.Max,   // 2^128 - 1
		Pow10Uint128[19],
		Pow10Uint128[38],
	}
	// the neighbors of the powers of ten, where carries cross limbs
	for _, k := range []int{19, 20, 38} {
		p := Pow10Uint128[k]
		lo, _ := p.SubBorrow(uint128.One)
		hi, _ := p.AddCarry(uint128.One)
		coefs = append(coefs, lo, hi)
	}

	var out []Dec128
	for _, c := range coefs {
		for _, s := range []uint8{0, 2, 6, MaxScale} {
			out = append(out, Dec128{coef: c, scale: s})
			if !c.IsZero() {
				out = append(out, Dec128{coef: c, scale: s, state: state.Neg})
			}
		}
	}
	return out
}

// TestBoundaryCorpusCrossProduct crosses every boundary value with every other and checks Add, Sub and Mul against
// the scale-rule oracle, and QuoRem against exact integer division. Random operands land on these coefficients only
// by chance; crossing them exhaustively is what catches a carry that only misbehaves at 2^64 or at 10^38.
func TestBoundaryCorpusCrossProduct(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())

	corpus := boundaryCorpus()
	t.Logf("%d boundary values, %d ordered pairs", len(corpus), len(corpus)*len(corpus))

	for _, mode := range []RoundingMode{ROUND_TOWARD_ZERO, ROUND_BANK, ROUND_HALF_AWAY_FROM_ZERO} {
		SetArithmeticRounding(mode)
		for _, x := range corpus {
			for _, y := range corpus {
				scale := max(x.scale, y.scale)
				bx, by := bigAtScale(x, scale), bigAtScale(y, scale)
				checkFit(t, "Add", x, y, x.Add(y), new(big.Int).Add(bx, by), scale, mode)
				checkFit(t, "Sub", x, y, x.Sub(y), new(big.Int).Sub(bx, by), scale, mode)
				checkFit(t, "Mul", x, y, x.Mul(y),
					new(big.Int).Mul(bigAtScale(x, x.scale), bigAtScale(y, y.scale)), x.scale+y.scale, mode)

				if t.Failed() {
					t.Fatalf("stopping at %s / %s under %v", x.StringFixed(), y.StringFixed(), mode)
				}
			}
		}
	}

	// QuoRem is exact at any scale, so it gets the plain integer oracle.
	for _, x := range corpus {
		for _, y := range corpus {
			q, r := x.QuoRem(y)
			if y.coef.IsZero() {
				if q.ErrorDetails() != state.DivisionByZero.Error() {
					t.Fatalf("QuoRem(%s, 0) = %v, want DivisionByZero", x.StringFixed(), q.ErrorDetails())
				}
				continue
			}
			if q.IsNaN() {
				continue // a quotient of 129 bits or more
			}
			scale := max(x.scale, y.scale)
			wq, wr := new(big.Int).QuoRem(bigAtScale(x, scale), bigAtScale(y, scale), new(big.Int))
			if exactCoef(q).Cmp(wq) != 0 || bigAtScale(r, scale).Cmp(wr) != 0 || r.scale != scale {
				t.Fatalf("QuoRem(%s, %s) = (%s, %s), want (%s, %s at scale %d)",
					x.StringFixed(), y.StringFixed(), q.StringFixed(), r.StringFixed(), wq, wr, scale)
			}
		}
	}
}

// TestUninitializedValue exercises the whole method surface on the zero value of the type. doc.go promises that an
// uninitialized Dec128 is a valid zero, which means no method may treat it as NaN or as negative.
func TestUninitializedValue(t *testing.T) {
	var a, b Dec128

	derived := []struct {
		name string
		d    Dec128
	}{
		{"itself", a},
		{"Abs", a.Abs()},
		{"Neg", a.Neg()},
		{"Add", a.Add(b)},
		{"Sub", a.Sub(b)},
		{"Mul", a.Mul(b)},
		{"MulRound", a.MulRound(b, 4, ROUND_BANK)},
		{"Rescale", a.Rescale(5)},
		{"RescaleRound", a.RescaleRound(5, ROUND_BANK)},
		{"Canonical", a.Canonical()},
		{"Round", a.Round(2, ROUND_BANK)},
		{"Trunc", a.Trunc(2)},
		{"RoundBank", a.RoundBank(2)},
		{"NextUp then NextDown", a.NextUp().NextDown()},
		{"Sqrt", a.Sqrt()},
		{"SqrtRound", a.SqrtRound(4, ROUND_BANK)},
		{"PowInt64", a.PowInt64(3)},
		{"Div by one", a.Div(One)},
		{"DivRound by one", a.DivRound(One, 2, ROUND_BANK)},
		{"Mod by one", a.Mod(One)},
		{"Sum", Sum(a, b)},
		{"Avg", Avg(a, b)},
		{"Min", Min(a, b)},
		{"Max", Max(a, b)},
	}
	for _, c := range derived {
		d := c.d
		switch {
		case d.IsNaN():
			t.Errorf("%s of the zero value is NaN(%v)", c.name, d.ErrorDetails())
		case !d.IsZero():
			t.Errorf("%s of the zero value is not zero: %v", c.name, d)
		case d.IsNegative() || d.state == state.Neg:
			t.Errorf("%s of the zero value is negative", c.name)
		case d.Sign() != 0:
			t.Errorf("%s of the zero value has sign %d", c.name, d.Sign())
		case d.String() != "0":
			t.Errorf("%s of the zero value prints as %q, want \"0\"", c.name, d.String())
		}
	}

	// It equals every other spelling of zero, and division by it is division by zero.
	for _, z := range []Dec128{Zero, Decimal0, FromString("0"), FromString("0.00"), FromString("-0")} {
		if !a.Equal(z) || a.Compare(z) != 0 {
			t.Errorf("the zero value does not equal %v", z)
		}
	}
	if a.Div(b).ErrorDetails() != state.DivisionByZero.Error() {
		t.Error("dividing by the zero value must be DivisionByZero")
	}
	if i, err := a.Int64(); err != nil || i != 0 {
		t.Errorf("Int64 of the zero value = %d, %v", i, err)
	}
	if f, err := a.InexactFloat64(); err != nil || f != 0 {
		t.Errorf("InexactFloat64 of the zero value = %v, %v", f, err)
	}
}

// TestEveryScaleParameterValue pushes all 256 values a uint8 scale parameter can hold through every method that takes
// one. Scales above MaxScale must come back as NaN(ScaleOutOfRange) or leave the value alone, never index a table out
// of range: the internal tables are sized for 0..MaxScale and the parameter is not.
func TestEveryScaleParameterValue(t *testing.T) {
	values := []Dec128{
		Zero, One, NegativeOne, FromString("1.5"), FromString("-0.000001"),
		MaxAtScale(0), MaxAtScale(MaxScale), MinAtScale(MaxScale),
		NaN(state.Overflow), Null(),
	}
	buf := make([]byte, Int128Bytes)

	for s := range 256 {
		p := uint8(s)
		valid := p <= MaxScale

		// constructors: a scale above MaxScale is a NaN, never a value whose later use misbehaves
		for _, got := range []Dec128{
			MaxAtScale(p), MinAtScale(p), QuantumAtScale(p),
			DecodeFromUint64(1, p), DecodeFromInt64(-1, p), DecodeFromUint128(uint128.One, p),
			New(uint128.One, p, true),
		} {
			if got.IsNaN() == valid {
				t.Fatalf("scale %d: constructor returned NaN=%v, want %v", p, got.IsNaN(), !valid)
			}
			if !valid && got.ErrorDetails() != state.ScaleOutOfRange.Error() {
				t.Fatalf("scale %d: constructor returned NaN(%v), want ScaleOutOfRange", p, got.ErrorDetails())
			}
		}

		for _, v := range values {
			// Rescale and the exact-scale operations refuse an out-of-range scale
			for name, got := range map[string]Dec128{
				"Rescale":      v.Rescale(p),
				"RescaleRound": v.RescaleRound(p, ROUND_BANK),
				"MulRound":     v.MulRound(One, p, ROUND_BANK),
				"DivRound":     v.DivRound(Decimal3, p, ROUND_BANK),
				"SqrtRound":    v.SqrtRound(p, ROUND_BANK),
			} {
				if !valid && !got.IsNaN() {
					t.Fatalf("%s(%s, scale %d) = %s, want NaN", name, v.StringFixed(), p, got.StringFixed())
				}
				if got.scale > MaxScale {
					t.Fatalf("%s(%s, scale %d) produced scale %d", name, v.StringFixed(), p, got.scale)
				}
			}
			// the Round* family leaves a value alone when the scale is not lower, so an out-of-range scale is a no-op
			for name, got := range map[string]Dec128{
				"Round":     v.Round(p, ROUND_BANK),
				"Trunc":     v.Trunc(p),
				"RoundBank": v.RoundBank(p),
				"RoundUp":   v.RoundUp(p),
			} {
				if !valid && got != v {
					t.Fatalf("%s(%s, scale %d) = %v, want the value unchanged", name, v.StringFixed(), p, got)
				}
				if got.scale > MaxScale {
					t.Fatalf("%s(%s, scale %d) produced scale %d", name, v.StringFixed(), p, got.scale)
				}
			}
			// the encoders take a scale too, and must report rather than misbehave
			_, _ = v.EncodeToInt64(p)
			_, _ = v.EncodeToUint64(p)
			_, _ = v.EncodeToUint128(p)
			_ = v.FitsInt128(p)
			_, _ = v.EncodeInt128(buf, p, binary.BigEndian)
			var d Dec128
			if err := d.DecodeInt128(buf, p, binary.BigEndian); err != nil {
				t.Fatalf("DecodeInt128 at scale %d: %v", p, err)
			}
			if d.scale > MaxScale {
				t.Fatalf("DecodeInt128 at scale %d produced scale %d", p, d.scale)
			}
			// and everything must still print
			_ = d.String()
			_ = d.StringFixed()
			_ = d.StringSci()
		}
	}
}

// TestCrossScaleRoundingAgrees is the RoundBank-anomaly class: 2.5 and 2.50 are the same number, so every mode must
// round them alike. An implementation that looks only at the first discarded digit gets 2.50 wrong.
func TestCrossScaleRoundingAgrees(t *testing.T) {
	// the named case first
	for _, mode := range allModes {
		a, b := FromString("2.5").Round(0, mode), FromString("2.50").Round(0, mode)
		if !a.Equal(b) {
			t.Errorf("%v: 2.5 rounds to %s but 2.50 rounds to %s", mode, a.StringFixed(), b.StringFixed())
		}
	}
	if got := FromString("2.50").RoundBank(0); got.StringFixed() != "2" {
		t.Errorf("RoundBank(2.50, 0) = %s, want 2", got.StringFixed())
	}

	r := rand.New(rand.NewSource(20260922))
	for range 40000 {
		base := Dec128{coef: uint128.Uint128{Lo: r.Uint64() % 1_000_000_000_000}, scale: uint8(1 + r.Intn(6))}
		if r.Intn(2) == 0 && !base.coef.IsZero() {
			base.state = state.Neg
		}
		wider := base.Rescale(base.scale + uint8(1+r.Intn(7)))
		if wider.IsNaN() || !wider.Equal(base) {
			continue
		}
		for places := uint8(0); places < base.scale; places++ {
			for _, mode := range allModes {
				a, b := base.Round(places, mode), wider.Round(places, mode)
				if a.IsNaN() != b.IsNaN() || (!a.IsNaN() && !a.Equal(b)) {
					t.Fatalf("%s and %s are equal, but Round(%d, %v) gives %s and %s",
						base.StringFixed(), wider.StringFixed(), places, mode, a.StringFixed(), b.StringFixed())
				}
			}
		}
	}
}

// TestNoDoubleRounding pins the reason DivRound and MulRound exist: they decide once, on the exact remainder, where
// rounding an already-rounded quotient can land a step away.
func TestNoDoubleRounding(t *testing.T) {
	defer SetDefaultScale(DefaultScale())
	SetDefaultScale(MaxScale)

	n, d := FromString("5000000000000000001"), FromString("1000000000000000000000")
	// the exact quotient is 0.000000000000000000005000000000000000001, just above the tie at two places
	if got := n.DivRound(d, 2, ROUND_BANK).StringFixed(); got != "0.01" {
		t.Errorf("DivRound decided on a rounded quotient: got %s, want 0.01", got)
	}
	// the two-step form loses the digits that broke the tie and rounds the other way, which is the trap
	if got := n.Div(d).RoundBank(2).StringFixed(); got != "0.00" {
		t.Errorf("Div then RoundBank = %s, want 0.00 (the double-rounded answer)", got)
	}

	// The same for MulRound: the full 256-bit product decides, not the product reduced to MaxScale first.
	a, b := FromString("0.0000000000000000001"), FromString("0.5000000000000000001")
	if got := a.MulRound(b, 19, ROUND_BANK).StringFixed(); got != "0.0000000000000000001" {
		t.Errorf("MulRound = %s", got)
	}

	// And in general: for random operands, DivRound at a scale must equal the big oracle, never the two-step answer.
	r := rand.New(rand.NewSource(20260923))
	differed := 0
	for range 20000 {
		x, y := randDec(r), randDec(r)
		if y.IsZero() || x.IsZero() {
			continue
		}
		scale := uint8(r.Intn(int(MaxScale) + 1))
		got := x.DivRound(y, scale, ROUND_BANK)
		if got.IsNaN() {
			continue
		}
		want, neg, _ := divOracle(x, y, scale, ROUND_BANK)
		if new(big.Int).Abs(exactCoef(got)).Cmp(want) != 0 {
			t.Fatalf("DivRound(%s, %s, %d) = %s, want coefficient %s", x.StringFixed(), y.StringFixed(), scale, got.StringFixed(), want)
		}
		_ = neg
		if two := x.Div(y).Round(scale, ROUND_BANK); !two.IsNaN() && !two.Equal(got) {
			differed++
		}
	}
	if differed == 0 {
		t.Error("no operand pair distinguished one-step from two-step rounding, so this test proves nothing")
	}
	t.Logf("one-step and two-step rounding differed on %d of the random pairs", differed)
}

// TestFloatRoundTrip checks the property FromFloat64 is built on: it takes the shortest decimal that round-trips the
// float, so converting back must return the same float bit for bit.
func TestFloatRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(20260924))
	checked := 0

	check := func(f float64) {
		t.Helper()
		d := FromFloat64(f)
		if d.IsNaN() {
			return // outside the type's range
		}
		checked++
		back, err := d.InexactFloat64()
		if err != nil {
			t.Fatalf("InexactFloat64 of FromFloat64(%v) = %s: %v", f, d.StringFixed(), err)
		}
		if back != f {
			t.Fatalf("FromFloat64(%v) = %s, which converts back to %v", f, d.StringFixed(), back)
		}
	}

	for _, f := range []float64{
		0, 1, -1, 0.5, -0.5, 0.1, 1.0 / 3, 2, 4, 1e-19, 1e18, math.MaxInt64, math.SmallestNonzeroFloat64,
		math.MaxFloat64, 1e-20, 0.3, 12345.6789, -98765.4321,
	} {
		check(f)
	}
	// realistic magnitudes: a mantissa scaled by a decimal exponent in range
	for range 100000 {
		m := float64(r.Uint64()>>11) / (1 << 53) // in [0, 1)
		e := r.Intn(38) - 19
		check(m * math.Pow(10, float64(e)))
		check(-m * math.Pow(10, float64(e)))
	}
	// exact powers of two, where the shortest representation is most delicate
	for e := -60; e <= 60; e++ {
		check(math.Ldexp(1, e))
		check(math.Ldexp(3, e))
	}
	if checked < 1000 {
		t.Errorf("only %d floats were in range; the generator is not exercising much", checked)
	}
	t.Logf("round-tripped %d floats", checked)
}

// TestErrorSentinels pins the classification contract in ErrorDetails' documentation: one sentinel per reason, the
// same value every time, comparable with == and with errors.Is, and distinct between reasons.
func TestErrorSentinels(t *testing.T) {
	reasons := []state.State{
		state.Error, state.NaN, state.DivisionByZero, state.Overflow, state.Underflow,
		state.NegativeInUnsignedOp, state.NotEnoughBytes, state.InvalidFormat, state.SqrtNegative,
		state.ScaleOutOfRange, state.RescaleToLowerScale, state.Null, state.Inexact, state.InvalidRoundingMode,
	}
	seen := map[error]state.State{}
	for _, s := range reasons {
		e := NaN(s).ErrorDetails()
		if e == nil {
			t.Errorf("%v: ErrorDetails is nil for a NaN", s)
			continue
		}
		if e != NaN(s).ErrorDetails() || e != s.Error() {
			t.Errorf("%v: sentinel is not stable across calls", s)
		}
		if !errors.Is(e, s.Error()) {
			t.Errorf("%v: errors.Is against its own sentinel is false", s)
		}
		if prev, dup := seen[e]; dup {
			t.Errorf("%v shares a sentinel with %v", s, prev)
		}
		seen[e] = s
	}

	// the reasons arithmetic actually produces, classified the way a caller would
	if err := One.Div(Zero).ErrorDetails(); !errors.Is(err, state.DivisionByZero.Error()) {
		t.Errorf("1/0 reports %v", err)
	}
	if err := NegativeOne.Sqrt().ErrorDetails(); !errors.Is(err, state.SqrtNegative.Error()) {
		t.Errorf("sqrt(-1) reports %v", err)
	}
	if err := MaxAtScale(0).Add(MaxAtScale(0)).ErrorDetails(); !errors.Is(err, state.Overflow.Error()) {
		t.Errorf("max+max reports %v", err)
	}
	// a valid value has no error
	if One.ErrorDetails() != nil {
		t.Error("a valid value reports an error")
	}
}

// TestOmitZeroAndIsZero pins how encoding/json treats a Dec128 field. Because the type has an IsZero method, Go's
// omitzero tag uses it, so a zero at any scale is dropped from the output while a NaN and a NULL are not. Anyone
// tagging a currency field with omitzero should see that in a test rather than in production.
func TestOmitZeroAndIsZero(t *testing.T) {
	type row struct {
		A Dec128 `json:"a,omitzero"`
		B Dec128 `json:"b"`
	}
	cases := []struct {
		in   Dec128
		want string
	}{
		{Zero, `{"b":"0"}`},                  // omitted: IsZero is true
		{FromString("0.00"), `{"b":"0.00"}`}, // omitted too, even though the scale is meaningful
		{FromString("1.5"), `{"a":"1.5","b":"1.5"}`},
		{Null(), `{"a":null,"b":null}`}, // a NULL is not zero, so it survives
		{NaN(state.Overflow), `{"a":"NaN","b":"NaN"}`},
	}
	for _, c := range cases {
		b, err := json.Marshal(row{A: c.in, B: c.in})
		if err != nil {
			t.Fatalf("marshalling %v: %v", c.in, err)
		}
		if string(b) != c.want {
			t.Errorf("%s with omitzero: got %s, want %s", c.in.StringFixed(), b, c.want)
		}
	}
}

// TestMarshalResultIsPrivate pins the ownership half of the allocation contract in alloc_test.go: the marshallers
// return an owned slice. The shortcut cases used to hand out the package's own ZeroStrBytes, NaNStrBytes,
// NaNJsonStrBytes and nullValue, so a caller writing into the result corrupted the constant for the whole process -
// silently, and for every later value.
func TestMarshalResultIsPrivate(t *testing.T) {
	defer SetTrimOutput(TrimOutput())

	constants := func() []string {
		return []string{
			string(ZeroStrBytes), string(ZeroJsonStrBytes), string(NaNStrBytes),
			string(NaNJsonStrBytes), string(nullValue), ZeroStr, NaNStr,
		}
	}

	for _, trim := range []bool{false, true} {
		SetTrimOutput(trim)
		for _, d := range []Dec128{Zero, FromString("0.00"), FromString("1.5"), Null(), NaN(state.Overflow), NaN(state.NaN)} {
			before := constants()

			for _, m := range []struct {
				name string
				fn   func() ([]byte, error)
			}{
				{"MarshalText", d.MarshalText},
				{"MarshalJSON", d.MarshalJSON},
				{"MarshalBinary", d.MarshalBinary},
				{"AppendText", func() ([]byte, error) { return d.AppendText(nil) }},
				{"AppendBinary", func() ([]byte, error) { return d.AppendBinary(nil) }},
				{"AppendPgNumeric", func() ([]byte, error) { return d.AppendPgNumeric(nil) }},
			} {
				b, err := m.fn()
				if err != nil {
					continue // AppendPgNumeric refuses a NULL, which is its documented behavior
				}
				for i := range b {
					b[i] ^= 0xFF // scribble over every byte the caller was handed
				}
				for j, s := range constants() {
					if s != before[j] {
						t.Fatalf("%s of %v (trim=%v) handed out a package constant: %q became %q",
							m.name, d, trim, before[j], s)
					}
				}
			}

			// and the value itself must still marshal correctly afterwards
			if got, _ := d.MarshalText(); d.state != state.Null && len(got) == 0 {
				t.Fatalf("after scribbling, MarshalText of %v is empty", d)
			}
		}
	}
}
