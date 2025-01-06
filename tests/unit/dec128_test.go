package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/uint128"
)

func TestDecimalParseStringHLE(t *testing.T) {
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
			d := dec128.FromString(tc.i)
			if d.IsNaN() {
				t.Errorf("Expected no error, got: %v", d.ErrorDetails())
			}
			u, e, err := d.Uint128()
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if u.Hi != tc.h || u.Lo != tc.l || e != tc.e {
				t.Errorf("Expected %d %d %d, got: %d %d %d", tc.h, tc.l, tc.e, u.Hi, u.Lo, e)
			}
		})
	}
}

func TestDecimalConvString(t *testing.T) {
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
			d := dec128.FromString(tc.i)
			if tc.e != "" && !d.IsNaN() {
				t.Errorf("Expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && d.IsNaN() {
				t.Errorf("Expected no error, got: %v", d.ErrorDetails())
			}
			s := d.String()
			if s != tc.s {
				t.Errorf("Expected '%s', got: %s", tc.s, s)
			}
		})
	}
}

func TestDecimalBasics(t *testing.T) {
	var d dec128.Dec128

	d = dec128.FromString("NaN")
	if !d.IsNaN() {
		t.Errorf("Expected NaN, got: %s", d.String())
	}
	if d.IsZero() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if d.IsNeg() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if d.IsPos() {
		t.Errorf("Expected false, got: %s", d.String())
	}

	d = dec128.FromString("0")
	if !d.IsZero() {
		t.Errorf("Expected zero, got: %s", d.String())
	}
	if d.IsNeg() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if d.IsPos() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if d.IsNaN() {
		t.Errorf("Expected false, got: %s", d.String())
	}

	d = dec128.FromString("1")
	if d.IsZero() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if d.IsNeg() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if !d.IsPos() {
		t.Errorf("Expected true, got: %s", d.String())
	}
	if d.IsNaN() {
		t.Errorf("Expected false, got: %s", d.String())
	}

	d = dec128.FromString("-1")
	if d.IsZero() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if !d.IsNeg() {
		t.Errorf("Expected true, got: %s", d.String())
	}
	if d.IsPos() {
		t.Errorf("Expected false, got: %s", d.String())
	}
	if d.IsNaN() {
		t.Errorf("Expected false, got: %s", d.String())
	}
}

func TestDecimalFromUint64(t *testing.T) {
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
			d := dec128.FromUint64(tc.i, tc.p)
			s := d.String()
			if s != tc.s {
				t.Errorf("Expected '%s', got: %s", tc.s, s)
			}
		})
	}
}

func TestDecimalConvToUint64(t *testing.T) {
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
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalConvToUint64(%s)", tc.i), func(t *testing.T) {
			d := dec128.FromString(tc.i)
			u, p, err := d.Uint64()
			if tc.e != "" && err == nil {
				t.Errorf("Expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if u != tc.u {
				t.Errorf("Expected %d, got: %d", tc.u, u)
			}
			if p != tc.p {
				t.Errorf("Expected %d, got: %d", tc.p, p)
			}
		})
	}
}

func TestDecimalConvUint128(t *testing.T) {
	d := dec128.FromString("NaN")
	_, _, err := d.Uint128()
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	d = dec128.FromString("-1")
	_, _, err = d.Uint128()
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	d = dec128.FromString("340282366920938463463374607431768211456")
	_, _, err = d.Uint128()
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	testCases := [...]string{"0", "1", "1234567890", "1234567890.123456789", "340282366920938463463374607431768211455", "12345678901234567890.123456789"}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalConvUint128(%s)", tc), func(t *testing.T) {
			d := dec128.FromString(tc)
			if d.IsNaN() {
				t.Errorf("Expected no error, got: %v", d.ErrorDetails())
			}
			u, p, err := d.Uint128()
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			d2 := dec128.FromUint128(u, p)
			if !d.Equal(d2) {
				t.Errorf("Expected %s, got: %s", d.String(), d2.String())
			}
			s := d2.String()
			if s != tc {
				t.Errorf("Expected '%s', got: %s", tc, s)
			}
		})
	}
}

func TestDecimalEqual(t *testing.T) {
	var a, b dec128.Dec128

	a = dec128.FromString("NaN")
	b = dec128.FromString("NaN")
	if !a.Equal(b) {
		t.Errorf("Expected true, got false")
	}

	a = dec128.FromString("0")
	b = dec128.FromString("NaN")
	if a.Equal(b) {
		t.Errorf("Expected false, got true")
	}

	a = dec128.FromString("0")
	b = dec128.FromString("0")
	if !a.Equal(b) {
		t.Errorf("Expected true, got false")
	}

	a = dec128.FromString("1")
	b = dec128.FromString("-1")
	if a.Equal(b) {
		t.Errorf("Expected false, got true")
	}

	a = dec128.FromUint64(1000, 1)
	b = dec128.FromUint64(10000, 2)
	if !a.Equal(b) {
		t.Errorf("Expected true, got false")
	}

	a = dec128.FromUint64(123456, 3)
	b = dec128.FromUint64(123456000, 6)
	if !a.Equal(b) {
		t.Errorf("Expected true, got false")
	}

	a = dec128.FromString("123.456")
	b = dec128.FromString("123.4560000")
	if !a.Equal(b) {
		t.Errorf("Expected true, got false")
	}
}

func TestDecimalCompare(t *testing.T) {
	var a, b dec128.Dec128

	a = dec128.FromString("NaN")
	b = dec128.FromString("NaN")
	if a.Compare(b) != 0 {
		t.Errorf("Expected 0, got %d", a.Compare(b))
	}

	a = dec128.FromString("0")
	b = dec128.FromString("NaN")
	if a.Compare(b) != 1 {
		t.Errorf("Expected 1, got %d", a.Compare(b))
	}
	if b.Compare(a) != -1 {
		t.Errorf("Expected -1, got %d", b.Compare(a))
	}

	a = dec128.FromString("0")
	b = dec128.FromString("0")
	if a.Compare(b) != 0 {
		t.Errorf("Expected 0, got %d", a.Compare(b))
	}

	a = dec128.FromString("1")
	b = dec128.FromString("-1")
	if a.Compare(b) != 1 {
		t.Errorf("Expected 1, got %d", a.Compare(b))
	}
	if b.Compare(a) != -1 {
		t.Errorf("Expected -1, got %d", b.Compare(a))
	}

	a = dec128.FromString(uint128.MaxStr)
	b = dec128.FromString("0.0001")
	if a.Compare(b) != 1 {
		t.Errorf("Expected 1, got %d", a.Compare(b))
	}
	if b.Compare(a) != -1 {
		t.Errorf("Expected -1, got %d", b.Compare(a))
	}

	a = dec128.FromUint64(1000, 1)
	b = dec128.FromUint64(10000, 2)
	if a.Compare(b) != 0 {
		t.Errorf("Expected 0, got %d", a.Compare(b))
	}

	a = dec128.FromString("123.456")
	b = dec128.FromString("123.4560000")
	if a.Compare(b) != 0 {
		t.Errorf("Expected 0, got %d", a.Compare(b))
	}

	a = dec128.FromString("123.456")
	b = dec128.FromString("123.4560001")
	if a.Compare(b) != -1 {
		t.Errorf("Expected -1, got %d", a.Compare(b))
	}
	if b.Compare(a) != 1 {
		t.Errorf("Expected 1, got %d", b.Compare(a))
	}
}

func TestDecimalCanonical(t *testing.T) {
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
			d := dec128.FromString(tc.i)
			if d.IsNaN() {
				t.Errorf("Expected no error, got: %v", d.ErrorDetails())
			}
			c := d.Canonical()
			if c.IsNaN() {
				t.Errorf("Expected no error, got: %v", c.ErrorDetails())
			}
			s := c.String()
			if s != tc.s {
				t.Errorf("Expected '%s', got: %s", tc.s, s)
			}
			if d.Precision() != tc.e1 {
				t.Errorf("Expected %d, got: %d", tc.e1, d.Precision())
			}
			if c.Precision() != tc.e2 {
				t.Errorf("Expected %d, got: %d", tc.e2, c.Precision())
			}
		})
	}
}

func TestDecimalAdd(t *testing.T) {
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
			a := dec128.FromString(tc.a)
			b := dec128.FromString(tc.b)
			c := a.Add(b)
			s := c.String()
			if s != tc.s {
				t.Errorf("Expected '%s', got: %s", tc.s, s)
			}
		})
	}
}

func TestDecimalSub(t *testing.T) {
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
			a := dec128.FromString(tc.a)
			b := dec128.FromString(tc.b)
			c := a.Sub(b)
			s := c.String()
			if s != tc.s {
				t.Errorf("Expected '%s', got: %s", tc.s, s)
			}
		})
	}
}

func TestDecimalMul(t *testing.T) {
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
			a := dec128.FromString(tc.a)
			b := dec128.FromString(tc.b)
			c := a.Mul(b)
			s := c.String()
			if s != tc.r {
				t.Errorf("Expected '%s', got: %s", tc.r, s)
			}
			if tc.e != "" && !c.IsNaN() {
				t.Errorf("Expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && c.IsNaN() {
				t.Errorf("Expected no error, got: %v", c.ErrorDetails())
			}
		})
	}
}

func TestDecimalDiv(t *testing.T) {
	type testCase struct {
		a string
		b string
		r string
		e string
	}

	dec128.SetDefaultPrecision(10)

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
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalDiv(%s / %s)", tc.a, tc.b), func(t *testing.T) {
			a := dec128.FromString(tc.a)
			b := dec128.FromString(tc.b)
			c := a.Div(b)
			s := c.String()
			if s != tc.r {
				t.Errorf("Expected '%s', got: %s", tc.r, s)
			}
			if tc.e != "" && !c.IsNaN() {
				t.Errorf("Expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && c.IsNaN() {
				t.Errorf("Expected no error, got: %v", c.ErrorDetails())
			}
		})
	}
}

func TestSign(t *testing.T) {
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
			a := dec128.FromString(tc.a)
			if a.IsNaN() {
				t.Errorf("Expected no error, got: %v", a.ErrorDetails())
			}

			c := a.Sign()
			if c != tc.want {
				t.Errorf("Expected %d, got: %d", tc.want, c)
			}
		})
	}
}
