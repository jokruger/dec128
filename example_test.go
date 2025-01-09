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
	a := FromString("1")
	b := FromString("3")
	fmt.Println(a.Div(b))
	// Output:
	// 0.3333333333333333333
}
