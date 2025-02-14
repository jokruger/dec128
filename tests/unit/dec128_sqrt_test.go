package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalSqrt(t *testing.T) {
	dec128.SetDefaultPrecision(19)

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
			d := dec128.FromString(tc.a).Sqrt()
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
	dec128.SetDefaultPrecision(6)

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
			d := dec128.FromString(tc.a).Sqrt()
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
