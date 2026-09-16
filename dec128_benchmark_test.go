package dec128

import (
	"encoding/json"
	"testing"
)

type testJsonStruct struct {
	A Dec128
	B Dec128
	C Dec128
}

func BenchmarkDec128FromString(b *testing.B) {
	ss := []string{
		"12345",
		"1234567890",
		"123456789012345678901234567890",
		"12345.12",
		"1234567890.12345",
		"123456789012345678901234567890.123456789",
		"-123.456",
		"0",
		"0.1",
		"9876.54321",
	}

	sz := len(ss)

	for i := 0; b.Loop(); i++ {
		//_ = dec128.FromString(ss[i%sz])
		_ = FromSafeString(ss[i%sz])
	}
}

func BenchmarkDec128ToString(b *testing.B) {
	ss := []string{
		"12345",
		"1234567890",
		"123456789012345678901234567890",
		"12345.12",
		"1234567890.12345",
		"123456789012345678901234567890.123456789",
		"-123.456",
		"0",
		"0.1",
		"9876.54321",
	}
	sz := len(ss)

	vs := make([]Dec128, sz)
	for i, s := range ss {
		vs[i] = FromString(s)
	}

	buf := [MaxStrLen]byte{}

	for i := 0; b.Loop(); i++ {
		//_ = vs[i%sz].String()
		_ = vs[i%sz].StringToBuf(buf[:])
	}
}

func BenchmarkDec128JsonUnmarshal(b *testing.B) {
	x := testJsonStruct{
		A: FromString("123.456789"),
		B: FromString("1234567890.1234"),
		C: FromString("123456789012345678901234567890.12"),
	}

	s, err := json.Marshal(x)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		var y testJsonStruct
		err := json.Unmarshal(s, &y)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDec128JsonMarshal(b *testing.B) {
	x := testJsonStruct{
		A: FromString("123.456789"),
		B: FromString("1234567890.1234"),
		C: FromString("123456789012345678901234567890.12"),
	}

	for b.Loop() {
		_, err := json.Marshal(x)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDec128BinMarshal(b *testing.B) {
	x := FromString("123.456789")

	for b.Loop() {
		_, err := x.MarshalBinary()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDec128BinUnmarshal(b *testing.B) {
	x := FromString("123.456789")
	bs, err := x.MarshalBinary()
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		var y Dec128
		err := y.UnmarshalBinary(bs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDec128Add(b *testing.B) {
	x := FromString("1234567890.123456789")
	y := FromString("1234567890.123456789")

	for b.Loop() {
		_ = x.Add(y)
	}
}

// The wide-register operations. Sum keeps a 256-bit register of its own and an Accumulator a 384-bit one, and the
// guarded powers carry 57 decimal places, so these are the benchmarks that move when that machinery changes.

func BenchmarkDec128Sum(b *testing.B) {
	xs := []Dec128{
		FromString("1.50"), FromString("-2.25"), FromString("1000.1234"), FromString("0.000001"),
		FromString("99.99"), FromString("-0.5"), FromString("12345.678"), FromString("7"),
	}

	for b.Loop() {
		_ = Sum(xs[0], xs[1:]...)
	}
}

func BenchmarkDec128Accumulator(b *testing.B) {
	xs := []Dec128{
		FromString("1.50"), FromString("-2.25"), FromString("1000.1234"), FromString("0.000001"),
		FromString("99.99"), FromString("-0.5"), FromString("12345.678"), FromString("7"),
	}
	f := FromString("0.9523809523809523810")

	for b.Loop() {
		var acc Accumulator
		for _, x := range xs {
			acc.Add(x)
		}
		for _, x := range xs[:4] {
			acc.AddMul(x, f)
		}
		_ = acc.Total(2, ROUND_BANK)
	}
}

func BenchmarkDec128PowInt64(b *testing.B) {
	x := FromString("1.05")

	for b.Loop() {
		_ = x.PowInt64(360)
	}
}

func BenchmarkDec128PowIntRound(b *testing.B) {
	x := FromString("1.05")

	for b.Loop() {
		_ = x.PowIntRound(360, 10, ROUND_BANK)
	}
}

// The fused multiply-divide. The general form aligns into the 384-bit register and divides by a full coefficient;
// the int64 form never leaves 256 bits and divides by a single limb, which is what the separate routine buys.

func BenchmarkDec128MulDivRound(b *testing.B) {
	x, y, z := FromString("1119.32"), FromString("25.12"), FromString("204.20")

	for b.Loop() {
		_ = x.MulDivRound(y, z, 2, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

// The alignment exponent is at its largest here, so the numerator needs every one of the 384 bits.
func BenchmarkDec128MulDivRoundWide(b *testing.B) {
	x := FromString("34028236692093846346337460743176821145")
	z := FromString("0.0000000000000000003")

	for b.Loop() {
		_ = x.MulDivRound(x, z, MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

// The three ways to take a percentage of an amount, over the same money-sized operands.
func BenchmarkDec128MulPercent(b *testing.B) {
	x, rate := FromString("1119.32"), FromString("7.5")

	for b.Loop() {
		_ = x.MulPercent(rate)
	}
}

func BenchmarkDec128MulPercentRound(b *testing.B) {
	x, rate := FromString("1119.32"), FromString("7.5")

	for b.Loop() {
		_ = x.MulPercentRound(rate, 2, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

func BenchmarkDec128MulThenDivByHundred(b *testing.B) {
	x, rate := FromString("1119.32"), FromString("7.5")

	for b.Loop() {
		_ = x.MulRound(rate, MaxScale, ROUND_HALF_AWAY_FROM_ZERO).DivRound(Decimal100, 2, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

func BenchmarkDec128MulDivRoundInt64(b *testing.B) {
	x := FromString("1119.32")

	for b.Loop() {
		_ = x.MulDivRoundInt64(31, 365, 2, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

func BenchmarkDec128SumRound(b *testing.B) {
	xs := []Dec128{
		FromString("1.50"), FromString("-2.25"), FromString("1000.1234"), FromString("0.000001"),
		FromString("99.99"), FromString("-0.5"), FromString("12345.678"), FromString("7"),
	}

	for b.Loop() {
		_ = SumRound(2, ROUND_BANK, xs[0], xs[1:]...)
	}
}

// PowRational and NthRootRound are the two operations that allocate. The degree drives the cost, so the benchmarks
// span the range the bound allows: a monthly conversion, a daily one, and the whole-schedule case the bound was
// raised to admit.
func BenchmarkDec128PowRationalMonthly(b *testing.B) {
	x := FromString("1.06")

	for b.Loop() {
		_ = x.PowRational(5, 12, MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

func BenchmarkDec128PowRationalDaily(b *testing.B) {
	x := FromString("1.06")

	for b.Loop() {
		_ = x.PowRational(37, 365, MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

func BenchmarkDec128NthRootRoundWholeSchedule(b *testing.B) {
	x := FromString("1.06")

	for b.Loop() {
		_ = x.NthRootRound(10950, MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

// The transcendentals. These carry transGuard places through a series, so they are the slowest operations in the
// package by a wide margin and the benchmarks exist to keep that margin from growing.
func BenchmarkDec128Ln(b *testing.B) {
	x := FromString("1.126825")

	for b.Loop() {
		_ = x.Ln(MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

func BenchmarkDec128Exp(b *testing.B) {
	x := FromString("1.126825")

	for b.Loop() {
		_ = x.Exp(MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

func BenchmarkDec128PowGeneral(b *testing.B) {
	x, e := FromString("1.06"), FromString("0.4166666666666666667")

	for b.Loop() {
		_ = x.Pow(e, MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

// Pow with an integer exponent takes the fixed-width arm, and should cost what PowIntRound costs.
func BenchmarkDec128PowInteger(b *testing.B) {
	x, e := FromString("1.05"), FromInt64(360)

	for b.Loop() {
		_ = x.Pow(e, 10, ROUND_BANK)
	}
}

// The operations added in the read-across. Log10 and the product family carry the same guard digits and the same
// math/big intermediates the transcendentals do; the rest are fixed-width and belong here to keep them that way.

func BenchmarkDec128Log10(b *testing.B) {
	x := FromString("1.126825")

	for b.Loop() {
		_ = x.Log10(MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

func BenchmarkDec128Log10ExactPowerOfTen(b *testing.B) {
	x := FromString("1000")

	for b.Loop() {
		_ = x.Log10(MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	}
}

// Chain-linking a year of daily factors: the case the product family exists for.
func BenchmarkDec128ProdChainLink(b *testing.B) {
	xs := make([]Dec128, 250)
	for i := range xs {
		xs[i] = FromString("1.0004")
	}

	for b.Loop() {
		_ = ProdSliceRound(xs, MaxScale, ROUND_BANK)
	}
}

func BenchmarkDec128ProdSmall(b *testing.B) {
	x, y, z := FromString("1.05"), FromString("0.98"), FromString("1.0004")

	for b.Loop() {
		_ = ProdRound(MaxScale, ROUND_BANK, x, y, z)
	}
}

func BenchmarkDec128AddQuoRound(b *testing.B) {
	x, y, z := FromString("1234.5678"), FromString("1"), FromString("365")

	for b.Loop() {
		_ = x.AddQuoRound(y, z, 8, ROUND_BANK)
	}
}

func BenchmarkDec128RemainderNear(b *testing.B) {
	x, y := FromString("1234.5678"), FromString("0.05")

	for b.Loop() {
		_ = x.RemainderNear(y)
	}
}

func BenchmarkDec128Rat(b *testing.B) {
	x := FromString("1234.5678")

	for b.Loop() {
		_, _ = x.Rat()
	}
}

func BenchmarkDec128DecomposeCompose(b *testing.B) {
	x := FromString("1234.5678")
	var buf [16]byte
	var back Dec128

	for b.Loop() {
		f, neg, c, e := x.Decompose(buf[:0])
		_ = back.Compose(f, neg, c, e)
	}
}
