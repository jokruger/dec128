package dec128

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Regression tests for the post-v1.1.0 review: reachable panics, negative zero, NaN semantics and codec boundaries.

// TestDivMulEqualTopLimbsNoPanic: the 256-by-128 division used to panic when the high half of the scaled dividend and
// the divisor agreed in their leading 64 significant bits. Both Div and Mul reached it.
func TestDivMulEqualTopLimbsNoPanic(t *testing.T) {
	// 2^127 / (5*10^37 + 1) * 10^-19: numerator high half 5*10^37, divisor 5*10^37 + 1
	a := FromString("170141183460469231731687303715884105728")
	b := FromString("5000000000000000000.0000000000000000001")
	want, neg, inexact := divOracle(a, b, MaxScale, ROUND_TOWARD_ZERO)
	checkQuotient(t, "DivRound", a, b, a.DivRound(b, MaxScale, ROUND_TOWARD_ZERO), MaxScale, ROUND_TOWARD_ZERO, want, neg, inexact, false)
	// Div rounds to the largest scale that fits; it must agree with DivRound at that scale
	if got := a.Div(b); got.IsNaN() || got.StringFixed() != a.DivRound(b, got.scale, ROUND_TOWARD_ZERO).StringFixed() {
		t.Errorf("Div = %v", got)
	} else {
		want, neg, inexact := divOracle(a, b, got.scale, ROUND_TOWARD_ZERO)
		checkQuotient(t, "Div", a, b, got, got.scale, ROUND_TOWARD_ZERO, want, neg, inexact, false)
	}
	q, r := a.QuoRem(b)
	if q.IsNaN() || r.IsNaN() {
		t.Errorf("QuoRem = %v %v", q, r)
	}

	// 2^127 * 10^-15 times (2*10^30 - 1) * 10^-15: the product's high half is 10^30 - 1 and fitWide divides by 10^30
	x := FromString("170141183460469231731687.303715884105728")
	y := FromString("1999999999999999.999999999999999")
	exact := new(big.Int).Mul(bigAtScale(x, x.scale), bigAtScale(y, y.scale))
	checkFit(t, "Mul", x, y, x.Mul(y), exact, x.scale+y.scale, ArithmeticRounding())
	checkFit(t, "Mul", y, x, y.Mul(x), exact, x.scale+y.scale, ArithmeticRounding())

	// constructed near-equal operands, checked against math/big
	rnd := rand.New(rand.NewSource(20260913))
	for i := 0; i < 20000; i++ {
		c := uint128.Uint128{Lo: rnd.Uint64(), Hi: rnd.Uint64() >> uint(rnd.Intn(64))}
		s1 := uint8(rnd.Intn(20))
		d := Dec128{coef: c, scale: s1}
		f := 19 + rnd.Intn(20) - int(s1)
		if f < 0 || f > 38 {
			continue
		}
		_, carry := c.MulCarry(Pow10Uint128[f])
		if carry.IsZero() {
			continue
		}
		v := carry
		v.Lo += uint64(rnd.Intn(1000)) + 1
		if v.Lo < carry.Lo {
			continue
		}
		o := Dec128{coef: v, scale: uint8(f - 19 + int(s1))}
		want, neg, inexact := divOracle(d, o, MaxScale, ROUND_TOWARD_ZERO)
		checkQuotient(t, "DivRound", d, o, d.DivRound(o, MaxScale, ROUND_TOWARD_ZERO), MaxScale, ROUND_TOWARD_ZERO, want, neg, inexact, false)
		if got := d.Div(o); got.IsNaN() {
			t.Errorf("Div(%s, %s) = NaN(%v)", d.StringFixed(), o.StringFixed(), got.ErrorDetails())
		}
	}
}

func TestPowInt64MinInt64(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
		st   state.State
	}{
		{"2", "", state.Overflow},
		{"0.5", "", state.Overflow},
		{"1", "1", state.Default},
		{"-1", "1", state.Default},
		{"0", "", state.DivisionByZero},
	} {
		got := FromString(tc.in).PowInt64(math.MinInt64)
		if got.state != tc.st || (tc.st < state.Error && got.String() != tc.want) {
			t.Errorf("PowInt64(%s, MinInt64) = %v (state %s), want %q/%s", tc.in, got, got.state, tc.want, tc.st)
		}
	}
	if got := NaN(state.Overflow).PowInt64(math.MinInt64); got.state != state.Overflow {
		t.Errorf("NaN propagation: %v", got)
	}
}

func TestScaleValidationOnConstructors(t *testing.T) {
	for name, d := range map[string]Dec128{
		"MaxAtScale":        MaxAtScale(MaxScale + 1),
		"MinAtScale":        MinAtScale(200),
		"QuantumAtScale":    QuantumAtScale(25),
		"DecodeFromUint128": DecodeFromUint128(uint128.One, 40),
		"DecodeFromUint64":  DecodeFromUint64(1, 25),
		"DecodeFromInt64":   DecodeFromInt64(-1, 20),
	} {
		if d.state != state.ScaleOutOfRange {
			t.Errorf("%s: state %s, want ScaleOutOfRange", name, d.state)
		}
		// the value must be safe to use
		_ = d.Add(One).String()
		_ = d.StringFixed()
	}
	if MaxAtScale(MaxScale).StringFixed() != "34028236692093846346.3374607431768211455" {
		t.Errorf("MaxAtScale(19) = %s", MaxAtScale(MaxScale).StringFixed())
	}
	if QuantumAtScale(MaxScale).StringFixed() != "0.0000000000000000001" {
		t.Errorf("QuantumAtScale(19) = %s", QuantumAtScale(MaxScale).StringFixed())
	}
}

func TestDecodeBinaryRejectsInvalid(t *testing.T) {
	bad := [][]byte{
		{0x1F},      // state code 31 is not defined
		{0x20, 200}, // scale 200
		{0x20, 20},  // scale just above MaxScale
		{0x13},      // state 19: first undefined code
	}
	for _, b := range bad {
		var d Dec128
		if _, err := d.DecodeBinary(b); err != state.InvalidFormat.Error() {
			t.Errorf("DecodeBinary(%x): err %v, want invalid format", b, err)
		}
		if d != Zero {
			t.Errorf("DecodeBinary(%x) modified the target: %v", b, d)
		}
		if err := d.UnmarshalBinary(b); err != state.InvalidFormat.Error() {
			t.Errorf("UnmarshalBinary(%x): err %v", b, err)
		}
		if err := d.ReadBinary(bytes.NewReader(b)); err != state.InvalidFormat.Error() {
			t.Errorf("ReadBinary(%x): err %v", b, err)
		}
	}
	// a negative zero on the wire decodes to zero
	var d Dec128
	if _, err := d.DecodeBinary([]byte{0x01}); err != nil || d.state != state.Default || !d.IsZero() {
		t.Errorf("DecodeBinary(negative zero) = %v %v", d, err)
	}
	if err := d.ReadBinary(bytes.NewReader([]byte{0x21, 2})); err != nil || d.state != state.Default || d.scale != 2 {
		t.Errorf("ReadBinary(negative zero at scale 2) = %v %v", d, err)
	}
	// every defined state and scale still round-trips
	for st := state.State(0); st.IsValid(); st++ {
		for _, sc := range []uint8{0, 1, MaxScale} {
			src := Dec128{coef: uint128.Uint128{Lo: 5, Hi: 7}, scale: sc, state: st}
			b, _ := src.MarshalBinary()
			var back Dec128
			if err := back.UnmarshalBinary(b); err != nil {
				t.Fatalf("state %d scale %d: %v", st, sc, err)
			}
			if back.state != st || (st < state.Error && (back.scale != sc || back.coef != src.coef)) {
				t.Errorf("state %d scale %d: got %+v", st, sc, back)
			}
		}
	}
}

func TestBinarySizeMatchesEncodeBinary(t *testing.T) {
	values := []Dec128{
		Zero, FromString("0.00"), FromString("0.0000000000000000000"), One, FromString("-1.50"),
		MaxAtScale(0), MaxAtScale(MaxScale), NaN(state.Overflow), Null(), FromString("18446744073709551616"),
		{coef: uint128.Uint128{Hi: 1}, scale: 3}, // Lo == 0, Hi != 0
		{coef: uint128.Uint128{Lo: 1, Hi: 1}, state: state.Error},
	}
	for _, v := range values {
		var buf [MaxBytes]byte
		n, err := v.EncodeBinary(buf[:])
		if err != nil {
			t.Fatal(err)
		}
		if n != v.BinarySize() {
			t.Errorf("%v: BinarySize %d, wrote %d", v, v.BinarySize(), n)
		}
	}
}

func TestDecodePgNumericPaddingGuard(t *testing.T) {
	// ndigits=1 weight=-11 dscale=0 digit=1: 44 padding digits below the display scale, which no stripped numeric has
	buf := []byte{0, 1, 0xFF, 0xF5, 0, 0, 0, 0, 0, 1}
	var d Dec128
	if err := d.DecodePgNumeric(buf); err != state.InvalidFormat.Error() {
		t.Errorf("err = %v, want invalid format", err)
	}
	// weight -32768
	buf = []byte{0, 1, 0x80, 0x00, 0, 0, 0, 0, 0, 1}
	if err := d.DecodePgNumeric(buf); err != state.InvalidFormat.Error() {
		t.Errorf("err = %v, want invalid format", err)
	}
	// three padding digits within the last group are still fine: 0.1 sent as digit 1000 at weight -1, dscale 1
	buf = []byte{0, 1, 0xFF, 0xFF, 0, 0, 0, 1, 0x03, 0xE8}
	if err := d.DecodePgNumeric(buf); err != nil || d.StringFixed() != "0.1" {
		t.Errorf("0.1: %v %v", d, err)
	}
}

func TestNoNegativeZero(t *testing.T) {
	check := func(name string, d Dec128) {
		t.Helper()
		if d.IsNaN() {
			t.Errorf("%s: unexpected NaN %v", name, d.ErrorDetails())
			return
		}
		if !d.IsZero() {
			t.Errorf("%s: %s is not zero", name, d.StringFixed())
			return
		}
		if d.state != state.Default {
			t.Errorf("%s: zero with state %s", name, d.state)
		}
		if !d.Equal(Zero) || d.Compare(Zero) != 0 {
			t.Errorf("%s: zero does not equal Zero", name)
		}
		j, _ := d.MarshalJSON()
		if bytes.HasPrefix(j, []byte(`"-`)) {
			t.Errorf("%s: JSON %s", name, j)
		}
	}
	m := FromString("-0.001")
	check("Trunc", m.Trunc(2))
	check("RoundDown", FromString("-0.000").RoundDown(2))
	check("RoundUp", m.RoundUp(2))
	check("RoundTowardZero", m.RoundTowardZero(2))
	check("RoundAwayFromZero", FromString("-0.000").RoundAwayFromZero(2))
	check("RoundHalfTowardZero", FromString("-0.005").RoundHalfTowardZero(2))
	check("RoundHalfAwayFromZero", FromString("-0.004").RoundHalfAwayFromZero(2))
	check("RoundBank", FromString("-0.005").RoundBank(2))
	check("Round ROUND_NAN", FromString("-0.000").Round(2, ROUND_NAN))
	check("Rescale", m.Rescale(2))
	check("RescaleRound", m.RescaleRound(2, ROUND_BANK))
	q, r := FromString("-1").QuoRem(FromString("2"))
	check("QuoRem quotient", q)
	if r.StringFixed() != "-1" {
		t.Errorf("QuoRem remainder = %s", r.StringFixed())
	}
	_, r = FromString("-4.00").QuoRem(FromString("2"))
	check("QuoRem remainder", r)
	check("Mod", FromString("-4.00").Mod(FromString("2")))
	check("NextUp", FromString("-0.01").NextUp())
	check("NextDown", FromString("0.01").NextDown())
	// and the sign survives when the result is not zero
	if got := m.RoundDown(2); got.StringFixed() != "-0.01" {
		t.Errorf("RoundDown(-0.001, 2) = %s", got.StringFixed())
	}
	if got := FromString("-1.5").Rescale(0); got.StringFixed() != "-1" {
		t.Errorf("Rescale(-1.5, 0) = %s", got.StringFixed())
	}
}

func TestEqualCompareNaNAgree(t *testing.T) {
	a, b := NaN(state.Overflow), NaN(state.DivisionByZero)
	if !a.Equal(b) || a.Compare(b) != 0 || !a.Equal(a) {
		t.Error("two NaN values must be equal")
	}
	if a.Equal(One) || One.Equal(a) || a.Compare(One) != -1 || One.Compare(a) != 1 {
		t.Error("NaN must not equal a value and sorts first")
	}
	if !a.LessThanOrEqual(b) || !a.GreaterThanOrEqual(b) || a.LessThan(b) || a.GreaterThan(b) {
		t.Error("ordering of two NaN values")
	}
	if !One.Equal(FromString("1.00")) || FromString("-1").Equal(One) {
		t.Error("numeric equality")
	}
}

func TestMaxMinPropagateNaN(t *testing.T) {
	n := NaN(state.Overflow)
	for name, got := range map[string]Dec128{
		"Max(NaN,1)": Max(n, One), "Max(1,NaN)": Max(One, n), "Max(1,2,NaN,3)": Max(One, Decimal2, n, Decimal3),
		"Min(NaN,1)": Min(n, One), "Min(1,NaN)": Min(One, n), "Min(1,2,NaN,3)": Min(One, Decimal2, n, Decimal3),
	} {
		if got.state != state.Overflow {
			t.Errorf("%s = %v", name, got)
		}
	}
	if Max(One, Decimal3, Decimal2).String() != "3" || Min(Decimal3, One, Decimal2).String() != "1" {
		t.Error("Max/Min on valid values")
	}
	if Max(n, Null()).state != state.Overflow || Min(Null(), n).state != state.Null {
		t.Error("the first NaN argument wins")
	}
}

func TestNextUpDownEdges(t *testing.T) {
	if got := MaxAtScale(2).NextUp(); got.state != state.Overflow {
		t.Errorf("MaxAtScale(2).NextUp() = %v", got)
	}
	if got := MinAtScale(2).NextDown(); got.state != state.Overflow {
		t.Errorf("MinAtScale(2).NextDown() = %v", got)
	}
	cases := []struct{ in, up, down string }{
		{"0.05", "0.06", "0.04"},
		{"0", "1", "-1"},
		{"0.00", "0.01", "-0.01"},
		{"-0.01", "0.00", "-0.02"},
		{"0.01", "0.02", "0.00"},
		{"-1", "0", "-2"},
		{"18446744073709551615", "18446744073709551616", "18446744073709551614"}, // limb carry
		{"-18446744073709551616", "-18446744073709551615", "-18446744073709551617"},
	}
	for _, c := range cases {
		d := FromString(c.in)
		if got := d.NextUp(); got.StringFixed() != c.up || got.scale != d.scale {
			t.Errorf("%s.NextUp() = %s", c.in, got.StringFixed())
		}
		if got := d.NextDown(); got.StringFixed() != c.down || got.scale != d.scale {
			t.Errorf("%s.NextDown() = %s", c.in, got.StringFixed())
		}
	}
	if got := NaN(state.Inexact).NextUp(); got.state != state.Inexact {
		t.Error("NextUp NaN propagation")
	}
	if got := Null().NextDown(); got.state != state.Null {
		t.Error("NextDown NaN propagation")
	}
}

func TestQuoRemDivisorOverflowAndScales(t *testing.T) {
	huge := FromString("170141183460469231731687303715884105728") // 2^127
	tiny := FromString("0.0000000000000000001")
	q, r := tiny.QuoRem(huge)
	if q.StringFixed() != "0" || r.StringFixed() != tiny.StringFixed() || r.scale != MaxScale {
		t.Errorf("tiny QuoRem big = %s, %s", q.StringFixed(), r.StringFixed())
	}
	if got := tiny.Mod(huge); got.StringFixed() != tiny.StringFixed() {
		t.Errorf("tiny Mod big = %s", got.StringFixed())
	}
	if got := tiny.Neg().Mod(huge); got.StringFixed() != "-"+tiny.StringFixed() {
		t.Errorf("-tiny Mod big = %s", got.StringFixed())
	}
	// zero dividend takes the larger scale
	q, r = FromString("0.00").QuoRem(FromString("3.0000"))
	if q.StringFixed() != "0" || r.StringFixed() != "0.0000" {
		t.Errorf("0.00 QuoRem 3.0000 = %s, %s", q.StringFixed(), r.StringFixed())
	}
	if got := FromString("0.00").Mod(Decimal3); got.StringFixed() != "0.00" {
		t.Errorf("0.00 Mod 3 = %s", got.StringFixed())
	}
	// a quotient that really does not fit
	q, r = MaxAtScale(0).QuoRem(tiny)
	if q.state != state.Overflow || r.state != state.Overflow {
		t.Errorf("overflowing QuoRem = %v, %v", q, r)
	}
	if got := MaxAtScale(0).Mod(tiny); got.state != state.Overflow {
		t.Errorf("overflowing Mod = %v", got)
	}
	// random values against math/big: truncated quotient, remainder with the dividend's sign and the larger scale
	rnd := rand.New(rand.NewSource(20260914))
	for i := 0; i < 20000; i++ {
		a, b := randDec(rnd), randDec(rnd)
		if b.IsZero() {
			continue
		}
		q, r := a.QuoRem(b)
		sc := max(a.scale, b.scale)
		A, B := bigAtScale(a, sc), bigAtScale(b, sc)
		wantQ, wantR := new(big.Int).QuoRem(A, B, new(big.Int))
		if new(big.Int).Abs(wantQ).Cmp(big2p128) >= 0 {
			if q.state != state.Overflow {
				t.Errorf("QuoRem(%s, %s): want overflow, got %v", a.StringFixed(), b.StringFixed(), q)
			}
			continue
		}
		if q.IsNaN() || r.IsNaN() || q.scale != 0 || r.scale != sc {
			t.Fatalf("QuoRem(%s, %s) = %v, %v", a.StringFixed(), b.StringFixed(), q, r)
		}
		if bigAtScale(q, 0).Cmp(wantQ) != 0 || bigAtScale(r, sc).Cmp(wantR) != 0 {
			t.Fatalf("QuoRem(%s, %s) = %s, %s; want %v, %v", a.StringFixed(), b.StringFixed(), q.StringFixed(), r.StringFixed(), wantQ, wantR)
		}
		if (q.state == state.Neg) != (wantQ.Sign() < 0) || (r.state == state.Neg) != (wantR.Sign() < 0) {
			t.Fatalf("QuoRem(%s, %s): signs", a.StringFixed(), b.StringFixed())
		}
	}
}

func TestTextNullPolicy(t *testing.T) {
	defer SetNullValue(NullValue())

	b, err := Null().MarshalText()
	if err != nil || len(b) != 0 || b == nil {
		t.Errorf("MarshalText(Null) = %q %v", b, err)
	}
	b, _ = Null().AppendText([]byte("x"))
	if string(b) != "x" {
		t.Errorf("AppendText(Null) = %q", b)
	}
	var d Dec128
	if err := d.UnmarshalText(nil); err != nil || !d.Equal(Zero) || d.IsNull() {
		t.Errorf("default policy: %v %v", d, err)
	}
	SetNullValue(Null())
	if err := d.UnmarshalText([]byte{}); err != nil || !d.IsNull() {
		t.Errorf("NULL policy: %v %v", d, err)
	}
	if err := d.UnmarshalText([]byte("NaN")); err != nil || d.state != state.NaN {
		t.Errorf("NaN text: %v %v", d, err)
	}
	if err := d.UnmarshalText([]byte("x")); err != state.InvalidFormat.Error() {
		t.Errorf("invalid text: %v", err)
	}
}

func TestFromStringTrailingPointBothPaths(t *testing.T) {
	for in, want := range map[string]string{
		"5.": "5", "-5.": "-5", "12345678901234567890.": "12345678901234567890",
		"-123456789012345678901234567890.":         "-123456789012345678901234567890",
		"340282366920938463463374607431768211455.": "340282366920938463463374607431768211455",
	} {
		if got := FromString(in); got.IsNaN() || got.StringFixed() != want || got.scale != 0 {
			t.Errorf("FromString(%q) = %v", in, got)
		}
	}
	for _, in := range []string{".", "-.", "+.", "12345678901234567890..", "12345678901234567890.x", "340282366920938463463374607431768211456."} {
		if got := FromString(in); !got.IsNaN() {
			t.Errorf("FromString(%q) = %v, want NaN", in, got)
		}
	}
}

func TestRoundPropagatesOperandNaN(t *testing.T) {
	if got := NaN(state.DivisionByZero).Round(2, RoundingMode(200)); got.state != state.DivisionByZero {
		t.Errorf("Round(NaN, bad mode) = %v", got)
	}
	if got := One.Round(2, RoundingMode(200)); got.state != state.InvalidRoundingMode {
		t.Errorf("Round(1, bad mode) = %v", got)
	}
}

func TestIsqrt256AgainstBig(t *testing.T) {
	check := func(lo, hi uint128.Uint128) {
		t.Helper()
		if lo.IsZero() && hi.IsZero() {
			return
		}
		n := hi.BigInt()
		n.Lsh(n, 128).Add(n, lo.BigInt())
		want := new(big.Int).Sqrt(n)
		if got := isqrt256(lo, hi).BigInt(); got.Cmp(want) != 0 {
			t.Fatalf("isqrt256(%v) = %v, want %v", n, got, want)
		}
	}
	rnd := rand.New(rand.NewSource(20260915))
	for i := 0; i < 100000; i++ {
		bl := rnd.Intn(256) + 1
		lo := uint128.Uint128{Lo: rnd.Uint64(), Hi: rnd.Uint64()}
		hi := uint128.Uint128{Lo: rnd.Uint64(), Hi: rnd.Uint64()}
		switch {
		case bl <= 64:
			lo.Lo &= ^uint64(0) >> (64 - bl)
			lo.Hi, hi = 0, uint128.Zero
		case bl <= 128:
			lo.Hi &= ^uint64(0) >> (128 - bl)
			hi = uint128.Zero
		case bl <= 192:
			hi.Lo &= ^uint64(0) >> (192 - bl)
			hi.Hi = 0
		default:
			hi.Hi &= ^uint64(0) >> (256 - bl)
		}
		check(lo, hi)
		// perfect squares and their neighbours
		root := uint128.Uint128{Lo: rnd.Uint64(), Hi: rnd.Uint64() >> uint(rnd.Intn(64))}
		if rnd.Intn(2) == 0 {
			root.Hi, root.Lo = 0, root.Lo>>uint(rnd.Intn(64))
		}
		sqLo, sqHi := root.MulCarry(root)
		check(sqLo, sqHi)
		if m1, b := sqLo.SubBorrow(uint128.One); b == 0 {
			check(m1, sqHi)
		}
		if p1, c := sqLo.AddCarry(root); c == 0 {
			if p2, c2 := p1.AddCarry(root); c2 == 0 {
				check(p2, sqHi) // (root+1)^2 - 1
			}
		}
	}
	for k := 0; k < 256; k++ {
		var lo, hi uint128.Uint128
		if k < 128 {
			lo = uint128.One.Lsh(uint(k))
		} else {
			hi = uint128.One.Lsh(uint(k - 128))
		}
		check(lo, hi)
		if m1, b := lo.SubBorrow(uint128.One); b == 0 {
			check(m1, hi)
		} else {
			check(m1, uint128.Uint128{Lo: hi.Lo - 1, Hi: hi.Hi})
		}
	}
	check(uint128.Max, uint128.Max)
	check(uint128.Max, uint128.Zero)
	check(uint128.Zero, uint128.Max)
}

func TestStripZerosAgainstNaive(t *testing.T) {
	naive := func(q uint128.Uint128, scale, ideal uint8) (uint128.Uint128, uint8) {
		for scale > ideal {
			q2, r, _ := q.QuoRemPow10(1)
			if r != 0 {
				break
			}
			q, scale = q2, scale-1
		}
		return q, scale
	}
	rnd := rand.New(rand.NewSource(20260916))
	for i := 0; i < 100000; i++ {
		q := uint128.Uint128{Lo: rnd.Uint64(), Hi: rnd.Uint64() >> uint(rnd.Intn(64))}
		k := rnd.Intn(20)
		q, _, _ = q.QuoRemPow10(uint8(k))
		q, _ = q.Mul(Pow10Uint128[rnd.Intn(20-k)])
		scale := uint8(rnd.Intn(20))
		ideal := uint8(rnd.Intn(int(scale) + 1))
		a1, a2 := stripZeros(q, scale, ideal)
		b1, b2 := naive(q, scale, ideal)
		if a1 != b1 || a2 != b2 {
			t.Fatalf("stripZeros(%v, %d, %d) = (%v, %d), want (%v, %d)", q, scale, ideal, a1, a2, b1, b2)
		}
	}
	// a power of two with many zero bits but no zero digits returns without stripping
	if q, s := stripZeros(uint128.One.Lsh(100), 19, 0); s != 19 || q != uint128.One.Lsh(100) {
		t.Errorf("2^100: %v %d", q, s)
	}
	if q, s := stripZeros(uint128.Zero, 19, 3); s != 3 || !q.IsZero() {
		t.Errorf("zero: %v %d", q, s)
	}
}

func TestAppendAndFixedBufForms(t *testing.T) {
	defer SetTrimOutput(TrimOutput())
	defer SetTrimOutput(false)
	values := []Dec128{Zero, FromString("0.00"), One, FromString("-1.50"), FromString("1234.5678"), MaxAtScale(7), MinAtScale(MaxScale), NaN(state.Overflow), Null()}
	for _, v := range values {
		var buf [MaxStrLen]byte
		if got := string(v.StringFixedToBuf(buf[:])); got != v.StringFixed() {
			t.Errorf("StringFixedToBuf(%v) = %q, want %q", v, got, v.StringFixed())
		}
		for _, trim := range []bool{false, true} {
			SetTrimOutput(trim)
			want, _ := v.MarshalText()
			got, err := v.AppendText([]byte("p:"))
			if err != nil || string(got) != "p:"+string(want) {
				t.Errorf("AppendText(%v, trim=%v) = %q, want %q", v, trim, got, want)
			}
		}
		SetTrimOutput(false)
		if v.IsNull() {
			if _, err := v.AppendIEEE(nil); err == nil {
				t.Error("AppendIEEE(Null) must fail")
			}
			if b, err := v.AppendInt128([]byte{9}, 0, binary.BigEndian); err == nil || len(b) != 1 {
				t.Error("AppendInt128(Null) must fail and leave buf unchanged")
			}
			continue
		}
		var ieee [IEEEBytes]byte
		n, _ := v.EncodeIEEE(ieee[:])
		if got, err := v.AppendIEEE([]byte{1}); err != nil || !bytes.Equal(got, append([]byte{1}, ieee[:n]...)) {
			t.Errorf("AppendIEEE(%v) = %x %v", v, got, err)
		}
		if v.IsNaN() {
			continue
		}
		var i128 [Int128Bytes]byte
		n, err := v.EncodeInt128(i128[:], v.scale, binary.LittleEndian)
		if err != nil {
			// MaxAtScale needs all 128 bits and has no two's-complement form; the append form must fail the same way
			if _, err2 := v.AppendInt128(nil, v.scale, binary.LittleEndian); err2 != err {
				t.Errorf("AppendInt128(%v): %v, want %v", v, err2, err)
			}
			continue
		}
		if got, err := v.AppendInt128([]byte{1}, v.scale, binary.LittleEndian); err != nil || !bytes.Equal(got, append([]byte{1}, i128[:n]...)) {
			t.Errorf("AppendInt128(%v) = %x %v", v, got, err)
		}
	}
	if _, err := One.AppendInt128(nil, MaxScale+1, binary.LittleEndian); err == nil {
		t.Error("AppendInt128 with a bad scale must fail")
	}
}

func TestIntTruncatesAndRangeChecks(t *testing.T) {
	for in, want := range map[string]int64{"1.99": 1, "-1.99": -1, "0.5": 0, "9223372036854775807.9": math.MaxInt64, "-9223372036854775808": math.MinInt64} {
		if got, err := FromString(in).Int64(); err != nil || got != want {
			t.Errorf("Int64(%s) = %d %v", in, got, err)
		}
		if got, err := FromString(in).Int(); err != nil || int64(got) != want {
			t.Errorf("Int(%s) = %d %v", in, got, err)
		}
	}
	if _, err := FromString("9223372036854775808").Int64(); err != state.Overflow.Error() {
		t.Errorf("Int64 overflow: %v", err)
	}
	if _, err := NaN(state.DivisionByZero).Int(); err != state.DivisionByZero.Error() {
		t.Errorf("Int NaN: %v", err)
	}
}
