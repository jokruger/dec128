package dec128

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/jokruger/dec128/state"
	"github.com/jokruger/dec128/uint128"
)

func assertDecimal(s string, isNaN bool, isZero bool, isNegative bool, isPositive bool) error {
	d := FromString(s)

	if isNaN {
		if !d.IsNaN() {
			return fmt.Errorf("expected NaN, got: %s", d.String())
		}
	} else {
		if d.IsNaN() {
			return fmt.Errorf("expected not NaN")
		}
		if d.String() != s {
			return fmt.Errorf("expected %s, got: %s", s, d.String())
		}
	}

	if isZero && !d.IsZero() {
		return fmt.Errorf("expected zero")
	}
	if !isZero && d.IsZero() {
		return fmt.Errorf("expected not zero")
	}

	if isNegative && !d.IsNegative() {
		return fmt.Errorf("expected negative")
	}
	if !isNegative && d.IsNegative() {
		return fmt.Errorf("expected not negative")
	}

	if isPositive && !d.IsPositive() {
		return fmt.Errorf("expected positive")
	}
	if !isPositive && d.IsPositive() {
		return fmt.Errorf("expected not positive")
	}

	return nil
}

func assertDecimalAbsNeg(s string, abs string, neg string) error {
	d := FromString(s)
	if d.Abs().String() != abs {
		return fmt.Errorf("expected %s, got: %s", abs, d.String())
	}
	if d.Neg().String() != neg {
		return fmt.Errorf("expected %s, got: %s", neg, d.String())
	}
	return nil
}

func TestDecimalBasics1(t *testing.T) {
	SetDefaultPrecision(19)

	type dt struct {
		s      string
		isNaN  bool
		isZero bool
		isNeg  bool
		isPos  bool
	}
	dts := []dt{
		{"NaN", true, false, false, false},
		{"0", false, true, false, false},
		{"0.1", false, false, false, true},
		{"1", false, false, false, true},
		{"-0.1", false, false, true, false},
		{"-1", false, false, true, false},
	}
	for _, e := range dts {
		if err := assertDecimal(e.s, e.isNaN, e.isZero, e.isNeg, e.isPos); err != nil {
			t.Errorf("assertDecimal failed for %s: %s", e.s, err.Error())
		}
	}

	type dtan struct {
		s   string
		abs string
		neg string
	}
	dtans := []dtan{
		{"-123.456", "123.456", "123.456"},
		{"123.456", "123.456", "-123.456"},
	}
	for _, e := range dtans {
		if err := assertDecimalAbsNeg(e.s, e.abs, e.neg); err != nil {
			t.Errorf("assertDecimalAbsNeg failed for %s: %s", e.s, err.Error())
		}
	}

	a := FromString("NaN").Abs()
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}

	a = FromString("NaN").Neg()
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}

	a = FromString("NaN").Sqrt()
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}

	a = FromString("NaN").PowInt(2)
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}

	a = FromString("NaN").Mod(FromInt64(1))
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
	a = FromInt64(1).Mod(FromString("NaN"))
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}

	for i := range 1000 {
		j := i - 500
		a = FromInt(j)
		if b, err := a.Int(); err != nil || b != j {
			t.Errorf("expected %d, got: %d, error: %v", j, b, err)
		}
	}
}

func TestDecimalBasics2(t *testing.T) {
	SetDefaultPrecision(19)

	a := FromString("123.456")
	b := FromString("123.5")
	if a.Compare(b) != -1 {
		t.Errorf("expected -1, got: %d", a.Compare(b))
	}
	if b.Compare(a) != 1 {
		t.Errorf("expected 1, got: %d", b.Compare(a))
	}
	if a.Compare(a) != 0 {
		t.Errorf("expected 0, got: %d", a.Compare(a))
	}
	if !a.LessThan(b) {
		t.Errorf("expected true, got: %t", a.LessThan(b))
	}
	if b.LessThan(a) {
		t.Errorf("expected false, got: %t", b.LessThan(a))
	}
	if a.LessThan(a) {
		t.Errorf("expected false, got: %t", a.LessThan(a))
	}
	if a.GreaterThan(b) {
		t.Errorf("expected false, got: %t", a.GreaterThan(b))
	}
	if !b.GreaterThan(a) {
		t.Errorf("expected true, got: %t", b.GreaterThan(a))
	}
	if a.GreaterThan(a) {
		t.Errorf("expected false, got: %t", a.GreaterThan(a))
	}
	if !a.LessThanOrEqual(b) {
		t.Errorf("expected true, got: %t", a.LessThanOrEqual(b))
	}
	if !a.GreaterThanOrEqual(a) {
		t.Errorf("expected true, got: %t", a.GreaterThanOrEqual(a))
	}
}

func TestDecimalBasics3(t *testing.T) {
	SetDefaultPrecision(19)

	a := FromString("4").ModInt(3)
	if a.String() != "1" {
		t.Errorf("expected '1', got: %s", a.String())
	}

	q, r := FromString("4").QuoRemInt(3)
	if q.String() != "1" {
		t.Errorf("expected '1', got: %s", q.String())
	}
	if r.String() != "1" {
		t.Errorf("expected '1', got: %s", r.String())
	}

	a = FromString("4").Rescale(19)
	b := FromString("3").Rescale(19)
	c := a.Mod(b)
	if c.String() != "1" {
		t.Errorf("expected '1', got: %s", c.String())
	}

	a = FromString("4").Rescale(19)
	b = FromString("3").Rescale(10)
	c = a.Mod(b)
	if c.String() != "1" {
		t.Errorf("expected '1', got: %s", c.String())
	}

	a = FromString("4000000000000000000").Rescale(19)
	b = FromString("3000000000000000000").Rescale(10)
	c = a.Mod(b)
	if c.String() != "1000000000000000000" {
		t.Errorf("expected '1000000000000000000', got: %s", c.String())
	}

	a = FromString("4000000000000000000").Rescale(10)
	b = FromString("3000000000000000000").Rescale(19)
	c = a.Mod(b)
	if c.String() != "1000000000000000000" {
		t.Errorf("expected '1000000000000000000', got: %s", c.String())
	}

	a = FromInt(1).Rescale(19)
	b = FromString("40000000000000000000")
	c = a.Mod(b)
	if c.String() != "1" {
		t.Errorf("expected '1', got: %s", c.String())
	}

	a = FromInt(1).Rescale(19)
	b = FromString("35000000000000000000")
	c = a.Mod(b)
	if c.String() != "1" {
		t.Errorf("expected '1', got: %s", c.String())
	}

	a = FromString("4").Rescale(10)
	b = FromString("3").Rescale(19)
	c = a.Mod(b)
	if c.String() != "1" {
		t.Errorf("expected '1', got: %s", c.String())
	}

	q, r = a.QuoRem(b)
	if q.String() != "1" {
		t.Errorf("expected '1', got: %s", q.String())
	}
	if r.String() != "1" {
		t.Errorf("expected '1', got: %s", r.String())
	}
}

func TestDecimalNew(t *testing.T) {
	a := New(uint128.FromUint64(1), 19, false)
	if a.IsNaN() {
		t.Errorf("expected no error, got: %v", a.ErrorDetails())
	}
	if a.String() != "0.0000000000000000001" {
		t.Errorf("expected '0.0000000000000000001', got: %s", a.String())
	}

	a = New(uint128.FromUint64(1), 19, true)
	if a.IsNaN() {
		t.Errorf("expected no error, got: %v", a.ErrorDetails())
	}
	if a.String() != "-0.0000000000000000001" {
		t.Errorf("expected '-0.0000000000000000001', got: %s", a.String())
	}

	a = New(uint128.FromUint64(1), 20, false)
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
}

func TestDecimalNaN(t *testing.T) {
	a := NaN(state.DivisionByZero)
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
	if a.ErrorDetails().Error() != "division by zero" {
		t.Errorf("expected 'division by zero', got: %s", a.ErrorDetails().Error())
	}

	a = NaN(state.Default)
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
	if a.ErrorDetails().Error() != "logical error" {
		t.Errorf("expected 'logical error', got: %s", a.ErrorDetails().Error())
	}
}

func TestDecimalRescale(t *testing.T) {
	a := Decimal1.Rescale(19)
	if a.IsNaN() {
		t.Errorf("expected no error, got: %v", a.ErrorDetails())
	}
	if a.String() != "1" {
		t.Errorf("expected '1', got: %s", a.String())
	}

	a = Decimal1.Rescale(20)
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
}

func TestSign(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		a    string
		want int
	}

	testCases := [...]testCase{
		{"1234567890123456789", 1},
		{"123.123", 1},
		{"-123.123", -1},
		{"-123.1234567890123456789", -1},
		{"123.1234567890123456789", 1},
		{"123.1230000000000000001", 1},
		{"-123.1230000000000000001", -1},
		{"123.1230000000000000002", 1},
		{"-123.1230000000000000002", -1},
		{"123.123000000001", 1},
		{"-123.123000000001", -1},
		{"123.1230000", 1},
		{"123.1001", 1},
		{"0", 0},
		{"0.0", 0},
		{"-0", 0},
		{"-0.000", 0},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestSign(%s)", tc.a), func(t *testing.T) {
			a := FromString(tc.a)
			if a.IsNaN() {
				t.Errorf("expected no error, got: %v", a.ErrorDetails())
			}

			c := a.Sign()
			if c != tc.want {
				t.Errorf("expected %d, got: %d", tc.want, c)
			}
		})
	}
}

func TestDecimalAdd(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		a string
		b string
		s string
	}

	testCases := [...]testCase{
		{"0", "0", "0"},
		{"0", "1", "1"},
		{"1", "0", "1"},
		{"1", "1", "2"},
		{"-1", "0", "-1"},
		{"0", "-1", "-1"},
		{"-1", "-1", "-2"},
		{"-1", "1", "0"},
		{"1", "-1", "0"},
		{"1", "10", "11"},
		{"10", "1", "11"},
		{"-1", "-10", "-11"},
		{"-10", "-1", "-11"},
		{"-1", "10", "9"},
		{"10", "-1", "9"},
		{"1000000", "-0.0000001", "999999.9999999"},
		{"999999.9999999", "0.0000001", "1000000"},
		{"340282366920938463463374607431768211454", "1", "340282366920938463463374607431768211455"},
		{"340282366920938463463374607431768211454", "1.00", "340282366920938463463374607431768211455"}, // overflow due to precision fixed by auto canonicalization
		{"NaN", "1", "NaN"},
		{"1", "NaN", "NaN"},
		{"NaN", "NaN", "NaN"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalAdd(%s + %s)", tc.a, tc.b), func(t *testing.T) {
			a := FromString(tc.a)
			b := FromString(tc.b)
			c := a.Add(b)
			s := c.String()
			if s != tc.s {
				t.Errorf("expected '%s', got: %s", tc.s, s)
			}
		})
	}

	a := FromString("340282366920938463463374607431768211454")
	a = a.AddInt64(1)
	if a.IsNaN() {
		t.Errorf("expected no error, got: %v", a.ErrorDetails())
	}
	a = a.AddInt64(1)
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}

	a = FromString("NaN").Add(FromInt64(1))
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
	a = FromInt64(1).Add(FromString("NaN"))
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
}

func TestDecimalSub(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		a string
		b string
		s string
	}

	testCases := [...]testCase{
		{"0", "0", "0"},
		{"0", "1", "-1"},
		{"1", "0", "1"},
		{"1", "1", "0"},
		{"-1", "0", "-1"},
		{"0", "-1", "1"},
		{"-1", "-1", "0"},
		{"-1", "1", "-2"},
		{"1", "-1", "2"},
		{"1", "10", "-9"},
		{"10", "1", "9"},
		{"-1", "-10", "9"},
		{"-10", "-1", "-9"},
		{"-1", "10", "-11"},
		{"10", "-1", "11"},
		{"1000000", "0.0000001", "999999.9999999"},
		{"999999.9999999", "-0.0000001", "1000000"},
		{"340282366920938463463374607431768211455", "1", "340282366920938463463374607431768211454"},
		{"340282366920938463463374607431768211455", "1.00", "340282366920938463463374607431768211454"}, // overflow due to precision fixed by auto canonicalization
		{"NaN", "1", "NaN"},
		{"1", "NaN", "NaN"},
		{"NaN", "NaN", "NaN"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalSub(%s - %s)", tc.a, tc.b), func(t *testing.T) {
			a := FromString(tc.a)
			b := FromString(tc.b)
			c := a.Sub(b)
			s := c.String()
			if s != tc.s {
				t.Errorf("expected '%s', got: %s", tc.s, s)
			}
		})
	}

	a := FromString("-340282366920938463463374607431768211454")
	a = a.SubInt64(1)
	if a.IsNaN() {
		t.Errorf("expected no error, got: %v", a.ErrorDetails())
	}
	a = a.SubInt64(1)
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}

	a = FromString("NaN").Sub(FromInt64(1))
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
	a = FromInt64(1).Sub(FromString("NaN"))
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
}

func TestDecimalCompare(t *testing.T) {
	SetDefaultPrecision(19)

	var a, b Dec128

	a = FromString("NaN")
	b = FromString("NaN")
	if a.Compare(b) != 0 {
		t.Errorf("expected 0, got %d", a.Compare(b))
	}

	a = FromString("0")
	b = FromString("NaN")
	if a.Compare(b) != 1 {
		t.Errorf("expected 1, got %d", a.Compare(b))
	}
	if b.Compare(a) != -1 {
		t.Errorf("expected -1, got %d", b.Compare(a))
	}

	a = FromString("0")
	b = FromString("0")
	if a.Compare(b) != 0 {
		t.Errorf("expected 0, got %d", a.Compare(b))
	}

	a = FromString("1")
	b = FromString("-1")
	if a.Compare(b) != 1 {
		t.Errorf("expected 1, got %d", a.Compare(b))
	}
	if b.Compare(a) != -1 {
		t.Errorf("expected -1, got %d", b.Compare(a))
	}

	a = FromString(uint128.MaxUint128Str)
	b = FromString("0.0001")
	if a.Compare(b) != 1 {
		t.Errorf("expected 1, got %d", a.Compare(b))
	}
	if b.Compare(a) != -1 {
		t.Errorf("expected -1, got %d", b.Compare(a))
	}

	a = New(uint128.FromUint64(1000), 1, false)
	b = New(uint128.FromUint64(10000), 2, false)
	if a.Compare(b) != 0 {
		t.Errorf("expected 0, got %d", a.Compare(b))
	}

	a = FromString("123.456")
	b = FromString("123.4560000")
	if a.Compare(b) != 0 {
		t.Errorf("expected 0, got %d", a.Compare(b))
	}

	a = FromString("123.456")
	b = FromString("123.4560001")
	if a.Compare(b) != -1 {
		t.Errorf("expected -1, got %d", a.Compare(b))
	}
	if b.Compare(a) != 1 {
		t.Errorf("expected 1, got %d", b.Compare(a))
	}
}

func TestDecimalEqual(t *testing.T) {
	SetDefaultPrecision(19)

	var a, b Dec128

	a = FromString("NaN")
	b = FromString("NaN")
	if !a.Equal(b) {
		t.Errorf("expected true, got false")
	}

	a = FromString("0")
	b = FromString("NaN")
	if a.Equal(b) {
		t.Errorf("expected false, got true")
	}

	a = FromString("0")
	b = FromString("0")
	if !a.Equal(b) {
		t.Errorf("expected true, got false")
	}

	a = FromString("1")
	b = FromString("-1")
	if a.Equal(b) {
		t.Errorf("expected false, got true")
	}

	a = New(uint128.FromUint64(1000), 1, false)
	b = New(uint128.FromUint64(10000), 2, false)
	if !a.Equal(b) {
		t.Errorf("expected true, got false")
	}

	a = New(uint128.FromUint64(123456), 3, false)
	b = New(uint128.FromUint64(123456000), 6, false)
	if !a.Equal(b) {
		t.Errorf("expected true, got false")
	}

	a = FromString("123.456")
	b = FromString("123.4560000")
	if !a.Equal(b) {
		t.Errorf("expected true, got false")
	}

	a = FromString("0")
	b = FromString("0.00")
	if !a.Equal(b) {
		t.Errorf("expected true, got false")
	}
}

func TestDecimalMul(t *testing.T) {
	SetDefaultPrecision(19)

	a := FromInt(1).Rescale(19)
	if a.IsNaN() {
		t.Errorf("expected no error, got: %v", a.ErrorDetails())
	}
	b := FromInt(2).Rescale(19)
	if b.IsNaN() {
		t.Errorf("expected no error, got: %v", b.ErrorDetails())
	}
	c := a.Mul(b)
	if c.IsNaN() {
		t.Errorf("expected no error, got: %v", c.ErrorDetails())
	}
	if c.String() != "2" {
		t.Errorf("expected '2', got: %s", c.String())
	}

	type testCase struct {
		a string
		b string
		r string
		e string
	}

	testCases := [...]testCase{
		{"0", "0", "0", ""},
		{"0", "1", "0", ""},
		{"1", "0", "0", ""},
		{"1", "1", "1", ""},
		{"-1", "1", "-1", ""},
		{"1", "-1", "-1", ""},
		{"-1", "-1", "1", ""},
		{"-1", "10", "-10", ""},
		{"10", "-1", "-10", ""},
		{"-1", "-10", "10", ""},
		{"-10", "-1", "10", ""},
		{"-1", "0.1", "-0.1", ""},
		{"0.1", "-1", "-0.1", ""},
		{"0.1", "0.1", "0.01", ""},
		{"0.0000001", "0.0000001", "0.00000000000001", ""},
		{"1234567890", "0.0000001", "123.456789", ""},
		{"1234567890", "0.0000000001", "0.123456789", ""},
		{"1234567890.123456789", "0.0000000001", "0.1234567890123456789", ""},
		{"340282366920938463463374607431768211455", "1", "340282366920938463463374607431768211455", ""},
		{"340282366920938463463374607431768211455", "1.000000", "340282366920938463463374607431768211455", ""}, // overflow due to precision fixed by auto canonicalization
		{"340282366920938463463374607431768211455", "1.1", "NaN", "overflow"},
		{"NaN", "1", "NaN", "invalid format"},
		{"1", "NaN", "NaN", "invalid format"},
		{"NaN", "NaN", "NaN", "invalid format"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalMul(%s * %s)", tc.a, tc.b), func(t *testing.T) {
			a := FromString(tc.a)
			b := FromString(tc.b)
			c := a.Mul(b)
			s := c.String()
			if s != tc.r {
				t.Errorf("expected '%s', got: %s", tc.r, s)
			}
			if tc.e != "" && !c.IsNaN() {
				t.Errorf("expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && c.IsNaN() {
				t.Errorf("expected no error, got: %v", c.ErrorDetails())
			}
		})
	}

	a = FromString("1.2").MulInt64(2)
	if a.String() != "2.4" {
		t.Errorf("expected '2.4', got: %s", a.String())
	}

	a = FromString("NaN").Mul(FromInt64(1))
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
	a = FromInt64(1).Mul(FromString("NaN"))
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
}

func TestDecimalDiv(t *testing.T) {
	SetDefaultPrecision(10)

	type testCase struct {
		a string
		b string
		r string
		e string
	}

	testCases := [...]testCase{
		{"0", "0", "NaN", "division by zero"},
		{"NaN", "1", "NaN", "invalid format"},
		{"1", "NaN", "NaN", "invalid format"},
		{"NaN", "NaN", "NaN", "invalid format"},
		{"0", "1", "0", ""},
		{"1", "1", "1", ""},
		{"-1", "1", "-1", ""},
		{"1", "-1", "-1", ""},
		{"-1", "-1", "1", ""},
		{"10", "10", "1", ""},
		{"10", "10.00", "1", ""},
		{"100", "10", "10", ""},
		{"1", "0.1", "10", ""},
		{"1", "10", "0.1", ""},
		{"1", "0.0000001", "10000000", ""},
		{"1234567890", "10", "123456789", ""},
		{"1234567890", "1000", "1234567.89", ""},
		{"1234567890.123456789", "1000", "1234567.8901234567", ""},
		{"18446744073709551615", "1", "18446744073709551615", ""},
		{"18446744073709551615", "0.1", "184467440737095516150", ""},
		{"18446744073709551615", "0.0001", "184467440737095516150000", ""},
		{"18446744073709551615.000000000000000000", "0.0001", "184467440737095516150000", ""}, // overflow due to precision fixed by auto canonicalization
		{"12345678901234567890", "365", "33823777811601555.8630136986", ""},
		{"1", "2", "0.5", ""},
		{"1", "3", "0.3333333333", ""},
		{"1", "4", "0.25", ""},
		{"1", "5", "0.2", ""},
		{"1", "6", "0.1666666666", ""},
		{"1", "7", "0.1428571428", ""},
		{"1", "8", "0.125", ""},
		{"1", "9", "0.1111111111", ""},
		{"1", "1234567890123456789012345678901234567890", "NaN", "overflow"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalDiv(%s / %s)", tc.a, tc.b), func(t *testing.T) {
			a := FromString(tc.a)
			b := FromString(tc.b)
			c := a.Div(b)
			s := c.String()
			if s != tc.r {
				t.Errorf("expected '%s', got: %s", tc.r, s)
			}
			if tc.e != "" && !c.IsNaN() {
				t.Errorf("expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && c.IsNaN() {
				t.Errorf("expected no error, got: %v", c.ErrorDetails())
			}
		})
	}

	a := FromString("NaN").Div(FromInt64(1))
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
	a = FromInt64(1).Div(FromString("NaN"))
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}

	a = Decimal1
	b := FromString("0.0000001")
	a = a.Div(b)
	a = a.Div(b)
	a = a.Div(b)
	a = a.Div(b)
	a = a.Div(b)
	if !a.IsNaN() {
		t.Errorf("expected NaN, got: %s", a.String())
	}
}

func TestDecimalDiv2(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		a string
		b string
		r string
		e string
	}

	testCases := [...]testCase{
		{"1", "0.0000001", "10000000", ""},
		{"12345678901234567890", "365", "33823777811601555.8630136986301369863", ""},
		{"1", "3", "0.3333333333333333333", ""},
		{"1", "6", "0.1666666666666666666", ""},
		{"1", "7", "0.1428571428571428571", ""},
		{"1", "9", "0.1111111111111111111", ""},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalDiv(%s / %s)", tc.a, tc.b), func(t *testing.T) {
			a := FromString(tc.a)
			b := FromString(tc.b)
			c := a.Div(b)
			s := c.String()
			if s != tc.r {
				t.Errorf("expected '%s', got: %s", tc.r, s)
			}
			if tc.e != "" && !c.IsNaN() {
				t.Errorf("expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && c.IsNaN() {
				t.Errorf("expected no error, got: %v", c.ErrorDetails())
			}
		})
	}
}

func TestDecimalDiv3(t *testing.T) {
	SetDefaultPrecision(6)

	type testCase struct {
		a string
		b string
		r string
		e string
	}

	testCases := [...]testCase{
		{"1", "0.0000001", "10000000", ""},
		{"12345678901234567890", "365", "33823777811601555.863013", ""},
		{"1", "3", "0.333333", ""},
		{"1", "6", "0.166666", ""},
		{"1", "7", "0.142857", ""},
		{"1", "9", "0.111111", ""},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalDiv(%s / %s)", tc.a, tc.b), func(t *testing.T) {
			a := FromString(tc.a)
			b := FromString(tc.b)
			c := a.Div(b)
			s := c.String()
			if s != tc.r {
				t.Errorf("expected '%s', got: %s", tc.r, s)
			}
			if tc.e != "" && !c.IsNaN() {
				t.Errorf("expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && c.IsNaN() {
				t.Errorf("expected no error, got: %v", c.ErrorDetails())
			}
		})
	}
}

func TestDecimalMod1(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		a string
		b string
		r string
		e string
	}

	testCases := [...]testCase{
		{"0", "0", "NaN", "division by zero"},
		{"123", "10", "3", ""},
		{"0", "1", "0", ""},
		{"12345678901234567890123456.123456789", "123456789012345678900", "123456.123456789", ""},
		{"12345678901234567890123", "1.1234567890123456789", "0.4794672386555312197", ""},
		{"12345678901234567890.123456789", "1.1234567890123456789", "0.592997984048161704", ""},
		{"123456789.1234567890123456789", "123.123456789", "37.1369289660123456789", ""},
		{"1234567890123456789", "1", "0", ""},
		{"11.234", "1.12", "0.034", ""},
		{"-11.234", "1.12", "-0.034", ""},
		{"11.234", "-1.12", "0.034", ""},
		{"-11.234", "-1.12", "-0.034", ""},
		{"123.456", "1.123", "1.049", ""},
		{"123.4560000000000", "1.1230000000000", "1.049", ""},
		{"3", "2", "1", ""},
		{"3451204593", "2454495034", "996709559", ""},
		{"9999999999", "1275", "324", ""},
		{"9999999999.9999998", "1275.49", "239.2399998", ""},
		{"24544.95034", "0.3451204593", "0.3283950433", ""},
		{"0.499999999999999999", "0.25", "0.249999999999999999", ""},
		{"0.989512958912895912", "0.000001", "0.000000958912895912", ""},
		{"0.1", "0.1", "0", ""},
		{"-7.5", "2", "-1.5", ""},
		{"7.5", "-2", "1.5", ""},
		{"-7.5", "-2", "-1.5", ""},
		{"41", "21", "20", ""},
		{"400000000001", "200000000001", "200000000000", ""},
		{"340282366920938463463374607431768211455", "3", "0", ""},
		{"340282366920938463463374607431768211455", "100000000000000000000000000000000000001", "40282366920938463463374607431768211452", ""},
		{"99999999999999999999999999999999999999", "100000000000000000000000000000000000001", "99999999999999999999999999999999999999", ""},
		{"123456789012345678901234567890123456789", "100000000000000000000000000000000000001", "23456789012345678901234567890123456788", ""},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalDiv(%s / %s)", tc.a, tc.b), func(t *testing.T) {
			a := FromString(tc.a)
			b := FromString(tc.b)
			c := a.Mod(b)
			s := c.String()
			if s != tc.r {
				t.Errorf("expected '%s', got: %s", tc.r, s)
			}
			if tc.e != "" && !c.IsNaN() {
				t.Errorf("expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && c.IsNaN() {
				t.Errorf("expected no error, got: %v", c.ErrorDetails())
			}
		})
	}
}

func TestDecimalQuoRem(t *testing.T) {
	type testCase struct {
		a string
		b string
		q string
		r string
		e string
	}

	testCases := [...]testCase{
		{"NaN", "1", "NaN", "NaN", "invalid format"},
		{"1", "NaN", "NaN", "NaN", "invalid format"},
		{"0", "0", "NaN", "NaN", "division by zero"},
		{"0", "1", "0", "0", ""},
		{"1", "0", "NaN", "NaN", "division by zero"},
		{"1", "1", "1", "0", ""},
		{"-1", "1", "-1", "0", ""},
		{"10", "1", "10", "0", ""},
		{"1", "10", "0", "1", ""},
		{"1", "4", "0", "1", ""},
		{"1", "8", "0", "1", ""},
		{"10", "3", "3", "1", ""},
		{"100", "3", "33", "1", ""},
		{"1000", "3", "333", "1", ""},
		{"1000", "10", "100", "0", ""},
		{"-4", "3", "-1", "-1", ""},
		{"-4", "-3", "1", "-1", ""},
		{"3451204593", "2454495034", "1", "996709559", ""},
		{"9999999999", "1275", "7843137", "324", ""},
		{"9999999999.9999998", "1275.49", "7840124", "239.2399998", ""},
		{"24544.95034", "0.3451204593", "71119", "0.3283950433", ""},
		{"0.499999999999999999", "0.25", "1", "0.249999999999999999", ""},
		{"0.989512958912895912", "0.000001", "989512", "0.000000958912895912", ""},
		{"400000000001", "200000000001", "1", "200000000000", ""},
		{"340282366920938463463374607431768211455", "3", "113427455640312821154458202477256070485", "0", ""},
		{"340282366920938463463374607431768211455", "100000000000000000000000000000000000001", "3", "40282366920938463463374607431768211452", ""},
		{"99999999999999999999999999999999999999", "100000000000000000000000000000000000001", "0", "99999999999999999999999999999999999999", ""},
		{"123456789012345678901234567890123456789", "100000000000000000000000000000000000001", "1", "23456789012345678901234567890123456788", ""},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalQuoRem(%s / %s)", tc.a, tc.b), func(t *testing.T) {
			a := FromString(tc.a)
			b := FromString(tc.b)
			q, r := a.QuoRem(b)
			s := q.String()
			if s != tc.q {
				t.Errorf("expected '%s', got: %s", tc.q, s)
			}
			s = r.String()
			if s != tc.r {
				t.Errorf("expected '%s', got: %s", tc.r, s)
			}
			if tc.e != "" && !q.IsNaN() {
				t.Errorf("expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && q.IsNaN() {
				t.Errorf("expected no error, got: %v", q.ErrorDetails())
			}
		})
	}
}

func TestDecimalPowInt(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		a string
		p int
		s string
		e string
	}

	testCases := [...]testCase{
		{"0", 0, "1", ""},
		{"0", 1, "0", ""},
		{"0", 2, "0", ""},
		{"0", 10, "0", ""},
		{"0", -1, "NaN", "division by zero"},
		{"1", 0, "1", ""},
		{"1", 1, "1", ""},
		{"1", 2, "1", ""},
		{"1", 10, "1", ""},
		{"1", -1, "1", ""},
		{"1", -2, "1", ""},
		{"1", -10, "1", ""},
		{"2", 0, "1", ""},
		{"2", 1, "2", ""},
		{"2", 2, "4", ""},
		{"2", 10, "1024", ""},
		{"2", -1, "0.5", ""},
		{"2", -2, "0.25", ""},
		{"2", -10, "0.0009765625", ""},
		{"0.000001", 0, "1", ""},
		{"0.000001", 1, "0.000001", ""},
		{"0.000001", 2, "0.000000000001", ""},
		{"0.000001", 10, "NaN", "overflow"},
		{"0.000001", -1, "1000000", ""},
		{"0.000001", -2, "1000000000000", ""},
		{"0.000001", -10, "NaN", "overflow"},
		{"12345.6789", 0, "1", ""},
		{"12345.6789", 1, "12345.6789", ""},
		{"12345.6789", 2, "152415787.50190521", ""},
		{"12345.6789", 3, "1881676371789.154860897069", ""},
		{"12345.6789", -1, "0.0000810000007371", ""},
		{"12345.6789", -2, "0.0000000065610001194", ""},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalPowInt(%s^%d)", tc.a, tc.p), func(t *testing.T) {
			r := FromString(tc.a).PowInt(tc.p)
			if r.String() != tc.s {
				t.Errorf("expected %s, got %s", tc.s, r.String())
			}
			if tc.e == "" && r.IsNaN() {
				t.Errorf("expected a valid result, got %s", r.ErrorDetails().Error())
			}
			if tc.e != "" && (!r.IsNaN() || r.ErrorDetails().Error() != tc.e) {
				t.Errorf("expected %s, got %v", tc.e, r.ErrorDetails())
			}
		})
	}
}

func TestDecimalSqrt(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		a string
		r string
		e string
	}

	testCases := [...]testCase{
		{"0", "0", ""},
		{"1", "1", ""},
		{"4", "2", ""},
		{"9", "3", ""},
		{"16", "4", ""},
		{"25", "5", ""},
		{"100", "10", ""},
		{"10000", "100", ""},
		{"2", "1.4142135623730950488", ""},
		{"1234567890.123456789", "35136.4182882014425309365", ""},
		{"0.1", "0.3162277660168379331", ""},
		{"-1", "NaN", "square root of negative number"},
		{"10000000000", "100000", ""},
		{"1000", "31.6227766016837933199", ""},
		{"31.6227766016837933199", "5.6234132519034908039", ""},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalSqrt(%s)", tc.a), func(t *testing.T) {
			d := FromString(tc.a).Sqrt()
			if d.String() != tc.r {
				t.Errorf("expected %s, got %s", tc.r, d.String())
			}
			if tc.e == "" && d.IsNaN() {
				t.Errorf("expected no error, got %s", d.ErrorDetails().Error())
			}
			if tc.e != "" && (!d.IsNaN() || d.ErrorDetails().Error() != tc.e) {
				t.Errorf("expected %s, got %s", tc.e, d.ErrorDetails().Error())
			}
		})
	}
}

func TestDecimalSqrt2(t *testing.T) {
	SetDefaultPrecision(6)

	type testCase struct {
		a string
		r string
		e string
	}

	testCases := [...]testCase{
		{"0", "0", ""},
		{"1", "1", ""},
		{"4", "2", ""},
		{"9", "3", ""},
		{"16", "4", ""},
		{"25", "5", ""},
		{"100", "10", ""},
		{"10000", "100", ""},
		{"2", "1.414213", ""},
		{"3", "1.73205", ""},
		{"0.1", "0.316227", ""},
		{"10000000000", "100000", ""},
		{"1000", "31.622776", ""},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalSqrt(%s)", tc.a), func(t *testing.T) {
			d := FromString(tc.a).Sqrt()
			if d.String() != tc.r {
				t.Errorf("expected %s, got %s", tc.r, d.String())
			}
			if tc.e == "" && d.IsNaN() {
				t.Errorf("expected no error, got %s", d.ErrorDetails().Error())
			}
			if tc.e != "" && (!d.IsNaN() || d.ErrorDetails().Error() != tc.e) {
				t.Errorf("expected %s, got %s", tc.e, d.ErrorDetails().Error())
			}
		})
	}
}

func TestDecimalCanonical(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i  string
		s  string
		e1 uint8
		e2 uint8
	}

	testCases := [...]testCase{
		{"0", "0", 0, 0},
		{"1", "1", 0, 0},
		{"10", "10", 0, 0},
		{"100", "100", 0, 0},
		{"1.0", "1", 1, 0},
		{"1.00", "1", 2, 0},
		{"1.000", "1", 3, 0},
		{"1.01", "1.01", 2, 2},
		{"1.010", "1.01", 3, 2},
		{"1.001", "1.001", 3, 3},
		{"1.0010", "1.001", 4, 3},
		{"1.00100", "1.001", 5, 3},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalCanonical(%s)", tc.i), func(t *testing.T) {
			d := FromString(tc.i)
			if d.IsNaN() {
				t.Errorf("expected no error, got: %v", d.ErrorDetails())
			}
			c := d.Canonical()
			if c.IsNaN() {
				t.Errorf("expected no error, got: %v", c.ErrorDetails())
			}
			s := c.String()
			if s != tc.s {
				t.Errorf("expected '%s', got: %s", tc.s, s)
			}
			if d.Precision() != tc.e1 {
				t.Errorf("expected %d, got: %d", tc.e1, d.Precision())
			}
			if c.Precision() != tc.e2 {
				t.Errorf("expected %d, got: %d", tc.e2, c.Precision())
			}
		})
	}

	if !FromString("NaN").Canonical().IsNaN() {
		t.Errorf("expected NaN, got: %s", FromString("NaN").Canonical().String())
	}
}

func TestDecimalToInt64(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		s string
		i int64
	}

	testCases := [...]testCase{
		{"0", 0},
		{"1", 1},
		{"-1", -1},
		{"123456.123456", 123456},
		{"1.999", 1},
		{"9223372036854775807", 9223372036854775807},
		{"-9223372036854775808", -9223372036854775808},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalToInt(%v)", tc.s), func(t *testing.T) {
			d := FromString(tc.s)
			i, err := d.Int64()
			if err != nil {
				t.Errorf("Int64() returned error: %v", err)
			}
			if i != tc.i {
				t.Errorf("Int64() returned %v, expected %v", i, tc.i)
			}
		})
	}
}

func TestDecimalInt64Encoding(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i int64
		p uint8
		s string
	}

	testCases := [...]testCase{
		{0, 0, "0"},
		{0, 1, "0"},
		{1, 0, "1"},
		{-1, 0, "-1"},
		{1, 1, "0.1"},
		{-1, 1, "-0.1"},
		{10, 1, "1"},
		{-10, 1, "-1"},
		{100, 1, "10"},
		{-100, 1, "-10"},
		{1000, 1, "100"},
		{-1000, 1, "-100"},
		{1, 10, "0.0000000001"},
		{-1, 10, "-0.0000000001"},
		{10, 10, "0.000000001"},
		{-10, 10, "-0.000000001"},
		{100, 10, "0.00000001"},
		{-100, 10, "-0.00000001"},
		{1000, 10, "0.0000001"},
		{-1000, 10, "-0.0000001"},
		{9223372036854775807, 0, "9223372036854775807"},
		{9223372036854775807, 1, "922337203685477580.7"},
		{9223372036854775807, 10, "922337203.6854775807"},
		{-9223372036854775808, 0, "-9223372036854775808"},
		{-9223372036854775808, 1, "-922337203685477580.8"},
		{-9223372036854775808, 10, "-922337203.6854775808"},
		{123456789, 3, "123456.789"},
		{-123456789, 3, "-123456.789"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("DecimalInt64Encoding(%v)", tc), func(t *testing.T) {
			d := DecodeFromInt64(tc.i, tc.p)
			s := d.String()
			if s != tc.s {
				t.Errorf("expected '%s', got: %s", tc.s, s)
			}

			d = FromString(tc.s)
			i, err := d.EncodeToInt64(tc.p)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if i != tc.i {
				t.Errorf("expected %d, got: %d", tc.i, i)
			}
		})
	}
}

func TestDecimalFromUint64Encoding(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i uint64
		p uint8
		s string
	}

	testCases := [...]testCase{
		{0, 0, "0"},
		{0, 1, "0"},
		{1, 0, "1"},
		{1, 1, "0.1"},
		{10, 1, "1"},
		{100, 1, "10"},
		{1000, 1, "100"},
		{1, 10, "0.0000000001"},
		{10, 10, "0.000000001"},
		{100, 10, "0.00000001"},
		{1000, 10, "0.0000001"},
		{18446744073709551615, 0, "18446744073709551615"},
		{18446744073709551615, 1, "1844674407370955161.5"},
		{18446744073709551615, 10, "1844674407.3709551615"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalFromUint64(%v)", tc), func(t *testing.T) {
			d := New(uint128.FromUint64(tc.i), tc.p, false)
			s := d.String()
			if s != tc.s {
				t.Errorf("expected '%s', got: %s", tc.s, s)
			}
		})
	}
}

func TestDecimalUint64Encoding(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		u uint64
		p uint8
		e string
	}

	testCases := [...]testCase{
		{"NaN", 0, 0, "not a number"},
		{"0", 0, 0, ""},
		{"1", 1, 0, ""},
		{"10", 10, 0, ""},
		{"100", 100, 0, ""},
		{"1000", 1000, 0, ""},
		{"1000000", 1000000, 0, ""},
		{"1.1", 11, 1, ""},
		{"1.01", 101, 2, ""},
		{"18446744073709551615", 18446744073709551615, 0, ""},
		{"1844674407370955161.5", 18446744073709551615, 1, ""},
		{"1844674407.3709551615", 18446744073709551615, 10, ""},
		{"18446744073709551616", 0, 0, "overflow"},
		{"-1", 0, 0, "negative"},
		{"1", 1000000, 6, ""},
		{"123", 123000000, 6, ""},
		{"123.456", 123456000, 6, ""},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalUint64Encoding(%s)", tc.i), func(t *testing.T) {
			d := FromString(tc.i)
			u, err := d.EncodeToUint64(tc.p)
			if tc.e != "" && err == nil {
				t.Errorf("expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if u != tc.u {
				t.Errorf("expected %d, got: %d", tc.u, u)
			}
		})
	}
}

func TestDecimalUint64Encoding2(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"0", 3, "0"},
		{"123", 3, "123"},
		{"123.456", 3, "123.456"},
		{"1234567890.123456", 3, "1234567890.123"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalUint64Encoding2(%s)", tc.i), func(t *testing.T) {
			d := FromString(tc.i)
			u, err := d.EncodeToUint64(tc.p)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			s := New(uint128.FromUint64(u), tc.p, false).String()
			if s != tc.s {
				t.Errorf("expected '%s', got: %s", tc.s, s)
			}
		})
	}
}

func TestDecimalUint128Encoding(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"0", 6, "0"},
		{"1", 6, "1"},
		{"1.1", 6, "1.1"},
		{"1.01", 6, "1.01"},
		{"123.456", 6, "123.456"},
		{"1234567890.1234567890", 6, "1234567890.123456"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalFromUint128(%v)", tc), func(t *testing.T) {
			d := FromString(tc.i)
			u, err := d.EncodeToUint128(tc.p)
			if err != nil {
				t.Errorf("error: %v", err)
			}
			s := New(u, tc.p, false).String()
			if s != tc.s {
				t.Errorf("expected: %v, got: %v", tc.s, s)
			}
		})
	}
}

func TestDecimalRoundDown(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"Nan", 0, "NaN"},
		{"0", 0, "0"},
		{"123.456000", 0, "123"},
		{"123.1234567890987654321", 0, "123"},
		{"123.12354", 0, "123"},
		{"123.12454", 0, "123"},
		{"123.1235", 0, "123"},
		{"123.1245", 0, "123"},
		{"-123.456000", 0, "-124"},
		{"-123.1234567890987654321", 0, "-124"},
		{"-123.12354", 0, "-124"},
		{"-123.12454", 0, "-124"},
		{"-123.1235", 0, "-124"},
		{"-123.1245", 0, "-124"},
		{"1.12345", 0, "1"},
		{"1.12335", 0, "1"},
		{"1.5", 0, "1"},
		{"2.5", 0, "2"},
		{"1", 0, "1"},
		{"-1.5", 0, "-2"},
		{"-2.5", 0, "-3"},
		{"-1", 0, "-1"},
		{"9999999999999999999.9999999999999999999", 0, "9999999999999999999"},
		{"-9999999999999999999.9999999999999999999", 0, "-10000000000000000000"},
		{"23.7", 0, "23"},
		{"-23.2", 0, "-24"},
		{"1.236", 2, "1.23"},
		{"1.235", 2, "1.23"},
		{"1.234", 2, "1.23"},
		{"-1.234", 2, "-1.24"},
		{"-1.235", 2, "-1.24"},
		{"-1.236", 2, "-1.24"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalRoundDown(%v)", tc), func(t *testing.T) {
			d := FromString(tc.i)
			s := d.RoundDown(tc.p).StringFixed()
			if s != tc.s {
				t.Errorf("RoundDown(%v) = %v, want %v", tc.i, s, tc.s)
			}
		})
	}
}

func TestDecimalRoundUp(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"Nan", 0, "NaN"},
		{"0", 0, "0"},
		{"123.456000", 0, "124"},
		{"-123.456000", 0, "-123"},
		{"123.1234567890987654321", 0, "124"},
		{"-123.1234567890987654321", 0, "-123"},
		{"123.12454", 0, "124"},
		{"123.1235", 0, "124"},
		{"123.1245", 0, "124"},
		{"-123.12354", 0, "-123"},
		{"-123.12454", 0, "-123"},
		{"-123.1235", 0, "-123"},
		{"-123.1245", 0, "-123"},
		{"1.12345", 0, "2"},
		{"1.12335", 0, "2"},
		{"1.5", 0, "2"},
		{"2.5", 0, "3"},
		{"1", 0, "1"},
		{"-1", 0, "-1"},
		{"-1.5", 0, "-1"},
		{"-2.5", 0, "-2"},
		{"9999999999999999999.9999999999999999999", 0, "10000000000000000000"},
		{"-9999999999999999999.9999999999999999999", 0, "-9999999999999999999"},
		{"23.2", 0, "24"},
		{"-23.7", 0, "-23"},
		{"1.236", 2, "1.24"},
		{"1.235", 2, "1.24"},
		{"1.234", 2, "1.24"},
		{"-1.234", 2, "-1.23"},
		{"-1.235", 2, "-1.23"},
		{"-1.236", 2, "-1.23"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalCeil(%v)", tc), func(t *testing.T) {
			d := FromString(tc.i)
			s := d.RoundUp(tc.p).StringFixed()
			if s != tc.s {
				t.Errorf("RoundUp(%v) = %v, want %v", tc.i, s, tc.s)
			}
		})
	}
}

func TestRoundTowardZero(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"Nan", 2, "NaN"},
		{"0", 0, "0"},
		{"0", 1, "0"},
		{"1.12345", 4, "1.1234"},
		{"1.12335", 4, "1.1233"},
		{"23.7", 0, "23"},
		{"-23.7", 0, "-23"},
		{"1.236", 2, "1.23"},
		{"1.235", 2, "1.23"},
		{"1.234", 2, "1.23"},
		{"-1.234", 2, "-1.23"},
		{"-1.235", 2, "-1.23"},
		{"-1.236", 2, "-1.23"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestRoundTowardZero(%v)", tc), func(t *testing.T) {
			d := FromString(tc.i)
			s := d.RoundTowardZero(tc.p).StringFixed()
			if s != tc.s {
				t.Errorf("RoundTowardZero(%v, %v) = %v, want %v", tc.i, tc.p, s, tc.s)
			}
		})
	}
}

func TestDecimalRoundAwayFromZero(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"Nan", 2, "NaN"},
		{"0", 0, "0"},
		{"1.12345", 4, "1.1235"},
		{"1.12335", 4, "1.1234"},
		{"1.5", 0, "2"},
		{"-1.5", 0, "-2"},
		{"1.12", 1, "1.2"},
		{"1.15", 1, "1.2"},
		{"-1.12", 1, "-1.2"},
		{"-1.15", 1, "-1.2"},
		{"9999999999999999999.9999999999999999999", 3, "10000000000000000000.000"},
		{"-9999999999999999999.9999999999999999999", 3, "-10000000000000000000.000"},
		{"123.456000", 0, "124"},
		{"123.456000", 4, "123.4560"},
		{"123.1234567890987654321", 6, "123.123457"},
		{"-123.456000", 7, "-123.456000"},
		{"-123.1234567890987654321", 7, "-123.1234568"},
		{"23.2", 0, "24"},
		{"-23.2", 0, "-24"},
		{"1.236", 2, "1.24"},
		{"1.235", 2, "1.24"},
		{"1.234", 2, "1.24"},
		{"-1.234", 2, "-1.24"},
		{"-1.235", 2, "-1.24"},
		{"-1.236", 2, "-1.24"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalRoundAwayFromZero(%v)", tc), func(t *testing.T) {
			d := FromString(tc.i)
			s := d.RoundAwayFromZero(tc.p).StringFixed()
			if s != tc.s {
				t.Errorf("RoundAwayFromZero(%v, %v) = %v, want %v", tc.i, tc.p, s, tc.s)
			}
		})
	}
}

func TestDecimalRoundHalfTowardZero(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"Nan", 2, "NaN"},
		{"0", 0, "0"},
		{"1.12345", 4, "1.1234"},
		{"1.12335", 4, "1.1233"},
		{"1.5", 0, "1"},
		{"-1.5", 0, "-1"},
		{"123.456000", 0, "123"},
		{"123.456000", 1, "123.5"},
		{"123.456000", 2, "123.46"},
		{"123.456000", 3, "123.456"},
		{"123.456000", 4, "123.4560"},
		{"123.456000", 5, "123.45600"},
		{"123.456000", 6, "123.456000"},
		{"123.456000", 7, "123.456000"},
		{"-123.456000", 0, "-123"},
		{"-123.456000", 1, "-123.5"},
		{"-123.456000", 2, "-123.46"},
		{"-123.456000", 3, "-123.456"},
		{"-123.456000", 4, "-123.4560"},
		{"-123.456000", 5, "-123.45600"},
		{"-123.456000", 6, "-123.456000"},
		{"-123.456000", 7, "-123.456000"},
		{"123.1234567890987654321", 0, "123"},
		{"123.1234567890987654321", 1, "123.1"},
		{"123.1234567890987654321", 2, "123.12"},
		{"123.1234567890987654321", 3, "123.123"},
		{"123.1234567890987654321", 4, "123.1235"},
		{"123.1234567890987654321", 5, "123.12346"},
		{"123.1234567890987654321", 6, "123.123457"},
		{"123.1234567890987654321", 7, "123.1234568"},
		{"123.1234567890987654321", 8, "123.12345679"},
		{"123.1234567890987654321", 9, "123.123456789"},
		{"123.1234567890987654321", 10, "123.1234567891"},
		{"123.1234567890987654321", 11, "123.12345678910"},
		{"123.1234567890987654321", 12, "123.123456789099"},
		{"123.1234567890987654321", 13, "123.1234567890988"},
		{"123.1234567890987654321", 14, "123.12345678909877"},
		{"123.1234567890987654321", 15, "123.123456789098765"},
		{"123.1234567890987654321", 16, "123.1234567890987654"},
		{"123.1234567890987654321", 17, "123.12345678909876543"},
		{"123.1234567890987654321", 18, "123.123456789098765432"},
		{"123.1234567890987654321", 19, "123.1234567890987654321"},
		{"123.1234567890987654321", 20, "123.1234567890987654321"},
		{"-123.1234567890987654321", 0, "-123"},
		{"-123.1234567890987654321", 1, "-123.1"},
		{"-123.1234567890987654321", 2, "-123.12"},
		{"-123.1234567890987654321", 3, "-123.123"},
		{"-123.1234567890987654321", 4, "-123.1235"},
		{"-123.1234567890987654321", 5, "-123.12346"},
		{"-123.1234567890987654321", 6, "-123.123457"},
		{"-123.1234567890987654321", 7, "-123.1234568"},
		{"-123.1234567890987654321", 8, "-123.12345679"},
		{"-123.1234567890987654321", 9, "-123.123456789"},
		{"-123.1234567890987654321", 10, "-123.1234567891"},
		{"-123.1234567890987654321", 11, "-123.12345678910"},
		{"-123.1234567890987654321", 12, "-123.123456789099"},
		{"-123.1234567890987654321", 13, "-123.1234567890988"},
		{"-123.1234567890987654321", 14, "-123.12345678909877"},
		{"-123.1234567890987654321", 15, "-123.123456789098765"},
		{"-123.1234567890987654321", 16, "-123.1234567890987654"},
		{"-123.1234567890987654321", 17, "-123.12345678909876543"},
		{"-123.1234567890987654321", 18, "-123.123456789098765432"},
		{"-123.1234567890987654321", 19, "-123.1234567890987654321"},
		{"-123.1234567890987654321", 20, "-123.1234567890987654321"},
		{"123.12354", 3, "123.124"},
		{"123.12454", 3, "123.125"},
		{"123.1235", 3, "123.123"},
		{"123.1245", 3, "123.124"},
		{"2.5", 0, "2"},
		{"1", 0, "1"},
		{"-123.12354", 3, "-123.124"},
		{"-123.12454", 3, "-123.125"},
		{"-123.1235", 3, "-123.123"},
		{"-123.1245", 3, "-123.124"},
		{"-2.5", 0, "-2"},
		{"-1", 0, "-1"},
		{"9999999999999999999.9999999999999999999", 3, "10000000000000000000.000"},
		{"-9999999999999999999.9999999999999999999", 3, "-10000000000000000000.000"},
		{"1.236", 2, "1.24"},
		{"1.235", 2, "1.23"},
		{"1.234", 2, "1.23"},
		{"-1.234", 2, "-1.23"},
		{"-1.235", 2, "-1.23"},
		{"-1.236", 2, "-1.24"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalRoundHalfTowardZero(%v)", tc), func(t *testing.T) {
			d := FromString(tc.i)
			s := d.RoundHalfTowardZero(tc.p).StringFixed()
			if s != tc.s {
				t.Errorf("RoundHalfTowardZero(%v, %v) = %v, want %v", tc.i, tc.p, s, tc.s)
			}
		})
	}
}

func TestDecimalRoundHalfAwayFromZero(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"Nan", 2, "NaN"},
		{"0", 0, "0"},
		{"1.12345", 4, "1.1235"},
		{"1.12335", 4, "1.1234"},
		{"1.5", 0, "2"},
		{"-1.5", 0, "-2"},
		{"123.456000", 0, "123"},
		{"123.456000", 1, "123.5"},
		{"123.456000", 2, "123.46"},
		{"123.456000", 3, "123.456"},
		{"123.456000", 4, "123.4560"},
		{"123.456000", 5, "123.45600"},
		{"123.456000", 6, "123.456000"},
		{"123.456000", 7, "123.456000"},
		{"-123.456000", 0, "-123"},
		{"-123.456000", 1, "-123.5"},
		{"-123.456000", 2, "-123.46"},
		{"-123.456000", 3, "-123.456"},
		{"-123.456000", 4, "-123.4560"},
		{"-123.456000", 5, "-123.45600"},
		{"-123.456000", 6, "-123.456000"},
		{"-123.456000", 7, "-123.456000"},
		{"123.1234567890987654321", 0, "123"},
		{"123.1234567890987654321", 1, "123.1"},
		{"123.1234567890987654321", 2, "123.12"},
		{"123.1234567890987654321", 3, "123.123"},
		{"123.1234567890987654321", 4, "123.1235"},
		{"123.1234567890987654321", 5, "123.12346"},
		{"123.1234567890987654321", 6, "123.123457"},
		{"123.1234567890987654321", 7, "123.1234568"},
		{"123.1234567890987654321", 8, "123.12345679"},
		{"123.1234567890987654321", 9, "123.123456789"},
		{"123.1234567890987654321", 10, "123.1234567891"},
		{"123.1234567890987654321", 11, "123.12345678910"},
		{"123.1234567890987654321", 12, "123.123456789099"},
		{"123.1234567890987654321", 13, "123.1234567890988"},
		{"123.1234567890987654321", 14, "123.12345678909877"},
		{"123.1234567890987654321", 15, "123.123456789098765"},
		{"123.1234567890987654321", 16, "123.1234567890987654"},
		{"123.1234567890987654321", 17, "123.12345678909876543"},
		{"123.1234567890987654321", 18, "123.123456789098765432"},
		{"123.1234567890987654321", 19, "123.1234567890987654321"},
		{"123.1234567890987654321", 20, "123.1234567890987654321"},
		{"-123.1234567890987654321", 0, "-123"},
		{"-123.1234567890987654321", 1, "-123.1"},
		{"-123.1234567890987654321", 2, "-123.12"},
		{"-123.1234567890987654321", 3, "-123.123"},
		{"-123.1234567890987654321", 4, "-123.1235"},
		{"-123.1234567890987654321", 5, "-123.12346"},
		{"-123.1234567890987654321", 6, "-123.123457"},
		{"-123.1234567890987654321", 7, "-123.1234568"},
		{"-123.1234567890987654321", 8, "-123.12345679"},
		{"-123.1234567890987654321", 9, "-123.123456789"},
		{"-123.1234567890987654321", 10, "-123.1234567891"},
		{"-123.1234567890987654321", 11, "-123.12345678910"},
		{"-123.1234567890987654321", 12, "-123.123456789099"},
		{"-123.1234567890987654321", 13, "-123.1234567890988"},
		{"-123.1234567890987654321", 14, "-123.12345678909877"},
		{"-123.1234567890987654321", 15, "-123.123456789098765"},
		{"-123.1234567890987654321", 16, "-123.1234567890987654"},
		{"-123.1234567890987654321", 17, "-123.12345678909876543"},
		{"-123.1234567890987654321", 18, "-123.123456789098765432"},
		{"-123.1234567890987654321", 19, "-123.1234567890987654321"},
		{"-123.1234567890987654321", 20, "-123.1234567890987654321"},
		{"123.12354", 3, "123.124"},
		{"123.12454", 3, "123.125"},
		{"123.1235", 3, "123.124"},
		{"123.1245", 3, "123.125"},
		{"2.5", 0, "3"},
		{"1", 0, "1"},
		{"-123.12354", 3, "-123.124"},
		{"-123.12454", 3, "-123.125"},
		{"-123.1235", 3, "-123.124"},
		{"-123.1245", 3, "-123.125"},
		{"-2.5", 0, "-3"},
		{"-1", 0, "-1"},
		{"9999999999999999999.9999999999999999999", 3, "10000000000000000000.000"},
		{"-9999999999999999999.9999999999999999999", 3, "-10000000000000000000.000"},
		{"1.236", 2, "1.24"},
		{"1.235", 2, "1.24"},
		{"1.234", 2, "1.23"},
		{"-1.234", 2, "-1.23"},
		{"-1.235", 2, "-1.24"},
		{"-1.236", 2, "-1.24"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalRoundHalfAwayFromZero(%v)", tc), func(t *testing.T) {
			d := FromString(tc.i)
			s := d.RoundHalfAwayFromZero(tc.p).StringFixed()
			if s != tc.s {
				t.Errorf("RoundHalfAwayFromZero(%v, %v) = %v, want %v", tc.i, tc.p, s, tc.s)
			}
		})
	}
}

func TestDecimalRoundBank(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"Nan", 2, "NaN"},
		{"0", 0, "0"},
		{"1.12345", 4, "1.1234"},
		{"1.12335", 4, "1.1234"},
		{"1.5", 0, "2"},
		{"-1.5", 0, "-2"},
		{"123.456000", 0, "123"},
		{"123.456000", 1, "123.5"},
		{"123.456000", 2, "123.46"},
		{"123.456000", 3, "123.456"},
		{"123.456000", 4, "123.4560"},
		{"123.456000", 5, "123.45600"},
		{"123.456000", 6, "123.456000"},
		{"123.456000", 7, "123.456000"},
		{"-123.456000", 0, "-123"},
		{"-123.456000", 1, "-123.5"},
		{"-123.456000", 2, "-123.46"},
		{"-123.456000", 3, "-123.456"},
		{"-123.456000", 4, "-123.4560"},
		{"-123.456000", 5, "-123.45600"},
		{"-123.456000", 6, "-123.456000"},
		{"-123.456000", 7, "-123.456000"},
		{"123.1234567890987654321", 0, "123"},
		{"123.1234567890987654321", 1, "123.1"},
		{"123.1234567890987654321", 2, "123.12"},
		{"123.1234567890987654321", 3, "123.123"},
		{"123.1234567890987654321", 4, "123.1235"},
		{"123.1234567890987654321", 5, "123.12346"},
		{"123.1234567890987654321", 6, "123.123457"},
		{"123.1234567890987654321", 7, "123.1234568"},
		{"123.1234567890987654321", 8, "123.12345679"},
		{"123.1234567890987654321", 9, "123.123456789"},
		{"123.1234567890987654321", 10, "123.1234567891"},
		{"123.1234567890987654321", 11, "123.12345678910"},
		{"123.1234567890987654321", 12, "123.123456789099"},
		{"123.1234567890987654321", 13, "123.1234567890988"},
		{"123.1234567890987654321", 14, "123.12345678909877"},
		{"123.1234567890987654321", 15, "123.123456789098765"},
		{"123.1234567890987654321", 16, "123.1234567890987654"},
		{"123.1234567890987654321", 17, "123.12345678909876543"},
		{"123.1234567890987654321", 18, "123.123456789098765432"},
		{"123.1234567890987654321", 19, "123.1234567890987654321"},
		{"123.1234567890987654321", 20, "123.1234567890987654321"},
		{"-123.1234567890987654321", 0, "-123"},
		{"-123.1234567890987654321", 1, "-123.1"},
		{"-123.1234567890987654321", 2, "-123.12"},
		{"-123.1234567890987654321", 3, "-123.123"},
		{"-123.1234567890987654321", 4, "-123.1235"},
		{"-123.1234567890987654321", 5, "-123.12346"},
		{"-123.1234567890987654321", 6, "-123.123457"},
		{"-123.1234567890987654321", 7, "-123.1234568"},
		{"-123.1234567890987654321", 8, "-123.12345679"},
		{"-123.1234567890987654321", 9, "-123.123456789"},
		{"-123.1234567890987654321", 10, "-123.1234567891"},
		{"-123.1234567890987654321", 11, "-123.12345678910"},
		{"-123.1234567890987654321", 12, "-123.123456789099"},
		{"-123.1234567890987654321", 13, "-123.1234567890988"},
		{"-123.1234567890987654321", 14, "-123.12345678909877"},
		{"-123.1234567890987654321", 15, "-123.123456789098765"},
		{"-123.1234567890987654321", 16, "-123.1234567890987654"},
		{"-123.1234567890987654321", 17, "-123.12345678909876543"},
		{"-123.1234567890987654321", 18, "-123.123456789098765432"},
		{"-123.1234567890987654321", 19, "-123.1234567890987654321"},
		{"-123.1234567890987654321", 20, "-123.1234567890987654321"},
		{"123.12354", 3, "123.124"},
		{"123.12454", 3, "123.125"},
		{"123.1235", 3, "123.124"},
		{"123.1245", 3, "123.124"},
		{"2.5", 0, "2"},
		{"1", 0, "1"},
		{"-123.12354", 3, "-123.124"},
		{"-123.12454", 3, "-123.125"},
		{"-123.1235", 3, "-123.124"},
		{"-123.1245", 3, "-123.124"},
		{"-2.5", 0, "-2"},
		{"-1", 0, "-1"},
		{"9999999999999999999.9999999999999999999", 3, "10000000000000000000.000"},
		{"-9999999999999999999.9999999999999999999", 3, "-10000000000000000000.000"},
		{"2.121", 2, "2.12"},
		{"2.125", 2, "2.12"},
		{"2.135", 2, "2.14"},
		{"2.1351", 2, "2.14"},
		{"2.127", 2, "2.13"},
		{"1.236", 2, "1.24"},
		{"1.235", 2, "1.24"},
		{"1.234", 2, "1.23"},
		{"-1.234", 2, "-1.23"},
		{"-1.235", 2, "-1.24"},
		{"-1.236", 2, "-1.24"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalRoundBank(%v)", tc), func(t *testing.T) {
			d := FromString(tc.i)
			s := d.RoundBank(tc.p).StringFixed()
			if s != tc.s {
				t.Errorf("RoundBank(%v, %v) = %v, want %v", tc.i, tc.p, s, tc.s)
			}
		})
	}
}

func TestDecimalTrunc(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		p uint8
		s string
	}

	testCases := [...]testCase{
		{"Nan", 2, "NaN"},
		{"0", 0, "0"},
		{"0", 1, "0"},
		{"1.12345", 4, "1.1234"},
		{"1.12335", 4, "1.1233"},
		{"1234567890123456789012345678912345678.5", 0, "1234567890123456789012345678912345678"},
		{"-1234567890123456789012345678912345678.5", 0, "-1234567890123456789012345678912345678"},
		{"9999999999999999999.9999999999999999999", 0, "9999999999999999999"},
		{"-9999999999999999999.9999999999999999999", 0, "-9999999999999999999"},
		{"123.456000", 0, "123"},
		{"123.456000", 1, "123.4"},
		{"123.456000", 2, "123.45"},
		{"123.456000", 3, "123.456"},
		{"123.456000", 4, "123.4560"},
		{"123.456000", 5, "123.45600"},
		{"123.456000", 6, "123.456000"},
		{"123.456000", 7, "123.456000"},
		{"-123.456000", 0, "-123"},
		{"-123.456000", 1, "-123.4"},
		{"-123.456000", 2, "-123.45"},
		{"-123.456000", 3, "-123.456"},
		{"-123.456000", 4, "-123.4560"},
		{"-123.456000", 5, "-123.45600"},
		{"-123.456000", 6, "-123.456000"},
		{"-123.456000", 7, "-123.456000"},
		{"123.1234567890987654321", 0, "123"},
		{"123.1234567890987654321", 1, "123.1"},
		{"123.1234567890987654321", 2, "123.12"},
		{"123.1234567890987654321", 3, "123.123"},
		{"123.1234567890987654321", 4, "123.1234"},
		{"123.1234567890987654321", 5, "123.12345"},
		{"123.1234567890987654321", 6, "123.123456"},
		{"123.1234567890987654321", 7, "123.1234567"},
		{"123.1234567890987654321", 8, "123.12345678"},
		{"123.1234567890987654321", 9, "123.123456789"},
		{"123.1234567890987654321", 10, "123.1234567890"},
		{"123.1234567890987654321", 11, "123.12345678909"},
		{"123.1234567890987654321", 12, "123.123456789098"},
		{"123.1234567890987654321", 13, "123.1234567890987"},
		{"123.1234567890987654321", 14, "123.12345678909876"},
		{"123.1234567890987654321", 15, "123.123456789098765"},
		{"123.1234567890987654321", 16, "123.1234567890987654"},
		{"123.1234567890987654321", 17, "123.12345678909876543"},
		{"123.1234567890987654321", 18, "123.123456789098765432"},
		{"123.1234567890987654321", 19, "123.1234567890987654321"},
		{"123.1234567890987654321", 20, "123.1234567890987654321"},
		{"-123.1234567890987654321", 0, "-123"},
		{"-123.1234567890987654321", 1, "-123.1"},
		{"-123.1234567890987654321", 2, "-123.12"},
		{"-123.1234567890987654321", 3, "-123.123"},
		{"-123.1234567890987654321", 4, "-123.1234"},
		{"-123.1234567890987654321", 5, "-123.12345"},
		{"-123.1234567890987654321", 6, "-123.123456"},
		{"-123.1234567890987654321", 7, "-123.1234567"},
		{"-123.1234567890987654321", 8, "-123.12345678"},
		{"-123.1234567890987654321", 9, "-123.123456789"},
		{"-123.1234567890987654321", 10, "-123.1234567890"},
		{"-123.1234567890987654321", 11, "-123.12345678909"},
		{"-123.1234567890987654321", 12, "-123.123456789098"},
		{"-123.1234567890987654321", 13, "-123.1234567890987"},
		{"-123.1234567890987654321", 14, "-123.12345678909876"},
		{"-123.1234567890987654321", 15, "-123.123456789098765"},
		{"-123.1234567890987654321", 16, "-123.1234567890987654"},
		{"-123.1234567890987654321", 17, "-123.12345678909876543"},
		{"-123.1234567890987654321", 18, "-123.123456789098765432"},
		{"-123.1234567890987654321", 19, "-123.1234567890987654321"},
		{"-123.1234567890987654321", 20, "-123.1234567890987654321"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalTrunc(%v)", tc), func(t *testing.T) {
			d := FromString(tc.i)
			s := d.Trunc(tc.p).StringFixed()
			if s != tc.s {
				t.Errorf("Trunc(%v, %v) = %v, want %v", tc.i, tc.p, s, tc.s)
			}
		})
	}
}

func TestDecimalParseStringHLE(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		h uint64
		l uint64
		e uint8
	}

	testCases := [...]testCase{
		{"", 0, 0, 0},
		{"0", 0, 0, 0},
		{"1", 0, 1, 0},
		{"10", 0, 10, 0},
		{"1.0", 0, 10, 1},
		{"1.00", 0, 100, 2},
		{"1.000", 0, 1000, 3},
		{"18446744073709551615", 0, 18446744073709551615, 0},
		{"18446744073709551616", 1, 0, 0},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalParseStringHLE(%s)", tc.i), func(t *testing.T) {
			d := FromString(tc.i)
			if d.IsNaN() {
				t.Errorf("expected no error, got: %v", d.ErrorDetails())
			}
			u := d.Coefficient()
			e := d.Exponent()
			err := d.ErrorDetails()
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if u.Hi != tc.h || u.Lo != tc.l || e != tc.e {
				t.Errorf("expected %d %d %d, got: %d %d %d", tc.h, tc.l, tc.e, u.Hi, u.Lo, e)
			}

			d = FromSafeString(tc.i)
			if d.IsNaN() {
				t.Errorf("expected no error, got: %v", d.ErrorDetails())
			}
			u = d.Coefficient()
			e = d.Exponent()
			err = d.ErrorDetails()
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if u.Hi != tc.h || u.Lo != tc.l || e != tc.e {
				t.Errorf("expected %d %d %d, got: %d %d %d", tc.h, tc.l, tc.e, u.Hi, u.Lo, e)
			}
		})
	}
}

func TestDecimalConvString(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		s string
		e string
	}

	testCases := [...]testCase{
		{"", "0", ""},
		{"0", "0", ""},
		{"1", "1", ""},
		{"10", "10", ""},
		{"100", "100", ""},
		{"1000", "1000", ""},
		{"1000000", "1000000", ""},
		{"1000000000", "1000000000", ""},
		{"1000000000000", "1000000000000", ""},
		{"1000000000000000", "1000000000000000", ""},
		{"1000000000000000000", "1000000000000000000", ""},
		{"1000000000000000000000", "1000000000000000000000", ""},
		{"1000000000000000000000000", "1000000000000000000000000", ""},
		{"-1", "-1", ""},
		{"-10", "-10", ""},
		{"-100", "-100", ""},
		{"-1000", "-1000", ""},
		{"-1000000", "-1000000", ""},
		{"-1000000000", "-1000000000", ""},
		{"-1000000000000", "-1000000000000", ""},
		{"-1000000000000000", "-1000000000000000", ""},
		{"-1000000000000000000", "-1000000000000000000", ""},
		{"-1000000000000000000000", "-1000000000000000000000", ""},
		{"-1000000000000000000000000", "-1000000000000000000000000", ""},
		{".1", "0.1", ""},
		{".01", "0.01", ""},
		{"0.1", "0.1", ""},
		{"0.01", "0.01", ""},
		{"0.001", "0.001", ""},
		{"0.0000001", "0.0000001", ""},
		{"0.0000000001", "0.0000000001", ""},
		{"0.0000000000001", "0.0000000000001", ""},
		{"0.0000000000000001", "0.0000000000000001", ""},
		{"0.0000000000000000001", "0.0000000000000000001", ""},
		{"-0.1", "-0.1", ""},
		{"-0.01", "-0.01", ""},
		{"-0.001", "-0.001", ""},
		{"-0.0000001", "-0.0000001", ""},
		{"-0.0000000001", "-0.0000000001", ""},
		{"-0.0000000000001", "-0.0000000000001", ""},
		{"-0.0000000000000001", "-0.0000000000000001", ""},
		{"-0.0000000000000000001", "-0.0000000000000000001", ""},
		{"NaN", "NaN", "invalid format"},
		{"1.2.3", "NaN", "invalid format"},
		{"-", "NaN", "invalid format"},
		{"-+", "NaN", "invalid format"},
		{"+", "NaN", "invalid format"},
		{".", "NaN", "invalid format"},
		{".+", "NaN", "invalid format"},
		{"..", "NaN", "invalid format"},
		{"+.", "NaN", "invalid format"},
		{"--123", "NaN", "invalid format"},
		{"+.123", "0.123", ""},  // not worth the effort to detect this as invalid
		{"-.123", "-0.123", ""}, // not worth the effort to detect this as invalid
		{"123.", "123", ""},     // not worth the effort to detect this as invalid
		{"12345678901234567890123456789012345678901234567890", "NaN", "overflow"},
		{".123", "0.123", ""},
		{".123456789012345678901234567890", "NaN", "overflow"},
		{"0.00", "0", ""},
		{"1.00", "1", ""},
		{"1.10", "1.1", ""},
		{"1.01", "1.01", ""},
		{"1.001", "1.001", ""},
		{"1.0000001", "1.0000001", ""},
		{"1.1000000", "1.1", ""},
		{"0.123", "0.123", ""},
		{"0.0000123456", "0.0000123456", ""},
		{"-0.0000123456", "-0.0000123456", ""},
		{"0.0101010101010101", "0.0101010101010101", ""},
		{"123.456000", "123.456", ""},
		{"-12345678912345678901.1234567890123456789", "-12345678912345678901.1234567890123456789", ""},
		{"123.0000", "123", ""},
		{"-0.123", "-0.123", ""},
		{"0.00000", "0", ""},
		{"-0", "0", ""},
		{"-0.00000", "0", ""},
		{"-123.0000", "-123", ""},
		{"0.9999999999999999999", "0.9999999999999999999", ""},
		{"-0.9999999999999999999", "-0.9999999999999999999", ""},
		{"123.456", "123.456", ""},
		{"123.456789012345678901", "123.456789012345678901", ""},
		{"123456789.123456789", "123456789.123456789", ""},
		{"-123.456", "-123.456", ""},
		{"-123.456789012345678901", "-123.456789012345678901", ""},
		{"-123456789.123456789", "-123456789.123456789", ""},
		{"-123456789123456789.123456789123456789", "-123456789123456789.123456789123456789", ""},
		{"1234567891234567890.0123456879123456789", "1234567891234567890.0123456879123456789", ""},
		{"9999999999999999999.9999999999999999999", "9999999999999999999.9999999999999999999", ""},
		{"-9999999999999999999.9999999999999999999", "-9999999999999999999.9999999999999999999", ""},
		{"123456.0000000000000000001", "123456.0000000000000000001", ""},
		{"-123456.0000000000000000001", "-123456.0000000000000000001", ""},
		{"+123456.123456", "123456.123456", ""},
		{"+123.123", "123.123", ""},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalConvString(%s)", tc.i), func(t *testing.T) {
			d := FromString(tc.i)
			if tc.e != "" && !d.IsNaN() {
				t.Errorf("expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && d.IsNaN() {
				t.Errorf("expected no error, got: %v", d.ErrorDetails())
			}
			s := d.String()
			if s != tc.s {
				t.Errorf("expected '%s', got: %s", tc.s, s)
			}

			if tc.e == "" {
				d = FromSafeString(tc.i)
				if d.IsNaN() {
					t.Errorf("expected no error, got: %v", d.ErrorDetails())
				}
				s = d.String()
				if s != tc.s {
					t.Errorf("expected '%s', got: %s", tc.s, s)
				}
			}
		})
	}
}

func TestDecimalToStringFixed(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i uint64
		e uint8
		s string
	}

	testCases := [...]testCase{
		{0, 0, "0"},
		{0, 1, "0.0"},
		{0, 2, "0.00"},
		{0, 3, "0.000"},
		{1, 0, "1"},
		{1, 1, "0.1"},
		{1, 2, "0.01"},
		{1, 3, "0.001"},
		{1, 6, "0.000001"},
		{10, 6, "0.000010"},
		{100, 6, "0.000100"},
		{1000, 6, "0.001000"},
		{10000, 6, "0.010000"},
		{100000, 6, "0.100000"},
		{1000000, 6, "1.000000"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalToStringFixed(%v)", tc), func(t *testing.T) {
			d := New(uint128.FromUint64(tc.i), tc.e, false)
			s := d.StringFixed()
			if s != tc.s {
				t.Errorf("expected '%s', got: %s", tc.s, s)
			}
		})
	}
}

func TestDecimalToStringFixed2(t *testing.T) {
	SetDefaultPrecision(19)

	type testCase struct {
		i string
		s string
	}

	testCases := [...]testCase{
		{"0", "0"},
		{"0.0", "0.0"},
		{"0.00", "0.00"},
		{"1", "1"},
		{"0.1", "0.1"},
		{"0.01", "0.01"},
		{"0.001", "0.001"},
		{"1.0", "1.0"},
		{"1.00", "1.00"},
		{"1.000", "1.000"},
		{"1.000000", "1.000000"},
		{"1.000001", "1.000001"},
		{"1.000010", "1.000010"},
		{"1.000100", "1.000100"},
		{"1.001000", "1.001000"},
		{"1.010000", "1.010000"},
		{"1.100000", "1.100000"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalToStringFixed2(%v)", tc), func(t *testing.T) {
			d := FromString(tc.i)
			s := d.StringFixed()
			if s != tc.s {
				t.Errorf("expected '%s', got: %s", tc.s, s)
			}
		})
	}
}

func TestDecimalJson(t *testing.T) {
	SetDefaultPrecision(19)

	type testStruct struct {
		D Dec128 `json:"d"`
	}

	type testCase struct {
		t testStruct
		s string
	}

	tests := [...]testCase{
		{testStruct{Zero}, `{"d":"0"}`},
		{testStruct{FromString("1")}, `{"d":"1"}`},
		{testStruct{FromString("1.01")}, `{"d":"1.01"}`},
		{testStruct{FromString("1.000001")}, `{"d":"1.000001"}`},
		{testStruct{FromString("12345678901234567890.123456789")}, `{"d":"12345678901234567890.123456789"}`},
		{testStruct{FromString("-1")}, `{"d":"-1"}`},
		{testStruct{FromString("-1.01")}, `{"d":"-1.01"}`},
		{testStruct{FromString("-1.000001")}, `{"d":"-1.000001"}`},
		{testStruct{FromString("-12345678901234567890.123456789")}, `{"d":"-12345678901234567890.123456789"}`},
	}

	for _, test := range tests {
		s, err := json.Marshal(test.t)
		if err != nil {
			t.Errorf("error marshalling %v: %v", test, err)
		}
		if string(s) != test.s {
			t.Errorf("expected '%v', got '%v'", test.s, string(s))
		}
		var q testStruct
		if err := json.Unmarshal(s, &q); err != nil {
			t.Errorf("error unmarshaling %v: %v", test, err)
		}
		if !q.D.Equal(test.t.D) {
			t.Errorf("expected '%v', got '%v'", test.t.D, q.D)
		}
	}

	a := FromString("NaN")
	bs, err := a.MarshalJSON()
	if err != nil {
		t.Errorf("unexpected error marshalling NaN: %v", err)
	}
	if string(bs) != `"NaN"` {
		t.Errorf(`expected "NaN", got %s`, string(bs))
	}

	if err := a.UnmarshalJSON(nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if a.String() != "0" {
		t.Errorf("expected '0', got '%s'", a.String())
	}
}

type GobTestStruct struct {
	A Dec128
	B Dec128
	C []Dec128
}

func TestDecimalBinary(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		a := Decimal1
		_, err := a.EncodeBinary(nil)
		if err == nil {
			t.Errorf("expected error for nil slice, got nil")
		}
		_, err = a.DecodeBinary(nil)
		if err == nil {
			t.Errorf("expected error for nil slice, got nil")
		}

		a = FromString("NaN")
		_, err = a.EncodeBinary(nil)
		if err == nil {
			t.Errorf("expected error for nil slice, got nil")
		}
	})

	t.Run("zero", func(t *testing.T) {
		a := Zero
		var b Dec128
		var bs []byte

		bs = make([]byte, a.BinarySize())
		_, err := a.EncodeBinary(bs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		_, err = b.DecodeBinary(bs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !a.Equal(b) {
			t.Errorf("expected %s, got %s", a.String(), b.String())
		}
	})

	t.Run("nan", func(t *testing.T) {
		a := FromString("NaN")
		var b Dec128
		var bs []byte

		bs = make([]byte, a.BinarySize())
		_, err := a.EncodeBinary(bs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		_, err = b.DecodeBinary(bs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !a.Equal(b) {
			t.Errorf("expected %s, got %s", a.String(), b.String())
		}
	})

	t.Run("small decimal", func(t *testing.T) {
		a := FromString("123.456")
		var b Dec128
		var bs []byte

		bs = make([]byte, a.BinarySize())
		_, err := a.EncodeBinary(bs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		_, err = b.DecodeBinary(bs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !a.Equal(b) {
			t.Errorf("expected %s, got %s", a.String(), b.String())
		}
	})

	t.Run("big decimal", func(t *testing.T) {
		a := FromString("123456789012345678901234567890.123456")
		var b Dec128
		var bs []byte

		bs = make([]byte, a.BinarySize())
		_, err := a.EncodeBinary(bs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		_, err = b.DecodeBinary(bs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !a.Equal(b) {
			t.Errorf("expected %s, got %s", a.String(), b.String())
		}
	})

	t.Run("small int", func(t *testing.T) {
		a := FromString("123")
		var b Dec128
		var bs []byte

		bs = make([]byte, a.BinarySize())
		_, err := a.EncodeBinary(bs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		_, err = b.DecodeBinary(bs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !a.Equal(b) {
			t.Errorf("expected %s, got %s", a.String(), b.String())
		}
	})

	t.Run("big int", func(t *testing.T) {
		a := FromString("123456789012345678901234567890")
		var b Dec128
		var bs []byte

		bs = make([]byte, a.BinarySize())
		_, err := a.EncodeBinary(bs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		_, err = b.DecodeBinary(bs)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !a.Equal(b) {
			t.Errorf("expected %s, got %s", a.String(), b.String())
		}
	})
	t.Run("single", func(t *testing.T) {
		type test struct {
			d string
			s int
		}
		tests := []test{
			{"not a number", 1},
			{"0", 1},
			{"1", 9},
			{"-1", 9},
			{"1234567890", 9},
			{"-1234567890", 9},
			{"123456789012345678901234567890", 17},
			{"-123456789012345678901234567890", 17},
			{"0.1234", 10},
			{"-0.1234", 10},
			{"1234567890.1234567890", 10},
			{"-1234567890.1234567890", 10},
			{"123456789012345678901234567890.123", 18},
			{"-123456789012345678901234567890.123", 18},
		}

		for _, tc := range tests {
			t.Run(tc.d, func(t *testing.T) {
				d := FromString(tc.d)
				b, err := d.MarshalBinary()
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(b) != tc.s {
					t.Errorf("expected %d bytes, got %d", tc.s, len(b))
				}

				var d2 Dec128
				if err := d2.UnmarshalBinary(b); err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if !d.Equal(d2) {
					t.Errorf("expected %s, got %s", d.String(), d2.String())
				}
			})
		}
	})

	t.Run("struct", func(t *testing.T) {
		var tc GobTestStruct
		tc.A = FromString("1234567890.1234567890")
		tc.B = FromString("123")
		tc.C = []Dec128{
			FromString("0.123456"),
			FromString("12345678901234567890.1234567890"),
			FromString("0.123456789012345678901234567890"),
			FromString("123456789012345678901234567890.1234567890"),
		}

		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		dec := gob.NewDecoder(&buf)

		if err := enc.Encode(tc); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		bs := buf.Bytes()
		if len(bs) != 161 {
			t.Errorf("expected 161 bytes, got %d", len(bs))
		}

		var tc2 GobTestStruct
		if err := dec.Decode(&tc2); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !tc.A.Equal(tc2.A) {
			t.Errorf("expected %s, got %s", tc.A.String(), tc2.A.String())
		}
		if !tc.B.Equal(tc2.B) {
			t.Errorf("expected %s, got %s", tc.B.String(), tc2.B.String())
		}
		if len(tc.C) != len(tc2.C) {
			t.Errorf("expected %d elements, got %d", len(tc.C), len(tc2.C))
		}
		if !tc.C[0].Equal(tc2.C[0]) {
			t.Errorf("expected %s, got %s", tc.C[0].String(), tc2.C[0].String())
		}
		if !tc.C[1].Equal(tc2.C[1]) {
			t.Errorf("expected %s, got %s", tc.C[1].String(), tc2.C[1].String())
		}
		if !tc.C[2].Equal(tc2.C[2]) {
			t.Errorf("expected %s, got %s", tc.C[2].String(), tc2.C[2].String())
		}
		if !tc.C[3].Equal(tc2.C[3]) {
			t.Errorf("expected %s, got %s", tc.C[3].String(), tc2.C[3].String())
		}
	})

	t.Run("append", func(t *testing.T) {
		a := FromString("1.23")
		bs, err := a.AppendBinary(make([]byte, 0))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		var b Dec128
		if err := b.UnmarshalBinary(bs); err != nil {
			t.Errorf("unexpected error unmarshaling binary: %v", err)
		}
		if b.String() != "1.23" {
			t.Errorf("expected '1.23', got '%s'", b.String())
		}
	})

	t.Run("rw", func(t *testing.T) {
		var b bytes.Buffer
		if err := FromString("1.23").WriteBinary(&b); err != nil {
			t.Errorf("unexpected error writing binary: %v", err)
		}
		var a Dec128
		if err := a.ReadBinary(&b); err != nil {
			t.Errorf("unexpected error reading binary: %v", err)
		}
		if a.String() != "1.23" {
			t.Errorf("expected '1.23', got '%s'", a.String())
		}
	})

	t.Run("marshal", func(t *testing.T) {
		var a Dec128
		if err := a.UnmarshalBinary([]byte("aaaa")); err == nil {
			t.Errorf("expected error unmarshaling binary, got nil")
		}
	})
}

func TestDecimalMarshalText(t *testing.T) {
	a := FromString("NaN")
	bs, err := a.MarshalText()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(bs) != "NaN" {
		t.Errorf("expected 'NaN', got '%s'", string(bs))
	}

	a = Decimal0
	bs, err = a.MarshalText()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(bs) != "0" {
		t.Errorf("expected '0', got '%s'", string(bs))
	}

	a = Decimal1000
	bs, err = a.MarshalText()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(bs) != "1000" {
		t.Errorf("expected '1000', got '%s'", string(bs))
	}

	var b Dec128
	if err := b.UnmarshalText(bs); err != nil {
		t.Errorf("unexpected error unmarshaling text: %v", err)
	}
	if b.String() != "1000" {
		t.Errorf("expected '1000', got '%s'", b.String())
	}

	if err := b.UnmarshalText(nil); err != nil {
		t.Errorf("unexpected error unmarshaling text: %v", err)
	}
	if b.String() != "0" {
		t.Errorf("expected '0', got '%s'", b.String())
	}
}

func TestDecimalFloat(t *testing.T) {
	a := FromFloat64(1.2)
	b, err := a.InexactFloat64()
	if err != nil {
		t.Errorf("unexpected error converting to float64: %v", err)
	}
	if math.Abs(b-1.2) > 0.0000000001 {
		t.Errorf("expected 1.2, got %f", b)
	}

	a = FromString("NaN")
	if _, err := a.InexactFloat64(); err == nil {
		t.Errorf("expected error for NaN, got nil")
	}
}

func TestDecimalSetDefaultPrecision(t *testing.T) {
	t.Run("panic", func(t *testing.T) {
		var f bool
		defer func() {
			if r := recover(); r != nil {
				f = true
			}
		}()
		SetDefaultPrecision(19)
		if f {
			t.Errorf("expected no panic, got one")
		}
		SetDefaultPrecision(20)
		if !f {
			t.Errorf("expected panic, got none")
		}
	})
}

func TestDecimalCopy(t *testing.T) {
	a := FromString("1")
	b := a.Copy()
	if !a.Equal(b) {
		t.Errorf("expected %s, got %s", a.String(), b.String())
	}
	if &a == &b {
		t.Errorf("expected different pointers, got same: %p", &b)
	}
}

func TestDecimalScan(t *testing.T) {
	var a Dec128

	if err := a.Scan("NaN"); err == nil {
		t.Errorf("expected error for NaN, got nil")
	}

	if err := a.Scan("123.456"); err != nil {
		t.Errorf("unexpected error scanning '123.456': %v", err)
	}
	if a.String() != "123.456" {
		t.Errorf("expected '123.456', got '%s'", a.String())
	}

	if err := a.Scan(int(123)); err != nil {
		t.Errorf("unexpected error scanning '123': %v", err)
	}
	if a.String() != "123" {
		t.Errorf("expected '123', got '%s'", a.String())
	}

	if err := a.Scan(int64(123)); err != nil {
		t.Errorf("unexpected error scanning '123': %v", err)
	}
	if a.String() != "123" {
		t.Errorf("expected '123', got '%s'", a.String())
	}

	if err := a.Scan(nil); err != nil {
		t.Errorf("unexpected error scanning '123': %v", err)
	}
	if a.String() != "0" {
		t.Errorf("expected '0', got '%s'", a.String())
	}

	if err := a.Scan(true); err == nil {
		t.Errorf("expected error for boolean, got nil")
	}
}

func TestDecimalValue(t *testing.T) {
	v, err := Decimal1.Value()
	if err != nil {
		t.Errorf("unexpected error getting value: %v", err)
	}
	if v.(string) != "1" {
		t.Errorf("expected '1', got '%s'", v.(string))
	}
}

func TestDecimalDecode(t *testing.T) {
	u := uint128.FromUint64(123)
	a := DecodeFromUint128(u, 2)
	if a.String() != "1.23" {
		t.Errorf("expected '1.23', got '%s'", a.String())
	}

	a = DecodeFromInt64(123, 2)
	if a.String() != "1.23" {
		t.Errorf("expected '1.23', got '%s'", a.String())
	}

	a = DecodeFromUint64(123, 2)
	if a.String() != "1.23" {
		t.Errorf("expected '1.23', got '%s'", a.String())
	}
}

func TestDecimalTo(t *testing.T) {
	a := FromString("NaN")
	if _, err := a.EncodeToInt64(1); err == nil {
		t.Errorf("expected error for NaN, got nil")
	}
	if _, err := a.EncodeToUint64(1); err == nil {
		t.Errorf("expected error for NaN, got nil")
	}
	if _, err := a.EncodeToUint128(1); err == nil {
		t.Errorf("expected error for NaN, got nil")
	}

	a = FromString("123456789012345678901234567890")
	if a.String() != "123456789012345678901234567890" {
		t.Errorf("expected '123456789012345678901234567890', got '%s'", a.String())
	}
	if _, err := a.EncodeToInt64(1); err == nil {
		t.Errorf("expected error for overflow, got nil")
	}
	if _, err := a.EncodeToUint64(1); err == nil {
		t.Errorf("expected error for overflow, got nil")
	}

	a = FromString("-123456789012345678901234567890")
	if a.String() != "-123456789012345678901234567890" {
		t.Errorf("expected '-123456789012345678901234567890', got '%s'", a.String())
	}
	if _, err := a.EncodeToInt64(1); err == nil {
		t.Errorf("expected error for overflow, got nil")
	}
	if _, err := a.EncodeToUint64(1); err == nil {
		t.Errorf("expected error for overflow, got nil")
	}
	if _, err := a.EncodeToUint128(1); err == nil {
		t.Errorf("expected error for negative, got nil")
	}

	a = FromString("-9223372036854775809")
	if _, err := a.EncodeToInt64(0); err == nil {
		t.Errorf("expected error, got nil")
	}

	a = FromString("9223372036854775808")
	if _, err := a.EncodeToUint64(0); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if _, err := a.EncodeToInt64(0); err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestDecimalSymmetry(t *testing.T) {
	SetDefaultPrecision(6)

	var a, b, c Dec128

	i, _ := FromInt(10).AddInt(3).SubInt(5).Int()
	if i != 8 {
		t.Errorf("expected 8, got %d", i)
	}

	i, _ = FromInt(10).MulInt(4).DivInt(2).Int()
	if i != 20 {
		t.Errorf("expected 20, got %d", i)
	}

	tcs := [...]string{
		"0",
		"1",
		"0.1",
		"0.01",
		"0.1000",
		"0.1000000000000000000",
	}

	for _, e := range tcs {
		a = FromString(e)
		if a.IsNaN() {
			t.Errorf("expected no error, got: %v", a.ErrorDetails())
		}

		t.Run(fmt.Sprintf("add-sub %s", e), func(t *testing.T) {
			c = a.Add(a).Sub(a)
			if c.String() != a.String() {
				t.Errorf("expected %s, got %s", a.String(), c.String())
			}

			c = a.Sub(a).Add(a)
			if c.String() != a.String() {
				t.Errorf("expected %s, got %s", a.String(), c.String())
			}

			for _, i := range []string{"1", "5", "10", "100", "1000", "1000000000", "0.1000000", "0.1000000000000000000"} {
				b = FromString(i)
				c = a.Add(b).Sub(b)
				if c.String() != a.String() {
					t.Errorf("expected %s, got %s (b = %s)", a.String(), c.String(), b.String())
				}

				c = a.Sub(b).Add(b)
				if c.String() != a.String() {
					t.Errorf("expected %s, got %s (b = %s)", a.String(), c.String(), b.String())
				}
			}
		})

		t.Run(fmt.Sprintf("mul-div %s", e), func(t *testing.T) {
			for _, i := range []string{"1", "2", "3", "10", "100", "1000", "0.1000000", "0.1000000000000000000"} {
				b = FromString(i)
				c = a.Mul(b).Div(b)
				if c.String() != a.String() {
					t.Errorf("expected %s, got %s (b = %s)", a.String(), c.String(), b.String())
				}
			}
		})

		t.Run(fmt.Sprintf("div-mul %s", e), func(t *testing.T) {
			for _, i := range []string{"1", "2", "4", "5", "8", "10", "100", "1000", "0.1000000", "0.1000000000000000000"} {
				b = FromString(i)
				c = a.Div(b).Mul(b)
				if c.String() != a.String() {
					t.Errorf("expected %s, got %s (b = %s)", a.String(), c.String(), b.String())
				}
			}
		})

		c = a.PowInt(2).Sqrt()
		if c.String() != a.String() {
			t.Errorf("expected %s, got %s", a.String(), c.String())
		}
	}
}
