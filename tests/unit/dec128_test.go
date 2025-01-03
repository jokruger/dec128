package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/uint128"
)

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
		{"12345678901234567890123456789012345678901234567890", "NaN", "overflow"},
		{".123456789012345678901234567890", "NaN", "overflow"},
		{"0.00", "0", ""},
		{"1.00", "1", ""},
		{"1.10", "1.1", ""},
		{"1.01", "1.01", ""},
		{"1.001", "1.001", ""},
		{"1.0000001", "1.0000001", ""},
		{"1.1000000", "1.1", ""},
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
