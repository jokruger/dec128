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
	SetDefaultScale(19)
	a := FromString("1")
	b := FromString("3")
	fmt.Println(a.Div(b))
	// Output:
	// 0.3333333333333333333
}

func ExampleDec128_Sqrt() {
	a := FromString("4")
	fmt.Println(a.Sqrt())
	// Output:
	// 2
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
