package dec128

import (
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"math/rand"
	"testing"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

// Reference vectors computed independently (Python: struct for PostgreSQL, integer
// arithmetic for BID), not derived from this package.
var pgVectors = []struct{ in, hex string }{
	{"0", "0000000000000000"},
	{"0.00", "0000000000000002"},
	{"1", "00010000000000000001"},
	{"1.50", "000200000000000200011388"},
	{"-1.50", "000200004000000200011388"},
	{"0.05", "0001ffff0000000201f4"},
	{"100.00", "00010000000000020064"},
	{"10000", "00010001000000000001"},
	{"12345678.90123456789", "000500010000000b04d2162e23340d801ed2"},
	{"-0.0000000000000000001", "0001fffb40000013000a"},
	{"0.00005", "0001fffe000000051388"},
	{"340282366920938463463374607431768211455", "000a00090000000001540b071a2403aa121a18c111ff10dd1aa505af"},
	{"-125275677262976559589.939974206718074617", "000b000540000012000109df162d0a451de7257524b71cfc1a3e02ea06a4"},
	{"9999.99990000", "0002000000000008270f270f"},
}

func TestPgNumericVectors(t *testing.T) {
	var buf [MaxPgNumericBytes]byte
	for _, v := range pgVectors {
		d := FromString(v.in)
		n, err := d.EncodePgNumeric(buf[:])
		if err != nil || hex.EncodeToString(buf[:n]) != v.hex {
			t.Errorf("EncodePgNumeric(%s) = %x, %v; want %s", v.in, buf[:n], err, v.hex)
		}
		want, _ := hex.DecodeString(v.hex)
		var back Dec128
		if err := back.DecodePgNumeric(want); err != nil || back != d {
			t.Errorf("DecodePgNumeric(%s) = %v (%v), want %v", v.hex, back, err, d)
		}
	}
	// specials
	nan, _ := hex.DecodeString("00000000c0000000")
	pinf, _ := hex.DecodeString("00000000d0000000")
	ninf, _ := hex.DecodeString("00000000f0000000")
	var d Dec128
	if err := d.DecodePgNumeric(nan); err != nil || d.state != state.NaN {
		t.Errorf("PG NaN: %v, %v", d, err)
	}
	for _, inf := range [][]byte{pinf, ninf} {
		if err := d.DecodePgNumeric(inf); err != nil || d.state != state.Overflow {
			t.Errorf("PG Infinity: %v, %v", d, err)
		}
	}
	n, err := NaN(state.DivisionByZero).EncodePgNumeric(buf[:])
	if err != nil || hex.EncodeToString(buf[:n]) != "00000000c0000000" {
		t.Errorf("Encode NaN = %x, %v", buf[:n], err)
	}
	if _, err := Null().EncodePgNumeric(buf[:]); err == nil {
		t.Error("a NULL is not a numeric and must not encode")
	}
	if _, err := FromString("1.5").EncodePgNumeric(buf[:9]); err == nil {
		t.Error("short buffer must fail")
	}
	if _, err := FromString("1.5").EncodePgNumeric(buf[:4]); err == nil {
		t.Error("short buffer must fail")
	}
	out, err := FromString("1.50").AppendPgNumeric([]byte{0xAA})
	if err != nil || hex.EncodeToString(out) != "aa000200000000000200011388" {
		t.Errorf("AppendPgNumeric = %x, %v", out, err)
	}
	if _, err := Null().AppendPgNumeric(nil); err == nil {
		t.Error("AppendPgNumeric of NULL must fail")
	}
}

func TestPgNumericDecodeEdges(t *testing.T) {
	hexOf := func(s string) []byte { b, _ := hex.DecodeString(s); return b }
	var d Dec128
	// 1.5 sent with dscale 25: the trailing digits are zero, so it fits at scale 19
	if err := d.DecodePgNumeric(hexOf("00020000000000190001" + "1388")); err != nil || d.StringFixed() != "1.5000000000000000000" {
		t.Errorf("dscale 25 with zero tail: %v (%v)", d, err)
	}
	// 1e-25 (group 1000 at weight -7, dscale 25): cannot fit
	if err := d.DecodePgNumeric(hexOf("0001fff90000001903e8")); err != nil || d.state != state.ScaleOutOfRange {
		t.Errorf("dscale 25 with nonzero tail: %v (%v)", d, err)
	}
	// 1e40: one digit at weight 10
	if err := d.DecodePgNumeric(hexOf("0001000a000000000001")); err != nil || d.state != state.Overflow {
		t.Errorf("1e40: %v (%v)", d, err)
	}
	// 40 digits of 9: more than fit
	forty := "000a0009000000" + "00" + "270f270f270f270f270f270f270f270f270f270f"
	if err := d.DecodePgNumeric(hexOf(forty)); err != nil || d.state != state.Overflow {
		t.Errorf("40 nines: %v (%v)", d, err)
	}
	// too many groups for even 192 bits: 16 groups of 9999
	if err := d.DecodePgNumeric(hexOf("0010000f00000000" + "270f270f270f270f270f270f270f270f270f270f270f270f270f270f270f270f")); err != nil || d.state != state.Overflow {
		t.Errorf("16 groups: %v (%v)", d, err)
	}
	// 9999e36: the groups fit, the scaled coefficient does not
	if err := d.DecodePgNumeric(hexOf("0001000900000000270f")); err != nil || d.state != state.Overflow {
		t.Errorf("9999e36: %v (%v)", d, err)
	}
	// 11 groups of 9999 (44 digits): with dscale 0 the value is integral and too wide;
	// with weight 9 and dscale 3 the last digit is padding and the quotient is too wide
	if err := d.DecodePgNumeric(hexOf("000b000a00000000" + "270f270f270f270f270f270f270f270f270f270f270f")); err != nil || d.state != state.Overflow {
		t.Errorf("44 integral digits: %v (%v)", d, err)
	}
	if err := d.DecodePgNumeric(hexOf("000b000900000003" + "270f270f270f270f270f270f270f270f270f270f270f")); err != nil || d.state != state.Overflow {
		t.Errorf("44 digits with padding: %v (%v)", d, err)
	}
	// zero with dscale 30 keeps the largest representable scale
	if err := d.DecodePgNumeric(hexOf("000000000000001e")); err != nil || d.StringFixed() != "0.0000000000000000000" {
		t.Errorf("zero at dscale 30: %v (%v)", d, err)
	}
	// malformed input is an error, not a value
	// a digit below dscale that is not padding (5 at 10^-28 with dscale 25) is malformed too
	for _, bad := range []string{"", "0001", "00010000000000000001ff", "0000000010000000", "000100000000000027ff", "0001fff9000000190005"} {
		if err := d.DecodePgNumeric(hexOf(bad)); err == nil {
			t.Errorf("malformed %q must fail, got %v", bad, d)
		}
	}
}

func TestPgNumericRoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(20260914))
	var buf [MaxPgNumericBytes]byte
	for range 30000 {
		d := randDec(r)
		n, err := d.EncodePgNumeric(buf[:])
		if err != nil {
			t.Fatalf("encode %v: %v", d, err)
		}
		var back Dec128
		if err := back.DecodePgNumeric(buf[:n]); err != nil || back != d {
			t.Fatalf("round trip %v -> %x -> %v (%v)", d, buf[:n], back, err)
		}
	}
}

var bidVectors = []struct{ in, hex string }{
	{"0", "00000000000000000000000000004030"},
	{"0.00", "00000000000000000000000000003c30"},
	{"1", "01000000000000000000000000004030"},
	{"1.50", "96000000000000000000000000003c30"},
	{"-1.50", "96000000000000000000000000003cb0"},
	{"0.05", "05000000000000000000000000003c30"},
	{"-0.0000000000000000001", "01000000000000000000000000001ab0"},
	{"9999999999999999999999999999999999", "ffffffff638e8d37c087adbe09ed4130"},
}

func TestIEEEVectors(t *testing.T) {
	var buf [IEEEBytes]byte
	for _, v := range bidVectors {
		d := FromString(v.in)
		if !d.FitsIEEE() {
			t.Errorf("%s must fit", v.in)
		}
		n, err := d.EncodeIEEE(buf[:])
		if err != nil || hex.EncodeToString(buf[:n]) != v.hex {
			t.Errorf("EncodeIEEE(%s) = %x, %v; want %s", v.in, buf[:n], err, v.hex)
		}
		want, _ := hex.DecodeString(v.hex)
		var back Dec128
		if err := back.DecodeIEEE(want); err != nil || back != d {
			t.Errorf("DecodeIEEE(%s) = %v (%v), want %v", v.hex, back, err, d)
		}
	}
	hexOf := func(s string) []byte { b, _ := hex.DecodeString(s); return b }
	var d Dec128
	cases := []struct {
		hex  string
		st   state.State
		desc string
	}{
		{"0000000000000000000000000000007c", state.NaN, "quiet NaN without payload"},
		{"0500000000000000000000000000007c", state.Overflow, "NaN with our Overflow payload"},
		{"0f00000000000000000000000000007c", state.NaN, "NaN with the Null payload is a plain NaN"},
		{"ff00000000000000000000000000007c", state.NaN, "NaN with an unknown payload"},
		{"0000000000000000000000000000007e", state.NaN, "signalling NaN"},
		{"00000000000000000000000000000078", state.Overflow, "+Inf"},
		{"000000000000000000000000000000f8", state.Overflow, "-Inf"},
		{"01000000000000000000000000009030", state.Overflow, "1e40"},
		{"01000000000000000000000000000e30", state.ScaleOutOfRange, "1e-25"},
		{"dc050000000000000000000000000a30", state.ScaleOutOfRange, "1500e-27"},
	}
	for _, c := range cases {
		if err := d.DecodeIEEE(hexOf(c.hex)); err != nil || d.state != c.st {
			t.Errorf("%s: %v (%v), want state %s", c.desc, d, err, c.st)
		}
	}
	if err := d.DecodeIEEE(hexOf("00000000000000000000000010000060")); err != nil || d != Zero {
		t.Errorf("non-canonical 11 form must decode to zero, got %v (%v)", d, err)
	}
	// a 113-bit coefficient above 10^34 - 1 in the ordinary form is non-canonical too
	var nc [IEEEBytes]byte
	binary.LittleEndian.PutUint64(nc[0:], ^uint64(0))
	binary.LittleEndian.PutUint64(nc[8:], uint64(ieeeBias)<<49|(1<<49-1))
	if err := d.DecodeIEEE(nc[:]); err != nil || d != Zero {
		t.Errorf("non-canonical coefficient must decode to zero, got %v (%v)", d, err)
	}
	// 1e-25 with a zero tail fits: 1000000e-25 = 0.1e-18
	if err := d.DecodeIEEE(hexOf("40420f00000000000000000000000e30")); err != nil || d.StringFixed() != "0.0000000000000000001" {
		t.Errorf("1000000e-25: %v (%v)", d, err)
	}
	// -0 decodes to zero; 2e3 (positive exponent) multiplies out
	if err := d.DecodeIEEE(hexOf("000000000000000000000000000040b0")); err != nil || d != Zero {
		t.Errorf("-0: %v (%v)", d, err)
	}
	if err := d.DecodeIEEE(hexOf("02000000000000000000000000004630")); err != nil || d.String() != "2000" || d.Scale() != 0 {
		t.Errorf("2e3: %v (%v)", d, err)
	}
	if err := d.DecodeIEEE(hexOf("0000")); err == nil {
		t.Error("short input must fail")
	}
	// NaN round trip keeps the reason
	n, _ := NaN(state.DivisionByZero).EncodeIEEE(buf[:])
	if err := d.DecodeIEEE(buf[:n]); err != nil || d.state != state.DivisionByZero {
		t.Errorf("NaN round trip: %v (%v)", d, err)
	}
	if _, err := Null().EncodeIEEE(buf[:]); err == nil || Null().FitsIEEE() {
		t.Error("NULL must not encode")
	}
	if _, err := One.EncodeIEEE(buf[:8]); err == nil {
		t.Error("short buffer must fail")
	}
}

func TestIEEERoundingOnExport(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	var buf [IEEEBytes]byte
	wide := FromString("340282366920938463463374607431768211455") // 39 digits
	if wide.FitsIEEE() {
		t.Error("39 digits must not fit")
	}
	for _, c := range []struct {
		mode RoundingMode
		want string
	}{
		{ROUND_TOWARD_ZERO, "340282366920938463463374607431768200000"},
		{ROUND_HALF_AWAY_FROM_ZERO, "340282366920938463463374607431768200000"},
		{ROUND_UP, "NaN"}, // ...683e5 is above 2^128: decodes as NaN(Overflow)
	} {
		SetArithmeticRounding(c.mode)
		var back Dec128
		if _, err := wide.EncodeIEEE(buf[:]); err != nil {
			t.Fatalf("encode under %s: %v", c.mode, err)
		}
		if err := back.DecodeIEEE(buf[:]); err != nil || back.String() != c.want {
			t.Errorf("export under %s: %s, want %s", c.mode, back.String(), c.want)
		}
	}
	// a carry out of 34 nines: 99999999999999999999999999999999995 rounds to 1e34, which
	// is 10^33 with one more digit of exponent
	SetArithmeticRounding(ROUND_HALF_AWAY_FROM_ZERO)
	nines := FromString("99999999999999999999999999999999995")
	var back Dec128
	_, _ = nines.EncodeIEEE(buf[:])
	if err := back.DecodeIEEE(buf[:]); err != nil || back.String() != "100000000000000000000000000000000000" {
		t.Errorf("carry out of 34 digits: %s (%v)", back.String(), err)
	}
	// negative and with a scale: -1234567890123456789012345678901234567.89 (39 digits, scale 2)
	neg := FromString("-1234567890123456789012345678901234567.89")
	_, _ = neg.EncodeIEEE(buf[:])
	if err := back.DecodeIEEE(buf[:]); err != nil || back.String() != "-1234567890123456789012345678901235000" {
		t.Errorf("negative half away: %s (%v)", back.String(), err)
	}
	SetArithmeticRounding(ROUND_NAN)
	if _, err := wide.EncodeIEEE(buf[:]); err == nil || err.Error() != "inexact result" {
		t.Errorf("ROUND_NAN must refuse an inexact export, got %v", err)
	}
	exactWide := FromString("340282366920938463463374600000000000000")
	if _, err := exactWide.EncodeIEEE(buf[:]); err != nil {
		t.Errorf("ROUND_NAN must accept an exact export, got %v", err)
	}
}

func TestIEEERoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(20260915))
	var buf [IEEEBytes]byte
	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(34), nil)
	for range 30000 {
		d := randDec(r)
		fits := new(big.Int).Abs(bigAtScale(d, d.scale)).Cmp(limit) < 0
		if d.FitsIEEE() != fits {
			t.Fatalf("FitsIEEE(%v) = %v", d, !fits)
		}
		if !fits {
			continue
		}
		if _, err := d.EncodeIEEE(buf[:]); err != nil {
			t.Fatalf("encode %v: %v", d, err)
		}
		var back Dec128
		if err := back.DecodeIEEE(buf[:]); err != nil || back != d {
			t.Fatalf("round trip %v -> %x -> %v (%v)", d, buf, back, err)
		}
	}
}

func TestInt128Codec(t *testing.T) {
	var buf [Int128Bytes]byte
	d := FromString("-1.50")
	n, err := d.EncodeInt128(buf[:], 2, binary.LittleEndian)
	if err != nil || hex.EncodeToString(buf[:n]) != "6affffffffffffffffffffffffffffff" {
		t.Errorf("LE: %x, %v", buf[:n], err)
	}
	n, err = d.EncodeInt128(buf[:], 2, binary.BigEndian)
	if err != nil || hex.EncodeToString(buf[:n]) != "ffffffffffffffffffffffffffffff6a" {
		t.Errorf("BE: %x, %v", buf[:n], err)
	}
	var back Dec128
	if err := back.DecodeInt128(buf[:], 2, binary.BigEndian); err != nil || back != d {
		t.Errorf("BE decode: %v (%v)", back, err)
	}
	// rescaling to the column scale: 1.5 into a scale-4 column is 15000
	n, _ = FromString("1.5").EncodeInt128(buf[:], 4, binary.LittleEndian)
	if err := back.DecodeInt128(buf[:n], 4, binary.LittleEndian); err != nil || back.StringFixed() != "1.5000" {
		t.Errorf("rescaled: %v (%v)", back, err)
	}
	// digits would be lost: refuse
	if _, err := FromString("1.55").EncodeInt128(buf[:], 1, binary.LittleEndian); err == nil || err.Error() != "inexact result" {
		t.Errorf("inexact must fail, got %v", err)
	}
	if FromString("1.55").FitsInt128(1) || !FromString("1.50").FitsInt128(1) {
		t.Error("FitsInt128 disagrees with the exactness rule")
	}
	// 2^127 does not fit positive, but does fit negative
	if _, err := FromString("170141183460469231731687303715884105728").EncodeInt128(buf[:], 0, binary.LittleEndian); err == nil || err.Error() != "overflow" {
		t.Errorf("2^127 must overflow, got %v", err)
	}
	if _, err := FromString("170141183460469231731687303715884105727").EncodeInt128(buf[:], 0, binary.LittleEndian); err != nil {
		t.Errorf("2^127-1 must fit, got %v", err)
	}
	minInt := FromString("-170141183460469231731687303715884105728")
	if !minInt.FitsInt128(0) || FromString("-170141183460469231731687303715884105729").FitsInt128(0) {
		t.Error("FitsInt128 at the negative limit is wrong")
	}
	n, err = minInt.EncodeInt128(buf[:], 0, binary.LittleEndian)
	if err != nil || hex.EncodeToString(buf[:n]) != "00000000000000000000000000000080" {
		t.Errorf("min int128: %x, %v", buf[:n], err)
	}
	if err := back.DecodeInt128(buf[:], 0, binary.LittleEndian); err != nil || back != minInt {
		t.Errorf("min int128 decode: %v (%v)", back, err)
	}
	// NaN, NULL, short buffers
	if _, err := NaN(state.Overflow).EncodeInt128(buf[:], 0, binary.LittleEndian); err == nil || NaN(state.Overflow).FitsInt128(0) {
		t.Error("NaN must not encode")
	}
	if _, err := One.EncodeInt128(buf[:3], 0, binary.LittleEndian); err == nil {
		t.Error("short buffer must fail")
	}
	if err := back.DecodeInt128(buf[:3], 0, binary.LittleEndian); err == nil {
		t.Error("short input must fail")
	}
	if _, err := One.EncodeInt128(buf[:], MaxScale+1, binary.LittleEndian); err == nil || !One.Rescale(MaxScale+1).IsNaN() {
		t.Error("a scale above MaxScale must fail")
	}
	if One.FitsInt128(MaxScale + 1) {
		t.Error("FitsInt128 above MaxScale must be false")
	}
	// decoding at a schema scale above MaxScale: zero tail fits, nonzero does not
	binary.LittleEndian.PutUint64(buf[0:], 1500000000)
	binary.LittleEndian.PutUint64(buf[8:], 0)
	if err := back.DecodeInt128(buf[:], 25, binary.LittleEndian); err != nil || back.StringFixed() != "0.0000000000000001500" {
		t.Errorf("scale 25 with zero tail: %v (%v)", back, err)
	}
	binary.LittleEndian.PutUint64(buf[0:], 1500000001)
	if err := back.DecodeInt128(buf[:], 25, binary.LittleEndian); err != nil || back.state != state.ScaleOutOfRange {
		t.Errorf("scale 25 with nonzero tail: %v (%v)", back, err)
	}
	// negative zero cannot arise: -0 is 0 in two's complement, but a zero from a
	// negative column value keeps state Default
	for i := range buf {
		buf[i] = 0
	}
	if err := back.DecodeInt128(buf[:], 3, binary.BigEndian); err != nil || back.IsNegative() || back.StringFixed() != "0.000" {
		t.Errorf("zero: %v (%v)", back, err)
	}
}

func TestInt128RoundTrip(t *testing.T) {
	r := rand.New(rand.NewSource(20260916))
	var buf [Int128Bytes]byte
	for range 30000 {
		d := randDec(r)
		order := binary.ByteOrder(binary.LittleEndian)
		if r.Intn(2) == 0 {
			order = binary.BigEndian
		}
		if !d.FitsInt128(d.scale) {
			if _, err := d.EncodeInt128(buf[:], d.scale, order); err == nil {
				t.Fatalf("%v must not encode", d)
			}
			continue
		}
		if _, err := d.EncodeInt128(buf[:], d.scale, order); err != nil {
			t.Fatalf("encode %v: %v", d, err)
		}
		var back Dec128
		if err := back.DecodeInt128(buf[:], d.scale, order); err != nil || back != d {
			t.Fatalf("round trip %v -> %x -> %v (%v)", d, buf, back, err)
		}
	}
}

// TestEncodeIEEERoundMatchesGlobal holds the new per-call encoder and the global one to the same bytes: EncodeIEEE is
// EncodeIEEERound at whatever the process globals happen to say, and nothing else about it changed.
func TestEncodeIEEERoundMatchesGlobal(t *testing.T) {
	defer SetArithmeticRounding(ArithmeticRounding())
	defer SetLossPolicy(CurrentLossPolicy())

	// Values that need more than 34 digits are the only ones where the mode is consulted at all.
	wide := []Dec128{
		FromString("12345678901234567890123456789012345678"),
		FromString("3.4028236692093846346337460743176821145"),
		FromString("-99999999999999999999999999999999999999"),
		FromString("1234.5678"),
		Zero,
		NaN(state.Overflow),
	}

	// SetArithmeticRounding also writes the loss policy - ROUND_NAN is the deprecated spelling of LossNaNOnInexact -
	// and the two encoders agree exactly while it is left that way: the global form reads the policy where the
	// per-call form reads ROUND_NAN, and those are the same condition until SetLossPolicy separates them.
	for _, mode := range allModes {
		SetArithmeticRounding(mode)

		for _, d := range wide {
			var a, b [IEEEBytes]byte
			nA, errA := d.EncodeIEEE(a[:])
			nB, errB := d.EncodeIEEERound(b[:], mode)
			if (errA == nil) != (errB == nil) || (errA != nil && errA.Error() != errB.Error()) {
				t.Fatalf("%s under %s: EncodeIEEE err = %v, EncodeIEEERound err = %v", d.StringFixed(), mode, errA, errB)
			}
			if errA == nil && (nA != nB || a != b) {
				t.Fatalf("%s under %s: EncodeIEEE and EncodeIEEERound disagree", d.StringFixed(), mode)
			}

			// the Append forms follow the Encode ones
			gotA, errAA := d.AppendIEEE(nil)
			gotB, errBB := d.AppendIEEERound(nil, mode)
			if (errAA == nil) != (errBB == nil) || string(gotA) != string(gotB) {
				t.Fatalf("%s under %s: AppendIEEE and AppendIEEERound disagree", d.StringFixed(), mode)
			}
		}
	}

	// LossNaNOnInexact is the global form's other input, and ROUND_NAN is how the per-call form spells it.
	SetArithmeticRounding(ROUND_BANK)
	SetLossPolicy(LossNaNOnInexact)
	big := FromString("12345678901234567890123456789012345678")
	var buf [IEEEBytes]byte
	if _, err := big.EncodeIEEE(buf[:]); err == nil {
		t.Error("EncodeIEEE under LossNaNOnInexact: want an error for a coefficient that must lose digits")
	}
	if _, err := big.EncodeIEEERound(buf[:], ROUND_NAN); err == nil {
		t.Error("EncodeIEEERound(ROUND_NAN): want an error for a coefficient that must lose digits")
	}
	if _, err := big.EncodeIEEERound(buf[:], ROUND_BANK); err != nil {
		t.Errorf("EncodeIEEERound(ROUND_BANK) must not read the loss policy: %v", err)
	}

	// An undefined mode is rejected rather than used.
	if _, err := big.EncodeIEEERound(buf[:], RoundingMode(99)); err == nil || err != state.InvalidRoundingMode.Error() {
		t.Errorf("EncodeIEEERound with an undefined mode: got %v, want %v", err, state.InvalidRoundingMode.Error())
	}
	if _, err := big.AppendIEEERound(nil, RoundingMode(99)); err == nil || err != state.InvalidRoundingMode.Error() {
		t.Errorf("AppendIEEERound with an undefined mode: got %v, want %v", err, state.InvalidRoundingMode.Error())
	}
}

// shedScale is the one reduction every decoder performs on the way in, and the four callers used to carry a copy of
// it each. The chunked step has to agree digit for digit with the one-at-a-time loop they had, including across the
// MaxScale boundary where the first chunk is a full 19 digits and the second is the remainder.
func TestShedScale(t *testing.T) {
	one := uint128.One
	pow38Plus1, _ := Pow10Uint128[38].Add64(1)
	tests := []struct {
		name      string
		coef      uint128.Uint128
		scale     int
		wantCoef  uint128.Uint128
		wantScale uint8
		wantOK    bool
	}{
		{"already in range", uint128.FromUint64(15), 2, uint128.FromUint64(15), 2, true},
		{"at the cap", uint128.FromUint64(15), int(MaxScale), uint128.FromUint64(15), MaxScale, true},
		{"scale 0", one, 0, one, 0, true},
		{"one zero to shed", uint128.FromUint64(150), int(MaxScale) + 1, uint128.FromUint64(15), MaxScale, true},
		{"a nonzero digit below the cap", uint128.FromUint64(151), int(MaxScale) + 1, uint128.FromUint64(151), 0, false},
		// the chunk boundary: 19 zeros go in one division, the 20th in a second
		{"exactly one chunk", Pow10Uint128[19], 2 * int(MaxScale), one, MaxScale, true},
		{"one past a chunk", Pow10Uint128[20], 2*int(MaxScale) + 1, one, MaxScale, true},
		{"two full chunks", Pow10Uint128[38], 3 * int(MaxScale), one, MaxScale, true},
		{"a nonzero digit inside the second chunk", pow38Plus1, 3 * int(MaxScale), pow38Plus1, 0, false},
		{"zero coefficient", uint128.Zero, 3 * int(MaxScale), uint128.Zero, MaxScale, true},
	}

	for _, tt := range tests {
		coef, scale, ok := shedScale(tt.coef, tt.scale)
		if ok != tt.wantOK {
			t.Errorf("%s: ok = %v, want %v", tt.name, ok, tt.wantOK)
			continue
		}
		// a refused reduction hands back the operand as it arrived, not the digits it managed to shed first
		if !coef.Equal(tt.wantCoef) || scale != tt.wantScale {
			t.Errorf("%s: got %v at scale %d, want %v at scale %d", tt.name, coef, scale, tt.wantCoef, tt.wantScale)
		}
	}
}

// and it agrees with the loop the callers used to run, over the whole reachable range of scales
func TestShedScaleMatchesSingleSteps(t *testing.T) {
	single := func(coef uint128.Uint128, scale int) (uint128.Uint128, uint8, bool) {
		for scale > int(MaxScale) {
			q, r, _ := coef.QuoRemPow10(1)
			if r != 0 {
				return coef, 0, false
			}
			coef, scale = q, scale-1
		}
		return coef, uint8(scale), true
	}

	r := rand.New(rand.NewSource(20260916))
	for i := 0; i < 20000; i++ {
		c := uint128.Uint128{Lo: r.Uint64(), Hi: r.Uint64() >> uint(r.Intn(64))}
		if r.Intn(2) == 0 {
			// a value with trailing zeros, so the accepting path is exercised too
			k := uint8(r.Intn(39))
			c, _, _ = c.QuoRemPow10(k)
			c, _ = c.Mul(Pow10Uint128[k])
		}
		scale := r.Intn(4 * int(MaxScale))

		gotCoef, gotScale, gotOK := shedScale(c, scale)
		wantCoef, wantScale, wantOK := single(c, scale)
		if gotOK != wantOK || (gotOK && (!gotCoef.Equal(wantCoef) || gotScale != wantScale)) {
			t.Fatalf("shedScale(%v, %d) = %v/%d/%v, single steps = %v/%d/%v",
				c, scale, gotCoef, gotScale, gotOK, wantCoef, wantScale, wantOK)
		}
	}
}
