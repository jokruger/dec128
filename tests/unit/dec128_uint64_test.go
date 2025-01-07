package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
	"github.com/jokruger/dec128/uint128"
)

func TestDecimalFromUint64Encoding(t *testing.T) {
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
			d := dec128.New(uint128.FromUint64(tc.i), tc.p, false)
			s := d.String()
			if s != tc.s {
				t.Errorf("Expected '%s', got: %s", tc.s, s)
			}
		})
	}
}

func TestDecimalUint64Encoding(t *testing.T) {
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
			d := dec128.FromString(tc.i)
			u, err := d.EncodeToUint64(tc.p)
			if tc.e != "" && err == nil {
				t.Errorf("Expected error '%s', got nil", tc.e)
			}
			if tc.e == "" && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if u != tc.u {
				t.Errorf("Expected %d, got: %d", tc.u, u)
			}
		})
	}
}

func TestDecimalUint64Encoding2(t *testing.T) {
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
			d := dec128.FromString(tc.i)
			u, err := d.EncodeToUint64(tc.p)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			s := dec128.New(uint128.FromUint64(u), tc.p, false).String()
			if s != tc.s {
				t.Errorf("Expected '%s', got: %s", tc.s, s)
			}
		})
	}
}
