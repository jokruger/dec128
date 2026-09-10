package dec128

import "fmt"

func ExampleFromString() {
	a := FromString("0.123456789")
	fmt.Println(a.String())
	// Output:
	// 0.123456789
}

func ExampleDec128_Abs() {
	a := FromString("-123.45")
	fmt.Println(a.Abs())
	// Output:
	// 123.45
}

func ExampleDec128_Add() {
	a := FromString("123.45")
	b := FromString("678.90")
	fmt.Println(a.Add(b))
	// Output:
	// 802.35
}

func ExampleDec128_Sub() {
	a := FromString("123.45")
	b := FromString("678.90")
	fmt.Println(a.Sub(b))
	// Output:
	// -555.45
}

func ExampleDec128_Mul() {
	a := FromString("123.45")
	b := FromString("678.90")
	fmt.Println(a.Mul(b))
	// Output:
	// 83810.205
}

func ExampleDec128_Div() {
	// Div falls back to the package default scale, so pin it for the example
	defer SetDefaultScale(DefaultScale())
	SetDefaultScale(6)

	a := FromString("1")
	b := FromString("3")
	fmt.Println(a.Div(b))
	// Output:
	// 0.333333
}

func ExampleDec128_DivRound() {
	a := FromString("5.0")
	b := FromString("365")

	fmt.Println(a.DivRound(b, 19, ROUND_TOWARD_ZERO))
	fmt.Println(a.DivRound(b, 2, ROUND_HALF_AWAY_FROM_ZERO))
	fmt.Println(FromInt64(200).DivRound(FromInt64(3), 2, ROUND_HALF_AWAY_FROM_ZERO))
	// Output:
	// 0.0136986301369863013
	// 0.01
	// 66.67
}

func ExampleDec128_Sqrt() {
	a := FromString("4")
	fmt.Println(a.Sqrt())
	// Output:
	// 2
}

func ExampleSetNullValue() {
	defer SetNullValue(NullValue())
	// by default a SQL NULL or a JSON null decodes to zero
	var a Dec128
	_ = a.Scan(nil)
	fmt.Println(a, a.IsNull())

	// make NULL round-trip instead
	SetNullValue(Null())
	defer SetNullValue(Zero)

	var b Dec128
	_ = b.Scan(nil)
	v, _ := b.Value()
	fmt.Println(b.IsNull(), v)

	// and it propagates the way SQL NULL does
	fmt.Println(b.Add(FromInt64(1)).IsNull())
	// Output:
	// 0 false
	// true <nil>
	// true
}

func ExampleMax() {
	a := FromString("1.1")
	b := FromString("1.2")
	c := FromString("1.3")
	d := FromString("-1")
	fmt.Println(Max(a, b))
	fmt.Println(Max(a, b, c))
	fmt.Println(Max(a, b, c, d))
	// Output:
	// 1.2
	// 1.3
	// 1.3
}

func ExampleMin() {
	a := FromString("1.1")
	b := FromString("1.2")
	c := FromString("1.3")
	d := FromString("-1")
	fmt.Println(Min(a, b))
	fmt.Println(Min(a, b, c))
	fmt.Println(Min(a, b, c, d))
	// Output:
	// 1.1
	// 1.1
	// -1
}

func ExampleDec128_PowInt() {
	a := FromString("2")
	fmt.Println(a.PowInt(-3))
	// Output:
	// 0.125
}

func ExampleDec128_Mod() {
	a := FromString("7")
	b := FromString("3")
	fmt.Println(a.Mod(b))
	// Output:
	// 1
}

func ExampleSum() {
	a := FromString("1")
	b := FromString("2")
	c := FromString("3.1")
	fmt.Println(Sum(a, b))
	fmt.Println(Sum(a, b, c))
	// Output:
	// 3
	// 6.1
}

func ExampleAvg() {
	a := FromString("1")
	b := FromString("2")
	c := FromString("3")
	d := FromString("1.1")
	fmt.Println(Avg(a, b))
	fmt.Println(Avg(a, b, c))
	fmt.Println(Avg(a, b, c, d))
	// Output:
	// 1.5
	// 2
	// 1.775
}

func ExampleFromString_scientific() {
	// the regular and the scientific form are both accepted
	a := FromString("1.5e3")
	b := FromString("-2.5E-2")
	fmt.Println(a, b)
	// Output:
	// 1500 -0.025
}

func ExampleDec128_StringSci() {
	a := FromString("12345")
	b := FromString("0.00015")
	c := FromString("-1200")
	// String stays in the regular form, StringSci is the opt-in
	fmt.Println(a.String(), a.StringSci())
	fmt.Println(b.StringSci(), c.StringSci())
	// Output:
	// 12345 1.2345e+4
	// 1.5e-4 -1.2e+3
}

func ExampleDec128_MulRound() {
	price := FromString("1234.5678")
	qty := FromString("8765.4321")

	fmt.Println(price.MulRound(qty, 2, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	fmt.Println(FromString("1.005").MulRound(One, 2, ROUND_BANK).StringFixed())
	// Output:
	// 10821520.22
	// 1.00
}

func ExampleDec128_RescaleRound() {
	d := FromString("1.5")

	// Round* methods leave a shorter value unchanged; RescaleRound yields exactly n places
	fmt.Println(d.RoundBank(2).StringFixed())
	fmt.Println(d.RescaleRound(2, ROUND_BANK).StringFixed())
	fmt.Println(FromString("2.345").RescaleRound(2, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	// Output:
	// 1.5
	// 1.50
	// 2.35
}

func ExampleDec128_Div_idealScale() {
	defer SetDefaultScale(DefaultScale())
	SetDefaultScale(MaxScale)

	// an exact quotient takes its ideal scale; an inexact one keeps the default scale
	fmt.Println(FromString("1").Div(FromString("2")).StringFixed())
	fmt.Println(FromString("1.00").Div(FromString("2")).StringFixed())
	fmt.Println(FromString("1").Div(FromString("3")).StringFixed())
	// Output:
	// 0.5
	// 0.50
	// 0.3333333333333333333
}

func ExampleSetArithmeticRounding() {
	defer SetArithmeticRounding(ArithmeticRounding())
	a, b := FromString("0.1234567890123456789"), FromString("0.9876543210987654321")

	// the exact product has 38 places; only 19 fit
	SetArithmeticRounding(ROUND_NAN)
	fmt.Println(a.Mul(b).ErrorDetails())

	SetArithmeticRounding(ROUND_BANK)
	fmt.Println(a.Mul(b).StringFixed())
	// Output:
	// inexact result
	// 0.1219326311370217952
}

func ExampleSetTrimOutput() {
	defer SetTrimOutput(TrimOutput())
	d := FromString("1.50")

	v, _ := d.Value()
	fmt.Println(v)

	SetTrimOutput(true)
	v, _ = d.Value()
	fmt.Println(v)
	// Output:
	// 1.50
	// 1.5
}

func ExampleSum_exact() {
	fmt.Println(Sum(FromString("0.10"), FromString("0.20"), FromString("-0.30")).StringFixed())
	fmt.Println(Sum(FromString("1.5"), FromString("2.25")).StringFixed())
	// Output:
	// 0.00
	// 3.75
}

func ExampleDec128_EncodePgNumeric() {
	var buf [MaxPgNumericBytes]byte
	n, _ := FromString("1.50").EncodePgNumeric(buf[:])
	fmt.Printf("%x\n", buf[:n])

	var back Dec128
	_ = back.DecodePgNumeric(buf[:n])
	fmt.Println(back.StringFixed())
	// Output:
	// 000200000000000200011388
	// 1.50
}
