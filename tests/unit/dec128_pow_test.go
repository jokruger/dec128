package unit

import (
	"fmt"
	"testing"

	"github.com/jokruger/dec128"
)

func TestDecimalPowInt(t *testing.T) {
	dec128.SetDefaultPrecision(19)

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
			r := dec128.FromString(tc.a).PowInt(tc.p)
			if r.String() != tc.s {
				t.Errorf("Expected %s, got %s", tc.s, r.String())
			}
			if tc.e == "" && r.IsNaN() {
				t.Errorf("Expected a valid result, got %s", r.ErrorDetails().Error())
			}
			if tc.e != "" && (!r.IsNaN() || r.ErrorDetails().Error() != tc.e) {
				t.Errorf("Expected %s, got %v", tc.e, r.ErrorDetails())
			}
		})
	}
}
