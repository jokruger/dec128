package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/uint128"
)

func TestDecimalParseStringHLE(t *testing.T) {
	dec128.SetDefaultPrecision(19)

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

			d = dec128.FromSafeString(tc.i)
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
	dec128.SetDefaultPrecision(19)

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
				d = dec128.FromSafeString(tc.i)
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
	dec128.SetDefaultPrecision(19)

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
			d := dec128.New(uint128.FromUint64(tc.i), tc.e, false)
			s := d.StringFixed()
			if s != tc.s {
				t.Errorf("expected '%s', got: %s", tc.s, s)
			}
		})
	}
}

func TestDecimalToStringFixed2(t *testing.T) {
	dec128.SetDefaultPrecision(19)

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
			d := dec128.FromString(tc.i)
			s := d.StringFixed()
			if s != tc.s {
				t.Errorf("expected '%s', got: %s", tc.s, s)
			}
		})
	}
}
