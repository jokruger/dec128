package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalMul(t *testing.T) {
	dec128.SetDefaultPrecision(19)

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
	dec128.SetDefaultPrecision(10)

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

func TestDecimalDiv2(t *testing.T) {
	dec128.SetDefaultPrecision(19)

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

func TestDecimalDiv3(t *testing.T) {
	dec128.SetDefaultPrecision(6)

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

func TestDecimalMod1(t *testing.T) {
	dec128.SetDefaultPrecision(19)

	type testCase struct {
		a string
		b string
		r string
		e string
	}

	testCases := [...]testCase{
		{"0", "0", "NaN", "division by zero"},
		{"123", "10", "3", ""},
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
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("TestDecimalDiv(%s / %s)", tc.a, tc.b), func(t *testing.T) {
			a := dec128.FromString(tc.a)
			b := dec128.FromString(tc.b)
			c := a.Mod(b)
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
