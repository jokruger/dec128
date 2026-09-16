package dec128

import (
	"fmt"
	"strings"
)

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

func ExampleSetLossPolicy() {
	defer SetLossPolicy(CurrentLossPolicy())
	third := FromInt64(1).Div(FromInt64(3))
	tiny := FromString("0.0000000001") // 1e-10; squared it is 1e-20, below the smallest representable value

	// The default rounds every loss, so a product of two small values can reach zero.
	SetLossPolicy(LossRound)
	fmt.Println(third.StringFixed(), tiny.Mul(tiny).StringFixed())

	// LossNaNOnUnderflow refuses only a total loss of significance; 1/3 is still a value.
	SetLossPolicy(LossNaNOnUnderflow)
	fmt.Println(FromInt64(1).Div(FromInt64(3)).StringFixed(), tiny.Mul(tiny).ErrorDetails())

	// LossNaNOnInexact refuses any discarded digit, 1/3 included.
	SetLossPolicy(LossNaNOnInexact)
	fmt.Println(FromInt64(1).Div(FromInt64(3)).ErrorDetails())
	// Output:
	// 0.3333333333333333333 0.0000000000000000000
	// 0.3333333333333333333 underflow
	// inexact result
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

func ExampleAccumulator() {
	// The present value of three cashflows, discounted by factors carried to full precision. Each term is a product
	// that is not representable on its own; the accumulator keeps them all exact and rounds once.
	cashflows := []Dec128{FromString("1000.00"), FromString("1000.00"), FromString("1000.00")}
	factors := []Dec128{
		FromString("0.9523809523809523810"),
		FromString("0.9070294784580498866"),
		FromString("0.8638376937695713206"),
	}

	npv := NewAccumulator(2)
	for i, cf := range cashflows {
		npv.AddMul(cf, factors[i])
	}
	fmt.Println(npv.Count(), npv.Total(2, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	// Output:
	// 3 2723.25
}

func ExampleAccumulator_exact() {
	// Splitting a payment three ways and adding the parts back is exact, because the accumulator never rounds a term.
	third := FromString("0.3333333333333333333")

	acc := NewAccumulator(2)
	acc.Add(third)
	acc.Add(third)
	acc.Add(third)
	fmt.Println(acc.Total(MaxScale, ROUND_BANK).StringFixed())
	fmt.Println(acc.Total(2, ROUND_BANK).StringFixed())
	// Output:
	// 0.9999999999999999999
	// 1.00
}

func ExampleDec128_PowIntRound() {
	// Ten years of a 6% annual rate at daily rest, to the last place a Dec128 has. PowInt64 shares the same guarded
	// core but takes the scale the scale rule gives it, and is one unit in the last place low here.
	factor := FromString("1.000164383561643836")
	fmt.Println(factor.PowIntRound(3650, MaxScale, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	fmt.Println(factor.PowInt64(3650).StringFixed())

	// A negative exponent is the power of the reciprocal, so it is exact where the reciprocal is.
	fmt.Println(FromString("0.5").PowIntRound(-100, 0, ROUND_BANK).StringFixed())
	// Output:
	// 1.8220289545384488980
	// 1.8220289545384488979
	// 1267650600228229401496703205376
}

func ExampleDec128_Allocate() {
	// A fee split three ways: the odd cent goes to the largest remainder, and the parts add up to the whole.
	fee := FromString("100.00")
	shares, _ := fee.Allocate([]Dec128{FromInt64(1), FromInt64(1), FromInt64(1)}, 2)
	fmt.Println(shares[0].StringFixed(), shares[1].StringFixed(), shares[2].StringFixed())
	fmt.Println(Sum(shares[0], shares[1:]...).StringFixed())

	// Weighted, at a finer scale than the amount itself.
	shares, _ = FromString("1000.00").Allocate([]Dec128{FromString("0.5"), FromString("0.3"), FromString("0.2")}, 2)
	fmt.Println(shares[0].StringFixed(), shares[1].StringFixed(), shares[2].StringFixed())
	// Output:
	// 33.34 33.33 33.33
	// 100.00
	// 500.00 300.00 200.00
}

func ExampleDec128_RoundToMultiple() {
	// Swiss cash rounding to the nearest five centimes, and the retail-lending "next whole ten".
	fmt.Println(FromString("2.37").RoundToMultiple(FromString("0.05"), ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	fmt.Println(FromString("183.47").RoundToMultiple(FromInt64(10), ROUND_UP).StringFixed())
	// Disclosure rounding to the nearest thousand.
	fmt.Println(FromString("1234567").RoundToPlaces(-3, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	// A percentage as a fraction, without a division.
	fmt.Println(FromString("5.25").ScaleByPow10(-2).StringFixed())
	// Output:
	// 2.35
	// 190
	// 1235000
	// 0.0525
}

func ExampleDec128_SplitResidual() {
	// A seven-installment schedule where the last one absorbs the rounding of the other six, which is the
	// convention a loan agreement names. The installments still sum to the principal exactly.
	parts, _ := FromString("1000.00").SplitResidual(7, 2, 6, ROUND_HALF_AWAY_FROM_ZERO)
	texts := make([]string, len(parts))
	for i, p := range parts {
		texts[i] = p.StringFixed()
	}
	fmt.Println(strings.Join(texts, " "))
	fmt.Println(SumSlice(parts).StringFixed())

	// Largest remainder instead, where no share is named and none is more than a cent from its proportion.
	parts, _ = FromString("1000.00").Split(7, 2)
	fmt.Println(parts[0].StringFixed(), parts[6].StringFixed())
	// Output:
	// 142.86 142.86 142.86 142.86 142.86 142.86 142.84
	// 1000.00
	// 142.86 142.85
}

func ExampleDec128_NthRootRound() {
	// The monthly factor behind an annual one: the inverse of raising it to the twelfth power.
	annual := FromString("1.126825")
	monthly := annual.NthRootRound(12, 10, ROUND_HALF_AWAY_FROM_ZERO)
	fmt.Println(monthly.StringFixed())
	fmt.Println(monthly.PowIntRound(12, 6, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())

	// An exact root is exact, and a tie is broken by the mode like any other.
	fmt.Println(FromString("8").NthRootRound(3, 2, ROUND_BANK).StringFixed())
	fmt.Println(FromString("0.25").NthRootRound(2, 0, ROUND_BANK).StringFixed())
	// Output:
	// 1.0099999977
	// 1.126825
	// 2.00
	// 0
}

func ExampleDec128_FitsNumeric() {
	// The check before the insert: the column is NUMERIC(19,4), and a value that does not fit is either made to fit
	// or reported, rather than left for the database to refuse mid-batch.
	for _, s := range []string{"1234.5", "1234.56789", "1234567890123456.789"} {
		d := FromString(s)
		if d.FitsNumeric(19, 4) {
			fmt.Printf("%-22s fits\n", s)
			continue
		}
		rounded, lost := d.RescaleRoundInexact(4, ROUND_BANK)
		if rounded.FitsNumeric(19, 4) {
			fmt.Printf("%-22s fits as %s (lost digits: %v)\n", s, rounded.StringFixed(), lost)
			continue
		}
		fmt.Printf("%-22s needs %d integer digits, and the column has %d\n", s, d.IntegerDigits(), 19-4)
	}
	// Output:
	// 1234.5                 fits
	// 1234.56789             fits as 1234.5679 (lost digits: true)
	// 1234567890123456.789   needs 16 integer digits, and the column has 15
}

func ExampleDec128_Format() {
	d := FromString("1234.50")
	fmt.Printf("%v|%s|%f|%.2f|%.0f\n", d, d, d, d, d)
	fmt.Printf("%10.2f|%-10.2f|%010.2f\n", d, d, d)
	fmt.Printf("%e|%d|%+.2f\n", d, d, d)

	// The rounding is the decimal one, so it does not inherit a float's representation error: the float64 nearest
	// 2.675 is a shade below it, which is why Go's own %.2f gives 2.67 there.
	fmt.Printf("%.2f %.2f\n", FromString("2.675"), 2.675)
	// Output:
	// 1234.5|1234.5|1234.50|1234.50|1234
	//    1234.50|1234.50   |0001234.50
	// 1.2345e+3|1234|+1234.50
	// 2.68 2.67
}

func ExampleAccumulator_Mean() {
	// The mean of a batch, rounded once: the exact total is divided by the count and brought to the scale in one
	// decision, rather than rounding a total and then rounding a quotient.
	acc := NewAccumulator(2)
	for _, s := range []string{"10.00", "20.00", "40.05"} {
		acc.Add(FromString(s))
	}
	fmt.Println(acc.Count(), acc.Total(2, ROUND_BANK).StringFixed(), acc.Mean(2, ROUND_BANK).StringFixed())

	// and the accumulator is reusable for the next batch
	acc.Reset()
	acc.Add(FromString("1.00"))
	fmt.Println(acc.Count(), acc.Mean(2, ROUND_BANK).StringFixed())
	// Output:
	// 3 70.05 23.35
	// 1 1.00
}

func ExampleDec128_MulDivRound() {
	// Proration: a fee shared out in proportion to a weight. The exact share is a rational whose decimal expansion
	// does not terminate, so there is no factor to convert to a decimal and multiply by - the numerator and the
	// divisor have to stay apart until the single rounding.
	fee := FromString("1119.32")
	weight, total := FromString("25.12"), FromString("204.20")

	fmt.Println(fee.MulDivRound(weight, total, 2, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	// The composed form rounds the product first, and that rounding moves the quotient by a quantum.
	fmt.Println(fee.MulRound(weight, 2, ROUND_HALF_AWAY_FROM_ZERO).
		DivRound(total, 2, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	// Output:
	// 137.69
	// 137.70
}

func ExampleDec128_MulDivRoundInt64() {
	// Interest accrued over 31 days on an Act/365 basis. 31/365 has no decimal form, and the day count arrives as a
	// pair of integers rather than as a decimal, which is the case this form is for.
	interest := FromString("10000.00").MulRound(FromString("0.0525"), MaxScale, ROUND_HALF_AWAY_FROM_ZERO)

	fmt.Println(interest.MulDivRoundInt64(31, 365, 2, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	// Output:
	// 44.59
}

func ExampleSumRound() {
	// The terms are totalled exactly and only the total is rounded, so a payment split three ways adds back up to
	// the whole rather than to a quantum short of it.
	third := FromString("0.3333333333333333333")

	fmt.Println(SumRound(MaxScale, ROUND_HALF_AWAY_FROM_ZERO, third, third, third).StringFixed())
	fmt.Println(SumRound(2, ROUND_HALF_AWAY_FROM_ZERO, third, third, third).StringFixed())
	// Output:
	// 0.9999999999999999999
	// 1.00
}

func ExampleAvgRound() {
	// The exact total divided by the count, with the division and the reduction to scale taking one rounding
	// decision together rather than one each.
	amounts := []Dec128{FromString("10.00"), FromString("20.00"), FromString("30.01")}

	fmt.Println(AvgRound(2, ROUND_HALF_AWAY_FROM_ZERO, amounts[0], amounts[1:]...).StringFixed())
	fmt.Println(AvgRound(MaxScale, ROUND_HALF_AWAY_FROM_ZERO, amounts[0], amounts[1:]...).StringFixed())
	// Output:
	// 20.00
	// 20.0033333333333333333
}

func ExampleDec128_PowRational() {
	// A 6% annual factor over five months of a twelve-month year. The exponent 5/12 has no decimal form, so the
	// alternative is a twelfth root followed by a fifth power - and the root is where the significant digits go, so
	// raising it afterwards amplifies that error rather than cancelling it.
	annual := FromString("1.06")

	fmt.Println(annual.PowRational(5, 12, MaxScale, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	fmt.Println(annual.NthRootRound(12, MaxScale, ROUND_HALF_AWAY_FROM_ZERO).
		PowIntRound(5, MaxScale, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	// Output:
	// 1.0245758393924285985
	// 1.0245758393924285983
}

func ExampleDec128_PowRational_exact() {
	// Where the value is exact it comes back exact, in every mode, because the rounding decision is an exact integer
	// comparison rather than a guarded approximation.
	fmt.Println(FromInt64(8).PowRational(2, 3, 0, ROUND_BANK).StringFixed())
	fmt.Println(FromInt64(4).PowRational(-3, 2, 3, ROUND_BANK).StringFixed())
	// The fraction is reduced first, which for a negative base is the whole of the answer: 2/6 is 1/3, so this is
	// the cube root and not the sixth root of the square.
	fmt.Println(FromInt64(-8).PowRational(2, 6, 0, ROUND_BANK).StringFixed())
	// Output:
	// 4
	// 0.125
	// -2
}

func ExampleDec128_Ln() {
	// The two constants, to every place a Dec128 has.
	fmt.Println(FromInt64(2).Ln(MaxScale, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	fmt.Println(One.Exp(MaxScale, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	// Output:
	// 0.6931471805599453094
	// 2.7182818284590452354
}

func ExampleDec128_Ln1p() {
	// A 6% effective rate as a continuously-compounded force of interest, and back again. Ln1p and Expm1 are the
	// pair that converts between the two without the caller writing One.Add and Sub around them.
	rate := FromString("0.06")

	force := rate.Ln1p(MaxScale, ROUND_HALF_AWAY_FROM_ZERO)
	fmt.Println(force.StringFixed())
	fmt.Println(force.Expm1(MaxScale, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	// Output:
	// 0.0582689081239757755
	// 0.0600000000000000000
}

func ExampleDec128_Pow() {
	// Pow dispatches on the exponent. An integer goes to PowIntRound and allocates nothing.
	fmt.Println(FromString("1.05").Pow(FromInt64(360), 10, ROUND_BANK).StringFixed())
	// A fraction that reduces to small halves goes to PowRational and is correctly rounded, so a square root
	// arrives exactly.
	fmt.Println(FromInt64(9).Pow(FromString("0.5"), 4, ROUND_BANK).StringFixed())
	// Anything else is exp(e*ln d), which is faithfully rounded.
	fmt.Println(FromString("1.06").Pow(FromString("0.4166666666666666667"), MaxScale, ROUND_HALF_AWAY_FROM_ZERO).StringFixed())
	// Output:
	// 42476396.4086800204
	// 3.0000
	// 1.0245758393924285985
}
